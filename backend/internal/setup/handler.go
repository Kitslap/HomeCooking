// Package setup gère l'assistant de configuration initiale de l'instance.
//
// Au premier lancement, la table users est vide : l'instance est en mode "setup".
// Deux endpoints non-authentifiés sont exposés :
//   - GET  /setup/status → indique si le setup est nécessaire
//   - POST /setup        → crée le compte administrateur initial
//
// Dès qu'un utilisateur existe, POST /setup renvoie systématiquement 403.
// Ce verrouillage est dérivé de l'état réel de la donnée (COUNT users),
// pas d'un flag manipulable.
package setup

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/Kitslap/HomeCooking/internal/auth"
)

// ── Dépendances injectées ────────────────────────────────────────────────────

// Deps regroupe les dépendances nécessaires aux handlers du setup.
// Même pattern d'injection que auth.HandlerDeps.
type Deps struct {
	DB         *sql.DB
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	IsProd     bool
}

// ── Payloads ─────────────────────────────────────────────────────────────────

type setupInput struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=10,max=128"`
}

type statusResponse struct {
	NeedsSetup bool `json:"needs_setup"`
}

// ── RegisterRoutes ───────────────────────────────────────────────────────────

// RegisterRoutes enregistre les routes du setup sur le groupe fourni (public).
//
//	GET  /setup/status — l'instance a-t-elle besoin du setup initial ?
//	POST /setup        — création du compte administrateur (verrouillé si déjà fait)
func RegisterRoutes(r *gin.RouterGroup, deps Deps) {
	g := r.Group("/setup")
	g.GET("/status", statusHandler(deps))
	g.POST("", createHandler(deps))
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// statusHandler indique si l'instance nécessite une configuration initiale.
// Retourne { "needs_setup": true } si aucun utilisateur n'existe en base,
// { "needs_setup": false } sinon. Ne leak aucune donnée sensible.
func statusHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		empty, err := isUsersTableEmpty(c.Request.Context(), deps.DB)
		if err != nil {
			log.Error().Err(err).Msg("setup/status: erreur comptage utilisateurs")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}
		c.JSON(http.StatusOK, statusResponse{NeedsSetup: empty})
	}
}

// createHandler crée le premier compte administrateur de l'instance.
//
// Sécurité :
//   - Vérifie COUNT(*) FROM users == 0 avant toute action
//   - Si un utilisateur existe → 403 Forbidden immédiat (endpoint mort)
//   - Valide le format username (alphanum + ._-)
//   - Hash bcrypt coût 12 (≈250ms, protection brute-force)
//   - Insère avec role = 'admin'
//   - Émet une paire access/refresh token → login automatique après setup
func createHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ── Guard : le setup n'est possible que si aucun utilisateur n'existe ──
		empty, err := isUsersTableEmpty(c.Request.Context(), deps.DB)
		if err != nil {
			log.Error().Err(err).Msg("setup/create: erreur comptage utilisateurs")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}
		if !empty {
			c.JSON(http.StatusForbidden, gin.H{"error": "instance déjà configurée"})
			return
		}

		// ── Validation du payload ─────────────────────────────────────────────
		var input setupInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validation du format username : lettres, chiffres, ._-
		if !isValidUsername(input.Username) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "nom d'utilisateur invalide (lettres, chiffres, ._- uniquement)",
			})
			return
		}

		// ── Création du compte administrateur ─────────────────────────────────
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
		if err != nil {
			log.Error().Err(err).Msg("setup/create: bcrypt")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		userID, err := createAdminUser(c.Request.Context(), deps.DB, input.Username, string(hash))
		if err != nil {
			log.Error().Err(err).Msg("setup/create: insertion admin")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		log.Info().
			Int64("user_id", userID).
			Str("username", input.Username).
			Msg("setup: compte administrateur créé — instance configurée")

		// ── Émission des tokens → login automatique ──────────────────────────
		accessToken, err := auth.GenerateAccessToken(userID, input.Username, deps.JWTSecret, deps.AccessTTL)
		if err != nil {
			log.Error().Err(err).Msg("setup/create: génération access token")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		refreshToken, err := auth.GenerateRefreshToken(userID, deps.JWTSecret, deps.RefreshTTL)
		if err != nil {
			log.Error().Err(err).Msg("setup/create: génération refresh token")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		// Stockage du refresh token en base
		if err := storeRefreshToken(c.Request.Context(), deps.DB, userID, refreshToken, deps.RefreshTTL); err != nil {
			log.Error().Err(err).Msg("setup/create: stockage refresh token")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		// Cookie httpOnly — inaccessible au JS, protégé CSRF par SameSite
		c.SetCookie(
			"refresh_token",
			refreshToken,
			int(deps.RefreshTTL.Seconds()),
			"/api/v1/auth",
			"",
			deps.IsProd,
			true,
		)

		c.JSON(http.StatusCreated, gin.H{
			"access_token": accessToken,
			"expires_in":   int64(deps.AccessTTL.Seconds()),
		})
	}
}

// ── Validation ───────────────────────────────────────────────────────────────

// isValidUsername vérifie le format du nom d'utilisateur : 3-32 caractères,
// uniquement lettres, chiffres, points, tirets et underscores.
func isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 32 {
		return false
	}
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// ── Requêtes SQL ─────────────────────────────────────────────────────────────

// isUsersTableEmpty retourne true si aucun utilisateur n'existe en base.
// Requête légère : EXISTS est plus performant que COUNT(*) sur SQLite.
func isUsersTableEmpty(ctx context.Context, db *sql.DB) (bool, error) {
	var exists int
	err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users LIMIT 1)`,
	).Scan(&exists)
	return exists == 0, err
}

// createAdminUser insère le premier utilisateur avec le rôle 'admin'.
func createAdminUser(ctx context.Context, db *sql.DB, username, passwordHash string) (int64, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'admin')`,
		username, passwordHash,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// storeRefreshToken persiste le refresh token en base pour la rotation/révocation.
func storeRefreshToken(ctx context.Context, db *sql.DB, userID int64, token string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)
	_, err := db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (user_id, token, expires_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(token) DO UPDATE SET expires_at = excluded.expires_at`,
		userID, token, expiresAt.UTC().Format(time.RFC3339),
	)
	return err
}
