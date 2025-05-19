package auth

import (
	"fmt"
	"time"

	"rubxy/config"
	"rubxy/db"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken returns the signed token string and its expiration time
func GenerateToken(username string, cfg *config.Config, isRefresh bool) (string, time.Time, error) {
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
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}

	// Store refresh token in DB
	if isRefresh {
		err := db.SaveRefreshToken(signedToken, username, claims.ExpiresAt.Time)
		if err != nil {
			return "", time.Time{}, err
		}
	}

	return signedToken, claims.ExpiresAt.Time, nil
}

// ValidateToken validates token string and returns claims
func ValidateToken(tokenStr string, cfg *config.Config, isRefresh bool) (*Claims, error) {
	claims := &Claims{}
	secret := cfg.AccessSecret
	if isRefresh {
		secret = cfg.RefreshSecret
	}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	if isRefresh {
		exists, err := db.CheckRefreshTokenExists(tokenStr)
		if err != nil || !exists {
			return nil, fmt.Errorf("refresh token revoked or not found")
		}
	}

	return claims, nil
}
