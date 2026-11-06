package reports

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// =====================================================================
//  Revenue over time report (orders, READ-ONLY).
//
//  Sums tbl_order.total (excluding CANCELLED) grouped by day / week / month
//  over an optional date range. Returns the series plus a summary (total
//  revenue, order count, average order value, best period). Scoped by
//  company_id.
// =====================================================================

type RevenuePoint struct {
	Period  string  `json:"period"` // bucket start, YYYY-MM-DD
	Orders  int     `json:"orders"`
	Revenue float64 `json:"revenue"`
}

type RevenueReport struct {
	Granularity   string         `json:"granularity"`
	TotalRevenue  float64        `json:"totalRevenue"`
	TotalOrders   int            `json:"totalOrders"`
	AvgOrderValue float64        `json:"avgOrderValue"`
	BestPeriod    string         `json:"bestPeriod"`
	BestRevenue   float64        `json:"bestRevenue"`
	Points        []RevenuePoint `json:"points"`
}

// validGranularity whitelists the date_trunc unit (prevents SQL injection via
// the unit and guarantees a valid value).
func validGranularity(g string) string {
	switch strings.ToLower(strings.TrimSpace(g)) {
	case "day":
		return "day"
	case "week":
		return "week"
	default:
		return "month"
	}
}

// RevenueOverTime returns the revenue series + summary.
func RevenueOverTime(companyID string, from, to *time.Time, granularity string) (*RevenueReport, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}
	unit := validGranularity(granularity)
	rep := &RevenueReport{Granularity: unit, Points: make([]RevenuePoint, 0)}

	// $2 is the date_trunc unit (safe: whitelisted above).
	args := []interface{}{companyID, unit}
	where := "company_id = $1 AND status <> 'CANCELLED'"
	if from != nil {
		args = append(args, *from)
		where += fmt.Sprintf(" AND created_at::date >= $%d", len(args))
	}
	if to != nil {
		args = append(args, *to)
		where += fmt.Sprintf(" AND created_at::date <= $%d", len(args))
	}

	q := `
		SELECT
			to_char(date_trunc($2, created_at), 'YYYY-MM-DD') AS period,
			COUNT(*)::int AS orders,
			COALESCE(SUM(total), 0)::float8 AS revenue
		FROM public.tbl_order
		WHERE ` + where + `
		GROUP BY date_trunc($2, created_at)
		ORDER BY date_trunc($2, created_at)`

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("revenue over time failed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p RevenuePoint
		if err := rows.Scan(&p.Period, &p.Orders, &p.Revenue); err != nil {
			return nil, fmt.Errorf("revenue over time scan: %w", err)
		}
		rep.Points = append(rep.Points, p)
		rep.TotalRevenue += p.Revenue
		rep.TotalOrders += p.Orders
		if p.Revenue > rep.BestRevenue {
			rep.BestRevenue = p.Revenue
			rep.BestPeriod = p.Period
		}
	}
	if rep.TotalOrders > 0 {
		rep.AvgOrderValue = rep.TotalRevenue / float64(rep.TotalOrders)
	}
	return rep, rows.Err()
}
