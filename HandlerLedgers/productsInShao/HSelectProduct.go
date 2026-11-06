package productsinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blproducts "github.com/milbertk/shao/bussunessLedger/products"
)

// HSelectProduct returns a single product of the active company by its
// internal id.
//
//	GET /companies/{companyId}/products/{productInternalId}
func HSelectProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeProductResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeProductResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeProductResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	routeVariables := mux.Vars(r)
	productInternalID := strings.TrimSpace(routeVariables["productInternalId"])
	if productInternalID == "" {
		writeProductResponse(w, http.StatusBadRequest, "productInternalId is required", nil)
		return
	}

	result, err := blproducts.BSelectProductByInternalID(companyID, productInternalID)
	if err != nil {
		status := productErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while selecting the product"
		}

		log.Printf("[HSelectProduct] Business operation failed: status=%d error=%v", status, err)
		writeProductResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectProduct", handlertracking.Details{"productInternalId": productInternalID})
	writeProductResponse(w, http.StatusOK, "Product retrieved successfully.", result)
}
