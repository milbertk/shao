package blnotifications

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
)

// Customer cancellation REQUESTS (2026-10-23).
// The customer asks from the "manage my request" link. Nothing is cancelled
// automatically: the company receives an email with all the data and cancels
// manually in AlephTelos. The customer receives a short confirmation.

// Entity kinds (same codes as tbl_notification_log.entity_type).
const (
	EntityAppointment = "APPOINTMENT"
	EntityWebOrder    = "WEB_ORDER"
	EntityOrder       = "ORDER"
)

var (
	// ErrNoCompanyChannel: the company has no notification email or no working
	// SMTP, so the request could not reach it.
	ErrNoCompanyChannel = errors.New("the company can not receive cancellation requests right now")
)

// OrderStatusIsFinal says if an order can no longer be cancelled.
func OrderStatusIsFinal(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "DELIVERED", "DONE", "COMPLETED", "CANCELLED", "REJECTED":
		return true
	}
	return false
}

// cancelRequestData is what both emails need, loaded once.
type cancelRequestData struct {
	reference     string
	customerName  string
	customerPhone string
	customerEmail string
	status        string
	kindLabel     string
	details       string // appointment lines or items + total
	whatsappMsg   string
}

// NotifyCancelRequest emails the company (synchronously, so the customer
// knows it arrived) and then the customer (in the background).
// Returns ErrNoCompanyChannel when the company email could not be sent.
func NotifyCancelRequest(kind, companyID, entityID, reason string, requestedAt time.Time) error {
	companyID, entityID = strings.TrimSpace(companyID), strings.TrimSpace(entityID)
	if companyID == "" || entityID == "" {
		return errors.New("invalid cancellation request")
	}

	c := loadCompany(companyID)
	t := c.texts()

	data, err := loadCancelRequestData(c, kind, companyID, entityID)
	if err != nil {
		return err
	}

	link := manageLinkFor(kind, companyID, entityID)
	// On the company's clock (2026-11-04); the server runs in UTC.
	location := c.Location
	if location == nil {
		location = time.UTC
	}
	when := requestedAt.In(location).Format("2006-01-02 15:04")

	// 1) Company email: everything needed to cancel it by hand.
	coSubject := fmt.Sprintf(t.coCancelReqSubject, data.reference, data.customerName)
	coBody := t.hiCompany + "\n\n" + fmt.Sprintf(t.coCancelReqIntro, data.kindLabel) + "\n\n" +
		line(t.labelReference, data.reference) +
		line(t.labelCustomer, data.customerName) +
		line(t.labelPhone, data.customerPhone) +
		line(t.labelEmail, data.customerEmail) +
		line(t.labelRequestedAt, when) +
		line(t.labelCurrentStatus, statusLabel(c, data.status)) +
		line(t.reasonLabel, strings.TrimSpace(reason)) +
		"\n" + data.details

	status := deliver(c, kind, entityID, EventCancelRequested, AudienceCompany, c.NotificationEmail, coSubject, coBody)
	if status != "SENT" {
		return ErrNoCompanyChannel
	}

	// 2) Customer confirmation (best-effort, background).
	runAsync("cancel request "+kind, func() {
		subject := fmt.Sprintf(t.cancelReqSubject, data.reference)
		body := fmt.Sprintf(t.hi, data.customerName) + "\n\n" +
			fmt.Sprintf(t.cancelReqBody, data.kindLabel, data.reference, c.displayName()) + "\n" +
			footer(c, link, t.manageLink, data.whatsappMsg, false)
		deliver(c, kind, entityID, EventCancelRequested, AudienceCustomer, data.customerEmail, subject, body)
	})
	return nil
}

func loadCancelRequestData(c companyContext, kind, companyID, entityID string) (*cancelRequestData, error) {
	t := c.texts()
	d := &cancelRequestData{}

	switch kind {
	case EntityAppointment:
		a := appointment.AppointmentS{AppointmentID: entityID, CompanyID: companyID}
		if err := a.Select(); err != nil {
			return nil, fmt.Errorf("appointment not loaded: %w", err)
		}
		service, _ := productInfo(companyID, a.ServiceID)
		if service == "" {
			service = t.labelService
		}
		when := timeRange(c, a.StartHour, a.EndHour)
		d.reference = service + ", " + a.AppointmentDate + " " + when
		d.customerName = a.CustomerName
		d.customerPhone = deref(a.Phone)
		d.customerEmail = appointmentEmail(companyID, &a) // typed or linked customer (2026-10-30)
		d.status = a.Status
		d.kindLabel = t.kindAppointment
		d.details = line(t.labelService, service) +
			line(t.labelDate, a.AppointmentDate) +
			line(t.labelTime, timeRangeOn(c, a.AppointmentDate, a.StartHour, a.EndHour)) +
			line(t.labelComments, deref(a.Comments))
		d.whatsappMsg = fmt.Sprintf(t.waAboutBooking, d.reference)

	case EntityWebOrder:
		w, err := weborder.SelectWebOrderByID(companyID, entityID)
		if err != nil {
			return nil, fmt.Errorf("web order not loaded: %w", err)
		}
		items, _ := weborder.SelectWebOrderItems(companyID, entityID)
		d.reference = w.RequestCode
		d.customerName = w.CustomerName
		d.customerPhone = deref(w.Phone)
		d.customerEmail = webOrderEmail(w) // typed or registered user (2026-10-30)
		d.status = w.Status
		d.kindLabel = t.kindWebOrder
		d.details = line(t.labelDelivery, w.DeliveryMode) +
			line(t.labelAddress, deref(w.Address)) +
			line(t.labelNotes, deref(w.Notes)) + "\n" +
			itemsSummary(t, items, c.Currency) +
			line(t.labelTotal, formatMoney(w.Total, c.Currency))
		d.whatsappMsg = fmt.Sprintf(t.waAboutOrder, d.reference)

	case EntityOrder:
		o := order.OrderS{OrderInternalID: entityID, CompanyID: companyID}
		if err := o.Select(); err != nil {
			return nil, fmt.Errorf("order not loaded: %w", err)
		}
		d.reference = OrderReference(&o)
		d.customerName = deref(o.CustomerName)
		d.customerEmail = orderEmail(companyID, &o) // typed, customer, store request or user (2026-10-30)
		if contact, err := order.SelectOrderContact(companyID, entityID); err == nil && contact != nil {
			if d.customerName == "" {
				d.customerName = strings.TrimSpace(contact.Name)
			}
			if d.customerEmail == "" {
				d.customerEmail = strings.TrimSpace(contact.Email)
			}
			d.customerPhone = strings.TrimSpace(contact.Cellphone)
		}
		d.status = o.Status
		d.kindLabel = t.kindOrder
		d.details = orderItemsSummary(c, companyID, entityID) +
			line(t.labelTotal, formatMoney(o.Total, c.Currency)) +
			line(t.labelNotes, deref(o.Notes))
		d.whatsappMsg = fmt.Sprintf(t.waAboutOrder, d.reference)

	default:
		return nil, errors.New("invalid cancellation request")
	}

	if d.customerName == "" {
		d.customerName = "-"
	}
	return d, nil
}

// orderItemsSummary lists the products of an internal order.
func orderItemsSummary(c companyContext, companyID, orderInternalID string) string {
	items, err := order.SelectItemsByOrder(companyID, orderInternalID)
	if err != nil {
		log.Printf("[notifications] order items not loaded: %v", err)
		return ""
	}
	if len(items) == 0 {
		return ""
	}
	t := c.texts()
	var b strings.Builder
	b.WriteString(t.labelItems + ":\n")
	for _, it := range items {
		name, _ := productInfo(companyID, it.ProductID)
		if name == "" {
			name = "-"
		}
		b.WriteString(fmt.Sprintf("  • %s × %d\n", name, it.Quantity))
	}
	return b.String()
}

// manageLinkFor returns the customer's existing private link ("" if none).
func manageLinkFor(kind, companyID, entityID string) string {
	var token string
	var err error
	switch kind {
	case EntityAppointment:
		token, err = ensureAppointmentToken(companyID, entityID)
	case EntityWebOrder:
		token, err = ensureWebOrderToken(companyID, entityID)
	case EntityOrder:
		token, err = ensureOrderToken(companyID, entityID)
	}
	if err != nil {
		return ""
	}
	return manageLink(companyID, token)
}
