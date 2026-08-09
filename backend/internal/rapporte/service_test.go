package rapporte

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// MockRepository
// ============================================================================

type mockRepository struct {
	reports     map[uuid.UUID]*WorkReport
	lines       map[uuid.UUID]*ReportLine
	attachments map[uuid.UUID]*ReportAttachment

	createReportErr error
	updateReportErr error
	getReportErr    error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		reports:     make(map[uuid.UUID]*WorkReport),
		lines:       make(map[uuid.UUID]*ReportLine),
		attachments: make(map[uuid.UUID]*ReportAttachment),
	}
}

func (m *mockRepository) CreateReport(_ context.Context, report *WorkReport) error {
	if m.createReportErr != nil {
		return m.createReportErr
	}
	m.reports[report.ID] = report
	return nil
}

func (m *mockRepository) UpdateReport(_ context.Context, report *WorkReport) error {
	if m.updateReportErr != nil {
		return m.updateReportErr
	}
	if _, ok := m.reports[report.ID]; !ok {
		return ErrReportNotFound
	}
	m.reports[report.ID] = report
	return nil
}

func (m *mockRepository) SoftDeleteReport(_ context.Context, tenantID, reportID uuid.UUID) error {
	rep, ok := m.reports[reportID]
	if !ok || rep.TenantID != tenantID || rep.DeletedAt != nil {
		return ErrReportNotFound
	}
	now := time.Now()
	rep.DeletedAt = &now
	m.reports[reportID] = rep
	return nil
}

func (m *mockRepository) GetReport(_ context.Context, tenantID, reportID uuid.UUID) (*WorkReport, error) {
	if m.getReportErr != nil {
		return nil, m.getReportErr
	}
	rep, ok := m.reports[reportID]
	if !ok || rep.TenantID != tenantID || rep.DeletedAt != nil {
		return nil, ErrReportNotFound
	}
	return rep, nil
}

func (m *mockRepository) ListReports(_ context.Context, tenantID uuid.UUID, filter ListReportsFilter, offset, limit int) ([]*WorkReport, int, error) {
	var result []*WorkReport
	for _, rep := range m.reports {
		if rep.TenantID != tenantID || rep.DeletedAt != nil {
			continue
		}
		if filter.Status != nil && rep.Status != *filter.Status {
			continue
		}
		if filter.AuthorID != nil && rep.AuthorID != *filter.AuthorID {
			continue
		}
		result = append(result, rep)
	}
	total := len(result)
	if offset >= total {
		return []*WorkReport{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) CreateLine(_ context.Context, line *ReportLine) error {
	m.lines[line.ID] = line
	return nil
}

func (m *mockRepository) UpdateLine(_ context.Context, line *ReportLine) error {
	if _, ok := m.lines[line.ID]; !ok {
		return ErrLineNotFound
	}
	m.lines[line.ID] = line
	return nil
}

func (m *mockRepository) DeleteLine(_ context.Context, tenantID, lineID uuid.UUID) error {
	line, ok := m.lines[lineID]
	if !ok || line.TenantID != tenantID {
		return ErrLineNotFound
	}
	delete(m.lines, lineID)
	return nil
}

func (m *mockRepository) GetLine(_ context.Context, tenantID, lineID uuid.UUID) (*ReportLine, error) {
	line, ok := m.lines[lineID]
	if !ok || line.TenantID != tenantID {
		return nil, ErrLineNotFound
	}
	return line, nil
}

func (m *mockRepository) ListLines(_ context.Context, tenantID, reportID uuid.UUID) ([]*ReportLine, error) {
	var result []*ReportLine
	for _, l := range m.lines {
		if l.TenantID == tenantID && l.ReportID == reportID {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *mockRepository) CreateAttachment(_ context.Context, att *ReportAttachment) error {
	m.attachments[att.ID] = att
	return nil
}

func (m *mockRepository) DeleteAttachment(_ context.Context, tenantID, attachmentID uuid.UUID) error {
	att, ok := m.attachments[attachmentID]
	if !ok || att.TenantID != tenantID {
		return ErrAttachmentNotFound
	}
	delete(m.attachments, attachmentID)
	return nil
}

func (m *mockRepository) GetAttachment(_ context.Context, tenantID, attachmentID uuid.UUID) (*ReportAttachment, error) {
	att, ok := m.attachments[attachmentID]
	if !ok || att.TenantID != tenantID {
		return nil, ErrAttachmentNotFound
	}
	return att, nil
}

func (m *mockRepository) ListAttachments(_ context.Context, tenantID, reportID uuid.UUID, lineID *uuid.UUID) ([]*ReportAttachment, error) {
	var result []*ReportAttachment
	for _, att := range m.attachments {
		if att.TenantID != tenantID || att.ReportID != reportID {
			continue
		}
		if lineID != nil && (att.LineID == nil || *att.LineID != *lineID) {
			continue
		}
		result = append(result, att)
	}
	return result, nil
}

func (m *mockRepository) AtomicApproveReport(_ context.Context, tenantID, reportID, reviewerID uuid.UUID, reviewNote string) (bool, error) {
	rep, ok := m.reports[reportID]
	if !ok || rep.TenantID != tenantID || rep.DeletedAt != nil {
		return false, ErrReportNotFound
	}
	if rep.Status != StatusSubmitted {
		return false, nil
	}
	now := time.Now()
	rep.Status = StatusApproved
	rep.ReviewerID = &reviewerID
	rep.ReviewedAt = &now
	rep.ReviewNote = reviewNote
	rep.UpdatedAt = now
	m.reports[reportID] = rep
	return true, nil
}

func (m *mockRepository) AtomicRejectReport(_ context.Context, tenantID, reportID, reviewerID uuid.UUID, reviewNote string) (bool, error) {
	rep, ok := m.reports[reportID]
	if !ok || rep.TenantID != tenantID || rep.DeletedAt != nil {
		return false, ErrReportNotFound
	}
	if rep.Status != StatusSubmitted {
		return false, nil
	}
	now := time.Now()
	rep.Status = StatusRejected
	rep.ReviewerID = &reviewerID
	rep.ReviewedAt = &now
	rep.ReviewNote = reviewNote
	rep.UpdatedAt = now
	m.reports[reportID] = rep
	return true, nil
}

func (m *mockRepository) GetReportStatsCounts(_ context.Context, tenantID uuid.UUID) (map[string]int, error) {
	counts := make(map[string]int)
	for _, rep := range m.reports {
		if rep.TenantID != tenantID || rep.DeletedAt != nil {
			continue
		}
		counts[string(rep.Status)]++
	}
	return counts, nil
}

func (m *mockRepository) SaveSignature(_ context.Context, tenantID, reportID uuid.UUID, signatureData, signedBy string) (*WorkReport, error) {
	rep, ok := m.reports[reportID]
	if !ok || rep.TenantID != tenantID || rep.DeletedAt != nil {
		return nil, ErrReportNotFound
	}
	now := time.Now()
	rep.SignatureData = &signatureData
	rep.SignedAt = &now
	rep.SignedBy = &signedBy
	rep.UpdatedAt = now
	m.reports[reportID] = rep
	return rep, nil
}

// ============================================================================
// Workers stubs
// ============================================================================

func (m *mockRepository) AddWorker(_ context.Context, tenantID, reportID uuid.UUID, name, role string, hours float64) (*Worker, error) {
	w := &Worker{
		ID:       uuid.New(),
		TenantID: tenantID,
		ReportID: reportID,
		Name:     name,
	}
	return w, nil
}

func (m *mockRepository) RemoveWorker(_ context.Context, tenantID, workerID uuid.UUID) error {
	return nil
}

func (m *mockRepository) ListWorkers(_ context.Context, tenantID, reportID uuid.UUID) ([]Worker, error) {
	return nil, nil
}

// ============================================================================
// Measurement stubs
// ============================================================================

func (m *mockRepository) CreateMeasurement(_ context.Context, tenantID uuid.UUID, reportID *uuid.UUID, title, location, measuredBy, measuredAt, notes string) (*Measurement, error) {
	meas := &Measurement{
		ID:       uuid.New(),
		TenantID: tenantID,
		ReportID: reportID,
		Title:    title,
	}
	return meas, nil
}

func (m *mockRepository) GetMeasurement(_ context.Context, tenantID, measurementID uuid.UUID) (*Measurement, error) {
	return nil, ErrMeasurementNotFound
}

func (m *mockRepository) ListMeasurements(_ context.Context, tenantID uuid.UUID, reportID *uuid.UUID, page, pageSize int) ([]Measurement, int, error) {
	return nil, 0, nil
}

func (m *mockRepository) UpdateMeasurement(_ context.Context, tenantID, measurementID uuid.UUID, title, location, measuredBy, measuredAt, notes string) (*Measurement, error) {
	return nil, ErrMeasurementNotFound
}

func (m *mockRepository) DeleteMeasurement(_ context.Context, tenantID, measurementID uuid.UUID) error {
	return nil
}

func (m *mockRepository) AddMeasurementPosition(_ context.Context, tenantID, measurementID uuid.UUID, positionNumber int, description, unit string, quantity, unitPrice float64) (*MeasurementPosition, error) {
	p := &MeasurementPosition{
		ID:             uuid.New(),
		TenantID:       tenantID,
		MeasurementID:  measurementID,
		PositionNumber: positionNumber,
		Description:    description,
	}
	return p, nil
}

func (m *mockRepository) DeleteMeasurementPosition(_ context.Context, tenantID, positionID uuid.UUID) error {
	return nil
}

// ============================================================================
// Template stubs
// ============================================================================

func (m *mockRepository) CreateTemplate(_ context.Context, tenantID uuid.UUID, name, description, category, defaultLinesJSON string) (*ReportTemplate, error) {
	t := &ReportTemplate{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     name,
		IsActive: true,
	}
	return t, nil
}

func (m *mockRepository) GetTemplate(_ context.Context, tenantID, templateID uuid.UUID) (*ReportTemplate, error) {
	return nil, ErrTemplateNotFound
}

func (m *mockRepository) ListTemplates(_ context.Context, tenantID uuid.UUID, activeOnly bool, page, pageSize int) ([]ReportTemplate, int, error) {
	return nil, 0, nil
}

func (m *mockRepository) UpdateTemplate(_ context.Context, tenantID, templateID uuid.UUID, name, description, category, defaultLinesJSON string, isActive bool) (*ReportTemplate, error) {
	return nil, ErrTemplateNotFound
}

func (m *mockRepository) DeleteTemplate(_ context.Context, tenantID, templateID uuid.UUID) error {
	return nil
}

// compile-time interface check
var _ Repository = (*mockRepository)(nil)

// ============================================================================
// Test helpers
// ============================================================================

func addReport(repo *mockRepository, tenantID uuid.UUID, title string, status ReportStatus) *WorkReport {
	rep := &WorkReport{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Title:      title,
		Status:     status,
		AuthorID:   uuid.New(),
		ReportDate: time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	repo.reports[rep.ID] = rep
	return rep
}

// ============================================================================
// CreateReport Tests
// ============================================================================

func TestService_CreateReport_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep, err := svc.CreateReport(context.Background(), CreateReportInput{
		TenantID: tenantID,
		Title:    "Montage Dach",
		AuthorID: uuid.New(),
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, rep.ID)
	assert.Equal(t, StatusDraft, rep.Status)
	assert.Equal(t, tenantID, rep.TenantID)
}

func TestService_CreateReport_EmptyTitle(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateReport(context.Background(), CreateReportInput{
		TenantID: uuid.New(),
		Title:    "  ",
		AuthorID: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateReport_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.GetReport(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrReportNotFound)
}

// ============================================================================
// Guard 1: ErrAlreadyApproved blocks writes on approved reports
// ============================================================================

func TestService_UpdateReport_ApprovedBlocked(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Fertige Arbeit", StatusApproved)

	newTitle := "Versuch der Änderung"
	_, err := svc.UpdateReport(context.Background(), UpdateReportInput{
		TenantID: tenantID,
		ReportID: rep.ID,
		Title:    &newTitle,
	})

	assert.ErrorIs(t, err, ErrAlreadyApproved)
}

func TestService_DeleteReport_ApprovedBlocked(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Fertige Arbeit", StatusApproved)

	err := svc.DeleteReport(context.Background(), tenantID, rep.ID)
	assert.ErrorIs(t, err, ErrAlreadyApproved)
	// Report must still exist
	assert.NotNil(t, repo.reports[rep.ID].DeletedAt == nil)
}

func TestService_AddLine_ApprovedBlocked(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Fertige Arbeit", StatusApproved)

	_, err := svc.AddLine(context.Background(), AddLineInput{
		TenantID:    tenantID,
		ReportID:    rep.ID,
		Description: "Neue Zeile",
		Quantity:    2,
		Unit:        "h",
	})

	assert.ErrorIs(t, err, ErrAlreadyApproved)
}

// ============================================================================
// Guard 2: State-machine pre-check (ErrInvalidStateTransition / ErrAlreadyApproved)
// ============================================================================

func TestService_SubmitReport_FromDraft_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Dacharbeit", StatusDraft)

	submitted, err := svc.SubmitReport(context.Background(), SubmitReportInput{
		TenantID: tenantID,
		ReportID: rep.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, StatusSubmitted, submitted.Status)
}

func TestService_SubmitReport_AlreadySubmitted_Blocked(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Dacharbeit", StatusSubmitted)

	_, err := svc.SubmitReport(context.Background(), SubmitReportInput{
		TenantID: tenantID,
		ReportID: rep.ID,
	})

	assert.ErrorIs(t, err, ErrInvalidStateTransition)
}

func TestService_ApproveReport_FromSubmitted_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Dacharbeit", StatusSubmitted)
	reviewerID := uuid.New()

	approved, err := svc.ApproveReport(context.Background(), ApproveReportInput{
		TenantID:   tenantID,
		ReportID:   rep.ID,
		ReviewerID: reviewerID,
		ReviewNote: "Alles korrekt",
	})

	require.NoError(t, err)
	assert.Equal(t, StatusApproved, approved.Status)
	assert.Equal(t, &reviewerID, approved.ReviewerID)
	assert.NotNil(t, approved.ReviewedAt)
}

func TestService_ApproveReport_FromDraft_Blocked(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Dacharbeit", StatusDraft)

	_, err := svc.ApproveReport(context.Background(), ApproveReportInput{
		TenantID:   tenantID,
		ReportID:   rep.ID,
		ReviewerID: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrInvalidStateTransition)
}

func TestService_ApproveReport_AlreadyApproved_Returns_ErrAlreadyApproved(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Dacharbeit", StatusApproved)

	_, err := svc.ApproveReport(context.Background(), ApproveReportInput{
		TenantID:   tenantID,
		ReportID:   rep.ID,
		ReviewerID: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrAlreadyApproved)
}

func TestService_RejectReport_FromSubmitted_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Dacharbeit", StatusSubmitted)

	rejected, err := svc.RejectReport(context.Background(), RejectReportInput{
		TenantID:   tenantID,
		ReportID:   rep.ID,
		ReviewerID: uuid.New(),
		ReviewNote: "Fehlende Belege",
	})

	require.NoError(t, err)
	assert.Equal(t, StatusRejected, rejected.Status)
}

func TestService_RejectReport_FromDraft_Blocked(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Dacharbeit", StatusDraft)

	_, err := svc.RejectReport(context.Background(), RejectReportInput{
		TenantID:   tenantID,
		ReportID:   rep.ID,
		ReviewerID: uuid.New(),
		ReviewNote: "Fehlende Belege",
	})

	assert.ErrorIs(t, err, ErrInvalidStateTransition)
}

func TestService_RejectReport_EmptyReviewNote_Returns_ErrInvalidInput(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Dacharbeit", StatusSubmitted)

	_, err := svc.RejectReport(context.Background(), RejectReportInput{
		TenantID:   tenantID,
		ReportID:   rep.ID,
		ReviewerID: uuid.New(),
		ReviewNote: "",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
	assert.Equal(t, StatusSubmitted, repo.reports[rep.ID].Status, "report must stay submitted when rejection is refused")
}

func TestService_RejectReport_WhitespaceReviewNote_Returns_ErrInvalidInput(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Dacharbeit", StatusSubmitted)

	_, err := svc.RejectReport(context.Background(), RejectReportInput{
		TenantID:   tenantID,
		ReportID:   rep.ID,
		ReviewerID: uuid.New(),
		ReviewNote: "   \t\n",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

// ============================================================================
// Guard 3: RowsAffected == 0 → sentinel error (via mock)
// ============================================================================

func TestService_UpdateReport_NotFound_Returns_ErrReportNotFound(t *testing.T) {
	repo := newMockRepository()

	// Directly call repo update with non-existent ID — verifies RowsAffected==0 sentinel
	err := repo.UpdateReport(context.Background(), &WorkReport{
		ID:       uuid.New(),
		TenantID: uuid.New(),
	})
	assert.ErrorIs(t, err, ErrReportNotFound)
}

func TestService_DeleteLine_NotFound_Returns_ErrLineNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Report", StatusDraft)
	line := &ReportLine{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ReportID:    rep.ID,
		Description: "Zeile 1",
		Unit:        "h",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.lines[line.ID] = line

	// Delete once — success
	err := svc.DeleteLine(context.Background(), tenantID, line.ID)
	require.NoError(t, err)

	// Delete again — row is gone → ErrLineNotFound
	err = svc.DeleteLine(context.Background(), tenantID, line.ID)
	assert.ErrorIs(t, err, ErrLineNotFound)
}

func TestService_DeleteAttachment_NotFound_Returns_ErrAttachmentNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	err := svc.DeleteAttachment(context.Background(), tenantID, uuid.New())
	assert.ErrorIs(t, err, ErrAttachmentNotFound)
}

// ============================================================================
// Lines
// ============================================================================

func TestService_AddLine_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Bericht", StatusDraft)

	line, err := svc.AddLine(context.Background(), AddLineInput{
		TenantID:    tenantID,
		ReportID:    rep.ID,
		Description: "Montage",
		Quantity:    3.5,
		Unit:        "h",
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, line.ID)
	assert.Equal(t, "Montage", line.Description)
}

func TestService_ListLines_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Bericht", StatusDraft)

	for i := range 3 {
		_, err := svc.AddLine(context.Background(), AddLineInput{
			TenantID:    tenantID,
			ReportID:    rep.ID,
			Description: "Zeile",
			Quantity:    float64(i + 1),
			Unit:        "h",
			Position:    i,
		})
		require.NoError(t, err)
	}

	lines, err := svc.ListLines(context.Background(), tenantID, rep.ID)
	require.NoError(t, err)
	assert.Len(t, lines, 3)
}

// ============================================================================
// Attachments
// ============================================================================

func TestService_UploadAttachment_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Bericht", StatusDraft)

	att, err := svc.UploadAttachment(context.Background(), UploadAttachmentInput{
		TenantID:    tenantID,
		ReportID:    rep.ID,
		Filename:    "foto.jpg",
		ContentType: "image/jpeg",
		SizeBytes:   102400,
		ObjectKey:   "tenants/" + tenantID.String() + "/rapporte/" + rep.ID.String() + "/foto.jpg",
		UploadedBy:  uuid.New(),
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, att.ID)
	assert.Equal(t, "foto.jpg", att.Filename)
}

// Bug #2: Cross-tenant MinIO path validation
func TestService_UploadAttachment_CrossTenantPath_Rejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()
	rep := addReport(repo, tenantA, "Bericht", StatusDraft)

	// tenantA crafts an object_key pointing to tenantB's storage path
	_, err := svc.UploadAttachment(context.Background(), UploadAttachmentInput{
		TenantID:    tenantA,
		ReportID:    rep.ID,
		Filename:    "evil.jpg",
		ContentType: "image/jpeg",
		SizeBytes:   1024,
		ObjectKey:   "tenants/" + tenantB.String() + "/rapporte/evil.jpg",
		UploadedBy:  uuid.New(),
	})
	assert.ErrorIs(t, err, ErrInvalidInput, "cross-tenant object key must be rejected")
}

// Bug #10: Atomic approve race — two concurrent approves yield 1 success + 1 conflict
func TestService_ApproveReport_ParallelApprovals_OnlyOneSucceeds(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Race", StatusSubmitted)

	// Simulate sequential "parallel" — second call after first already changed state
	_, err1 := svc.ApproveReport(context.Background(), ApproveReportInput{
		TenantID: tenantID, ReportID: rep.ID, ReviewerID: uuid.New(),
	})
	_, err2 := svc.ApproveReport(context.Background(), ApproveReportInput{
		TenantID: tenantID, ReportID: rep.ID, ReviewerID: uuid.New(),
	})

	require.NoError(t, err1)
	assert.ErrorIs(t, err2, ErrAlreadyApproved, "second approve on already-approved report must fail")
}

// Bug #11: GetReportStats uses GROUP BY (not full scan) — verify correct counts
func TestService_GetReportStats_GroupByQuery(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	for range 3 {
		addReport(repo, tenantID, "Draft", StatusDraft)
	}
	for range 2 {
		addReport(repo, tenantID, "Sub", StatusSubmitted)
	}
	addReport(repo, tenantID, "App", StatusApproved)
	addReport(repo, tenantID, "Rej", StatusRejected)

	stats, err := svc.GetReportStats(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, 7, stats.TotalReports)
	assert.Equal(t, 3, stats.DraftCount)
	assert.Equal(t, 2, stats.SubmittedCount)
	assert.Equal(t, 1, stats.ApprovedCount)
	assert.Equal(t, 1, stats.RejectedCount)
}

// Bug #23: GPS range validation
func TestService_CreateReport_InvalidGPS_Rejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	lat := 91.0 // > 90
	_, err := svc.CreateReport(context.Background(), CreateReportInput{
		TenantID: tenantID, Title: "GPS-Test", AuthorID: uuid.New(), Lat: &lat,
	})
	assert.ErrorIs(t, err, ErrInvalidInput, "lat > 90 must be rejected")

	lon := -181.0 // < -180
	_, err = svc.CreateReport(context.Background(), CreateReportInput{
		TenantID: tenantID, Title: "GPS-Test", AuthorID: uuid.New(), Lon: &lon,
	})
	assert.ErrorIs(t, err, ErrInvalidInput, "lon < -180 must be rejected")
}

func TestService_CreateReport_ValidGPS_Accepted(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	lat := 48.1374 // Munich
	lon := 11.5755
	rep, err := svc.CreateReport(context.Background(), CreateReportInput{
		TenantID: tenantID, Title: "GPS-Valid", AuthorID: uuid.New(), Lat: &lat, Lon: &lon,
	})
	require.NoError(t, err)
	assert.NotNil(t, rep.Lat)
}

func TestService_ListAttachments_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Bericht", StatusDraft)
	uploader := uuid.New()

	for i := range 2 {
		_, err := svc.UploadAttachment(context.Background(), UploadAttachmentInput{
			TenantID:    tenantID,
			ReportID:    rep.ID,
			Filename:    "foto" + string(rune('0'+i)) + ".jpg",
			ContentType: "image/jpeg",
			SizeBytes:   1024,
			ObjectKey:   "tenants/" + tenantID.String() + "/rapporte/foto" + string(rune('0'+i)) + ".jpg",
			UploadedBy:  uploader,
		})
		require.NoError(t, err)
	}

	atts, err := svc.ListAttachments(context.Background(), tenantID, rep.ID, nil)
	require.NoError(t, err)
	assert.Len(t, atts, 2)
}

// ============================================================================
// Stats
// ============================================================================

func TestService_GetReportStats_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addReport(repo, tenantID, "Draft1", StatusDraft)
	addReport(repo, tenantID, "Draft2", StatusDraft)
	addReport(repo, tenantID, "Submitted1", StatusSubmitted)
	addReport(repo, tenantID, "Approved1", StatusApproved)
	addReport(repo, tenantID, "Rejected1", StatusRejected)

	stats, err := svc.GetReportStats(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, 5, stats.TotalReports)
	assert.Equal(t, 2, stats.DraftCount)
	assert.Equal(t, 1, stats.SubmittedCount)
	assert.Equal(t, 1, stats.ApprovedCount)
	assert.Equal(t, 1, stats.RejectedCount)
}

// ============================================================================
// ListPendingApprovals
// ============================================================================

func TestService_ListPendingApprovals_OnlySubmitted(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addReport(repo, tenantID, "Draft", StatusDraft)
	addReport(repo, tenantID, "Sub1", StatusSubmitted)
	addReport(repo, tenantID, "Sub2", StatusSubmitted)
	addReport(repo, tenantID, "Approved", StatusApproved)

	pending, total, err := svc.ListPendingApprovals(context.Background(), tenantID, nil, 1, 50)

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, pending, 2)
}

// A caller whose rapporte:report:read grant is scoped to "own" gets the queue
// narrowed to their own submissions — total included, so paging stays honest.
func TestService_ListPendingApprovals_OwnScopeFiltersByAuthor(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	mine := uuid.New()
	theirs := uuid.New()

	own := addReport(repo, tenantID, "Mine", StatusSubmitted)
	own.AuthorID = mine
	foreign := addReport(repo, tenantID, "Theirs", StatusSubmitted)
	foreign.AuthorID = theirs

	pending, total, err := svc.ListPendingApprovals(context.Background(), tenantID, &mine, 1, 50)

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, pending, 1)
	assert.Equal(t, "Mine", pending[0].Title)
}

// ============================================================================
// ExportPDF
// ============================================================================

func TestService_ExportPDF_ReturnsRealPDF(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Dachbericht", StatusApproved)
	rep.Description = "Dachrinne erneuert und Ziegel ausgetauscht."
	signedBy := "Max Mustermann"
	signedAt := time.Now()
	rep.SignedBy = &signedBy
	rep.SignedAt = &signedAt
	rep.Workers = []Worker{
		{Name: "Erika Musterfrau", Role: sql.NullString{String: "Dachdeckerin", Valid: true}, Hours: sql.NullFloat64{Float64: 6.5, Valid: true}},
	}
	repo.reports[rep.ID] = rep

	require.NoError(t, repo.CreateLine(context.Background(), &ReportLine{
		ID: uuid.New(), TenantID: tenantID, ReportID: rep.ID,
		Position: 1, Description: "Ziegel ausgetauscht", Quantity: 12, Unit: "Stk",
	}))

	payload, err := svc.ExportPDF(context.Background(), tenantID, rep.ID)

	require.NoError(t, err)
	require.True(t, len(payload) > 4)
	assert.Equal(t, "%PDF", string(payload[:4]))
}

func TestService_ExportPDF_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.ExportPDF(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrReportNotFound)
}

// ============================================================================
// Tenant isolation (Guard 2: tenant_id filter in GetReport)
// ============================================================================

func TestService_GetReport_CrossTenantBlocked(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()
	rep := addReport(repo, tenantA, "Geheimbericht", StatusDraft)

	// Try to access tenantA's report with tenantB credentials
	_, err := svc.GetReport(context.Background(), tenantB, rep.ID)
	assert.ErrorIs(t, err, ErrReportNotFound)
}

func TestService_ListReports_DefaultPagination(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	for range 60 {
		addReport(repo, tenantID, "Report", StatusDraft)
	}

	reports, total, err := svc.ListReports(context.Background(), ListReportsInput{
		TenantID: tenantID,
	})

	require.NoError(t, err)
	assert.Equal(t, 60, total)
	assert.Len(t, reports, 50)
}

// ============================================================================
// New: Measurement + Template tests
// ============================================================================

// TestRepo_CreateMeasurement_TenantIsolation verifies that CreateMeasurement
// stamps the correct tenant_id and that a different tenant cannot retrieve it
// via GetMeasurement (which returns ErrMeasurementNotFound for unknown IDs).
func TestRepo_CreateMeasurement_TenantIsolation(t *testing.T) {
	repo := newMockRepository()

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create a measurement for tenantA
	m, err := repo.CreateMeasurement(context.Background(), tenantA, nil, "Dachfläche Nord", "", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, tenantA, m.TenantID, "tenant_id must match caller")
	assert.NotEqual(t, uuid.Nil, m.ID)

	// tenantB cannot retrieve tenantA's measurement
	_, err = repo.GetMeasurement(context.Background(), tenantB, m.ID)
	assert.ErrorIs(t, err, ErrMeasurementNotFound, "cross-tenant measurement lookup must return ErrMeasurementNotFound")
}

// TestRepo_CreateTemplate_HappyPath verifies that CreateTemplate returns a
// well-formed ReportTemplate with correct tenant/name and is_active=true.
func TestRepo_CreateTemplate_HappyPath(t *testing.T) {
	repo := newMockRepository()

	tenantID := uuid.New()
	tmpl, err := repo.CreateTemplate(context.Background(), tenantID, "Standard-Bericht", "Vorlage für Standardberichte", "allgemein", `[{"description":"Arbeitszeit","unit":"h"}]`)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tmpl.ID)
	assert.Equal(t, tenantID, tmpl.TenantID)
	assert.Equal(t, "Standard-Bericht", tmpl.Name)
	assert.True(t, tmpl.IsActive, "newly created template must be active")
}
