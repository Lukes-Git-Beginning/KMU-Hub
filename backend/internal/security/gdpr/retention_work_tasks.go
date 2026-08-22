package gdpr

// TaskRetentionHandler implements RetentionHandler for resource_type "tasks"
// — sixth handler on the retention engine from A10 (retention.go). Fund aus
// scan-personal-data-tables-without-retention-mapping (Iteration 43): the
// Work module (tasks, task_comments) collects personal data (titles,
// descriptions, comment text with name references) without a mapping in the
// retention registry, even though A8 (dsar_search.go, tasksModule /
// taskCommentsModule) already surfaces the same tables in DSAR export.
//
// Scope is COMPLETED tasks only — the same closed-status gate A13 uses for
// helpdesk tickets. completed_at is stamped exactly once, by
// task.Service.MoveTask on the transition into a closed-category status
// (service.go:423-426), and cleared again if the task is reopened, so it is
// both the completion timestamp and a reliable "is this task actually done"
// flag in one column — no separate status lookup against project_statuses
// needed.
//
// task_comments.task_id is ON DELETE CASCADE onto tasks (migration 000026),
// so a hard delete of a completed task takes its comment trail with it for
// free, the same relationship A13 relies on for ticket_messages. Anonymize
// does not get that for free — it never removes the row — so this handler
// anonymizes task_comments.content explicitly, exactly as A13 anonymizes
// ticket_messages.body.
//
// tasks.parent_task_id is ALSO ON DELETE CASCADE onto tasks (migration
// 000025: subtasks). That is a second cascade a ticket handler never had to
// consider: deleting a completed parent task would silently delete every
// subtask underneath it, including one that is still active. Plan excludes a
// completed task from Due (delete only) if it has any direct child whose
// completed_at is still NULL, and reports why instead of letting the
// cascade remove live data. Deeper hierarchies (a completed grandchild under
// an still-active child) are not walked recursively here — the direct-child
// guard is the boundary this handler draws; a task tree with an active leaf
// three levels down is not observed in the current usage of this module.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TaskRetentionHandler is the tasks entry in the retention registry.
type TaskRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewTaskRetentionHandler wires the handler.
func NewTaskRetentionHandler(pool *pgxpool.Pool) *TaskRetentionHandler {
	return &TaskRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for tasks.
func (h *TaskRetentionHandler) ResourceType() string { return "tasks" }

// Table names the governed table for the admin view.
func (h *TaskRetentionHandler) Table() string { return "tasks" }

// DateColumn discloses the retention clock. For anonymize, updated_at must
// also be past the cutoff — Apply bumps it on anonymize, the same
// idempotency technique HelpdeskTicketRetentionHandler uses.
func (h *TaskRetentionHandler) DateColumn() string {
	return "completed_at (nur abgeschlossene Aufgaben ohne aktive Unteraufgabe; bei anonymize zusaetzlich updated_at)"
}

// SupportsAction reports that completed tasks can be both hard-deleted and
// anonymized.
func (h *TaskRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete || action == models.RetentionActionAnonymize
}

// Plan lists completed tasks past their retention cutoff. Never modifies
// data. For action == delete, a candidate with an active (not yet completed)
// direct subtask is moved to Skipped instead of Due — deleting it would
// cascade the active subtask away via parent_task_id ON DELETE CASCADE.
func (h *TaskRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, action string) (*RetentionPlan, error) {
	query := `SELECT id FROM tasks WHERE tenant_id = $1 AND completed_at IS NOT NULL AND completed_at < $2`
	if action == models.RetentionActionAnonymize {
		query += ` AND updated_at < $2`
	}
	query += ` ORDER BY completed_at ASC`

	rows, err := h.pool.Query(ctx, query, tenantID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("task retention: list candidates: %w", err)
	}
	defer rows.Close()

	candidates := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("task retention: scan candidate: %w", scanErr)
		}
		candidates = append(candidates, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("task retention: iterate candidates: %w", err)
	}

	plan := &RetentionPlan{Due: candidates}
	if action != models.RetentionActionDelete || len(candidates) == 0 {
		return plan, nil
	}

	blocked, err := h.parentsWithActiveChildren(ctx, tenantID, candidates)
	if err != nil {
		return nil, err
	}
	if len(blocked) == 0 {
		return plan, nil
	}

	plan.Due = make([]uuid.UUID, 0, len(candidates))
	for _, id := range candidates {
		if blocked[id] {
			plan.Skipped = append(plan.Skipped, RetentionSkip{
				RecordID: id,
				Reason:   "hat eine noch offene Unteraufgabe; wird nicht geloescht, um die aktive Unteraufgabe nicht per Kaskade zu verlieren",
			})
			continue
		}
		plan.Due = append(plan.Due, id)
	}
	return plan, nil
}

// parentsWithActiveChildren returns the subset of candidateParents that have
// at least one direct subtask whose completed_at is still NULL.
func (h *TaskRetentionHandler) parentsWithActiveChildren(ctx context.Context, tenantID uuid.UUID, candidateParents []uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT DISTINCT parent_task_id FROM tasks
		  WHERE tenant_id = $1 AND parent_task_id = ANY($2) AND completed_at IS NULL`,
		tenantID, candidateParents,
	)
	if err != nil {
		return nil, fmt.Errorf("task retention: check active subtasks: %w", err)
	}
	defer rows.Close()

	blocked := make(map[uuid.UUID]bool)
	for rows.Next() {
		var parentID uuid.UUID
		if scanErr := rows.Scan(&parentID); scanErr != nil {
			return nil, fmt.Errorf("task retention: scan blocked parent: %w", scanErr)
		}
		blocked[parentID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("task retention: iterate blocked parents: %w", err)
	}
	return blocked, nil
}

// Apply deletes or anonymizes the given tasks.
//
// Delete relies on the ON DELETE CASCADE from task_comments to tasks
// (migration 000026) to take the comment trail with it, and on the same
// cascade from parent_task_id to remove completed subtasks — Plan already
// excluded any parent with a still-active one.
//
// Anonymize clears the task's two free-text fields (title, description),
// then does the same for every comment under the task — a completed task
// with its title cleared but its comments intact would still carry the same
// personal data one layer down.
func (h *TaskRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, anonymizedLabel string) (int, error) {
	switch action {
	case models.RetentionActionDelete:
		tag, err := h.pool.Exec(ctx,
			`DELETE FROM tasks WHERE tenant_id = $1 AND id = ANY($2)`,
			tenantID, ids,
		)
		if err != nil {
			return 0, fmt.Errorf("task retention: delete: %w", err)
		}
		return int(tag.RowsAffected()), nil
	case models.RetentionActionAnonymize:
		tag, err := h.pool.Exec(ctx,
			`UPDATE tasks
			    SET title = '[' || $3 || ']',
			        description = '[' || $3 || ']',
			        updated_at = NOW()
			  WHERE tenant_id = $1 AND id = ANY($2)`,
			tenantID, ids, anonymizedLabel,
		)
		if err != nil {
			return 0, fmt.Errorf("task retention: anonymize: %w", err)
		}
		affected := int(tag.RowsAffected())

		if _, err := h.pool.Exec(ctx,
			`UPDATE task_comments SET content = '[' || $3 || ']'
			  WHERE tenant_id = $1 AND task_id = ANY($2)`,
			tenantID, ids, anonymizedLabel,
		); err != nil {
			return affected, fmt.Errorf("task retention: anonymize comments: %w", err)
		}
		return affected, nil
	default:
		return 0, fmt.Errorf("task retention: unsupported action %q", action)
	}
}
