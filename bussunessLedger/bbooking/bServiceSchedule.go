package blbooking

import (
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/booking"
)

// ServiceScheduleExistsError means the service already has a booking
// schedule (the rule is ONE schedule per service). It carries the existing
// schedule so the screen can offer to open it.
type ServiceScheduleExistsError struct {
	BookingConfigID string `json:"existingBookingConfigId"`
	Name            string `json:"existingName"`
}

func (e *ServiceScheduleExistsError) Error() string {
	return fmt.Sprintf("this service already has a booking schedule (%s)", e.Name)
}

// findScheduleByService returns the booking schedule linked to the service in
// this company, or nil when the service has none.
func findScheduleByService(companyID, serviceProductID string) (*booking.BookingConfigS, error) {
	list, err := booking.SearchBookingConfigs(booking.BookingConfigSearchFilters{
		CompanyID:        companyID,
		ServiceProductID: strings.TrimSpace(serviceProductID),
		Limit:            1,
	})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return &list[0], nil
}

// ensureServiceIsFree returns ServiceScheduleExistsError when ANOTHER
// schedule (not the one being saved) already uses the service.
func ensureServiceIsFree(companyID, serviceProductID, savingConfigID string) error {
	existing, err := findScheduleByService(companyID, serviceProductID)
	if err != nil {
		return fmt.Errorf("failed to check the service schedule: %w", err)
	}
	if existing != nil && existing.BookingConfigID != strings.TrimSpace(savingConfigID) {
		return &ServiceScheduleExistsError{
			BookingConfigID: existing.BookingConfigID,
			Name:            existing.Name,
		}
	}
	return nil
}

// scheduleConflictOr turns a database unique violation (two schedules for one
// service, e.g. two users saving at the same time) into the friendly
// ServiceScheduleExistsError; any other error is returned unchanged.
func scheduleConflictOr(companyID, serviceProductID string, err error) error {
	if err == nil || !isUniqueViolation(err) {
		return err
	}
	existing, findErr := findScheduleByService(companyID, serviceProductID)
	if findErr != nil || existing == nil {
		return err
	}
	return &ServiceScheduleExistsError{
		BookingConfigID: existing.BookingConfigID,
		Name:            existing.Name,
	}
}

func isUniqueViolation(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "unique constraint") ||
		strings.Contains(message, "23505")
}

// IsServiceScheduleExists reports whether err is a ServiceScheduleExistsError
// and returns it.
func IsServiceScheduleExists(err error) (*ServiceScheduleExistsError, bool) {
	var exists *ServiceScheduleExistsError
	if errors.As(err, &exists) {
		return exists, true
	}
	return nil, false
}
