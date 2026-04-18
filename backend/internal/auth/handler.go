package auth

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/Kitslap/HomeCooking/internal/httperror"
)

// ── Dépendances injectées ──────────────────────────────────────────────────

// HandlerDeps regroupe les dépendances nécessaires aux handlers d'authentification.
type HandlerDeps struct {
	DB            *sql.DB
	JWTSecret     string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	IsProd        bool // détermine les flags du cookie (Secure, SameSite)
}

// ── Payloads ──────────────────────────────────────────────────────────────

type registerInput struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=10,max=128"`
}

type loginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"` // secondes
}

// ── RegisterPublicRoutes ──────────────────────────────────────────────────

// RegisterPublicRoutes enregistre les routes d'authentification publiques (sans JWT).
//
//	POST /auth/login    — connexion
//	POST /auth/refresh  — renouvellement de l'access token via refresh cookie
//	POST /auth/logout   — révocation du refresh token
func RegisterPublicRoutes(r *gin.RouterGroup, deps HandlerDeps) {
	g := r.Group("/auth")
	g.POST("/login",   loginHandler(deps))
	g.POST("/refresh", refreshHandler(deps))
	g.POST("/logout",  logoutHandler(deps))
}

// ── RegisterAdminRoutes ──────────────────────────────────────────────────

// RegisterAdminRoutes enregistre les routes d'authentification protégées.
// Doit être appelé sur un groupe déjà protégé par JWTAuth.
//
//	POST /auth/register — création de compte (admin uniquement)
func RegisterAdminRoutes(r *gin.RouterGroup, deps HandlerDeps) {
	g := r.Group("/auth")
	g.POST("/register", adminGuard(deps.DB), registerHandler(deps))
}

// adminGuard vérifie que l'utilisateur connecté a le rôle 'admin'.
// Retourne 403 si ce n'est pas le cas.
func adminGuard(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Accès refusé."})
			return
		}

		var role string
		err := db.QueryRowContext(c.Request.Context(),
			`SELECT role FROM users WHERE id = ?`, userID,
		).Scan(&role)
		if err != nil || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Droits administrateur requis."})
			return
		}

		c.Next()
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────

// registerHandler crée un nouveau compte utilisateur.
// Le mot de passe est hashé avec bcrypt (coût 12) avant toute persistance.
func registerHandler(deps HandlerDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input registerInput
		if err := c.ShouldBindJSON(&input); err != nil {
			log.Debug().Err(err).Msg("register: payload invalide")
			c.JSON(http.StatusBadRequest, gin.H{"error": httperror.FormatBindingError(err)})
			return
		}

		// Validation du format username : alphanum + ._-
		if !isValidUsername(input.Username) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Le nom d'utilisateur ne peut contenir que des lettres, chiffres et les caractères « . », « _ » et « - »."})
			return
		}

		ctx := c.Request.Context()

		// Vérification si le username existe déjà (réponse identique à "succès"
		// pour éviter l'énumération de comptes — timing-safe via bcrypt delay)
		exists, err := usernameExists(ctx, deps.DB, input.Username)
		if err != nil {
			log.Error().Err(err).Msg("register: vérification username")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Une erreur interne est survenue, veuillez réessayer."})
			return
		}
		if exists {
			// Délai artificiel identique au hash pour éviter le timing side-channel
			bcrypt.GenerateFromPassword([]byte(input.Password), 12) //nolint:errcheck
			c.JSON(http.StatusConflict, gin.H{"error": "Ce nom d'utilisateur est déjà utilisé."})
			return
		}

		// Hash bcrypt — coût 12 (≈250ms sur matériel moderne = protection brute-force)
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
		if err != nil {
			log.Error().Err(err).Msg("register: bcrypt")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Une erreur interne est survenue, veuillez réessayer."})
			return
		}

		// Insertion en base (role = 'user' par défaut pour les comptes créés via register)
		userID, err := createUser(ctx, deps.DB, input.Username, string(hash))
		if err != nil {
			log.Error().Err(err).Msg("register: insertion utilisateur")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Une erreur interne est survenue, veuillez réessayer."})
			return
		}

		log.Info().Int64("user_id", userID).Str("username", input.Username).Msg("register: nouvel utilisateur créé")

		// Retourne directement une paire de tokens pour une UX fluide
		issueTokenPair(c, deps, userID, input.Username)
	}
}

// loginHandler vérifie les credentials et émet une paire access/refresh token.
func loginHandler(deps HandlerDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input loginInput
		if err := c.ShouldBindJSON(&input); err != nil {
			log.Debug().Err(err).Msg("login: payload invalide")
			c.JSON(http.StatusBadRequest, gin.H{"error": httperror.FormatBindingError(err)})
			return
		}

		ctx := c.Request.Context()

		// Récupération de l'utilisateur
		user, err := getUserByUsername(ctx, deps.DB, input.Username)
		if err != nil {
			// Réponse identique quel que soit le cas (username inconnu ou mdp erroné)
			// pour éviter l'énumération de comptes
			bcrypt.CompareHashAndPassword([]byte("$2a$12$placeholder"), []byte(input.Password)) //nolint:errcheck
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants incorrects."})
			return
		}

		// Vérification du mot de passe
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
			log.Warn().Int64("user_id", user.ID).Str("ip", c.ClientIP()).Msg("login: mot de passe incorrect")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants incorrects."})
			return
		}

		log.Info().Int64("user_id", user.ID).Msg("login: authentification réussie")
		issueTokenPair(c, deps, user.ID, user.Username)
	}
}

// refreshHandler renouvelle l'access token depuis le refresh token httpOnly cookie.
func refreshHandler(deps HandlerDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken, err := c.Cookie("refresh_token")
		if err != nil || refreshToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expirée — veuillez vous reconnecter."})
			return
		}

		claims, err := ValidateRefreshToken(refreshToken, deps.JWTSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expirée — veuillez vous reconnecter."})
			return
		}

		// Vérification que le refresh token est bien en base (révocation possible)
		valid, err := isRefreshTokenValid(c.Request.Context(), deps.DB, claims.UserID, refreshToken)
		if err != nil || !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session révoquée — veuillez vous reconnecter."})
			return
		}

		// Récupération du username courant
		user, err := getUserByID(c.Request.Context(), deps.DB, claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur introuvable."})
			return
		}

		// Rotation : révoque l'ancien, émet un nouveau
		issueTokenPair(c, deps, user.ID, user.Username)
	}
}

// logoutHandler révoque le refresh token en base et supprime le cookie.
func logoutHandler(deps HandlerDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken, err := c.Cookie("refresh_token")
		if err == nil && refreshToken != "" {
			_ = revokeRefreshToken(c.Request.Context(), deps.DB, refreshToken)
		}
		// Suppression du cookie côté client
		c.SetCookie("refresh_token", "", -1, "/api/v1/auth", "", deps.IsProd, true)
		c.JSON(http.StatusOK, gin.H{"message": "Déconnecté."})
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

// issueTokenPair génère une paire access/refresh, stocke le refresh en base
// et le pose dans un cookie httpOnly.
func issueTokenPair(c *gin.Context, deps HandlerDeps, userID int64, username string) {
	accessToken, err := GenerateAccessToken(userID, username, deps.JWTSecret, deps.AccessTTL)
	if err != nil {
		log.Error().Err(err).Msg("issueTokenPair: access token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
		return
	}

	refreshToken, err := GenerateRefreshToken(userID, deps.JWTSecret, deps.RefreshTTL)
	if err != nil {
		log.Error().Err(err).Msg("issueTokenPair: refresh token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
		return
	}

	// Rotation : stocker le nouveau refresh token en base
	if err := storeRefreshToken(c.Request.Context(), deps.DB, userID, refreshToken, deps.RefreshTTL); err != nil {
		log.Error().Err(err).Msg("issueTokenPair: stockage refresh token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
		return
	}

	// Cookie httpOnly — inaccessible au JS, protégé CSRF par SameSite=Strict
	c.SetCookie(
		"refresh_token",
		refreshToken,
		int(deps.RefreshTTL.Seconds()),
		"/api/v1/auth",   // Path limité aux routes auth
		"",               // Domain vide = domaine courant
		deps.IsProd,      // Secure = true en production (HTTPS obligatoire)
		true,             // HttpOnly = vrai — inaccessible au JS
	)

	c.JSON(http.StatusOK, tokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   int64(deps.AccessTTL.Seconds()),
	})
}

// ── Requêtes SQL ──────────────────────────────────────────────────────────

type dbUser struct {
	ID           int64
	Username     string
	PasswordHash string
}

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

func usernameExists(ctx context.Context, db *sql.DB, username string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM users WHERE username = ?`, username,
	).Scan(&count)
	return count > 0, err
}

func createUser(ctx context.Context, db *sql.DB, username, passwordHash string) (int64, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'user')`, username, passwordHash,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func getUserByUsername(ctx context.Context, db *sql.DB, username string) (dbUser, error) {
	var u dbUser
	err := db.QueryRowContext(ctx,
		`SELECT id, username, password_hash FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash)
	return u, err
}

func getUserByID(ctx context.Context, db *sql.DB, id int64) (dbUser, error) {
	var u dbUser
	err := db.QueryRowContext(ctx,
		`SELECT id, username, password_hash FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash)
	return u, err
}

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

func isRefreshTokenValid(ctx context.Context, db *sql.DB, userID int64, token string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM refresh_tokens
		 WHERE user_id = ? AND token = ? AND expires_at > datetime('now') AND revoked = 0`,
		userID, token,
	).Scan(&count)
	return count > 0, err
}

func revokeRefreshToken(ctx context.Context, db *sql.DB, token string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked = 1 WHERE token = ?`, token,
	)
	return err
}
