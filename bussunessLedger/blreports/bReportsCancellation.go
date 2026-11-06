package blreports

import (
	"errors"
	"strings"

	reports "github.com/milbertk/shao/dataBasesLedger/reportes"
)

// BCancellationRate — appointment cancellation rate by month + overall.
// Reuses parseDate from bReports.go (same package).
func BCancellationRate(companyID, fromStr, toStr string) (*reports.CancellationReport, error) {
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
	return reports.CancellationRate(companyID, from, to)
}
