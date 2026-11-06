package blpageconfiguration

import (
	"strings"
	"testing"
)

func TestNormalizeCurrencySymbol(t *testing.T) {
	cases := map[string]string{"": "$", "  ₡ ": "₡", "R$": "R$", "COL$": "COL$"}
	for in, want := range cases {
		got, err := normalizeCurrencySymbol(in)
		if err != nil || got != want {
			t.Errorf("normalizeCurrencySymbol(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := normalizeCurrencySymbol("TOOLONG"); err == nil {
		t.Error("expected an error for 7 characters")
	}
	if _, err := normalizeCurrencySymbol("$\n"); err != nil {
		t.Error("trailing new line should be trimmed")
	}
}

func TestNormalizePaymentInstructions(t *testing.T) {
	if v, err := normalizePaymentInstructions(s("  \r\n ")); err != nil || v != nil {
		t.Fatal("blank should be nil")
	}
	v, err := normalizePaymentInstructions(s(" SINPE Móvil 8888-7777\r\nBN 123 "))
	if err != nil || *v != "SINPE Móvil 8888-7777\nBN 123" {
		t.Fatalf("got %q %v", *v, err)
	}
	if _, err := normalizePaymentInstructions(s(strings.Repeat("a", 1001))); err == nil {
		t.Fatal("expected max length error")
	}
}

func TestNormalizeLegalText(t *testing.T) {
	if v, err := normalizeLegalText("termsText", s("  \r\n ")); err != nil || v != nil {
		t.Fatalf("empty text must be nil, got %v %v", v, err)
	}
	v, err := normalizeLegalText("termsText", s(" Line 1\r\nLine 2 "))
	if err != nil || v == nil || *v != "Line 1\nLine 2" {
		t.Fatalf("got %v %v", v, err)
	}
	if _, err := normalizeLegalText("privacyText", s(strings.Repeat("a", 20001))); err == nil {
		t.Fatal("too long must fail")
	}
}
