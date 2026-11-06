package blplan

import (
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	assistant "github.com/milbertk/shao/dataBasesLedger/assistant"
	"github.com/milbertk/shao/dataBasesLedger/configuration"
	"github.com/milbertk/shao/dataBasesLedger/plans"
)

// AI assistant interactions per plan (2026-10-25, security S14).
//
// RULE (plan validation): the AI assistant of a plan may cost AT MOST 10% of
// the plan's monthly price (the price agreed with the company when there is
// one). One interaction = one question to an assistant (internal Aleph or the
// store assistant, counted together for the company).
//
//	limit = floor(price x 10% / cost of one interaction)
//
// The cost of one interaction is AI_COST_PER_INTERACTION_USD in
// tbl_configuration (default US$0.006: about 4,000 input + 300 output tokens
// with Claude Haiku at US$1 / US$5 per million tokens, rounded up).
//
//   - tbl_plans.ai_interactions_month can LOWER the limit, never raise it
//     above 10% of the price.
//   - Free plans (price 0, TRIAL): the 10% rule gives 0, so the fixed
//     ai_interactions_month is used (default 50).
//   - Plans without a price (SPECIAL, "ask for the price"):
//     ai_interactions_month, NULL = unlimited.
//   - Plans without the AI assistant: 0.

const (
	// AIBudgetShare is the maximum part of the plan price the AI may cost.
	AIBudgetShare = 0.10
	// AIBudgetPercent is AIBudgetShare as a whole percent (for the screens).
	AIBudgetPercent = 10

	defaultAICostPerInteractionUSD = 0.006
	defaultFreePlanAIInteractions  = 50
)

// AICostPerInteractionUSD reads the configured cost of one interaction.
func AICostPerInteractionUSD() float64 {
	value, err := configuration.GetParameter("AI_COST_PER_INTERACTION_USD")
	if err == nil && strings.TrimSpace(value) != "" {
		if number, convErr := strconv.ParseFloat(strings.TrimSpace(value), 64); convErr == nil && number > 0 {
			return number
		}
	}
	return defaultAICostPerInteractionUSD
}

// AIInteractionsLimit returns the monthly interactions of a plan (nil =
// unlimited). agreedPrice is the price negotiated with the company (nil = the
// plan price); enabled tells if the company may use the assistant at all.
// Pure logic (tested).
func AIInteractionsLimit(plan plans.PlanS, agreedPrice *float64, enabled bool, costPerInteraction float64) *int {
	zero := 0
	if !enabled {
		return &zero
	}
	price := plan.MonthlyPriceUSD
	if agreedPrice != nil {
		price = agreedPrice
	}
	if price == nil {
		return plan.AIInteractionsMonth // no price: the configured value, nil = unlimited
	}
	if *price <= 0 {
		if plan.AIInteractionsMonth != nil {
			return plan.AIInteractionsMonth
		}
		free := defaultFreePlanAIInteractions
		return &free
	}
	if costPerInteraction <= 0 {
		costPerInteraction = defaultAICostPerInteractionUSD
	}
	maxByPrice := int(math.Floor(*price*AIBudgetShare/costPerInteraction + 1e-9))
	if plan.AIInteractionsMonth != nil && *plan.AIInteractionsMonth < maxByPrice {
		lower := *plan.AIInteractionsMonth
		if lower < 0 {
			lower = 0
		}
		return &lower
	}
	return &maxByPrice
}

// MonthStart is the first instant of the current month (server time).
func MonthStart(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

// fillAIUsage sets the AI limit and this month's questions on a company plan.
func fillAIUsage(companyID string, result *CompanyPlan) {
	result.AIBudgetPercent = AIBudgetPercent
	result.AIInteractionsLimit = AIInteractionsLimit(result.Plan, result.AgreedPriceUSD, result.Plan.AIAssistant, AICostPerInteractionUSD())
	if used, err := assistant.CountCompanyQuestionsSince(companyID, MonthStart(time.Now())); err == nil {
		result.AIInteractionsUsed = used
	} else {
		log.Printf("[blplan] AI questions not counted: companyId=%s error=%v", companyID, err)
	}
}

// withAIInteractions returns the plans with ai_interactions_month replaced by
// the real monthly limit of each plan, for the plans window.
func withAIInteractions(list []plans.PlanS) []plans.PlanS {
	cost := AICostPerInteractionUSD()
	result := make([]plans.PlanS, len(list))
	for i, p := range list {
		p.AIInteractionsMonth = AIInteractionsLimit(p, nil, p.AIAssistant, cost)
		result[i] = p
	}
	return result
}
