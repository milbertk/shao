package blappointment

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/bussunessLedger/bllocations"
	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
	"github.com/milbertk/shao/dataBasesLedger/locations"
)

// AppointmentSaveResult wraps the saved appointment plus the number of
// scheduling conflicts detected (a warning; the save still succeeds).
type AppointmentSaveResult struct {
	Appointment   appointment.AppointmentS `json:"appointment"`
	ConflictCount int                      `json:"conflictCount"`
	// OtherLocationCount (2026-11-07): appointments of ANY service at the
	// same time in ANOTHER location, when the company attends in one
	// location at a time (a warning; the save still succeeds).
	OtherLocationCount int `json:"otherLocationCount"`
}

func emptyToNil(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// BSaveAppointment creates or updates an appointment.
//
//   - companyID is authoritative and forced onto the record.
//   - Empty/unknown appointmentId -> create; otherwise -> update.
//   - New records default to status SCHEDULED (set by the caller/DB).
//   - Returns the saved appointment AND a conflict count (a non-blocking
//     warning): how many other non-cancelled appointments overlap the same
//     company + service + date + hour.
func BSaveAppointment(companyID string, input appointment.AppointmentS) (*AppointmentSaveResult, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	input.CompanyID = companyID
	input.CustomerID = emptyToNil(input.CustomerID)
	input.Phone = emptyToNil(input.Phone)
	input.Email = emptyToNil(input.Email)
	input.SocialLink = emptyToNil(input.SocialLink)
	input.Comments = emptyToNil(input.Comments)
	input.RegisteredBy = emptyToNil(input.RegisteredBy)

	// Basic required checks (the DB also enforces these).
	if strings.TrimSpace(input.ServiceID) == "" {
		return nil, errors.New("serviceId is required")
	}
	if strings.TrimSpace(input.CustomerName) == "" {
		return nil, errors.New("customerName is required")
	}
	if strings.TrimSpace(input.AppointmentDate) == "" {
		return nil, errors.New("appointmentDate is required")
	}
	if strings.TrimSpace(input.StartHour) == "" || strings.TrimSpace(input.EndHour) == "" {
		return nil, errors.New("startHour and endHour are required")
	}

	// A new appointment defaults to SCHEDULED.
	appointmentID := strings.TrimSpace(input.AppointmentID)

	// Location (2026-11-07): optional; it must belong to the company. A
	// disabled location is kept only when the appointment already had it.
	location, err := validateAppointmentLocation(companyID, appointmentID, &input)
	if err != nil {
		return nil, err
	}
	if appointmentID == "" && strings.TrimSpace(input.Status) == "" {
		input.Status = "SCHEDULED"
	}

	// ---- Create or update ----
	createdNow := false
	// What the customer saw before an update (2026-10-30): to tell them
	// about a new status or a new service / date / time.
	var before blnotifications.AppointmentSnapshot
	haveBefore := false
	if appointmentID == "" {
		if err := input.Insert(); err != nil {
			return nil, fmt.Errorf("failed to create appointment: %w", err)
		}
		createdNow = true
	} else {
		existing := appointment.AppointmentS{
			AppointmentID: appointmentID,
			CompanyID:     companyID,
		}
		err := existing.Select()
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				input.AppointmentID = ""
				if err := input.Insert(); err != nil {
					return nil, fmt.Errorf("failed to create appointment: %w", err)
				}
				createdNow = true
			} else {
				return nil, fmt.Errorf("failed to validate appointment: %w", err)
			}
		} else {
			before = blnotifications.SnapshotAppointment(existing)
			haveBefore = true
			if err := input.Update(); err != nil {
				return nil, fmt.Errorf("failed to update appointment: %w", err)
			}
		}
	}

	// ---- Conflict warning (non-blocking) ----
	// Locations count only while the company uses them (2026-11-08); the
	// location stays saved either way.
	locationsOn, _ := bllocations.LocationsEnabled(companyID)
	locationID := ""
	if location != nil {
		if locationsOn {
			locationID = location.LocationID
		}
		input.LocationName = &location.Name
		input.LocationAddress = location.Address
		input.LocationLatitude, input.LocationLongitude = location.Latitude, location.Longitude
	}
	conflicts, err := appointment.CountConflicts(
		companyID,
		input.ServiceID,
		input.AppointmentDate,
		input.StartHour,
		input.EndHour,
		input.AppointmentID, // exclude self
		locationID,
	)
	if err != nil {
		// A conflict-count failure must not fail the save; report zero.
		conflicts = 0
	}
	otherLocations := 0
	if locationID != "" {
		if active, err := bllocations.OneLocationAtATime(companyID); err == nil && active {
			if n, err := appointment.CountOtherLocationConflicts(companyID, input.AppointmentDate,
				input.StartHour, input.EndHour, locationID, input.AppointmentID); err == nil {
				otherLocations = n
			}
		}
	}

	// Tell the customer that the company scheduled the appointment.
	if createdNow && strings.EqualFold(input.Status, "SCHEDULED") {
		blnotifications.NotifyAppointment(companyID, input.AppointmentID, blnotifications.EventCreated, "")
	}
	// Updates (2026-10-30): a new status, or a new service / date / time.
	if !createdNow && haveBefore {
		notifyAppointmentUpdate(companyID, input.AppointmentID, before, blnotifications.SnapshotAppointment(input))
	}

	return &AppointmentSaveResult{
		Appointment:        input,
		ConflictCount:      conflicts,
		OtherLocationCount: otherLocations,
	}, nil
}

// validateAppointmentLocation normalizes input.LocationID, clears the read
// only location fields and returns the location (nil = none).
//
//   - "locationId" not sent (older screens): an existing appointment KEEPS
//     its location; a new one has none.
//   - "locationId": "" removes the location.
func validateAppointmentLocation(companyID, appointmentID string, input *appointment.AppointmentS) (*locations.CompanyLocationS, error) {
	input.LocationName, input.LocationAddress = nil, nil
	input.LocationLatitude, input.LocationLongitude = nil, nil

	var previous *appointment.AppointmentS
	loaded := false
	loadPrevious := func() *appointment.AppointmentS {
		if !loaded && appointmentID != "" {
			loaded = true
			item := appointment.AppointmentS{AppointmentID: appointmentID, CompanyID: companyID}
			if err := item.Select(); err == nil {
				previous = &item
			}
		}
		return previous
	}

	if input.LocationID == nil {
		if prev := loadPrevious(); prev != nil {
			input.LocationID = prev.LocationID
		}
	}
	input.LocationID = bllocations.NormalizeID(input.LocationID)

	location, err := bllocations.ValidateLocationForCompany(companyID, input.LocationID, false)
	if err != nil || location == nil {
		return nil, err
	}
	input.LocationID = &location.LocationID
	if location.Active {
		return location, nil
	}
	if prev := loadPrevious(); prev != nil && prev.LocationID != nil &&
		strings.EqualFold(*prev.LocationID, location.LocationID) {
		return location, nil
	}
	return nil, bllocations.ErrLocationDisabled
}

// appointmentUpdateEvent decides which email an appointment update sends
// (pure, tested): "" = none, EventModified = new service / date / time,
// otherwise the status event. An empty new status keeps the old one.
func appointmentUpdateEvent(before, after blnotifications.AppointmentSnapshot) string {
	if after.Status == "" {
		after.Status = before.Status
	}
	if after.Status != before.Status {
		switch after.Status {
		case "DONE":
			return blnotifications.EventDone
		case "CANCELLED":
			return blnotifications.EventCancelled
		case "SCHEDULED":
			if before.Status == "PENDING" {
				return blnotifications.EventApproved
			}
			return blnotifications.EventReopened
		case "REJECTED":
			return blnotifications.EventRejected
		}
		return ""
	}
	if after.Status == "CANCELLED" || after.Status == "REJECTED" || after.Status == "DONE" {
		return "" // nothing ahead to tell
	}
	if blnotifications.AppointmentWhenChanged(before, after) {
		return blnotifications.EventModified
	}
	return ""
}

func notifyAppointmentUpdate(companyID, appointmentID string, before, after blnotifications.AppointmentSnapshot) {
	switch event := appointmentUpdateEvent(before, after); event {
	case "":
	case blnotifications.EventModified:
		blnotifications.NotifyAppointmentModified(companyID, appointmentID, before)
	default:
		blnotifications.NotifyAppointment(companyID, appointmentID, event, "")
	}
}
