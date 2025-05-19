package auth

import (
	"time"

	"rubxy/config"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GenerateToken(username string, cfg *config.Config, isRefresh bool) (string, error) {
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.AccessTTL)),
		},
	}
	if isRefresh {
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(cfg.RefreshTTL))
	}

	secret := cfg.AccessSecret
	if isRefresh {
		secret = cfg.RefreshSecret
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
func ValidateToken(tokenStr string, cfg *config.Config, isRefresh bool) (*Claims, error) {
	claims := &Claims{}
	secret := cfg.AccessSecret
	if isRefresh {
		secret = cfg.RefreshSecret
	}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	return claims, err
}
