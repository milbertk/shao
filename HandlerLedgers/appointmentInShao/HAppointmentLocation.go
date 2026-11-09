package appointmentsinshao

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blappointment "github.com/milbertk/shao/bussunessLedger/blappontment"
	"github.com/milbertk/shao/bussunessLedger/bllocations"
)

type appointmentLocationBody struct {
	LocationID string `json:"locationId"` // "" = no location
}

// HSetAppointmentLocation changes only the location of an appointment or a
// booking request (2026-11-08).
//
//	PUT /companies/{companyId}/appointments/{appointmentId}/location
//	Body: { "locationId": "<uuid>" | "" }
func HSetAppointmentLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writeAppointmentResponse(w, http.StatusMethodNotAllowed, "Use PUT", nil)
		return
	}
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeAppointmentResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeAppointmentResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}
	appointmentID := strings.TrimSpace(mux.Vars(r)["appointmentId"])
	if appointmentID == "" {
		writeAppointmentResponse(w, http.StatusBadRequest, "appointmentId is required", nil)
		return
	}
	if r.Body == nil {
		writeAppointmentResponse(w, http.StatusBadRequest, "request body is required", nil)
		return
	}
	var body appointmentLocationBody
	dec := json.NewDecoder(io.LimitReader(r.Body, 4*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeAppointmentResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeAppointmentResponse(w, http.StatusBadRequest, "body must contain a single JSON object", nil)
		return
	}

	result, err := blappointment.BSetAppointmentLocation(companyID, appointmentID, body.LocationID)
	if err != nil {
		status := appointmentLocationErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while changing the location"
		}
		log.Printf("[HSetAppointmentLocation] Failed: status=%d error=%v", status, err)
		writeAppointmentResponse(w, status, message, nil)
		return
	}

	handlertracking.Record(r, "SetAppointmentLocation", handlertracking.Details{
		"appointmentId":      appointmentID,
		"locationId":         strings.TrimSpace(body.LocationID),
		"otherLocationCount": result.OtherLocationCount,
	})
	writeAppointmentResponse(w, http.StatusOK, "Appointment location updated.", result)
}

// appointmentLocationErrorStatus maps the business errors to HTTP codes.
func appointmentLocationErrorStatus(err error) int {
	switch {
	case errors.Is(err, blappointment.ErrAppointmentNotFound):
		return http.StatusNotFound
	case errors.Is(err, bllocations.ErrLocationNotFound),
		errors.Is(err, bllocations.ErrLocationDisabled),
		errors.Is(err, blappointment.ErrAppointmentClosed):
		return http.StatusBadRequest
	}
	return appointmentErrorStatus(err)
}
