package weborder

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CustomerWebOrderRow is one web order of a signed-in customer ("My orders").
type CustomerWebOrderRow struct {
	WebOrderRequestID string    `json:"-"`
	RequestCode       string    `json:"reference"`
	Status            string    `json:"status"`
	Total             float64   `json:"total"`
	ItemsCount        int       `json:"itemsCount"`
	ItemsSummary      string    `json:"itemsSummary"`
	DeliveryMode      string    `json:"deliveryMode"`
	CreatedAt         time.Time `json:"createdAt"`
	PublicToken       *string   `json:"-"`
}

// SelectWebOrdersForCustomer returns the web orders of one company made by
// the platform user userCode OR (when email is not empty) with that email,
// newest first. email must only be passed when it is VERIFIED.
func SelectWebOrdersForCustomer(companyID, userCode, email string, limit int) ([]CustomerWebOrderRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	userCode = strings.TrimSpace(userCode)
	email = strings.TrimSpace(email)
	if companyID == "" || (userCode == "" && email == "") {
		return nil, fmt.Errorf("companyId and a customer are required")
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	rows, err := db.Query(`
		SELECT
			h.web_order_request_id,
			h.request_code,
			h.status,
			h.total,
			COALESCE(SUM(i.quantity), 0)::int AS items_count,
			COALESCE(string_agg(COALESCE(i.product_name, '-') || ' x' || i.quantity::text, ', ' ORDER BY i.item_order), '') AS items_summary,
			h.delivery_mode,
			h.created_at,
			h.public_token
		FROM public.tbl_web_order_request h
		LEFT JOIN public.tbl_web_order_request_item i
		       ON i.web_order_request_id = h.web_order_request_id
		WHERE h.company_id::text = $1
		  AND (
		        ($2::text <> '' AND h.user_code = $2::text)
		     OR ($3::text <> '' AND lower(h.email) = lower($3::text))
		  )
		GROUP BY h.web_order_request_id, h.request_code, h.status, h.total,
		         h.delivery_mode, h.created_at, h.public_token
		ORDER BY h.created_at DESC
		LIMIT $4
	`, companyID, userCode, email, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to select customer web orders: %w", err)
	}
	defer rows.Close()

	list := make([]CustomerWebOrderRow, 0)
	for rows.Next() {
		var r CustomerWebOrderRow
		if err := rows.Scan(
			&r.WebOrderRequestID, &r.RequestCode, &r.Status, &r.Total,
			&r.ItemsCount, &r.ItemsSummary, &r.DeliveryMode, &r.CreatedAt, &r.PublicToken,
		); err != nil {
			return nil, fmt.Errorf("failed to scan customer web order: %w", err)
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading customer web orders: %w", err)
	}
	return list, nil
}
