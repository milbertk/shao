package blnotifications

import (
	"strings"
	"testing"

	emailnotifier "github.com/milbertk/shao/shared"
)

func TestCompanyAlreadyInformed(t *testing.T) {
	yes := [][2]string{
		{"APPOINTMENT", EventRequested}, {"APPOINTMENT", EventCustomerCancelled}, {"APPOINTMENT", EventRescheduleRequested},
		{"WEB_ORDER", EventCreated}, {"WEB_ORDER", EventCustomerCancelled}, {"ORDER", EventCancelRequested},
	}
	no := [][2]string{
		{"APPOINTMENT", EventApproved}, {"APPOINTMENT", EventCreated}, {"APPOINTMENT", EventModified},
		{"WEB_ORDER", EventApproved}, {"WEB_ORDER", EventRejected}, {"ORDER", EventCreated},
		{"ORDER", EventStatusChanged}, {"ORDER", EventUpdated}, {"ORDER", EventItemStep},
	}
	for _, c := range yes {
		if !companyAlreadyInformed(c[0], c[1]) {
			t.Errorf("%v should be already informed", c)
		}
	}
	for _, c := range no {
		if companyAlreadyInformed(c[0], c[1]) {
			t.Errorf("%v should get a copy", c)
		}
	}
}

func TestCompanyCopyRecipient(t *testing.T) {
	t.Setenv("SMTP_FROM", "notify@alephk.com")
	t.Setenv("SMTP_USER", "notify@alephk.com")
	own := &emailnotifier.SMTPSettings{FromEmail: "ventas@kire.com"}
	def := &emailnotifier.SMTPSettings{FromEmail: "notify@alephk.com"}
	cases := []struct {
		name string
		c    companyContext
		want string
	}{
		{"own smtp + notification", companyContext{SMTP: own, SMTPSource: "COMPANY", NotificationEmail: "owner@kire.com"}, "owner@kire.com"},
		{"own smtp, no notification", companyContext{SMTP: own, SMTPSource: "COMPANY"}, "ventas@kire.com"},
		{"default + owner email", companyContext{SMTP: def, SMTPSource: "DEFAULT", NotificationEmail: "owner@kire.com"}, "owner@kire.com"},
		{"default, no email", companyContext{SMTP: def, SMTPSource: "DEFAULT"}, ""},
		{"default, AlephK address", companyContext{SMTP: def, SMTPSource: "DEFAULT", NotificationEmail: "Notify@AlephK.com"}, ""},
		{"no smtp", companyContext{NotificationEmail: "owner@kire.com"}, ""},
	}
	for _, c := range cases {
		if got := companyCopyRecipient(c.c); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestCompanyCopyBody(t *testing.T) {
	en := companyContext{Language: "en"}
	es := companyContext{Language: "es"}
	if b := companyCopyBody(en, "ana@x.com", "SENT", "", "BODY"); !strings.Contains(b, "Sent to the customer: ana@x.com") || !strings.HasSuffix(b, "BODY") {
		t.Fatal(b)
	}
	if b := companyCopyBody(es, "", "SKIPPED", "customer has no email", "B"); !strings.Contains(b, "NO se envió al cliente: el cliente no tiene correo") {
		t.Fatal(b)
	}
	if b := companyCopyBody(en, "ana@x.com", "FAILED", "timeout", "B"); !strings.Contains(b, "NOT sent to the customer: ana@x.com — timeout") {
		t.Fatal(b)
	}
}

func TestCompanyCopyTargets(t *testing.T) {
	own := &emailnotifier.SMTPSettings{FromEmail: "ventas@kire.com"}
	base := companyContext{SMTP: own, SMTPSource: "COMPANY", NotificationEmail: "owner@kire.com"}

	cs := base
	cs.CustomerServiceEmail = "cs@kire.com"
	if to, bcc, skip := companyCopyTargets(cs, "SENT"); skip || to != "owner@kire.com" || len(bcc) != 0 {
		t.Fatalf("sent: %q %v %v", to, bcc, skip)
	}
	if to, bcc, skip := companyCopyTargets(cs, "SKIPPED"); skip || to != "owner@kire.com" || len(bcc) != 1 || bcc[0] != "cs@kire.com" {
		t.Fatalf("not sent: %q %v %v", to, bcc, skip)
	}
	same := base
	same.CustomerServiceEmail = "OWNER@kire.com"
	if _, _, skip := companyCopyTargets(same, "SENT"); !skip {
		t.Fatal("same address already got the blind copy")
	}
	if to, bcc, skip := companyCopyTargets(same, "SKIPPED"); skip || to != "owner@kire.com" || len(bcc) != 0 {
		t.Fatalf("same, not sent: %q %v", to, bcc)
	}
	def := companyContext{SMTP: &emailnotifier.SMTPSettings{FromEmail: "notify@alephk.com"}, SMTPSource: "DEFAULT", CustomerServiceEmail: "cs@kire.com"}
	if to, _, skip := companyCopyTargets(def, "SKIPPED"); skip || to != "cs@kire.com" {
		t.Fatalf("no copy address: %q", to)
	}
	if got := customerServiceBcc(cs, "CS@kire.com"); got != nil {
		t.Fatal("no blind copy to the same customer address")
	}
	if got := customerServiceBcc(cs, "ana@x.com"); len(got) != 1 {
		t.Fatal("blind copy expected")
	}
}
