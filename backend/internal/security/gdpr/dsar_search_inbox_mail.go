package gdpr

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Unified inbox and mail accounts of a matched user — Art. 15 GDPR.
//
// Read-only. Secrets stay out: email_accounts.password_encrypted is never
// selected. Mailbox content (email_messages) is not disclosed here; it is
// reachable through the contact branch via email_contact_links.
// ---------------------------------------------------------------------------

// userRowsModule runs a tenant-scoped query and turns every row into one
// record via scan. Returns nil when the user has no rows.
func userRowsModule(ctx context.Context, pool *pgxpool.Pool, what, module string, columns []string, sql string, args []any, scan func(rows pgx.Rows) (DSARRecord, error)) (*DSARModule, error) {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("dsar: query %s: %w", what, err)
	}
	defer rows.Close()

	mod := DSARModule{Module: module, Columns: columns}
	for rows.Next() {
		rec, scanErr := scan(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("dsar: scan %s: %w", what, scanErr)
		}
		mod.Records = append(mod.Records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: %s rows: %w", what, err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// inboxMessagesModule discloses unified-inbox entries the user owns, is
// assigned to, or sent. One entry can be several of these at once.
func inboxMessagesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Rolle", "Kanal", "Absender", "Absender-E-Mail", "Betreff", "Vorschau", "Empfangen"}
	return userRowsModule(ctx, pool, "inbox messages", "Posteingang", columns,
		`SELECT user_id = $2, assigned_to = $2, sender_id = $2, channel, sender_name,
		        COALESCE(sender_email, ''), subject, preview, received_at
		 FROM inbox_messages
		 WHERE tenant_id = $1 AND (user_id = $2 OR assigned_to = $2 OR sender_id = $2)
		 ORDER BY received_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			// assigned_to and sender_id are nullable, so their comparison is too.
			var owner, assignee, sender *bool
			var channel, senderName, senderEmail, subject, preview string
			var receivedAt time.Time
			if err := rows.Scan(&owner, &assignee, &sender, &channel, &senderName, &senderEmail, &subject, &preview, &receivedAt); err != nil {
				return DSARRecord{}, err
			}
			var roles []string
			for _, r := range []struct {
				is    *bool
				label string
			}{{owner, "Postfach-Inhaber"}, {assignee, "Zugewiesen"}, {sender, "Absender"}} {
				if r.is != nil && *r.is {
					roles = append(roles, r.label)
				}
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Rolle", Value: strings.Join(roles, ", ")},
				{Key: "Kanal", Value: channel},
				{Key: "Absender", Value: senderName},
				{Key: "Absender-E-Mail", Value: senderEmail},
				{Key: "Betreff", Value: subject},
				{Key: "Vorschau", Value: preview},
				{Key: "Empfangen", Value: receivedAt.Format(dsarTimeLayout)},
			}}, nil
		})
}

// inboxThreadMessagesModule discloses the replies the user wrote in inbox
// conversations (inbox_thread_messages.author_id), not the other party's.
func inboxThreadMessagesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Betreff", "Richtung", "Text", "Verfasst"}
	return userRowsModule(ctx, pool, "inbox thread messages", "Posteingang-Antworten", columns,
		`SELECT COALESCE(m.subject, ''), tm.direction, tm.body, tm.created_at
		 FROM inbox_thread_messages tm
		 LEFT JOIN inbox_messages m ON m.id = tm.message_id AND m.tenant_id = tm.tenant_id
		 WHERE tm.tenant_id = $1 AND tm.author_id = $2
		 ORDER BY tm.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var subject, direction, body string
			var createdAt time.Time
			if err := rows.Scan(&subject, &direction, &body, &createdAt); err != nil {
				return DSARRecord{}, err
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Betreff", Value: subject},
				{Key: "Richtung", Value: direction},
				{Key: "Text", Value: body},
				{Key: "Verfasst", Value: createdAt.Format(dsarTimeLayout)},
			}}, nil
		})
}

func teamInboxMembershipsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Team-Posteingang", "Rolle", "Mitglied seit"}
	return userRowsModule(ctx, pool, "team inbox memberships", "Team-Posteingänge", columns,
		`SELECT ti.name, tim.role, tim.created_at
		 FROM team_inbox_members tim
		 JOIN team_inboxes ti ON ti.id = tim.team_inbox_id AND ti.tenant_id = tim.tenant_id
		 WHERE tim.tenant_id = $1 AND tim.user_id = $2
		 ORDER BY tim.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var name, role string
			var createdAt time.Time
			if err := rows.Scan(&name, &role, &createdAt); err != nil {
				return DSARRecord{}, err
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Team-Posteingang", Value: name},
				{Key: "Rolle", Value: role},
				{Key: "Mitglied seit", Value: createdAt.Format(dsarTimeLayout)},
			}}, nil
		})
}

// emailAccountsModule discloses connected mailboxes. Credentials
// (password_encrypted) are deliberately not selected.
func emailAccountsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"E-Mail-Adresse", "Anzeigename", "Benutzername", "IMAP-Server", "SMTP-Server", "Synchronisiert", "Letzte Synchronisierung", "Angelegt"}
	return userRowsModule(ctx, pool, "email accounts", "E-Mail-Konten", columns,
		`SELECT email_address, COALESCE(display_name, ''), username, imap_host, smtp_host,
		        sync_enabled, last_sync_at, created_at
		 FROM email_accounts
		 WHERE tenant_id = $1 AND user_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var address, displayName, username, imapHost, smtpHost string
			var syncEnabled bool
			var lastSync *time.Time
			var createdAt time.Time
			if err := rows.Scan(&address, &displayName, &username, &imapHost, &smtpHost, &syncEnabled, &lastSync, &createdAt); err != nil {
				return DSARRecord{}, err
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "E-Mail-Adresse", Value: address},
				{Key: "Anzeigename", Value: displayName},
				{Key: "Benutzername", Value: username},
				{Key: "IMAP-Server", Value: imapHost},
				{Key: "SMTP-Server", Value: smtpHost},
				{Key: "Synchronisiert", Value: boolLabel(syncEnabled)},
				{Key: "Letzte Synchronisierung", Value: derefDateTimeOrEmpty(lastSync)},
				{Key: "Angelegt", Value: createdAt.Format(dsarTimeLayout)},
			}}, nil
		})
}

func emailSignaturesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Name", "Standard", "Inhalt", "Geändert"}
	return userRowsModule(ctx, pool, "email signatures", "E-Mail-Signaturen", columns,
		`SELECT name, is_default, html_content, updated_at
		 FROM email_signatures
		 WHERE tenant_id = $1 AND user_id = $2
		 ORDER BY updated_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var name, content string
			var isDefault bool
			var updatedAt time.Time
			if err := rows.Scan(&name, &isDefault, &content, &updatedAt); err != nil {
				return DSARRecord{}, err
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Name", Value: name},
				{Key: "Standard", Value: boolLabel(isDefault)},
				{Key: "Inhalt", Value: content},
				{Key: "Geändert", Value: updatedAt.Format(dsarTimeLayout)},
			}}, nil
		})
}

// inboxMailModules collects the five modules for one user; empty ones are
// skipped.
func inboxMailModules(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) ([]DSARModule, error) {
	var out []DSARModule
	for _, load := range []func(context.Context, *pgxpool.Pool, uuid.UUID, uuid.UUID) (*DSARModule, error){
		inboxMessagesModule, inboxThreadMessagesModule, teamInboxMembershipsModule,
		emailAccountsModule, emailSignaturesModule,
	} {
		mod, err := load(ctx, pool, tenantID, userID)
		if err != nil {
			return nil, err
		}
		if mod != nil {
			out = append(out, *mod)
		}
	}
	return out, nil
}
