package weborder

import (
	"fmt"
	"strconv"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// WebOrderSearchRow is one row for the approval list: the header fields plus a
// small item summary (first item + total quantity + item count).
type WebOrderSearchRow struct {
	WebOrderRequestID     string  `json:"webOrderRequestId"`
	RequestCode           string  `json:"requestCode"`
	OrderType             string  `json:"orderType"`
	RequesterType         string  `json:"requesterType"` // GUEST | PLATFORM_USER
	UseRegisteredCustomer bool    `json:"useRegisteredCustomer"`
	UserCode              *string `json:"userCode,omitempty"`
	CustomerID            *string `json:"customerId,omitempty"`
	CustomerName          string  `json:"customerName"`
	Phone                 *string `json:"phone,omitempty"`
	Email                 *string `json:"email,omitempty"`
	Region                *string `json:"region,omitempty"`
	DeliveryMode          string  `json:"deliveryMode"`
	Total                 float64 `json:"total"`
	Status                string  `json:"status"`
	CreatedAt             string  `json:"createdAt"` // ISO-ish string
	// Item summary.
	FirstItemName *string `json:"firstItemName,omitempty"`
	FirstItemCode *string `json:"firstItemCode,omitempty"`
	TotalQuantity int     `json:"totalQuantity"`
	ItemCount     int     `json:"itemCount"`
}

// WebOrderSearchFilters — CompanyID required; the rest optional.
type WebOrderSearchFilters struct {
	CompanyID     string
	Term          string // matches request_code + customer_name + email
	Status        string // PENDING | APPROVED | REJECTED
	Region        string
	OrderType     string // PRODUCT | SERVICE | MIXED
	RequesterType string // GUEST | PLATFORM_USER
	DateFrom      string // created_at >= (YYYY-MM-DD)
	DateTo        string // created_at <= (YYYY-MM-DD)
	Limit         int
	Offset        int
}

// SearchWebOrders returns the web order requests of ONE company matching the
// filters, newest first, each with a small item summary.
func SearchWebOrders(filters WebOrderSearchFilters) ([]WebOrderSearchRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(filters.CompanyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	conditions := []string{"h.company_id = $1"}
	args := []interface{}{companyID}
	pos := 2

	addEq := func(col, val string) {
		val = strings.TrimSpace(val)
		if val == "" {
			return
		}
		conditions = append(conditions, fmt.Sprintf("%s = $%d", col, pos))
		args = append(args, val)
		pos++
	}

	term := strings.TrimSpace(filters.Term)
	if term != "" {
		conditions = append(conditions, fmt.Sprintf(
			`(
				h.request_code ILIKE $%d
				OR h.customer_name ILIKE $%d
				OR COALESCE(h.email, '') ILIKE $%d
			)`, pos, pos, pos))
		args = append(args, "%"+term+"%")
		pos++
	}

	addEq("h.status", strings.ToUpper(filters.Status))
	addEq("h.region", filters.Region)
	addEq("h.order_type", strings.ToUpper(filters.OrderType))
	addEq("h.requester_type", strings.ToUpper(filters.RequesterType))

	if v := strings.TrimSpace(filters.DateFrom); v != "" {
		conditions = append(conditions, fmt.Sprintf("h.created_at >= $%d", pos))
		args = append(args, v)
		pos++
	}
	if v := strings.TrimSpace(filters.DateTo); v != "" {
		conditions = append(conditions, fmt.Sprintf("h.created_at <= ($%d::date + INTERVAL '1 day')", pos))
		args = append(args, v)
		pos++
	}

	where := "WHERE " + strings.Join(conditions, "\n\t\t\t  AND ")

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}
	limitPh := "$" + strconv.Itoa(pos)
	pos++
	offsetPh := "$" + strconv.Itoa(pos)

	// A lateral subquery builds the item summary per order.
	query := fmt.Sprintf(`
		SELECT
			h.web_order_request_id,
			h.request_code,
			h.order_type,
			h.requester_type,
			h.use_registered_customer,
			h.user_code,
			h.customer_id,
			h.customer_name,
			h.phone,
			h.email,
			h.region,
			h.delivery_mode,
			h.total,
			h.status,
			to_char(h.created_at, 'YYYY-MM-DD"T"HH24:MI:SS'),
			s.first_item_name,
			s.first_item_code,
			COALESCE(s.total_quantity, 0),
			COALESCE(s.item_count, 0)
		FROM public.tbl_web_order_request h
		LEFT JOIN LATERAL (
			SELECT
				(array_agg(i.product_name ORDER BY i.item_order))[1] AS first_item_name,
				(array_agg(i.product_code ORDER BY i.item_order))[1] AS first_item_code,
				SUM(i.quantity) AS total_quantity,
				COUNT(*) AS item_count
			FROM public.tbl_web_order_request_item i
			WHERE i.web_order_request_id = h.web_order_request_id
		) s ON TRUE
		%s
		ORDER BY h.created_at DESC
		LIMIT %s OFFSET %s
	`, where, limitPh, offsetPh)

	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search web orders: %w", err)
	}
	defer rows.Close()

	list := make([]WebOrderSearchRow, 0)
	for rows.Next() {
		var r WebOrderSearchRow
		if err := rows.Scan(
			&r.WebOrderRequestID,
			&r.RequestCode,
			&r.OrderType,
			&r.RequesterType,
			&r.UseRegisteredCustomer,
			&r.UserCode,
			&r.CustomerID,
			&r.CustomerName,
			&r.Phone,
			&r.Email,
			&r.Region,
			&r.DeliveryMode,
			&r.Total,
			&r.Status,
			&r.CreatedAt,
			&r.FirstItemName,
			&r.FirstItemCode,
			&r.TotalQuantity,
			&r.ItemCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan web order: %w", err)
		}
		list = append(list, r)
	}

	return list, rows.Err()
}
