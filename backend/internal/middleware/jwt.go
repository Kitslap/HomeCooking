package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authjwt "github.com/pierreburbaud/cooking-home/internal/auth"
)

// ctxKeyUserID est la clé utilisée pour stocker l'ID utilisateur dans le contexte Gin.
type ctxKeyUserID struct{}

const CtxKeyUserID = "userID"
const CtxKeyEmail  = "email"

// JWTAuth valide le token Bearer dans le header Authorization.
// En cas de token absent, expiré ou altéré, la requête est immédiatement
// rejetée avec un 401. Les détails de l'erreur JWT ne sont jamais exposés
// au client pour éviter toute fuite d'information.
func JWTAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authentification requise",
			})
			return
		}

		// Format attendu : "Bearer <token>"
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "format d'autorisation invalide (attendu : Bearer <token>)",
			})
			return
		}

		claims, err := authjwt.ValidateAccessToken(parts[1], jwtSecret)
		if err != nil {
			// Réponse générique — ne jamais exposer la raison exacte (expiry, signature, etc.)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token invalide ou expiré",
			})
			return
		}

		// Injection des claims dans le contexte pour les handlers aval
		c.Set(CtxKeyUserID, claims.UserID)
		c.Set(CtxKeyEmail,  claims.Email)
		c.Next()
	}
}

// UserIDFromCtx extrait l'ID utilisateur injecté par JWTAuth.
// Panique si appelé hors d'une route protégée (bug développeur).
func UserIDFromCtx(c *gin.Context) int64 {
	v, exists := c.Get(CtxKeyUserID)
	if !exists {
		panic("UserIDFromCtx appelé sans le middleware JWTAuth")
	}
	return v.(int64)
}

// EmailFromCtx extrait l'email injecté par JWTAuth.
func EmailFromCtx(c *gin.Context) string {
	v, _ := c.Get(CtxKeyEmail)
	s, _ := v.(string)
	return s
}
