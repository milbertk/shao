package emailnotifier

import (
	"net/http"
	"reflect"
	"testing"
)

func TestParseDomainList(t *testing.T) {
	got := ParseDomainList(" Aleph-K.com, https://*.aleph-k.online/ ; .aleph-k.com localhost")
	want := []string{"aleph-k.com", "aleph-k.online"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestOriginAllowedByDomains(t *testing.T) {
	domains := []string{"aleph-k.com", "aleph-k.online"}
	ok := []string{
		"https://aleph-k.com", "http://aleph-k.com", "https://app.aleph-k.com",
		"https://telos.aleph-k.online", "http://voting.aleph-k.online", "https://a.b.aleph-k.com:443",
	}
	bad := []string{
		"https://evil-aleph-k.com", "https://aleph-k.com.evil.com", "ftp://aleph-k.com",
		"https://app.aleph-k.com:8443", "", "null", "https://example.com",
	}
	for _, o := range ok {
		if !OriginAllowedByDomains(o, domains) {
			t.Errorf("%s should be allowed", o)
		}
	}
	for _, o := range bad {
		if OriginAllowedByDomains(o, domains) {
			t.Errorf("%s should be rejected", o)
		}
	}
	if OriginAllowedByDomains("https://aleph-k.com", nil) {
		t.Error("no domains = nothing allowed")
	}
}

func TestMatchDomainAndHTTPS(t *testing.T) {
	if MatchDomain("api.aleph-k.online:8081", []string{"aleph-k.com", "aleph-k.online"}) != "aleph-k.online" {
		t.Fatal("match")
	}
	r, _ := http.NewRequest("GET", "http://api.aleph-k.com/x", nil)
	if RequestIsHTTPS(r) {
		t.Fatal("http")
	}
	r.Header.Set("X-Forwarded-Proto", "https")
	if !RequestIsHTTPS(r) {
		t.Fatal("proxy https")
	}
}
