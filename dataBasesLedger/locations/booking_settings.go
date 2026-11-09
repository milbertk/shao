package locations

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// BookingSettingsS is the row of public.tbl_company_booking_settings
// (2026-11-07). No row = default values.
type BookingSettingsS struct {
	CompanyID string `json:"companyId"`
	// LocationsEnabled (2026-11-08): "We use locations". FALSE (default) =
	// AlephK works as before locations: no location in schedules, forms, the
	// store or emails (the saved locations stay for later).
	LocationsEnabled bool `json:"locationsEnabled"`
	// LocationsSameTime: TRUE = the company can attend in more than one
	// location at the same time (several people / offices). FALSE (default)
	// = one location at a time: schedules can not have the same hours in two
	// different locations.
	LocationsSameTime bool       `json:"locationsSameTime"`
	UpdatedAt         *time.Time `json:"updatedAt,omitempty"`
}

// SelectBookingSettings returns the settings of the company (defaults when
// it has no row yet).
func SelectBookingSettings(companyID string) (BookingSettingsS, error) {
	settings := BookingSettingsS{CompanyID: strings.TrimSpace(companyID)}
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return settings, fmt.Errorf("database connection error: %w", err)
	}

	var updatedAt time.Time
	err = db.QueryRow(`
		SELECT locations_enabled, locations_same_time, updated_at
		FROM public.tbl_company_booking_settings
		WHERE company_id = $1
	`, settings.CompanyID).Scan(&settings.LocationsEnabled, &settings.LocationsSameTime, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return settings, nil
		}
		return settings, fmt.Errorf("failed to select booking settings: %w", err)
	}
	settings.UpdatedAt = &updatedAt
	return settings, nil
}

// SaveBookingSettings creates or updates the settings of the company.
func SaveBookingSettings(companyID string, locationsEnabled, locationsSameTime bool) (BookingSettingsS, error) {
	settings := BookingSettingsS{
		CompanyID:         strings.TrimSpace(companyID),
		LocationsEnabled:  locationsEnabled,
		LocationsSameTime: locationsSameTime,
	}
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return settings, fmt.Errorf("database connection error: %w", err)
	}

	var updatedAt time.Time
	err = db.QueryRow(`
		INSERT INTO public.tbl_company_booking_settings (company_id, locations_enabled, locations_same_time, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (company_id) DO UPDATE
		SET locations_enabled = EXCLUDED.locations_enabled,
		    locations_same_time = EXCLUDED.locations_same_time,
		    updated_at = now()
		RETURNING updated_at
	`, settings.CompanyID, locationsEnabled, locationsSameTime).Scan(&updatedAt)
	if err != nil {
		return settings, fmt.Errorf("failed to save booking settings: %w", err)
	}
	settings.UpdatedAt = &updatedAt
	return settings, nil
}
