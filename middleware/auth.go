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
				logger.InfoLogger.Println("Unauthorized access attempt")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, claims.Username)
			logger.InfoLogger.Printf("Authenticated request by user: %s", claims.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserFromContext(r *http.Request) string {
	user, _ := r.Context().Value(userContextKey).(string)
	return user
}
