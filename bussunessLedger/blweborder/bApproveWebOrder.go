package blweborder

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	blorder "github.com/milbertk/shao/bussunessLedger/blorder"
	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
)

// ErrWebOrderAlreadyApproved is returned when trying to approve twice.
var ErrWebOrderAlreadyApproved = errors.New("web order is already approved")

// ApproveResult wraps the updated web order bundle and the created real order.
type ApproveResult struct {
	WebOrder     *WebOrderBundle      `json:"webOrder"`
	CreatedOrder *blorder.OrderBundle `json:"createdOrder"`
}

func ptr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// BApproveWebOrder approves a web order:
//  1. guards against double-approval,
//  2. creates a REAL order (via blorder.BSaveOrderInTx) from the web order,
//  3. links the order to the web order and sets it to APPROVED.
//
// Steps 2 and 3 run in ONE database transaction (2026-10-24): either the
// order exists AND the web order is approved, or nothing changed.
//
// companyID is authoritative. Returns the updated web order + the created order.
func BApproveWebOrder(companyID, webOrderRequestID string) (*ApproveResult, error) {
	companyID = strings.TrimSpace(companyID)
	webOrderRequestID = strings.TrimSpace(webOrderRequestID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if webOrderRequestID == "" {
		return nil, errors.New("webOrderRequestId is required")
	}

	// Load the web order (header + items).
	bundle, err := BSelectWebOrderFull(companyID, webOrderRequestID)
	if err != nil {
		return nil, err // includes ErrWebOrderNotFound
	}

	// Guard: do not approve twice (avoid duplicate real orders).
	if strings.EqualFold(bundle.Order.Status, "APPROVED") {
		return nil, ErrWebOrderAlreadyApproved
	}

	// ---- Build the real order from the web order ----
	// Customer reference: prefer the platform user's code; otherwise use the
	// request code so the order always has a (non-empty) customer reference.
	customerID := deref(bundle.Order.UserCode)
	if customerID == "" {
		customerID = bundle.Order.RequestCode
	}

	// Notes: keep a trace of the origin.
	notes := deref(bundle.Order.Notes)
	origin := "Web order " + bundle.Order.RequestCode
	if notes == "" {
		notes = origin
	} else {
		notes = origin + " | " + notes
	}

	items := make([]order.OrderItemS, 0, len(bundle.Items))
	for i := range bundle.Items {
		wi := bundle.Items[i]
		items = append(items, order.OrderItemS{
			CompanyID:    companyID,
			ProductID:    wi.ProductID,
			InternalCode: wi.ProductCode, // *string already
			Quantity:     wi.Quantity,
			UnitPrice:    wi.UnitPrice,
			ItemOrder:    wi.ItemOrder,
		})
	}

	orderInput := blorder.OrderBundle{
		Order: order.OrderS{
			CompanyID:          companyID,
			CustomerID:         customerID,
			CustomerName:       ptr(bundle.Order.CustomerName),
			Channel:            "web",
			Notes:              ptr(notes),
			CustomerEmail:      bundle.Order.Email, // for order status notifications
			UseGeneralTracking: false,
			Status:             blorder.OrderStatusPlaced, // order status (was "SCHEDULED", rejected by ck_order_status_valid)
		},
		Items: items,
	}

	var effects *blorder.OrderSaveEffects
	err = dbmanager.WithTx(func(tx *sql.Tx) error {
		var err error
		effects, err = blorder.BSaveOrderInTx(tx, companyID, &orderInput)
		if err != nil {
			return fmt.Errorf("failed to create order from web order: %w", err)
		}

		// Link the created order back to this web-order request so the order
		// Details can show the customer's contact info + map location.
		if err := order.SetOrderWebRequestLinkWith(tx, companyID, orderInput.Order.OrderInternalID, webOrderRequestID); err != nil {
			return err
		}

		// Mark the web order as APPROVED (only if it was not approved meanwhile).
		affected, err := weborder.MarkApprovedWith(tx, companyID, webOrderRequestID)
		if err != nil {
			return err
		}
		if affected == 0 {
			return ErrWebOrderAlreadyApproved
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// After COMMIT: storage count, then tell the customer (background).
	effects.Apply()
	blnotifications.NotifyWebOrder(companyID, webOrderRequestID, blnotifications.EventApproved)

	createdOrder, err := blorder.BSelectOrderFull(companyID, orderInput.Order.OrderInternalID)
	if err != nil {
		return nil, err
	}
	updated, err := BSelectWebOrderFull(companyID, webOrderRequestID)
	if err != nil {
		return nil, err
	}

	return &ApproveResult{
		WebOrder:     updated,
		CreatedOrder: createdOrder,
	}, nil
}
