// Package blcaptcha checks the CAPTCHA token sent by the browser
// (2026-10-25, security S4 — Cloudflare Turnstile).
//
// It is the ONE replaceable piece: the forms send the token in the
// X-Captcha-Token header and the handlers call BVerifyCaptcha. To change the
// provider later (hCaptcha, Google reCAPTCHA) only this file changes.
//
// Environment:
//
//	TURNSTILE_SECRET_KEY  secret key of the widget (Cloudflare dashboard).
//	                      EMPTY = CAPTCHA OFF (every request passes): use it
//	                      on your computer, or use the test secret
//	                      1x0000000000000000000000000000000AA (always passes).
//	CAPTCHA_PROVIDER      "turnstile" (default) or "none" (off).
//	CAPTCHA_FAIL_OPEN     "1" = when Cloudflare cannot be reached, let the
//	                      request pass. Default (empty) = refuse it (safer).
//	TURNSTILE_ALLOWED_HOSTNAMES  optional, comma separated: only tokens
//	                      solved on these hostnames are accepted.
package blcaptcha

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	emailnotifier "github.com/milbertk/shao/shared"
)

// HeaderName is the request header that carries the token.
const HeaderName = "X-Captcha-Token"

var (
	// ErrCaptchaRequired: the CAPTCHA is on and no token was sent.
	ErrCaptchaRequired = errors.New("CAPTCHA_REQUIRED: please complete the security check")
	// ErrCaptchaFailed: the token is invalid, expired or already used.
	ErrCaptchaFailed = errors.New("CAPTCHA_FAILED: the security check failed, please try again")
	// ErrCaptchaUnavailable: the CAPTCHA service could not be reached.
	ErrCaptchaUnavailable = errors.New("CAPTCHA_UNAVAILABLE: the security check is not available right now, please try again in a moment")
)

const maxTokenLength = 2048

// turnstileVerifyURL is a variable so tests can point it to a fake server.
var turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

var httpClient = &http.Client{Timeout: 10 * time.Second}

func provider() string {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("CAPTCHA_PROVIDER")))
	if value == "" {
		return "turnstile"
	}
	return value
}

func secretKey() string {
	return strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY"))
}

// Enabled reports whether requests must carry a valid CAPTCHA token.
func Enabled() bool {
	return provider() == "turnstile" && secretKey() != ""
}

// Code returns the short code of a CAPTCHA error for the frontend
// (CAPTCHA_REQUIRED, CAPTCHA_FAILED, CAPTCHA_UNAVAILABLE) or "".
func Code(err error) string {
	switch {
	case errors.Is(err, ErrCaptchaRequired):
		return "CAPTCHA_REQUIRED"
	case errors.Is(err, ErrCaptchaFailed):
		return "CAPTCHA_FAILED"
	case errors.Is(err, ErrCaptchaUnavailable):
		return "CAPTCHA_UNAVAILABLE"
	}
	return ""
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
	Hostname   string   `json:"hostname"`
}

// BVerifyCaptcha returns nil when the token is valid (or the CAPTCHA is off).
// Each token can be verified only once (Cloudflare rule), so the browser must
// get a new one for every attempt.
func BVerifyCaptcha(token string) error {
	if !Enabled() {
		return nil
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrCaptchaRequired
	}
	if len(token) > maxTokenLength {
		return ErrCaptchaFailed
	}

	result, err := callSiteverify(token)
	if err != nil {
		if os.Getenv("CAPTCHA_FAIL_OPEN") == "1" {
			log.Printf("[captcha] service not reachable, request allowed (CAPTCHA_FAIL_OPEN=1): %v", err)
			return nil
		}
		log.Printf("[captcha] service not reachable: %v", err)
		return ErrCaptchaUnavailable
	}
	if !result.Success {
		log.Printf("[captcha] token rejected: %v", result.ErrorCodes)
		return ErrCaptchaFailed
	}
	if !hostnameAllowed(result.Hostname) {
		log.Printf("[captcha] token solved on a hostname that is not allowed: %s", result.Hostname)
		return ErrCaptchaFailed
	}
	return nil
}

// hostnameAllowed checks where the CAPTCHA was solved (2026-10-31: domains).
//
//	TURNSTILE_ALLOWED_HOSTNAMES=app.aleph-k.com,*.aleph-k.online
//	  exact hostnames, or "*.domain" / ".domain" = the domain and its subdomains.
//	Empty: with APP_ENV=production and ALLOWED_DOMAINS set, only those domains
//	  (and subdomains); otherwise any hostname (as before).
func hostnameAllowed(hostname string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	allowed := strings.TrimSpace(os.Getenv("TURNSTILE_ALLOWED_HOSTNAMES"))
	if allowed == "" {
		env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
		domains := emailnotifier.AllowedDomains()
		if (env == "production" || env == "prod") && len(domains) > 0 {
			return emailnotifier.MatchDomain(hostname, domains) != ""
		}
		return true
	}
	for _, item := range strings.Split(allowed, ",") {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "*.") || strings.HasPrefix(item, ".") {
			if emailnotifier.HostMatchesDomain(hostname, item) {
				return true
			}
			continue
		}
		if item == hostname {
			return true
		}
	}
	return false
}

// callSiteverify asks Cloudflare if the token is valid. The visitor IP is
// not sent: behind proxies it can be wrong and make good tokens fail.
func callSiteverify(token string) (*siteverifyResponse, error) {
	form := url.Values{}
	form.Set("secret", secretKey())
	form.Set("response", token)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("siteverify answered HTTP %d", response.StatusCode)
	}

	var result siteverifyResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&result); err != nil {
		return nil, fmt.Errorf("siteverify answer not readable: %w", err)
	}
	return &result, nil
}
