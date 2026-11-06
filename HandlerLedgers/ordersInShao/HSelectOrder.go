package ordersinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blorder "github.com/milbertk/shao/bussunessLedger/blorder"
)

// HSelectOrder returns one order with its header + items, scoped to the company.
//
//	GET /companies/{companyId}/orders/{orderInternalId}
func HSelectOrder(w http.ResponseWriter, r *http.Request) {
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

	routeVariables := mux.Vars(r)
	orderInternalID := strings.TrimSpace(routeVariables["orderInternalId"])
	if orderInternalID == "" {
		writeOrderResponse(w, http.StatusBadRequest, "orderInternalId is required", nil)
		return
	}

	result, err := blorder.BSelectOrderFull(companyID, orderInternalID)
	if err != nil {
		status := orderErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while selecting the order"
		}
		log.Printf("[HSelectOrder] Business operation failed: status=%d error=%v", status, err)
		writeOrderResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectOrder", handlertracking.Details{"orderInternalId": orderInternalID})
	writeOrderResponse(w, http.StatusOK, "Order retrieved successfully.", result)
}
