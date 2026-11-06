package blpageconfiguration

import "testing"

func s(v string) *string { return &v }

func TestWhatsapp(t *testing.T) {
	got, err := normalizeWhatsapp(s(" +506 8888-7777 "))
	if err != nil || *got != "+50688887777" {
		t.Fatal(err, got)
	}
	if _, err := normalizeWhatsapp(s("123")); err == nil {
		t.Fatal("short accepted")
	}
	if _, err := normalizeWhatsapp(s("abc123456789")); err == nil {
		t.Fatal("letters accepted")
	}
	if v, err := normalizeWhatsapp(s("  ")); err != nil || v != nil {
		t.Fatal("blank")
	}
}
