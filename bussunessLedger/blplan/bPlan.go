package blplan

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/milbertk/shao/dataBasesLedger/companyplan"
	"github.com/milbertk/shao/dataBasesLedger/plans"
)

// =====================================================================
//  Company plans (2026-10-22).
//
//  The plan of a company for shao is public.tbl_company_plan in the shao
//  database. It is not usersAppAK's companies.plan: each platform has its
//  own plan. Its rules come from public.tbl_plans; when that table cannot
//  be read (migration not run yet) the built-in copy below is used, so the
//  app keeps working.
//
//  2026-10-23: there is no free plan anymore. Only TRIAL is free (for
//  testing). FREE became ENTREPRENEUR (paid). A company WITHOUT an active
//  plan (no row, unknown plan, or its plan/trial ended) works as NONE:
//  READ-ONLY (it sees its data but cannot create or change anything) until
//  a plan is assigned. SPECIAL has no limits and is assigned only by the
//  platform owner (SQL); it cannot be requested.
//
//  Every plan has every option (all reports, store bookings). Plans differ
//  only in: users, products, storage, AI assistant and human support.
//  A TRIAL without an end date gets start + trial_days the first time it is
//  read. Existing data is never removed: a company over its limit keeps
//  everything but cannot add more.
// =====================================================================

// Plan codes.
const (
	PlanEntrepreneur = "ENTREPRENEUR" // replaced FREE (2026-10-23), paid
	PlanBasic        = "BASIC"
	PlanPro          = "PRO"
	PlanEnterprise   = "ENTERPRISE"
	PlanTrial        = "TRIAL"   // the only free plan, for testing
	PlanSpecial      = "SPECIAL" // no limits, assigned only by the platform owner
	// PlanNone: no active plan -> read-only. Not a row of tbl_plans.
	PlanNone = "NONE"
	// PlanFree is the old code; companies still on it work as ENTREPRENEUR.
	PlanFree = "FREE"
)

// Feature is an option that some plans do not include.
type Feature string

const (
	FeatureAllReports  Feature = "ALL_REPORTS"  // every report (FREE: only Orders by status)
	FeatureWebBookings Feature = "WEB_BOOKINGS" // bookings from the public store
	FeatureAIAssistant Feature = "AI_ASSISTANT" // Aleph + store assistant
)

// Error codes sent to the screens in result.code.
const (
	CodeLimitUsers    = "PLAN_LIMIT_USERS"
	CodeLimitProducts = "PLAN_LIMIT_PRODUCTS"
	CodeLimitStorage  = "PLAN_LIMIT_STORAGE" // Limit = storage in GB
	CodeFeature       = "PLAN_FEATURE_LOCKED"
	CodeNoPlan        = "PLAN_REQUIRED" // no active plan: read-only
)

// PlanLimitError tells that the company's plan does not allow the action.
type PlanLimitError struct {
	Code     string  `json:"code"`
	Plan     string  `json:"plan"`
	PlanName string  `json:"planName"`
	Limit    int     `json:"limit,omitempty"`
	Feature  Feature `json:"feature,omitempty"`
}

func (e *PlanLimitError) Error() string {
	switch e.Code {
	case CodeLimitUsers:
		return fmt.Sprintf("Your %s plan allows %d active user(s). Request a plan to add more users.", e.PlanName, e.Limit)
	case CodeLimitProducts:
		return fmt.Sprintf("Your %s plan allows %d product(s). Request a plan to add more products.", e.PlanName, e.Limit)
	case CodeLimitStorage:
		return fmt.Sprintf("Your %s plan includes %d GB of storage and it is full. Delete files or request a plan with more storage.", e.PlanName, e.Limit)
	case CodeNoPlan:
		return "Your company has no active plan, so it is read-only. Choose a plan or request a free trial to continue."
	default:
		return fmt.Sprintf("This option is not included in your %s plan. Request a plan to use it.", e.PlanName)
	}
}

// AsPlanLimitError returns the plan error inside err, if any.
func AsPlanLimitError(err error) (*PlanLimitError, bool) {
	var limitErr *PlanLimitError
	if errors.As(err, &limitErr) {
		return limitErr, true
	}
	return nil, false
}

// CompanyPlan is the plan a company works with right now.
type CompanyPlan struct {
	PlanCode     string `json:"planCode"`     // effective plan (NONE when there is no active plan)
	AssignedPlan string `json:"assignedPlan"` // tbl_company_plan.plan_code
	// Active is false when the company has no active plan (read-only).
	Active      bool        `json:"active"`
	Plan        plans.PlanS `json:"plan"`
	StartedAt   *time.Time  `json:"startedAt,omitempty"`
	EndsAt      *time.Time  `json:"endsAt,omitempty"`
	DaysLeft    *int        `json:"daysLeft,omitempty"`
	Expired     bool        `json:"expired"` // the assigned plan ended (read-only)
	CompanyName string      `json:"companyName"`
	// Price agreed with this company (negotiated); nil = the plan price.
	AgreedPriceUSD *float64 `json:"agreedPriceUsd,omitempty"`
	UsersCount     int      `json:"usersCount"`
	ProductsCount  int      `json:"productsCount"`
	// AI assistant questions this month and the plan limit (nil = unlimited);
	// the limit is at most AIBudgetPercent (10%) of the price (2026-10-25).
	AIInteractionsLimit *int `json:"aiInteractionsLimit"`
	AIInteractionsUsed  int  `json:"aiInteractionsUsed"`
	AIBudgetPercent     int  `json:"aiBudgetPercent"`
}

func intPtr(v int) *int { return &v }

// builtInPlans mirrors the seed of the plan migrations
// (sql/migration_plans_entrepreneur_special.sql). nil limit = unlimited.
var builtInPlans = []plans.PlanS{
	{PlanCode: PlanTrial, Name: "Trial", MonthlyPriceUSD: floatPtr(0), MaxUsers: intPtr(10), AllReports: true, WebBookings: true, AIAssistant: true, TrialDays: intPtr(30), SortOrder: 5, Requestable: true, StorageGB: intPtr(5), AIInteractionsMonth: intPtr(defaultFreePlanAIInteractions)},
	{PlanCode: PlanEntrepreneur, Name: "Entrepreneur", MonthlyPriceUSD: floatPtr(10), MaxUsers: intPtr(1), MaxProducts: intPtr(100), AllReports: true, WebBookings: true, SortOrder: 10, Requestable: true, StorageGB: intPtr(5)},
	{PlanCode: PlanBasic, Name: "Basic", MaxUsers: intPtr(10), AllReports: true, WebBookings: true, SortOrder: 20, Requestable: true, StorageGB: intPtr(10)},
	{PlanCode: PlanPro, Name: "Pro", MaxUsers: intPtr(20), AllReports: true, WebBookings: true, AIAssistant: true, SortOrder: 30, Requestable: true, SupportHoursMonth: 1, StorageGB: intPtr(10)},
	{PlanCode: PlanEnterprise, Name: "Enterprise", MaxUsers: intPtr(100), AllReports: true, WebBookings: true, AIAssistant: true, SortOrder: 40, Requestable: true, SupportHoursMonth: 2, StorageGB: intPtr(25)},
	{PlanCode: PlanSpecial, Name: "Special", AllReports: true, WebBookings: true, AIAssistant: true, SortOrder: 60, Requestable: false},
}

// nonePlan is used when the company has no active plan: read-only, no AI.
var nonePlan = plans.PlanS{PlanCode: PlanNone, Name: "No plan", MaxUsers: intPtr(0), MaxProducts: intPtr(0), StorageGB: intPtr(0), AllReports: true, WebBookings: false}

func floatPtr(v float64) *float64 { return &v }

// BVisiblePlans returns the plans a company can see in the plans window:
// the requestable ones plus its own plan (so SPECIAL is shown only to the
// company that has it).
func BVisiblePlans(currentPlanCode string) []plans.PlanS {
	var result []plans.PlanS
	for _, p := range BPlanCatalog() {
		if p.Requestable || p.PlanCode == strings.ToUpper(strings.TrimSpace(currentPlanCode)) {
			result = append(result, p)
		}
	}
	// Show the real monthly AI interactions of each plan (max 10% of price).
	return withAIInteractions(result)
}

// BPlanCatalog returns the active plans (database, or the built-in copy).
func BPlanCatalog() []plans.PlanS {
	list, err := plans.SelectActivePlans()
	if err != nil || len(list) == 0 {
		if err != nil {
			log.Printf("[blplan] plans table not read, using built-in plans: %v", err)
		}
		return append([]plans.PlanS(nil), builtInPlans...)
	}
	return list
}

func findPlan(catalog []plans.PlanS, code string) (plans.PlanS, bool) {
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, p := range catalog {
		if p.PlanCode == code {
			return p, true
		}
	}
	return plans.PlanS{}, false
}

// resolvePlan decides the effective plan (pure logic, tested).
func resolvePlan(info companyplan.CompanyPlanInfoS, catalog []plans.PlanS, now time.Time) CompanyPlan {
	assigned := strings.ToUpper(strings.TrimSpace(info.Plan))

	result := CompanyPlan{
		PlanCode:     PlanNone,
		AssignedPlan: assigned,
		Plan:         nonePlan,
		StartedAt:    info.PlanStartedAt,
		EndsAt:       info.PlanEndsAt,
		CompanyName:  info.CompanyName,
	}
	if assigned == "" {
		return result // no plan row: read-only
	}

	plan, ok := findPlan(catalog, assigned)
	if !ok && assigned == PlanFree {
		// Old FREE companies (migration not run yet) work as ENTREPRENEUR.
		plan, ok = findPlan(catalog, PlanEntrepreneur)
		if !ok {
			plan, ok = findPlan(builtInPlans, PlanEntrepreneur)
		}
	}
	if !ok {
		log.Printf("[blplan] unknown plan %q for company %s, no active plan", assigned, info.CompanyID)
		return result
	}

	result.PlanCode = plan.PlanCode
	result.Plan = plan
	result.Active = true
	// Negotiated price of this company (only while its plan is active).
	result.AgreedPriceUSD = info.AgreedPriceUSD

	// A TRIAL without an end date ends trial_days after its start (or now).
	if result.EndsAt == nil && plan.TrialDays != nil && *plan.TrialDays > 0 {
		start := now
		if info.PlanStartedAt != nil {
			start = *info.PlanStartedAt
		}
		end := start.AddDate(0, 0, *plan.TrialDays)
		result.StartedAt = &start
		result.EndsAt = &end
	}

	if result.EndsAt != nil {
		left := int(result.EndsAt.Sub(now).Hours() / 24)
		if result.EndsAt.After(now) {
			// Round up: "ends in 1 day" until the last hour.
			if result.EndsAt.Sub(now) > time.Duration(left)*24*time.Hour {
				left++
			}
			result.DaysLeft = &left
		} else {
			// Ended: read-only until a new plan is assigned.
			result.Expired = true
			result.Active = false
			result.AgreedPriceUSD = nil
			result.PlanCode = PlanNone
			result.Plan = nonePlan
			zero := 0
			result.DaysLeft = &zero
		}
	}
	return result
}

// BGetCompanyPlan returns the plan the company works with now, with its
// current number of active users and products.
func BGetCompanyPlan(companyID string) (*CompanyPlan, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	info, err := companyplan.SelectCompanyPlanInfo(companyID)
	if err != nil {
		// Columns missing (migration not run) -> plan only.
		log.Printf("[blplan] plan info not read, trying plan only: companyId=%s error=%v", companyID, err)
		code, planErr := companyplan.SelectCompanyPlan(companyID)
		if planErr != nil {
			return nil, planErr
		}
		info = &companyplan.CompanyPlanInfoS{CompanyID: companyID, Plan: code}
	}
	if info == nil {
		info = &companyplan.CompanyPlanInfoS{CompanyID: companyID}
	}

	now := time.Now()
	result := resolvePlan(*info, BPlanCatalog(), now)

	// Start the TRIAL clock in the database the first time.
	if info.PlanEndsAt == nil && result.EndsAt != nil && result.StartedAt != nil {
		if err := companyplan.SetCompanyPlanDates(companyID, *result.StartedAt, *result.EndsAt); err != nil {
			log.Printf("[blplan] trial dates not saved: companyId=%s error=%v", companyID, err)
		}
	}

	if count, err := plans.CountActiveCompanyUsers(companyID); err == nil {
		result.UsersCount = count
	} else {
		log.Printf("[blplan] users not counted: companyId=%s error=%v", companyID, err)
	}
	if count, err := plans.CountActiveProducts(companyID); err == nil {
		result.ProductsCount = count
	} else {
		log.Printf("[blplan] products not counted: companyId=%s error=%v", companyID, err)
	}
	fillAIUsage(companyID, &result)
	return &result, nil
}

// effectivePlan reads the plan without counting (cheaper, for checks).
func effectivePlan(companyID string) (*CompanyPlan, error) {
	info, err := companyplan.SelectCompanyPlanInfo(companyID)
	if err != nil {
		code, planErr := companyplan.SelectCompanyPlan(companyID)
		if planErr != nil {
			return nil, planErr
		}
		info = &companyplan.CompanyPlanInfoS{CompanyID: companyID, Plan: code}
	}
	if info == nil {
		info = &companyplan.CompanyPlanInfoS{CompanyID: companyID}
	}
	result := resolvePlan(*info, BPlanCatalog(), time.Now())
	return &result, nil
}

// planHasFeature tells if a plan includes an option.
func planHasFeature(plan plans.PlanS, feature Feature) bool {
	switch feature {
	case FeatureAllReports:
		return plan.AllReports
	case FeatureWebBookings:
		return plan.WebBookings
	case FeatureAIAssistant:
		return plan.AIAssistant
	default:
		return true
	}
}

// BHasFeature tells if the company's current plan includes an option.
// When the plan cannot be read the option is allowed (never block the
// business because of a reading error); the error is logged.
func BHasFeature(companyID string, feature Feature) bool {
	plan, err := effectivePlan(strings.TrimSpace(companyID))
	if err != nil {
		log.Printf("[blplan] plan not read, allowing %s: companyId=%s error=%v", feature, companyID, err)
		return true
	}
	return planHasFeature(plan.Plan, feature)
}

// BCheckFeature returns a *PlanLimitError when the plan lacks the option.
func BCheckFeature(companyID string, feature Feature) error {
	plan, err := effectivePlan(strings.TrimSpace(companyID))
	if err != nil {
		log.Printf("[blplan] plan not read, allowing %s: companyId=%s error=%v", feature, companyID, err)
		return nil
	}
	if planHasFeature(plan.Plan, feature) {
		return nil
	}
	return &PlanLimitError{Code: CodeFeature, Plan: plan.PlanCode, PlanName: plan.Plan.Name, Feature: feature}
}

// checkLimit compares a count with a limit (nil = unlimited).
func checkLimit(plan *CompanyPlan, limit *int, count int, code string) error {
	if limit == nil || count < *limit {
		return nil
	}
	return &PlanLimitError{Code: code, Plan: plan.PlanCode, PlanName: plan.Plan.Name, Limit: *limit}
}

// BCheckCanAddUser returns a *PlanLimitError when one more active user
// would pass the plan limit.
func BCheckCanAddUser(companyID string) error {
	companyID = strings.TrimSpace(companyID)
	plan, err := effectivePlan(companyID)
	if err != nil {
		log.Printf("[blplan] plan not read, allowing new user: companyId=%s error=%v", companyID, err)
		return nil
	}
	if plan.Plan.MaxUsers == nil {
		return nil
	}
	count, err := plans.CountActiveCompanyUsers(companyID)
	if err != nil {
		log.Printf("[blplan] users not counted, allowing new user: companyId=%s error=%v", companyID, err)
		return nil
	}
	return checkLimit(plan, plan.Plan.MaxUsers, count, CodeLimitUsers)
}

// BCheckCanAddProduct returns a *PlanLimitError when one more active
// product would pass the plan limit.
func BCheckCanAddProduct(companyID string) error {
	companyID = strings.TrimSpace(companyID)
	plan, err := effectivePlan(companyID)
	if err != nil {
		log.Printf("[blplan] plan not read, allowing new product: companyId=%s error=%v", companyID, err)
		return nil
	}
	if plan.Plan.MaxProducts == nil {
		return nil
	}
	count, err := plans.CountActiveProducts(companyID)
	if err != nil {
		log.Printf("[blplan] products not counted, allowing new product: companyId=%s error=%v", companyID, err)
		return nil
	}
	return checkLimit(plan, plan.Plan.MaxProducts, count, CodeLimitProducts)
}

// BCheckActivePlan returns a *PlanLimitError (PLAN_REQUIRED) when the
// company has no active plan (read-only). When the plan cannot be read the
// action is allowed (never block the business because of a reading error).
func BCheckActivePlan(companyID string) error {
	companyID = strings.TrimSpace(companyID)
	plan, err := effectivePlan(companyID)
	if err != nil {
		log.Printf("[blplan] plan not read, allowing action: companyId=%s error=%v", companyID, err)
		return nil
	}
	if plan.Active {
		return nil
	}
	return &PlanLimitError{Code: CodeNoPlan, Plan: plan.PlanCode, PlanName: plan.Plan.Name}
}

// BIsPlanActive tells if the company has an active plan (true on errors).
func BIsPlanActive(companyID string) bool {
	return BCheckActivePlan(companyID) == nil
}

// StorageLimitBytes returns the storage limit of a plan in bytes
// (nil = unlimited).
func StorageLimitBytes(plan plans.PlanS) *int64 {
	if plan.StorageGB == nil {
		return nil
	}
	limit := int64(*plan.StorageGB) * 1024 * 1024 * 1024
	return &limit
}

// CheckStorage returns a *PlanLimitError (PLAN_LIMIT_STORAGE) when adding
// extraBytes to usedBytes passes the plan storage (pure logic, tested).
func CheckStorage(plan *CompanyPlan, usedBytes, extraBytes int64) error {
	limit := StorageLimitBytes(plan.Plan)
	if limit == nil || usedBytes+extraBytes <= *limit {
		return nil
	}
	gb := 0
	if plan.Plan.StorageGB != nil {
		gb = *plan.Plan.StorageGB
	}
	return &PlanLimitError{Code: CodeLimitStorage, Plan: plan.PlanCode, PlanName: plan.Plan.Name, Limit: gb}
}

// BGetEffectivePlan returns the plan the company works with now, without
// counting users and products (for quick checks in other modules).
func BGetEffectivePlan(companyID string) (*CompanyPlan, error) {
	return effectivePlan(strings.TrimSpace(companyID))
}
