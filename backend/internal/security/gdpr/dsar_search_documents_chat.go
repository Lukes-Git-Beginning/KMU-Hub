package gdpr

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Files, comments and authorship of a matched user outside of messages and
// tasks — Art. 15 GDPR.
//
// Read-only. Only content and file NAMES of the person are disclosed, never
// file bytes or storage keys. Deleted files stay listed, with the deletion
// time in the "Gelöscht am" column, instead of being silently omitted.
// Pure structure references (folder/tag/template created_by) are not part of
// this file: the user row is anonymised on erasure and those references then
// resolve to the anonymised label.
// ---------------------------------------------------------------------------

// valuesRecord turns a row of plain columns (text, numbers, uuids, timestamps,
// bools, NULLs) into one record, keyed by keys in column order.
func valuesRecord(keys []string) func(rows pgx.Rows) (DSARRecord, error) {
	return func(rows pgx.Rows) (DSARRecord, error) {
		vals, err := rows.Values()
		if err != nil {
			return DSARRecord{}, err
		}
		if len(vals) != len(keys) {
			return DSARRecord{}, fmt.Errorf("got %d columns, want %d", len(vals), len(keys))
		}
		fields := make([]DSARField, len(keys))
		for i, v := range vals {
			fields[i] = DSARField{Key: keys[i], Value: dsarCell(v)}
		}
		return DSARRecord{Fields: fields}, nil
	}
}

// dsarCell renders one value from rows.Values() for a person to read. pgx
// hands NUMERIC back as pgtype.Numeric and UUID as [16]byte; fmt.Sprint would
// print those as a struct dump and a byte list.
func dsarCell(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case time.Time:
		return x.Format(dsarTimeLayout)
	case bool:
		if x {
			return "ja"
		}
		return "nein"
	case [16]byte:
		return uuid.UUID(x).String()
	case driver.Valuer:
		dv, err := x.Value()
		if err != nil || dv == nil {
			return ""
		}
		return fmt.Sprint(dv)
	default:
		return fmt.Sprint(x)
	}
}

// documentsOwnedModule lists files the user owns (document_files.owner_id).
// The contact-side "Dokumente" module only finds files linked to a contact.
func documentsOwnedModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Dateiname", "Typ", "Größe (Byte)", "Hochgeladen am", "Gelöscht am"}
	return userRowsModule(ctx, pool, "owned documents", "Eigene Dokumente", columns,
		`SELECT filename, mime_type, file_size, created_at,
		        CASE WHEN is_deleted THEN COALESCE(deleted_at, updated_at) END
		 FROM document_files
		 WHERE tenant_id = $1 AND owner_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(columns))
}

func documentCommentsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Dokument", "Kommentar", "Verfasst", "Geändert"}
	return userRowsModule(ctx, pool, "document comments", "Dokument-Kommentare", columns,
		`SELECT f.filename, c.content, c.created_at, c.updated_at
		 FROM document_file_comments c
		 JOIN document_files f ON f.id = c.file_id AND f.tenant_id = c.tenant_id
		 WHERE c.tenant_id = $1 AND c.author_id = $2
		 ORDER BY c.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(columns))
}

// documentSharesModule covers both directions: shares made to the user and
// shares the user made. The shared entity is a file or a folder.
func documentSharesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Rolle", "Objekt", "Berechtigung", "Freigegeben am"}
	return userRowsModule(ctx, pool, "document shares", "Dokument-Freigaben", columns,
		`SELECT CASE WHEN s.shared_with_user_id = $2 THEN 'Empfänger' ELSE 'Freigebender' END,
		        COALESCE(f.filename, fo.name, ''), s.permission, s.created_at
		 FROM document_shares s
		 LEFT JOIN document_files f ON f.id = s.entity_id AND f.tenant_id = s.tenant_id
		 LEFT JOIN document_folders fo ON fo.id = s.entity_id AND fo.tenant_id = s.tenant_id
		 WHERE s.tenant_id = $1 AND (s.shared_with_user_id = $2 OR s.shared_by = $2)
		 ORDER BY s.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(columns))
}

func chatUploadsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Dateiname", "Kanal", "Typ", "Hochgeladen am", "Gelöscht am"}
	return userRowsModule(ctx, pool, "chat uploads", "Chat-Uploads", columns,
		`SELECT cf.filename, ch.name, cf.mime_type, cf.created_at,
		        CASE WHEN cf.is_deleted THEN COALESCE(cf.deleted_at, cf.created_at) END
		 FROM chat_files cf
		 JOIN channels ch ON ch.id = cf.channel_id AND ch.tenant_id = cf.tenant_id
		 WHERE cf.tenant_id = $1 AND cf.uploaded_by = $2
		 ORDER BY cf.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(columns))
}

// chatReactionsModule lists emoji reactions the user left. The reacted-to
// message belongs to somebody else, so only its channel is named.
func chatReactionsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Reaktion", "Kanal", "Datum"}
	return userRowsModule(ctx, pool, "chat reactions", "Chat-Reaktionen", columns,
		`SELECT r.emoji, ch.name, r.created_at
		 FROM message_reactions r
		 JOIN messages m ON m.id = r.message_id AND m.tenant_id = r.tenant_id
		 JOIN channels ch ON ch.id = m.channel_id AND ch.tenant_id = m.tenant_id
		 WHERE r.tenant_id = $1 AND r.user_id = $2
		 ORDER BY r.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(columns))
}

func taskFilesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Dateiname", "Aufgabe", "Typ", "Hochgeladen am"}
	return userRowsModule(ctx, pool, "task files", "Aufgaben-Dateien", columns,
		`SELECT tf.filename, t.title, tf.mime_type, tf.created_at
		 FROM task_files tf
		 JOIN tasks t ON t.id = tf.task_id AND t.tenant_id = tf.tenant_id
		 WHERE tf.tenant_id = $1 AND tf.uploaded_by = $2
		 ORDER BY tf.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(columns))
}

// taskActivitiesModule is the audit trail of what the user changed on tasks.
// Old and new values are shown as stored (JSON text), cut to 200 characters.
func taskActivitiesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Aufgabe", "Aktion", "Feld", "Alter Wert", "Neuer Wert", "Datum"}
	return userRowsModule(ctx, pool, "task activities", "Aufgaben-Aktivitäten", columns,
		`SELECT t.title, a.action, COALESCE(a.field_name, ''),
		        COALESCE(left(a.old_value::text, 200), ''), COALESCE(left(a.new_value::text, 200), ''),
		        a.created_at
		 FROM task_activities a
		 JOIN tasks t ON t.id = a.task_id AND t.tenant_id = a.tenant_id
		 WHERE a.tenant_id = $1 AND a.actor_id = $2
		 ORDER BY a.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(columns))
}

// wikiArticlesModule discloses authorship (title, status, dates), not the
// article body — it is company knowledge, not personal data of the author.
func wikiArticlesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Titel", "Veröffentlicht", "Erstellt", "Geändert"}
	return userRowsModule(ctx, pool, "wiki articles", "Wiki-Artikel", columns,
		`SELECT title, published, created_at, updated_at
		 FROM wiki_articles
		 WHERE tenant_id = $1 AND author_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(columns))
}

func wikiAttachmentsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Artikel", "Typ", "Größe (Byte)", "Hochgeladen am"}
	return userRowsModule(ctx, pool, "wiki attachments", "Wiki-Anhänge", columns,
		`SELECT w.title, a.mime, a.size, a.created_at
		 FROM wiki_attachments a
		 JOIN wiki_articles w ON w.id = a.article_id AND w.tenant_id = a.tenant_id
		 WHERE a.tenant_id = $1 AND a.uploaded_by = $2
		 ORDER BY a.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(columns))
}

// documentChatModules collects the modules for one user; empty ones are
// skipped.
func documentChatModules(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) ([]DSARModule, error) {
	var out []DSARModule
	for _, load := range []func(context.Context, *pgxpool.Pool, uuid.UUID, uuid.UUID) (*DSARModule, error){
		documentsOwnedModule, documentCommentsModule, documentSharesModule, chatUploadsModule,
		chatReactionsModule, taskFilesModule, taskActivitiesModule, wikiArticlesModule, wikiAttachmentsModule,
	} {
		mod, err := load(ctx, pool, tenantID, userID)
		if err != nil {
			return nil, fmt.Errorf("dsar: documents and chat: %w", err)
		}
		if mod != nil {
			out = append(out, *mod)
		}
	}
	return out, nil
}
