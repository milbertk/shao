package ordertrackinginshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blordertracking "github.com/milbertk/shao/bussunessLedger/blordertracking"
)

// HSelectOrderTracking returns all stages (with children) of one order.
//
//	GET /companies/{companyId}/order-trackings/{orderInternalId}
func HSelectOrderTracking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}
	orderID := strings.TrimSpace(mux.Vars(r)["orderInternalId"])
	if orderID == "" {
		writeResponse(w, http.StatusBadRequest, "orderInternalId is required", nil)
		return
	}

	result, err := blordertracking.BSelectOrderTracking(companyID, orderID)
	if err != nil {
		status := errorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while selecting the order tracking"
		}
		log.Printf("[HSelectOrderTracking] Failed: status=%d error=%v", status, err)
		writeResponse(w, status, message, nil)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectOrderTracking", handlertracking.Details{"orderInternalId": orderID})
	writeResponse(w, http.StatusOK, "Order tracking retrieved successfully.", result)
}
