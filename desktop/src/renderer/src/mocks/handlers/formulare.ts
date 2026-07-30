import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { CURRENT_USER } from '@/mocks/data/shared-ids'
import type {
  CreateFormSchemaInput,
  CreateShareLinkInput,
  CreateSubmissionInput,
  CreateWebhookInput,
  DuplicateFormSchemaInput,
  FormField,
  FormSchema,
  FormShareLink,
  FormSubmission,
  FormWebhook,
  ShareLinkStatus,
  UpdateFormSchemaInput,
  UpdateShareLinkInput,
  UpdateSubmissionStatusInput,
  UpdateWebhookInput,
  WebhookDelivery,
} from '@/api/formulare-types'

const API = API_BASE_URL
const TENANT = 't-demo'

// ---------------------------------------------------------------------------
// Formulare (Forms) demo backend.
//
// The /api/v1/formulare gateway exists in Go but is not reachable in demo mode;
// the renderer calls it directly via formulare-client.ts. This handler serves a
// stateful, session-persistent surface so the whole module lives: form list,
// builder, submissions, status lifecycle, exports and module settings. All
// shapes mirror formulare-types.ts (camelCase) exactly — except the stats
// endpoint, which mirrors the real backend's protojson envelope/snake_case
// wire shape (see formulare-client.ts:getFormStats). State is in-memory and
// resets on reload. Backend wiring (real persistence, webhooks delivery) stays
// 🔒 Luke's lane — tracked in backend-gaps.md.
// ---------------------------------------------------------------------------

function isoAgo(ms: number): string {
  return new Date(Date.now() - ms).toISOString()
}

const MIN = 60_000
const HOUR = 60 * MIN
const DAY = 24 * HOUR

let fieldSeq = 1
function f(field: Omit<FormField, 'id'>): FormField {
  return { id: `fld-${fieldSeq++}`, ...field }
}

const STD_CONSENT_TEXT =
  'Ich willige ein, dass meine Angaben zur Bearbeitung meiner Anfrage gemäß der Datenschutzerklärung verarbeitet werden. Die Einwilligung kann jederzeit widerrufen werden.'

/** Standard DSGVO consent field reused by the DACH template pack (FT-2b). */
function consentField(): FormField {
  return f({
    type: 'consent',
    label: 'Datenschutz-Einwilligung',
    required: true,
    consentText: STD_CONSENT_TEXT,
    privacyUrl: 'https://www.zentria.tech/datenschutz',
  })
}

// ---------------------------------------------------------------------------
// Seed — schemas (forms + templates)
// ---------------------------------------------------------------------------

function buildSchemas(): FormSchema[] {
  const base = (
    over: Partial<FormSchema> & Pick<FormSchema, 'id' | 'title' | 'fields'>,
  ): FormSchema => ({
    tenantId: TENANT,
    description: '',
    status: 'active',
    isTemplate: false,
    isPublic: false,
    pageCount: 1,
    submissionCount: 0,
    notifications: { recipients: [], confirmToSubmitter: false },
    createdBy: 'Lena Hoffmann',
    createdAt: isoAgo(40 * DAY),
    updatedAt: isoAgo(2 * DAY),
    deletedAt: null,
    ...over,
  })

  return [
    base({
      id: 'form-feedback',
      title: 'Kundenfeedback',
      description: 'Sammelt Rückmeldungen zu Produkt und Service nach dem Kauf.',
      status: 'active',
      isPublic: true,
      submissionCount: 6,
      notifications: { recipients: ['feedback@zentria.tech'], confirmToSubmitter: false },
      createdAt: isoAgo(58 * DAY),
      updatedAt: isoAgo(6 * HOUR),
      fields: (() => {
        // Capture the rating field so the conditional follow-up can target it.
        const rating = f({ type: 'select', label: 'Gesamtbewertung', required: true, options: ['Sehr zufrieden', 'Zufrieden', 'Neutral', 'Unzufrieden'] })
        return [
          f({ type: 'text', label: 'Name', required: false, placeholder: 'Vor- und Nachname' }),
          f({ type: 'email', label: 'E-Mail', required: true, placeholder: 'name@beispiel.de' }),
          rating,
          f({ type: 'textarea', label: 'Was können wir verbessern?', required: false, placeholder: 'Ihr Kommentar …' }),
          // FO-6 — conditional follow-up: only asked when the rating is 'Unzufrieden'.
          f({
            type: 'textarea',
            label: 'Was genau lief schief?',
            required: false,
            placeholder: 'Bitte beschreiben Sie, was schiefgelaufen ist …',
            conditionalLogic: { fieldId: rating.id, operator: 'equals', value: 'Unzufrieden' },
          }),
          f({
            type: 'consent',
            label: 'Datenschutz-Einwilligung',
            required: true,
            consentText:
              'Ich willige ein, dass meine Angaben zur Bearbeitung meiner Anfrage gemäß der Datenschutzerklärung verarbeitet werden. Die Einwilligung kann jederzeit widerrufen werden.',
            privacyUrl: 'https://www.zentria.tech/datenschutz',
          }),
        ]
      })(),
    }),
    base({
      id: 'form-kontakt',
      title: 'Kontaktanfrage',
      description: 'Allgemeines Kontaktformular für die Website.',
      status: 'active',
      isPublic: true,
      submissionCount: 4,
      notifications: { recipients: ['vertrieb@zentria.tech'], confirmToSubmitter: true },
      createdAt: isoAgo(35 * DAY),
      updatedAt: isoAgo(1 * DAY),
      fields: [
        f({ type: 'text', label: 'Name', required: true, placeholder: 'Vor- und Nachname' }),
        f({ type: 'email', label: 'E-Mail', required: true, placeholder: 'name@beispiel.de' }),
        f({ type: 'text', label: 'Betreff', required: false, placeholder: 'Worum geht es?' }),
        f({ type: 'textarea', label: 'Nachricht', required: true, placeholder: 'Ihre Nachricht …' }),
        f({
          type: 'consent',
          label: 'Datenschutz-Einwilligung',
          required: true,
          consentText:
            'Ich willige ein, dass meine Angaben zur Bearbeitung meiner Anfrage gemäß der Datenschutzerklärung verarbeitet werden. Die Einwilligung kann jederzeit widerrufen werden.',
          privacyUrl: 'https://www.zentria.tech/datenschutz',
        }),
      ],
    }),
    base({
      id: 'form-bewerbung',
      title: 'Bewerbungsformular',
      description: 'Initiativbewerbungen und Stellenbewerbungen erfassen.',
      status: 'active',
      isPublic: true,
      submissionCount: 3,
      notifications: { recipients: ['hr@zentria.tech', 'lena.hoffmann@zentria.tech'], confirmToSubmitter: true },
      createdAt: isoAgo(22 * DAY),
      updatedAt: isoAgo(3 * DAY),
      fields: [
        f({ type: 'text', label: 'Name', required: true, placeholder: 'Vor- und Nachname' }),
        f({ type: 'email', label: 'E-Mail', required: true, placeholder: 'name@beispiel.de' }),
        f({ type: 'select', label: 'Position', required: true, options: ['Vertrieb', 'Entwicklung', 'Buchhaltung', 'Initiativbewerbung'] }),
        f({ type: 'date', label: 'Verfügbar ab', required: false }),
        f({ type: 'file', label: 'Lebenslauf (PDF)', required: true }),
        f({
          type: 'consent',
          label: 'Datenschutz-Einwilligung',
          required: true,
          consentText:
            'Ich willige ein, dass meine Angaben zur Bearbeitung meiner Anfrage gemäß der Datenschutzerklärung verarbeitet werden. Die Einwilligung kann jederzeit widerrufen werden.',
          privacyUrl: 'https://www.zentria.tech/datenschutz',
        }),
      ],
    }),
    base({
      id: 'form-event',
      title: 'Eventanmeldung Sommerfest',
      description: 'Anmeldung zum jährlichen Kundenevent.',
      // Closed lifecycle demo: registration period is over, but the responses
      // stay visible for evaluation and the form can be re-opened.
      status: 'closed',
      isPublic: true,
      // Multi-page demo (FT-3b): drives the per-page drop-off funnel.
      pageCount: 2,
      closedMessage:
        'Die Anmeldung zum Sommerfest ist abgeschlossen. Vielen Dank für Ihr Interesse — wir freuen uns auf das nächste Event!',
      submissionCount: 5,
      createdAt: isoAgo(14 * DAY),
      updatedAt: isoAgo(8 * HOUR),
      fields: [
        f({ type: 'text', label: 'Name', required: true, placeholder: 'Vor- und Nachname' }),
        f({ type: 'email', label: 'E-Mail', required: true, placeholder: 'name@beispiel.de' }),
        f({ type: 'number', label: 'Anzahl Personen', required: true, placeholder: '1' }),
        f({ type: 'text', label: '__page_break__', required: false, page: 1, pageTitle: 'Verpflegung & Einwilligung' }),
        f({ type: 'radio', label: 'Verpflegung', required: true, options: ['Mit Fleisch', 'Vegetarisch', 'Vegan'] }),
        f({ type: 'checkbox', label: 'Newsletter abonnieren', required: false }),
        f({
          type: 'consent',
          label: 'Datenschutz-Einwilligung',
          required: true,
          consentText:
            'Ich willige ein, dass meine Angaben zur Bearbeitung meiner Anfrage gemäß der Datenschutzerklärung verarbeitet werden. Die Einwilligung kann jederzeit widerrufen werden.',
          privacyUrl: 'https://www.zentria.tech/datenschutz',
        }),
      ],
    }),
    base({
      id: 'form-newsletter',
      title: 'Newsletter-Anmeldung',
      description: 'Schlankes Double-Opt-in-Formular für den Newsletter.',
      // Archived lifecycle demo: read-only, kept for the record.
      status: 'archived',
      isPublic: false,
      submissionCount: 2,
      createdAt: isoAgo(9 * DAY),
      updatedAt: isoAgo(20 * HOUR),
      fields: [
        f({ type: 'email', label: 'E-Mail', required: true, placeholder: 'name@beispiel.de' }),
        f({
          type: 'consent',
          label: 'Datenschutz-Einwilligung',
          required: true,
          consentText:
            'Ich willige ein, dass meine Angaben zur Bearbeitung meiner Anfrage gemäß der Datenschutzerklärung verarbeitet werden. Die Einwilligung kann jederzeit widerrufen werden.',
          privacyUrl: 'https://www.zentria.tech/datenschutz',
        }),
      ],
    }),
    base({
      id: 'form-support-draft',
      title: 'Support-Ticket (Entwurf)',
      description: 'Internes Ticket-Formular — noch in Arbeit.',
      status: 'draft',
      isPublic: false,
      submissionCount: 0,
      createdAt: isoAgo(4 * DAY),
      updatedAt: isoAgo(4 * HOUR),
      fields: [
        f({ type: 'email', label: 'E-Mail', required: true, placeholder: 'name@beispiel.de' }),
        f({ type: 'select', label: 'Kategorie', required: true, options: ['Technisch', 'Abrechnung', 'Sonstiges'] }),
        f({ type: 'textarea', label: 'Problembeschreibung', required: true, placeholder: 'Was ist passiert?' }),
      ],
    }),
    // ── Intake channel forms (Ticket-Intake, real editable forms per channel) ──
    // Bound to the Helpdesk ticket target; the editor's Kanäle panel assigns one
    // per channel. These are real forms (isTemplate: false) so they are directly
    // editable — unlike the seed *template* below, which is a duplication start.
    base({
      id: 'form-ticket-selfservice',
      title: 'Support-Ticket (Selfservice)',
      description:
        'Schlankes Ticket-Formular für Mitarbeitende und externe Melder — Kontaktdaten werden bei internen Meldern automatisch übernommen.',
      status: 'active',
      isPublic: true,
      submissionCount: 0,
      createdAt: isoAgo(30 * DAY),
      updatedAt: isoAgo(3 * DAY),
      intakeTargetId: 'helpdesk_ticket',
      fields: [
        f({ type: 'text', label: 'Betreff', required: true, placeholder: 'Kurz: worum geht es?', role: 'subject' }),
        f({ type: 'textarea', label: 'Beschreibung', required: true, placeholder: 'Bitte schildern Sie Ihr Anliegen …', role: 'description' }),
        f({ type: 'select', label: 'Kategorie', required: false, options: ['Technisch', 'Abrechnung', 'Allgemein', 'Sonstiges'], role: 'category' }),
        f({ type: 'select', label: 'Priorität', required: false, options: ['Niedrig', 'Normal', 'Hoch', 'Dringend'], role: 'priority' }),
        f({ type: 'text', label: 'Ihr Name', required: true, placeholder: 'Vor- und Nachname', role: 'requester_name' }),
        f({ type: 'email', label: 'Ihre E-Mail', required: true, placeholder: 'name@beispiel.de', role: 'requester_email' }),
        consentField(),
      ],
    }),
    base({
      id: 'form-ticket-agent',
      title: 'Agent-Ticket (intern)',
      description:
        'Erweitertes Ticket-Formular für Agenten — mit technischen Zusatzfeldern. Kontakt, Priorität und Zuweisung kommen aus den Agent-Werkzeugen im Dialog.',
      status: 'active',
      isPublic: false,
      submissionCount: 0,
      createdAt: isoAgo(30 * DAY),
      updatedAt: isoAgo(3 * DAY),
      intakeTargetId: 'helpdesk_ticket',
      fields: [
        f({ type: 'text', label: 'Betreff', required: true, placeholder: 'Kurz: worum geht es?', role: 'subject' }),
        f({ type: 'textarea', label: 'Beschreibung', required: true, placeholder: 'Problem und bisherige Schritte …', role: 'description' }),
        f({ type: 'select', label: 'Kategorie', required: false, options: ['Technisch', 'Abrechnung', 'Allgemein', 'Sonstiges'], role: 'category' }),
        f({ type: 'text', label: 'Gerät / Anlage', required: false, placeholder: 'z.B. Kassensystem Filiale 3' }),
        f({ type: 'text', label: 'Fehlercode', required: false, placeholder: 'z.B. E-4711' }),
      ],
    }),
    // ── Templates ──
    // P2 — intake template: submissions become Helpdesk tickets. Each field is
    // mapped to a ticket role via `role`; the unmarked field flows into the
    // ticket's custom_fields. Bound via `intakeTargetId: 'helpdesk_ticket'`.
    base({
      id: 'tmpl-ticket-intake',
      title: 'Support-Ticket (Helpdesk)',
      description:
        'Einreichungen dieses Formulars kommen als Helpdesk-Tickets an — die Felder sind Ticket-Rollen zugeordnet.',
      status: 'active',
      isTemplate: true,
      isPublic: true,
      submissionCount: 0,
      createdBy: null,
      createdAt: isoAgo(90 * DAY),
      updatedAt: isoAgo(90 * DAY),
      intakeTargetId: 'helpdesk_ticket',
      fields: [
        f({ type: 'text', label: 'Betreff', required: true, placeholder: 'Kurz: worum geht es?', role: 'subject' }),
        f({ type: 'textarea', label: 'Beschreibung', required: true, placeholder: 'Bitte schildern Sie Ihr Anliegen …', role: 'description' }),
        f({ type: 'select', label: 'Priorität', required: false, options: ['Niedrig', 'Normal', 'Hoch', 'Dringend'], role: 'priority' }),
        f({ type: 'select', label: 'Kategorie', required: false, options: ['Technisch', 'Abrechnung', 'Allgemein', 'Sonstiges'], role: 'category' }),
        f({ type: 'text', label: 'Ihr Name', required: true, placeholder: 'Vor- und Nachname', role: 'requester_name' }),
        f({ type: 'email', label: 'Ihre E-Mail', required: true, placeholder: 'name@beispiel.de', role: 'requester_email' }),
        f({ type: 'text', label: 'Bestell-/Kundennummer', required: false, placeholder: 'z.B. 2026-04711' }),
        consentField(),
      ],
    }),
    base({
      id: 'tmpl-zufriedenheit',
      title: 'Zufriedenheitsumfrage',
      description: 'Vorlage für eine kurze Zufriedenheitsumfrage.',
      status: 'active',
      isTemplate: true,
      submissionCount: 0,
      createdBy: null,
      createdAt: isoAgo(90 * DAY),
      updatedAt: isoAgo(90 * DAY),
      fields: [
        f({ type: 'select', label: 'Wie zufrieden sind Sie?', required: true, options: ['Sehr gut', 'Gut', 'Mittel', 'Schlecht'] }),
        f({ type: 'textarea', label: 'Begründung', required: false }),
      ],
    }),
    base({
      id: 'tmpl-veranstaltung',
      title: 'Veranstaltungsanmeldung',
      description: 'Vorlage für eine Event-Anmeldung mit Teilnehmerzahl.',
      status: 'active',
      isTemplate: true,
      submissionCount: 0,
      createdBy: null,
      createdAt: isoAgo(90 * DAY),
      updatedAt: isoAgo(90 * DAY),
      fields: [
        f({ type: 'text', label: 'Name', required: true }),
        f({ type: 'email', label: 'E-Mail', required: true }),
        f({ type: 'number', label: 'Anzahl Personen', required: true }),
      ],
    }),
    // ── DACH-KMU template pack (FT-2b) ──
    base({
      id: 'tmpl-reklamation',
      title: 'Reklamationserfassung',
      description: 'Strukturierte Erfassung von Kundenreklamationen mit Foto-Upload.',
      status: 'active',
      isTemplate: true,
      submissionCount: 0,
      createdBy: null,
      createdAt: isoAgo(90 * DAY),
      updatedAt: isoAgo(90 * DAY),
      fields: [
        f({ type: 'text', label: 'Name', required: true, placeholder: 'Vor- und Nachname' }),
        f({ type: 'text', label: 'Bestell- / Rechnungsnummer', required: true, placeholder: 'z.B. 2026-04711' }),
        f({ type: 'select', label: 'Reklamationsgrund', required: true, options: ['Falschlieferung', 'Beschädigung', 'Qualitätsmangel', 'Fehlmenge', 'Sonstiges'] }),
        f({ type: 'textarea', label: 'Beschreibung', required: true, placeholder: 'Was ist passiert?' }),
        f({ type: 'file', label: 'Foto / Beleg', required: false }),
        consentField(),
      ],
    }),
    base({
      id: 'tmpl-urlaubsantrag',
      title: 'Urlaubsantrag (intern)',
      description: 'Interner Antrag auf Erholungsurlaub mit Vertretungsregelung.',
      status: 'active',
      isTemplate: true,
      submissionCount: 0,
      createdBy: null,
      createdAt: isoAgo(90 * DAY),
      updatedAt: isoAgo(90 * DAY),
      fields: [
        f({ type: 'text', label: 'Name', required: true, placeholder: 'Vor- und Nachname' }),
        f({ type: 'text', label: 'Abteilung', required: true, placeholder: 'z.B. Vertrieb' }),
        f({ type: 'date', label: 'Von', required: true }),
        f({ type: 'date', label: 'Bis', required: true }),
        f({ type: 'text', label: 'Vertretung', required: false, placeholder: 'Wer übernimmt?' }),
        f({ type: 'textarea', label: 'Anmerkung', required: false }),
        consentField(),
      ],
    }),
    base({
      id: 'tmpl-lieferantenbewertung',
      title: 'Lieferantenbewertung',
      description: 'Bewertung von Lieferanten nach Liefertreue, Qualität und Weiterempfehlung.',
      status: 'active',
      isTemplate: true,
      submissionCount: 0,
      createdBy: null,
      createdAt: isoAgo(90 * DAY),
      updatedAt: isoAgo(90 * DAY),
      fields: [
        f({ type: 'text', label: 'Lieferant', required: true, placeholder: 'Firmenname' }),
        f({ type: 'rating', label: 'Liefertreue', required: true, ratingScale: 5 }),
        f({ type: 'rating', label: 'Qualität', required: true, ratingScale: 5 }),
        f({ type: 'rating', label: 'Weiterempfehlung (0–10)', required: true, ratingScale: 10 }),
        f({ type: 'textarea', label: 'Anmerkungen', required: false, placeholder: 'Optionales Feedback …' }),
        consentField(),
      ],
    }),
    base({
      id: 'tmpl-schadensanzeige',
      title: 'Schadensanzeige',
      description: 'Meldung eines Schadens mit Hergang und geschätzter Schadenshöhe.',
      status: 'active',
      isTemplate: true,
      submissionCount: 0,
      createdBy: null,
      createdAt: isoAgo(90 * DAY),
      updatedAt: isoAgo(90 * DAY),
      fields: [
        f({ type: 'text', label: 'Name', required: true, placeholder: 'Vor- und Nachname' }),
        f({ type: 'date', label: 'Schadensdatum', required: true }),
        f({ type: 'text', label: 'Schadensort', required: true, placeholder: 'Ort / Adresse' }),
        f({ type: 'textarea', label: 'Schadenshergang', required: true, placeholder: 'Bitte schildern Sie den Ablauf.' }),
        f({ type: 'number', label: 'Geschätzte Schadenshöhe (EUR)', required: false, placeholder: '0', validation: { min: 0 } }),
        f({ type: 'file', label: 'Foto / Nachweis', required: false }),
        consentField(),
      ],
    }),
    base({
      id: 'tmpl-aufmass',
      title: 'Aufmaßerfassung Handwerk',
      description: 'Aufnahme von Raummaßen für Angebote und Kalkulation vor Ort.',
      status: 'active',
      isTemplate: true,
      submissionCount: 0,
      createdBy: null,
      createdAt: isoAgo(90 * DAY),
      updatedAt: isoAgo(90 * DAY),
      fields: [
        f({ type: 'text', label: 'Kunde', required: true, placeholder: 'Name / Firma' }),
        f({ type: 'text', label: 'Objektadresse', required: true, placeholder: 'Straße, PLZ, Ort' }),
        f({ type: 'text', label: 'Raum / Bereich', required: true, placeholder: 'z.B. Wohnzimmer' }),
        f({ type: 'number', label: 'Länge (m)', required: true, placeholder: '0.00', validation: { min: 0 } }),
        f({ type: 'number', label: 'Breite (m)', required: true, placeholder: '0.00', validation: { min: 0 } }),
        f({ type: 'number', label: 'Höhe (m)', required: false, placeholder: '0.00', validation: { min: 0 } }),
        f({ type: 'textarea', label: 'Bemerkung', required: false, placeholder: 'Besonderheiten vor Ort …' }),
        consentField(),
      ],
    }),
  ]
}

let SCHEMAS: FormSchema[] = buildSchemas()

// ---------------------------------------------------------------------------
// Seed — submissions (per schema, by field id)
// ---------------------------------------------------------------------------

const SUBMITTERS = [
  'Max Mustermann', 'Anna Schmidt', 'Thomas Weber', 'Julia Becker', 'Stefan Wagner',
  'Sandra Fischer', 'Michael Koch', 'Petra Klein', 'Andreas Richter', 'Nicole Wolf',
]

let subSeq = 1
function buildSubmissions(): FormSubmission[] {
  const out: FormSubmission[] = []

  const make = (
    schema: FormSchema,
    answersByLabel: Record<string, unknown>,
    over: Partial<FormSubmission> = {},
  ): FormSubmission => {
    // Map label-keyed answers onto the schema's field ids.
    const answers: Record<string, unknown> = {}
    for (const field of schema.fields) {
      if (field.label in answersByLabel) answers[field.id] = answersByLabel[field.label]
    }
    const submission: FormSubmission = {
      id: `sub-${subSeq++}`,
      formSchemaId: schema.id,
      tenantId: TENANT,
      answers,
      status: 'new',
      // Prefer the entered "Name" answer so the Absender column matches the
      // detail; fall back to a rotating roster for forms without a name field.
      submittedBy:
        (answersByLabel['Name'] as string | undefined) ??
        SUBMITTERS[(subSeq + 3) % SUBMITTERS.length],
      ipAddress: `192.168.${(subSeq * 7) % 250}.${(subSeq * 13) % 250}`,
      submittedAt: isoAgo(subSeq * 9 * HOUR),
      ...over,
    }
    // FO-4 — attach a mocked dispatch record when the form notifies on submit.
    const cfg = schema.notifications
    if (!submission.notifications && cfg && (cfg.recipients.length > 0 || cfg.confirmToSubmitter)) {
      const emailField = schema.fields.find((fld) => fld.type === 'email')
      const submitterEmail = emailField
        ? String(answersByLabel[emailField.label] ?? '').trim()
        : ''
      submission.notifications = {
        recipients: [...cfg.recipients],
        confirmationTo: cfg.confirmToSubmitter && submitterEmail ? submitterEmail : null,
        sentAt: submission.submittedAt,
      }
    }
    return submission
  }

  const byId = (id: string) => SCHEMAS.find((s) => s.id === id)!

  const feedback = byId('form-feedback')
  out.push(
    make(feedback, { Name: 'Max Mustermann', 'E-Mail': 'max@example.com', Gesamtbewertung: 'Sehr zufrieden', 'Was können wir verbessern?': 'Weiter so, top Service!', 'Datenschutz-Einwilligung': true }, { status: 'new', submittedAt: isoAgo(5 * HOUR) }),
    make(feedback, { Name: 'Anna Schmidt', 'E-Mail': 'anna@example.com', Gesamtbewertung: 'Zufrieden', 'Was können wir verbessern?': 'Lieferung war etwas langsam.', 'Datenschutz-Einwilligung': true }, { status: 'read', submittedAt: isoAgo(1 * DAY) }),
    make(feedback, { 'E-Mail': 'gast@example.com', Gesamtbewertung: 'Neutral', 'Was können wir verbessern?': '', 'Datenschutz-Einwilligung': true }, { status: 'read', submittedAt: isoAgo(2 * DAY), submittedBy: null }),
    make(feedback, { Name: 'Thomas Weber', 'E-Mail': 'thomas@example.com', Gesamtbewertung: 'Unzufrieden', 'Was können wir verbessern?': 'Support hat nicht reagiert.', 'Was genau lief schief?': 'Der Support hat 3 Tage nicht auf mein Ticket reagiert.', 'Datenschutz-Einwilligung': true }, { status: 'new', submittedAt: isoAgo(3 * DAY) }),
    make(feedback, { Name: 'Julia Becker', 'E-Mail': 'julia@example.com', Gesamtbewertung: 'Sehr zufrieden', 'Was können wir verbessern?': 'Alles bestens.', 'Datenschutz-Einwilligung': true }, { status: 'archived', submittedAt: isoAgo(10 * DAY) }),
    make(feedback, { Name: 'Stefan Wagner', 'E-Mail': 'stefan@example.com', Gesamtbewertung: 'Zufrieden', 'Was können wir verbessern?': 'Mehr Zahlungsarten wären schön.', 'Datenschutz-Einwilligung': true }, { status: 'read', submittedAt: isoAgo(14 * DAY) }),
  )

  const kontakt = byId('form-kontakt')
  out.push(
    make(kontakt, { Name: 'Sandra Fischer', 'E-Mail': 'sandra@example.com', Betreff: 'Angebot anfordern', Nachricht: 'Bitte schicken Sie mir ein Angebot für 50 Lizenzen.', 'Datenschutz-Einwilligung': true }, { status: 'new', submittedAt: isoAgo(3 * HOUR) }),
    make(kontakt, { Name: 'Michael Koch', 'E-Mail': 'michael@example.com', Betreff: 'Frage zur Rechnung', Nachricht: 'Auf meiner letzten Rechnung stimmt der Betrag nicht.', 'Datenschutz-Einwilligung': true }, { status: 'new', submittedAt: isoAgo(1 * DAY) }),
    make(kontakt, { Name: 'Petra Klein', 'E-Mail': 'petra@example.com', Betreff: 'Demo-Termin', Nachricht: 'Können wir nächste Woche eine Demo vereinbaren?', 'Datenschutz-Einwilligung': true }, { status: 'read', submittedAt: isoAgo(4 * DAY) }),
    make(kontakt, { Name: 'Andreas Richter', 'E-Mail': 'andreas@example.com', Betreff: '', Nachricht: 'Wie kündige ich mein Abo?', 'Datenschutz-Einwilligung': true }, { status: 'archived', submittedAt: isoAgo(12 * DAY) }),
  )

  const bewerbung = byId('form-bewerbung')
  out.push(
    make(bewerbung, { Name: 'Nicole Wolf', 'E-Mail': 'nicole@example.com', Position: 'Vertrieb', 'Verfügbar ab': '2026-09-01', 'Lebenslauf (PDF)': 'lebenslauf_wolf.pdf', 'Datenschutz-Einwilligung': true }, { status: 'new', submittedAt: isoAgo(6 * HOUR) }),
    make(bewerbung, { Name: 'David Hofer', 'E-Mail': 'david@example.com', Position: 'Entwicklung', 'Verfügbar ab': '2026-08-15', 'Lebenslauf (PDF)': 'cv_hofer.pdf', 'Datenschutz-Einwilligung': true }, { status: 'read', submittedAt: isoAgo(2 * DAY) }),
    make(bewerbung, { Name: 'Laura Maier', 'E-Mail': 'laura@example.com', Position: 'Initiativbewerbung', 'Verfügbar ab': '', 'Lebenslauf (PDF)': 'maier_cv.pdf', 'Datenschutz-Einwilligung': true }, { status: 'new', submittedAt: isoAgo(5 * DAY) }),
  )

  const event = byId('form-event')
  out.push(
    make(event, { Name: 'Markus Braun', 'E-Mail': 'markus@example.com', 'Anzahl Personen': 2, Verpflegung: 'Vegetarisch', 'Newsletter abonnieren': true, 'Datenschutz-Einwilligung': true }, { status: 'new', submittedAt: isoAgo(2 * HOUR) }),
    make(event, { Name: 'Sabine Krüger', 'E-Mail': 'sabine@example.com', 'Anzahl Personen': 1, Verpflegung: 'Vegan', 'Newsletter abonnieren': false, 'Datenschutz-Einwilligung': true }, { status: 'new', submittedAt: isoAgo(18 * HOUR) }),
    make(event, { Name: 'Frank Schulz', 'E-Mail': 'frank@example.com', 'Anzahl Personen': 4, Verpflegung: 'Mit Fleisch', 'Newsletter abonnieren': true, 'Datenschutz-Einwilligung': true }, { status: 'read', submittedAt: isoAgo(2 * DAY) }),
    make(event, { Name: 'Christina Vogel', 'E-Mail': 'christina@example.com', 'Anzahl Personen': 2, Verpflegung: 'Vegetarisch', 'Newsletter abonnieren': false, 'Datenschutz-Einwilligung': true }, { status: 'read', submittedAt: isoAgo(4 * DAY) }),
    make(event, { Name: 'Oliver Lang', 'E-Mail': 'oliver@example.com', 'Anzahl Personen': 3, Verpflegung: 'Mit Fleisch', 'Newsletter abonnieren': true, 'Datenschutz-Einwilligung': true }, { status: 'archived', submittedAt: isoAgo(8 * DAY) }),
  )

  const newsletter = byId('form-newsletter')
  out.push(
    make(newsletter, { 'E-Mail': 'interessent1@example.com', 'Datenschutz-Einwilligung': true }, { status: 'new', submittedAt: isoAgo(10 * HOUR) }),
    make(newsletter, { 'E-Mail': 'interessent2@example.com', 'Datenschutz-Einwilligung': true }, { status: 'read', submittedAt: isoAgo(3 * DAY) }),
  )

  return out
}

let SUBMISSIONS: FormSubmission[] = buildSubmissions()

// Keep each schema's submissionCount in sync with the live submission list.
function recountSubmissions(schemaId: string): void {
  const schema = SCHEMAS.find((s) => s.id === schemaId)
  if (schema) {
    schema.submissionCount = SUBMISSIONS.filter((s) => s.formSchemaId === schemaId).length
  }
}

// ---------------------------------------------------------------------------
// Seed — share links (FD-1 distribution layer; powers the FD-2 overview).
// `status` is derived on read (expiry / answer limit override 'active'); a
// 'disabled' value is sticky. Real token + public route stay Luke's lane (FD-4).
// ---------------------------------------------------------------------------

const SHARE_HOST = 'https://forms.zentria.tech/r/'

function genToken(): string {
  // URL-safe random token; crypto is available in the MSW browser worker.
  const bytes = new Uint8Array(8)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (b) => b.toString(36).padStart(2, '0'))
    .join('')
    .slice(0, 12)
}

function deriveShareStatus(link: FormShareLink): ShareLinkStatus {
  if (link.status === 'disabled') return 'disabled'
  if (link.expiresAt && new Date(link.expiresAt).getTime() < Date.now()) return 'expired'
  if (link.maxSubmissions != null && link.submissions >= link.maxSubmissions) return 'expired'
  return 'active'
}

/** Output view of a link with its status derived from expiry / limit. */
function shareOut(link: FormShareLink): FormShareLink {
  return { ...link, status: deriveShareStatus(link) }
}

let shareSeq = 1
const newShareId = () => `shl-${shareSeq++}`

function makeShareLink(
  over: Partial<FormShareLink> & Pick<FormShareLink, 'formSchemaId' | 'channel'>,
): FormShareLink {
  const token = over.token ?? genToken()
  return {
    id: newShareId(),
    tenantId: TENANT,
    token,
    url: `${SHARE_HOST}${token}`,
    expiresAt: null,
    maxSubmissions: null,
    views: 0,
    submissions: 0,
    status: 'active',
    createdBy: CURRENT_USER.name,
    createdAt: isoAgo(5 * DAY),
    ...over,
  }
}

let SHARE_LINKS: FormShareLink[] = [
  makeShareLink({
    formSchemaId: 'form-feedback',
    channel: 'link',
    views: 142,
    submissions: 6,
    createdAt: isoAgo(30 * DAY),
  }),
  makeShareLink({
    formSchemaId: 'form-feedback',
    channel: 'qr',
    views: 38,
    submissions: 2,
    createdAt: isoAgo(12 * DAY),
  }),
  makeShareLink({
    formSchemaId: 'form-kontakt',
    channel: 'email',
    views: 64,
    submissions: 4,
    expiresAt: isoAgo(-20 * DAY), // expires in 20 days
    createdAt: isoAgo(18 * DAY),
  }),
  makeShareLink({
    formSchemaId: 'form-event',
    channel: 'link',
    views: 210,
    submissions: 5,
    expiresAt: isoAgo(3 * DAY), // already expired (registration over)
    createdAt: isoAgo(25 * DAY),
  }),
]

// ---------------------------------------------------------------------------
// Seed — webhooks + deliveries (not surfaced by the page yet; kept consistent)
// ---------------------------------------------------------------------------

let WEBHOOKS: FormWebhook[] = [
  {
    id: 'wh-1',
    formSchemaId: 'form-kontakt',
    tenantId: TENANT,
    url: 'https://hooks.zentria.tech/forms/kontakt',
    secret: '••••a1b2',
    events: ['submission.created'],
    active: true,
    lastTriggeredAt: isoAgo(1 * DAY),
    lastStatus: '200',
    createdAt: isoAgo(30 * DAY),
    updatedAt: isoAgo(1 * DAY),
  },
]

const DELIVERIES: WebhookDelivery[] = [
  {
    id: 'dlv-1',
    webhookId: 'wh-1',
    submissionId: 'sub-7',
    tenantId: TENANT,
    payload: { event: 'submission.created' },
    status: 'delivered',
    attemptCount: 1,
    maxAttempts: 5,
    nextAttemptAt: isoAgo(1 * DAY),
    lastError: null,
    lastResponseCode: 200,
    createdAt: isoAgo(1 * DAY),
    deliveredAt: isoAgo(1 * DAY),
  },
]

// ---------------------------------------------------------------------------
// ID helpers
// ---------------------------------------------------------------------------

let schemaSeq = 100
let webhookSeq = 100
const newSchemaId = () => `form-${schemaSeq++}`
const newSubmissionId = () => `sub-${subSeq++}`
const newWebhookId = () => `wh-${webhookSeq++}`

function reqId(): string {
  return Math.random().toString(36).slice(2, 10)
}

// ---------------------------------------------------------------------------
// Export helpers (real Blob download — CSV; xlsx mirrors berichte: CSV bytes
// with the spreadsheet content-type, sufficient for the demo download path).
// ---------------------------------------------------------------------------

function escapeCsv(value: unknown): string {
  if (value === null || value === undefined) return ''
  const s = typeof value === 'boolean' ? (value ? 'Ja' : 'Nein') : String(value)
  return /[";\r\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

function submissionsToCsv(schema: FormSchema, subs: FormSubmission[]): string {
  const dataFields = schema.fields.filter((fld) => fld.label !== '__page_break__')
  const header = ['Eingegangen am', 'Absender', 'Status', ...dataFields.map((fld) => fld.label)]
  const rows = subs.map((sub) => [
    escapeCsv(sub.submittedAt),
    escapeCsv(sub.submittedBy ?? ''),
    escapeCsv(sub.status),
    ...dataFields.map((fld) => escapeCsv(sub.answers[fld.id])),
  ])
  // UTF-8 BOM so Excel opens umlauts correctly.
  return '﻿' + [header.map(escapeCsv).join(';'), ...rows.map((r) => r.join(';'))].join('\r\n')
}

function slugify(name: string): string {
  return name
    .replace(/ä/g, 'ae').replace(/ö/g, 'oe').replace(/ü/g, 'ue').replace(/ß/g, 'ss')
    .replace(/[^a-zA-Z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .toLowerCase()
}

const EXPORT_CONTENT_TYPES: Record<string, string> = {
  csv: 'text/csv;charset=utf-8',
  xlsx: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const formulareHandlers = [
  // ── Schemas: list ──
  http.get(`${API}/api/v1/formulare/schemas`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    const isTemplate = url.searchParams.get('isTemplate')
    const search = url.searchParams.get('search')?.toLowerCase()
    const limit = Number(url.searchParams.get('limit') ?? '0')
    const offset = Number(url.searchParams.get('offset') ?? '0')

    let items = SCHEMAS.filter((s) => !s.deletedAt)
    if (status) items = items.filter((s) => s.status === status)
    if (isTemplate != null) items = items.filter((s) => s.isTemplate === (isTemplate === 'true'))
    if (search) {
      items = items.filter(
        (s) => s.title.toLowerCase().includes(search) || s.description.toLowerCase().includes(search),
      )
    }
    const total = items.length
    if (offset) items = items.slice(offset)
    if (limit) items = items.slice(0, limit)
    return HttpResponse.json({ items, total })
  }),

  // ── Schemas: create ──
  http.post(`${API}/api/v1/formulare/schemas`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as CreateFormSchemaInput
    const now = new Date().toISOString()
    const schema: FormSchema = {
      id: newSchemaId(),
      tenantId: TENANT,
      title: body.title ?? 'Neues Formular',
      description: body.description ?? '',
      fields: (body.fields ?? []).map((fld, i) => ({ ...fld, id: fld.id || `fld-new-${reqId()}-${i}` })),
      status: body.status ?? 'draft',
      isTemplate: body.isTemplate ?? false,
      isPublic: body.isPublic ?? false,
      pageCount: body.pageCount ?? 1,
      closedMessage: body.closedMessage,
      thankYouMessage: body.thankYouMessage,
      redirectUrl: body.redirectUrl,
      notifications: body.notifications ?? { recipients: [], confirmToSubmitter: false },
      intakeTargetId: body.intakeTargetId,
      submissionCount: 0,
      createdBy: 'Du',
      createdAt: now,
      updatedAt: now,
      deletedAt: null,
    }
    SCHEMAS = [schema, ...SCHEMAS]
    return HttpResponse.json(schema, { status: 201 })
  }),

  // ── Schemas: export submissions (before the single-schema GET) ──
  http.get(
    `${API}/api/v1/formulare/schemas/:schemaId/submissions/export`,
    ({ params, request }) => {
      const schema = SCHEMAS.find((s) => s.id === params.schemaId)
      if (!schema) return new HttpResponse(null, { status: 404 })
      const url = new URL(request.url)
      const format = (url.searchParams.get('format') ?? 'csv') as 'csv' | 'xlsx'
      const subs = SUBMISSIONS.filter((s) => s.formSchemaId === schema.id)
      const filename = `${slugify(schema.title)}_eingaenge.${format}`
      return new HttpResponse(submissionsToCsv(schema, subs), {
        headers: {
          'Content-Type': EXPORT_CONTENT_TYPES[format] ?? 'application/octet-stream',
          'Content-Disposition': `attachment; filename="${filename}"`,
        },
      })
    },
  ),

  // ── Schemas: stats (mirrors the real backend's 4-counter FormStats proto,
  // envelope + snake_case, so the client's unwrap/mapping is exercised as-is
  // against the demo backend too — see formulare-client.ts:getFormStats) ──
  http.get(`${API}/api/v1/formulare/schemas/:schemaId/stats`, ({ params }) => {
    const schema = SCHEMAS.find((s) => s.id === params.schemaId)
    if (!schema) return new HttpResponse(null, { status: 404 })
    const subs = SUBMISSIONS.filter((s) => s.formSchemaId === schema.id)
    const weekAgo = Date.now() - 7 * DAY
    const monthAgo = Date.now() - 30 * DAY

    return HttpResponse.json({
      stats: {
        form_schema_id: schema.id,
        total_count: subs.length,
        new_count: subs.filter((s) => s.status === 'new').length,
        last_7d_count: subs.filter((s) => new Date(s.submittedAt).getTime() >= weekAgo).length,
        last_30d_count: subs.filter((s) => new Date(s.submittedAt).getTime() >= monthAgo).length,
      },
    })
  }),

  // ── Submissions: list by schema ──
  http.get(`${API}/api/v1/formulare/schemas/:schemaId/submissions`, ({ params, request }) => {
    const schema = SCHEMAS.find((s) => s.id === params.schemaId)
    if (!schema) return new HttpResponse(null, { status: 404 })
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    const limit = Number(url.searchParams.get('limit') ?? '0')
    const offset = Number(url.searchParams.get('offset') ?? '0')

    let items = SUBMISSIONS.filter((s) => s.formSchemaId === schema.id)
    if (status) items = items.filter((s) => s.status === status)
    items = items.sort((a, b) => b.submittedAt.localeCompare(a.submittedAt))
    const total = items.length
    if (offset) items = items.slice(offset)
    if (limit) items = items.slice(0, limit)
    return HttpResponse.json({ items, total })
  }),

  // ── Submissions: create ──
  http.post(`${API}/api/v1/formulare/schemas/:schemaId/submissions`, async ({ params, request }) => {
    const schema = SCHEMAS.find((s) => s.id === params.schemaId)
    if (!schema) return new HttpResponse(null, { status: 404 })
    const body = (await request.json().catch(() => ({}))) as CreateSubmissionInput
    const now = new Date().toISOString()
    const submission: FormSubmission = {
      id: newSubmissionId(),
      formSchemaId: schema.id,
      tenantId: TENANT,
      answers: body.answers ?? {},
      status: 'new',
      submittedBy: body.submittedBy ?? null,
      ipAddress: '127.0.0.1',
      submittedAt: now,
    }
    // FO-4 — mock the notification dispatch off the schema's config.
    const cfg = schema.notifications
    if (cfg && (cfg.recipients.length > 0 || cfg.confirmToSubmitter)) {
      const emailField = schema.fields.find((fld) => fld.type === 'email')
      const submitterEmail = emailField
        ? String((body.answers ?? {})[emailField.id] ?? '').trim()
        : ''
      submission.notifications = {
        recipients: [...cfg.recipients],
        confirmationTo: cfg.confirmToSubmitter && submitterEmail ? submitterEmail : null,
        sentAt: now,
      }
    }
    SUBMISSIONS = [submission, ...SUBMISSIONS]
    recountSubmissions(schema.id)
    return HttpResponse.json(submission, { status: 201 })
  }),

  // ── Share links: list by schema (FD-1 / FD-2) ──
  http.get(`${API}/api/v1/formulare/schemas/:schemaId/share-links`, ({ params }) => {
    const links = SHARE_LINKS.filter((l) => l.formSchemaId === params.schemaId)
      .map(shareOut)
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
    return HttpResponse.json(links)
  }),

  // ── Share links: create ──
  http.post(`${API}/api/v1/formulare/schemas/:schemaId/share-links`, async ({ params, request }) => {
    const schema = SCHEMAS.find((s) => s.id === params.schemaId)
    if (!schema) return new HttpResponse(null, { status: 404 })
    const body = (await request.json().catch(() => ({}))) as CreateShareLinkInput
    const link = makeShareLink({
      formSchemaId: schema.id,
      channel: body.channel ?? 'link',
      expiresAt: body.expiresAt ?? null,
      maxSubmissions:
        body.maxSubmissions != null && body.maxSubmissions > 0
          ? body.maxSubmissions
          : null,
      views: 0,
      submissions: 0,
      createdAt: new Date().toISOString(),
    })
    SHARE_LINKS = [link, ...SHARE_LINKS]
    return HttpResponse.json(shareOut(link), { status: 201 })
  }),

  // ── Webhooks: list by schema ──
  http.get(`${API}/api/v1/formulare/schemas/:schemaId/webhooks`, ({ params }) => {
    return HttpResponse.json(WEBHOOKS.filter((w) => w.formSchemaId === params.schemaId))
  }),

  // ── Webhooks: create ──
  http.post(`${API}/api/v1/formulare/schemas/:schemaId/webhooks`, async ({ params, request }) => {
    const body = (await request.json().catch(() => ({}))) as CreateWebhookInput
    const now = new Date().toISOString()
    const webhook: FormWebhook = {
      id: newWebhookId(),
      formSchemaId: String(params.schemaId),
      tenantId: TENANT,
      url: body.url,
      secret: '••••' + reqId().slice(0, 4),
      events: body.events ?? [],
      active: body.active ?? true,
      lastTriggeredAt: null,
      lastStatus: null,
      createdAt: now,
      updatedAt: now,
    }
    WEBHOOKS = [webhook, ...WEBHOOKS]
    return HttpResponse.json(webhook, { status: 201 })
  }),

  // ── Schemas: duplicate (before single-schema GET/PATCH/DELETE) ──
  http.post(`${API}/api/v1/formulare/schemas/:id/duplicate`, async ({ params, request }) => {
    const source = SCHEMAS.find((s) => s.id === params.id)
    if (!source) return new HttpResponse(null, { status: 404 })
    const body = (await request.json().catch(() => ({}))) as DuplicateFormSchemaInput
    const now = new Date().toISOString()
    const copy: FormSchema = {
      ...source,
      id: newSchemaId(),
      title: body.title?.trim() || `${source.title} (Kopie)`,
      // A duplicate of a template becomes a regular, editable draft form.
      isTemplate: false,
      status: source.isTemplate ? 'draft' : source.status,
      submissionCount: 0,
      fields: source.fields.map((fld, i) => ({ ...fld, id: `fld-cp-${reqId()}-${i}` })),
      createdBy: 'Du',
      createdAt: now,
      updatedAt: now,
      deletedAt: null,
    }
    SCHEMAS = [copy, ...SCHEMAS]
    return HttpResponse.json(copy, { status: 201 })
  }),

  // ── Schemas: get one ──
  http.get(`${API}/api/v1/formulare/schemas/:id`, ({ params }) => {
    const schema = SCHEMAS.find((s) => s.id === params.id && !s.deletedAt)
    if (!schema) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json(schema)
  }),

  // ── Schemas: update ──
  http.patch(`${API}/api/v1/formulare/schemas/:id`, async ({ params, request }) => {
    const schema = SCHEMAS.find((s) => s.id === params.id)
    if (!schema) return new HttpResponse(null, { status: 404 })
    const body = (await request.json().catch(() => ({}))) as UpdateFormSchemaInput
    if (body.title !== undefined) schema.title = body.title
    if (body.description !== undefined) schema.description = body.description
    if (body.fields !== undefined) {
      schema.fields = body.fields.map((fld, i) => ({ ...fld, id: fld.id || `fld-new-${reqId()}-${i}` }))
    }
    if (body.status !== undefined) schema.status = body.status
    if (body.isTemplate !== undefined) schema.isTemplate = body.isTemplate
    if (body.isPublic !== undefined) schema.isPublic = body.isPublic
    if (body.pageCount !== undefined) schema.pageCount = body.pageCount
    if (body.closedMessage !== undefined) schema.closedMessage = body.closedMessage
    if (body.thankYouMessage !== undefined) schema.thankYouMessage = body.thankYouMessage
    if (body.redirectUrl !== undefined) schema.redirectUrl = body.redirectUrl
    if (body.notifications !== undefined) schema.notifications = body.notifications
    if (body.intakeTargetId !== undefined) schema.intakeTargetId = body.intakeTargetId
    schema.updatedAt = new Date().toISOString()
    return HttpResponse.json(schema)
  }),

  // ── Schemas: delete (soft) ──
  http.delete(`${API}/api/v1/formulare/schemas/:id`, ({ params }) => {
    const exists = SCHEMAS.some((s) => s.id === params.id)
    if (!exists) return new HttpResponse(null, { status: 404 })
    SCHEMAS = SCHEMAS.filter((s) => s.id !== params.id)
    SUBMISSIONS = SUBMISSIONS.filter((s) => s.formSchemaId !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // ── Submissions: get one ──
  http.get(`${API}/api/v1/formulare/submissions/:id`, ({ params }) => {
    const submission = SUBMISSIONS.find((s) => s.id === params.id)
    if (!submission) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json(submission)
  }),

  // ── Submissions: update status ──
  http.patch(`${API}/api/v1/formulare/submissions/:id`, async ({ params, request }) => {
    const submission = SUBMISSIONS.find((s) => s.id === params.id)
    if (!submission) return new HttpResponse(null, { status: 404 })
    const body = (await request.json().catch(() => ({}))) as UpdateSubmissionStatusInput
    if (body.status !== undefined) submission.status = body.status
    return HttpResponse.json(submission)
  }),

  // ── Share links: update (disable / re-activate) ──
  http.patch(`${API}/api/v1/formulare/share-links/:id`, async ({ params, request }) => {
    const link = SHARE_LINKS.find((l) => l.id === params.id)
    if (!link) return new HttpResponse(null, { status: 404 })
    const body = (await request.json().catch(() => ({}))) as UpdateShareLinkInput
    if (body.status !== undefined) link.status = body.status
    return HttpResponse.json(shareOut(link))
  }),

  // ── Share links: delete ──
  http.delete(`${API}/api/v1/formulare/share-links/:id`, ({ params }) => {
    const exists = SHARE_LINKS.some((l) => l.id === params.id)
    if (!exists) return new HttpResponse(null, { status: 404 })
    SHARE_LINKS = SHARE_LINKS.filter((l) => l.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // ── Deliveries: by webhook (before the single-webhook GET) ──
  http.get(`${API}/api/v1/formulare/webhooks/:id/deliveries`, ({ params }) => {
    return HttpResponse.json(DELIVERIES.filter((d) => d.webhookId === params.id))
  }),

  // ── Webhooks: get one ──
  http.get(`${API}/api/v1/formulare/webhooks/:id`, ({ params }) => {
    const webhook = WEBHOOKS.find((w) => w.id === params.id)
    if (!webhook) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json(webhook)
  }),

  // ── Webhooks: update ──
  http.patch(`${API}/api/v1/formulare/webhooks/:id`, async ({ params, request }) => {
    const webhook = WEBHOOKS.find((w) => w.id === params.id)
    if (!webhook) return new HttpResponse(null, { status: 404 })
    const body = (await request.json().catch(() => ({}))) as UpdateWebhookInput
    if (body.url !== undefined) webhook.url = body.url
    if (body.events !== undefined) webhook.events = body.events
    if (body.active !== undefined) webhook.active = body.active
    webhook.updatedAt = new Date().toISOString()
    return HttpResponse.json(webhook)
  }),

  // ── Webhooks: delete ──
  http.delete(`${API}/api/v1/formulare/webhooks/:id`, ({ params }) => {
    WEBHOOKS = WEBHOOKS.filter((w) => w.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // ── Deliveries: list all ──
  http.get(`${API}/api/v1/formulare/deliveries`, ({ request }) => {
    const url = new URL(request.url)
    const webhookId = url.searchParams.get('webhook_id')
    const submissionId = url.searchParams.get('submission_id')
    const status = url.searchParams.get('status')
    let items = DELIVERIES
    if (webhookId) items = items.filter((d) => d.webhookId === webhookId)
    if (submissionId) items = items.filter((d) => d.submissionId === submissionId)
    if (status) items = items.filter((d) => d.status === status)
    return HttpResponse.json(items)
  }),
]
