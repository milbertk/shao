package tracking

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// TrackingConfigS represents one row from public.tbl_tracking_config.
type TrackingConfigS struct {
	TrackingConfigID string    `json:"trackingConfigId"`
	CompanyID        string    `json:"companyId"`
	TrackingNumber   *string   `json:"trackingNumber,omitempty"`
	InternalCode     string    `json:"internalCode"`
	Name             string    `json:"name"`
	CategoryID       *string   `json:"categoryId,omitempty"`
	ProductID        *string   `json:"productId,omitempty"`
	TargetType       string    `json:"targetType"` // CATEGORY | PRODUCT
	Active           bool      `json:"active"`
	IsDefault        bool      `json:"isDefault"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// Insert creates one tracking header for a company.
func (a *TrackingConfigS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		INSERT INTO public.tbl_tracking_config (
			company_id,
			tracking_number,
			internal_code,
			name,
			category_id,
			product_id,
			target_type,
			active,
			is_default
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING tracking_config_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.CompanyID,
		a.TrackingNumber,
		a.InternalCode,
		a.Name,
		a.CategoryID,
		a.ProductID,
		a.TargetType,
		a.Active,
		a.IsDefault,
	).Scan(
		&a.TrackingConfigID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert tracking config: %w", err)
	}

	return nil
}
