// internal/auth/middleware.go
package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const TenantIDKey contextKey = "tenant_id"

func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims, err := ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Inject tenant_id into context
		ctx := context.WithValue(r.Context(), TenantIDKey, claims.TenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTenantID extracts tenant_id from context
func GetTenantID(r *http.Request) string {
	if val := r.Context().Value(TenantIDKey); val != nil {
		return val.(string)
	}
	return ""
}
