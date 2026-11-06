package blreports

import (
	"errors"
	"strings"
	"time"

	reports "github.com/milbertk/shao/dataBasesLedger/reportes"
)

// =====================================================================
//  Reports business layer. Validates inputs, parses the date range, and
//  delegates to the read-only data layer. No writes.
// =====================================================================

const dateLayout = "2006-01-02"

// parseDate turns "YYYY-MM-DD" into *time.Time; empty -> nil (no bound).
func parseDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	d, err := time.Parse(dateLayout, s)
	if err != nil {
		return nil, errors.New("invalid date: use YYYY-MM-DD")
	}
	return &d, nil
}

// BOrdersByStatus — orders report.
func BOrdersByStatus(companyID, fromStr, toStr, statusKey string) (*reports.OrdersReport, error) {
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
	if statusKey == "all" {
		statusKey = ""
	}
	return reports.OrdersByStatus(companyID, from, to, statusKey)
}

// BAppointmentsSummary — appointments report.
func BAppointmentsSummary(companyID, fromStr, toStr, bucket string) (*reports.AppointmentsReport, error) {
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
	if bucket == "all" {
		bucket = ""
	}
	return reports.AppointmentsSummary(companyID, from, to, bucket)
}

// BTopCustomers — top customers report.
func BTopCustomers(companyID string, limit int) (*reports.TopCustomersReport, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	return reports.TopCustomers(companyID, limit)
}

// BBestSellers — best-seller products report.
func BBestSellers(companyID, typeFilter string, limit int) (*reports.BestSellersReport, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if typeFilter == "all" {
		typeFilter = ""
	}
	return reports.BestSellers(companyID, typeFilter, limit)
}
