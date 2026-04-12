package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig contient les paramètres CORS transmis au middleware.
type CORSConfig struct {
	// AllowedOrigins est la liste explicite des origines autorisées.
	// Jamais de wildcard "*" en présence de cookies / Authorization.
	AllowedOrigins []string
}

// CORS retourne un middleware Gin qui applique une politique CORS stricte.
// Seules les origines déclarées dans cfg.AllowedOrigins sont acceptées.
// Les requêtes preflight OPTIONS reçoivent une réponse immédiate 204.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	// Méthodes et headers autorisés — liste fermée, pas de wildcard
	allowedMethods := "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowedHeaders := "Origin, Content-Type, Authorization, X-Request-ID"
	exposedHeaders := "X-Request-ID"

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Pas d'en-tête Origin → requête same-origin ou non-browser, on passe
		if origin == "" {
			c.Next()
			return
		}

		// Vérification de l'origine — comparaison exacte, pas de glob
		if !isOriginAllowed(origin, cfg.AllowedOrigins) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "origin non autorisée",
			})
			return
		}

		// En-têtes CORS sur toutes les réponses
		c.Header("Access-Control-Allow-Origin",   origin)
		c.Header("Access-Control-Allow-Methods",  allowedMethods)
		c.Header("Access-Control-Allow-Headers",  allowedHeaders)
		c.Header("Access-Control-Expose-Headers", exposedHeaders)
		c.Header("Access-Control-Max-Age",        "86400") // 24h de cache preflight
		c.Header("Vary",                          "Origin")

		// Réponse immédiate pour les requêtes preflight OPTIONS
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isOriginAllowed vérifie si l'origine est dans la liste autorisée.
// La comparaison est insensible à la casse pour le schéma/hôte.
func isOriginAllowed(origin string, allowed []string) bool {
	return slices.ContainsFunc(allowed, func(a string) bool {
		return strings.EqualFold(a, origin)
	})
}
