package auth

import (
	"encoding/json"
	"net/http"
	"rubxy/config"
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
		json.NewDecoder(r.Body).Decode(&req)

		if !users.Authenticate(req.Username, req.Password) {
			logger.InfoLogger.Printf("Failed login attempt: %s", req.Username)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		logger.InfoLogger.Printf("Successful login for user: %s", req.Username)
		access, _ := GenerateToken(req.Username, cfg, false)
		refresh, _ := GenerateToken(req.Username, cfg, true)

		json.NewEncoder(w).Encode(TokenResponse{AccessToken: access, RefreshToken: refresh})
	}
}

func HandleRefresh(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		claims, err := ValidateToken(req.RefreshToken, cfg, true)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		access, _ := GenerateToken(claims.Username, cfg, false)
		json.NewEncoder(w).Encode(TokenResponse{AccessToken: access})
	}
}

func HandleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		json.NewDecoder(r.Body).Decode(&req)

		err := users.Register(req.Username, req.Password)
		if err != nil {
			logger.ErrorLogger.Printf("Registration failed for %s: %v", req.Username, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "User registered"})
	}
}
