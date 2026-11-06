package weborderinshao

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blweborder "github.com/milbertk/shao/bussunessLedger/blweborder"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
)

type WebOrderResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeWebOrderResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(WebOrderResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// HSearchWebOrders returns the web order requests of the active company for the
// approval screen. AUTHENTICATED (company member).
//
//	GET /companies/{companyId}/web-orders/search
//	    ?term=&status=&region=&orderType=&requesterType=&dateFrom=&dateTo=&limit=&offset=
func HSearchWebOrders(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSearchWebOrders] Handler reached: method=%s path=%s query=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeWebOrderResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeWebOrderResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeWebOrderResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	query := r.URL.Query()
	parseInt := func(v string) int {
		v = strings.TrimSpace(v)
		if v == "" {
			return 0
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return n
	}

	filters := weborder.WebOrderSearchFilters{
		Term:          strings.TrimSpace(query.Get("term")),
		Status:        strings.TrimSpace(query.Get("status")),
		Region:        strings.TrimSpace(query.Get("region")),
		OrderType:     strings.TrimSpace(query.Get("orderType")),
		RequesterType: strings.TrimSpace(query.Get("requesterType")),
		DateFrom:      strings.TrimSpace(query.Get("dateFrom")),
		DateTo:        strings.TrimSpace(query.Get("dateTo")),
		Limit:         parseInt(query.Get("limit")),
		Offset:        parseInt(query.Get("offset")),
	}

	results, err := blweborder.BSearchWebOrders(companyID, filters)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		if strings.Contains(strings.ToLower(message), "is required") {
			status = http.StatusBadRequest
		} else {
			message = "An internal error occurred while searching web orders"
		}
		log.Printf("[HSearchWebOrders] Failed: status=%d error=%v", status, err)
		writeWebOrderResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSearchWebOrders] Found %d web order(s) for companyId=%s", len(results), companyID)
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SearchWebOrders", handlertracking.Details{"filters": handlertracking.SearchFilters(r), "resultCount": len(results)})
	writeWebOrderResponse(w, http.StatusOK, "Web orders retrieved successfully.", results)
}
