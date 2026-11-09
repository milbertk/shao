package blnotifications

import (
	"fmt"
	"math"
	"strings"

	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
)

// =====================================================================
//  More customer updates (2026-10-30): the customer is told about every
//  change the company makes, not only about status changes:
//    - an appointment's service / date / time changed   (MODIFIED)
//    - an order's details changed (dates, items, total)  (UPDATED)
//    - an order item moved to another tracking step      (ITEM_STEP)
//    - a new tracking stage was added to an order        (TRACKING_UPDATE)
//  Same rules as the other emails: company SMTP or the AlephK default,
//  daily limit per address, everything in tbl_notification_log.
// =====================================================================

const (
	EventModified       = "MODIFIED"        // appointment service / date / time changed
	EventUpdated        = "UPDATED"         // order details changed
	EventItemStep       = "ITEM_STEP"       // order item moved to another step
	EventTrackingUpdate = "TRACKING_UPDATE" // new tracking stage on an order
)

// AppointmentSnapshot is what the customer saw before a change.
type AppointmentSnapshot struct {
	ServiceID string
	Date      string
	StartHour string
	EndHour   string
	Status    string
	// Location (2026-11-07): id and name ("" = none).
	LocationID   string
	LocationName string
}

// SnapshotAppointment keeps the fields that matter to the customer.
func SnapshotAppointment(a appointment.AppointmentS) AppointmentSnapshot {
	return AppointmentSnapshot{
		ServiceID:    strings.TrimSpace(a.ServiceID),
		Date:         strings.TrimSpace(a.AppointmentDate),
		StartHour:    shortHour(a.StartHour),
		EndHour:      shortHour(a.EndHour),
		Status:       strings.ToUpper(strings.TrimSpace(a.Status)),
		LocationID:   strings.ToLower(deref(a.LocationID)),
		LocationName: deref(a.LocationName),
	}
}

// AppointmentWhenChanged tells if service, date, time or location changed
// (pure, tested).
func AppointmentWhenChanged(before, after AppointmentSnapshot) bool {
	return before.ServiceID != after.ServiceID || before.Date != after.Date ||
		before.StartHour != after.StartHour || before.EndHour != after.EndHour ||
		before.LocationID != after.LocationID
}

// NotifyAppointmentModified tells the customer the new service / date / time
// (and what it was before). Runs in the background.
func NotifyAppointmentModified(companyID, appointmentID string, before AppointmentSnapshot) {
	companyID, appointmentID = strings.TrimSpace(companyID), strings.TrimSpace(appointmentID)
	if companyID == "" || appointmentID == "" {
		return
	}
	runAsync("appointment "+EventModified, func() {
		c := loadCompany(companyID)
		t := c.texts()

		a := appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID}
		if err := a.Select(); err != nil {
			logNotSent(companyID, "APPOINTMENT", appointmentID, EventModified, "appointment not loaded: "+err.Error())
			return
		}
		hideLocationWhenOff(&a)
		if a.LocationName == nil {
			before.LocationID, before.LocationName = "", ""
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

		previousService := service
		if before.ServiceID != "" && before.ServiceID != strings.TrimSpace(a.ServiceID) {
			if name, _ := productInfo(companyID, before.ServiceID); name != "" {
				previousService = name
			}
		}
		previous := previousService + ", " + before.Date + " (" + timeRangeOn(c, before.Date, before.StartHour, before.EndHour) + ")"
		if before.LocationID != strings.ToLower(deref(a.LocationID)) && before.LocationName != "" {
			previous += ", " + before.LocationName
		}

		subject := fmt.Sprintf(t.apptModifiedSubject, c.displayName())
		body := fmt.Sprintf(t.apptModifiedBody, service, a.AppointmentDate, when) +
			appointmentLocationText(t, &a) + "\n" +
			line(t.labelBefore, previous)
		reference := service + ", " + a.AppointmentDate + " " + timeRange(c, a.StartHour, a.EndHour)
		customerBody := fmt.Sprintf(t.hi, a.CustomerName) + "\n\n" + body + "\n" +
			footer(c, link, t.manageLink, fmt.Sprintf(t.waAboutBooking, reference), true)
		deliver(c, "APPOINTMENT", appointmentID, EventModified, AudienceCustomer, appointmentEmail(companyID, &a), subject, customerBody)
	})
}

// OrderSnapshot is what the customer saw before an order change.
type OrderSnapshot struct {
	StartDate    string
	DeliveryDate string
	TotalCents   int64
	Items        string // product:quantity list, in order
}

// SnapshotOrder keeps the order fields that matter to the customer.
func SnapshotOrder(o order.OrderS, items []order.OrderItemS) OrderSnapshot {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, fmt.Sprintf("%s:%d", strings.TrimSpace(it.ProductID), it.Quantity))
	}
	return OrderSnapshot{
		StartDate:    deref(o.StartDate),
		DeliveryDate: deref(o.EstimatedDeliveryDate),
		TotalCents:   int64(math.Round(o.Total * 100)),
		Items:        strings.Join(parts, ","),
	}
}

// OrderDetailsChanged tells if dates, items or total changed (pure, tested).
func OrderDetailsChanged(before, after OrderSnapshot) bool {
	return before != after
}

// NotifyOrderUpdated tells the customer that the order details changed
// (dates, items, total) and shows the current ones. Runs in the background.
func NotifyOrderUpdated(companyID, orderInternalID string) {
	companyID, orderInternalID = strings.TrimSpace(companyID), strings.TrimSpace(orderInternalID)
	if companyID == "" || orderInternalID == "" {
		return
	}
	runAsync("order "+EventUpdated, func() {
		c := loadCompany(companyID)
		t := c.texts()

		o := order.OrderS{OrderInternalID: orderInternalID, CompanyID: companyID}
		if err := o.Select(); err != nil {
			logNotSent(companyID, "ORDER", orderInternalID, EventUpdated, "order not loaded: "+err.Error())
			return
		}
		ref := OrderReference(&o)
		link := ""
		if token, err := ensureOrderToken(companyID, orderInternalID); err == nil {
			link = manageLink(companyID, token)
		}

		var details strings.Builder
		if items, err := order.SelectItemsByOrder(companyID, orderInternalID); err == nil && len(items) > 0 {
			details.WriteString(t.labelItems + ":\n")
			for _, it := range items {
				name, _ := productInfo(companyID, it.ProductID)
				if name == "" {
					name = deref(it.InternalCode)
				}
				details.WriteString(fmt.Sprintf("  - %d x %s\n", it.Quantity, name))
			}
		}
		details.WriteString(line(t.labelTotal, fmt.Sprintf("%s%.2f", c.Currency, o.Total)))
		details.WriteString(line(t.labelDelivery, deref(o.EstimatedDeliveryDate)))
		details.WriteString(line(t.labelStatus, statusLabel(c, o.Status)))

		subject := fmt.Sprintf(t.orderUpdatedSubject, ref)
		body := fmt.Sprintf(t.orderUpdatedBody, ref) + "\n\n" + details.String()
		deliverOrderEmail(c, &o, EventUpdated, subject, body, link)
	})
}

// NotifyOrderItemStep tells the customer that one product of the order moved
// to another tracking step (step name + step status + order status).
func NotifyOrderItemStep(companyID, orderInternalID, productName, stepName, stepStatus string) {
	companyID, orderInternalID = strings.TrimSpace(companyID), strings.TrimSpace(orderInternalID)
	if companyID == "" || orderInternalID == "" {
		return
	}
	runAsync("order "+EventItemStep, func() {
		c := loadCompany(companyID)
		t := c.texts()

		o := order.OrderS{OrderInternalID: orderInternalID, CompanyID: companyID}
		if err := o.Select(); err != nil {
			logNotSent(companyID, "ORDER", orderInternalID, EventItemStep, "order not loaded: "+err.Error())
			return
		}
		ref := OrderReference(&o)
		link := ""
		if token, err := ensureOrderToken(companyID, orderInternalID); err == nil {
			link = manageLink(companyID, token)
		}
		product := strings.TrimSpace(productName)
		if product == "" {
			product = t.labelItems
		}
		step := strings.TrimSpace(stepName)
		if step == "" {
			step = "—"
		}

		subject := fmt.Sprintf(t.orderItemStepSubject, ref, step)
		body := fmt.Sprintf(t.orderItemStepBody, product, step, statusLabel(c, stepStatus)) + "\n\n" +
			line(t.labelReference, ref) +
			line(t.labelStatus, statusLabel(c, o.Status))
		deliverOrderEmail(c, &o, EventItemStep, subject, body, link)
	})
}

// NotifyOrderStage tells the customer about a new tracking stage of the order.
func NotifyOrderStage(companyID, orderInternalID, stageName, stageStatus string) {
	companyID, orderInternalID = strings.TrimSpace(companyID), strings.TrimSpace(orderInternalID)
	if companyID == "" || orderInternalID == "" {
		return
	}
	runAsync("order "+EventTrackingUpdate, func() {
		c := loadCompany(companyID)
		t := c.texts()

		o := order.OrderS{OrderInternalID: orderInternalID, CompanyID: companyID}
		if err := o.Select(); err != nil {
			logNotSent(companyID, "ORDER", orderInternalID, EventTrackingUpdate, "order not loaded: "+err.Error())
			return
		}
		ref := OrderReference(&o)
		link := ""
		if token, err := ensureOrderToken(companyID, orderInternalID); err == nil {
			link = manageLink(companyID, token)
		}
		stage := strings.TrimSpace(stageName)
		subject := fmt.Sprintf(t.orderStageSubject, ref, stage)
		body := fmt.Sprintf(t.orderStageBody, ref, stage, statusLabel(c, stageStatus))
		deliverOrderEmail(c, &o, EventTrackingUpdate, subject, body, link)
	})
}

// deliverOrderEmail adds the greeting + footer and sends an order email.
func deliverOrderEmail(c companyContext, o *order.OrderS, event, subject, body, link string) {
	t := c.texts()
	ref := OrderReference(o)
	greeting := t.hiCompany
	if name := deref(o.CustomerName); name != "" {
		greeting = fmt.Sprintf(t.hi, name)
	}
	customerBody := greeting + "\n\n" + body + "\n" +
		footer(c, link, t.viewOrder, fmt.Sprintf(t.waAboutOrder, ref), !OrderStatusIsFinal(o.Status))
	deliver(c, "ORDER", o.OrderInternalID, event, AudienceCustomer, orderEmail(c.CompanyID, o), subject, customerBody)
}
