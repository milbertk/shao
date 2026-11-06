package steptemplateinshao

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/milbertk/security"
	blsteptemplate "github.com/milbertk/shao/bussunessLedger/blsteeptemplate"
	"github.com/milbertk/tracking"
)

// HSaveStepTemplate creates/updates a step template (header + steps).
//
//	POST /companies/{companyId}/step-templates
func HSaveStepTemplate(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSaveStepTemplate] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	actorUserID := strings.TrimSpace(claims.UID)

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}
	actorEmail, _ := security.GetActiveCompanyEmail(r)
	actorRole, _ := security.GetActiveCompanyRole(r)

	bundle, err := decodeSingleBundle(r)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	wasUpdate := strings.TrimSpace(bundle.Template.StepTemplateID) != ""

	result, err := blsteptemplate.BSaveStepTemplate(companyID, bundle)
	if err != nil {
		status := errorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the step template"
		}
		log.Printf("[HSaveStepTemplate] Failed: status=%d error=%v", status, err)
		writeResponse(w, status, message, nil)
		return
	}

	insertStepTemplateTracking(r, actorUserID, actorEmail, companyID, actorRole, wasUpdate, result)

	log.Printf("[HSaveStepTemplate] Saved OK: id=%s code=%s",
		result.Template.StepTemplateID, result.Template.InternalCode)
	writeResponse(w, http.StatusOK, "Step template saved successfully.", result)
}

func insertStepTemplateTracking(
	r *http.Request,
	actorUserID, actorEmail, activeCompanyID, activeCompanyRole string,
	wasUpdate bool,
	result *blsteptemplate.StepTemplateBundle,
) {
	actionName := "CreateStepTemplate"
	if wasUpdate {
		actionName = "UpdateStepTemplate"
	}
	payload := struct {
		CompanyID        string                             `json:"companyId"`
		ActorCompanyRole string                             `json:"actorCompanyRole"`
		Result           *blsteptemplate.StepTemplateBundle `json:"result"`
	}{activeCompanyID, activeCompanyRole, result}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[HSaveStepTemplate] Error creating tracking payload: %v", err)
		return
	}
	lt := tracking.LoginTracking{
		UserID:         actorUserID,
		Email:          strings.TrimSpace(actorEmail),
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             readClientIP(r),
		Platform:       "Shao",
		MacAddress:     "0",
		Browser:        r.UserAgent(),
		CountryCode:    "",
		GMTTime:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		Lang:           r.Header.Get("Accept-Language"),
		Action:         actionName,
		Jsonstring:     string(jsonPayload),
	}
	if err := lt.Insert(); err != nil {
		log.Printf("[HSaveStepTemplate] Error inserting tracking: %v", err)
	}
}
