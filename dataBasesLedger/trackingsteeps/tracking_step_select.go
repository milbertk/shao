package tracking

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const trackingStepColumns = `
	tracking_step_id,
	tracking_config_id,
	company_id,
	step_name,
	step_order,
	depends_on_step_id,
	active,
	created_at,
	updated_at
`

func scanTrackingStep(row interface {
	Scan(dest ...any) error
}, s *TrackingStepS) error {
	return row.Scan(
		&s.TrackingStepID,
		&s.TrackingConfigID,
		&s.CompanyID,
		&s.StepName,
		&s.StepOrder,
		&s.DependsOnStepID,
		&s.Active,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
}

// SelectStepsByConfig returns every step of a tracking header, scoped to the
// company, ORDERED BY step_order (then name) so the sequence is preserved.
func SelectStepsByConfig(companyID, trackingConfigID string) ([]TrackingStepS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	trackingConfigID = strings.TrimSpace(trackingConfigID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}
	if trackingConfigID == "" {
		return nil, fmt.Errorf("trackingConfigId is required")
	}

	query := `
		SELECT ` + trackingStepColumns + `
		FROM public.tbl_tracking_step
		WHERE tracking_config_id = $1
		  AND company_id = $2
		ORDER BY step_order ASC, lower(step_name)
	`

	rows, err := db.Query(query, trackingConfigID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to select tracking steps: %w", err)
	}
	defer rows.Close()

	list := make([]TrackingStepS, 0)
	for rows.Next() {
		var item TrackingStepS
		if err := scanTrackingStep(rows, &item); err != nil {
			return nil, fmt.Errorf("failed to scan tracking step: %w", err)
		}
		list = append(list, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading tracking steps: %w", err)
	}

	return list, nil
}
