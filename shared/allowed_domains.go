package emailnotifier

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// =====================================================================
//  Allowed domains (2026-10-31).
//
//  ONE setting in .env lists the public domains of AlephK, for example:
//
//	ALLOWED_DOMAINS=aleph-k.com,aleph-k.online
//
//  Each domain also covers ALL its subdomains (app.aleph-k.com,
//  telos.aleph-k.com, voting.aleph-k.online...), with http and https.
//  It is used by:
//    - CORS: browsers on those domains may call the API (router).
//    - CAPTCHA: with APP_ENV=production and no TURNSTILE_ALLOWED_HOSTNAMES,
//      tokens must be solved on one of those domains.
//  Adding or removing a domain = edit .env and restart. No code change.
// =====================================================================

// NormalizeDomain turns "https://*.Aleph-K.com:443/" into "aleph-k.com".
func NormalizeDomain(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if i := strings.Index(value, "://"); i >= 0 {
		value = value[i+3:]
	}
	if i := strings.IndexAny(value, "/?#"); i >= 0 {
		value = value[:i]
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.TrimPrefix(value, "*")
	return strings.Trim(value, ".")
}

// ParseDomainList splits a comma / space separated list of domains.
func ParseDomainList(raw string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, item := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == ';' || r == '\n' }) {
		if d := NormalizeDomain(item); d != "" && strings.Contains(d, ".") && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

// AllowedDomains returns the domains of ALLOWED_DOMAINS (normalized).
func AllowedDomains() []string {
	return ParseDomainList(os.Getenv("ALLOWED_DOMAINS"))
}

// HostMatchesDomain: host is the domain itself or one of its subdomains.
func HostMatchesDomain(host, domain string) bool {
	host = NormalizeDomain(host)
	domain = NormalizeDomain(domain)
	if host == "" || domain == "" {
		return false
	}
	return host == domain || strings.HasSuffix(host, "."+domain)
}

// MatchDomain returns the first domain that host belongs to ("" = none).
func MatchDomain(host string, domains []string) string {
	for _, d := range domains {
		if HostMatchesDomain(host, d) {
			return NormalizeDomain(d)
		}
	}
	return ""
}

// OriginAllowedByDomains: a browser Origin ("https://app.aleph-k.com") on
// one of the domains, with http or https and the standard port.
func OriginAllowedByDomains(origin string, domains []string) bool {
	if len(domains) == 0 {
		return false
	}
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if port := parsed.Port(); port != "" && port != "80" && port != "443" {
		return false
	}
	return MatchDomain(parsed.Hostname(), domains) != ""
}

// RequestIsHTTPS tells if the browser used https (directly or through a
// proxy that sets X-Forwarded-Proto).
func RequestIsHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]), "https")
}
