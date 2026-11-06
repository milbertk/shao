package trackinginshao

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/milbertk/security"
	"github.com/milbertk/tracking"

	bltracking "github.com/milbertk/shao/bussunessLedger/bltracking"
)

// HSaveTracking creates or updates a tracking and rewrites its steps.
//
//	POST /companies/{companyId}/trackings
//
// Body: a TrackingBundle JSON object { config, steps }.
//
// Behavior (decided by the business layer):
//   - Header has an id that exists -> update header, delete steps, rewrite them.
//   - Header id empty or not found  -> create header, create steps.
//
// The company comes from the validated context, never from the body. The
// category/product target is validated exactly once inside the business layer.
// A successful save is recorded with login tracking (best-effort).
func HSaveTracking(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSaveTracking] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeTrackingResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeTrackingResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	actorUserID := strings.TrimSpace(claims.UID)

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeTrackingResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	// Actor context for the login tracking (best-effort reads).
	actorEmail, _ := security.GetActiveCompanyEmail(r)
	actorRole, _ := security.GetActiveCompanyRole(r)

	bundle, err := decodeSingleTrackingBundle(r)
	if err != nil {
		writeTrackingResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	wasUpdate := strings.TrimSpace(bundle.Config.TrackingConfigID) != ""

	log.Printf(
		"[HSaveTracking] Saving: companyId=%s trackingConfigId=%q targetType=%q steps=%d",
		companyID, bundle.Config.TrackingConfigID, bundle.Config.TargetType, len(bundle.Steps),
	)

	// The business layer creates/updates the header, rewrites the steps, and
	// validates the category/product target exactly once.
	result, err := bltracking.BSaveTracking(companyID, bundle)
	if err != nil {
		status := trackingErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the tracking"
		}

		log.Printf("[HSaveTracking] Business operation failed: status=%d error=%v", status, err)
		writeTrackingResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSaveTracking] Saved OK: trackingConfigId=%s", result.Config.TrackingConfigID)

	// Record the successful action (best-effort).
	insertTrackingLoginTracking(
		r,
		actorUserID,
		actorEmail,
		companyID,
		actorRole,
		wasUpdate,
		result,
	)

	writeTrackingResponse(w, http.StatusOK, "Tracking saved successfully.", result)
}

// insertTrackingLoginTracking records the successful save into the login
// tracking table. It is best-effort: a tracking failure is logged but never
// undoes the completed tracking save.
func insertTrackingLoginTracking(
	r *http.Request,
	actorUserID string,
	actorEmail string,
	activeCompanyID string,
	activeCompanyRole string,
	wasUpdate bool,
	result *bltracking.TrackingBundle,
) {
	actionName := "CreateTracking"
	if wasUpdate {
		actionName = "UpdateTracking"
	}

	trackingPayload := struct {
		CompanyID        string                     `json:"companyId"`
		ActorCompanyRole string                     `json:"actorCompanyRole"`
		Result           *bltracking.TrackingBundle `json:"result"`
	}{
		CompanyID:        activeCompanyID,
		ActorCompanyRole: activeCompanyRole,
		Result:           result,
	}

	jsonTrackingPayload, err := json.Marshal(trackingPayload)
	if err != nil {
		log.Printf("[HSaveTracking] Error creating tracking payload: %v", err)
		return
	}

	lt := tracking.LoginTracking{
		UserID:         actorUserID,
		Email:          strings.TrimSpace(actorEmail),
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             readTrackingClientIP(r),
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
		log.Printf("[HSaveTracking] Error inserting tracking: %v", err)
	}
}

// readTrackingClientIP extracts the client IP from common proxy headers,
// falling back to the request remote address.
func readTrackingClientIP(r *http.Request) string {
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
