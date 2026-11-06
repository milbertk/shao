package trackinginshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	bltracking "github.com/milbertk/shao/bussunessLedger/bltracking"
)

// TrackingHandlerResponse is the standard JSON envelope for tracking handlers.
type TrackingHandlerResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeTrackingResponse(
	w http.ResponseWriter,
	status int,
	message string,
	result interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(TrackingHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// trackingErrorStatus maps a business error to an HTTP status code.
func trackingErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, bltracking.ErrTrackingNotFound) {
		return http.StatusNotFound
	}

	message := strings.ToLower(err.Error())

	switch {
	case strings.Contains(message, "duplicate"),
		strings.Contains(message, "unique constraint"),
		strings.Contains(message, "already exists"):
		return http.StatusConflict

	case strings.Contains(message, "is required"),
		strings.Contains(message, "invalid"):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

// decodeSingleTrackingBundle decodes exactly one JSON object from the request
// body into a TrackingBundle { config, steps }, rejecting unknown fields and
// any trailing content.
func decodeSingleTrackingBundle(r *http.Request) (bltracking.TrackingBundle, error) {
	var bundle bltracking.TrackingBundle

	if r.Body == nil {
		return bundle, errors.New("request body is required")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&bundle); err != nil {
		return bundle, fmt.Errorf("invalid JSON body: %w", err)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return bundle, errors.New("body must contain a single JSON object")
	}

	return bundle, nil
}
