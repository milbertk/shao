package blcustomers

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	customers "github.com/milbertk/shao/dataBasesLedger/customers"
)

// ErrCustomerNotFound is returned when no customer matches the criteria.
var ErrCustomerNotFound = errors.New("customer not found")

// BSelectCustomerByInternalID loads a customer by primary key within a company.
func BSelectCustomerByInternalID(companyID, customerInternalID string) (*customers.CustomerS, error) {
	companyID = strings.TrimSpace(companyID)
	customerInternalID = strings.TrimSpace(customerInternalID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if customerInternalID == "" {
		return nil, errors.New("customerInternalId is required")
	}

	customer := customers.CustomerS{
		CompanyID:          companyID,
		CustomerInternalID: customerInternalID,
	}

	if err := customer.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("failed to select customer by internal id: %w", err)
	}

	return &customer, nil
}

// BSelectCustomerByID loads a single customer by business ID within a company.
func BSelectCustomerByID(companyID, customerID string) (*customers.CustomerS, error) {
	customerID = strings.TrimSpace(customerID)
	if customerID == "" {
		return nil, errors.New("customerId is required")
	}

	return selectExactlyOne(
		companyID,
		customers.CustomerSearchFilters{CustomerID: customerID},
		func(c customers.CustomerS) bool {
			return strings.EqualFold(strings.TrimSpace(c.CustomerID), customerID)
		},
		"customerId",
	)
}

// BSelectCustomerByEmail loads a single customer by email within a company.
func BSelectCustomerByEmail(companyID, email string) (*customers.CustomerS, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, errors.New("email is required")
	}

	return selectExactlyOne(
		companyID,
		customers.CustomerSearchFilters{Email: email},
		func(c customers.CustomerS) bool {
			return c.Email != nil &&
				strings.EqualFold(strings.TrimSpace(*c.Email), email)
		},
		"email",
	)
}

// BSelectCustomerByPhone loads a single customer by phone within a company.
func BSelectCustomerByPhone(companyID, phone string) (*customers.CustomerS, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil, errors.New("phone is required")
	}

	return selectExactlyOne(
		companyID,
		customers.CustomerSearchFilters{Cellphone: phone},
		func(c customers.CustomerS) bool {
			return c.Cellphone != nil &&
				strings.TrimSpace(*c.Cellphone) == phone
		},
		"phone",
	)
}

// BSelectCustomerBySocialNetworkLink loads a single customer by social link.
func BSelectCustomerBySocialNetworkLink(companyID, link string) (*customers.CustomerS, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return nil, errors.New("socialNetworkLink is required")
	}

	return selectExactlyOne(
		companyID,
		customers.CustomerSearchFilters{SocialNetworkLink: link},
		func(c customers.CustomerS) bool {
			return c.SocialNetworkLink != nil &&
				strings.EqualFold(strings.TrimSpace(*c.SocialNetworkLink), link)
		},
		"socialNetworkLink",
	)
}

// selectExactlyOne runs a company-scoped targeted search and returns the first
// record that matches exactly, avoiding false partial ILIKE hits.
func selectExactlyOne(
	companyID string,
	filters customers.CustomerSearchFilters,
	isExactMatch func(customers.CustomerS) bool,
	fieldName string,
) (*customers.CustomerS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	filters.CompanyID = companyID
	filters.Limit = 50

	results, err := customers.SearchCustomers(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to select customer by %s: %w", fieldName, err)
	}

	for _, candidate := range results {
		if isExactMatch(candidate) {
			match := candidate
			return &match, nil
		}
	}

	return nil, ErrCustomerNotFound
}
