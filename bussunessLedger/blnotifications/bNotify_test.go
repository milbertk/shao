package blnotifications

import "testing"

func TestMoney(t *testing.T) {
	cases := map[string]string{
		formatMoney(45000, "₡"): "₡45,000", formatMoney(12.5, "$"): "$12.50",
		formatMoney(19.5, "R$"): "R$ 19.50", formatMoney(1234567, ""): "$1,234,567", formatMoney(0, "€"): "€0",
	}
	for got, want := range cases {
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	}
}

func TestTimesAndStatus(t *testing.T) {
	en := companyContext{Language: "en"}
	es := companyContext{Language: "es"}
	if timeRange(en, "09:00:00", "10:00:00") != "09:00–10:00" {
		t.Fatal(timeRange(en, "09:00:00", "10:00:00"))
	}
	if timeRange(es, "00:00", "23:59") != "todo el día" {
		t.Fatal("allday")
	}
	if statusLabel(es, "in_progress") != "En proceso" || statusLabel(en, "WAITING_PARTS") != "Waiting parts" {
		t.Fatal(statusLabel(en, "WAITING_PARTS"))
	}
	if whatsappLink("+506 8888-7777", "Hola WEB-1") != "https://wa.me/50688887777?text=Hola+WEB-1" {
		t.Fatal(whatsappLink("+506 8888-7777", "Hola WEB-1"))
	}
	if whatsappLink("12", "x") != "" {
		t.Fatal("short")
	}
}
