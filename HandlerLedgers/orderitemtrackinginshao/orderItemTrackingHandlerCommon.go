package orderitemtrackinginshao

import (
	"encoding/json"
	"net/http"
	"strings"
)

type OITResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeOITResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(OITResponse{Status: http.StatusText(status), Message: message, Result: result})
}

func oitErrorStatus(message string) int {
	m := strings.ToLower(message)
	switch {
	case strings.Contains(m, "not found"):
		return http.StatusNotFound
	case strings.Contains(m, "is required"), strings.Contains(m, "invalid"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func readOITClientIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}
	return strings.TrimSpace(r.RemoteAddr)
}
