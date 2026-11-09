package locations

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update modifies one location by primary key AND company.
// Returns sql.ErrNoRows (wrapped) when it does not exist in the company.
func (l *CompanyLocationS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	locationID := strings.TrimSpace(l.LocationID)
	companyID := strings.TrimSpace(l.CompanyID)
	if locationID == "" {
		return fmt.Errorf("locationId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		UPDATE public.tbl_company_location
		SET
			name = $3,
			address = $4,
			details = $5,
			latitude = $6,
			longitude = $7,
			active = $8,
			updated_at = now()
		WHERE location_id::text = $1
		  AND company_id = $2
		RETURNING location_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		locationID,
		companyID,
		l.Name,
		l.Address,
		l.Details,
		l.Latitude,
		l.Longitude,
		l.Active,
	).Scan(
		&l.LocationID,
		&l.CreatedAt,
		&l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("location %q does not exist: %w", locationID, sql.ErrNoRows)
		}
		return fmt.Errorf("failed to update location: %w", err)
	}
	return nil
}

// SetLocationActive enables or disables one location of the company.
// Returns sql.ErrNoRows when it does not exist in the company.
func SetLocationActive(companyID, locationID string, active bool) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	res, err := db.Exec(`
		UPDATE public.tbl_company_location
		SET active = $3, updated_at = now()
		WHERE location_id::text = $1
		  AND company_id = $2
	`, strings.TrimSpace(locationID), strings.TrimSpace(companyID), active)
	if err != nil {
		return fmt.Errorf("failed to change location status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
