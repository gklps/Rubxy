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
	ActivityID []string `json:"activity_id"`
	UserDID    string   `json:"user_did"`
	AdminDID   string   `json:"admin_did"`
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

// sendErrorResponse sends an error response using the FinalResponse format
func sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := FinalResponse{
		Status:  false,
		Message: message,
		Result:  nil,
	}

	// Encode to buffer first to handle errors before writing to response
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(errorResp); err != nil {
		logger.ErrorLogger.Printf("Failed to encode error response: %v", err)
		// Fallback to plain text if JSON encoding fails
		http.Error(w, message, statusCode)
		return
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		logger.ErrorLogger.Printf("Failed to write error response: %v", err)
	}
}

func HandleAdminActivityAdd(w http.ResponseWriter, r *http.Request) {
	var activityReq ActivityAddRequest
	if err := json.NewDecoder(r.Body).Decode(&activityReq); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	reqBody, err := json.Marshal(activityReq)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to marshal request")
		return
	}

	resp, err := http.Post("http://localhost:9000/api/activity/add", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		sendErrorResponse(w, http.StatusBadGateway, "Failed to forward request")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to read response")
		return
	}

	var transferResp TransferResponse
	if err := json.Unmarshal(body, &transferResp); err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to parse response")
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
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to encode final response")
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
		sendErrorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	logger.InfoLogger.Printf("[ADMIN PAYOUTS] Request body: %s", string(bodyBytes))

	// Recreate body for JSON decoder
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// First decode into a map to validate activity_id is an array
	var rawPayload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&rawPayload); err != nil {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] Failed to decode request body: %v", err)
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate that activity_id is an array, not a string
	activityIDRaw, exists := rawPayload["activity_id"]
	if !exists {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] activity_id field is missing")
		sendErrorResponse(w, http.StatusBadRequest, "activity_id field is required")
		return
	}

	// Check if activity_id is a string (which we don't accept)
	if _, isString := activityIDRaw.(string); isString {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] activity_id must be an array, not a string")
		sendErrorResponse(w, http.StatusBadRequest, "activity_id must be an array, not a string")
		return
	}

	// Check if activity_id is an array
	activityIDArray, isArray := activityIDRaw.([]interface{})
	if !isArray {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] activity_id must be an array")
		sendErrorResponse(w, http.StatusBadRequest, "activity_id must be an array")
		return
	}

	// Validate that the array is not empty
	if len(activityIDArray) == 0 {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] activity_id array cannot be empty")
		sendErrorResponse(w, http.StatusBadRequest, "activity_id array cannot be empty")
		return
	}

	// Convert []interface{} to []string
	activityIDs := make([]string, 0, len(activityIDArray))
	for i, item := range activityIDArray {
		activityIDStr, ok := item.(string)
		if !ok {
			logger.ErrorLogger.Printf("[ADMIN PAYOUTS] activity_id[%d] must be a string", i)
			sendErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("activity_id[%d] must be a string", i))
			return
		}
		activityIDs = append(activityIDs, activityIDStr)
	}

	// Now decode into the struct with validated data
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	var reqPayload RewardTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&reqPayload); err != nil {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] Failed to decode request body: %v", err)
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	logger.InfoLogger.Printf("[ADMIN PAYOUTS] Parsed payload - ActivityID: %v, UserDID: %s, AdminDID: %s",
		reqPayload.ActivityID, reqPayload.UserDID, reqPayload.AdminDID)

	// Marshal the payload to JSON
	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to marshal request")
		return
	}

	// Send POST request to the external API
	logger.InfoLogger.Printf("[ADMIN PAYOUTS] Forwarding request to: http://localhost:9000/api/rewards/transfer")
	resp, err := http.Post("http://localhost:9000/api/rewards/transfer", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] Failed to call external API: %v", err)
		sendErrorResponse(w, http.StatusBadGateway, "Failed to call external API")
		return
	}
	defer resp.Body.Close()

	logger.InfoLogger.Printf("[ADMIN PAYOUTS] External API response status: %d", resp.StatusCode)

	// Read and parse the response
	var apiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		logger.ErrorLogger.Printf("[ADMIN PAYOUTS] Failed to parse external API response: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to parse external API response")
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
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to encode final response")
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
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read file: %v", err))
		return
	}

	var activities []ActivityData
	if err := json.Unmarshal(file, &activities); err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse JSON: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Encode to buffer first to handle errors before writing to response
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(activities); err != nil {
		logger.ErrorLogger.Printf("[ADMIN ACTIVITY LIST] Failed to encode response: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		logger.ErrorLogger.Printf("[ADMIN ACTIVITY LIST] Failed to write response: %v", err)
	}
}

func HandleAdminAddUser(w http.ResponseWriter, r *http.Request) {
	var req AdminAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to marshal request")
		return
	}

	resp, err := http.Post("http://localhost:9000/api/admin/add", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		sendErrorResponse(w, http.StatusBadGateway, "Failed to forward request")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to read response")
		return
	}

	// Parse the initial response
	var transferResp TransferResponse
	if err := json.Unmarshal(body, &transferResp); err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to parse response")
		return
	}

	// Now parse the `data` string field (which is an escaped JSON string)
	var sctData SCTData
	if err := json.Unmarshal([]byte(transferResp.Data), &sctData); err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to parse inner JSON from data")
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
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to encode final response")
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
		sendErrorResponse(w, http.StatusBadRequest, "user_did is required")
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
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to create request")
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
		sendErrorResponse(w, http.StatusBadGateway, "Failed to forward request")
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
