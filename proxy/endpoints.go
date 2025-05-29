package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type ActivityAddRequest struct {
	ActivityID   string `json:"activity_id"`
	RewardPoints int    `json:"reward_points"`
	AdminDID     string `json:"admin_did"`
}

type RewardTransferRequest struct {
	ActivityID string `json:"activity_id"`
	UserDID    string `json:"user_did"`
	AdminDID   string `json:"admin_did"`
}
type ActivityData struct {
	ActivityID   string `json:"activity_id"`
	BlockHash    string `json:"block_hash"`
	RewardPoints int    `json:"reward_points"`
}

type TransferResponse struct {
	Data    string `json:"data"`
	Message string `json:"message"`
}

type SCTData struct {
	Status       bool            `json:"status"`
	Message      string          `json:"message"`
	Result       interface{}     `json:"result"`
	SCTDataReply json.RawMessage `json:"SCTDataReply"`
}

type FinalResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

func HandleAdminActivityAdd(w http.ResponseWriter, r *http.Request) {
	var activityReq ActivityAddRequest
	if err := json.NewDecoder(r.Body).Decode(&activityReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	reqBody, err := json.Marshal(activityReq)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	resp, err := http.Post("http://localhost:9000/api/activity/add", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		http.Error(w, "Failed to forward request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var transferResp TransferResponse
	if err := json.Unmarshal(body, &transferResp); err != nil {
		http.Error(w, "Failed to parse response", http.StatusInternalServerError)
		return
	}

	var sctData SCTData
	if err := json.Unmarshal([]byte(transferResp.Data), &sctData); err != nil {
		// if 'data' is null or invalid, return a fallback error
		finalResp := FinalResponse{
			Status:  false,
			Message: transferResp.Message,
			Result:  nil,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(finalResp)
		return
	}

	finalResp := FinalResponse{
		Status:  sctData.Status,
		Message: "Activity added successfully",
		Result:  sctData.SCTDataReply,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(finalResp)
}

func HandleAdminRewardTransfer(w http.ResponseWriter, r *http.Request) {
	var reqPayload RewardTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&reqPayload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Marshal the payload to JSON
	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	// Send POST request to the external API
	resp, err := http.Post("http://localhost:9000/api/rewards/transfer", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, "Failed to call external API", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read and parse the response
	var apiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		http.Error(w, "Failed to parse external API response", http.StatusInternalServerError)
		return
	}

	// Prepare the final response
	finalResp := map[string]interface{}{
		"status":  true,
		"message": apiResp["message"],
		"result":  apiResp["data"],
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

func HandleGetAllActivities(w http.ResponseWriter, r *http.Request) {
	const filePath = "/home/rubix/github/ymca-wellness-cafe/dappServer/test.json"

	file, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	var activities []ActivityData
	if err := json.Unmarshal(file, &activities); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse JSON: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(activities); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
