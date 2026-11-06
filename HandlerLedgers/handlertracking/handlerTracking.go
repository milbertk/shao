// Package handlertracking records handler actions into the login tracking
// table (github.com/milbertk/tracking). It centralizes the LoginTracking
// insert that each handler used to build by hand.
//
// Every function is best-effort: a failure is logged and never blocks or
// changes the handler response.
package handlertracking

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/milbertk/security"
	"github.com/milbertk/tracking"
)

const (
	platformShao     = "Shao"
	platformShowcase = "ShaoShowcase"
	publicGuestID    = "public-guest"
)

// Details holds the action-specific fields stored in Jsonstring, for example
// the id of the record that was read or saved, or the filters of a search.
type Details map[string]any

// Record tracks an action of an authenticated company member. The user, email,
// active company and role are taken from the validated request context (set
// by RequireActiveCompanyMember), never from the request body.
func Record(r *http.Request, action string, details Details) {
	userID := ""
	if claims, ok := security.GetClaims(r); ok && claims != nil {
		userID = strings.TrimSpace(claims.UID)
	}
	companyID, _ := security.GetActiveCompanyID(r)
	email, _ := security.GetActiveCompanyEmail(r)
	role, _ := security.GetActiveCompanyRole(r)

	payload := Details{
		"companyId":        strings.TrimSpace(companyID),
		"actorCompanyRole": role,
	}
	for key, value := range details {
		payload[key] = value
	}

	insert(r, action, userID, email, platformShao, payload)
}

// RecordPublic tracks an action of the public showcase (no company session).
// The company comes from the URL. A logged-in visitor is recorded by their
// user id; anonymous visitors are recorded as "public-guest".
func RecordPublic(r *http.Request, action, companyID string, details Details) {
	userID := publicGuestID
	if claims, ok := security.GetClaims(r); ok && claims != nil && strings.TrimSpace(claims.UID) != "" {
		userID = strings.TrimSpace(claims.UID)
	}

	payload := Details{"companyId": strings.TrimSpace(companyID)}
	for key, value := range details {
		payload[key] = value
	}

	insert(r, action, userID, "", platformShowcase, payload)
}

// SearchFilters returns the non-empty query-string values of a search request,
// so the tracking stores what the user searched for.
func SearchFilters(r *http.Request) map[string]string {
	filters := map[string]string{}
	for key, values := range r.URL.Query() {
		if len(values) == 0 {
			continue
		}
		if value := strings.TrimSpace(values[0]); value != "" {
			filters[key] = value
		}
	}
	return filters
}

func insert(r *http.Request, action, userID, email, platform string, payload Details) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[handlertracking] Error creating tracking payload: action=%s error=%v", action, err)
		return
	}

	lt := tracking.LoginTracking{
		UserID:         userID,
		Email:          strings.TrimSpace(email),
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             ClientIP(r),
		Platform:       platform,
		MacAddress:     "0",
		Browser:        r.UserAgent(),
		CountryCode:    "",
		GMTTime:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		Lang:           r.Header.Get("Accept-Language"),
		Action:         action,
		Jsonstring:     string(jsonPayload),
	}
	if err := lt.Insert(); err != nil {
		log.Printf("[handlertracking] Error inserting tracking: action=%s error=%v", action, err)
	}
}

// ClientIP returns the visitor IP (2026-10-25, security S4).
//
// X-Forwarded-For can be written by anyone, so it is trusted ONLY when the
// server runs behind your own proxy / load balancer and TRUSTED_PROXY_HOPS
// says how many of them add an entry (1 = Nginx or Caddy in front, 2 = e.g.
// Google load balancer + Cloud Run). Then the entry added by your outermost
// proxy is used (counted from the right), which a visitor cannot fake.
// Without TRUSTED_PROXY_HOPS the TCP address is used.
func ClientIP(r *http.Request) string {
	hops, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("TRUSTED_PROXY_HOPS")))
	if hops > 0 {
		var entries []string
		for _, header := range r.Header.Values("X-Forwarded-For") {
			for _, part := range strings.Split(header, ",") {
				if part = strings.TrimSpace(part); part != "" {
					entries = append(entries, part)
				}
			}
		}
		if len(entries) >= hops {
			return entries[len(entries)-hops]
		}
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
			return realIP
		}
	}
	if host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
