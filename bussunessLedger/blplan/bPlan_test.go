package blplan

import (
	"errors"
	"testing"
	"time"

	"github.com/milbertk/shao/dataBasesLedger/companyplan"
)

func TestResolvePlan(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	past := now.AddDate(0, 0, -1)
	future := now.Add(36 * time.Hour)
	started := now.AddDate(0, 0, -10)

	cases := []struct {
		name     string
		info     companyplan.CompanyPlanInfoS
		want     string
		active   bool
		expired  bool
		daysLeft int
	}{
		{"no plan row is NONE (read-only)", companyplan.CompanyPlanInfoS{}, PlanNone, false, false, -1},
		{"unknown plan is NONE", companyplan.CompanyPlanInfoS{Plan: "GOLD"}, PlanNone, false, false, -1},
		{"old FREE works as ENTREPRENEUR", companyplan.CompanyPlanInfoS{Plan: PlanFree}, PlanEntrepreneur, true, false, -1},
		{"entrepreneur", companyplan.CompanyPlanInfoS{Plan: PlanEntrepreneur}, PlanEntrepreneur, true, false, -1},
		{"pro", companyplan.CompanyPlanInfoS{Plan: PlanPro}, PlanPro, true, false, -1},
		{"special", companyplan.CompanyPlanInfoS{Plan: PlanSpecial}, PlanSpecial, true, false, -1},
		{"trial running", companyplan.CompanyPlanInfoS{Plan: PlanTrial, PlanEndsAt: &future}, PlanTrial, true, false, 2},
		{"trial ended is NONE", companyplan.CompanyPlanInfoS{Plan: PlanTrial, PlanEndsAt: &past}, PlanNone, false, true, 0},
		{"trial without end uses start + 30", companyplan.CompanyPlanInfoS{Plan: PlanTrial, PlanStartedAt: &started}, PlanTrial, true, false, 20},
		{"paid plan with past end is NONE", companyplan.CompanyPlanInfoS{Plan: PlanBasic, PlanEndsAt: &past}, PlanNone, false, true, 0},
	}
	for _, c := range cases {
		got := resolvePlan(c.info, builtInPlans, now)
		if got.PlanCode != c.want || got.Expired != c.expired || got.Active != c.active {
			t.Errorf("%s: got %s active=%v expired=%v, want %s active=%v expired=%v",
				c.name, got.PlanCode, got.Active, got.Expired, c.want, c.active, c.expired)
		}
		if c.daysLeft >= 0 && (got.DaysLeft == nil || *got.DaysLeft != c.daysLeft) {
			t.Errorf("%s: daysLeft=%v, want %d", c.name, got.DaysLeft, c.daysLeft)
		}
	}
}

func TestLimitsAndFeatures(t *testing.T) {
	ent := resolvePlan(companyplan.CompanyPlanInfoS{Plan: PlanEntrepreneur}, builtInPlans, time.Now())
	if *ent.Plan.MonthlyPriceUSD != 10 {
		t.Fatal("ENTREPRENEUR costs US$10")
	}
	if err := checkLimit(&ent, ent.Plan.MaxUsers, 0, CodeLimitUsers); err != nil {
		t.Fatalf("first user must be allowed: %v", err)
	}
	err := checkLimit(&ent, ent.Plan.MaxUsers, 1, CodeLimitUsers)
	if limitErr, ok := AsPlanLimitError(err); !ok || limitErr.Limit != 1 || limitErr.Code != CodeLimitUsers {
		t.Fatalf("second user must be blocked, got %v", err)
	}
	if err := checkLimit(&ent, ent.Plan.MaxProducts, 99, CodeLimitProducts); err != nil {
		t.Fatal("100th product must be allowed")
	}
	if err := checkLimit(&ent, ent.Plan.MaxProducts, 100, CodeLimitProducts); err == nil {
		t.Fatal("101st product must be blocked")
	}
	basic := resolvePlan(companyplan.CompanyPlanInfoS{Plan: PlanBasic}, builtInPlans, time.Now())
	if err := checkLimit(&basic, basic.Plan.MaxProducts, 5000, CodeLimitProducts); err != nil {
		t.Fatal("BASIC has unlimited products")
	}
	// Every plan has every option; only AI differs.
	for _, p := range []CompanyPlan{ent, basic} {
		if !planHasFeature(p.Plan, FeatureAllReports) || !planHasFeature(p.Plan, FeatureWebBookings) {
			t.Fatalf("%s must have all reports and store bookings", p.PlanCode)
		}
		if planHasFeature(p.Plan, FeatureAIAssistant) {
			t.Fatalf("%s has no AI assistant", p.PlanCode)
		}
	}
	pro := resolvePlan(companyplan.CompanyPlanInfoS{Plan: PlanPro}, builtInPlans, time.Now())
	if !planHasFeature(pro.Plan, FeatureAIAssistant) || *pro.Plan.MaxUsers != 20 {
		t.Fatal("PRO: AI and 20 users")
	}
	special := resolvePlan(companyplan.CompanyPlanInfoS{Plan: PlanSpecial}, builtInPlans, time.Now())
	if special.Plan.MaxUsers != nil || special.Plan.MaxProducts != nil || StorageLimitBytes(special.Plan) != nil || special.Plan.Requestable {
		t.Fatal("SPECIAL: no limits and not requestable")
	}
}

func TestStorage(t *testing.T) {
	const gb = int64(1024 * 1024 * 1024)
	ent := resolvePlan(companyplan.CompanyPlanInfoS{Plan: PlanEntrepreneur}, builtInPlans, time.Now())
	if err := CheckStorage(&ent, 4*gb, gb); err != nil {
		t.Fatalf("exactly 5 GB must fit: %v", err)
	}
	err := CheckStorage(&ent, 5*gb, 1)
	if limitErr, ok := AsPlanLimitError(err); !ok || limitErr.Code != CodeLimitStorage || limitErr.Limit != 5 {
		t.Fatalf("over 5 GB must be blocked, got %v", err)
	}
	special := resolvePlan(companyplan.CompanyPlanInfoS{Plan: PlanSpecial}, builtInPlans, time.Now())
	if err := CheckStorage(&special, 900*gb, 900*gb); err != nil {
		t.Fatal("SPECIAL has unlimited storage")
	}
	for code, want := range map[string]int{PlanTrial: 5, PlanEntrepreneur: 5, PlanBasic: 10, PlanPro: 10, PlanEnterprise: 25} {
		p := resolvePlan(companyplan.CompanyPlanInfoS{Plan: code}, builtInPlans, time.Now())
		if p.Plan.StorageGB == nil || *p.Plan.StorageGB != want {
			t.Errorf("%s storage: got %v, want %d GB", code, p.Plan.StorageGB, want)
		}
	}
}

func TestVisiblePlansHideSpecial(t *testing.T) {
	visible := func(current string) map[string]bool {
		out := map[string]bool{}
		for _, p := range builtInPlans {
			if p.Requestable || p.PlanCode == current {
				out[p.PlanCode] = true
			}
		}
		return out
	}
	if visible(PlanPro)[PlanSpecial] {
		t.Fatal("SPECIAL must be hidden from other companies")
	}
	if !visible(PlanSpecial)[PlanSpecial] {
		t.Fatal("a SPECIAL company sees its plan")
	}
}

func TestNormalizePlanRequest(t *testing.T) {
	catalog := []PlanCatalogEntry{{Code: PlanSpecial}, {Code: PlanPro, Name: "Pro", Requestable: true}}
	ok := PlanRequestInput{RequestedPlan: " pro ", ContactName: "Ana", ContactEmail: "ana@example.com", Language: "es-CR"}
	got, err := normalizePlanRequest(ok, catalog)
	if err != nil || got.RequestedPlan != PlanPro || got.Language != "es" {
		t.Fatalf("valid request rejected: %+v %v", got, err)
	}
	bad := []PlanRequestInput{
		{RequestedPlan: "SPECIAL", ContactName: "Ana", ContactEmail: "ana@example.com"},
		{RequestedPlan: "PRO", ContactEmail: "ana@example.com"},
		{RequestedPlan: "PRO", ContactName: "Ana", ContactEmail: "not-an-email"},
	}
	for _, input := range bad {
		if _, err := normalizePlanRequest(input, catalog); !errors.Is(err, ErrPlanRequestInvalid) {
			t.Errorf("expected invalid for %+v, got %v", input, err)
		}
	}
}

func TestAgreedPriceAndSupport(t *testing.T) {
	now := time.Now()
	price := 25.0
	past := now.AddDate(0, 0, -1)
	pro := resolvePlan(companyplan.CompanyPlanInfoS{Plan: PlanPro, AgreedPriceUSD: &price}, builtInPlans, now)
	if pro.AgreedPriceUSD == nil || *pro.AgreedPriceUSD != 25 || pro.Plan.SupportHoursMonth != 1 || *pro.Plan.StorageGB != 10 {
		t.Fatalf("PRO: agreed 25, 1 support hour, 10 GB; got %+v", pro)
	}
	ended := resolvePlan(companyplan.CompanyPlanInfoS{Plan: PlanPro, AgreedPriceUSD: &price, PlanEndsAt: &past}, builtInPlans, now)
	if ended.AgreedPriceUSD != nil || ended.PlanCode != PlanNone || ended.Active {
		t.Fatal("an ended plan has no agreed price and is read-only (NONE)")
	}
	ent := resolvePlan(companyplan.CompanyPlanInfoS{Plan: PlanEnterprise}, builtInPlans, now)
	if ent.Plan.SupportHoursMonth != 2 || *ent.Plan.StorageGB != 25 {
		t.Fatal("ENTERPRISE: 2 support hours, 25 GB")
	}
}
