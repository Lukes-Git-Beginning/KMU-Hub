package gateway

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/security"
	"github.com/kmuhub/kmuhub/internal/server/response"
	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// BookingRoutes handles HTTP routes for booking-page administration.
// Public booking routes are registered via RegisterPublicRoutes (called outside
// the registrars loop, following the guest-routes pattern).
// Both admin and public routes reuse the "work" gRPC connection (calendar service).
type BookingRoutes struct {
	registry *ServiceRegistry
	captcha  *security.CaptchaVerifier
}

// NewBookingRoutes creates a new BookingRoutes.
// captcha controls the optional Captcha check on public booking submissions.
// When captcha.Enabled() is false (empty secret) the check is skipped entirely.
func NewBookingRoutes(registry *ServiceRegistry, captcha *security.CaptchaVerifier) *BookingRoutes {
	return &BookingRoutes{registry: registry, captcha: captcha}
}

// ServiceName satisfies RouteRegistrar (admin routes, reuses "work" gRPC port).
func (br *BookingRoutes) ServiceName() string { return "work" }

func (br *BookingRoutes) getCalendarClient() (calv1.CalendarServiceClient, error) {
	conn, err := br.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return calv1.NewCalendarServiceClient(conn), nil
}

// RegisterRoutes mounts the authenticated admin booking-page CRUD endpoints.
func (br *BookingRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Additive guards: keep the legacy coarse "booking-pages" key AND accept
	// kalender:booking_page:manage. Reads have no catalogue counterpart (the
	// mini-catalogue only curates management), so they stay coarse-only.
	var (
		bookingPageWrite  = middleware.RequirePermissionAny([2]string{"booking-pages", "write"}, [2]string{"kalender:booking_page", "manage"})
		bookingPageDelete = middleware.RequirePermissionAny([2]string{"booking-pages", "delete"}, [2]string{"kalender:booking_page", "manage"})
	)

	r.Route("/api/v1/calendar/booking-pages", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		r.With(middleware.RequirePermission("booking-pages", "read")).
			Get("/", br.HandleListBookingPages)
		r.With(bookingPageWrite).
			Post("/", br.HandleCreateBookingPage)
		r.With(middleware.RequirePermission("booking-pages", "read")).
			Get("/{id}", br.HandleGetBookingPage)
		r.With(bookingPageWrite).
			Put("/{id}", br.HandleUpdateBookingPage)
		r.With(bookingPageDelete).
			Delete("/{id}", br.HandleDeleteBookingPage)
	})
}

// RegisterPublicRoutes mounts the unauthenticated public booking endpoints.
// Called from cmd/gateway/main.go OUTSIDE the registrars loop, directly on r.
// publicRateLimit is applied to all routes in this group — it should be a
// stricter limiter than the global one (e.g. 10 RPS instead of 100).
func (br *BookingRoutes) RegisterPublicRoutes(r chi.Router, publicRateLimit func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(publicRateLimit)

		r.Route("/api/v1/public/booking-pages", func(r chi.Router) {
			r.Get("/{slug}", br.HandleGetPublicBookingPage)
			r.Get("/{slug}/availability", br.HandleGetAvailability)
		})
		r.Post("/api/v1/public/bookings", br.HandleCreatePublicBooking)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createBookingPageRequest struct {
	CalendarID            string                       `json:"calendar_id"           validate:"required,uuid"`
	Slug                  string                       `json:"slug"                  validate:"required"`
	CompanyName           string                       `json:"company_name"          validate:"required"`
	LogoURL               *string                      `json:"logo_url,omitempty"`
	Services              []bookingServiceRequestItem  `json:"services"`
	AvailabilityRulesJSON string                       `json:"availability_rules"    validate:"required"`
}

type bookingServiceRequestItem struct {
	ID          string  `json:"id"           validate:"required"`
	Name        string  `json:"name"         validate:"required"`
	DurationMin int32   `json:"duration_min" validate:"required,gt=0"`
	Price       string  `json:"price"        validate:"required"`
	Description *string `json:"description,omitempty"`
}

type updateBookingPageRequest struct {
	CompanyName           *string                      `json:"company_name,omitempty"`
	LogoURL               *string                      `json:"logo_url,omitempty"`
	Services              []bookingServiceRequestItem  `json:"services,omitempty"`
	AvailabilityRulesJSON *string                      `json:"availability_rules,omitempty"`
	Active                *bool                        `json:"active,omitempty"`
}

type publicBookingCustomerRequest struct {
	Name  string  `json:"name"  validate:"required"`
	Email string  `json:"email" validate:"required,email"`
	Phone *string `json:"phone,omitempty" validate:"omitempty,phone_dach"`
	Notes *string `json:"notes,omitempty"`
}

type createPublicBookingRequest struct {
	Slug      string                       `json:"slug"       validate:"required"`
	ServiceID string                       `json:"service_id" validate:"required"`
	Date      string                       `json:"date"       validate:"required"`
	TimeSlot  string                       `json:"time_slot"  validate:"required"`
	Customer  publicBookingCustomerRequest  `json:"customer"   validate:"required"`
	// Website is a honeypot field. Legitimate clients never populate it.
	// Any submission with a non-empty value is silently rejected as a bot.
	// No validate tag — the field must be accepted by the decoder but never validated.
	Website string `json:"website,omitempty"`
	// CaptchaToken is the client-side challenge token from the captcha widget.
	// Only checked when the CaptchaVerifier is enabled (CAPTCHA_SECRET is set).
	CaptchaToken string `json:"captcha_token,omitempty"`
}

// ============================================================================
// Admin Handlers
// ============================================================================

func (br *BookingRoutes) HandleListBookingPages(w http.ResponseWriter, r *http.Request) {
	client, err := br.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}
	includeInactive := r.URL.Query().Get("include_inactive") == "true"
	resp, err := client.ListBookingPages(r.Context(), &calv1.ListBookingPagesRequest{
		IncludeInactive: includeInactive,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (br *BookingRoutes) HandleCreateBookingPage(w http.ResponseWriter, r *http.Request) {
	client, err := br.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createBookingPageRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &calv1.CreateBookingPageRequest{
		CalendarId:            req.CalendarID,
		Slug:                  req.Slug,
		CompanyName:           req.CompanyName,
		AvailabilityRulesJson: req.AvailabilityRulesJSON,
	}
	if req.LogoURL != nil {
		grpcReq.LogoUrl = req.LogoURL
	}
	for _, svc := range req.Services {
		p := &calv1.BookingServiceProto{
			Id:          svc.ID,
			Name:        svc.Name,
			DurationMin: svc.DurationMin,
			Price:       svc.Price,
		}
		if svc.Description != nil {
			p.Description = svc.Description
		}
		grpcReq.Services = append(grpcReq.Services, p)
	}

	resp, err := client.CreateBookingPage(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (br *BookingRoutes) HandleGetBookingPage(w http.ResponseWriter, r *http.Request) {
	client, err := br.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	resp, err := client.GetBookingPage(r.Context(), &calv1.GetBookingPageRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (br *BookingRoutes) HandleUpdateBookingPage(w http.ResponseWriter, r *http.Request) {
	client, err := br.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[updateBookingPageRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &calv1.UpdateBookingPageRequest{Id: id}
	if req.CompanyName != nil {
		grpcReq.CompanyName = req.CompanyName
	}
	if req.LogoURL != nil {
		grpcReq.LogoUrl = req.LogoURL
	}
	if len(req.Services) > 0 {
		for _, svc := range req.Services {
			p := &calv1.BookingServiceProto{
				Id:          svc.ID,
				Name:        svc.Name,
				DurationMin: svc.DurationMin,
				Price:       svc.Price,
			}
			if svc.Description != nil {
				p.Description = svc.Description
			}
			grpcReq.Services = append(grpcReq.Services, p)
		}
	}
	if req.AvailabilityRulesJSON != nil {
		grpcReq.AvailabilityRulesJson = req.AvailabilityRulesJSON
	}
	if req.Active != nil {
		grpcReq.Active = req.Active
	}

	resp, err := client.UpdateBookingPage(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (br *BookingRoutes) HandleDeleteBookingPage(w http.ResponseWriter, r *http.Request) {
	client, err := br.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	_, err = client.DeleteBookingPage(r.Context(), &calv1.DeleteBookingPageRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "booking page deleted"})
}

// ============================================================================
// Public Handlers
// ============================================================================

func (br *BookingRoutes) HandleGetPublicBookingPage(w http.ResponseWriter, r *http.Request) {
	client, err := br.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}
	slug := chi.URLParam(r, "slug")
	resp, err := client.GetPublicBookingPage(r.Context(), &calv1.GetPublicBookingPageRequest{Slug: slug})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (br *BookingRoutes) HandleGetAvailability(w http.ResponseWriter, r *http.Request) {
	client, err := br.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}
	slug := chi.URLParam(r, "slug")
	q := r.URL.Query()

	dateFrom := q.Get("date_from")
	dateTo := q.Get("date_to")
	if dateFrom == "" || dateTo == "" {
		response.Error(w, http.StatusBadRequest, "date_from and date_to query parameters are required")
		return
	}

	grpcReq := &calv1.GetAvailabilityRequest{
		Slug:     slug,
		DateFrom: dateFrom,
		DateTo:   dateTo,
	}
	if svcID := q.Get("service_id"); svcID != "" {
		grpcReq.ServiceId = &svcID
	}

	resp, err := client.GetAvailability(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (br *BookingRoutes) HandleCreatePublicBooking(w http.ResponseWriter, r *http.Request) {
	client, err := br.getCalendarClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createPublicBookingRequest](w, r)
	if !ok {
		return
	}

	// SECURITY: Honeypot — any submission with the hidden "website" field
	// populated is almost certainly from a bot. Reject with a generic error
	// that gives no hint about the honeypot mechanism.
	if req.Website != "" {
		slog.Warn("public booking honeypot triggered",
			"remote_ip", middleware.ClientIP(r),
			"slug", req.Slug,
		)
		response.Error(w, http.StatusBadRequest, "invalid request")
		return
	}

	// SECURITY: Captcha verification (provider-agnostic siteverify).
	// Skipped entirely when the verifier is disabled (empty CAPTCHA_SECRET).
	if br.captcha.Enabled() {
		captchaOK, captchaErr := br.captcha.Verify(r.Context(), req.CaptchaToken, middleware.ClientIP(r))
		if captchaErr != nil {
			slog.Warn("captcha verification error",
				"error", captchaErr,
				"remote_ip", middleware.ClientIP(r),
			)
			response.Error(w, http.StatusServiceUnavailable, "captcha verification failed")
			return
		}
		if !captchaOK {
			response.Error(w, http.StatusBadRequest, "captcha verification failed")
			return
		}
	}

	grpcReq := &calv1.CreatePublicBookingRequest{
		Slug:      req.Slug,
		ServiceId: req.ServiceID,
		Date:      req.Date,
		TimeSlot:  req.TimeSlot,
		Customer: &calv1.PublicBookingCustomerProto{
			Name:  req.Customer.Name,
			Email: req.Customer.Email,
		},
	}
	if req.Customer.Phone != nil {
		grpcReq.Customer.Phone = req.Customer.Phone
	}
	if req.Customer.Notes != nil {
		grpcReq.Customer.Notes = req.Customer.Notes
	}

	resp, err := client.CreatePublicBooking(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}
