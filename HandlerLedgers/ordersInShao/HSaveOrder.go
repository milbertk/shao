package ordersinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/blorder"
)

// HSaveOrder creates or updates an order and rewrites its items.
//
//	POST /companies/{companyId}/orders
//
// Body: an OrderBundle JSON object { order, items }. The company is taken from
// the validated context.
func HSaveOrder(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSaveOrder] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeOrderResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
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

	bundle, err := decodeSingleOrderBundle(r)
	if err != nil {
		writeOrderResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	log.Printf(
		"[HSaveOrder] Saving: companyId=%s orderInternalId=%q items=%d",
		companyID, bundle.Order.OrderInternalID, len(bundle.Items),
	)

	wasUpdate := strings.TrimSpace(bundle.Order.OrderInternalID) != ""

	result, err := blorder.BSaveOrder(companyID, bundle)
	if err != nil {
		status := orderErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the order"
		}
		log.Printf("[HSaveOrder] Business operation failed: status=%d error=%v", status, err)
		writeOrderResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSaveOrder] Saved OK: orderInternalId=%s", result.Order.OrderInternalID)
	// Best-effort action tracking (never blocks the response).
	action := "CreateOrder"
	if wasUpdate {
		action = "UpdateOrder"
	}
	handlertracking.Record(r, action, handlertracking.Details{"result": result})
	writeOrderResponse(w, http.StatusOK, "Order saved successfully.", result)
}
