package middleware

import (
	"context"
	"net/http"
	"strings"

	"rubxy/auth"
	"rubxy/config"
	"rubxy/logger"
)

type contextKey string

const userContextKey = contextKey("user")

func Authenticate(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := auth.ValidateToken(token, cfg, false)
			if err != nil {
				// Log failed authentication attempts (simplified)
				logger.InfoLogger.Printf("[AUTH] Unauthorized - Path: %s, Error: %v", r.URL.Path, err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Log successful authentication
			ctx := context.WithValue(r.Context(), userContextKey, claims.Username)
			logger.InfoLogger.Printf("[AUTH] Authenticated - User: %s, Path: %s", claims.Username, r.URL.Path)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserFromContext(r *http.Request) string {
	user, _ := r.Context().Value(userContextKey).(string)
	return user
}

// CleanPath trims trailing spaces and normalizes the request path
func CleanPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Trim trailing spaces from the path
		r.URL.Path = strings.TrimRight(r.URL.Path, " ")
		next.ServeHTTP(w, r)
	})
}
