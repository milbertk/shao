package blorderitemtracking

import (
	"errors"
	"fmt"
	"strings"

	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
)

// OrderTrackingView is an order's items with their current step + the computed
// order status.
type OrderTrackingView struct {
	OrderInternalID string                          `json:"orderInternalId"`
	OrderStatus     string                          `json:"orderStatus"`
	Items           []order.OrderItemTrackingRow    `json:"items"`
	History         []order.OrderItemStepHistoryRow `json:"history"`
}

// recomputeOrderStatus applies the frontend rule on the server:
//
//	no item started            -> PLACED
//	some started, none done     -> IN_PROGRESS
//	at least one done, not all  -> PARTIAL_DONE
//	all items done              -> DONE
//
// (CANCELLED is set explicitly elsewhere and is never overwritten here.)
func recomputeOrderStatus(companyID, orderInternalID string) (string, error) {
	total, started, done, err := order.CountItemsDoneStatus(companyID, orderInternalID)
	if err != nil {
		return "", err
	}
	status := "PLACED"
	switch {
	case total == 0:
		status = "PLACED"
	case done == total && total > 0:
		status = "DONE"
	case done >= 1:
		status = "PARTIAL_DONE"
	case started >= 1:
		status = "IN_PROGRESS"
	default:
		status = "PLACED"
	}
	return status, nil
}

// BSelectOrderTracking returns the items + computed order status.
func BSelectOrderTracking(companyID, orderInternalID string) (*OrderTrackingView, error) {
	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if orderInternalID == "" {
		return nil, errors.New("orderInternalId is required")
	}

	items, err := order.SelectOrderItemsTracking(companyID, orderInternalID)
	if err != nil {
		return nil, err
	}
	status, err := recomputeOrderStatus(companyID, orderInternalID)
	if err != nil {
		return nil, err
	}
	history, err := order.SelectOrderItemStepHistory(companyID, orderInternalID)
	if err != nil {
		return nil, err
	}
	return &OrderTrackingView{
		OrderInternalID: orderInternalID,
		OrderStatus:     status,
		Items:           items,
		History:         history,
	}, nil
}

// BUpdateOrderItemStep updates one item's step, then recomputes + persists the
// order status (unless the order is CANCELLED), records a history entry, and
// returns the fresh view.
//
// Validation (looser requires_previous rule): a target step flagged
// requires_previous can only be set when the item already has an EARLIER step
// set (current step_order < target step_order).
func BUpdateOrderItemStep(
	companyID, orderInternalID, orderItemID string,
	stepID string, stepStatus string,
	actorUserID, actorEmail string,
) (*OrderTrackingView, error) {
	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	orderItemID = strings.TrimSpace(orderItemID)
	if companyID == "" || orderInternalID == "" || orderItemID == "" {
		return nil, errors.New("companyId, orderInternalId and orderItemId are required")
	}

	trimmedStep := strings.TrimSpace(stepID)
	var stepPtr *string
	var stepName string
	notifyStep := false // the step is checked "Notify the customer" (2026-10-31)
	if trimmedStep != "" {
		stepPtr = &trimmedStep

		// Load the target step (for requires_previous + name snapshot).
		target, err := order.GetStepInfo(companyID, trimmedStep)
		if err != nil {
			return nil, err
		}
		if target == nil {
			return nil, errors.New("the selected step does not exist")
		}
		stepName = target.Name
		notifyStep = target.NotifyCustomer

		if target.RequiresPrevious {
			currentOrder, err := order.GetItemCurrentStepOrder(companyID, orderInternalID, orderItemID)
			if err != nil {
				return nil, err
			}
			// Only enforce when moving FORWARD into this step. Moving backward
			// (a correction) is always allowed. Moving forward into a step that
			// requires a previous one is allowed only from the IMMEDIATELY
			// preceding step, so you cannot skip ahead.
			if target.StepOrder > currentOrder {
				if currentOrder < 0 || target.StepOrder-currentOrder != 1 {
					return nil, errors.New("invalid step: the immediately previous step must be set first")
				}
			}
		}
	}

	// The item before the change (2026-10-30): to email the customer only when
	// its step or step status really changes, with the product name.
	previousStep, previousStepStatus, productName := "", "", ""
	if rows, err := order.SelectOrderItemsTracking(companyID, orderInternalID); err == nil {
		for _, row := range rows {
			if row.OrderItemID == orderItemID {
				if row.CurrentStepID != nil {
					previousStep = strings.TrimSpace(*row.CurrentStepID)
				}
				previousStepStatus = strings.ToUpper(strings.TrimSpace(row.CurrentStepStatus))
				if row.ProductName != nil {
					productName = strings.TrimSpace(*row.ProductName)
				}
				break
			}
		}
	}

	affected, err := order.UpdateOrderItemStep(companyID, orderInternalID, orderItemID, stepPtr, stepStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to update order item step: %w", err)
	}
	if affected == 0 {
		return nil, errors.New("order item not found")
	}

	previousStatus := blnotifications.CurrentOrderStatus(companyID, orderInternalID)

	status, err := recomputeOrderStatus(companyID, orderInternalID)
	if err != nil {
		return nil, err
	}
	if err := order.SetOrderStatus(companyID, orderInternalID, status); err != nil {
		return nil, err
	}
	// Tell the customer (2026-10-31): see itemStepEmail.
	stepChanged := trimmedStep != previousStep ||
		!strings.EqualFold(strings.TrimSpace(stepStatus), previousStepStatus)
	switch itemStepEmail(previousStatus, status, stepChanged && trimmedStep != "", notifyStep) {
	case emailItemStep:
		blnotifications.NotifyOrderItemStep(companyID, orderInternalID, productName, stepName, stepStatus)
	case emailOrderStatus:
		blnotifications.NotifyOrder(companyID, orderInternalID, blnotifications.EventStatusChanged)
	}

	// Record the movement in the timeline (best-effort: a history failure should
	// not undo the successful update).
	if herr := order.InsertOrderItemStepHistory(
		companyID, orderInternalID, orderItemID, stepPtr, stepName, stepStatus, status, actorUserID, actorEmail,
	); herr != nil {
		// Log-only; the update already succeeded.
		fmt.Printf("[BUpdateOrderItemStep] history insert failed: %v\n", herr)
	}

	return BSelectOrderTracking(companyID, orderInternalID)
}

// Which customer email an item step change sends (2026-10-31).
const (
	emailNone        = ""
	emailItemStep    = "ITEM_STEP"
	emailOrderStatus = "ORDER_STATUS"
)

// itemStepEmail (pure, tested):
//   - CANCELLED order: nothing.
//   - the item moved to a step checked "Notify the customer": the step email
//     (it also shows the order status, so no second email);
//   - otherwise, when the ORDER status changed (Placed -> In progress ->
//     Partly done -> Done): the order status email, always;
//   - otherwise nothing: an unchecked step is internal information.
func itemStepEmail(previousStatus, newStatus string, stepChanged, stepNotifies bool) string {
	if strings.EqualFold(strings.TrimSpace(previousStatus), "CANCELLED") {
		return emailNone
	}
	if stepChanged && stepNotifies {
		return emailItemStep
	}
	if strings.TrimSpace(previousStatus) != "" && !strings.EqualFold(previousStatus, newStatus) {
		return emailOrderStatus
	}
	return emailNone
}
