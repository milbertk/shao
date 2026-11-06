package blorder

import (
	"errors"
	"fmt"
	"strings"

	order "github.com/milbertk/shao/dataBasesLedger/orders"
)

// BSearchOrders returns the order headers of ONE company matching the filters.
func BSearchOrders(
	companyID string,
	filters order.OrderSearchFilters,
) ([]order.OrderS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	filters.CompanyID = companyID

	results, err := order.SearchOrders(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search orders: %w", err)
	}

	return results, nil
}
