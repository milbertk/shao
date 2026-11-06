package orderitemtrackinginshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blorderitemtracking "github.com/milbertk/shao/bussunessLedger/blorderitemtracking"
)

// HSelectOrderItemTracking returns an order's items + computed order status.
//
//	GET /companies/{companyId}/orders/{orderInternalId}/item-tracking
func HSelectOrderItemTracking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeOITResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeOITResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeOITResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	orderID := strings.TrimSpace(mux.Vars(r)["orderInternalId"])
	if orderID == "" {
		writeOITResponse(w, http.StatusBadRequest, "orderInternalId is required", nil)
		return
	}

	result, err := blorderitemtracking.BSelectOrderTracking(companyID, orderID)
	if err != nil {
		status := oitErrorStatus(err.Error())
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while loading the item tracking"
		}
		log.Printf("[HSelectOrderItemTracking] Failed: status=%d error=%v", status, err)
		writeOITResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectOrderItemTracking", handlertracking.Details{"orderInternalId": orderID})
	writeOITResponse(w, http.StatusOK, "Item tracking retrieved successfully.", result)
}
