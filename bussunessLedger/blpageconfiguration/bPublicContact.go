package blpageconfiguration

import (
	"strings"

	emailnotifier "github.com/milbertk/shao/shared"
)

// Public contact of a company (2026-11-08): after a request that needs
// approval (booking, order, new time, cancellation) the store says "No email?
// Contact <company>" with the company WhatsApp, the customer service email
// (Company page) and the social link. The customer service email is already
// the Reply-To of every customer email, so it is public by design.

// PublicContactEmail returns the customer service email when it is a valid
// address, nil otherwise (pure, tested).
func PublicContactEmail(customerServiceEmail *string) *string {
	if customerServiceEmail == nil {
		return nil
	}
	value := strings.TrimSpace(*customerServiceEmail)
	if value == "" || !emailnotifier.ValidEmail(value) {
		return nil
	}
	return &value
}
