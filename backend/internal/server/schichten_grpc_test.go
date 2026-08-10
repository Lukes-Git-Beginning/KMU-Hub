package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/schichten"
	schichtenv1 "github.com/kmuhub/kmuhub/proto/schichten/v1"
)

// ---------------------------------------------------------------------------
// stub schichten.Repository
// ---------------------------------------------------------------------------

// errStubSchichtenFailure is a generic non-sentinel error used to reach the
// fallback "slog.Error + codes.Internal" branch of mapSchichtenError.
var errStubSchichtenFailure = errors.New("stub schichten repository failure")

type stubSchichtenRepo struct {
	forceErr error

	shifts      map[uuid.UUID]*schichten.Shift
	assignments map[uuid.UUID]*schichten.ShiftAssignment
	templates   map[uuid.UUID]*schichten.ShiftTemplate
	swaps       map[uuid.UUID]*schichten.SwapRequest

	// employeeShiftEnds/employeeShiftStarts are consulted by
	// LatestShiftEndBeforeForEmployee/EarliestShiftStartAfterForEmployee.
	priorShiftEnd    *time.Time
	nextShiftStart   *time.Time
	minorEmployees   map[uuid.UUID]bool
	existingTemplate bool // ShiftExistsForTemplate
}

func newStubSchichtenRepo() *stubSchichtenRepo {
	return &stubSchichtenRepo{
		shifts:         make(map[uuid.UUID]*schichten.Shift),
		assignments:    make(map[uuid.UUID]*schichten.ShiftAssignment),
		templates:      make(map[uuid.UUID]*schichten.ShiftTemplate),
		swaps:          make(map[uuid.UUID]*schichten.SwapRequest),
		minorEmployees: make(map[uuid.UUID]bool),
	}
}

func (r *stubSchichtenRepo) CreateShift(_ context.Context, shift *schichten.Shift) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.shifts[shift.ID] = shift
	return nil
}

func (r *stubSchichtenRepo) UpdateShift(_ context.Context, shift *schichten.Shift) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.shifts[shift.ID] = shift
	return nil
}

func (r *stubSchichtenRepo) DeleteShift(_ context.Context, tenantID, shiftID uuid.UUID) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	s, ok := r.shifts[shiftID]
	if !ok || s.TenantID != tenantID {
		return schichten.ErrShiftNotFound
	}
	delete(r.shifts, shiftID)
	return nil
}

func (r *stubSchichtenRepo) GetShift(_ context.Context, tenantID, shiftID uuid.UUID) (*schichten.Shift, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	s, ok := r.shifts[shiftID]
	if !ok || s.TenantID != tenantID {
		return nil, schichten.ErrShiftNotFound
	}
	return s, nil
}

func (r *stubSchichtenRepo) ListShifts(_ context.Context, tenantID uuid.UUID, filter schichten.ListShiftsFilter, offset, limit int) ([]*schichten.Shift, int, error) {
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var matched []*schichten.Shift
	for _, s := range r.shifts {
		if s.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && s.Status != *filter.Status {
			continue
		}
		if filter.From != nil && s.StartTime.Before(*filter.From) {
			continue
		}
		if filter.To != nil && s.EndTime.After(*filter.To) {
			continue
		}
		matched = append(matched, s)
	}
	total := len(matched)
	if offset >= len(matched) {
		return nil, total, nil
	}
	end := min(offset+limit, len(matched))
	return matched[offset:end], total, nil
}

func (r *stubSchichtenRepo) PublishShifts(_ context.Context, tenantID uuid.UUID, from, to time.Time) (int64, error) {
	if r.forceErr != nil {
		return 0, r.forceErr
	}
	var count int64
	for _, s := range r.shifts {
		if s.TenantID != tenantID {
			continue
		}
		if s.Status != schichten.ShiftStatusDraft {
			continue
		}
		if s.StartTime.Before(from) || s.StartTime.After(to) {
			continue
		}
		s.Status = schichten.ShiftStatusPublished
		count++
	}
	return count, nil
}

func (r *stubSchichtenRepo) CreateAssignment(_ context.Context, a *schichten.ShiftAssignment) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.assignments[a.ID] = a
	return nil
}

func (r *stubSchichtenRepo) DeleteAssignment(_ context.Context, tenantID, shiftID, employeeID uuid.UUID) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	for id, a := range r.assignments {
		if a.TenantID == tenantID && a.ShiftID == shiftID && a.EmployeeID == employeeID {
			delete(r.assignments, id)
			return nil
		}
	}
	return schichten.ErrAssignmentNotFound
}

func (r *stubSchichtenRepo) GetAssignment(_ context.Context, tenantID, shiftID, employeeID uuid.UUID) (*schichten.ShiftAssignment, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	for _, a := range r.assignments {
		if a.TenantID == tenantID && a.ShiftID == shiftID && a.EmployeeID == employeeID {
			return a, nil
		}
	}
	return nil, schichten.ErrAssignmentNotFound
}

func (r *stubSchichtenRepo) ListAssignments(_ context.Context, tenantID, shiftID uuid.UUID) ([]*schichten.ShiftAssignment, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	var out []*schichten.ShiftAssignment
	for _, a := range r.assignments {
		if a.TenantID == tenantID && a.ShiftID == shiftID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *stubSchichtenRepo) CountAssignments(_ context.Context, tenantID, shiftID uuid.UUID) (int, error) {
	if r.forceErr != nil {
		return 0, r.forceErr
	}
	count := 0
	for _, a := range r.assignments {
		if a.TenantID == tenantID && a.ShiftID == shiftID {
			count++
		}
	}
	return count, nil
}

func (r *stubSchichtenRepo) LatestShiftEndBeforeForEmployee(_ context.Context, _, _ uuid.UUID, _ time.Time) (*time.Time, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	return r.priorShiftEnd, nil
}

func (r *stubSchichtenRepo) EarliestShiftStartAfterForEmployee(_ context.Context, _, _ uuid.UUID, _ time.Time) (*time.Time, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	return r.nextShiftStart, nil
}

func (r *stubSchichtenRepo) ShiftExistsForTemplate(_ context.Context, _ uuid.UUID, _, _ time.Time, _ string) (bool, error) {
	if r.forceErr != nil {
		return false, r.forceErr
	}
	return r.existingTemplate, nil
}

func (r *stubSchichtenRepo) IsMinorEmployee(_ context.Context, _, employeeID uuid.UUID) (bool, error) {
	if r.forceErr != nil {
		return false, r.forceErr
	}
	return r.minorEmployees[employeeID], nil
}

func (r *stubSchichtenRepo) CreateTemplate(_ context.Context, t *schichten.ShiftTemplate) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.templates[t.ID] = t
	return nil
}

func (r *stubSchichtenRepo) UpdateTemplate(_ context.Context, t *schichten.ShiftTemplate) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.templates[t.ID] = t
	return nil
}

func (r *stubSchichtenRepo) DeleteTemplate(_ context.Context, tenantID, templateID uuid.UUID) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	t, ok := r.templates[templateID]
	if !ok || t.TenantID != tenantID {
		return schichten.ErrTemplateNotFound
	}
	delete(r.templates, templateID)
	return nil
}

func (r *stubSchichtenRepo) GetTemplate(_ context.Context, tenantID, templateID uuid.UUID) (*schichten.ShiftTemplate, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	t, ok := r.templates[templateID]
	if !ok || t.TenantID != tenantID {
		return nil, schichten.ErrTemplateNotFound
	}
	return t, nil
}

func (r *stubSchichtenRepo) ListTemplates(_ context.Context, tenantID uuid.UUID, offset, limit int) ([]*schichten.ShiftTemplate, int, error) {
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var matched []*schichten.ShiftTemplate
	for _, t := range r.templates {
		if t.TenantID == tenantID {
			matched = append(matched, t)
		}
	}
	total := len(matched)
	if offset >= len(matched) {
		return nil, total, nil
	}
	end := min(offset+limit, len(matched))
	return matched[offset:end], total, nil
}

func (r *stubSchichtenRepo) GetStats(_ context.Context, tenantID uuid.UUID, _, _ *time.Time) (*schichten.ShiftStats, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	stats := &schichten.ShiftStats{}
	uniqueEmployees := make(map[uuid.UUID]bool)
	for _, s := range r.shifts {
		if s.TenantID != tenantID {
			continue
		}
		stats.TotalShifts++
		if s.Status == schichten.ShiftStatusPublished {
			stats.PublishedShifts++
		} else {
			stats.DraftShifts++
		}
	}
	for _, a := range r.assignments {
		if a.TenantID != tenantID {
			continue
		}
		stats.TotalAssignments++
		uniqueEmployees[a.EmployeeID] = true
	}
	stats.UniqueEmployees = len(uniqueEmployees)
	return stats, nil
}

func (r *stubSchichtenRepo) CreateSwapRequest(_ context.Context, req *schichten.SwapRequest) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.swaps[req.ID] = req
	return nil
}

func (r *stubSchichtenRepo) GetSwapRequest(_ context.Context, tenantID, requestID uuid.UUID) (*schichten.SwapRequest, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	req, ok := r.swaps[requestID]
	if !ok || req.TenantID != tenantID {
		return nil, schichten.ErrSwapRequestNotFound
	}
	return req, nil
}

func (r *stubSchichtenRepo) ListSwapRequests(_ context.Context, tenantID uuid.UUID, filter schichten.SwapRequestFilter, offset, limit int) ([]*schichten.SwapRequest, int, error) {
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var matched []*schichten.SwapRequest
	for _, req := range r.swaps {
		if req.TenantID != tenantID {
			continue
		}
		if filter.ShiftID != nil && req.ShiftID != *filter.ShiftID {
			continue
		}
		if filter.Status != nil && req.Status != *filter.Status {
			continue
		}
		matched = append(matched, req)
	}
	total := len(matched)
	if offset >= len(matched) {
		return nil, total, nil
	}
	end := min(offset+limit, len(matched))
	return matched[offset:end], total, nil
}

func (r *stubSchichtenRepo) UpdateSwapRequestStatus(_ context.Context, tenantID, requestID uuid.UUID, status schichten.SwapRequestStatus) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	req, ok := r.swaps[requestID]
	if !ok || req.TenantID != tenantID {
		return schichten.ErrSwapRequestNotFound
	}
	req.Status = status
	return nil
}

func (r *stubSchichtenRepo) SwapAssignmentsForRequest(_ context.Context, _ *schichten.SwapRequest) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	return nil
}

// ---------------------------------------------------------------------------
// test server construction
// ---------------------------------------------------------------------------

// newTestSchichtenServer builds a server with a nil *schichten.Service. Only
// safe for request shapes that fail UUID parsing before the handler ever
// touches s.svc — every RPC in schichten_grpc.go parses tenant_id first.
func newTestSchichtenServer() *SchichtenGRPCServer {
	return NewSchichtenGRPCServer(nil)
}

func newSchichtenServerWithRepo(repo schichten.Repository) *SchichtenGRPCServer {
	return NewSchichtenGRPCServer(schichten.NewService(repo))
}

// ---------------------------------------------------------------------------
// UUID / required-field validation — table test across every RPC
// ---------------------------------------------------------------------------

func TestSchichten_ValidationPaths(t *testing.T) {
	srv := newTestSchichtenServer()
	ctx := context.Background()
	badID := "not-a-uuid"
	validTenant := uuid.New().String()

	cases := []struct {
		name string
		call func() error
	}{
		{"CreateShift/tenant_id", func() error {
			_, err := srv.CreateShift(ctx, &schichtenv1.CreateShiftRequest{TenantId: badID})
			return err
		}},
		{"CreateShift/created_by", func() error {
			cb := badID
			_, err := srv.CreateShift(ctx, &schichtenv1.CreateShiftRequest{TenantId: validTenant, Title: "t", CreatedBy: &cb})
			return err
		}},
		{"GetShift/tenant_id", func() error {
			_, err := srv.GetShift(ctx, &schichtenv1.GetShiftRequest{TenantId: badID})
			return err
		}},
		{"GetShift/shift_id", func() error {
			_, err := srv.GetShift(ctx, &schichtenv1.GetShiftRequest{TenantId: validTenant, ShiftId: badID})
			return err
		}},
		{"UpdateShift/tenant_id", func() error {
			_, err := srv.UpdateShift(ctx, &schichtenv1.UpdateShiftRequest{TenantId: badID})
			return err
		}},
		{"UpdateShift/shift_id", func() error {
			_, err := srv.UpdateShift(ctx, &schichtenv1.UpdateShiftRequest{TenantId: validTenant, ShiftId: badID})
			return err
		}},
		{"DeleteShift/tenant_id", func() error {
			_, err := srv.DeleteShift(ctx, &schichtenv1.DeleteShiftRequest{TenantId: badID})
			return err
		}},
		{"DeleteShift/shift_id", func() error {
			_, err := srv.DeleteShift(ctx, &schichtenv1.DeleteShiftRequest{TenantId: validTenant, ShiftId: badID})
			return err
		}},
		{"ListShifts/tenant_id", func() error {
			_, err := srv.ListShifts(ctx, &schichtenv1.ListShiftsRequest{TenantId: badID})
			return err
		}},
		{"PublishShifts/tenant_id", func() error {
			_, err := srv.PublishShifts(ctx, &schichtenv1.PublishShiftsRequest{TenantId: badID})
			return err
		}},
		{"AssignEmployee/tenant_id", func() error {
			_, err := srv.AssignEmployee(ctx, &schichtenv1.AssignEmployeeRequest{TenantId: badID})
			return err
		}},
		{"AssignEmployee/shift_id", func() error {
			_, err := srv.AssignEmployee(ctx, &schichtenv1.AssignEmployeeRequest{TenantId: validTenant, ShiftId: badID})
			return err
		}},
		{"AssignEmployee/employee_id", func() error {
			_, err := srv.AssignEmployee(ctx, &schichtenv1.AssignEmployeeRequest{TenantId: validTenant, ShiftId: uuid.New().String(), EmployeeId: badID})
			return err
		}},
		{"AssignEmployee/assigned_by", func() error {
			ab := badID
			_, err := srv.AssignEmployee(ctx, &schichtenv1.AssignEmployeeRequest{
				TenantId: validTenant, ShiftId: uuid.New().String(), EmployeeId: uuid.New().String(), AssignedBy: &ab,
			})
			return err
		}},
		{"UnassignEmployee/tenant_id", func() error {
			_, err := srv.UnassignEmployee(ctx, &schichtenv1.UnassignEmployeeRequest{TenantId: badID})
			return err
		}},
		{"UnassignEmployee/shift_id", func() error {
			_, err := srv.UnassignEmployee(ctx, &schichtenv1.UnassignEmployeeRequest{TenantId: validTenant, ShiftId: badID})
			return err
		}},
		{"UnassignEmployee/employee_id", func() error {
			_, err := srv.UnassignEmployee(ctx, &schichtenv1.UnassignEmployeeRequest{TenantId: validTenant, ShiftId: uuid.New().String(), EmployeeId: badID})
			return err
		}},
		{"ListAssignments/tenant_id", func() error {
			_, err := srv.ListAssignments(ctx, &schichtenv1.ListAssignmentsRequest{TenantId: badID})
			return err
		}},
		{"ListAssignments/shift_id", func() error {
			_, err := srv.ListAssignments(ctx, &schichtenv1.ListAssignmentsRequest{TenantId: validTenant, ShiftId: badID})
			return err
		}},
		{"CreateTemplate/tenant_id", func() error {
			_, err := srv.CreateTemplate(ctx, &schichtenv1.CreateTemplateRequest{TenantId: badID})
			return err
		}},
		{"UpdateTemplate/tenant_id", func() error {
			_, err := srv.UpdateTemplate(ctx, &schichtenv1.UpdateTemplateRequest{TenantId: badID})
			return err
		}},
		{"UpdateTemplate/template_id", func() error {
			_, err := srv.UpdateTemplate(ctx, &schichtenv1.UpdateTemplateRequest{TenantId: validTenant, TemplateId: badID})
			return err
		}},
		{"DeleteTemplate/tenant_id", func() error {
			_, err := srv.DeleteTemplate(ctx, &schichtenv1.DeleteTemplateRequest{TenantId: badID})
			return err
		}},
		{"DeleteTemplate/template_id", func() error {
			_, err := srv.DeleteTemplate(ctx, &schichtenv1.DeleteTemplateRequest{TenantId: validTenant, TemplateId: badID})
			return err
		}},
		{"ListTemplates/tenant_id", func() error {
			_, err := srv.ListTemplates(ctx, &schichtenv1.ListTemplatesRequest{TenantId: badID})
			return err
		}},
		{"ApplyTemplate/tenant_id", func() error {
			_, err := srv.ApplyTemplate(ctx, &schichtenv1.ApplyTemplateRequest{TenantId: badID})
			return err
		}},
		{"ApplyTemplate/template_id", func() error {
			_, err := srv.ApplyTemplate(ctx, &schichtenv1.ApplyTemplateRequest{TenantId: validTenant, TemplateId: badID})
			return err
		}},
		{"ApplyTemplate/created_by", func() error {
			cb := badID
			_, err := srv.ApplyTemplate(ctx, &schichtenv1.ApplyTemplateRequest{
				TenantId: validTenant, TemplateId: uuid.New().String(), CreatedBy: &cb,
			})
			return err
		}},
		{"CheckArbzgCompliance/tenant_id", func() error {
			_, err := srv.CheckArbzgCompliance(ctx, &schichtenv1.CheckArbzgComplianceRequest{TenantId: badID})
			return err
		}},
		{"CheckArbzgCompliance/employee_id", func() error {
			_, err := srv.CheckArbzgCompliance(ctx, &schichtenv1.CheckArbzgComplianceRequest{TenantId: validTenant, EmployeeId: badID})
			return err
		}},
		{"GetShiftStats/tenant_id", func() error {
			_, err := srv.GetShiftStats(ctx, &schichtenv1.GetShiftStatsRequest{TenantId: badID})
			return err
		}},
		{"CreateSwapRequest/tenant_id", func() error {
			_, err := srv.CreateSwapRequest(ctx, &schichtenv1.CreateSwapRequestRequest{TenantId: badID})
			return err
		}},
		{"CreateSwapRequest/assignment_id", func() error {
			_, err := srv.CreateSwapRequest(ctx, &schichtenv1.CreateSwapRequestRequest{TenantId: validTenant, AssignmentId: badID})
			return err
		}},
		{"CreateSwapRequest/shift_id", func() error {
			_, err := srv.CreateSwapRequest(ctx, &schichtenv1.CreateSwapRequestRequest{
				TenantId: validTenant, AssignmentId: uuid.New().String(), ShiftId: badID,
			})
			return err
		}},
		{"CreateSwapRequest/requested_by_employee_id", func() error {
			_, err := srv.CreateSwapRequest(ctx, &schichtenv1.CreateSwapRequestRequest{
				TenantId: validTenant, AssignmentId: uuid.New().String(), ShiftId: uuid.New().String(), RequestedByEmployeeId: badID,
			})
			return err
		}},
		{"CreateSwapRequest/swap_with_employee_id", func() error {
			_, err := srv.CreateSwapRequest(ctx, &schichtenv1.CreateSwapRequestRequest{
				TenantId: validTenant, AssignmentId: uuid.New().String(), ShiftId: uuid.New().String(),
				RequestedByEmployeeId: uuid.New().String(), SwapWithEmployeeId: badID,
			})
			return err
		}},
		{"ListSwapRequests/tenant_id", func() error {
			_, err := srv.ListSwapRequests(ctx, &schichtenv1.ListSwapRequestsRequest{TenantId: badID})
			return err
		}},
		{"ListSwapRequests/shift_id", func() error {
			sid := badID
			_, err := srv.ListSwapRequests(ctx, &schichtenv1.ListSwapRequestsRequest{TenantId: validTenant, ShiftId: &sid})
			return err
		}},
		{"ApproveSwapRequest/tenant_id", func() error {
			_, err := srv.ApproveSwapRequest(ctx, &schichtenv1.ApproveSwapRequestRequest{TenantId: badID})
			return err
		}},
		{"ApproveSwapRequest/request_id", func() error {
			_, err := srv.ApproveSwapRequest(ctx, &schichtenv1.ApproveSwapRequestRequest{TenantId: validTenant, RequestId: badID})
			return err
		}},
		{"RejectSwapRequest/tenant_id", func() error {
			_, err := srv.RejectSwapRequest(ctx, &schichtenv1.RejectSwapRequestRequest{TenantId: badID})
			return err
		}},
		{"RejectSwapRequest/request_id", func() error {
			_, err := srv.RejectSwapRequest(ctx, &schichtenv1.RejectSwapRequestRequest{TenantId: validTenant, RequestId: badID})
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
// Shift RPCs — happy path + proto mapping
// ---------------------------------------------------------------------------

func TestSchichten_CreateGetUpdateDeleteShift(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	createdBy := uuid.New().String()
	loc := "Halle 3"
	start := time.Now().Add(24 * time.Hour)
	end := start.Add(4 * time.Hour)

	createResp, err := srv.CreateShift(ctx, &schichtenv1.CreateShiftRequest{
		TenantId:  tenantID,
		Title:     "Fruehschicht",
		StartTime: timestamppb.New(start),
		EndTime:   timestamppb.New(end),
		Location:  &loc,
		CreatedBy: &createdBy,
	})
	require.NoError(t, err)
	require.NotNil(t, createResp.Shift)
	assert.Equal(t, "Fruehschicht", createResp.Shift.Title)
	assert.Equal(t, "draft", createResp.Shift.Status)
	require.NotNil(t, createResp.Shift.Location)
	assert.Equal(t, loc, *createResp.Shift.Location)
	require.NotNil(t, createResp.Shift.CreatedBy)
	assert.Equal(t, createdBy, *createResp.Shift.CreatedBy)

	shiftID := createResp.Shift.Id

	getResp, err := srv.GetShift(ctx, &schichtenv1.GetShiftRequest{TenantId: tenantID, ShiftId: shiftID})
	require.NoError(t, err)
	assert.Equal(t, shiftID, getResp.Shift.Id)

	newTitle := "Spaetschicht"
	updateResp, err := srv.UpdateShift(ctx, &schichtenv1.UpdateShiftRequest{
		TenantId: tenantID, ShiftId: shiftID, Title: &newTitle,
	})
	require.NoError(t, err)
	assert.Equal(t, newTitle, updateResp.Shift.Title)

	_, err = srv.DeleteShift(ctx, &schichtenv1.DeleteShiftRequest{TenantId: tenantID, ShiftId: shiftID})
	require.NoError(t, err)

	_, err = srv.GetShift(ctx, &schichtenv1.GetShiftRequest{TenantId: tenantID, ShiftId: shiftID})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSchichten_CreateShift_EmptyTitleAndInvertedRange(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	start := time.Now().Add(24 * time.Hour)

	_, err := srv.CreateShift(ctx, &schichtenv1.CreateShiftRequest{
		TenantId: tenantID, Title: "  ", StartTime: timestamppb.New(start), EndTime: timestamppb.New(start.Add(time.Hour)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.CreateShift(ctx, &schichtenv1.CreateShiftRequest{
		TenantId: tenantID, Title: "t", StartTime: timestamppb.New(start), EndTime: timestamppb.New(start.Add(-time.Hour)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSchichten_GetShift_NotFound(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)

	_, err := srv.GetShift(context.Background(), &schichtenv1.GetShiftRequest{
		TenantId: uuid.New().String(), ShiftId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSchichten_ListShifts_FiltersAndPagination(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	for i := range 3 {
		start := time.Now().Add(time.Duration(i) * 24 * time.Hour)
		_, err := srv.CreateShift(ctx, &schichtenv1.CreateShiftRequest{
			TenantId: tenantID, Title: "shift", StartTime: timestamppb.New(start), EndTime: timestamppb.New(start.Add(time.Hour)),
		})
		require.NoError(t, err)
	}

	listResp, err := srv.ListShifts(ctx, &schichtenv1.ListShiftsRequest{TenantId: tenantID})
	require.NoError(t, err)
	assert.Equal(t, int32(3), listResp.Total)
	assert.Len(t, listResp.Shifts, 3)

	statusFilter := "published"
	filtered, err := srv.ListShifts(ctx, &schichtenv1.ListShiftsRequest{TenantId: tenantID, Status: &statusFilter})
	require.NoError(t, err)
	assert.Equal(t, int32(0), filtered.Total)
	assert.Empty(t, filtered.Shifts)
}

func TestSchichten_ListShifts_EmptyIsEmptySliceNotNull(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)

	resp, err := srv.ListShifts(context.Background(), &schichtenv1.ListShiftsRequest{TenantId: uuid.New().String()})
	require.NoError(t, err)
	assert.Equal(t, int32(0), resp.Total)
	assert.Empty(t, resp.Shifts)
}

func TestSchichten_PublishShifts(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	start := time.Now().Add(24 * time.Hour)

	_, err := srv.CreateShift(ctx, &schichtenv1.CreateShiftRequest{
		TenantId: tenantID, Title: "t", StartTime: timestamppb.New(start), EndTime: timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)

	resp, err := srv.PublishShifts(ctx, &schichtenv1.PublishShiftsRequest{
		TenantId: tenantID,
		From:     timestamppb.New(start.Add(-time.Hour)),
		To:       timestamppb.New(start.Add(2 * time.Hour)),
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.PublishedCount)
}

func TestSchichten_PublishShifts_InvalidRange(t *testing.T) {
	srv := newTestSchichtenServer()
	tenantID := uuid.New().String()
	now := time.Now()

	_, err := srv.PublishShifts(context.Background(), &schichtenv1.PublishShiftsRequest{
		TenantId: tenantID, From: timestamppb.New(now), To: timestamppb.New(now.Add(-time.Hour)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ---------------------------------------------------------------------------
// Assignment RPCs
// ---------------------------------------------------------------------------

func TestSchichten_AssignUnassignListAssignments(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	employeeID := uuid.New().String()
	assignedBy := uuid.New().String()
	start := time.Now().Add(24 * time.Hour)

	shiftResp, err := srv.CreateShift(ctx, &schichtenv1.CreateShiftRequest{
		TenantId: tenantID, Title: "t", StartTime: timestamppb.New(start), EndTime: timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)
	shiftID := shiftResp.Shift.Id

	assignResp, err := srv.AssignEmployee(ctx, &schichtenv1.AssignEmployeeRequest{
		TenantId: tenantID, ShiftId: shiftID, EmployeeId: employeeID, AssignedBy: &assignedBy,
	})
	require.NoError(t, err)
	assert.Equal(t, employeeID, assignResp.Assignment.EmployeeId)
	require.NotNil(t, assignResp.Assignment.AssignedBy)
	assert.Equal(t, assignedBy, *assignResp.Assignment.AssignedBy)

	listResp, err := srv.ListAssignments(ctx, &schichtenv1.ListAssignmentsRequest{TenantId: tenantID, ShiftId: shiftID})
	require.NoError(t, err)
	assert.Len(t, listResp.Assignments, 1)

	_, err = srv.UnassignEmployee(ctx, &schichtenv1.UnassignEmployeeRequest{TenantId: tenantID, ShiftId: shiftID, EmployeeId: employeeID})
	require.NoError(t, err)

	listResp, err = srv.ListAssignments(ctx, &schichtenv1.ListAssignmentsRequest{TenantId: tenantID, ShiftId: shiftID})
	require.NoError(t, err)
	assert.Empty(t, listResp.Assignments)
}

func TestSchichten_AssignEmployee_ShiftNotFound(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)

	_, err := srv.AssignEmployee(context.Background(), &schichtenv1.AssignEmployeeRequest{
		TenantId: uuid.New().String(), ShiftId: uuid.New().String(), EmployeeId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSchichten_AssignEmployee_ArbzgViolation(t *testing.T) {
	repo := newStubSchichtenRepo()
	tenantID := uuid.New()
	start := time.Now().Add(24 * time.Hour)
	// The employee's previous shift ends 3h before the new one starts — well
	// under the 11h ArbZG rest requirement.
	priorEnd := start.Add(-3 * time.Hour)
	repo.priorShiftEnd = &priorEnd

	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()

	shiftResp, err := srv.CreateShift(ctx, &schichtenv1.CreateShiftRequest{
		TenantId: tenantID.String(), Title: "t", StartTime: timestamppb.New(start), EndTime: timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)

	_, err = srv.AssignEmployee(ctx, &schichtenv1.AssignEmployeeRequest{
		TenantId: tenantID.String(), ShiftId: shiftResp.Shift.Id, EmployeeId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

// ---------------------------------------------------------------------------
// Template RPCs
// ---------------------------------------------------------------------------

func TestSchichten_TemplateCRUDAndList(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	loc := "Lager"

	createResp, err := srv.CreateTemplate(ctx, &schichtenv1.CreateTemplateRequest{
		TenantId: tenantID, Name: "Montags", DayOfWeek: 1, StartHour: 8, StartMinute: 0, DurationMinutes: 480, Location: &loc,
	})
	require.NoError(t, err)
	require.NotNil(t, createResp.Template.Location)
	assert.Equal(t, loc, *createResp.Template.Location)

	templateID := createResp.Template.Id
	newName := "Montags neu"
	updateResp, err := srv.UpdateTemplate(ctx, &schichtenv1.UpdateTemplateRequest{
		TenantId: tenantID, TemplateId: templateID, Name: &newName, Location: &loc,
	})
	require.NoError(t, err)
	assert.Equal(t, newName, updateResp.Template.Name)
	// UpdateTemplate, unlike CreateTemplate, does wire Location through correctly.
	require.NotNil(t, updateResp.Template.Location)
	assert.Equal(t, loc, *updateResp.Template.Location)

	listResp, err := srv.ListTemplates(ctx, &schichtenv1.ListTemplatesRequest{TenantId: tenantID})
	require.NoError(t, err)
	assert.Equal(t, int32(1), listResp.Total)

	_, err = srv.DeleteTemplate(ctx, &schichtenv1.DeleteTemplateRequest{TenantId: tenantID, TemplateId: templateID})
	require.NoError(t, err)

	_, err = srv.DeleteTemplate(ctx, &schichtenv1.DeleteTemplateRequest{TenantId: tenantID, TemplateId: templateID})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSchichten_CreateTemplate_InvalidDayOfWeek(t *testing.T) {
	srv := newSchichtenServerWithRepo(newStubSchichtenRepo())

	_, err := srv.CreateTemplate(context.Background(), &schichtenv1.CreateTemplateRequest{
		TenantId: uuid.New().String(), Name: "t", DayOfWeek: 9, StartHour: 8, DurationMinutes: 60,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSchichten_ApplyTemplate(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	// Find the next Monday so ApplyTemplate has at least one matching day in range.
	rangeStart := time.Now()
	for rangeStart.Weekday() != time.Monday {
		rangeStart = rangeStart.AddDate(0, 0, 1)
	}
	rangeStart = time.Date(rangeStart.Year(), rangeStart.Month(), rangeStart.Day(), 0, 0, 0, 0, time.UTC)
	rangeEnd := rangeStart.AddDate(0, 0, 1)

	createResp, err := srv.CreateTemplate(ctx, &schichtenv1.CreateTemplateRequest{
		TenantId: tenantID, Name: "Montags", DayOfWeek: 1, StartHour: 8, DurationMinutes: 60,
	})
	require.NoError(t, err)

	applyResp, err := srv.ApplyTemplate(ctx, &schichtenv1.ApplyTemplateRequest{
		TenantId: tenantID, TemplateId: createResp.Template.Id,
		RangeStart: timestamppb.New(rangeStart), RangeEnd: timestamppb.New(rangeEnd),
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), applyResp.CreatedCount)
	assert.Len(t, applyResp.Shifts, 1)
}

func TestSchichten_ApplyTemplate_InvalidRange(t *testing.T) {
	srv := newTestSchichtenServer()
	now := time.Now()

	_, err := srv.ApplyTemplate(context.Background(), &schichtenv1.ApplyTemplateRequest{
		TenantId: uuid.New().String(), TemplateId: uuid.New().String(),
		RangeStart: timestamppb.New(now), RangeEnd: timestamppb.New(now.Add(-time.Hour)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ---------------------------------------------------------------------------
// Compliance & Stats
// ---------------------------------------------------------------------------

func TestSchichten_CheckArbzgCompliance_CompliantAndViolation(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	employeeID := uuid.New().String()
	newStart := time.Now().Add(48 * time.Hour)

	okResp, err := srv.CheckArbzgCompliance(ctx, &schichtenv1.CheckArbzgComplianceRequest{
		TenantId: tenantID, EmployeeId: employeeID,
		NewShiftStart: timestamppb.New(newStart), NewShiftEnd: timestamppb.New(newStart.Add(time.Hour)),
	})
	require.NoError(t, err)
	assert.True(t, okResp.Compliant)
	assert.Empty(t, okResp.ViolationReason)

	priorEnd := newStart.Add(-3 * time.Hour)
	repo.priorShiftEnd = &priorEnd

	violationResp, err := srv.CheckArbzgCompliance(ctx, &schichtenv1.CheckArbzgComplianceRequest{
		TenantId: tenantID, EmployeeId: employeeID,
		NewShiftStart: timestamppb.New(newStart), NewShiftEnd: timestamppb.New(newStart.Add(time.Hour)),
	})
	require.NoError(t, err)
	assert.False(t, violationResp.Compliant)
	assert.NotEmpty(t, violationResp.ViolationReason)
}

func TestSchichten_GetShiftStats(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	start := time.Now().Add(24 * time.Hour)

	shiftResp, err := srv.CreateShift(ctx, &schichtenv1.CreateShiftRequest{
		TenantId: tenantID, Title: "t", StartTime: timestamppb.New(start), EndTime: timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)
	_, err = srv.AssignEmployee(ctx, &schichtenv1.AssignEmployeeRequest{
		TenantId: tenantID, ShiftId: shiftResp.Shift.Id, EmployeeId: uuid.New().String(),
	})
	require.NoError(t, err)

	stats, err := srv.GetShiftStats(ctx, &schichtenv1.GetShiftStatsRequest{TenantId: tenantID})
	require.NoError(t, err)
	assert.Equal(t, int32(1), stats.TotalShifts)
	assert.Equal(t, int32(1), stats.DraftShifts)
	assert.Equal(t, int32(1), stats.TotalAssignments)
	assert.Equal(t, int32(1), stats.UniqueEmployees)
}

// ---------------------------------------------------------------------------
// Swap request RPCs — lifecycle transitions and the own-scope premise
// ---------------------------------------------------------------------------

func TestSchichten_SwapRequest_CreateApproveLifecycle(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	requestedBy := uuid.New().String()
	swapWith := uuid.New().String()

	createResp, err := srv.CreateSwapRequest(ctx, &schichtenv1.CreateSwapRequestRequest{
		TenantId: tenantID, AssignmentId: uuid.New().String(), ShiftId: uuid.New().String(),
		RequestedByEmployeeId: requestedBy, SwapWithEmployeeId: swapWith,
		Reason: "Arzttermin", IdempotencyKey: "idem-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "pending", createResp.SwapRequest.Status)
	requestID := createResp.SwapRequest.Id

	approveResp, err := srv.ApproveSwapRequest(ctx, &schichtenv1.ApproveSwapRequestRequest{TenantId: tenantID, RequestId: requestID})
	require.NoError(t, err)
	assert.Equal(t, "approved", approveResp.SwapRequest.Status)

	// Approving (or rejecting) an already-processed request must fail, not
	// silently re-process it.
	_, err = srv.ApproveSwapRequest(ctx, &schichtenv1.ApproveSwapRequestRequest{TenantId: tenantID, RequestId: requestID})
	requireGRPCCode(t, err, codes.FailedPrecondition)

	_, err = srv.RejectSwapRequest(ctx, &schichtenv1.RejectSwapRequestRequest{TenantId: tenantID, RequestId: requestID})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestSchichten_SwapRequest_RejectLifecycle(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	createResp, err := srv.CreateSwapRequest(ctx, &schichtenv1.CreateSwapRequestRequest{
		TenantId: tenantID, AssignmentId: uuid.New().String(), ShiftId: uuid.New().String(),
		RequestedByEmployeeId: uuid.New().String(), SwapWithEmployeeId: uuid.New().String(),
		IdempotencyKey: "idem-2",
	})
	require.NoError(t, err)

	rejectResp, err := srv.RejectSwapRequest(ctx, &schichtenv1.RejectSwapRequestRequest{
		TenantId: tenantID, RequestId: createResp.SwapRequest.Id,
	})
	require.NoError(t, err)
	assert.Equal(t, "rejected", rejectResp.SwapRequest.Status)

	_, err = srv.ApproveSwapRequest(ctx, &schichtenv1.ApproveSwapRequestRequest{TenantId: tenantID, RequestId: createResp.SwapRequest.Id})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestSchichten_SwapRequest_NotFound(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	tenantID := uuid.New().String()

	_, err := srv.ApproveSwapRequest(context.Background(), &schichtenv1.ApproveSwapRequestRequest{TenantId: tenantID, RequestId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)

	_, err = srv.RejectSwapRequest(context.Background(), &schichtenv1.RejectSwapRequestRequest{TenantId: tenantID, RequestId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSchichten_CreateSwapRequest_SameEmployee(t *testing.T) {
	srv := newSchichtenServerWithRepo(newStubSchichtenRepo())
	employeeID := uuid.New().String()

	_, err := srv.CreateSwapRequest(context.Background(), &schichtenv1.CreateSwapRequestRequest{
		TenantId: uuid.New().String(), AssignmentId: uuid.New().String(), ShiftId: uuid.New().String(),
		RequestedByEmployeeId: employeeID, SwapWithEmployeeId: employeeID, IdempotencyKey: "idem-3",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSchichten_ListSwapRequests_FiltersAndPagination(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	shiftID := uuid.New().String()

	_, err := srv.CreateSwapRequest(ctx, &schichtenv1.CreateSwapRequestRequest{
		TenantId: tenantID, AssignmentId: uuid.New().String(), ShiftId: shiftID,
		RequestedByEmployeeId: uuid.New().String(), SwapWithEmployeeId: uuid.New().String(), IdempotencyKey: "idem-4",
	})
	require.NoError(t, err)

	listResp, err := srv.ListSwapRequests(ctx, &schichtenv1.ListSwapRequestsRequest{TenantId: tenantID, ShiftId: &shiftID})
	require.NoError(t, err)
	assert.Equal(t, int32(1), listResp.Total)

	otherShift := uuid.New().String()
	emptyResp, err := srv.ListSwapRequests(ctx, &schichtenv1.ListSwapRequestsRequest{TenantId: tenantID, ShiftId: &otherShift})
	require.NoError(t, err)
	assert.Equal(t, int32(0), emptyResp.Total)
	assert.Empty(t, emptyResp.SwapRequests)
}

// TestSchichten_ListSwapRequests_NoOwnScopeFiltering documents a verified
// premise check for this coverage unit's "own-Scope ueber beide
// Mitarbeiterfelder" requirement (BACKLOG.yml b-cov-server-schichten): no
// such filtering exists anywhere in the stack. schichten.SwapRequestFilter
// (repository.go) carries only ShiftID/Status, ListSwapRequestsInput
// (service.go) does not accept an employee, and the gateway route
// (route_schichten.go) guards /swap-requests with a single plain
// "schichten:swap read" permission — no RequirePermissionAny variant scoped
// to the caller's own employee record, unlike e.g. helpdesk/rapporte's
// documented own-scope pattern. A caller with "schichten:swap read" sees
// every tenant's swap request regardless of whether they are the requester
// or the swap partner. Documented here rather than silently assumed; see the
// JOURNAL entry for this iteration for the resulting Lauf-8 backlog unit.
func TestSchichten_ListSwapRequests_NoOwnScopeFiltering(t *testing.T) {
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	// A request between two employees neither of which is the caller.
	_, err := srv.CreateSwapRequest(ctx, &schichtenv1.CreateSwapRequestRequest{
		TenantId: tenantID, AssignmentId: uuid.New().String(), ShiftId: uuid.New().String(),
		RequestedByEmployeeId: uuid.New().String(), SwapWithEmployeeId: uuid.New().String(), IdempotencyKey: "idem-5",
	})
	require.NoError(t, err)

	// ListSwapRequests has no employee-scoped parameter at all — a
	// tenant-wide read returns it regardless of caller identity.
	listResp, err := srv.ListSwapRequests(ctx, &schichtenv1.ListSwapRequestsRequest{TenantId: tenantID})
	require.NoError(t, err)
	assert.Equal(t, int32(1), listResp.Total)
}

// ---------------------------------------------------------------------------
// mapSchichtenError — table test over every sentinel
// ---------------------------------------------------------------------------

func TestMapSchichtenError_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"shift_not_found", schichten.ErrShiftNotFound, codes.NotFound},
		{"template_not_found", schichten.ErrTemplateNotFound, codes.NotFound},
		{"assignment_not_found", schichten.ErrAssignmentNotFound, codes.NotFound},
		{"swap_request_not_found", schichten.ErrSwapRequestNotFound, codes.NotFound},
		{"invalid_input", schichten.ErrInvalidInput, codes.InvalidArgument},
		{"already_assigned", schichten.ErrAlreadyAssigned, codes.AlreadyExists},
		{"arbzg_violation", schichten.ErrArbzgViolation, codes.FailedPrecondition},
		{"jarbschg_night_work", schichten.ErrJArbSchGNightWork, codes.FailedPrecondition},
		{"jarbschg_daily_hours", schichten.ErrJArbSchGDailyHours, codes.FailedPrecondition},
		{"jarbschg_weekend", schichten.ErrJArbSchGWeekend, codes.FailedPrecondition},
		{"swap_already_processed", schichten.ErrSwapAlreadyProcessed, codes.FailedPrecondition},
		{"generic_fallback", errStubSchichtenFailure, codes.Internal},
		{"shift_full", schichten.ErrShiftFull, codes.FailedPrecondition},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapSchichtenError(tc.err), tc.code)
		})
	}
}

func TestSchichten_AssignEmployee_CapacityExceeded_MapsToFailedPrecondition(t *testing.T) {
	// End-to-end confirmation, exercised through the real handler, not just the table test.
	repo := newStubSchichtenRepo()
	srv := newSchichtenServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	start := time.Now().Add(24 * time.Hour)

	capacity := 1
	shift := &schichten.Shift{
		ID: uuid.New(), TenantID: uuid.MustParse(tenantID), Title: "t",
		StartTime: start, EndTime: start.Add(time.Hour), Status: schichten.ShiftStatusDraft, Capacity: &capacity,
	}
	repo.shifts[shift.ID] = shift
	repo.assignments[uuid.New()] = &schichten.ShiftAssignment{
		ID: uuid.New(), TenantID: shift.TenantID, ShiftID: shift.ID, EmployeeID: uuid.New(), AssignedAt: time.Now(),
	}

	_, err := srv.AssignEmployee(ctx, &schichtenv1.AssignEmployeeRequest{
		TenantId: tenantID, ShiftId: shift.ID.String(), EmployeeId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

// ---------------------------------------------------------------------------
// Proto conversion helpers — nil-optional-field cases
// ---------------------------------------------------------------------------

func TestSchichten_ProtoConversion_NilOptionalFields(t *testing.T) {
	now := time.Now()
	shift := &schichten.Shift{
		ID: uuid.New(), TenantID: uuid.New(), Title: "t", StartTime: now, EndTime: now.Add(time.Hour),
		Status: schichten.ShiftStatusDraft, CreatedAt: now, UpdatedAt: now,
	}
	pbShift := shiftToProto(shift)
	assert.Nil(t, pbShift.Location)
	assert.Nil(t, pbShift.CreatedBy)

	assignment := &schichten.ShiftAssignment{ID: uuid.New(), TenantID: uuid.New(), ShiftID: uuid.New(), EmployeeID: uuid.New(), AssignedAt: now}
	pbAssignment := assignmentToProto(assignment)
	assert.Nil(t, pbAssignment.AssignedBy)

	tmpl := &schichten.ShiftTemplate{ID: uuid.New(), TenantID: uuid.New(), Name: "t", CreatedAt: now, UpdatedAt: now}
	pbTemplate := shiftTemplateToProto(tmpl)
	assert.Nil(t, pbTemplate.Location)
}
