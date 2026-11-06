package appointmentsinshao

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/milbertk/security"
	"github.com/milbertk/tracking"

	blappointment "github.com/milbertk/shao/bussunessLedger/blappontment"
)

// HSaveAppointment creates or updates an appointment.
//
//	POST /companies/{companyId}/appointments
//
// The company comes from the validated context. The response result includes
// the saved appointment and a conflictCount (non-blocking warning).
func HSaveAppointment(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSaveAppointment] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeAppointmentResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
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

	item, err := decodeSingleAppointment(r)
	if err != nil {
		writeAppointmentResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	wasUpdate := strings.TrimSpace(item.AppointmentID) != ""

	log.Printf(
		"[HSaveAppointment] Saving: companyId=%s appointmentId=%q date=%q",
		companyID, item.AppointmentID, item.AppointmentDate,
	)

	result, err := blappointment.BSaveAppointment(companyID, item)
	if err != nil {
		status := appointmentErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the appointment"
		}
		log.Printf("[HSaveAppointment] Business operation failed: status=%d error=%v", status, err)
		writeAppointmentResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSaveAppointment] Saved OK: appointmentId=%s conflicts=%d",
		result.Appointment.AppointmentID, result.ConflictCount)

	// Record the successful action (best-effort login tracking).
	actorEmail, _ := security.GetActiveCompanyEmail(r)
	actorRole, _ := security.GetActiveCompanyRole(r)
	insertAppointmentLoginTracking(
		r,
		strings.TrimSpace(claims.UID),
		actorEmail,
		companyID,
		actorRole,
		wasUpdate,
		result,
	)

	writeAppointmentResponse(w, http.StatusOK, "Appointment saved successfully.", result)
}

// insertAppointmentLoginTracking records the successful save into the login
// tracking table. Best-effort: a tracking failure is logged but never undoes
// the completed appointment save.
func insertAppointmentLoginTracking(
	r *http.Request,
	actorUserID string,
	actorEmail string,
	activeCompanyID string,
	activeCompanyRole string,
	wasUpdate bool,
	result *blappointment.AppointmentSaveResult,
) {
	actionName := "SaveAppointment"
	if wasUpdate {
		actionName = "UpdateAppointment"
	}

	trackingPayload := struct {
		CompanyID        string                               `json:"companyId"`
		ActorCompanyRole string                               `json:"actorCompanyRole"`
		Result           *blappointment.AppointmentSaveResult `json:"result"`
	}{
		CompanyID:        activeCompanyID,
		ActorCompanyRole: activeCompanyRole,
		Result:           result,
	}

	jsonTrackingPayload, err := json.Marshal(trackingPayload)
	if err != nil {
		log.Printf("[HSaveAppointment] Error creating tracking payload: %v", err)
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
		Action:         actionName,
		Jsonstring:     string(jsonTrackingPayload),
	}

	if err := lt.Insert(); err != nil {
		log.Printf("[HSaveAppointment] Error inserting tracking: %v", err)
	}
}

// readAppointmentClientIP extracts the client IP from common proxy headers,
// falling back to the request remote address.
func readAppointmentClientIP(r *http.Request) string {
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
