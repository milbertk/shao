package blpublicshowcase

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/milbertk/shao/bussunessLedger/bllocations"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
	"github.com/milbertk/shao/dataBasesLedger/configuration"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
	emailnotifier "github.com/milbertk/shao/shared"
)

// Protection of the public store forms (2026-10-25, security S4):
//   - per IP: in-memory daily counter (same limiter as the store assistant),
//     checked in the handler (HandlerLedgers/hPublicShowCase/publicFormLimits.go);
//   - per company: daily cap counted in the database (survives restarts);
//   - formats: email, phone and text lengths are checked on the server;
//   - customer emails: daily cap per address (see blnotifications.deliver).
// A CAPTCHA (Cloudflare Turnstile or Google reCAPTCHA) is NOT included yet:
// it needs an external account and will be added as a separate item.
//
// Limits can be changed in tbl_configuration without a code change:
//   PUBLIC_FORM_IP_DAILY_LIMIT       (default 20 per IP, per company, per form, per day)
//   PUBLIC_FORM_COMPANY_DAILY_LIMIT  (default 300 per company, per form, per day)

// Form kinds, used as part of the limiter keys.
const (
	PublicFormWebOrder    = "WEB_ORDER"
	PublicFormAppointment = "APPOINTMENT"
)

const (
	defaultPublicFormIPDailyLimit      = 20
	defaultPublicFormCompanyDailyLimit = 300

	maxPublicNameLength    = 120
	maxPublicEmailLength   = 254
	maxPublicShortText     = 300 // social link, contact value, region
	maxPublicAddressLength = 500
	maxPublicNotesLength   = 2000
	maxPublicItems         = 50
	maxPublicItemQuantity  = 999
)

var (
	// ErrPublicFormTooMany: the visitor (IP) or the company reached today's limit.
	ErrPublicFormTooMany = errors.New("PUBLIC_FORM_LIMIT: too many requests today")
	// ErrPublicFormInvalid wraps the validation messages (400 for the visitor).
	ErrPublicFormInvalid = errors.New("invalid")
)

// PublicFormLimit reads a limit from tbl_configuration (fallback when unset).
func PublicFormLimit(parameter string, fallback int) int {
	value, err := configuration.GetParameter(parameter)
	if err != nil || strings.TrimSpace(value) == "" {
		return fallback
	}
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number <= 0 {
		return fallback
	}
	return number
}

func startOfToday(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// PublicFormIPDailyLimit is the per-IP daily limit of one store form.
func PublicFormIPDailyLimit() int {
	return PublicFormLimit("PUBLIC_FORM_IP_DAILY_LIMIT", defaultPublicFormIPDailyLimit)
}

// BCheckPublicFormCompanyLimit returns ErrPublicFormTooMany when the company
// already received too many requests of this form today (counted in the
// database). Checked BEFORE saving, so a blocked request costs nothing.
func BCheckPublicFormCompanyLimit(form, companyID string) error {
	companyID = strings.TrimSpace(companyID)

	companyLimit := PublicFormLimit("PUBLIC_FORM_COMPANY_DAILY_LIMIT", defaultPublicFormCompanyDailyLimit)
	since := startOfToday(time.Now())
	var (
		count int
		err   error
	)
	switch form {
	case PublicFormWebOrder:
		count, err = weborder.CountRequestsSince(companyID, since)
	case PublicFormAppointment:
		count, err = appointment.CountPublicRequestsSince(companyID, since)
	}
	if err != nil {
		// Do not block real customers because of a counting error.
		log.Printf("[publicFormGuard] company count failed: form=%s companyId=%s error=%v", form, companyID, err)
		return nil
	}
	if count >= companyLimit {
		return ErrPublicFormTooMany
	}
	return nil
}

func invalidField(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrPublicFormInvalid, fmt.Sprintf(format, args...))
}

func tooLong(value string, max int) bool {
	return utf8.RuneCountInString(strings.TrimSpace(value)) > max
}

// ValidPublicPhone accepts 7 to 15 digits with optional +, spaces, dots,
// hyphens and parentheses (e.g. "+506 8888-8888"). Empty is allowed.
func ValidPublicPhone(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	if len(value) > 25 {
		return false
	}
	digits := 0
	for i, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '+' && i == 0:
		case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
		default:
			return false
		}
	}
	return digits >= 7 && digits <= 15
}

// validatePublicContact checks name, email and phone of a store form.
func validatePublicContact(name, email, phone string) error {
	if tooLong(name, maxPublicNameLength) {
		return invalidField("name is too long (maximum %d characters)", maxPublicNameLength)
	}
	email = strings.TrimSpace(email)
	if email != "" && (len(email) > maxPublicEmailLength || !emailnotifier.ValidEmail(email)) {
		return invalidField("email address is not valid")
	}
	if !ValidPublicPhone(phone) {
		return invalidField("phone number is not valid")
	}
	return nil
}

// ValidatePublicWebOrderRequest checks formats and sizes of a store order.
func ValidatePublicWebOrderRequest(req PublicWebOrderRequest) error {
	if err := validatePublicContact(req.CustomerName, req.Email, req.Phone); err != nil {
		return err
	}
	if tooLong(req.SocialLink, maxPublicShortText) || tooLong(req.ContactValue, maxPublicShortText) ||
		tooLong(req.Region, maxPublicShortText) || tooLong(req.PreferredContact, 40) || tooLong(req.OrderType, 40) {
		return invalidField("one of the contact fields is too long")
	}
	if tooLong(req.Address, maxPublicAddressLength) {
		return invalidField("address is too long (maximum %d characters)", maxPublicAddressLength)
	}
	if tooLong(req.Notes, maxPublicNotesLength) {
		return invalidField("notes are too long (maximum %d characters)", maxPublicNotesLength)
	}
	if len(req.Items) > maxPublicItems {
		return invalidField("too many items (maximum %d)", maxPublicItems)
	}
	for _, it := range req.Items {
		if it.Quantity > maxPublicItemQuantity {
			return invalidField("quantity is too large (maximum %d)", maxPublicItemQuantity)
		}
		if tooLong(it.ProductName, 200) || tooLong(it.ProductCode, 100) || tooLong(it.ProductID, 100) {
			return invalidField("item data is too long")
		}
	}
	if req.Latitude != nil && (*req.Latitude < -90 || *req.Latitude > 90) {
		return invalidField("map location is not valid")
	}
	if req.Longitude != nil && (*req.Longitude < -180 || *req.Longitude > 180) {
		return invalidField("map location is not valid")
	}
	return nil
}

// ValidatePublicAppointmentRequest checks formats and sizes of a store booking.
func ValidatePublicAppointmentRequest(req PublicAppointmentRequest) error {
	if err := validatePublicContact(req.CustomerName, req.Email, req.Phone); err != nil {
		return err
	}
	if tooLong(req.SocialLink, maxPublicShortText) {
		return invalidField("social link is too long")
	}
	if tooLong(req.Comments, maxPublicNotesLength) {
		return invalidField("comments are too long (maximum %d characters)", maxPublicNotesLength)
	}
	if id := strings.TrimSpace(req.LocationID); id != "" && !bllocations.ValidID(id) {
		return invalidField("location is not valid")
	}
	return nil
}
