package order

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// OrderItemTrackingRow is one item's tracking view for an order.
type OrderItemTrackingRow struct {
	OrderItemID       string                `json:"orderItemId"`
	OrderInternalID   string                `json:"orderInternalId"`
	CompanyID         string                `json:"companyId"`
	ProductID         string                `json:"productId"`
	ProductName       *string               `json:"productName,omitempty"`
	InternalCode      *string               `json:"internalCode,omitempty"`
	Quantity          int                   `json:"quantity"`
	CurrentStepID     *string               `json:"currentStepId,omitempty"`
	CurrentStepStatus string                `json:"currentStepStatus"`
	StepOptions       []OrderItemStepOption `json:"stepOptions"`
}

// OrderItemStepOption is one selectable step of the product's process.
type OrderItemStepOption struct {
	StepTemplateStepID string `json:"stepTemplateStepId"`
	Name               string `json:"name"`
	StepOrder          int    `json:"stepOrder"`
	IsDoneStep         bool   `json:"isDoneStep"`
}

// SelectOrderItemsTracking returns the items of an order with their current step.
func SelectOrderItemsTracking(companyID, orderInternalID string) ([]OrderItemTrackingRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	if companyID == "" || orderInternalID == "" {
		return nil, fmt.Errorf("companyId and orderInternalId are required")
	}

	query := `
		SELECT
			oi.order_item_id,
			oi.order_internal_id,
			oi.company_id,
			oi.product_id,
			p.name AS product_name,
			oi.internal_code,
			oi.quantity,
			oi.current_step_id,
			COALESCE(oi.current_step_status, '')
		FROM public.tbl_order_item oi
		LEFT JOIN public.tbl_products p
			ON p.product_internal_id::text = oi.product_id
		WHERE oi.order_internal_id = $1 AND oi.company_id = $2
		ORDER BY oi.item_order ASC
	`
	rows, err := db.Query(query, orderInternalID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to select order items tracking: %w", err)
	}
	defer rows.Close()

	list := make([]OrderItemTrackingRow, 0)
	for rows.Next() {
		var r OrderItemTrackingRow
		if err := rows.Scan(
			&r.OrderItemID, &r.OrderInternalID, &r.CompanyID, &r.ProductID,
			&r.ProductName, &r.InternalCode, &r.Quantity,
			&r.CurrentStepID, &r.CurrentStepStatus,
		); err != nil {
			return nil, fmt.Errorf("failed to scan order item tracking: %w", err)
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Enrich each item with the steps of its product's step_template.
	for i := range list {
		opts, err := selectStepOptionsForProduct(companyID, list[i].ProductID)
		if err != nil {
			return nil, err
		}
		list[i].StepOptions = opts
	}
	return list, nil
}

// selectStepOptionsForProduct returns the steps of the step_template assigned to
// a product (empty when the product has none).
func selectStepOptionsForProduct(companyID, productID string) ([]OrderItemStepOption, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return []OrderItemStepOption{}, nil
	}
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	query := `
		SELECT s.step_template_step_id, s.name, s.step_order, s.is_done_step
		FROM public.tbl_products p
		JOIN public.tbl_step_template_step s
			ON s.step_template_id = p.step_template_id
		WHERE p.product_internal_id::text = $1 AND p.company_id = $2
		ORDER BY s.step_order ASC
	`
	rows, err := db.Query(query, productID, strings.TrimSpace(companyID))
	if err != nil {
		return nil, fmt.Errorf("failed to select step options: %w", err)
	}
	defer rows.Close()
	out := make([]OrderItemStepOption, 0)
	for rows.Next() {
		var o OrderItemStepOption
		if err := rows.Scan(&o.StepTemplateStepID, &o.Name, &o.StepOrder, &o.IsDoneStep); err != nil {
			return nil, fmt.Errorf("failed to scan step option: %w", err)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// UpdateOrderItemStep sets one item's current step + status, scoped to company
// and order. Returns rows affected.
func UpdateOrderItemStep(companyID, orderInternalID, orderItemID string, stepID *string, stepStatus string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	result, err := db.Exec(`
		UPDATE public.tbl_order_item
		SET current_step_id = $4, current_step_status = $5
		WHERE order_item_id = $1 AND order_internal_id = $2 AND company_id = $3
	`, strings.TrimSpace(orderItemID), strings.TrimSpace(orderInternalID),
		strings.TrimSpace(companyID), stepID, strings.TrimSpace(stepStatus))
	if err != nil {
		return 0, fmt.Errorf("failed to update order item step: %w", err)
	}
	return result.RowsAffected()
}

// SetOrderStatus sets the order's status (used by the recompute).
func SetOrderStatus(companyID, orderInternalID, status string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	_, err = db.Exec(`
		UPDATE public.tbl_order SET status = $3
		WHERE order_internal_id = $1 AND company_id = $2
	`, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID), strings.TrimSpace(status))
	if err != nil {
		return fmt.Errorf("failed to set order status: %w", err)
	}
	return nil
}

// CountItemsDoneStatus returns (total items, items whose current_step is the
// product's DONE step). A step is "done" when tbl_step_template_step.is_done_step
// is TRUE for the item's current_step_id.
func CountItemsDoneStatus(companyID, orderInternalID string) (total int, started int, done int, err error) {
	db, e := dbmanager.GetPostgresDB()
	if e != nil {
		return 0, 0, 0, fmt.Errorf("database connection error: %w", e)
	}
	// An item counts as DONE when its current step is flagged is_done_step,
	// OR it is the LAST step (max step_order) of its product's process. This is
	// robust even if the process has no explicit done step configured.
	query := `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE oi.current_step_id IS NOT NULL) AS started,
			COUNT(*) FILTER (
				WHERE s.is_done_step = TRUE
				   OR (maxs.max_order IS NOT NULL AND s.step_order = maxs.max_order)
			) AS done
		FROM public.tbl_order_item oi
		LEFT JOIN public.tbl_step_template_step s
			ON s.step_template_step_id = oi.current_step_id
		LEFT JOIN (
			SELECT step_template_id, MAX(step_order) AS max_order
			FROM public.tbl_step_template_step
			GROUP BY step_template_id
		) maxs ON maxs.step_template_id = s.step_template_id
		WHERE oi.order_internal_id = $1 AND oi.company_id = $2
	`
	e = db.QueryRow(query, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID)).
		Scan(&total, &started, &done)
	if e != nil {
		return 0, 0, 0, fmt.Errorf("failed to count item statuses: %w", e)
	}
	return total, started, done, nil
}

// -------------------------------------------------------------------
//  Step history + validation support
// -------------------------------------------------------------------

// StepInfo describes one step (used for requires_previous validation).
type StepInfo struct {
	StepTemplateStepID string
	StepOrder          int
	RequiresPrevious   bool
	IsDoneStep         bool
	Name               string
	NotifyCustomer     bool // email the customer when an item reaches this step (2026-10-31)
}

// GetStepInfo loads a single step by id (company-scoped via its template owner).
func GetStepInfo(companyID, stepTemplateStepID string) (*StepInfo, error) {
	stepTemplateStepID = strings.TrimSpace(stepTemplateStepID)
	if stepTemplateStepID == "" {
		return nil, nil
	}
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	var s StepInfo
	err = db.QueryRow(`
		SELECT step_template_step_id, step_order, requires_previous, is_done_step, name, notify_customer
		FROM public.tbl_step_template_step
		WHERE step_template_step_id = $1 AND company_id = $2
	`, stepTemplateStepID, strings.TrimSpace(companyID)).
		Scan(&s.StepTemplateStepID, &s.StepOrder, &s.RequiresPrevious, &s.IsDoneStep, &s.Name, &s.NotifyCustomer)
	if err != nil {
		return nil, fmt.Errorf("failed to load step info: %w", err)
	}
	return &s, nil
}

// GetItemCurrentStepOrder returns the step_order of an item's current step, or
// (-1, nil) when the item has no current step (not started).
func GetItemCurrentStepOrder(companyID, orderInternalID, orderItemID string) (int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return -1, fmt.Errorf("database connection error: %w", err)
	}
	var stepOrder *int
	err = db.QueryRow(`
		SELECT s.step_order
		FROM public.tbl_order_item oi
		LEFT JOIN public.tbl_step_template_step s
			ON s.step_template_step_id = oi.current_step_id
		WHERE oi.order_item_id = $1 AND oi.order_internal_id = $2 AND oi.company_id = $3
	`, strings.TrimSpace(orderItemID), strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID)).
		Scan(&stepOrder)
	if err != nil {
		return -1, fmt.Errorf("failed to load current step order: %w", err)
	}
	if stepOrder == nil {
		return -1, nil
	}
	return *stepOrder, nil
}

// OrderItemStepHistoryRow is one timeline entry.
type OrderItemStepHistoryRow struct {
	HistoryID       string  `json:"historyId"`
	OrderItemID     string  `json:"orderItemId"`
	ProductName     *string `json:"productName,omitempty"`
	StepID          *string `json:"stepTemplateStepId,omitempty"`
	StepName        string  `json:"stepName"`
	StepStatus      string  `json:"stepStatus"`
	OrderStatus     string  `json:"orderStatus"`
	ChangedByUserID string  `json:"changedByUserId"`
	ChangedByEmail  string  `json:"changedByEmail"`
	ChangedAt       string  `json:"changedAt"`
}

// InsertOrderItemStepHistory records one movement.
func InsertOrderItemStepHistory(
	companyID, orderInternalID, orderItemID string,
	stepID *string, stepName, stepStatus, orderStatus, userID, email string,
) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	_, err = db.Exec(`
		INSERT INTO public.tbl_order_item_step_history
			(company_id, order_internal_id, order_item_id, step_template_step_id,
			 step_name, step_status, order_status, changed_by_user_id, changed_by_email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, strings.TrimSpace(companyID), strings.TrimSpace(orderInternalID), strings.TrimSpace(orderItemID),
		stepID, stepName, stepStatus, orderStatus, strings.TrimSpace(userID), strings.TrimSpace(email))
	if err != nil {
		return fmt.Errorf("failed to insert step history: %w", err)
	}
	return nil
}

// SelectOrderItemStepHistory returns the timeline for an order (all items),
// newest last (ascending by changed_at), with the product name joined.
func SelectOrderItemStepHistory(companyID, orderInternalID string) ([]OrderItemStepHistoryRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	query := `
		SELECT
			h.history_id,
			h.order_item_id,
			p.name AS product_name,
			h.step_template_step_id,
			COALESCE(h.step_name, ''),
			COALESCE(h.step_status, ''),
			COALESCE(h.order_status, ''),
			COALESCE(h.changed_by_user_id, ''),
			COALESCE(h.changed_by_email, ''),
			to_char(h.changed_at, 'YYYY-MM-DD"T"HH24:MI:SS')
		FROM public.tbl_order_item_step_history h
		LEFT JOIN public.tbl_order_item oi ON oi.order_item_id = h.order_item_id
		LEFT JOIN public.tbl_products p ON p.product_internal_id::text = oi.product_id
		WHERE h.order_internal_id = $1 AND h.company_id = $2
		ORDER BY h.changed_at ASC
	`
	rows, err := db.Query(query, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID))
	if err != nil {
		return nil, fmt.Errorf("failed to select step history: %w", err)
	}
	defer rows.Close()
	out := make([]OrderItemStepHistoryRow, 0)
	for rows.Next() {
		var r OrderItemStepHistoryRow
		if err := rows.Scan(
			&r.HistoryID, &r.OrderItemID, &r.ProductName, &r.StepID,
			&r.StepName, &r.StepStatus, &r.OrderStatus,
			&r.ChangedByUserID, &r.ChangedByEmail, &r.ChangedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan step history: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
