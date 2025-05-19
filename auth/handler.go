package auth

import (
	"encoding/json"
	"net/http"

	"rubxy/config"
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

		if req.Username != "admin" || req.Password != "password" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

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
