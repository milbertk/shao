package blappointment

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
)

// ErrAppointmentNotFound distinguishes a missing appointment from a real error.
var ErrAppointmentNotFound = errors.New("appointment not found")

// BSelectAppointmentByID loads one appointment, scoped to the company.
func BSelectAppointmentByID(companyID, appointmentID string) (*appointment.AppointmentS, error) {
	companyID = strings.TrimSpace(companyID)
	appointmentID = strings.TrimSpace(appointmentID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if appointmentID == "" {
		return nil, errors.New("appointmentId is required")
	}

	a := appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID}
	if err := a.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAppointmentNotFound
		}
		return nil, fmt.Errorf("failed to select appointment: %w", err)
	}

	return &a, nil
}
