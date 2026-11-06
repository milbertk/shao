package blassistant

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	blpublicshowcase "github.com/milbertk/shao/bussunessLedger/bPublicShowcase"
	blappointment "github.com/milbertk/shao/bussunessLedger/blappontment"
	"github.com/milbertk/shao/bussunessLedger/blorder"
	"github.com/milbertk/shao/bussunessLedger/blorderitemtracking"
	"github.com/milbertk/shao/bussunessLedger/blweborder"
	blproducts "github.com/milbertk/shao/bussunessLedger/products"
	usersbycompany "github.com/milbertk/shao/bussunessLedger/usersByCompany"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
	"github.com/milbertk/shao/dataBasesLedger/products"
	usersinplatform "github.com/milbertk/shao/dataBasesLedger/usersInPlatform"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
)

// ---- Tool configuration (adjust here) ----
const (
	maxToolRows        = 30    // rows returned to Claude per tool call
	maxToolResultChars = 24000 // protects the prompt size (and the cost)
	toolDateLayout     = "2006-01-02"
)

// Permission code per tool (same NAV_* codes used by RequireOptionPermission).
// Empty = any active company member, exactly like the current routes. When a
// route gets its real NAV_* code, put the same code here.
var toolPermissions = map[string]string{
	"search_products_and_services": "",
	"get_available_slots":          "",
	"search_appointments":          "",
	"search_web_orders":            "",
	"search_orders":                "",
	"get_order_tracking":           "",
	// Insight tools (bAssistantInsightTools.go).
	"get_day_summary":         "",
	"get_orders_due_soon":     "",
	"rank_orders":             "",
	"get_aging_orders":        "",
	"get_orders_report":       "",
	"get_appointments_report": "",
	"get_top_customers":       "",
	"get_best_sellers":        "",
	"get_revenue":             "",
	"get_cancellation_rate":   "",
}

// fullAccessToolRoles always pass the tool permission check (same rule as
// middlewares.RequireOptionPermission).
var fullAccessToolRoles = map[string]struct{}{
	"OWNER":   {},
	"MANAGER": {},
}

// toolContext carries the AUTHORITATIVE identity of the request. Tools never
// take the company or the user from Claude's input.
type toolContext struct {
	CompanyID    string
	UserID       string
	Role         string
	Now          time.Time         // request time (local), for "today"
	serviceNames map[string]string // lazy cache: product id -> name
}

// assistantToolDefinitions returns the tools offered to Claude.
func assistantToolDefinitions() []claudeTool {
	tools := []claudeTool{
		{
			Name: "search_products_and_services",
			Description: "Lists the company's products and services (id, code, name, type, status). " +
				"Use it to find the product_id of a service before checking availability, " +
				"or when the user asks what the company sells or offers.",
			InputSchema: objectSchema(map[string]any{
				"term": stringProp("Optional text to search in code, name or description."),
			}),
		},
		{
			Name: "get_available_slots",
			Description: "Returns the free booking slots of ONE service on ONE date, with remaining capacity, " +
				"or the reason why the date is not available. Requires the product_id from " +
				"search_products_and_services. Call it once per date (you may call it several times).",
			InputSchema: objectSchema(map[string]any{
				"product_id": stringProp("productInternalId of the service."),
				"date":       stringProp("Date in YYYY-MM-DD format."),
			}, "product_id", "date"),
		},
		{
			Name: "search_appointments",
			Description: "Searches the company's appointments (date, hours, customer, service, status). " +
				"For one day use the same date in date_from and date_to. Status values: PENDING (a booking " +
				"request from the public page waiting for approval), SCHEDULED, CONFIRMED, DONE, CANCELLED, " +
				"REJECTED. Use status PENDING to list booking requests.",
			InputSchema: objectSchema(map[string]any{
				"date_from":     stringProp("Optional start date YYYY-MM-DD (inclusive)."),
				"date_to":       stringProp("Optional end date YYYY-MM-DD (inclusive)."),
				"status":        stringProp("Optional status filter."),
				"customer_name": stringProp("Optional part of the customer name."),
			}),
		},
		{
			Name: "search_web_orders",
			Description: "Searches web order requests that arrived from the public page (request code, customer, " +
				"total, status, first item). Status values: PENDING, APPROVED, REJECTED. " +
				"Use status PENDING for orders waiting for approval.",
			InputSchema: objectSchema(map[string]any{
				"status":    stringProp("Optional: PENDING, APPROVED or REJECTED."),
				"term":      stringProp("Optional text: request code, customer name or email."),
				"date_from": stringProp("Optional start date YYYY-MM-DD."),
				"date_to":   stringProp("Optional end date YYYY-MM-DD."),
			}),
		},
		{
			Name: "search_orders",
			Description: "Searches the company's orders (order code, customer, channel, dates, total, status). " +
				"Status values: PLACED, IN_PROGRESS, PARTIAL_DONE, DONE, CANCELLED.",
			InputSchema: objectSchema(map[string]any{
				"term":      stringProp("Optional text: order code or customer name."),
				"status":    stringProp("Optional status filter."),
				"date_from": stringProp("Optional start date YYYY-MM-DD."),
				"date_to":   stringProp("Optional end date YYYY-MM-DD."),
			}),
		},
		{
			Name: "get_order_tracking",
			Description: "Returns the tracking of ONE order: order status, each item with its current step, " +
				"and the latest step changes. Pass order_code (as the user knows it) or order_internal_id.",
			InputSchema: objectSchema(map[string]any{
				"order_code":        stringProp("Order code shown to users."),
				"order_internal_id": stringProp("orderInternalId from search_orders."),
			}),
		},
	}
	return append(tools, insightToolDefinitions()...)
}

// runAssistantTool executes one tool and returns the text for the tool_result.
// Errors are returned as is_error results so Claude can explain them.
func runAssistantTool(tc *toolContext, name string, input json.RawMessage) (string, bool) {
	code, known := toolPermissions[name]
	if !known {
		return "Unknown tool: " + name, true
	}

	allowed, err := canUseAssistantTool(tc, code)
	if err != nil {
		log.Printf("[blassistant] Tool permission check failed: tool=%s error=%v", name, err)
		return "Could not validate the user's permission for this information.", true
	}
	if !allowed {
		return "The user does not have permission to see this information.", true
	}

	if len(input) == 0 {
		input = json.RawMessage(`{}`)
	}

	var result any
	switch name {
	case "search_products_and_services":
		result, err = toolSearchProducts(tc, input)
	case "get_available_slots":
		result, err = toolAvailableSlots(tc, input)
	case "search_appointments":
		result, err = toolSearchAppointments(tc, input)
	case "search_web_orders":
		result, err = toolSearchWebOrders(tc, input)
	case "search_orders":
		result, err = toolSearchOrders(tc, input)
	case "get_order_tracking":
		result, err = toolOrderTracking(tc, input)
	default:
		var handled bool
		result, handled, err = runInsightTool(tc, name, input)
		if !handled {
			return "Unknown tool: " + name, true
		}
	}

	if err != nil {
		var userErr toolInputError
		if errors.As(err, &userErr) {
			return userErr.Error(), true
		}
		log.Printf("[blassistant] Tool failed: tool=%s companyId=%s error=%v", name, tc.CompanyID, err)
		return "The information could not be loaded because of an internal error.", true
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return "The information could not be prepared.", true
	}
	if len(encoded) > maxToolResultChars {
		return "Too many results. Ask the user for more specific filters (dates, status or a name).", true
	}

	return string(encoded), false
}

// canUseAssistantTool mirrors middlewares.RequireOptionPermission.
func canUseAssistantTool(tc *toolContext, optionCode string) (bool, error) {
	optionCode = strings.TrimSpace(optionCode)
	if optionCode == "" {
		return true, nil
	}

	role := strings.ToUpper(strings.TrimSpace(tc.Role))
	if _, ok := fullAccessToolRoles[role]; ok {
		return true, nil
	}

	users, err := usersinplatform.SelectCompanyUsersByCompanyID(tc.CompanyID)
	if err != nil {
		return false, err
	}
	companyUserID := ""
	for _, user := range users {
		if strings.TrimSpace(user.UserID) == tc.UserID {
			companyUserID = strings.TrimSpace(user.CompanyUserID)
			break
		}
	}
	if companyUserID == "" {
		return false, nil
	}

	return usersbycompany.BValidateCompanyUserPermission(companyUserID, optionCode)
}

// ---------------------------------------------------------------------------
// Tools
// ---------------------------------------------------------------------------

type toolProductRow struct {
	ProductID   string   `json:"productId"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Status      string   `json:"status"`
	BasePrice   *float64 `json:"basePrice,omitempty"`
	ShowOnWeb   bool     `json:"showOnWeb"`
	Description string   `json:"description,omitempty"`
}

func toolSearchProducts(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		Term string `json:"term"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}

	list, err := blproducts.BSearchProducts(tc.CompanyID, products.ProductSearchFilters{
		Term:  strings.TrimSpace(args.Term),
		Limit: maxToolRows,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]toolProductRow, 0, len(list))
	for _, p := range list {
		rows = append(rows, toolProductRow{
			ProductID:   p.ProductInternalID,
			Code:        p.ProductCode,
			Name:        p.Name,
			Type:        p.ProductType,
			Status:      p.Status,
			BasePrice:   p.BasePrice,
			ShowOnWeb:   p.ShowOnWeb,
			Description: truncateRunes(derefString(p.Description), 120),
		})
	}
	return map[string]any{"count": len(rows), "items": rows, "limitedTo": maxToolRows}, nil
}

func toolAvailableSlots(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		ProductID string `json:"product_id"`
		Date      string `json:"date"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	if strings.TrimSpace(args.ProductID) == "" {
		return nil, toolInputError("product_id is required: call search_products_and_services first")
	}
	if err := validateToolDate("date", args.Date, true); err != nil {
		return nil, err
	}

	day, err := blpublicshowcase.BGetAvailabilityForDate(tc.CompanyID, args.ProductID, args.Date)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"service":      tc.serviceName(args.ProductID),
		"availability": day,
	}, nil
}

type toolAppointmentRow struct {
	Date         string `json:"date"`
	Start        string `json:"start"`
	End          string `json:"end"`
	CustomerName string `json:"customerName"`
	Service      string `json:"service"`
	Status       string `json:"status"`
	Comments     string `json:"comments,omitempty"`
}

func toolSearchAppointments(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		DateFrom     string `json:"date_from"`
		DateTo       string `json:"date_to"`
		Status       string `json:"status"`
		CustomerName string `json:"customer_name"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	if err := validateToolDate("date_from", args.DateFrom, false); err != nil {
		return nil, err
	}
	if err := validateToolDate("date_to", args.DateTo, false); err != nil {
		return nil, err
	}

	list, err := blappointment.BSearchAppointments(tc.CompanyID, appointment.AppointmentSearchFilters{
		DateFrom:     args.DateFrom,
		DateTo:       args.DateTo,
		Status:       args.Status,
		CustomerName: args.CustomerName,
		Limit:        maxToolRows,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]toolAppointmentRow, 0, len(list))
	for _, a := range list {
		rows = append(rows, toolAppointmentRow{
			Date:         a.AppointmentDate,
			Start:        a.StartHour,
			End:          a.EndHour,
			CustomerName: a.CustomerName,
			Service:      tc.serviceName(a.ServiceID),
			Status:       a.Status,
			Comments:     truncateRunes(derefString(a.Comments), 120),
		})
	}
	return map[string]any{"count": len(rows), "appointments": rows, "limitedTo": maxToolRows}, nil
}

type toolWebOrderRow struct {
	RequestCode   string  `json:"requestCode"`
	OrderType     string  `json:"orderType"`
	CustomerName  string  `json:"customerName"`
	DeliveryMode  string  `json:"deliveryMode"`
	Total         float64 `json:"total"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"createdAt"`
	FirstItem     string  `json:"firstItem,omitempty"`
	ItemCount     int     `json:"itemCount"`
	TotalQuantity int     `json:"totalQuantity"`
}

func toolSearchWebOrders(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		Status   string `json:"status"`
		Term     string `json:"term"`
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	if err := validateToolDate("date_from", args.DateFrom, false); err != nil {
		return nil, err
	}
	if err := validateToolDate("date_to", args.DateTo, false); err != nil {
		return nil, err
	}

	list, err := blweborder.BSearchWebOrders(tc.CompanyID, weborder.WebOrderSearchFilters{
		Status:   args.Status,
		Term:     args.Term,
		DateFrom: args.DateFrom,
		DateTo:   args.DateTo,
		Limit:    maxToolRows,
	})
	if err != nil {
		return nil, err
	}

	// Phones and emails are NOT sent to Claude.
	rows := make([]toolWebOrderRow, 0, len(list))
	for _, w := range list {
		rows = append(rows, toolWebOrderRow{
			RequestCode:   w.RequestCode,
			OrderType:     w.OrderType,
			CustomerName:  w.CustomerName,
			DeliveryMode:  w.DeliveryMode,
			Total:         w.Total,
			Status:        w.Status,
			CreatedAt:     w.CreatedAt,
			FirstItem:     derefString(w.FirstItemName),
			ItemCount:     w.ItemCount,
			TotalQuantity: w.TotalQuantity,
		})
	}
	return map[string]any{"count": len(rows), "webOrders": rows, "limitedTo": maxToolRows}, nil
}

type toolOrderRow struct {
	OrderInternalID       string  `json:"orderInternalId"`
	OrderCode             string  `json:"orderCode,omitempty"`
	CustomerName          string  `json:"customerName,omitempty"`
	Channel               string  `json:"channel"`
	StartDate             string  `json:"startDate,omitempty"`
	EstimatedDeliveryDate string  `json:"estimatedDeliveryDate,omitempty"`
	Total                 float64 `json:"total"`
	Status                string  `json:"status"`
}

func toolSearchOrders(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		Term     string `json:"term"`
		Status   string `json:"status"`
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	if err := validateToolDate("date_from", args.DateFrom, false); err != nil {
		return nil, err
	}
	if err := validateToolDate("date_to", args.DateTo, false); err != nil {
		return nil, err
	}

	list, err := blorder.BSearchOrders(tc.CompanyID, order.OrderSearchFilters{
		Term:     args.Term,
		Status:   args.Status,
		DateFrom: args.DateFrom,
		DateTo:   args.DateTo,
		Limit:    maxToolRows,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]toolOrderRow, 0, len(list))
	for _, o := range list {
		rows = append(rows, toolOrderRow{
			OrderInternalID:       o.OrderInternalID,
			OrderCode:             derefString(o.OrderCode),
			CustomerName:          derefString(o.CustomerName),
			Channel:               o.Channel,
			StartDate:             derefString(o.StartDate),
			EstimatedDeliveryDate: derefString(o.EstimatedDeliveryDate),
			Total:                 o.Total,
			Status:                o.Status,
		})
	}
	return map[string]any{"count": len(rows), "orders": rows, "limitedTo": maxToolRows}, nil
}

type toolTrackingItem struct {
	Product     string `json:"product"`
	Quantity    int    `json:"quantity"`
	CurrentStep string `json:"currentStep"`
	StepStatus  string `json:"stepStatus"`
	StepNumber  int    `json:"stepNumber,omitempty"`
	TotalSteps  int    `json:"totalSteps"`
}

type toolTrackingChange struct {
	Product    string `json:"product"`
	Step       string `json:"step"`
	StepStatus string `json:"stepStatus"`
	ChangedAt  string `json:"changedAt"`
}

const maxTrackingHistory = 8

func toolOrderTracking(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		OrderCode       string `json:"order_code"`
		OrderInternalID string `json:"order_internal_id"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}

	orderID := strings.TrimSpace(args.OrderInternalID)
	orderCode := strings.TrimSpace(args.OrderCode)
	if orderID == "" {
		if orderCode == "" {
			return nil, toolInputError("order_code or order_internal_id is required")
		}
		found, err := findOrderByCode(tc.CompanyID, orderCode)
		if err != nil {
			return nil, err
		}
		orderID = found.OrderInternalID
		orderCode = derefString(found.OrderCode)
	}

	view, err := blorderitemtracking.BSelectOrderTracking(tc.CompanyID, orderID)
	if err != nil {
		return nil, err
	}

	items := make([]toolTrackingItem, 0, len(view.Items))
	for _, item := range view.Items {
		row := toolTrackingItem{
			Product:     derefString(item.ProductName),
			Quantity:    item.Quantity,
			CurrentStep: "not started",
			StepStatus:  item.CurrentStepStatus,
			TotalSteps:  len(item.StepOptions),
		}
		if item.CurrentStepID != nil {
			for _, option := range item.StepOptions {
				if option.StepTemplateStepID == *item.CurrentStepID {
					row.CurrentStep = option.Name
					row.StepNumber = option.StepOrder
					break
				}
			}
		}
		items = append(items, row)
	}

	// History comes oldest -> newest; keep only the latest changes.
	history := view.History
	if len(history) > maxTrackingHistory {
		history = history[len(history)-maxTrackingHistory:]
	}
	changes := make([]toolTrackingChange, 0, len(history))
	for _, h := range history {
		changes = append(changes, toolTrackingChange{
			Product:    derefString(h.ProductName),
			Step:       h.StepName,
			StepStatus: h.StepStatus,
			ChangedAt:  h.ChangedAt,
		})
	}

	return map[string]any{
		"orderCode":     orderCode,
		"orderStatus":   view.OrderStatus,
		"items":         items,
		"latestChanges": changes,
	}, nil
}

// findOrderByCode resolves the code a user knows into the internal order.
func findOrderByCode(companyID, code string) (*order.OrderS, error) {
	list, err := blorder.BSearchOrders(companyID, order.OrderSearchFilters{
		Term:  code,
		Limit: 10,
	})
	if err != nil {
		return nil, err
	}
	for i := range list {
		if strings.EqualFold(derefString(list[i].OrderCode), code) {
			return &list[i], nil
		}
	}
	if len(list) == 1 {
		return &list[0], nil
	}
	if len(list) == 0 {
		return nil, toolInputError("no order found with code " + code)
	}
	return nil, toolInputError("several orders match " + code + ": use search_orders and pass order_internal_id")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// toolInputError is an error caused by Claude's input; its text is safe to
// return to Claude.
type toolInputError string

func (e toolInputError) Error() string { return string(e) }

func decodeToolInput(input json.RawMessage, target any) error {
	if err := json.Unmarshal(input, target); err != nil {
		return toolInputError("invalid tool input: " + err.Error())
	}
	return nil
}

func validateToolDate(field, value string, required bool) error {
	value = strings.TrimSpace(value)
	if value == "" {
		if required {
			return toolInputError(field + " is required (YYYY-MM-DD)")
		}
		return nil
	}
	if _, err := time.Parse(toolDateLayout, value); err != nil {
		return toolInputError(fmt.Sprintf("%s must use the format YYYY-MM-DD (got %q)", field, value))
	}
	return nil
}

// serviceName resolves a product id to its name (cached per request).
func (tc *toolContext) serviceName(productID string) string {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return ""
	}
	if tc.serviceNames == nil {
		tc.serviceNames = map[string]string{}
		list, err := blproducts.BSearchProducts(tc.CompanyID, products.ProductSearchFilters{Limit: 200})
		if err != nil {
			log.Printf("[blassistant] Could not load service names: %v", err)
		}
		for _, p := range list {
			tc.serviceNames[p.ProductInternalID] = p.Name
		}
	}
	if name, ok := tc.serviceNames[productID]; ok {
		return name
	}
	return productID
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func stringProp(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
