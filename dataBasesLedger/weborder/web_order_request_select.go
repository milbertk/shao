package weborder

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// SelectWebOrderByID loads one web order header, scoped to the company.
// Returns sql.ErrNoRows when it does not exist.
func SelectWebOrderByID(companyID, webOrderRequestID string) (*WebOrderRequestS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	webOrderRequestID = strings.TrimSpace(webOrderRequestID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}
	if webOrderRequestID == "" {
		return nil, fmt.Errorf("webOrderRequestId is required")
	}

	query := `
		SELECT
			web_order_request_id,
			company_id,
			request_code,
			order_type,
			use_registered_customer,
			customer_id,
			customer_name,
			phone,
			email,
			social_link,
			preferred_contact,
			contact_value,
			region,
			address,
			delivery_mode,
			latitude,
			longitude,
			notes,
			total,
			status,
			requester_type,
			user_code,
			created_at,
			updated_at
		FROM public.tbl_web_order_request
		WHERE web_order_request_id = $1
		  AND company_id = $2
	`

	var a WebOrderRequestS
	err = db.QueryRow(query, webOrderRequestID, companyID).Scan(
		&a.WebOrderRequestID,
		&a.CompanyID,
		&a.RequestCode,
		&a.OrderType,
		&a.UseRegisteredCustomer,
		&a.CustomerID,
		&a.CustomerName,
		&a.Phone,
		&a.Email,
		&a.SocialLink,
		&a.PreferredContact,
		&a.ContactValue,
		&a.Region,
		&a.Address,
		&a.DeliveryMode,
		&a.Latitude,
		&a.Longitude,
		&a.Notes,
		&a.Total,
		&a.Status,
		&a.RequesterType,
		&a.UserCode,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to select web order: %w", err)
	}

	return &a, nil
}

// SelectWebOrderItems returns the items of a web order, ordered.
func SelectWebOrderItems(companyID, webOrderRequestID string) ([]WebOrderRequestItemS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT
			web_order_request_item_id,
			web_order_request_id,
			company_id,
			product_id,
			product_code,
			product_name,
			quantity,
			unit_price,
			item_order,
			created_at
		FROM public.tbl_web_order_request_item
		WHERE web_order_request_id = $1
		  AND company_id = $2
		ORDER BY item_order ASC
	`

	rows, err := db.Query(query, strings.TrimSpace(webOrderRequestID), strings.TrimSpace(companyID))
	if err != nil {
		return nil, fmt.Errorf("failed to select web order items: %w", err)
	}
	defer rows.Close()

	list := make([]WebOrderRequestItemS, 0)
	for rows.Next() {
		var it WebOrderRequestItemS
		if err := rows.Scan(
			&it.WebOrderRequestItemID,
			&it.WebOrderRequestID,
			&it.CompanyID,
			&it.ProductID,
			&it.ProductCode,
			&it.ProductName,
			&it.Quantity,
			&it.UnitPrice,
			&it.ItemOrder,
			&it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan web order item: %w", err)
		}
		list = append(list, it)
	}
	return list, rows.Err()
}

// MarkApprovedWith sets the status to APPROVED inside a transaction, only
// when it is not APPROVED yet (two people approving at the same time cannot
// create two orders). Returns the rows affected (0 = not found or already
// approved).
func MarkApprovedWith(db dbmanager.Executor, companyID, webOrderRequestID string) (int64, error) {
	result, err := db.Exec(`
		UPDATE public.tbl_web_order_request
		SET status = 'APPROVED'
		WHERE web_order_request_id = $1
		  AND company_id = $2
		  AND status <> 'APPROVED'
	`, strings.TrimSpace(webOrderRequestID), strings.TrimSpace(companyID))
	if err != nil {
		return 0, fmt.Errorf("failed to approve web order: %w", err)
	}
	return result.RowsAffected()
}

// UpdateWebOrderStatus changes only the status, scoped to the company.
// Returns the number of rows affected (0 if not found in that company).
func UpdateWebOrderStatus(companyID, webOrderRequestID, status string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	webOrderRequestID = strings.TrimSpace(webOrderRequestID)
	status = strings.ToUpper(strings.TrimSpace(status))
	if companyID == "" || webOrderRequestID == "" {
		return 0, fmt.Errorf("companyId and webOrderRequestId are required")
	}

	result, err := db.Exec(`
		UPDATE public.tbl_web_order_request
		SET status = $3
		WHERE web_order_request_id = $1
		  AND company_id = $2
	`, webOrderRequestID, companyID, status)
	if err != nil {
		return 0, fmt.Errorf("failed to update web order status: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count updated web order: %w", err)
	}
	return affected, nil
}
