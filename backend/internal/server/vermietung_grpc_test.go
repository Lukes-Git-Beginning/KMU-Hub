package server

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/vermietung"
	vermietungv1 "github.com/kmuhub/kmuhub/proto/vermietung/v1"
)

// ---------------------------------------------------------------------------
// stub vermietung.Repository
// ---------------------------------------------------------------------------

var errStubVermietungFailure = errors.New("stub vermietung repository failure")

type stubVermietungRepo struct {
	forceErr error

	objects     map[uuid.UUID]*vermietung.RentalObject
	rentals     map[uuid.UUID]*vermietung.Rental
	inspections map[uuid.UUID]*vermietung.RentalInspection
}

func newStubVermietungRepo() *stubVermietungRepo {
	return &stubVermietungRepo{
		objects:     make(map[uuid.UUID]*vermietung.RentalObject),
		rentals:     make(map[uuid.UUID]*vermietung.Rental),
		inspections: make(map[uuid.UUID]*vermietung.RentalInspection),
	}
}

func (r *stubVermietungRepo) CreateObject(_ context.Context, obj *vermietung.RentalObject) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.objects[obj.ID] = obj
	return nil
}

func (r *stubVermietungRepo) UpdateObject(_ context.Context, obj *vermietung.RentalObject) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.objects[obj.ID] = obj
	return nil
}

func (r *stubVermietungRepo) SoftDeleteObject(_ context.Context, tenantID, objectID uuid.UUID) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	obj, ok := r.objects[objectID]
	if !ok || obj.TenantID != tenantID {
		return vermietung.ErrObjectNotFound
	}
	delete(r.objects, objectID)
	return nil
}

func (r *stubVermietungRepo) GetObject(_ context.Context, tenantID, objectID uuid.UUID) (*vermietung.RentalObject, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	obj, ok := r.objects[objectID]
	if !ok || obj.TenantID != tenantID {
		return nil, vermietung.ErrObjectNotFound
	}
	return obj, nil
}

func (r *stubVermietungRepo) ListObjects(_ context.Context, tenantID uuid.UUID, filter vermietung.ListObjectsFilter, offset, limit int) ([]*vermietung.RentalObject, int, error) {
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var matched []*vermietung.RentalObject
	for _, o := range r.objects {
		if o.TenantID != tenantID {
			continue
		}
		if filter.ActiveOnly && !o.Active {
			continue
		}
		if filter.Category != nil && o.Category != *filter.Category {
			continue
		}
		matched = append(matched, o)
	}
	total := len(matched)
	if offset >= len(matched) {
		return []*vermietung.RentalObject{}, total, nil
	}
	end := min(offset+limit, len(matched))
	return matched[offset:end], total, nil
}

func (r *stubVermietungRepo) CreateRental(_ context.Context, rental *vermietung.Rental) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.rentals[rental.ID] = rental
	return nil
}

func (r *stubVermietungRepo) UpdateRental(_ context.Context, rental *vermietung.Rental) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.rentals[rental.ID] = rental
	return nil
}

func (r *stubVermietungRepo) DeleteRental(_ context.Context, tenantID, rentalID uuid.UUID) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	rental, ok := r.rentals[rentalID]
	if !ok || rental.TenantID != tenantID {
		return vermietung.ErrRentalNotFound
	}
	delete(r.rentals, rentalID)
	return nil
}

func (r *stubVermietungRepo) GetRental(_ context.Context, tenantID, rentalID uuid.UUID) (*vermietung.Rental, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	rental, ok := r.rentals[rentalID]
	if !ok || rental.TenantID != tenantID {
		return nil, vermietung.ErrRentalNotFound
	}
	return rental, nil
}

func (r *stubVermietungRepo) ListRentals(_ context.Context, tenantID uuid.UUID, filter vermietung.ListRentalsFilter, offset, limit int) ([]*vermietung.Rental, int, error) {
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var matched []*vermietung.Rental
	for _, rental := range r.rentals {
		if rental.TenantID != tenantID {
			continue
		}
		if filter.ObjectID != nil && rental.ObjectID != *filter.ObjectID {
			continue
		}
		if filter.Status != nil && rental.Status != *filter.Status {
			continue
		}
		if filter.From != nil && rental.EndDate.Before(*filter.From) {
			continue
		}
		if filter.To != nil && rental.StartDate.After(*filter.To) {
			continue
		}
		matched = append(matched, rental)
	}
	total := len(matched)
	if offset >= len(matched) {
		return []*vermietung.Rental{}, total, nil
	}
	end := min(offset+limit, len(matched))
	return matched[offset:end], total, nil
}

func (r *stubVermietungRepo) SaveSignature(_ context.Context, tenantID, rentalID, signatureData, signedBy string) (*vermietung.Rental, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	rid, err := uuid.Parse(rentalID)
	if err != nil {
		return nil, vermietung.ErrRentalNotFound
	}
	rental, ok := r.rentals[rid]
	if !ok || rental.TenantID.String() != tenantID {
		return nil, vermietung.ErrRentalNotFound
	}
	sd := signatureData
	sb := signedBy
	now := time.Now()
	rental.SignatureData = &sd
	rental.SignedBy = &sb
	rental.SignedAt = &now
	return rental, nil
}

func (r *stubVermietungRepo) HasOverlap(_ context.Context, tenantID, objectID uuid.UUID, start, end time.Time, excludeRentalID *uuid.UUID) (bool, error) {
	if r.forceErr != nil {
		return false, r.forceErr
	}
	for _, rental := range r.rentals {
		if rental.TenantID != tenantID || rental.ObjectID != objectID {
			continue
		}
		if rental.Status == vermietung.RentalStatusCancelled {
			continue
		}
		if excludeRentalID != nil && rental.ID == *excludeRentalID {
			continue
		}
		if rental.StartDate.Before(end) && start.Before(rental.EndDate) {
			return true, nil
		}
	}
	return false, nil
}

func (r *stubVermietungRepo) CreateInspection(_ context.Context, ins *vermietung.RentalInspection) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.inspections[ins.ID] = ins
	return nil
}

func (r *stubVermietungRepo) UpdateInspection(_ context.Context, ins *vermietung.RentalInspection) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.inspections[ins.ID] = ins
	return nil
}

func (r *stubVermietungRepo) GetInspection(_ context.Context, tenantID, inspectionID uuid.UUID) (*vermietung.RentalInspection, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	ins, ok := r.inspections[inspectionID]
	if !ok || ins.TenantID != tenantID {
		return nil, vermietung.ErrInspectionNotFound
	}
	return ins, nil
}

func (r *stubVermietungRepo) GetInspectionByKind(_ context.Context, tenantID, rentalID uuid.UUID, kind vermietung.InspectionKind) (*vermietung.RentalInspection, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	for _, ins := range r.inspections {
		if ins.TenantID == tenantID && ins.RentalID == rentalID && ins.Kind == kind {
			return ins, nil
		}
	}
	return nil, vermietung.ErrInspectionNotFound
}

func (r *stubVermietungRepo) ListInspections(_ context.Context, tenantID, rentalID uuid.UUID, offset, limit int) ([]*vermietung.RentalInspection, int, error) {
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var matched []*vermietung.RentalInspection
	for _, ins := range r.inspections {
		if ins.TenantID != tenantID || ins.RentalID != rentalID {
			continue
		}
		matched = append(matched, ins)
	}
	total := len(matched)
	if offset >= len(matched) {
		return []*vermietung.RentalInspection{}, total, nil
	}
	end := min(offset+limit, len(matched))
	return matched[offset:end], total, nil
}

var _ vermietung.Repository = (*stubVermietungRepo)(nil)

// ---------------------------------------------------------------------------
// server constructors
// ---------------------------------------------------------------------------

func newTestVermietungServer() *VermietungGRPCServer {
	return NewVermietungGRPCServer(nil)
}

func newVermietungServerWithRepo(repo vermietung.Repository) *VermietungGRPCServer {
	return NewVermietungGRPCServer(vermietung.NewService(repo))
}

// ---------------------------------------------------------------------------
// UUID / required-field validation — table test across every RPC
// ---------------------------------------------------------------------------

func TestVermietung_ValidationPaths(t *testing.T) {
	srv := newTestVermietungServer()
	ctx := context.Background()
	badID := "not-a-uuid"
	validTenant := uuid.New().String()

	cases := []struct {
		name string
		call func() error
	}{
		{"CreateObject/tenant_id", func() error {
			_, err := srv.CreateObject(ctx, &vermietungv1.CreateObjectRequest{TenantId: badID})
			return err
		}},
		{"UpdateObject/tenant_id", func() error {
			_, err := srv.UpdateObject(ctx, &vermietungv1.UpdateObjectRequest{TenantId: badID})
			return err
		}},
		{"UpdateObject/object_id", func() error {
			_, err := srv.UpdateObject(ctx, &vermietungv1.UpdateObjectRequest{TenantId: validTenant, ObjectId: badID})
			return err
		}},
		{"DeleteObject/tenant_id", func() error {
			_, err := srv.DeleteObject(ctx, &vermietungv1.DeleteObjectRequest{TenantId: badID})
			return err
		}},
		{"DeleteObject/object_id", func() error {
			_, err := srv.DeleteObject(ctx, &vermietungv1.DeleteObjectRequest{TenantId: validTenant, ObjectId: badID})
			return err
		}},
		{"GetObject/tenant_id", func() error {
			_, err := srv.GetObject(ctx, &vermietungv1.GetObjectRequest{TenantId: badID})
			return err
		}},
		{"GetObject/object_id", func() error {
			_, err := srv.GetObject(ctx, &vermietungv1.GetObjectRequest{TenantId: validTenant, ObjectId: badID})
			return err
		}},
		{"ListObjects/tenant_id", func() error {
			_, err := srv.ListObjects(ctx, &vermietungv1.ListObjectsRequest{TenantId: badID})
			return err
		}},
		{"CheckAvailability/tenant_id", func() error {
			_, err := srv.CheckAvailability(ctx, &vermietungv1.CheckAvailabilityRequest{TenantId: badID})
			return err
		}},
		{"CheckAvailability/object_id", func() error {
			_, err := srv.CheckAvailability(ctx, &vermietungv1.CheckAvailabilityRequest{TenantId: validTenant, ObjectId: badID})
			return err
		}},
		{"CheckAvailability/exclude_rental_id", func() error {
			eid := badID
			_, err := srv.CheckAvailability(ctx, &vermietungv1.CheckAvailabilityRequest{
				TenantId: validTenant, ObjectId: uuid.New().String(),
				StartDate: timestamppb.Now(), EndDate: timestamppb.Now(), ExcludeRentalId: &eid,
			})
			return err
		}},
		{"CreateRental/tenant_id", func() error {
			_, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{TenantId: badID})
			return err
		}},
		{"CreateRental/object_id", func() error {
			_, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{TenantId: validTenant, ObjectId: badID})
			return err
		}},
		{"CreateRental/contact_id", func() error {
			cid := badID
			_, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
				TenantId: validTenant, ObjectId: uuid.New().String(), ContactId: &cid,
			})
			return err
		}},
		{"UpdateRental/tenant_id", func() error {
			_, err := srv.UpdateRental(ctx, &vermietungv1.UpdateRentalRequest{TenantId: badID})
			return err
		}},
		{"UpdateRental/rental_id", func() error {
			_, err := srv.UpdateRental(ctx, &vermietungv1.UpdateRentalRequest{TenantId: validTenant, RentalId: badID})
			return err
		}},
		{"DeleteRental/tenant_id", func() error {
			_, err := srv.DeleteRental(ctx, &vermietungv1.DeleteRentalRequest{TenantId: badID})
			return err
		}},
		{"DeleteRental/rental_id", func() error {
			_, err := srv.DeleteRental(ctx, &vermietungv1.DeleteRentalRequest{TenantId: validTenant, RentalId: badID})
			return err
		}},
		{"GetRental/tenant_id", func() error {
			_, err := srv.GetRental(ctx, &vermietungv1.GetRentalRequest{TenantId: badID})
			return err
		}},
		{"GetRental/rental_id", func() error {
			_, err := srv.GetRental(ctx, &vermietungv1.GetRentalRequest{TenantId: validTenant, RentalId: badID})
			return err
		}},
		{"ListRentals/tenant_id", func() error {
			_, err := srv.ListRentals(ctx, &vermietungv1.ListRentalsRequest{TenantId: badID})
			return err
		}},
		{"ListRentals/object_id", func() error {
			oid := badID
			_, err := srv.ListRentals(ctx, &vermietungv1.ListRentalsRequest{TenantId: validTenant, ObjectId: &oid})
			return err
		}},
		{"StartRental/tenant_id", func() error {
			_, err := srv.StartRental(ctx, &vermietungv1.StartRentalRequest{TenantId: badID})
			return err
		}},
		{"StartRental/rental_id", func() error {
			_, err := srv.StartRental(ctx, &vermietungv1.StartRentalRequest{TenantId: validTenant, RentalId: badID})
			return err
		}},
		{"EndRental/tenant_id", func() error {
			_, err := srv.EndRental(ctx, &vermietungv1.EndRentalRequest{TenantId: badID})
			return err
		}},
		{"EndRental/rental_id", func() error {
			_, err := srv.EndRental(ctx, &vermietungv1.EndRentalRequest{TenantId: validTenant, RentalId: badID})
			return err
		}},
		{"CreateInspection/tenant_id", func() error {
			_, err := srv.CreateInspection(ctx, &vermietungv1.CreateInspectionRequest{TenantId: badID})
			return err
		}},
		{"CreateInspection/rental_id", func() error {
			_, err := srv.CreateInspection(ctx, &vermietungv1.CreateInspectionRequest{TenantId: validTenant, RentalId: badID})
			return err
		}},
		{"CreateInspection/performed_by", func() error {
			pb := badID
			_, err := srv.CreateInspection(ctx, &vermietungv1.CreateInspectionRequest{
				TenantId: validTenant, RentalId: uuid.New().String(), PerformedBy: &pb,
			})
			return err
		}},
		{"UpdateInspection/tenant_id", func() error {
			_, err := srv.UpdateInspection(ctx, &vermietungv1.UpdateInspectionRequest{TenantId: badID})
			return err
		}},
		{"UpdateInspection/inspection_id", func() error {
			_, err := srv.UpdateInspection(ctx, &vermietungv1.UpdateInspectionRequest{TenantId: validTenant, InspectionId: badID})
			return err
		}},
		{"GetInspection/tenant_id", func() error {
			_, err := srv.GetInspection(ctx, &vermietungv1.GetInspectionRequest{TenantId: badID})
			return err
		}},
		{"GetInspection/inspection_id", func() error {
			_, err := srv.GetInspection(ctx, &vermietungv1.GetInspectionRequest{TenantId: validTenant, InspectionId: badID})
			return err
		}},
		{"ListInspections/tenant_id", func() error {
			_, err := srv.ListInspections(ctx, &vermietungv1.ListInspectionsRequest{TenantId: badID})
			return err
		}},
		{"ListInspections/rental_id", func() error {
			_, err := srv.ListInspections(ctx, &vermietungv1.ListInspectionsRequest{TenantId: validTenant, RentalId: badID})
			return err
		}},
		{"SaveSignature/tenant_id", func() error {
			_, err := srv.SaveSignature(ctx, &vermietungv1.SaveRentalSignatureRequest{TenantId: badID})
			return err
		}},
		{"SaveSignature/rental_id", func() error {
			_, err := srv.SaveSignature(ctx, &vermietungv1.SaveRentalSignatureRequest{TenantId: validTenant, RentalId: badID})
			return err
		}},
		{"GetRentalCalendar/tenant_id", func() error {
			_, err := srv.GetRentalCalendar(ctx, &vermietungv1.GetRentalCalendarRequest{TenantId: badID})
			return err
		}},
		{"ExportRentalReport/tenant_id", func() error {
			_, err := srv.ExportRentalReport(ctx, &vermietungv1.ExportRentalReportRequest{TenantId: badID})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, tc.call(), codes.InvalidArgument)
		})
	}
}

// ---------------------------------------------------------------------------
// Object RPCs — happy path, proto mapping, service-level validation
// ---------------------------------------------------------------------------

func TestVermietung_ObjectCRUDAndList(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	loc := "Halle 3"
	createResp, err := srv.CreateObject(ctx, &vermietungv1.CreateObjectRequest{
		TenantId: tenantID, Name: "Betonmischer", Category: "geraete", DailyRate: 45.5, Deposit: 200, Location: &loc,
	})
	require.NoError(t, err)
	obj := createResp.GetObject()
	require.NotEmpty(t, obj.GetId())
	assert.Equal(t, "Betonmischer", obj.GetName())
	assert.True(t, obj.GetActive())
	assert.Equal(t, "Halle 3", obj.GetLocation())

	getResp, err := srv.GetObject(ctx, &vermietungv1.GetObjectRequest{TenantId: tenantID, ObjectId: obj.GetId()})
	require.NoError(t, err)
	assert.Equal(t, obj.GetId(), getResp.GetObject().GetId())

	newName := "Betonmischer XL"
	active := false
	updateResp, err := srv.UpdateObject(ctx, &vermietungv1.UpdateObjectRequest{
		TenantId: tenantID, ObjectId: obj.GetId(), Name: &newName, Active: &active,
	})
	require.NoError(t, err)
	assert.Equal(t, "Betonmischer XL", updateResp.GetObject().GetName())
	assert.False(t, updateResp.GetObject().GetActive())

	listResp, err := srv.ListObjects(ctx, &vermietungv1.ListObjectsRequest{TenantId: tenantID, ActiveOnly: boolPtr(true)})
	require.NoError(t, err)
	assert.Equal(t, int32(0), listResp.GetTotal(), "the object was set inactive, active_only must exclude it")

	_, err = srv.DeleteObject(ctx, &vermietungv1.DeleteObjectRequest{TenantId: tenantID, ObjectId: obj.GetId()})
	require.NoError(t, err)

	_, err = srv.GetObject(ctx, &vermietungv1.GetObjectRequest{TenantId: tenantID, ObjectId: obj.GetId()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestVermietung_CreateObject_EmptyNameAndNegativeRates(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	cases := []struct {
		name string
		req  *vermietungv1.CreateObjectRequest
	}{
		{"blank_name", &vermietungv1.CreateObjectRequest{TenantId: tenantID, Name: "   ", Category: "geraete"}},
		{"negative_daily_rate", &vermietungv1.CreateObjectRequest{TenantId: tenantID, Name: "x", DailyRate: -1}},
		{"negative_deposit", &vermietungv1.CreateObjectRequest{TenantId: tenantID, Name: "x", Deposit: -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := srv.CreateObject(ctx, tc.req)
			requireGRPCCode(t, err, codes.InvalidArgument)
		})
	}
}

// ---------------------------------------------------------------------------
// Rental lifecycle — the core state-machine guard this unit targets:
// return-before-handover (EndRental before StartRental) must fail.
// ---------------------------------------------------------------------------

func createTestObject(t *testing.T, srv *VermietungGRPCServer, tenantID string) string {
	t.Helper()
	resp, err := srv.CreateObject(context.Background(), &vermietungv1.CreateObjectRequest{
		TenantId: tenantID, Name: "obj", Category: "geraete", DailyRate: 10,
	})
	require.NoError(t, err)
	return resp.GetObject().GetId()
}

func TestVermietung_RentalLifecycle_ReturnBeforeHandoverFails(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	end := start.Add(48 * time.Hour)
	createResp, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "Muster GmbH",
		StartDate: timestamppb.New(start), EndDate: timestamppb.New(end), TotalPrice: 300,
	})
	require.NoError(t, err)
	rentalID := createResp.GetRental().GetId()
	assert.Equal(t, "reserved", createResp.GetRental().GetStatus())

	// EndRental (Rücknahme) before StartRental (Übergabe) — the reservation
	// was never handed over, so ending it is not a valid transition.
	_, err = srv.EndRental(ctx, &vermietungv1.EndRentalRequest{TenantId: tenantID, RentalId: rentalID})
	requireGRPCCode(t, err, codes.FailedPrecondition)

	startResp, err := srv.StartRental(ctx, &vermietungv1.StartRentalRequest{TenantId: tenantID, RentalId: rentalID})
	require.NoError(t, err)
	assert.Equal(t, "active", startResp.GetRental().GetStatus())

	// Starting an already-active rental is equally invalid.
	_, err = srv.StartRental(ctx, &vermietungv1.StartRentalRequest{TenantId: tenantID, RentalId: rentalID})
	requireGRPCCode(t, err, codes.FailedPrecondition)

	endResp, err := srv.EndRental(ctx, &vermietungv1.EndRentalRequest{TenantId: tenantID, RentalId: rentalID})
	require.NoError(t, err)
	assert.Equal(t, "completed", endResp.GetRental().GetStatus())

	// Ending an already-completed rental is equally invalid.
	_, err = srv.EndRental(ctx, &vermietungv1.EndRentalRequest{TenantId: tenantID, RentalId: rentalID})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestVermietung_UpdateRental_AllFields(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	end := start.Add(2 * time.Hour)
	createResp, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "A", StartDate: timestamppb.New(start), EndDate: timestamppb.New(end),
	})
	require.NoError(t, err)
	rentalID := createResp.GetRental().GetId()

	newStart := start.Add(time.Hour)
	newEnd := newStart.Add(3 * time.Hour)
	newName := "B GmbH"
	newPrice := 999.0
	newDeposit := true
	newNotes := "Sonderkonditionen"
	updateResp, err := srv.UpdateRental(ctx, &vermietungv1.UpdateRentalRequest{
		TenantId: tenantID, RentalId: rentalID, RenterName: &newName,
		StartDate: timestamppb.New(newStart), EndDate: timestamppb.New(newEnd),
		TotalPrice: &newPrice, DepositPaid: &newDeposit, Notes: &newNotes,
	})
	require.NoError(t, err)
	assert.Equal(t, "B GmbH", updateResp.GetRental().GetRenterName())
	assert.Equal(t, newStart.Unix(), updateResp.GetRental().GetStartDate().AsTime().Unix())
	assert.Equal(t, newEnd.Unix(), updateResp.GetRental().GetEndDate().AsTime().Unix())
	assert.Equal(t, 999.0, updateResp.GetRental().GetTotalPrice())
	assert.True(t, updateResp.GetRental().GetDepositPaid())
	assert.Equal(t, "Sonderkonditionen", updateResp.GetRental().GetNotes())
}

func TestVermietung_UpdateRental_DateChangeConflict(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	_, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "A", StartDate: timestamppb.New(start), EndDate: timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)

	blockerStart := start.Add(48 * time.Hour)
	blockerResp, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "B", StartDate: timestamppb.New(blockerStart), EndDate: timestamppb.New(blockerStart.Add(time.Hour)),
	})
	require.NoError(t, err)

	// Moving "A" into "B"'s slot must be rejected — the overlap re-check on
	// update excludes only the rental being updated, not the whole tenant.
	_, err = srv.UpdateRental(ctx, &vermietungv1.UpdateRentalRequest{
		TenantId: tenantID, RentalId: blockerResp.GetRental().GetId(),
		StartDate: timestamppb.New(start), EndDate: timestamppb.New(start.Add(time.Hour)),
	})
	requireGRPCCode(t, err, codes.AlreadyExists)
}

func TestVermietung_UpdateRental_NotFound(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	_, err := srv.UpdateRental(ctx, &vermietungv1.UpdateRentalRequest{TenantId: tenantID, RentalId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestVermietung_CreateRental_ConflictMapsToAlreadyExists(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	end := start.Add(48 * time.Hour)
	_, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "A", StartDate: timestamppb.New(start), EndDate: timestamppb.New(end),
	})
	require.NoError(t, err)

	overlapStart := start.Add(24 * time.Hour)
	overlapEnd := overlapStart.Add(48 * time.Hour)
	_, err = srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "B", StartDate: timestamppb.New(overlapStart), EndDate: timestamppb.New(overlapEnd),
	})
	requireGRPCCode(t, err, codes.AlreadyExists)
}

func TestVermietung_CreateRental_EmptyRenterNameAndInvertedRange(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)

	_, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "  ", StartDate: timestamppb.New(start), EndDate: timestamppb.New(start.Add(time.Hour)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "x", StartDate: timestamppb.New(start), EndDate: timestamppb.New(start.Add(-time.Hour)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestVermietung_DeleteRental_NotFound(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	_, err := srv.DeleteRental(ctx, &vermietungv1.DeleteRentalRequest{TenantId: tenantID, RentalId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestVermietung_GetRental_NotFound(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	_, err := srv.GetRental(ctx, &vermietungv1.GetRentalRequest{TenantId: tenantID, RentalId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestVermietung_ListRentals_FiltersAndPagination(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectA := createTestObject(t, srv, tenantID)
	objectB := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	_, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectA, RenterName: "A", StartDate: timestamppb.New(start), EndDate: timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)
	_, err = srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectB, RenterName: "B", StartDate: timestamppb.New(start.Add(48 * time.Hour)), EndDate: timestamppb.New(start.Add(49 * time.Hour)),
	})
	require.NoError(t, err)

	byObject, err := srv.ListRentals(ctx, &vermietungv1.ListRentalsRequest{TenantId: tenantID, ObjectId: &objectA})
	require.NoError(t, err)
	assert.Equal(t, int32(1), byObject.GetTotal())

	statusReserved := "reserved"
	byStatus, err := srv.ListRentals(ctx, &vermietungv1.ListRentalsRequest{TenantId: tenantID, Status: &statusReserved})
	require.NoError(t, err)
	assert.Equal(t, int32(2), byStatus.GetTotal())

	paged, err := srv.ListRentals(ctx, &vermietungv1.ListRentalsRequest{TenantId: tenantID, Page: 1, PageSize: 1})
	require.NoError(t, err)
	assert.Equal(t, int32(2), paged.GetTotal())
	assert.Len(t, paged.GetRentals(), 1)
}

func TestVermietung_CheckAvailability(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	end := start.Add(48 * time.Hour)
	_, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "A", StartDate: timestamppb.New(start), EndDate: timestamppb.New(end),
	})
	require.NoError(t, err)

	conflictResp, err := srv.CheckAvailability(ctx, &vermietungv1.CheckAvailabilityRequest{
		TenantId: tenantID, ObjectId: objectID, StartDate: timestamppb.New(start), EndDate: timestamppb.New(end),
	})
	require.NoError(t, err)
	assert.False(t, conflictResp.GetAvailable())
	assert.Len(t, conflictResp.GetConflictingRentals(), 1)

	freeStart := end.Add(24 * time.Hour)
	freeResp, err := srv.CheckAvailability(ctx, &vermietungv1.CheckAvailabilityRequest{
		TenantId: tenantID, ObjectId: objectID, StartDate: timestamppb.New(freeStart), EndDate: timestamppb.New(freeStart.Add(time.Hour)),
	})
	require.NoError(t, err)
	assert.True(t, freeResp.GetAvailable())
	assert.Empty(t, freeResp.GetConflictingRentals())
}

// ---------------------------------------------------------------------------
// Inspections — duplicate-kind guard and the mapVermietungError gap it hits
// ---------------------------------------------------------------------------

func TestVermietung_InspectionCRUDAndList(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	rentalResp, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "A", StartDate: timestamppb.New(start), EndDate: timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)
	rentalID := rentalResp.GetRental().GetId()

	createResp, err := srv.CreateInspection(ctx, &vermietungv1.CreateInspectionRequest{
		TenantId: tenantID, RentalId: rentalID, Kind: "handover", Notes: "ok",
		Checklist: []*vermietungv1.ChecklistItem{{Label: "Windschutzscheibe", Condition: "intakt"}},
	})
	require.NoError(t, err)
	inspectionID := createResp.GetInspection().GetId()
	assert.Equal(t, "handover", createResp.GetInspection().GetKind())
	assert.Len(t, createResp.GetInspection().GetChecklist(), 1)

	getResp, err := srv.GetInspection(ctx, &vermietungv1.GetInspectionRequest{TenantId: tenantID, InspectionId: inspectionID})
	require.NoError(t, err)
	assert.Equal(t, "ok", getResp.GetInspection().GetNotes())

	newNotes := "updated"
	updateResp, err := srv.UpdateInspection(ctx, &vermietungv1.UpdateInspectionRequest{
		TenantId: tenantID, InspectionId: inspectionID, Notes: &newNotes,
		ReplacePhotos: true, PhotoUrls: []string{"https://example.test/a.jpg"},
	})
	require.NoError(t, err)
	assert.Equal(t, "updated", updateResp.GetInspection().GetNotes())
	assert.Equal(t, []string{"https://example.test/a.jpg"}, updateResp.GetInspection().GetPhotoUrls())

	listResp, err := srv.ListInspections(ctx, &vermietungv1.ListInspectionsRequest{TenantId: tenantID, RentalId: rentalID})
	require.NoError(t, err)
	assert.Equal(t, int32(1), listResp.GetTotal())
}

func TestVermietung_CreateInspection_DuplicateKind_MapsToInternal(t *testing.T) {
	// documents current gap: vermietung.ErrInspectionKindExists (returned by
	// Service.CreateInspection when a second inspection of the same kind is
	// attempted for a rental) has no case in mapVermietungError and falls
	// through to the generic Internal branch — a duplicate-kind attempt
	// surfaces as a 500 instead of a client-actionable 4xx (AlreadyExists
	// would match the sibling ErrRentalConflict pattern). See the JOURNAL
	// entry for this iteration for the resulting Lauf-8 backlog unit.
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	rentalResp, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "A", StartDate: timestamppb.New(start), EndDate: timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)
	rentalID := rentalResp.GetRental().GetId()

	_, err = srv.CreateInspection(ctx, &vermietungv1.CreateInspectionRequest{TenantId: tenantID, RentalId: rentalID, Kind: "handover"})
	require.NoError(t, err)

	_, err = srv.CreateInspection(ctx, &vermietungv1.CreateInspectionRequest{TenantId: tenantID, RentalId: rentalID, Kind: "handover"})
	requireGRPCCode(t, err, codes.Internal)
}

func TestVermietung_CreateInspection_InvalidKind(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	rentalResp, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "A", StartDate: timestamppb.New(start), EndDate: timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)

	_, err = srv.CreateInspection(ctx, &vermietungv1.CreateInspectionRequest{
		TenantId: tenantID, RentalId: rentalResp.GetRental().GetId(), Kind: "not-a-kind",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ---------------------------------------------------------------------------
// Signature
// ---------------------------------------------------------------------------

func TestVermietung_SaveSignature(t *testing.T) {
	// documents current gap: rentalToProto (internal/server/vermietung_grpc.go)
	// never maps Rental.SignatureData/SignedAt/SignedBy onto the wire Rental
	// message, even though the proto carries all three fields and
	// Service.SaveSignature persists them on the domain model. Every RPC that
	// returns a Rental — including SaveSignature's own response — silently
	// drops the signature the caller just saved. Confirmed at the repo level:
	// the stub repository below does set SignatureData/SignedBy/SignedAt on
	// the returned *vermietung.Rental, so this failure is proto-mapping only,
	// not a repository gap. See the JOURNAL entry for this iteration for the
	// resulting Lauf-8 backlog unit.
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	rentalResp, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "A", StartDate: timestamppb.New(start), EndDate: timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)
	rentalID := rentalResp.GetRental().GetId()

	resp, err := srv.SaveSignature(ctx, &vermietungv1.SaveRentalSignatureRequest{
		TenantId: tenantID, RentalId: rentalID,
		SignatureData: "data:image/png;base64,AAAA", SignedBy: "Max Muster",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.GetRental().GetSignatureData(), "documents current gap — rentalToProto drops SignatureData")
	assert.Empty(t, resp.GetRental().GetSignedBy(), "documents current gap — rentalToProto drops SignedBy")
	assert.Nil(t, resp.GetRental().GetSignedAt(), "documents current gap — rentalToProto drops SignedAt")

	// Prove the repository layer actually has the signature — the gap is
	// strictly in rentalToProto, not in Service.SaveSignature or the repo.
	stored := repo.rentals[uuid.MustParse(rentalID)]
	require.NotNil(t, stored.SignatureData)
	assert.Equal(t, "data:image/png;base64,AAAA", *stored.SignatureData)
	require.NotNil(t, stored.SignedBy)
	assert.Equal(t, "Max Muster", *stored.SignedBy)
}

func TestVermietung_SaveSignature_InvalidPrefix(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	rentalResp, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "A", StartDate: timestamppb.New(start), EndDate: timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)

	_, err = srv.SaveSignature(ctx, &vermietungv1.SaveRentalSignatureRequest{
		TenantId: tenantID, RentalId: rentalResp.GetRental().GetId(),
		SignatureData: "not-a-data-uri", SignedBy: "Max Muster",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ---------------------------------------------------------------------------
// Calendar & report
// ---------------------------------------------------------------------------

func TestVermietung_GetRentalCalendar(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	year, month, _ := time.Now().Date()
	monthStart := time.Date(year, time.Now().Month(), 15, 10, 0, 0, 0, time.UTC)
	_, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "A",
		StartDate: timestamppb.New(monthStart), EndDate: timestamppb.New(monthStart.Add(time.Hour)),
	})
	require.NoError(t, err)

	resp, err := srv.GetRentalCalendar(ctx, &vermietungv1.GetRentalCalendarRequest{
		TenantId: tenantID, Year: int32(year), Month: int32(month),
	})
	require.NoError(t, err)
	require.Len(t, resp.GetEntries(), 1)
	assert.Equal(t, objectID, resp.GetEntries()[0].GetObjectId())
	assert.Len(t, resp.GetEntries()[0].GetRentals(), 1)
}

func TestVermietung_GetRentalCalendar_DefaultsToCurrentMonth(t *testing.T) {
	// year=0 / month out of [1,12] must default to "now", not error or
	// zero-value into a bogus date.
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	resp, err := srv.GetRentalCalendar(ctx, &vermietungv1.GetRentalCalendarRequest{TenantId: tenantID})
	require.NoError(t, err)
	assert.Empty(t, resp.GetEntries())
}

func TestVermietung_ExportRentalReport(t *testing.T) {
	repo := newStubVermietungRepo()
	srv := newVermietungServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	objectID := createTestObject(t, srv, tenantID)

	start := time.Now().Add(24 * time.Hour)
	_, err := srv.CreateRental(ctx, &vermietungv1.CreateRentalRequest{
		TenantId: tenantID, ObjectId: objectID, RenterName: "Muster GmbH",
		StartDate: timestamppb.New(start), EndDate: timestamppb.New(start.Add(time.Hour)), TotalPrice: 123.45,
	})
	require.NoError(t, err)

	resp, err := srv.ExportRentalReport(ctx, &vermietungv1.ExportRentalReportRequest{TenantId: tenantID})
	require.NoError(t, err)
	assert.Equal(t, "text/csv", resp.GetContentType())
	assert.True(t, strings.HasPrefix(string(resp.GetPayload()), "id,object_id,renter_name,start_date,end_date,status,total_price,deposit_paid"))
	assert.Contains(t, string(resp.GetPayload()), "Muster GmbH")
	assert.Contains(t, resp.GetFilename(), tenantID[:8])
}

// ---------------------------------------------------------------------------
// mapVermietungError — table test over every sentinel
// ---------------------------------------------------------------------------

func TestMapVermietungError_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"object_not_found", vermietung.ErrObjectNotFound, codes.NotFound},
		{"rental_not_found", vermietung.ErrRentalNotFound, codes.NotFound},
		{"inspection_not_found", vermietung.ErrInspectionNotFound, codes.NotFound},
		{"rental_conflict", vermietung.ErrRentalConflict, codes.AlreadyExists},
		{"invalid_state_transition", vermietung.ErrInvalidStateTransition, codes.FailedPrecondition},
		{"invalid_input", vermietung.ErrInvalidInput, codes.InvalidArgument},
		{"generic_fallback", errStubVermietungFailure, codes.Internal},
		// documents current gap: see TestVermietung_CreateInspection_DuplicateKind_MapsToInternal.
		{"inspection_kind_exists_documents_current_gap", vermietung.ErrInspectionKindExists, codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapVermietungError(tc.err), tc.code)
		})
	}
}

// ---------------------------------------------------------------------------
// Proto conversion helpers — nil-optional-field cases
// ---------------------------------------------------------------------------

func TestVermietung_ProtoConversion_NilOptionalFields(t *testing.T) {
	assert.Nil(t, objectToProto(nil))
	assert.Nil(t, rentalToProto(nil))
	assert.Nil(t, inspectionToProto(nil))

	now := time.Now()
	obj := &vermietung.RentalObject{ID: uuid.New(), TenantID: uuid.New(), Name: "x", Category: "geraete", CreatedAt: now, UpdatedAt: now}
	pbObj := objectToProto(obj)
	assert.Empty(t, pbObj.GetDescription())
	assert.Empty(t, pbObj.GetLocation())
	assert.Empty(t, pbObj.GetNotes())

	rental := &vermietung.Rental{ID: uuid.New(), TenantID: uuid.New(), ObjectID: uuid.New(), RenterName: "x", StartDate: now, EndDate: now, CreatedAt: now, UpdatedAt: now}
	pbRental := rentalToProto(rental)
	assert.Empty(t, pbRental.GetContactId())

	ins := &vermietung.RentalInspection{ID: uuid.New(), TenantID: uuid.New(), RentalID: uuid.New(), Kind: vermietung.InspectionKindHandover, CreatedAt: now, UpdatedAt: now}
	pbIns := inspectionToProto(ins)
	assert.Empty(t, pbIns.GetPerformedBy())
	assert.NotNil(t, pbIns.GetChecklist())
	assert.Empty(t, pbIns.GetChecklist())

	assert.Empty(t, checklistToProto(nil))
	assert.NotNil(t, checklistToProto(nil), "must be an empty slice, not nil, so JSON serializes [] not null")
	assert.Nil(t, checklistFromProto(nil))
	assert.Nil(t, checklistFromProto([]*vermietungv1.ChecklistItem{}))
}

func boolPtr(b bool) *bool { return &b }
