package produktion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Work steps, quality checks and machine bookings hang off a production order. The FK has no
// RLS, so without a tenant-scoped lookup an unknown order surfaced as a 500 and an order of
// another tenant went through; a completed or cancelled order kept accepting new rows.
func TestOrderChildren_RequireOpenOrder_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID, otherTenant := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Produktion Order Children Tenant")
	testutil.EnsureTenant(t, pool, otherTenant, "Produktion Order Children Other Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	otherCtx := testutil.WithTenantCtx(context.Background(), otherTenant)

	seedIn := func(t *testing.T, ctx context.Context, tenant uuid.UUID, status OrderStatus) uuid.UUID {
		t.Helper()
		o := seedCoreOrder(t, repo, ctx, pool, tenant, "KID-"+uuid.NewString()[:8])
		o.Status = status
		require.NoError(t, repo.UpdateOrder(ctx, o))
		return o.ID
	}

	now := time.Now().UTC().Truncate(time.Second)
	creates := map[string]func(orderID uuid.UUID) error{
		"work step": func(orderID uuid.UUID) error {
			_, err := svc.CreateWorkStep(ctx, CreateWorkStepInput{TenantID: tenantID, OrderID: orderID, StepNr: 1, Name: "Cut"})
			return err
		},
		"quality check": func(orderID uuid.UUID) error {
			_, err := svc.CreateQualityCheck(ctx, CreateQualityCheckInput{TenantID: tenantID, OrderID: orderID, Inspector: "Jane", CheckedAt: now})
			return err
		},
		"machine booking": func(orderID uuid.UUID) error {
			_, err := svc.CreateMachineBooking(ctx, CreateBookingInput{
				TenantID: tenantID, MachineID: "M-" + orderID.String(), ProductionOrderID: orderID,
				StartsAt: now, EndsAt: now.Add(time.Hour),
			})
			return err
		},
	}
	rows := map[string]func(orderID uuid.UUID) int{
		"work step": func(orderID uuid.UUID) int {
			l, err := repo.ListWorkSteps(ctx, tenantID, orderID)
			require.NoError(t, err)
			return len(l)
		},
		"quality check": func(orderID uuid.UUID) int {
			l, _, err := repo.ListQualityChecks(ctx, tenantID, &orderID, 0, 50)
			require.NoError(t, err)
			return len(l)
		},
		"machine booking": func(orderID uuid.UUID) int {
			l, _, err := repo.ListBookings(ctx, tenantID, ListBookingsFilter{ProductionOrderID: &orderID}, 0, 50)
			require.NoError(t, err)
			return len(l)
		},
	}

	for name, create := range creates {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, create(uuid.New()), ErrOrderNotFound, "unknown order")

			foreign := seedIn(t, otherCtx, otherTenant, OrderStatusPlanned)
			require.True(t, errors.Is(create(foreign), ErrOrderNotFound), "order of another tenant")

			for _, st := range []OrderStatus{OrderStatusCompleted, OrderStatusCancelled} {
				id := seedIn(t, ctx, tenantID, st)
				assert.ErrorIs(t, create(id), ErrOrderInvalidTransition, "order is %s", st)
				assert.Zero(t, rows[name](id), "no row for an order that is %s", st)
			}

			for _, st := range []OrderStatus{OrderStatusPlanned, OrderStatusInProgress} {
				id := seedIn(t, ctx, tenantID, st)
				require.NoError(t, create(id), "order is %s", st)
				assert.Equal(t, 1, rows[name](id), "one row for an order that is %s", st)
			}
		})
	}
}
