package blassistant

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/milbertk/shao/bussunessLedger/blplan"
	assistant "github.com/milbertk/shao/dataBasesLedger/assistant"
	"github.com/milbertk/shao/dataBasesLedger/companyplan"
	"github.com/milbertk/shao/dataBasesLedger/configuration"
)

// Access rules of the AI assistants (2026-10-22, interactions 2026-10-25):
//   - Plan feature: only companies whose plan has ai_assistant in tbl_plans
//     (PRO, ENTERPRISE, TRIAL, SPECIAL; without an active plan: no) can use them,
//     unless tbl_company_ai_settings forces it. AI_PLANS is the fallback
//     when the plans cannot be read.
//   - Monthly cost cap per company (default AI_DEFAULT_MONTHLY_CAP_USD = 5):
//     tokens used this month x AI_PRICE_*_PER_MTOK. Checked BEFORE calling
//     Claude, so a blocked question costs nothing.
//   - Interactions per month from the plan (blplan.AIInteractionsLimit): at
//     most 10% of the plan price. Checked before calling Claude too.

var (
	// ErrAssistantNotInPlan: the company's plan does not include the assistant.
	ErrAssistantNotInPlan = errors.New("AI_NOT_IN_PLAN: the AI assistant is not included in this company's plan")
	// ErrMonthlyCapReached: the company used its monthly AI budget.
	ErrMonthlyCapReached = errors.New("AI_MONTHLY_LIMIT: the company reached this month's AI usage limit")
	// ErrMonthlyInteractionsReached: the company used this month's AI
	// interactions of its plan (max 10% of the plan price, 2026-10-25).
	// It wraps ErrMonthlyCapReached, so every handler answers 429 for it.
	ErrMonthlyInteractionsReached = fmt.Errorf("AI_MONTHLY_INTERACTIONS: the company used this month's AI interactions of its plan: %w", ErrMonthlyCapReached)
)

const (
	defaultAIPlans          = "PRO,ENTERPRISE"
	defaultMonthlyCapUSD    = 5.0
	defaultPriceInputPerMT  = 1.0
	defaultPriceOutputPerMT = 5.0
)

// AssistantUsage is the monthly AI usage of a company (for logs/tracking).
type AssistantUsage struct {
	Plan         string  `json:"plan"`
	Enabled      bool    `json:"enabled"`
	MonthCostUSD float64 `json:"monthCostUsd"`
	CapUSD       float64 `json:"capUsd"`
	// Plan interactions this month (nil limit = unlimited), 2026-10-25.
	InteractionsLimit *int `json:"interactionsLimit,omitempty"`
	InteractionsUsed  int  `json:"interactionsUsed"`
}

func configNumber(parameter string, fallback float64) float64 {
	value, err := configuration.GetParameter(parameter)
	if err != nil || strings.TrimSpace(value) == "" {
		return fallback
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || number < 0 {
		return fallback
	}
	return number
}

func allowedAIPlans() map[string]bool {
	value, err := configuration.GetParameter("AI_PLANS")
	if err != nil || strings.TrimSpace(value) == "" {
		value = defaultAIPlans
	}
	plans := map[string]bool{}
	for _, plan := range strings.Split(value, ",") {
		if plan = strings.ToUpper(strings.TrimSpace(plan)); plan != "" {
			plans[plan] = true
		}
	}
	return plans
}

// assistantEnabled decides the plan rule (with the per-company override).
func assistantEnabled(companyID string, settings *assistant.CompanyAISettingsS) (bool, string) {
	plan, err := companyplan.SelectCompanyPlan(companyID)
	if err != nil {
		log.Printf("[assistantAccess] plan not read: companyId=%s error=%v", companyID, err)
	}
	if settings != nil && settings.EnabledOverride != nil {
		return *settings.EnabledOverride, plan
	}
	// Plan rules from tbl_plans (2026-10-22).
	if current, err := blplan.BGetEffectivePlan(companyID); err == nil {
		return current.Plan.AIAssistant, current.PlanCode
	} else {
		log.Printf("[assistantAccess] plan rules not read, using AI_PLANS: companyId=%s error=%v", companyID, err)
	}
	return allowedAIPlans()[plan], plan
}

// monthStart is the first instant of the current month (server time).
func monthStart(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

// tokensCostUSD converts tokens to US$ with the configured prices.
func tokensCostUSD(input, output int64, priceInput, priceOutput float64) float64 {
	cost := float64(input)/1_000_000*priceInput + float64(output)/1_000_000*priceOutput
	return math.Round(cost*10000) / 10000
}

// BCheckAssistantAccess returns nil when the company may ask the assistant
// now, ErrAssistantNotInPlan or ErrMonthlyCapReached otherwise.
func BCheckAssistantAccess(companyID string) (*AssistantUsage, error) {
	companyID = strings.TrimSpace(companyID)
	settings, err := assistant.SelectCompanyAISettings(companyID)
	if err != nil {
		log.Printf("[assistantAccess] settings not read: companyId=%s error=%v", companyID, err)
		settings = nil
	}

	enabled, plan := assistantEnabled(companyID, settings)
	usage := &AssistantUsage{Plan: plan, Enabled: enabled}
	if !enabled {
		return usage, ErrAssistantNotInPlan
	}

	// Plan interactions this month: at most 10% of the plan price
	// (2026-10-25, security S14). Counted in the database, so the limit
	// survives restarts and works with several servers.
	if current, err := blplan.BGetEffectivePlan(companyID); err == nil {
		usage.InteractionsLimit = blplan.AIInteractionsLimit(current.Plan, current.AgreedPriceUSD, true, blplan.AICostPerInteractionUSD())
		if usage.InteractionsLimit != nil {
			used, countErr := assistant.CountCompanyQuestionsSince(companyID, monthStart(time.Now()))
			if countErr != nil {
				log.Printf("[assistantAccess] questions not counted: companyId=%s error=%v", companyID, countErr)
			} else {
				usage.InteractionsUsed = used
				if used >= *usage.InteractionsLimit {
					return usage, ErrMonthlyInteractionsReached
				}
			}
		}
	} else {
		log.Printf("[assistantAccess] plan not read for interactions: companyId=%s error=%v", companyID, err)
	}

	usage.CapUSD = configNumber("AI_DEFAULT_MONTHLY_CAP_USD", defaultMonthlyCapUSD)
	if settings != nil && settings.MonthlyCapUSD != nil {
		usage.CapUSD = *settings.MonthlyCapUSD
	}

	input, output, err := assistant.SumCompanyTokensSince(companyID, monthStart(time.Now()))
	if err != nil {
		// Do not block the assistant because of a reporting error.
		log.Printf("[assistantAccess] usage not read: companyId=%s error=%v", companyID, err)
		return usage, nil
	}
	usage.MonthCostUSD = tokensCostUSD(input, output,
		configNumber("AI_PRICE_INPUT_PER_MTOK", defaultPriceInputPerMT),
		configNumber("AI_PRICE_OUTPUT_PER_MTOK", defaultPriceOutputPerMT))

	if usage.MonthCostUSD >= usage.CapUSD {
		return usage, ErrMonthlyCapReached
	}
	return usage, nil
}

// BAssistantAvailable tells the public store whether to show the assistant
// button (plan rule only; the monthly cap is answered inside the chat).
func BAssistantAvailable(companyID string) bool {
	settings, err := assistant.SelectCompanyAISettings(strings.TrimSpace(companyID))
	if err != nil {
		settings = nil
	}
	enabled, _ := assistantEnabled(strings.TrimSpace(companyID), settings)
	return enabled
}
