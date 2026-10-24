package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/testutil"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// UpdateEmployee must hand the six contact fields to the service with presence intact
// (absent keeps, "" clears) and answer a malformed country code with InvalidArgument,
// which the gateway renders as 400.
func TestUpdateEmployee_ContactFields_RealSQL(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	f := newTeamTimeFixture(t, pool)
	f.srv = NewHRGRPCServer(nil, nil, employee.NewService(employee.NewPostgresEmployeeRepo(pool), nil, nil), nil, nil, nil)

	str := func(s string) *string { return &s }
	ctx := f.ctxAs(f.manager)

	resp, err := f.srv.UpdateEmployee(ctx, &hrv1.UpdateEmployeeReq{
		UserId:                f.reportA.String(),
		EmergencyContactName:  str("Erika Muster"),
		EmergencyContactPhone: str("+49 30 1234567"),
		AddressStreet:         str("Hauptstr. 1"),
		AddressCity:           str("Berlin"),
		AddressPostalCode:     str("10115"),
		AddressCountry:        str("AT"),
	})
	require.NoError(t, err)
	e := resp.GetEmployee()
	assert.Equal(t, "Erika Muster", e.GetEmergencyContactName())
	assert.Equal(t, "+49 30 1234567", e.GetEmergencyContactPhone())
	assert.Equal(t, "Hauptstr. 1", e.GetAddressStreet())
	assert.Equal(t, "Berlin", e.GetAddressCity())
	assert.Equal(t, "10115", e.GetAddressPostalCode())
	assert.Equal(t, "AT", e.GetAddressCountry())

	// Read back through GetEmployee: an unrelated update keeps them, "" clears only one.
	_, err = f.srv.UpdateEmployee(ctx, &hrv1.UpdateEmployeeReq{UserId: f.reportA.String(), Department: "Sales"})
	require.NoError(t, err)
	_, err = f.srv.UpdateEmployee(ctx, &hrv1.UpdateEmployeeReq{UserId: f.reportA.String(), AddressStreet: str("")})
	require.NoError(t, err)
	got, err := f.srv.GetEmployee(ctx, &hrv1.GetEmployeeReq{UserId: f.reportA.String()})
	require.NoError(t, err)
	assert.Empty(t, got.GetEmployee().GetAddressStreet())
	assert.Equal(t, "Berlin", got.GetEmployee().GetAddressCity())
	assert.Equal(t, "Sales", got.GetEmployee().GetDepartment())

	_, err = f.srv.UpdateEmployee(ctx, &hrv1.UpdateEmployeeReq{UserId: f.reportA.String(), AddressCountry: str("")})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
