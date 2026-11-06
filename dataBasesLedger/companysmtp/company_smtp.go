package companysmtp

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CompanySmtpS represents one row from public.tbl_company_smtp: the company's
// own mail server. PasswordEncrypted is never sent to the frontend.
type CompanySmtpS struct {
	CompanySmtpID     string     `json:"companySmtpId"`
	CompanyID         string     `json:"companyId"`
	Host              string     `json:"host"`
	Port              int        `json:"port"`
	Security          string     `json:"security"`
	Username          string     `json:"username"`
	PasswordEncrypted string     `json:"-"`
	FromEmail         string     `json:"fromEmail"`
	FromName          *string    `json:"fromName,omitempty"`
	Enabled           bool       `json:"enabled"`
	LastTestAt        *time.Time `json:"lastTestAt,omitempty"`
	LastTestOK        *bool      `json:"lastTestOk,omitempty"`
	LastTestError     *string    `json:"lastTestError,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

const companySmtpColumns = `
	company_smtp_id,
	company_id,
	host,
	port,
	security,
	username,
	password_encrypted,
	from_email,
	from_name,
	enabled,
	last_test_at,
	last_test_ok,
	last_test_error,
	created_at,
	updated_at
`

func scanCompanySmtp(row interface {
	Scan(dest ...any) error
}, c *CompanySmtpS) error {
	return row.Scan(
		&c.CompanySmtpID,
		&c.CompanyID,
		&c.Host,
		&c.Port,
		&c.Security,
		&c.Username,
		&c.PasswordEncrypted,
		&c.FromEmail,
		&c.FromName,
		&c.Enabled,
		&c.LastTestAt,
		&c.LastTestOK,
		&c.LastTestError,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
}

// SelectByCompany loads the SMTP of the company in CompanyID.
// Returns a wrapped sql.ErrNoRows when the company has none.
func (a *CompanySmtpS) SelectByCompany() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(a.CompanyID)
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		SELECT ` + companySmtpColumns + `
		FROM public.tbl_company_smtp
		WHERE company_id = $1
	`
	if err := scanCompanySmtp(db.QueryRow(query, companyID), a); err != nil {
		return fmt.Errorf("failed to select company smtp: %w", err)
	}
	return nil
}

// Upsert creates or replaces the SMTP of the company.
func (a *CompanySmtpS) Upsert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(a.CompanyID)
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		INSERT INTO public.tbl_company_smtp (
			company_id, host, port, security, username,
			password_encrypted, from_email, from_name, enabled
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (company_id) DO UPDATE SET
			host               = EXCLUDED.host,
			port               = EXCLUDED.port,
			security           = EXCLUDED.security,
			username           = EXCLUDED.username,
			password_encrypted = EXCLUDED.password_encrypted,
			from_email         = EXCLUDED.from_email,
			from_name          = EXCLUDED.from_name,
			enabled            = EXCLUDED.enabled,
			updated_at         = now()
		RETURNING ` + companySmtpColumns

	err = scanCompanySmtp(db.QueryRow(
		query,
		companyID,
		a.Host,
		a.Port,
		a.Security,
		a.Username,
		a.PasswordEncrypted,
		a.FromEmail,
		a.FromName,
		a.Enabled,
	), a)
	if err != nil {
		return fmt.Errorf("failed to save company smtp: %w", err)
	}
	return nil
}

// SaveTestResult stores the result of the last "send test email".
func SaveTestResult(companyID string, ok bool, errorMessage string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	var errValue sql.NullString
	if strings.TrimSpace(errorMessage) != "" {
		errValue = sql.NullString{String: errorMessage, Valid: true}
	}

	_, err = db.Exec(`
		UPDATE public.tbl_company_smtp
		SET last_test_at = now(), last_test_ok = $2, last_test_error = $3
		WHERE company_id = $1
	`, strings.TrimSpace(companyID), ok, errValue)
	if err != nil {
		return fmt.Errorf("failed to save smtp test result: %w", err)
	}
	return nil
}
