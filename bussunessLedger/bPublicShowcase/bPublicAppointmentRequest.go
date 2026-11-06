package blpublicshowcase

import (
	"errors"
	"fmt"
	"strings"

	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
)

// PublicAppointmentRequest is the payload a guest sends from the showcase to
// request an appointment. The company comes from the URL (not the body).
type PublicAppointmentRequest struct {
	ServiceID       string `json:"serviceId"`
	CustomerName    string `json:"customerName"`
	Phone           string `json:"phone"`
	Email           string `json:"email"`
	SocialLink      string `json:"socialLink"`
	AppointmentDate string `json:"appointmentDate"` // YYYY-MM-DD
	StartHour       string `json:"startHour"`       // HH:MM
	EndHour         string `json:"endHour"`         // HH:MM
	Comments        string `json:"comments"`
}

func strOrNil(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

// BCreatePublicAppointmentRequest saves a guest appointment request as PENDING
// (for the provider to approve) after checking ON THE SERVER that the time is
// a real, free slot of the service (see ValidateRequestedSlot). Then it
// notifies the customer and the company (background, company SMTP).
func BCreatePublicAppointmentRequest(companyID string, req PublicAppointmentRequest) (*appointment.AppointmentS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if strings.TrimSpace(req.ServiceID) == "" {
		return nil, errors.New("serviceId is required")
	}
	if strings.TrimSpace(req.CustomerName) == "" {
		return nil, errors.New("customerName is required")
	}
	if strings.TrimSpace(req.AppointmentDate) == "" {
		return nil, errors.New("appointmentDate is required")
	}
	// Formats and sizes checked on the server (2026-10-25, security S4).
	if err := ValidatePublicAppointmentRequest(req); err != nil {
		return nil, err
	}

	serviceID := strings.TrimSpace(req.ServiceID)
	date := strings.TrimSpace(req.AppointmentDate)

	// One request at a time per service + date, so two visitors cannot take
	// the last place of the same slot at the same moment.
	unlock := lockSlot(companyID, serviceID, date)
	defer unlock()

	startHour, endHour, err := ValidateRequestedSlot(
		companyID, serviceID, date, req.StartHour, req.EndHour,
	)
	if err != nil {
		return nil, err
	}

	item := appointment.AppointmentS{
		CompanyID:             companyID,
		ServiceID:             serviceID,
		UseRegisteredCustomer: false, // it's a guest from the showcase
		CustomerID:            nil,
		CustomerName:          strings.TrimSpace(req.CustomerName),
		Phone:                 strOrNil(req.Phone),
		Email:                 strOrNil(req.Email),
		SocialLink:            strOrNil(req.SocialLink),
		AppointmentDate:       date,
		StartHour:             startHour,
		EndHour:               endHour,
		Status:                "PENDING", // waiting for the provider to approve
		Comments:              strOrNil(req.Comments),
		RegisteredBy:          strOrNil("showcase-guest"),
	}

	if err := item.Insert(); err != nil {
		return nil, fmt.Errorf("failed to create appointment request: %w", err)
	}

	// Customer "we received it" + company "new request" (background).
	blnotifications.NotifyAppointment(companyID, item.AppointmentID, blnotifications.EventRequested, "")

	return &item, nil
}
