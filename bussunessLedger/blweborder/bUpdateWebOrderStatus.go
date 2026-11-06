package blweborder

import (
	"errors"
	"fmt"
	"strings"

	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
)

var validWebOrderStatuses = map[string]bool{
	"PENDING":  true,
	"APPROVED": true,
	"REJECTED": true,
}

// BUpdateWebOrderStatus changes the status of a web order (PENDING/APPROVED/
// REJECTED), scoped to the company. Returns the fresh full bundle.
func BUpdateWebOrderStatus(companyID, webOrderRequestID, status string) (*WebOrderBundle, error) {
	companyID = strings.TrimSpace(companyID)
	webOrderRequestID = strings.TrimSpace(webOrderRequestID)
	status = strings.ToUpper(strings.TrimSpace(status))

	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if webOrderRequestID == "" {
		return nil, errors.New("webOrderRequestId is required")
	}
	if !validWebOrderStatuses[status] {
		return nil, errors.New("invalid status (use PENDING, APPROVED or REJECTED)")
	}

	affected, err := weborder.UpdateWebOrderStatus(companyID, webOrderRequestID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to update web order status: %w", err)
	}
	if affected == 0 {
		return nil, ErrWebOrderNotFound
	}

	// Tell the customer (background, best-effort).
	switch status {
	case "APPROVED":
		blnotifications.NotifyWebOrder(companyID, webOrderRequestID, blnotifications.EventApproved)
	case "REJECTED":
		blnotifications.NotifyWebOrder(companyID, webOrderRequestID, blnotifications.EventRejected)
	}

	return BSelectWebOrderFull(companyID, webOrderRequestID)
}
