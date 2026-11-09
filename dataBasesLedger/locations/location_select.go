package locations

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const locationColumns = `
	location_id,
	company_id,
	name,
	address,
	details,
	latitude,
	longitude,
	active,
	created_at,
	updated_at
`

func scanLocation(row interface {
	Scan(dest ...any) error
}, l *CompanyLocationS) error {
	return row.Scan(
		&l.LocationID,
		&l.CompanyID,
		&l.Name,
		&l.Address,
		&l.Details,
		&l.Latitude,
		&l.Longitude,
		&l.Active,
		&l.CreatedAt,
		&l.UpdatedAt,
	)
}

// Select loads one location by primary key, scoped to its company.
// Returns sql.ErrNoRows (wrapped) when it does not exist in the company.
func (l *CompanyLocationS) Select() error {
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
		SELECT ` + locationColumns + `
		FROM public.tbl_company_location
		WHERE location_id::text = $1
		  AND company_id = $2
	`
	if err := scanLocation(db.QueryRow(query, locationID, companyID), l); err != nil {
		return fmt.Errorf("failed to select location: %w", err)
	}
	return nil
}

// SelectLocationsByCompany returns the locations of one company ordered by
// name. includeInactive=false returns only the active ones.
func SelectLocationsByCompany(companyID string, includeInactive bool) ([]CompanyLocationS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	query := `
		SELECT ` + locationColumns + `
		FROM public.tbl_company_location
		WHERE company_id = $1
		  AND ($2 OR active = TRUE)
		ORDER BY lower(name), created_at
	`
	rows, err := db.Query(query, companyID, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("failed to select locations: %w", err)
	}
	defer rows.Close()

	list := make([]CompanyLocationS, 0)
	for rows.Next() {
		var item CompanyLocationS
		if err := scanLocation(rows, &item); err != nil {
			return nil, fmt.Errorf("failed to scan location: %w", err)
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading locations: %w", err)
	}
	return list, nil
}

// NameExists tells if another location of the company already has this name
// (upper/lower case and spaces ignored). excludeID skips the location being
// edited.
func NameExists(companyID, name, excludeID string) (bool, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return false, fmt.Errorf("database connection error: %w", err)
	}

	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM public.tbl_company_location
			WHERE company_id = $1
			  AND lower(btrim(name)) = lower(btrim($2))
			  AND ($3 = '' OR location_id::text <> $3)
		)
	`, strings.TrimSpace(companyID), name, strings.TrimSpace(excludeID)).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check location name: %w", err)
	}
	return exists, nil
}

// CountLocations returns how many locations (active or not) a company has.
func CountLocations(companyID string) (int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM public.tbl_company_location WHERE company_id = $1
	`, strings.TrimSpace(companyID)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count locations: %w", err)
	}
	return count, nil
}
