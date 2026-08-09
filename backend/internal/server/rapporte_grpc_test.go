package server

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/rapporte"
	rapportev1 "github.com/kmuhub/kmuhub/proto/rapporte/v1"
)

// ============================================================================
// Stub repository (server-package copy — rapporte's own test doubles live in
// that package's service_test.go and aren't importable here). Reports are
// held in-memory with real status transitions so the atomic approve/reject
// RPCs exercise the same TOCTOU-safe semantics as the real repository: an
// approve/reject only succeeds when the row is currently 'submitted'.
// ============================================================================

type stubRapporteRepo struct {
	mu  sync.Mutex
	err error

	reports      map[uuid.UUID]*rapporte.WorkReport
	lines        map[uuid.UUID]*rapporte.ReportLine
	attachments  map[uuid.UUID]*rapporte.ReportAttachment
	workers      map[uuid.UUID]*rapporte.Worker
	measurements map[uuid.UUID]*rapporte.Measurement
	positions    map[uuid.UUID]*rapporte.MeasurementPosition
	templates    map[uuid.UUID]*rapporte.ReportTemplate
}

func newStubRapporteRepo() *stubRapporteRepo {
	return &stubRapporteRepo{
		reports:      make(map[uuid.UUID]*rapporte.WorkReport),
		lines:        make(map[uuid.UUID]*rapporte.ReportLine),
		attachments:  make(map[uuid.UUID]*rapporte.ReportAttachment),
		workers:      make(map[uuid.UUID]*rapporte.Worker),
		measurements: make(map[uuid.UUID]*rapporte.Measurement),
		positions:    make(map[uuid.UUID]*rapporte.MeasurementPosition),
		templates:    make(map[uuid.UUID]*rapporte.ReportTemplate),
	}
}

func (r *stubRapporteRepo) seedReport(rep *rapporte.WorkReport) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *rep
	r.reports[rep.ID] = &cp
}

// --- Reports ---

func (r *stubRapporteRepo) CreateReport(_ context.Context, report *rapporte.WorkReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	cp := *report
	r.reports[report.ID] = &cp
	return nil
}

func (r *stubRapporteRepo) UpdateReport(_ context.Context, report *rapporte.WorkReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	cp := *report
	r.reports[report.ID] = &cp
	return nil
}

func (r *stubRapporteRepo) SoftDeleteReport(_ context.Context, tenantID, reportID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	rep, ok := r.reports[reportID]
	if !ok || rep.TenantID != tenantID {
		return rapporte.ErrReportNotFound
	}
	delete(r.reports, reportID)
	return nil
}

func (r *stubRapporteRepo) GetReport(_ context.Context, tenantID, reportID uuid.UUID) (*rapporte.WorkReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	rep, ok := r.reports[reportID]
	if !ok || rep.TenantID != tenantID {
		return nil, rapporte.ErrReportNotFound
	}
	cp := *rep
	return &cp, nil
}

func (r *stubRapporteRepo) ListReports(_ context.Context, tenantID uuid.UUID, filter rapporte.ListReportsFilter, offset, limit int) ([]*rapporte.WorkReport, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, 0, r.err
	}
	var all []*rapporte.WorkReport
	for _, rep := range r.reports {
		if rep.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && rep.Status != *filter.Status {
			continue
		}
		if filter.AuthorID != nil && rep.AuthorID != *filter.AuthorID {
			continue
		}
		cp := *rep
		all = append(all, &cp)
	}
	total := len(all)
	end := min(offset+limit, total)
	start := min(offset, total)
	if limit <= 0 {
		return all, total, nil
	}
	return all[start:end], total, nil
}

func (r *stubRapporteRepo) AtomicApproveReport(_ context.Context, tenantID, reportID, reviewerID uuid.UUID, reviewNote string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return false, r.err
	}
	rep, ok := r.reports[reportID]
	if !ok || rep.TenantID != tenantID || rep.Status != rapporte.StatusSubmitted {
		return false, nil
	}
	now := time.Now()
	rep.Status = rapporte.StatusApproved
	rep.ReviewerID = &reviewerID
	rep.ReviewNote = reviewNote
	rep.ReviewedAt = &now
	return true, nil
}

func (r *stubRapporteRepo) AtomicRejectReport(_ context.Context, tenantID, reportID, reviewerID uuid.UUID, reviewNote string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return false, r.err
	}
	rep, ok := r.reports[reportID]
	if !ok || rep.TenantID != tenantID || rep.Status != rapporte.StatusSubmitted {
		return false, nil
	}
	now := time.Now()
	rep.Status = rapporte.StatusRejected
	rep.ReviewerID = &reviewerID
	rep.ReviewNote = reviewNote
	rep.ReviewedAt = &now
	return true, nil
}

func (r *stubRapporteRepo) SaveSignature(_ context.Context, tenantID, reportID uuid.UUID, signatureData, signedBy string) (*rapporte.WorkReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	rep, ok := r.reports[reportID]
	if !ok || rep.TenantID != tenantID {
		return nil, rapporte.ErrReportNotFound
	}
	now := time.Now()
	sd := signatureData
	sb := signedBy
	rep.SignatureData = &sd
	rep.SignedBy = &sb
	rep.SignedAt = &now
	cp := *rep
	return &cp, nil
}

func (r *stubRapporteRepo) GetReportStatsCounts(_ context.Context, tenantID uuid.UUID) (map[string]int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	counts := make(map[string]int)
	for _, rep := range r.reports {
		if rep.TenantID != tenantID {
			continue
		}
		counts[string(rep.Status)]++
	}
	return counts, nil
}

// --- Lines ---

func (r *stubRapporteRepo) CreateLine(_ context.Context, line *rapporte.ReportLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	cp := *line
	r.lines[line.ID] = &cp
	return nil
}

func (r *stubRapporteRepo) UpdateLine(_ context.Context, line *rapporte.ReportLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	cp := *line
	r.lines[line.ID] = &cp
	return nil
}

func (r *stubRapporteRepo) DeleteLine(_ context.Context, tenantID, lineID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	l, ok := r.lines[lineID]
	if !ok || l.TenantID != tenantID {
		return rapporte.ErrLineNotFound
	}
	delete(r.lines, lineID)
	return nil
}

func (r *stubRapporteRepo) GetLine(_ context.Context, tenantID, lineID uuid.UUID) (*rapporte.ReportLine, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	l, ok := r.lines[lineID]
	if !ok || l.TenantID != tenantID {
		return nil, rapporte.ErrLineNotFound
	}
	cp := *l
	return &cp, nil
}

func (r *stubRapporteRepo) ListLines(_ context.Context, tenantID, reportID uuid.UUID) ([]*rapporte.ReportLine, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	var out []*rapporte.ReportLine
	for _, l := range r.lines {
		if l.TenantID == tenantID && l.ReportID == reportID {
			cp := *l
			out = append(out, &cp)
		}
	}
	return out, nil
}

// --- Attachments ---

func (r *stubRapporteRepo) CreateAttachment(_ context.Context, att *rapporte.ReportAttachment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	cp := *att
	r.attachments[att.ID] = &cp
	return nil
}

func (r *stubRapporteRepo) DeleteAttachment(_ context.Context, tenantID, attachmentID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	a, ok := r.attachments[attachmentID]
	if !ok || a.TenantID != tenantID {
		return rapporte.ErrAttachmentNotFound
	}
	delete(r.attachments, attachmentID)
	return nil
}

func (r *stubRapporteRepo) GetAttachment(_ context.Context, tenantID, attachmentID uuid.UUID) (*rapporte.ReportAttachment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	a, ok := r.attachments[attachmentID]
	if !ok || a.TenantID != tenantID {
		return nil, rapporte.ErrAttachmentNotFound
	}
	cp := *a
	return &cp, nil
}

func (r *stubRapporteRepo) ListAttachments(_ context.Context, tenantID, reportID uuid.UUID, lineID *uuid.UUID) ([]*rapporte.ReportAttachment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	var out []*rapporte.ReportAttachment
	for _, a := range r.attachments {
		if a.TenantID != tenantID || a.ReportID != reportID {
			continue
		}
		if lineID != nil && (a.LineID == nil || *a.LineID != *lineID) {
			continue
		}
		cp := *a
		out = append(out, &cp)
	}
	return out, nil
}

// --- Workers ---

func (r *stubRapporteRepo) AddWorker(_ context.Context, tenantID, reportID uuid.UUID, name, role string, hours float64) (*rapporte.Worker, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	w := &rapporte.Worker{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ReportID:  reportID,
		Name:      name,
		Role:      sql.NullString{String: role, Valid: role != ""},
		Hours:     sql.NullFloat64{Float64: hours, Valid: true},
		CreatedAt: time.Now(),
	}
	r.workers[w.ID] = w
	cp := *w
	return &cp, nil
}

func (r *stubRapporteRepo) RemoveWorker(_ context.Context, tenantID, workerID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	w, ok := r.workers[workerID]
	if !ok || w.TenantID != tenantID {
		return rapporte.ErrWorkerNotFound
	}
	delete(r.workers, workerID)
	return nil
}

func (r *stubRapporteRepo) ListWorkers(_ context.Context, tenantID, reportID uuid.UUID) ([]rapporte.Worker, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	var out []rapporte.Worker
	for _, w := range r.workers {
		if w.TenantID == tenantID && w.ReportID == reportID {
			out = append(out, *w)
		}
	}
	return out, nil
}

// --- Measurements ---

func (r *stubRapporteRepo) CreateMeasurement(_ context.Context, tenantID uuid.UUID, reportID *uuid.UUID, title, location, measuredBy, measuredAt, notes string) (*rapporte.Measurement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	m := &rapporte.Measurement{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ReportID:   reportID,
		Title:      title,
		Location:   sql.NullString{String: location, Valid: location != ""},
		MeasuredBy: sql.NullString{String: measuredBy, Valid: measuredBy != ""},
		MeasuredAt: sql.NullString{String: measuredAt, Valid: measuredAt != ""},
		Notes:      sql.NullString{String: notes, Valid: notes != ""},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	r.measurements[m.ID] = m
	cp := *m
	return &cp, nil
}

func (r *stubRapporteRepo) GetMeasurement(_ context.Context, tenantID, measurementID uuid.UUID) (*rapporte.Measurement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	m, ok := r.measurements[measurementID]
	if !ok || m.TenantID != tenantID {
		return nil, rapporte.ErrMeasurementNotFound
	}
	cp := *m
	return &cp, nil
}

func (r *stubRapporteRepo) ListMeasurements(_ context.Context, tenantID uuid.UUID, reportID *uuid.UUID, page, pageSize int) ([]rapporte.Measurement, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, 0, r.err
	}
	var out []rapporte.Measurement
	for _, m := range r.measurements {
		if m.TenantID != tenantID {
			continue
		}
		if reportID != nil && (m.ReportID == nil || *m.ReportID != *reportID) {
			continue
		}
		out = append(out, *m)
	}
	return out, len(out), nil
}

func (r *stubRapporteRepo) UpdateMeasurement(_ context.Context, tenantID, measurementID uuid.UUID, title, location, measuredBy, measuredAt, notes string) (*rapporte.Measurement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	m, ok := r.measurements[measurementID]
	if !ok || m.TenantID != tenantID {
		return nil, rapporte.ErrMeasurementNotFound
	}
	m.Title = title
	m.Location = sql.NullString{String: location, Valid: location != ""}
	m.MeasuredBy = sql.NullString{String: measuredBy, Valid: measuredBy != ""}
	m.MeasuredAt = sql.NullString{String: measuredAt, Valid: measuredAt != ""}
	m.Notes = sql.NullString{String: notes, Valid: notes != ""}
	m.UpdatedAt = time.Now()
	cp := *m
	return &cp, nil
}

func (r *stubRapporteRepo) DeleteMeasurement(_ context.Context, tenantID, measurementID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	m, ok := r.measurements[measurementID]
	if !ok || m.TenantID != tenantID {
		return rapporte.ErrMeasurementNotFound
	}
	delete(r.measurements, measurementID)
	return nil
}

func (r *stubRapporteRepo) AddMeasurementPosition(_ context.Context, tenantID, measurementID uuid.UUID, positionNumber int, description, unit string, quantity, unitPrice float64) (*rapporte.MeasurementPosition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	p := &rapporte.MeasurementPosition{
		ID:             uuid.New(),
		TenantID:       tenantID,
		MeasurementID:  measurementID,
		PositionNumber: positionNumber,
		Description:    description,
		Unit:           sql.NullString{String: unit, Valid: unit != ""},
		Quantity:       sql.NullFloat64{Float64: quantity, Valid: true},
		UnitPrice:      sql.NullFloat64{Float64: unitPrice, Valid: true},
		CreatedAt:      time.Now(),
	}
	r.positions[p.ID] = p
	cp := *p
	return &cp, nil
}

func (r *stubRapporteRepo) DeleteMeasurementPosition(_ context.Context, tenantID, positionID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	p, ok := r.positions[positionID]
	if !ok || p.TenantID != tenantID {
		return rapporte.ErrPositionNotFound
	}
	delete(r.positions, positionID)
	return nil
}

// --- Templates ---

func (r *stubRapporteRepo) CreateTemplate(_ context.Context, tenantID uuid.UUID, name, description, category, defaultLinesJSON string) (*rapporte.ReportTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	t := &rapporte.ReportTemplate{
		ID:               uuid.New(),
		TenantID:         tenantID,
		Name:             name,
		Description:      sql.NullString{String: description, Valid: description != ""},
		Category:         sql.NullString{String: category, Valid: category != ""},
		DefaultLinesJSON: sql.NullString{String: defaultLinesJSON, Valid: defaultLinesJSON != ""},
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	r.templates[t.ID] = t
	cp := *t
	return &cp, nil
}

func (r *stubRapporteRepo) GetTemplate(_ context.Context, tenantID, templateID uuid.UUID) (*rapporte.ReportTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	t, ok := r.templates[templateID]
	if !ok || t.TenantID != tenantID {
		return nil, rapporte.ErrTemplateNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *stubRapporteRepo) ListTemplates(_ context.Context, tenantID uuid.UUID, activeOnly bool, page, pageSize int) ([]rapporte.ReportTemplate, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, 0, r.err
	}
	var out []rapporte.ReportTemplate
	for _, t := range r.templates {
		if t.TenantID != tenantID {
			continue
		}
		if activeOnly && !t.IsActive {
			continue
		}
		out = append(out, *t)
	}
	return out, len(out), nil
}

func (r *stubRapporteRepo) UpdateTemplate(_ context.Context, tenantID, templateID uuid.UUID, name, description, category, defaultLinesJSON string, isActive bool) (*rapporte.ReportTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	t, ok := r.templates[templateID]
	if !ok || t.TenantID != tenantID {
		return nil, rapporte.ErrTemplateNotFound
	}
	t.Name = name
	t.Description = sql.NullString{String: description, Valid: description != ""}
	t.Category = sql.NullString{String: category, Valid: category != ""}
	t.DefaultLinesJSON = sql.NullString{String: defaultLinesJSON, Valid: defaultLinesJSON != ""}
	t.IsActive = isActive
	t.UpdatedAt = time.Now()
	cp := *t
	return &cp, nil
}

func (r *stubRapporteRepo) DeleteTemplate(_ context.Context, tenantID, templateID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	t, ok := r.templates[templateID]
	if !ok || t.TenantID != tenantID {
		return rapporte.ErrTemplateNotFound
	}
	delete(r.templates, templateID)
	return nil
}

// ============================================================================
// Test helpers
// ============================================================================

func newRapporteTestServer(repo *stubRapporteRepo) *RapporteGRPCServer {
	return NewRapporteGRPCServer(rapporte.NewService(repo), repo)
}

func rapporteCtxWithTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

func newDraftReport(tenantID uuid.UUID) *rapporte.WorkReport {
	now := time.Now()
	return &rapporte.WorkReport{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Title:      "Baustelle Muster",
		Status:     rapporte.StatusDraft,
		AuthorID:   uuid.New(),
		ReportDate: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// ============================================================================
// Report RPCs
// ============================================================================

func TestRapporteReportHandlers(t *testing.T) {
	tenantID := uuid.New()
	reportID := uuid.New()

	t.Run("CreateReport missing tenant is unauthenticated", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.CreateReport(context.Background(), &rapportev1.CreateReportRequest{Title: "x", AuthorId: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("CreateReport invalid author_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.CreateReport(rapporteCtxWithTenant(tenantID), &rapportev1.CreateReportRequest{Title: "x", AuthorId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateReport invalid report_date", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.CreateReport(rapporteCtxWithTenant(tenantID), &rapportev1.CreateReportRequest{
			Title: "x", AuthorId: uuid.New().String(), ReportDate: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateReport empty title maps to invalid argument", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.CreateReport(rapporteCtxWithTenant(tenantID), &rapportev1.CreateReportRequest{
			Title: "   ", AuthorId: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateReport scopes the write to the context tenant, not a client-suppliable value", func(t *testing.T) {
		repo := newStubRapporteRepo()
		s := newRapporteTestServer(repo)
		resp, err := s.CreateReport(rapporteCtxWithTenant(tenantID), &rapportev1.CreateReportRequest{
			TenantId: uuid.New().String(), // must be ignored — tenant comes from ctx
			Title:    "Baustelle A",
			AuthorId: uuid.New().String(),
		})
		requireGRPCOK(t, err)
		require.Equal(t, tenantID.String(), resp.Report.TenantId)
		require.Equal(t, "draft", resp.Report.Status)
	})

	t.Run("CreateReport repo error maps through", func(t *testing.T) {
		repo := newStubRapporteRepo()
		repo.err = rapporte.ErrInvalidInput
		s := newRapporteTestServer(repo)
		_, err := s.CreateReport(rapporteCtxWithTenant(tenantID), &rapportev1.CreateReportRequest{
			Title: "x", AuthorId: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetReport missing tenant is unauthenticated", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.GetReport(context.Background(), &rapportev1.GetReportRequest{ReportId: reportID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("GetReport invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.GetReport(rapporteCtxWithTenant(tenantID), &rapportev1.GetReportRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetReport unknown id maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.GetReport(rapporteCtxWithTenant(tenantID), &rapportev1.GetReportRequest{ReportId: reportID.String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("GetReport happy path", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		resp, err := s.GetReport(rapporteCtxWithTenant(tenantID), &rapportev1.GetReportRequest{ReportId: rep.ID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, rep.ID.String(), resp.Report.Id)
	})

	t.Run("UpdateReport invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.UpdateReport(rapporteCtxWithTenant(tenantID), &rapportev1.UpdateReportRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateReport invalid report_date", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		bad := "nope"
		_, err := s.UpdateReport(rapporteCtxWithTenant(tenantID), &rapportev1.UpdateReportRequest{
			ReportId: rep.ID.String(), ReportDate: &bad,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateReport on approved report is rejected", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		rep.Status = rapporte.StatusApproved
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		title := "Neuer Titel"
		_, err := s.UpdateReport(rapporteCtxWithTenant(tenantID), &rapportev1.UpdateReportRequest{
			ReportId: rep.ID.String(), Title: &title,
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("DeleteReport invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.DeleteReport(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteReportRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteReport on approved report is rejected", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		rep.Status = rapporte.StatusApproved
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		_, err := s.DeleteReport(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteReportRequest{ReportId: rep.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("DeleteReport happy path", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		_, err := s.DeleteReport(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteReportRequest{ReportId: rep.ID.String()})
		requireGRPCOK(t, err)
	})

	t.Run("ListReports invalid author_id filter", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		bad := "bad"
		_, err := s.ListReports(rapporteCtxWithTenant(tenantID), &rapportev1.ListReportsRequest{AuthorId: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListReports empty result wraps as empty slice not null", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		resp, err := s.ListReports(rapporteCtxWithTenant(tenantID), &rapportev1.ListReportsRequest{})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Reports)
		require.Empty(t, resp.Reports)
	})

	t.Run("ListReports filters by status", func(t *testing.T) {
		repo := newStubRapporteRepo()
		draft := newDraftReport(tenantID)
		submitted := newDraftReport(tenantID)
		submitted.Status = rapporte.StatusSubmitted
		repo.seedReport(draft)
		repo.seedReport(submitted)
		s := newRapporteTestServer(repo)
		status := "submitted"
		resp, err := s.ListReports(rapporteCtxWithTenant(tenantID), &rapportev1.ListReportsRequest{Status: &status})
		requireGRPCOK(t, err)
		require.Len(t, resp.Reports, 1)
		require.Equal(t, submitted.ID.String(), resp.Reports[0].Id)
	})
}

// ============================================================================
// State machine RPCs
// ============================================================================

func TestRapporteStateMachineHandlers(t *testing.T) {
	tenantID := uuid.New()

	t.Run("SubmitReport invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.SubmitReport(rapporteCtxWithTenant(tenantID), &rapportev1.SubmitReportRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("SubmitReport from draft succeeds", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		resp, err := s.SubmitReport(rapporteCtxWithTenant(tenantID), &rapportev1.SubmitReportRequest{ReportId: rep.ID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, "submitted", resp.Report.Status)
	})

	t.Run("SubmitReport from submitted is an invalid transition", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		rep.Status = rapporte.StatusSubmitted
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		_, err := s.SubmitReport(rapporteCtxWithTenant(tenantID), &rapportev1.SubmitReportRequest{ReportId: rep.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("ApproveReport invalid reviewer_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.ApproveReport(rapporteCtxWithTenant(tenantID), &rapportev1.ApproveReportRequest{
			ReportId: uuid.New().String(), ReviewerId: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ApproveReport from submitted succeeds", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		rep.Status = rapporte.StatusSubmitted
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		resp, err := s.ApproveReport(rapporteCtxWithTenant(tenantID), &rapportev1.ApproveReportRequest{
			ReportId: rep.ID.String(), ReviewerId: uuid.New().String(), ReviewNote: "Alles korrekt",
		})
		requireGRPCOK(t, err)
		require.Equal(t, "approved", resp.Report.Status)
	})

	t.Run("ApproveReport from draft is an invalid transition, not silently ignored", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		_, err := s.ApproveReport(rapporteCtxWithTenant(tenantID), &rapportev1.ApproveReportRequest{
			ReportId: rep.ID.String(), ReviewerId: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("ApproveReport twice fails the second time with already-approved", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		rep.Status = rapporte.StatusSubmitted
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		reviewerID := uuid.New().String()
		_, err := s.ApproveReport(rapporteCtxWithTenant(tenantID), &rapportev1.ApproveReportRequest{
			ReportId: rep.ID.String(), ReviewerId: reviewerID,
		})
		requireGRPCOK(t, err)

		_, err = s.ApproveReport(rapporteCtxWithTenant(tenantID), &rapportev1.ApproveReportRequest{
			ReportId: rep.ID.String(), ReviewerId: reviewerID,
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("RejectReport invalid reviewer_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.RejectReport(rapporteCtxWithTenant(tenantID), &rapportev1.RejectReportRequest{
			ReportId: uuid.New().String(), ReviewerId: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RejectReport from submitted succeeds", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		rep.Status = rapporte.StatusSubmitted
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		resp, err := s.RejectReport(rapporteCtxWithTenant(tenantID), &rapportev1.RejectReportRequest{
			ReportId: rep.ID.String(), ReviewerId: uuid.New().String(), ReviewNote: "Fehlende Belege",
		})
		requireGRPCOK(t, err)
		require.Equal(t, "rejected", resp.Report.Status)
	})

	t.Run("RejectReport on an already-approved report fails with already-approved, not a generic invalid transition", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		rep.Status = rapporte.StatusApproved
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		_, err := s.RejectReport(rapporteCtxWithTenant(tenantID), &rapportev1.RejectReportRequest{
			ReportId: rep.ID.String(), ReviewerId: uuid.New().String(), ReviewNote: "Fehlende Belege",
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("RejectReport without a reason is rejected as invalid argument", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		rep.Status = rapporte.StatusSubmitted
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		_, err := s.RejectReport(rapporteCtxWithTenant(tenantID), &rapportev1.RejectReportRequest{
			ReportId: rep.ID.String(), ReviewerId: uuid.New().String(), ReviewNote: "   ",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

// ============================================================================
// Line RPCs
// ============================================================================

func TestRapporteLineHandlers(t *testing.T) {
	tenantID := uuid.New()

	t.Run("AddLine invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.AddLine(rapporteCtxWithTenant(tenantID), &rapportev1.AddLineRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("AddLine empty description maps to invalid argument", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		_, err := s.AddLine(rapporteCtxWithTenant(tenantID), &rapportev1.AddLineRequest{ReportId: rep.ID.String(), Description: "  "})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("AddLine on approved report is rejected", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		rep.Status = rapporte.StatusApproved
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		_, err := s.AddLine(rapporteCtxWithTenant(tenantID), &rapportev1.AddLineRequest{ReportId: rep.ID.String(), Description: "Material"})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("AddLine happy path", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		resp, err := s.AddLine(rapporteCtxWithTenant(tenantID), &rapportev1.AddLineRequest{
			ReportId: rep.ID.String(), Description: "Material", Quantity: 3, Unit: "Stk",
		})
		requireGRPCOK(t, err)
		require.Equal(t, "Material", resp.Line.Description)
		require.Equal(t, rep.ID.String(), resp.Line.ReportId)
	})

	t.Run("UpdateLine invalid line_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.UpdateLine(rapporteCtxWithTenant(tenantID), &rapportev1.UpdateLineRequest{LineId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateLine unknown line maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.UpdateLine(rapporteCtxWithTenant(tenantID), &rapportev1.UpdateLineRequest{LineId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("UpdateLine empty description maps to invalid argument", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		line := &rapporte.ReportLine{ID: uuid.New(), TenantID: tenantID, ReportID: rep.ID, Description: "Material"}
		repo.lines[line.ID] = line
		s := newRapporteTestServer(repo)
		empty := "  "
		_, err := s.UpdateLine(rapporteCtxWithTenant(tenantID), &rapportev1.UpdateLineRequest{LineId: line.ID.String(), Description: &empty})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteLine invalid line_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.DeleteLine(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteLineRequest{LineId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteLine unknown line maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.DeleteLine(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteLineRequest{LineId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("ListLines invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.ListLines(rapporteCtxWithTenant(tenantID), &rapportev1.ListLinesRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListLines unknown report maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.ListLines(rapporteCtxWithTenant(tenantID), &rapportev1.ListLinesRequest{ReportId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("ListLines empty result wraps as empty slice not null", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		resp, err := s.ListLines(rapporteCtxWithTenant(tenantID), &rapportev1.ListLinesRequest{ReportId: rep.ID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Lines)
		require.Empty(t, resp.Lines)
	})
}

// ============================================================================
// Attachment RPCs
// ============================================================================

func TestRapporteAttachmentHandlers(t *testing.T) {
	tenantID := uuid.New()

	t.Run("UploadAttachment invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.UploadAttachment(rapporteCtxWithTenant(tenantID), &rapportev1.UploadAttachmentRequest{
			ReportId: "bad", UploadedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UploadAttachment invalid uploaded_by", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.UploadAttachment(rapporteCtxWithTenant(tenantID), &rapportev1.UploadAttachmentRequest{
			ReportId: uuid.New().String(), UploadedBy: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UploadAttachment invalid line_id", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		bad := "bad"
		_, err := s.UploadAttachment(rapporteCtxWithTenant(tenantID), &rapportev1.UploadAttachmentRequest{
			ReportId: rep.ID.String(), UploadedBy: uuid.New().String(), LineId: &bad,
			Filename: "beleg.pdf", ObjectKey: "tenants/" + tenantID.String() + "/rapporte/x",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UploadAttachment object_key outside tenant prefix is rejected", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		_, err := s.UploadAttachment(rapporteCtxWithTenant(tenantID), &rapportev1.UploadAttachmentRequest{
			ReportId: rep.ID.String(), UploadedBy: uuid.New().String(),
			Filename: "beleg.pdf", ObjectKey: "tenants/" + uuid.New().String() + "/rapporte/x",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UploadAttachment happy path", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		resp, err := s.UploadAttachment(rapporteCtxWithTenant(tenantID), &rapportev1.UploadAttachmentRequest{
			ReportId: rep.ID.String(), UploadedBy: uuid.New().String(),
			Filename: "beleg.pdf", ObjectKey: "tenants/" + tenantID.String() + "/rapporte/" + rep.ID.String() + "/beleg.pdf",
		})
		requireGRPCOK(t, err)
		require.Equal(t, "beleg.pdf", resp.Attachment.Filename)
	})

	t.Run("ListAttachments invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.ListAttachments(rapporteCtxWithTenant(tenantID), &rapportev1.ListAttachmentsRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListAttachments invalid line_id filter", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		bad := "bad"
		_, err := s.ListAttachments(rapporteCtxWithTenant(tenantID), &rapportev1.ListAttachmentsRequest{ReportId: rep.ID.String(), LineId: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListAttachments empty result wraps as empty slice not null", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		resp, err := s.ListAttachments(rapporteCtxWithTenant(tenantID), &rapportev1.ListAttachmentsRequest{ReportId: rep.ID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Attachments)
		require.Empty(t, resp.Attachments)
	})

	t.Run("DeleteAttachment invalid attachment_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.DeleteAttachment(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteAttachmentRequest{AttachmentId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteAttachment unknown id maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.DeleteAttachment(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteAttachmentRequest{AttachmentId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// Signature RPC
// ============================================================================

func TestRapporteSignatureHandler(t *testing.T) {
	tenantID := uuid.New()

	t.Run("SaveSignature invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.SaveSignature(rapporteCtxWithTenant(tenantID), &rapportev1.SaveReportSignatureRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("SaveSignature invalid data URL prefix is rejected before hitting the repo", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.SaveSignature(rapporteCtxWithTenant(tenantID), &rapportev1.SaveReportSignatureRequest{
			ReportId: uuid.New().String(), SignatureData: "not-a-data-url", SignedBy: "Max Muster",
		})
		requireGRPCCode(t, err, codes.Internal) // service returns a plain error, mapped via default branch
	})

	t.Run("SaveSignature happy path", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		resp, err := s.SaveSignature(rapporteCtxWithTenant(tenantID), &rapportev1.SaveReportSignatureRequest{
			ReportId: rep.ID.String(), SignatureData: "data:image/png;base64,abc", SignedBy: "Max Muster",
		})
		requireGRPCOK(t, err)
		require.Equal(t, "data:image/png;base64,abc", resp.Report.SignatureData)
		require.Equal(t, "Max Muster", resp.Report.SignedBy)
	})
}

// ============================================================================
// Stats & Export RPCs
// ============================================================================

func TestRapporteStatsAndExportHandlers(t *testing.T) {
	tenantID := uuid.New()

	t.Run("GetReportStats missing tenant is unauthenticated", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.GetReportStats(context.Background(), &rapportev1.GetReportStatsRequest{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("GetReportStats counts by status", func(t *testing.T) {
		repo := newStubRapporteRepo()
		draft := newDraftReport(tenantID)
		submitted := newDraftReport(tenantID)
		submitted.Status = rapporte.StatusSubmitted
		repo.seedReport(draft)
		repo.seedReport(submitted)
		s := newRapporteTestServer(repo)
		resp, err := s.GetReportStats(rapporteCtxWithTenant(tenantID), &rapportev1.GetReportStatsRequest{})
		requireGRPCOK(t, err)
		require.Equal(t, int32(2), resp.TotalReports)
		require.Equal(t, int32(1), resp.DraftCount)
		require.Equal(t, int32(1), resp.SubmittedCount)
	})

	t.Run("ListPendingApprovals invalid author_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		bad := "bad"
		_, err := s.ListPendingApprovals(rapporteCtxWithTenant(tenantID), &rapportev1.ListPendingApprovalsRequest{AuthorId: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListPendingApprovals scoped to author when set restricts the own-scope queue", func(t *testing.T) {
		repo := newStubRapporteRepo()
		mine := newDraftReport(tenantID)
		mine.Status = rapporte.StatusSubmitted
		other := newDraftReport(tenantID)
		other.Status = rapporte.StatusSubmitted
		repo.seedReport(mine)
		repo.seedReport(other)
		s := newRapporteTestServer(repo)
		authorID := mine.AuthorID.String()
		resp, err := s.ListPendingApprovals(rapporteCtxWithTenant(tenantID), &rapportev1.ListPendingApprovalsRequest{AuthorId: &authorID})
		requireGRPCOK(t, err)
		require.Len(t, resp.Reports, 1)
		require.Equal(t, mine.ID.String(), resp.Reports[0].Id)
	})

	t.Run("ExportPDF invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.ExportPDF(rapporteCtxWithTenant(tenantID), &rapportev1.ExportPDFRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ExportPDF unknown report maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.ExportPDF(rapporteCtxWithTenant(tenantID), &rapportev1.ExportPDFRequest{ReportId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("ExportPDF happy path returns a pdf payload and filename", func(t *testing.T) {
		repo := newStubRapporteRepo()
		rep := newDraftReport(tenantID)
		repo.seedReport(rep)
		s := newRapporteTestServer(repo)
		resp, err := s.ExportPDF(rapporteCtxWithTenant(tenantID), &rapportev1.ExportPDFRequest{ReportId: rep.ID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, "application/pdf", resp.ContentType)
		require.NotEmpty(t, resp.Payload)
		require.Contains(t, resp.Filename, rep.ID.String()[:8])
	})
}

// ============================================================================
// Worker RPCs
// ============================================================================

func TestRapporteWorkerHandlers(t *testing.T) {
	tenantID := uuid.New()
	reportID := uuid.New()

	t.Run("AddWorker invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.AddWorker(rapporteCtxWithTenant(tenantID), &rapportev1.AddWorkerRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("AddWorker repo error maps through", func(t *testing.T) {
		repo := newStubRapporteRepo()
		repo.err = rapporte.ErrReportNotFound
		s := newRapporteTestServer(repo)
		_, err := s.AddWorker(rapporteCtxWithTenant(tenantID), &rapportev1.AddWorkerRequest{ReportId: reportID.String(), Name: "Max"})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("AddWorker happy path", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		resp, err := s.AddWorker(rapporteCtxWithTenant(tenantID), &rapportev1.AddWorkerRequest{
			ReportId: reportID.String(), Name: "Max Muster", Role: "Techniker", Hours: 4.5,
		})
		requireGRPCOK(t, err)
		require.Equal(t, "Max Muster", resp.Worker.Name)
		require.Equal(t, 4.5, resp.Worker.Hours)
	})

	t.Run("RemoveWorker invalid worker_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.RemoveWorker(rapporteCtxWithTenant(tenantID), &rapportev1.RemoveWorkerRequest{WorkerId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RemoveWorker unknown id maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.RemoveWorker(rapporteCtxWithTenant(tenantID), &rapportev1.RemoveWorkerRequest{WorkerId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("ListWorkers invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.ListWorkers(rapporteCtxWithTenant(tenantID), &rapportev1.ListWorkersRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListWorkers empty result wraps as empty slice not null", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		resp, err := s.ListWorkers(rapporteCtxWithTenant(tenantID), &rapportev1.ListWorkersRequest{ReportId: reportID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Workers)
		require.Empty(t, resp.Workers)
	})
}

// ============================================================================
// Measurement RPCs
// ============================================================================

func TestRapporteMeasurementHandlers(t *testing.T) {
	tenantID := uuid.New()

	t.Run("CreateMeasurement invalid report_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.CreateMeasurement(rapporteCtxWithTenant(tenantID), &rapportev1.CreateMeasurementRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateMeasurement without report_id is a standalone measurement", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		resp, err := s.CreateMeasurement(rapporteCtxWithTenant(tenantID), &rapportev1.CreateMeasurementRequest{Title: "Aufmass 1"})
		requireGRPCOK(t, err)
		require.Equal(t, "Aufmass 1", resp.Measurement.Title)
		require.Empty(t, resp.Measurement.ReportId)
	})

	t.Run("GetMeasurement invalid measurement_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.GetMeasurement(rapporteCtxWithTenant(tenantID), &rapportev1.GetMeasurementRequest{MeasurementId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetMeasurement unknown id maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.GetMeasurement(rapporteCtxWithTenant(tenantID), &rapportev1.GetMeasurementRequest{MeasurementId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("ListMeasurements invalid report_id filter", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.ListMeasurements(rapporteCtxWithTenant(tenantID), &rapportev1.ListMeasurementsRequest{ReportId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListMeasurements empty result wraps as empty slice not null", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		resp, err := s.ListMeasurements(rapporteCtxWithTenant(tenantID), &rapportev1.ListMeasurementsRequest{})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Measurements)
		require.Empty(t, resp.Measurements)
	})

	t.Run("UpdateMeasurement invalid measurement_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.UpdateMeasurement(rapporteCtxWithTenant(tenantID), &rapportev1.UpdateMeasurementRequest{MeasurementId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateMeasurement unknown id maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.UpdateMeasurement(rapporteCtxWithTenant(tenantID), &rapportev1.UpdateMeasurementRequest{MeasurementId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("DeleteMeasurement invalid measurement_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.DeleteMeasurement(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteMeasurementRequest{MeasurementId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteMeasurement unknown id maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.DeleteMeasurement(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteMeasurementRequest{MeasurementId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("AddMeasurementPosition invalid measurement_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.AddMeasurementPosition(rapporteCtxWithTenant(tenantID), &rapportev1.AddMeasurementPositionRequest{MeasurementId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("AddMeasurementPosition happy path", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		resp, err := s.AddMeasurementPosition(rapporteCtxWithTenant(tenantID), &rapportev1.AddMeasurementPositionRequest{
			MeasurementId: uuid.New().String(), PositionNumber: 1, Description: "Fenster", Unit: "m2", Quantity: 2.5, UnitPrice: 120,
		})
		requireGRPCOK(t, err)
		require.Equal(t, "Fenster", resp.Position.Description)
		require.Equal(t, 2.5, resp.Position.Quantity)
	})

	t.Run("DeleteMeasurementPosition invalid position_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.DeleteMeasurementPosition(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteMeasurementPositionRequest{PositionId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteMeasurementPosition unknown id maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.DeleteMeasurementPosition(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteMeasurementPositionRequest{PositionId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// Template RPCs
// ============================================================================

func TestRapporteTemplateHandlers(t *testing.T) {
	tenantID := uuid.New()

	t.Run("CreateTemplate happy path defaults to active", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		resp, err := s.CreateTemplate(rapporteCtxWithTenant(tenantID), &rapportev1.CreateTemplateRequest{Name: "Standard"})
		requireGRPCOK(t, err)
		require.Equal(t, "Standard", resp.Template.Name)
		require.True(t, resp.Template.IsActive)
	})

	t.Run("CreateTemplate repo error maps through", func(t *testing.T) {
		repo := newStubRapporteRepo()
		repo.err = rapporte.ErrInvalidInput
		s := newRapporteTestServer(repo)
		_, err := s.CreateTemplate(rapporteCtxWithTenant(tenantID), &rapportev1.CreateTemplateRequest{Name: "x"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetTemplate invalid template_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.GetTemplate(rapporteCtxWithTenant(tenantID), &rapportev1.GetTemplateRequest{TemplateId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetTemplate unknown id maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.GetTemplate(rapporteCtxWithTenant(tenantID), &rapportev1.GetTemplateRequest{TemplateId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("ListTemplates empty result wraps as empty slice not null", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		resp, err := s.ListTemplates(rapporteCtxWithTenant(tenantID), &rapportev1.ListTemplatesRequest{})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Templates)
		require.Empty(t, resp.Templates)
	})

	t.Run("ListTemplates active_only filters inactive templates", func(t *testing.T) {
		repo := newStubRapporteRepo()
		s := newRapporteTestServer(repo)
		_, err := s.CreateTemplate(rapporteCtxWithTenant(tenantID), &rapportev1.CreateTemplateRequest{Name: "Aktiv"})
		requireGRPCOK(t, err)

		inactive := &rapporte.ReportTemplate{ID: uuid.New(), TenantID: tenantID, Name: "Inaktiv", IsActive: false, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		repo.templates[inactive.ID] = inactive

		resp, err := s.ListTemplates(rapporteCtxWithTenant(tenantID), &rapportev1.ListTemplatesRequest{ActiveOnly: true})
		requireGRPCOK(t, err)
		require.Len(t, resp.Templates, 1)
		require.Equal(t, "Aktiv", resp.Templates[0].Name)
	})

	t.Run("UpdateTemplate invalid template_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.UpdateTemplate(rapporteCtxWithTenant(tenantID), &rapportev1.UpdateTemplateRequest{TemplateId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateTemplate unknown id maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.UpdateTemplate(rapporteCtxWithTenant(tenantID), &rapportev1.UpdateTemplateRequest{TemplateId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("DeleteTemplate invalid template_id", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.DeleteTemplate(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteTemplateRequest{TemplateId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteTemplate unknown id maps to not found", func(t *testing.T) {
		s := newRapporteTestServer(newStubRapporteRepo())
		_, err := s.DeleteTemplate(rapporteCtxWithTenant(tenantID), &rapportev1.DeleteTemplateRequest{TemplateId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// Error mapping — table test
// ============================================================================

func TestMapRapporteError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"nil", nil, codes.OK},
		{"report not found", rapporte.ErrReportNotFound, codes.NotFound},
		{"line not found", rapporte.ErrLineNotFound, codes.NotFound},
		{"attachment not found", rapporte.ErrAttachmentNotFound, codes.NotFound},
		{"worker not found", rapporte.ErrWorkerNotFound, codes.NotFound},
		{"measurement not found", rapporte.ErrMeasurementNotFound, codes.NotFound},
		{"position not found", rapporte.ErrPositionNotFound, codes.NotFound},
		{"template not found", rapporte.ErrTemplateNotFound, codes.NotFound},
		{"already approved", rapporte.ErrAlreadyApproved, codes.FailedPrecondition},
		{"invalid state transition", rapporte.ErrInvalidStateTransition, codes.FailedPrecondition},
		{"invalid input", rapporte.ErrInvalidInput, codes.InvalidArgument},
		{"unmapped error defaults to internal", errUnmappedRapporte, codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapRapporteError(tc.err)
			if tc.err == nil {
				require.NoError(t, err)
				return
			}
			requireGRPCCode(t, err, tc.code)
		})
	}
}

var errUnmappedRapporte = &rapporteTestOpaqueError{}

type rapporteTestOpaqueError struct{}

func (e *rapporteTestOpaqueError) Error() string { return "opaque backend failure" }
