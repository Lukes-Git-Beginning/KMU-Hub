package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/formulare"
	formularev1 "github.com/kmuhub/kmuhub/proto/formulare/v1"
)

// ============================================================================
// Stub repository (server-package copy — service_test.go is internal)
// ============================================================================

type stubFormulareRepo struct {
	mu          sync.Mutex
	schemas     map[uuid.UUID]*formulare.FormSchema
	submissions map[uuid.UUID]*formulare.FormSubmission
	webhooks    map[uuid.UUID]*formulare.FormWebhook
	deliveries  map[uuid.UUID]*formulare.WebhookDelivery
	shareLinks  map[uuid.UUID]*formulare.FormShareLink

	// Error injection
	getSchemaErr    error
	getWebhookErr   error
	getSubmissionErr error
}

func newStubFormulareRepo() *stubFormulareRepo {
	return &stubFormulareRepo{
		schemas:     make(map[uuid.UUID]*formulare.FormSchema),
		submissions: make(map[uuid.UUID]*formulare.FormSubmission),
		webhooks:    make(map[uuid.UUID]*formulare.FormWebhook),
		deliveries:  make(map[uuid.UUID]*formulare.WebhookDelivery),
	}
}

// --- Schema ---

func (r *stubFormulareRepo) CreateSchema(_ context.Context, s *formulare.FormSchema) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *s
	r.schemas[s.ID] = &cp
	return nil
}

func (r *stubFormulareRepo) GetSchema(_ context.Context, id, tenantID uuid.UUID) (*formulare.FormSchema, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getSchemaErr != nil {
		return nil, r.getSchemaErr
	}
	s, ok := r.schemas[id]
	if !ok || s.TenantID != tenantID {
		return nil, formulare.ErrSchemaNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *stubFormulareRepo) UpdateSchema(_ context.Context, s *formulare.FormSchema) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.schemas[s.ID]
	if !ok || existing.TenantID != s.TenantID {
		return formulare.ErrSchemaNotFound
	}
	cp := *s
	r.schemas[s.ID] = &cp
	return nil
}

func (r *stubFormulareRepo) SoftDeleteSchema(_ context.Context, id, tenantID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.schemas[id]
	if !ok || existing.TenantID != tenantID {
		return formulare.ErrSchemaNotFound
	}
	now := time.Now().UTC()
	existing.DeletedAt = &now
	return nil
}

func (r *stubFormulareRepo) ListSchemas(_ context.Context, tenantID uuid.UUID, filter formulare.ListSchemasFilter) ([]*formulare.FormSchema, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []*formulare.FormSchema
	for _, s := range r.schemas {
		if s.TenantID != tenantID || s.DeletedAt != nil {
			continue
		}
		cp := *s
		all = append(all, &cp)
	}
	total := len(all)
	limit := filter.Limit
	if limit <= 0 || limit > total {
		limit = total
	}
	offset := min(filter.Offset, total)
	end := min(offset+limit, total)
	return all[offset:end], total, nil
}

func (r *stubFormulareRepo) DuplicateSchema(_ context.Context, id, tenantID uuid.UUID, newTitle string) (*formulare.FormSchema, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	original, ok := r.schemas[id]
	if !ok || original.TenantID != tenantID {
		return nil, formulare.ErrSchemaNotFound
	}
	title := newTitle
	if title == "" {
		title = original.Title + " (Copy)"
	}
	now := time.Now().UTC()
	cp := *original
	cp.ID = uuid.New()
	cp.Title = title
	cp.Status = formulare.FormSchemaStatusDraft
	cp.IsPublic = false
	cp.SubmissionCount = 0
	cp.CreatedAt = now
	cp.UpdatedAt = now
	cp.DeletedAt = nil
	r.schemas[cp.ID] = &cp
	result := cp
	return &result, nil
}

// --- Submission ---

func (r *stubFormulareRepo) CreateSubmission(_ context.Context, s *formulare.FormSubmission, _ []uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *s
	r.submissions[s.ID] = &cp
	return nil
}

func (r *stubFormulareRepo) GetSubmission(_ context.Context, id, tenantID uuid.UUID) (*formulare.FormSubmission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getSubmissionErr != nil {
		return nil, r.getSubmissionErr
	}
	s, ok := r.submissions[id]
	if !ok || s.TenantID != tenantID {
		return nil, formulare.ErrSubmissionNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *stubFormulareRepo) ListSubmissions(_ context.Context, tenantID uuid.UUID, filter formulare.ListSubmissionsFilter) ([]*formulare.FormSubmission, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []*formulare.FormSubmission
	for _, s := range r.submissions {
		if s.TenantID != tenantID {
			continue
		}
		if filter.FormSchemaID != nil && (s.FormSchemaID == nil || *s.FormSchemaID != *filter.FormSchemaID) {
			continue
		}
		cp := *s
		all = append(all, &cp)
	}
	total := len(all)
	return all, total, nil
}

func (r *stubFormulareRepo) UpdateSubmissionStatus(_ context.Context, id, tenantID uuid.UUID, status formulare.FormSubmissionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.submissions[id]
	if !ok || s.TenantID != tenantID {
		return formulare.ErrSubmissionNotFound
	}
	s.Status = status
	return nil
}

func (r *stubFormulareRepo) ListSubmissionsForExport(_ context.Context, formSchemaID, tenantID uuid.UUID, _ formulare.ExportFilter) ([]*formulare.FormSubmission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []*formulare.FormSubmission
	for _, s := range r.submissions {
		if s.TenantID != tenantID {
			continue
		}
		if s.FormSchemaID == nil || *s.FormSchemaID != formSchemaID {
			continue
		}
		cp := *s
		all = append(all, &cp)
	}
	return all, nil
}

// --- Webhook ---

func (r *stubFormulareRepo) CreateWebhook(_ context.Context, w *formulare.FormWebhook) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *w
	r.webhooks[w.ID] = &cp
	return nil
}

func (r *stubFormulareRepo) GetWebhook(_ context.Context, id, tenantID uuid.UUID) (*formulare.FormWebhook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getWebhookErr != nil {
		return nil, r.getWebhookErr
	}
	w, ok := r.webhooks[id]
	if !ok || w.TenantID != tenantID {
		return nil, formulare.ErrWebhookNotFound
	}
	cp := *w
	return &cp, nil
}

func (r *stubFormulareRepo) UpdateWebhook(_ context.Context, w *formulare.FormWebhook) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.webhooks[w.ID]
	if !ok || existing.TenantID != w.TenantID {
		return formulare.ErrWebhookNotFound
	}
	cp := *w
	r.webhooks[w.ID] = &cp
	return nil
}

func (r *stubFormulareRepo) DeleteWebhook(_ context.Context, id, tenantID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.webhooks[id]
	if !ok || w.TenantID != tenantID {
		return formulare.ErrWebhookNotFound
	}
	delete(r.webhooks, id)
	return nil
}

func (r *stubFormulareRepo) ListWebhooks(_ context.Context, formSchemaID, tenantID uuid.UUID) ([]*formulare.FormWebhook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*formulare.FormWebhook
	for _, w := range r.webhooks {
		if w.FormSchemaID == formSchemaID && w.TenantID == tenantID {
			cp := *w
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubFormulareRepo) ListActiveWebhooksForSchema(_ context.Context, _ uuid.UUID, formSchemaID uuid.UUID) ([]*formulare.FormWebhook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*formulare.FormWebhook
	for _, w := range r.webhooks {
		if w.FormSchemaID == formSchemaID && w.Active {
			cp := *w
			out = append(out, &cp)
		}
	}
	return out, nil
}

// --- Delivery ---

func (r *stubFormulareRepo) ClaimPendingDeliveries(_ context.Context, batchSize int) ([]*formulare.WebhookDelivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*formulare.WebhookDelivery
	for _, d := range r.deliveries {
		if d.Status == formulare.WebhookDeliveryStatusPending {
			cp := *d
			out = append(out, &cp)
			if len(out) >= batchSize {
				break
			}
		}
	}
	return out, nil
}

func (r *stubFormulareRepo) MarkDeliveryResult(_ context.Context, id uuid.UUID, status formulare.WebhookDeliveryStatus, attemptCount int, _ *time.Time, lastError string, lastCode int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.deliveries[id]
	if !ok {
		return formulare.ErrDeliveryNotFound
	}
	d.Status = status
	d.AttemptCount = attemptCount
	if lastError != "" {
		d.LastError = &lastError
	}
	if lastCode != 0 {
		d.LastResponseCode = &lastCode
	}
	return nil
}

func (r *stubFormulareRepo) ListWebhookDeliveries(_ context.Context, tenantID uuid.UUID, _ formulare.ListDeliveriesFilter) ([]*formulare.WebhookDelivery, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []*formulare.WebhookDelivery
	for _, d := range r.deliveries {
		if d.TenantID != tenantID {
			continue
		}
		cp := *d
		all = append(all, &cp)
	}
	return all, len(all), nil
}

func (r *stubFormulareRepo) GetWebhookDelivery(_ context.Context, id, tenantID uuid.UUID) (*formulare.WebhookDelivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.deliveries[id]
	if !ok || d.TenantID != tenantID {
		return nil, formulare.ErrDeliveryNotFound
	}
	cp := *d
	return &cp, nil
}

func (r *stubFormulareRepo) GetFormStats(_ context.Context, formSchemaID, tenantID uuid.UUID) (*formulare.FormStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var total, newCount, last7d, last30d int
	cutoff7d := time.Now().UTC().Add(-7 * 24 * time.Hour)
	cutoff30d := time.Now().UTC().Add(-30 * 24 * time.Hour)
	for _, s := range r.submissions {
		if s.TenantID != tenantID {
			continue
		}
		if s.FormSchemaID == nil || *s.FormSchemaID != formSchemaID {
			continue
		}
		total++
		if s.Status == formulare.FormSubmissionStatusNew {
			newCount++
		}
		if s.SubmittedAt.After(cutoff7d) {
			last7d++
		}
		if s.SubmittedAt.After(cutoff30d) {
			last30d++
		}
	}
	return &formulare.FormStats{
		FormSchemaID: formSchemaID,
		TotalCount:   total,
		NewCount:     newCount,
		Last7dCount:  last7d,
		Last30dCount: last30d,
	}, nil
}

// ============================================================================
// Test helpers
// ============================================================================

func newTestFormulareServer() *FormulareGRPCServer {
	return NewFormulareGRPCServer(nil)
}

func newFormulareServerWithRepo(repo formulare.Repository) *FormulareGRPCServer {
	svc := formulare.NewService(repo, nil)
	return NewFormulareGRPCServer(svc)
}

// ============================================================================
// Schema Cluster Tests
// ============================================================================

func TestCreateFormSchema_InvalidTenantUUID(t *testing.T) {
	srv := newTestFormulareServer()
	_, err := srv.CreateFormSchema(context.Background(), &formularev1.CreateFormSchemaRequest{
		TenantId: "not-a-uuid",
		Title:    "Test",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateFormSchema_HappyPath(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	resp, err := srv.CreateFormSchema(context.Background(), &formularev1.CreateFormSchemaRequest{
		TenantId: uuid.New().String(),
		Title:    "Kontaktformular",
		Fields:   []byte(`[{"id":"name","type":"text","label":"Name","required":true}]`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Schema == nil {
		t.Fatal("expected schema in response")
	}
	if resp.Schema.Title != "Kontaktformular" {
		t.Errorf("title mismatch: got %s", resp.Schema.Title)
	}
}

func TestGetFormSchema_InvalidID(t *testing.T) {
	srv := newTestFormulareServer()
	_, err := srv.GetFormSchema(context.Background(), &formularev1.GetFormSchemaRequest{
		TenantId: uuid.New().String(),
		SchemaId: "bad-id",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetFormSchema_NotFound(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	_, err := srv.GetFormSchema(context.Background(), &formularev1.GetFormSchemaRequest{
		TenantId: uuid.New().String(),
		SchemaId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestGetFormSchema_HappyPath(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	// Create first
	createResp, err := srv.CreateFormSchema(context.Background(), &formularev1.CreateFormSchemaRequest{
		TenantId: tenantID.String(),
		Title:    "My Form",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	getResp, err := srv.GetFormSchema(context.Background(), &formularev1.GetFormSchemaRequest{
		TenantId: tenantID.String(),
		SchemaId: createResp.Schema.Id,
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if getResp.Schema.Id != createResp.Schema.Id {
		t.Errorf("id mismatch")
	}
}

func TestUpdateFormSchema_PartialUpdate(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	createResp, err := srv.CreateFormSchema(context.Background(), &formularev1.CreateFormSchemaRequest{
		TenantId: tenantID.String(),
		Title:    "Original",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newTitle := "Updated"
	updateResp, err := srv.UpdateFormSchema(context.Background(), &formularev1.UpdateFormSchemaRequest{
		TenantId: tenantID.String(),
		SchemaId: createResp.Schema.Id,
		Title:    &newTitle,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updateResp.Schema.Title != "Updated" {
		t.Errorf("title not updated: got %s", updateResp.Schema.Title)
	}
}

func TestUpdateFormSchema_NotFound(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	newTitle := "Nope"
	_, err := srv.UpdateFormSchema(context.Background(), &formularev1.UpdateFormSchemaRequest{
		TenantId: uuid.New().String(),
		SchemaId: uuid.New().String(),
		Title:    &newTitle,
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestDeleteFormSchema_ReturnsEmptyResponse(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	createResp, err := srv.CreateFormSchema(context.Background(), &formularev1.CreateFormSchemaRequest{
		TenantId: tenantID.String(),
		Title:    "To Delete",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	resp, err := srv.DeleteFormSchema(context.Background(), &formularev1.DeleteFormSchemaRequest{
		TenantId: tenantID.String(),
		SchemaId: createResp.Schema.Id,
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil empty response")
	}
}

func TestListFormSchemas_HappyPath(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	for range 3 {
		_, err := srv.CreateFormSchema(context.Background(), &formularev1.CreateFormSchemaRequest{
			TenantId: tenantID.String(),
			Title:    "Form",
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	resp, err := srv.ListFormSchemas(context.Background(), &formularev1.ListFormSchemasRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
	if len(resp.Schemas) != 3 {
		t.Errorf("expected 3 schemas, got %d", len(resp.Schemas))
	}
}

func TestDuplicateFormSchema_HappyPath(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	createResp, err := srv.CreateFormSchema(context.Background(), &formularev1.CreateFormSchemaRequest{
		TenantId: tenantID.String(),
		Title:    "Original Form",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newTitle := "Copy of Form"
	dupResp, err := srv.DuplicateFormSchema(context.Background(), &formularev1.DuplicateFormSchemaRequest{
		TenantId: tenantID.String(),
		SchemaId: createResp.Schema.Id,
		NewTitle: &newTitle,
	})
	if err != nil {
		t.Fatalf("duplicate: %v", err)
	}
	if dupResp.Schema.Id == createResp.Schema.Id {
		t.Error("duplicate should have a new ID")
	}
	if dupResp.Schema.Title != "Copy of Form" {
		t.Errorf("title mismatch: got %s", dupResp.Schema.Title)
	}
}

// ============================================================================
// Submission Cluster Tests
// ============================================================================

func TestCreateSubmission_HappyPath(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	schemaID := uuid.New()

	resp, err := srv.CreateSubmission(context.Background(), &formularev1.CreateSubmissionRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID.String(),
		Answers:      []byte(`{"name":"Max Mustermann"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Submission == nil {
		t.Fatal("expected submission in response")
	}
}

func TestCreateSubmission_InvalidFormSchemaUUID(t *testing.T) {
	srv := newTestFormulareServer()
	_, err := srv.CreateSubmission(context.Background(), &formularev1.CreateSubmissionRequest{
		TenantId:     uuid.New().String(),
		FormSchemaId: "bad-uuid",
		Answers:      []byte(`{"x":"y"}`),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetSubmission_NotFound(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	_, err := srv.GetSubmission(context.Background(), &formularev1.GetSubmissionRequest{
		TenantId:     uuid.New().String(),
		SubmissionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestListSubmissions_HappyPath(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	schemaID := uuid.New()

	for range 2 {
		_, err := srv.CreateSubmission(context.Background(), &formularev1.CreateSubmissionRequest{
			TenantId:     tenantID.String(),
			FormSchemaId: schemaID.String(),
			Answers:      []byte(`{"q":"a"}`),
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	schemaIDStr := schemaID.String()
	resp, err := srv.ListSubmissions(context.Background(), &formularev1.ListSubmissionsRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: &schemaIDStr,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Total)
	}
}

func TestUpdateSubmissionStatus_HappyPath(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	createResp, err := srv.CreateSubmission(context.Background(), &formularev1.CreateSubmissionRequest{
		TenantId: tenantID.String(),
		Answers:  []byte(`{"q":"a"}`),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	resp, err := srv.UpdateSubmissionStatus(context.Background(), &formularev1.UpdateSubmissionStatusRequest{
		TenantId:     tenantID.String(),
		SubmissionId: createResp.Submission.Id,
		Status:       formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_READ,
	})
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if resp.Submission.Status != formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_READ {
		t.Errorf("status not updated: got %v", resp.Submission.Status)
	}
}

func TestExportSubmissions_CSV(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	schemaID := uuid.New()

	// Pre-populate submission directly in repo since we need matching schema
	sub := &formulare.FormSubmission{
		ID:           uuid.New(),
		TenantID:     tenantID,
		FormSchemaID: &schemaID,
		Answers:      []byte(`{"name":"Test"}`),
		Status:       formulare.FormSubmissionStatusNew,
		SubmittedAt:  time.Now().UTC(),
	}
	repo.mu.Lock()
	repo.submissions[sub.ID] = sub
	repo.mu.Unlock()

	resp, err := srv.ExportSubmissions(context.Background(), &formularev1.ExportSubmissionsRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID.String(),
		Format:       formularev1.ExportFormat_EXPORT_FORMAT_CSV,
	})
	if err != nil {
		t.Fatalf("export CSV: %v", err)
	}
	if len(resp.Content) == 0 {
		t.Error("expected non-empty CSV content")
	}
	if resp.MimeType != "text/csv" {
		t.Errorf("mime type mismatch: got %s", resp.MimeType)
	}
}

func TestExportSubmissions_XLSX(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	schemaID := uuid.New()

	sub := &formulare.FormSubmission{
		ID:           uuid.New(),
		TenantID:     tenantID,
		FormSchemaID: &schemaID,
		Answers:      []byte(`{"city":"Berlin"}`),
		Status:       formulare.FormSubmissionStatusNew,
		SubmittedAt:  time.Now().UTC(),
	}
	repo.mu.Lock()
	repo.submissions[sub.ID] = sub
	repo.mu.Unlock()

	resp, err := srv.ExportSubmissions(context.Background(), &formularev1.ExportSubmissionsRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID.String(),
		Format:       formularev1.ExportFormat_EXPORT_FORMAT_XLSX,
	})
	if err != nil {
		t.Fatalf("export XLSX: %v", err)
	}
	if len(resp.Content) == 0 {
		t.Error("expected non-empty XLSX content")
	}
	if resp.MimeType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("mime type mismatch: got %s", resp.MimeType)
	}
}

// ============================================================================
// Webhook Cluster Tests
// ============================================================================

func TestCreateWebhook_HappyPath(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	schemaID := uuid.New()

	// Put schema in repo so CreateWebhook can verify it exists
	repo.mu.Lock()
	repo.schemas[schemaID] = &formulare.FormSchema{
		ID:       schemaID,
		TenantID: tenantID,
		Title:    "Test",
		Fields:   []byte("[]"),
	}
	repo.mu.Unlock()

	resp, err := srv.CreateWebhook(context.Background(), &formularev1.CreateWebhookRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID.String(),
		Url:          "https://example.com/webhook",
		Active:       true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Webhook == nil {
		t.Fatal("expected webhook in response")
	}
	if resp.Webhook.Url != "https://example.com/webhook" {
		t.Errorf("url mismatch: got %s", resp.Webhook.Url)
	}
}

func TestCreateWebhook_InvalidURL(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	schemaID := uuid.New()

	repo.mu.Lock()
	repo.schemas[schemaID] = &formulare.FormSchema{
		ID:       schemaID,
		TenantID: tenantID,
		Title:    "Test",
		Fields:   []byte("[]"),
	}
	repo.mu.Unlock()

	_, err := srv.CreateWebhook(context.Background(), &formularev1.CreateWebhookRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID.String(),
		Url:          "not-a-valid-url",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetWebhook_SecretIsMasked(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	schemaID := uuid.New()

	repo.mu.Lock()
	repo.schemas[schemaID] = &formulare.FormSchema{
		ID:       schemaID,
		TenantID: tenantID,
		Title:    "Test",
		Fields:   []byte("[]"),
	}
	repo.mu.Unlock()

	secret := "mysupersecret123"
	createResp, err := srv.CreateWebhook(context.Background(), &formularev1.CreateWebhookRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID.String(),
		Url:          "https://example.com/wh",
		Secret:       &secret,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	getResp, err := srv.GetWebhook(context.Background(), &formularev1.GetWebhookRequest{
		TenantId:  tenantID.String(),
		WebhookId: createResp.Webhook.Id,
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if getResp.Webhook.Secret == secret {
		t.Error("secret should be masked, not returned in plaintext")
	}
	if getResp.Webhook.Secret == "" {
		t.Error("secret should be present (masked), not empty")
	}
}

func TestUpdateWebhook_PartialUpdate(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	schemaID := uuid.New()

	repo.mu.Lock()
	repo.schemas[schemaID] = &formulare.FormSchema{
		ID:       schemaID,
		TenantID: tenantID,
		Title:    "Test",
		Fields:   []byte("[]"),
	}
	repo.mu.Unlock()

	createResp, err := srv.CreateWebhook(context.Background(), &formularev1.CreateWebhookRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID.String(),
		Url:          "https://example.com/original",
		Active:       true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newURL := "https://example.com/updated"
	active := false
	resp, err := srv.UpdateWebhook(context.Background(), &formularev1.UpdateWebhookRequest{
		TenantId:  tenantID.String(),
		WebhookId: createResp.Webhook.Id,
		Url:       &newURL,
		Active:    &active,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if resp.Webhook.Url != newURL {
		t.Errorf("url not updated: got %s", resp.Webhook.Url)
	}
	if resp.Webhook.Active {
		t.Error("webhook should be inactive")
	}
}

func TestDeleteWebhook_ReturnsEmptyResponse(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	schemaID := uuid.New()

	repo.mu.Lock()
	repo.schemas[schemaID] = &formulare.FormSchema{
		ID:       schemaID,
		TenantID: tenantID,
		Title:    "Test",
		Fields:   []byte("[]"),
	}
	repo.mu.Unlock()

	createResp, err := srv.CreateWebhook(context.Background(), &formularev1.CreateWebhookRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID.String(),
		Url:          "https://example.com/wh",
		Active:       true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	resp, err := srv.DeleteWebhook(context.Background(), &formularev1.DeleteWebhookRequest{
		TenantId:  tenantID.String(),
		WebhookId: createResp.Webhook.Id,
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil empty response")
	}
}

func TestListWebhooks_AllSecretsMasked(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	schemaID := uuid.New()

	repo.mu.Lock()
	repo.schemas[schemaID] = &formulare.FormSchema{
		ID:       schemaID,
		TenantID: tenantID,
		Title:    "Test",
		Fields:   []byte("[]"),
	}
	repo.mu.Unlock()

	plainSecret := "plaintextsecret"
	_, err := srv.CreateWebhook(context.Background(), &formularev1.CreateWebhookRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID.String(),
		Url:          "https://example.com/wh",
		Secret:       &plainSecret,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	resp, err := srv.ListWebhooks(context.Background(), &formularev1.ListWebhooksRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID.String(),
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, w := range resp.Webhooks {
		if w.Secret == plainSecret {
			t.Error("secret exposed in list response")
		}
	}
}

// ============================================================================
// Delivery Cluster Tests
// ============================================================================

func TestListWebhookDeliveries_HappyPath(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	webhookID := uuid.New()
	submissionID := uuid.New()

	d := &formulare.WebhookDelivery{
		ID:            uuid.New(),
		WebhookID:     webhookID,
		SubmissionID:  submissionID,
		TenantID:      tenantID,
		Payload:       []byte(`{}`),
		Status:        formulare.WebhookDeliveryStatusDelivered,
		AttemptCount:  1,
		MaxAttempts:   5,
		NextAttemptAt: time.Time{},
		CreatedAt:     time.Now().UTC(),
	}
	repo.mu.Lock()
	repo.deliveries[d.ID] = d
	repo.mu.Unlock()

	resp, err := srv.ListWebhookDeliveries(context.Background(), &formularev1.ListWebhookDeliveriesRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected total=1, got %d", resp.Total)
	}
	if resp.Deliveries[0].Status != formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_DELIVERED {
		t.Errorf("status mismatch")
	}
}

// ============================================================================
// Stats Tests
// ============================================================================

func TestGetFormStats_HappyPath(t *testing.T) {
	repo := newStubFormulareRepo()
	srv := newFormulareServerWithRepo(repo)

	tenantID := uuid.New()
	schemaID := uuid.New()

	// Seed 3 submissions
	for range 3 {
		sub := &formulare.FormSubmission{
			ID:           uuid.New(),
			TenantID:     tenantID,
			FormSchemaID: &schemaID,
			Answers:      []byte(`{"q":"a"}`),
			Status:       formulare.FormSubmissionStatusNew,
			SubmittedAt:  time.Now().UTC(),
		}
		repo.mu.Lock()
		repo.submissions[sub.ID] = sub
		repo.mu.Unlock()
	}

	resp, err := srv.GetFormStats(context.Background(), &formularev1.GetFormStatsRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Stats == nil {
		t.Fatal("expected stats in response")
	}
	if resp.Stats.TotalCount != 3 {
		t.Errorf("expected TotalCount=3, got %d", resp.Stats.TotalCount)
	}
	if resp.Stats.NewCount != 3 {
		t.Errorf("expected NewCount=3, got %d", resp.Stats.NewCount)
	}
	if resp.Stats.Last_7DCount != 3 {
		t.Errorf("expected Last_7DCount=3, got %d", resp.Stats.Last_7DCount)
	}
	if resp.Stats.Last_30DCount != 3 {
		t.Errorf("expected Last_30DCount=3, got %d", resp.Stats.Last_30DCount)
	}
}

func TestGetFormStats_InvalidTenantID(t *testing.T) {
	srv := newTestFormulareServer()
	_, err := srv.GetFormStats(context.Background(), &formularev1.GetFormStatsRequest{
		TenantId:     "not-a-uuid",
		FormSchemaId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetFormStats_InvalidFormSchemaID(t *testing.T) {
	srv := newTestFormulareServer()
	_, err := srv.GetFormStats(context.Background(), &formularev1.GetFormStatsRequest{
		TenantId:     uuid.New().String(),
		FormSchemaId: "bad-uuid",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Error mapping tests
// ============================================================================

func TestFormulareMapError_Nil(t *testing.T) {
	if got := mapFormulareError(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestFormulareMapError_SchemaNotFound(t *testing.T) {
	assertGRPCCode(t, mapFormulareError(formulare.ErrSchemaNotFound), codes.NotFound)
}

func TestFormulareMapError_SubmissionNotFound(t *testing.T) {
	assertGRPCCode(t, mapFormulareError(formulare.ErrSubmissionNotFound), codes.NotFound)
}

func TestFormulareMapError_WebhookNotFound(t *testing.T) {
	assertGRPCCode(t, mapFormulareError(formulare.ErrWebhookNotFound), codes.NotFound)
}

func TestFormulareMapError_DeliveryNotFound(t *testing.T) {
	assertGRPCCode(t, mapFormulareError(formulare.ErrDeliveryNotFound), codes.NotFound)
}

func TestFormulareMapError_InvalidFields(t *testing.T) {
	assertGRPCCode(t, mapFormulareError(formulare.ErrInvalidFields), codes.InvalidArgument)
}

func TestFormulareMapError_InvalidURL(t *testing.T) {
	assertGRPCCode(t, mapFormulareError(formulare.ErrInvalidURL), codes.InvalidArgument)
}

func TestFormulareMapError_InvalidExportFormat(t *testing.T) {
	assertGRPCCode(t, mapFormulareError(formulare.ErrInvalidExportFormat), codes.InvalidArgument)
}

func TestFormulareMapError_WebhookInactive(t *testing.T) {
	assertGRPCCode(t, mapFormulareError(formulare.ErrWebhookInactive), codes.FailedPrecondition)
}

func TestFormulareMapError_Unknown(t *testing.T) {
	assertGRPCCode(t, mapFormulareError(formulare.ErrSchemaDeleted), codes.NotFound)
}

// ============================================================================
// Conversion helper tests
// ============================================================================

func TestToProtoFormSchema_Nil(t *testing.T) {
	if got := toProtoFormSchema(nil); got != nil {
		t.Fatalf("expected nil for nil input, got %v", got)
	}
}

func TestToProtoFormSchema_Fields(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	createdBy := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	s := &formulare.FormSchema{
		ID:          id,
		TenantID:    tenantID,
		Title:       "My Form",
		Description: "desc",
		Status:      formulare.FormSchemaStatusActive,
		CreatedBy:   &createdBy,
		IsTemplate:  true,
		IsPublic:    false,
		PageCount:   2,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	p := toProtoFormSchema(s)
	if p.Id != id.String() {
		t.Errorf("id mismatch")
	}
	if p.Status != formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ACTIVE {
		t.Errorf("status mismatch: got %v", p.Status)
	}
	if p.CreatedBy != createdBy.String() {
		t.Errorf("created_by mismatch")
	}
	if !p.IsTemplate {
		t.Error("is_template should be true")
	}
}

func TestToProtoFormSubmission_Nil(t *testing.T) {
	if got := toProtoFormSubmission(nil); got != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestToProtoFormWebhook_Nil(t *testing.T) {
	if got := toProtoFormWebhook(nil); got != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestToProtoWebhookDelivery_Nil(t *testing.T) {
	if got := toProtoWebhookDelivery(nil); got != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestStatusRoundTrip_FormSchema(t *testing.T) {
	cases := []struct {
		domain formulare.FormSchemaStatus
		proto  formularev1.FormSchemaStatus
	}{
		{formulare.FormSchemaStatusDraft, formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_DRAFT},
		{formulare.FormSchemaStatusActive, formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ACTIVE},
		{formulare.FormSchemaStatusArchived, formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ARCHIVED},
	}
	for _, tc := range cases {
		p := toProtoFormSchemaStatus(tc.domain)
		if p != tc.proto {
			t.Errorf("toProto(%v) = %v, want %v", tc.domain, p, tc.proto)
		}
		d := fromProtoFormSchemaStatus(tc.proto)
		if d != tc.domain {
			t.Errorf("fromProto(%v) = %v, want %v", tc.proto, d, tc.domain)
		}
	}
}

func TestStatusRoundTrip_Submission(t *testing.T) {
	cases := []struct {
		domain formulare.FormSubmissionStatus
		proto  formularev1.FormSubmissionStatus
	}{
		{formulare.FormSubmissionStatusNew, formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_NEW},
		{formulare.FormSubmissionStatusRead, formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_READ},
		{formulare.FormSubmissionStatusArchived, formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_ARCHIVED},
	}
	for _, tc := range cases {
		p := toProtoSubmissionStatus(tc.domain)
		if p != tc.proto {
			t.Errorf("toProto(%v) = %v, want %v", tc.domain, p, tc.proto)
		}
		d := fromProtoSubmissionStatus(tc.proto)
		if d != tc.domain {
			t.Errorf("fromProto(%v) = %v, want %v", tc.proto, d, tc.domain)
		}
	}
}

func TestStatusRoundTrip_Delivery(t *testing.T) {
	cases := []struct {
		domain formulare.WebhookDeliveryStatus
		proto  formularev1.WebhookDeliveryStatus
	}{
		{formulare.WebhookDeliveryStatusPending, formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_PENDING},
		{formulare.WebhookDeliveryStatusDelivered, formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_DELIVERED},
		{formulare.WebhookDeliveryStatusFailed, formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_FAILED},
		{formulare.WebhookDeliveryStatusDead, formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_DEAD},
	}
	for _, tc := range cases {
		p := toProtoDeliveryStatus(tc.domain)
		if p != tc.proto {
			t.Errorf("toProto(%v) = %v, want %v", tc.domain, p, tc.proto)
		}
		d := fromProtoDeliveryStatus(tc.proto)
		if d != tc.domain {
			t.Errorf("fromProto(%v) = %v, want %v", tc.proto, d, tc.domain)
		}
	}
}

// --- Share links (B8) ---

func (r *stubFormulareRepo) CreateShareLink(_ context.Context, link *formulare.FormShareLink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.shareLinks == nil {
		r.shareLinks = map[uuid.UUID]*formulare.FormShareLink{}
	}
	cp := *link
	r.shareLinks[link.ID] = &cp
	return nil
}

func (r *stubFormulareRepo) ListShareLinks(_ context.Context, formSchemaID, tenantID uuid.UUID) ([]*formulare.FormShareLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*formulare.FormShareLink, 0)
	for _, l := range r.shareLinks {
		if l.FormSchemaID == formSchemaID && l.TenantID == tenantID {
			cp := *l
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubFormulareRepo) RevokeShareLink(_ context.Context, id, tenantID uuid.UUID, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.shareLinks[id]
	if !ok || l.TenantID != tenantID {
		return formulare.ErrShareLinkNotFound
	}
	if l.RevokedAt == nil {
		l.RevokedAt = &now
	}
	return nil
}

func (r *stubFormulareRepo) GetShareLinkByToken(_ context.Context, token string) (*formulare.FormShareLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.shareLinks {
		if l.Token == token {
			cp := *l
			return &cp, nil
		}
	}
	return nil, formulare.ErrShareLinkNotFound
}

func (r *stubFormulareRepo) RedeemShareLinkTx(
	_ context.Context,
	linkID, tenantID uuid.UUID,
	submission *formulare.FormSubmission,
	_ []uuid.UUID,
	now time.Time,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.shareLinks[linkID]
	if !ok || l.TenantID != tenantID {
		return false, nil
	}
	if l.RevokedAt != nil {
		return false, nil
	}
	if l.ExpiresAt != nil && !now.Before(*l.ExpiresAt) {
		return false, nil
	}
	if l.MaxSubmissions != nil && l.SubmissionCount >= *l.MaxSubmissions {
		return false, nil
	}
	l.SubmissionCount++
	cp := *submission
	r.submissions[submission.ID] = &cp
	return true, nil
}
