package blcategories

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/categories"
)

// ErrCategoryNotFound allows handlers to distinguish a missing category from
// an internal database error.
var ErrCategoryNotFound = errors.New("category not found")

// BSelectCategoryByInternalID loads one category by its generated internal id,
// scoped to the authoritative company.
func BSelectCategoryByInternalID(
	companyID string,
	categoryInternalID string,
) (*categories.CategoryS, error) {
	companyID = strings.TrimSpace(companyID)
	categoryInternalID = strings.TrimSpace(categoryInternalID)

	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if categoryInternalID == "" {
		return nil, errors.New("categoryInternalId is required")
	}

	category := categories.CategoryS{
		CategoryInternalID: categoryInternalID,
		CompanyID:          companyID,
	}

	if err := category.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrCategoryNotFound, categoryInternalID)
		}
		return nil, fmt.Errorf("failed to select category: %w", err)
	}

	return &category, nil
}
