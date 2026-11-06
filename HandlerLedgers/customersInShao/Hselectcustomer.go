package customersinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blcustomers "github.com/milbertk/shao/bussunessLedger/bcostumers"
	customers "github.com/milbertk/shao/dataBasesLedger/customers"
)

// HSelectCustomer returns a single customer of the active company. Lookup by
// priority: path {customerInternalId}, then query customerId, email, phone,
// socialNetworkLink.
func HSelectCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeCustomerResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
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

	routeVariables := mux.Vars(r)
	query := r.URL.Query()

	internalID := strings.TrimSpace(routeVariables["customerInternalId"])
	customerID := strings.TrimSpace(query.Get("customerId"))
	email := strings.TrimSpace(query.Get("email"))
	phone := strings.TrimSpace(query.Get("phone"))
	socialNetworkLink := strings.TrimSpace(query.Get("socialNetworkLink"))

	var (
		result *customers.CustomerS
		err    error
	)

	switch {
	case internalID != "":
		result, err = blcustomers.BSelectCustomerByInternalID(companyID, internalID)
	case customerID != "":
		result, err = blcustomers.BSelectCustomerByID(companyID, customerID)
	case email != "":
		result, err = blcustomers.BSelectCustomerByEmail(companyID, email)
	case phone != "":
		result, err = blcustomers.BSelectCustomerByPhone(companyID, phone)
	case socialNetworkLink != "":
		result, err = blcustomers.BSelectCustomerBySocialNetworkLink(companyID, socialNetworkLink)
	default:
		writeCustomerResponse(
			w,
			http.StatusBadRequest,
			"Provide customerInternalId (path) or one of the query filters: customerId, email, phone, socialNetworkLink",
			nil,
		)
		return
	}

	if err != nil {
		status := customerErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while selecting the customer"
		}

		log.Printf("[HSelectCustomer] Business operation failed: status=%d error=%v", status, err)
		writeCustomerResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectCustomer", handlertracking.Details{"customerInternalId": internalID, "lookup": handlertracking.SearchFilters(r)})
	writeCustomerResponse(w, http.StatusOK, "Customer retrieved successfully.", result)
}
