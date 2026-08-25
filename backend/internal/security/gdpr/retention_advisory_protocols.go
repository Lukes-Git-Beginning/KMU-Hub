package gdpr

// AdvisoryProtocolRetentionHandler implements RetentionHandler for
// resource_type "advisory_protocols" — harden-advisory-protocols-retention-guard
// (A6 Teilentscheidung, 2026-08-24). advisory_protocols (migration 000137)
// documents financial advisory sessions under MiFID II, §64 WpHG, §§16-18
// FinVermV and §61 VVG (IDD); §18a FinVermV sets a 10-year retention
// requirement once a protocol is handed over.
//
// This is deliberately NOT an active deletion handler. The real risk here is
// not that a protocol outlives its retention period — nobody has built a
// deletion path for it, and it must stay that way until a row actually
// crosses 10 years — the risk is a future generic cleanup pass reading the
// engine's default "unmapped" status as "nobody thought about this table"
// and wiring one in on a misunderstanding. SupportsAction refuses both
// actions unconditionally, and the refusal names the legal basis so an
// auditor reading a run report sees a decision, not a gap.
//
// lean: this handler is entirely read-only — Plan counts, nothing acts.
// Upgrade trigger: the day a tenant has an advisory_protocols row whose
// handed_over_at is more than 10 years in the past, this needs a real
// delete path (and a decision on whether §18a's 10 years or a longer
// overlapping duty applies) — not before, since an unused delete path on
// the most sensitive table in the schema would be untested code exactly
// where it is least affordable.
import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// advisoryProtocolRetentionReason is the shared explanation for why this
// handler never supports an action — surfaced through UnsupportedReason so
// the retention engine's run report names the legal basis instead of the
// generic "does not support this" message.
const advisoryProtocolRetentionReason = "advisory_protocols unterliegt der 10-Jahres-Aufbewahrungspflicht " +
	"nach §18a FinVermV (i.V.m. MiFID II, §64 WpHG, §§16-18 FinVermV, §61 VVG/IDD); " +
	"dieser Handler loescht und anonymisiert bewusst nicht, solange kein Bestand die Frist ueberschreitet"

// AdvisoryProtocolRetentionHandler is the advisory_protocols entry in the
// retention registry.
type AdvisoryProtocolRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewAdvisoryProtocolRetentionHandler wires the handler.
func NewAdvisoryProtocolRetentionHandler(pool *pgxpool.Pool) *AdvisoryProtocolRetentionHandler {
	return &AdvisoryProtocolRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for advisory
// protocols.
func (h *AdvisoryProtocolRetentionHandler) ResourceType() string { return "advisory_protocols" }

// Table names the governed table for the admin view.
func (h *AdvisoryProtocolRetentionHandler) Table() string { return "advisory_protocols" }

// DateColumn discloses the clock the 10-year duty would run on, even though
// this handler never starts it — a protocol that was never handed over
// (still status='draft', handed_over_at IS NULL) has not triggered the duty
// at all.
func (h *AdvisoryProtocolRetentionHandler) DateColumn() string {
	return "handed_over_at (nur finalisierte Protokolle; ein Draft startet die Frist nicht)"
}

// SupportsAction always refuses — see the file header and
// advisoryProtocolRetentionReason.
func (h *AdvisoryProtocolRetentionHandler) SupportsAction(string) bool {
	return false
}

// UnsupportedReason implements the engine's optional richer-message hook
// (retention.go) so a run report names the legal basis instead of the
// generic "handler does not support this action" text.
func (h *AdvisoryProtocolRetentionHandler) UnsupportedReason(string) string {
	return advisoryProtocolRetentionReason
}

// Plan counts finalized protocols past cutoff so an admin report can show
// how much of the tenant's advisory history is already beyond the 10-year
// mark, without acting on it. The engine never reaches this in practice —
// SupportsAction gates it out first — but the query is real and directly
// testable, so the count is trustworthy the day a delete path is built on
// top of it. Never modifies data.
func (h *AdvisoryProtocolRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id FROM advisory_protocols
		  WHERE tenant_id = $1 AND handed_over_at IS NOT NULL AND handed_over_at < $2
		  ORDER BY handed_over_at ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("advisory protocol retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("advisory protocol retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("advisory protocol retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids, Skipped: make([]RetentionSkip, 0)}, nil
}

// Apply always errors — this handler has no delete or anonymize path. The
// engine never calls it in practice (SupportsAction gates it out first);
// this is the defensive fallback if that ever changes by mistake.
func (h *AdvisoryProtocolRetentionHandler) Apply(context.Context, uuid.UUID, []uuid.UUID, string, string) (int, error) {
	return 0, fmt.Errorf("advisory protocol retention: %s", advisoryProtocolRetentionReason)
}

var (
	_ RetentionHandler    = (*AdvisoryProtocolRetentionHandler)(nil)
	_ unsupportedReasoner = (*AdvisoryProtocolRetentionHandler)(nil)
)
