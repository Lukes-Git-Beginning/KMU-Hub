package gdpr

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// DSAR (Data Subject Access Request) cross-module search — Art. 15 GDPR.
// ---------------------------------------------------------------------------

// DSARField is a single key/value cell within a DSAR record. Order matters, so
// records are modelled as ordered field lists rather than maps.
type DSARField struct {
	Key   string
	Value string
}

// DSARRecord is one logical row of disclosed data.
type DSARRecord struct {
	Fields []DSARField
}

// DSARModule groups a data subject's disclosed records from one module.
type DSARModule struct {
	Module  string
	Columns []string
	Records []DSARRecord
}

// DSARPerson is a matched data subject and the data held about them.
type DSARPerson struct {
	ID      string
	Name    string
	Email   string
	Company string
	Avatar  string
	Modules []DSARModule
}

const (
	dsarMaxSubjects = 10
	dsarMaxRows     = 50
	dsarTimeLayout  = "2006-01-02 15:04"
)

// SearchByQuery performs an Art. 15 GDPR cross-module lookup for a person within
// a tenant. It matches contacts and users by name/email and, for each match,
// aggregates the data held about them across CRM, consent and dialer modules.
// Coverage is a deliberate subset (contacts + users + consent + dialer); other
// modules can be folded in incrementally. All queries are tenant-scoped and run
// behind RLS as defense-in-depth.
func SearchByQuery(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, query string) ([]DSARPerson, error) {
	pattern := "%" + query + "%"
	persons := make([]DSARPerson, 0)

	contacts, err := matchContacts(ctx, pool, tenantID, pattern)
	if err != nil {
		return nil, err
	}
	for _, c := range contacts {
		name := joinName(c.first, c.last)
		person := DSARPerson{
			ID:      c.id.String(),
			Name:    name,
			Email:   c.email,
			Company: c.company,
			Avatar:  initials(c.first, c.last),
			Modules: []DSARModule{{
				Module:  "CRM Kontakte",
				Columns: []string{"Feld", "Wert"},
				Records: []DSARRecord{
					fieldValueRecord("Name", name),
					fieldValueRecord("E-Mail", c.email),
					fieldValueRecord("Telefon", c.phone),
					fieldValueRecord("Position", c.position),
					fieldValueRecord("Unternehmen", c.company),
					fieldValueRecord("Erstellt", c.createdAt.Format(dsarTimeLayout)),
				},
			}},
		}

		consent, cErr := consentModule(ctx, pool, c.id)
		if cErr != nil {
			return nil, cErr
		}
		if consent != nil {
			person.Modules = append(person.Modules, *consent)
		}

		calls, dErr := dialerModule(ctx, pool, tenantID, c.id)
		if dErr != nil {
			return nil, dErr
		}
		if calls != nil {
			person.Modules = append(person.Modules, *calls)
		}

		persons = append(persons, person)
	}

	users, err := matchUsers(ctx, pool, tenantID, pattern)
	if err != nil {
		return nil, err
	}
	persons = append(persons, users...)

	return persons, nil
}

// ---------------------------------------------------------------------------
// Subject matching
// ---------------------------------------------------------------------------

type contactRow struct {
	id                                              uuid.UUID
	first, last, email, phone, position, company string
	createdAt                                       time.Time
}

func matchContacts(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, pattern string) ([]contactRow, error) {
	rows, err := pool.Query(ctx,
		`SELECT c.id, c.first_name, c.last_name, COALESCE(c.email, ''),
		        COALESCE(c.phone, ''), COALESCE(c.position, ''), COALESCE(co.name, ''), c.created_at
		 FROM contacts c
		 LEFT JOIN companies co ON co.id = c.company_id
		 WHERE c.tenant_id = $1
		   AND (c.first_name ILIKE $2 OR c.last_name ILIKE $2 OR c.email ILIKE $2
		        OR (c.first_name || ' ' || c.last_name) ILIKE $2)
		 ORDER BY c.created_at DESC
		 LIMIT $3`, tenantID, pattern, dsarMaxSubjects)
	if err != nil {
		return nil, fmt.Errorf("dsar: query contacts: %w", err)
	}
	defer rows.Close()

	var out []contactRow
	for rows.Next() {
		var c contactRow
		if scanErr := rows.Scan(&c.id, &c.first, &c.last, &c.email, &c.phone, &c.position, &c.company, &c.createdAt); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan contact: %w", scanErr)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: contact rows: %w", err)
	}
	return out, nil
}

func matchUsers(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, pattern string) ([]DSARPerson, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, COALESCE(email, ''), first_name, last_name, is_active, created_at
		 FROM users
		 WHERE tenant_id = $1
		   AND (first_name ILIKE $2 OR last_name ILIKE $2 OR email ILIKE $2
		        OR (first_name || ' ' || last_name) ILIKE $2)
		 ORDER BY created_at DESC
		 LIMIT $3`, tenantID, pattern, dsarMaxSubjects)
	if err != nil {
		return nil, fmt.Errorf("dsar: query users: %w", err)
	}
	defer rows.Close()

	var out []DSARPerson
	for rows.Next() {
		var id uuid.UUID
		var email, first, last string
		var isActive bool
		var createdAt time.Time
		if scanErr := rows.Scan(&id, &email, &first, &last, &isActive, &createdAt); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan user: %w", scanErr)
		}
		name := joinName(first, last)
		out = append(out, DSARPerson{
			ID:     id.String(),
			Name:   name,
			Email:  email,
			Avatar: initials(first, last),
			Modules: []DSARModule{{
				Module:  "Benutzerkonto",
				Columns: []string{"Feld", "Wert"},
				Records: []DSARRecord{
					fieldValueRecord("Name", name),
					fieldValueRecord("E-Mail", email),
					fieldValueRecord("Aktiv", boolLabel(isActive)),
					fieldValueRecord("Erstellt", createdAt.Format(dsarTimeLayout)),
				},
			}},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: user rows: %w", err)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Per-module aggregation for a matched contact
// ---------------------------------------------------------------------------

func consentModule(ctx context.Context, pool *pgxpool.Pool, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT consent_type::text, granted, legal_basis::text,
		        COALESCE(granted_at, created_at), revoked_at
		 FROM consent_records
		 WHERE contact_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`, contactID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query consent: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Einwilligungen", Columns: []string{"Typ", "Status", "Rechtsgrundlage", "Datum"}}
	for rows.Next() {
		var ctype, basis string
		var granted bool
		var at time.Time
		var revoked *time.Time
		if scanErr := rows.Scan(&ctype, &granted, &basis, &at, &revoked); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan consent: %w", scanErr)
		}
		status := "Erteilt"
		if !granted {
			status = "Verweigert"
		}
		if revoked != nil {
			status = "Widerrufen"
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Typ", Value: ctype},
			{Key: "Status", Value: status},
			{Key: "Rechtsgrundlage", Value: basis},
			{Key: "Datum", Value: at.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: consent rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func dialerModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT dcs.created_at, COALESCE(dcs.duration_seconds, 0), COALESCE(dcs.notes, '')
		 FROM dialer_call_sessions dcs
		 JOIN dialer_campaign_contacts dcc ON dcc.id = dcs.campaign_contact_id
		 WHERE dcc.contact_id = $1 AND dcc.tenant_id = $2
		 ORDER BY dcs.created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query dialer: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Anrufe", Columns: []string{"Datum", "Dauer (s)", "Notiz"}}
	for rows.Next() {
		var at time.Time
		var dur int
		var notes string
		if scanErr := rows.Scan(&at, &dur, &notes); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan dialer: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Datum", Value: at.Format(dsarTimeLayout)},
			{Key: "Dauer (s)", Value: fmt.Sprintf("%d", dur)},
			{Key: "Notiz", Value: notes},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: dialer rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func fieldValueRecord(field, value string) DSARRecord {
	return DSARRecord{Fields: []DSARField{
		{Key: "Feld", Value: field},
		{Key: "Wert", Value: value},
	}}
}

func joinName(first, last string) string {
	name := strings.TrimSpace(first + " " + last)
	if name == "" {
		return "—"
	}
	return name
}

func initials(first, last string) string {
	var b strings.Builder
	if first != "" {
		b.WriteString(string([]rune(first)[0:1]))
	}
	if last != "" {
		b.WriteString(string([]rune(last)[0:1]))
	}
	if b.Len() == 0 {
		return "?"
	}
	return strings.ToUpper(b.String())
}

func boolLabel(v bool) string {
	if v {
		return "Ja"
	}
	return "Nein"
}
