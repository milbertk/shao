package bltracking

import (
	"errors"
	"fmt"
	"strings"

	tracking "github.com/milbertk/shao/dataBasesLedger/trackingsteeps"
)

// BSearchTracking returns the tracking headers of ONE company that match the
// optional filters. The company is REQUIRED and always applied, so results
// never cross company boundaries.
//
// The usual search fields are the name and the internal code (the "term"
// filter matches both, plus the tracking number).
func BSearchTracking(
	companyID string,
	filters tracking.TrackingConfigSearchFilters,
) ([]tracking.TrackingConfigS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	// Force the company so the caller cannot search another company's data.
	filters.CompanyID = companyID

	results, err := tracking.SearchTrackingConfigs(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search tracking: %w", err)
	}

	return results, nil
}
