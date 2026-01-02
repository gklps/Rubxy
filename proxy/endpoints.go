package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"rubxy/logger"

	"github.com/go-chi/chi/v5"
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
type AdminAddRequest struct {
	NewAdminDID      string `json:"new_admin_did"`
	ExistingAdminDID string `json:"existing_admin_did"`
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

	// Encode to buffer first to handle errors before writing to response
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(finalResp); err != nil {
		logger.ErrorLogger.Printf("[ADMIN ACTIVITY ADD] Failed to encode final response: %v", err)
		http.Error(w, "Failed to encode final response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(buf.Bytes()); err != nil {
		logger.ErrorLogger.Printf("[ADMIN ACTIVITY ADD] Failed to write response: %v", err)
	}
}

func HandleAdminRewardTransfer(w http.ResponseWriter, r *http.Request) {
	// Log incoming request
	logger.InfoLogger.Printf("[ADMIN PAYOUTS] Incoming request - Method: %s, Path: %s, RemoteAddr: %s", r.Method, r.URL.Path, r.RemoteAddr)
	logger.InfoLogger.Printf("[ADMIN PAYOUTS] Headers: %v", r.Header)

	// Read body for logging (we'll need to recreate it for decoding)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	logger.InfoLogger.Printf("[ADMIN PAYOUTS] Request body: %s", string(bodyBytes))

	// Recreate body for JSON decoder
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var reqPayload RewardTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&reqPayload); err != nil {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logger.InfoLogger.Printf("[ADMIN PAYOUTS] Parsed payload - ActivityID: %s, UserDID: %s, AdminDID: %s",
		reqPayload.ActivityID, reqPayload.UserDID, reqPayload.AdminDID)

	// Marshal the payload to JSON
	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	// Send POST request to the external API
	logger.InfoLogger.Printf("[ADMIN PAYOUTS] Forwarding request to: http://localhost:9000/api/rewards/transfer")
	resp, err := http.Post("http://localhost:9000/api/rewards/transfer", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] Failed to call external API: %v", err)
		http.Error(w, "Failed to call external API", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	logger.InfoLogger.Printf("[ADMIN PAYOUTS] External API response status: %d", resp.StatusCode)

	// Read and parse the response
	var apiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] Failed to parse external API response: %v", err)
		http.Error(w, "Failed to parse external API response", http.StatusInternalServerError)
		return
	}

	logger.InfoLogger.Printf("[ADMIN PAYOUTS] External API response: %+v", apiResp)

	// Prepare the final response
	finalResp := map[string]interface{}{
		"status":  true,
		"message": apiResp["message"],
		"result":  apiResp["data"],
	}

	logger.InfoLogger.Printf("[ADMIN PAYOUTS] Sending final response: %+v", finalResp)
	
	// Encode to buffer first to handle errors before writing to response
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(finalResp); err != nil {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] Failed to encode final response: %v", err)
		http.Error(w, "Failed to encode final response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(buf.Bytes()); err != nil {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] Failed to write response: %v", err)
	}
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

func HandleAdminAddUser(w http.ResponseWriter, r *http.Request) {
	var req AdminAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	resp, err := http.Post("http://localhost:9000/api/admin/add", "application/json", bytes.NewBuffer(reqBody))
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

	// Parse the initial response
	var transferResp TransferResponse
	if err := json.Unmarshal(body, &transferResp); err != nil {
		http.Error(w, "Failed to parse response", http.StatusInternalServerError)
		return
	}

	// Now parse the `data` string field (which is an escaped JSON string)
	var sctData SCTData
	if err := json.Unmarshal([]byte(transferResp.Data), &sctData); err != nil {
		http.Error(w, "Failed to parse inner JSON from data", http.StatusInternalServerError)
		return
	}

	// Final clean response
	finalResp := FinalResponse{
		Status:  sctData.Status,
		Message: sctData.Message,
		Result:  sctData.SCTDataReply,
	}

	// Encode to buffer first to handle errors before writing to response
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(finalResp); err != nil {
		logger.ErrorLogger.Printf("[ADMIN ADD USER] Failed to encode final response: %v", err)
		http.Error(w, "Failed to encode final response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(buf.Bytes()); err != nil {
		logger.ErrorLogger.Printf("[ADMIN ADD USER] Failed to write response: %v", err)
	}
}

func HandleUserPayouts(w http.ResponseWriter, r *http.Request) {
	// Extract user_did from URL path
	userDID := chi.URLParam(r, "user_did")
	if userDID == "" {
		http.Error(w, "user_did is required", http.StatusBadRequest)
		return
	}

	// Build the target URL with proper query encoding
	targetURL := fmt.Sprintf("http://localhost:20000/api/get-ft-info-by-did?did=%s", url.QueryEscape(userDID))

	// Create a new request to the target server
	// GET requests should not have a body per HTTP semantics
	var body io.Reader
	if r.Method != "GET" && r.Method != "HEAD" {
		body = r.Body
	}
	req, err := http.NewRequest(r.Method, targetURL, body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers from original request
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make the request to the target server
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to forward request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		logger.ErrorLogger.Printf("Failed to copy response body: %v", err)
	}
}
