package blproducts

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/milbertk/shao/bussunessLedger/blplan"
	"github.com/milbertk/shao/bussunessLedger/blstorage"
	blcategories "github.com/milbertk/shao/bussunessLedger/categories"
	"github.com/milbertk/shao/dataBasesLedger/products"
)

// emptyProductStringToNil returns nil when a *string is blank, so optional
// columns receive NULL instead of an empty string.
func emptyProductStringToNil(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// derefProductString returns the trimmed value of an optional string ("" for nil).
func derefProductString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

// normalizeProductOptionalFields converts blank optional strings to nil, and
// clears base_price when the product has no price.
func normalizeProductOptionalFields(product *products.ProductS) {
	product.Description = emptyProductStringToNil(product.Description)
	product.Comments = emptyProductStringToNil(product.Comments)
	product.ImageURL = emptyProductStringToNil(product.ImageURL)
	product.ImagePath = emptyProductStringToNil(product.ImagePath)
	product.WebContactPhone = emptyProductStringToNil(product.WebContactPhone)
	product.WebMessengerLink = emptyProductStringToNil(product.WebMessengerLink)
	product.StepTemplateID = emptyProductStringToNil(product.StepTemplateID)

	if !product.HasPrice {
		product.BasePrice = nil
	}

	// Tax consistency:
	//   - tax free       -> never show taxes, no percentage.
	//   - not free, hidden -> keep tax_free/show flags but drop the percentage.
	//   - not free, shown  -> keep the percentage as provided.
	if product.TaxFree {
		product.ShowTaxes = false
		product.TaxPercentage = nil
	} else if !product.ShowTaxes {
		product.TaxPercentage = nil
	}
}

// DefaultCurrencySymbol is used when a product has no currency symbol.
const DefaultCurrencySymbol = "$"

// maxCurrencySymbolLength is the column size (VARCHAR(5)), counted in characters.
const maxCurrencySymbolLength = 5

// normalizeCurrencySymbol trims the symbol, uses the default when blank and
// rejects symbols longer than the column allows (e.g. "$", "₡", "€", "S/", "R$").
func normalizeCurrencySymbol(symbol string) (string, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return DefaultCurrencySymbol, nil
	}
	if utf8.RuneCountInString(symbol) > maxCurrencySymbolLength {
		return "", fmt.Errorf("currencySymbol must have at most %d characters", maxCurrencySymbolLength)
	}
	return symbol, nil
}

// validateCategoryBelongsToCompany ensures the given category exists in the
// authoritative company. Reuses the categories business layer.
func validateCategoryBelongsToCompany(companyID, categoryID string) error {
	categoryID = strings.TrimSpace(categoryID)
	if categoryID == "" {
		return errors.New("categoryId is required")
	}

	_, err := blcategories.BSelectCategoryByInternalID(companyID, categoryID)
	if err != nil {
		if errors.Is(err, blcategories.ErrCategoryNotFound) {
			return fmt.Errorf("category %q does not exist in this company", categoryID)
		}
		return fmt.Errorf("failed to validate category: %w", err)
	}

	return nil
}

// BSaveProduct creates or updates a product inside a company.
//
//   - companyID is authoritative and always applied.
//   - The category must exist in the same company (validated here).
//   - If ProductInternalID is empty or does not exist in that company, the
//     product is created; otherwise it is updated.
func BSaveProduct(companyID string, input products.ProductS) (*products.ProductS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	// The company always comes from the validated context, never from the body.
	input.CompanyID = companyID

	// Currency symbol: trimmed and validated. Blank stays blank here so an
	// update keeps the current symbol; a new product gets the default.
	if v := strings.TrimSpace(input.CurrencySymbol); v != "" {
		symbol, err := normalizeCurrencySymbol(v)
		if err != nil {
			return nil, err
		}
		input.CurrencySymbol = symbol
	}

	// The linked category must exist in this company.
	if err := validateCategoryBelongsToCompany(companyID, input.CategoryID); err != nil {
		return nil, err
	}

	// The image must be a file of this company in our bucket (2026-10-25,
	// security S13).
	if err := blstorage.ValidateCompanyFile(companyID, derefProductString(input.ImagePath), derefProductString(input.ImageURL)); err != nil {
		return nil, err
	}

	internalID := strings.TrimSpace(input.ProductInternalID)

	// No internal id -> new product (plan limit of products, 2026-10-22).
	if internalID == "" {
		if err := blplan.BCheckCanAddProduct(companyID); err != nil {
			return nil, err
		}
		return createProduct(input)
	}

	// Id provided -> load within the authoritative company.
	existing, err := BSelectProductByInternalID(companyID, internalID)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			if err := blplan.BCheckCanAddProduct(companyID); err != nil {
				return nil, err
			}
			return createProduct(input)
		}
		return nil, fmt.Errorf("failed to validate product before saving: %w", err)
	}

	merged := mergeProduct(*existing, input)
	merged.CompanyID = companyID

	// Re-activating an inactive product also uses one place of the plan.
	if strings.EqualFold(existing.Status, "INACTIVE") && !strings.EqualFold(merged.Status, "INACTIVE") {
		if err := blplan.BCheckCanAddProduct(companyID); err != nil {
			return nil, err
		}
	}
	normalizeProductOptionalFields(&merged)

	if err := merged.Update(); err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	return &merged, nil
}

// createProduct inserts a product. ProductInternalID is cleared so PostgreSQL
// generates its UUID.
func createProduct(input products.ProductS) (*products.ProductS, error) {
	newProduct := input
	newProduct.ProductInternalID = ""
	if strings.TrimSpace(newProduct.CurrencySymbol) == "" {
		newProduct.CurrencySymbol = DefaultCurrencySymbol
	}

	if strings.TrimSpace(newProduct.CompanyID) == "" {
		return nil, errors.New("companyId is required")
	}

	normalizeProductOptionalFields(&newProduct)

	if err := newProduct.Insert(); err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return &newProduct, nil
}

// mergeProduct overlays editable values from input onto existing. The primary
// key and company always remain those of the existing product. Boolean fields
// are always copied because false is a valid explicit value.
func mergeProduct(existing products.ProductS, input products.ProductS) products.ProductS {
	merged := existing

	merged.ProductInternalID = existing.ProductInternalID
	merged.CompanyID = existing.CompanyID

	if v := strings.TrimSpace(input.ProductCode); v != "" {
		merged.ProductCode = v
	}
	if v := strings.TrimSpace(input.CategoryID); v != "" {
		merged.CategoryID = v
	}
	if v := strings.TrimSpace(input.Name); v != "" {
		merged.Name = v
	}
	if v := strings.TrimSpace(input.ProductType); v != "" {
		merged.ProductType = v
	}
	if v := strings.TrimSpace(input.WebBehavior); v != "" {
		merged.WebBehavior = v
	}
	if v := strings.TrimSpace(input.Status); v != "" {
		merged.Status = v
	}

	if input.Description != nil {
		merged.Description = input.Description
	}
	if input.Comments != nil {
		merged.Comments = input.Comments
	}
	if input.BasePrice != nil {
		merged.BasePrice = input.BasePrice
	}
	// Blank keeps the current symbol (older screens do not send it).
	if v := strings.TrimSpace(input.CurrencySymbol); v != "" {
		merged.CurrencySymbol = v
	}
	if strings.TrimSpace(merged.CurrencySymbol) == "" {
		merged.CurrencySymbol = DefaultCurrencySymbol
	}

	// Booleans: always copied (false is a valid explicit value).
	merged.HasPrice = input.HasPrice
	merged.TaxFree = input.TaxFree
	merged.ShowTaxes = input.ShowTaxes
	merged.ShowOnWeb = input.ShowOnWeb
	merged.AvailableForWebSales = input.AvailableForWebSales

	// Tax percentage and image are always taken from the input (nil clears them).
	merged.TaxPercentage = input.TaxPercentage
	merged.ImageURL = input.ImageURL
	merged.ImagePath = input.ImagePath

	// Web contact channels are always taken from the input (nil clears them).
	merged.WebContactPhone = input.WebContactPhone
	merged.WebMessengerLink = input.WebMessengerLink
	merged.StepTemplateID = input.StepTemplateID

	return merged
}
