package blcategories

import (
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/categories"
)

// emptyCategoryStringToNil returns nil when a *string contains only blanks, so
// optional columns receive NULL instead of an empty string.
func emptyCategoryStringToNil(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

// normalizeCategoryOptionalFields converts blank optional strings to nil.
func normalizeCategoryOptionalFields(category *categories.CategoryS) {
	category.Description = emptyCategoryStringToNil(category.Description)
}

// BSaveCategory creates or updates a category inside a company.
//
//   - companyID is authoritative and always applied.
//   - If CategoryInternalID is empty or does not exist in that company, the
//     category is created.
//   - If CategoryInternalID exists in that company, the category is updated.
//
// The database layer validates the required published code and category name.
func BSaveCategory(companyID string, input categories.CategoryS) (*categories.CategoryS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	// The company always comes from the validated context, never from the body.
	input.CompanyID = companyID
	internalID := strings.TrimSpace(input.CategoryInternalID)

	// No internal id means that this is a new category.
	if internalID == "" {
		return createCategory(input)
	}

	// An id was provided: load it only within the authoritative company.
	existing, err := BSelectCategoryByInternalID(companyID, internalID)
	if err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			return createCategory(input)
		}
		return nil, fmt.Errorf("failed to validate category before saving: %w", err)
	}

	merged := mergeCategory(*existing, input)
	merged.CompanyID = companyID
	normalizeCategoryOptionalFields(&merged)

	if err := merged.Update(); err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	return &merged, nil
}

// createCategory inserts a category. CategoryInternalID is cleared so the
// PostgreSQL database generates its UUID.
func createCategory(input categories.CategoryS) (*categories.CategoryS, error) {
	newCategory := input
	newCategory.CategoryInternalID = ""

	if strings.TrimSpace(newCategory.CompanyID) == "" {
		return nil, errors.New("companyId is required")
	}

	normalizeCategoryOptionalFields(&newCategory)

	if err := newCategory.Insert(); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return &newCategory, nil
}

// mergeCategory overlays the editable values from input onto existing. The
// primary key and company always remain those of the existing category.
// Active is always copied because false is a valid explicit value.
func mergeCategory(existing categories.CategoryS, input categories.CategoryS) categories.CategoryS {
	merged := existing

	merged.CategoryInternalID = existing.CategoryInternalID
	merged.CompanyID = existing.CompanyID

	if value := strings.TrimSpace(input.PublishedCode); value != "" {
		merged.PublishedCode = value
	}
	if value := strings.TrimSpace(input.CategoryName); value != "" {
		merged.CategoryName = value
	}
	if input.Description != nil {
		merged.Description = input.Description
	}

	merged.Active = input.Active

	return merged
}
