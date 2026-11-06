package tracking

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// TrackingStepS represents one row from public.tbl_tracking_step.
type TrackingStepS struct {
	TrackingStepID   string    `json:"trackingStepId"`
	TrackingConfigID string    `json:"trackingConfigId"`
	CompanyID        string    `json:"companyId"`
	StepName         string    `json:"stepName"`
	StepOrder        int       `json:"stepOrder"`
	DependsOnStepID  *string   `json:"dependsOnStepId,omitempty"`
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// Insert creates one tracking step for a tracking header.
func (a *TrackingStepS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	if a.StepOrder < 1 {
		a.StepOrder = 1
	}

	query := `
		INSERT INTO public.tbl_tracking_step (
			tracking_config_id,
			company_id,
			step_name,
			step_order,
			depends_on_step_id,
			active
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING tracking_step_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.TrackingConfigID,
		a.CompanyID,
		a.StepName,
		a.StepOrder,
		a.DependsOnStepID,
		a.Active,
	).Scan(
		&a.TrackingStepID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert tracking step: %w", err)
	}

	return nil
}
