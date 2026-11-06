package blproducts

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/products"
)

// ErrProductNotFound allows handlers to distinguish a missing product from an
// internal database error.
var ErrProductNotFound = errors.New("product not found")

// BSelectProductByInternalID loads one product by its internal id, scoped to
// the authoritative company.
func BSelectProductByInternalID(
	companyID string,
	productInternalID string,
) (*products.ProductS, error) {
	companyID = strings.TrimSpace(companyID)
	productInternalID = strings.TrimSpace(productInternalID)

	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if productInternalID == "" {
		return nil, errors.New("productInternalId is required")
	}

	product := products.ProductS{
		ProductInternalID: productInternalID,
		CompanyID:         companyID,
	}

	if err := product.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrProductNotFound, productInternalID)
		}
		return nil, fmt.Errorf("failed to select product: %w", err)
	}

	return &product, nil
}
