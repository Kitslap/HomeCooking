// Package config charge et valide la configuration de l'application
// depuis les variables d'environnement (fichier .env en développement).
// Toute valeur manquante ou invalide provoque un arrêt immédiat au démarrage.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config regroupe l'ensemble des paramètres de l'application.
// Elle est construite une seule fois au démarrage et transmise par valeur.
type Config struct {
	// Serveur
	Port string
	Env  string // "development" | "production"

	// JWT
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	// Base de données
	DBPath string

	// CORS — liste des origines autorisées
	CORSOrigins []string

	// Rate limiting
	RateLimitRPS   float64 // tokens par seconde par IP
	RateLimitBurst int     // rafale maximale autorisée
}

// IsDev retourne true si l'application tourne en mode développement.
func (c Config) IsDev() bool { return c.Env == "development" }

// Load lit le fichier .env (ignoré si absent en production),
// puis construit et valide la Config. Toute erreur est fatale.
func Load() (Config, error) {
	// Chargement du .env — silencieux si absent (production via env système)
	_ = godotenv.Load()

	cfg := Config{}
	var errs []string

	// ── Serveur ───────────────────────────────────────────────────────────
	cfg.Port = envOrDefault("PORT", "8080")
	cfg.Env = envOrDefault("ENV", "development")

	// ── JWT ───────────────────────────────────────────────────────────────
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" || cfg.JWTSecret == "CHANGE_ME_generate_with_openssl_rand_hex_64" {
		errs = append(errs, "JWT_SECRET est vide ou utilise la valeur par défaut — génère-en un avec : openssl rand -hex 64")
	}
	if len(cfg.JWTSecret) < 32 {
		errs = append(errs, "JWT_SECRET doit faire au moins 32 caractères")
	}

	var err error
	cfg.JWTAccessTTL, err = parseDuration("JWT_ACCESS_TTL", "15m")
	if err != nil {
		errs = append(errs, err.Error())
	}
	cfg.JWTRefreshTTL, err = parseDuration("JWT_REFRESH_TTL", "168h") // 7 jours
	if err != nil {
		errs = append(errs, err.Error())
	}

	// ── Base de données ───────────────────────────────────────────────────
	cfg.DBPath = envOrDefault("DB_PATH", "./data/cooking-home.db")

	// ── CORS ──────────────────────────────────────────────────────────────
	rawOrigins := envOrDefault("CORS_ORIGINS", "http://localhost:5173")
	for _, o := range strings.Split(rawOrigins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			cfg.CORSOrigins = append(cfg.CORSOrigins, o)
		}
	}

	// ── Rate limiting ─────────────────────────────────────────────────────
	cfg.RateLimitRPS, err = parseFloat("RATE_LIMIT_RPS", 20.0)
	if err != nil {
		errs = append(errs, err.Error())
	}
	cfg.RateLimitBurst, err = parseInt("RATE_LIMIT_BURST", 40)
	if err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return Config{}, errors.New("configuration invalide :\n  - " + strings.Join(errs, "\n  - "))
	}
	return cfg, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// parseDuration lit une durée Go ou une valeur avec suffixe "d" (jours).
// Ex : "15m", "1h", "7d".
func parseDuration(key, def string) (time.Duration, error) {
	raw := envOrDefault(key, def)
	// Support du suffixe "d" pour les jours
	if strings.HasSuffix(raw, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(raw, "d"))
		if err != nil {
			return 0, fmt.Errorf("%s: valeur invalide %q", key, raw)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: valeur invalide %q (attendu ex: 15m, 1h, 7d)", key, raw)
	}
	return d, nil
}

func parseFloat(key string, def float64) (float64, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: valeur invalide %q (attendu un nombre flottant)", key, raw)
	}
	return v, nil
}

func parseInt(key string, def int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: valeur invalide %q (attendu un entier)", key, raw)
	}
	return v, nil
}
