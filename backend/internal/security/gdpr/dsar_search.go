package gdpr

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	dsarDateLayout  = "2006-01-02"
)

// SearchByQuery performs an Art. 15 GDPR cross-module lookup for a person within
// a tenant. It matches contacts and users by name/email and, for each match,
// aggregates the data held about them across CRM, consent, dialer, finance,
// meetings, helpdesk, contracts, email, deals and activities modules. Other
// modules can still be folded in incrementally. All queries are tenant-scoped
// and run behind RLS as defense-in-depth.
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

		customFields, cfErr := customFieldsModule(ctx, pool, tenantID, c.id)
		if cfErr != nil {
			return nil, cfErr
		}
		if customFields != nil {
			person.Modules = append(person.Modules, *customFields)
		}

		tags, tgErr := tagsModule(ctx, pool, tenantID, c.id)
		if tgErr != nil {
			return nil, tgErr
		}
		if tags != nil {
			person.Modules = append(person.Modules, *tags)
		}

		documents, docErr := documentsModule(ctx, pool, tenantID, c.id)
		if docErr != nil {
			return nil, docErr
		}
		if documents != nil {
			person.Modules = append(person.Modules, *documents)
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

		invoices, fErr := financeModule(ctx, pool, tenantID, c.id)
		if fErr != nil {
			return nil, fErr
		}
		if invoices != nil {
			person.Modules = append(person.Modules, *invoices)
		}

		meetings, mErr := meetingsModule(ctx, pool, tenantID, c.id)
		if mErr != nil {
			return nil, mErr
		}
		if meetings != nil {
			person.Modules = append(person.Modules, *meetings)
		}

		tickets, tErr := helpdeskModule(ctx, pool, tenantID, c.id)
		if tErr != nil {
			return nil, tErr
		}
		if tickets != nil {
			person.Modules = append(person.Modules, *tickets)
		}

		helpdeskMessages, hmErr := helpdeskMessagesModule(ctx, pool, tenantID, c.id)
		if hmErr != nil {
			return nil, hmErr
		}
		if helpdeskMessages != nil {
			person.Modules = append(person.Modules, *helpdeskMessages)
		}

		contracts, ctErr := contractsModule(ctx, pool, tenantID, c.id)
		if ctErr != nil {
			return nil, ctErr
		}
		if contracts != nil {
			person.Modules = append(person.Modules, *contracts)
		}

		emails, eErr := emailModule(ctx, pool, tenantID, c.id)
		if eErr != nil {
			return nil, eErr
		}
		if emails != nil {
			person.Modules = append(person.Modules, *emails)
		}

		deals, dlErr := dealsModule(ctx, pool, tenantID, c.id)
		if dlErr != nil {
			return nil, dlErr
		}
		if deals != nil {
			person.Modules = append(person.Modules, *deals)
		}

		formSubmissions, fsErr := formSubmissionsModule(ctx, pool, tenantID, c.email)
		if fsErr != nil {
			return nil, fsErr
		}
		if formSubmissions != nil {
			person.Modules = append(person.Modules, *formSubmissions)
		}

		activities, acErr := activitiesModule(ctx, pool, tenantID, c.id)
		if acErr != nil {
			return nil, acErr
		}
		if activities != nil {
			person.Modules = append(person.Modules, *activities)
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
	id                                           uuid.UUID
	first, last, email, phone, position, company string
	createdAt                                    time.Time
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

// customFieldsModule reads a contact's custom field values. There is no
// tenant_id on contact_custom_field_values itself -- its RLS policy scopes
// via a join back to contacts, so this query joins the same way rather than
// relying on RLS alone (defense-in-depth, matching every other module here).
func customFieldsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT cfd.field_label, cfv.value
		 FROM contact_custom_field_values cfv
		 JOIN custom_field_definitions cfd ON cfd.id = cfv.field_id
		 JOIN contacts c ON c.id = cfv.contact_id
		 WHERE cfv.contact_id = $1 AND c.tenant_id = $2
		 ORDER BY cfd.sort_order, cfd.field_label
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query custom fields: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Benutzerdefinierte Felder", Columns: []string{"Feld", "Wert"}}
	for rows.Next() {
		var label string
		var valueJSON []byte
		if scanErr := rows.Scan(&label, &valueJSON); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan custom field: %w", scanErr)
		}
		mod.Records = append(mod.Records, fieldValueRecord(label, formatCustomFieldValue(valueJSON)))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: custom field rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// formatCustomFieldValue renders a contact_custom_field_values.value JSONB
// cell as plain text for the disclosure. field_type on the definition
// (text/number/date/boolean/select/multiselect) governs how the value was
// written, but decoding by Go type covers all of them without needing to
// look up the definition's field_type separately.
func formatCustomFieldValue(raw []byte) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case bool:
		return boolLabel(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", val)
	}
}

func tagsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT t.name, ct.created_at
		 FROM contact_tags ct
		 JOIN tags t ON t.id = ct.tag_id
		 WHERE ct.contact_id = $1 AND ct.tenant_id = $2
		 ORDER BY ct.created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query tags: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Tags", Columns: []string{"Tag", "Zugewiesen"}}
	for rows.Next() {
		var name string
		var at time.Time
		if scanErr := rows.Scan(&name, &at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan tag: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Tag", Value: name},
			{Key: "Zugewiesen", Value: at.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: tag rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// documentsModule discloses metadata only for files linked to a contact via
// document_entity_links (entity_type="contact", the literal used throughout
// internal/gateway/route_crm_contact_files.go and internal/document/file --
// there is no shared constant for it). No storage_key, thumbnail_key or
// content_text leaves this function: an Art. 15 export leaves the building,
// and a path or signed URL in it would be a document-store access path that
// bypasses the permission check a real download goes through.
//
// Soft-deleted files (is_deleted) are excluded: from the data subject's
// perspective they no longer exist once moved to trash, same as the file
// list UI already treats them.
func documentsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT f.filename, f.mime_type, f.file_size, f.created_at
		 FROM document_files f
		 JOIN document_entity_links del ON del.file_id = f.id
		 WHERE del.entity_type = 'contact' AND del.entity_id = $1 AND del.tenant_id = $2
		   AND f.tenant_id = $2 AND NOT f.is_deleted
		 ORDER BY f.created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query documents: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Dokumente", Columns: []string{"Dateiname", "Typ", "Größe", "Hochgeladen am"}}
	for rows.Next() {
		var filename, mimeType string
		var size int64
		var at time.Time
		if scanErr := rows.Scan(&filename, &mimeType, &size, &at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan document: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Dateiname", Value: filename},
			{Key: "Typ", Value: mimeType},
			{Key: "Größe", Value: fmt.Sprintf("%d Bytes", size)},
			{Key: "Hochgeladen am", Value: at.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: document rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// formFieldMeta is a partial decode of one entry in form_schemas.fields --
// enough to label an answer for disclosure without importing the formulare
// package's full FormField (which also carries options/conditional logic a
// data subject export has no use for).
type formFieldMeta struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

// formSubmissionsModule discloses public form submissions matched to a
// contact by comparing an "email"-typed field's answer against the contact's
// own address. This is a heuristic, not a foreign key: form_submissions
// carries no contact_id at all, and submitted_by is not the respondent's
// identity -- it holds the staff user who logged an authenticated
// submission, or nothing whatsoever for a public share-link submission
// (formulare/form_share.go SubmitByShareToken never sets it). The
// respondent's own data lives only inside the JSONB answers, keyed by
// whichever field id the form author chose, which is why the match has to
// go through the schema's field definitions instead of a column.
//
// A submission whose schema has since been deleted (form_schema_id set NULL
// by form_schemas' ON DELETE SET NULL) is excluded: without the schema there
// is no way to tell which answer, if any, held an email address.
func formSubmissionsModule(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, contactEmail string) (*DSARModule, error) {
	if contactEmail == "" {
		return nil, nil
	}
	rows, err := pool.Query(ctx,
		`SELECT fs.title, fs.fields, fsub.answers, fsub.submitted_at
		 FROM form_submissions fsub
		 JOIN form_schemas fs ON fs.id = fsub.form_schema_id
		 WHERE fsub.tenant_id = $1 AND fs.tenant_id = $1
		   AND EXISTS (
		     SELECT 1 FROM jsonb_array_elements(fs.fields) AS field(elem)
		     WHERE field.elem ->> 'type' = 'email'
		       AND LOWER(fsub.answers ->> (field.elem ->> 'id')) = LOWER($2)
		   )
		 ORDER BY fsub.submitted_at DESC
		 LIMIT $3`, tenantID, contactEmail, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query form submissions: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Formulareinreichungen", Columns: []string{"Formular", "Datum", "Feld", "Wert"}}
	for rows.Next() {
		var title string
		var fieldsRaw, answersRaw []byte
		var at time.Time
		if scanErr := rows.Scan(&title, &fieldsRaw, &answersRaw, &at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan form submission: %w", scanErr)
		}

		var defs []formFieldMeta
		if err := json.Unmarshal(fieldsRaw, &defs); err != nil {
			return nil, fmt.Errorf("dsar: decode form fields: %w", err)
		}
		var answers map[string]json.RawMessage
		if err := json.Unmarshal(answersRaw, &answers); err != nil {
			// The disclosure must still surface stored data even if it is not
			// the JSON object shape every submission is expected to have.
			mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
				{Key: "Formular", Value: title},
				{Key: "Datum", Value: at.Format(dsarTimeLayout)},
				{Key: "Feld", Value: "(unstrukturiert)"},
				{Key: "Wert", Value: string(answersRaw)},
			}})
			continue
		}

		for _, def := range defs {
			raw, ok := answers[def.ID]
			if !ok {
				continue
			}
			label := def.Label
			if label == "" {
				label = def.ID
			}
			mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
				{Key: "Formular", Value: title},
				{Key: "Datum", Value: at.Format(dsarTimeLayout)},
				{Key: "Feld", Value: label},
				{Key: "Wert", Value: formatCustomFieldValue(raw)},
			}})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: form submission rows: %w", err)
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

func financeModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT invoice_number, status, invoice_date, gross_total::float8
		 FROM finance_invoices
		 WHERE contact_id = $1 AND tenant_id = $2
		 ORDER BY invoice_date DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query finance invoices: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Rechnungen", Columns: []string{"Rechnungsnummer", "Status", "Datum", "Betrag"}}
	for rows.Next() {
		var number, status string
		var at time.Time
		var gross float64
		if scanErr := rows.Scan(&number, &status, &at, &gross); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan finance invoice: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Rechnungsnummer", Value: number},
			{Key: "Status", Value: status},
			{Key: "Datum", Value: at.Format(dsarDateLayout)},
			// lean: EUR hardcoded — finance_invoices carries no currency column
			// because the product is DACH/EUR-only; revisit if multi-currency
			// billing lands.
			{Key: "Betrag", Value: fmt.Sprintf("%.2f EUR", gross)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: finance invoice rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func meetingsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT title, status, scheduled_start, scheduled_end
		 FROM meetings
		 WHERE contact_id = $1 AND tenant_id = $2
		 ORDER BY scheduled_start DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query meetings: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Meetings", Columns: []string{"Titel", "Status", "Beginn", "Ende"}}
	for rows.Next() {
		var title, status string
		var start, end time.Time
		if scanErr := rows.Scan(&title, &status, &start, &end); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan meeting: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Titel", Value: title},
			{Key: "Status", Value: status},
			{Key: "Beginn", Value: start.Format(dsarTimeLayout)},
			{Key: "Ende", Value: end.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: meeting rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func helpdeskModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT subject, status, priority, created_at
		 FROM tickets
		 WHERE contact_id = $1 AND tenant_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query tickets: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Helpdesk-Tickets", Columns: []string{"Betreff", "Status", "Priorität", "Erstellt"}}
	for rows.Next() {
		var subject, status, priority string
		var at time.Time
		if scanErr := rows.Scan(&subject, &status, &priority, &at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan ticket: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Betreff", Value: subject},
			{Key: "Status", Value: status},
			{Key: "Priorität", Value: priority},
			{Key: "Erstellt", Value: at.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: ticket rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// helpdeskMessagesModule discloses the customer-facing conversation on a
// contact's tickets — what the person wrote and what they were told, which is
// the actual personal content behind the ticket metadata in helpdeskModule.
// Internal notes (internal = true) are excluded on purpose: they are agent
// working material and routinely contain a colleague's assessment OF the
// data subject, not communication addressed to them.
func helpdeskMessagesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT t.subject, tm.body, tm.created_at
		 FROM ticket_messages tm
		 JOIN tickets t ON t.id = tm.ticket_id
		 WHERE t.contact_id = $1 AND tm.tenant_id = $2 AND NOT tm.internal
		 ORDER BY tm.created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows+1)
	if err != nil {
		return nil, fmt.Errorf("dsar: query helpdesk messages: %w", err)
	}
	defer rows.Close()

	type message struct {
		subject, body string
		at            time.Time
	}
	var messages []message
	for rows.Next() {
		var m message
		if scanErr := rows.Scan(&m.subject, &m.body, &m.at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan helpdesk message: %w", scanErr)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: helpdesk message rows: %w", err)
	}
	if len(messages) == 0 {
		return nil, nil
	}

	// Query fetched dsarMaxRows+1 newest-first so truncation, if any, drops
	// the oldest messages rather than silently hiding the most recent ones.
	truncated := len(messages) > dsarMaxRows
	if truncated {
		messages = messages[:dsarMaxRows]
	}
	// Reverse to chronological (oldest-first) order for a readable transcript.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	mod := DSARModule{Module: "Helpdesk-Nachrichten", Columns: []string{"Ticket", "Nachricht", "Datum"}}
	if truncated {
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Ticket", Value: ""},
			{Key: "Nachricht", Value: fmt.Sprintf("(gekürzt auf die %d neuesten Nachrichten)", dsarMaxRows)},
			{Key: "Datum", Value: ""},
		}})
	}
	for _, m := range messages {
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Ticket", Value: m.subject},
			{Key: "Nachricht", Value: m.body},
			{Key: "Datum", Value: m.at.Format(dsarTimeLayout)},
		}})
	}
	return &mod, nil
}

// contractsModule resolves via contract_parties: contracts carries no
// contact_id of its own, a contact is one of possibly several parties to a
// contract (contact/company/external).
func contractsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT c.contract_number, c.title, c.status, c.starts_on
		 FROM contracts c
		 JOIN contract_parties cp ON cp.contract_id = c.id
		 WHERE cp.contact_id = $1 AND cp.tenant_id = $2
		 ORDER BY c.starts_on DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query contracts: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Verträge", Columns: []string{"Vertragsnummer", "Titel", "Status", "Beginn"}}
	for rows.Next() {
		var number, title, status string
		var startsOn time.Time
		if scanErr := rows.Scan(&number, &title, &status, &startsOn); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan contract: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Vertragsnummer", Value: number},
			{Key: "Titel", Value: title},
			{Key: "Status", Value: status},
			{Key: "Beginn", Value: startsOn.Format(dsarDateLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: contract rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

// emailModule resolves via email_contact_links: the junction table carries no
// tenant_id of its own, so tenant scoping happens on the joined email_messages
// row instead.
func emailModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT em.date, COALESCE(em.subject, ''), COALESCE(NULLIF(em.from_name, ''), em.from_email)
		 FROM email_messages em
		 JOIN email_contact_links ecl ON ecl.message_id = em.id
		 WHERE ecl.contact_id = $1 AND em.tenant_id = $2
		 ORDER BY em.date DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query emails: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "E-Mails", Columns: []string{"Datum", "Betreff", "Von"}}
	for rows.Next() {
		var subject, from string
		var at time.Time
		if scanErr := rows.Scan(&at, &subject, &from); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan email: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Datum", Value: at.Format(dsarTimeLayout)},
			{Key: "Betreff", Value: subject},
			{Key: "Von", Value: from},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: email rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func dealsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT d.name, d.value::float8, d.currency, ps.name, d.created_at
		 FROM deals d
		 JOIN pipeline_stages ps ON ps.id = d.stage_id
		 WHERE d.contact_id = $1 AND d.tenant_id = $2
		 ORDER BY d.created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query deals: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Deals", Columns: []string{"Bezeichnung", "Wert", "Phase", "Erstellt"}}
	for rows.Next() {
		var name, currency, stage string
		var value float64
		var at time.Time
		if scanErr := rows.Scan(&name, &value, &currency, &stage, &at); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan deal: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Bezeichnung", Value: name},
			{Key: "Wert", Value: fmt.Sprintf("%.2f %s", value, currency)},
			{Key: "Phase", Value: stage},
			{Key: "Erstellt", Value: at.Format(dsarTimeLayout)},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: deal rows: %w", err)
	}
	if len(mod.Records) == 0 {
		return nil, nil
	}
	return &mod, nil
}

func activitiesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, contactID uuid.UUID) (*DSARModule, error) {
	rows, err := pool.Query(ctx,
		`SELECT created_at, activity_type::text, subject, COALESCE(description, '')
		 FROM activities
		 WHERE contact_id = $1 AND tenant_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`, contactID, tenantID, dsarMaxRows)
	if err != nil {
		return nil, fmt.Errorf("dsar: query activities: %w", err)
	}
	defer rows.Close()

	mod := DSARModule{Module: "Aktivitäten", Columns: []string{"Datum", "Typ", "Betreff", "Notiz"}}
	for rows.Next() {
		var activityType, subject, note string
		var at time.Time
		if scanErr := rows.Scan(&at, &activityType, &subject, &note); scanErr != nil {
			return nil, fmt.Errorf("dsar: scan activity: %w", scanErr)
		}
		mod.Records = append(mod.Records, DSARRecord{Fields: []DSARField{
			{Key: "Datum", Value: at.Format(dsarTimeLayout)},
			{Key: "Typ", Value: activityType},
			{Key: "Betreff", Value: subject},
			{Key: "Notiz", Value: note},
		}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dsar: activity rows: %w", err)
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
