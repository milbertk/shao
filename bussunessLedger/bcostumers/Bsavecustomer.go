package blcustomers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/customers"
)

// emptyToNil returns nil when a *string points to an empty/blank value, so
// optional columns receive NULL instead of "" (which breaks typed columns like
// DATE). A trimmed non-empty value is returned as a pointer.
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

// normalizeOptionalFields converts blank optional string pointers to nil, so the
// database stores NULL. This fixes errors such as:
//
//	pq: invalid input syntax for type date: ""
func normalizeOptionalFields(c *customers.CustomerS) {
	c.Birthday = emptyToNil(c.Birthday)
	c.Alias = emptyToNil(c.Alias)
	c.Cellphone = emptyToNil(c.Cellphone)
	c.SocialNetworkLink = emptyToNil(c.SocialNetworkLink)
	c.Email = emptyToNil(c.Email)
	c.Address = emptyToNil(c.Address)
	c.Website = emptyToNil(c.Website)
	c.Comments = emptyToNil(c.Comments)
}

// BSaveCustomer creates or updates a customer inside a company.
//
//   - companyId is authoritative and always applied.
//   - If CustomerInternalID is empty OR does not exist in that company -> create.
//   - If it exists in that company -> partial update (only provided fields).
//
// The database enforces the real requirements (customer_id and name not blank,
// customer_id unique per company).
func BSaveCustomer(companyID string, input customers.CustomerS) (*customers.CustomerS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	// The company always comes from the validated context, never the body.
	input.CompanyID = companyID

	internalID := strings.TrimSpace(input.CustomerInternalID)

	// No id -> create directly.
	if internalID == "" {
		return createCustomer(input)
	}

	// Id provided -> try to load it within this company.
	existing, err := BSelectCustomerByInternalID(companyID, internalID)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			return createCustomer(input)
		}
		return nil, fmt.Errorf("failed to validate customer before saving: %w", err)
	}

	// Id exists -> partial update.
	merged := mergeCustomer(*existing, input)
	merged.CompanyID = companyID

	// Blank optional strings -> NULL.
	normalizeOptionalFields(&merged)

	if err := merged.Update(); err != nil {
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	return &merged, nil
}

// createCustomer inserts a new customer. CustomerInternalID is blanked so the
// database generates the primary key; CompanyID must already be set.
func createCustomer(input customers.CustomerS) (*customers.CustomerS, error) {
	newCustomer := input
	newCustomer.CustomerInternalID = ""

	if strings.TrimSpace(newCustomer.CompanyID) == "" {
		return nil, errors.New("companyId is required")
	}

	// Blank optional strings -> NULL (avoids "invalid input syntax for type date").
	normalizeOptionalFields(&newCustomer)

	if err := newCustomer.Insert(); err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	return &newCustomer, nil
}

// mergeCustomer overlays provided fields of input onto existing. The primary
// key and company are always kept from existing.
func mergeCustomer(existing customers.CustomerS, input customers.CustomerS) customers.CustomerS {
	merged := existing

	merged.CustomerInternalID = existing.CustomerInternalID
	merged.CompanyID = existing.CompanyID

	if v := strings.TrimSpace(input.CustomerID); v != "" {
		merged.CustomerID = v
	}
	if v := strings.TrimSpace(input.Name); v != "" {
		merged.Name = v
	}
	if v := strings.TrimSpace(input.SocialNetwork); v != "" {
		merged.SocialNetwork = v
	}
	if v := strings.TrimSpace(input.Status); v != "" {
		merged.Status = v
	}
	if v := strings.TrimSpace(input.Category); v != "" {
		merged.Category = v
	}
	if v := strings.TrimSpace(input.PreferredContactMethod); v != "" {
		merged.PreferredContactMethod = v
	}

	if input.Alias != nil {
		merged.Alias = input.Alias
	}
	if input.Cellphone != nil {
		merged.Cellphone = input.Cellphone
	}
	if input.SocialNetworkLink != nil {
		merged.SocialNetworkLink = input.SocialNetworkLink
	}
	if input.Birthday != nil {
		merged.Birthday = input.Birthday
	}
	if input.BirthYear != nil {
		merged.BirthYear = input.BirthYear
	}
	if input.Email != nil {
		merged.Email = input.Email
	}
	if input.Address != nil {
		merged.Address = input.Address
	}
	if input.Website != nil {
		merged.Website = input.Website
	}
	if input.Comments != nil {
		merged.Comments = input.Comments
	}

	return merged
}
