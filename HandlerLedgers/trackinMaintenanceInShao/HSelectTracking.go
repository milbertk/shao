package trackinginshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	bltracking "github.com/milbertk/shao/bussunessLedger/bltracking"
)

// HSelectTracking returns one tracking with its header + ordered steps (the
// full bundle), scoped to the active company.
//
//	GET /companies/{companyId}/trackings/{trackingConfigId}
func HSelectTracking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeTrackingResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeTrackingResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeTrackingResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	routeVariables := mux.Vars(r)
	trackingConfigID := strings.TrimSpace(routeVariables["trackingConfigId"])
	if trackingConfigID == "" {
		writeTrackingResponse(w, http.StatusBadRequest, "trackingConfigId is required", nil)
		return
	}

	result, err := bltracking.BSelectTrackingFull(companyID, trackingConfigID)
	if err != nil {
		status := trackingErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while selecting the tracking"
		}

		log.Printf("[HSelectTracking] Business operation failed: status=%d error=%v", status, err)
		writeTrackingResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectTrackingConfig", handlertracking.Details{"trackingConfigId": trackingConfigID})
	writeTrackingResponse(w, http.StatusOK, "Tracking retrieved successfully.", result)
}
