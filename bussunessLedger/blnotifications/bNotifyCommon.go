// Package blnotifications sends the emails that keep customers and the
// company informed (new requests, approvals, status changes, cancellations).
//
// Rules:
//   - Emails are sent with the COMPANY's own SMTP (tbl_company_smtp). When the
//     company has none (or it is disabled / cannot be read) AND checked "use
//     the AlephK general email" in Company page, the AlephK default SMTP from
//     the environment (.env) is used (2026-10-30). Otherwise nothing is sent
//     and the attempt is logged as SKIPPED.
//   - The customer email is the one typed on the record, or else the linked
//     customer's, the store request's or the registered user's (bRecipients.go).
//     Without any, the customer email is skipped.
//   - Company emails go to the company notification email (page configuration).
//   - Company copy (2026-10-30): every customer email is also sent as a
//     "[Copy]" to the company (bCompanyCopy.go), so the company can double
//     check every event by email.
//   - Customer emails use the language chosen by the company (en default, es).
//   - Everything runs in the background: a slow mail server never blocks a
//     screen, and a failure never breaks the operation that triggered it.
//   - Every attempt is written to tbl_notification_log.
package blnotifications

import (
	"database/sql"
	"errors"
	"log"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	blcompanysmtp "github.com/milbertk/shao/bussunessLedger/blcompanysmtp"
	"github.com/milbertk/shao/bussunessLedger/bltimezone"
	"github.com/milbertk/shao/dataBasesLedger/configuration"
	"github.com/milbertk/shao/dataBasesLedger/notifications"
	"github.com/milbertk/shao/dataBasesLedger/pageconfiguration"
	"github.com/milbertk/shao/dataBasesLedger/products"
	emailnotifier "github.com/milbertk/shao/shared"
)

const (
	AudienceCustomer    = "CUSTOMER"
	AudienceCompany     = "COMPANY"
	AudienceCompanyCopy = "COMPANY_COPY" // copy of a customer email (2026-10-30)
	// blind copy of a customer email to the company customer service (2026-10-31)
	AudienceCustomerService = "CUSTOMER_SERVICE"

	defaultShowcaseBaseURL = "http://192.168.100.110:3001"
)

// companyContext is everything needed to write and send one company's emails.
type companyContext struct {
	CompanyID         string
	Name              string
	NotificationEmail string
	Whatsapp          string
	Language          string // en | es
	Currency          string // one currency for the whole store ("$" default)
	PaymentText       string // payment instructions ("" when not set)
	SMTP              *emailnotifier.SMTPSettings
	SMTPProblem       string // why SMTP is not usable ("" when it is)
	SMTPSource        string // COMPANY | DEFAULT ("" when none)
	UseDefaultSMTP    bool   // Company page: "use the AlephK general email" (2026-10-30)
	LogoURL           string // company logo (Company page), shown in HTML emails (2026-10-31)
	// CustomerServiceEmail (2026-10-31): blind copy of every customer email
	// (Company page). "" = none.
	CustomerServiceEmail string
	// Time zone of the company (2026-11-04) and its optional text, shown next
	// to appointment hours. nil = no zone text (tests).
	Location  *time.Location
	ZoneLabel string
}

// defaultSMTPFromEnv returns the AlephK default SMTP (2026-10-30), used when a
// company has no SMTP of its own. Same variable names as usersAppAK:
//
//	SMTP_HOST, SMTP_PORT (587), SMTP_USER, SMTP_PASSWORD,
//	SMTP_FROM (default SMTP_USER), SMTP_FROM_NAME (default "AlephK"),
//	SMTP_SECURITY (STARTTLS, or SSL; port 465 = SSL by default)
//
// Returns nil when SMTP_HOST or SMTP_FROM/SMTP_USER is missing.
func defaultSMTPFromEnv() *emailnotifier.SMTPSettings {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = user
	}
	if host == "" || from == "" {
		return nil
	}
	port := 587
	if value, err := strconv.Atoi(strings.TrimSpace(os.Getenv("SMTP_PORT"))); err == nil && value > 0 {
		port = value
	}
	security := strings.ToUpper(strings.TrimSpace(os.Getenv("SMTP_SECURITY")))
	if security != "SSL" && security != "STARTTLS" {
		security = "STARTTLS"
		if port == 465 {
			security = "SSL"
		}
	}
	name := strings.TrimSpace(os.Getenv("SMTP_FROM_NAME"))
	if name == "" {
		name = "AlephK"
	}
	return &emailnotifier.SMTPSettings{
		Host:      host,
		Port:      port,
		Security:  security,
		Username:  user,
		Password:  os.Getenv("SMTP_PASSWORD"),
		FromEmail: from,
		FromName:  name,
	}
}

func loadCompany(companyID string) companyContext {
	ctx := companyContext{CompanyID: companyID, Language: "en", Currency: "$", Location: bltimezone.DefaultLocation()}

	page := pageconfiguration.PageConfigurationS{CompanyID: companyID}
	if err := page.SelectByCompany(); err == nil {
		ctx.Location = bltimezone.LocationFor(page.TimeZone)
		if page.TimeZoneLabel != nil {
			ctx.ZoneLabel = strings.TrimSpace(*page.TimeZoneLabel)
		}
		ctx.Name = strings.TrimSpace(page.DisplayName)
		if page.NotificationEmail != nil {
			ctx.NotificationEmail = strings.TrimSpace(*page.NotificationEmail)
		}
		if page.ContactWhatsapp != nil {
			ctx.Whatsapp = strings.TrimSpace(*page.ContactWhatsapp)
		}
		if strings.EqualFold(page.NotificationLanguage, "es") {
			ctx.Language = "es"
		}
		if symbol := strings.TrimSpace(page.CurrencySymbol); symbol != "" {
			ctx.Currency = symbol
		}
		if page.PaymentInstructions != nil {
			ctx.PaymentText = strings.TrimSpace(*page.PaymentInstructions)
		}
		ctx.UseDefaultSMTP = page.UseDefaultSMTP
		if page.LogoURL != nil {
			ctx.LogoURL = strings.TrimSpace(*page.LogoURL)
		}
		if page.CustomerServiceEmail != nil && emailnotifier.ValidEmail(strings.TrimSpace(*page.CustomerServiceEmail)) {
			ctx.CustomerServiceEmail = strings.TrimSpace(*page.CustomerServiceEmail)
		}
	}

	settings, err := blcompanysmtp.LoadCompanySmtpSettings(companyID)
	switch {
	case err == nil:
		ctx.SMTP = settings
		ctx.SMTPSource = "COMPANY"
		if settings.FromName == "" {
			ctx.SMTP.FromName = ctx.Name
		}
		return ctx
	case errors.Is(err, blcompanysmtp.ErrCompanySmtpNotConfigured):
		ctx.SMTPProblem = "company SMTP not configured or disabled"
	default:
		ctx.SMTPProblem = "company SMTP could not be loaded: " + err.Error()
		log.Printf("[notifications] company %s: %s; trying the AlephK default SMTP", companyID, ctx.SMTPProblem)
	}

	// No usable company SMTP: the AlephK default SMTP from the environment,
	// ONLY when the company checked "use the AlephK general email" in Company
	// page. Otherwise nothing is sent.
	if !ctx.UseDefaultSMTP {
		ctx.SMTPProblem += "; the AlephK general email is not enabled in Company page"
		return ctx
	}
	if fallback := defaultSMTPFromEnv(); fallback != nil {
		if ctx.Name != "" {
			// "Kire via AlephK" / "Kire vía AlephK" (2026-10-31); customers
			// reply to the company email (Reply-To).
			fallback.FromName = ctx.Name + " " + ctx.texts().viaWord + " " + fallback.FromName
		}
		ctx.SMTP = fallback
		ctx.SMTPSource = "DEFAULT"
		ctx.SMTPProblem = ""
		return ctx
	}
	ctx.SMTPProblem += "; AlephK default SMTP not set (SMTP_HOST / SMTP_FROM in .env)"
	return ctx
}

// logNotSent records an email that could not even be prepared (for example
// the order was not found), so tbl_notification_log always explains why a
// customer got nothing (2026-10-30).
func logNotSent(companyID, entityType, entityID, event, reason string) {
	log.Printf("[notifications] %s %s %s not sent: %s", entityType, entityID, event, reason)
	entry := notifications.NotificationLogS{
		CompanyID:    companyID,
		EntityType:   entityType,
		EntityID:     entityID,
		Event:        event,
		Audience:     AudienceCustomer,
		Status:       "FAILED",
		ErrorMessage: reason,
	}
	if err := entry.Insert(); err != nil {
		log.Printf("[notifications] could not write log: %v", err)
	}
}

// texts returns the dictionary in the company language.
func (c companyContext) texts() textSet {
	if c.Language == "es" {
		return textsES
	}
	return textsEN
}

func (c companyContext) displayName() string {
	if c.Name != "" {
		return c.Name
	}
	return c.texts().ourTeam
}

// deliver sends one email (or logs why it was skipped) and records it.
// It returns the logged status: SENT, FAILED or SKIPPED.
// A customer email is also copied to the company (2026-10-30, bCompanyCopy.go),
// even when it could not be sent to the customer.
func deliver(c companyContext, entityType, entityID, event, audience, to, subject, body string) string {
	var bcc []string
	if audience == AudienceCustomer {
		bcc = customerServiceBcc(c, to)
	}
	status, reason := sendAndLog(c, entityType, entityID, event, audience, to, subject, body, "", bcc)
	if audience == AudienceCustomer {
		deliverCompanyCopy(c, entityType, entityID, event, to, status, reason, subject, body)
	}
	return status
}

// sendAndLog sends one email (or skips it), writes tbl_notification_log and
// returns the status and the skip / failure reason. replyTo "" = default.
// bcc (2026-10-31) = blind copies (customer service); each one also gets a
// log row (audience CUSTOMER_SERVICE) with the same status.
func sendAndLog(c companyContext, entityType, entityID, event, audience, to, subject, body, replyTo string, bcc []string) (string, string) {
	entry := notifications.NotificationLogS{
		CompanyID:  c.CompanyID,
		EntityType: entityType,
		EntityID:   entityID,
		Event:      event,
		Audience:   audience,
		Recipient:  strings.TrimSpace(to),
	}

	switch {
	case strings.TrimSpace(to) == "":
		entry.Status = "SKIPPED"
		if audience == AudienceCompany {
			entry.ErrorMessage = "company has no notification email"
		} else if audience == AudienceCompanyCopy {
			entry.ErrorMessage = "no email for copies (Company page notification email)"
		} else {
			entry.ErrorMessage = "customer has no email"
		}
	case !emailnotifier.ValidEmail(to):
		entry.Status = "SKIPPED"
		entry.ErrorMessage = "invalid email address"
	case c.SMTP == nil:
		entry.Status = "SKIPPED"
		entry.ErrorMessage = c.SMTPProblem
	case audience == AudienceCustomer && customerEmailLimitReached(to):
		// Anti-flood (2026-10-25, security S4): the store forms accept any
		// address, so one address receives at most N customer emails a day.
		entry.Status = "SKIPPED"
		entry.ErrorMessage = "daily email limit reached for this address"
	default:
		if replyTo == "" && audience == AudienceCustomer {
			// Customers answer to customer service (2026-10-31), else the company.
			replyTo = c.CustomerServiceEmail
			if replyTo == "" {
				replyTo = c.NotificationEmail
			}
		}
		// HTML version with the logos + "sent on behalf of" line (2026-10-31).
		textBody := body + onBehalfText(c)
		err := emailnotifier.SendWithSettings(*c.SMTP, emailnotifier.EmailMessage{
			To:       to,
			ReplyTo:  replyTo,
			Subject:  subject,
			Body:     textBody,
			HTMLBody: renderEmailHTML(c, body),
			Bcc:      bcc,
		})
		if err != nil {
			entry.Status = "FAILED"
			entry.ErrorMessage = err.Error()
		} else {
			entry.Status = "SENT"
			if c.SMTPSource == "DEFAULT" {
				log.Printf("[notifications] %s %s %s sent with the AlephK default SMTP", entityType, event, audience)
			}
		}
	}

	if entry.Status != "SENT" {
		// Recipient masked in the server log (2026-10-25, security S11); the
		// full address stays in tbl_notification_log for the company.
		log.Printf("[notifications] %s %s %s to %s: %s (%s)",
			entityType, event, audience, emailnotifier.MaskEmail(entry.Recipient), entry.Status, entry.ErrorMessage)
	}
	if err := entry.Insert(); err != nil {
		log.Printf("[notifications] could not write log: %v", err)
	}
	// One row per blind copy (customer service), only when an email went out
	// or failed (a skipped email had no copies).
	if entry.Status != "SKIPPED" {
		for _, blind := range bcc {
			copyEntry := entry
			copyEntry.Audience = AudienceCustomerService
			copyEntry.Recipient = blind
			if err := copyEntry.Insert(); err != nil {
				log.Printf("[notifications] could not write log: %v", err)
			}
		}
	}
	return entry.Status, entry.ErrorMessage
}

// customerServiceBcc returns the blind copy for a customer email: the
// company customer service email, when set and different from the customer.
func customerServiceBcc(c companyContext, to string) []string {
	cs := strings.TrimSpace(c.CustomerServiceEmail)
	if cs == "" || strings.EqualFold(cs, strings.TrimSpace(to)) {
		return nil
	}
	return []string{cs}
}

// defaultCustomerEmailDailyLimit is the default maximum of customer emails
// one address receives per day, from all companies together
// (tbl_configuration CUSTOMER_EMAIL_DAILY_LIMIT overrides it).
const defaultCustomerEmailDailyLimit = 10

// customerEmailLimitReached tells if the address already received today's
// maximum of customer emails (false on counting errors: never block real mail).
func customerEmailLimitReached(to string) bool {
	limit := defaultCustomerEmailDailyLimit
	if value, err := configuration.GetParameter("CUSTOMER_EMAIL_DAILY_LIMIT"); err == nil && strings.TrimSpace(value) != "" {
		if number, convErr := strconv.Atoi(strings.TrimSpace(value)); convErr == nil && number > 0 {
			limit = number
		}
	}
	now := time.Now()
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	count, err := notifications.CountSentToRecipientSince(to, since)
	if err != nil {
		log.Printf("[notifications] daily email count failed: %v", err)
		return false
	}
	return count >= limit
}

// runAsync runs fn in the background and never lets a panic escape.
func runAsync(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[notifications] %s panic: %v", name, r)
			}
		}()
		fn()
	}()
}

// manageLink builds the customer's private link to the showcase page.
func manageLink(companyID, token string) string {
	if strings.TrimSpace(token) == "" {
		return ""
	}
	// .env SHOWCASE_BASE_URL wins (2026-10-31), then tbl_configuration.
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("SHOWCASE_BASE_URL")), "/")
	if base == "" {
		base, _ = configuration.GetParameter("SHOWCASE_BASE_URL")
		base = strings.TrimRight(strings.TrimSpace(base), "/")
	}
	if base == "" {
		base = defaultShowcaseBaseURL
	}
	return base + "/showcase/" + url.PathEscape(companyID) + "/r/" + url.PathEscape(token)
}

// whatsappLink returns a wa.me link with a prefilled message, or "".
func whatsappLink(number, message string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, number)
	if len(digits) < 8 {
		return ""
	}
	return "https://wa.me/" + digits + "?text=" + url.QueryEscape(message)
}

// footer adds the manage link, the WhatsApp link and the signature.
// withCancel adds a small "Need to cancel? Request it here" line that opens
// the cancellation request form on the manage page.
func footer(c companyContext, link, linkLabel, whatsappMessage string, withCancel bool) string {
	t := c.texts()
	var b strings.Builder
	if link != "" {
		b.WriteString("\n" + linkLabel + ":\n" + link + "\n")
		if withCancel {
			b.WriteString("\n" + t.cancelRequestLine + ":\n" + link + "?cancel=1\n")
		}
	}
	if wa := whatsappLink(c.Whatsapp, whatsappMessage); wa != "" {
		b.WriteString("\n" + t.whatsappLine + ":\n" + wa + "\n")
	}
	b.WriteString("\n" + t.thanks + "\n" + c.displayName() + "\n")
	return b.String()
}

// paymentBlock tells the customer how to pay: the company's instructions
// (when set) and that online payments are not available yet.
func paymentBlock(c companyContext) string {
	t := c.texts()
	var b strings.Builder
	b.WriteString("\n" + t.paymentTitle + ":\n")
	if c.PaymentText != "" {
		b.WriteString(c.PaymentText + "\n")
	}
	b.WriteString(t.paymentSoon + "\n")
	return b.String()
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// productInfo returns the product name and currency symbol ("$" default).
func productInfo(companyID, productID string) (string, string) {
	p := products.ProductS{CompanyID: companyID, ProductInternalID: productID}
	if err := p.Select(); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("[notifications] product lookup failed: %v", err)
		}
		return "", "$"
	}
	symbol := strings.TrimSpace(p.CurrencySymbol)
	if symbol == "" {
		symbol = "$"
	}
	return p.Name, symbol
}

// formatMoney renders 45000 as "₡45,000" and 12.5 as "$12.50".
func formatMoney(amount float64, symbol string) string {
	if symbol == "" {
		symbol = "$"
	}
	decimals := 2
	if amount == math.Trunc(amount) {
		decimals = 0
	}
	raw := strconv.FormatFloat(math.Abs(amount), 'f', decimals, 64)
	intPart, fracPart := raw, ""
	if dot := strings.IndexByte(raw, '.'); dot >= 0 {
		intPart, fracPart = raw[:dot], raw[dot:]
	}
	var grouped strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			grouped.WriteByte(',')
		}
		grouped.WriteRune(r)
	}
	sign := ""
	if amount < 0 {
		sign = "-"
	}
	space := ""
	if len([]rune(symbol)) > 1 {
		space = " "
	}
	return sign + symbol + space + grouped.String() + fracPart
}

// statusLabel turns a status code into words in the company language.
func statusLabel(c companyContext, status string) string {
	code := strings.ToUpper(strings.TrimSpace(status))
	if label, ok := c.texts().statuses[code]; ok {
		return label
	}
	words := strings.Fields(strings.ToLower(strings.ReplaceAll(code, "_", " ")))
	if len(words) == 0 {
		return status
	}
	words[0] = strings.ToUpper(words[0][:1]) + words[0][1:]
	return strings.Join(words, " ")
}

// timeRange renders "10:00–11:00" or the all-day text.
func timeRange(c companyContext, start, end string) string {
	start, end = shortHour(start), shortHour(end)
	if (start == "00:00" && (end == "23:59" || end == "00:00")) || (start == "" && end == "") {
		return c.texts().allDay
	}
	if end == "" || end == start {
		return start
	}
	return start + "–" + end
}

// timeRangeOn is timeRange plus the company's time zone ON THAT DATE
// (2026-11-04, summer time included), e.g.
// "10:00–11:00 · Hora de Costa Rica, GMT−6".
func timeRangeOn(c companyContext, date, start, end string) string {
	when := timeRange(c, start, end)
	if c.Location == nil {
		return when
	}
	hour := shortHour(start)
	if when == c.texts().allDay {
		hour = "12:00"
	}
	return when + " · " + bltimezone.ZoneTextOn(c.Location, c.ZoneLabel, date, hour)
}

func shortHour(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 5 {
		return value[:5]
	}
	return value
}

func line(label, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return label + ": " + value + "\n"
}
