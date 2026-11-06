package blplan

import (
	"testing"

	"github.com/milbertk/shao/dataBasesLedger/plans"
)

func TestAIInteractionsLimit(t *testing.T) {
	price := func(v float64) *float64 { return &v }
	n := func(v int) *int { return &v }
	cost := 0.006

	cases := []struct {
		name    string
		plan    plans.PlanS
		agreed  *float64
		enabled bool
		want    *int
	}{
		{"10 USD plan", plans.PlanS{MonthlyPriceUSD: price(10)}, nil, true, n(166)},
		{"25 USD plan", plans.PlanS{MonthlyPriceUSD: price(25)}, nil, true, n(416)},
		{"60 USD plan = 1000", plans.PlanS{MonthlyPriceUSD: price(60)}, nil, true, n(1000)},
		{"column lowers", plans.PlanS{MonthlyPriceUSD: price(60), AIInteractionsMonth: n(300)}, nil, true, n(300)},
		{"column cannot raise above 10%", plans.PlanS{MonthlyPriceUSD: price(10), AIInteractionsMonth: n(5000)}, nil, true, n(166)},
		{"agreed price wins", plans.PlanS{MonthlyPriceUSD: price(60)}, price(30), true, n(500)},
		{"free trial default", plans.PlanS{MonthlyPriceUSD: price(0)}, nil, true, n(50)},
		{"free trial column", plans.PlanS{MonthlyPriceUSD: price(0), AIInteractionsMonth: n(80)}, nil, true, n(80)},
		{"no price, unlimited", plans.PlanS{}, nil, true, nil},
		{"no price, column", plans.PlanS{AIInteractionsMonth: n(2000)}, nil, true, n(2000)},
		{"AI not enabled", plans.PlanS{MonthlyPriceUSD: price(100)}, nil, false, n(0)},
	}
	for _, c := range cases {
		got := AIInteractionsLimit(c.plan, c.agreed, c.enabled, cost)
		switch {
		case c.want == nil && got != nil:
			t.Errorf("%s: want unlimited, got %d", c.name, *got)
		case c.want != nil && (got == nil || *got != *c.want):
			t.Errorf("%s: want %d, got %v", c.name, *c.want, got)
		}
	}
}
