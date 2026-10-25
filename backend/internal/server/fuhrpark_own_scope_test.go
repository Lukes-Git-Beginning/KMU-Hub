package server

import (
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	fuhrparkv1 "github.com/kmuhub/kmuhub/proto/fuhrpark/v1"
)

// The gateway sets owner_user_id for a caller whose booking/license scope is "own". The server has
// to turn a record of somebody else into NotFound before it writes anything. (The stub repo hands
// out bookings and licenses owned by a fresh random user, so no caller owns them.)
func TestFuhrparkOwnScope_ForeignRecordIsNotFound(t *testing.T) {
	tenantID := uuid.New()
	id := uuid.NewString()
	caller := uuid.NewString()
	ctx := fuhrparkCtxWithTenant(tenantID)

	t.Run("UpdateVehicleBooking", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		purpose := "x"
		_, err := s.UpdateVehicleBooking(ctx, &fuhrparkv1.UpdateVehicleBookingRequest{Id: id, Purpose: &purpose, OwnerUserId: caller})
		requireGRPCCode(t, err, codes.NotFound)
	})
	t.Run("DeleteVehicleBooking", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteVehicleBooking(ctx, &fuhrparkv1.DeleteVehicleBookingRequest{Id: id, OwnerUserId: caller})
		requireGRPCCode(t, err, codes.NotFound)
	})
	t.Run("UpdateDriverLicense", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.UpdateDriverLicense(ctx, &fuhrparkv1.UpdateDriverLicenseRequest{
			Id: id, NextCheckDueDate: "2027-01-01", OwnerUserId: caller,
		})
		requireGRPCCode(t, err, codes.NotFound)
	})
	t.Run("DeleteDriverLicense", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteDriverLicense(ctx, &fuhrparkv1.DeleteDriverLicenseRequest{Id: id, OwnerUserId: caller})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("an unparsable owner is refused, not ignored", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		_, err := s.DeleteVehicleBooking(ctx, &fuhrparkv1.DeleteVehicleBookingRequest{Id: id, OwnerUserId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
		_, err = s.DeleteDriverLicense(ctx, &fuhrparkv1.DeleteDriverLicenseRequest{Id: id, OwnerUserId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no owner means unrestricted", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		if _, err := s.DeleteVehicleBooking(ctx, &fuhrparkv1.DeleteVehicleBookingRequest{Id: id}); err != nil {
			t.Fatalf("unrestricted booking delete: %v", err)
		}
		if _, err := s.DeleteDriverLicense(ctx, &fuhrparkv1.DeleteDriverLicenseRequest{Id: id}); err != nil {
			t.Fatalf("unrestricted license delete: %v", err)
		}
	})

	t.Run("ListDamages forwards reported_by and rejects a bad one", func(t *testing.T) {
		s := newFuhrparkTestServer(newStubFuhrparkRepo())
		bad := "bad"
		_, err := s.ListDamages(ctx, &fuhrparkv1.ListDamagesRequest{TenantId: tenantID.String(), ReportedBy: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}
