package customersinshao

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blcustomers "github.com/milbertk/shao/bussunessLedger/bcostumers"
	customers "github.com/milbertk/shao/dataBasesLedger/customers"
)

// HSearchCustomers returns the customers of the active company matching the
// optional query filters.
func HSearchCustomers(w http.ResponseWriter, r *http.Request) {
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

	query := r.URL.Query()

	parseInt := func(value string) int {
		value = strings.TrimSpace(value)
		if value == "" {
			return 0
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0
		}
		return parsed
	}

	filters := customers.CustomerSearchFilters{
		Term:              strings.TrimSpace(query.Get("term")),
		CustomerID:        strings.TrimSpace(query.Get("customerId")),
		Name:              strings.TrimSpace(query.Get("name")),
		Alias:             strings.TrimSpace(query.Get("alias")),
		Email:             strings.TrimSpace(query.Get("email")),
		Cellphone:         strings.TrimSpace(query.Get("cellphone")),
		SocialNetworkLink: strings.TrimSpace(query.Get("socialNetworkLink")),
		SocialNetwork:     strings.TrimSpace(query.Get("socialNetwork")),
		Status:            strings.TrimSpace(query.Get("status")),
		Category:          strings.TrimSpace(query.Get("category")),
		Limit:             parseInt(query.Get("limit")),
		Offset:            parseInt(query.Get("offset")),
	}

	results, err := blcustomers.BSearchCustomers(companyID, filters)
	if err != nil {
		status := customerErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while searching customers"
		}

		log.Printf("[HSearchCustomers] Business operation failed: status=%d error=%v", status, err)
		writeCustomerResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SearchCustomers", handlertracking.Details{"filters": handlertracking.SearchFilters(r), "resultCount": len(results)})
	writeCustomerResponse(w, http.StatusOK, "Customers retrieved successfully.", results)
}
