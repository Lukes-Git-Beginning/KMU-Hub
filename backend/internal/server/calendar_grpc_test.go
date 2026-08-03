package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/resource"
	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// bookingAuthzMockRepo is a resource.Repository backed by an in-memory map. Only
// GetBooking/CancelBooking are exercised by the tests below; everything else
// returns zero values since CancelBooking never reaches them.
type bookingAuthzMockRepo struct {
	bookings map[uuid.UUID]*models.ResourceBooking
}

func newBookingAuthzMockRepo() *bookingAuthzMockRepo {
	return &bookingAuthzMockRepo{bookings: make(map[uuid.UUID]*models.ResourceBooking)}
}

func (r *bookingAuthzMockRepo) Create(_ context.Context, _ *models.Resource) error { return nil }
func (r *bookingAuthzMockRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*models.Resource, error) {
	return nil, resource.ErrResourceNotFound
}
func (r *bookingAuthzMockRepo) List(_ context.Context, _ resource.ResourceFilters) ([]models.Resource, error) {
	return nil, nil
}
func (r *bookingAuthzMockRepo) Update(_ context.Context, _ *models.Resource) error { return nil }
func (r *bookingAuthzMockRepo) Delete(_ context.Context, _, _ uuid.UUID) error     { return nil }
func (r *bookingAuthzMockRepo) SetTags(_ context.Context, _ uuid.UUID, _ []string) error {
	return nil
}
func (r *bookingAuthzMockRepo) CreateBooking(_ context.Context, b *models.ResourceBooking) error {
	r.bookings[b.ID] = b
	return nil
}
func (r *bookingAuthzMockRepo) CancelBooking(_ context.Context, bookingID, _ uuid.UUID) error {
	b, ok := r.bookings[bookingID]
	if !ok {
		return resource.ErrBookingNotFound
	}
	now := time.Now()
	b.CancelledAt = &now
	return nil
}
func (r *bookingAuthzMockRepo) ListBookings(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]models.ResourceBooking, error) {
	return nil, nil
}
func (r *bookingAuthzMockRepo) ListBookingsByEvent(_ context.Context, _ uuid.UUID) ([]models.ResourceBooking, error) {
	return nil, nil
}
func (r *bookingAuthzMockRepo) GetBooking(_ context.Context, bookingID, _ uuid.UUID) (*models.ResourceBooking, error) {
	b, ok := r.bookings[bookingID]
	if !ok {
		return nil, resource.ErrBookingNotFound
	}
	return b, nil
}
func (r *bookingAuthzMockRepo) FindAvailableResources(_ context.Context, _, _ time.Time, _ resource.ResourceFilters) ([]models.Resource, error) {
	return nil, nil
}
func (r *bookingAuthzMockRepo) FindAlternatives(_ context.Context, _ uuid.UUID, _, _ time.Time, _ string, _ uuid.UUID) ([]resource.AlternativeResource, error) {
	return nil, nil
}

// ctxWithActorAndTenant mirrors how middleware.TenantInboundUnaryInterceptor
// populates both keys from the gateway-propagated x-user-id/x-tenant-id
// metadata on the real path.
func ctxWithActorAndTenant(userID, tenantID uuid.UUID) context.Context {
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID.String())
	return context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
}

// Regression coverage for fix-calendar-cancel-booking-actor: CancelBooking used
// to call resource.Service.CancelBooking with a hardcoded uuid.Nil actor, so the
// author check in the service rejected every real caller including the booking's
// own owner. These tests exercise the real gRPC handler, not just the service.

func TestCancelBooking_OwnerCanCancel(t *testing.T) {
	repo := newBookingAuthzMockRepo()
	tenantID := uuid.New()
	ownerID := uuid.New()
	bookingID := uuid.New()
	repo.bookings[bookingID] = &models.ResourceBooking{ID: bookingID, TenantID: tenantID, BookedBy: ownerID}

	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)
	_, err := srv.CancelBooking(ctxWithActorAndTenant(ownerID, tenantID), &calv1.CancelBookingRequest{
		Id: bookingID.String(),
	})
	require.NoError(t, err)
	assert.NotNil(t, repo.bookings[bookingID].CancelledAt)
}

func TestCancelBooking_NonOwnerDenied(t *testing.T) {
	repo := newBookingAuthzMockRepo()
	tenantID := uuid.New()
	ownerID := uuid.New()
	otherID := uuid.New()
	bookingID := uuid.New()
	repo.bookings[bookingID] = &models.ResourceBooking{ID: bookingID, TenantID: tenantID, BookedBy: ownerID}

	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)
	_, err := srv.CancelBooking(ctxWithActorAndTenant(otherID, tenantID), &calv1.CancelBookingRequest{
		Id: bookingID.String(),
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
	assert.Nil(t, repo.bookings[bookingID].CancelledAt, "booking must not be cancelled")
}

func TestCancelBooking_MissingActorContext(t *testing.T) {
	repo := newBookingAuthzMockRepo()
	tenantID := uuid.New()
	bookingID := uuid.New()
	repo.bookings[bookingID] = &models.ResourceBooking{ID: bookingID, TenantID: tenantID, BookedBy: uuid.New()}

	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)
	_, err := srv.CancelBooking(ctxWithTenant(tenantID), &calv1.CancelBookingRequest{
		Id: bookingID.String(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}
