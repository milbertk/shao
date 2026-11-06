package assistant

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// SumCompanyTokensSince returns the input and output tokens used by ALL the
// assistants of a company (internal and store) since the given time.
func SumCompanyTokensSince(companyID string, since time.Time) (int64, int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, 0, fmt.Errorf("database connection error: %w", err)
	}
	var input, output int64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0)
		FROM public.tbl_ai_message
		WHERE company_id = $1
		  AND created_at >= $2
	`, strings.TrimSpace(companyID), since).Scan(&input, &output)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to sum assistant tokens: %w", err)
	}
	return input, output, nil
}

// CountCompanyQuestionsSince returns how many questions (user messages) ALL
// the assistants of a company (internal and store) received since the given
// time. One question = one interaction (2026-10-25, security S14).
func CountCompanyQuestionsSince(companyID string, since time.Time) (int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	var total int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM public.tbl_ai_message
		WHERE company_id = $1
		  AND role = 'user'
		  AND created_at >= $2
	`, strings.TrimSpace(companyID), since).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count assistant questions: %w", err)
	}
	return total, nil
}

// CompanyAISettingsS is one row of public.tbl_company_ai_settings: per-company
// cap and on/off override (set by hand / future super-admin screen).
type CompanyAISettingsS struct {
	CompanyID       string
	MonthlyCapUSD   *float64
	EnabledOverride *bool
}

// SelectCompanyAISettings returns the settings of a company, or nil when it
// has none.
func SelectCompanyAISettings(companyID string) (*CompanyAISettingsS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	item := CompanyAISettingsS{CompanyID: strings.TrimSpace(companyID)}
	var capUSD sql.NullFloat64
	var enabled sql.NullBool
	err = db.QueryRow(`
		SELECT monthly_cap_usd, enabled_override
		FROM public.tbl_company_ai_settings
		WHERE company_id = $1
	`, item.CompanyID).Scan(&capUSD, &enabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read company AI settings: %w", err)
	}
	if capUSD.Valid {
		v := capUSD.Float64
		item.MonthlyCapUSD = &v
	}
	if enabled.Valid {
		v := enabled.Bool
		item.EnabledOverride = &v
	}
	return &item, nil
}
