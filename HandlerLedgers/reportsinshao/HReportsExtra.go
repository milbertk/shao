package reportsinshao

import (
	"net/http"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blreports "github.com/milbertk/shao/bussunessLedger/blreports"
)

// HCancellationRate - GET /companies/{companyId}/reports/cancellation-rate
//
//	query: from=YYYY-MM-DD & to=YYYY-MM-DD
func HCancellationRate(w http.ResponseWriter, r *http.Request) {
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
	result, err := blreports.BCancellationRate(companyID, q.Get("from"), q.Get("to"))
	if err != nil {
		writeReportError(w, "HCancellationRate", err)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "ReportCancellationRate", handlertracking.Details{"filters": handlertracking.SearchFilters(r)})
	writeRPResponse(w, http.StatusOK, "Cancellation rate report generated.", result)
}

// HRevenueOverTime - GET /companies/{companyId}/reports/revenue-over-time
//
//	query: from & to & granularity=day|week|month
func HRevenueOverTime(w http.ResponseWriter, r *http.Request) {
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
	result, err := blreports.BRevenueOverTime(companyID, q.Get("from"), q.Get("to"), q.Get("granularity"))
	if err != nil {
		writeReportError(w, "HRevenueOverTime", err)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "ReportRevenueOverTime", handlertracking.Details{"filters": handlertracking.SearchFilters(r)})
	writeRPResponse(w, http.StatusOK, "Revenue over time report generated.", result)
}
