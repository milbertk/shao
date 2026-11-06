package appointmentrequestinshao

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

	blappointmentrequest "github.com/milbertk/shao/bussunessLedger/blappointmentrequest"
)

// HApproveAppointmentRequest: PENDING -> SCHEDULED + email.
//
//	POST /companies/{companyId}/appointment-requests/{appointmentId}/approve
func HApproveAppointmentRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeARResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}
	companyID, actorUserID, actorEmail, ok := arAuthScope(w, r)
	if !ok {
		return
	}
	appointmentID := strings.TrimSpace(mux.Vars(r)["appointmentId"])
	if appointmentID == "" {
		writeARResponse(w, http.StatusBadRequest, "appointmentId is required", nil)
		return
	}

	result, err := blappointmentrequest.BApproveAppointmentRequest(companyID, appointmentID)
	if err != nil {
		status := arErrorStatus(err.Error())
		msg := err.Error()
		if status == http.StatusInternalServerError {
			msg = "An internal error occurred while approving the request"
		}
		log.Printf("[HApproveAppointmentRequest] Failed: status=%d error=%v", status, err)
		writeARResponse(w, status, msg, nil)
		return
	}
	insertARTracking(r, actorUserID, actorEmail, companyID, appointmentID, "ApproveAppointmentRequest")
	writeARResponse(w, http.StatusOK, "Appointment request approved.", result)
}

type rejectBody struct {
	Reason string `json:"reason"`
}

// HRejectAppointmentRequest: PENDING -> REJECTED (+ optional reason email).
//
//	POST /companies/{companyId}/appointment-requests/{appointmentId}/reject
func HRejectAppointmentRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeARResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}
	companyID, actorUserID, actorEmail, ok := arAuthScope(w, r)
	if !ok {
		return
	}
	appointmentID := strings.TrimSpace(mux.Vars(r)["appointmentId"])
	if appointmentID == "" {
		writeARResponse(w, http.StatusBadRequest, "appointmentId is required", nil)
		return
	}

	reason := ""
	if r.Body != nil {
		var body rejectBody
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil && err != io.EOF {
			writeARResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
			return
		}
		reason = strings.TrimSpace(body.Reason)
	}

	result, err := blappointmentrequest.BRejectAppointmentRequest(companyID, appointmentID, reason)
	if err != nil {
		status := arErrorStatus(err.Error())
		msg := err.Error()
		if status == http.StatusInternalServerError {
			msg = "An internal error occurred while rejecting the request"
		}
		log.Printf("[HRejectAppointmentRequest] Failed: status=%d error=%v", status, err)
		writeARResponse(w, status, msg, nil)
		return
	}
	insertARTracking(r, actorUserID, actorEmail, companyID, appointmentID, "RejectAppointmentRequest")
	writeARResponse(w, http.StatusOK, "Appointment request rejected.", result)
}

func arAuthScope(w http.ResponseWriter, r *http.Request) (string, string, string, bool) {
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeARResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return "", "", "", false
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeARResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return "", "", "", false
	}
	actorEmail, _ := security.GetActiveCompanyEmail(r)
	return companyID, strings.TrimSpace(claims.UID), actorEmail, true
}

func insertARTracking(r *http.Request, userID, email, companyID, appointmentID, action string) {
	payload := struct {
		CompanyID     string `json:"companyId"`
		AppointmentID string `json:"appointmentId"`
	}{companyID, appointmentID}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[%s] Error creating tracking payload: %v", action, err)
		return
	}
	lt := tracking.LoginTracking{
		UserID:         userID,
		Email:          strings.TrimSpace(email),
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             readARClientIP(r),
		Platform:       "Shao",
		MacAddress:     "0",
		Browser:        r.UserAgent(),
		CountryCode:    "",
		GMTTime:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		Lang:           r.Header.Get("Accept-Language"),
		Action:         action,
		Jsonstring:     string(jsonPayload),
	}
	if err := lt.Insert(); err != nil {
		log.Printf("[%s] Error inserting tracking: %v", action, err)
	}
}
