package gdpr

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// External visitors without a contact or user row — Art. 15 GDPR.
//
// A person who only booked through the public booking page or only chatted as
// a guest exists nowhere in contacts/users, so the contact and user branches of
// SearchByQuery never see them. Each distinct visitor (grouped by e-mail; an
// e-mail-less chat guest session stands alone) is disclosed as its own subject.
// Read-only: retention/erasure for these tables is decided elsewhere.
// ---------------------------------------------------------------------------

// visitorSubject is one grouped external visitor found by name or e-mail.
type visitorSubject struct {
	key, name, email string
}

// matchVisitors runs a grouping query (columns: key, name, email) and builds one
// DSARPerson per visitor, filling the modules via load(key).
func matchVisitors(ctx context.Context, pool *pgxpool.Pool, idPrefix, groupSQL string, args []any, load func(key string) (*DSARModule, error)) ([]DSARPerson, error) {
	rows, err := pool.Query(ctx, groupSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("dsar: query %s visitors: %w", idPrefix, err)
	}
	var subjects []visitorSubject
	for rows.Next() {
		var s visitorSubject
		if scanErr := rows.Scan(&s.key, &s.name, &s.email); scanErr != nil {
			rows.Close()
			return nil, fmt.Errorf("dsar: scan %s visitor: %w", idPrefix, scanErr)
		}
		subjects = append(subjects, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: %s visitor rows: %w", idPrefix, err)
	}

	var out []DSARPerson
	for _, s := range subjects {
		mod, loadErr := load(s.key)
		if loadErr != nil {
			return nil, loadErr
		}
		if mod == nil {
			continue
		}
		out = append(out, DSARPerson{
			ID:      idPrefix + ":" + s.key,
			Name:    s.name,
			Email:   s.email,
			Avatar:  initials(s.name, ""),
			Modules: []DSARModule{*mod},
		})
	}
	return out, nil
}

// matchPublicBookingVisitors finds people who booked an appointment through a
// public booking page. public_bookings carries no contact link, so the match is
// by customer name/e-mail; all bookings of a matched e-mail are disclosed.
func matchPublicBookingVisitors(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, pattern string) ([]DSARPerson, error) {
	return matchVisitors(ctx, pool, "booking",
		`SELECT lower(customer_email),
		        (array_agg(customer_name ORDER BY created_at DESC))[1],
		        (array_agg(customer_email ORDER BY created_at DESC))[1]
		 FROM public_bookings
		 WHERE tenant_id = $1 AND (customer_name ILIKE $2 OR customer_email ILIKE $2)
		 GROUP BY lower(customer_email)
		 ORDER BY max(created_at) DESC
		 LIMIT $3`,
		[]any{tenantID, pattern, dsarMaxSubjects},
		func(key string) (*DSARModule, error) { return publicBookingsModule(ctx, pool, tenantID, key) })
}

func publicBookingsModule(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, email string) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT pb.customer_name, pb.customer_email, COALESCE(pb.customer_phone, ''),
		        COALESCE(pb.notes, ''), COALESCE(bp.company_name, ''),
		        pb.date, pb.time_slot, pb.status, pb.created_at
		 FROM public_bookings pb
		 LEFT JOIN booking_pages bp ON bp.id = pb.booking_page_id AND bp.tenant_id = pb.tenant_id
		 WHERE pb.tenant_id = $1 AND lower(pb.customer_email) = $2
		 ORDER BY pb.date DESC, pb.time_slot DESC
		 LIMIT $3`, tenantID, email, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query public bookings: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{
		Module:  "Online-Terminbuchungen",
		Columns: []string{"Name", "E-Mail", "Telefon", "Notizen", "Buchungsseite", "Datum", "Uhrzeit", "Status", "Erstellt"},
	}
	for rows.Next() {
		var name, mail, phone, notes, page, slot, status string
		var date, createdAt time.Time
		if scanErr := rows.Scan(&name, &mail, &phone, &notes, &page, &date, &slot, &status, &createdAt); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan public booking: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Name", Value: name},
			{Key: "E-Mail", Value: mail},
			{Key: "Telefon", Value: phone},
			{Key: "Notizen", Value: notes},
			{Key: "Buchungsseite", Value: page},
			{Key: "Datum", Value: date.Format(dsarDateLayout)},
			{Key: "Uhrzeit", Value: slot},
			{Key: "Status", Value: status},
			{Key: "Erstellt", Value: createdAt.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: public booking rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// guestVisitorKey groups chat guest sessions by e-mail when given. A session
// without an e-mail is its own subject: grouping by display name would hand one
// requester every "Max" who ever chatted as a guest in the tenant.
const guestVisitorKey = `COALESCE(NULLIF(lower(gs.email), ''), 'session:' || gs.id::text)`

// matchGuestSessionVisitors finds chat guests by display name or e-mail.
// token_hash is the session credential and is never selected.
func matchGuestSessionVisitors(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, pattern string) ([]DSARPerson, error) {
	return matchVisitors(ctx, pool, "guest",
		`SELECT `+guestVisitorKey+`,
		        (array_agg(gs.display_name ORDER BY gs.created_at DESC))[1],
		        COALESCE((array_agg(gs.email ORDER BY gs.created_at DESC))[1], '')
		 FROM guest_sessions gs
		 WHERE gs.tenant_id = $1 AND (gs.display_name ILIKE $2 OR gs.email ILIKE $2)
		 GROUP BY `+guestVisitorKey+`
		 ORDER BY max(gs.created_at) DESC
		 LIMIT $3`,
		[]any{tenantID, pattern, dsarMaxSubjects},
		func(key string) (*DSARModule, error) { return guestSessionsModule(ctx, pool, tenantID, key) })
}

func guestSessionsModule(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, key string) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT gs.display_name, COALESCE(gs.email, ''), COALESCE(host(gs.ip_address), ''),
		        COALESCE(gs.user_agent, ''), COALESCE(c.name, ''),
		        gs.created_at, gs.last_activity_at, gs.is_active
		 FROM guest_sessions gs
		 LEFT JOIN channels c ON c.id = gs.channel_id AND c.tenant_id = gs.tenant_id
		 WHERE gs.tenant_id = $1 AND `+guestVisitorKey+` = $2
		 ORDER BY gs.created_at DESC
		 LIMIT $3`, tenantID, key, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query guest sessions: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{
		Module:  "Chat-Gastsitzungen",
		Columns: []string{"Anzeigename", "E-Mail", "IP-Adresse", "Browser", "Kanal", "Erstellt", "Letzte Aktivität", "Aktiv"},
	}
	for rows.Next() {
		var name, mail, ip, agent, channel string
		var createdAt, lastActivity time.Time
		var active bool
		if scanErr := rows.Scan(&name, &mail, &ip, &agent, &channel, &createdAt, &lastActivity, &active); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan guest session: %w", scanErr)
		}
		activeLabel := "Nein"
		if active {
			activeLabel = "Ja"
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Anzeigename", Value: name},
			{Key: "E-Mail", Value: mail},
			{Key: "IP-Adresse", Value: ip},
			{Key: "Browser", Value: agent},
			{Key: "Kanal", Value: channel},
			{Key: "Erstellt", Value: createdAt.Format(dsarTimeLayout)},
			{Key: "Letzte Aktivität", Value: lastActivity.Format(dsarTimeLayout)},
			{Key: "Aktiv", Value: activeLabel},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: guest session rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}
