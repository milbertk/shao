package assistantinshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/milbertk/shao/bussunessLedger/blassistant"
)

// maxAssistantBodyBytes protects the endpoint from very large bodies.
const maxAssistantBodyBytes = 16 * 1024

type AssistantHandlerResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeAssistantResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(AssistantHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

func assistantErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch {
	case errors.Is(err, blassistant.ErrConversationNotFound):
		return http.StatusNotFound
	case errors.Is(err, blassistant.ErrAssistantNotInPlan):
		return http.StatusForbidden
	case errors.Is(err, blassistant.ErrDailyLimitReached),
		errors.Is(err, blassistant.ErrMonthlyCapReached):
		return http.StatusTooManyRequests
	case errors.Is(err, blassistant.ErrAssistantUnavailable):
		return http.StatusServiceUnavailable
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "is required"),
		strings.Contains(message, "invalid"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func decodeAssistantInput(w http.ResponseWriter, r *http.Request) (blassistant.AskInput, error) {
	var item blassistant.AskInput
	if r.Body == nil {
		return item, errors.New("request body is required")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAssistantBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&item); err != nil {
		return item, fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return item, errors.New("body must contain a single JSON object")
	}
	return item, nil
}
