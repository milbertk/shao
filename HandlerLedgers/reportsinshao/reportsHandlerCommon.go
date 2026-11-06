package reportsinshao

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// =====================================================================
//  Shared helpers for the reports handlers (read-only GET endpoints).
// =====================================================================

type RPResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeRPResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(RPResponse{Status: http.StatusText(status), Message: message, Result: result})
}

func rpErrorStatus(message string) int {
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

// queryLimit reads a positive ?limit= value, or returns the fallback.
func queryLimit(r *http.Request, fallback int) int {
	v := strings.TrimSpace(r.URL.Query().Get("limit"))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	if n > 500 {
		n = 500
	}
	return n
}
