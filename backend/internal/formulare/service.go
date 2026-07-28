package formulare

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

// ============================================================================
// Service
// ============================================================================

// Service holds formulare business logic. Thick-service pattern: handlers
// (HTTP, gRPC) translate transport shapes and call through.
type Service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService creates a formulare service.
func NewService(repo Repository, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, logger: logger}
}

// ============================================================================
// Helpers
// ============================================================================

var validFieldTypes = map[string]struct{}{
	"text": {}, "textarea": {}, "email": {}, "number": {},
	"select": {}, "radio": {}, "checkbox": {}, "date": {}, "file": {},
}

// validateFields unmarshals a JSONB fields array and checks each field.Type.
func validateFields(fields []byte) error {
	if len(fields) == 0 || string(fields) == "[]" || string(fields) == "null" {
		return nil
	}
	var parsed []FormField
	if err := json.Unmarshal(fields, &parsed); err != nil {
		return ErrInvalidFields
	}
	for _, f := range parsed {
		if strings.TrimSpace(f.ID) == "" {
			return ErrInvalidFields
		}
		if strings.TrimSpace(f.Label) == "" {
			return ErrInvalidFields
		}
		if _, ok := validFieldTypes[f.Type]; !ok {
			return ErrInvalidFields
		}
	}
	return nil
}

// normalizePaging clamps limit (default 20, max 100) and offset (min 0).
func normalizePaging(limit, offset int32) (int, int) {
	l := int(limit)
	o := int(offset)
	if l <= 0 {
		l = 20
	}
	if l > 100 {
		l = 100
	}
	if o < 0 {
		o = 0
	}
	return l, o
}

// maskSecret masks a webhook secret for read responses: last 4 chars with dot padding.
func maskSecret(s string) string {
	if len(s) < 4 {
		return "****"
	}
	return strings.Repeat(".", len(s)-4) + s[len(s)-4:]
}

// validateWebhookURL checks that a URL is valid and uses http or https scheme.
func validateWebhookURL(u string) error {
	if strings.TrimSpace(u) == "" {
		return ErrInvalidURL
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}
	if parsed.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

// parseIPAddress parses an IP address string; returns nil on parse failure (optional).
func parseIPAddress(ip string) *string {
	if ip == "" {
		return nil
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil
	}
	s := addr.String()
	return &s
}

// maskWebhookSecret overwrites the Secret field in a FormWebhook copy for read responses.
func maskWebhookSecret(w *FormWebhook) *FormWebhook {
	if w == nil {
		return nil
	}
	copy := *w
	if copy.Secret != nil {
		masked := maskSecret(*copy.Secret)
		copy.Secret = &masked
	}
	return &copy
}

// ============================================================================
// Input structs
// ============================================================================

// CreateSchemaInput captures data for creating a form schema.
type CreateSchemaInput struct {
	TenantID    uuid.UUID
	Title       string
	Description string
	Fields      []byte // raw JSON array
	Status      FormSchemaStatus
	IsTemplate  bool
	IsPublic    bool
	PageCount   int
	CreatedBy   *uuid.UUID
}

// UpdateSchemaInput captures partial updates on a form schema.
type UpdateSchemaInput struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Title       *string
	Description *string
	Fields      []byte // empty = no change
	Status      *FormSchemaStatus
	IsTemplate  *bool
	IsPublic    *bool
	PageCount   *int
}

// ListSchemasInput is the user-facing schema list request.
type ListSchemasInput struct {
	TenantID   uuid.UUID
	Status     *FormSchemaStatus
	IsTemplate *bool
	Search     string
	Limit      int32
	Offset     int32
}

// CreateSubmissionInput captures data for creating a form submission.
type CreateSubmissionInput struct {
	TenantID     uuid.UUID
	FormSchemaID *uuid.UUID
	Answers      []byte
	SubmittedBy  *string
	IPAddress    string
}

// ListSubmissionsInput is the user-facing submission list request.
type ListSubmissionsInput struct {
	TenantID     uuid.UUID
	FormSchemaID *uuid.UUID
	Status       *FormSubmissionStatus
	Limit        int32
	Offset       int32
}

// ExportInput drives a submission export.
type ExportInput struct {
	TenantID     uuid.UUID
	FormSchemaID uuid.UUID
	Format       ExportFormat
	Status       *FormSubmissionStatus
	SubmittedAfter  *time.Time
	SubmittedBefore *time.Time
}

// ExportResult holds the generated export file.
type ExportResult struct {
	Content  []byte
	Filename string
	MimeType string
}

// CreateWebhookInput captures data for creating a webhook.
type CreateWebhookInput struct {
	TenantID     uuid.UUID
	FormSchemaID uuid.UUID
	URL          string
	Secret       *string
	Events       []string
	Active       bool
}

// UpdateWebhookInput captures partial updates on a webhook.
type UpdateWebhookInput struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	URL       *string
	Secret    *string
	Events    []string
	EventsSet bool
	Active    *bool
}

// ListDeliveriesInput is the user-facing delivery list request.
type ListDeliveriesInput struct {
	TenantID     uuid.UUID
	WebhookID    *uuid.UUID
	SubmissionID *uuid.UUID
	Status       *WebhookDeliveryStatus
	Limit        int32
	Offset       int32
}

// ============================================================================
// Schema methods
// ============================================================================

// CreateFormSchema validates and creates a new form schema.
func (s *Service) CreateFormSchema(ctx context.Context, in CreateSchemaInput) (*FormSchema, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required: %w", ErrInvalidFields)
	}

	fields := in.Fields
	if len(fields) == 0 {
		fields = []byte("[]")
	}
	if err := validateFields(fields); err != nil {
		return nil, err
	}

	status := in.Status
	if status == "" {
		status = FormSchemaStatusDraft
	}

	pageCount := max(in.PageCount, 1)

	now := time.Now().UTC()
	schema := &FormSchema{
		ID:          uuid.New(),
		TenantID:    in.TenantID,
		Title:       title,
		Description: strings.TrimSpace(in.Description),
		Fields:      fields,
		Status:      status,
		IsTemplate:  in.IsTemplate,
		IsPublic:    in.IsPublic,
		PageCount:   pageCount,
		CreatedBy:   in.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateSchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("create form schema: %w", err)
	}

	s.logger.Info("formulare schema created",
		"schema_id", schema.ID,
		"tenant_id", schema.TenantID,
		"title", schema.Title,
	)
	return schema, nil
}

// GetFormSchema retrieves a form schema by ID.
func (s *Service) GetFormSchema(ctx context.Context, id, tenantID uuid.UUID) (*FormSchema, error) {
	return s.repo.GetSchema(ctx, id, tenantID)
}

// UpdateFormSchema applies partial changes to an existing form schema.
func (s *Service) UpdateFormSchema(ctx context.Context, in UpdateSchemaInput) (*FormSchema, error) {
	schema, err := s.repo.GetSchema(ctx, in.ID, in.TenantID)
	if err != nil {
		return nil, err
	}

	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			return nil, fmt.Errorf("title cannot be empty: %w", ErrInvalidFields)
		}
		schema.Title = t
	}
	if in.Description != nil {
		schema.Description = strings.TrimSpace(*in.Description)
	}
	if len(in.Fields) > 0 {
		if err := validateFields(in.Fields); err != nil {
			return nil, err
		}
		schema.Fields = append([]byte(nil), in.Fields...)
	}
	if in.Status != nil {
		schema.Status = *in.Status
	}
	if in.IsTemplate != nil {
		schema.IsTemplate = *in.IsTemplate
	}
	if in.IsPublic != nil {
		schema.IsPublic = *in.IsPublic
	}
	if in.PageCount != nil {
		schema.PageCount = max(*in.PageCount, 1)
	}

	schema.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateSchema(ctx, schema); err != nil {
		return nil, err
	}

	s.logger.Info("formulare schema updated",
		"schema_id", schema.ID,
		"tenant_id", schema.TenantID,
	)
	return schema, nil
}

// DeleteFormSchema soft-deletes a form schema.
func (s *Service) DeleteFormSchema(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.SoftDeleteSchema(ctx, id, tenantID); err != nil {
		return err
	}
	s.logger.Info("formulare schema deleted",
		"schema_id", id,
		"tenant_id", tenantID,
	)
	return nil
}

// ListFormSchemas returns a paginated list of form schemas.
func (s *Service) ListFormSchemas(ctx context.Context, in ListSchemasInput) ([]*FormSchema, int, error) {
	limit, offset := normalizePaging(in.Limit, in.Offset)
	filter := ListSchemasFilter{
		Status:     in.Status,
		IsTemplate: in.IsTemplate,
		Search:     in.Search,
		Limit:      limit,
		Offset:     offset,
	}
	return s.repo.ListSchemas(ctx, in.TenantID, filter)
}

// DuplicateFormSchema creates a copy of an existing schema in draft state.
func (s *Service) DuplicateFormSchema(ctx context.Context, id, tenantID uuid.UUID, newTitle string) (*FormSchema, error) {
	schema, err := s.repo.DuplicateSchema(ctx, id, tenantID, newTitle)
	if err != nil {
		return nil, err
	}
	s.logger.Info("formulare schema duplicated",
		"original_id", id,
		"new_id", schema.ID,
		"tenant_id", tenantID,
	)
	return schema, nil
}

// ============================================================================
// Submission methods
// ============================================================================

// CreateSubmission validates and creates a new form submission, enqueueing
// webhook deliveries for all active webhooks on the schema.
func (s *Service) CreateSubmission(ctx context.Context, in CreateSubmissionInput) (*FormSubmission, error) {
	if len(in.Answers) == 0 {
		return nil, fmt.Errorf("answers are required: %w", ErrInvalidFields)
	}

	ipAddr := parseIPAddress(in.IPAddress)

	// Collect active webhook IDs for transactional enqueue.
	var webhookIDs []uuid.UUID
	if in.FormSchemaID != nil {
		webhooks, err := s.repo.ListActiveWebhooksForSchema(ctx, in.TenantID, *in.FormSchemaID)
		if err != nil {
			return nil, fmt.Errorf("list active webhooks: %w", err)
		}
		for _, w := range webhooks {
			webhookIDs = append(webhookIDs, w.ID)
		}
	}

	now := time.Now().UTC()
	submission := &FormSubmission{
		ID:           uuid.New(),
		FormSchemaID: in.FormSchemaID,
		TenantID:     in.TenantID,
		Answers:      in.Answers,
		Status:       FormSubmissionStatusNew,
		SubmittedBy:  in.SubmittedBy,
		IPAddress:    ipAddr,
		SubmittedAt:  now,
	}

	if err := s.repo.CreateSubmission(ctx, submission, webhookIDs); err != nil {
		return nil, fmt.Errorf("create submission: %w", err)
	}

	s.logger.Info("formulare submission created",
		"submission_id", submission.ID,
		"form_schema_id", in.FormSchemaID,
		"tenant_id", in.TenantID,
		"webhook_count", len(webhookIDs),
	)
	return submission, nil
}

// GetSubmission retrieves a form submission by ID.
func (s *Service) GetSubmission(ctx context.Context, id, tenantID uuid.UUID) (*FormSubmission, error) {
	return s.repo.GetSubmission(ctx, id, tenantID)
}

// ListSubmissions returns a paginated list of submissions.
func (s *Service) ListSubmissions(ctx context.Context, in ListSubmissionsInput) ([]*FormSubmission, int, error) {
	limit, offset := normalizePaging(in.Limit, in.Offset)
	filter := ListSubmissionsFilter{
		FormSchemaID: in.FormSchemaID,
		Status:       in.Status,
		Limit:        limit,
		Offset:       offset,
	}
	return s.repo.ListSubmissions(ctx, in.TenantID, filter)
}

// UpdateSubmissionStatus changes the review status of a submission.
func (s *Service) UpdateSubmissionStatus(ctx context.Context, id, tenantID uuid.UUID, status FormSubmissionStatus) (*FormSubmission, error) {
	if err := s.repo.UpdateSubmissionStatus(ctx, id, tenantID, status); err != nil {
		return nil, err
	}
	return s.repo.GetSubmission(ctx, id, tenantID)
}

// ExportSubmissions exports all submissions for a form schema as CSV or XLSX.
func (s *Service) ExportSubmissions(ctx context.Context, in ExportInput) (*ExportResult, error) {
	if in.Format != ExportFormatCSV && in.Format != ExportFormatXLSX {
		return nil, ErrInvalidExportFormat
	}

	filter := ExportFilter{
		Status:          in.Status,
		SubmittedAfter:  in.SubmittedAfter,
		SubmittedBefore: in.SubmittedBefore,
	}
	submissions, err := s.repo.ListSubmissionsForExport(ctx, in.FormSchemaID, in.TenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("list submissions for export: %w", err)
	}

	// Build header keys from the first submission's answers, or return empty file.
	headerKeys := extractAnswerKeys(submissions)

	dateStr := time.Now().UTC().Format("2006-01-02")
	schemaIDShort := in.FormSchemaID.String()[:8]

	switch in.Format {
	case ExportFormatCSV:
		content, err := buildCSV(submissions, headerKeys)
		if err != nil {
			return nil, fmt.Errorf("build CSV: %w", err)
		}
		return &ExportResult{
			Content:  content,
			Filename: fmt.Sprintf("form_%s_%s.csv", schemaIDShort, dateStr),
			MimeType: "text/csv",
		}, nil

	case ExportFormatXLSX:
		content, err := buildXLSX(submissions, headerKeys)
		if err != nil {
			return nil, fmt.Errorf("build XLSX: %w", err)
		}
		return &ExportResult{
			Content:  content,
			Filename: fmt.Sprintf("form_%s_%s.xlsx", schemaIDShort, dateStr),
			MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		}, nil
	}

	return nil, ErrInvalidExportFormat
}

// extractAnswerKeys collects all unique answer keys from submissions, preserving
// insertion order of first occurrence.
func extractAnswerKeys(submissions []*FormSubmission) []string {
	seen := make(map[string]struct{})
	var keys []string
	for _, sub := range submissions {
		var answers map[string]any
		if err := json.Unmarshal(sub.Answers, &answers); err != nil {
			continue
		}
		for k := range answers {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				keys = append(keys, k)
			}
		}
	}
	return keys
}

// buildCSV produces a UTF-8 BOM-prefixed CSV byte slice.
func buildCSV(submissions []*FormSubmission, headerKeys []string) ([]byte, error) {
	var buf bytes.Buffer
	// UTF-8 BOM
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(&buf)

	// Header row: meta columns + answer keys
	header := append([]string{"id", "submitted_at", "status", "submitted_by", "ip_address"}, headerKeys...)
	if err := w.Write(header); err != nil {
		return nil, err
	}

	for _, sub := range submissions {
		var answers map[string]any
		_ = json.Unmarshal(sub.Answers, &answers)

		row := make([]string, 0, len(header))
		row = append(row, sub.ID.String())
		row = append(row, sub.SubmittedAt.Format(time.RFC3339))
		row = append(row, string(sub.Status))
		if sub.SubmittedBy != nil {
			row = append(row, *sub.SubmittedBy)
		} else {
			row = append(row, "")
		}
		if sub.IPAddress != nil {
			row = append(row, *sub.IPAddress)
		} else {
			row = append(row, "")
		}
		for _, k := range headerKeys {
			if v, ok := answers[k]; ok {
				row = append(row, fmt.Sprintf("%v", v))
			} else {
				row = append(row, "")
			}
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildXLSX produces an XLSX file byte slice using excelize.
func buildXLSX(submissions []*FormSubmission, headerKeys []string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close() //nolint:errcheck

	sheet := "Submissions"
	f.SetSheetName("Sheet1", sheet) //nolint:errcheck,gosec

	header := append([]string{"id", "submitted_at", "status", "submitted_by", "ip_address"}, headerKeys...)
	for col, h := range header {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, h) //nolint:errcheck,gosec
	}

	for rowIdx, sub := range submissions {
		var answers map[string]any
		_ = json.Unmarshal(sub.Answers, &answers)

		xlRow := rowIdx + 2
		meta := []any{
			sub.ID.String(),
			sub.SubmittedAt.Format(time.RFC3339),
			string(sub.Status),
		}
		if sub.SubmittedBy != nil {
			meta = append(meta, *sub.SubmittedBy)
		} else {
			meta = append(meta, "")
		}
		if sub.IPAddress != nil {
			meta = append(meta, *sub.IPAddress)
		} else {
			meta = append(meta, "")
		}
		for col, v := range meta {
			cell, _ := excelize.CoordinatesToCellName(col+1, xlRow)
			f.SetCellValue(sheet, cell, v) //nolint:errcheck,gosec
		}
		for ki, k := range headerKeys {
			cell, _ := excelize.CoordinatesToCellName(5+ki+1, xlRow)
			if v, ok := answers[k]; ok {
				f.SetCellValue(sheet, cell, fmt.Sprintf("%v", v)) //nolint:errcheck,gosec
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ============================================================================
// Webhook methods
// ============================================================================

// CreateWebhook validates and creates a new webhook for a form schema.
func (s *Service) CreateWebhook(ctx context.Context, in CreateWebhookInput) (*FormWebhook, error) {
	if err := validateWebhookURL(in.URL); err != nil {
		return nil, err
	}

	// Verify the schema exists.
	if _, err := s.repo.GetSchema(ctx, in.FormSchemaID, in.TenantID); err != nil {
		return nil, err
	}

	events := in.Events
	if len(events) == 0 {
		events = []string{"submission.created"}
	}

	now := time.Now().UTC()
	webhook := &FormWebhook{
		ID:           uuid.New(),
		FormSchemaID: in.FormSchemaID,
		TenantID:     in.TenantID,
		URL:          in.URL,
		Secret:       in.Secret,
		Events:       events,
		Active:       in.Active,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateWebhook(ctx, webhook); err != nil {
		return nil, fmt.Errorf("create webhook: %w", err)
	}

	s.logger.Info("formulare webhook created",
		"webhook_id", webhook.ID,
		"form_schema_id", webhook.FormSchemaID,
		"tenant_id", webhook.TenantID,
	)
	return maskWebhookSecret(webhook), nil
}

// GetWebhook retrieves a webhook by ID, masking the secret.
func (s *Service) GetWebhook(ctx context.Context, id, tenantID uuid.UUID) (*FormWebhook, error) {
	w, err := s.repo.GetWebhook(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	return maskWebhookSecret(w), nil
}

// UpdateWebhook applies partial changes to an existing webhook.
func (s *Service) UpdateWebhook(ctx context.Context, in UpdateWebhookInput) (*FormWebhook, error) {
	webhook, err := s.repo.GetWebhook(ctx, in.ID, in.TenantID)
	if err != nil {
		return nil, err
	}

	if in.URL != nil {
		if err := validateWebhookURL(*in.URL); err != nil {
			return nil, err
		}
		webhook.URL = *in.URL
	}
	if in.Secret != nil {
		webhook.Secret = in.Secret
	}
	if in.EventsSet {
		if len(in.Events) == 0 {
			webhook.Events = []string{"submission.created"}
		} else {
			webhook.Events = in.Events
		}
	}
	if in.Active != nil {
		webhook.Active = *in.Active
	}

	webhook.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateWebhook(ctx, webhook); err != nil {
		return nil, err
	}

	s.logger.Info("formulare webhook updated",
		"webhook_id", webhook.ID,
		"tenant_id", webhook.TenantID,
	)
	return maskWebhookSecret(webhook), nil
}

// DeleteWebhook removes a webhook permanently.
func (s *Service) DeleteWebhook(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.DeleteWebhook(ctx, id, tenantID); err != nil {
		return err
	}
	s.logger.Info("formulare webhook deleted",
		"webhook_id", id,
		"tenant_id", tenantID,
	)
	return nil
}

// ListWebhooks returns all webhooks for a form schema, with masked secrets.
func (s *Service) ListWebhooks(ctx context.Context, formSchemaID, tenantID uuid.UUID) ([]*FormWebhook, error) {
	webhooks, err := s.repo.ListWebhooks(ctx, formSchemaID, tenantID)
	if err != nil {
		return nil, err
	}
	masked := make([]*FormWebhook, len(webhooks))
	for i, w := range webhooks {
		masked[i] = maskWebhookSecret(w)
	}
	return masked, nil
}

// ============================================================================
// Delivery observability methods
// ============================================================================

// ListWebhookDeliveries returns a paginated list of webhook deliveries.
func (s *Service) ListWebhookDeliveries(ctx context.Context, in ListDeliveriesInput) ([]*WebhookDelivery, int, error) {
	limit, offset := normalizePaging(in.Limit, in.Offset)
	filter := ListDeliveriesFilter{
		WebhookID:    in.WebhookID,
		SubmissionID: in.SubmissionID,
		Status:       in.Status,
		Limit:        limit,
		Offset:       offset,
	}
	return s.repo.ListWebhookDeliveries(ctx, in.TenantID, filter)
}

// GetWebhookDelivery retrieves a single webhook delivery by ID.
func (s *Service) GetWebhookDelivery(ctx context.Context, id, tenantID uuid.UUID) (*WebhookDelivery, error) {
	return s.repo.GetWebhookDelivery(ctx, id, tenantID)
}

// ============================================================================
// Stats methods
// ============================================================================

// GetFormStatsInput drives a form stats lookup.
type GetFormStatsInput struct {
	TenantID     uuid.UUID
	FormSchemaID uuid.UUID
}

// GetFormStats returns aggregated submission statistics for a form schema.
func (s *Service) GetFormStats(ctx context.Context, in GetFormStatsInput) (*FormStats, error) {
	if in.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required: %w", ErrInvalidFields)
	}
	if in.FormSchemaID == uuid.Nil {
		return nil, fmt.Errorf("form_schema_id is required: %w", ErrInvalidFields)
	}
	return s.repo.GetFormStats(ctx, in.FormSchemaID, in.TenantID)
}
