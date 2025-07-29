package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var JWTSecret []byte

// SetSecret sets the JWT secret key (e.g., from config)
func SetSecret(secret string) {
	JWTSecret = []byte(secret)
}

// Claims represents the JWT payload
type Claims struct {
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for the given tenant
func GenerateToken(tenantID string) (string, error) {
	if len(JWTSecret) == 0 {
		return "", errors.New("JWT secret not set")
	}

	claims := Claims{
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// ValidateToken parses and verifies a JWT string
func ValidateToken(tokenStr string) (*Claims, error) {
	if len(JWTSecret) == 0 {
		return nil, errors.New("JWT secret not set")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claims, nil
}
