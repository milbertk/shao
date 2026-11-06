package reports

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// =====================================================================
//  Cancellation rate report (appointments, READ-ONLY).
//
//  Rate = cancelled / total, where "cancelled" counts CANCELLED + REJECTED.
//  (The system has no separate no-show status, so only cancellations are
//  measured.) Broken down by month, plus an overall summary. Scoped by
//  company_id.
// =====================================================================

type CancellationMonth struct {
	Period    string  `json:"period"` // YYYY-MM
	Total     int     `json:"total"`
	Cancelled int     `json:"cancelled"`
	RatePct   float64 `json:"ratePct"`
}

type CancellationReport struct {
	Total     int                 `json:"total"`
	Cancelled int                 `json:"cancelled"`
	RatePct   float64             `json:"ratePct"`
	Months    []CancellationMonth `json:"months"`
}

func round1(x float64) float64 {
	return float64(int(x*10+0.5)) / 10
}

// CancellationRate returns cancellation counts/percentage by month + overall.
func CancellationRate(companyID string, from, to *time.Time) (*CancellationReport, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	rep := &CancellationReport{Months: make([]CancellationMonth, 0)}

	args := []interface{}{companyID}
	where := "company_id = $1"
	if from != nil {
		args = append(args, *from)
		where += fmt.Sprintf(" AND appointment_date >= $%d", len(args))
	}
	if to != nil {
		args = append(args, *to)
		where += fmt.Sprintf(" AND appointment_date <= $%d", len(args))
	}

	q := `
		SELECT
			to_char(date_trunc('month', appointment_date), 'YYYY-MM') AS period,
			COUNT(*)::int AS total,
			COUNT(*) FILTER (WHERE status IN ('CANCELLED','REJECTED'))::int AS cancelled
		FROM public.tbl_appointment
		WHERE ` + where + `
		GROUP BY 1
		ORDER BY 1`

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("cancellation rate failed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var m CancellationMonth
		if err := rows.Scan(&m.Period, &m.Total, &m.Cancelled); err != nil {
			return nil, fmt.Errorf("cancellation rate scan: %w", err)
		}
		if m.Total > 0 {
			m.RatePct = round1(float64(m.Cancelled) / float64(m.Total) * 100)
		}
		rep.Months = append(rep.Months, m)
		rep.Total += m.Total
		rep.Cancelled += m.Cancelled
	}
	if rep.Total > 0 {
		rep.RatePct = round1(float64(rep.Cancelled) / float64(rep.Total) * 100)
	}
	return rep, rows.Err()
}
