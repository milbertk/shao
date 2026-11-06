package reportsinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blreports "github.com/milbertk/shao/bussunessLedger/blreports"
)

// =====================================================================
//  Reports handlers — READ-ONLY (GET).
//
//  company_id comes from the VALIDATED company context that
//  security.RequireActiveCompanyMember populates (same as every other
//  /companies/{companyId}/... handler). The {companyId} in the route is
//  only there so that middleware can validate it; we never trust the path
//  or the body for the company. Register these routes WITH that middleware
//  (see router_snippet_reports.txt) or GetActiveCompanyID returns ok=false
//  and every call is 403.
//
//  No login tracking here: these endpoints do not mutate anything.
// =====================================================================

// activeCompany validates claims + company context, writing the proper
// error response and returning ok=false when either is missing.
func activeCompany(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeRPResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return "", false
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeRPResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return "", false
	}
	return companyID, true
}

// HOrdersByStatus - GET /companies/{companyId}/reports/orders-by-status
//
//	query: from=YYYY-MM-DD & to=YYYY-MM-DD & status=placed|inProgress|partialDone|done|cancelled|all
func HOrdersByStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeRPResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}
	companyID, ok := activeCompany(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	result, err := blreports.BOrdersByStatus(companyID, q.Get("from"), q.Get("to"), q.Get("status"))
	if err != nil {
		writeReportError(w, "HOrdersByStatus", err)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "ReportOrdersByStatus", handlertracking.Details{"filters": handlertracking.SearchFilters(r)})
	writeRPResponse(w, http.StatusOK, "Orders report generated.", result)
}

// HAppointmentsSummary - GET /companies/{companyId}/reports/appointments-summary
//
//	query: from & to & status=scheduled|finished|cancelled|all
func HAppointmentsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeRPResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}
	companyID, ok := activeCompany(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	result, err := blreports.BAppointmentsSummary(companyID, q.Get("from"), q.Get("to"), q.Get("status"))
	if err != nil {
		writeReportError(w, "HAppointmentsSummary", err)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "ReportAppointmentsSummary", handlertracking.Details{"filters": handlertracking.SearchFilters(r)})
	writeRPResponse(w, http.StatusOK, "Appointments report generated.", result)
}

// HTopCustomers - GET /companies/{companyId}/reports/top-customers?limit=10
func HTopCustomers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeRPResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}
	companyID, ok := activeCompany(w, r)
	if !ok {
		return
	}
	result, err := blreports.BTopCustomers(companyID, queryLimit(r, 10))
	if err != nil {
		writeReportError(w, "HTopCustomers", err)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "ReportTopCustomers", handlertracking.Details{"filters": handlertracking.SearchFilters(r)})
	writeRPResponse(w, http.StatusOK, "Top customers report generated.", result)
}

// HBestSellers - GET /companies/{companyId}/reports/best-sellers?type=order|booking|all&limit=50
func HBestSellers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeRPResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}
	companyID, ok := activeCompany(w, r)
	if !ok {
		return
	}
	result, err := blreports.BBestSellers(companyID, r.URL.Query().Get("type"), queryLimit(r, 50))
	if err != nil {
		writeReportError(w, "HBestSellers", err)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "ReportBestSellers", handlertracking.Details{"filters": handlertracking.SearchFilters(r)})
	writeRPResponse(w, http.StatusOK, "Best sellers report generated.", result)
}

// writeReportError maps a business/data error to an HTTP status and hides
// internal details on 500s (the real error is logged).
func writeReportError(w http.ResponseWriter, where string, err error) {
	status := rpErrorStatus(err.Error())
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "An internal error occurred while generating the report"
	}
	log.Printf("[%s] Failed: status=%d error=%v", where, status, err)
	writeRPResponse(w, status, msg, nil)
}
