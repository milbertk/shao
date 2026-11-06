package blnotifications

import (
	"fmt"
	"os"
	"strings"
)

// =====================================================================
//  Company copy of every customer email (2026-10-30).
//
//  So the company can double check every event by email (bookings,
//  approvals, cancellations, reschedules, order changes, tracking steps...),
//  each customer email is also sent to the company as "[Copy] <subject>",
//  with the same text and a first line saying if the customer got it.
//
//  Where the copy goes:
//    - Company's own SMTP:        the notification email in Company page;
//                                 when empty, the company's own sender address.
//    - AlephK general email:      ONLY the notification email in Company page
//                                 (the owner's email). Never the AlephK account;
//                                 when empty, no copy (logged as SKIPPED).
//    - No usable SMTP:            no copy (the customer row already says why).
//
//  Events that already send their own company email (requests, customer
//  cancellations, reschedule and cancellation requests, new web orders) do
//  not get a second copy. Copies are logged with audience COMPANY_COPY and do
//  not count toward the customer's daily email limit.
//
//  Customer service email (2026-10-31): every customer email also goes to it
//  as a blind copy (bNotifyCommon.go); companyCopyTargets avoids duplicates.
// =====================================================================

// companyAlreadyInformed tells if this event already sends its own company
// email (with all the details), so no copy is added.
func companyAlreadyInformed(entityType, event string) bool {
	switch event {
	case EventCancelRequested:
		return true
	case EventCustomerCancelled:
		return entityType == "APPOINTMENT" || entityType == "WEB_ORDER"
	case EventRequested, EventRescheduleRequested:
		return entityType == "APPOINTMENT"
	case EventCreated:
		return entityType == "WEB_ORDER"
	}
	return false
}

// isAlephKAddress tells if the address is the AlephK general account (.env).
func isAlephKAddress(c companyContext, address string) bool {
	address = strings.TrimSpace(address)
	if address == "" {
		return false
	}
	for _, own := range []string{os.Getenv("SMTP_FROM"), os.Getenv("SMTP_USER")} {
		if strings.EqualFold(strings.TrimSpace(own), address) {
			return true
		}
	}
	return c.SMTPSource == "DEFAULT" && c.SMTP != nil && strings.EqualFold(strings.TrimSpace(c.SMTP.FromEmail), address)
}

// companyCopyRecipient returns where the company copy goes ("" = nowhere).
func companyCopyRecipient(c companyContext) string {
	if c.SMTP == nil {
		return ""
	}
	notification := strings.TrimSpace(c.NotificationEmail)
	switch c.SMTPSource {
	case "COMPANY":
		if notification != "" {
			return notification
		}
		return strings.TrimSpace(c.SMTP.FromEmail)
	case "DEFAULT":
		if isAlephKAddress(c, notification) {
			return ""
		}
		return notification
	}
	return ""
}

// companyCopyBody builds the copy text (pure, tested).
func companyCopyBody(c companyContext, customerTo, customerStatus, customerReason, body string) string {
	t := c.texts()
	var sent string
	switch {
	case customerStatus == "SENT":
		sent = fmt.Sprintf(t.copySentTo, strings.TrimSpace(customerTo))
	case strings.TrimSpace(customerTo) == "":
		sent = fmt.Sprintf(t.copyNotSent, t.copyNoEmail)
	default:
		sent = fmt.Sprintf(t.copyNotSent, strings.TrimSpace(customerTo)+" — "+customerReason)
	}
	return t.copyIntro + "\n" + sent + "\n\n----------------------------------------\n\n" + body
}

// deliverCompanyCopy sends the company copy of one customer email.
func deliverCompanyCopy(c companyContext, entityType, entityID, event, customerTo, customerStatus, customerReason, subject, body string) {
	if c.SMTP == nil || companyAlreadyInformed(entityType, event) {
		return
	}
	to, bcc, skip := companyCopyTargets(c, customerStatus)
	if skip {
		return
	}
	copyBody := companyCopyBody(c, customerTo, customerStatus, customerReason, body)
	// Answering the copy writes to the customer (when it has an email).
	replyTo := strings.TrimSpace(customerTo)
	sendAndLog(c, entityType, entityID, event, AudienceCompanyCopy, to, c.texts().copySubjectPrefix+subject, copyBody, replyTo, bcc)
}

// companyCopyTargets (2026-10-31, customer service email) decides who gets
// the "[Copy]" so nobody receives the same event twice:
//   - customer email SENT: customer service already got it as a blind copy,
//     so the copy goes only to the company copy address, and is skipped when
//     that address IS the customer service email.
//   - customer email NOT sent: customer service got nothing, so it is added
//     to the copy (as recipient when there is no company copy address).
func companyCopyTargets(c companyContext, customerStatus string) (to string, bcc []string, skip bool) {
	to = companyCopyRecipient(c)
	cs := strings.TrimSpace(c.CustomerServiceEmail)
	if customerStatus == "SENT" {
		if cs != "" && strings.EqualFold(cs, to) {
			return "", nil, true
		}
		return to, nil, false
	}
	switch {
	case cs == "":
	case to == "":
		to = cs
	case !strings.EqualFold(cs, to):
		bcc = []string{cs}
	}
	return to, bcc, false
}
