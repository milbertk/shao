package tracking

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const trackingConfigColumns = `
	tracking_config_id,
	company_id,
	tracking_number,
	internal_code,
	name,
	category_id,
	product_id,
	target_type,
	active,
	is_default,
	created_at,
	updated_at
`

func scanTrackingConfig(row interface {
	Scan(dest ...any) error
}, c *TrackingConfigS) error {
	return row.Scan(
		&c.TrackingConfigID,
		&c.CompanyID,
		&c.TrackingNumber,
		&c.InternalCode,
		&c.Name,
		&c.CategoryID,
		&c.ProductID,
		&c.TargetType,
		&c.Active,
		&c.IsDefault,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
}

// Select loads one tracking header by primary key, scoped to its company.
func (a *TrackingConfigS) Select() error {
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
		SELECT ` + trackingConfigColumns + `
		FROM public.tbl_tracking_config
		WHERE tracking_config_id = $1
		  AND company_id = $2
	`

	if err := scanTrackingConfig(db.QueryRow(query, trackingConfigID, companyID), a); err != nil {
		return fmt.Errorf("failed to select tracking config: %w", err)
	}

	return nil
}
