package plans

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// PlanS is one row of public.tbl_plans. Nil limits mean "unlimited";
// a nil price means "ask for the price".
type PlanS struct {
	PlanCode        string   `json:"planCode"`
	Name            string   `json:"name"`
	Description     *string  `json:"description,omitempty"`
	MonthlyPriceUSD *float64 `json:"monthlyPriceUsd"`
	MaxUsers        *int     `json:"maxUsers"`
	MaxProducts     *int     `json:"maxProducts"`
	AllReports      bool     `json:"allReports"`
	WebBookings     bool     `json:"webBookings"`
	AIAssistant     bool     `json:"aiAssistant"`
	TrialDays       *int     `json:"trialDays,omitempty"`
	// Human support hours included each month (0 = none) and storage in GB
	// (nil = not defined). Shown in the plans window (2026-10-22).
	SupportHoursMonth float64 `json:"supportHoursMonth"`
	StorageGB         *int    `json:"storageGb"`
	// AI assistant questions per month (internal + store together). nil =
	// computed from the price (max 10%, see blplan.AIInteractionsLimit);
	// for free plans (TRIAL) it is the fixed limit (2026-10-25, security S14).
	AIInteractionsMonth *int `json:"aiInteractionsMonth"`
	SortOrder           int  `json:"sortOrder"`
	Requestable         bool `json:"requestable"`
}

const planColumns = `
	plan_code, name, description, monthly_price_usd, max_users, max_products,
	all_reports, web_bookings, ai_assistant, trial_days, sort_order, requestable,
	support_hours_month, storage_gb, ai_interactions_month
`

func scanPlan(scanner interface{ Scan(...any) error }) (*PlanS, error) {
	var (
		p           PlanS
		description sql.NullString
		price       sql.NullFloat64
		maxUsers    sql.NullInt64
		maxProducts sql.NullInt64
		trialDays   sql.NullInt64
		storageGB   sql.NullInt64
		aiMonth     sql.NullInt64
	)
	if err := scanner.Scan(
		&p.PlanCode, &p.Name, &description, &price, &maxUsers, &maxProducts,
		&p.AllReports, &p.WebBookings, &p.AIAssistant, &trialDays, &p.SortOrder, &p.Requestable,
		&p.SupportHoursMonth, &storageGB, &aiMonth,
	); err != nil {
		return nil, err
	}
	p.PlanCode = strings.ToUpper(strings.TrimSpace(p.PlanCode))
	if description.Valid {
		p.Description = &description.String
	}
	if price.Valid {
		p.MonthlyPriceUSD = &price.Float64
	}
	p.MaxUsers = nullIntPtr(maxUsers)
	p.MaxProducts = nullIntPtr(maxProducts)
	p.TrialDays = nullIntPtr(trialDays)
	p.StorageGB = nullIntPtr(storageGB)
	p.AIInteractionsMonth = nullIntPtr(aiMonth)
	return &p, nil
}

func nullIntPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int64)
	return &v
}

// SelectActivePlans returns the active plans ordered by sort_order.
func SelectActivePlans() ([]PlanS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	rows, err := db.Query(`
		SELECT ` + planColumns + `
		FROM public.tbl_plans
		WHERE active = TRUE
		ORDER BY sort_order, plan_code
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to read plans: %w", err)
	}
	defer rows.Close()

	var result []PlanS
	for rows.Next() {
		plan, err := scanPlan(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
		result = append(result, *plan)
	}
	return result, rows.Err()
}

// SelectPlan returns one plan by code (nil, nil when it does not exist).
func SelectPlan(planCode string) (*PlanS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	plan, err := scanPlan(db.QueryRow(`
		SELECT `+planColumns+`
		FROM public.tbl_plans
		WHERE plan_code = $1
	`, strings.ToUpper(strings.TrimSpace(planCode))))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plan: %w", err)
	}
	return plan, nil
}
