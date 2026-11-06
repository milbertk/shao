package middlewares

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/HandlerLedgers/planinshao"
	"github.com/milbertk/shao/bussunessLedger/blplan"
)

// RequireActivePlan lets the request pass only when the company has an
// active plan (2026-10-23). A company without an active plan (new company,
// or its plan/trial ended) is READ-ONLY: every create / change / delete
// answers 403 with result.code = PLAN_REQUIRED. Use it on write routes only;
// reading (GET) and the plan request form are always allowed.
//
// The company is taken from the validated context (after
// security.RequireActiveCompanyMember) or, on public routes, from the
// {companyId} route value already turned into the id by ResolvePublicCompany
// (so put it INSIDE ResolvePublicCompany).
func RequireActivePlan(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		companyID, ok := security.GetActiveCompanyID(r)
		public := false
		if !ok || strings.TrimSpace(companyID) == "" {
			companyID = mux.Vars(r)["companyId"]
			public = true
		}
		companyID = strings.TrimSpace(companyID)
		if companyID == "" {
			next.ServeHTTP(w, r)
			return
		}

		if err := blplan.BCheckActivePlan(companyID); err != nil {
			details := handlertracking.Details{"path": r.URL.Path, "method": r.Method}
			if public {
				handlertracking.RecordPublic(r, "PlanRequiredBlocked", companyID, details)
			} else {
				handlertracking.Record(r, "PlanRequiredBlocked", details)
			}
			planinshao.WritePlanLimit(w, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}
