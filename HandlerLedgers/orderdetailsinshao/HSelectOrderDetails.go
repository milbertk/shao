package orderdetailsinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blorderdetails "github.com/milbertk/shao/bussunessLedger/borderdetails"
)

// HSelectOrderDetails returns the whole-order comments + attachments.
//
//	GET /companies/{companyId}/orders/{orderInternalId}/details
func HSelectOrderDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeODResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeODResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeODResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}
	orderID := strings.TrimSpace(mux.Vars(r)["orderInternalId"])
	if orderID == "" {
		writeODResponse(w, http.StatusBadRequest, "orderInternalId is required", nil)
		return
	}
	result, err := blorderdetails.BSelectOrderDetails(companyID, orderID)
	if err != nil {
		status := odErrorStatus(err.Error())
		msg := err.Error()
		if status == http.StatusInternalServerError {
			msg = "An internal error occurred while loading the order details"
		}
		log.Printf("[HSelectOrderDetails] Failed: status=%d error=%v", status, err)
		writeODResponse(w, status, msg, nil)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectOrderDetails", handlertracking.Details{"orderInternalId": orderID})
	writeODResponse(w, http.StatusOK, "Order details retrieved successfully.", result)
}
