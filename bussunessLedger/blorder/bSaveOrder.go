package blorder

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	"github.com/milbertk/shao/bussunessLedger/blstorage"
	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
	"github.com/milbertk/shao/dataBasesLedger/storage"
	emailnotifier "github.com/milbertk/shao/shared"
)

func emptyToNil(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// Order statuses (allowed by the database check ck_order_status_valid):
// PLACED -> IN_PROGRESS -> PARTIAL_DONE -> DONE, or CANCELLED. The status moves
// with the item tracking (bOrderItemTracking); a new order always starts PLACED.
// ("SCHEDULED" is an APPOINTMENT status and is rejected for orders.)
const OrderStatusPlaced = "PLACED"

// resolveOrderStatus (2026-10-30): the status to save. A new order is always
// PLACED, whatever the screen sends; an update keeps the current status
// (status changes come from the tracking, not from the order form).
func resolveOrderStatus(create bool, currentStatus string) string {
	currentStatus = strings.ToUpper(strings.TrimSpace(currentStatus))
	if create || currentStatus == "" {
		return OrderStatusPlaced
	}
	return currentStatus
}

// OrderSaveEffects is what must happen AFTER the transaction of an order
// save is committed (storage count + customer emails).
type OrderSaveEffects struct {
	companyID      string
	orderID        string
	channel        string
	status         string
	createdNow     bool
	previousStatus string
	savedFiles     []order.OrderAttachmentS
	// detailsChanged (2026-10-30): dates, items or total changed in an update.
	detailsChanged bool
}

// Apply runs the after-commit work. Call it only after COMMIT.
func (e *OrderSaveEffects) Apply() {
	if e == nil {
		return
	}
	// Count the attachments in the company storage (2026-10-23; never blocks).
	for _, att := range e.savedFiles {
		if att.StoragePath != nil {
			blstorage.TrackSavedFile(e.companyID, "", *att.StoragePath, att.FileSize, storage.KindOrderAttachment)
		}
	}

	// Customer notifications (background). Orders created from an approved
	// web order (channel "web") are announced by the web-order approval email.
	newStatus := strings.ToUpper(strings.TrimSpace(e.status))
	if newStatus == "" {
		newStatus = OrderStatusPlaced
	}
	switch {
	case e.createdNow && !strings.EqualFold(e.channel, "web"):
		blnotifications.NotifyOrder(e.companyID, e.orderID, blnotifications.EventCreated)
	case !e.createdNow && e.previousStatus != "" && e.previousStatus != newStatus:
		blnotifications.NotifyOrder(e.companyID, e.orderID, blnotifications.EventStatusChanged)
	case !e.createdNow && e.detailsChanged:
		// Same status but new dates / items / total (2026-10-30).
		blnotifications.NotifyOrderUpdated(e.companyID, e.orderID)
	}
}

// BSaveOrder creates or updates an order header, then REWRITES its items
// (delete all + insert the provided list) in ONE database transaction.
//
//   - companyID is authoritative and forced onto the header and every item.
//   - Header: empty/unknown orderInternalId -> create; otherwise -> update.
//   - Items: full replace.
//   - The total is recomputed from the items on the server.
//
// Returns the fresh header + items.
func BSaveOrder(companyID string, input OrderBundle) (*OrderBundle, error) {
	var effects *OrderSaveEffects
	err := dbmanager.WithTx(func(tx *sql.Tx) error {
		var err error
		effects, err = BSaveOrderInTx(tx, companyID, &input)
		return err
	})
	if err != nil {
		return nil, err
	}
	effects.Apply()
	return BSelectOrderFull(effects.companyID, effects.orderID)
}

// BSaveOrderInTx saves the order inside the caller's transaction (2026-10-24),
// so other steps (e.g. approving a web order) are saved together with it.
// After COMMIT the caller must run effects.Apply().
func BSaveOrderInTx(tx *sql.Tx, companyID string, input *OrderBundle) (*OrderSaveEffects, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	input.Order.CompanyID = companyID
	input.Order.OrderCode = emptyToNil(input.Order.OrderCode)
	input.Order.CustomerName = emptyToNil(input.Order.CustomerName)
	input.Order.StartDate = emptyToNil(input.Order.StartDate)
	input.Order.EstimatedDeliveryDate = emptyToNil(input.Order.EstimatedDeliveryDate)
	input.Order.Notes = emptyToNil(input.Order.Notes)
	input.Order.GeneralTrackingID = emptyToNil(input.Order.GeneralTrackingID)
	input.Order.CustomerEmail = emptyToNil(input.Order.CustomerEmail)
	if input.Order.CustomerEmail != nil && !emailnotifier.ValidEmail(*input.Order.CustomerEmail) {
		return nil, errors.New("invalid customerEmail: enter one valid email address or leave it empty")
	}

	// For notifications: status and details before the change.
	previousStatus := ""
	var previousDetails blnotifications.OrderSnapshot
	havePrevious := false

	if strings.TrimSpace(input.Order.CustomerID) == "" {
		return nil, errors.New("customerId is required")
	}
	// Appointments are booked, never added to an order (2026-10-31).
	if err := BCheckNoAppointments(companyID, orderItemProductIDs(input.Items)); err != nil {
		return nil, err
	}

	// Recompute the total from the items (server is the source of truth).
	// Math per line: base -> apply discount % -> tax the discounted amount.
	var total float64
	for i := range input.Items {
		item := &input.Items[i]

		base := float64(item.Quantity) * (item.UnitPrice + item.AdditionalPrice)

		// Discount (percentage) applied first.
		discountPct := 0.0
		if item.DiscountPercentage != nil {
			discountPct = *item.DiscountPercentage
		}
		discountAmount := base * (discountPct / 100.0)
		taxedBase := base - discountAmount

		// Tax is computed on the discounted amount.
		taxPct := 0.0
		if item.TaxPercentage != nil {
			taxPct = *item.TaxPercentage
		}
		taxAmount := taxedBase * (taxPct / 100.0)

		// Store the resolved amounts as a fixed record.
		item.DiscountAmount = discountAmount
		item.TaxAmount = taxAmount

		total += taxedBase + taxAmount
	}
	input.Order.Total = total

	// ---- Decide create or update ----
	orderID := strings.TrimSpace(input.Order.OrderInternalID)
	create := orderID == ""
	if !create {
		existing := order.OrderS{OrderInternalID: orderID, CompanyID: companyID}
		if err := existing.Select(); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("failed to validate order: %w", err)
			}
			input.Order.OrderInternalID = ""
			create = true
		} else {
			previousStatus = strings.ToUpper(strings.TrimSpace(existing.Status))
			input.Order.Status = resolveOrderStatus(false, existing.Status)
			if oldItems, err := order.SelectItemsByOrder(companyID, orderID); err == nil {
				previousDetails = blnotifications.SnapshotOrder(existing, oldItems)
				havePrevious = true
			}
		}
	}

	// ---- Header + items + attachments (all inside the transaction) ----
	var savedFiles []order.OrderAttachmentS
	if create {
		input.Order.Status = resolveOrderStatus(true, "")
		if err := input.Order.InsertWith(tx); err != nil {
			return nil, fmt.Errorf("failed to create order: %w", err)
		}
	} else if err := input.Order.UpdateWith(tx); err != nil {
		return nil, fmt.Errorf("failed to update order: %w", err)
	}
	txOrderID := input.Order.OrderInternalID

	// Rewrite items (full replace).
	if _, err := order.DeleteItemsByOrderWith(tx, companyID, txOrderID); err != nil {
		return nil, fmt.Errorf("failed to clear order items: %w", err)
	}
	for i := range input.Items {
		item := &input.Items[i]
		item.OrderInternalID = txOrderID
		item.CompanyID = companyID
		item.InternalCode = emptyToNil(item.InternalCode)
		if item.ItemOrder < 1 {
			item.ItemOrder = i + 1
		}
		if item.TaxPercentage != nil && *item.TaxPercentage < 0 {
			item.TaxPercentage = nil
		}
		if item.DiscountPercentage != nil && *item.DiscountPercentage < 0 {
			item.DiscountPercentage = nil
		}
		if err := item.InsertWith(tx); err != nil {
			return nil, fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	// Rewrite attachments (references only; files live in Firebase).
	if _, err := order.DeleteAttachmentsByOrderWith(tx, companyID, txOrderID); err != nil {
		return nil, fmt.Errorf("failed to clear order attachments: %w", err)
	}
	for i := range input.Attachments {
		att := &input.Attachments[i]
		att.OrderInternalID = txOrderID
		att.CompanyID = companyID
		att.StoragePath = emptyToNil(att.StoragePath)
		att.FileName = emptyToNil(att.FileName)
		att.FileType = emptyToNil(att.FileType)
		if strings.TrimSpace(att.URL) == "" {
			continue // skip empty references
		}
		// Only files of this company in our bucket (2026-10-25, security S13).
		path := ""
		if att.StoragePath != nil {
			path = *att.StoragePath
		}
		if err := blstorage.ValidateCompanyFile(companyID, path, att.URL); err != nil {
			return nil, err
		}
		if err := att.InsertWith(tx); err != nil {
			return nil, fmt.Errorf("failed to insert order attachment: %w", err)
		}
		savedFiles = append(savedFiles, *att)
	}

	return &OrderSaveEffects{
		companyID:      companyID,
		orderID:        input.Order.OrderInternalID,
		channel:        input.Order.Channel,
		status:         input.Order.Status,
		createdNow:     create,
		previousStatus: previousStatus,
		savedFiles:     savedFiles,
		detailsChanged: !create && havePrevious &&
			blnotifications.OrderDetailsChanged(previousDetails, blnotifications.SnapshotOrder(input.Order, input.Items)),
	}, nil
}
