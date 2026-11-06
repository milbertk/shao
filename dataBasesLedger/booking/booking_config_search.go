package booking

import (
	"fmt"
	"strconv"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// BookingConfigSearchFilters holds optional criteria. CompanyID is REQUIRED
// and always applied, so a search never crosses company boundaries.
type BookingConfigSearchFilters struct {
	CompanyID        string
	Term             string // matches name (partial, case-insensitive)
	Name             string
	ServiceProductID string
	AvailabilityMode string
	Limit            int
	Offset           int
}

// SearchBookingConfigs returns the booking configs of ONE company that match
// the filters, ordered by name.
func SearchBookingConfigs(filters BookingConfigSearchFilters) ([]BookingConfigS, error) {
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

	// Free-text term: the schedule name OR the linked service (product) name.
	// The product must belong to the same company.
	term := strings.TrimSpace(filters.Term)
	if term != "" {
		conditions = append(conditions, fmt.Sprintf(
			`(
				tbl_booking_config.name ILIKE $%d
				OR EXISTS (
					SELECT 1
					FROM public.tbl_products p
					WHERE p.company_id = tbl_booking_config.company_id
					  AND p.product_internal_id::text = tbl_booking_config.service_product_id::text
					  AND p.name ILIKE $%d
				)
			)`,
			position, position,
		))
		args = append(args, "%"+term+"%")
		position++
	}

	addLike("name", filters.Name)
	addEqual("service_product_id", filters.ServiceProductID)
	addEqual("availability_mode", strings.ToUpper(filters.AvailabilityMode))

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
		SELECT `+bookingConfigColumns+`
		FROM public.tbl_booking_config
		%s
		ORDER BY lower(name)
		LIMIT %s OFFSET %s
	`, whereClause, limitPlaceholder, offsetPlaceholder)

	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search booking configs: %w", err)
	}
	defer rows.Close()

	list := make([]BookingConfigS, 0)
	for rows.Next() {
		var config BookingConfigS
		if err := scanBookingConfig(rows, &config); err != nil {
			return nil, fmt.Errorf("failed to scan booking config: %w", err)
		}
		list = append(list, config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading booking configs: %w", err)
	}

	return list, nil
}
