package order

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const orderColumns = `
	order_internal_id,
	company_id,
	order_code,
	customer_id,
	customer_name,
	channel,
	to_char(start_date, 'YYYY-MM-DD'),
	to_char(estimated_delivery_date, 'YYYY-MM-DD'),
	notes,
	use_general_tracking,
	general_tracking_id,
	total,
	status,
	customer_email,
	created_at,
	updated_at
`

func scanOrder(row interface {
	Scan(dest ...any) error
}, o *OrderS) error {
	return row.Scan(
		&o.OrderInternalID,
		&o.CompanyID,
		&o.OrderCode,
		&o.CustomerID,
		&o.CustomerName,
		&o.Channel,
		&o.StartDate,
		&o.EstimatedDeliveryDate,
		&o.Notes,
		&o.UseGeneralTracking,
		&o.GeneralTrackingID,
		&o.Total,
		&o.Status,
		&o.CustomerEmail,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
}

// Select loads one order header by primary key, scoped to its company.
func (a *OrderS) Select() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	orderInternalID := strings.TrimSpace(a.OrderInternalID)
	companyID := strings.TrimSpace(a.CompanyID)
	if orderInternalID == "" {
		return fmt.Errorf("orderInternalId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		SELECT ` + orderColumns + `
		FROM public.tbl_order
		WHERE order_internal_id = $1
		  AND company_id = $2
	`

	if err := scanOrder(db.QueryRow(query, orderInternalID, companyID), a); err != nil {
		return fmt.Errorf("failed to select order: %w", err)
	}

	return nil
}
