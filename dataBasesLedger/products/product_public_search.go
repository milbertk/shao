package products

import (
	"fmt"
	"strconv"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// PublicProductSearchFilters holds the optional criteria a PUBLIC (unauthenticated)
// showcase visitor may use. CompanyID is REQUIRED. The public filter
// (show_on_web = TRUE AND status = 'ACTIVE') is ALWAYS forced here, so private
// or hidden products can never be returned.
type PublicProductSearchFilters struct {
	CompanyID   string
	Term        string // matches name + description
	CategoryID  string
	ProductType string // SERVICE | ORDER_PRODUCT | INVENTORY_PRODUCT
	ProductID   string // product_internal_id (exact)
	Limit       int
	Offset      int
}

// SearchPublicProducts returns ONLY the web-available, active products of one
// company. This is the read-only query used by the public showcase.
func SearchPublicProducts(filters PublicProductSearchFilters) ([]ProductS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(filters.CompanyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	// Forced public conditions: company + visible on web + active.
	conditions := []string{
		"company_id = $1",
		"show_on_web = TRUE",
		"status = 'ACTIVE'",
	}
	args := []interface{}{companyID}
	position := 2

	addEqual := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, position))
		args = append(args, value)
		position++
	}

	// Free-text term across name + description.
	term := strings.TrimSpace(filters.Term)
	if term != "" {
		conditions = append(conditions, fmt.Sprintf(
			`(
				name ILIKE $%d
				OR COALESCE(description, '') ILIKE $%d
			)`,
			position, position,
		))
		args = append(args, "%"+term+"%")
		position++
	}

	addEqual("category_id", filters.CategoryID)
	addEqual("product_type", strings.ToUpper(filters.ProductType))
	addEqual("product_internal_id", filters.ProductID)

	whereClause := "WHERE " + strings.Join(conditions, "\n\t\t\t  AND ")

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	limitPlaceholder := "$" + strconv.Itoa(position)
	position++
	offsetPlaceholder := "$" + strconv.Itoa(position)

	query := fmt.Sprintf(`
		SELECT `+productSelectColumns+`
		FROM public.tbl_products
		%s
		ORDER BY lower(name)
		LIMIT %s OFFSET %s
	`, whereClause, limitPlaceholder, offsetPlaceholder)

	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search public products: %w", err)
	}
	defer rows.Close()

	list := make([]ProductS, 0)
	for rows.Next() {
		var product ProductS
		if err := scanProduct(rows, &product); err != nil {
			return nil, fmt.Errorf("failed to scan public product: %w", err)
		}
		list = append(list, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading public products: %w", err)
	}

	return list, nil
}
