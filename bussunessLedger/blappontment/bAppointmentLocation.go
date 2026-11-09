package blappointment

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/bussunessLedger/bllocations"
	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
)

// ErrAppointmentClosed: a cancelled or rejected appointment keeps its location.
var ErrAppointmentClosed = errors.New("invalid: a cancelled or rejected appointment can not change its location")

// AppointmentLocationResult is the answer of BSetAppointmentLocation.
type AppointmentLocationResult struct {
	Appointment appointment.AppointmentS `json:"appointment"`
	// OtherLocationCount: appointments at the same time in ANOTHER location
	// while the company attends in one location at a time (a warning).
	OtherLocationCount int `json:"otherLocationCount"`
}

// BSetAppointmentLocation changes only the location of an appointment or a
// booking request (2026-11-08): from Booking requests (before approving) and
// from Appointments. "" = no location. The location must belong to the
// company and be active (the current one may stay even if disabled). A
// scheduled appointment sends the customer the "appointment changed" email.
func BSetAppointmentLocation(companyID, appointmentID, locationID string) (*AppointmentLocationResult, error) {
	companyID, appointmentID = strings.TrimSpace(companyID), strings.TrimSpace(appointmentID)
	if companyID == "" || appointmentID == "" {
		return nil, errors.New("companyId and appointmentId are required")
	}
	current := appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID}
	if err := current.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAppointmentNotFound
		}
		return nil, fmt.Errorf("failed to load appointment: %w", err)
	}
	status := strings.ToUpper(strings.TrimSpace(current.Status))
	if status == "CANCELLED" || status == "REJECTED" {
		return nil, ErrAppointmentClosed
	}

	id := bllocations.NormalizeID(&locationID)
	location, err := bllocations.ValidateLocationForCompany(companyID, id, false)
	if err != nil {
		return nil, err
	}
	var newID *string
	if location != nil {
		sameAsNow := current.LocationID != nil && strings.EqualFold(*current.LocationID, location.LocationID)
		if !location.Active && !sameAsNow {
			return nil, bllocations.ErrLocationDisabled
		}
		newID = &location.LocationID
	}

	before := blnotifications.SnapshotAppointment(current)
	if _, err := appointment.UpdateAppointmentLocation(companyID, appointmentID, newID); err != nil {
		return nil, err
	}
	saved := appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID}
	if err := saved.Select(); err != nil {
		return nil, fmt.Errorf("failed to reload appointment: %w", err)
	}

	result := &AppointmentLocationResult{Appointment: saved}
	if newID != nil {
		if active, err := bllocations.OneLocationAtATime(companyID); err == nil && active {
			if n, err := appointment.CountOtherLocationConflicts(companyID, saved.AppointmentDate,
				saved.StartHour, saved.EndHour, *newID, appointmentID); err == nil {
				result.OtherLocationCount = n
			}
		}
	}

	// The customer is told only about a scheduled appointment; a pending
	// request shows the location in the approval email.
	if status == "SCHEDULED" && blnotifications.AppointmentWhenChanged(before, blnotifications.SnapshotAppointment(saved)) {
		blnotifications.NotifyAppointmentModified(companyID, appointmentID, before)
	}
	return result, nil
}
