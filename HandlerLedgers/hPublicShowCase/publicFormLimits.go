package publicshowcaseinshao

import (
	"errors"
	"net/http"
	"strings"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blpublicshowcase "github.com/milbertk/shao/bussunessLedger/bPublicShowcase"
	blassistant "github.com/milbertk/shao/bussunessLedger/blassistant"
	"github.com/milbertk/shao/bussunessLedger/blcaptcha"
)

// publicFormIPLimiter is the store assistant's in-memory daily limiter,
// reused for the order and booking forms (2026-10-25, security S4).
var publicFormIPLimiter = &blassistant.PublicDailyLimiter{}

// maxPublicFormBodyBytes limits the size of a store form request.
const maxPublicFormBodyBytes = 64 << 10 // 64 KB

// checkPublicFormLimits answers 429 (and tracks it) when the visitor IP or the
// company reached today's limit for this form. Returns false when blocked.
func checkPublicFormLimits(w http.ResponseWriter, r *http.Request, form, companyID string) bool {
	clientIP := handlertracking.ClientIP(r)
	key := form + "|" + strings.TrimSpace(companyID) + "|" + clientIP

	blocked := ""
	if !publicFormIPLimiter.Allow(key, blpublicshowcase.PublicFormIPDailyLimit()) {
		blocked = "IP"
	} else if err := blpublicshowcase.BCheckPublicFormCompanyLimit(form, companyID); errors.Is(err, blpublicshowcase.ErrPublicFormTooMany) {
		blocked = "COMPANY"
	}
	if blocked == "" {
		return true
	}

	handlertracking.RecordPublic(r, "PublicFormLimitReached", companyID, handlertracking.Details{
		"form":  form,
		"limit": blocked,
	})
	writeShowcaseResponse(w, http.StatusTooManyRequests,
		"Too many requests were sent today. Please try again tomorrow or contact the business directly.",
		map[string]string{"code": "PUBLIC_FORM_LIMIT"})
	return false
}

// checkPublicCaptcha verifies the CAPTCHA token of a store form (header
// X-Captcha-Token, Cloudflare Turnstile, 2026-10-25). It answers 400
// CAPTCHA_REQUIRED / CAPTCHA_FAILED or 503 CAPTCHA_UNAVAILABLE (and tracks
// it) and returns false when the request must stop. Off when
// TURNSTILE_SECRET_KEY is empty.
func checkPublicCaptcha(w http.ResponseWriter, r *http.Request, form, companyID string) bool {
	err := blcaptcha.BVerifyCaptcha(r.Header.Get(blcaptcha.HeaderName))
	if err == nil {
		return true
	}
	code := blcaptcha.Code(err)
	handlertracking.RecordPublic(r, "PublicCaptchaFailed", companyID, handlertracking.Details{
		"form": form,
		"code": code,
	})
	status := http.StatusBadRequest
	message := "Please complete the security check and try again."
	if code == "CAPTCHA_UNAVAILABLE" {
		status = http.StatusServiceUnavailable
		message = "The security check is not available right now. Please try again in a moment."
	}
	writeShowcaseResponse(w, status, message, map[string]string{"code": code})
	return false
}

// isPublicFormInvalid tells if the error is a format/size validation error.
func isPublicFormInvalid(err error) bool {
	return errors.Is(err, blpublicshowcase.ErrPublicFormInvalid)
}
