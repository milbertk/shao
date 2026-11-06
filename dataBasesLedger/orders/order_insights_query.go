package order

import (
	"fmt"
	"strconv"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// =====================================================================
//  Order insights (READ-ONLY): orders close to their due date and order
//  rankings. Used by the AI assistant. Scoped by company_id.
//  "Open" orders are the ones that are not DONE and not CANCELLED.
// =====================================================================

const openOrderCondition = `UPPER(status) NOT IN ('DONE', 'CANCELLED', 'CANCELED')`

// OrderDueRow is one open order with its due date (estimated delivery date).
type OrderDueRow struct {
	OrderInternalID       string  `json:"orderInternalId"`
	OrderCode             string  `json:"orderCode"`
	CustomerName          string  `json:"customerName"`
	Status                string  `json:"status"`
	EstimatedDeliveryDate string  `json:"estimatedDeliveryDate"` // YYYY-MM-DD
	DaysLeft              int     `json:"daysLeft"`              // negative = overdue
	Total                 float64 `json:"total"`
}

// SelectOpenOrdersDueBy returns the open orders whose estimated delivery date
// is on or before untilDate (overdue ones included), earliest first.
// today is the caller's local date, used to compute DaysLeft.
// Also returns how many orders match in total (before the limit).
func SelectOpenOrdersDueBy(companyID, today, untilDate string, limit int) ([]OrderDueRow, int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, 0, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, 0, fmt.Errorf("companyId is required")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `
		SELECT
			order_internal_id,
			COALESCE(order_code, ''),
			COALESCE(customer_name, ''),
			status,
			to_char(estimated_delivery_date, 'YYYY-MM-DD'),
			(estimated_delivery_date - $2::date),
			total::float8,
			COUNT(*) OVER ()
		FROM public.tbl_order
		WHERE company_id = $1
		  AND estimated_delivery_date IS NOT NULL
		  AND estimated_delivery_date <= $3::date
		  AND ` + openOrderCondition + `
		ORDER BY estimated_delivery_date ASC, created_at ASC
		LIMIT $4
	`

	rows, err := db.Query(query, companyID, strings.TrimSpace(today), strings.TrimSpace(untilDate), limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to select orders due: %w", err)
	}
	defer rows.Close()

	list := make([]OrderDueRow, 0)
	totalMatches := 0
	for rows.Next() {
		var r OrderDueRow
		if err := rows.Scan(
			&r.OrderInternalID,
			&r.OrderCode,
			&r.CustomerName,
			&r.Status,
			&r.EstimatedDeliveryDate,
			&r.DaysLeft,
			&r.Total,
			&totalMatches,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan order due: %w", err)
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed while reading orders due: %w", err)
	}

	return list, totalMatches, nil
}

// Order rankings (whitelisted ORDER BY clauses, never built from input).
var orderRankingSQL = map[string]string{
	"highest_total": "total DESC, created_at DESC",
	"oldest":        "created_at ASC",
	"newest":        "created_at DESC",
}

// OrderRankingFilters — CompanyID and SortKey are required.
type OrderRankingFilters struct {
	CompanyID string
	SortKey   string // highest_total | oldest | newest
	OpenOnly  bool   // only orders that are not DONE / CANCELLED
	DateFrom  string // created_at >= (YYYY-MM-DD), optional
	DateTo    string // created_at <  DateTo + 1 day, optional
	Limit     int
}

// SelectOrdersRanked returns the orders of ONE company sorted by the ranking.
func SelectOrdersRanked(filters OrderRankingFilters) ([]OrderS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(filters.CompanyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}
	orderBy, ok := orderRankingSQL[strings.TrimSpace(filters.SortKey)]
	if !ok {
		return nil, fmt.Errorf("invalid sort: use highest_total, oldest or newest")
	}

	conditions := []string{"company_id = $1"}
	args := []interface{}{companyID}
	position := 2

	if filters.OpenOnly {
		conditions = append(conditions, openOrderCondition)
	}
	if v := strings.TrimSpace(filters.DateFrom); v != "" {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d::date", position))
		args = append(args, v)
		position++
	}
	if v := strings.TrimSpace(filters.DateTo); v != "" {
		conditions = append(conditions, fmt.Sprintf("created_at < ($%d::date + INTERVAL '1 day')", position))
		args = append(args, v)
		position++
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	query := `
		SELECT ` + orderColumns + `
		FROM public.tbl_order
		WHERE ` + strings.Join(conditions, "\n\t\t  AND ") + `
		ORDER BY ` + orderBy + `
		LIMIT $` + strconv.Itoa(position)

	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select ranked orders: %w", err)
	}
	defer rows.Close()

	list := make([]OrderS, 0)
	for rows.Next() {
		var o OrderS
		if err := scanOrder(rows, &o); err != nil {
			return nil, fmt.Errorf("failed to scan ranked order: %w", err)
		}
		list = append(list, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading ranked orders: %w", err)
	}

	return list, nil
}
