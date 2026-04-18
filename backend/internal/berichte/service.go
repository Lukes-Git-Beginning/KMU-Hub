package berichte

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
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
