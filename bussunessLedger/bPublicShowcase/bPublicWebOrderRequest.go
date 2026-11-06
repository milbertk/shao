package blpublicshowcase

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	blorder "github.com/milbertk/shao/bussunessLedger/blorder"
	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
)

// PublicWebOrderItem is one cart item sent from the showcase.
type PublicWebOrderItem struct {
	ProductID   string  `json:"productId"`
	ProductCode string  `json:"productCode"`
	ProductName string  `json:"productName"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
}

// PublicWebOrderRequest is the guest cart submission from the showcase.
type PublicWebOrderRequest struct {
	OrderType             string               `json:"orderType"`
	UseRegisteredCustomer bool                 `json:"useRegisteredCustomer"`
	CustomerID            string               `json:"customerId"`
	CustomerName          string               `json:"customerName"`
	Phone                 string               `json:"phone"`
	Email                 string               `json:"email"`
	SocialLink            string               `json:"socialLink"`
	PreferredContact      string               `json:"preferredContact"`
	ContactValue          string               `json:"contactValue"`
	Region                string               `json:"region"`
	Address               string               `json:"address"`
	DeliveryMode          string               `json:"deliveryMode"`
	Latitude              *float64             `json:"latitude"`
	Longitude             *float64             `json:"longitude"`
	Notes                 string               `json:"notes"`
	Items                 []PublicWebOrderItem `json:"items"`
}

func normalizeDeliveryMode(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	if v == "PICKUP" {
		return "PICKUP"
	}
	return "DELIVERY"
}

func s2nil(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

// BCreatePublicWebOrderRequest saves a guest web order (header + items) as
// PENDING and sends a best-effort notification email. Scoped by company.
func BCreatePublicWebOrderRequest(companyID string, userCode string, req PublicWebOrderRequest) (*weborder.WebOrderRequestS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if strings.TrimSpace(req.CustomerName) == "" {
		return nil, errors.New("customerName is required")
	}
	if len(req.Items) == 0 {
		return nil, errors.New("at least one item is required")
	}
	// Appointments are booked, never ordered (2026-10-31).
	productIDs := make([]string, 0, len(req.Items))
	for _, it := range req.Items {
		productIDs = append(productIDs, it.ProductID)
	}
	if err := blorder.BCheckNoAppointments(companyID, productIDs); err != nil {
		return nil, err
	}
	// Formats and sizes checked on the server (2026-10-25, security S4).
	if err := ValidatePublicWebOrderRequest(req); err != nil {
		return nil, err
	}
	// Delivery requires an address or map coordinates; pickup does not.
	if normalizeDeliveryMode(req.DeliveryMode) == "DELIVERY" {
		hasAddress := strings.TrimSpace(req.Address) != ""
		hasCoords := req.Latitude != nil && req.Longitude != nil
		if !hasAddress && !hasCoords {
			return nil, errors.New("delivery requires an address or a map location")
		}
	}

	// Recompute the total on the server.
	var total float64
	for _, it := range req.Items {
		qty := it.Quantity
		if qty < 1 {
			qty = 1
		}
		total += float64(qty) * it.UnitPrice
	}

	header := weborder.WebOrderRequestS{
		CompanyID:             companyID,
		OrderType:             strings.ToUpper(strings.TrimSpace(req.OrderType)),
		UseRegisteredCustomer: req.UseRegisteredCustomer,
		CustomerID:            s2nil(req.CustomerID),
		CustomerName:          strings.TrimSpace(req.CustomerName),
		Phone:                 s2nil(req.Phone),
		Email:                 s2nil(req.Email),
		SocialLink:            s2nil(req.SocialLink),
		PreferredContact:      s2nil(req.PreferredContact),
		ContactValue:          s2nil(req.ContactValue),
		Region:                s2nil(req.Region),
		Address:               s2nil(req.Address),
		DeliveryMode:          normalizeDeliveryMode(req.DeliveryMode),
		Latitude:              req.Latitude,
		Longitude:             req.Longitude,
		Notes:                 s2nil(req.Notes),
		Total:                 total,
		Status:                "PENDING",
	}

	// Requester reference: platform user (from the session) vs guest.
	userCode = strings.TrimSpace(userCode)
	if userCode != "" {
		header.RequesterType = "PLATFORM_USER"
		header.UserCode = &userCode
		header.UseRegisteredCustomer = true
	} else {
		header.RequesterType = "GUEST"
		header.UserCode = nil
	}

	// Header + items in ONE transaction (2026-10-24): no request without items.
	err := dbmanager.WithTx(func(tx *sql.Tx) error {
		if err := header.InsertWith(tx); err != nil {
			return fmt.Errorf("failed to create web order request: %w", err)
		}
		return insertPublicWebOrderItems(tx, companyID, header.WebOrderRequestID, req.Items)
	})
	if err != nil {
		return nil, err
	}

	// Customer "we received it" + company "new web order" (background,
	// company SMTP, company language).
	blnotifications.NotifyWebOrder(companyID, header.WebOrderRequestID, blnotifications.EventCreated)

	return &header, nil
}

// insertPublicWebOrderItems saves the cart lines inside the transaction.
func insertPublicWebOrderItems(tx *sql.Tx, companyID, webOrderRequestID string, items []PublicWebOrderItem) error {
	for index, it := range items {
		item := weborder.WebOrderRequestItemS{
			WebOrderRequestID: webOrderRequestID,
			CompanyID:         companyID,
			ProductID:         strings.TrimSpace(it.ProductID),
			ProductCode:       s2nil(it.ProductCode),
			ProductName:       s2nil(it.ProductName),
			Quantity:          it.Quantity,
			UnitPrice:         it.UnitPrice,
			ItemOrder:         index + 1,
		}
		if item.ProductID == "" {
			continue // skip invalid line
		}
		if err := item.InsertWith(tx); err != nil {
			return fmt.Errorf("failed to insert web order item: %w", err)
		}
	}
	return nil
}
