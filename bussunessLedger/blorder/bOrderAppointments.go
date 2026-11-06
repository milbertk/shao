package blorder

import (
	"errors"
	"fmt"
	"strings"

	order "github.com/milbertk/shao/dataBasesLedger/orders"
	"github.com/milbertk/shao/dataBasesLedger/products"
)

// ErrAppointmentInOrder (2026-10-31): a product marked "This is an
// appointment" (web_behavior BOOKING) is booked, never added to an order —
// neither in manual orders nor in store orders.
var ErrAppointmentInOrder = errors.New("invalid item: appointments are booked, they cannot be added to an order")

// IsAppointmentBehavior tells if a product web behavior is an appointment.
func IsAppointmentBehavior(webBehavior string) bool {
	return strings.EqualFold(strings.TrimSpace(webBehavior), "BOOKING")
}

func orderItemProductIDs(items []order.OrderItemS) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ProductID)
	}
	return ids
}

// BCheckNoAppointments returns ErrAppointmentInOrder (with the product name)
// when one of the products is an appointment. Unknown products are left to
// the normal checks.
func BCheckNoAppointments(companyID string, productIDs []string) error {
	seen := map[string]bool{}
	for _, id := range productIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		p := products.ProductS{ProductInternalID: id, CompanyID: strings.TrimSpace(companyID)}
		if err := p.Select(); err != nil {
			continue
		}
		if IsAppointmentBehavior(p.WebBehavior) {
			return fmt.Errorf("%w (%q)", ErrAppointmentInOrder, strings.TrimSpace(p.Name))
		}
	}
	return nil
}
