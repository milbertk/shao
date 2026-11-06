package products

import (
	"fmt"
	"strconv"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// ProductSearchFilters holds optional search criteria. CompanyID is REQUIRED
// and always applied, so a search never crosses company boundaries.
type ProductSearchFilters struct {
	CompanyID   string
	Term        string
	ProductCode string
	Name        string
	CategoryID  string
	ProductType string
	Status      string
	Limit       int
	Offset      int
}

// SearchProducts returns the products of ONE company that match the filters,
// ordered by name.
func SearchProducts(filters ProductSearchFilters) ([]ProductS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(filters.CompanyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	conditions := []string{"company_id = $1"}
	args := []interface{}{companyID}
	position := 2

	addLike := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", column, position))
		args = append(args, "%"+value+"%")
		position++
	}

	addEqual := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, position))
		args = append(args, value)
		position++
	}

	term := strings.TrimSpace(filters.Term)
	if term != "" {
		conditions = append(conditions, fmt.Sprintf(
			`(
				product_code ILIKE $%d
				OR name ILIKE $%d
				OR COALESCE(description, '') ILIKE $%d
				OR COALESCE(comments, '') ILIKE $%d
			)`,
			position, position, position, position,
		))
		args = append(args, "%"+term+"%")
		position++
	}

	addLike("product_code", filters.ProductCode)
	addLike("name", filters.Name)

	addEqual("category_id", filters.CategoryID)
	addEqual("product_type", strings.ToUpper(filters.ProductType))
	addEqual("status", strings.ToUpper(filters.Status))

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
		return nil, fmt.Errorf("failed to search products: %w", err)
	}
	defer rows.Close()

	list := make([]ProductS, 0)
	for rows.Next() {
		var product ProductS
		if err := scanProduct(rows, &product); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		list = append(list, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading products: %w", err)
	}

	return list, nil
}
