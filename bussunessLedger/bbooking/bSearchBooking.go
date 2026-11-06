package blbooking

import (
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/booking"
)

// BSearchBookings returns the booking configurations (headers) of ONE company
// matching the optional filters. companyID is authoritative.
func BSearchBookings(
	companyID string,
	filters booking.BookingConfigSearchFilters,
) ([]booking.BookingConfigS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	filters.CompanyID = companyID

	results, err := booking.SearchBookingConfigs(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search bookings: %w", err)
	}

	return results, nil
}
