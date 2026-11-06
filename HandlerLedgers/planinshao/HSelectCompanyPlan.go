package planinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/blplan"
	"github.com/milbertk/shao/dataBasesLedger/planrequests"
	"github.com/milbertk/shao/dataBasesLedger/plans"
)

// companyPlanResult is what the "Your plan" badge and window need.
type companyPlanResult struct {
	Current        *blplan.CompanyPlan        `json:"current"`
	Plans          []plans.PlanS              `json:"plans"`
	PendingRequest *planrequests.PlanRequestS `json:"pendingRequest,omitempty"`
	// Basic company fields still missing in Company page (2026-10-29):
	// name, logo, header. Empty = complete, a plan can be requested.
	BasicCompanyMissing []string `json:"basicCompanyMissing"`
}

// HSelectCompanyPlan returns the company's current plan (with usage), the
// list of plans with prices, and the last pending request.
//
//	GET /companies/{companyId}/plan   (any active company member)
func HSelectCompanyPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writePlanResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
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

	current, err := blplan.BGetCompanyPlan(companyID)
	if err != nil {
		log.Printf("[HSelectCompanyPlan] failed: companyId=%s error=%v", companyID, err)
		writePlanResponse(w, http.StatusInternalServerError, "An internal error occurred while loading the plan", nil)
		return
	}

	// Requestable plans + the company's own (SPECIAL only for who has it).
	result := companyPlanResult{Current: current, Plans: blplan.BVisiblePlans(current.AssignedPlan)}
	if pending, err := planrequests.SelectLastPendingRequest(companyID); err == nil {
		result.PendingRequest = pending
	} else {
		log.Printf("[HSelectCompanyPlan] pending request not read: companyId=%s error=%v", companyID, err)
	}
	if missing, err := blplan.BBasicCompanyMissing(companyID); err == nil {
		result.BasicCompanyMissing = missing
	} else {
		// Never block the plan window: the plan request checks it again.
		log.Printf("[HSelectCompanyPlan] basic company not read: companyId=%s error=%v", companyID, err)
		result.BasicCompanyMissing = []string{}
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectCompanyPlan", handlertracking.Details{"plan": current.PlanCode, "active": current.Active, "basicCompanyMissing": result.BasicCompanyMissing})
	writePlanResponse(w, http.StatusOK, "Plan retrieved successfully.", result)
}
