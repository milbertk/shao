package blassistant

import (
	"encoding/json"
	"strings"
	"time"

	blappointment "github.com/milbertk/shao/bussunessLedger/blappontment"
	"github.com/milbertk/shao/bussunessLedger/blorder"
	"github.com/milbertk/shao/bussunessLedger/blreports"
	"github.com/milbertk/shao/bussunessLedger/blweborder"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
)

// =====================================================================
//  Insight tools (READ-ONLY): day summary, due dates, rankings and the
//  reports of the Reports screen. They reuse the existing business
//  functions; results are trimmed to keep the prompt small.
// =====================================================================

const (
	maxReportRows      = 15 // rows of a report sent to Claude
	maxRankingRows     = 20 // top N orders / customers / products
	maxSeriesPoints    = 31 // revenue points
	maxCancelMonths    = 12 // cancellation months
	maxSummaryListRows = 10 // rows per list in the day summary
)

func insightToolDefinitions() []claudeTool {
	return []claudeTool{
		{
			Name: "get_day_summary",
			Description: "Briefing for ONE date (default today): the appointments of that day, booking requests " +
				"waiting for approval (appointments with status PENDING, any date), web orders waiting for approval, " +
				"and open orders due on or before that date (overdue included). Use it for questions like " +
				"\"what do I have tomorrow?\", \"what is pending?\" or \"summary of today\".",
			InputSchema: objectSchema(map[string]any{
				"date": stringProp("Date YYYY-MM-DD. Default: today."),
			}),
		},
		{
			Name: "get_orders_due_soon",
			Description: "Open orders (not DONE / CANCELLED) whose estimated delivery date is within the next " +
				"days_ahead days, INCLUDING overdue ones, earliest first, with daysLeft (negative = overdue).",
			InputSchema: objectSchema(map[string]any{
				"days_ahead": intProp("How many days ahead from today (0-90). Default 7. Use 0 for today/overdue only."),
			}),
		},
		{
			Name: "rank_orders",
			Description: "Orders sorted by: highest_total (biggest orders, \"top 10\"), oldest (first created) or " +
				"newest (last created). Optionally only open orders and/or created within a date range.",
			InputSchema: objectSchema(map[string]any{
				"sort":      enumProp("Ranking.", "highest_total", "oldest", "newest"),
				"open_only": boolProp("true = only orders not DONE / CANCELLED."),
				"limit":     intProp("How many orders (1-20). Default 10."),
				"date_from": stringProp("Optional: created on or after YYYY-MM-DD."),
				"date_to":   stringProp("Optional: created on or before YYYY-MM-DD."),
			}, "sort"),
		},
		{
			Name: "get_aging_orders",
			Description: "Open orders that have been waiting the longest (by days since creation), oldest first, " +
				"with totals per age group: fresh (0-7 days), aging (8-30), overdue (over 30).",
			InputSchema: objectSchema(map[string]any{
				"status":   enumProp("Optional status filter.", "all", "placed", "inProgress", "partialDone"),
				"min_days": intProp("Optional: only orders at least this many days old."),
			}),
		},
		{
			Name: "get_orders_report",
			Description: "Orders report for a period (by creation date): count per status, number of orders, " +
				"revenue, and the latest orders. Empty dates = all time.",
			InputSchema: objectSchema(map[string]any{
				"date_from": stringProp("Optional start YYYY-MM-DD."),
				"date_to":   stringProp("Optional end YYYY-MM-DD."),
				"status":    enumProp("Optional status.", "all", "placed", "inProgress", "partialDone", "done", "cancelled"),
			}),
		},
		{
			Name: "get_appointments_report",
			Description: "Appointments report for a period: counts per group (scheduled, finished, cancelled), " +
				"total, time invested in minutes, and the latest appointments. Empty dates = all time.",
			InputSchema: objectSchema(map[string]any{
				"date_from": stringProp("Optional start YYYY-MM-DD."),
				"date_to":   stringProp("Optional end YYYY-MM-DD."),
				"group":     enumProp("Optional group.", "all", "scheduled", "finished", "cancelled"),
			}),
		},
		{
			Name:        "get_top_customers",
			Description: "Best customers ranked by number of orders, then amount spent (also counts appointments).",
			InputSchema: objectSchema(map[string]any{
				"limit": intProp("How many customers (1-20). Default 10."),
			}),
		},
		{
			Name:        "get_best_sellers",
			Description: "Best-selling products and services by units sold (orders and/or bookings), with revenue.",
			InputSchema: objectSchema(map[string]any{
				"type":  enumProp("What to rank.", "all", "order", "booking"),
				"limit": intProp("How many (1-20). Default 10."),
			}),
		},
		{
			Name: "get_revenue",
			Description: "Revenue of orders (cancelled excluded) over a period, grouped by day, week or month, " +
				"with total revenue, order count, average order value and best period.",
			InputSchema: objectSchema(map[string]any{
				"date_from":   stringProp("Optional start YYYY-MM-DD."),
				"date_to":     stringProp("Optional end YYYY-MM-DD."),
				"granularity": enumProp("Grouping.", "day", "week", "month"),
			}),
		},
		{
			Name:        "get_cancellation_rate",
			Description: "Appointment cancellation rate (cancelled + rejected / total), overall and by month.",
			InputSchema: objectSchema(map[string]any{
				"date_from": stringProp("Optional start YYYY-MM-DD."),
				"date_to":   stringProp("Optional end YYYY-MM-DD."),
			}),
		},
	}
}

// runInsightTool executes one insight tool. ok=false when the name is unknown.
func runInsightTool(tc *toolContext, name string, input json.RawMessage) (any, bool, error) {
	var (
		result any
		err    error
	)
	switch name {
	case "get_day_summary":
		result, err = toolDaySummary(tc, input)
	case "get_orders_due_soon":
		result, err = toolOrdersDueSoon(tc, input)
	case "rank_orders":
		result, err = toolRankOrders(tc, input)
	case "get_aging_orders":
		result, err = toolAgingOrders(tc, input)
	case "get_orders_report":
		result, err = toolOrdersReport(tc, input)
	case "get_appointments_report":
		result, err = toolAppointmentsReport(tc, input)
	case "get_top_customers":
		result, err = toolTopCustomers(tc, input)
	case "get_best_sellers":
		result, err = toolBestSellers(tc, input)
	case "get_revenue":
		result, err = toolRevenue(tc, input)
	case "get_cancellation_rate":
		result, err = toolCancellationRate(tc, input)
	default:
		return nil, false, nil
	}
	return result, true, err
}

// ---------------------------------------------------------------------------
// Day summary
// ---------------------------------------------------------------------------

type toolRequestRow struct {
	Date         string `json:"date"`
	Start        string `json:"start"`
	CustomerName string `json:"customerName"`
	Service      string `json:"service"`
}

type toolPendingWebOrderRow struct {
	RequestCode  string  `json:"requestCode"`
	CustomerName string  `json:"customerName"`
	Total        float64 `json:"total"`
	CreatedAt    string  `json:"createdAt"`
	FirstItem    string  `json:"firstItem,omitempty"`
}

func toolDaySummary(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		Date string `json:"date"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	date := strings.TrimSpace(args.Date)
	if date == "" {
		date = tc.today().Format(toolDateLayout)
	}
	if err := validateToolDate("date", date, true); err != nil {
		return nil, err
	}

	// 1) Appointments of the day.
	dayList, err := blappointment.BSearchAppointments(tc.CompanyID, appointment.AppointmentSearchFilters{
		DateFrom: date,
		DateTo:   date,
		Limit:    maxToolRows,
	})
	if err != nil {
		return nil, err
	}
	dayRows := make([]toolAppointmentRow, 0, len(dayList))
	for _, a := range dayList {
		dayRows = append(dayRows, toolAppointmentRow{
			Date:         a.AppointmentDate,
			Start:        a.StartHour,
			End:          a.EndHour,
			CustomerName: a.CustomerName,
			Service:      tc.serviceName(a.ServiceID),
			Status:       a.Status,
			Location:     derefString(a.LocationName),
		})
	}

	// 2) Booking requests waiting for approval (any date).
	pending, err := blappointment.BSearchAppointments(tc.CompanyID, appointment.AppointmentSearchFilters{
		Status: "PENDING",
		Limit:  maxToolRows,
	})
	if err != nil {
		return nil, err
	}
	requests := make([]toolRequestRow, 0, maxSummaryListRows)
	for i, a := range pending {
		if i >= maxSummaryListRows {
			break
		}
		requests = append(requests, toolRequestRow{
			Date:         a.AppointmentDate,
			Start:        a.StartHour,
			CustomerName: a.CustomerName,
			Service:      tc.serviceName(a.ServiceID),
		})
	}

	// 3) Web orders waiting for approval.
	webOrders, err := blweborder.BSearchWebOrders(tc.CompanyID, weborder.WebOrderSearchFilters{
		Status: "PENDING",
		Limit:  maxToolRows,
	})
	if err != nil {
		return nil, err
	}
	webRows := make([]toolPendingWebOrderRow, 0, maxSummaryListRows)
	for i, w := range webOrders {
		if i >= maxSummaryListRows {
			break
		}
		webRows = append(webRows, toolPendingWebOrderRow{
			RequestCode:  w.RequestCode,
			CustomerName: w.CustomerName,
			Total:        w.Total,
			CreatedAt:    w.CreatedAt,
			FirstItem:    derefString(w.FirstItemName),
		})
	}

	// 4) Open orders due on or before the date (overdue included).
	due, err := blorder.BOrdersDueBy(tc.CompanyID, tc.today(), date, maxSummaryListRows)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"date": date,
		"appointments": map[string]any{
			"count": len(dayRows), "limitedTo": maxToolRows, "items": dayRows,
		},
		"bookingRequestsPending": map[string]any{
			"count": len(pending), "limitedTo": maxToolRows, "first": requests,
		},
		"webOrdersPending": map[string]any{
			"count": len(webOrders), "limitedTo": maxToolRows, "first": webRows,
		},
		"ordersDueByDate": map[string]any{
			"count": due.TotalMatches, "overdueInList": due.Overdue, "first": due.Orders,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Orders: due dates and rankings
// ---------------------------------------------------------------------------

func toolOrdersDueSoon(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		DaysAhead *int `json:"days_ahead"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	days := 7
	if args.DaysAhead != nil {
		days = *args.DaysAhead
	}
	if days < 0 || days > 90 {
		return nil, toolInputError("days_ahead must be between 0 and 90")
	}

	return blorder.BOrdersDueSoon(tc.CompanyID, tc.today(), days, maxToolRows)
}

func toolRankOrders(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		Sort     string `json:"sort"`
		OpenOnly bool   `json:"open_only"`
		Limit    int    `json:"limit"`
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	switch args.Sort {
	case "highest_total", "oldest", "newest":
	default:
		return nil, toolInputError("sort must be highest_total, oldest or newest")
	}
	if err := validateToolDate("date_from", args.DateFrom, false); err != nil {
		return nil, err
	}
	if err := validateToolDate("date_to", args.DateTo, false); err != nil {
		return nil, err
	}

	list, err := blorder.BOrdersRanked(tc.CompanyID, order.OrderRankingFilters{
		SortKey:  args.Sort,
		OpenOnly: args.OpenOnly,
		DateFrom: args.DateFrom,
		DateTo:   args.DateTo,
		Limit:    clampLimit(args.Limit, 10, maxRankingRows),
	})
	if err != nil {
		return nil, err
	}

	rows := make([]map[string]any, 0, len(list))
	for i, o := range list {
		rows = append(rows, map[string]any{
			"rank":                  i + 1,
			"orderCode":             derefString(o.OrderCode),
			"customerName":          derefString(o.CustomerName),
			"total":                 o.Total,
			"status":                o.Status,
			"createdAt":             o.CreatedAt.Format(toolDateLayout),
			"estimatedDeliveryDate": derefString(o.EstimatedDeliveryDate),
		})
	}
	return map[string]any{"sort": args.Sort, "openOnly": args.OpenOnly, "orders": rows}, nil
}

func toolAgingOrders(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		Status  string `json:"status"`
		MinDays *int   `json:"min_days"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	minDays := ""
	if args.MinDays != nil {
		if *args.MinDays < 0 {
			return nil, toolInputError("min_days must be 0 or more")
		}
		minDays = intToString(*args.MinDays)
	}
	status := args.Status
	if status == "" {
		status = "all"
	}

	report, err := blreports.BAgingOrders(tc.CompanyID, status, minDays)
	if err != nil {
		return nil, err
	}
	rows := report.Rows
	if len(rows) > maxRankingRows {
		rows = rows[:maxRankingRows]
	}
	return map[string]any{
		"totalOpen":  report.TotalOpen,
		"fresh":      report.Fresh,
		"aging":      report.Aging,
		"overdue":    report.Overdue,
		"oldestDays": report.OldestDays,
		"oldest":     rows,
		"limitedTo":  maxRankingRows,
	}, nil
}

// ---------------------------------------------------------------------------
// Reports (same data as the Reports screen)
// ---------------------------------------------------------------------------

func toolOrdersReport(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
		Status   string `json:"status"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	if err := validateRange(args.DateFrom, args.DateTo); err != nil {
		return nil, err
	}
	status := args.Status
	if status == "" {
		status = "all"
	}

	report, err := blreports.BOrdersByStatus(tc.CompanyID, args.DateFrom, args.DateTo, status)
	if err != nil {
		return nil, err
	}
	rows := report.Rows
	if len(rows) > maxReportRows {
		rows = rows[:maxReportRows]
	}
	return map[string]any{
		"countsByStatus": report.Counts,
		"totalOrders":    report.Total,
		"revenue":        report.Revenue,
		"latestOrders":   rows,
		"latestLimit":    maxReportRows,
	}, nil
}

func toolAppointmentsReport(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
		Group    string `json:"group"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	if err := validateRange(args.DateFrom, args.DateTo); err != nil {
		return nil, err
	}
	group := args.Group
	if group == "" {
		group = "all"
	}

	report, err := blreports.BAppointmentsSummary(tc.CompanyID, args.DateFrom, args.DateTo, group)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, maxReportRows)
	for i, r := range report.Rows {
		if i >= maxReportRows {
			break
		}
		rows = append(rows, map[string]any{
			"date":         r.Date,
			"customerName": r.CustomerName,
			"service":      r.ServiceName,
			"status":       r.Status,
			"durationMin":  r.DurationMin,
		})
	}
	return map[string]any{
		"countsByGroup":      report.Counts,
		"total":              report.Total,
		"timeMinutes":        report.TimeMinutes,
		"latestAppointments": rows,
		"latestLimit":        maxReportRows,
	}, nil
}

func toolTopCustomers(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		Limit int `json:"limit"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	limit := clampLimit(args.Limit, 10, maxRankingRows)

	report, err := blreports.BTopCustomers(tc.CompanyID, limit)
	if err != nil {
		return nil, err
	}
	// Only names and totals: the identifier may be an email or a phone.
	rows := make([]map[string]any, 0, len(report.Rows))
	for i, r := range report.Rows {
		if i >= limit {
			break
		}
		rows = append(rows, map[string]any{
			"rank":         i + 1,
			"name":         r.Name,
			"orders":       r.Orders,
			"appointments": r.Appointments,
			"spent":        r.Spent,
		})
	}
	return map[string]any{"customers": rows}, nil
}

func toolBestSellers(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		Type  string `json:"type"`
		Limit int    `json:"limit"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	kind := args.Type
	if kind == "" {
		kind = "all"
	}
	limit := clampLimit(args.Limit, 10, maxRankingRows)

	report, err := blreports.BBestSellers(tc.CompanyID, kind, limit)
	if err != nil {
		return nil, err
	}
	rows := report.Rows
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return map[string]any{"products": rows}, nil
}

func toolRevenue(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		DateFrom    string `json:"date_from"`
		DateTo      string `json:"date_to"`
		Granularity string `json:"granularity"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	if err := validateRange(args.DateFrom, args.DateTo); err != nil {
		return nil, err
	}
	granularity := args.Granularity
	if granularity == "" {
		granularity = "month"
	}

	report, err := blreports.BRevenueOverTime(tc.CompanyID, args.DateFrom, args.DateTo, granularity)
	if err != nil {
		return nil, err
	}
	points := report.Points
	if len(points) > maxSeriesPoints {
		points = points[len(points)-maxSeriesPoints:]
	}
	return map[string]any{
		"granularity":   report.Granularity,
		"totalRevenue":  report.TotalRevenue,
		"totalOrders":   report.TotalOrders,
		"avgOrderValue": report.AvgOrderValue,
		"bestPeriod":    report.BestPeriod,
		"bestRevenue":   report.BestRevenue,
		"points":        points,
	}, nil
}

func toolCancellationRate(tc *toolContext, input json.RawMessage) (any, error) {
	var args struct {
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	}
	if err := decodeToolInput(input, &args); err != nil {
		return nil, err
	}
	if err := validateRange(args.DateFrom, args.DateTo); err != nil {
		return nil, err
	}

	report, err := blreports.BCancellationRate(tc.CompanyID, args.DateFrom, args.DateTo)
	if err != nil {
		return nil, err
	}
	months := report.Months
	if len(months) > maxCancelMonths {
		months = months[len(months)-maxCancelMonths:]
	}
	return map[string]any{
		"total":     report.Total,
		"cancelled": report.Cancelled,
		"ratePct":   report.RatePct,
		"months":    months,
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// today is the request's local date (Now is set by BAskAssistant).
func (tc *toolContext) today() time.Time {
	now := tc.Now
	if now.IsZero() {
		now = time.Now()
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func validateRange(dateFrom, dateTo string) error {
	if err := validateToolDate("date_from", dateFrom, false); err != nil {
		return err
	}
	return validateToolDate("date_to", dateTo, false)
}

func clampLimit(value, fallback, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func intToString(value int) string {
	b, _ := json.Marshal(value)
	return string(b)
}

func intProp(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

func boolProp(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}

func enumProp(description string, values ...string) map[string]any {
	return map[string]any{"type": "string", "description": description, "enum": values}
}
