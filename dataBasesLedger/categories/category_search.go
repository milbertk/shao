package categories

import (
	"fmt"
	"strconv"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CategorySearchFilters holds optional search criteria. CompanyID is REQUIRED
// and always applied, so a search never crosses company boundaries.
type CategorySearchFilters struct {
	CompanyID     string
	Term          string
	PublishedCode string
	CategoryName  string
	Active        *bool
	Limit         int
	Offset        int
}

// SearchCategories returns the categories of ONE company that match the
// filters, ordered by category name.
func SearchCategories(filters CategorySearchFilters) ([]CategoryS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(filters.CompanyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	// company_id is always the first condition.
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

	term := strings.TrimSpace(filters.Term)
	if term != "" {
		conditions = append(conditions, fmt.Sprintf(
			`(
				published_code ILIKE $%d
				OR category_name ILIKE $%d
				OR COALESCE(description, '') ILIKE $%d
			)`,
			position, position, position,
		))
		args = append(args, "%"+term+"%")
		position++
	}

	addLike("published_code", filters.PublishedCode)
	addLike("category_name", filters.CategoryName)

	if filters.Active != nil {
		conditions = append(conditions, fmt.Sprintf("active = $%d", position))
		args = append(args, *filters.Active)
		position++
	}

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
		SELECT `+categorySelectColumns+`
		FROM public.tbl_categories
		%s
		ORDER BY LOWER(category_name), LOWER(published_code)
		LIMIT %s OFFSET %s
	`, whereClause, limitPlaceholder, offsetPlaceholder)

	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search categories: %w", err)
	}
	defer rows.Close()

	categories := make([]CategoryS, 0)
	for rows.Next() {
		var category CategoryS
		if err := scanCategory(rows, &category); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading categories: %w", err)
	}

	return categories, nil
}
