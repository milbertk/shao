package bllocations

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/milbertk/shao/bussunessLedger/bltimezone"
	"github.com/milbertk/shao/dataBasesLedger/locations"
)

// Company locations ("Sedes", 2026-11-07): the places where a company
// attends appointments. A location is optional on every schedule range,
// manual slot and appointment.

var (
	// ErrLocationNotFound: the location does not exist in this company.
	ErrLocationNotFound = errors.New("location not found")
	// ErrLocationDisabled: a disabled location can not be used for new hours
	// or new appointments.
	ErrLocationDisabled = errors.New("invalid location: this location is disabled")
)

const (
	maxNameLength          = 120
	maxAddressLength       = 300
	maxDetailsLength       = 500
	maxLocationsPerCompany = 50 // safety limit (a company normally has a few)
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidID tells if the value looks like a location id (UUID).
func ValidID(id string) bool {
	return uuidPattern.MatchString(strings.TrimSpace(id))
}

// LocationView is one location as the screens show it.
type LocationView struct {
	locations.CompanyLocationS
	MapsURL string                  `json:"mapsUrl,omitempty"`
	Usage   locations.LocationUsage `json:"usage"`
}

// LocationInput is what the Locations screen sends to save one location.
type LocationInput struct {
	LocationID string   `json:"locationId"`
	Name       string   `json:"name"`
	Address    *string  `json:"address"`
	Details    *string  `json:"details"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	Active     *bool    `json:"active"` // nil = keep (new: active)
}

// DeleteResult tells what happened when deleting a location.
type DeleteResult struct {
	Deleted  bool                    `json:"deleted"`  // removed
	Disabled bool                    `json:"disabled"` // used -> disabled instead
	Usage    locations.LocationUsage `json:"usage"`
}

// ActiveResult is the answer of enabling / disabling a location.
type ActiveResult struct {
	Location LocationView `json:"location"`
	// FutureAppointments: appointments from today in this location. They keep
	// their location; the store just stops offering it (warning only).
	FutureAppointments int `json:"futureAppointments"`
}

func trimmedOrNil(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return &v
}

func runeLen(value *string) int {
	if value == nil {
		return 0
	}
	return len([]rune(*value))
}

// validatePoint checks the map point: both values or none, inside the world.
func validatePoint(latitude, longitude *float64) error {
	if (latitude == nil) != (longitude == nil) {
		return errors.New("invalid map point: give both latitude and longitude, or none")
	}
	if latitude == nil {
		return nil
	}
	lat, lng := *latitude, *longitude
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) {
		return errors.New("invalid map point")
	}
	if lat < -90 || lat > 90 {
		return errors.New("invalid latitude: it must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return errors.New("invalid longitude: it must be between -180 and 180")
	}
	if lat == 0 && lng == 0 {
		return errors.New("invalid map point: 0,0 is not a real place")
	}
	return nil
}

// ValidateLocationInput normalizes and checks one location (pure, tested).
func ValidateLocationInput(input *LocationInput) error {
	input.LocationID = strings.TrimSpace(input.LocationID)
	input.Name = strings.Join(strings.Fields(input.Name), " ")
	input.Address = trimmedOrNil(input.Address)
	input.Details = trimmedOrNil(input.Details)

	if input.LocationID != "" && !ValidID(input.LocationID) {
		return errors.New("invalid locationId")
	}
	if input.Name == "" {
		return errors.New("name is required")
	}
	if len([]rune(input.Name)) > maxNameLength {
		return fmt.Errorf("invalid name: maximum %d characters", maxNameLength)
	}
	if runeLen(input.Address) > maxAddressLength {
		return fmt.Errorf("invalid address: maximum %d characters", maxAddressLength)
	}
	if runeLen(input.Details) > maxDetailsLength {
		return fmt.Errorf("invalid details: maximum %d characters", maxDetailsLength)
	}
	return validatePoint(input.Latitude, input.Longitude)
}

// toView adds the Maps link and the usage. today is the company's date.
func toView(companyID, today string, l locations.CompanyLocationS) LocationView {
	view := LocationView{CompanyLocationS: l, MapsURL: MapsURL(l.Address, l.Latitude, l.Longitude)}
	if usage, err := locations.CountLocationUsage(companyID, l.LocationID, today); err == nil {
		view.Usage = usage
	}
	return view
}

// BListLocations returns the locations of the company (with where each one
// is used). includeInactive=false returns only the active ones.
func BListLocations(companyID string, includeInactive bool) ([]LocationView, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	list, err := locations.SelectLocationsByCompany(companyID, includeInactive)
	if err != nil {
		return nil, err
	}
	today := bltimezone.CompanyToday(companyID)
	views := make([]LocationView, 0, len(list))
	for _, l := range list {
		views = append(views, toView(companyID, today, l))
	}
	return views, nil
}

// loadLocation returns one location of the company or ErrLocationNotFound.
func loadLocation(companyID, locationID string) (*locations.CompanyLocationS, error) {
	locationID = strings.TrimSpace(locationID)
	if !ValidID(locationID) {
		return nil, ErrLocationNotFound
	}
	l := locations.CompanyLocationS{LocationID: locationID, CompanyID: companyID}
	if err := l.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLocationNotFound
		}
		return nil, err
	}
	return &l, nil
}

// BSelectLocation returns one location of the company.
func BSelectLocation(companyID, locationID string) (*LocationView, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	l, err := loadLocation(companyID, locationID)
	if err != nil {
		return nil, err
	}
	view := toView(companyID, bltimezone.CompanyToday(companyID), *l)
	return &view, nil
}

// BSaveLocation creates (empty locationId) or updates one location.
func BSaveLocation(companyID string, input LocationInput) (*LocationView, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if err := ValidateLocationInput(&input); err != nil {
		return nil, err
	}

	exists, err := locations.NameExists(companyID, input.Name, input.LocationID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("a location named %q already exists", input.Name)
	}

	item := locations.CompanyLocationS{
		LocationID: input.LocationID,
		CompanyID:  companyID,
		Name:       input.Name,
		Address:    input.Address,
		Details:    input.Details,
		Latitude:   input.Latitude,
		Longitude:  input.Longitude,
		Active:     true,
	}

	if input.LocationID == "" {
		count, err := locations.CountLocations(companyID)
		if err != nil {
			return nil, err
		}
		if count >= maxLocationsPerCompany {
			return nil, fmt.Errorf("invalid: a company can have at most %d locations", maxLocationsPerCompany)
		}
		if input.Active != nil {
			item.Active = *input.Active
		}
		if err := item.Insert(); err != nil {
			return nil, uniqueNameOr(input.Name, err)
		}
	} else {
		current, err := loadLocation(companyID, input.LocationID)
		if err != nil {
			return nil, err
		}
		item.Active = current.Active
		if input.Active != nil {
			item.Active = *input.Active
		}
		if err := item.Update(); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrLocationNotFound
			}
			return nil, uniqueNameOr(input.Name, err)
		}
	}

	saved, err := loadLocation(companyID, item.LocationID)
	if err != nil {
		return nil, err
	}
	view := toView(companyID, bltimezone.CompanyToday(companyID), *saved)
	return &view, nil
}

// uniqueNameOr turns a unique-index error (two users saving the same name at
// the same moment) into the friendly "already exists".
func uniqueNameOr(name string, err error) error {
	m := strings.ToLower(err.Error())
	if strings.Contains(m, "ux_company_location_name") || strings.Contains(m, "duplicate key") {
		return fmt.Errorf("a location named %q already exists", name)
	}
	return err
}

// BDeleteLocation deletes a location that nothing uses. A used location is
// DISABLED instead (its history stays), and the result says so.
func BDeleteLocation(companyID, locationID string) (*DeleteResult, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	l, err := loadLocation(companyID, locationID)
	if err != nil {
		return nil, err
	}

	usage, err := locations.CountLocationUsage(companyID, l.LocationID, bltimezone.CompanyToday(companyID))
	if err != nil {
		return nil, err
	}
	result := &DeleteResult{Usage: usage}

	if !usage.Used() {
		err := locations.DeleteLocation(companyID, l.LocationID)
		if err == nil {
			result.Deleted = true
			return result, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLocationNotFound
		}
		// Someone used it a moment ago (foreign key): disable it instead.
		if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			return nil, err
		}
	}

	if err := locations.SetLocationActive(companyID, l.LocationID, false); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLocationNotFound
		}
		return nil, err
	}
	result.Disabled = true
	return result, nil
}

// BSetLocationActive enables or disables a location. Disabling is allowed
// even with future appointments (they keep their location); the answer
// tells how many there are, so the screen can warn.
func BSetLocationActive(companyID, locationID string, active bool) (*ActiveResult, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	l, err := loadLocation(companyID, locationID)
	if err != nil {
		return nil, err
	}

	if active && !l.Active {
		// Enabling again: with one location at a time, its hours must not
		// clash with the hours of the other locations.
		if err := checkEnableWithoutOverlap(companyID, l.LocationID); err != nil {
			return nil, err
		}
	}

	if err := locations.SetLocationActive(companyID, l.LocationID, active); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLocationNotFound
		}
		return nil, err
	}
	l.Active = active
	view := toView(companyID, bltimezone.CompanyToday(companyID), *l)
	return &ActiveResult{Location: view, FutureAppointments: view.Usage.FutureAppointments}, nil
}

// ValidateLocationForCompany checks an optional location id used by a
// schedule or an appointment. nil/empty = no location (returns nil, nil).
// requireActive refuses a disabled location.
func ValidateLocationForCompany(companyID string, locationID *string, requireActive bool) (*locations.CompanyLocationS, error) {
	if locationID == nil || strings.TrimSpace(*locationID) == "" {
		return nil, nil
	}
	l, err := loadLocation(strings.TrimSpace(companyID), *locationID)
	if err != nil {
		if errors.Is(err, ErrLocationNotFound) {
			return nil, fmt.Errorf("invalid location: %q does not exist in this company", strings.TrimSpace(*locationID))
		}
		return nil, err
	}
	if requireActive && !l.Active {
		return nil, ErrLocationDisabled
	}
	return l, nil
}

// CompanyLocations returns every location of the company by id (active and
// disabled), for screens and emails that show names.
func CompanyLocations(companyID string) (map[string]locations.CompanyLocationS, error) {
	list, err := locations.SelectLocationsByCompany(strings.TrimSpace(companyID), true)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]locations.CompanyLocationS, len(list))
	for _, l := range list {
		byID[strings.ToLower(l.LocationID)] = l
	}
	return byID, nil
}

// NormalizeID returns the trimmed id, or nil for empty values.
func NormalizeID(locationID *string) *string {
	return trimmedOrNil(locationID)
}
