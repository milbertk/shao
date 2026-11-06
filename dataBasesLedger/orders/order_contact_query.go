package order

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// SetOrderWebRequestLink records which web-order request an order came from.
func SetOrderWebRequestLink(companyID, orderInternalID, webOrderRequestID string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	return SetOrderWebRequestLinkWith(db, companyID, orderInternalID, webOrderRequestID)
}

// SetOrderWebRequestLinkWith is SetOrderWebRequestLink inside a transaction
// (or any executor).
func SetOrderWebRequestLinkWith(db dbmanager.Executor, companyID, orderInternalID, webOrderRequestID string) error {
	_, err := db.Exec(`
		UPDATE public.tbl_order
		SET web_order_request_id = $3
		WHERE order_internal_id = $1 AND company_id = $2
	`, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID), strings.TrimSpace(webOrderRequestID))
	if err != nil {
		return fmt.Errorf("failed to link order to web request: %w", err)
	}
	return nil
}

// OrderContactRow is the contact info the customer submitted for THIS order
// (taken from the linked web-order request).
type OrderContactRow struct {
	Name                   string   `json:"name"`
	Cellphone              string   `json:"cellphone"`
	Email                  string   `json:"email"`
	PreferredContactMethod string   `json:"preferredContactMethod"`
	ContactValue           string   `json:"contactValue"`
	SocialNetwork          string   `json:"socialNetwork"`
	SocialNetworkLink      string   `json:"socialNetworkLink"`
	Address                string   `json:"address"`
	Latitude               *float64 `json:"latitude,omitempty"`
	Longitude              *float64 `json:"longitude,omitempty"`
}

// SelectOrderContact returns the contact info from the linked web-order request,
// or nil when the order has no link (e.g. created directly, not from the web).
func SelectOrderContact(companyID, orderInternalID string) (*OrderContactRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	var c OrderContactRow
	err = db.QueryRow(`
		SELECT
			COALESCE(w.customer_name, ''),
			COALESCE(w.phone, ''),
			COALESCE(w.email, ''),
			COALESCE(w.preferred_contact, ''),
			COALESCE(w.contact_value, ''),
			COALESCE(w.social_link, ''),
			COALESCE(w.address, ''),
			w.latitude,
			w.longitude
		FROM public.tbl_order o
		JOIN public.tbl_web_order_request w
			ON w.web_order_request_id = o.web_order_request_id
		WHERE o.order_internal_id = $1 AND o.company_id = $2
	`, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID)).
		Scan(&c.Name, &c.Cellphone, &c.Email, &c.PreferredContactMethod, &c.ContactValue,
			&c.SocialNetworkLink, &c.Address, &c.Latitude, &c.Longitude)
	if err == sql.ErrNoRows {
		return nil, nil // no linked web request → no contact to show
	}
	if err != nil {
		return nil, fmt.Errorf("failed to select order contact: %w", err)
	}
	return &c, nil
}
