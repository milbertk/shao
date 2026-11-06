package blcustomers

import (
	"errors"
	"fmt"
	"strings"

	customers "github.com/milbertk/shao/dataBasesLedger/customers"
)

// BSearchCustomers searches customers within one company using the data-layer
// search. companyId is forced onto the filters, so results never cross
// companies.
func BSearchCustomers(
	companyID string,
	filters customers.CustomerSearchFilters,
) ([]customers.CustomerS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	filters.CompanyID = companyID

	results, err := customers.SearchCustomers(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search customers: %w", err)
	}

	return results, nil
}
