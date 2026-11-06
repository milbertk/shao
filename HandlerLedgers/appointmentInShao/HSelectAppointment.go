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

// HSelectAppointment returns one appointment, scoped to the company.
//
//	GET /companies/{companyId}/appointments/{appointmentId}
func HSelectAppointment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeAppointmentResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
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

	result, err := blappointment.BSelectAppointmentByID(companyID, appointmentID)
	if err != nil {
		status := appointmentErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while selecting the appointment"
		}
		log.Printf("[HSelectAppointment] Business operation failed: status=%d error=%v", status, err)
		writeAppointmentResponse(w, status, message, nil)
		return
	}

	// Record the view (best-effort login tracking).
	actorEmail, _ := security.GetActiveCompanyEmail(r)
	actorRole, _ := security.GetActiveCompanyRole(r)
	insertAppointmentSelectTracking(
		r,
		strings.TrimSpace(claims.UID),
		actorEmail,
		companyID,
		actorRole,
		appointmentID,
	)

	writeAppointmentResponse(w, http.StatusOK, "Appointment retrieved successfully.", result)
}

// insertAppointmentSelectTracking records a "view one appointment" action into
// the login tracking table. Best-effort: a failure is logged, never blocks the
// read. Reuses readAppointmentClientIP (defined in HSaveAppointment.go, same
// package).
func insertAppointmentSelectTracking(
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
		log.Printf("[HSelectAppointment] Error creating tracking payload: %v", err)
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
		Action:         "SelectAppointment",
		Jsonstring:     string(jsonTrackingPayload),
	}

	if err := lt.Insert(); err != nil {
		log.Printf("[HSelectAppointment] Error inserting tracking: %v", err)
	}
}
