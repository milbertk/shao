package blnotifications

import (
	"fmt"
	"strings"

	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
	emailnotifier "github.com/milbertk/shao/shared"
)

// Event names (also stored in tbl_notification_log.event).
const (
	EventRequested           = "REQUESTED"            // customer asked from the store
	EventCreated             = "CREATED"              // created by the company / web order received
	EventApproved            = "APPROVED"             // request accepted
	EventRejected            = "REJECTED"             // request rejected
	EventDone                = "DONE"                 // appointment finished
	EventCancelled           = "CANCELLED"            // cancelled by the company
	EventReopened            = "REOPENED"             // scheduled again
	EventCustomerCancelled   = "CUSTOMER_CANCELLED"   // cancelled by the customer (link)
	EventRescheduleRequested = "RESCHEDULE_REQUESTED" // customer asked another time
	EventStatusChanged       = "STATUS_CHANGED"       // order status changed
	EventCancelRequested     = "CANCEL_REQUESTED"     // customer asked to cancel (link); the company cancels manually
)

// ---------------------------------------------------------------------------
// Appointments
// ---------------------------------------------------------------------------

// NotifyAppointment emails the customer (and the company when relevant)
// about an appointment event. Runs in the background.
func NotifyAppointment(companyID, appointmentID, event, reason string) {
	companyID, appointmentID = strings.TrimSpace(companyID), strings.TrimSpace(appointmentID)
	if companyID == "" || appointmentID == "" {
		return
	}
	runAsync("appointment "+event, func() {
		c := loadCompany(companyID)
		t := c.texts()

		a := appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID}
		if err := a.Select(); err != nil {
			logNotSent(companyID, "APPOINTMENT", appointmentID, event, "appointment not loaded: "+err.Error())
			return
		}

		link := ""
		if token, err := ensureAppointmentToken(companyID, appointmentID); err == nil {
			link = manageLink(companyID, token)
		}

		service, _ := productInfo(companyID, a.ServiceID)
		if service == "" {
			service = t.labelService
		}
		when := timeRangeOn(c, a.AppointmentDate, a.StartHour, a.EndHour)
		date := a.AppointmentDate
		company := c.displayName()

		var subject, body string
		switch event {
		case EventRequested:
			subject = fmt.Sprintf(t.apptRequestedSubject, company)
			body = fmt.Sprintf(t.apptRequestedBody, service, date, when, company)
		case EventCreated:
			subject = fmt.Sprintf(t.apptCreatedSubject, company)
			body = fmt.Sprintf(t.apptCreatedBody, service, date, when)
		case EventApproved:
			subject = fmt.Sprintf(t.apptApprovedSubject, company)
			body = fmt.Sprintf(t.apptApprovedBody, service, date, when)
		case EventRejected:
			subject = fmt.Sprintf(t.apptRejectedSubject, company)
			body = fmt.Sprintf(t.apptRejectedBody, service, date, when)
			if strings.TrimSpace(reason) != "" {
				body += "\n" + t.reasonLabel + ": " + strings.TrimSpace(reason)
			}
		case EventDone:
			subject = fmt.Sprintf(t.apptDoneSubject, company)
			body = fmt.Sprintf(t.apptDoneBody, service, date)
			link = "" // nothing left to manage
		case EventCancelled:
			subject = fmt.Sprintf(t.apptCancelledSubject, company)
			body = fmt.Sprintf(t.apptCancelledBody, service, date, when)
		case EventReopened:
			subject = fmt.Sprintf(t.apptReopenedSubject, company)
			body = fmt.Sprintf(t.apptReopenedBody, service, date, when)
		case EventCustomerCancelled:
			subject = fmt.Sprintf(t.apptCustCancelSubject, company)
			body = fmt.Sprintf(t.apptCustCancelBody, service, date, when)
		case EventRescheduleRequested:
			subject = fmt.Sprintf(t.apptReschedSubject, company)
			body = fmt.Sprintf(t.apptReschedBody, service, date, when)
		default:
			return
		}

		reference := service + ", " + date + " " + timeRange(c, a.StartHour, a.EndHour)
		// Small "request a cancellation" line while the appointment is still ahead.
		withCancel := event == EventRequested || event == EventCreated || event == EventApproved ||
			event == EventReopened || event == EventRescheduleRequested
		customerBody := fmt.Sprintf(t.hi, a.CustomerName) + "\n\n" + body + "\n" +
			footer(c, link, t.manageLink, fmt.Sprintf(t.waAboutBooking, reference), withCancel)
		deliver(c, "APPOINTMENT", appointmentID, event, AudienceCustomer, appointmentEmail(companyID, &a), subject, customerBody)

		// Company copy for what the company must act on.
		var coSubject, coIntro string
		switch event {
		case EventRequested:
			coSubject = fmt.Sprintf(t.coApptRequestedSubject, service, date)
			coIntro = t.coApptIntro + " " + t.coApprovePending
		case EventCustomerCancelled:
			coSubject = fmt.Sprintf(t.coApptCustCancelSubject, a.CustomerName, date)
			coIntro = t.coApptCancelIntro
		case EventRescheduleRequested:
			coSubject = fmt.Sprintf(t.coApptReschedSubject, a.CustomerName, date)
			coIntro = t.coApptReschedIntro
		default:
			return
		}
		coBody := t.hiCompany + "\n\n" + coIntro + "\n\n" +
			line(t.labelService, service) +
			line(t.labelDate, date) +
			line(t.labelTime, when) +
			line(t.labelCustomer, a.CustomerName) +
			line(t.labelPhone, deref(a.Phone)) +
			line(t.labelEmail, deref(a.Email)) +
			line(t.labelComments, deref(a.Comments))
		deliver(c, "APPOINTMENT", appointmentID, event, AudienceCompany, c.NotificationEmail, coSubject, coBody)
	})
}

// ---------------------------------------------------------------------------
// Web orders (store cart requests)
// ---------------------------------------------------------------------------

// NotifyWebOrder emails the customer (and the company when relevant) about a
// web order event. Runs in the background.
func NotifyWebOrder(companyID, webOrderRequestID, event string) {
	companyID, webOrderRequestID = strings.TrimSpace(companyID), strings.TrimSpace(webOrderRequestID)
	if companyID == "" || webOrderRequestID == "" {
		return
	}
	runAsync("web order "+event, func() {
		c := loadCompany(companyID)
		t := c.texts()

		w, err := weborder.SelectWebOrderByID(companyID, webOrderRequestID)
		if err != nil {
			logNotSent(companyID, "WEB_ORDER", webOrderRequestID, event, "web order not loaded: "+err.Error())
			return
		}
		items, _ := weborder.SelectWebOrderItems(companyID, webOrderRequestID)

		// One currency for the whole store (Company page).
		symbol := c.Currency

		link := ""
		if token, err := ensureWebOrderToken(companyID, webOrderRequestID); err == nil {
			link = manageLink(companyID, token)
		}

		ref := w.RequestCode
		company := c.displayName()

		var subject, body string
		switch event {
		case EventCreated:
			subject = fmt.Sprintf(t.webCreatedSubject, ref, company)
			body = fmt.Sprintf(t.webCreatedBody, ref, company)
		case EventApproved:
			subject = fmt.Sprintf(t.webApprovedSubject, ref)
			body = fmt.Sprintf(t.webApprovedBody, ref)
		case EventRejected:
			subject = fmt.Sprintf(t.webRejectedSubject, ref)
			body = fmt.Sprintf(t.webRejectedBody, ref)
		case EventCustomerCancelled:
			subject = fmt.Sprintf(t.webCustCancelSubject, ref)
			body = fmt.Sprintf(t.webCustCancelBody, ref)
		default:
			return
		}

		summary := itemsSummary(t, items, symbol) + line(t.labelTotal, formatMoney(w.Total, symbol))
		payment := ""
		if event == EventCreated || event == EventApproved {
			payment = paymentBlock(c)
		}
		withCancel := event == EventCreated || event == EventApproved
		customerBody := fmt.Sprintf(t.hi, w.CustomerName) + "\n\n" + body + "\n\n" + summary + payment +
			footer(c, link, t.manageLink, fmt.Sprintf(t.waAboutOrder, ref), withCancel)
		deliver(c, "WEB_ORDER", webOrderRequestID, event, AudienceCustomer, webOrderEmail(w), subject, customerBody)

		var coSubject, coIntro string
		switch event {
		case EventCreated:
			coSubject = fmt.Sprintf(t.coWebCreatedSubject, ref, w.CustomerName)
			coIntro = t.coWebCreatedIntro
		case EventCustomerCancelled:
			coSubject = fmt.Sprintf(t.coWebCustCancelSubject, ref)
			coIntro = fmt.Sprintf(t.coWebCustCancelIntro, ref, w.CustomerName)
		default:
			return
		}
		coBody := t.hiCompany + "\n\n" + coIntro + "\n\n" +
			line(t.labelReference, ref) +
			line(t.labelCustomer, w.CustomerName) +
			line(t.labelPhone, deref(w.Phone)) +
			line(t.labelEmail, deref(w.Email)) +
			line(t.labelDelivery, w.DeliveryMode) +
			line(t.labelAddress, deref(w.Address)) +
			line(t.labelNotes, deref(w.Notes)) + "\n" + summary
		deliver(c, "WEB_ORDER", webOrderRequestID, event, AudienceCompany, c.NotificationEmail, coSubject, coBody)
	})
}

func itemsSummary(t textSet, items []weborder.WebOrderRequestItemS, symbol string) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(t.labelItems + ":\n")
	for _, it := range items {
		name := deref(it.ProductName)
		if name == "" {
			name = "-"
		}
		b.WriteString(fmt.Sprintf("  • %s × %d — %s\n", name, it.Quantity, formatMoney(it.UnitPrice*float64(it.Quantity), symbol)))
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Orders (internal orders and orders created from approved web orders)
// ---------------------------------------------------------------------------

// NotifyOrder emails the customer about an order event (CREATED or
// STATUS_CHANGED). Runs in the background.
func NotifyOrder(companyID, orderInternalID, event string) {
	companyID, orderInternalID = strings.TrimSpace(companyID), strings.TrimSpace(orderInternalID)
	if companyID == "" || orderInternalID == "" {
		return
	}
	runAsync("order "+event, func() {
		c := loadCompany(companyID)
		t := c.texts()

		o := order.OrderS{OrderInternalID: orderInternalID, CompanyID: companyID}
		if err := o.Select(); err != nil {
			logNotSent(companyID, "ORDER", orderInternalID, event, "order not loaded: "+err.Error())
			return
		}

		ref := OrderReference(&o)
		link := ""
		if token, err := ensureOrderToken(companyID, orderInternalID); err == nil {
			link = manageLink(companyID, token)
		}

		var subject, body string
		switch event {
		case EventCreated:
			subject = fmt.Sprintf(t.orderCreatedSubject, ref, c.displayName())
			body = fmt.Sprintf(t.orderCreatedBody, ref, c.displayName())
		case EventStatusChanged:
			status := statusLabel(c, o.Status)
			subject = fmt.Sprintf(t.orderStatusSubject, ref, status)
			body = fmt.Sprintf(t.orderStatusBody, ref, status)
		default:
			return
		}

		name := deref(o.CustomerName)
		greeting := t.hiCompany
		if name != "" {
			greeting = fmt.Sprintf(t.hi, name)
		}
		payment := ""
		if event == EventCreated {
			payment = paymentBlock(c)
		}
		withCancel := !OrderStatusIsFinal(o.Status)
		customerBody := greeting + "\n\n" + body + "\n" + payment +
			footer(c, link, t.viewOrder, fmt.Sprintf(t.waAboutOrder, ref), withCancel)
		deliver(c, "ORDER", orderInternalID, event, AudienceCustomer, orderEmail(companyID, &o), subject, customerBody)
	})
}

// OrderReference returns the order code, or the first 8 characters of the id.
func OrderReference(o *order.OrderS) string {
	if code := deref(o.OrderCode); code != "" {
		return code
	}
	id := strings.ToUpper(o.OrderInternalID)
	if len(id) > 8 {
		id = id[:8]
	}
	return "#" + id
}

// CurrentOrderStatus returns the saved status of an order ("" if not found).
// Used to send STATUS_CHANGED only when the status really changes.
func CurrentOrderStatus(companyID, orderInternalID string) string {
	o := order.OrderS{OrderInternalID: strings.TrimSpace(orderInternalID), CompanyID: strings.TrimSpace(companyID)}
	if err := o.Select(); err != nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(o.Status))
}

// ---------------------------------------------------------------------------
// Private tokens
// ---------------------------------------------------------------------------

func ensureAppointmentToken(companyID, id string) (string, error) {
	token, err := emailnotifier.NewPublicToken("a")
	if err != nil {
		return "", err
	}
	return appointment.EnsurePublicToken(companyID, id, token)
}

func ensureWebOrderToken(companyID, id string) (string, error) {
	token, err := emailnotifier.NewPublicToken("w")
	if err != nil {
		return "", err
	}
	return weborder.EnsurePublicToken(companyID, id, token)
}

func ensureOrderToken(companyID, id string) (string, error) {
	token, err := emailnotifier.NewPublicToken("o")
	if err != nil {
		return "", err
	}
	return order.EnsurePublicToken(companyID, id, token)
}
