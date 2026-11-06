package blassistant

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	blpublicshowcase "github.com/milbertk/shao/bussunessLedger/bPublicShowcase"
	"github.com/milbertk/shao/dataBasesLedger/products"
)

// =====================================================================
//  Showcase (public) tools. They ONLY read the company's public catalog
//  (active products shown on the web) and its public booking slots.
//  "Action" tools never change data: they return an action that the
//  showcase page runs on screen (add to cart, open booking, ...).
//  There is NO tool to place an order or send a booking request: the
//  visitor always confirms those with their own click.
// =====================================================================

const (
	maxShowcaseSearchRows = 20
	maxShowcaseActions    = 10
	maxCartQuantity       = 20
)

// Action types understood by the showcase frontend.
const (
	ActionAddToCart       = "add_to_cart"
	ActionSetCartQuantity = "set_cart_quantity"
	ActionOpenBooking     = "open_booking"
	ActionShowContact     = "show_contact"
	ActionOpenCart        = "open_cart"
	ActionOpenLogin       = "open_login"
)

// ShowcaseAction is one step the showcase page runs on screen.
type ShowcaseAction struct {
	Type      string `json:"type"`
	ProductID string `json:"productId,omitempty"`
	Quantity  int    `json:"quantity,omitempty"`
	Date      string `json:"date,omitempty"`
	// Product is the public item, so the page can show it even when it is
	// not in the list it loaded.
	Product *blpublicshowcase.PublicShowcaseItem `json:"product,omitempty"`
}

// showcaseToolContext: the company always comes from the public URL.
type showcaseToolContext struct {
	CompanyID string
	Now       time.Time
	actions   []ShowcaseAction
	items     map[string]*blpublicshowcase.PublicShowcaseItem // cache by id
}

func showcaseToolDefinitions() []claudeTool {
	productID := stringProp("productInternalId from search_catalog.")
	return []claudeTool{
		{
			Name: "search_catalog",
			Description: "Searches this company's public catalog. Each item has howToGet: \"cart\" (can be added " +
				"to the cart and ordered online), \"booking\" (a service booked by appointment) or \"contact\" " +
				"(only by WhatsApp/Messenger). Empty term = first items of the catalog.",
			InputSchema: objectSchema(map[string]any{
				"term":       stringProp("Optional text: name, code or description."),
				"how_to_get": enumProp("Optional filter.", "cart", "booking", "contact"),
			}),
		},
		{
			Name:        "get_available_slots",
			Description: "Free booking slots of ONE booking service (howToGet = booking) on ONE date.",
			InputSchema: objectSchema(map[string]any{
				"product_id": productID,
				"date":       stringProp("Date YYYY-MM-DD."),
			}, "product_id", "date"),
		},
		{
			Name: "add_to_cart",
			Description: "Adds a product (howToGet = cart) to the visitor's cart on screen. It does NOT place the " +
				"order: the visitor reviews the cart and confirms the order.",
			InputSchema: objectSchema(map[string]any{
				"product_id": productID,
				"quantity":   intProp("Units to add (1-20). Default 1."),
			}, "product_id"),
		},
		{
			Name:        "set_cart_quantity",
			Description: "Sets the quantity of a product already in the cart. 0 removes it from the cart.",
			InputSchema: objectSchema(map[string]any{
				"product_id": productID,
				"quantity":   intProp("New quantity (0-20)."),
			}, "product_id", "quantity"),
		},
		{
			Name: "open_booking",
			Description: "Opens the booking form of a booking service (howToGet = booking), optionally on a date. " +
				"The visitor completes the form and sends the request.",
			InputSchema: objectSchema(map[string]any{
				"product_id": productID,
				"date":       stringProp("Optional date YYYY-MM-DD."),
			}, "product_id"),
		},
		{
			Name:        "show_contact",
			Description: "Shows the WhatsApp / Messenger buttons of a product (for howToGet = contact, or questions).",
			InputSchema: objectSchema(map[string]any{
				"product_id": productID,
			}, "product_id"),
		},
		{
			Name:        "open_cart",
			Description: "Opens the cart so the visitor can review it and go to checkout (as guest or with an account).",
			InputSchema: objectSchema(map[string]any{}),
		},
		{
			Name:        "open_login",
			Description: "Opens the sign in / create account window.",
			InputSchema: objectSchema(map[string]any{}),
		},
	}
}

type showcaseCatalogRow struct {
	ProductID    string   `json:"productId"`
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Price        *float64 `json:"price,omitempty"`
	Currency     string   `json:"currencySymbol,omitempty"`
	HowToGet     string   `json:"howToGet"`
	HasWhatsapp  bool     `json:"hasWhatsapp"`
	HasMessenger bool     `json:"hasMessenger"`
}

// runShowcaseTool executes one showcase tool.
func runShowcaseTool(tc *showcaseToolContext, name string, input json.RawMessage) (string, bool) {
	if len(input) == 0 {
		input = json.RawMessage(`{}`)
	}

	var (
		result any
		err    error
	)
	switch name {
	case "search_catalog":
		result, err = showcaseSearch(tc, input)
	case "get_available_slots":
		result, err = showcaseSlots(tc, input)
	case "add_to_cart":
		result, err = showcaseAddToCart(tc, input)
	case "set_cart_quantity":
		result, err = showcaseSetQuantity(tc, input)
	case "open_booking":
		result, err = showcaseOpenBooking(tc, input)
	case "show_contact":
		result, err = showcaseShowContact(tc, input)
	case "open_cart":
		result, err = tc.addAction(ShowcaseAction{Type: ActionOpenCart}, "The cart is open on the screen.")
	case "open_login":
		result, err = tc.addAction(ShowcaseAction{Type: ActionOpenLogin}, "The sign in window is open on the screen.")
	default:
		return "Unknown tool: " + name, true
	}

	if err != nil {
		if userErr, ok := err.(toolInputError); ok {
			return userErr.Error(), true
		}
		log.Printf("[blassistant] Showcase tool failed: tool=%s companyId=%s error=%v", name, tc.CompanyID, err)
		return "The information could not be loaded because of an internal error.", true
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return "The information could not be prepared.", true
	}
	if len(encoded) > maxToolResultChars {
		return "Too many results. Ask the visitor for a more specific name.", true
	}
	return string(encoded), false
}

func showcaseSearch(tc *showcaseToolContext, input json.RawMessage) (any, error) {
	var args struct {
		Term     string `json:"term"`
		HowToGet string `json:"how_to_get"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}

	items, err := blpublicshowcase.BSearchShowcase(tc.CompanyID, products.PublicProductSearchFilters{
		Term:  truncateRunes(strings.TrimSpace(args.Term), 80),
		Limit: 60,
	})
	if err != nil {
		return nil, err
	}

	filter := strings.ToLower(strings.TrimSpace(args.HowToGet))
	rows := make([]showcaseCatalogRow, 0, maxShowcaseSearchRows)
	for i := range items {
		item := items[i]
		tc.remember(&item)
		row := toCatalogRow(item)
		if filter != "" && row.HowToGet != filter {
			continue
		}
		if len(rows) < maxShowcaseSearchRows {
			rows = append(rows, row)
		}
	}
	return map[string]any{"count": len(rows), "items": rows, "limitedTo": maxShowcaseSearchRows}, nil
}

func showcaseSlots(tc *showcaseToolContext, input json.RawMessage) (any, error) {
	var args struct {
		ProductID string `json:"product_id"`
		Date      string `json:"date"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	if err := validateToolDate("date", args.Date, true); err != nil {
		return nil, err
	}
	item, err := tc.findItem(args.ProductID)
	if err != nil {
		return nil, err
	}
	if howToGet(*item) != "booking" {
		return nil, toolInputError(item.Name + " is not booked by appointment (howToGet = " + howToGet(*item) + ")")
	}

	day, err := blpublicshowcase.BGetAvailabilityForDate(tc.CompanyID, item.ProductInternalID, args.Date)
	if err != nil {
		return nil, err
	}
	return map[string]any{"service": item.Name, "availability": day}, nil
}

func showcaseAddToCart(tc *showcaseToolContext, input json.RawMessage) (any, error) {
	var args struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	quantity := args.Quantity
	if quantity == 0 {
		quantity = 1
	}
	if quantity < 1 || quantity > maxCartQuantity {
		return nil, toolInputError("quantity must be between 1 and 20")
	}
	item, err := tc.findItem(args.ProductID)
	if err != nil {
		return nil, err
	}
	switch howToGet(*item) {
	case "booking":
		return nil, toolInputError(item.Name + " is booked by appointment: use open_booking instead of the cart")
	case "contact":
		return nil, toolInputError(item.Name + " is only available by contact (WhatsApp/Messenger): use show_contact")
	}

	return tc.addAction(ShowcaseAction{
		Type:      ActionAddToCart,
		ProductID: item.ProductInternalID,
		Quantity:  quantity,
		Product:   item,
	}, "Added to the cart on the screen. The order is NOT placed: the visitor must review the cart and confirm it.")
}

func showcaseSetQuantity(tc *showcaseToolContext, input json.RawMessage) (any, error) {
	var args struct {
		ProductID string `json:"product_id"`
		Quantity  *int   `json:"quantity"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	if args.Quantity == nil || *args.Quantity < 0 || *args.Quantity > maxCartQuantity {
		return nil, toolInputError("quantity must be between 0 and 20")
	}
	item, err := tc.findItem(args.ProductID)
	if err != nil {
		return nil, err
	}
	if howToGet(*item) != "cart" {
		return nil, toolInputError(item.Name + " cannot be in the cart")
	}

	return tc.addAction(ShowcaseAction{
		Type:      ActionSetCartQuantity,
		ProductID: item.ProductInternalID,
		Quantity:  *args.Quantity,
		Product:   item,
	}, "Cart updated on the screen.")
}

func showcaseOpenBooking(tc *showcaseToolContext, input json.RawMessage) (any, error) {
	var args struct {
		ProductID string `json:"product_id"`
		Date      string `json:"date"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	if err := validateToolDate("date", args.Date, false); err != nil {
		return nil, err
	}
	item, err := tc.findItem(args.ProductID)
	if err != nil {
		return nil, err
	}
	if howToGet(*item) != "booking" {
		return nil, toolInputError(item.Name + " is not booked by appointment (howToGet = " + howToGet(*item) + ")")
	}

	return tc.addAction(ShowcaseAction{
		Type:      ActionOpenBooking,
		ProductID: item.ProductInternalID,
		Date:      strings.TrimSpace(args.Date),
		Product:   item,
	}, "The booking form is open on the screen. The visitor chooses the time, fills in the form and sends the request.")
}

func showcaseShowContact(tc *showcaseToolContext, input json.RawMessage) (any, error) {
	var args struct {
		ProductID string `json:"product_id"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	item, err := tc.findItem(args.ProductID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(item.WebContactPhone) == "" && strings.TrimSpace(item.WebMessengerLink) == "" {
		return nil, toolInputError(item.Name + " has no WhatsApp or Messenger contact configured")
	}

	return tc.addAction(ShowcaseAction{
		Type:      ActionShowContact,
		ProductID: item.ProductInternalID,
		Product:   item,
	}, "The contact buttons (WhatsApp / Messenger) are shown in the chat.")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (tc *showcaseToolContext) addAction(action ShowcaseAction, message string) (any, error) {
	if len(tc.actions) >= maxShowcaseActions {
		return nil, toolInputError("too many actions in one answer")
	}
	tc.actions = append(tc.actions, action)
	return map[string]any{"ok": true, "message": message}, nil
}

func (tc *showcaseToolContext) remember(item *blpublicshowcase.PublicShowcaseItem) {
	if tc.items == nil {
		tc.items = map[string]*blpublicshowcase.PublicShowcaseItem{}
	}
	copied := *item
	tc.items[item.ProductInternalID] = &copied
}

// findItem loads ONE public item of this company (active and shown on the web).
func (tc *showcaseToolContext) findItem(productID string) (*blpublicshowcase.PublicShowcaseItem, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, toolInputError("product_id is required: use search_catalog first")
	}
	if item, ok := tc.items[productID]; ok {
		return item, nil
	}

	items, err := blpublicshowcase.BSearchShowcase(tc.CompanyID, products.PublicProductSearchFilters{
		ProductID: productID,
		Limit:     1,
	})
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax") {
			return nil, toolInputError("product not found in this catalog: use search_catalog")
		}
		return nil, err
	}
	if len(items) == 0 {
		return nil, toolInputError("product not found in this catalog: use search_catalog")
	}
	tc.remember(&items[0])
	return tc.items[productID], nil
}

// howToGet mirrors the showcase frontend rules (showcaseMappers.js).
func howToGet(item blpublicshowcase.PublicShowcaseItem) string {
	kind := strings.ToUpper(strings.TrimSpace(item.Kind))
	behavior := strings.ToUpper(strings.TrimSpace(item.WebBehavior))
	switch {
	case kind == "BOOKING" || behavior == "BOOKING":
		return "booking"
	case kind == "PRODUCT":
		return "cart"
	default:
		return "contact"
	}
}

func toCatalogRow(item blpublicshowcase.PublicShowcaseItem) showcaseCatalogRow {
	row := showcaseCatalogRow{
		ProductID:    item.ProductInternalID,
		Code:         item.ProductCode,
		Name:         item.Name,
		Description:  truncateRunes(item.Description, 160),
		HowToGet:     howToGet(item),
		HasWhatsapp:  strings.TrimSpace(item.WebContactPhone) != "",
		HasMessenger: strings.TrimSpace(item.WebMessengerLink) != "",
	}
	if item.HasPrice {
		row.Price = item.BasePrice
		row.Currency = item.CurrencySymbol
	}
	return row
}
