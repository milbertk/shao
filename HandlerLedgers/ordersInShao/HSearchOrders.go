package ordersinshao

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blorder "github.com/milbertk/shao/bussunessLedger/blorder"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
)

// HSearchOrders returns the orders of the active company matching the filters.
//
//	GET /companies/{companyId}/orders/search
//	    ?term=&customerId=&customerName=&status=&dateFrom=&dateTo=&limit=&offset=
func HSearchOrders(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSearchOrders] Handler reached: method=%s path=%s query=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeOrderResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeOrderResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeOrderResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	query := r.URL.Query()
	parseInt := func(value string) int {
		value = strings.TrimSpace(value)
		if value == "" {
			return 0
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0
		}
		return parsed
	}

	filters := order.OrderSearchFilters{
		Term:         strings.TrimSpace(query.Get("term")),
		CustomerID:   strings.TrimSpace(query.Get("customerId")),
		CustomerName: strings.TrimSpace(query.Get("customerName")),
		Status:       strings.TrimSpace(query.Get("status")),
		DateFrom:     strings.TrimSpace(query.Get("dateFrom")),
		DateTo:       strings.TrimSpace(query.Get("dateTo")),
		Limit:        parseInt(query.Get("limit")),
		Offset:       parseInt(query.Get("offset")),
	}

	results, err := blorder.BSearchOrders(companyID, filters)
	if err != nil {
		status := orderErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while searching orders"
		}
		log.Printf("[HSearchOrders] Business operation failed: status=%d error=%v", status, err)
		writeOrderResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSearchOrders] Found %d order(s) for companyId=%s", len(results), companyID)
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SearchOrders", handlertracking.Details{"filters": handlertracking.SearchFilters(r), "resultCount": len(results)})
	writeOrderResponse(w, http.StatusOK, "Orders retrieved successfully.", results)
}
