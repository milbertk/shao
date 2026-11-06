package blweborder

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/weborder"
)

// ErrWebOrderNotFound distinguishes a missing web order from a real error.
var ErrWebOrderNotFound = errors.New("web order not found")

// WebOrderBundle groups a web order header with its items.
type WebOrderBundle struct {
	Order weborder.WebOrderRequestS       `json:"order"`
	Items []weborder.WebOrderRequestItemS `json:"items"`
}

// BSelectWebOrderFull returns the header + items for one web order.
func BSelectWebOrderFull(companyID, webOrderRequestID string) (*WebOrderBundle, error) {
	companyID = strings.TrimSpace(companyID)
	webOrderRequestID = strings.TrimSpace(webOrderRequestID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if webOrderRequestID == "" {
		return nil, errors.New("webOrderRequestId is required")
	}

	header, err := weborder.SelectWebOrderByID(companyID, webOrderRequestID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWebOrderNotFound
		}
		return nil, fmt.Errorf("failed to select web order: %w", err)
	}

	items, err := weborder.SelectWebOrderItems(companyID, webOrderRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to select web order items: %w", err)
	}

	return &WebOrderBundle{Order: *header, Items: items}, nil
}
