package locations

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CompanyLocationS represents one row from public.tbl_company_location
// (2026-11-07): a place where the company attends appointments ("Sedes").
type CompanyLocationS struct {
	LocationID string    `json:"locationId"`
	CompanyID  string    `json:"companyId"`
	Name       string    `json:"name"`
	Address    *string   `json:"address,omitempty"`
	Details    *string   `json:"details,omitempty"` // floor, office number, parking...
	Latitude   *float64  `json:"latitude,omitempty"`
	Longitude  *float64  `json:"longitude,omitempty"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Insert creates one location.
func (l *CompanyLocationS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		INSERT INTO public.tbl_company_location (
			company_id,
			name,
			address,
			details,
			latitude,
			longitude,
			active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING location_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		l.CompanyID,
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
		return fmt.Errorf("failed to insert location: %w", err)
	}
	return nil
}
