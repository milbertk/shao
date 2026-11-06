package bookinginshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	blbooking "github.com/milbertk/shao/bussunessLedger/bbooking"
)

// BookingHandlerResponse is the standard JSON envelope for booking handlers.
// Result holds a *blbooking.BookingBundle, a []booking.BookingConfigS, etc.
type BookingHandlerResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeBookingResponse(
	w http.ResponseWriter,
	status int,
	message string,
	result interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(BookingHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// bookingErrorStatus maps a business error to an HTTP status code.
func bookingErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, blbooking.ErrBookingNotFound) {
		return http.StatusNotFound
	}

	message := strings.ToLower(err.Error())

	switch {
	case strings.Contains(message, "duplicate"),
		strings.Contains(message, "unique constraint"),
		strings.Contains(message, "already exists"),
		strings.Contains(message, "overlap"),
		strings.Contains(message, "conflicting key"):
		return http.StatusConflict

	case strings.Contains(message, "does not exist in this company"),
		strings.Contains(message, "is required"),
		strings.Contains(message, "invalid"),
		strings.Contains(message, "violates check"):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

// decodeSingleBookingBundle decodes exactly one JSON object from the request
// body into a BookingBundle, rejecting unknown fields and trailing content.
func decodeSingleBookingBundle(r *http.Request) (blbooking.BookingBundle, error) {
	var bundle blbooking.BookingBundle

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
