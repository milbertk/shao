package tracking

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update modifies one tracking header by primary key AND company.
func (a *TrackingConfigS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	trackingConfigID := strings.TrimSpace(a.TrackingConfigID)
	companyID := strings.TrimSpace(a.CompanyID)
	if trackingConfigID == "" {
		return fmt.Errorf("trackingConfigId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		UPDATE public.tbl_tracking_config
		SET
			tracking_number = $3,
			internal_code = $4,
			name = $5,
			category_id = $6,
			product_id = $7,
			target_type = $8,
			active = $9,
			is_default = $10
		WHERE tracking_config_id = $1
		  AND company_id = $2
		RETURNING tracking_config_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		trackingConfigID,
		companyID,
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
		if err == sql.ErrNoRows {
			return fmt.Errorf("tracking config %q does not exist", trackingConfigID)
		}
		return fmt.Errorf("failed to update tracking config: %w", err)
	}

	return nil
}
