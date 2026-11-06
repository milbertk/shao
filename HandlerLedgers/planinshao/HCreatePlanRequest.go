package planinshao

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/blcaptcha"
	"github.com/milbertk/shao/bussunessLedger/blplan"
)

const maxPlanRequestBodyBytes = 8 * 1024

// HCreatePlanRequest saves a plan request and emails it to the AlephK team
// ("An agent will contact you soon").
//
//	POST /companies/{companyId}/plan-requests   (any active company member)
//	body: { requestedPlan, contactName, contactEmail, contactPhone, message, language }
func HCreatePlanRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writePlanResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writePlanResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writePlanResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	// CAPTCHA (Cloudflare Turnstile, 2026-10-25): off when
	// TURNSTILE_SECRET_KEY is empty.
	if err := blcaptcha.BVerifyCaptcha(r.Header.Get(blcaptcha.HeaderName)); err != nil {
		code := blcaptcha.Code(err)
		handlertracking.Record(r, "CreatePlanRequestCaptchaFailed", handlertracking.Details{"code": code})
		status := http.StatusBadRequest
		if code == "CAPTCHA_UNAVAILABLE" {
			status = http.StatusServiceUnavailable
		}
		writePlanResponse(w, status, "Please complete the security check and try again.", map[string]string{"code": code})
		return
	}

	var input blplan.PlanRequestInput
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxPlanRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writePlanResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writePlanResponse(w, http.StatusBadRequest, "the request body must contain only one JSON object", nil)
		return
	}

	// The signed-in user's email goes in the request email (who sent it).
	userEmail := strings.TrimSpace(claims.Email)
	if contextEmail, ok := security.GetActiveCompanyEmail(r); ok && strings.TrimSpace(contextEmail) != "" {
		userEmail = strings.TrimSpace(contextEmail)
	}
	request, err := blplan.BCreatePlanRequest(companyID, claims.UID, userEmail, input)
	if err != nil {
		basicErr, missingBasic := blplan.AsBasicCompanyError(err)
		switch {
		case missingBasic:
			// Basic company (name, logo, header) not complete (2026-10-29).
			handlertracking.Record(r, "CreatePlanRequestBasicCompanyMissing", handlertracking.Details{"missing": basicErr.Missing})
			writePlanResponse(w, http.StatusConflict, basicErr.Error(), basicErr)
		case errors.Is(err, blplan.ErrPlanRequestInvalid):
			writePlanResponse(w, http.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, blplan.ErrTooManyPlanRequests):
			handlertracking.Record(r, "CreatePlanRequestLimitReached", nil)
			writePlanResponse(w, http.StatusTooManyRequests, err.Error(), map[string]string{"code": "PLAN_REQUEST_LIMIT"})
		default:
			log.Printf("[HCreatePlanRequest] failed: companyId=%s error=%v", companyID, err)
			writePlanResponse(w, http.StatusInternalServerError, "An internal error occurred while sending the request", nil)
		}
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "CreatePlanRequest", handlertracking.Details{
		"planRequestId": request.PlanRequestID,
		"currentPlan":   request.CurrentPlan,
		"requestedPlan": request.RequestedPlan,
	})
	writePlanResponse(w, http.StatusCreated, "Request sent. An agent will contact you soon.", request)
}
