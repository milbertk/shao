package blnotifications

import (
	"log"
	"strings"

	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
	"github.com/milbertk/shao/dataBasesLedger/notifications"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
	emailnotifier "github.com/milbertk/shao/shared"
)

// =====================================================================
//  Customer email lookup (2026-10-30). The first VALID email wins:
//    appointment: typed on it -> linked customer
//    order:       typed on it -> linked customer -> its store request ->
//                 registered user of that request -> customer id as a
//                 registered user (orders from store requests keep the uid)
//    web order:   typed on the request -> registered user
//  No email = the customer email is skipped (logged as SKIPPED, nothing is
//  sent and no mail server is used).
// =====================================================================

// firstEmail returns the first valid email of the candidates, asking the
// lookups only while nothing was found (pure order, tested).
func firstEmail(candidates ...func() string) string {
	for _, candidate := range candidates {
		if email := strings.TrimSpace(candidate()); email != "" && emailnotifier.ValidEmail(email) {
			return email
		}
	}
	return ""
}

func fixed(value string) func() string { return func() string { return value } }

func lookup(name string, fn func() (string, error)) func() string {
	return func() string {
		value, err := fn()
		if err != nil {
			log.Printf("[notifications] %s: %v", name, err)
			return ""
		}
		return value
	}
}

func appointmentEmail(companyID string, a *appointment.AppointmentS) string {
	return firstEmail(
		fixed(deref(a.Email)),
		lookup("appointment customer email", func() (string, error) {
			return notifications.CustomerEmail(companyID, deref(a.CustomerID))
		}),
	)
}

func orderEmail(companyID string, o *order.OrderS) string {
	var requestEmail, requestUser string
	requestLoaded := false
	loadRequest := func() {
		if requestLoaded {
			return
		}
		requestLoaded = true
		var err error
		requestEmail, requestUser, err = notifications.OrderWebRequestContact(companyID, o.OrderInternalID)
		if err != nil {
			log.Printf("[notifications] order store request: %v", err)
		}
	}
	return firstEmail(
		fixed(deref(o.CustomerEmail)),
		lookup("order customer email", func() (string, error) {
			return notifications.CustomerEmail(companyID, o.CustomerID)
		}),
		func() string { loadRequest(); return requestEmail },
		lookup("order registered user email", func() (string, error) {
			loadRequest()
			return notifications.RegisteredUserEmail(requestUser)
		}),
		lookup("order customer as registered user", func() (string, error) {
			return notifications.RegisteredUserEmail(o.CustomerID)
		}),
	)
}

func webOrderEmail(w *weborder.WebOrderRequestS) string {
	return firstEmail(
		fixed(deref(w.Email)),
		lookup("web order registered user email", func() (string, error) {
			return notifications.RegisteredUserEmail(deref(w.UserCode))
		}),
	)
}
