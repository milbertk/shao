package reports

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// =====================================================================
//  Reports data layer (READ-ONLY aggregations).
//
//  All money columns are cast to ::float8 in SQL so lib/pq returns real
//  float64 values (numeric would otherwise scan as []byte). Everything is
//  scoped by company_id. No tables are written here.
// =====================================================================

// ---- status normalization -------------------------------------------------

// normalizeOrderStatus maps the DB order status to the frontend key.
func normalizeOrderStatus(db string) string {
	switch strings.ToUpper(strings.TrimSpace(db)) {
	case "PLACED":
		return "placed"
	case "IN_PROGRESS":
		return "inProgress"
	case "PARTIAL_DONE":
		return "partialDone"
	case "DONE":
		return "done"
	case "CANCELLED", "CANCELED":
		return "cancelled"
	default:
		return "placed"
	}
}

// orderStatusToDB maps a frontend order key back to the DB value (for filters).
func orderStatusToDB(key string) string {
	switch strings.TrimSpace(key) {
	case "placed":
		return "PLACED"
	case "inProgress":
		return "IN_PROGRESS"
	case "partialDone":
		return "PARTIAL_DONE"
	case "done":
		return "DONE"
	case "cancelled":
		return "CANCELLED"
	default:
		return ""
	}
}

// normalizeApptStatus folds the DB appointment status into 3 report buckets.
func normalizeApptStatus(db string) string {
	switch strings.ToUpper(strings.TrimSpace(db)) {
	case "DONE":
		return "finished"
	case "CANCELLED", "CANCELED", "REJECTED":
		return "cancelled"
	default: // PENDING, SCHEDULED, CONFIRMED
		return "scheduled"
	}
}

// apptBucketSQL returns the SQL status set for a report bucket (whitelisted).
func apptBucketSQL(bucket string) string {
	switch strings.TrimSpace(bucket) {
	case "scheduled":
		return "('PENDING','SCHEDULED','CONFIRMED')"
	case "finished":
		return "('DONE')"
	case "cancelled":
		return "('CANCELLED','REJECTED')"
	default:
		return ""
	}
}

// ---- Orders by status -----------------------------------------------------

type OrderRow struct {
	OrderCode       string  `json:"orderCode"`
	OrderInternalID string  `json:"orderInternalId"`
	CustomerName    string  `json:"customerName"`
	Status          string  `json:"status"`
	Items           int     `json:"items"`
	CreatedAt       string  `json:"createdAt"`
	Total           float64 `json:"total"`
}

type OrdersReport struct {
	Counts  map[string]int `json:"counts"`
	Total   int            `json:"total"`
	Revenue float64        `json:"revenue"`
	Rows    []OrderRow     `json:"rows"`
}

// OrdersByStatus returns counts/revenue over the period and the order rows.
func OrdersByStatus(companyID string, from, to *time.Time, statusKey string) (*OrdersReport, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	rep := &OrdersReport{Counts: map[string]int{
		"placed": 0, "inProgress": 0, "partialDone": 0, "done": 0, "cancelled": 0,
	}, Rows: make([]OrderRow, 0)}

	// Summary (period only, ignores the status filter).
	args := []interface{}{companyID}
	where := "company_id = $1"
	if from != nil {
		args = append(args, *from)
		where += fmt.Sprintf(" AND created_at::date >= $%d", len(args))
	}
	if to != nil {
		args = append(args, *to)
		where += fmt.Sprintf(" AND created_at::date <= $%d", len(args))
	}
	sumQ := `
		SELECT status,
		       COUNT(*)::int,
		       COALESCE(SUM(total) FILTER (WHERE status <> 'CANCELLED'), 0)::float8
		FROM public.tbl_order
		WHERE ` + where + `
		GROUP BY status`
	rows, err := db.Query(sumQ, args...)
	if err != nil {
		return nil, fmt.Errorf("orders summary failed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var cnt int
		var rev float64
		if err := rows.Scan(&st, &cnt, &rev); err != nil {
			return nil, fmt.Errorf("orders summary scan: %w", err)
		}
		key := normalizeOrderStatus(st)
		rep.Counts[key] += cnt
		rep.Total += cnt
		rep.Revenue += rev
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Rows (period + optional status filter).
	rArgs := []interface{}{companyID}
	rWhere := "o.company_id = $1"
	if from != nil {
		rArgs = append(rArgs, *from)
		rWhere += fmt.Sprintf(" AND o.created_at::date >= $%d", len(rArgs))
	}
	if to != nil {
		rArgs = append(rArgs, *to)
		rWhere += fmt.Sprintf(" AND o.created_at::date <= $%d", len(rArgs))
	}
	if dbStatus := orderStatusToDB(statusKey); dbStatus != "" {
		rArgs = append(rArgs, dbStatus)
		rWhere += fmt.Sprintf(" AND o.status = $%d", len(rArgs))
	}
	rowQ := `
		SELECT
			COALESCE(o.order_code, ''),
			o.order_internal_id::text,
			COALESCE(NULLIF(o.customer_name, ''), o.customer_id, ''),
			o.status,
			COALESCE((SELECT COUNT(*) FROM public.tbl_order_item oi WHERE oi.order_internal_id = o.order_internal_id), 0)::int,
			to_char(o.created_at, 'YYYY-MM-DD'),
			COALESCE(o.total, 0)::float8
		FROM public.tbl_order o
		WHERE ` + rWhere + `
		ORDER BY o.created_at DESC
		LIMIT 500`
	r2, err := db.Query(rowQ, rArgs...)
	if err != nil {
		return nil, fmt.Errorf("orders rows failed: %w", err)
	}
	defer r2.Close()
	for r2.Next() {
		var row OrderRow
		var st string
		if err := r2.Scan(&row.OrderCode, &row.OrderInternalID, &row.CustomerName,
			&st, &row.Items, &row.CreatedAt, &row.Total); err != nil {
			return nil, fmt.Errorf("orders row scan: %w", err)
		}
		row.Status = normalizeOrderStatus(st)
		if row.OrderCode == "" {
			row.OrderCode = row.OrderInternalID
		}
		rep.Rows = append(rep.Rows, row)
	}
	return rep, r2.Err()
}

// ---- Appointments summary -------------------------------------------------

type AppointmentRow struct {
	AppointmentID string `json:"appointmentId"`
	CustomerName  string `json:"customerName"`
	ServiceName   string `json:"serviceName"`
	Status        string `json:"status"`
	Date          string `json:"date"`
	DurationMin   int    `json:"durationMin"`
}

type AppointmentsReport struct {
	Counts        map[string]int   `json:"counts"`
	Total         int              `json:"total"`
	TimeMinutes   int              `json:"timeMinutes"`
	Rows          []AppointmentRow `json:"rows"`
}

// AppointmentsSummary returns bucket counts, time invested, and rows.
func AppointmentsSummary(companyID string, from, to *time.Time, bucket string) (*AppointmentsReport, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	rep := &AppointmentsReport{Counts: map[string]int{
		"scheduled": 0, "finished": 0, "cancelled": 0,
	}, Rows: make([]AppointmentRow, 0)}

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
	sumQ := `
		SELECT status,
		       COUNT(*)::int,
		       COALESCE(SUM(EXTRACT(EPOCH FROM (end_hour - start_hour)) / 60)
		                FILTER (WHERE status NOT IN ('CANCELLED','REJECTED')), 0)::float8
		FROM public.tbl_appointment
		WHERE ` + where + `
		GROUP BY status`
	rows, err := db.Query(sumQ, args...)
	if err != nil {
		return nil, fmt.Errorf("appointments summary failed: %w", err)
	}
	defer rows.Close()
	var minutes float64
	for rows.Next() {
		var st string
		var cnt int
		var mins float64
		if err := rows.Scan(&st, &cnt, &mins); err != nil {
			return nil, fmt.Errorf("appointments summary scan: %w", err)
		}
		rep.Counts[normalizeApptStatus(st)] += cnt
		rep.Total += cnt
		minutes += mins
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rep.TimeMinutes = int(minutes + 0.5)

	// Rows (+ service name via join, + optional bucket filter).
	rArgs := []interface{}{companyID}
	rWhere := "a.company_id = $1"
	if from != nil {
		rArgs = append(rArgs, *from)
		rWhere += fmt.Sprintf(" AND a.appointment_date >= $%d", len(rArgs))
	}
	if to != nil {
		rArgs = append(rArgs, *to)
		rWhere += fmt.Sprintf(" AND a.appointment_date <= $%d", len(rArgs))
	}
	if set := apptBucketSQL(bucket); set != "" {
		rWhere += " AND a.status IN " + set
	}
	rowQ := `
		SELECT
			a.appointment_id::text,
			a.customer_name,
			COALESCE(p.name, ''),
			a.status,
			to_char(a.appointment_date, 'YYYY-MM-DD'),
			COALESCE(EXTRACT(EPOCH FROM (a.end_hour - a.start_hour)) / 60, 0)::int
		FROM public.tbl_appointment a
		LEFT JOIN public.tbl_products p ON p.product_internal_id::text = a.service_id
		WHERE ` + rWhere + `
		ORDER BY a.appointment_date DESC, a.start_hour DESC
		LIMIT 500`
	r2, err := db.Query(rowQ, rArgs...)
	if err != nil {
		return nil, fmt.Errorf("appointments rows failed: %w", err)
	}
	defer r2.Close()
	for r2.Next() {
		var row AppointmentRow
		var st string
		if err := r2.Scan(&row.AppointmentID, &row.CustomerName, &row.ServiceName,
			&st, &row.Date, &row.DurationMin); err != nil {
			return nil, fmt.Errorf("appointments row scan: %w", err)
		}
		row.Status = normalizeApptStatus(st)
		rep.Rows = append(rep.Rows, row)
	}
	return rep, r2.Err()
}

// ---- Top customers --------------------------------------------------------

type CustomerRow struct {
	Name         string  `json:"name"`
	IDType       string  `json:"idType"`
	IDValue      string  `json:"idValue"`
	Orders       int     `json:"orders"`
	Appointments int     `json:"appointments"`
	Spent        float64 `json:"spent"`
}

type TopCustomersReport struct {
	Rows []CustomerRow `json:"rows"`
}

// TopCustomers ranks customers by purchases (orders), then by amount spent.
// A customer is identified by customer_id when present, otherwise by email,
// phone, social link, or name (appointments made without a registered id).
func TopCustomers(companyID string, limit int) (*TopCustomersReport, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}
	if limit <= 0 {
		limit = 10
	}

	q := `
		WITH events AS (
			SELECT
				customer_id AS id_value,
				'customerId' AS id_type,
				COALESCE(NULLIF(customer_name, ''), customer_id) AS name,
				1 AS is_order,
				0 AS is_appt,
				CASE WHEN status <> 'CANCELLED' THEN COALESCE(total, 0) ELSE 0 END AS amount
			FROM public.tbl_order
			WHERE company_id = $1
			UNION ALL
			SELECT
				COALESCE(NULLIF(customer_id, ''), NULLIF(email, ''), NULLIF(phone, ''), NULLIF(social_link, ''), customer_name) AS id_value,
				CASE
					WHEN COALESCE(customer_id, '') <> '' THEN 'customerId'
					WHEN COALESCE(email, '') <> '' THEN 'email'
					WHEN COALESCE(phone, '') <> '' THEN 'phone'
					WHEN COALESCE(social_link, '') <> '' THEN 'link'
					ELSE 'name'
				END AS id_type,
				customer_name AS name,
				0 AS is_order,
				1 AS is_appt,
				0 AS amount
			FROM public.tbl_appointment
			WHERE company_id = $1
		)
		SELECT
			COALESCE(id_value, '') AS id_value,
			MIN(id_type) AS id_type,
			MAX(name) AS name,
			SUM(is_order)::int AS orders,
			SUM(is_appt)::int AS appointments,
			COALESCE(SUM(amount), 0)::float8 AS spent
		FROM events
		WHERE COALESCE(id_value, '') <> ''
		GROUP BY id_value
		ORDER BY orders DESC, spent DESC, appointments DESC
		LIMIT $2`
	rows, err := db.Query(q, companyID, limit)
	if err != nil {
		return nil, fmt.Errorf("top customers failed: %w", err)
	}
	defer rows.Close()
	rep := &TopCustomersReport{Rows: make([]CustomerRow, 0)}
	for rows.Next() {
		var c CustomerRow
		if err := rows.Scan(&c.IDValue, &c.IDType, &c.Name, &c.Orders, &c.Appointments, &c.Spent); err != nil {
			return nil, fmt.Errorf("top customers scan: %w", err)
		}
		rep.Rows = append(rep.Rows, c)
	}
	return rep, rows.Err()
}

// ---- Best-seller products -------------------------------------------------

type ProductRow struct {
	Name    string  `json:"name"`
	Type    string  `json:"type"` // "order" | "booking"
	Units   int     `json:"units"`
	Revenue float64 `json:"revenue"`
}

type BestSellersReport struct {
	Rows []ProductRow `json:"rows"`
}

// BestSellers merges order products (from tbl_order_item) and booking products
// (from tbl_appointment via service_id) into one ranking by units sold.
// typeFilter: "order", "booking", or "" (all).
func BestSellers(companyID string, typeFilter string, limit int) (*BestSellersReport, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}
	if limit <= 0 {
		limit = 50
	}

	q := `
		WITH oi AS (
			SELECT product_id AS pid,
			       SUM(quantity)::int AS units,
			       SUM(quantity * unit_price + additional_price - discount_amount + tax_amount)::float8 AS revenue
			FROM public.tbl_order_item
			WHERE company_id = $1
			GROUP BY product_id
		),
		bk AS (
			SELECT service_id AS pid, COUNT(*)::int AS units
			FROM public.tbl_appointment
			WHERE company_id = $1 AND status NOT IN ('CANCELLED','REJECTED')
			GROUP BY service_id
		),
		merged AS (
			SELECT COALESCE(oi.pid, bk.pid) AS pid,
			       COALESCE(oi.units, 0) AS order_units,
			       COALESCE(oi.revenue, 0) AS order_revenue,
			       COALESCE(bk.units, 0) AS booking_units
			FROM oi FULL OUTER JOIN bk ON oi.pid = bk.pid
		)
		SELECT
			COALESCE(NULLIF(p.name, ''), m.pid) AS name,
			CASE WHEN lower(COALESCE(p.product_type, '')) LIKE '%serv%' THEN 'booking' ELSE 'order' END AS type,
			(m.order_units + m.booking_units)::int AS units,
			(m.order_revenue + m.booking_units * COALESCE(p.base_price, 0))::float8 AS revenue
		FROM merged m
		LEFT JOIN public.tbl_products p ON p.product_internal_id::text = m.pid
		ORDER BY units DESC`
	rows, err := db.Query(q, companyID)
	if err != nil {
		return nil, fmt.Errorf("best sellers failed: %w", err)
	}
	defer rows.Close()
	rep := &BestSellersReport{Rows: make([]ProductRow, 0)}
	tf := strings.TrimSpace(typeFilter)
	for rows.Next() {
		var p ProductRow
		if err := rows.Scan(&p.Name, &p.Type, &p.Units, &p.Revenue); err != nil {
			return nil, fmt.Errorf("best sellers scan: %w", err)
		}
		if tf != "" && tf != "all" && p.Type != tf {
			continue
		}
		rep.Rows = append(rep.Rows, p)
		if len(rep.Rows) >= limit {
			break
		}
	}
	return rep, rows.Err()
}
