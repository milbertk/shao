package order

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const orderItemColumns = `
	order_item_id,
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
	item_order,
	created_at,
	updated_at
`

func scanOrderItem(row interface {
	Scan(dest ...any) error
}, i *OrderItemS) error {
	return row.Scan(
		&i.OrderItemID,
		&i.OrderInternalID,
		&i.CompanyID,
		&i.ProductID,
		&i.InternalCode,
		&i.Quantity,
		&i.UnitPrice,
		&i.AdditionalPrice,
		&i.DiscountPercentage,
		&i.DiscountAmount,
		&i.TaxPercentage,
		&i.TaxAmount,
		&i.ItemOrder,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
}

// SelectItemsByOrder returns every item of an order, scoped to the company,
// ordered by item_order.
func SelectItemsByOrder(companyID, orderInternalID string) ([]OrderItemS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}
	if orderInternalID == "" {
		return nil, fmt.Errorf("orderInternalId is required")
	}

	query := `
		SELECT ` + orderItemColumns + `
		FROM public.tbl_order_item
		WHERE order_internal_id = $1
		  AND company_id = $2
		ORDER BY item_order ASC
	`

	rows, err := db.Query(query, orderInternalID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to select order items: %w", err)
	}
	defer rows.Close()

	list := make([]OrderItemS, 0)
	for rows.Next() {
		var item OrderItemS
		if err := scanOrderItem(rows, &item); err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		list = append(list, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading order items: %w", err)
	}

	return list, nil
}
