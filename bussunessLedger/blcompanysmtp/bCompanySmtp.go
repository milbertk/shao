package blcompanysmtp

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/milbertk/shao/dataBasesLedger/companysmtp"
	"github.com/milbertk/shao/dataBasesLedger/notifications"
	"github.com/milbertk/shao/dataBasesLedger/pageconfiguration"
	emailnotifier "github.com/milbertk/shao/shared"
)

// ErrCompanySmtpNotConfigured means the company has not saved its SMTP yet.
var ErrCompanySmtpNotConfigured = errors.New("smtp not configured")

var allowedSecurity = map[string]bool{"STARTTLS": true, "SSL": true}

// CompanySmtpView is what the management app sees. The password is NEVER
// returned; HasPassword only tells whether one is saved.
type CompanySmtpView struct {
	Configured    bool       `json:"configured"`
	Host          string     `json:"host"`
	Port          int        `json:"port"`
	Security      string     `json:"security"`
	Username      string     `json:"username"`
	FromEmail     string     `json:"fromEmail"`
	FromName      string     `json:"fromName"`
	Enabled       bool       `json:"enabled"`
	HasPassword   bool       `json:"hasPassword"`
	LastTestAt    *time.Time `json:"lastTestAt,omitempty"`
	LastTestOK    *bool      `json:"lastTestOk,omitempty"`
	LastTestError *string    `json:"lastTestError,omitempty"`
}

// CompanySmtpInput is the editable data. An empty Password keeps the saved one.
type CompanySmtpInput struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Security  string `json:"security"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FromEmail string `json:"fromEmail"`
	FromName  string `json:"fromName"`
	Enabled   bool   `json:"enabled"`
}

func toView(c *companysmtp.CompanySmtpS) *CompanySmtpView {
	if c == nil {
		return &CompanySmtpView{Configured: false, Port: 587, Security: "STARTTLS", Enabled: true}
	}
	fromName := ""
	if c.FromName != nil {
		fromName = *c.FromName
	}
	return &CompanySmtpView{
		Configured:    true,
		Host:          c.Host,
		Port:          c.Port,
		Security:      c.Security,
		Username:      c.Username,
		FromEmail:     c.FromEmail,
		FromName:      fromName,
		Enabled:       c.Enabled,
		HasPassword:   strings.TrimSpace(c.PasswordEncrypted) != "",
		LastTestAt:    c.LastTestAt,
		LastTestOK:    c.LastTestOK,
		LastTestError: c.LastTestError,
	}
}

func loadRow(companyID string) (*companysmtp.CompanySmtpS, error) {
	row := companysmtp.CompanySmtpS{CompanyID: companyID}
	if err := row.SelectByCompany(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// BSelectCompanySmtp returns the company's SMTP (without password).
func BSelectCompanySmtp(companyID string) (*CompanySmtpView, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	row, err := loadRow(companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load smtp configuration: %w", err)
	}
	return toView(row), nil
}

// BSaveCompanySmtp validates and saves the company's SMTP. The password is
// encrypted before it reaches the database.
func BSaveCompanySmtp(companyID string, input CompanySmtpInput) (*CompanySmtpView, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	host := strings.ToLower(strings.TrimSpace(input.Host))
	security := strings.ToUpper(strings.TrimSpace(input.Security))
	username := strings.TrimSpace(input.Username)
	fromEmail := strings.TrimSpace(input.FromEmail)
	fromName := strings.TrimSpace(input.FromName)

	switch {
	case host == "" || strings.ContainsAny(host, " /\\:@"):
		return nil, errors.New("invalid host: enter only the server name, e.g. smtp.gmail.com")
	case input.Port < 1 || input.Port > 65535:
		return nil, errors.New("invalid port: use 587 (STARTTLS) or 465 (SSL)")
	case !allowedSecurity[security]:
		return nil, errors.New("invalid security: use STARTTLS or SSL")
	case username == "":
		return nil, errors.New("username is required")
	case !emailnotifier.ValidEmail(fromEmail):
		return nil, errors.New("invalid fromEmail: enter one valid email address")
	case utf8.RuneCountInString(fromName) > 120:
		return nil, errors.New("invalid fromName: maximum 120 characters")
	}

	existing, err := loadRow(companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load smtp configuration: %w", err)
	}

	encrypted := ""
	if input.Password != "" {
		encrypted, err = emailnotifier.EncryptSecret(input.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to protect the password: %w", err)
		}
	} else if existing != nil {
		encrypted = existing.PasswordEncrypted // keep the saved password
	} else {
		return nil, errors.New("password is required")
	}

	row := companysmtp.CompanySmtpS{
		CompanyID:         companyID,
		Host:              host,
		Port:              input.Port,
		Security:          security,
		Username:          username,
		PasswordEncrypted: encrypted,
		FromEmail:         fromEmail,
		Enabled:           input.Enabled,
	}
	if fromName != "" {
		row.FromName = &fromName
	}

	if err := row.Upsert(); err != nil {
		return nil, fmt.Errorf("failed to save smtp configuration: %w", err)
	}
	return toView(&row), nil
}

// LoadCompanySmtpSettings returns the ready-to-use (decrypted) SMTP of the
// company, or ErrCompanySmtpNotConfigured when it is missing or disabled.
func LoadCompanySmtpSettings(companyID string) (*emailnotifier.SMTPSettings, error) {
	row, err := loadRow(strings.TrimSpace(companyID))
	if err != nil {
		return nil, err
	}
	if row == nil || !row.Enabled {
		return nil, ErrCompanySmtpNotConfigured
	}
	password, err := emailnotifier.DecryptSecret(row.PasswordEncrypted)
	if err != nil {
		return nil, err
	}
	fromName := ""
	if row.FromName != nil {
		fromName = *row.FromName
	}
	return &emailnotifier.SMTPSettings{
		Host:      row.Host,
		Port:      row.Port,
		Security:  row.Security,
		Username:  row.Username,
		Password:  password,
		FromEmail: row.FromEmail,
		FromName:  fromName,
	}, nil
}

// TestResult is returned by BTestCompanySmtp.
type TestResult struct {
	OK       bool   `json:"ok"`
	SentTo   string `json:"sentTo"`
	Error    string `json:"error,omitempty"`
	TestedAt string `json:"testedAt"`
}

// BTestCompanySmtp sends a test email with the saved SMTP. The recipient is
// `to` when given, otherwise the company notification email, otherwise the
// sender address. The result is stored and logged.
func BTestCompanySmtp(companyID, to string) (*TestResult, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	row, err := loadRow(companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load smtp configuration: %w", err)
	}
	if row == nil {
		return nil, errors.New("save the SMTP configuration first (smtp is required)")
	}

	to = strings.TrimSpace(to)
	displayName := ""
	language := "en"
	page := pageconfiguration.PageConfigurationS{CompanyID: companyID}
	if err := page.SelectByCompany(); err == nil {
		displayName = page.DisplayName
		if page.NotificationLanguage != "" {
			language = page.NotificationLanguage
		}
		if to == "" && page.NotificationEmail != nil {
			to = *page.NotificationEmail
		}
	}
	if to == "" {
		to = row.FromEmail
	}
	if !emailnotifier.ValidEmail(to) {
		return nil, errors.New("invalid test recipient email")
	}

	result := &TestResult{SentTo: to, TestedAt: time.Now().Format(time.RFC3339)}

	settings, err := LoadCompanySmtpSettings(companyID)
	if err == nil {
		subject := "Test email — your notifications are working"
		body := "Hello,\n\nThis is a test email from " + nameOr(displayName, "your company") + ".\n" +
			"If you can read it, your SMTP is configured correctly and customers will receive notifications.\n"
		if language == "es" {
			subject = "Correo de prueba — sus notificaciones funcionan"
			body = "Hola:\n\nEste es un correo de prueba de " + nameOr(displayName, "su empresa") + ".\n" +
				"Si puede leerlo, su SMTP está bien configurado y sus clientes recibirán notificaciones.\n"
		}
		err = emailnotifier.SendWithSettings(*settings, emailnotifier.EmailMessage{
			To:      to,
			Subject: subject,
			Body:    body,
		})
	}

	result.OK = err == nil
	if err != nil {
		result.Error = err.Error()
	}

	_ = companysmtp.SaveTestResult(companyID, result.OK, result.Error)
	status := "SENT"
	if !result.OK {
		status = "FAILED"
	}
	log := notifications.NotificationLogS{
		CompanyID: companyID, EntityType: "SMTP_TEST", EntityID: companyID,
		Event: "TEST", Audience: "COMPANY", Recipient: to, Status: status, ErrorMessage: result.Error,
	}
	_ = log.Insert()

	return result, nil
}

func nameOr(name, fallback string) string {
	if strings.TrimSpace(name) == "" {
		return fallback
	}
	return name
}
