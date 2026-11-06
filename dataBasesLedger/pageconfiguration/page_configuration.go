package pageconfiguration

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// PageConfigurationS represents one row from public.tbl_page_configuration.
// There is at most one row per company (company_id is UNIQUE).
type PageConfigurationS struct {
	PageConfigurationID string  `json:"pageConfigurationId"`
	CompanyID           string  `json:"companyId"`
	DisplayName         string  `json:"displayName"`
	FontFamily          *string `json:"fontFamily,omitempty"`
	LogoURL             *string `json:"logoUrl,omitempty"`
	LogoPath            *string `json:"logoPath,omitempty"`
	HeaderURL           *string `json:"headerUrl,omitempty"`
	HeaderPath          *string `json:"headerPath,omitempty"`
	// Communication with customers (2026-10-22).
	NotificationEmail    *string `json:"notificationEmail,omitempty"` // company inbox for new orders/bookings
	ContactWhatsapp      *string `json:"contactWhatsapp,omitempty"`   // used in "Talk to us on WhatsApp" links
	NotificationLanguage string  `json:"notificationLanguage"`        // en | es (emails to customers)
	// Readable store address, e.g. "maderas-finas" (2026-10-22). Unique.
	Slug *string `json:"slug,omitempty"`
	// Store money and payment (2026-10-22): one currency for every price of
	// the company; how customers pay (shown after orders and in emails).
	CurrencySymbol      string  `json:"currencySymbol"`
	PaymentInstructions *string `json:"paymentInstructions,omitempty"`
	// Legal texts of the company (2026-10-24), written by the owner in the
	// Company page. NULL = not shown in the store.
	TermsText   *string `json:"termsText,omitempty"`
	PrivacyText *string `json:"privacyText,omitempty"`
	// AlephK general email (2026-10-30): when the company has no SMTP of its
	// own, its customer emails go out with the AlephK default SMTP only if
	// this is TRUE. Column use_default_smtp (sql/migration_default_smtp_option.sql).
	UseDefaultSMTP bool `json:"useDefaultSmtp"`
	// Company contact for the store and customer service (2026-10-31,
	// sql/migration_company_contact_customer_service.sql): one social network
	// link + its type (FACEBOOK | INSTAGRAM | TIKTOK | OTHER, only for the icon)
	// and the customer service email (blind copy of every customer email).
	SocialLink           *string `json:"socialLink,omitempty"`
	SocialLinkType       *string `json:"socialLinkType,omitempty"`
	CustomerServiceEmail *string `json:"customerServiceEmail,omitempty"`
	// Time zone of the company (2026-11-04, sql/migration_company_time_zone.sql):
	// IANA name (e.g. America/Costa_Rica) used for every "now" of the company,
	// plus an optional text shown to customers (e.g. "Hora de Costa Rica").
	TimeZone      string    `json:"timeZone"`
	TimeZoneLabel *string   `json:"timeZoneLabel,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

const pageConfigurationSelectColumns = `
	page_configuration_id,
	company_id,
	display_name,
	font_family,
	logo_url,
	logo_path,
	header_url,
	header_path,
	notification_email,
	contact_whatsapp,
	notification_language,
	slug,
	currency_symbol,
	payment_instructions,
	terms_text,
	privacy_text,
	use_default_smtp,
	social_link,
	social_link_type,
	customer_service_email,
	time_zone,
	time_zone_label,
	created_at,
	updated_at
`

// scanPageConfiguration scans one row (in the select columns order) into p.
func scanPageConfiguration(row interface {
	Scan(dest ...any) error
}, p *PageConfigurationS) error {
	return row.Scan(
		&p.PageConfigurationID,
		&p.CompanyID,
		&p.DisplayName,
		&p.FontFamily,
		&p.LogoURL,
		&p.LogoPath,
		&p.HeaderURL,
		&p.HeaderPath,
		&p.NotificationEmail,
		&p.ContactWhatsapp,
		&p.NotificationLanguage,
		&p.Slug,
		&p.CurrencySymbol,
		&p.PaymentInstructions,
		&p.TermsText,
		&p.PrivacyText,
		&p.UseDefaultSMTP,
		&p.SocialLink,
		&p.SocialLinkType,
		&p.CustomerServiceEmail,
		&p.TimeZone,
		&p.TimeZoneLabel,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
}

// SelectByCompany loads the configuration of the company in CompanyID.
// Returns a wrapped sql.ErrNoRows when the company has none yet.
func (a *PageConfigurationS) SelectByCompany() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(a.CompanyID)
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		SELECT ` + pageConfigurationSelectColumns + `
		FROM public.tbl_page_configuration
		WHERE company_id = $1
	`

	if err := scanPageConfiguration(db.QueryRow(query, companyID), a); err != nil {
		return fmt.Errorf("failed to select page configuration: %w", err)
	}

	return nil
}

// Upsert creates the configuration of the company or replaces the existing
// one. CompanyID and DisplayName are mandatory.
func (a *PageConfigurationS) Upsert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(a.CompanyID)
	displayName := strings.TrimSpace(a.DisplayName)

	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}
	if displayName == "" {
		return fmt.Errorf("displayName is required")
	}

	query := `
		INSERT INTO public.tbl_page_configuration (
			company_id,
			display_name,
			font_family,
			logo_url,
			logo_path,
			header_url,
			header_path,
			notification_email,
			contact_whatsapp,
			notification_language,
			slug,
			currency_symbol,
			payment_instructions,
			terms_text,
			privacy_text,
			use_default_smtp,
			social_link,
			social_link_type,
			customer_service_email,
			time_zone,
			time_zone_label
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, COALESCE(NULLIF($10, ''), 'en'), $11,
			COALESCE(NULLIF($12, ''), '$'), $13, $14, $15, $16, $17, $18, $19,
			COALESCE(NULLIF($20, ''), '` + DefaultTimeZone + `'), $21)
		ON CONFLICT (company_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			font_family  = EXCLUDED.font_family,
			logo_url     = EXCLUDED.logo_url,
			logo_path    = EXCLUDED.logo_path,
			header_url   = EXCLUDED.header_url,
			header_path  = EXCLUDED.header_path,
			notification_email    = EXCLUDED.notification_email,
			contact_whatsapp      = EXCLUDED.contact_whatsapp,
			notification_language = EXCLUDED.notification_language,
			slug                  = EXCLUDED.slug,
			currency_symbol       = EXCLUDED.currency_symbol,
			payment_instructions  = EXCLUDED.payment_instructions,
			terms_text            = EXCLUDED.terms_text,
			privacy_text          = EXCLUDED.privacy_text,
			use_default_smtp      = EXCLUDED.use_default_smtp,
			social_link           = EXCLUDED.social_link,
			social_link_type      = EXCLUDED.social_link_type,
			customer_service_email = EXCLUDED.customer_service_email,
			time_zone             = EXCLUDED.time_zone,
			time_zone_label       = EXCLUDED.time_zone_label,
			updated_at   = now()
		RETURNING ` + pageConfigurationSelectColumns

	err = scanPageConfiguration(db.QueryRow(
		query,
		companyID,
		displayName,
		a.FontFamily,
		a.LogoURL,
		a.LogoPath,
		a.HeaderURL,
		a.HeaderPath,
		a.NotificationEmail,
		a.ContactWhatsapp,
		a.NotificationLanguage,
		a.Slug,
		a.CurrencySymbol,
		a.PaymentInstructions,
		a.TermsText,
		a.PrivacyText,
		a.UseDefaultSMTP,
		a.SocialLink,
		a.SocialLinkType,
		a.CustomerServiceEmail,
		a.TimeZone,
		a.TimeZoneLabel,
	), a)
	if err != nil {
		return fmt.Errorf("failed to save page configuration: %w", err)
	}

	return nil
}

// SelectBySlug loads the configuration whose slug matches (case-insensitive).
// Returns a wrapped sql.ErrNoRows when no company uses that slug.
func (a *PageConfigurationS) SelectBySlug(slug string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	slug = strings.TrimSpace(slug)
	if slug == "" {
		return fmt.Errorf("slug is required")
	}

	query := `
		SELECT ` + pageConfigurationSelectColumns + `
		FROM public.tbl_page_configuration
		WHERE lower(slug) = lower($1)
	`
	if err := scanPageConfiguration(db.QueryRow(query, slug), a); err != nil {
		return fmt.Errorf("failed to select page configuration by slug: %w", err)
	}
	return nil
}

// SlugInUse reports whether another company already uses the slug.
func SlugInUse(slug, exceptCompanyID string) (bool, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return false, fmt.Errorf("database connection error: %w", err)
	}

	var count int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM public.tbl_page_configuration
		WHERE lower(slug) = lower($1)
		  AND company_id <> $2
	`, strings.TrimSpace(slug), strings.TrimSpace(exceptCompanyID)).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check slug: %w", err)
	}
	return count > 0, nil
}

// DefaultTimeZone is the zone of a company that has not chosen one
// (2026-11-04): Costa Rica, GMT-6 all year.
const DefaultTimeZone = "America/Costa_Rica"

// SelectTimeZone returns the time zone of a company and its optional label.
// A company without page configuration gets DefaultTimeZone (no error).
func SelectTimeZone(companyID string) (string, *string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return DefaultTimeZone, nil, fmt.Errorf("database connection error: %w", err)
	}

	var zone string
	var label *string
	err = db.QueryRow(`
		SELECT time_zone, time_zone_label
		FROM public.tbl_page_configuration
		WHERE company_id = $1
	`, strings.TrimSpace(companyID)).Scan(&zone, &label)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultTimeZone, nil, nil
	}
	if err != nil {
		return DefaultTimeZone, nil, fmt.Errorf("failed to select time zone: %w", err)
	}
	return zone, label, nil
}
