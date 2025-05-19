package auth

import (
	"encoding/json"
	"net/http"
	"rubxy/config"
	"rubxy/db"
	"rubxy/logger"
	"rubxy/users"
)

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func HandleToken(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if !users.Authenticate(req.Username, req.Password) {
			logger.InfoLogger.Printf("Failed login attempt: %s", req.Username)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		logger.InfoLogger.Printf("Successful login for user: %s", req.Username)

		accessToken, _, err := GenerateToken(req.Username, cfg, false)
		if err != nil {
			http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
			return
		}

		refreshToken, expiresAt, err := GenerateToken(req.Username, cfg, true)
		if err != nil {
			http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
			return
		}

		// Store refresh token in DB
		err = db.SaveRefreshToken(refreshToken, req.Username, expiresAt)
		if err != nil {
			logger.ErrorLogger.Printf("Failed to insert refresh token: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(TokenResponse{AccessToken: accessToken, RefreshToken: refreshToken})
	}
}

func HandleRefresh(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check if refresh token is valid and not revoked or expired
		valid, err := db.IsRefreshTokenValid(req.RefreshToken)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if !valid {
			http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
			return
		}

		claims, err := ValidateToken(req.RefreshToken, cfg, true)
		if err != nil {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		accessToken, _, err := GenerateToken(claims.Username, cfg, false)
		if err != nil {
			http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(TokenResponse{AccessToken: accessToken})
	}
}

func HandleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := users.Register(req.Username, req.Password); err != nil {
			logger.ErrorLogger.Printf("Registration failed for user %s: %v", req.Username, err)
			http.Error(w, "Registration failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "User registered"})
	}
}

func HandleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		err := db.RevokeRefreshToken(req.RefreshToken)
		if err != nil {
			logger.ErrorLogger.Printf("Logout failed: %v", err)
			http.Error(w, "Failed to logout", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"message": "Logged out"})
	}
}
