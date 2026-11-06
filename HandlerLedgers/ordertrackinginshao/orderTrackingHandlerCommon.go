package ordertrackinginshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	blordertracking "github.com/milbertk/shao/bussunessLedger/blordertracking"
)

type OTResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(OTResponse{Status: http.StatusText(status), Message: message, Result: result})
}

func errorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, blordertracking.ErrStageNotFound) {
		return http.StatusNotFound
	}
	m := strings.ToLower(err.Error())
	switch {
	case strings.Contains(m, "duplicate"), strings.Contains(m, "unique constraint"):
		return http.StatusConflict
	case strings.Contains(m, "is required"), strings.Contains(m, "invalid"), strings.Contains(m, "violates check"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func decodeSingleStageBundle(r *http.Request) (blordertracking.StageBundle, error) {
	var b blordertracking.StageBundle
	if r.Body == nil {
		return b, errors.New("request body is required")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		return b, fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return b, errors.New("body must contain a single JSON object")
	}
	return b, nil
}

func readClientIP(r *http.Request) string {
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
