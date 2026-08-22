package gdpr

// Retention engine — DSGVO Art. 5(1)(e) Speicherbegrenzung.
//
// `retention_policies` (Migration 000233) has been editable through the UI
// since it was added, but nothing ever executed a policy. A policy that
// deletes nothing still reads like a fulfilled deletion duty in an audit,
// which is worse than having no policy at all. This file is the missing
// executor: a registry that maps `resource_type` onto a handler, and an
// engine that walks the enabled policies of a tenant and writes down what it
// found and what it did.
//
// Two rules run through the whole file:
//
//  1. Dry run is the default. Only an explicit RetentionModeEnforce touches
//     data. A wrong mapping in enforce mode is not recoverable, so arming is
//     a deliberate act, never a fallback.
//  2. A `resource_type` without a handler is reported as "nicht zugeordnet",
//     never skipped in silence. Otherwise a policy for "kontakte" quietly
//     does nothing while the registry knows "contacts".

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// retentionMaxSkipReasons bounds how many per-record reasons a run item
// stores. The count stays exact either way; only the reason list is cut.
//
// lean: first N reasons per policy; give skips their own table if a run ever
// needs the complete list.
const retentionMaxSkipReasons = 50

// RetentionMode decides whether a run touches data.
type RetentionMode string

const (
	// RetentionModeDryRun counts what would happen and changes nothing.
	RetentionModeDryRun RetentionMode = "dry_run"
	// RetentionModeEnforce actually deletes or anonymizes.
	RetentionModeEnforce RetentionMode = "enforce"
)

// Run item statuses, mirrored by the CHECK constraint on retention_run_items.
const (
	// RetentionItemDryRun — counted, nothing changed.
	RetentionItemDryRun = "dry_run"
	// RetentionItemApplied — the handler carried the action out.
	RetentionItemApplied = "applied"
	// RetentionItemUnmapped — no handler registered for this resource_type.
	RetentionItemUnmapped = "unmapped"
	// RetentionItemUnsupported — handler exists but cannot do this action.
	RetentionItemUnsupported = "unsupported"
	// RetentionItemFailed — the handler returned an error.
	RetentionItemFailed = "failed"
)

// ParseRetentionMode maps a configured value onto a mode. Anything that is
// not exactly "enforce" — empty, misspelled, unknown — resolves to dry run.
// The second return value is false when the input was not recognised, so the
// caller can log the typo instead of silently running disarmed.
func ParseRetentionMode(s string) (RetentionMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(RetentionModeEnforce):
		return RetentionModeEnforce, true
	case string(RetentionModeDryRun), "":
		return RetentionModeDryRun, true
	default:
		return RetentionModeDryRun, false
	}
}

// RetentionSkip is one record the handler refused to process, with the reason
// it gave. A skipped record without a reason answers none of the questions
// that get asked when somebody audits a retention run.
type RetentionSkip struct {
	RecordID uuid.UUID `json:"record_id"`
	Reason   string    `json:"reason"`
}

// RetentionPlan is what a handler found for one policy: the records whose
// retention period elapsed and may be acted on, plus the ones it will not
// touch.
type RetentionPlan struct {
	Due     []uuid.UUID
	Skipped []RetentionSkip
}

// RetentionHandler executes retention for exactly one resource_type.
//
// Implementations must be idempotent: after a successful Apply, a following
// Plan with the same cutoff must no longer return those records. For deletion
// that is automatic; an anonymizing handler has to exclude the records it
// already anonymized, otherwise every run rewrites the same rows forever.
type RetentionHandler interface {
	// ResourceType is the value a policy carries in retention_policies.
	ResourceType() string

	// Table and DateColumn disclose the mapping this handler stands for.
	// They exist so an administrator can see which table a policy governs and
	// which column starts the clock — "created_at" and "last activity" are
	// very different retention periods.
	Table() string
	DateColumn() string

	// SupportsAction reports whether this handler can carry out the policy's
	// action (models.RetentionActionDelete / ...Anonymize). A handler that
	// cannot anonymize must say so instead of deleting instead.
	SupportsAction(action string) bool

	// Plan lists the records older than cutoff. action is the policy's
	// configured action (models.RetentionActionDelete / ...Anonymize) — a
	// handler whose delete path is blocked for some records (e.g. by a
	// RESTRICT foreign key) needs to know which action is in play to decide
	// whether those records belong in Due or Skipped; the same records can be
	// perfectly fine for anonymize. Plan must not modify data.
	Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, action string) (*RetentionPlan, error)

	// Apply performs action on the given records and returns how many were
	// affected. Only ever called in enforce mode.
	Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, anonymizedLabel string) (int, error)
}

// RetentionRegistry maps resource_type onto its handler. It is deliberately
// explicit and empty by default: every policy without a registered handler
// shows up as "nicht zugeordnet" until somebody wires one.
type RetentionRegistry struct {
	handlers map[string]RetentionHandler
}

// NewRetentionRegistry builds a registry from the given handlers.
func NewRetentionRegistry(handlers ...RetentionHandler) *RetentionRegistry {
	r := &RetentionRegistry{handlers: make(map[string]RetentionHandler, len(handlers))}
	for _, h := range handlers {
		r.Register(h)
	}
	return r
}

// Register adds a handler, replacing any earlier one for the same
// resource_type.
func (r *RetentionRegistry) Register(h RetentionHandler) {
	if h == nil {
		return
	}
	if r.handlers == nil {
		r.handlers = make(map[string]RetentionHandler)
	}
	r.handlers[h.ResourceType()] = h
}

// Lookup returns the handler for a resource_type. The comparison is exact —
// no trimming, no case folding, no guessing. A policy for "Contacts " is not
// a policy for "contacts", and pretending otherwise would delete rows nobody
// asked for.
func (r *RetentionRegistry) Lookup(resourceType string) (RetentionHandler, bool) {
	h, ok := r.handlers[resourceType]
	return h, ok
}

// ResourceTypes lists every mapped resource_type, sorted. Feeds the admin
// view so an operator can compare it against the policies they created.
func (r *RetentionRegistry) ResourceTypes() []string {
	out := make([]string, 0, len(r.handlers))
	for k := range r.handlers {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// RetentionRunItem is the outcome for one policy inside one run.
type RetentionRunItem struct {
	PolicyID      uuid.UUID       `json:"policy_id"`
	ResourceType  string          `json:"resource_type"`
	Action        string          `json:"action"`
	RetentionDays int             `json:"retention_days"`
	Cutoff        time.Time       `json:"cutoff"`
	Status        string          `json:"status"`
	Matched       int             `json:"matched"`
	Affected      int             `json:"affected"`
	Skipped       int             `json:"skipped"`
	SkipReasons   []RetentionSkip `json:"skip_reasons"`
	Message       string          `json:"message,omitempty"`
}

// RetentionRunResult is one complete run.
type RetentionRunResult struct {
	RunID           uuid.UUID          `json:"run_id"`
	Mode            RetentionMode      `json:"mode"`
	TriggeredBy     string             `json:"triggered_by"`
	StartedAt       time.Time          `json:"started_at"`
	FinishedAt      time.Time          `json:"finished_at"`
	PoliciesTotal   int                `json:"policies_total"`
	RecordsMatched  int                `json:"records_matched"`
	RecordsAffected int                `json:"records_affected"`
	RecordsSkipped  int                `json:"records_skipped"`
	Items           []RetentionRunItem `json:"items"`
}

// RetentionEngine walks the enabled policies of a tenant and hands each one
// to its handler.
type RetentionEngine struct {
	pool     *pgxpool.Pool
	repo     Repository
	registry *RetentionRegistry
}

// NewRetentionEngine wires the engine. repo supplies the anonymization label
// through the same GetNextAnonymizedLabel the erasure service uses — there is
// exactly one label mechanism in this package, not two.
func NewRetentionEngine(pool *pgxpool.Pool, repo Repository, registry *RetentionRegistry) *RetentionEngine {
	if registry == nil {
		registry = NewRetentionRegistry()
	}
	return &RetentionEngine{pool: pool, repo: repo, registry: registry}
}

// Registry exposes the mapping for the admin view.
func (e *RetentionEngine) Registry() *RetentionRegistry { return e.registry }

// Run executes every enabled policy of the caller's tenant and records the
// outcome in retention_runs / retention_run_items.
//
// triggeredBy is a short free label ("manual", "schedule") so an unexpected
// deletion can be traced back to whatever started it.
func (e *RetentionEngine) Run(ctx context.Context, mode RetentionMode, triggeredBy string) (*RetentionRunResult, error) {
	// Anything but an explicit enforce is a dry run. The zero value of
	// RetentionMode must never be the armed mode.
	if mode != RetentionModeEnforce {
		mode = RetentionModeDryRun
	}
	if triggeredBy == "" {
		triggeredBy = "manual"
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("retention: determine tenant: %w", err)
	}

	result := &RetentionRunResult{
		RunID:       uuid.New(),
		Mode:        mode,
		TriggeredBy: triggeredBy,
		StartedAt:   time.Now().UTC(),
		Items:       make([]RetentionRunItem, 0),
	}

	if _, err := e.pool.Exec(ctx,
		`INSERT INTO retention_runs (id, tenant_id, mode, status, triggered_by, started_at)
		 VALUES ($1, $2, $3, 'running', $4, $5)`,
		result.RunID, tenantID, string(mode), triggeredBy, result.StartedAt,
	); err != nil {
		return nil, fmt.Errorf("retention: open run log: %w", err)
	}

	policies, err := e.enabledPolicies(ctx, tenantID)
	if err != nil {
		e.failRun(ctx, result.RunID, tenantID, err)
		return nil, err
	}
	result.PoliciesTotal = len(policies)

	// The label is generated once per run and only when an enforce run really
	// needs it, so a dry run leaves no trace in the erasure counter.
	var label string
	for i := range policies {
		item := e.runPolicy(ctx, tenantID, policies[i], mode, result.StartedAt, &label)
		result.Items = append(result.Items, item)
		result.RecordsMatched += item.Matched
		result.RecordsAffected += item.Affected
		result.RecordsSkipped += item.Skipped

		if itemErr := e.writeItem(ctx, result.RunID, tenantID, item); itemErr != nil {
			slog.Error("retention run item not recorded",
				"run_id", result.RunID, "resource_type", item.ResourceType, "error", itemErr)
		}
	}

	result.FinishedAt = time.Now().UTC()
	if _, err := e.pool.Exec(ctx,
		`UPDATE retention_runs
		    SET status = 'completed', finished_at = $1, policies_total = $2,
		        records_matched = $3, records_affected = $4, records_skipped = $5
		  WHERE id = $6 AND tenant_id = $7`,
		result.FinishedAt, result.PoliciesTotal, result.RecordsMatched,
		result.RecordsAffected, result.RecordsSkipped, result.RunID, tenantID,
	); err != nil {
		return nil, fmt.Errorf("retention: close run log: %w", err)
	}

	slog.Info("retention run finished",
		"run_id", result.RunID,
		"mode", string(mode),
		"triggered_by", triggeredBy,
		"policies", result.PoliciesTotal,
		"matched", result.RecordsMatched,
		"affected", result.RecordsAffected,
		"skipped", result.RecordsSkipped,
	)

	return result, nil
}

// runPolicy resolves one policy. It never returns an error: a handler that
// blows up is recorded as a failed item, because a run that aborts on the
// first bad policy never gets to the good ones.
func (e *RetentionEngine) runPolicy(
	ctx context.Context,
	tenantID uuid.UUID,
	p models.RetentionPolicy,
	mode RetentionMode,
	now time.Time,
	label *string,
) RetentionRunItem {
	item := RetentionRunItem{
		PolicyID:      p.ID,
		ResourceType:  p.ResourceType,
		Action:        p.Action,
		RetentionDays: p.RetentionDays,
		Cutoff:        now.AddDate(0, 0, -p.RetentionDays),
		SkipReasons:   make([]RetentionSkip, 0),
	}

	handler, ok := e.registry.Lookup(p.ResourceType)
	if !ok {
		item.Status = RetentionItemUnmapped
		item.Message = fmt.Sprintf(
			"nicht zugeordnet: fuer resource_type %q ist kein Handler registriert; die Regel loescht nichts",
			p.ResourceType)
		slog.Warn("retention policy without handler",
			"resource_type", p.ResourceType, "policy_id", p.ID)
		return item
	}

	if !handler.SupportsAction(p.Action) {
		item.Status = RetentionItemUnsupported
		item.Message = fmt.Sprintf("Handler fuer %q beherrscht die Aktion %q nicht",
			p.ResourceType, p.Action)
		return item
	}

	plan, err := handler.Plan(ctx, tenantID, item.Cutoff, p.Action)
	if err != nil {
		item.Status = RetentionItemFailed
		item.Message = err.Error()
		slog.Error("retention plan failed",
			"resource_type", p.ResourceType, "policy_id", p.ID, "error", err)
		return item
	}
	item.Matched = len(plan.Due)
	item.Skipped = len(plan.Skipped)
	item.SkipReasons = truncateSkips(plan.Skipped)

	if mode != RetentionModeEnforce {
		item.Status = RetentionItemDryRun
		return item
	}
	if len(plan.Due) == 0 {
		item.Status = RetentionItemApplied
		return item
	}

	if p.Action == models.RetentionActionAnonymize && *label == "" {
		next, labelErr := e.repo.GetNextAnonymizedLabel(ctx)
		if labelErr != nil {
			item.Status = RetentionItemFailed
			item.Message = fmt.Sprintf("Anonymisierungs-Label nicht ermittelbar: %v", labelErr)
			return item
		}
		*label = next
	}

	affected, err := handler.Apply(ctx, tenantID, plan.Due, p.Action, *label)
	if err != nil {
		item.Status = RetentionItemFailed
		item.Message = err.Error()
		slog.Error("retention apply failed",
			"resource_type", p.ResourceType, "policy_id", p.ID, "error", err)
		return item
	}
	item.Status = RetentionItemApplied
	item.Affected = affected
	return item
}

func (e *RetentionEngine) enabledPolicies(ctx context.Context, tenantID uuid.UUID) ([]models.RetentionPolicy, error) {
	rows, err := e.pool.Query(ctx,
		`SELECT id, resource_type, retention_days, action
		   FROM retention_policies
		  WHERE tenant_id = $1 AND enabled = true
		  ORDER BY resource_type ASC`, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("retention: list policies: %w", err)
	}
	defer rows.Close()

	policies := make([]models.RetentionPolicy, 0)
	for rows.Next() {
		var p models.RetentionPolicy
		if scanErr := rows.Scan(&p.ID, &p.ResourceType, &p.RetentionDays, &p.Action); scanErr != nil {
			return nil, fmt.Errorf("retention: scan policy: %w", scanErr)
		}
		p.TenantID = tenantID
		policies = append(policies, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("retention: iterate policies: %w", err)
	}
	return policies, nil
}

func (e *RetentionEngine) writeItem(ctx context.Context, runID, tenantID uuid.UUID, item RetentionRunItem) error {
	reasons, err := json.Marshal(item.SkipReasons)
	if err != nil {
		reasons = []byte("[]")
	}
	_, err = e.pool.Exec(ctx,
		`INSERT INTO retention_run_items
		   (id, tenant_id, run_id, policy_id, resource_type, action, retention_days,
		    cutoff, status, matched, affected, skipped, skip_reasons, message)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		uuid.New(), tenantID, runID, item.PolicyID, item.ResourceType, item.Action,
		item.RetentionDays, item.Cutoff, item.Status, item.Matched, item.Affected,
		item.Skipped, reasons, nullIfEmpty(item.Message),
	)
	return err
}

func (e *RetentionEngine) failRun(ctx context.Context, runID, tenantID uuid.UUID, cause error) {
	if _, err := e.pool.Exec(ctx,
		`UPDATE retention_runs SET status = 'failed', finished_at = NOW(), error = $1
		  WHERE id = $2 AND tenant_id = $3`,
		cause.Error(), runID, tenantID,
	); err != nil {
		slog.Error("retention run not marked as failed", "run_id", runID, "error", err)
	}
}

func truncateSkips(skips []RetentionSkip) []RetentionSkip {
	out := make([]RetentionSkip, 0, len(skips))
	for i, s := range skips {
		if i >= retentionMaxSkipReasons {
			break
		}
		out = append(out, s)
	}
	return out
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
