package blpublicshowcase

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/milbertk/shao/bussunessLedger/bllocations"
	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	blpageconfiguration "github.com/milbertk/shao/bussunessLedger/blpageconfiguration"
	"github.com/milbertk/shao/bussunessLedger/bltimezone"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
	"github.com/milbertk/shao/dataBasesLedger/notifications"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
	"github.com/milbertk/shao/dataBasesLedger/pageconfiguration"
	"github.com/milbertk/shao/dataBasesLedger/products"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
)

// Customer self-service ("manage my request" link sent by email).
// The token is the only key: it is long and random, and it never exposes
// internal ids to the page.

var (
	// ErrSelfServiceNotFound: unknown or invalid link.
	ErrSelfServiceNotFound = errors.New("this link is not valid")
	// ErrSelfServiceNotAllowed: the action is not possible in the current status.
	ErrSelfServiceNotAllowed = errors.New("this action is not possible anymore")
	// ErrSameTime: the customer picked the time they already have.
	ErrSameTime = errors.New("invalid time: choose a different time from the current one")
	// ErrCancelAlreadyRequested: a cancellation request was already sent.
	ErrCancelAlreadyRequested = errors.New("a cancellation request was already sent")
	// ErrCancelUnavailable: the company can not receive the request by email.
	ErrCancelUnavailable = errors.New("the company can not receive cancellation requests right now")
)

// maxCancelReasonLength limits the optional reason written by the customer.
const maxCancelReasonLength = 500

// minChangeNotice: appointments can be cancelled or moved only until this
// time before they start.
const minChangeNotice = 2 * time.Hour

const (
	KindAppointment = "APPOINTMENT"
	KindWebOrder    = "WEB_ORDER"
	KindOrder       = "ORDER"
)

// SelfServiceAppointment is the appointment part of the view.
type SelfServiceAppointment struct {
	ServiceID   string `json:"serviceId"` // public product id (needed to load free slots)
	ServiceName string `json:"serviceName"`
	Date        string `json:"date"`
	StartHour   string `json:"startHour"`
	EndHour     string `json:"endHour"`
	AllDay      bool   `json:"allDay"`
	// Location (2026-11-07); empty when the appointment has none.
	LocationID      string `json:"locationId,omitempty"`
	LocationName    string `json:"locationName,omitempty"`
	LocationAddress string `json:"locationAddress,omitempty"`
	MapsURL         string `json:"mapsUrl,omitempty"`
}

// SelfServiceItem is one line of a web order.
type SelfServiceItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Amount   float64 `json:"amount"`
}

// SelfServiceView is what the customer page shows.
type SelfServiceView struct {
	Kind           string                  `json:"kind"`
	CompanyID      string                  `json:"companyId"`
	CompanyName    string                  `json:"companyName"`
	Reference      string                  `json:"reference"`
	Status         string                  `json:"status"`
	CustomerName   string                  `json:"customerName"`
	Appointment    *SelfServiceAppointment `json:"appointment,omitempty"`
	Items          []SelfServiceItem       `json:"items,omitempty"`
	Total          *float64                `json:"total,omitempty"`
	CurrencySymbol string                  `json:"currencySymbol,omitempty"`
	// How to pay (web orders only; empty when the company has none).
	PaymentInstructions string `json:"paymentInstructions,omitempty"`
	// The customer can only REQUEST a cancellation; the company cancels manually.
	CanRequestCancel  bool   `json:"canRequestCancel"`
	CancelRequestedAt string `json:"cancelRequestedAt,omitempty"` // RFC 3339, when already requested
	CanReschedule     bool   `json:"canReschedule"`
	Whatsapp          string `json:"whatsapp,omitempty"`
	// Company contact for the "pending approval" pop-up (2026-11-08): customer
	// service email and social link, each only when set.
	ContactEmail   string `json:"contactEmail,omitempty"`
	SocialLink     string `json:"socialLink,omitempty"`
	SocialLinkType string `json:"socialLinkType,omitempty"`
	MinNoticeHours int    `json:"minNoticeHours"`
	// Time zone of the company (2026-11-04): the appointment hours are in it.
	TimeZone      string `json:"timeZone"`
	TimeZoneLabel string `json:"timeZoneLabel,omitempty"`
}

func tokenKind(token string) string {
	switch {
	case strings.HasPrefix(token, "a_"):
		return KindAppointment
	case strings.HasPrefix(token, "w_"):
		return KindWebOrder
	case strings.HasPrefix(token, "o_"):
		return KindOrder
	}
	return ""
}

func validToken(token string) bool {
	if len(token) < 20 || len(token) > 64 {
		return false
	}
	for _, r := range token {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' {
			return false
		}
	}
	return tokenKind(token) != ""
}

// companyPublic is the public name and contact of a company (2026-11-08:
// customer service email and social link added).
type companyPublic struct {
	Name, Whatsapp, Email, SocialLink, SocialLinkType string
}

func companyPublicInfo(companyID string) companyPublic {
	var info companyPublic
	page := pageconfiguration.PageConfigurationS{CompanyID: companyID}
	if err := page.SelectByCompany(); err == nil {
		info.Name = page.DisplayName
		info.Whatsapp = derefText(page.ContactWhatsapp)
		info.Email = derefText(blpageconfiguration.PublicContactEmail(page.CustomerServiceEmail))
		info.SocialLink = derefText(page.SocialLink)
		info.SocialLinkType = derefText(page.SocialLinkType)
	}
	return info
}

func derefText(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// appointmentStart returns the start time of the appointment on the
// COMPANY's clock (its time zone, 2026-11-04), so "until 2 hours before"
// is right for a company in any country while the server runs in UTC.
func appointmentStart(a *appointment.AppointmentS) (time.Time, bool) {
	start := hhmm(a.StartHour)
	if start == "" {
		start = allDayStart
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", a.AppointmentDate+" "+start, bltimezone.CompanyLocation(a.CompanyID))
	return t, err == nil
}

func appointmentCanChange(a *appointment.AppointmentS) bool {
	status := strings.ToUpper(a.Status)
	if status != "PENDING" && status != "SCHEDULED" {
		return false
	}
	start, ok := appointmentStart(a)
	return ok && start.Sub(bltimezone.Now()) > minChangeNotice
}

// appointmentCanRequestCancel: pending or scheduled, and not started yet.
func appointmentCanRequestCancel(a *appointment.AppointmentS) bool {
	status := strings.ToUpper(a.Status)
	if status != "PENDING" && status != "SCHEDULED" {
		return false
	}
	start, ok := appointmentStart(a)
	return ok && start.Sub(bltimezone.Now()) > 0
}

// BSelfServiceView returns the customer's request behind a token.
func BSelfServiceView(token string) (*SelfServiceView, error) {
	view, _, err := selfServiceLookup(token)
	return view, err
}

// selfServiceLookup builds the view and also returns the internal id of the
// record (never sent to the page).
func selfServiceLookup(token string) (*SelfServiceView, string, error) {
	token = strings.TrimSpace(token)
	if !validToken(token) {
		return nil, "", ErrSelfServiceNotFound
	}
	entityID := ""

	view := &SelfServiceView{Kind: tokenKind(token), MinNoticeHours: int(minChangeNotice.Hours())}

	switch view.Kind {
	case KindAppointment:
		a, err := appointment.SelectByPublicToken(token)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, "", ErrSelfServiceNotFound
			}
			return nil, "", err
		}
		view.CompanyID = a.CompanyID
		view.Status = strings.ToUpper(a.Status)
		view.CustomerName = a.CustomerName
		serviceName := ""
		p := products.ProductS{CompanyID: a.CompanyID, ProductInternalID: a.ServiceID}
		if err := p.Select(); err == nil {
			serviceName = p.Name
		}
		start, end := hhmm(a.StartHour), hhmm(a.EndHour)
		view.Appointment = &SelfServiceAppointment{
			ServiceID:   a.ServiceID,
			ServiceName: serviceName,
			Date:        a.AppointmentDate,
			StartHour:   start,
			EndHour:     end,
			AllDay:      start == allDayStart && end == allDayEnd,
		}
		if a.LocationID != nil && locationsOn(a.CompanyID) {
			view.Appointment.LocationID = *a.LocationID
			view.Appointment.LocationName = derefText(a.LocationName)
			view.Appointment.LocationAddress = derefText(a.LocationAddress)
			view.Appointment.MapsURL = bllocations.MapsURL(a.LocationAddress, a.LocationLatitude, a.LocationLongitude)
		}
		view.Reference = strings.TrimSpace(serviceName + " " + a.AppointmentDate)
		view.CanRequestCancel = appointmentCanRequestCancel(a)
		view.CanReschedule = appointmentCanChange(a)
		entityID = a.AppointmentID

	case KindWebOrder:
		companyID, id, err := weborder.FindByPublicToken(token)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, "", ErrSelfServiceNotFound
			}
			return nil, "", err
		}
		w, err := weborder.SelectWebOrderByID(companyID, id)
		if err != nil {
			return nil, "", err
		}
		items, _ := weborder.SelectWebOrderItems(companyID, id)
		view.CompanyID = companyID
		view.Reference = w.RequestCode
		view.Status = strings.ToUpper(w.Status)
		view.CustomerName = w.CustomerName
		// One currency per company (Company page) + how to pay.
		store := blpageconfiguration.BCompanyStoreSettings(companyID)
		view.CurrencySymbol = store.CurrencySymbol
		view.PaymentInstructions = store.PaymentInstructions
		for _, it := range items {
			view.Items = append(view.Items, SelfServiceItem{
				Name:     derefText(it.ProductName),
				Quantity: it.Quantity,
				Amount:   it.UnitPrice * float64(it.Quantity),
			})
		}
		total := w.Total
		view.Total = &total
		view.CanRequestCancel = view.Status == "PENDING" || view.Status == "APPROVED"
		entityID = id

	case KindOrder:
		companyID, id, err := order.FindByPublicToken(token)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, "", ErrSelfServiceNotFound
			}
			return nil, "", err
		}
		o := order.OrderS{OrderInternalID: id, CompanyID: companyID}
		if err := o.Select(); err != nil {
			return nil, "", err
		}
		view.CompanyID = companyID
		view.Reference = blnotifications.OrderReference(&o)
		view.Status = strings.ToUpper(o.Status)
		view.CustomerName = derefText(o.CustomerName)
		view.CanRequestCancel = !blnotifications.OrderStatusIsFinal(o.Status)
		entityID = id
	}

	// Already requested? (company email SENT for this record)
	if last, err := notifications.LastSentAt(view.CompanyID, view.Kind, entityID,
		blnotifications.EventCancelRequested, blnotifications.AudienceCompany); err == nil && last != nil {
		view.CancelRequestedAt = last.Format(time.RFC3339)
		view.CanRequestCancel = false
	}

	company := companyPublicInfo(view.CompanyID)
	view.CompanyName, view.Whatsapp = company.Name, company.Whatsapp
	view.ContactEmail, view.SocialLink, view.SocialLinkType = company.Email, company.SocialLink, company.SocialLinkType
	zone, label := bltimezone.CompanyZone(view.CompanyID)
	view.TimeZone, view.TimeZoneLabel = zone.String(), label
	return view, entityID, nil
}

// CancelRequestInput is the optional reason written by the customer.
type CancelRequestInput struct {
	Reason string `json:"reason"`
}

// BSelfServiceRequestCancel sends a cancellation REQUEST to the company by
// email. Nothing is cancelled here: the manager cancels it manually.
func BSelfServiceRequestCancel(token string, input CancelRequestInput) (*SelfServiceView, error) {
	view, entityID, err := selfServiceLookup(token)
	if err != nil {
		return nil, err
	}
	if view.CancelRequestedAt != "" {
		return nil, ErrCancelAlreadyRequested
	}
	if !view.CanRequestCancel || entityID == "" {
		return nil, ErrSelfServiceNotAllowed
	}

	reason := strings.TrimSpace(input.Reason)
	if runes := []rune(reason); len(runes) > maxCancelReasonLength {
		reason = string(runes[:maxCancelReasonLength])
	}

	requestedAt := time.Now()
	if err := blnotifications.NotifyCancelRequest(view.Kind, view.CompanyID, entityID, reason, requestedAt); err != nil {
		if errors.Is(err, blnotifications.ErrNoCompanyChannel) {
			return nil, ErrCancelUnavailable
		}
		return nil, fmt.Errorf("failed to send cancellation request: %w", err)
	}

	view.CancelRequestedAt = requestedAt.Format(time.RFC3339)
	view.CanRequestCancel = false
	return view, nil
}

// RescheduleInput is the new time chosen by the customer.
type RescheduleInput struct {
	AppointmentDate string `json:"appointmentDate"`
	StartHour       string `json:"startHour"`
	EndHour         string `json:"endHour"`
	LocationID      string `json:"locationId"` // location of the new time (2026-11-07, optional)
}

// BSelfServiceReschedule moves an appointment to a new free slot. The
// appointment goes back to PENDING so the company confirms the new time.
func BSelfServiceReschedule(token string, input RescheduleInput) (*SelfServiceView, error) {
	if !strings.HasPrefix(strings.TrimSpace(token), "a_") {
		return nil, ErrSelfServiceNotAllowed
	}
	view, err := BSelfServiceView(token)
	if err != nil {
		return nil, err
	}
	if !view.CanReschedule {
		return nil, ErrSelfServiceNotAllowed
	}

	a, err := appointment.SelectByPublicToken(token)
	if err != nil {
		return nil, err
	}

	date := strings.TrimSpace(input.AppointmentDate)
	if date == "" {
		return nil, errors.New("appointmentDate is required")
	}

	unlock := lockSlot(a.CompanyID, a.ServiceID, date)
	defer unlock()

	if id := strings.TrimSpace(input.LocationID); id != "" && !bllocations.ValidID(id) {
		return nil, errors.New("invalid location")
	}
	start, end, locationID, err := ValidateRequestedSlot(a.CompanyID, a.ServiceID, date, input.StartHour, input.EndHour, input.LocationID)
	if err != nil {
		return nil, err
	}
	newLocation := strOrNil(locationID)
	if !locationsOn(a.CompanyID) {
		// The company does not use locations (2026-11-08): keep the one saved.
		newLocation, locationID = a.LocationID, derefText(a.LocationID)
	}
	sameLocation := strings.EqualFold(locationID, derefText(a.LocationID))
	if date == a.AppointmentDate && start == hhmm(a.StartHour) && end == hhmm(a.EndHour) && sameLocation {
		return nil, ErrSameTime
	}

	if err := appointment.MoveToNewTime(a.CompanyID, a.AppointmentID, date, start, end, newLocation); err != nil {
		return nil, err
	}
	blnotifications.NotifyAppointment(a.CompanyID, a.AppointmentID, blnotifications.EventRescheduleRequested, "")

	return BSelfServiceView(token)
}

// locationsOn tells if the company uses locations (2026-11-08). On a read
// error the answer is "no" (nothing about locations is shown).
func locationsOn(companyID string) bool {
	on, err := bllocations.LocationsEnabled(companyID)
	return err == nil && on
}
