package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL message repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, msg *models.EmailMessage) error {
	toJSON, _ := json.Marshal(msg.ToAddresses)
	ccJSON, _ := json.Marshal(msg.CcAddresses)
	bccJSON, _ := json.Marshal(msg.BccAddresses)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_messages (id, tenant_id, account_id, folder_id, uid, message_id,
			in_reply_to, "references", thread_id, from_name, from_email,
			to_addresses, cc_addresses, bcc_addresses, subject, preview,
			body_text, body_html, is_read, is_starred, is_draft,
			has_attachments, date, size_bytes, raw_headers, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)`,
		msg.ID, msg.TenantID, msg.AccountID, msg.FolderID, msg.UID, msg.MessageID,
		msg.InReplyTo, msg.References, msg.ThreadID, msg.FromName, msg.FromEmail,
		toJSON, ccJSON, bccJSON, msg.Subject, msg.Preview,
		msg.BodyText, msg.BodyHTML, msg.IsRead, msg.IsStarred, msg.IsDraft,
		msg.HasAttachments, msg.Date, msg.SizeBytes, msg.RawHeaders,
		msg.CreatedAt, msg.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailMessage, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, account_id, folder_id, uid, message_id, in_reply_to,
			"references", thread_id, from_name, from_email, to_addresses,
			cc_addresses, bcc_addresses, subject, preview, body_text, body_html,
			is_read, is_starred, is_draft, has_attachments, date, size_bytes,
			raw_headers, label_ids, created_at, updated_at
		 FROM email_messages WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return scanMessage(row)
}

func (r *PostgresRepository) GetByFolderUID(ctx context.Context, folderID uuid.UUID, uid uint32) (*models.EmailMessage, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, account_id, folder_id, uid, message_id, in_reply_to,
			"references", thread_id, from_name, from_email, to_addresses,
			cc_addresses, bcc_addresses, subject, preview, body_text, body_html,
			is_read, is_starred, is_draft, has_attachments, date, size_bytes,
			raw_headers, label_ids, created_at, updated_at
		 FROM email_messages WHERE folder_id = $1 AND uid = $2`, folderID, uid,
	)
	return scanMessage(row)
}

func (r *PostgresRepository) ListByFolder(ctx context.Context, folderID uuid.UUID, opts ListOpts) ([]*models.EmailMessage, int, error) {
	if opts.PerPage <= 0 || opts.PerPage > 100 {
		opts.PerPage = 50
	}
	if opts.Page < 1 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.PerPage

	// Count total
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_messages WHERE folder_id = $1`, folderID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	orderDir := "DESC"
	if strings.ToUpper(opts.SortDir) == "ASC" {
		orderDir = "ASC"
	}

	rows, err := r.pool.Query(ctx,
		fmt.Sprintf(`SELECT id, account_id, folder_id, uid, message_id, in_reply_to,
			"references", thread_id, from_name, from_email, to_addresses,
			cc_addresses, bcc_addresses, subject, preview, body_text, body_html,
			is_read, is_starred, is_draft, has_attachments, date, size_bytes,
			raw_headers, label_ids, created_at, updated_at
		 FROM email_messages WHERE folder_id = $1
		 ORDER BY date %s LIMIT $2 OFFSET $3`, orderDir),
		folderID, opts.PerPage, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	messages, err := collectMessages(rows)
	return messages, total, err
}

func (r *PostgresRepository) ListByThread(ctx context.Context, threadID uuid.UUID) ([]*models.EmailMessage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, account_id, folder_id, uid, message_id, in_reply_to,
			"references", thread_id, from_name, from_email, to_addresses,
			cc_addresses, bcc_addresses, subject, preview, body_text, body_html,
			is_read, is_starred, is_draft, has_attachments, date, size_bytes,
			raw_headers, label_ids, created_at, updated_at
		 FROM email_messages WHERE thread_id = $1
		 ORDER BY date ASC`, threadID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectMessages(rows)
}

func (r *PostgresRepository) Search(ctx context.Context, accountID uuid.UUID, query string, opts ListOpts) ([]*models.EmailMessage, int, error) {
	if opts.PerPage <= 0 || opts.PerPage > 100 {
		opts.PerPage = 50
	}
	if opts.Page < 1 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.PerPage

	// Count total matching results
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_messages
		 WHERE account_id = $1 AND search_vector @@ plainto_tsquery('german', $2)`,
		accountID, query,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, account_id, folder_id, uid, message_id, in_reply_to,
			"references", thread_id, from_name, from_email, to_addresses,
			cc_addresses, bcc_addresses, subject, preview, body_text, body_html,
			is_read, is_starred, is_draft, has_attachments, date, size_bytes,
			raw_headers, label_ids, created_at, updated_at
		 FROM email_messages
		 WHERE account_id = $1 AND search_vector @@ plainto_tsquery('german', $2)
		 ORDER BY ts_rank(search_vector, plainto_tsquery('german', $2)) DESC
		 LIMIT $3 OFFSET $4`,
		accountID, query, opts.PerPage, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	messages, err := collectMessages(rows)
	return messages, total, err
}

func (r *PostgresRepository) UpdateFlags(ctx context.Context, id uuid.UUID, isRead, isStarred *bool) error {
	var setClauses []string
	var args []any
	argNum := 2

	args = append(args, id)

	if isRead != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_read = $%d", argNum))
		args = append(args, *isRead)
		argNum++
	}
	if isStarred != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_starred = $%d", argNum))
		args = append(args, *isStarred)
		argNum++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now().UTC())

	query := fmt.Sprintf("UPDATE email_messages SET %s WHERE id = $1", strings.Join(setClauses, ", "))
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *PostgresRepository) UpdateThreadID(ctx context.Context, id uuid.UUID, threadID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_messages SET thread_id = $2, updated_at = $3 WHERE id = $1`,
		id, threadID, time.Now().UTC(),
	)
	return err
}

func (r *PostgresRepository) MoveToFolder(ctx context.Context, id uuid.UUID, folderID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_messages SET folder_id = $2, updated_at = $3 WHERE id = $1`,
		id, folderID, time.Now().UTC(),
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_messages WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) DeleteByFolder(ctx context.Context, folderID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_messages WHERE folder_id = $1`, folderID)
	return err
}

func (r *PostgresRepository) GetHighestUID(ctx context.Context, folderID uuid.UUID) (uint32, error) {
	var uid *int64
	err := r.pool.QueryRow(ctx,
		`SELECT MAX(uid) FROM email_messages WHERE folder_id = $1`, folderID,
	).Scan(&uid)
	if err != nil {
		return 0, err
	}
	if uid == nil {
		return 0, nil
	}
	return uint32(*uid), nil
}

func (r *PostgresRepository) GetByMessageIDHeader(ctx context.Context, messageIDHeader string) (*models.EmailMessage, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, account_id, folder_id, uid, message_id, in_reply_to,
			"references", thread_id, from_name, from_email, to_addresses,
			cc_addresses, bcc_addresses, subject, preview, body_text, body_html,
			is_read, is_starred, is_draft, has_attachments, date, size_bytes,
			raw_headers, label_ids, created_at, updated_at
		 FROM email_messages WHERE message_id = $1
		 ORDER BY date DESC LIMIT 1`, messageIDHeader,
	)
	return scanMessage(row)
}

func (r *PostgresRepository) FindThreadByReferences(ctx context.Context, accountID uuid.UUID, references []string, inReplyTo string) (uuid.UUID, error) {
	// First try InReplyTo
	if inReplyTo != "" {
		var threadID uuid.UUID
		err := r.pool.QueryRow(ctx,
			`SELECT thread_id FROM email_messages
			 WHERE account_id = $1 AND message_id = $2 AND thread_id IS NOT NULL
			 LIMIT 1`,
			accountID, inReplyTo,
		).Scan(&threadID)
		if err == nil {
			return threadID, nil
		}
	}

	// Then try references (last element first)
	for i := len(references) - 1; i >= 0; i-- {
		var threadID uuid.UUID
		err := r.pool.QueryRow(ctx,
			`SELECT thread_id FROM email_messages
			 WHERE account_id = $1 AND message_id = $2 AND thread_id IS NOT NULL
			 LIMIT 1`,
			accountID, references[i],
		).Scan(&threadID)
		if err == nil {
			return threadID, nil
		}
	}

	return uuid.Nil, ErrThreadNotFound
}

func (r *PostgresRepository) CountUnreadByFolder(ctx context.Context, folderID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_messages WHERE folder_id = $1 AND is_read = false`,
		folderID,
	).Scan(&count)
	return count, err
}

func (r *PostgresRepository) FindBySubjectAndParticipants(ctx context.Context, accountID uuid.UUID, normalizedSubject string, fromEmail string, since, until interface{}) (*models.EmailMessage, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, account_id, folder_id, uid, message_id, in_reply_to,
			"references", thread_id, from_name, from_email, to_addresses,
			cc_addresses, bcc_addresses, subject, preview, body_text, body_html,
			is_read, is_starred, is_draft, has_attachments, date, size_bytes,
			raw_headers, label_ids, created_at, updated_at
		 FROM email_messages
		 WHERE account_id = $1 AND thread_id IS NOT NULL
			AND LOWER(subject) LIKE '%' || LOWER($2) || '%'
			AND (from_email = $3 OR to_addresses::text LIKE '%' || $3 || '%')
			AND date BETWEEN $4 AND $5
		 ORDER BY date DESC LIMIT 1`,
		accountID, normalizedSubject, fromEmail, since, until,
	)
	return scanMessage(row)
}

// --- Folder Repository ---

// PostgresFolderRepository implements FolderRepository using PostgreSQL.
type PostgresFolderRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresFolderRepository creates a new PostgreSQL folder repository.
func NewPostgresFolderRepository(pool *pgxpool.Pool) *PostgresFolderRepository {
	return &PostgresFolderRepository{pool: pool}
}

func (r *PostgresFolderRepository) Create(ctx context.Context, folder *models.EmailFolder) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_folders (id, account_id, name, imap_name, folder_type,
			uid_validity, highest_uid, message_count, unread_count, sort_order,
			created_at, updated_at, tenant_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
		         (SELECT tenant_id FROM email_accounts WHERE id = $2))`,
		folder.ID, folder.AccountID, folder.Name, folder.IMAPName, folder.FolderType,
		folder.UIDValidity, folder.HighestUID, folder.MessageCount, folder.UnreadCount,
		folder.SortOrder, folder.CreatedAt, folder.UpdatedAt,
	)
	return err
}

func (r *PostgresFolderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.EmailFolder, error) {
	var f models.EmailFolder
	err := r.pool.QueryRow(ctx,
		`SELECT id, account_id, name, imap_name, folder_type, uid_validity,
			highest_uid, message_count, unread_count, sort_order, created_at, updated_at
		 FROM email_folders WHERE id = $1`, id,
	).Scan(&f.ID, &f.AccountID, &f.Name, &f.IMAPName, &f.FolderType,
		&f.UIDValidity, &f.HighestUID, &f.MessageCount, &f.UnreadCount,
		&f.SortOrder, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFolderNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *PostgresFolderRepository) GetByIMAPName(ctx context.Context, accountID uuid.UUID, imapName string) (*models.EmailFolder, error) {
	var f models.EmailFolder
	err := r.pool.QueryRow(ctx,
		`SELECT id, account_id, name, imap_name, folder_type, uid_validity,
			highest_uid, message_count, unread_count, sort_order, created_at, updated_at
		 FROM email_folders WHERE account_id = $1 AND imap_name = $2`,
		accountID, imapName,
	).Scan(&f.ID, &f.AccountID, &f.Name, &f.IMAPName, &f.FolderType,
		&f.UIDValidity, &f.HighestUID, &f.MessageCount, &f.UnreadCount,
		&f.SortOrder, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFolderNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *PostgresFolderRepository) ListByAccount(ctx context.Context, accountID uuid.UUID) ([]*models.EmailFolder, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, account_id, name, imap_name, folder_type, uid_validity,
			highest_uid, message_count, unread_count, sort_order, created_at, updated_at
		 FROM email_folders WHERE account_id = $1
		 ORDER BY sort_order ASC`, accountID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*models.EmailFolder
	for rows.Next() {
		var f models.EmailFolder
		err := rows.Scan(&f.ID, &f.AccountID, &f.Name, &f.IMAPName, &f.FolderType,
			&f.UIDValidity, &f.HighestUID, &f.MessageCount, &f.UnreadCount,
			&f.SortOrder, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		folders = append(folders, &f)
	}
	return folders, rows.Err()
}

func (r *PostgresFolderRepository) GetByAccountAndType(ctx context.Context, accountID uuid.UUID, folderType string) (*models.EmailFolder, error) {
	var f models.EmailFolder
	err := r.pool.QueryRow(ctx,
		`SELECT id, account_id, name, imap_name, folder_type, uid_validity,
			highest_uid, message_count, unread_count, sort_order, created_at, updated_at
		 FROM email_folders WHERE account_id = $1 AND folder_type = $2
		 ORDER BY sort_order ASC LIMIT 1`, accountID, folderType,
	).Scan(&f.ID, &f.AccountID, &f.Name, &f.IMAPName, &f.FolderType,
		&f.UIDValidity, &f.HighestUID, &f.MessageCount, &f.UnreadCount,
		&f.SortOrder, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFolderNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *PostgresFolderRepository) UpdateCounts(ctx context.Context, folderID uuid.UUID, messageCount, unreadCount int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_folders SET message_count = $2, unread_count = $3, updated_at = $4 WHERE id = $1`,
		folderID, messageCount, unreadCount, time.Now().UTC(),
	)
	return err
}

func (r *PostgresFolderRepository) UpdateUIDValidity(ctx context.Context, folderID uuid.UUID, uidValidity int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_folders SET uid_validity = $2, updated_at = $3 WHERE id = $1`,
		folderID, uidValidity, time.Now().UTC(),
	)
	return err
}

func (r *PostgresFolderRepository) DeleteMessagesByFolder(ctx context.Context, folderID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_messages WHERE folder_id = $1`, folderID)
	return err
}

// --- Scan helpers ---

func scanMessage(row pgx.Row) (*models.EmailMessage, error) {
	var msg models.EmailMessage
	var toJSON, ccJSON, bccJSON []byte

	err := row.Scan(
		&msg.ID, &msg.AccountID, &msg.FolderID, &msg.UID, &msg.MessageID,
		&msg.InReplyTo, &msg.References, &msg.ThreadID, &msg.FromName, &msg.FromEmail,
		&toJSON, &ccJSON, &bccJSON, &msg.Subject, &msg.Preview,
		&msg.BodyText, &msg.BodyHTML, &msg.IsRead, &msg.IsStarred, &msg.IsDraft,
		&msg.HasAttachments, &msg.Date, &msg.SizeBytes, &msg.RawHeaders,
		&msg.LabelIDs, &msg.CreatedAt, &msg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	// Parse JSON address arrays
	if len(toJSON) > 0 {
		if err := json.Unmarshal(toJSON, &msg.ToAddresses); err != nil {
			return nil, fmt.Errorf("failed to parse To addresses for message %s: %w", msg.ID, err)
		}
	}
	if len(ccJSON) > 0 {
		if err := json.Unmarshal(ccJSON, &msg.CcAddresses); err != nil {
			return nil, fmt.Errorf("failed to parse Cc addresses for message %s: %w", msg.ID, err)
		}
	}
	if len(bccJSON) > 0 {
		if err := json.Unmarshal(bccJSON, &msg.BccAddresses); err != nil {
			return nil, fmt.Errorf("failed to parse Bcc addresses for message %s: %w", msg.ID, err)
		}
	}

	return &msg, nil
}

func collectMessages(rows pgx.Rows) ([]*models.EmailMessage, error) {
	var messages []*models.EmailMessage
	for rows.Next() {
		var msg models.EmailMessage
		var toJSON, ccJSON, bccJSON []byte

		err := rows.Scan(
			&msg.ID, &msg.AccountID, &msg.FolderID, &msg.UID, &msg.MessageID,
			&msg.InReplyTo, &msg.References, &msg.ThreadID, &msg.FromName, &msg.FromEmail,
			&toJSON, &ccJSON, &bccJSON, &msg.Subject, &msg.Preview,
			&msg.BodyText, &msg.BodyHTML, &msg.IsRead, &msg.IsStarred, &msg.IsDraft,
			&msg.HasAttachments, &msg.Date, &msg.SizeBytes, &msg.RawHeaders,
			&msg.LabelIDs, &msg.CreatedAt, &msg.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(toJSON) > 0 {
			if err := json.Unmarshal(toJSON, &msg.ToAddresses); err != nil {
				return nil, fmt.Errorf("failed to parse To addresses for message %s: %w", msg.ID, err)
			}
		}
		if len(ccJSON) > 0 {
			if err := json.Unmarshal(ccJSON, &msg.CcAddresses); err != nil {
				return nil, fmt.Errorf("failed to parse Cc addresses for message %s: %w", msg.ID, err)
			}
		}
		if len(bccJSON) > 0 {
			if err := json.Unmarshal(bccJSON, &msg.BccAddresses); err != nil {
				return nil, fmt.Errorf("failed to parse Bcc addresses for message %s: %w", msg.ID, err)
			}
		}

		messages = append(messages, &msg)
	}
	return messages, rows.Err()
}
