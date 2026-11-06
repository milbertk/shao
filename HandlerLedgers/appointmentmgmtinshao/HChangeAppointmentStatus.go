package appointmentmgmtinshao

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/tracking"

	blappointmentmgmt "github.com/milbertk/shao/bussunessLedger/blappointmentmgmt"
)

type changeStatusBody struct {
	Status string `json:"status"` // DONE | CANCELLED | SCHEDULED
}

// HChangeAppointmentStatus finishes/cancels a real appointment.
//
//	POST /companies/{companyId}/appointments/{appointmentId}/status
//	Body: { status }
func HChangeAppointmentStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeAMResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeAMResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	actorUserID := strings.TrimSpace(claims.UID)

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeAMResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	appointmentID := strings.TrimSpace(mux.Vars(r)["appointmentId"])
	if appointmentID == "" {
		writeAMResponse(w, http.StatusBadRequest, "appointmentId is required", nil)
		return
	}

	if r.Body == nil {
		writeAMResponse(w, http.StatusBadRequest, "request body is required", nil)
		return
	}
	var body changeStatusBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeAMResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeAMResponse(w, http.StatusBadRequest, "body must contain a single JSON object", nil)
		return
	}

	result, err := blappointmentmgmt.BChangeAppointmentStatus(companyID, appointmentID, body.Status)
	if err != nil {
		status := amErrorStatus(err.Error())
		msg := err.Error()
		if status == http.StatusInternalServerError {
			msg = "An internal error occurred while changing the status"
		}
		log.Printf("[HChangeAppointmentStatus] Failed: status=%d error=%v", status, err)
		writeAMResponse(w, status, msg, nil)
		return
	}

	actorEmail, _ := security.GetActiveCompanyEmail(r)
	insertAMTracking(r, actorUserID, actorEmail, companyID, appointmentID, strings.ToUpper(strings.TrimSpace(body.Status)))

	writeAMResponse(w, http.StatusOK, "Appointment status updated.", result)
}

func insertAMTracking(r *http.Request, userID, email, companyID, appointmentID, newStatus string) {
	payload := struct {
		CompanyID     string `json:"companyId"`
		AppointmentID string `json:"appointmentId"`
		NewStatus     string `json:"newStatus"`
	}{companyID, appointmentID, newStatus}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[ChangeAppointmentStatus] Error creating tracking payload: %v", err)
		return
	}
	lt := tracking.LoginTracking{
		UserID:         userID,
		Email:          strings.TrimSpace(email),
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             readAMClientIP(r),
		Platform:       "Shao",
		MacAddress:     "0",
		Browser:        r.UserAgent(),
		CountryCode:    "",
		GMTTime:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		Lang:           r.Header.Get("Accept-Language"),
		Action:         "ChangeAppointmentStatus",
		Jsonstring:     string(jsonPayload),
	}
	if err := lt.Insert(); err != nil {
		log.Printf("[ChangeAppointmentStatus] Error inserting tracking: %v", err)
	}
}
