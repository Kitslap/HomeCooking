// Package main est le point d'entrée de la gateway Cooking Home.
//
// Démarrage :
//
//	cp .env.example .env   # configurer JWT_SECRET et les autres variables
//	go run ./cmd/server
//
// Ou via le Makefile :
//
//	make run
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Kitslap/HomeCooking/internal/auth"
	"github.com/Kitslap/HomeCooking/internal/config"
	"github.com/Kitslap/HomeCooking/internal/db"
	"github.com/Kitslap/HomeCooking/internal/middleware"
	"github.com/Kitslap/HomeCooking/internal/recipe"
	"github.com/Kitslap/HomeCooking/internal/storage"
)

func main() {
	// ── Logging ───────────────────────────────────────────────────────────
	// JSON en production, console colorée en développement
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	// ── Configuration ─────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config: échec du chargement")
	}

	// Mode Gin adapté à l'environnement
	if cfg.IsDev() {
		gin.SetMode(gin.DebugMode)
		// Format console lisible en développement
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"})
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	log.Info().Str("env", cfg.Env).Str("port", cfg.Port).Msg("gateway: démarrage")

	// ── Base de données ───────────────────────────────────────────────────
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("db: impossible d'ouvrir la base")
	}
	defer database.Close()

	// ── Router Gin ────────────────────────────────────────────────────────
	r := gin.New() // gin.New() plutôt que gin.Default() — middlewares explicites

	// ── Middlewares globaux (ordre important) ─────────────────────────────

	// 1. Recovery : capture les panics et retourne un 500 propre
	r.Use(gin.Recovery())

	// 2. Logger structuré : log chaque requête avec latence, status, IP
	r.Use(middleware.Logger())

	// 3. En-têtes de sécurité HTTP (CSP, HSTS, X-Frame-Options…)
	r.Use(middleware.SecurityHeaders(!cfg.IsDev()))

	// 4. CORS : whitelist stricte, pas de wildcard
	r.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: cfg.CORSOrigins,
	}))

	// 5. Rate limiting par IP (token bucket)
	r.Use(middleware.RateLimiter(middleware.RateLimiterConfig{
		RPS:   cfg.RateLimitRPS,
		Burst: cfg.RateLimitBurst,
	}))

	// ── Routes ────────────────────────────────────────────────────────────

	// Healthcheck public (monitoring, load balancer)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "0.1.0",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Groupe API versionné
	apiV1 := r.Group("/api/v1")

	// Routes publiques — authentification
	auth.RegisterRoutes(apiV1, auth.HandlerDeps{
		DB:         database,
		JWTSecret:  cfg.JWTSecret,
		AccessTTL:  cfg.JWTAccessTTL,
		RefreshTTL: cfg.JWTRefreshTTL,
		IsProd:     !cfg.IsDev(),
	})

	// Routes protégées — JWT obligatoire sur tout le groupe
	protected := apiV1.Group("", middleware.JWTAuth(cfg.JWTSecret))

	// Stub de profil utilisateur (à déplacer dans un package dédié)
	protected.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": middleware.UserIDFromCtx(c),
			"email":   middleware.EmailFromCtx(c),
		})
	})

	// Recipes API — CRUD complet + recherche FTS5
	recipe.RegisterRoutes(protected, recipe.NewRepository(database))

	// Storage API — inventaire, alertes, liste de courses
	storage.RegisterRoutes(protected, storage.NewRepository(database))

	// Route 404 personnalisée (évite la page HTML par défaut de Gin)
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "route introuvable"})
	})

	// ── Serveur HTTP avec graceful shutdown ───────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Démarrage dans une goroutine pour libérer le main
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("gateway: en écoute")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("gateway: erreur serveur")
		}
	}()

	// Attente des signaux d'arrêt (SIGINT = Ctrl-C, SIGTERM = systemd/Docker)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("gateway: arrêt gracieux en cours…")

	// Délai de 10s pour finir les requêtes en cours
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("gateway: arrêt forcé")
	}

	log.Info().Msg("gateway: arrêté proprement")
}
