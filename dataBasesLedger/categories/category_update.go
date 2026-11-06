package categories

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update modifies one category identified by primary key
// (CategoryInternalID) AND company (CompanyID), so a category belonging to
// another company can never be updated. updated_at is refreshed by the
// database trigger.
func (a *CategoryS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	categoryInternalID := strings.TrimSpace(a.CategoryInternalID)
	companyID := strings.TrimSpace(a.CompanyID)
	publishedCode := strings.ToUpper(strings.TrimSpace(a.PublishedCode))
	categoryName := strings.TrimSpace(a.CategoryName)

	if categoryInternalID == "" {
		return fmt.Errorf("categoryInternalId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}
	if publishedCode == "" {
		return fmt.Errorf("publishedCode is required")
	}
	if categoryName == "" {
		return fmt.Errorf("categoryName is required")
	}

	query := `
		UPDATE public.tbl_categories
		SET
			published_code = $3,
			category_name = $4,
			description = $5,
			active = $6
		WHERE category_id = $1
		  AND company_id = $2
		RETURNING category_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		categoryInternalID,
		companyID,
		publishedCode,
		categoryName,
		a.Description,
		a.Active,
	).Scan(
		&a.CategoryInternalID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("category %q does not exist", categoryInternalID)
		}
		return fmt.Errorf("failed to update category: %w", err)
	}

	a.CompanyID = companyID
	a.PublishedCode = publishedCode
	a.CategoryName = categoryName

	return nil
}
