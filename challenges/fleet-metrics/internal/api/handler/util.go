package handler

import (
	"encoding/json"
	"net/http"
)

// writeError writes a JSON error response with the given HTTP status code and message.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"msg": msg})
}
