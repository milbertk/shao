package orderdetailsinshao

import (
	"encoding/json"
	"net/http"
	"strings"
)

type ODResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeODResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ODResponse{Status: http.StatusText(status), Message: message, Result: result})
}

func odErrorStatus(message string) int {
	m := strings.ToLower(message)
	switch {
	case strings.Contains(m, "not found"):
		return http.StatusNotFound
	case strings.Contains(m, "is required"), strings.Contains(m, "needs a"), strings.Contains(m, "no attachments"), strings.Contains(m, "invalid"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func readODClientIP(r *http.Request) string {
	fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	if rip := strings.TrimSpace(r.Header.Get("X-Real-IP")); rip != "" {
		return rip
	}
	return strings.TrimSpace(r.RemoteAddr)
}
