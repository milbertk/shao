package blplan

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/milbertk/shao/bussunessLedger/bltimezone"
	"github.com/milbertk/shao/dataBasesLedger/configuration"
	"github.com/milbertk/shao/dataBasesLedger/planrequests"
	emailnotifier "github.com/milbertk/shao/shared"
)

// Plan requests (2026-10-22): a company member asks for another plan from
// the "Your plan" window (also without an active plan: read-only companies
// must be able to ask for a plan). TRIAL can always be requested; the email
// shows the company's previous trial requests so the owner decides. The request is saved as PENDING and emailed with
// the PLATFORM SMTP (tbl_configuration SMTP_*) to PLAN_REQUEST_EMAIL_TO
// (SMTP_TO_DEFAULT when empty). The requester gets a confirmation email.
// Payments are manual for now: an agent contacts the company.

var (
	// ErrPlanRequestInvalid wraps validation problems (HTTP 400).
	ErrPlanRequestInvalid = errors.New("invalid plan request")
	// ErrTooManyPlanRequests: too many requests from the company today (HTTP 429).
	ErrTooManyPlanRequests = errors.New("PLAN_REQUEST_LIMIT: too many plan requests today, an agent will contact you soon")
)

const maxPlanRequestsPerDay = 5

// PlanRequestInput is what the request form sends.
type PlanRequestInput struct {
	RequestedPlan string `json:"requestedPlan"`
	ContactName   string `json:"contactName"`
	ContactEmail  string `json:"contactEmail"`
	ContactPhone  string `json:"contactPhone"`
	Message       string `json:"message"`
	Language      string `json:"language"`
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrPlanRequestInvalid, fmt.Sprintf(format, args...))
}

func tooLong(value string, max int) bool { return utf8.RuneCountInString(value) > max }

// normalizePlanRequest trims and validates the form (pure logic, tested).
func normalizePlanRequest(input PlanRequestInput, catalog []PlanCatalogEntry) (PlanRequestInput, error) {
	input.RequestedPlan = strings.ToUpper(strings.TrimSpace(input.RequestedPlan))
	input.ContactName = strings.TrimSpace(input.ContactName)
	input.ContactEmail = strings.TrimSpace(input.ContactEmail)
	input.ContactPhone = strings.TrimSpace(input.ContactPhone)
	input.Message = strings.TrimSpace(input.Message)
	input.Language = strings.ToLower(strings.TrimSpace(input.Language))
	if !strings.HasPrefix(input.Language, "es") {
		input.Language = "en"
	} else {
		input.Language = "es"
	}

	requestable := false
	for _, p := range catalog {
		if p.Code == input.RequestedPlan && p.Requestable {
			requestable = true
			break
		}
	}
	switch {
	case input.RequestedPlan == "":
		return input, invalid("requestedPlan is required")
	case !requestable:
		return input, invalid("the plan %q cannot be requested", input.RequestedPlan)
	case input.ContactName == "":
		return input, invalid("contactName is required")
	case tooLong(input.ContactName, 150):
		return input, invalid("contactName must have at most 150 characters")
	case input.ContactEmail == "":
		return input, invalid("contactEmail is required")
	case tooLong(input.ContactEmail, 200) || !emailnotifier.ValidEmail(input.ContactEmail):
		return input, invalid("contactEmail is not a valid email")
	case tooLong(input.ContactPhone, 40):
		return input, invalid("contactPhone must have at most 40 characters")
	case tooLong(input.Message, 1000):
		return input, invalid("message must have at most 1000 characters")
	}
	return input, nil
}

// PlanCatalogEntry is the minimum the validation needs from a plan.
type PlanCatalogEntry struct {
	Code        string
	Name        string
	Requestable bool
}

func catalogEntries() []PlanCatalogEntry {
	var entries []PlanCatalogEntry
	for _, p := range BPlanCatalog() {
		entries = append(entries, PlanCatalogEntry{Code: p.PlanCode, Name: p.Name, Requestable: p.Requestable})
	}
	return entries
}

// BCreatePlanRequest validates, saves and emails a plan request.
// userEmail is the signed-in user who sent the form (shown in the email).
func BCreatePlanRequest(companyID, userID, userEmail string, input PlanRequestInput) (*planrequests.PlanRequestS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, invalid("companyId is required")
	}

	entries := catalogEntries()
	input, err := normalizePlanRequest(input, entries)
	if err != nil {
		return nil, err
	}

	// Basic company first (2026-10-29): name, logo and header in Company page.
	if err := BCheckBasicCompany(companyID); err != nil {
		return nil, err
	}

	count, err := planrequests.CountCompanyRequestsSince(companyID, time.Now().Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}
	if count >= maxPlanRequestsPerDay {
		return nil, ErrTooManyPlanRequests
	}

	current, err := BGetCompanyPlan(companyID)
	if err != nil {
		return nil, err
	}

	// Previous requests of the same plan (read BEFORE saving this one).
	history, err := planrequests.SelectPlanHistory(companyID, input.RequestedPlan)
	if err != nil {
		log.Printf("[blplan] plan history not read: companyId=%s error=%v", companyID, err)
		history = nil
	}

	request := planrequests.PlanRequestS{
		CompanyID:         companyID,
		CompanyName:       current.CompanyName,
		CurrentPlan:       current.PlanCode,
		RequestedPlan:     input.RequestedPlan,
		ContactName:       input.ContactName,
		ContactEmail:      input.ContactEmail,
		ContactPhone:      input.ContactPhone,
		Message:           input.Message,
		Language:          input.Language,
		RequestedByUserID: strings.TrimSpace(userID),
	}
	if err := request.Insert(); err != nil {
		return nil, err
	}

	requestedName := input.RequestedPlan
	for _, e := range entries {
		if e.Code == input.RequestedPlan {
			requestedName = e.Name
		}
	}

	// Emails never block the answer.
	go sendPlanRequestEmails(request, current, requestedName, strings.TrimSpace(userEmail), history)
	return &request, nil
}

// platformEmailReady tells if the platform SMTP and a recipient are set.
func platformEmailReady() (string, bool) {
	cfg, err := configuration.GetParameters("SMTP_HOST", "SMTP_USER", "PLAN_REQUEST_EMAIL_TO", "SMTP_TO_DEFAULT")
	if err != nil {
		return "", false
	}
	to := strings.TrimSpace(cfg["PLAN_REQUEST_EMAIL_TO"])
	if to == "" {
		to = strings.TrimSpace(cfg["SMTP_TO_DEFAULT"])
	}
	ready := to != "" && strings.TrimSpace(cfg["SMTP_HOST"]) != "" && strings.TrimSpace(cfg["SMTP_USER"]) != ""
	return to, ready
}

// historyLine tells the owner how many times the company asked for this
// plan before (important for TRIAL, which is approved by hand).
func historyLine(history *planrequests.PlanHistory, requestedName string) string {
	if history == nil {
		return "Previous " + requestedName + " requests: (not available)"
	}
	if history.Requests == 0 {
		return "Previous " + requestedName + " requests: none (first time)"
	}
	last := "-"
	if history.Last != nil {
		// AlephK's own emails use Costa Rica time (the server runs in UTC).
		last = history.Last.In(bltimezone.DefaultLocation()).Format("2006-01-02")
	}
	return fmt.Sprintf("Previous %s requests: %d (approved: %d), last one on %s", requestedName, history.Requests, history.Approved, last)
}

func sendPlanRequestEmails(request planrequests.PlanRequestS, current *CompanyPlan, requestedName, userEmail string, history *planrequests.PlanHistory) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[blplan] plan request email panic: %v", r)
		}
	}()

	to, ready := platformEmailReady()
	if !ready {
		log.Printf("[blplan] plan request %s saved; email not sent (set PLAN_REQUEST_EMAIL_TO and the platform SMTP_*)", request.PlanRequestID)
	} else {
		body := strings.Join([]string{
			"New plan request from AlephK",
			"",
			"Company name: " + emptyDash(request.CompanyName),
			"Company ID: " + request.CompanyID,
			"Sent by user: " + emptyDash(userEmail) + " (user ID " + emptyDash(request.RequestedByUserID) + ")",
			"Current plan: " + current.PlanCode + activeText(current),
			"Requested plan: " + requestedName + " (" + request.RequestedPlan + ")",
			historyLine(history, requestedName),
			fmt.Sprintf("Active users: %d · Products: %d", current.UsersCount, current.ProductsCount),
			"",
			"Contact: " + request.ContactName,
			"Email: " + request.ContactEmail,
			"Phone: " + emptyDash(request.ContactPhone),
			"Language: " + request.Language,
			"",
			"Message:",
			emptyDash(request.Message),
			"",
			"Request id: " + request.PlanRequestID,
			"Approve it with the SQL in sql/migration_plans_entrepreneur_special.sql (HOW TO USE).",
		}, "\n")
		err := emailnotifier.Send(emailnotifier.EmailMessage{
			To:      to,
			ReplyTo: request.ContactEmail,
			Subject: fmt.Sprintf("[AlephK] Plan request: %s (%s) -> %s", emptyDash(request.CompanyName), request.CompanyID, requestedName),
			Body:    body,
		})
		if err != nil {
			log.Printf("[blplan] plan request email failed: id=%s error=%v", request.PlanRequestID, err)
		} else if err := planrequests.MarkEmailSent(request.PlanRequestID); err != nil {
			log.Printf("[blplan] email_sent not saved: id=%s error=%v", request.PlanRequestID, err)
		}
	}

	if !ready {
		return
	}
	subject, body := confirmationText(request, requestedName)
	if err := emailnotifier.Send(emailnotifier.EmailMessage{
		To:      request.ContactEmail,
		Subject: subject,
		Body:    body,
	}); err != nil {
		log.Printf("[blplan] plan request confirmation failed: id=%s error=%v", request.PlanRequestID, err)
	}
}

func confirmationText(request planrequests.PlanRequestS, requestedName string) (string, string) {
	if request.Language == "es" {
		return "Recibimos tu solicitud de plan " + requestedName,
			"Hola " + request.ContactName + ",\n\n" +
				"Recibimos tu solicitud del plan " + requestedName + " para " + request.CompanyName + ".\n" +
				"Un agente se pondrá en contacto contigo pronto con los pasos y el pago.\n\n" +
				"Gracias por usar AlephK."
	}
	return "We received your " + requestedName + " plan request",
		"Hello " + request.ContactName + ",\n\n" +
			"We received your request for the " + requestedName + " plan for " + request.CompanyName + ".\n" +
			"An agent will contact you soon with the next steps and the payment.\n\n" +
			"Thank you for using AlephK."
}

func activeText(current *CompanyPlan) string {
	if current.Active {
		return ""
	}
	return " (no active plan: read-only)"
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
