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
			logger.InfoLogger.Printf("[AUTH MIDDLEWARE] Checking authentication - Method: %s, Path: %s, RemoteAddr: %s",
				r.Method, r.URL.Path, r.RemoteAddr)

			authHeader := r.Header.Get("Authorization")
			hasAuthHeader := authHeader != ""
			token := strings.TrimPrefix(authHeader, "Bearer ")
			hasToken := token != "" && token != authHeader

			logger.InfoLogger.Printf("[AUTH MIDDLEWARE] Authorization header present: %v, Token present: %v, Token length: %d",
				hasAuthHeader, hasToken, len(token))

			claims, err := auth.ValidateToken(token, cfg, false)
			if err != nil {
				logger.InfoLogger.Printf("[AUTH MIDDLEWARE] Unauthorized access attempt - Path: %s, Error: %v", r.URL.Path, err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, claims.Username)
			logger.InfoLogger.Printf("[AUTH MIDDLEWARE] Authenticated request by user: %s, Path: %s", claims.Username, r.URL.Path)
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
