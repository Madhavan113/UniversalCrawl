package api

import (
	"encoding/json"
	"net/http"
)

type successResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

type errorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeSuccess writes a successful JSON response.
func writeSuccess(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: data})
}

// writeError writes an error JSON response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Success: false, Error: msg})
}
