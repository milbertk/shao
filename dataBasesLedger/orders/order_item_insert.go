package order

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// OrderItemS represents one row from public.tbl_order_item.
type OrderItemS struct {
	OrderItemID        string    `json:"orderItemId"`
	OrderInternalID    string    `json:"orderInternalId"`
	CompanyID          string    `json:"companyId"`
	ProductID          string    `json:"productId"`
	InternalCode       *string   `json:"internalCode,omitempty"`
	Quantity           int       `json:"quantity"`
	UnitPrice          float64   `json:"unitPrice"`
	AdditionalPrice    float64   `json:"additionalPrice"`
	TaxPercentage      *float64  `json:"taxPercentage,omitempty"`
	DiscountPercentage *float64  `json:"discountPercentage,omitempty"`
	DiscountAmount     float64   `json:"discountAmount"`
	TaxAmount          float64   `json:"taxAmount"`
	ItemOrder          int       `json:"itemOrder"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// Insert creates one order item.
func (a *OrderItemS) InsertWith(db dbmanager.Executor) error {
	var err error

	if a.Quantity < 1 {
		a.Quantity = 1
	}
	if a.ItemOrder < 1 {
		a.ItemOrder = 1
	}

	query := `
		INSERT INTO public.tbl_order_item (
			order_internal_id,
			company_id,
			product_id,
			internal_code,
			quantity,
			unit_price,
			additional_price,
			discount_percentage,
			discount_amount,
			tax_percentage,
			tax_amount,
			item_order
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING order_item_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.OrderInternalID,
		a.CompanyID,
		a.ProductID,
		a.InternalCode,
		a.Quantity,
		a.UnitPrice,
		a.AdditionalPrice,
		a.DiscountPercentage,
		a.DiscountAmount,
		a.TaxPercentage,
		a.TaxAmount,
		a.ItemOrder,
	).Scan(
		&a.OrderItemID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert order item: %w", err)
	}

	return nil
}

// Insert creates one order item (own connection, no transaction).
func (a *OrderItemS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	return a.InsertWith(db)
}
