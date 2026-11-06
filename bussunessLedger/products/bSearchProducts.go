package blproducts

import (
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/products"
)

// BSearchProducts searches products inside one authoritative company.
func BSearchProducts(
	companyID string,
	filters products.ProductSearchFilters,
) ([]products.ProductS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	filters.CompanyID = companyID

	result, err := products.SearchProducts(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	return result, nil
}
