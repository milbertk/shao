package notifications

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Recipient lookups (2026-10-30): when an appointment / order has no email
// typed on it, the customer email can come from the linked customer
// (tbl_customers), the store request it came from (tbl_web_order_request)
// or the registered user (usersAK.usfirebasedata). "" = not found.

// CustomerEmail returns the email of a company customer, found by its
// internal id or by its customer id.
func CustomerEmail(companyID, customerRef string) (string, error) {
	customerRef = strings.TrimSpace(customerRef)
	if customerRef == "" {
		return "", nil
	}
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", fmt.Errorf("database connection error: %w", err)
	}
	var email sql.NullString
	err = db.QueryRow(`
		SELECT email FROM public.tbl_customers
		WHERE company_id = $1 AND (customer_internal_id::text = $2 OR customer_id::text = $2)
		  AND email IS NOT NULL AND btrim(email) <> ''
		LIMIT 1`,
		strings.TrimSpace(companyID), customerRef).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read customer email: %w", err)
	}
	return strings.TrimSpace(email.String), nil
}

// OrderWebRequestContact returns the email and the registered user (user
// code) of the store request an order was created from ("" when none).
func OrderWebRequestContact(companyID, orderInternalID string) (string, string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", "", fmt.Errorf("database connection error: %w", err)
	}
	var email, userCode sql.NullString
	err = db.QueryRow(`
		SELECT w.email, w.user_code
		FROM public.tbl_order o
		JOIN public.tbl_web_order_request w ON w.web_order_request_id::text = o.web_order_request_id::text
		WHERE o.order_internal_id::text = $1 AND o.company_id = $2
		LIMIT 1`,
		strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID)).Scan(&email, &userCode)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to read the order's store request: %w", err)
	}
	return strings.TrimSpace(email.String), strings.TrimSpace(userCode.String), nil
}

// RegisteredUserEmail returns the email of a registered user (Firebase uid)
// from the usersAK database.
func RegisteredUserEmail(uid string) (string, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "", nil
	}
	db, err := dbmanager.GetUsersDB()
	if err != nil {
		return "", fmt.Errorf("usersAK connection error: %w", err)
	}
	var email sql.NullString
	err = db.QueryRow(`SELECT email FROM public.usfirebasedata WHERE uid = $1 LIMIT 1`, uid).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read registered user email: %w", err)
	}
	return strings.TrimSpace(email.String), nil
}
