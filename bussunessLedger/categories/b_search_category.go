package blcategories

import (
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/categories"
)

// BSearchCategories searches categories inside one authoritative company.
// CompanyID from filters is ignored so callers cannot search another company.
func BSearchCategories(
	companyID string,
	filters categories.CategorySearchFilters,
) ([]categories.CategoryS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	filters.CompanyID = companyID
	filters.Term = strings.TrimSpace(filters.Term)
	filters.PublishedCode = strings.TrimSpace(filters.PublishedCode)
	filters.CategoryName = strings.TrimSpace(filters.CategoryName)

	result, err := categories.SearchCategories(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search categories: %w", err)
	}

	return result, nil
}
