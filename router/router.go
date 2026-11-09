package router

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	usersinshao "github.com/milbertk/shao/HandlerLedgers/UsersInShao"
	appointmentsinshao "github.com/milbertk/shao/HandlerLedgers/appointmentInShao"
	"github.com/milbertk/shao/HandlerLedgers/appointmentmgmtinshao"
	"github.com/milbertk/shao/HandlerLedgers/appointmentrequestinshao"
	"github.com/milbertk/shao/HandlerLedgers/assistantinshao"
	publicshowcaseinshao "github.com/milbertk/shao/HandlerLedgers/hPublicShowCase"
	bookinginshao "github.com/milbertk/shao/HandlerLedgers/hbooking"
	"github.com/milbertk/shao/HandlerLedgers/locationsinshao"
	"github.com/milbertk/shao/HandlerLedgers/orderdetailsinshao"
	"github.com/milbertk/shao/HandlerLedgers/orderitemtrackinginshao"
	"github.com/milbertk/shao/HandlerLedgers/ordertrackinginshao"
	"github.com/milbertk/shao/HandlerLedgers/pageconfigurationinshao"
	"github.com/milbertk/shao/HandlerLedgers/planinshao"
	"github.com/milbertk/shao/HandlerLedgers/reportsinshao"
	"github.com/milbertk/shao/HandlerLedgers/steptemplateinshao"
	"github.com/milbertk/shao/HandlerLedgers/storageinshao"
	trackinginshao "github.com/milbertk/shao/HandlerLedgers/trackinMaintenanceInShao"
	"github.com/milbertk/shao/HandlerLedgers/weborderinshao"

	categoriesinshao "github.com/milbertk/shao/HandlerLedgers/categories"
	customersinshao "github.com/milbertk/shao/HandlerLedgers/customersInShao"
	ordersinshao "github.com/milbertk/shao/HandlerLedgers/ordersInShao"
	productsinshao "github.com/milbertk/shao/HandlerLedgers/productsInShao"

	"github.com/milbertk/shao/bussunessLedger/blplan"
	"github.com/milbertk/shao/middlewares"
)

// Keep this list restricted to frontend origins that you trust.
// These local / LAN origins are used only when APP_ENV is NOT "production"
// (see cors_config.go, 2026-10-25).
var allowedOrigins = map[string]bool{
	"http://localhost:3000":       true,
	"http://localhost:3001":       true,
	"http://localhost:3002":       true,
	"http://127.0.0.1:3000":       true,
	"http://192.168.100.110:3000": true,
	"http://192.168.100.110:3001": true,
	"http://192.168.100.110":      true,
}

const (
	allowedMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowedHeaders = "Accept, Authorization, Content-Type, X-Requested-With, X-Captcha-Token"
	maxAgeSeconds  = "3600"
)

// init loads the CORS rules for the current environment (APP_ENV):
// production = only CORS_ALLOWED_ORIGINS + the origins file; test = any
// origin; not set = the local list + CORS_ALLOWED_ORIGINS (as before).
func init() {
	allowedOrigins = loadCORSOrigins(allowedOrigins)
}

func isOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	if corsAllowAnyOrigin() {
		return true
	}
	return allowedOrigins[origin] || originAllowedByDomain(origin)
}

func writeCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if !isOriginAllowed(origin) {
		return
	}

	header := w.Header()
	header.Set("Access-Control-Allow-Origin", origin)
	header.Add("Vary", "Origin")
	header.Set("Access-Control-Allow-Credentials", "true")
	header.Set("Access-Control-Allow-Methods", allowedMethods)
	header.Set("Access-Control-Allow-Headers", allowedHeaders)
	header.Set("Access-Control-Max-Age", maxAgeSeconds)
}

func corsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCORS(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewRouter() http.Handler {
	r := mux.NewRouter()

	// One log line per API call + request id + panic -> clean 500 (2026-11-01).
	// Runs after the route is matched (it logs the route template, not the path).
	r.Use(middlewares.RequestLog)

	r.HandleFunc("/", HomeHandler).Methods(http.MethodGet)
	r.HandleFunc("/health", HealthHandler).Methods(http.MethodGet)

	// PERMISSIONS (2026-10-24): every route that CREATES, CHANGES or DELETES
	// data needs the NAV_* permission of its screen (several codes separated
	// by commas = any of them). OWNER / MANAGER always pass. Searches and
	// lookups (GET) stay open to every member because several screens use
	// them (e.g. New order searches products and customers); reports need
	// NAV_REPORTS_ACCESS.

	// =================================================================
	//  COMPANY USERS (management)  -> OWNER / MANAGER only.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/company-users",
		security.RequireActiveCompanyUser(
			middlewares.RequireActivePlan( // read-only without an active plan
				http.HandlerFunc(usersinshao.HCreateOrUpdateCompanyUser),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/company-users/permissions",
		security.RequireActiveCompanyUser(
			middlewares.RequireActivePlan( // read-only without an active plan
				http.HandlerFunc(usersinshao.HReplaceCompanyUserPermissions),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/company-users/search",
		security.RequireActiveCompanyUser(
			http.HandlerFunc(usersinshao.HSearchCompanyUsers),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/company-users/{companyUserId}/permissions",
		security.RequireActiveCompanyUser(
			http.HandlerFunc(usersinshao.HSelectCompanyUserPermissions),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/company-users/{companyUserId}",
		security.RequireActiveCompanyUser(
			http.HandlerFunc(usersinshao.HSelectCompanyUser),
		),
	).Methods(http.MethodGet)

	// =================================================================
	//  MY ACCESS  -> any active company member (menu options to show).
	// =================================================================
	r.Handle(
		"/companies/{companyId}/my-access",
		security.RequireActiveCompanyMember(
			http.HandlerFunc(usersinshao.HSelectMyAccess),
		),
	).Methods(http.MethodGet)

	// =================================================================
	//  COMPANY PLAN  -> any active company member.
	//  GET: current plan + usage + plans with prices + pending request.
	//  POST plan-requests: "Request a plan" form (saved + emailed).
	// =================================================================
	r.Handle(
		"/companies/{companyId}/plan",
		security.RequireActiveCompanyMember(
			http.HandlerFunc(planinshao.HSelectCompanyPlan),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/plan-requests",
		security.RequireActiveCompanyMember(
			http.HandlerFunc(planinshao.HCreatePlanRequest),
		),
	).Methods(http.MethodPost)

	// =================================================================
	//  COMPANY STORAGE (2026-10-23)  -> any active company member.
	//  Files go from the app to Firebase; shao counts their size per company
	//  and compares it with the plan storage (tbl_plans.storage_gb).
	//  check: before uploading; files: register after uploading (images);
	//  files/delete: stop counting a deleted file (always allowed).
	// =================================================================
	r.Handle(
		"/companies/{companyId}/storage",
		security.RequireActiveCompanyMember(
			http.HandlerFunc(storageinshao.HSelectStorageUsage),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/storage/check",
		security.RequireActiveCompanyMember(
			http.HandlerFunc(storageinshao.HCheckStorage),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/storage/files",
		security.RequireActiveCompanyMember(
			// Product images and company page images (2026-10-24).
			// No RequireActivePlan here (2026-10-29): BRegisterFile checks the
			// plan itself and lets a company without a plan register only its
			// basic company images (logo, header). Products stay blocked.
			middlewares.RequireOptionPermission("NAV_PRODUCTS_ACCESS,NAV_PAGE_CONFIGURATION_ACCESS")(
				http.HandlerFunc(storageinshao.HRegisterStorageFile),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/storage/files/delete",
		security.RequireActiveCompanyMember(
			http.HandlerFunc(storageinshao.HDeleteStorageFile),
		),
	).Methods(http.MethodPost)

	// =================================================================
	//  CUSTOMERS  -> any active company member + option-permission gate.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/customers",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_CUSTOMERS_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(customersinshao.HSaveCustomer),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/customers/search",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(customersinshao.HSearchCustomers),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/customers/lookup",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(customersinshao.HSelectCustomer),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/customers/{customerInternalId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(customersinshao.HSelectCustomer),
			),
		),
	).Methods(http.MethodGet)

	// =================================================================
	//  CATEGORIES  -> any active company member + option-permission gate.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/categories",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_CATEGORIES_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(categoriesinshao.HSaveCategory),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/categories/search",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(categoriesinshao.HSearchCategories),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/categories/{categoryInternalId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(categoriesinshao.HSelectCategory),
			),
		),
	).Methods(http.MethodGet)

	// =================================================================
	//  PRODUCTS  -> any active company member + option-permission gate.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/products",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_PRODUCTS_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(productsinshao.HSaveProduct),
				),
			),
		),
	).Methods(http.MethodPost)
	r.Handle(
		"/companies/{companyId}/products/search",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(productsinshao.HSearchProducts),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/products/{productInternalId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(productsinshao.HSelectProduct),
			),
		),
	).Methods(http.MethodGet)

	// =================================================================
	//  BOOKINGS  -> any active company member + option-permission gate.
	//  Order (mux): fixed "search" BEFORE "{bookingConfigId}".
	//  Replace "" with "NAV_BOOKING_ACCESS" later to enable the gate.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/bookings",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_BOOKING_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(bookinginshao.HSaveBooking),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/bookings/search",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(bookinginshao.HSearchBookings),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/bookings/{bookingConfigId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(bookinginshao.HSelectBooking),
			),
		),
	).Methods(http.MethodGet)

	// =================================================================
	//  LOCATIONS ("Sedes", 2026-11-07) -> any active company member.
	//  Reading is open to every member (the booking and appointment forms
	//  need the list); changes need NAV_LOCATIONS_ACCESS or NAV_BOOKING_ACCESS
	//  (OWNER / MANAGER always). Order (mux): ".../active" is its own path.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/locations",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(locationsinshao.HListLocations),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/locations",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_LOCATIONS_ACCESS,NAV_BOOKING_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(locationsinshao.HSaveLocation),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/locations/{locationId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(locationsinshao.HSelectLocation),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/locations/{locationId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_LOCATIONS_ACCESS,NAV_BOOKING_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(locationsinshao.HDeleteLocation),
				),
			),
		),
	).Methods(http.MethodDelete)

	r.Handle(
		"/companies/{companyId}/locations/{locationId}/active",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_LOCATIONS_ACCESS,NAV_BOOKING_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(locationsinshao.HSetLocationActive),
				),
			),
		),
	).Methods(http.MethodPut)

	r.Handle(
		"/companies/{companyId}/booking-settings",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(locationsinshao.HSelectBookingSettings),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/booking-settings",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_LOCATIONS_ACCESS,NAV_BOOKING_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(locationsinshao.HSaveBookingSettings),
				),
			),
		),
	).Methods(http.MethodPut)

	// =================================================================
	//  TRACKING  -> any active company member + option-permission gate.
	//  Order (mux): fixed "search" BEFORE the dynamic "{trackingConfigId}".
	//  Replace "" with "NAV_TRACKING_MAINTENANCE_ACCESS" later to enable it.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/trackings",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_TRACKING_MAINTENANCE_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(trackinginshao.HSaveTracking),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/trackings/search",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(trackinginshao.HSearchTracking),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/trackings/{trackingConfigId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(trackinginshao.HSelectTracking),
			),
		),
	).Methods(http.MethodGet)

	// =================================================================
	//  ORDERS  -> any active company member + option-permission gate.
	//  Order (mux): fixed "search" BEFORE the dynamic "{orderInternalId}".
	//  Replace "" with "NAV_ORDER_CREATE_ACCESS" later to enable the gate.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/orders",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_ORDER_CREATE_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(ordersinshao.HSaveOrder),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/orders/search",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(ordersinshao.HSearchOrders),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/orders/{orderInternalId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(ordersinshao.HSelectOrder),
			),
		),
	).Methods(http.MethodGet)

	// =================================================================
	//  APPOINTMENTS  -> any active company member + option-permission gate.
	//  Order (mux): fixed "search" BEFORE the dynamic "{appointmentId}".
	//  The DELETE and GET share the "{appointmentId}" path (different methods).
	//  Replace "" with the real NAV_* code later to enable the gate.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/appointments",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_SELF_BOOKING,NAV_APPOINTMENTS_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(appointmentsinshao.HSaveAppointment),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/appointments/search",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(appointmentsinshao.HSearchAppointments),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/appointments/{appointmentId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(appointmentsinshao.HSelectAppointment),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/appointments/{appointmentId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_SELF_BOOKING,NAV_APPOINTMENTS_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(appointmentsinshao.HDeleteAppointment),
				),
			),
		),
	).Methods(http.MethodDelete)

	// Location of an appointment or booking request (2026-11-08): from
	// Appointments, Booking requests (approval) and Book for a customer.
	r.Handle(
		"/companies/{companyId}/appointments/{appointmentId}/location",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_APPOINTMENTS_ACCESS,NAV_APPROVAL_B_ACCESS,NAV_SELF_BOOKING")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(appointmentsinshao.HSetAppointmentLocation),
				),
			),
		),
	).Methods(http.MethodPut)

	// =================================================================
	//  PAGE CONFIGURATION  -> one customer-page configuration per company.
	//  GET loads it (empty result when not created yet); PUT upserts it.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/page-configuration",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(pageconfigurationinshao.HSelectPageConfiguration),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/page-configuration",
		security.RequireActiveCompanyMember(
			// Allowed WITHOUT a plan (2026-10-29): the basic company (name, logo,
			// header) is set up here before requesting a plan. The company SMTP
			// below and every other option stay read-only without a plan.
			middlewares.RequireOptionPermission("NAV_PAGE_CONFIGURATION_ACCESS")(
				http.HandlerFunc(pageconfigurationinshao.HSavePageConfiguration),
			),
		),
	).Methods(http.MethodPut)

	// =================================================================
	//  COMPANY SMTP  -> the company's own mail server for notifications.
	//  The password is write-only (never returned) and stored encrypted.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/smtp-configuration",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_PAGE_CONFIGURATION_ACCESS")(
				http.HandlerFunc(pageconfigurationinshao.HSelectCompanySmtp),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/smtp-configuration",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_PAGE_CONFIGURATION_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(pageconfigurationinshao.HSaveCompanySmtp),
				),
			),
		),
	).Methods(http.MethodPut)

	r.Handle(
		"/companies/{companyId}/smtp-configuration/test",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_PAGE_CONFIGURATION_ACCESS")(
				http.HandlerFunc(pageconfigurationinshao.HTestCompanySmtp),
			),
		),
	).Methods(http.MethodPost)

	// =================================================================
	//  AI ASSISTANT (Aleph)  -> any logged-in active company member.
	//  No option-permission gate for now (empty code passes through).
	// =================================================================
	r.Handle(
		"/companies/{companyId}/assistant/ask",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(assistantinshao.HAskAssistant),
			),
		),
	).Methods(http.MethodPost)

	// =================================================================
	//  PUBLIC SHOWCASE  -> UNAUTHENTICATED, read-only.
	// =================================================================
	// {companyId} on the public store routes accepts the internal id OR the
	// company's slug (middlewares.ResolvePublicCompany turns a slug into the id).

	// Share link with preview (WhatsApp / Facebook): /s/{slug or id}.
	r.HandleFunc(
		"/s/{companyKey}",
		publicshowcaseinshao.HPublicSharePage,
	).Methods(http.MethodGet, http.MethodHead)

	r.Handle(
		"/public/companies/{companyId}/showcase",
		middlewares.ResolvePublicCompany(
			http.HandlerFunc(publicshowcaseinshao.HPublicShowcase),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/public/companies/{companyId}/page-configuration",
		middlewares.ResolvePublicCompany(
			http.HandlerFunc(publicshowcaseinshao.HPublicPageConfiguration),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/public/companies/{companyId}/products/{productId}/slots",
		middlewares.ResolvePublicCompany(
			// Store bookings only on plans with web_bookings (tbl_plans).
			middlewares.RequirePlanFeature(blplan.FeatureWebBookings)(
				http.HandlerFunc(publicshowcaseinshao.HPublicBookingSlots),
			),
		),
	).Methods(http.MethodGet)

	// PUBLIC APPOINTMENT REQUEST route (UNAUTHENTICATED).
	r.Handle(
		"/public/companies/{companyId}/appointment-requests",
		middlewares.ResolvePublicCompany(
			// No active plan -> the store does not take orders (read-only);
			// then store bookings only on plans with web_bookings (tbl_plans).
			middlewares.RequireActivePlan(
				middlewares.RequirePlanFeature(blplan.FeatureWebBookings)(
					http.HandlerFunc(publicshowcaseinshao.HPublicAppointmentRequest),
				),
			),
		),
	).Methods(http.MethodPost)

	// PUBLIC AI ASSISTANT (Aleph, shopping assistant) route (UNAUTHENTICATED).
	// Reads only the public catalog; never places orders or booking requests.
	r.Handle(
		"/public/companies/{companyId}/assistant/ask",
		middlewares.ResolvePublicCompany(
			http.HandlerFunc(publicshowcaseinshao.HPublicAssistant),
		),
	).Methods(http.MethodPost)

	// MY ORDERS / MY BOOKINGS of the signed-in store customer (session cookie
	// required; {companyId} = id or slug).
	r.Handle(
		"/public/companies/{companyId}/my-requests",
		security.RequireActiveUser(
			middlewares.ResolvePublicCompany(
				http.HandlerFunc(publicshowcaseinshao.HPublicMyRequests),
			),
		),
	).Methods(http.MethodGet)

	// CUSTOMER SELF-SERVICE ("manage my request" link sent by email).
	// PUBLIC: the long random token is the key. View, cancellation request
	// (email to the company, manual cancel), new time.
	r.HandleFunc(
		"/public/self-service/{token}",
		publicshowcaseinshao.HPublicSelfServiceView,
	).Methods(http.MethodGet)

	r.HandleFunc(
		"/public/self-service/{token}/cancel-request",
		publicshowcaseinshao.HPublicSelfServiceCancelRequest,
	).Methods(http.MethodPost)

	r.HandleFunc(
		"/public/self-service/{token}/reschedule",
		publicshowcaseinshao.HPublicSelfServiceReschedule,
	).Methods(http.MethodPost)

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/service", ServiceHandler).Methods(http.MethodGet)

	r.Handle(
		"/public/companies/{companyId}/web-order-requests",
		middlewares.ResolvePublicCompany(
			// No active plan -> the store does not take orders (read-only).
			middlewares.RequireActivePlan(
				http.HandlerFunc(publicshowcaseinshao.HPublicWebOrderRequest),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/web-orders/search",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(weborderinshao.HSearchWebOrders),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/web-orders/{webOrderRequestId}/status",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_APPROVAL_A_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(weborderinshao.HUpdateWebOrderStatus),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/web-orders/{webOrderRequestId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(weborderinshao.HSelectWebOrder),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/web-orders/{webOrderRequestId}/approve",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_APPROVAL_A_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(weborderinshao.HApproveWebOrder),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/order-trackings/stages",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_ORDER_TRACKING_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(ordertrackinginshao.HSaveOrderTrackingStage),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/order-trackings/{orderInternalId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(ordertrackinginshao.HSelectOrderTracking),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/step-templates",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_TRACKING_MAINTENANCE_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(steptemplateinshao.HSaveStepTemplate),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/step-templates/search",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(steptemplateinshao.HSearchStepTemplates),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/step-templates/{stepTemplateId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(steptemplateinshao.HSelectStepTemplate),
			),
		),
	).Methods(http.MethodGet)

	// Delete an unused process / enable or disable a process (2026-10-30).
	r.Handle(
		"/companies/{companyId}/step-templates/{stepTemplateId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_TRACKING_MAINTENANCE_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(steptemplateinshao.HDeleteStepTemplate),
				),
			),
		),
	).Methods(http.MethodDelete)

	r.Handle(
		"/companies/{companyId}/step-templates/{stepTemplateId}/active",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_TRACKING_MAINTENANCE_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(steptemplateinshao.HSetStepTemplateActive),
				),
			),
		),
	).Methods(http.MethodPut)

	r.Handle(
		"/companies/{companyId}/orders/{orderInternalId}/item-tracking",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(orderitemtrackinginshao.HSelectOrderItemTracking),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/orders/{orderInternalId}/item-tracking",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_ORDER_TRACKING_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(orderitemtrackinginshao.HUpdateOrderItemStep),
				),
			),
		),
	).Methods(http.MethodPost)

	// GET order details (comments + attachments)
	r.Handle(
		"/companies/{companyId}/orders/{orderInternalId}/details",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("")(
				http.HandlerFunc(orderdetailsinshao.HSelectOrderDetails),
			),
		),
	).Methods(http.MethodGet)

	// POST add a comment
	r.Handle(
		"/companies/{companyId}/orders/{orderInternalId}/comments",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_ORDER_TRACKING_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(orderdetailsinshao.HAddOrderComment),
				),
			),
		),
	).Methods(http.MethodPost)

	// DELETE a comment
	r.Handle(
		"/companies/{companyId}/orders/{orderInternalId}/comments/{commentId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_ORDER_TRACKING_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(orderdetailsinshao.HDeleteOrderComment),
				),
			),
		),
	).Methods(http.MethodDelete)

	// POST record uploaded attachments (Firebase URLs + metadata)
	r.Handle(
		"/companies/{companyId}/orders/{orderInternalId}/attachments",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_ORDER_TRACKING_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(orderdetailsinshao.HAddOrderAttachments),
				),
			),
		),
	).Methods(http.MethodPost)

	// DELETE an attachment (returns storagePath so the client can delete from Firebase)
	r.Handle(
		"/companies/{companyId}/orders/{orderInternalId}/attachments/{attachmentId}",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_ORDER_TRACKING_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(orderdetailsinshao.HDeleteOrderAttachment),
				),
			),
		),
	).Methods(http.MethodDelete)

	// =================================================================
	//  APPOINTMENT REQUEST management routes (AUTHENTICATED).
	// =================================================================
	r.Handle(
		"/companies/{companyId}/appointment-requests/{appointmentId}/approve",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_APPROVAL_B_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(appointmentrequestinshao.HApproveAppointmentRequest),
				),
			),
		),
	).Methods(http.MethodPost)

	r.Handle(
		"/companies/{companyId}/appointment-requests/{appointmentId}/reject",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_APPROVAL_B_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(appointmentrequestinshao.HRejectAppointmentRequest),
				),
			),
		),
	).Methods(http.MethodPost)

	// =================================================================
	//  APPOINTMENT MANAGEMENT route (AUTHENTICATED).
	// =================================================================
	r.Handle(
		"/companies/{companyId}/appointments/{appointmentId}/status",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_APPOINTMENTS_ACCESS")(
				middlewares.RequireActivePlan( // read-only without an active plan
					http.HandlerFunc(appointmentmgmtinshao.HChangeAppointmentStatus),
				),
			),
		),
	).Methods(http.MethodPost)

	// =================================================================
	//  REPORTS routes (read-only GET).
	//  Wrapped in RequireActiveCompanyMember so the handlers get the
	//  validated company context (security.GetActiveCompanyID). Replace
	//  "" with "NAV_REPORTS_ACCESS" later to enable the permission gate.
	// =================================================================
	r.Handle(
		"/companies/{companyId}/reports/orders-by-status",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_REPORTS_ACCESS")(
				http.HandlerFunc(reportsinshao.HOrdersByStatus),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/reports/appointments-summary",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_REPORTS_ACCESS")(
				// Reports need all_reports in tbl_plans (every plan has it since 2026-10-23).
				middlewares.RequirePlanFeature(blplan.FeatureAllReports)(
					http.HandlerFunc(reportsinshao.HAppointmentsSummary),
				),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/reports/top-customers",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_REPORTS_ACCESS")(
				// Reports need all_reports in tbl_plans (every plan has it since 2026-10-23).
				middlewares.RequirePlanFeature(blplan.FeatureAllReports)(
					http.HandlerFunc(reportsinshao.HTopCustomers),
				),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/reports/best-sellers",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_REPORTS_ACCESS")(
				// Reports need all_reports in tbl_plans (every plan has it since 2026-10-23).
				middlewares.RequirePlanFeature(blplan.FeatureAllReports)(
					http.HandlerFunc(reportsinshao.HBestSellers),
				),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/reports/aging-orders",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_REPORTS_ACCESS")(
				// Reports need all_reports in tbl_plans (every plan has it since 2026-10-23).
				middlewares.RequirePlanFeature(blplan.FeatureAllReports)(
					http.HandlerFunc(reportsinshao.HAgingOrders),
				),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/reports/cancellation-rate",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_REPORTS_ACCESS")(
				// Reports need all_reports in tbl_plans (every plan has it since 2026-10-23).
				middlewares.RequirePlanFeature(blplan.FeatureAllReports)(
					http.HandlerFunc(reportsinshao.HCancellationRate),
				),
			),
		),
	).Methods(http.MethodGet)

	r.Handle(
		"/companies/{companyId}/reports/revenue-over-time",
		security.RequireActiveCompanyMember(
			middlewares.RequireOptionPermission("NAV_REPORTS_ACCESS")(
				// Reports need all_reports in tbl_plans (every plan has it since 2026-10-23).
				middlewares.RequirePlanFeature(blplan.FeatureAllReports)(
					http.HandlerFunc(reportsinshao.HRevenueOverTime),
				),
			),
		),
	).Methods(http.MethodGet)

	return corsHandler(r)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Backend running correctly",
	})
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func ServiceHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "ServiceHub service endpoint",
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
