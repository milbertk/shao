package blpublicshowcase

import (
	"log"
	"strings"
	"time"

	blpageconfiguration "github.com/milbertk/shao/bussunessLedger/blpageconfiguration"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
	"github.com/milbertk/shao/dataBasesLedger/weborder"
	emailnotifier "github.com/milbertk/shao/shared"
)

// MyOrderView is one web order in the customer's "My orders" list.
type MyOrderView struct {
	Reference    string    `json:"reference"`
	Status       string    `json:"status"`
	Total        float64   `json:"total"`
	ItemsCount   int       `json:"itemsCount"`
	ItemsSummary string    `json:"itemsSummary"`
	DeliveryMode string    `json:"deliveryMode"`
	CreatedAt    time.Time `json:"createdAt"`
	Token        string    `json:"token,omitempty"` // opens the "manage my request" page
}

// MyBookingView is one appointment in the customer's "My bookings" list.
type MyBookingView struct {
	ServiceName string    `json:"serviceName"`
	Date        string    `json:"date"`
	StartHour   string    `json:"startHour"`
	EndHour     string    `json:"endHour"`
	AllDay      bool      `json:"allDay"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	Token       string    `json:"token,omitempty"`
}

// MyRequestsView is what a signed-in customer sees in "My orders / My
// bookings" for ONE company.
type MyRequestsView struct {
	Email               string          `json:"email"`
	EmailVerified       bool            `json:"emailVerified"`
	CurrencySymbol      string          `json:"currencySymbol"`
	PaymentInstructions string          `json:"paymentInstructions,omitempty"`
	Orders              []MyOrderView   `json:"orders"`
	Bookings            []MyBookingView `json:"bookings"`
}

const myRequestsLimit = 100

// BMyRequests returns the web orders and bookings of the signed-in customer
// in one company.
//   - Orders made while signed in are matched by the user id.
//   - Orders and bookings made with the same email are matched ONLY when the
//     email is verified, so nobody can see another person's requests by
//     registering their address.
func BMyRequests(companyID, userID, email string, emailVerified bool) (*MyRequestsView, error) {
	companyID = strings.TrimSpace(companyID)
	userID = strings.TrimSpace(userID)
	email = strings.ToLower(strings.TrimSpace(email))

	store := blpageconfiguration.BCompanyStoreSettings(companyID)
	view := &MyRequestsView{
		Email:               email,
		EmailVerified:       emailVerified,
		CurrencySymbol:      store.CurrencySymbol,
		PaymentInstructions: store.PaymentInstructions,
		Orders:              []MyOrderView{},
		Bookings:            []MyBookingView{},
	}

	matchEmail := ""
	if emailVerified {
		matchEmail = email
	}

	if userID != "" || matchEmail != "" {
		orders, err := weborder.SelectWebOrdersForCustomer(companyID, userID, matchEmail, myRequestsLimit)
		if err != nil {
			return nil, err
		}
		for _, o := range orders {
			view.Orders = append(view.Orders, MyOrderView{
				Reference:    o.RequestCode,
				Status:       strings.ToUpper(strings.TrimSpace(o.Status)),
				Total:        o.Total,
				ItemsCount:   o.ItemsCount,
				ItemsSummary: o.ItemsSummary,
				DeliveryMode: o.DeliveryMode,
				CreatedAt:    o.CreatedAt,
				Token:        webOrderToken(companyID, o),
			})
		}
	}

	if matchEmail != "" {
		bookings, err := appointment.SelectAppointmentsForCustomer(companyID, matchEmail, myRequestsLimit)
		if err != nil {
			return nil, err
		}
		for _, b := range bookings {
			view.Bookings = append(view.Bookings, MyBookingView{
				ServiceName: b.ServiceName,
				Date:        b.AppointmentDate,
				StartHour:   b.StartHour,
				EndHour:     b.EndHour,
				AllDay:      b.StartHour == "00:00" && b.EndHour == "23:59",
				Status:      strings.ToUpper(strings.TrimSpace(b.Status)),
				CreatedAt:   b.CreatedAt,
				Token:       appointmentToken(companyID, b),
			})
		}
	}

	return view, nil
}

// webOrderToken returns the private link token, creating it when missing.
func webOrderToken(companyID string, o weborder.CustomerWebOrderRow) string {
	if o.PublicToken != nil && strings.TrimSpace(*o.PublicToken) != "" {
		return strings.TrimSpace(*o.PublicToken)
	}
	fresh, err := emailnotifier.NewPublicToken("w")
	if err != nil {
		return ""
	}
	token, err := weborder.EnsurePublicToken(companyID, o.WebOrderRequestID, fresh)
	if err != nil {
		log.Printf("[BMyRequests] web order token not created: %v", err)
		return ""
	}
	return token
}

func appointmentToken(companyID string, b appointment.CustomerAppointmentRow) string {
	if b.PublicToken != nil && strings.TrimSpace(*b.PublicToken) != "" {
		return strings.TrimSpace(*b.PublicToken)
	}
	fresh, err := emailnotifier.NewPublicToken("a")
	if err != nil {
		return ""
	}
	token, err := appointment.EnsurePublicToken(companyID, b.AppointmentID, fresh)
	if err != nil {
		log.Printf("[BMyRequests] appointment token not created: %v", err)
		return ""
	}
	return token
}
