package appointmentsinshao

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/tracking"

	blappointment "github.com/milbertk/shao/bussunessLedger/blappontment"
)

// HDeleteAppointment hard-deletes one appointment, scoped to the company.
//
//	DELETE /companies/{companyId}/appointments/{appointmentId}
func HDeleteAppointment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		writeAppointmentResponse(w, http.StatusMethodNotAllowed, "Use DELETE", nil)
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

	routeVariables := mux.Vars(r)
	appointmentID := strings.TrimSpace(routeVariables["appointmentId"])
	if appointmentID == "" {
		writeAppointmentResponse(w, http.StatusBadRequest, "appointmentId is required", nil)
		return
	}

	err := blappointment.BDeleteAppointment(companyID, appointmentID)
	if err != nil {
		status := appointmentErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while deleting the appointment"
		}
		log.Printf("[HDeleteAppointment] Business operation failed: status=%d error=%v", status, err)
		writeAppointmentResponse(w, status, message, nil)
		return
	}

	// Record the deletion (best-effort login tracking).
	actorEmail, _ := security.GetActiveCompanyEmail(r)
	actorRole, _ := security.GetActiveCompanyRole(r)
	insertAppointmentDeleteTracking(
		r,
		strings.TrimSpace(claims.UID),
		actorEmail,
		companyID,
		actorRole,
		appointmentID,
	)

	writeAppointmentResponse(w, http.StatusOK, "Appointment deleted successfully.", nil)
}

// insertAppointmentDeleteTracking records a delete action into the login
// tracking table. Best-effort.
func insertAppointmentDeleteTracking(
	r *http.Request,
	actorUserID string,
	actorEmail string,
	activeCompanyID string,
	activeCompanyRole string,
	appointmentID string,
) {
	trackingPayload := struct {
		CompanyID        string `json:"companyId"`
		ActorCompanyRole string `json:"actorCompanyRole"`
		AppointmentID    string `json:"appointmentId"`
	}{
		CompanyID:        activeCompanyID,
		ActorCompanyRole: activeCompanyRole,
		AppointmentID:    appointmentID,
	}

	jsonTrackingPayload, err := json.Marshal(trackingPayload)
	if err != nil {
		log.Printf("[HDeleteAppointment] Error creating tracking payload: %v", err)
		return
	}

	lt := tracking.LoginTracking{
		UserID:         actorUserID,
		Email:          strings.TrimSpace(actorEmail),
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             readAppointmentClientIP(r),
		Platform:       "Shao",
		MacAddress:     "0",
		Browser:        r.UserAgent(),
		CountryCode:    "",
		GMTTime:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		Lang:           r.Header.Get("Accept-Language"),
		Action:         "DeleteAppointment",
		Jsonstring:     string(jsonTrackingPayload),
	}

	if err := lt.Insert(); err != nil {
		log.Printf("[HDeleteAppointment] Error inserting tracking: %v", err)
	}
}
