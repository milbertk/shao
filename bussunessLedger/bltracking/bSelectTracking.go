package bltracking

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	tracking "github.com/milbertk/shao/dataBasesLedger/trackingsteeps"
)

// ErrTrackingNotFound lets callers distinguish a missing tracking header from
// a real database error.
var ErrTrackingNotFound = errors.New("tracking configuration not found")

// TrackingBundle groups a tracking header with its ordered steps.
type TrackingBundle struct {
	Config tracking.TrackingConfigS `json:"config"`
	Steps  []tracking.TrackingStepS `json:"steps"`
}

// BSelectTrackingByID loads one tracking HEADER, scoped to the company.
// Returns ErrTrackingNotFound when it does not exist in that company.
func BSelectTrackingByID(companyID, trackingConfigID string) (*tracking.TrackingConfigS, error) {
	companyID = strings.TrimSpace(companyID)
	trackingConfigID = strings.TrimSpace(trackingConfigID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if trackingConfigID == "" {
		return nil, errors.New("trackingConfigId is required")
	}

	config := tracking.TrackingConfigS{
		TrackingConfigID: trackingConfigID,
		CompanyID:        companyID,
	}

	if err := config.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTrackingNotFound
		}
		return nil, fmt.Errorf("failed to select tracking config: %w", err)
	}

	return &config, nil
}

// BSelectTrackingSteps returns the ordered STEPS of a tracking header, scoped
// to the company. (The data layer sorts them by step_order.)
func BSelectTrackingSteps(companyID, trackingConfigID string) ([]tracking.TrackingStepS, error) {
	companyID = strings.TrimSpace(companyID)
	trackingConfigID = strings.TrimSpace(trackingConfigID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if trackingConfigID == "" {
		return nil, errors.New("trackingConfigId is required")
	}

	steps, err := tracking.SelectStepsByConfig(companyID, trackingConfigID)
	if err != nil {
		return nil, fmt.Errorf("failed to select tracking steps: %w", err)
	}

	return steps, nil
}

// BSelectTrackingFull returns the HEADER together with its ordered STEPS
// (the steps linked to the header) in a single bundle. This is the one to use
// when opening a tracking to view or edit it.
func BSelectTrackingFull(companyID, trackingConfigID string) (*TrackingBundle, error) {
	config, err := BSelectTrackingByID(companyID, trackingConfigID)
	if err != nil {
		return nil, err
	}

	steps, err := BSelectTrackingSteps(companyID, trackingConfigID)
	if err != nil {
		return nil, err
	}

	return &TrackingBundle{
		Config: *config,
		Steps:  steps,
	}, nil
}
