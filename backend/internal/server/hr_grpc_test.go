package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/hr/absence"
	"github.com/kmuhub/kmuhub/internal/biz/hr/changerequest"
	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/models"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// newTestHRServer builds an HRGRPCServer with every service left nil, mirroring
// the newTestFormulareServer trick in formulare_grpc_test.go: it is only safe
// for the request-validation branches that return before touching a service
// field. Any subtest that reaches the nil service would panic - that panic
// itself is the signal the subtest picked the wrong (non-error) input.
func newTestHRServer() *HRGRPCServer {
	return NewHRGRPCServer(nil, nil, nil, nil, nil, nil)
}

// ---------------------------------------------------------------------------
// mapHRError - one gRPC code per sentinel
// ---------------------------------------------------------------------------

func TestMapHRError_AllSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"nil", nil, codes.OK},

		{"LeaveRequestNotFound", leave.ErrLeaveRequestNotFound, codes.NotFound},
		{"InvalidTransition", leave.ErrInvalidTransition, codes.FailedPrecondition},
		{"InsufficientBalance", leave.ErrInsufficientBalance, codes.FailedPrecondition},
		{"CannotApproveOwnRequest", leave.ErrCannotApproveOwnRequest, codes.PermissionDenied},
		{"NotAuthorizedToApprove", leave.ErrNotAuthorizedToApprove, codes.PermissionDenied},
		{"LeaveInvalidDateRange", leave.ErrInvalidDateRange, codes.InvalidArgument},
		{"CannotCancelPastLeave", leave.ErrCannotCancelPastLeave, codes.FailedPrecondition},

		{"AlreadyClockedIn", timetracking.ErrAlreadyClockedIn, codes.FailedPrecondition},
		{"NotClockedIn", timetracking.ErrNotClockedIn, codes.FailedPrecondition},
		{"AlreadyOnBreak", timetracking.ErrAlreadyOnBreak, codes.FailedPrecondition},
		{"NotOnBreak", timetracking.ErrNotOnBreak, codes.FailedPrecondition},
		{"MaxDailyHoursExceeded", timetracking.ErrMaxDailyHoursExceeded, codes.FailedPrecondition},
		{"CorrectionNotFound", timetracking.ErrCorrectionNotFound, codes.NotFound},
		{"WorkTimeEntryNotFound", timetracking.ErrWorkTimeEntryNotFound, codes.NotFound},
		{"TimeCategoryNotFound", timetracking.ErrTimeCategoryNotFound, codes.NotFound},
		{"TimeTemplateNotFound", timetracking.ErrTimeTemplateNotFound, codes.NotFound},
		{"TimeProjectNotFound", timetracking.ErrTimeProjectNotFound, codes.NotFound},
		{"WeekApprovalNotFound", timetracking.ErrWeekApprovalNotFound, codes.NotFound},
		{"WeekAlreadySubmitted", timetracking.ErrWeekAlreadySubmitted, codes.FailedPrecondition},
		{"WeekNotSubmitted", timetracking.ErrWeekNotSubmitted, codes.FailedPrecondition},
		{"WeekLocked", timetracking.ErrWeekLocked, codes.FailedPrecondition},
		{"WeekNotLocked", timetracking.ErrWeekNotLocked, codes.FailedPrecondition},
		{"InvalidManualEntry", timetracking.ErrInvalidManualEntry, codes.InvalidArgument},

		{"EmployeeNotFound", employee.ErrEmployeeNotFound, codes.NotFound},
		{"ProfileAlreadyExists", employee.ErrProfileAlreadyExists, codes.AlreadyExists},
		{"UnauthorizedFieldUpdate", employee.ErrUnauthorizedFieldUpdate, codes.PermissionDenied},
		{"DocumentCategoryNotFound", employee.ErrDocumentCategoryNotFound, codes.NotFound},
		{"DocumentNotFound", employee.ErrDocumentNotFound, codes.NotFound},
		{"EmployeeRequired", employee.ErrEmployeeRequired, codes.InvalidArgument},
		{"SelfOffboard", employee.ErrSelfOffboard, codes.PermissionDenied},
		{"LastRoleAdmin", employee.ErrLastRoleAdmin, codes.FailedPrecondition},
		{"AlreadyOffboarded", employee.ErrAlreadyOffboarded, codes.FailedPrecondition},
		{"SuccessorRequired", employee.ErrSuccessorRequired, codes.OutOfRange},
		{"InvalidSuccessor", employee.ErrInvalidSuccessor, codes.InvalidArgument},
		{"InvalidExitType", employee.ErrInvalidExitType, codes.InvalidArgument},
		{"ExitBeforeLastWorkDay", employee.ErrExitBeforeLastWorkDay, codes.InvalidArgument},

		{"ChangeRequestNotFound", changerequest.ErrNotFound, codes.NotFound},
		{"PendingRequestExists", changerequest.ErrPendingRequestExists, codes.AlreadyExists},
		{"NotPending", changerequest.ErrNotPending, codes.FailedPrecondition},
		{"FieldNotProposable", changerequest.ErrFieldNotProposable, codes.InvalidArgument},
		{"NotProposer", changerequest.ErrNotProposer, codes.PermissionDenied},
		{"OutOfScope", changerequest.ErrOutOfScope, codes.PermissionDenied},
		{"ChangeRequestProfileNotFound", changerequest.ErrProfileNotFound, codes.NotFound},
		{"ReasonRequired", changerequest.ErrReasonRequired, codes.OutOfRange},

		{"AbsenceInvalidDateRange", absence.ErrInvalidDateRange, codes.InvalidArgument},

		{"UnknownError", errUnmappedHR, codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapHRError(tc.err)
			if tc.err == nil {
				require.NoError(t, err)
				return
			}
			requireGRPCCode(t, err, tc.code)
		})
	}
}

// errUnmappedHR stands in for any biz error mapHRError has never seen -
// exercising the switch's default branch (internal server error, not a leak
// of the underlying message).
var errUnmappedHR = &customHRError{"some unmapped error"}

type customHRError struct{ msg string }

func (e *customHRError) Error() string { return e.msg }

// ---------------------------------------------------------------------------
// toProto* converters - nil guard
// ---------------------------------------------------------------------------

func TestHRToProtoConverters_Nil(t *testing.T) {
	require.Nil(t, toProtoLeaveRequest(nil))
	require.Nil(t, toProtoLeaveBalance(nil))
	require.Nil(t, toProtoLeaveType(nil))
	require.Nil(t, toProtoWorkTimeEntry(nil))
	require.Nil(t, toProtoBreakEntry(nil))
	require.Nil(t, toProtoEmployeeProfile(nil))
	require.Nil(t, toProtoEmployeeDocument(nil))
	require.Nil(t, toProtoDocumentCategory(nil))
	require.Nil(t, toProtoHRSettings(nil))
	require.Nil(t, toProtoDailySummary(nil))
	require.Nil(t, toProtoWeeklySummary(nil))
	require.Nil(t, toProtoTimeCategory(nil))
	require.Nil(t, toProtoTimeTemplate(nil))
	require.Nil(t, toProtoTimeProject(nil))
	require.Nil(t, toProtoWeekApproval(nil))
	// toProtoChangeRequest has no nil guard (every caller ranges over a
	// service-returned slice/pointer that is never nil on the success path) -
	// documented in the journal, not exercised here to avoid a spurious panic.
}

// ---------------------------------------------------------------------------
// toProto* converters - fully populated object
// ---------------------------------------------------------------------------

func TestToProtoLeaveRequest_Populated(t *testing.T) {
	approvedBy := uuid.New()
	approvedAt := time.Now()
	auFileID := uuid.New()
	morning := models.HalfDayMorning
	afternoon := models.HalfDayAfternoon

	lr := &models.LeaveRequest{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		EmployeeID:         uuid.New(),
		LeaveTypeID:        uuid.New(),
		StartDate:          time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:            time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		IsHalfDayStart:     true,
		HalfDayPeriodStart: &morning,
		IsHalfDayEnd:       true,
		HalfDayPeriodEnd:   &afternoon,
		TotalDays:          decimal.NewFromFloat(2.5),
		Reason:             "vacation",
		Status:             models.LeaveStatusApproved,
		ApprovedBy:         &approvedBy,
		ApprovalComment:    "ok",
		ApprovedAt:         &approvedAt,
		AUDocumentRequired: true,
		AUDocumentFileID:   &auFileID,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		EmployeeName:       "Jane Doe",
		LeaveTypeName:      "Urlaub",
		LeaveTypeColor:     "#fff",
	}
	p := toProtoLeaveRequest(lr)
	require.Equal(t, lr.ID.String(), p.Id)
	require.Equal(t, "2026-03-01", p.StartDate)
	require.Equal(t, "2026-03-03", p.EndDate)
	require.Equal(t, hrv1.LeaveRequestStatus_LEAVE_APPROVED, p.Status)
	require.Equal(t, hrv1.HalfDayPeriod_HALF_DAY_MORNING, p.HalfDayPeriodStart)
	require.Equal(t, hrv1.HalfDayPeriod_HALF_DAY_AFTERNOON, p.HalfDayPeriodEnd)
	require.Equal(t, approvedBy.String(), p.ApprovedBy)
	require.NotNil(t, p.ApprovedAt)
	require.Equal(t, auFileID.String(), p.AuDocumentFileId)
	require.Equal(t, "Jane Doe", p.EmployeeName)
}

func TestToProtoLeaveBalance_Populated(t *testing.T) {
	expiresAt := time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)
	b := &models.HRLeaveBalance{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		EmployeeID:         uuid.New(),
		Year:               2026,
		Entitlement:        decimal.NewFromInt(30),
		CarriedOver:        decimal.NewFromInt(2),
		Used:                decimal.NewFromInt(5),
		Remaining:          decimal.NewFromInt(27),
		CarryoverExpiresAt: &expiresAt,
		CarryoverNotified:  true,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	p := toProtoLeaveBalance(b)
	require.Equal(t, int32(2026), p.Year)
	require.Equal(t, "30", p.Entitlement)
	require.Equal(t, "27", p.Remaining)
	require.Equal(t, "2027-03-31", p.CarryoverExpiresAt)
	require.True(t, p.CarryoverNotified)
}

func TestToProtoLeaveType_Populated(t *testing.T) {
	lt := &models.LeaveType{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		Name:               "Urlaub",
		Key:                "vacation",
		Color:              "#00ff00",
		DeductsFromBalance: true,
		RequiresApproval:   true,
		RequiresAUDocument: false,
		IsSystem:           true,
		SortOrder:          3,
		CreatedAt:          time.Now(),
	}
	p := toProtoLeaveType(lt)
	require.Equal(t, "Urlaub", p.Name)
	require.Equal(t, "vacation", p.Key)
	require.True(t, p.DeductsFromBalance)
	require.True(t, p.IsSystem)
	require.Equal(t, int32(3), p.SortOrder)
}

func TestToProtoWorkTimeEntry_Populated(t *testing.T) {
	clockOut := time.Now()
	netMinutes := 420
	originalID := uuid.New()
	approvedBy := uuid.New()
	approvedAt := time.Now()
	projectID := uuid.New()

	e := &models.HRWorkTimeEntry{
		ID:                   uuid.New(),
		TenantID:             uuid.New(),
		EmployeeID:           uuid.New(),
		ClockIn:              time.Now().Add(-8 * time.Hour),
		ClockOut:             &clockOut,
		BreakMinutes:         30,
		AutoBreakDeducted:    15,
		NetWorkMinutes:       &netMinutes,
		Status:               models.WorkTimeStatusCompleted,
		IsCorrection:         true,
		OriginalEntryID:      &originalID,
		CorrectionReason:     "forgot to clock out",
		CorrectionApprovedBy: &approvedBy,
		CorrectionApprovedAt: &approvedAt,
		ProjectID:            &projectID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		EmployeeName:         "Jane Doe",
		Breaks: []models.HRBreakEntry{
			{ID: uuid.New(), WorkTimeEntryID: uuid.New(), StartTime: time.Now()},
		},
	}
	p := toProtoWorkTimeEntry(e)
	require.Equal(t, hrv1.WorkTimeEntryStatus_WORK_TIME_COMPLETED, p.Status)
	require.NotNil(t, p.ClockOut)
	require.Equal(t, int32(420), p.NetWorkMinutes)
	require.Equal(t, originalID.String(), p.OriginalEntryId)
	require.Equal(t, approvedBy.String(), p.CorrectionApprovedBy)
	require.Equal(t, projectID.String(), p.ProjectId)
	require.Len(t, p.Breaks, 1)
}

func TestToProtoBreakEntry_Populated(t *testing.T) {
	end := time.Now()
	duration := 15
	b := &models.HRBreakEntry{
		ID:              uuid.New(),
		WorkTimeEntryID: uuid.New(),
		StartTime:       time.Now().Add(-time.Hour),
		EndTime:         &end,
		DurationMinutes: &duration,
	}
	p := toProtoBreakEntry(b)
	require.NotNil(t, p.EndTime)
	require.Equal(t, int32(15), p.DurationMinutes)
}

func TestToProtoEmployeeProfile_Populated(t *testing.T) {
	managerID := uuid.New()
	lastWorkDay := time.Now()
	exitDate := time.Now()
	e := &models.EmployeeProfile{
		ID:                    uuid.New(),
		TenantID:              uuid.New(),
		UserID:                uuid.New(),
		Department:            "Sales",
		PositionTitle:         "AE",
		ContractType:          models.HRContractPartTime,
		WorkDaysPerWeek:       3,
		AnnualLeaveDays:       20,
		ManagerUserID:         &managerID,
		StartDate:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EmergencyContactName:  "John",
		EmergencyContactPhone: "+491234",
		AddressStreet:         "Main St",
		AddressCity:           "Berlin",
		AddressPostalCode:     "10115",
		AddressCountry:        "DE",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
		IsMinor:               true,
		Status:                models.EmployeeStatusInactive,
		LastWorkDay:           &lastWorkDay,
		ExitDate:              &exitDate,
		ExitType:              "resignation",
		ExitReason:            "new job",
		UserName:              "Jane Doe",
		UserEmail:             "jane@example.com",
		ManagerName:           "Boss",
	}
	p := toProtoEmployeeProfile(e)
	require.Equal(t, hrv1.ContractType_CONTRACT_PART_TIME, p.ContractType)
	require.Equal(t, managerID.String(), p.ManagerUserId)
	require.Equal(t, "2025-01-01", p.StartDate)
	require.True(t, p.IsMinor)
	require.Equal(t, "inactive", p.Status)
	require.NotEmpty(t, p.LastWorkDay)
	require.NotEmpty(t, p.ExitDate)
	require.Equal(t, "resignation", p.ExitType)
}

func TestToProtoEmployeeDocument_Populated(t *testing.T) {
	fileID := uuid.New()
	expiresAt := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	d := &models.EmployeeDocument{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		EmployeeID:        uuid.New(),
		CategoryID:        uuid.New(),
		FileID:            &fileID,
		UploadedBy:        uuid.New(),
		Notes:             "n",
		CreatedAt:         time.Now(),
		Title:             "Contract",
		ExpiresAt:         &expiresAt,
		CategoryName:      "Contracts",
		CategoryKey:       "contracts",
		Visibility:        "hr_only",
		FileName:          "contract.pdf",
		FileSize:          "1024",
		UploadedByName:    "Admin",
		EmployeeName:      "Jane Doe",
		EmployeeProfileID: "profile-1",
	}
	p := toProtoEmployeeDocument(d)
	require.Equal(t, fileID.String(), p.FileId)
	require.Equal(t, "2026-12-31", p.ExpiresAt)
	require.Equal(t, "Contract", p.Title)
	require.Equal(t, "contracts", p.CategoryKey)
}

func TestToProtoEmployeeDocument_MissingOptionalFields(t *testing.T) {
	// A document without a linked file or expiry must serialize as an empty
	// string, never a nil-pointer panic through FileID.String()/ExpiresAt.Format().
	d := &models.EmployeeDocument{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		EmployeeID: uuid.New(),
		CategoryID: uuid.New(),
		UploadedBy: uuid.New(),
	}
	p := toProtoEmployeeDocument(d)
	require.Equal(t, "", p.FileId)
	require.Equal(t, "", p.ExpiresAt)
}

func TestToProtoDocumentCategory_Populated(t *testing.T) {
	c := &models.HRDocumentCategory{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		Name:       "Contracts",
		Key:        "contracts",
		Visibility: models.HRDocVisibilityManager,
		IsSystem:   true,
		SortOrder:  1,
		CreatedAt:  time.Now(),
	}
	p := toProtoDocumentCategory(c)
	require.Equal(t, "manager", p.Visibility)
	require.True(t, p.IsSystem)
}

func TestToProtoHRSettings_Populated(t *testing.T) {
	s := &models.HRCompanySettings{
		ID:                     uuid.New(),
		TenantID:               uuid.New(),
		AUThresholdDays:        3,
		ShowAbsenceReason:      true,
		DefaultAnnualLeaveDays: 28,
		Timezone:               "Europe/Berlin",
		WorkHoursPerDay:        8,
		MaxDailyHours:          10,
		BreakAfterHours:        6,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}
	p := toProtoHRSettings(s)
	require.Equal(t, int32(3), p.AuThresholdDays)
	require.Equal(t, "Europe/Berlin", p.Timezone)
	require.Equal(t, int32(8), p.WorkHoursPerDay)
	require.Equal(t, int32(10), p.MaxDailyHours)
	require.Equal(t, int32(6), p.BreakAfterHours)
}

func TestToProtoDailySummary_Populated(t *testing.T) {
	s := &timetracking.DailySummary{
		Date:               time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
		TotalWorkedMinutes: 480,
		TotalBreakMinutes:  30,
		NetWorkMinutes:     450,
		OvertimeMinutes:    30,
	}
	p := toProtoDailySummary(s)
	require.Equal(t, "2026-03-05", p.Date)
	require.Equal(t, int32(450), p.NetWorkMinutes)
	require.Equal(t, int32(30), p.OvertimeMinutes)
}

func TestToProtoWeeklySummary_Populated(t *testing.T) {
	s := &timetracking.WeeklySummary{
		WeekStart:          time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		TotalWorkedMinutes: 2400,
		TotalBreakMinutes:  150,
		NetWorkMinutes:     2250,
		TotalOvertimeMinutes: 60,
		Days: []timetracking.DailySummary{
			{Date: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), TotalWorkedMinutes: 480},
			{Date: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), TotalWorkedMinutes: 0},
		},
	}
	p := toProtoWeeklySummary(s)
	require.Equal(t, "2026-03-02", p.WeekStart)
	require.Equal(t, "2026-03-08", p.WeekEnd)
	require.Equal(t, int32(60), p.OvertimeMinutes)
	require.Len(t, p.DailySummaries, 2)
	// Only the day with worked minutes > 0 counts toward WorkDays.
	require.Equal(t, int32(1), p.WorkDays)
}

func TestToProtoTimeCategory_Populated(t *testing.T) {
	c := &models.HRTimeCategory{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Name:      "Meetings",
		Color:     "#123",
		Icon:      "calendar",
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	p := toProtoTimeCategory(c)
	require.Equal(t, "Meetings", p.Name)
	require.True(t, p.IsDefault)
}

func TestToProtoTimeTemplate_Populated(t *testing.T) {
	catID := uuid.New()
	tpl := &models.HRTimeTemplate{
		ID:               uuid.New(),
		TenantID:         uuid.New(),
		Name:             "Standup",
		CategoryID:       &catID,
		Description:      "daily",
		EstimatedMinutes: 15,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	p := toProtoTimeTemplate(tpl)
	require.Equal(t, catID.String(), p.CategoryId)
	require.Equal(t, int32(15), p.EstimatedMinutes)
}

func TestToProtoTimeTemplate_NoCategory(t *testing.T) {
	tpl := &models.HRTimeTemplate{ID: uuid.New(), TenantID: uuid.New(), Name: "Ad-hoc"}
	p := toProtoTimeTemplate(tpl)
	require.Equal(t, "", p.CategoryId)
}

func TestToProtoTimeProject_Populated(t *testing.T) {
	proj := &models.HRTimeProject{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		Name:            "Project X",
		CustomerName:    "Acme",
		Color:           "#abc",
		BillableDefault: true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	p := toProtoTimeProject(proj)
	require.Equal(t, "Acme", p.CustomerName)
	require.True(t, p.BillableDefault)
}

func TestToProtoWeekApproval_Populated(t *testing.T) {
	submittedAt := time.Now()
	approvedBy := uuid.New()
	approvedAt := time.Now()
	reason := "missing entries"
	w := &models.HRWeekApproval{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		EmployeeID:      uuid.New(),
		WeekStart:       time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		Status:          models.WeekApprovalRejected,
		SubmittedAt:     &submittedAt,
		ApprovedBy:      &approvedBy,
		ApprovedAt:      &approvedAt,
		RejectionReason: &reason,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	p := toProtoWeekApproval(w)
	require.Equal(t, "2026-03-02", p.WeekStart)
	require.Equal(t, "rejected", p.Status)
	require.NotNil(t, p.SubmittedAt)
	require.Equal(t, approvedBy.String(), p.ApprovedBy)
	require.Equal(t, "missing entries", p.RejectionReason)
}

func TestToProtoWeekApproval_NoRejectionReason(t *testing.T) {
	w := &models.HRWeekApproval{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		EmployeeID: uuid.New(),
		WeekStart:  time.Now(),
		Status:     models.WeekApprovalOpen,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	p := toProtoWeekApproval(w)
	require.Equal(t, "", p.RejectionReason)
}

func TestToProtoChangeRequest_Populated(t *testing.T) {
	decidedAt := time.Now()
	r := &models.HRProfileChangeRequest{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		UserName:      "Jane Doe",
		Drawer:        "personal",
		Field:         "phone",
		FieldLabel:    "Phone",
		OldValue:      "123",
		NewValue:      "456",
		Status:        models.HRChangeRequestApproved,
		Reason:        "typo",
		CreatedAt:     time.Now(),
		DecidedAt:     &decidedAt,
		DecidedByName: "Boss",
	}
	p := toProtoChangeRequest(r)
	require.Equal(t, "approved", p.Status)
	require.Equal(t, "456", p.NewValue)
	require.NotNil(t, p.DecidedAt)
	require.Equal(t, "Boss", p.DecidedByName)
}

// ---------------------------------------------------------------------------
// Enum conversions - the contract-type enumeration is the one that has
// diverged between FE and BE before, so it gets full round-trip coverage.
// ---------------------------------------------------------------------------

func TestContractTypeRoundTrip(t *testing.T) {
	cases := []struct {
		domain models.HRContractType
		proto  hrv1.ContractType
	}{
		{models.HRContractFullTime, hrv1.ContractType_CONTRACT_FULL_TIME},
		{models.HRContractPartTime, hrv1.ContractType_CONTRACT_PART_TIME},
		{models.HRContractMiniJob, hrv1.ContractType_CONTRACT_MINI_JOB},
		{models.HRContractIntern, hrv1.ContractType_CONTRACT_INTERN},
		{models.HRContractTemporary, hrv1.ContractType_CONTRACT_TEMPORARY},
	}
	for _, tc := range cases {
		t.Run(string(tc.domain), func(t *testing.T) {
			require.Equal(t, tc.proto, contractTypeToProto(tc.domain))
			require.Equal(t, tc.domain, contractTypeFromProto(tc.proto))
		})
	}
}

func TestContractTypeFromProto_UnspecifiedDefaultsToFullTime(t *testing.T) {
	// contractTypeFromProto has no error return, so an unspecified/unknown
	// wire value must resolve to something - full_time is the documented
	// default rather than silently zero-valuing the field.
	require.Equal(t, models.HRContractFullTime, contractTypeFromProto(hrv1.ContractType_CONTRACT_TYPE_UNSPECIFIED))
}

func TestContractTypeToProto_UnknownDomainValue(t *testing.T) {
	require.Equal(t, hrv1.ContractType_CONTRACT_TYPE_UNSPECIFIED, contractTypeToProto(models.HRContractType("bogus")))
}

func TestHalfDayPeriodRoundTrip(t *testing.T) {
	cases := []struct {
		domain models.HalfDayPeriod
		proto  hrv1.HalfDayPeriod
	}{
		{models.HalfDayMorning, hrv1.HalfDayPeriod_HALF_DAY_MORNING},
		{models.HalfDayAfternoon, hrv1.HalfDayPeriod_HALF_DAY_AFTERNOON},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, halfDayPeriodToProto(tc.domain))
		require.Equal(t, tc.domain, halfDayPeriodFromProto(tc.proto))
	}
}

func TestHalfDayPeriodFromProto_UnspecifiedDefaultsToMorning(t *testing.T) {
	require.Equal(t, models.HalfDayMorning, halfDayPeriodFromProto(hrv1.HalfDayPeriod_HALF_DAY_PERIOD_UNSPECIFIED))
}

func TestLeaveStatusToProto_AllValues(t *testing.T) {
	cases := []struct {
		domain models.LeaveRequestStatus
		proto  hrv1.LeaveRequestStatus
	}{
		{models.LeaveStatusPending, hrv1.LeaveRequestStatus_LEAVE_PENDING},
		{models.LeaveStatusApproved, hrv1.LeaveRequestStatus_LEAVE_APPROVED},
		{models.LeaveStatusRejected, hrv1.LeaveRequestStatus_LEAVE_REJECTED},
		{models.LeaveStatusCancelled, hrv1.LeaveRequestStatus_LEAVE_CANCELLED},
		{models.LeaveRequestStatus("bogus"), hrv1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_UNSPECIFIED},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, leaveStatusToProto(tc.domain))
	}
}

func TestWorkTimeStatusToProto_AllValues(t *testing.T) {
	cases := []struct {
		domain models.WorkTimeEntryStatus
		proto  hrv1.WorkTimeEntryStatus
	}{
		{models.WorkTimeStatusActive, hrv1.WorkTimeEntryStatus_WORK_TIME_ACTIVE},
		{models.WorkTimeStatusCompleted, hrv1.WorkTimeEntryStatus_WORK_TIME_COMPLETED},
		{models.WorkTimeStatusCorrectionPending, hrv1.WorkTimeEntryStatus_WORK_TIME_CORRECTION_PENDING},
		{models.WorkTimeStatusCorrectionApproved, hrv1.WorkTimeEntryStatus_WORK_TIME_CORRECTION_APPROVED},
		{models.WorkTimeStatusSuperseded, hrv1.WorkTimeEntryStatus_WORK_TIME_SUPERSEDED},
		{models.WorkTimeEntryStatus("bogus"), hrv1.WorkTimeEntryStatus_WORK_TIME_ENTRY_STATUS_UNSPECIFIED},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, workTimeStatusToProto(tc.domain))
	}
}

// ---------------------------------------------------------------------------
// Handler validation paths - at least one invalid-input case per RPC in
// hr_grpc.go and hr_grpc_changerequest.go. All of these must return before
// touching a service field (every service on ts is nil, see newTestHRServer).
// ---------------------------------------------------------------------------

func TestHRHandlers_Validation(t *testing.T) {
	someID := uuid.New().String()
	tenant := uuid.New()
	bg := context.Background()
	tenantCtx := ctxWithTenant(tenant)

	t.Run("CreateLeaveRequest_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateLeaveRequest(bg, &hrv1.CreateLeaveRequestReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("CreateLeaveRequest_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateLeaveRequest(tenantCtx, &hrv1.CreateLeaveRequestReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateLeaveRequest_InvalidLeaveTypeID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateLeaveRequest(tenantCtx, &hrv1.CreateLeaveRequestReq{UserId: someID, LeaveTypeId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateLeaveRequest_InvalidStartDate", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateLeaveRequest(tenantCtx, &hrv1.CreateLeaveRequestReq{UserId: someID, LeaveTypeId: someID, StartDate: "not-a-date"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateLeaveRequest_InvalidEndDate", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateLeaveRequest(tenantCtx, &hrv1.CreateLeaveRequestReq{UserId: someID, LeaveTypeId: someID, StartDate: "2026-03-01", EndDate: "not-a-date"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetLeaveRequest_InvalidID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetLeaveRequest(bg, &hrv1.GetLeaveRequestReq{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListLeaveRequests_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListLeaveRequests(bg, &hrv1.ListLeaveRequestsReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("ListLeaveRequests_InvalidEmployeeID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListLeaveRequests(tenantCtx, &hrv1.ListLeaveRequestsReq{EmployeeId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ApproveLeaveRequest_InvalidID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ApproveLeaveRequest(bg, &hrv1.ApproveLeaveRequestReq{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ApproveLeaveRequest_InvalidApproverID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ApproveLeaveRequest(bg, &hrv1.ApproveLeaveRequestReq{Id: someID, ApproverId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RejectLeaveRequest_InvalidID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RejectLeaveRequest(bg, &hrv1.RejectLeaveRequestReq{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RejectLeaveRequest_InvalidApproverID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RejectLeaveRequest(bg, &hrv1.RejectLeaveRequestReq{Id: someID, ApproverId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CancelLeaveRequest_InvalidID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CancelLeaveRequest(bg, &hrv1.CancelLeaveRequestReq{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CancelLeaveRequest_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CancelLeaveRequest(bg, &hrv1.CancelLeaveRequestReq{Id: someID, UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetLeaveBalance_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetLeaveBalance(bg, &hrv1.GetLeaveBalanceReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("GetLeaveBalance_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetLeaveBalance(tenantCtx, &hrv1.GetLeaveBalanceReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetEmployeeLeaveBalance_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetEmployeeLeaveBalance(bg, &hrv1.GetEmployeeLeaveBalanceReq{EmployeeId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("GetEmployeeLeaveBalance_InvalidEmployeeID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetEmployeeLeaveBalance(tenantCtx, &hrv1.GetEmployeeLeaveBalanceReq{EmployeeId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListLeaveTypes_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListLeaveTypes(bg, &hrv1.ListLeaveTypesReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("RecordSickLeave_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RecordSickLeave(bg, &hrv1.RecordSickLeaveReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("RecordSickLeave_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RecordSickLeave(tenantCtx, &hrv1.RecordSickLeaveReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RecordSickLeave_InvalidStartDate", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RecordSickLeave(tenantCtx, &hrv1.RecordSickLeaveReq{UserId: someID, StartDate: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RecordSickLeave_InvalidEndDate", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RecordSickLeave(tenantCtx, &hrv1.RecordSickLeaveReq{UserId: someID, StartDate: "2026-03-01", EndDate: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ClockIn_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ClockIn(bg, &hrv1.ClockInReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("ClockIn_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ClockIn(tenantCtx, &hrv1.ClockInReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ClockOut_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ClockOut(bg, &hrv1.ClockOutReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("ClockOut_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ClockOut(tenantCtx, &hrv1.ClockOutReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("StartBreak_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.StartBreak(bg, &hrv1.StartBreakReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("StartBreak_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.StartBreak(tenantCtx, &hrv1.StartBreakReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("EndBreak_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.EndBreak(bg, &hrv1.EndBreakReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("EndBreak_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.EndBreak(tenantCtx, &hrv1.EndBreakReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetActiveShift_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetActiveShift(bg, &hrv1.GetActiveShiftReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("GetActiveShift_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetActiveShift(tenantCtx, &hrv1.GetActiveShiftReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListWorkTimeEntries_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListWorkTimeEntries(bg, &hrv1.ListWorkTimeEntriesReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("ListWorkTimeEntries_InvalidEmployeeID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListWorkTimeEntries(tenantCtx, &hrv1.ListWorkTimeEntriesReq{EmployeeId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetDailySummary_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetDailySummary(bg, &hrv1.GetDailySummaryReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("GetDailySummary_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetDailySummary(tenantCtx, &hrv1.GetDailySummaryReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetDailySummary_InvalidDate", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetDailySummary(tenantCtx, &hrv1.GetDailySummaryReq{UserId: someID, Date: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetWeeklySummary_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetWeeklySummary(bg, &hrv1.GetWeeklySummaryReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("GetWeeklySummary_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetWeeklySummary(tenantCtx, &hrv1.GetWeeklySummaryReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetWeeklySummary_InvalidWeekStart", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetWeeklySummary(tenantCtx, &hrv1.GetWeeklySummaryReq{UserId: someID, WeekStart: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("SubmitTimeCorrection_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.SubmitTimeCorrection(bg, &hrv1.SubmitTimeCorrectionReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("SubmitTimeCorrection_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.SubmitTimeCorrection(tenantCtx, &hrv1.SubmitTimeCorrectionReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("SubmitTimeCorrection_InvalidOriginalEntryID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.SubmitTimeCorrection(tenantCtx, &hrv1.SubmitTimeCorrectionReq{UserId: someID, OriginalEntryId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ApproveTimeCorrection_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ApproveTimeCorrection(bg, &hrv1.ApproveTimeCorrectionReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("ApproveTimeCorrection_InvalidID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ApproveTimeCorrection(tenantCtx, &hrv1.ApproveTimeCorrectionReq{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ApproveTimeCorrection_InvalidApproverID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ApproveTimeCorrection(tenantCtx, &hrv1.ApproveTimeCorrectionReq{Id: someID, ApproverId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetAbsenceCalendar_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetAbsenceCalendar(bg, &hrv1.GetAbsenceCalendarReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("GetAbsenceCalendar_InvalidStartDate", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetAbsenceCalendar(tenantCtx, &hrv1.GetAbsenceCalendarReq{StartDate: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetAbsenceCalendar_InvalidEndDate", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetAbsenceCalendar(tenantCtx, &hrv1.GetAbsenceCalendarReq{StartDate: "2026-03-01", EndDate: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListEmployees_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListEmployees(bg, &hrv1.ListEmployeesReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("GetEmployee_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetEmployee(bg, &hrv1.GetEmployeeReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateEmployee_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.UpdateEmployee(bg, &hrv1.UpdateEmployeeReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("OffboardEmployee_InvalidEmployeeID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.OffboardEmployee(bg, &hrv1.OffboardEmployeeReq{EmployeeId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("OffboardEmployee_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.OffboardEmployee(bg, &hrv1.OffboardEmployeeReq{EmployeeId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("OffboardEmployee_MissingActorInContext", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.OffboardEmployee(tenantCtx, &hrv1.OffboardEmployeeReq{EmployeeId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("OffboardEmployee_InvalidExitDate", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.OffboardEmployee(ctxWithActorAndTenant(uuid.New(), tenant), &hrv1.OffboardEmployeeReq{EmployeeId: someID, ExitDate: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("OffboardEmployee_InvalidLastWorkDay", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.OffboardEmployee(ctxWithActorAndTenant(uuid.New(), tenant), &hrv1.OffboardEmployeeReq{EmployeeId: someID, ExitDate: "2026-03-01", LastWorkDay: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("OffboardEmployee_InvalidSuccessorID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.OffboardEmployee(ctxWithActorAndTenant(uuid.New(), tenant), &hrv1.OffboardEmployeeReq{EmployeeId: someID, ExitDate: "2026-03-01", SuccessorUserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateSelfProfile_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.UpdateSelfProfile(bg, &hrv1.UpdateSelfProfileReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListEmployeeDocuments_InvalidEmployeeID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListEmployeeDocuments(bg, &hrv1.ListEmployeeDocumentsReq{EmployeeId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListPersonnelDocuments_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListPersonnelDocuments(bg, &hrv1.ListPersonnelDocumentsReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("UploadEmployeeDocument_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.UploadEmployeeDocument(bg, &hrv1.UploadEmployeeDocumentReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("UploadEmployeeDocument_InvalidEmployeeID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.UploadEmployeeDocument(tenantCtx, &hrv1.UploadEmployeeDocumentReq{EmployeeId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UploadEmployeeDocument_MissingCategory", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.UploadEmployeeDocument(tenantCtx, &hrv1.UploadEmployeeDocumentReq{EmployeeId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UploadEmployeeDocument_InvalidCategoryID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.UploadEmployeeDocument(tenantCtx, &hrv1.UploadEmployeeDocumentReq{EmployeeId: someID, CategoryId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UploadEmployeeDocument_InvalidFileID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.UploadEmployeeDocument(tenantCtx, &hrv1.UploadEmployeeDocumentReq{EmployeeId: someID, CategoryKey: "contracts", FileId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UploadEmployeeDocument_InvalidExpiresAt", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.UploadEmployeeDocument(tenantCtx, &hrv1.UploadEmployeeDocumentReq{EmployeeId: someID, CategoryKey: "contracts", ExpiresAt: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UploadEmployeeDocument_InvalidUploadedBy", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.UploadEmployeeDocument(tenantCtx, &hrv1.UploadEmployeeDocumentReq{EmployeeId: someID, CategoryKey: "contracts", UploadedBy: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListDocumentCategories_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListDocumentCategories(bg, &hrv1.ListDocumentCategoriesReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("CreateEmployee_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateEmployee(bg, &hrv1.CreateEmployeeReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("CreateEmployee_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateEmployee(tenantCtx, &hrv1.CreateEmployeeReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateManualEntry_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateManualEntry(bg, &hrv1.CreateManualEntryReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("CreateManualEntry_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateManualEntry(tenantCtx, &hrv1.CreateManualEntryReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateManualEntry_MissingClockTimes", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateManualEntry(tenantCtx, &hrv1.CreateManualEntryReq{UserId: someID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetTimeBalance_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetTimeBalance(bg, &hrv1.GetTimeBalanceReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("GetTimeBalance_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetTimeBalance(tenantCtx, &hrv1.GetTimeBalanceReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetTimeAnalytics_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetTimeAnalytics(bg, &hrv1.GetTimeAnalyticsReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("GetTimeAnalytics_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetTimeAnalytics(tenantCtx, &hrv1.GetTimeAnalyticsReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetTeamTime_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetTeamTime(bg, &hrv1.GetTeamTimeReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("GetTeamTime_InvalidWeekStart", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetTeamTime(tenantCtx, &hrv1.GetTeamTimeReq{WeekStart: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetMyWeekStatus_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetMyWeekStatus(bg, &hrv1.GetMyWeekStatusReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("GetMyWeekStatus_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetMyWeekStatus(tenantCtx, &hrv1.GetMyWeekStatusReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetMyWeekStatus_InvalidWeekStart", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetMyWeekStatus(tenantCtx, &hrv1.GetMyWeekStatusReq{UserId: someID, WeekStart: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("SubmitWeek_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.SubmitWeek(bg, &hrv1.SubmitWeekReq{UserId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("SubmitWeek_InvalidUserID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.SubmitWeek(tenantCtx, &hrv1.SubmitWeekReq{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("SubmitWeek_InvalidWeekStart", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.SubmitWeek(tenantCtx, &hrv1.SubmitWeekReq{UserId: someID, WeekStart: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ApproveWeek_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ApproveWeek(bg, &hrv1.ApproveWeekReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("ApproveWeek_InvalidApproverID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ApproveWeek(tenantCtx, &hrv1.ApproveWeekReq{ApproverId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ApproveWeek_InvalidEmployeeID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ApproveWeek(tenantCtx, &hrv1.ApproveWeekReq{ApproverId: someID, EmployeeId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ApproveWeek_InvalidWeekStart", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ApproveWeek(tenantCtx, &hrv1.ApproveWeekReq{ApproverId: someID, EmployeeId: someID, WeekStart: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RejectWeek_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RejectWeek(bg, &hrv1.RejectWeekReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("RejectWeek_InvalidApproverID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RejectWeek(tenantCtx, &hrv1.RejectWeekReq{ApproverId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RejectWeek_InvalidEmployeeID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RejectWeek(tenantCtx, &hrv1.RejectWeekReq{ApproverId: someID, EmployeeId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RejectWeek_InvalidWeekStart", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RejectWeek(tenantCtx, &hrv1.RejectWeekReq{ApproverId: someID, EmployeeId: someID, WeekStart: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ReopenWeek_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ReopenWeek(bg, &hrv1.ReopenWeekReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("ReopenWeek_InvalidApproverID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ReopenWeek(tenantCtx, &hrv1.ReopenWeekReq{ApproverId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ReopenWeek_InvalidEmployeeID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ReopenWeek(tenantCtx, &hrv1.ReopenWeekReq{ApproverId: someID, EmployeeId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ReopenWeek_InvalidWeekStart", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ReopenWeek(tenantCtx, &hrv1.ReopenWeekReq{ApproverId: someID, EmployeeId: someID, WeekStart: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListTimeCategories_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListTimeCategories(bg, &hrv1.ListTimeCategoriesReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("CreateTimeCategory_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateTimeCategory(bg, &hrv1.CreateTimeCategoryReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("UpdateTimeCategory_InvalidID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.UpdateTimeCategory(bg, &hrv1.UpdateTimeCategoryReq{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("DeleteTimeCategory_InvalidID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.DeleteTimeCategory(bg, &hrv1.DeleteTimeCategoryReq{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListTimeTemplates_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListTimeTemplates(bg, &hrv1.ListTimeTemplatesReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("CreateTimeTemplate_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateTimeTemplate(bg, &hrv1.CreateTimeTemplateReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("DeleteTimeTemplate_InvalidID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.DeleteTimeTemplate(bg, &hrv1.DeleteTimeTemplateReq{Id: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListTimeProjects_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListTimeProjects(bg, &hrv1.ListTimeProjectsReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("CreateTimeProject_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateTimeProject(bg, &hrv1.CreateTimeProjectReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("GetHRSettings_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.GetHRSettings(bg, &hrv1.GetHRSettingsReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("UpdateHRSettings_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.UpdateHRSettings(bg, &hrv1.UpdateHRSettingsReq{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	// --- Profile change requests (hr_grpc_changerequest.go) ---

	t.Run("ListProfileChangeRequests_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListProfileChangeRequests(bg, &hrv1.ListProfileChangeRequestsReq{ActorId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("ListProfileChangeRequests_InvalidActorID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ListProfileChangeRequests(tenantCtx, &hrv1.ListProfileChangeRequestsReq{ActorId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateProfileChangeRequest_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateProfileChangeRequest(bg, &hrv1.CreateProfileChangeRequestReq{ActorId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("CreateProfileChangeRequest_InvalidActorID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CreateProfileChangeRequest(tenantCtx, &hrv1.CreateProfileChangeRequestReq{ActorId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ApproveProfileChangeRequest_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ApproveProfileChangeRequest(bg, &hrv1.ApproveProfileChangeRequestReq{ActorId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("ApproveProfileChangeRequest_InvalidRequestID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.ApproveProfileChangeRequest(tenantCtx, &hrv1.ApproveProfileChangeRequestReq{ActorId: someID, RequestId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RejectProfileChangeRequest_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RejectProfileChangeRequest(bg, &hrv1.RejectProfileChangeRequestReq{ActorId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("RejectProfileChangeRequest_InvalidRequestID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.RejectProfileChangeRequest(tenantCtx, &hrv1.RejectProfileChangeRequestReq{ActorId: someID, RequestId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CancelProfileChangeRequest_MissingTenant", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CancelProfileChangeRequest(bg, &hrv1.CancelProfileChangeRequestReq{ActorId: someID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("CancelProfileChangeRequest_InvalidActorID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CancelProfileChangeRequest(tenantCtx, &hrv1.CancelProfileChangeRequestReq{ActorId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CancelProfileChangeRequest_InvalidRequestID", func(t *testing.T) {
		ts := newTestHRServer()
		_, err := ts.CancelProfileChangeRequest(tenantCtx, &hrv1.CancelProfileChangeRequestReq{ActorId: someID, RequestId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

// Suppress unused import warnings when a case set above happens to not touch
// every helper type directly (timestamppb is used via toProto* helpers only).
var _ = timestamppb.Now
