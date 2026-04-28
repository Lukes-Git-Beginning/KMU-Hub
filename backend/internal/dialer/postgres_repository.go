package dialer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Shared pool wrapper and scan helper
// ---------------------------------------------------------------------------

type scannable interface {
	Scan(dest ...any) error
}

// ---------------------------------------------------------------------------
// PostgresCampaignRepository — implements CampaignRepository
// ---------------------------------------------------------------------------

// PostgresCampaignRepository handles campaigns and their contact queues.
type PostgresCampaignRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresCampaignRepository creates a CampaignRepository backed by pool.
func NewPostgresCampaignRepository(pool *pgxpool.Pool) *PostgresCampaignRepository {
	return &PostgresCampaignRepository{pool: pool}
}

func (r *PostgresCampaignRepository) Create(ctx context.Context, c *Campaign) error {
	settingsJSON, err := json.Marshal(c.Settings)
	if err != nil {
		return fmt.Errorf("marshal campaign settings: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO dialer_campaigns
		    (id, tenant_id, name, description, status, mode, settings,
		     created_by, assigned_agent_ids, contact_count, completed_count,
		     created_at, updated_at, started_at, completed_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		c.ID, c.TenantID, c.Name, c.Description, c.Status, c.Mode, settingsJSON,
		c.CreatedBy, c.AssignedAgentIDs, c.ContactCount, c.CompletedCount,
		c.CreatedAt, c.UpdatedAt, c.StartedAt, c.CompletedAt,
	)
	return err
}

func (r *PostgresCampaignRepository) GetByID(ctx context.Context, id uuid.UUID) (*Campaign, error) {
	// NOTE: tenantID filter is enforced at the service layer via GetByIDForTenant where
	// cross-tenant isolation is required. This method is used for internal lookup by the
	// dialer worker (which uses campaign IDs from its own queue, not user input).
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, status, mode, settings,
		        created_by, assigned_agent_ids, contact_count, completed_count,
		        created_at, updated_at, started_at, completed_at
		 FROM dialer_campaigns WHERE id = $1`, id,
	)
	return scanCampaign(row)
}

func (r *PostgresCampaignRepository) GetByIDForTenant(ctx context.Context, id, tenantID uuid.UUID) (*Campaign, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, status, mode, settings,
		        created_by, assigned_agent_ids, contact_count, completed_count,
		        created_at, updated_at, started_at, completed_at
		 FROM dialer_campaigns WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return scanCampaign(row)
}

func (r *PostgresCampaignRepository) List(ctx context.Context, tenantID uuid.UUID, statusFilter *string, page, pageSize int) ([]*Campaign, int, error) {
	offset := (page - 1) * pageSize

	var (
		whereExtra string
		args       []any
	)
	args = append(args, tenantID)
	if statusFilter != nil {
		whereExtra = " AND status = $2"
		args = append(args, *statusFilter)
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM dialer_campaigns WHERE tenant_id = $1%s", whereExtra)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg := len(args) + 1
	offsetArg := limitArg + 1
	args = append(args, pageSize, offset)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, name, description, status, mode, settings,
		        created_by, assigned_agent_ids, contact_count, completed_count,
		        created_at, updated_at, started_at, completed_at
		 FROM dialer_campaigns
		 WHERE tenant_id = $1%s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		whereExtra, limitArg, offsetArg,
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var campaigns []*Campaign
	for rows.Next() {
		c, err := scanCampaignFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		campaigns = append(campaigns, c)
	}
	return campaigns, total, rows.Err()
}

func (r *PostgresCampaignRepository) Update(ctx context.Context, c *Campaign) error {
	settingsJSON, err := json.Marshal(c.Settings)
	if err != nil {
		return fmt.Errorf("marshal campaign settings: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE dialer_campaigns
		 SET name = $1, description = $2, status = $3, mode = $4, settings = $5,
		     assigned_agent_ids = $6, updated_at = $7, started_at = $8, completed_at = $9
		 WHERE id = $10`,
		c.Name, c.Description, c.Status, c.Mode, settingsJSON,
		c.AssignedAgentIDs, c.UpdatedAt, c.StartedAt, c.CompletedAt,
		c.ID,
	)
	return err
}

func (r *PostgresCampaignRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE dialer_campaigns SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	return err
}

func (r *PostgresCampaignRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM dialer_campaigns WHERE id = $1`, id)
	return err
}

// AddContacts inserts campaign contacts as a batch, skipping duplicate
// (campaign_id, contact_id) pairs. Returns counts of added and skipped rows.
func (r *PostgresCampaignRepository) AddContacts(ctx context.Context, campaignID uuid.UUID, contacts []CampaignContact) (added int, skipped int, err error) {
	if len(contacts) == 0 {
		return 0, 0, nil
	}

	batch := &pgx.Batch{}
	for _, cc := range contacts {
		batch.Queue(
			`INSERT INTO dialer_campaign_contacts
			    (id, campaign_id, contact_id, position, status, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, 'pending', NOW(), NOW())
			 ON CONFLICT ON CONSTRAINT uq_campaign_contact DO NOTHING`,
			cc.ID, campaignID, cc.ContactID, cc.Position,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range contacts {
		tag, execErr := br.Exec()
		if execErr != nil {
			return added, skipped, execErr
		}
		if tag.RowsAffected() == 1 {
			added++
		} else {
			skipped++
		}
	}
	return added, skipped, nil
}

// GetNextPendingContact atomically claims the next pending contact in the campaign
// queue using SELECT … FOR UPDATE SKIP LOCKED to prevent double-assignment under
// concurrent agents.
func (r *PostgresCampaignRepository) GetNextPendingContact(ctx context.Context, campaignID uuid.UUID) (*CampaignContact, error) {
	row := r.pool.QueryRow(ctx,
		`WITH next AS (
		    SELECT id FROM dialer_campaign_contacts
		    WHERE campaign_id = $1 AND status = 'pending'
		    ORDER BY position ASC
		    LIMIT 1
		    FOR UPDATE SKIP LOCKED
		)
		UPDATE dialer_campaign_contacts
		SET status = 'in_progress', updated_at = NOW()
		FROM next
		WHERE dialer_campaign_contacts.id = next.id
		RETURNING
		    dialer_campaign_contacts.id,
		    dialer_campaign_contacts.campaign_id,
		    dialer_campaign_contacts.contact_id,
		    dialer_campaign_contacts.position,
		    dialer_campaign_contacts.status,
		    dialer_campaign_contacts.outcome_id,
		    dialer_campaign_contacts.notes,
		    dialer_campaign_contacts.callback_at,
		    dialer_campaign_contacts.last_called_at,
		    dialer_campaign_contacts.call_count,
		    dialer_campaign_contacts.created_at,
		    dialer_campaign_contacts.updated_at`,
		campaignID,
	)
	cc, err := scanCampaignContact(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoContactsAvailable
	}
	return cc, err
}

func (r *PostgresCampaignRepository) ListContacts(ctx context.Context, campaignID uuid.UUID, statusFilter *string, page, pageSize int) ([]*CampaignContact, int, error) {
	offset := (page - 1) * pageSize

	var (
		whereExtra string
		args       []any
	)
	args = append(args, campaignID)
	if statusFilter != nil {
		whereExtra = " AND status = $2"
		args = append(args, *statusFilter)
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM dialer_campaign_contacts WHERE campaign_id = $1%s", whereExtra)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg := len(args) + 1
	offsetArg := limitArg + 1
	args = append(args, pageSize, offset)

	query := fmt.Sprintf(
		`SELECT id, campaign_id, contact_id, position, status, outcome_id, notes,
		        callback_at, last_called_at, call_count, created_at, updated_at
		 FROM dialer_campaign_contacts
		 WHERE campaign_id = $1%s
		 ORDER BY position ASC
		 LIMIT $%d OFFSET $%d`,
		whereExtra, limitArg, offsetArg,
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*CampaignContact
	for rows.Next() {
		var cc CampaignContact
		if err := rows.Scan(
			&cc.ID, &cc.CampaignID, &cc.ContactID, &cc.Position, &cc.Status,
			&cc.OutcomeID, &cc.Notes, &cc.CallbackAt, &cc.LastCalledAt,
			&cc.CallCount, &cc.CreatedAt, &cc.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, &cc)
	}
	return result, total, rows.Err()
}

func (r *PostgresCampaignRepository) UpdateContactStatus(ctx context.Context, id uuid.UUID, status string, outcomeID *uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE dialer_campaign_contacts
		 SET status = $1, outcome_id = $2, updated_at = NOW()
		 WHERE id = $3`,
		status, outcomeID, id,
	)
	return err
}

func (r *PostgresCampaignRepository) SetContactCallback(ctx context.Context, id uuid.UUID, callbackAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE dialer_campaign_contacts
		 SET status = 'callback', callback_at = $1, updated_at = NOW()
		 WHERE id = $2`,
		callbackAt, id,
	)
	return err
}

func (r *PostgresCampaignRepository) SkipContact(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE dialer_campaign_contacts
		 SET status = 'skipped', updated_at = NOW()
		 WHERE id = $1`,
		id,
	)
	return err
}

func (r *PostgresCampaignRepository) RequeueContact(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE dialer_campaign_contacts
		 SET status = 'pending', outcome_id = NULL, callback_at = NULL, updated_at = NOW()
		 WHERE id = $1`,
		id,
	)
	return err
}

func (r *PostgresCampaignRepository) GetCampaignContactByID(ctx context.Context, id uuid.UUID) (*CampaignContact, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, campaign_id, contact_id, position, status, outcome_id, notes,
		        callback_at, last_called_at, call_count, created_at, updated_at
		 FROM dialer_campaign_contacts WHERE id = $1`, id,
	)
	cc, err := scanCampaignContact(row)
	if errors.Is(err, ErrNoContactsAvailable) {
		return nil, ErrCampaignContactNotFound
	}
	return cc, err
}

func (r *PostgresCampaignRepository) IncrementContactCallCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE dialer_campaign_contacts
		 SET call_count = call_count + 1, last_called_at = NOW(), updated_at = NOW()
		 WHERE id = $1`,
		id,
	)
	return err
}

// GetCampaignStats returns aggregated progress and outcome breakdown for a campaign.
func (r *PostgresCampaignRepository) GetCampaignStats(ctx context.Context, campaignID uuid.UUID) (*CampaignStats, error) {
	var stats CampaignStats

	if err := r.pool.QueryRow(ctx,
		`SELECT
		    COUNT(*)                                                      AS total,
		    COUNT(*) FILTER (WHERE status = 'completed')                  AS completed,
		    COUNT(*) FILTER (WHERE status = 'pending')                    AS pending,
		    COUNT(*) FILTER (WHERE status = 'skipped')                    AS skipped,
		    COUNT(*) FILTER (WHERE status = 'callback')                   AS callback
		 FROM dialer_campaign_contacts
		 WHERE campaign_id = $1`,
		campaignID,
	).Scan(
		&stats.TotalContacts,
		&stats.CompletedContacts,
		&stats.PendingContacts,
		&stats.SkippedContacts,
		&stats.CallbackContacts,
	); err != nil {
		return nil, err
	}

	if err := r.pool.QueryRow(ctx,
		`SELECT
		    COUNT(*)                                                                              AS total_calls,
		    COUNT(*) FILTER (WHERE o.is_positive = true)                                         AS positive_calls,
		    COALESCE(AVG(s.duration_seconds) FILTER (WHERE s.duration_seconds IS NOT NULL), 0)   AS avg_duration
		 FROM dialer_call_sessions s
		 JOIN dialer_campaign_contacts cc ON cc.id = s.campaign_contact_id
		 LEFT JOIN dialer_call_outcomes o ON o.id = s.outcome_id
		 WHERE cc.campaign_id = $1`,
		campaignID,
	).Scan(&stats.TotalCalls, &stats.PositiveCalls, &stats.AvgDurationSecs); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT o.id, o.label, o.color, COUNT(s.id) AS cnt
		 FROM dialer_call_sessions s
		 JOIN dialer_campaign_contacts cc ON cc.id = s.campaign_contact_id
		 JOIN dialer_call_outcomes o ON o.id = s.outcome_id
		 WHERE cc.campaign_id = $1
		 GROUP BY o.id, o.label, o.color
		 ORDER BY cnt DESC`,
		campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var oc OutcomeCount
		if err := rows.Scan(&oc.OutcomeID, &oc.OutcomeLabel, &oc.OutcomeColor, &oc.Count); err != nil {
			return nil, err
		}
		stats.OutcomeBreakdown = append(stats.OutcomeBreakdown, oc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetAgentStats returns today's aggregated performance figures for a given agent.
func (r *PostgresCampaignRepository) GetAgentStats(ctx context.Context, agentID uuid.UUID) (*AgentStats, error) {
	var stats AgentStats

	if err := r.pool.QueryRow(ctx,
		`SELECT
		    COUNT(*)                                                                              AS total_calls,
		    COALESCE(AVG(duration_seconds) FILTER (WHERE duration_seconds IS NOT NULL), 0)       AS avg_duration
		 FROM dialer_call_sessions
		 WHERE agent_id = $1
		   AND created_at >= CURRENT_DATE`,
		agentID,
	).Scan(&stats.TotalCallsToday, &stats.AvgDurationSecs); err != nil {
		return nil, err
	}

	// calls-per-hour: total calls / hours elapsed since midnight (floor at 1 h)
	var hoursSinceMidnight float64
	if err := r.pool.QueryRow(ctx,
		`SELECT GREATEST(EXTRACT(EPOCH FROM (NOW() - CURRENT_DATE)) / 3600.0, 1.0)`,
	).Scan(&hoursSinceMidnight); err != nil {
		return nil, err
	}
	stats.CallsPerHour = float64(stats.TotalCallsToday) / hoursSinceMidnight

	rows, err := r.pool.Query(ctx,
		`SELECT o.id, o.label, o.color, COUNT(s.id) AS cnt
		 FROM dialer_call_sessions s
		 JOIN dialer_call_outcomes o ON o.id = s.outcome_id
		 WHERE s.agent_id = $1
		   AND s.created_at >= CURRENT_DATE
		 GROUP BY o.id, o.label, o.color
		 ORDER BY cnt DESC`,
		agentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var oc OutcomeCount
		if err := rows.Scan(&oc.OutcomeID, &oc.OutcomeLabel, &oc.OutcomeColor, &oc.Count); err != nil {
			return nil, err
		}
		stats.OutcomeBreakdown = append(stats.OutcomeBreakdown, oc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Active campaign — most recent on_call/wrap_up log entry with a campaign
	if err := r.pool.QueryRow(ctx,
		`SELECT l.campaign_id, c.name
		 FROM dialer_agent_status_log l
		 JOIN dialer_campaigns c ON c.id = l.campaign_id
		 WHERE l.user_id = $1
		   AND l.campaign_id IS NOT NULL
		   AND l.status IN ('on_call','wrap_up')
		 ORDER BY l.changed_at DESC
		 LIMIT 1`,
		agentID,
	).Scan(&stats.ActiveCampaignID, &stats.ActiveCampaignName); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	return &stats, nil
}

// UpdateCampaignCounts recalculates contact_count and completed_count from live data.
func (r *PostgresCampaignRepository) UpdateCampaignCounts(ctx context.Context, campaignID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE dialer_campaigns c
		 SET
		     contact_count   = (SELECT COUNT(*) FROM dialer_campaign_contacts WHERE campaign_id = c.id),
		     completed_count = (SELECT COUNT(*) FROM dialer_campaign_contacts WHERE campaign_id = c.id AND status = 'completed'),
		     updated_at      = NOW()
		 WHERE id = $1`,
		campaignID,
	)
	return err
}

// ---------------------------------------------------------------------------
// PostgresCallRepository — implements CallRepository
// ---------------------------------------------------------------------------

// PostgresCallRepository handles call sessions and their event log.
type PostgresCallRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresCallRepository creates a CallRepository backed by pool.
func NewPostgresCallRepository(pool *pgxpool.Pool) *PostgresCallRepository {
	return &PostgresCallRepository{pool: pool}
}

func (r *PostgresCallRepository) CreateSession(ctx context.Context, s *CallSession) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO dialer_call_sessions
		    (id, campaign_contact_id, call_session_id, agent_id, outcome_id,
		     duration_seconds, notes, next_action, appointment_id,
		     created_at, updated_at, wrap_up_completed_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		s.ID, s.CampaignContactID, s.CallSessionID, s.AgentID, s.OutcomeID,
		s.DurationSeconds, s.Notes, s.NextAction, s.AppointmentID,
		s.CreatedAt, s.UpdatedAt, s.WrapUpCompletedAt,
	)
	return err
}

func (r *PostgresCallRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*CallSession, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, campaign_contact_id, call_session_id, agent_id, outcome_id,
		        duration_seconds, notes, next_action, appointment_id,
		        created_at, updated_at, wrap_up_completed_at
		 FROM dialer_call_sessions WHERE id = $1`, id,
	)
	return scanCallSession(row)
}

func (r *PostgresCallRepository) UpdateSession(ctx context.Context, s *CallSession) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE dialer_call_sessions
		 SET call_session_id = $1, outcome_id = $2, duration_seconds = $3,
		     notes = $4, next_action = $5, appointment_id = $6,
		     updated_at = $7, wrap_up_completed_at = $8
		 WHERE id = $9`,
		s.CallSessionID, s.OutcomeID, s.DurationSeconds,
		s.Notes, s.NextAction, s.AppointmentID,
		s.UpdatedAt, s.WrapUpCompletedAt,
		s.ID,
	)
	return err
}

func (r *PostgresCallRepository) AppendEvent(ctx context.Context, e *CallEvent) error {
	payloadJSON, err := json.Marshal(e.Payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO dialer_call_events
		    (id, dialer_call_session_id, event_type, payload, occurred_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		e.ID, e.DialerCallSessionID, e.EventType, payloadJSON, e.OccurredAt,
	)
	return err
}

func (r *PostgresCallRepository) ListEventsBySession(ctx context.Context, sessionID uuid.UUID) ([]*CallEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, dialer_call_session_id, event_type, payload, occurred_at
		 FROM dialer_call_events
		 WHERE dialer_call_session_id = $1
		 ORDER BY occurred_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*CallEvent
	for rows.Next() {
		var (
			e           CallEvent
			payloadJSON []byte
		)
		if err := rows.Scan(&e.ID, &e.DialerCallSessionID, &e.EventType, &payloadJSON, &e.OccurredAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payloadJSON, &e.Payload); err != nil {
			return nil, fmt.Errorf("unmarshal event payload: %w", err)
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

// ---------------------------------------------------------------------------
// PostgresOutcomeRepository — implements OutcomeRepository
// ---------------------------------------------------------------------------

// PostgresOutcomeRepository handles call outcomes (per-tenant configurable).
type PostgresOutcomeRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresOutcomeRepository creates an OutcomeRepository backed by pool.
func NewPostgresOutcomeRepository(pool *pgxpool.Pool) *PostgresOutcomeRepository {
	return &PostgresOutcomeRepository{pool: pool}
}

func (r *PostgresOutcomeRepository) Create(ctx context.Context, o *CallOutcome) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO dialer_call_outcomes
		    (id, tenant_id, label, color, is_positive, is_callback, is_appointment,
		     sort_order, is_active, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		o.ID, o.TenantID, o.Label, o.Color, o.IsPositive, o.IsCallback, o.IsAppointment,
		o.SortOrder, o.IsActive, o.CreatedAt,
	)
	return err
}

func (r *PostgresOutcomeRepository) GetByID(ctx context.Context, id uuid.UUID) (*CallOutcome, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, label, color, is_positive, is_callback, is_appointment,
		        sort_order, is_active, created_at
		 FROM dialer_call_outcomes WHERE id = $1`, id,
	)
	return scanOutcome(row)
}

func (r *PostgresOutcomeRepository) List(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]*CallOutcome, error) {
	query := `SELECT id, tenant_id, label, color, is_positive, is_callback, is_appointment,
		         sort_order, is_active, created_at
		  FROM dialer_call_outcomes
		  WHERE tenant_id = $1`
	if !includeInactive {
		query += " AND is_active = true"
	}
	query += " ORDER BY sort_order ASC, label ASC"

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outcomes []*CallOutcome
	for rows.Next() {
		o, err := scanOutcomeFromRows(rows)
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, o)
	}
	return outcomes, rows.Err()
}

func (r *PostgresOutcomeRepository) Update(ctx context.Context, o *CallOutcome) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE dialer_call_outcomes
		 SET label = $1, color = $2, is_positive = $3, is_callback = $4,
		     is_appointment = $5, sort_order = $6, is_active = $7
		 WHERE id = $8`,
		o.Label, o.Color, o.IsPositive, o.IsCallback,
		o.IsAppointment, o.SortOrder, o.IsActive,
		o.ID,
	)
	return err
}

func (r *PostgresOutcomeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM dialer_call_outcomes WHERE id = $1`, id)
	return err
}

// EnsureDefaults inserts the 4 standard DACH outcomes for a tenant if none exist yet.
func (r *PostgresOutcomeRepository) EnsureDefaults(ctx context.Context, tenantID uuid.UUID) error {
	var count int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM dialer_call_outcomes WHERE tenant_id = $1`, tenantID,
	).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	type seed struct {
		label         string
		color         string
		isPositive    bool
		isCallback    bool
		isAppointment bool
		sortOrder     int
	}

	seeds := []seed{
		{"Erreicht", "#22c55e", true, false, false, 1},
		{"Nicht erreicht", "#ef4444", false, false, false, 2},
		{"Wiedervorlage", "#f59e0b", false, true, false, 3},
		{"Termin", "#3b82f6", true, false, true, 4},
	}

	batch := &pgx.Batch{}
	for _, s := range seeds {
		batch.Queue(
			`INSERT INTO dialer_call_outcomes
			    (id, tenant_id, label, color, is_positive, is_callback, is_appointment,
			     sort_order, is_active, created_at)
			 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, true, NOW())`,
			tenantID, s.label, s.color, s.isPositive, s.isCallback, s.isAppointment, s.sortOrder,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range seeds {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// PostgresAgentStatusRepository — implements AgentStatusRepository
// ---------------------------------------------------------------------------

// PostgresAgentStatusRepository appends agent status change log entries.
type PostgresAgentStatusRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAgentStatusRepository creates an AgentStatusRepository backed by pool.
func NewPostgresAgentStatusRepository(pool *pgxpool.Pool) *PostgresAgentStatusRepository {
	return &PostgresAgentStatusRepository{pool: pool}
}

func (r *PostgresAgentStatusRepository) LogStatusChange(ctx context.Context, entry *AgentStatusLogEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO dialer_agent_status_log
		    (id, user_id, campaign_id, status, previous_status, changed_at, changed_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		entry.ID, entry.UserID, entry.CampaignID, entry.Status,
		entry.PreviousStatus, entry.ChangedAt, entry.ChangedBy,
	)
	return err
}

// ---------------------------------------------------------------------------
// Package-level scan helpers (shared across repository types)
// ---------------------------------------------------------------------------

func scanCampaign(row scannable) (*Campaign, error) {
	var (
		c            Campaign
		settingsJSON []byte
	)
	err := row.Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description,
		&c.Status, &c.Mode, &settingsJSON,
		&c.CreatedBy, &c.AssignedAgentIDs,
		&c.ContactCount, &c.CompletedCount,
		&c.CreatedAt, &c.UpdatedAt, &c.StartedAt, &c.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCampaignNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(settingsJSON, &c.Settings); err != nil {
		return nil, fmt.Errorf("unmarshal campaign settings: %w", err)
	}
	return &c, nil
}

func scanCampaignFromRows(rows pgx.Rows) (*Campaign, error) {
	var (
		c            Campaign
		settingsJSON []byte
	)
	err := rows.Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description,
		&c.Status, &c.Mode, &settingsJSON,
		&c.CreatedBy, &c.AssignedAgentIDs,
		&c.ContactCount, &c.CompletedCount,
		&c.CreatedAt, &c.UpdatedAt, &c.StartedAt, &c.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(settingsJSON, &c.Settings); err != nil {
		return nil, fmt.Errorf("unmarshal campaign settings: %w", err)
	}
	return &c, nil
}

func scanCampaignContact(row scannable) (*CampaignContact, error) {
	var cc CampaignContact
	err := row.Scan(
		&cc.ID, &cc.CampaignID, &cc.ContactID, &cc.Position,
		&cc.Status, &cc.OutcomeID, &cc.Notes,
		&cc.CallbackAt, &cc.LastCalledAt, &cc.CallCount,
		&cc.CreatedAt, &cc.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoContactsAvailable
	}
	if err != nil {
		return nil, err
	}
	return &cc, nil
}

func scanCallSession(row scannable) (*CallSession, error) {
	var s CallSession
	err := row.Scan(
		&s.ID, &s.CampaignContactID, &s.CallSessionID, &s.AgentID, &s.OutcomeID,
		&s.DurationSeconds, &s.Notes, &s.NextAction, &s.AppointmentID,
		&s.CreatedAt, &s.UpdatedAt, &s.WrapUpCompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCallSessionNotFound
	}
	return &s, err
}

func scanOutcome(row scannable) (*CallOutcome, error) {
	var o CallOutcome
	err := row.Scan(
		&o.ID, &o.TenantID, &o.Label, &o.Color,
		&o.IsPositive, &o.IsCallback, &o.IsAppointment,
		&o.SortOrder, &o.IsActive, &o.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOutcomeNotFound
	}
	return &o, err
}

func scanOutcomeFromRows(rows pgx.Rows) (*CallOutcome, error) {
	var o CallOutcome
	err := rows.Scan(
		&o.ID, &o.TenantID, &o.Label, &o.Color,
		&o.IsPositive, &o.IsCallback, &o.IsAppointment,
		&o.SortOrder, &o.IsActive, &o.CreatedAt,
	)
	return &o, err
}
