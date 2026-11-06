package companyplan

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CompanyPlanInfoS is the shao plan of a company with its dates (2026-10-22).
type CompanyPlanInfoS struct {
	CompanyID     string
	CompanyName   string     // Company page name (tbl_page_configuration.display_name)
	Plan          string     // upper case, "" when the company has no plan row
	PlanStartedAt *time.Time // tbl_company_plan.started_at
	PlanEndsAt    *time.Time // tbl_company_plan.ends_at (TRIAL end)
	// Price agreed with this company (negotiation), nil = the plan price.
	AgreedPriceUSD *float64 // tbl_company_plan.agreed_price_usd
}

// SelectCompanyPlanInfo returns the plan, its dates and the company name.
// A company without a plan row returns Plan "" (it works as FREE).
func SelectCompanyPlanInfo(companyID string) (*CompanyPlanInfoS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	var (
		name      sql.NullString
		plan      sql.NullString
		startedAt sql.NullTime
		endsAt    sql.NullTime
		agreed    sql.NullFloat64
	)
	err = db.QueryRow(`
		SELECT
			(SELECT pc.display_name FROM public.tbl_page_configuration pc WHERE pc.company_id::text = $1 LIMIT 1),
			cp.plan_code, cp.started_at, cp.ends_at, cp.agreed_price_usd
		FROM (SELECT 1) AS one
		LEFT JOIN public.tbl_company_plan cp ON cp.company_id = $1
	`, companyID).Scan(&name, &plan, &startedAt, &endsAt, &agreed)
	if err != nil {
		return nil, fmt.Errorf("failed to read company plan: %w", err)
	}

	info := CompanyPlanInfoS{
		CompanyID:   companyID,
		CompanyName: strings.TrimSpace(name.String),
		Plan:        strings.ToUpper(strings.TrimSpace(plan.String)),
	}
	if startedAt.Valid {
		info.PlanStartedAt = &startedAt.Time
	}
	if endsAt.Valid {
		info.PlanEndsAt = &endsAt.Time
	}
	if agreed.Valid {
		info.AgreedPriceUSD = &agreed.Float64
	}
	return &info, nil
}

// SetCompanyPlanDates fills the start/end of the current plan only when they
// are still empty (used to start the TRIAL clock the first time).
func SetCompanyPlanDates(companyID string, startedAt, endsAt time.Time) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	_, err = db.Exec(`
		UPDATE public.tbl_company_plan
		SET started_at = COALESCE(started_at, $2),
		    ends_at    = COALESCE(ends_at, $3),
		    updated_at = now()
		WHERE company_id = $1
	`, strings.TrimSpace(companyID), startedAt, endsAt)
	if err != nil {
		return fmt.Errorf("failed to set company plan dates: %w", err)
	}
	return nil
}
