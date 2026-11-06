package order

import (
	"fmt"
	"strconv"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// OrderSearchFilters holds optional criteria. CompanyID is REQUIRED and always
// applied.
type OrderSearchFilters struct {
	CompanyID    string
	Term         string // matches order_code + customer_name
	CustomerID   string
	CustomerName string
	Status       string
	DateFrom     string // start_date >= DateFrom (YYYY-MM-DD)
	DateTo       string // start_date <= DateTo   (YYYY-MM-DD)
	Limit        int
	Offset       int
}

// SearchOrders returns the order headers of ONE company matching the filters,
// most recent first.
func SearchOrders(filters OrderSearchFilters) ([]OrderS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(filters.CompanyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	conditions := []string{"company_id = $1"}
	args := []interface{}{companyID}
	position := 2

	addLike := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", column, position))
		args = append(args, "%"+value+"%")
		position++
	}

	addEqual := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, position))
		args = append(args, value)
		position++
	}

	// Free-text term across order_code + customer_name.
	term := strings.TrimSpace(filters.Term)
	if term != "" {
		conditions = append(conditions, fmt.Sprintf(
			`(
				COALESCE(order_code, '') ILIKE $%d
				OR COALESCE(customer_name, '') ILIKE $%d
			)`,
			position, position,
		))
		args = append(args, "%"+term+"%")
		position++
	}

	addEqual("customer_id", filters.CustomerID)
	addLike("COALESCE(customer_name, '')", filters.CustomerName)
	addEqual("status", strings.ToUpper(filters.Status))

	// Date range on start_date.
	if v := strings.TrimSpace(filters.DateFrom); v != "" {
		conditions = append(conditions, fmt.Sprintf("start_date >= $%d", position))
		args = append(args, v)
		position++
	}
	if v := strings.TrimSpace(filters.DateTo); v != "" {
		conditions = append(conditions, fmt.Sprintf("start_date <= $%d", position))
		args = append(args, v)
		position++
	}

	whereClause := "WHERE " + strings.Join(conditions, "\n\t\t\t  AND ")

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	limitPlaceholder := "$" + strconv.Itoa(position)
	position++
	offsetPlaceholder := "$" + strconv.Itoa(position)

	query := fmt.Sprintf(`
		SELECT `+orderColumns+`
		FROM public.tbl_order
		%s
		ORDER BY created_at DESC
		LIMIT %s OFFSET %s
	`, whereClause, limitPlaceholder, offsetPlaceholder)

	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search orders: %w", err)
	}
	defer rows.Close()

	list := make([]OrderS, 0)
	for rows.Next() {
		var o OrderS
		if err := scanOrder(rows, &o); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		list = append(list, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading orders: %w", err)
	}

	return list, nil
}
