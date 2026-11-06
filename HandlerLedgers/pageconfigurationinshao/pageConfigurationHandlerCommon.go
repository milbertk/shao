package pageconfigurationinshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	blpageconfiguration "github.com/milbertk/shao/bussunessLedger/blpageconfiguration"
	"github.com/milbertk/shao/dataBasesLedger/pageconfiguration"
)

// PageConfigurationHandlerResponse is the standard JSON envelope for page
// configuration handlers. Result holds a *pageconfiguration.PageConfigurationS.
type PageConfigurationHandlerResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

// writePageConfigurationResponse writes a JSON response with the given status.
func writePageConfigurationResponse(
	w http.ResponseWriter,
	status int,
	message string,
	result interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(PageConfigurationHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// pageConfigurationErrorStatus maps a business error to an HTTP status code.
func pageConfigurationErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, blpageconfiguration.ErrPageConfigurationNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, blpageconfiguration.ErrSlugTaken) {
		return http.StatusConflict
	}

	message := strings.ToLower(err.Error())

	switch {
	case strings.Contains(message, "is required"),
		strings.Contains(message, "must contain"),
		strings.Contains(message, "invalid"):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

// pageConfigurationInput is the accepted request body. Only the editable
// fields are allowed; ids and timestamps are owned by the server.
type pageConfigurationInput struct {
	DisplayName string  `json:"displayName"`
	FontFamily  *string `json:"fontFamily"`
	LogoURL     *string `json:"logoUrl"`
	LogoPath    *string `json:"logoPath"`
	HeaderURL   *string `json:"headerUrl"`
	HeaderPath  *string `json:"headerPath"`

	NotificationEmail    *string `json:"notificationEmail"`
	ContactWhatsapp      *string `json:"contactWhatsapp"`
	NotificationLanguage string  `json:"notificationLanguage"`

	// Readable store address; empty = use the internal id.
	Slug *string `json:"slug"`

	// One currency for the whole store ("$" when empty) and how customers pay.
	CurrencySymbol      string  `json:"currencySymbol"`
	PaymentInstructions *string `json:"paymentInstructions"`

	// Legal texts of the company (optional, shown in the store when set).
	TermsText   *string `json:"termsText"`
	PrivacyText *string `json:"privacyText"`

	// Use the AlephK general email while the company has no SMTP (2026-10-30).
	UseDefaultSMTP bool `json:"useDefaultSmtp"`

	// Company contact + customer service (2026-10-31).
	SocialLink           *string `json:"socialLink"`
	SocialLinkType       *string `json:"socialLinkType"`
	CustomerServiceEmail *string `json:"customerServiceEmail"`

	// Time zone of the company (2026-11-04): IANA name + optional text.
	TimeZone      string  `json:"timeZone"`
	TimeZoneLabel *string `json:"timeZoneLabel"`
}

// decodePageConfiguration decodes exactly one JSON object from the request
// body, rejecting unknown fields and trailing content.
func decodePageConfiguration(r *http.Request) (pageconfiguration.PageConfigurationS, error) {
	var config pageconfiguration.PageConfigurationS

	if r.Body == nil {
		return config, errors.New("request body is required")
	}

	decoder := json.NewDecoder(io.LimitReader(r.Body, 256<<10)) // legal texts can be long
	decoder.DisallowUnknownFields()

	var input pageConfigurationInput
	if err := decoder.Decode(&input); err != nil {
		return config, fmt.Errorf("invalid JSON body: %w", err)
	}

	// Reject a second JSON object in the same body.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return config, errors.New("body must contain a single JSON object")
	}

	config.DisplayName = input.DisplayName
	config.FontFamily = input.FontFamily
	config.LogoURL = input.LogoURL
	config.LogoPath = input.LogoPath
	config.HeaderURL = input.HeaderURL
	config.HeaderPath = input.HeaderPath
	config.NotificationEmail = input.NotificationEmail
	config.ContactWhatsapp = input.ContactWhatsapp
	config.NotificationLanguage = input.NotificationLanguage
	config.Slug = input.Slug
	config.CurrencySymbol = input.CurrencySymbol
	config.PaymentInstructions = input.PaymentInstructions
	config.TermsText = input.TermsText
	config.PrivacyText = input.PrivacyText
	config.UseDefaultSMTP = input.UseDefaultSMTP
	config.SocialLink = input.SocialLink
	config.SocialLinkType = input.SocialLinkType
	config.CustomerServiceEmail = input.CustomerServiceEmail
	config.TimeZone = input.TimeZone
	config.TimeZoneLabel = input.TimeZoneLabel

	return config, nil
}
