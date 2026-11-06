package emailnotifier

import (
	"os"
	"strings"
	"testing"
)

func TestSecretRoundTrip(t *testing.T) {
	os.Setenv("JWT_SECRET", "abc")
	enc, err := EncryptSecret("p@ss wörd")
	if err != nil || !strings.HasPrefix(enc, "v1:") {
		t.Fatal(err, enc)
	}
	dec, err := DecryptSecret(enc)
	if err != nil || dec != "p@ss wörd" {
		t.Fatal(err, dec)
	}
	os.Setenv("SMTP_ENCRYPTION_KEY", "other")
	if _, err := DecryptSecret(enc); err == nil {
		t.Fatal("expected failure with another key")
	}
}

func TestAddresses(t *testing.T) {
	for _, ok := range []string{"a@b.com", "ana.perez@mail.co.cr"} {
		if !ValidEmail(ok) {
			t.Fatal(ok)
		}
	}
	for _, bad := range []string{"", "a", "a@b.com\r\nBcc: x@y.com", "Ana <a@b.com>", "a@b.com,c@d.com"} {
		if ValidEmail(bad) {
			t.Fatal("accepted", bad)
		}
	}
}

func TestMIME(t *testing.T) {
	m := buildMIME("no@shop.cr", "Maderas Ñ", "c@x.com", "r@shop.cr", "Su cita — confirmada", "Hola\nlínea 2")
	if !strings.Contains(m, "Subject: =?utf-8?q?") || !strings.Contains(m, "Reply-To: r@shop.cr") || !strings.Contains(m, "Hola\r\nlínea 2") {
		t.Fatal(m)
	}
}

func TestToken(t *testing.T) {
	a, _ := NewPublicToken("a")
	b, _ := NewPublicToken("a")
	if a == b || len(a) != 50 || !strings.HasPrefix(a, "a_") {
		t.Fatal(a, b)
	}
}

func TestMaskEmail(t *testing.T) {
	cases := map[string]string{
		"maria.perez@gmail.com": "m***@gmail.com",
		"a@b.co":                "a***@b.co",
		"":                      "",
		"no-at-sign":            "***",
	}
	for in, want := range cases {
		if got := MaskEmail(in); got != want {
			t.Errorf("MaskEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMIMEAlternative(t *testing.T) {
	m := buildMIMEAlternative("no@shop.cr", "Kire via AlephK", "c@x.com", "r@shop.cr", "Su cita", "Hola\nlínea", "<p>Hola <b>línea</b></p>")
	for _, want := range []string{"multipart/alternative; boundary=", "Content-Type: text/plain; charset=\"utf-8\"", "Content-Type: text/html; charset=\"utf-8\"", "quoted-printable", "Reply-To: r@shop.cr"} {
		if !strings.Contains(m, want) {
			t.Fatalf("missing %q in\n%s", want, m)
		}
	}
	if strings.Count(m, "Content-Type: text/plain") != 1 {
		t.Fatal("the single-part header must be replaced")
	}
}

func TestCleanBcc(t *testing.T) {
	got := cleanBcc("ana@x.com", []string{"CS@kire.com", "ana@x.com", "bad", "cs@kire.com", " ", "owner@kire.com"})
	if strings.Join(got, ",") != "CS@kire.com,owner@kire.com" {
		t.Fatalf("got %v", got)
	}
	m := buildMIME("no@shop.cr", "Kire", "ana@x.com", "", "S", "B")
	if strings.Contains(strings.ToLower(m), "bcc") {
		t.Fatal("blind copies must never appear in the headers")
	}
}
