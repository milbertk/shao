package blappointment

import (
	"errors"
	"fmt"
	"strings"

	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
)

// BDeleteAppointment hard-deletes an appointment, scoped to the company.
// Returns ErrAppointmentNotFound if nothing was deleted.
func BDeleteAppointment(companyID, appointmentID string) error {
	companyID = strings.TrimSpace(companyID)
	appointmentID = strings.TrimSpace(appointmentID)
	if companyID == "" {
		return errors.New("companyId is required")
	}
	if appointmentID == "" {
		return errors.New("appointmentId is required")
	}

	deleted, err := appointment.Delete(companyID, appointmentID)
	if err != nil {
		return fmt.Errorf("failed to delete appointment: %w", err)
	}
	if deleted == 0 {
		return ErrAppointmentNotFound
	}

	return nil
}
