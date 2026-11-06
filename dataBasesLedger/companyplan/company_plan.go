package companyplan

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// The plan of a company for the shao platform lives in shao
// (public.tbl_company_plan, one row per company). It does not use
// public.companies.plan of usersAppAK: in the future one company can have
// different plans in different platforms (2026-10-22).

// SelectCompanyPlan returns the shao plan of a company in upper case.
// Returns "" when the company has no plan row (it works as FREE).
func SelectCompanyPlan(companyID string) (string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", fmt.Errorf("database connection error: %w", err)
	}

	var plan sql.NullString
	err = db.QueryRow(`
		SELECT plan_code
		FROM public.tbl_company_plan
		WHERE company_id = $1
	`, strings.TrimSpace(companyID)).Scan(&plan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read company plan: %w", err)
	}
	return strings.ToUpper(strings.TrimSpace(plan.String)), nil
}
