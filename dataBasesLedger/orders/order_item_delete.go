package order

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// DeleteItemsByOrder removes ALL items of an order, scoped to the company.
// Used by the "rewrite items" flow (delete all + re-insert on every save).
func DeleteItemsByOrderWith(db dbmanager.Executor, companyID, orderInternalID string) (int64, error) {
	var err error

	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	if companyID == "" {
		return 0, fmt.Errorf("companyId is required")
	}
	if orderInternalID == "" {
		return 0, fmt.Errorf("orderInternalId is required")
	}

	result, err := db.Exec(`
		DELETE FROM public.tbl_order_item
		WHERE order_internal_id = $1
		  AND company_id = $2
	`, orderInternalID, companyID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete order items: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted order items: %w", err)
	}

	return deleted, nil
}

// DeleteItemsByOrder removes ALL items of an order (own connection).
func DeleteItemsByOrder(companyID, orderInternalID string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	return DeleteItemsByOrderWith(db, companyID, orderInternalID)
}
