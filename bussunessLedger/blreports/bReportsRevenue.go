package blreports

import (
	"errors"
	"strings"

	reports "github.com/milbertk/shao/dataBasesLedger/reportes"
)

// BRevenueOverTime — order revenue by day/week/month over a date range.
// Reuses parseDate from bReports.go (same package).
func BRevenueOverTime(companyID, fromStr, toStr, granularity string) (*reports.RevenueReport, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	from, err := parseDate(fromStr)
	if err != nil {
		return nil, err
	}
	to, err := parseDate(toStr)
	if err != nil {
		return nil, err
	}
	return reports.RevenueOverTime(companyID, from, to, granularity)
}
