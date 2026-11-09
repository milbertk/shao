package blnotifications

import (
	"strings"
	"testing"

	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
)

func TestAppointmentWhenChanged(t *testing.T) {
	a := AppointmentSnapshot{ServiceID: "s1", Date: "2026-11-02", StartHour: "10:00", EndHour: "11:00", Status: "SCHEDULED"}
	if AppointmentWhenChanged(a, a) {
		t.Fatal("same appointment must not count as changed")
	}
	b := a
	b.StartHour = "11:00"
	if !AppointmentWhenChanged(a, b) {
		t.Fatal("new hour must count as changed")
	}
	c := a
	c.Status = "DONE"
	if AppointmentWhenChanged(a, c) {
		t.Fatal("status alone is not a service/date/time change")
	}
}

func TestOrderDetailsChanged(t *testing.T) {
	date := "2026-11-05"
	o := order.OrderS{EstimatedDeliveryDate: &date, Total: 10.004}
	items := []order.OrderItemS{{ProductID: "p1", Quantity: 2}}
	before := SnapshotOrder(o, items)
	if OrderDetailsChanged(before, SnapshotOrder(o, items)) {
		t.Fatal("same order must not count as changed")
	}
	o2 := o
	o2.Total = 10.001 // same cents
	if OrderDetailsChanged(before, SnapshotOrder(o2, items)) {
		t.Fatal("a fraction of a cent is not a change")
	}
	more := []order.OrderItemS{{ProductID: "p1", Quantity: 3}}
	if !OrderDetailsChanged(before, SnapshotOrder(o, more)) {
		t.Fatal("new quantity must count as changed")
	}
	newDate := "2026-11-09"
	o3 := o
	o3.EstimatedDeliveryDate = &newDate
	if !OrderDetailsChanged(before, SnapshotOrder(o3, items)) {
		t.Fatal("new delivery date must count as changed")
	}
}

func TestDefaultSMTPFromEnv(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_FROM", "")
	if defaultSMTPFromEnv() != nil {
		t.Fatal("no host -> no default SMTP")
	}
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_USER", "notify@example.com")
	t.Setenv("SMTP_PORT", "465")
	t.Setenv("SMTP_SECURITY", "")
	t.Setenv("SMTP_FROM_NAME", "")
	s := defaultSMTPFromEnv()
	if s == nil || s.FromEmail != "notify@example.com" || s.Port != 465 || s.Security != "SSL" || s.FromName != "AlephK" {
		t.Fatalf("unexpected default SMTP %+v", s)
	}
}

func TestAppointmentLocationChange(t *testing.T) {
	a := AppointmentSnapshot{ServiceID: "s", Date: "2026-11-20", StartHour: "10:00", EndHour: "11:00", Status: "SCHEDULED", LocationID: "her"}
	b := a
	b.LocationID = "sp"
	if !AppointmentWhenChanged(a, b) {
		t.Fatal("a new location must send the 'changed' email")
	}
	c := a
	c.LocationName = "renamed"
	if AppointmentWhenChanged(a, c) {
		t.Fatal("only the name changed: no email")
	}
}

func TestAppointmentLocationText(t *testing.T) {
	name, address := "Heredia", "Av. 1"
	lat, lng := 9.998, -84.117
	a := appointment.AppointmentS{LocationName: &name, LocationAddress: &address, LocationLatitude: &lat, LocationLongitude: &lng}
	got := appointmentLocationText(textsEN, &a)
	want := "\n📍 Location: Heredia, Av. 1\nOpen in Maps: https://www.google.com/maps/search/?api=1&query=9.998000,-84.117000"
	if got != want {
		t.Fatalf("%q", got)
	}
	if es := appointmentLocationText(textsES, &a); !strings.Contains(es, "📍 Sede: Heredia, Av. 1") || !strings.Contains(es, "Abrir en Maps: ") {
		t.Fatal(es)
	}
	if appointmentLocationText(textsEN, &appointment.AppointmentS{}) != "" {
		t.Fatal("no location, no text")
	}
	html := bodyToHTML("x" + got)
	if !strings.Contains(html, `<a href="https://www.google.com/maps/search/?api=1&amp;query=9.998000,-84.117000"`) {
		t.Fatal(html)
	}
}
