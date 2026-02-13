package handlers

import (
	"encoding/json"
	"net/http"

	"napkin-backend/lib/responses"
)

// HealthHandler returns the server health status
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	response := responses.HealthResponse{
		Status:  "ok",
		Message: "Server is running",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HelloHandler returns a greeting message
func HelloHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "Hello from Napkin Backend!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
