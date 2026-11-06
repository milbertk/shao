package blappointmentmgmt

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
)

// Allowed target statuses from this management screen.
var allowedTargets = map[string]bool{
	"DONE":      true,
	"CANCELLED": true,
	"SCHEDULED": true, // allow re-opening a cancelled one if needed
}

// BChangeAppointmentStatus changes a real appointment's status (finish / cancel).
// It never touches PENDING/REJECTED (those belong to the request-approval flow).
func BChangeAppointmentStatus(companyID, appointmentID, newStatus string) (*appointment.AppointmentS, error) {
	companyID = strings.TrimSpace(companyID)
	appointmentID = strings.TrimSpace(appointmentID)
	newStatus = strings.ToUpper(strings.TrimSpace(newStatus))

	if companyID == "" || appointmentID == "" {
		return nil, errors.New("companyId and appointmentId are required")
	}
	if !allowedTargets[newStatus] {
		return nil, errors.New("invalid target status")
	}

	current, err := appointment.GetAppointmentStatus(companyID, appointmentID)
	if err == sql.ErrNoRows {
		return nil, errors.New("appointment not found")
	}
	if err != nil {
		return nil, err
	}
	cur := strings.ToUpper(current)

	// Guard: this screen manages real appointments, not pending/rejected requests.
	if cur == "PENDING" || cur == "REJECTED" {
		return nil, fmt.Errorf("this appointment is %s and is managed from booking requests", strings.ToLower(cur))
	}

	if _, err := appointment.UpdateAppointmentStatus(companyID, appointmentID, newStatus); err != nil {
		return nil, err
	}

	// Tell the customer when the status really changed.
	if cur != newStatus {
		switch newStatus {
		case "DONE":
			blnotifications.NotifyAppointment(companyID, appointmentID, blnotifications.EventDone, "")
		case "CANCELLED":
			blnotifications.NotifyAppointment(companyID, appointmentID, blnotifications.EventCancelled, "")
		case "SCHEDULED":
			blnotifications.NotifyAppointment(companyID, appointmentID, blnotifications.EventReopened, "")
		}
	}

	item := appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID}
	if err := item.Select(); err != nil {
		return &appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID, Status: newStatus}, nil
	}
	return &item, nil
}
