package berichte

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"golang.org/x/crypto/bcrypt"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// ============================================================================
// Collaborators
// ============================================================================

// Executor runs the aggregation for a given definition and produces a
// ReportResult. Implemented in the executor subpackage; the service only
// depends on the interface so tests can mock it.
type Executor interface {
	Run(ctx context.Context, def *Definition, params json.RawMessage) (*ReportResult, error)
	DashboardKPIs(ctx context.Context, tenantID uuid.UUID, modules []string) ([]KPI, error)
}

// Clock abstracts time.Now so tests can inject fixed clocks.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// ============================================================================
// Service
// ============================================================================

// Service holds berichte business logic. Thick-service pattern: handlers
// (HTTP, gRPC) translate transport shapes into inputs and call through.
type Service struct {
	repo       Repository
	executor   Executor
	clock      Clock
	cacheTTL   time.Duration
	cronParser cron.Parser
}

// Options tweak the service behaviour. Zero values fall back to sensible
// defaults (real clock, 15-minute cache TTL, standard 5-field cron).
type Options struct {
	Clock    Clock
	CacheTTL time.Duration
}

// NewService creates a berichte service. The executor may be nil; in that
// case RunReport and GetDashboardKPIs return ErrExecutorUnavailable.
func NewService(repo Repository, executor Executor, opts Options) *Service {
	clk := opts.Clock
	if clk == nil {
		clk = realClock{}
	}
	ttl := opts.CacheTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return &Service{
		repo:       repo,
		executor:   executor,
		clock:      clk,
		cacheTTL:   ttl,
		cronParser: parser,
	}
}

// ============================================================================
// Valid enum sets
// ============================================================================

var (
	validModules = map[string]struct{}{
		"finanzen": {}, "crm": {}, "helpdesk": {}, "inventar": {},
		"produktion": {}, "cross": {},
	}
	validKinds   = map[string]struct{}{"system": {}, "custom": {}}
	validFormats = map[string]struct{}{"pdf": {}, "csv": {}, "xlsx": {}}
)

func isValidModule(m string) bool { _, ok := validModules[m]; return ok }
func isValidKind(k string) bool   { _, ok := validKinds[k]; return ok }
func isValidFormat(f string) bool { _, ok := validFormats[f]; return ok }

// ============================================================================
// Inputs
// ============================================================================

// CreateDefinitionInput captures data for creating a report definition.
type CreateDefinitionInput struct {
	TenantID      uuid.UUID
	Name          string
	Description   string
	Module        string
	Kind          string // "" defaults to "custom"
	QueryConfig   json.RawMessage
	DefaultFormat string // "" defaults to "pdf"
	CreatedBy     *uuid.UUID
	IsPublished   bool
}

// UpdateDefinitionInput captures partial updates on a definition.
type UpdateDefinitionInput struct {
	TenantID      uuid.UUID
	DefinitionID  uuid.UUID
	Name          *string
	Description   *string
	Module        *string
	QueryConfig   json.RawMessage // empty = no change
	DefaultFormat *string
	IsPublished   *bool
}

// ListDefinitionsInput is the user-facing list request.
type ListDefinitionsInput struct {
	TenantID    uuid.UUID
	Module      *string
	Kind        *string
	IsPublished *bool
	Search      string
	Page        int
	PageSize    int
	SortBy      string
	SortDesc    bool
}

// RunReportInput drives cache-aware report execution.
type RunReportInput struct {
	TenantID     uuid.UUID
	DefinitionID uuid.UUID
	Params       json.RawMessage
	ForceRefresh bool
	Trigger      string // defaults to "manual"
}

// CreateScheduleInput captures data for creating a schedule.
type CreateScheduleInput struct {
	TenantID       uuid.UUID
	DefinitionID   uuid.UUID
	Name           string
	CronExpression string
	Recipients     []string
	Format         string
	Params         json.RawMessage
	Active         bool
	CreatedBy      *uuid.UUID
}

// UpdateScheduleInput captures partial updates on a schedule.
// RecipientsSet=true replaces the recipients list; false keeps the current one.
type UpdateScheduleInput struct {
	TenantID       uuid.UUID
	ScheduleID     uuid.UUID
	Name           *string
	CronExpression *string
	Recipients     []string
	RecipientsSet  bool
	Format         *string
	Params         json.RawMessage // empty = no change
	Active         *bool
}

// ListSchedulesInput is the user-facing schedule list request.
type ListSchedulesInput struct {
	TenantID     uuid.UUID
	DefinitionID *uuid.UUID
	Active       *bool
	Page         int
	PageSize     int
}

// ============================================================================
// Definitions
// ============================================================================

func (s *Service) CreateDefinition(ctx context.Context, in CreateDefinitionInput) (*Definition, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrInvalidQueryConfig
	}
	if !isValidModule(in.Module) {
		return nil, ErrInvalidModule
	}

	kind := in.Kind
	if kind == "" {
		kind = "custom"
	}
	if !isValidKind(kind) {
		return nil, ErrInvalidKind
	}

	format := in.DefaultFormat
	if format == "" {
		format = "pdf"
	}
	if !isValidFormat(format) {
		return nil, ErrInvalidFormat
	}

	queryConfig := normalizeJSONB(in.QueryConfig)
	if !json.Valid(queryConfig) {
		return nil, ErrInvalidQueryConfig
	}

	now := s.clock.Now()
	def := &Definition{
		ID:            uuid.New(),
		TenantID:      in.TenantID,
		Name:          name,
		Description:   strings.TrimSpace(in.Description),
		Module:        in.Module,
		Kind:          kind,
		QueryConfig:   queryConfig,
		DefaultFormat: format,
		CreatedBy:     in.CreatedBy,
		IsPublished:   in.IsPublished,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.CreateDefinition(ctx, def); err != nil {
		return nil, fmt.Errorf("create definition: %w", err)
	}

	slog.Info("berichte definition created",
		"definition_id", def.ID,
		"tenant_id", def.TenantID,
		"module", def.Module,
		"kind", def.Kind,
	)
	return def, nil
}

func (s *Service) UpdateDefinition(ctx context.Context, in UpdateDefinitionInput) (*Definition, error) {
	def, err := s.repo.GetDefinition(ctx, in.TenantID, in.DefinitionID)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return nil, ErrInvalidQueryConfig
		}
		def.Name = n
	}
	if in.Description != nil {
		def.Description = strings.TrimSpace(*in.Description)
	}
	if in.Module != nil {
		if !isValidModule(*in.Module) {
			return nil, ErrInvalidModule
		}
		def.Module = *in.Module
	}
	if len(in.QueryConfig) > 0 {
		if !json.Valid(in.QueryConfig) {
			return nil, ErrInvalidQueryConfig
		}
		def.QueryConfig = append([]byte(nil), in.QueryConfig...)
	}
	if in.DefaultFormat != nil {
		if !isValidFormat(*in.DefaultFormat) {
			return nil, ErrInvalidFormat
		}
		def.DefaultFormat = *in.DefaultFormat
	}
	if in.IsPublished != nil {
		def.IsPublished = *in.IsPublished
	}

	def.UpdatedAt = s.clock.Now()

	if err := s.repo.UpdateDefinition(ctx, def); err != nil {
		return nil, err
	}

	slog.Info("berichte definition updated",
		"definition_id", def.ID,
		"tenant_id", def.TenantID,
	)
	return def, nil
}

func (s *Service) DeleteDefinition(ctx context.Context, tenantID, definitionID uuid.UUID) error {
	if err := s.repo.DeleteDefinition(ctx, tenantID, definitionID); err != nil {
		return err
	}
	slog.Info("berichte definition deleted",
		"definition_id", definitionID,
		"tenant_id", tenantID,
	)
	return nil
}

func (s *Service) GetDefinition(ctx context.Context, tenantID, definitionID uuid.UUID) (*Definition, error) {
	return s.repo.GetDefinition(ctx, tenantID, definitionID)
}

func (s *Service) ListDefinitions(ctx context.Context, in ListDefinitionsInput) ([]*Definition, int, error) {
	page, pageSize := normalizePaging(in.Page, in.PageSize)
	filter := ListDefinitionsFilter{
		Module:      in.Module,
		Kind:        in.Kind,
		IsPublished: in.IsPublished,
		Search:      in.Search,
		SortBy:      in.SortBy,
		SortDesc:    in.SortDesc,
	}
	offset := (page - 1) * pageSize
	return s.repo.ListDefinitions(ctx, in.TenantID, filter, offset, pageSize)
}

// ============================================================================
// Run & Cache
// ============================================================================

// RunReport resolves a definition, returns a cached result when fresh, or
// delegates to the executor and caches the fresh result. A matching Run
// record is always inserted so auditing sees every execution.
func (s *Service) RunReport(ctx context.Context, in RunReportInput) (*ReportResult, *Run, error) {
	if s.executor == nil {
		return nil, nil, ErrExecutorUnavailable
	}

	def, err := s.repo.GetDefinition(ctx, in.TenantID, in.DefinitionID)
	if err != nil {
		return nil, nil, err
	}

	params := normalizeJSONB(in.Params)
	hash := hashParams(params)

	trigger := in.Trigger
	if trigger == "" {
		trigger = "manual"
	}
	if trigger != "manual" && trigger != "scheduled" && trigger != "api" {
		trigger = "manual"
	}

	// Cache lookup
	if !in.ForceRefresh {
		if entry, cErr := s.repo.GetCacheEntry(ctx, in.TenantID, in.DefinitionID, hash); cErr == nil {
			if entry.ExpiresAt.After(s.clock.Now()) {
				var cached ReportResult
				jErr := json.Unmarshal(entry.Result, &cached)
				if jErr == nil {
					cached.Meta.FromCache = true
					run := s.recordRun(ctx, def, in, trigger, params, cached.Meta.RowCount, 0, "success", nil)
					return &cached, run, nil
				}
				slog.Warn("berichte cache unmarshal failed, falling back to executor",
					"definition_id", def.ID, "error", jErr)
			}
		}
	}

	started := s.clock.Now()
	result, execErr := s.executor.Run(ctx, def, params)
	durationMs := int(s.clock.Now().Sub(started) / time.Millisecond)

	if execErr != nil {
		errStr := execErr.Error()
		_ = s.insertRun(ctx, &Run{
			ID:           uuid.New(),
			TenantID:     in.TenantID,
			DefinitionID: def.ID,
			Trigger:      trigger,
			Params:       params,
			DurationMs:   &durationMs,
			Status:       "failed",
			Error:        &errStr,
			StartedAt:    started,
			CompletedAt:  pointerTime(s.clock.Now()),
		})
		return nil, nil, fmt.Errorf("execute report: %w", execErr)
	}

	if result.Meta.GeneratedAt.IsZero() {
		result.Meta.GeneratedAt = s.clock.Now()
	}
	if result.Meta.DefinitionID == uuid.Nil {
		result.Meta.DefinitionID = def.ID
	}
	if result.Meta.RowCount == 0 {
		result.Meta.RowCount = len(result.Rows)
	}

	// Cache the fresh result. Marshalling failures only skip caching; the
	// request still succeeds.
	if payload, jErr := json.Marshal(result); jErr == nil {
		_ = s.repo.UpsertCacheEntry(ctx, &CacheEntry{
			ID:           uuid.New(),
			TenantID:     in.TenantID,
			DefinitionID: def.ID,
			ParamsHash:   hash,
			Result:       payload,
			RowCount:     result.Meta.RowCount,
			ComputedAt:   s.clock.Now(),
			ExpiresAt:    s.clock.Now().Add(s.cacheTTL),
		})
	}

	run := s.recordRun(ctx, def, in, trigger, params, result.Meta.RowCount, durationMs, "success", nil)
	return result, run, nil
}

// GetCachedResult returns a cached result or ErrCacheMiss. Does not fall
// through to the executor.
func (s *Service) GetCachedResult(ctx context.Context, tenantID, definitionID uuid.UUID, params json.RawMessage) (*ReportResult, error) {
	normalized := normalizeJSONB(params)
	hash := hashParams(normalized)
	entry, err := s.repo.GetCacheEntry(ctx, tenantID, definitionID, hash)
	if err != nil {
		return nil, err
	}
	if !entry.ExpiresAt.After(s.clock.Now()) {
		return nil, ErrCacheMiss
	}
	var res ReportResult
	if jErr := json.Unmarshal(entry.Result, &res); jErr != nil {
		return nil, fmt.Errorf("decode cached result: %w", jErr)
	}
	res.Meta.FromCache = true
	return &res, nil
}

func (s *Service) InvalidateCache(ctx context.Context, tenantID, definitionID uuid.UUID) (int, error) {
	evicted, err := s.repo.InvalidateCache(ctx, tenantID, definitionID)
	if err != nil {
		return 0, err
	}
	slog.Info("berichte cache invalidated",
		"definition_id", definitionID,
		"tenant_id", tenantID,
		"evicted", evicted,
	)
	return evicted, nil
}

// ============================================================================
// Schedules
// ============================================================================

func (s *Service) CreateSchedule(ctx context.Context, in CreateScheduleInput) (*Schedule, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrInvalidQueryConfig
	}
	if _, err := s.cronParser.Parse(in.CronExpression); err != nil {
		return nil, ErrInvalidCron
	}

	format := in.Format
	if format == "" {
		format = "pdf"
	}
	if !isValidFormat(format) {
		return nil, ErrInvalidFormat
	}

	// Ensure the definition exists and belongs to the same tenant.
	if _, err := s.repo.GetDefinition(ctx, in.TenantID, in.DefinitionID); err != nil {
		return nil, err
	}

	recipients := sanitizeRecipients(in.Recipients)
	params := normalizeJSONB(in.Params)

	now := s.clock.Now()
	sch := &Schedule{
		ID:             uuid.New(),
		TenantID:       in.TenantID,
		DefinitionID:   in.DefinitionID,
		Name:           name,
		CronExpression: in.CronExpression,
		Recipients:     recipients,
		Format:         format,
		Params:         params,
		Active:         in.Active,
		CreatedBy:      in.CreatedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.CreateSchedule(ctx, sch); err != nil {
		return nil, fmt.Errorf("create schedule: %w", err)
	}
	slog.Info("berichte schedule created",
		"schedule_id", sch.ID,
		"definition_id", sch.DefinitionID,
		"tenant_id", sch.TenantID,
	)
	return sch, nil
}

func (s *Service) UpdateSchedule(ctx context.Context, in UpdateScheduleInput) (*Schedule, error) {
	sch, err := s.repo.GetSchedule(ctx, in.TenantID, in.ScheduleID)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return nil, ErrInvalidQueryConfig
		}
		sch.Name = n
	}
	if in.CronExpression != nil {
		if _, parseErr := s.cronParser.Parse(*in.CronExpression); parseErr != nil {
			return nil, ErrInvalidCron
		}
		sch.CronExpression = *in.CronExpression
	}
	if in.RecipientsSet {
		sch.Recipients = sanitizeRecipients(in.Recipients)
	}
	if in.Format != nil {
		if !isValidFormat(*in.Format) {
			return nil, ErrInvalidFormat
		}
		sch.Format = *in.Format
	}
	if len(in.Params) > 0 {
		if !json.Valid(in.Params) {
			return nil, ErrInvalidQueryConfig
		}
		sch.Params = append([]byte(nil), in.Params...)
	}
	if in.Active != nil {
		sch.Active = *in.Active
	}

	sch.UpdatedAt = s.clock.Now()
	if err := s.repo.UpdateSchedule(ctx, sch); err != nil {
		return nil, err
	}
	slog.Info("berichte schedule updated",
		"schedule_id", sch.ID,
		"tenant_id", sch.TenantID,
	)
	return sch, nil
}

func (s *Service) DeleteSchedule(ctx context.Context, tenantID, scheduleID uuid.UUID) error {
	if err := s.repo.DeleteSchedule(ctx, tenantID, scheduleID); err != nil {
		return err
	}
	slog.Info("berichte schedule deleted",
		"schedule_id", scheduleID,
		"tenant_id", tenantID,
	)
	return nil
}

func (s *Service) ListSchedules(ctx context.Context, in ListSchedulesInput) ([]*Schedule, int, error) {
	page, pageSize := normalizePaging(in.Page, in.PageSize)
	filter := ListSchedulesFilter{
		DefinitionID: in.DefinitionID,
		Active:       in.Active,
	}
	offset := (page - 1) * pageSize
	return s.repo.ListSchedules(ctx, in.TenantID, filter, offset, pageSize)
}

func (s *Service) ToggleSchedule(ctx context.Context, tenantID, scheduleID uuid.UUID, active bool) (*Schedule, error) {
	sch, err := s.repo.GetSchedule(ctx, tenantID, scheduleID)
	if err != nil {
		return nil, err
	}
	sch.Active = active
	sch.UpdatedAt = s.clock.Now()
	if err := s.repo.UpdateSchedule(ctx, sch); err != nil {
		return nil, err
	}
	slog.Info("berichte schedule toggled",
		"schedule_id", sch.ID,
		"active", active,
	)
	return sch, nil
}

// ValidateCron exposes the service's cron parser for transport-layer
// pre-checks without instantiating a duplicate parser.
func (s *Service) ValidateCron(expr string) error {
	if _, err := s.cronParser.Parse(expr); err != nil {
		return ErrInvalidCron
	}
	return nil
}

// ============================================================================
// Dashboard KPIs
// ============================================================================

func (s *Service) GetDashboardKPIs(ctx context.Context, tenantID uuid.UUID, modules []string) ([]KPI, error) {
	if s.executor == nil {
		return nil, ErrExecutorUnavailable
	}
	mods := sanitizeModules(modules)
	return s.executor.DashboardKPIs(ctx, tenantID, mods)
}

// ============================================================================
// Documents (multi-page authoring)
// ============================================================================

const (
	maxDocumentTitleLen = 200
	// lean: one flat size cap for the whole block tree instead of per-block
	// limits; revisit when a real report hits it (the editor has no cap today).
	maxDocumentPayloadBytes = 4 << 20 // 4 MiB
)

var validDocStatuses = map[string]struct{}{
	"draft": {}, "final": {}, "released": {}, "archived": {},
}

func isValidDocStatus(s string) bool { _, ok := validDocStatuses[s]; return ok }

// CreateDocumentInput captures data for creating a report document.
type CreateDocumentInput struct {
	TenantID    uuid.UUID
	Title       string // "" defaults to "Neuer Bericht"
	Description string
	Module      string // "" defaults to "cross"
	TemplateID  *string
	Rows        json.RawMessage // empty defaults to []
	Settings    json.RawMessage // empty defaults to {}
	CreatedBy   *uuid.UUID
}

// UpdateDocumentInput captures partial updates on a report document.
type UpdateDocumentInput struct {
	TenantID    uuid.UUID
	DocumentID  uuid.UUID
	Title       *string
	Description *string
	Module      *string
	Status      *string
	Rows        json.RawMessage // empty = no change
	Settings    json.RawMessage // empty = no change
}

// ListDocumentsInput is the user-facing document list request.
type ListDocumentsInput struct {
	TenantID uuid.UUID
	Module   *string
	Status   *string
	Search   string
	Page     int
	PageSize int
}

func (s *Service) CreateDocument(ctx context.Context, in CreateDocumentInput) (*Document, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "Neuer Bericht"
	}
	if len(title) > maxDocumentTitleLen {
		return nil, ErrInvalidTitle
	}

	module := in.Module
	if module == "" {
		module = "cross"
	}
	if !isValidModule(module) {
		return nil, ErrInvalidModule
	}

	rows, err := normalizeDocumentRows(in.Rows)
	if err != nil {
		return nil, err
	}
	settings, err := normalizeDocumentSettings(in.Settings)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	doc := &Document{
		ID:          uuid.New(),
		TenantID:    in.TenantID,
		Title:       title,
		Description: strings.TrimSpace(in.Description),
		Module:      module,
		Status:      "draft",
		Rows:        rows,
		Settings:    settings,
		TemplateID:  in.TemplateID,
		CreatedBy:   in.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}

	slog.Info("berichte document created",
		"document_id", doc.ID,
		"tenant_id", doc.TenantID,
		"module", doc.Module,
	)
	return doc, nil
}

func (s *Service) UpdateDocument(ctx context.Context, in UpdateDocumentInput) (*Document, error) {
	doc, err := s.repo.GetDocument(ctx, in.TenantID, in.DocumentID)
	if err != nil {
		return nil, err
	}

	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" || len(t) > maxDocumentTitleLen {
			return nil, ErrInvalidTitle
		}
		doc.Title = t
	}
	if in.Description != nil {
		doc.Description = strings.TrimSpace(*in.Description)
	}
	if in.Module != nil {
		if !isValidModule(*in.Module) {
			return nil, ErrInvalidModule
		}
		doc.Module = *in.Module
	}
	if in.Status != nil {
		if !isValidDocStatus(*in.Status) {
			return nil, ErrInvalidStatus
		}
		doc.Status = *in.Status
	}
	if len(in.Rows) > 0 {
		rows, rowsErr := normalizeDocumentRows(in.Rows)
		if rowsErr != nil {
			return nil, rowsErr
		}
		doc.Rows = rows
	}
	if len(in.Settings) > 0 {
		settings, settingsErr := normalizeDocumentSettings(in.Settings)
		if settingsErr != nil {
			return nil, settingsErr
		}
		doc.Settings = settings
	}

	now := s.clock.Now()
	doc.UpdatedAt = now
	// Stamp the release date once, on the first transition into "released".
	if doc.Status == "released" && doc.ReleasedAt == nil {
		doc.ReleasedAt = pointerTime(now)
	}

	if err := s.repo.UpdateDocument(ctx, doc); err != nil {
		return nil, err
	}

	slog.Info("berichte document updated",
		"document_id", doc.ID,
		"tenant_id", doc.TenantID,
		"status", doc.Status,
	)
	return doc, nil
}

func (s *Service) GetDocument(ctx context.Context, tenantID, documentID uuid.UUID) (*Document, error) {
	return s.repo.GetDocument(ctx, tenantID, documentID)
}

func (s *Service) DeleteDocument(ctx context.Context, tenantID, documentID uuid.UUID) error {
	if err := s.repo.DeleteDocument(ctx, tenantID, documentID); err != nil {
		return err
	}
	slog.Info("berichte document deleted", "document_id", documentID, "tenant_id", tenantID)
	return nil
}

func (s *Service) ListDocuments(ctx context.Context, in ListDocumentsInput) ([]*Document, int, error) {
	if in.Module != nil && !isValidModule(*in.Module) {
		return nil, 0, ErrInvalidModule
	}
	if in.Status != nil && !isValidDocStatus(*in.Status) {
		return nil, 0, ErrInvalidStatus
	}

	page, pageSize := in.Page, in.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return s.repo.ListDocuments(ctx, in.TenantID, ListDocumentsFilter{
		Module: in.Module,
		Status: in.Status,
		Search: strings.TrimSpace(in.Search),
	}, (page-1)*pageSize, pageSize)
}

// ListTemplates returns the static starter templates for "Neuer Bericht aus
// Vorlage" (see templates_data.go). Templates are frontend-owned starter
// content, not tenant data, so this reads no repository and needs no
// tenant scoping.
func (s *Service) ListTemplates() []Template {
	return demoTemplates
}

// normalizeDocumentRows validates the opaque block tree: it must be a JSON
// array within the size cap. Block semantics stay frontend-owned.
func normalizeDocumentRows(raw json.RawMessage) ([]byte, error) {
	return normalizeDocumentJSON(raw, '[', "[]", ErrInvalidRows)
}

// normalizeDocumentSettings validates the document-level page setup: it must be
// a JSON object within the size cap.
func normalizeDocumentSettings(raw json.RawMessage) ([]byte, error) {
	return normalizeDocumentJSON(raw, '{', "{}", ErrInvalidSettings)
}

func normalizeDocumentJSON(raw json.RawMessage, prefix byte, empty string, shapeErr error) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return []byte(empty), nil
	}
	if len(trimmed) > maxDocumentPayloadBytes {
		return nil, ErrDocumentTooLarge
	}
	if trimmed[0] != prefix || !json.Valid(trimmed) {
		return nil, shapeErr
	}
	return append([]byte(nil), trimmed...), nil
}

// ============================================================================
// Helpers
// ============================================================================

// normalizeJSONB returns a JSON-valid byte slice. Empty input becomes `{}`.
func normalizeJSONB(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	return append([]byte(nil), raw...)
}

// hashParams produces a stable sha256 hex for a params payload. Empty input
// normalises to `{}` so cache lookups from different encodings of "no params"
// collapse to the same entry.
func hashParams(params []byte) string {
	if len(params) == 0 {
		params = []byte("{}")
	}
	sum := sha256.Sum256(params)
	return hex.EncodeToString(sum[:])
}

func normalizePaging(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	return page, pageSize
}

func sanitizeRecipients(in []string) []string {
	out := make([]string, 0, len(in))
	for _, r := range in {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		out = append(out, r)
	}
	return out
}

func sanitizeModules(in []string) []string {
	out := make([]string, 0, len(in))
	for _, m := range in {
		m = strings.TrimSpace(m)
		if m == "" || !isValidModule(m) {
			continue
		}
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

func (s *Service) recordRun(ctx context.Context, def *Definition, in RunReportInput, trigger string, params []byte, rowCount, durationMs int, status string, errStr *string) *Run {
	run := &Run{
		ID:           uuid.New(),
		TenantID:     in.TenantID,
		DefinitionID: def.ID,
		Trigger:      trigger,
		Params:       params,
		DurationMs:   &durationMs,
		RowCount:     &rowCount,
		Status:       status,
		Error:        errStr,
		StartedAt:    s.clock.Now(),
		CompletedAt:  pointerTime(s.clock.Now()),
	}
	_ = s.insertRun(ctx, run)
	return run
}

func (s *Service) insertRun(ctx context.Context, run *Run) error {
	if err := s.repo.InsertRun(ctx, run); err != nil {
		slog.Warn("berichte failed to persist run", "run_id", run.ID, "error", err)
		return err
	}
	return nil
}

func pointerTime(t time.Time) *time.Time { return &t }

// ============================================================================
// Share tokens (external, unauthenticated read links)
// ============================================================================

const (
	// shareTokenBytes is the entropy behind a share link. 32 bytes is not a
	// round number picked for looks: the public route answers on the token
	// alone, so the token is the entire access control. At 256 bits a guessing
	// attack is not slowed by the rate limit, it is impossible.
	shareTokenBytes = 32
	// maxSharePasswordLen caps what is fed to bcrypt. bcrypt silently ignores
	// input past 72 bytes, so an uncapped field would accept two different
	// passwords as the same one.
	maxSharePasswordLen = 72
	// maxShareExpiryDays bounds a link's lifetime. An external read path that
	// outlives anyone's memory of creating it is the failure mode here.
	maxShareExpiryDays = 365
	// shareBcryptCost matches internal/auth.
	shareBcryptCost = 12
)

// CreateShareTokenInput is the request to hand out an external read link.
type CreateShareTokenInput struct {
	TenantID      uuid.UUID
	DocumentID    uuid.UUID
	ExpiresInDays *int32 // nil or 0 = never expires
	Password      string // empty = no password
	CreatedBy     *uuid.UUID
}

// CreateShareToken issues an external read link for one document.
//
// The document is read first, tenant-scoped: a link may only ever be minted for
// a document the caller's own tenant owns, and that read is what proves it.
// Without it a caller could name any UUID and get back a working public link to
// another tenant's report.
func (s *Service) CreateShareToken(ctx context.Context, in CreateShareTokenInput) (*ShareToken, error) {
	if _, err := s.repo.GetDocument(ctx, in.TenantID, in.DocumentID); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	token := &ShareToken{
		ID:         uuid.New(),
		TenantID:   in.TenantID,
		DocumentID: in.DocumentID,
		CreatedBy:  in.CreatedBy,
		CreatedAt:  now,
	}

	if in.ExpiresInDays != nil && *in.ExpiresInDays != 0 {
		days := *in.ExpiresInDays
		if days < 0 || days > maxShareExpiryDays {
			return nil, ErrInvalidShareExpiry
		}
		expiry := now.AddDate(0, 0, int(days))
		token.ExpiresAt = &expiry
	}

	if in.Password != "" {
		if len(in.Password) > maxSharePasswordLen {
			return nil, ErrSharePasswordTooLong
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), shareBcryptCost)
		if err != nil {
			return nil, fmt.Errorf("hash share password: %w", err)
		}
		hashStr := string(hash)
		token.PasswordHash = &hashStr
	}

	secret, err := newShareSecret()
	if err != nil {
		return nil, err
	}
	token.Token = secret

	if err := s.repo.CreateShareToken(ctx, token); err != nil {
		return nil, fmt.Errorf("create share token: %w", err)
	}

	// The secret itself is never logged: this line would otherwise be a
	// working public link sitting in the log pipeline.
	slog.Info("berichte share link created",
		"share_id", token.ID,
		"document_id", token.DocumentID,
		"tenant_id", token.TenantID,
		"has_password", token.PasswordHash != nil,
		"expires_at", token.ExpiresAt,
	)
	return token, nil
}

func (s *Service) ListShareTokens(ctx context.Context, tenantID, documentID uuid.UUID) ([]*ShareToken, error) {
	// Same reason as CreateShareToken: prove the document is ours before
	// listing anything attached to it.
	if _, err := s.repo.GetDocument(ctx, tenantID, documentID); err != nil {
		return nil, err
	}
	return s.repo.ListShareTokens(ctx, tenantID, documentID)
}

func (s *Service) RevokeShareToken(ctx context.Context, tenantID, shareID uuid.UUID) error {
	if err := s.repo.RevokeShareToken(ctx, tenantID, shareID, s.clock.Now()); err != nil {
		return err
	}
	slog.Info("berichte share link revoked", "share_id", shareID, "tenant_id", tenantID)
	return nil
}

// GetSharedDocument serves the unauthenticated public read.
//
// This is the only path in the service that starts without a tenant, so the
// order of the checks is the security property, not a style choice:
//
//  1. resolve the link by its secret in the system context — the one read that
//     must escape RLS, because which tenant may be seen is exactly what it
//     answers;
//  2. refuse revoked and expired links before anything else, all as the same
//     "not found" a wholly unknown token gets;
//  3. verify the password against the stored bcrypt hash;
//  4. only then re-enter tenant scope and read the one document the link names.
//
// Step 4 reads through GetDocument with the resolved tenant rather than
// trusting the link's DocumentID alone, so RLS still has the final say and the
// path can reach exactly one row: the shared report, never a listing.
func (s *Service) GetSharedDocument(ctx context.Context, secret, password string) (*Document, error) {
	share, err := s.repo.GetShareTokenBySecret(ctx, secret)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	if !share.Usable(now) {
		slog.Info("berichte share link refused",
			"share_id", share.ID,
			"revoked", share.RevokedAt != nil,
			"expired", share.ExpiresAt != nil && !now.Before(*share.ExpiresAt),
		)
		return nil, ErrShareNotFound
	}

	if share.PasswordHash != nil {
		if password == "" {
			return nil, ErrSharePasswordRequired
		}
		// bcrypt's own comparison is the constant-time one; a byte-wise
		// comparison of the hashes here would be the timing leak.
		if bcrypt.CompareHashAndPassword([]byte(*share.PasswordHash), []byte(password)) != nil {
			slog.Warn("berichte share link password rejected", "share_id", share.ID)
			return nil, ErrSharePasswordInvalid
		}
	}

	ctx = WithTenant(ctx, share.TenantID)
	doc, err := s.repo.GetDocument(ctx, share.TenantID, share.DocumentID)
	if err != nil {
		// A link whose document is gone is a dead link, not a server fault.
		if errors.Is(err, ErrDocumentNotFound) {
			return nil, ErrShareNotFound
		}
		return nil, err
	}

	// Counting the view must not fail the read the visitor already earned.
	if err := s.repo.IncrementShareView(ctx, share.TenantID, share.ID); err != nil {
		slog.Warn("berichte share view not counted", "share_id", share.ID, "error", err)
	}
	return doc, nil
}

// WithTenant attaches a tenant resolved from a share link to the context, so
// the repository and the RLS session GUCs see it. This is the single place a
// public path may set a tenant, and it is only ever reached with the tenant the
// link itself resolved to.
func WithTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
}

// newShareSecret draws a link secret from crypto/rand. base64url keeps it
// usable in a URL path segment without escaping.
func newShareSecret() (string, error) {
	buf := make([]byte, shareTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate share token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
