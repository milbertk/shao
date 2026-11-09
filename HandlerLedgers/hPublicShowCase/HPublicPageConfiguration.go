package publicshowcaseinshao

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/blassistant"
	blcompanyverification "github.com/milbertk/shao/bussunessLedger/blcompanyverification"
	blpageconfiguration "github.com/milbertk/shao/bussunessLedger/blpageconfiguration"
	"github.com/milbertk/shao/bussunessLedger/blplan"
	"github.com/milbertk/shao/bussunessLedger/bltimezone"
)

// publicPageConfiguration holds only the public-safe fields of the page
// configuration. Storage paths and internal ids are never exposed.
type publicPageConfiguration struct {
	DisplayName string  `json:"displayName"`
	FontFamily  *string `json:"fontFamily,omitempty"`
	LogoURL     *string `json:"logoUrl,omitempty"`
	HeaderURL   *string `json:"headerUrl,omitempty"`
	// Verified is true only when tbl_company_verification says so; the
	// store shows the "Verified company" badge only then.
	Verified bool `json:"verified"`
	// Store money (2026-10-22): one currency for every price, and how the
	// customer pays (shown after an order).
	CurrencySymbol      string  `json:"currencySymbol"`
	PaymentInstructions *string `json:"paymentInstructions,omitempty"`
	// AI shopping assistant included in the company plan (store shows the button).
	AssistantEnabled bool `json:"assistantEnabled"`
	// Store bookings included in the company plan (tbl_plans.web_bookings);
	// when false the store hides "Book now".
	BookingsEnabled bool `json:"bookingsEnabled"`
	// AcceptingOrders is false when the company has no active plan
	// (read-only, 2026-10-23): the store stays visible but does not take
	// orders or booking requests.
	AcceptingOrders bool `json:"acceptingOrders"`
	// Legal texts written by the company (2026-10-24); omitted when empty,
	// so the store shows the "Terms" / "Privacy" links only when they exist.
	TermsText   *string `json:"termsText,omitempty"`
	PrivacyText *string `json:"privacyText,omitempty"`

	// Store "Contact us" (2026-10-31): company WhatsApp + one social link.
	// Each one only when set; neither = no "Contact us" button.
	ContactWhatsapp *string `json:"contactWhatsapp,omitempty"`
	SocialLink      *string `json:"socialLink,omitempty"`
	SocialLinkType  *string `json:"socialLinkType,omitempty"`
	// Customer service email (Company page, 2026-11-08): shown in the "Request
	// sent — pending approval" pop-up ("No email? Contact ..."). Only when valid.
	ContactEmail *string `json:"contactEmail,omitempty"`

	// Time zone of the company (2026-11-04): the store shows the slots in
	// this zone and computes "today" with it.
	TimeZone      string  `json:"timeZone"`
	TimeZoneLabel *string `json:"timeZoneLabel,omitempty"`
}

// HPublicPageConfiguration returns the customer-page appearance of a company
// (name, font, logo and header images) for the public showcase. It is
// UNAUTHENTICATED and READ-ONLY. A company without configuration gets 200 with
// no result, so the showcase keeps its default look.
//
//	GET /public/companies/{companyId}/page-configuration
func HPublicPageConfiguration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeShowcaseResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	companyID := strings.TrimSpace(mux.Vars(r)["companyId"])
	if companyID == "" {
		writeShowcaseResponse(w, http.StatusBadRequest, "companyId is required", nil)
		return
	}

	config, err := blpageconfiguration.BSelectPageConfiguration(companyID)
	if err != nil {
		if errors.Is(err, blpageconfiguration.ErrPageConfigurationNotFound) {
			writeShowcaseResponse(w, http.StatusOK, "Page configuration not created yet.", nil)
			return
		}

		log.Printf("[HPublicPageConfiguration] Business operation failed: companyId=%s error=%v", companyID, err)
		writeShowcaseResponse(w, http.StatusInternalServerError,
			"An internal error occurred while loading the page configuration", nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.RecordPublic(r, "PublicViewPageConfiguration", companyID, nil)
	writeShowcaseResponse(w, http.StatusOK, "Page configuration retrieved successfully.", publicPageConfiguration{
		DisplayName: config.DisplayName,
		FontFamily:  config.FontFamily,
		LogoURL:     config.LogoURL,
		HeaderURL:   config.HeaderURL,
		Verified:    blcompanyverification.BIsCompanyVerified(companyID),

		CurrencySymbol:      publicCurrency(config.CurrencySymbol),
		PaymentInstructions: config.PaymentInstructions,
		AssistantEnabled:    blassistant.BAssistantAvailable(companyID),
		BookingsEnabled:     blplan.BHasFeature(companyID, blplan.FeatureWebBookings),
		AcceptingOrders:     blplan.BIsPlanActive(companyID),
		TermsText:           config.TermsText,
		PrivacyText:         config.PrivacyText,
		ContactWhatsapp:     config.ContactWhatsapp,
		SocialLink:          config.SocialLink,
		SocialLinkType:      config.SocialLinkType,
		ContactEmail:        blpageconfiguration.PublicContactEmail(config.CustomerServiceEmail),
		TimeZone:            bltimezone.LocationFor(config.TimeZone).String(),
		TimeZoneLabel:       config.TimeZoneLabel,
	})
}

// publicCurrency returns the company currency, "$" when not set.
func publicCurrency(symbol string) string {
	if symbol = strings.TrimSpace(symbol); symbol == "" {
		return blpageconfiguration.DefaultCurrencySymbol
	}
	return symbol
}
