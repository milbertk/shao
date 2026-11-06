package reports

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// =====================================================================
//  Aging orders report (READ-ONLY).
//
//  "Open" orders are those whose tracking status is NOT DONE and NOT
//  CANCELLED (i.e. PLACED / IN_PROGRESS / PARTIAL_DONE). Age is the number
//  of days since created_at. Rows come back oldest-first so the most
//  overdue work is at the top. Scoped by company_id.
//
//  Reuses orderStatusToDB + normalizeOrderStatus from reports_query.go.
// =====================================================================

// Age buckets (days). Kept here so the frontend and backend agree.
const (
	agingFreshMaxDays = 7  // 0..7   -> fresh
	agingAgingMaxDays = 30 // 8..30  -> aging ; >30 -> overdue
)

type AgingOrderRow struct {
	OrderCode       string  `json:"orderCode"`
	OrderInternalID string  `json:"orderInternalId"`
	CustomerName    string  `json:"customerName"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"createdAt"`
	AgeDays         int     `json:"ageDays"`
	Items           int     `json:"items"`
	Total           float64 `json:"total"`
}

type AgingOrdersReport struct {
	TotalOpen  int             `json:"totalOpen"`
	Fresh      int             `json:"fresh"`   // 0..7 days
	Aging      int             `json:"aging"`   // 8..30 days
	Overdue    int             `json:"overdue"` // >30 days
	OldestDays int             `json:"oldestDays"`
	Rows       []AgingOrderRow `json:"rows"`
}

// AgingOrders returns open orders (oldest first) with their age in days.
//   statusKey: "", "placed", "inProgress" or "partialDone" (done/cancelled
//              are never open, so they are ignored).
//   minDays:   when > 0, only orders at least this many days old.
func AgingOrders(companyID string, statusKey string, minDays int) (*AgingOrdersReport, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	rep := &AgingOrdersReport{Rows: make([]AgingOrderRow, 0)}

	args := []interface{}{companyID}
	where := "o.company_id = $1 AND o.status NOT IN ('DONE','CANCELLED')"
	if dbStatus := orderStatusToDB(statusKey); dbStatus != "" && dbStatus != "DONE" && dbStatus != "CANCELLED" {
		args = append(args, dbStatus)
		where += fmt.Sprintf(" AND o.status = $%d", len(args))
	}
	if minDays > 0 {
		args = append(args, minDays)
		where += fmt.Sprintf(" AND (CURRENT_DATE - o.created_at::date) >= $%d", len(args))
	}

	q := `
		SELECT
			COALESCE(o.order_code, ''),
			o.order_internal_id::text,
			COALESCE(NULLIF(o.customer_name, ''), o.customer_id, ''),
			o.status,
			to_char(o.created_at, 'YYYY-MM-DD'),
			(CURRENT_DATE - o.created_at::date)::int AS age_days,
			COALESCE((SELECT COUNT(*) FROM public.tbl_order_item oi WHERE oi.order_internal_id = o.order_internal_id), 0)::int,
			COALESCE(o.total, 0)::float8
		FROM public.tbl_order o
		WHERE ` + where + `
		ORDER BY o.created_at ASC
		LIMIT 500`

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("aging orders failed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var row AgingOrderRow
		var st string
		if err := rows.Scan(&row.OrderCode, &row.OrderInternalID, &row.CustomerName,
			&st, &row.CreatedAt, &row.AgeDays, &row.Items, &row.Total); err != nil {
			return nil, fmt.Errorf("aging orders scan: %w", err)
		}
		row.Status = normalizeOrderStatus(st)
		if row.OrderCode == "" {
			row.OrderCode = row.OrderInternalID
		}
		rep.Rows = append(rep.Rows, row)

		rep.TotalOpen++
		switch {
		case row.AgeDays <= agingFreshMaxDays:
			rep.Fresh++
		case row.AgeDays <= agingAgingMaxDays:
			rep.Aging++
		default:
			rep.Overdue++
		}
		if row.AgeDays > rep.OldestDays {
			rep.OldestDays = row.AgeDays
		}
	}
	return rep, rows.Err()
}
