package categories

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const categorySelectColumns = `
	category_id,
	company_id,
	published_code,
	category_name,
	description,
	active,
	created_at,
	updated_at
`

// scanCategory scans one row (in the categorySelectColumns order) into c.
func scanCategory(row interface {
	Scan(dest ...any) error
}, c *CategoryS) error {
	return row.Scan(
		&c.CategoryInternalID,
		&c.CompanyID,
		&c.PublishedCode,
		&c.CategoryName,
		&c.Description,
		&c.Active,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
}

// Select loads one category by primary key (CategoryInternalID), scoped to its
// company. Set CategoryInternalID and CompanyID before calling. Returns a
// wrapped sql.ErrNoRows when the category is not found in that company.
func (a *CategoryS) Select() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	categoryInternalID := strings.TrimSpace(a.CategoryInternalID)
	companyID := strings.TrimSpace(a.CompanyID)
	if categoryInternalID == "" {
		return fmt.Errorf("categoryInternalId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		SELECT ` + categorySelectColumns + `
		FROM public.tbl_categories
		WHERE category_id = $1
		  AND company_id = $2
	`

	if err := scanCategory(db.QueryRow(query, categoryInternalID, companyID), a); err != nil {
		return fmt.Errorf("failed to select category: %w", err)
	}

	return nil
}
