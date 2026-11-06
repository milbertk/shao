package pageconfigurationinshao

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blpageconfiguration "github.com/milbertk/shao/bussunessLedger/blpageconfiguration"
)

// HSelectPageConfiguration returns the page configuration of the active
// company. A company that has not configured its page yet gets 200 with no
// result, so the frontend can simply show an empty form.
//
//	GET /companies/{companyId}/page-configuration
func HSelectPageConfiguration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writePageConfigurationResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writePageConfigurationResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writePageConfigurationResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	result, err := blpageconfiguration.BSelectPageConfiguration(companyID)
	if err != nil {
		if errors.Is(err, blpageconfiguration.ErrPageConfigurationNotFound) {
			writePageConfigurationResponse(w, http.StatusOK, "Page configuration not created yet.", nil)
			return
		}

		status := pageConfigurationErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while loading the page configuration"
		}

		log.Printf("[HSelectPageConfiguration] Business operation failed: status=%d error=%v", status, err)
		writePageConfigurationResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectPageConfiguration", nil)
	writePageConfigurationResponse(w, http.StatusOK, "Page configuration retrieved successfully.", result)
}
