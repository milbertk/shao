package blorder

import (
	"errors"
	"strings"
	"time"

	order "github.com/milbertk/shao/dataBasesLedger/orders"
)

const insightsDateLayout = "2006-01-02"

// OrdersDueResult is the list of open orders due on or before UntilDate.
type OrdersDueResult struct {
	Today        string              `json:"today"`
	UntilDate    string              `json:"untilDate"`
	TotalMatches int                 `json:"totalMatches"`
	Overdue      int                 `json:"overdue"` // in the returned rows
	Orders       []order.OrderDueRow `json:"orders"`
}

// BOrdersDueSoon returns the open orders (not DONE / CANCELLED) whose
// estimated delivery date is within daysAhead days from today, including the
// overdue ones. companyID is authoritative.
func BOrdersDueSoon(companyID string, today time.Time, daysAhead int, limit int) (*OrdersDueResult, error) {
	until := today.AddDate(0, 0, daysAhead)
	return BOrdersDueBy(companyID, today, until.Format(insightsDateLayout), limit)
}

// BOrdersDueBy returns the open orders due on or before untilDate
// (YYYY-MM-DD), overdue ones included.
func BOrdersDueBy(companyID string, today time.Time, untilDate string, limit int) (*OrdersDueResult, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	untilDate = strings.TrimSpace(untilDate)
	if _, err := time.Parse(insightsDateLayout, untilDate); err != nil {
		return nil, errors.New("invalid date: use YYYY-MM-DD")
	}

	todayStr := today.Format(insightsDateLayout)
	rows, total, err := order.SelectOpenOrdersDueBy(companyID, todayStr, untilDate, limit)
	if err != nil {
		return nil, err
	}

	overdue := 0
	for _, r := range rows {
		if r.DaysLeft < 0 {
			overdue++
		}
	}

	return &OrdersDueResult{
		Today:        todayStr,
		UntilDate:    untilDate,
		TotalMatches: total,
		Overdue:      overdue,
		Orders:       rows,
	}, nil
}

// BOrdersRanked returns the company's orders sorted by highest_total, oldest
// or newest, optionally only open ones and within a creation date range.
func BOrdersRanked(companyID string, filters order.OrderRankingFilters) ([]order.OrderS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	filters.CompanyID = companyID

	for _, value := range []string{filters.DateFrom, filters.DateTo} {
		if v := strings.TrimSpace(value); v != "" {
			if _, err := time.Parse(insightsDateLayout, v); err != nil {
				return nil, errors.New("invalid date: use YYYY-MM-DD")
			}
		}
	}

	return order.SelectOrdersRanked(filters)
}
