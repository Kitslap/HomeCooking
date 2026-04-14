// Package auth gère l'authentification stateless par JWT double-token :
//   - Access token  : courte durée (15 min), transmis dans le header Authorization
//   - Refresh token : longue durée (7 jours), stocké en base + httpOnly cookie
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ── Types de claims ────────────────────────────────────────────────────────

// AccessClaims est embarqué dans le JWT d'accès.
type AccessClaims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// RefreshClaims est embarqué dans le JWT de rafraîchissement.
type RefreshClaims struct {
	UserID    int64  `json:"uid"`
	TokenType string `json:"typ"` // toujours "refresh"
	jwt.RegisteredClaims
}

// ── Génération ─────────────────────────────────────────────────────────────

// GenerateAccessToken crée un JWT d'accès signé HS256.
// Le secret doit faire au moins 32 octets (64 hex chars recommandé).
func GenerateAccessToken(userID int64, username, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "cooking-home",
			Subject:   "access",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken crée un JWT de rafraîchissement signé HS256.
// Il sera stocké en base pour pouvoir être révoqué (rotation).
func GenerateRefreshToken(userID int64, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := RefreshClaims{
		UserID:    userID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "cooking-home",
			Subject:   "refresh",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ── Validation ─────────────────────────────────────────────────────────────

// ValidateAccessToken parse et valide un access token.
// Retourne une erreur générique — ne jamais exposer les détails JWT au client.
func ValidateAccessToken(tokenStr, secret string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&AccessClaims{},
		keyFunc(secret),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("token invalide")
	}
	claims, ok := token.Claims.(*AccessClaims)
	if !ok {
		return nil, errors.New("claims malformés")
	}
	return claims, nil
}

// ValidateRefreshToken parse et valide un refresh token.
func ValidateRefreshToken(tokenStr, secret string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&RefreshClaims{},
		keyFunc(secret),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("refresh token invalide")
	}
	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || claims.TokenType != "refresh" {
		return nil, errors.New("type de token incorrect")
	}
	return claims, nil
}

// keyFunc retourne la fonction de clé utilisée par golang-jwt.
// Vérifie explicitement la méthode de signature pour éviter l'attaque "alg:none".
func keyFunc(secret string) jwt.Keyfunc {
	return func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("méthode de signature inattendue")
		}
		return []byte(secret), nil
	}
}
