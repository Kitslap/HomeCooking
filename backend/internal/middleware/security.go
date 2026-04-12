package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders ajoute les en-têtes de sécurité HTTP recommandés
// (OWASP Secure Headers Project, 2024).
// À activer en production ; en développement le HTTPS n'est pas requis.
func SecurityHeaders(isProd bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Empêche le navigateur de deviner le MIME type
		c.Header("X-Content-Type-Options", "nosniff")

		// Empêche le clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Désactive les anciens sniffers XSS (IE) — redondant avec CSP mais défense en profondeur
		c.Header("X-XSS-Protection", "0")

		// Référent limité à la même origine
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions API — principe du moindre privilège
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Content-Security-Policy — politique de base, à affiner selon le front
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline'; " + // unsafe-inline pour Vite en dev, à restreindre
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none';"
		c.Header("Content-Security-Policy", csp)

		// HSTS — uniquement en production (TLS obligatoire)
		if isProd {
			// 1 an, inclure les sous-domaines, préchargeable
			c.Header("Strict-Transport-Security",
				fmt.Sprintf("max-age=%d; includeSubDomains; preload", 365*24*3600))
		}

		c.Next()
	}
}
