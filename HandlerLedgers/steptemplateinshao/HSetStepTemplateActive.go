package steptemplateinshao

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blsteptemplate "github.com/milbertk/shao/bussunessLedger/blsteeptemplate"
)

type setActiveBody struct {
	Active *bool `json:"active"`
}

// HSetStepTemplateActive enables or disables a process (2026-10-30). A
// disabled process is not offered in the product form.
//
//	PUT /companies/{companyId}/step-templates/{stepTemplateId}/active   body: { "active": true|false }
func HSetStepTemplateActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writeResponse(w, http.StatusMethodNotAllowed, "Use PUT", nil)
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

	var body setActiveBody
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || body.Active == nil {
		writeResponse(w, http.StatusBadRequest, "body must be { \"active\": true|false }", nil)
		return
	}

	if err := blsteptemplate.BSetStepTemplateActive(companyID, id, *body.Active); err != nil {
		status := errorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while changing the process status"
		}
		log.Printf("[HSetStepTemplateActive] Failed: status=%d error=%v", status, err)
		writeResponse(w, status, message, nil)
		return
	}

	action := "DisableStepTemplate"
	message := "Process disabled."
	if *body.Active {
		action = "EnableStepTemplate"
		message = "Process enabled."
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, action, handlertracking.Details{"stepTemplateId": id})
	writeResponse(w, http.StatusOK, message, map[string]bool{"active": *body.Active})
}
