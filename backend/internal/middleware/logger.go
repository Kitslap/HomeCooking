// Package middleware regroupe tous les middlewares Gin de la gateway.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger retourne un middleware Gin qui produit un log structuré JSON
// pour chaque requête HTTP : méthode, path, status, latence, IP cliente.
// En développement, zerolog utilise un format coloré lisible.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start  := time.Now()
		path   := c.Request.URL.Path
		query  := c.Request.URL.RawQuery

		// Traitement de la requête par les handlers suivants
		c.Next()

		latency := time.Since(start)
		status  := c.Writer.Status()
		clientIP := c.ClientIP()

		// Niveau de log adapté au code HTTP
		var event *zerolog.Event
		switch {
		case status >= 500:
			event = log.Error()
		case status >= 400:
			event = log.Warn()
		default:
			event = log.Info()
		}

		event.
			Str("method",  c.Request.Method).
			Str("path",    path).
			Str("query",   query).
			Str("ip",      clientIP).
			Int("status",  status).
			Dur("latency", latency).
			Int("bytes",   c.Writer.Size()).
			Msg("request")

		// Propager les éventuelles erreurs Gin dans les logs
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.Error().Err(e.Err).Msg("gin error")
			}
		}
	}
}
