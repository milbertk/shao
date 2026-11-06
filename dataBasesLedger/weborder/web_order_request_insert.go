package weborder

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// WebOrderRequestS is the header of a web order request (intermediate table).
type WebOrderRequestS struct {
	WebOrderRequestID     string    `json:"webOrderRequestId"`
	CompanyID             string    `json:"companyId"`
	RequestCode           string    `json:"requestCode"`
	OrderType             string    `json:"orderType"`
	UseRegisteredCustomer bool      `json:"useRegisteredCustomer"`
	CustomerID            *string   `json:"customerId,omitempty"`
	CustomerName          string    `json:"customerName"`
	Phone                 *string   `json:"phone,omitempty"`
	Email                 *string   `json:"email,omitempty"`
	SocialLink            *string   `json:"socialLink,omitempty"`
	PreferredContact      *string   `json:"preferredContact,omitempty"`
	ContactValue          *string   `json:"contactValue,omitempty"`
	Region                *string   `json:"region,omitempty"`
	Address               *string   `json:"address,omitempty"`
	DeliveryMode          string    `json:"deliveryMode"`
	Latitude              *float64  `json:"latitude,omitempty"`
	Longitude             *float64  `json:"longitude,omitempty"`
	Notes                 *string   `json:"notes,omitempty"`
	Total                 float64   `json:"total"`
	RequesterType         string    `json:"requesterType"`
	UserCode              *string   `json:"userCode,omitempty"`
	Status                string    `json:"status"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

// WebOrderRequestItemS is one cart item of a web order request.
type WebOrderRequestItemS struct {
	WebOrderRequestItemID string    `json:"webOrderRequestItemId"`
	WebOrderRequestID     string    `json:"webOrderRequestId"`
	CompanyID             string    `json:"companyId"`
	ProductID             string    `json:"productId"`
	ProductCode           *string   `json:"productCode,omitempty"`
	ProductName           *string   `json:"productName,omitempty"`
	Quantity              int       `json:"quantity"`
	UnitPrice             float64   `json:"unitPrice"`
	ItemOrder             int       `json:"itemOrder"`
	CreatedAt             time.Time `json:"createdAt"`
}

// Insert creates the header. The request_code (WEB-XXXX) is generated from a
// sequence if a.RequestCode is empty.
func (a *WebOrderRequestS) InsertWith(db dbmanager.Executor) error {
	var err error

	// Generate WEB-XXXX from the sequence when not provided.
	if a.RequestCode == "" {
		var seq int64
		if err := db.QueryRow(
			`SELECT nextval('public.seq_web_order_request_code')`,
		).Scan(&seq); err != nil {
			return fmt.Errorf("failed to generate request code: %w", err)
		}
		a.RequestCode = fmt.Sprintf("WEB-%d", seq)
	}

	query := `
		INSERT INTO public.tbl_web_order_request (
			company_id,
			request_code,
			order_type,
			use_registered_customer,
			customer_id,
			customer_name,
			phone,
			email,
			social_link,
			preferred_contact,
			contact_value,
			region,
			address,
			delivery_mode,
			latitude,
			longitude,
			notes,
			total,
			status,
			requester_type,
			user_code
		)
		VALUES (
			$1, $2,
			COALESCE(NULLIF($3, ''), 'PRODUCT'),
			$4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			COALESCE(NULLIF($14, ''), 'DELIVERY'),
			$15, $16, $17, $18,
			COALESCE(NULLIF($19, ''), 'PENDING'),
			COALESCE(NULLIF($20, ''), 'GUEST'),
			$21
		)
		RETURNING web_order_request_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.CompanyID,
		a.RequestCode,
		a.OrderType,
		a.UseRegisteredCustomer,
		a.CustomerID,
		a.CustomerName,
		a.Phone,
		a.Email,
		a.SocialLink,
		a.PreferredContact,
		a.ContactValue,
		a.Region,
		a.Address,
		a.DeliveryMode,
		a.Latitude,
		a.Longitude,
		a.Notes,
		a.Total,
		a.Status,
		a.RequesterType,
		a.UserCode,
	).Scan(
		&a.WebOrderRequestID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert web order request: %w", err)
	}

	return nil
}

// Insert creates the header (own connection, no transaction).
func (a *WebOrderRequestS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	return a.InsertWith(db)
}

// Insert creates one item for a web order request.
func (a *WebOrderRequestItemS) InsertWith(db dbmanager.Executor) error {
	var err error

	if a.Quantity < 1 {
		a.Quantity = 1
	}
	if a.ItemOrder < 1 {
		a.ItemOrder = 1
	}

	query := `
		INSERT INTO public.tbl_web_order_request_item (
			web_order_request_id,
			company_id,
			product_id,
			product_code,
			product_name,
			quantity,
			unit_price,
			item_order
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING web_order_request_item_id, created_at
	`

	err = db.QueryRow(
		query,
		a.WebOrderRequestID,
		a.CompanyID,
		a.ProductID,
		a.ProductCode,
		a.ProductName,
		a.Quantity,
		a.UnitPrice,
		a.ItemOrder,
	).Scan(
		&a.WebOrderRequestItemID,
		&a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert web order request item: %w", err)
	}

	return nil
}

// Insert creates one item (own connection, no transaction).
func (a *WebOrderRequestItemS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	return a.InsertWith(db)
}
