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

// HDeleteStepTemplate deletes a process that no product or order uses
// (2026-10-30). A used one answers 409: disable it instead.
//
//	DELETE /companies/{companyId}/step-templates/{stepTemplateId}
func HDeleteStepTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		writeResponse(w, http.StatusMethodNotAllowed, "Use DELETE", nil)
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

	if err := blsteptemplate.BDeleteStepTemplate(companyID, id); err != nil {
		status := errorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while deleting the process"
		}
		log.Printf("[HDeleteStepTemplate] Failed: status=%d error=%v", status, err)
		handlertracking.Record(r, "DeleteStepTemplateBlocked", handlertracking.Details{"stepTemplateId": id, "status": status})
		writeResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "DeleteStepTemplate", handlertracking.Details{"stepTemplateId": id})
	writeResponse(w, http.StatusOK, "Process deleted.", nil)
}
