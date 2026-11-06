package pageconfigurationinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blpageconfiguration "github.com/milbertk/shao/bussunessLedger/blpageconfiguration"
)

// HSavePageConfiguration creates or replaces the page configuration of the
// active company.
//
//	PUT /companies/{companyId}/page-configuration
//
// The company is taken from the validated context (set by
// RequireActiveCompanyMember), never trusted from the body.
func HSavePageConfiguration(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSavePageConfiguration] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writePageConfigurationResponse(w, http.StatusMethodNotAllowed, "Use PUT", nil)
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

	config, err := decodePageConfiguration(r)
	if err != nil {
		writePageConfigurationResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	result, err := blpageconfiguration.BSavePageConfiguration(companyID, config)
	if err != nil {
		status := pageConfigurationErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the page configuration"
		}

		log.Printf("[HSavePageConfiguration] Business operation failed: status=%d error=%v", status, err)
		writePageConfigurationResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSavePageConfiguration] Saved OK: companyId=%s", companyID)
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SavePageConfiguration", handlertracking.Details{"result": result})
	writePageConfigurationResponse(w, http.StatusOK, "Page configuration saved successfully.", result)
}
