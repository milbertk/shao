package blpageconfiguration

import "testing"

func TestPublicContactEmail(t *testing.T) {
	text := func(s string) *string { return &s }
	cases := []struct {
		in   *string
		want string // "" = nil
	}{
		{nil, ""},
		{text(""), ""},
		{text("   "), ""},
		{text("not-an-email"), ""},
		{text("a@b.com\r\nBcc: x@y.com"), ""},
		{text("  service@kire.com "), "service@kire.com"},
	}
	for _, c := range cases {
		got := PublicContactEmail(c.in)
		if c.want == "" && got != nil {
			t.Errorf("PublicContactEmail(%v) = %q, want nil", c.in, *got)
		}
		if c.want != "" && (got == nil || *got != c.want) {
			t.Errorf("PublicContactEmail(%q) = %v, want %q", *c.in, got, c.want)
		}
	}
}
