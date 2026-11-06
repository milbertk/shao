package blorder

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	order "github.com/milbertk/shao/dataBasesLedger/orders"
)

// ErrOrderNotFound lets handlers distinguish a missing order from a real error.
var ErrOrderNotFound = errors.New("order not found")

// OrderBundle groups an order header with its items.
type OrderBundle struct {
	Order       order.OrderS             `json:"order"`
	Items       []order.OrderItemS       `json:"items"`
	Attachments []order.OrderAttachmentS `json:"attachments"`
}

// BSelectOrderByID loads one order header, scoped to the company.
func BSelectOrderByID(companyID, orderInternalID string) (*order.OrderS, error) {
	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if orderInternalID == "" {
		return nil, errors.New("orderInternalId is required")
	}

	o := order.OrderS{OrderInternalID: orderInternalID, CompanyID: companyID}
	if err := o.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to select order: %w", err)
	}

	return &o, nil
}

// BSelectOrderItems returns the items of an order.
func BSelectOrderItems(companyID, orderInternalID string) ([]order.OrderItemS, error) {
	items, err := order.SelectItemsByOrder(companyID, orderInternalID)
	if err != nil {
		return nil, fmt.Errorf("failed to select order items: %w", err)
	}
	return items, nil
}

// BSelectOrderFull returns the header together with its items.
func BSelectOrderFull(companyID, orderInternalID string) (*OrderBundle, error) {
	header, err := BSelectOrderByID(companyID, orderInternalID)
	if err != nil {
		return nil, err
	}

	items, err := BSelectOrderItems(companyID, orderInternalID)
	if err != nil {
		return nil, err
	}

	attachments, err := order.SelectAttachmentsByOrder(companyID, orderInternalID)
	if err != nil {
		return nil, fmt.Errorf("failed to select order attachments: %w", err)
	}

	return &OrderBundle{Order: *header, Items: items, Attachments: attachments}, nil
}
