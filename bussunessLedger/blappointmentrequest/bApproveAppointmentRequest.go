package blappointmentrequest

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
)

// BApproveAppointmentRequest moves a PENDING request to SCHEDULED (same status a
// locally-created appointment has) and emails the customer that it was accepted.
func BApproveAppointmentRequest(companyID, appointmentID string) (*appointment.AppointmentS, error) {
	companyID = strings.TrimSpace(companyID)
	appointmentID = strings.TrimSpace(appointmentID)
	if companyID == "" || appointmentID == "" {
		return nil, errors.New("companyId and appointmentId are required")
	}

	// Block double-processing: only a PENDING request can be approved.
	current, err := appointment.GetAppointmentStatus(companyID, appointmentID)
	if err == sql.ErrNoRows {
		return nil, errors.New("appointment request not found")
	}
	if err != nil {
		return nil, err
	}
	if strings.ToUpper(current) != "PENDING" {
		return nil, fmt.Errorf("this request is already %s and cannot be approved", strings.ToLower(current))
	}

	if _, err := appointment.UpdateAppointmentStatus(companyID, appointmentID, "SCHEDULED"); err != nil {
		return nil, err
	}

	// Load the full appointment for the response + email.
	item := appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID}
	if err := item.Select(); err != nil {
		// Status is already changed; return a minimal object rather than failing.
		return &appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID, Status: "SCHEDULED"}, nil
	}

	// Customer email in the company language, with the company SMTP.
	blnotifications.NotifyAppointment(companyID, appointmentID, blnotifications.EventApproved, "")
	return &item, nil
}

// BRejectAppointmentRequest moves a PENDING request to REJECTED (optionally with
// a reason email). No new appointment is created.
func BRejectAppointmentRequest(companyID, appointmentID, reason string) (*appointment.AppointmentS, error) {
	companyID = strings.TrimSpace(companyID)
	appointmentID = strings.TrimSpace(appointmentID)
	if companyID == "" || appointmentID == "" {
		return nil, errors.New("companyId and appointmentId are required")
	}

	current, err := appointment.GetAppointmentStatus(companyID, appointmentID)
	if err == sql.ErrNoRows {
		return nil, errors.New("appointment request not found")
	}
	if err != nil {
		return nil, err
	}
	if strings.ToUpper(current) != "PENDING" {
		return nil, fmt.Errorf("this request is already %s and cannot be rejected", strings.ToLower(current))
	}

	if _, err := appointment.UpdateAppointmentStatus(companyID, appointmentID, "REJECTED"); err != nil {
		return nil, err
	}

	item := appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID}
	if err := item.Select(); err != nil {
		return &appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID, Status: "REJECTED"}, nil
	}

	blnotifications.NotifyAppointment(companyID, appointmentID, blnotifications.EventRejected, strings.TrimSpace(reason))
	return &item, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}
