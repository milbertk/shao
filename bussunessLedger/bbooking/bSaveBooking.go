package blbooking

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/bussunessLedger/bllocations"
	"github.com/milbertk/shao/bussunessLedger/bltimezone"
	blproducts "github.com/milbertk/shao/bussunessLedger/products"
	"github.com/milbertk/shao/dataBasesLedger/booking"
)

// ---- small normalization helpers ----

func emptyToNilStr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// nilIfLessThanOne treats missing/zero/negative appointment overrides as "use
// the general count" (NULL in the database).
func nilIfLessThanOne(value *int) *int {
	if value == nil || *value < 1 {
		return nil
	}
	return value
}

// validateProductInCompany ensures the linked service product exists in the
// same company (the FK guarantees existence, not company ownership) and
// returns its name.
func validateProductInCompany(companyID, productInternalID string) (string, error) {
	productInternalID = strings.TrimSpace(productInternalID)
	if productInternalID == "" {
		return "", errors.New("serviceProductId is required")
	}

	product, err := blproducts.BSelectProductByInternalID(companyID, productInternalID)
	if err != nil {
		if errors.Is(err, blproducts.ErrProductNotFound) {
			return "", fmt.Errorf("product %q does not exist in this company", productInternalID)
		}
		return "", fmt.Errorf("failed to validate product: %w", err)
	}

	return strings.TrimSpace(product.Name), nil
}

// validateScheduleLocations normalizes the optional location of every range
// and slot, checks that it exists in the company, and checks the "one
// location at a time" rule (returns *bllocations.LocationOverlapError).
func validateScheduleLocations(companyID string, input *BookingBundle) error {
	byID, err := bllocations.CompanyLocations(companyID)
	if err != nil {
		return fmt.Errorf("failed to load locations: %w", err)
	}
	locationName := func(id *string) (string, error) {
		if id == nil {
			return "", nil
		}
		l, ok := byID[strings.ToLower(*id)]
		if !ok || !bllocations.ValidID(*id) {
			return "", fmt.Errorf("invalid location: %q does not exist in this company", *id)
		}
		return l.Name, nil
	}

	today := bltimezone.CompanyToday(companyID)
	manualMode := strings.EqualFold(strings.TrimSpace(input.Config.AvailabilityMode), "MANUAL")
	blocks := make([]bllocations.TimeBlock, 0)
	scheduleName := strings.TrimSpace(input.Config.Name)

	// "Customer chooses" (2026-11-08): the block has no location of its own
	// and counts as AnyLocation for the "one location at a time" rule.
	blockLocation := func(id *string, choice bool) string {
		if choice {
			return bllocations.AnyLocation
		}
		if id != nil {
			return *id
		}
		return ""
	}

	for i := range input.RecurringRanges {
		r := &input.RecurringRanges[i]
		r.LocationID = bllocations.NormalizeID(r.LocationID)
		if r.CustomerChoice {
			r.LocationID = nil
		}
		name, err := locationName(r.LocationID)
		if err != nil {
			return err
		}
		if where := blockLocation(r.LocationID, r.CustomerChoice); where != "" && r.Active && !manualMode {
			blocks = append(blocks, bllocations.TimeBlock{
				Kind: bllocations.KindRecurring, ScheduleName: scheduleName,
				DayOfWeek: r.DayOfWeek, StartTime: r.StartTime, EndTime: r.EndTime,
				LocationID: where, LocationName: name,
			})
		}
	}
	for i := range input.ManualSlots {
		s := &input.ManualSlots[i]
		s.LocationID = bllocations.NormalizeID(s.LocationID)
		if s.CustomerChoice {
			s.LocationID = nil
		}
		name, err := locationName(s.LocationID)
		if err != nil {
			return err
		}
		if where := blockLocation(s.LocationID, s.CustomerChoice); where != "" && manualMode && strings.TrimSpace(s.SlotDate) >= today {
			block := bllocations.TimeBlock{
				Kind: bllocations.KindManual, ScheduleName: scheduleName,
				Date: strings.TrimSpace(s.SlotDate), AllDay: s.AllDay,
				LocationID: where, LocationName: name,
			}
			if !s.AllDay && s.StartTime != nil && s.EndTime != nil {
				block.StartTime, block.EndTime = strings.TrimSpace(*s.StartTime), strings.TrimSpace(*s.EndTime)
			}
			blocks = append(blocks, block)
		}
	}

	// A disabled schedule offers nothing: nothing to clash with.
	if !input.Config.Active {
		return nil
	}
	return bllocations.CheckScheduleBlocks(companyID, strings.TrimSpace(input.Config.BookingConfigID), blocks)
}

// BSaveBooking creates or updates a booking, then REWRITES its schedules.
//
//   - companyID is authoritative and forced onto every record.
//   - Header: if config.BookingConfigID is empty or does not exist -> create;
//     otherwise -> update.
//   - Recurring ranges: full replace (delete all, insert the provided list).
//   - Manual slots: replace FROM TODAY FORWARD (past slots are kept as
//     history), then insert the provided list.
//   - Only the hours of the chosen mode are kept (2026-11-08); appointments
//     are never touched.
//
// Returns the fresh header + both schedule lists as stored.
func BSaveBooking(companyID string, input BookingBundle) (*BookingBundle, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	// The company always comes from the validated context, never the body.
	input.Config.CompanyID = companyID

	// The linked service product must exist in this company.
	serviceName, err := validateProductInCompany(companyID, input.Config.ServiceProductID)
	if err != nil {
		return nil, err
	}

	// The schedule name is the service name unless one is given.
	if strings.TrimSpace(input.Config.Name) == "" {
		input.Config.Name = serviceName
	}

	// One schedule per service: another schedule must not use this service.
	if err := ensureServiceIsFree(companyID, input.Config.ServiceProductID, input.Config.BookingConfigID); err != nil {
		return nil, err
	}

	// One mode only (2026-11-08): the hours of the other mode are deleted.
	KeepActiveMode(&input)
	// Past dates stay as history and are never inserted again (2026-11-08).
	DropPastDates(&input, bltimezone.CompanyToday(companyID))

	// Locations (2026-11-07): every location used must belong to the company,
	// and with "one location at a time" the hours must not clash with hours
	// in another location (any schedule of the company). Checked BEFORE
	// anything is written.
	if err := validateScheduleLocations(companyID, &input); err != nil {
		return nil, err
	}
	// Same location, same hours: refused BEFORE the saved hours are deleted.
	if err := CheckSameLocationOverlaps(input); err != nil {
		return nil, err
	}

	// Normalize header optional fields.
	input.Config.DisabledFrom = emptyToNilStr(input.Config.DisabledFrom)
	input.Config.DisabledTo = emptyToNilStr(input.Config.DisabledTo)
	if input.Config.GeneralAppointmentsPerSlot < 1 {
		input.Config.GeneralAppointmentsPerSlot = 1
	}
	if input.Config.AppointmentDurationMinutes < 1 {
		input.Config.AppointmentDurationMinutes = 60
	}

	// ---- Save header (create or update) ----
	configID := strings.TrimSpace(input.Config.BookingConfigID)

	// A unique-violation from the database (e.g. two users saving at the same
	// time) is turned into the same friendly "service already has a schedule".
	serviceID := input.Config.ServiceProductID

	if configID == "" {
		if err := input.Config.Insert(); err != nil {
			return nil, scheduleConflictOr(companyID, serviceID, fmt.Errorf("failed to create booking config: %w", err))
		}
	} else {
		existing := booking.BookingConfigS{
			BookingConfigID: configID,
			CompanyID:       companyID,
		}
		err := existing.Select()
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Provided id does not exist -> create a new record.
				input.Config.BookingConfigID = ""
				if err := input.Config.Insert(); err != nil {
					return nil, scheduleConflictOr(companyID, serviceID, fmt.Errorf("failed to create booking config: %w", err))
				}
			} else {
				return nil, fmt.Errorf("failed to validate booking config: %w", err)
			}
		} else {
			if err := input.Config.Update(); err != nil {
				return nil, scheduleConflictOr(companyID, serviceID, fmt.Errorf("failed to update booking config: %w", err))
			}
		}
	}

	configID = input.Config.BookingConfigID

	// ---- Rewrite recurring ranges (full replace) ----
	if _, err := booking.DeleteRecurringRangesByConfig(companyID, configID); err != nil {
		return nil, fmt.Errorf("failed to clear recurring ranges: %w", err)
	}
	for i := range input.RecurringRanges {
		input.RecurringRanges[i].BookingConfigID = configID
		input.RecurringRanges[i].CompanyID = companyID
		input.RecurringRanges[i].Appointments = nilIfLessThanOne(input.RecurringRanges[i].Appointments)

		if err := input.RecurringRanges[i].Insert(); err != nil {
			return nil, overlapOr(err, "failed to insert recurring range")
		}
	}

	// ---- Rewrite manual slots (from today forward) ----
	// "Today" of the company (its time zone, 2026-11-04), not of the server.
	if _, err := booking.DeleteManualSlotsFromTodayByConfig(companyID, configID, bltimezone.CompanyToday(companyID)); err != nil {
		return nil, fmt.Errorf("failed to clear future manual slots: %w", err)
	}
	for i := range input.ManualSlots {
		input.ManualSlots[i].BookingConfigID = configID
		input.ManualSlots[i].CompanyID = companyID
		input.ManualSlots[i].Appointments = nilIfLessThanOne(input.ManualSlots[i].Appointments)

		if input.ManualSlots[i].AllDay {
			input.ManualSlots[i].StartTime = nil
			input.ManualSlots[i].EndTime = nil
		} else {
			input.ManualSlots[i].StartTime = emptyToNilStr(input.ManualSlots[i].StartTime)
			input.ManualSlots[i].EndTime = emptyToNilStr(input.ManualSlots[i].EndTime)
		}

		if err := input.ManualSlots[i].Insert(); err != nil {
			return nil, overlapOr(err, "failed to insert manual slot")
		}
	}

	// ---- Return the fresh state ----
	return BSelectBookingFull(companyID, configID)
}
