package blnotifications

import (
	"strings"
	"testing"
)

func TestPaymentBlock(t *testing.T) {
	en := paymentBlock(companyContext{Language: "en", PaymentText: "SINPE Móvil 8888-7777"})
	if !strings.Contains(en, "How to pay:") || !strings.Contains(en, "SINPE Móvil 8888-7777") || !strings.Contains(en, "coming soon") {
		t.Fatalf("unexpected EN block: %q", en)
	}
	es := paymentBlock(companyContext{Language: "es"})
	if !strings.Contains(es, "Cómo pagar:") || !strings.Contains(es, "llegarán pronto") {
		t.Fatalf("unexpected ES block: %q", es)
	}
}
