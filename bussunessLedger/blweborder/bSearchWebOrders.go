package blweborder

import (
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/weborder"
)

// BSearchWebOrders returns the web order requests of ONE company matching the
// filters. companyID is authoritative (from the validated context).
func BSearchWebOrders(
	companyID string,
	filters weborder.WebOrderSearchFilters,
) ([]weborder.WebOrderSearchRow, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	filters.CompanyID = companyID

	results, err := weborder.SearchWebOrders(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search web orders: %w", err)
	}
	return results, nil
}
