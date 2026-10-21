package gdpr

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// CRM custom values, CSAT comments and webhook deliveries — Art. 15 GDPR.
//
// Read-only. Contact branch only.
//
// The company/deal/activity custom value tables carry no tenant_id (RLS runs
// via a join, migration 000270); every query here joins the parent AND the
// field definition and pins both to the tenant, like customFieldsModule.
//
// CSAT: only submitted responses are disclosed; the survey token is never
// selected. Webhook deliveries: the payload is a copy of the submission
// answers that formSubmissionsModule already discloses, so what is new here is
// WHERE it went — the receiver host and the delivery state. The webhook URL
// is reduced to its host (query strings and userinfo can carry credentials);
// the signing secret is never selected.
// ---------------------------------------------------------------------------

// contactCrmValueModules collects the modules for one contact; empty ones are
// skipped.
func contactCrmValueModules(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, c contactRow) ([]DSARModule, error) {
	var out []DSARModule
	add := func(mod *DSARModule, err error) error {
		if err != nil {
			return fmt.Errorf("dsar: crm values: %w", err)
		}
		if mod != nil {
			out = append(out, *mod)
		}
		return nil
	}

	for _, spec := range []struct {
		module, subject, query string
	}{
		{"Benutzerdefinierte Felder (Firma)", "Firma",
			`SELECT co.name, cfd.field_label, cfv.value
			 FROM contacts c
			 JOIN companies co ON co.id = c.company_id
			 JOIN company_custom_field_values cfv ON cfv.company_id = co.id
			 JOIN custom_field_definitions cfd ON cfd.id = cfv.field_id
			 WHERE c.id = $1 AND c.tenant_id = $2 AND co.tenant_id = $2 AND cfd.tenant_id = $2
			 ORDER BY co.name, cfd.sort_order, cfd.field_label
			 LIMIT $3`},
		{"Benutzerdefinierte Felder (Deals)", "Deal",
			`SELECT d.name, cfd.field_label, cfv.value
			 FROM deals d
			 JOIN deal_custom_field_values cfv ON cfv.deal_id = d.id
			 JOIN custom_field_definitions cfd ON cfd.id = cfv.field_id
			 WHERE d.contact_id = $1 AND d.tenant_id = $2 AND cfd.tenant_id = $2
			 ORDER BY d.created_at DESC, cfd.sort_order, cfd.field_label
			 LIMIT $3`},
		{"Benutzerdefinierte Felder (Aktivitäten)", "Aktivität",
			`SELECT a.subject, cfd.field_label, cfv.value
			 FROM activities a
			 JOIN activity_custom_field_values cfv ON cfv.activity_id = a.id
			 JOIN custom_field_definitions cfd ON cfd.id = cfv.field_id
			 WHERE a.contact_id = $1 AND a.tenant_id = $2 AND cfd.tenant_id = $2
			 ORDER BY a.created_at DESC, cfd.sort_order, cfd.field_label
			 LIMIT $3`},
	} {
		subjectKey := spec.subject
		if err := add(userRowsModule(ctx, pool, spec.module, spec.module, []string{subjectKey, "Feld", "Wert"},
			spec.query, []any{c.id, tenantID, dsarMaxRows},
			func(rows pgx.Rows) (DSARRecord, error) {
				var subject, label string
				var raw []byte
				if err := rows.Scan(&subject, &label, &raw); err != nil {
					return DSARRecord{}, err
				}
				return DSARRecord{Fields: []DSARField{
					{Key: subjectKey, Value: subject},
					{Key: "Feld", Value: label},
					{Key: "Wert", Value: formatCustomFieldValue(raw)},
				}}, nil
			})); err != nil {
			return nil, err
		}
	}

	if err := add(userRowsModule(ctx, pool, "csat responses", "Zufriedenheitsumfragen (CSAT)",
		[]string{"Ticket", "Bewertung", "Kommentar", "Abgegeben"},
		`SELECT t.subject, r.rating, COALESCE(r.comment, ''), r.submitted_at
		 FROM ticket_csat_responses r
		 JOIN tickets t ON t.id = r.ticket_id
		 WHERE t.contact_id = $1 AND t.tenant_id = $2 AND r.tenant_id = $2 AND r.submitted_at IS NOT NULL
		 ORDER BY r.submitted_at DESC
		 LIMIT $3`, []any{c.id, tenantID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var subject, comment string
			var rating *int16
			var at time.Time
			if err := rows.Scan(&subject, &rating, &comment, &at); err != nil {
				return DSARRecord{}, err
			}
			score := ""
			if rating != nil {
				score = strconv.Itoa(int(*rating)) + " von 5"
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Ticket", Value: subject},
				{Key: "Bewertung", Value: score},
				{Key: "Kommentar", Value: comment},
				{Key: "Abgegeben", Value: at.Format(dsarTimeLayout)},
			}}, nil
		})); err != nil {
		return nil, err
	}

	if email := strings.TrimSpace(c.email); email != "" {
		if err := add(webhookDeliveriesModule(ctx, pool, tenantID, email)); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// webhookDeliveriesModule lists where the contact's form submissions (matched
// by the e-mail field, same rule as formSubmissionsModule) were forwarded.
func webhookDeliveriesModule(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, email string) (*DSARModule, error) {
	return userRowsModule(ctx, pool, "form webhook deliveries", "Webhook-Weiterleitungen von Formulareinreichungen",
		[]string{"Formular", "Empfänger", "Status", "Versuche", "Eingereicht", "Zugestellt"},
		`SELECT fs.title, fw.url, d.status::text, d.attempt_count, fsub.submitted_at, d.delivered_at
		 FROM form_webhook_deliveries d
		 JOIN form_submissions fsub ON fsub.id = d.submission_id
		 JOIN form_schemas fs ON fs.id = fsub.form_schema_id
		 JOIN form_webhooks fw ON fw.id = d.webhook_id
		 WHERE d.tenant_id = $1 AND fsub.tenant_id = $1 AND fs.tenant_id = $1 AND fw.tenant_id = $1
		   AND EXISTS (
		     SELECT 1 FROM jsonb_array_elements(fs.fields) AS field(elem)
		     WHERE field.elem ->> 'type' = 'email'
		       AND LOWER(fsub.answers ->> (field.elem ->> 'id')) = LOWER($2)
		   )
		 ORDER BY fsub.submitted_at DESC, d.created_at DESC
		 LIMIT $3`, []any{tenantID, email, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var title, rawURL, status string
			var attempts int
			var submittedAt time.Time
			var deliveredAt *time.Time
			if err := rows.Scan(&title, &rawURL, &status, &attempts, &submittedAt, &deliveredAt); err != nil {
				return DSARRecord{}, err
			}
			delivered := ""
			if deliveredAt != nil {
				delivered = deliveredAt.Format(dsarTimeLayout)
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Formular", Value: title},
				{Key: "Empfänger", Value: webhookHost(rawURL)},
				{Key: "Status", Value: status},
				{Key: "Versuche", Value: strconv.Itoa(attempts)},
				{Key: "Eingereicht", Value: submittedAt.Format(dsarTimeLayout)},
				{Key: "Zugestellt", Value: delivered},
			}}, nil
		})
}

// webhookHost reduces a webhook URL to its host so userinfo, path and query
// (potential credentials) never reach the disclosure.
func webhookHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "(ungültige URL)"
	}
	return u.Host
}
