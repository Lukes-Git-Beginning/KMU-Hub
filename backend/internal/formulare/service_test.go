package formulare

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

// ============================================================================
// Fixtures
// ============================================================================

var testTenant = uuid.MustParse("11111111-1111-1111-1111-111111111111")

// ============================================================================
// Stub repository
// ============================================================================

type stubRepo struct {
	mu          sync.Mutex
	schemas     map[uuid.UUID]*FormSchema
	submissions map[uuid.UUID]*FormSubmission
	webhooks    map[uuid.UUID]*FormWebhook
	deliveries  map[uuid.UUID]*WebhookDelivery
	shareLinks  map[uuid.UUID]*FormShareLink

	// Error injection
	createSchemaErr    error
	getSchemaErr       error
	updateSchemaErr    error
	deleteSchemaErr    error
	createSubmissionErr error
	getSubmissionErr   error
	createWebhookErr   error
	getWebhookErr      error
	updateWebhookErr   error
	deleteWebhookErr   error
	createShareLinkErr error
	redeemShareLinkErr error

	// Track what was passed to CreateSubmission
	lastWebhookIDs []uuid.UUID
}

func newStubRepo() *stubRepo {
	return &stubRepo{
		schemas:     make(map[uuid.UUID]*FormSchema),
		submissions: make(map[uuid.UUID]*FormSubmission),
		webhooks:    make(map[uuid.UUID]*FormWebhook),
		deliveries:  make(map[uuid.UUID]*WebhookDelivery),
	}
}

// --- Schema ---

func (r *stubRepo) CreateSchema(_ context.Context, s *FormSchema) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createSchemaErr != nil {
		return r.createSchemaErr
	}
	cp := *s
	r.schemas[s.ID] = &cp
	return nil
}

func (r *stubRepo) GetSchema(_ context.Context, id, tenantID uuid.UUID) (*FormSchema, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getSchemaErr != nil {
		return nil, r.getSchemaErr
	}
	s, ok := r.schemas[id]
	if !ok || s.TenantID != tenantID {
		return nil, ErrSchemaNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *stubRepo) UpdateSchema(_ context.Context, s *FormSchema) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateSchemaErr != nil {
		return r.updateSchemaErr
	}
	existing, ok := r.schemas[s.ID]
	if !ok || existing.TenantID != s.TenantID {
		return ErrSchemaNotFound
	}
	cp := *s
	r.schemas[s.ID] = &cp
	return nil
}

func (r *stubRepo) SoftDeleteSchema(_ context.Context, id, tenantID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.deleteSchemaErr != nil {
		return r.deleteSchemaErr
	}
	existing, ok := r.schemas[id]
	if !ok || existing.TenantID != tenantID {
		return ErrSchemaNotFound
	}
	now := time.Now().UTC()
	existing.DeletedAt = &now
	return nil
}

func (r *stubRepo) ListSchemas(_ context.Context, tenantID uuid.UUID, filter ListSchemasFilter) ([]*FormSchema, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []*FormSchema
	for _, s := range r.schemas {
		if s.TenantID != tenantID || s.DeletedAt != nil {
			continue
		}
		if filter.Status != nil && s.Status != *filter.Status {
			continue
		}
		if filter.IsTemplate != nil && s.IsTemplate != *filter.IsTemplate {
			continue
		}
		cp := *s
		all = append(all, &cp)
	}
	total := len(all)
	limit := filter.Limit
	if limit <= 0 {
		limit = total
	}
	offset := min(filter.Offset, total)
	end := min(offset+limit, total)
	return all[offset:end], total, nil
}

func (r *stubRepo) DuplicateSchema(_ context.Context, id, tenantID uuid.UUID, newTitle string) (*FormSchema, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	original, ok := r.schemas[id]
	if !ok || original.TenantID != tenantID {
		return nil, ErrSchemaNotFound
	}
	title := newTitle
	if title == "" {
		title = original.Title + " (Copy)"
	}
	now := time.Now().UTC()
	cp := *original
	cp.ID = uuid.New()
	cp.Title = title
	cp.Status = FormSchemaStatusDraft
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

func (r *stubRepo) CreateSubmission(_ context.Context, s *FormSubmission, webhookIDs []uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createSubmissionErr != nil {
		return r.createSubmissionErr
	}
	cp := *s
	r.submissions[s.ID] = &cp
	r.lastWebhookIDs = webhookIDs
	return nil
}

func (r *stubRepo) GetSubmission(_ context.Context, id, tenantID uuid.UUID) (*FormSubmission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getSubmissionErr != nil {
		return nil, r.getSubmissionErr
	}
	s, ok := r.submissions[id]
	if !ok || s.TenantID != tenantID {
		return nil, ErrSubmissionNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *stubRepo) ListSubmissions(_ context.Context, tenantID uuid.UUID, filter ListSubmissionsFilter) ([]*FormSubmission, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []*FormSubmission
	for _, s := range r.submissions {
		if s.TenantID != tenantID {
			continue
		}
		if filter.FormSchemaID != nil && (s.FormSchemaID == nil || *s.FormSchemaID != *filter.FormSchemaID) {
			continue
		}
		if filter.Status != nil && s.Status != *filter.Status {
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

func (r *stubRepo) UpdateSubmissionStatus(_ context.Context, id, tenantID uuid.UUID, status FormSubmissionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.submissions[id]
	if !ok || s.TenantID != tenantID {
		return ErrSubmissionNotFound
	}
	s.Status = status
	return nil
}

func (r *stubRepo) ListSubmissionsForExport(_ context.Context, formSchemaID, tenantID uuid.UUID, _ ExportFilter) ([]*FormSubmission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []*FormSubmission
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

func (r *stubRepo) CreateWebhook(_ context.Context, w *FormWebhook) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createWebhookErr != nil {
		return r.createWebhookErr
	}
	cp := *w
	r.webhooks[w.ID] = &cp
	return nil
}

func (r *stubRepo) GetWebhook(_ context.Context, id, tenantID uuid.UUID) (*FormWebhook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getWebhookErr != nil {
		return nil, r.getWebhookErr
	}
	w, ok := r.webhooks[id]
	if !ok || w.TenantID != tenantID {
		return nil, ErrWebhookNotFound
	}
	cp := *w
	return &cp, nil
}

func (r *stubRepo) UpdateWebhook(_ context.Context, w *FormWebhook) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateWebhookErr != nil {
		return r.updateWebhookErr
	}
	existing, ok := r.webhooks[w.ID]
	if !ok || existing.TenantID != w.TenantID {
		return ErrWebhookNotFound
	}
	cp := *w
	r.webhooks[w.ID] = &cp
	return nil
}

func (r *stubRepo) DeleteWebhook(_ context.Context, id, tenantID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.deleteWebhookErr != nil {
		return r.deleteWebhookErr
	}
	w, ok := r.webhooks[id]
	if !ok || w.TenantID != tenantID {
		return ErrWebhookNotFound
	}
	delete(r.webhooks, id)
	return nil
}

func (r *stubRepo) ListWebhooks(_ context.Context, formSchemaID, tenantID uuid.UUID) ([]*FormWebhook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*FormWebhook
	for _, w := range r.webhooks {
		if w.FormSchemaID == formSchemaID && w.TenantID == tenantID {
			cp := *w
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubRepo) ListActiveWebhooksForSchema(_ context.Context, _ uuid.UUID, formSchemaID uuid.UUID) ([]*FormWebhook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*FormWebhook
	for _, w := range r.webhooks {
		if w.FormSchemaID == formSchemaID && w.Active {
			cp := *w
			out = append(out, &cp)
		}
	}
	return out, nil
}

// --- Delivery ---

func (r *stubRepo) ClaimPendingDeliveries(_ context.Context, batchSize int) ([]*WebhookDelivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*WebhookDelivery
	for _, d := range r.deliveries {
		if d.Status == WebhookDeliveryStatusPending {
			cp := *d
			out = append(out, &cp)
			if len(out) >= batchSize {
				break
			}
		}
	}
	return out, nil
}

func (r *stubRepo) MarkDeliveryResult(_ context.Context, id uuid.UUID, status WebhookDeliveryStatus, attemptCount int, _ *time.Time, lastError string, lastCode int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.deliveries[id]
	if !ok {
		return ErrDeliveryNotFound
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

func (r *stubRepo) ListWebhookDeliveries(_ context.Context, tenantID uuid.UUID, filter ListDeliveriesFilter) ([]*WebhookDelivery, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []*WebhookDelivery
	for _, d := range r.deliveries {
		if d.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && d.Status != *filter.Status {
			continue
		}
		cp := *d
		all = append(all, &cp)
	}
	total := len(all)
	return all, total, nil
}

func (r *stubRepo) GetWebhookDelivery(_ context.Context, id, tenantID uuid.UUID) (*WebhookDelivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.deliveries[id]
	if !ok || d.TenantID != tenantID {
		return nil, ErrDeliveryNotFound
	}
	cp := *d
	return &cp, nil
}

func (r *stubRepo) GetFormStats(_ context.Context, formSchemaID, tenantID uuid.UUID) (*FormStats, error) {
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
		if s.Status == FormSubmissionStatusNew {
			newCount++
		}
		if s.SubmittedAt.After(cutoff7d) {
			last7d++
		}
		if s.SubmittedAt.After(cutoff30d) {
			last30d++
		}
	}
	return &FormStats{
		FormSchemaID: formSchemaID,
		TotalCount:   total,
		NewCount:     newCount,
		Last7dCount:  last7d,
		Last30dCount: last30d,
	}, nil
}

// --- Share links ---

func (r *stubRepo) ensureShareLinks() {
	if r.shareLinks == nil {
		r.shareLinks = map[uuid.UUID]*FormShareLink{}
	}
}

func (r *stubRepo) CreateShareLink(_ context.Context, link *FormShareLink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createShareLinkErr != nil {
		return r.createShareLinkErr
	}
	r.ensureShareLinks()
	cp := *link
	r.shareLinks[link.ID] = &cp
	return nil
}

func (r *stubRepo) ListShareLinks(_ context.Context, formSchemaID, tenantID uuid.UUID) ([]*FormShareLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*FormShareLink, 0)
	for _, l := range r.shareLinks {
		if l.FormSchemaID == formSchemaID && l.TenantID == tenantID {
			cp := *l
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (r *stubRepo) RevokeShareLink(_ context.Context, id, tenantID uuid.UUID, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.shareLinks[id]
	if !ok || l.TenantID != tenantID {
		return ErrShareLinkNotFound
	}
	if l.RevokedAt == nil {
		l.RevokedAt = &now
	}
	return nil
}

func (r *stubRepo) GetShareLinkByToken(_ context.Context, token string) (*FormShareLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.shareLinks {
		if l.Token == token {
			cp := *l
			return &cp, nil
		}
	}
	return nil, ErrShareLinkNotFound
}

func (r *stubRepo) RedeemShareLinkTx(
	_ context.Context,
	linkID, tenantID uuid.UUID,
	submission *FormSubmission,
	webhookIDs []uuid.UUID,
	now time.Time,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.redeemShareLinkErr != nil {
		return false, r.redeemShareLinkErr
	}
	l, ok := r.shareLinks[linkID]
	if !ok || l.TenantID != tenantID {
		return false, nil
	}
	// Mirrors the SQL guard: live, unexpired, under quota.
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
	r.lastWebhookIDs = webhookIDs
	return true, nil
}

// compile-time interface check
var _ Repository = (*stubRepo)(nil)

// ============================================================================
// Helpers
// ============================================================================

func newSvc() (*Service, *stubRepo) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	return svc, repo
}

func mustCreateSchema(t *testing.T, svc *Service) *FormSchema {
	t.Helper()
	schema, err := svc.CreateFormSchema(context.Background(), CreateSchemaInput{
		TenantID: testTenant,
		Title:    "Contact Form",
		Fields:   []byte(`[{"id":"name","type":"text","label":"Name","required":true}]`),
	})
	if err != nil {
		t.Fatalf("mustCreateSchema: %v", err)
	}
	return schema
}

func mustCreateWebhook(t *testing.T, svc *Service, schemaID uuid.UUID, active bool) *FormWebhook {
	t.Helper()
	secret := "super-secret-key-1234"
	w, err := svc.CreateWebhook(context.Background(), CreateWebhookInput{
		TenantID:     testTenant,
		FormSchemaID: schemaID,
		URL:          "https://example.com/hook",
		Secret:       &secret,
		Active:       active,
	})
	if err != nil {
		t.Fatalf("mustCreateWebhook: %v", err)
	}
	return w
}

// ============================================================================
// Tests: CreateFormSchema
// ============================================================================

func TestCreateFormSchema_Success(t *testing.T) {
	svc, _ := newSvc()
	schema, err := svc.CreateFormSchema(context.Background(), CreateSchemaInput{
		TenantID: testTenant,
		Title:    "My Form",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if schema.Status != FormSchemaStatusDraft {
		t.Errorf("expected draft status, got %q", schema.Status)
	}
	if schema.PageCount != 1 {
		t.Errorf("expected page_count 1, got %d", schema.PageCount)
	}
}

func TestCreateFormSchema_EmptyTitle_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.CreateFormSchema(context.Background(), CreateSchemaInput{
		TenantID: testTenant,
		Title:    "   ",
	})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestCreateFormSchema_InvalidFieldsJSON_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.CreateFormSchema(context.Background(), CreateSchemaInput{
		TenantID: testTenant,
		Title:    "Form",
		Fields:   []byte(`not-json`),
	})
	if !errors.Is(err, ErrInvalidFields) {
		t.Errorf("expected ErrInvalidFields, got %v", err)
	}
}

func TestCreateFormSchema_UnknownFieldType_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.CreateFormSchema(context.Background(), CreateSchemaInput{
		TenantID: testTenant,
		Title:    "Form",
		Fields:   []byte(`[{"id":"f1","type":"unknown","label":"Unknown"}]`),
	})
	if !errors.Is(err, ErrInvalidFields) {
		t.Errorf("expected ErrInvalidFields for unknown field type, got %v", err)
	}
}

func TestCreateFormSchema_ValidAllNineFieldTypes(t *testing.T) {
	svc, _ := newSvc()
	fields := []byte(`[
		{"id":"f1","type":"text","label":"Text"},
		{"id":"f2","type":"textarea","label":"TextArea"},
		{"id":"f3","type":"email","label":"Email"},
		{"id":"f4","type":"number","label":"Number"},
		{"id":"f5","type":"select","label":"Select"},
		{"id":"f6","type":"radio","label":"Radio"},
		{"id":"f7","type":"checkbox","label":"Checkbox"},
		{"id":"f8","type":"date","label":"Date"},
		{"id":"f9","type":"file","label":"File"}
	]`)
	_, err := svc.CreateFormSchema(context.Background(), CreateSchemaInput{
		TenantID: testTenant,
		Title:    "All Types",
		Fields:   fields,
	})
	if err != nil {
		t.Fatalf("all nine field types should be valid, got: %v", err)
	}
}

func TestCreateFormSchema_MissingFieldID_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.CreateFormSchema(context.Background(), CreateSchemaInput{
		TenantID: testTenant,
		Title:    "Form",
		Fields:   []byte(`[{"id":"","type":"text","label":"Name"}]`),
	})
	if !errors.Is(err, ErrInvalidFields) {
		t.Errorf("expected ErrInvalidFields for empty field ID, got %v", err)
	}
}

func TestCreateFormSchema_UnknownRole_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.CreateFormSchema(context.Background(), CreateSchemaInput{
		TenantID: testTenant,
		Title:    "Form",
		Fields:   []byte(`[{"id":"f1","type":"text","label":"Name","role":"assignee"}]`),
	})
	if !errors.Is(err, ErrInvalidFields) {
		t.Errorf("expected ErrInvalidFields for unknown role, got %v", err)
	}
}

func TestCreateFormSchema_ValidRoles_AllSixAccepted(t *testing.T) {
	svc, _ := newSvc()
	fields := []byte(`[
		{"id":"f1","type":"text","label":"Subject","role":"subject"},
		{"id":"f2","type":"textarea","label":"Description","role":"description"},
		{"id":"f3","type":"select","label":"Priority","role":"priority"},
		{"id":"f4","type":"text","label":"Category","role":"category"},
		{"id":"f5","type":"text","label":"Name","role":"requester_name"},
		{"id":"f6","type":"email","label":"Email","role":"requester_email"},
		{"id":"f7","type":"text","label":"Unmarked"}
	]`)
	_, err := svc.CreateFormSchema(context.Background(), CreateSchemaInput{
		TenantID: testTenant,
		Title:    "Intake Form",
		Fields:   fields,
	})
	if err != nil {
		t.Fatalf("all six roles plus an unmarked field should be valid, got: %v", err)
	}
}

func TestCreateFormSchema_IntakeTargetID_Persisted(t *testing.T) {
	svc, _ := newSvc()
	targetID := "helpdesk_ticket"
	schema, err := svc.CreateFormSchema(context.Background(), CreateSchemaInput{
		TenantID:       testTenant,
		Title:          "Intake Form",
		IntakeTargetID: &targetID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.IntakeTargetID == nil || *schema.IntakeTargetID != "helpdesk_ticket" {
		t.Errorf("expected intake_target_id %q, got %v", targetID, schema.IntakeTargetID)
	}
}

func TestCreateFormSchema_BlankIntakeTargetID_StoredAsNil(t *testing.T) {
	svc, _ := newSvc()
	blank := "   "
	schema, err := svc.CreateFormSchema(context.Background(), CreateSchemaInput{
		TenantID:       testTenant,
		Title:          "Plain Form",
		IntakeTargetID: &blank,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.IntakeTargetID != nil {
		t.Errorf("expected nil intake_target_id for blank input, got %q", *schema.IntakeTargetID)
	}
}

// ============================================================================
// Tests: UpdateFormSchema
// ============================================================================

func TestUpdateFormSchema_PartialUpdate_OnlyUpdatesProvidedFields(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)

	newTitle := "Updated Title"
	updated, err := svc.UpdateFormSchema(context.Background(), UpdateSchemaInput{
		ID:       schema.ID,
		TenantID: testTenant,
		Title:    &newTitle,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("expected title updated, got %q", updated.Title)
	}
	// Fields should be unchanged
	if string(updated.Fields) != string(schema.Fields) {
		t.Error("expected fields to remain unchanged")
	}
}

func TestUpdateFormSchema_InvalidFieldType_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)

	_, err := svc.UpdateFormSchema(context.Background(), UpdateSchemaInput{
		ID:       schema.ID,
		TenantID: testTenant,
		Fields:   []byte(`[{"id":"f1","type":"video","label":"Video"}]`),
	})
	if !errors.Is(err, ErrInvalidFields) {
		t.Errorf("expected ErrInvalidFields for invalid type in update, got %v", err)
	}
}

func TestUpdateFormSchema_UnknownRole_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)

	_, err := svc.UpdateFormSchema(context.Background(), UpdateSchemaInput{
		ID:       schema.ID,
		TenantID: testTenant,
		Fields:   []byte(`[{"id":"f1","type":"text","label":"Name","role":"owner"}]`),
	})
	if !errors.Is(err, ErrInvalidFields) {
		t.Errorf("expected ErrInvalidFields for unknown role in update, got %v", err)
	}
}

func TestUpdateFormSchema_IntakeTargetID_SetThenCleared(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)

	targetID := "helpdesk_ticket"
	bound, err := svc.UpdateFormSchema(context.Background(), UpdateSchemaInput{
		ID:             schema.ID,
		TenantID:       testTenant,
		IntakeTargetID: &targetID,
	})
	if err != nil {
		t.Fatalf("unexpected error binding target: %v", err)
	}
	if bound.IntakeTargetID == nil || *bound.IntakeTargetID != "helpdesk_ticket" {
		t.Fatalf("expected intake_target_id set, got %v", bound.IntakeTargetID)
	}

	cleared := ""
	unbound, err := svc.UpdateFormSchema(context.Background(), UpdateSchemaInput{
		ID:             schema.ID,
		TenantID:       testTenant,
		IntakeTargetID: &cleared,
	})
	if err != nil {
		t.Fatalf("unexpected error clearing target: %v", err)
	}
	if unbound.IntakeTargetID != nil {
		t.Errorf("expected intake_target_id cleared to nil, got %q", *unbound.IntakeTargetID)
	}
}

func TestUpdateFormSchema_NotFound_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.UpdateFormSchema(context.Background(), UpdateSchemaInput{
		ID:       uuid.New(),
		TenantID: testTenant,
	})
	if !errors.Is(err, ErrSchemaNotFound) {
		t.Errorf("expected ErrSchemaNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: DeleteFormSchema
// ============================================================================

func TestDeleteFormSchema_SoftDeletes(t *testing.T) {
	svc, repo := newSvc()
	schema := mustCreateSchema(t, svc)

	if err := svc.DeleteFormSchema(context.Background(), schema.ID, testTenant); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be soft-deleted — check the stub directly.
	repo.mu.Lock()
	stored := repo.schemas[schema.ID]
	repo.mu.Unlock()
	if stored.DeletedAt == nil {
		t.Error("expected deleted_at to be set after soft delete")
	}
}

func TestDeleteFormSchema_NotFound_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	err := svc.DeleteFormSchema(context.Background(), uuid.New(), testTenant)
	if !errors.Is(err, ErrSchemaNotFound) {
		t.Errorf("expected ErrSchemaNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: ListFormSchemas
// ============================================================================

func TestListFormSchemas_RespectsPaging(t *testing.T) {
	svc, _ := newSvc()
	for range 5 {
		_, _ = svc.CreateFormSchema(context.Background(), CreateSchemaInput{
			TenantID: testTenant,
			Title:    "Form",
		})
	}

	out, total, err := svc.ListFormSchemas(context.Background(), ListSchemasInput{
		TenantID: testTenant,
		Limit:    2,
		Offset:   0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(out) != 2 {
		t.Errorf("expected 2 results with limit=2, got %d", len(out))
	}
}

func TestListFormSchemas_DefaultPagingWhenZeros(t *testing.T) {
	svc, _ := newSvc()
	mustCreateSchema(t, svc)

	out, total, err := svc.ListFormSchemas(context.Background(), ListSchemasInput{
		TenantID: testTenant,
		// Limit=0, Offset=0 => defaults (limit=20, offset=0)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(out) != 1 {
		t.Errorf("expected 1 result, got total=%d len=%d", total, len(out))
	}
}

// ============================================================================
// Tests: DuplicateFormSchema
// ============================================================================

func TestDuplicateFormSchema_CopiesFieldsResetsStatus(t *testing.T) {
	svc, _ := newSvc()
	original := mustCreateSchema(t, svc)

	// Set the original to active and bind an intake target.
	active := FormSchemaStatusActive
	targetID := "helpdesk_ticket"
	_, _ = svc.UpdateFormSchema(context.Background(), UpdateSchemaInput{
		ID:             original.ID,
		TenantID:       testTenant,
		Status:         &active,
		IntakeTargetID: &targetID,
	})

	dup, err := svc.DuplicateFormSchema(context.Background(), original.ID, testTenant, "Copy of Contact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup.ID == original.ID {
		t.Error("expected new ID for duplicate")
	}
	if dup.Status != FormSchemaStatusDraft {
		t.Errorf("expected duplicate to be draft, got %q", dup.Status)
	}
	if dup.Title != "Copy of Contact" {
		t.Errorf("expected custom title, got %q", dup.Title)
	}
	if dup.IntakeTargetID == nil || *dup.IntakeTargetID != "helpdesk_ticket" {
		t.Errorf("expected intake_target_id carried over to duplicate, got %v", dup.IntakeTargetID)
	}
}

// ============================================================================
// Tests: CreateSubmission
// ============================================================================

func TestCreateSubmission_EnqueuesWebhooksForActiveWebhooks(t *testing.T) {
	svc, repo := newSvc()
	schema := mustCreateSchema(t, svc)

	mustCreateWebhook(t, svc, schema.ID, true)  // active
	mustCreateWebhook(t, svc, schema.ID, false) // inactive

	schemaID := schema.ID
	_, err := svc.CreateSubmission(context.Background(), CreateSubmissionInput{
		TenantID:     testTenant,
		FormSchemaID: &schemaID,
		Answers:      []byte(`{"name":"Alice"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only 1 active webhook should be enqueued.
	repo.mu.Lock()
	ids := repo.lastWebhookIDs
	repo.mu.Unlock()
	if len(ids) != 1 {
		t.Errorf("expected 1 webhook ID enqueued, got %d", len(ids))
	}
}

func TestCreateSubmission_NoWebhooks_StillWorks(t *testing.T) {
	svc, repo := newSvc()
	schema := mustCreateSchema(t, svc)

	schemaID := schema.ID
	sub, err := svc.CreateSubmission(context.Background(), CreateSubmissionInput{
		TenantID:     testTenant,
		FormSchemaID: &schemaID,
		Answers:      []byte(`{"email":"test@example.com"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.ID == uuid.Nil {
		t.Error("expected non-nil submission ID")
	}
	repo.mu.Lock()
	ids := repo.lastWebhookIDs
	repo.mu.Unlock()
	if len(ids) != 0 {
		t.Errorf("expected no webhook IDs, got %d", len(ids))
	}
}

func TestCreateSubmission_InvalidIPAddress_StillSavesWithNilIP(t *testing.T) {
	svc, repo := newSvc()
	schema := mustCreateSchema(t, svc)

	schemaID := schema.ID
	sub, err := svc.CreateSubmission(context.Background(), CreateSubmissionInput{
		TenantID:     testTenant,
		FormSchemaID: &schemaID,
		Answers:      []byte(`{"name":"Bob"}`),
		IPAddress:    "not-an-ip",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repo.mu.Lock()
	stored := repo.submissions[sub.ID]
	repo.mu.Unlock()
	if stored.IPAddress != nil {
		t.Errorf("expected nil IP for invalid input, got %v", *stored.IPAddress)
	}
}

func TestCreateSubmission_ValidIPv4_StoresParsedIP(t *testing.T) {
	svc, repo := newSvc()
	schema := mustCreateSchema(t, svc)

	schemaID := schema.ID
	sub, err := svc.CreateSubmission(context.Background(), CreateSubmissionInput{
		TenantID:     testTenant,
		FormSchemaID: &schemaID,
		Answers:      []byte(`{"name":"Carol"}`),
		IPAddress:    "192.168.1.1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repo.mu.Lock()
	stored := repo.submissions[sub.ID]
	repo.mu.Unlock()
	if stored.IPAddress == nil || *stored.IPAddress != "192.168.1.1" {
		t.Errorf("expected stored IP 192.168.1.1, got %v", stored.IPAddress)
	}
}

// ============================================================================
// Tests: UpdateSubmissionStatus
// ============================================================================

func TestUpdateSubmissionStatus_Success(t *testing.T) {
	svc, repo := newSvc()
	schema := mustCreateSchema(t, svc)

	schemaID := schema.ID
	sub, _ := svc.CreateSubmission(context.Background(), CreateSubmissionInput{
		TenantID:     testTenant,
		FormSchemaID: &schemaID,
		Answers:      []byte(`{"x":1}`),
	})

	updated, err := svc.UpdateSubmissionStatus(context.Background(), sub.ID, testTenant, FormSubmissionStatusRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != FormSubmissionStatusRead {
		t.Errorf("expected status read, got %q", updated.Status)
	}

	repo.mu.Lock()
	stored := repo.submissions[sub.ID]
	repo.mu.Unlock()
	if stored.Status != FormSubmissionStatusRead {
		t.Errorf("expected stored status read, got %q", stored.Status)
	}
}

// ============================================================================
// Tests: ExportSubmissions
// ============================================================================

func prepareExportData(t *testing.T, svc *Service) uuid.UUID {
	t.Helper()
	schema := mustCreateSchema(t, svc)
	schemaID := schema.ID

	for range 3 {
		_, err := svc.CreateSubmission(context.Background(), CreateSubmissionInput{
			TenantID:     testTenant,
			FormSchemaID: &schemaID,
			Answers:      []byte(`{"name":"Test","score":42}`),
		})
		if err != nil {
			t.Fatalf("create test submission: %v", err)
		}
	}
	return schemaID
}

func TestExportSubmissions_CSV_ReturnsValidCSVWithBOM(t *testing.T) {
	svc, _ := newSvc()
	schemaID := prepareExportData(t, svc)

	result, err := svc.ExportSubmissions(context.Background(), ExportInput{
		TenantID:     testTenant,
		FormSchemaID: schemaID,
		Format:       ExportFormatCSV,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MimeType != "text/csv" {
		t.Errorf("expected text/csv, got %q", result.MimeType)
	}

	// BOM check
	if len(result.Content) < 3 || result.Content[0] != 0xEF || result.Content[1] != 0xBB || result.Content[2] != 0xBF {
		t.Error("expected UTF-8 BOM at start of CSV")
	}

	// Parse the CSV (skip BOM)
	r := csv.NewReader(bytes.NewReader(result.Content[3:]))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if len(records) < 2 {
		t.Errorf("expected header + at least 1 data row, got %d rows", len(records))
	}
	// Header should contain 'id'
	if records[0][0] != "id" {
		t.Errorf("expected first column 'id', got %q", records[0][0])
	}
}

func TestExportSubmissions_XLSX_ReturnsValidXLSX(t *testing.T) {
	svc, _ := newSvc()
	schemaID := prepareExportData(t, svc)

	result, err := svc.ExportSubmissions(context.Background(), ExportInput{
		TenantID:     testTenant,
		FormSchemaID: schemaID,
		Format:       ExportFormatXLSX,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MimeType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("unexpected MIME type: %q", result.MimeType)
	}

	// Parse the XLSX with excelize
	f, err := excelize.OpenReader(bytes.NewReader(result.Content))
	if err != nil {
		t.Fatalf("excelize open error: %v", err)
	}
	defer f.Close() //nolint:errcheck

	rows, err := f.GetRows("Submissions")
	if err != nil {
		t.Fatalf("get rows error: %v", err)
	}
	if len(rows) < 2 {
		t.Errorf("expected header + data rows, got %d", len(rows))
	}
	if rows[0][0] != "id" {
		t.Errorf("expected first cell 'id', got %q", rows[0][0])
	}
}

func TestExportSubmissions_UnsupportedFormat_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)

	_, err := svc.ExportSubmissions(context.Background(), ExportInput{
		TenantID:     testTenant,
		FormSchemaID: schema.ID,
		Format:       ExportFormat("pdf"), // not a valid export format
	})
	if !errors.Is(err, ErrInvalidExportFormat) {
		t.Errorf("expected ErrInvalidExportFormat, got %v", err)
	}
}

// ============================================================================
// Tests: CreateWebhook
// ============================================================================

func TestCreateWebhook_InvalidURL_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)

	_, err := svc.CreateWebhook(context.Background(), CreateWebhookInput{
		TenantID:     testTenant,
		FormSchemaID: schema.ID,
		URL:          "not-a-url",
	})
	if !errors.Is(err, ErrInvalidURL) {
		t.Errorf("expected ErrInvalidURL, got %v", err)
	}
}

func TestCreateWebhook_FTPScheme_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)

	_, err := svc.CreateWebhook(context.Background(), CreateWebhookInput{
		TenantID:     testTenant,
		FormSchemaID: schema.ID,
		URL:          "ftp://example.com/hook",
	})
	if !errors.Is(err, ErrInvalidURL) {
		t.Errorf("expected ErrInvalidURL for ftp scheme, got %v", err)
	}
}

func TestCreateWebhook_SchemaNotFound_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.CreateWebhook(context.Background(), CreateWebhookInput{
		TenantID:     testTenant,
		FormSchemaID: uuid.New(), // does not exist
		URL:          "https://example.com/hook",
	})
	if !errors.Is(err, ErrSchemaNotFound) {
		t.Errorf("expected ErrSchemaNotFound, got %v", err)
	}
}

func TestCreateWebhook_DefaultEvents(t *testing.T) {
	svc, repo := newSvc()
	schema := mustCreateSchema(t, svc)

	w, err := svc.CreateWebhook(context.Background(), CreateWebhookInput{
		TenantID:     testTenant,
		FormSchemaID: schema.ID,
		URL:          "https://example.com/hook",
		Active:       true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Check masked secret (no secret provided, so nil)
	if w.Secret != nil {
		t.Error("expected nil secret when none provided")
	}
	// Check raw stored events = default
	repo.mu.Lock()
	stored := repo.webhooks[w.ID]
	repo.mu.Unlock()
	if len(stored.Events) != 1 || stored.Events[0] != "submission.created" {
		t.Errorf("expected default event, got %v", stored.Events)
	}
}

// ============================================================================
// Tests: GetWebhook / ListWebhooks — secret masking
// ============================================================================

func TestGetWebhook_MasksSecret(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)
	w := mustCreateWebhook(t, svc, schema.ID, true)

	// Get via service — should be masked.
	fetched, err := svc.GetWebhook(context.Background(), w.ID, testTenant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched.Secret == nil {
		t.Fatal("expected masked secret, got nil")
	}
	if *fetched.Secret == "super-secret-key-1234" {
		t.Error("expected secret to be masked, got plaintext")
	}
	// Last 4 chars of "super-secret-key-1234" = "1234"
	if len(*fetched.Secret) < 4 {
		t.Error("masked secret too short")
	}
	masked := *fetched.Secret
	if masked[len(masked)-4:] != "1234" {
		t.Errorf("expected last 4 chars '1234', got %q", masked[len(masked)-4:])
	}
}

func TestListWebhooks_MasksAllSecrets(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)

	mustCreateWebhook(t, svc, schema.ID, true)
	mustCreateWebhook(t, svc, schema.ID, false)

	webhooks, err := svc.ListWebhooks(context.Background(), schema.ID, testTenant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(webhooks) != 2 {
		t.Errorf("expected 2 webhooks, got %d", len(webhooks))
	}
	for _, w := range webhooks {
		if w.Secret != nil && *w.Secret == "super-secret-key-1234" {
			t.Errorf("webhook %s has unmasked secret", w.ID)
		}
	}
}

// ============================================================================
// Tests: UpdateWebhook
// ============================================================================

func TestUpdateWebhook_PartialUpdate(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)
	w := mustCreateWebhook(t, svc, schema.ID, true)

	newURL := "https://new.example.com/hook"
	inactive := false
	updated, err := svc.UpdateWebhook(context.Background(), UpdateWebhookInput{
		ID:       w.ID,
		TenantID: testTenant,
		URL:      &newURL,
		Active:   &inactive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Active {
		t.Error("expected active=false after update")
	}
	// Secret should still be masked
	if updated.Secret != nil && *updated.Secret == "super-secret-key-1234" {
		t.Error("expected secret to be masked in update response")
	}
}

func TestUpdateWebhook_InvalidURL_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)
	w := mustCreateWebhook(t, svc, schema.ID, true)

	badURL := "://broken"
	_, err := svc.UpdateWebhook(context.Background(), UpdateWebhookInput{
		ID:       w.ID,
		TenantID: testTenant,
		URL:      &badURL,
	})
	if !errors.Is(err, ErrInvalidURL) {
		t.Errorf("expected ErrInvalidURL, got %v", err)
	}
}

// ============================================================================
// Tests: helpers
// ============================================================================

func TestMaskSecret_ShortSecret(t *testing.T) {
	if maskSecret("ab") != "****" {
		t.Error("expected **** for 2-char secret")
	}
	if maskSecret("") != "****" {
		t.Error("expected **** for empty secret")
	}
}

func TestMaskSecret_LongSecret(t *testing.T) {
	masked := maskSecret("abcdefgh")
	if masked[len(masked)-4:] != "efgh" {
		t.Errorf("expected last 4 'efgh', got %q", masked[len(masked)-4:])
	}
}

func TestNormalizePaging_Defaults(t *testing.T) {
	l, o := normalizePaging(0, 0)
	if l != 20 || o != 0 {
		t.Errorf("expected 20/0, got %d/%d", l, o)
	}
}

func TestNormalizePaging_ClampMax(t *testing.T) {
	l, _ := normalizePaging(500, 0)
	if l != 100 {
		t.Errorf("expected clamped to 100, got %d", l)
	}
}

func TestValidateWebhookURL_HTTPSScheme(t *testing.T) {
	if err := validateWebhookURL("https://example.com/hook"); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestValidateWebhookURL_HTTPScheme(t *testing.T) {
	if err := validateWebhookURL("http://localhost:8080/hook"); err != nil {
		t.Errorf("expected valid for http, got %v", err)
	}
}

func TestValidateWebhookURL_EmptyString(t *testing.T) {
	if err := validateWebhookURL(""); !errors.Is(err, ErrInvalidURL) {
		t.Errorf("expected ErrInvalidURL for empty string, got %v", err)
	}
}

func TestValidateWebhookURL_NoHost(t *testing.T) {
	if err := validateWebhookURL("https://"); !errors.Is(err, ErrInvalidURL) {
		t.Errorf("expected ErrInvalidURL for no host, got %v", err)
	}
}

// ============================================================================
// Tests: GetFormStats
// ============================================================================

func TestGetFormStats_Success(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)

	schemaID := schema.ID
	// Create 2 submissions (both status=new by default)
	for range 2 {
		_, err := svc.CreateSubmission(context.Background(), CreateSubmissionInput{
			TenantID:     testTenant,
			FormSchemaID: &schemaID,
			Answers:      []byte(`{"name":"Test"}`),
		})
		if err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	stats, err := svc.GetFormStats(context.Background(), GetFormStatsInput{
		TenantID:     testTenant,
		FormSchemaID: schemaID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalCount != 2 {
		t.Errorf("expected TotalCount=2, got %d", stats.TotalCount)
	}
	if stats.NewCount != 2 {
		t.Errorf("expected NewCount=2, got %d", stats.NewCount)
	}
	// All submissions were just created, so they fall within 7d and 30d
	if stats.Last7dCount != 2 {
		t.Errorf("expected Last7dCount=2, got %d", stats.Last7dCount)
	}
	if stats.Last30dCount != 2 {
		t.Errorf("expected Last30dCount=2, got %d", stats.Last30dCount)
	}
	if stats.FormSchemaID != schemaID {
		t.Errorf("expected FormSchemaID=%v, got %v", schemaID, stats.FormSchemaID)
	}
}

func TestGetFormStats_InvalidTenantID_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.GetFormStats(context.Background(), GetFormStatsInput{
		TenantID:     uuid.Nil,
		FormSchemaID: uuid.New(),
	})
	if !errors.Is(err, ErrInvalidFields) {
		t.Errorf("expected ErrInvalidFields for nil tenant_id, got %v", err)
	}
}

func TestGetFormStats_InvalidFormSchemaID_ReturnsError(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.GetFormStats(context.Background(), GetFormStatsInput{
		TenantID:     testTenant,
		FormSchemaID: uuid.Nil,
	})
	if !errors.Is(err, ErrInvalidFields) {
		t.Errorf("expected ErrInvalidFields for nil form_schema_id, got %v", err)
	}
}

func TestGetFormStats_EmptySchema_ReturnsZeroCounts(t *testing.T) {
	svc, _ := newSvc()
	schema := mustCreateSchema(t, svc)

	stats, err := svc.GetFormStats(context.Background(), GetFormStatsInput{
		TenantID:     testTenant,
		FormSchemaID: schema.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalCount != 0 || stats.NewCount != 0 || stats.Last7dCount != 0 || stats.Last30dCount != 0 {
		t.Errorf("expected all-zero stats for empty schema, got %+v", stats)
	}
}
