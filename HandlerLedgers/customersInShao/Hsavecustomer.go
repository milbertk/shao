package customersinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blcustomers "github.com/milbertk/shao/bussunessLedger/bcostumers"
)

// HSaveCustomer creates or updates a customer within a company.
//
//	POST /companies/{companyId}/customers
//
// The company is taken from the validated context (set by
// RequireActiveCompanyMember), never trusted from the body.
func HSaveCustomer(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSaveCustomer] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeCustomerResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeCustomerResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeCustomerResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	customer, err := decodeSingleCustomer(r)
	if err != nil {
		writeCustomerResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	log.Printf(
		"[HSaveCustomer] Saving: companyId=%s customerInternalId=%q customerId=%q",
		companyID, customer.CustomerInternalID, customer.CustomerID,
	)

	wasUpdate := strings.TrimSpace(customer.CustomerInternalID) != ""

	result, err := blcustomers.BSaveCustomer(companyID, customer)
	if err != nil {
		status := customerErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the customer"
		}

		log.Printf("[HSaveCustomer] Business operation failed: status=%d error=%v", status, err)
		writeCustomerResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSaveCustomer] Saved OK: customerInternalId=%s", result.CustomerInternalID)
	// Best-effort action tracking (never blocks the response).
	action := "CreateCustomer"
	if wasUpdate {
		action = "UpdateCustomer"
	}
	handlertracking.Record(r, action, handlertracking.Details{"result": result})
	writeCustomerResponse(w, http.StatusOK, "Customer saved successfully.", result)
}
