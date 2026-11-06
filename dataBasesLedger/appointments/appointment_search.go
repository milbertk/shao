package appointment

import (
	"fmt"
	"strconv"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// AppointmentSearchFilters holds optional criteria. CompanyID is REQUIRED.
type AppointmentSearchFilters struct {
	CompanyID    string
	Term         string // matches customer_name + appointment id text
	CustomerName string
	CustomerID   string
	ServiceID    string
	Status       string
	DateFrom     string // appointment_date >= DateFrom
	DateTo       string // appointment_date <= DateTo
	Limit        int
	Offset       int
}

// SearchAppointments returns the appointments of ONE company matching the
// filters, most recent date first.
func SearchAppointments(filters AppointmentSearchFilters) ([]AppointmentS, error) {
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

	term := strings.TrimSpace(filters.Term)
	if term != "" {
		conditions = append(conditions, fmt.Sprintf(
			`(
				customer_name ILIKE $%d
				OR CAST(appointment_id AS TEXT) ILIKE $%d
			)`,
			position, position,
		))
		args = append(args, "%"+term+"%")
		position++
	}

	addLike("customer_name", filters.CustomerName)
	addEqual("customer_id", filters.CustomerID)
	addEqual("service_id", filters.ServiceID)
	addEqual("status", strings.ToUpper(filters.Status))

	if v := strings.TrimSpace(filters.DateFrom); v != "" {
		conditions = append(conditions, fmt.Sprintf("appointment_date >= $%d", position))
		args = append(args, v)
		position++
	}
	if v := strings.TrimSpace(filters.DateTo); v != "" {
		conditions = append(conditions, fmt.Sprintf("appointment_date <= $%d", position))
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
		SELECT `+appointmentColumns+`
		FROM public.tbl_appointment
		%s
		ORDER BY appointment_date DESC, start_hour ASC
		LIMIT %s OFFSET %s
	`, whereClause, limitPlaceholder, offsetPlaceholder)

	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search appointments: %w", err)
	}
	defer rows.Close()

	list := make([]AppointmentS, 0)
	for rows.Next() {
		var item AppointmentS
		if err := scanAppointment(rows, &item); err != nil {
			return nil, fmt.Errorf("failed to scan appointment: %w", err)
		}
		list = append(list, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading appointments: %w", err)
	}

	return list, nil
}
