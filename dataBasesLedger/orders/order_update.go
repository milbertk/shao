package order

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update modifies one order header by primary key AND company.
func (a *OrderS) UpdateWith(db dbmanager.Executor) error {
	var err error

	orderInternalID := strings.TrimSpace(a.OrderInternalID)
	companyID := strings.TrimSpace(a.CompanyID)
	if orderInternalID == "" {
		return fmt.Errorf("orderInternalId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		UPDATE public.tbl_order
		SET
			order_code = $3,
			customer_id = $4,
			customer_name = $5,
			channel = COALESCE(NULLIF($6, ''), 'whatsapp'),
			start_date = $7,
			estimated_delivery_date = $8,
			notes = $9,
			use_general_tracking = $10,
			general_tracking_id = $11,
			total = $12,
			status = COALESCE(NULLIF($13, ''), 'PLACED'),
			customer_email = $14
		WHERE order_internal_id = $1
		  AND company_id = $2
		RETURNING order_internal_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		orderInternalID,
		companyID,
		a.OrderCode,
		a.CustomerID,
		a.CustomerName,
		a.Channel,
		a.StartDate,
		a.EstimatedDeliveryDate,
		a.Notes,
		a.UseGeneralTracking,
		a.GeneralTrackingID,
		a.Total,
		a.Status,
		a.CustomerEmail,
	).Scan(
		&a.OrderInternalID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("order %q does not exist", orderInternalID)
		}
		return fmt.Errorf("failed to update order: %w", err)
	}

	return nil
}

// Update modifies one order header (own connection, no transaction).
func (a *OrderS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	return a.UpdateWith(db)
}
