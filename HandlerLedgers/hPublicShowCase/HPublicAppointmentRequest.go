package publicshowcaseinshao

import (
	"encoding/json"
	"errors"

	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blpublicshowcase "github.com/milbertk/shao/bussunessLedger/bPublicShowcase"
)

// HPublicAppointmentRequest saves a guest appointment request (PENDING) and
// sends a notification email. PUBLIC (no authentication). The company comes
// from the URL, not from the body.
//
//	POST /public/companies/{companyId}/appointment-requests
//
// Body: PublicAppointmentRequest JSON { serviceId, customerName, phone, email,
//
//	socialLink, appointmentDate, startHour, endHour, comments }.
func HPublicAppointmentRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HPublicAppointmentRequest] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeShowcaseResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	companyID := strings.TrimSpace(mux.Vars(r)["companyId"])
	if companyID == "" {
		writeShowcaseResponse(w, http.StatusBadRequest, "companyId is required", nil)
		return
	}

	if r.Body == nil {
		writeShowcaseResponse(w, http.StatusBadRequest, "request body is required", nil)
		return
	}

	// Spam / email-flood protection (2026-10-25, security S4): body size,
	// per-IP and per-company daily limits, before anything is saved or sent.
	r.Body = http.MaxBytesReader(w, r.Body, maxPublicFormBodyBytes)
	if !checkPublicFormLimits(w, r, blpublicshowcase.PublicFormAppointment, companyID) {
		return
	}
	// CAPTCHA (Cloudflare Turnstile, 2026-10-25): bots cannot send the form.
	if !checkPublicCaptcha(w, r, blpublicshowcase.PublicFormAppointment, companyID) {
		return
	}

	var req blpublicshowcase.PublicAppointmentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeShowcaseResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeShowcaseResponse(w, http.StatusBadRequest, "body must contain a single JSON object", nil)
		return
	}

	result, err := blpublicshowcase.BCreatePublicAppointmentRequest(companyID, req)
	if errors.Is(err, blpublicshowcase.ErrSlotTaken) {
		// The time is not free anymore: the store reloads the available times.
		log.Printf("[HPublicAppointmentRequest] Slot taken: companyId=%s serviceId=%s date=%s %s-%s",
			companyID, req.ServiceID, req.AppointmentDate, req.StartHour, req.EndHour)
		handlertracking.RecordPublic(r, "PublicAppointmentRequestSlotTaken", companyID, handlertracking.Details{
			"serviceId": req.ServiceID, "date": req.AppointmentDate, "startHour": req.StartHour, "endHour": req.EndHour,
		})
		writeShowcaseResponse(w, http.StatusConflict,
			"This time was just taken. Please choose another time.", map[string]string{"code": "SLOT_TAKEN"})
		return
	}
	if errors.Is(err, blpublicshowcase.ErrServiceNotBookable) {
		writeShowcaseResponse(w, http.StatusBadRequest,
			"This service does not accept online bookings.", map[string]string{"code": "SERVICE_NOT_BOOKABLE"})
		return
	}
	if err != nil {
		message := err.Error()
		status := http.StatusInternalServerError
		if low := strings.ToLower(message); isPublicFormInvalid(err) || strings.Contains(low, "is required") || strings.HasPrefix(low, "invalid") {
			status = http.StatusBadRequest
		} else if isConflictLike(message) {
			status = http.StatusConflict
		} else {
			message = "An internal error occurred while saving the request"
		}
		log.Printf("[HPublicAppointmentRequest] Failed: status=%d error=%v", status, err)
		writeShowcaseResponse(w, status, message, nil)
		return
	}

	log.Printf("[HPublicAppointmentRequest] Saved OK: appointmentId=%s", result.AppointmentID)
	// Best-effort action tracking (never blocks the response).
	handlertracking.RecordPublic(r, "PublicAppointmentRequest", companyID, handlertracking.Details{"appointmentId": result.AppointmentID})
	writeShowcaseResponse(w, http.StatusOK, "Your appointment request was sent.", result)
}

func isConflictLike(message string) bool {
	m := strings.ToLower(message)
	return strings.Contains(m, "duplicate") ||
		strings.Contains(m, "unique constraint") ||
		strings.Contains(m, "overlap")
}
