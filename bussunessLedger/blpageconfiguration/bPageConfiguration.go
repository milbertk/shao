package blpageconfiguration

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/milbertk/shao/bussunessLedger/blstorage"
	"github.com/milbertk/shao/bussunessLedger/bltimezone"
	"github.com/milbertk/shao/dataBasesLedger/pageconfiguration"
	emailnotifier "github.com/milbertk/shao/shared"
)

// Languages for the emails sent to customers. Default: English.
var allowedNotificationLanguages = map[string]bool{"en": true, "es": true}

const defaultNotificationLanguage = "en"

// normalizeWhatsapp keeps "+" and digits only and requires 8–15 digits.
func normalizeWhatsapp(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	var b strings.Builder
	digits := 0
	for _, r := range strings.TrimSpace(*value) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			digits++
		case r == '+' && b.Len() == 0:
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '(' || r == ')':
			// ignore separators
		default:
			return nil, errors.New("invalid contactWhatsapp: use only digits, spaces and an optional +")
		}
	}
	if digits == 0 {
		return nil, nil
	}
	if digits < 8 || digits > 15 {
		return nil, errors.New("invalid contactWhatsapp: it must have between 8 and 15 digits (include the country code)")
	}
	result := b.String()
	return &result, nil
}

// ErrPageConfigurationNotFound allows handlers to distinguish a company that
// has not configured its page yet from an internal database error.
var ErrPageConfigurationNotFound = errors.New("page configuration not found")

const maxDisplayNameLength = 120

// allowedFonts must stay in sync with FONT_OPTIONS in the frontend
// PageConfiguration component.
var allowedFonts = map[string]bool{
	"Inter":        true,
	"Arial":        true,
	"Georgia":      true,
	"Trebuchet MS": true,
}

// emptyStringToNil returns nil when a *string contains only blanks, so
// optional columns receive NULL instead of an empty string.
func emptyStringToNil(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

// BSelectPageConfiguration loads the page configuration of one company.
func BSelectPageConfiguration(companyID string) (*pageconfiguration.PageConfigurationS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	config := pageconfiguration.PageConfigurationS{CompanyID: companyID}

	if err := config.SelectByCompany(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrPageConfigurationNotFound, companyID)
		}
		return nil, fmt.Errorf("failed to select page configuration: %w", err)
	}

	return &config, nil
}

// BSavePageConfiguration creates or replaces the page configuration of one
// company. companyID is authoritative; any id in the body is ignored.
func BSavePageConfiguration(
	companyID string,
	input pageconfiguration.PageConfigurationS,
) (*pageconfiguration.PageConfigurationS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	config := pageconfiguration.PageConfigurationS{
		CompanyID:   companyID,
		DisplayName: strings.TrimSpace(input.DisplayName),
		FontFamily:  emptyStringToNil(input.FontFamily),
		LogoURL:     emptyStringToNil(input.LogoURL),
		LogoPath:    emptyStringToNil(input.LogoPath),
		HeaderURL:   emptyStringToNil(input.HeaderURL),
		HeaderPath:  emptyStringToNil(input.HeaderPath),

		NotificationEmail:    emptyStringToNil(input.NotificationEmail),
		NotificationLanguage: strings.ToLower(strings.TrimSpace(input.NotificationLanguage)),
		// AlephK general email while the company has no SMTP (2026-10-30).
		UseDefaultSMTP: input.UseDefaultSMTP,
	}

	if config.NotificationLanguage == "" {
		config.NotificationLanguage = defaultNotificationLanguage
	}
	if !allowedNotificationLanguages[config.NotificationLanguage] {
		return nil, fmt.Errorf("invalid notificationLanguage: %q (use en or es)", config.NotificationLanguage)
	}
	if config.NotificationEmail != nil && !emailnotifier.ValidEmail(*config.NotificationEmail) {
		return nil, errors.New("invalid notificationEmail: enter one valid email address")
	}
	whatsapp, err := normalizeWhatsapp(input.ContactWhatsapp)
	if err != nil {
		return nil, err
	}
	config.ContactWhatsapp = whatsapp

	// Company social link + customer service email (2026-10-31).
	socialLink, socialType, err := normalizeSocialLink(input.SocialLink, input.SocialLinkType)
	if err != nil {
		return nil, err
	}
	config.SocialLink = socialLink
	config.SocialLinkType = socialType
	config.CustomerServiceEmail = emptyStringToNil(input.CustomerServiceEmail)
	if config.CustomerServiceEmail != nil && !emailnotifier.ValidEmail(*config.CustomerServiceEmail) {
		return nil, errors.New("invalid customerServiceEmail: enter one valid email address")
	}

	// Time zone of the company (2026-11-04): IANA name, Costa Rica when empty.
	zone, err := bltimezone.NormalizeTimeZone(input.TimeZone)
	if err != nil {
		return nil, err
	}
	config.TimeZone = zone
	zoneLabel, err := bltimezone.NormalizeLabel(input.TimeZoneLabel)
	if err != nil {
		return nil, err
	}
	config.TimeZoneLabel = zoneLabel

	// Readable store address (optional, unique). Empty -> uses the internal id.
	slug, err := prepareSlug(companyID, input.Slug)
	if err != nil {
		return nil, err
	}
	config.Slug = slug

	// One currency for the whole store, and how customers pay.
	currency, err := normalizeCurrencySymbol(input.CurrencySymbol)
	if err != nil {
		return nil, err
	}
	config.CurrencySymbol = currency
	payment, err := normalizePaymentInstructions(input.PaymentInstructions)
	if err != nil {
		return nil, err
	}
	config.PaymentInstructions = payment

	// Legal texts of the company (optional; empty = not shown in the store).
	terms, err := normalizeLegalText("termsText", input.TermsText)
	if err != nil {
		return nil, err
	}
	config.TermsText = terms
	privacy, err := normalizeLegalText("privacyText", input.PrivacyText)
	if err != nil {
		return nil, err
	}
	config.PrivacyText = privacy

	if config.DisplayName == "" {
		return nil, errors.New("displayName is required")
	}
	if utf8.RuneCountInString(config.DisplayName) > maxDisplayNameLength {
		return nil, fmt.Errorf("invalid displayName: maximum length is %d characters", maxDisplayNameLength)
	}

	if config.FontFamily != nil && !allowedFonts[*config.FontFamily] {
		return nil, fmt.Errorf("invalid fontFamily: %q is not supported", *config.FontFamily)
	}

	if err := validateImage(companyID, "logo", config.LogoURL, config.LogoPath); err != nil {
		return nil, err
	}
	if err := validateImage(companyID, "header", config.HeaderURL, config.HeaderPath); err != nil {
		return nil, err
	}

	if err := config.Upsert(); err != nil {
		if isSlugUniqueViolation(err) {
			return nil, ErrSlugTaken
		}
		return nil, fmt.Errorf("failed to save page configuration: %w", err)
	}
	bltimezone.Forget(companyID) // the new zone is used at once

	return &config, nil
}

// validateImage requires the url and path of one image (logo or header) to be
// sent together, the URL to be https, and the storage path to live inside the
// company's folder for that kind of image.
func validateImage(companyID, kind string, imageURL, imagePath *string) error {
	if imageURL == nil && imagePath == nil {
		return nil
	}
	if imageURL == nil || imagePath == nil {
		return fmt.Errorf("invalid %s: %sUrl and %sPath must be sent together", kind, kind, kind)
	}

	parsed, err := url.Parse(*imageURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("invalid %sUrl: an https URL is required", kind)
	}

	expectedPrefix := "companies/" + companyID + "/page-configuration/" + kind + "/"
	if !strings.HasPrefix(*imagePath, expectedPrefix) || strings.Contains(*imagePath, "..") {
		return fmt.Errorf("invalid %sPath: it must belong to the company page configuration %s folder", kind, kind)
	}

	// The URL must be a Firebase file of our bucket and the same file as the
	// path (2026-10-25, security S13).
	if err := blstorage.ValidateCompanyFile(companyID, *imagePath, *imageURL); err != nil {
		return fmt.Errorf("invalid %sUrl: %v", kind, err)
	}

	return nil
}

// Social link types (2026-10-31): only choose the icon shown in the store.
var allowedSocialLinkTypes = map[string]bool{"FACEBOOK": true, "INSTAGRAM": true, "TIKTOK": true, "OTHER": true}

const maxSocialLinkLength = 500

// normalizeSocialLink validates the company social network link (http/https,
// max 500 characters) and its type. No link = no type.
func normalizeSocialLink(link, linkType *string) (*string, *string, error) {
	value := emptyStringToNil(link)
	if value == nil {
		return nil, nil, nil
	}
	lower := strings.ToLower(*value)
	if !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "http://") {
		return nil, nil, errors.New("invalid socialLink: it must start with https://")
	}
	if len(*value) > maxSocialLinkLength || strings.ContainsAny(*value, " \"<>\r\n") {
		return nil, nil, fmt.Errorf("invalid socialLink: use one link of up to %d characters", maxSocialLinkLength)
	}
	kind := "OTHER"
	if linkType != nil {
		if t := strings.ToUpper(strings.TrimSpace(*linkType)); t != "" {
			kind = t
		}
	}
	if !allowedSocialLinkTypes[kind] {
		return nil, nil, fmt.Errorf("invalid socialLinkType: %q (use FACEBOOK, INSTAGRAM, TIKTOK or OTHER)", kind)
	}
	return value, &kind, nil
}
