package blbooking

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/booking"
)

// ErrBookingNotFound lets handlers distinguish a missing booking config from a
// real database error.
var ErrBookingNotFound = errors.New("booking configuration not found")

// BookingBundle groups a booking header with its schedules. It is used both as
// the input for saving and as the result of a full read.
type BookingBundle struct {
	Config          booking.BookingConfigS            `json:"config"`
	RecurringRanges []booking.BookingRecurringRangeS  `json:"recurringRanges"`
	ManualSlots     []booking.BookingManualSlotS      `json:"manualSlots"`
}

// BSelectBookingByID loads one booking header, scoped to the company.
func BSelectBookingByID(companyID, bookingConfigID string) (*booking.BookingConfigS, error) {
	companyID = strings.TrimSpace(companyID)
	bookingConfigID = strings.TrimSpace(bookingConfigID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if bookingConfigID == "" {
		return nil, errors.New("bookingConfigId is required")
	}

	config := booking.BookingConfigS{
		BookingConfigID: bookingConfigID,
		CompanyID:       companyID,
	}

	if err := config.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, fmt.Errorf("failed to select booking config: %w", err)
	}

	return &config, nil
}

// BSelectRecurringRanges returns the recurring (weekly) ranges of a booking.
func BSelectRecurringRanges(companyID, bookingConfigID string) ([]booking.BookingRecurringRangeS, error) {
	ranges, err := booking.SelectRecurringRangesByConfig(companyID, bookingConfigID)
	if err != nil {
		return nil, fmt.Errorf("failed to select recurring ranges: %w", err)
	}
	return ranges, nil
}

// BSelectManualSlots returns the manual (specific date) slots of a booking.
func BSelectManualSlots(companyID, bookingConfigID string) ([]booking.BookingManualSlotS, error) {
	slots, err := booking.SelectManualSlotsByConfig(companyID, bookingConfigID)
	if err != nil {
		return nil, fmt.Errorf("failed to select manual slots: %w", err)
	}
	return slots, nil
}

// BSelectBookingFull returns the header together with both schedule lists.
func BSelectBookingFull(companyID, bookingConfigID string) (*BookingBundle, error) {
	config, err := BSelectBookingByID(companyID, bookingConfigID)
	if err != nil {
		return nil, err
	}

	ranges, err := BSelectRecurringRanges(companyID, bookingConfigID)
	if err != nil {
		return nil, err
	}

	slots, err := BSelectManualSlots(companyID, bookingConfigID)
	if err != nil {
		return nil, err
	}

	return &BookingBundle{
		Config:          *config,
		RecurringRanges: ranges,
		ManualSlots:     slots,
	}, nil
}
