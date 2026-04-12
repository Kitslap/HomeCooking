package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig configure le token-bucket par IP.
type RateLimiterConfig struct {
	// RPS : tokens accordés par seconde (ex : 20.0)
	RPS float64
	// Burst : rafale maximale autorisée (ex : 40)
	Burst int
}

// visitor associe un limiter à une IP cliente.
type visitor struct {
	limiter *rate.Limiter
}

// ipLimiter maintient un limiter par IP en mémoire.
// Pour un usage en cluster, remplacer par Redis + lua script.
type ipLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*visitor
	rps      rate.Limit
	burst    int
}

func newIPLimiter(rps float64, burst int) *ipLimiter {
	return &ipLimiter{
		visitors: make(map[string]*visitor),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

// getLimiter retourne le limiter de l'IP, en le créant si nécessaire.
func (l *ipLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.RLock()
	v, ok := l.visitors[ip]
	l.mu.RUnlock()
	if ok {
		return v.limiter
	}

	// Création protégée par un write-lock (double-check)
	l.mu.Lock()
	defer l.mu.Unlock()
	if v, ok = l.visitors[ip]; ok {
		return v.limiter
	}
	lim := rate.NewLimiter(l.rps, l.burst)
	l.visitors[ip] = &visitor{limiter: lim}
	return lim
}

// RateLimiter retourne un middleware Gin qui limite les requêtes par IP
// en utilisant un algorithme token-bucket.
//
// Réponse en cas de dépassement :
//
//	429 Too Many Requests + header Retry-After
func RateLimiter(cfg RateLimiterConfig) gin.HandlerFunc {
	limiter := newIPLimiter(cfg.RPS, cfg.Burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		lim := limiter.getLimiter(ip)

		if !lim.Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "trop de requêtes — réessaie dans un instant",
			})
			return
		}
		c.Next()
	}
}
