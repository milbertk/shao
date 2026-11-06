package steptemplateinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blsteptemplate "github.com/milbertk/shao/bussunessLedger/blsteeptemplate"
)

// HSelectStepTemplate returns one template + its steps.
//
//	GET /companies/{companyId}/step-templates/{stepTemplateId}
func HSelectStepTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}
	id := strings.TrimSpace(mux.Vars(r)["stepTemplateId"])
	if id == "" {
		writeResponse(w, http.StatusBadRequest, "stepTemplateId is required", nil)
		return
	}
	result, err := blsteptemplate.BSelectStepTemplateFull(companyID, id)
	if err != nil {
		status := errorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while selecting the step template"
		}
		log.Printf("[HSelectStepTemplate] Failed: status=%d error=%v", status, err)
		writeResponse(w, status, message, nil)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectStepTemplate", handlertracking.Details{"stepTemplateId": id})
	writeResponse(w, http.StatusOK, "Step template retrieved successfully.", result)
}
