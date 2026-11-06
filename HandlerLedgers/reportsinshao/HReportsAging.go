package reportsinshao

import (
	"net/http"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blreports "github.com/milbertk/shao/bussunessLedger/blreports"
)

// HAgingOrders - GET /companies/{companyId}/reports/aging-orders
//
//	query: status=placed|inProgress|partialDone|all & minDays=<int>
//
// Reuses activeCompany / writeRPResponse / writeReportError from the other
// reports handlers (same package).
func HAgingOrders(w http.ResponseWriter, r *http.Request) {
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
	result, err := blreports.BAgingOrders(companyID, q.Get("status"), q.Get("minDays"))
	if err != nil {
		writeReportError(w, "HAgingOrders", err)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "ReportAgingOrders", handlertracking.Details{"filters": handlertracking.SearchFilters(r)})
	writeRPResponse(w, http.StatusOK, "Aging orders report generated.", result)
}
