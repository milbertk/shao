package order

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// OrderS represents one row from public.tbl_order (the order header).
type OrderS struct {
	OrderInternalID       string    `json:"orderInternalId"`
	CompanyID             string    `json:"companyId"`
	OrderCode             *string   `json:"orderCode,omitempty"`
	CustomerID            string    `json:"customerId"`
	CustomerName          *string   `json:"customerName,omitempty"`
	Channel               string    `json:"channel"`
	StartDate             *string   `json:"startDate,omitempty"`             // YYYY-MM-DD
	EstimatedDeliveryDate *string   `json:"estimatedDeliveryDate,omitempty"` // YYYY-MM-DD
	Notes                 *string   `json:"notes,omitempty"`
	UseGeneralTracking    bool      `json:"useGeneralTracking"`
	GeneralTrackingID     *string   `json:"generalTrackingId,omitempty"`
	Total                 float64   `json:"total"`
	Status                string    `json:"status"`
	CustomerEmail         *string   `json:"customerEmail,omitempty"` // optional: status notifications
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

// Insert creates one order header.
func (a *OrderS) InsertWith(db dbmanager.Executor) error {
	var err error

	query := `
		INSERT INTO public.tbl_order (
			company_id,
			order_code,
			customer_id,
			customer_name,
			channel,
			start_date,
			estimated_delivery_date,
			notes,
			use_general_tracking,
			general_tracking_id,
			total,
			status,
			customer_email
		)
		VALUES (
			$1, $2, $3, $4,
			COALESCE(NULLIF($5, ''), 'whatsapp'),
			$6, $7, $8, $9, $10, $11,
			COALESCE(NULLIF($12, ''), 'PLACED'),
			$13
		)
		RETURNING order_internal_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.CompanyID,
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
		return fmt.Errorf("failed to insert order: %w", err)
	}

	return nil
}

// Insert creates one order header (own connection, no transaction).
func (a *OrderS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	return a.InsertWith(db)
}
