package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements the audit Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new audit PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Create inserts a new audit entry with hash chain integrity.
// Uses pg_advisory_xact_lock to serialize writes and maintain chain ordering.
func (r *PostgresRepository) Create(ctx context.Context, entry *models.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Acquire advisory lock to serialize audit writes
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", AuditLockID); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}

	// Get the previous hash
	var previousHash string
	err = tx.QueryRow(ctx,
		"SELECT entry_hash FROM audit_log ORDER BY sequence_num DESC LIMIT 1",
	).Scan(&previousHash)
	if errors.Is(err, pgx.ErrNoRows) {
		previousHash = ""
	} else if err != nil {
		return fmt.Errorf("get last hash: %w", err)
	}

	entry.PreviousHash = previousHash
	entry.EntryHash = computeEntryHash(entry, previousHash)

	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	// ip_address is a nullable INET column; an empty string is not valid INET
	// input and fails the insert (SQLSTATE 22P02). Callers that don't have a
	// client IP (e.g. the CreateAuditEntry RPC when the caller omits it) pass
	// the model's zero value, so normalize it to SQL NULL here.
	var ipAddress any
	if entry.IPAddress != "" {
		ipAddress = entry.IPAddress
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO audit_log (id, tenant_id, timestamp, user_id, action, target, target_type, details, ip_address, user_agent, result, previous_hash, entry_hash)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		entry.ID, entry.TenantID, entry.Timestamp, entry.UserID, entry.Action,
		entry.Target, entry.TargetType, entry.Details,
		ipAddress, entry.UserAgent, entry.Result,
		entry.PreviousHash, entry.EntryHash,
	)
	if err != nil {
		return fmt.Errorf("insert audit entry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// computeEntryHash creates a SHA-256 hash of the entry fields plus the previous hash.
// Format: timestamp|user_id|action|target|details|ip_address|result|previousHash
func computeEntryHash(entry *models.AuditEntry, previousHash string) string {
	var userID string
	if entry.UserID != nil {
		userID = entry.UserID.String()
	}

	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		entry.Timestamp.UTC().Format(time.RFC3339Nano),
		userID,
		entry.Action,
		entry.Target,
		entry.Details,
		entry.IPAddress,
		entry.Result,
		previousHash,
	)

	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// List returns audit entries matching the filter with pagination.
// Always filters by tenant_id to enforce tenant isolation.
func (r *PostgresRepository) List(ctx context.Context, filter *models.AuditFilter) ([]*models.AuditEntry, int, error) {
	// tenant_id is always the first filter (audit_log.tenant_id is NOT NULL)
	conditions := []string{fmt.Sprintf("tenant_id = $%d", 1)}
	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, *filter.DateTo)
		argIdx++
	}
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, *filter.UserID)
		argIdx++
	}
	if filter.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, filter.Action)
		argIdx++
	}
	if filter.Result != "" {
		conditions = append(conditions, fmt.Sprintf("result = $%d", argIdx))
		args = append(args, filter.Result)
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total matching entries
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_log %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit entries: %w", err)
	}

	// Apply pagination defaults
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Fetch entries with pagination
	dataQuery := fmt.Sprintf(
		`SELECT id, sequence_num, timestamp, user_id, action, target, target_type, details, ip_address, user_agent, result, previous_hash, entry_hash
		 FROM audit_log %s
		 ORDER BY sequence_num DESC
		 LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit entries: %w", err)
	}
	defer rows.Close()

	var entries []*models.AuditEntry
	for rows.Next() {
		var e models.AuditEntry
		if err := rows.Scan(
			&e.ID, &e.SequenceNum, &e.Timestamp, &e.UserID, &e.Action,
			&e.Target, &e.TargetType, &e.Details, &e.IPAddress, &e.UserAgent,
			&e.Result, &e.PreviousHash, &e.EntryHash,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit entry: %w", err)
		}
		entries = append(entries, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	return entries, total, nil
}

// GetLastHash returns the entry_hash of the most recent audit entry.
// Returns empty string if no entries exist.
func (r *PostgresRepository) GetLastHash(ctx context.Context) (string, error) {
	var hash string
	err := r.pool.QueryRow(ctx,
		"SELECT entry_hash FROM audit_log ORDER BY sequence_num DESC LIMIT 1",
	).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get last hash: %w", err)
	}
	return hash, nil
}

// VerifyChain checks hash chain integrity between two sequence numbers.
// Returns (valid, firstBrokenSequence, error).
func (r *PostgresRepository) VerifyChain(ctx context.Context, fromSequence, toSequence int64) (bool, int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, sequence_num, timestamp, user_id, action, target, target_type, details, ip_address, user_agent, result, previous_hash, entry_hash
		 FROM audit_log
		 WHERE sequence_num >= $1 AND sequence_num <= $2
		 ORDER BY sequence_num ASC`,
		fromSequence, toSequence,
	)
	if err != nil {
		return false, 0, fmt.Errorf("query chain: %w", err)
	}
	defer rows.Close()

	var entries []*models.AuditEntry
	for rows.Next() {
		var e models.AuditEntry
		if err := rows.Scan(
			&e.ID, &e.SequenceNum, &e.Timestamp, &e.UserID, &e.Action,
			&e.Target, &e.TargetType, &e.Details, &e.IPAddress, &e.UserAgent,
			&e.Result, &e.PreviousHash, &e.EntryHash,
		); err != nil {
			return false, 0, fmt.Errorf("scan chain entry: %w", err)
		}
		entries = append(entries, &e)
	}

	if err := rows.Err(); err != nil {
		return false, 0, fmt.Errorf("rows iteration: %w", err)
	}

	if len(entries) == 0 {
		return true, 0, nil
	}

	// If verifying from the very first entry, get its predecessor hash
	var expectedPreviousHash string
	if fromSequence > 1 {
		err := r.pool.QueryRow(ctx,
			"SELECT entry_hash FROM audit_log WHERE sequence_num = $1",
			fromSequence-1,
		).Scan(&expectedPreviousHash)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return false, 0, fmt.Errorf("get predecessor hash: %w", err)
		}
	}

	for i, entry := range entries {
		// Verify the previous hash link
		if i == 0 {
			if entry.PreviousHash != expectedPreviousHash {
				return false, entry.SequenceNum, nil
			}
		} else {
			if entry.PreviousHash != entries[i-1].EntryHash {
				return false, entry.SequenceNum, nil
			}
		}

		// Recompute hash and verify
		recomputed := computeEntryHash(entry, entry.PreviousHash)
		if recomputed != entry.EntryHash {
			return false, entry.SequenceNum, nil
		}
	}

	return true, 0, nil
}
