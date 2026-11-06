package configuration

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// GetParameter returns the value of one configuration parameter. Returns ""
// (and no error) when the parameter does not exist.
func GetParameter(parameter string) (string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", fmt.Errorf("database connection error: %w", err)
	}

	parameter = strings.TrimSpace(parameter)
	if parameter == "" {
		return "", fmt.Errorf("parameter is required")
	}

	var value sql.NullString
	err = db.QueryRow(
		`SELECT value FROM public.tbl_configuration WHERE parameter = $1`,
		parameter,
	).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to read configuration %q: %w", parameter, err)
	}

	if !value.Valid {
		return "", nil
	}
	return strings.TrimSpace(value.String), nil
}

// GetParameters returns several parameters at once as a map. Missing ones are
// simply absent from the map.
func GetParameters(parameters ...string) (map[string]string, error) {
	result := make(map[string]string)
	if len(parameters) == 0 {
		return result, nil
	}

	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	// Build a parameter list for the IN clause.
	placeholders := make([]string, 0, len(parameters))
	args := make([]interface{}, 0, len(parameters))
	for i, p := range parameters {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, strings.TrimSpace(p))
	}

	query := fmt.Sprintf(
		`SELECT parameter, value FROM public.tbl_configuration WHERE parameter IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value sql.NullString
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("failed to scan configuration: %w", err)
		}
		if value.Valid {
			result[name] = strings.TrimSpace(value.String)
		} else {
			result[name] = ""
		}
	}

	return result, rows.Err()
}
