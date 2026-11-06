package weborderinshao

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blweborder "github.com/milbertk/shao/bussunessLedger/blweborder"
)

// HSelectWebOrder returns one web order with header + items, scoped to company.
//
//	GET /companies/{companyId}/web-orders/{webOrderRequestId}
func HSelectWebOrder(w http.ResponseWriter, r *http.Request) {
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

	webOrderID := strings.TrimSpace(mux.Vars(r)["webOrderRequestId"])
	if webOrderID == "" {
		writeWebOrderResponse(w, http.StatusBadRequest, "webOrderRequestId is required", nil)
		return
	}

	result, err := blweborder.BSelectWebOrderFull(companyID, webOrderID)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		if errors.Is(err, blweborder.ErrWebOrderNotFound) {
			status = http.StatusNotFound
		} else if strings.Contains(strings.ToLower(message), "is required") {
			status = http.StatusBadRequest
		} else {
			message = "An internal error occurred while selecting the web order"
		}
		log.Printf("[HSelectWebOrder] Failed: status=%d error=%v", status, err)
		writeWebOrderResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectWebOrder", handlertracking.Details{"webOrderRequestId": webOrderID})
	writeWebOrderResponse(w, http.StatusOK, "Web order retrieved successfully.", result)
}
