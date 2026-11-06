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

// RequirePlanFeature lets the request pass only when the company's plan
// includes the option (2026-10-22). Otherwise it answers 403 with
// result.code = PLAN_FEATURE_LOCKED.
//
// The company is taken from the validated context (after
// security.RequireActiveCompanyMember) or, on public routes, from the
// {companyId} route value already turned into the id by ResolvePublicCompany
// (so put it INSIDE ResolvePublicCompany).
func RequirePlanFeature(feature blplan.Feature) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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

			if err := blplan.BCheckFeature(companyID, feature); err != nil {
				details := handlertracking.Details{"feature": string(feature), "path": r.URL.Path}
				if public {
					handlertracking.RecordPublic(r, "PlanFeatureBlocked", companyID, details)
				} else {
					handlertracking.Record(r, "PlanFeatureBlocked", details)
				}
				planinshao.WritePlanLimit(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
