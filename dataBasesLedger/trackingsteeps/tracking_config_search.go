package tracking

import (
	"fmt"
	"strconv"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// TrackingConfigSearchFilters holds optional criteria. CompanyID is REQUIRED
// and always applied.
type TrackingConfigSearchFilters struct {
	CompanyID    string
	Term         string // matches name / internal_code / tracking_number
	Name         string
	InternalCode string
	TargetType   string // CATEGORY | PRODUCT
	CategoryID   string
	ProductID    string
	Limit        int
	Offset       int
}

// SearchTrackingConfigs returns the tracking headers of ONE company matching
// the filters, ordered by name.
func SearchTrackingConfigs(filters TrackingConfigSearchFilters) ([]TrackingConfigS, error) {
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
				name ILIKE $%d
				OR internal_code ILIKE $%d
				OR COALESCE(tracking_number, '') ILIKE $%d
			)`,
			position, position, position,
		))
		args = append(args, "%"+term+"%")
		position++
	}

	addLike("name", filters.Name)
	addLike("internal_code", filters.InternalCode)
	addEqual("target_type", strings.ToUpper(filters.TargetType))
	addEqual("category_id", filters.CategoryID)
	addEqual("product_id", filters.ProductID)

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
		SELECT `+trackingConfigColumns+`
		FROM public.tbl_tracking_config
		%s
		ORDER BY lower(name)
		LIMIT %s OFFSET %s
	`, whereClause, limitPlaceholder, offsetPlaceholder)

	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search tracking configs: %w", err)
	}
	defer rows.Close()

	list := make([]TrackingConfigS, 0)
	for rows.Next() {
		var config TrackingConfigS
		if err := scanTrackingConfig(rows, &config); err != nil {
			return nil, fmt.Errorf("failed to scan tracking config: %w", err)
		}
		list = append(list, config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading tracking configs: %w", err)
	}

	return list, nil
}
