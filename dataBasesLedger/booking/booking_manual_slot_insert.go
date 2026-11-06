package booking

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// BookingManualSlotS represents one row from tbl_booking_manual_slot.
type BookingManualSlotS struct {
	ManualSlotID    string    `json:"manualSlotId"`
	BookingConfigID string    `json:"bookingConfigId"`
	CompanyID       string    `json:"companyId"`
	SlotDate        string    `json:"slotDate"` // "YYYY-MM-DD"
	AllDay          bool      `json:"allDay"`
	StartTime       *string   `json:"startTime,omitempty"` // "HH:MM"
	EndTime         *string   `json:"endTime,omitempty"`   // "HH:MM"
	Appointments    *int      `json:"appointments,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Insert creates one manual slot for a booking config.
func (a *BookingManualSlotS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		INSERT INTO public.tbl_booking_manual_slot (
			booking_config_id,
			company_id,
			slot_date,
			all_day,
			start_time,
			end_time,
			appointments
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING manual_slot_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.BookingConfigID,
		a.CompanyID,
		a.SlotDate,
		a.AllDay,
		a.StartTime,
		a.EndTime,
		a.Appointments,
	).Scan(
		&a.ManualSlotID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert manual slot: %w", err)
	}

	return nil
}
