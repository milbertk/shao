package blappointment

import (
	"errors"
	"fmt"
	"strings"

	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
)

// BSearchAppointments returns the appointments of ONE company matching the
// filters. companyID is authoritative.
func BSearchAppointments(
	companyID string,
	filters appointment.AppointmentSearchFilters,
) ([]appointment.AppointmentS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	filters.CompanyID = companyID

	results, err := appointment.SearchAppointments(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search appointments: %w", err)
	}

	return results, nil
}
