package categories

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CategoryS represents one row from public.tbl_categories.
type CategoryS struct {
	CategoryInternalID string    `json:"categoryInternalId"`
	CompanyID          string    `json:"companyId"`
	PublishedCode      string    `json:"publishedCode"`
	CategoryName       string    `json:"categoryName"`
	Description        *string   `json:"description,omitempty"`
	Active             bool      `json:"active"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// Insert creates one category for a company. CompanyID, PublishedCode, and
// CategoryName are mandatory. The database generates CategoryInternalID.
func (a *CategoryS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(a.CompanyID)
	publishedCode := strings.ToUpper(strings.TrimSpace(a.PublishedCode))
	categoryName := strings.TrimSpace(a.CategoryName)

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
		INSERT INTO public.tbl_categories (
			company_id,
			published_code,
			category_name,
			description,
			active
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING category_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
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
		return fmt.Errorf("failed to insert category: %w", err)
	}

	a.CompanyID = companyID
	a.PublishedCode = publishedCode
	a.CategoryName = categoryName

	return nil
}
