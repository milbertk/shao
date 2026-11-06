package appointmentsinshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	blappointment "github.com/milbertk/shao/bussunessLedger/blappontment"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
)

type AppointmentHandlerResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeAppointmentResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(AppointmentHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

func appointmentErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, blappointment.ErrAppointmentNotFound) {
		return http.StatusNotFound
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "duplicate"),
		strings.Contains(message, "unique constraint"),
		strings.Contains(message, "already exists"):
		return http.StatusConflict
	case strings.Contains(message, "is required"),
		strings.Contains(message, "invalid"),
		strings.Contains(message, "violates check"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func decodeSingleAppointment(r *http.Request) (appointment.AppointmentS, error) {
	var item appointment.AppointmentS
	if r.Body == nil {
		return item, errors.New("request body is required")
	}
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
