import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS, CURRENT_USER } from '../data/shared-ids'
import { daysAgo, hoursAgo } from '../data/date-helpers'
import { WIKI_LINK_CLASS } from '@/modules/wiki/extensions/WikiLinkExtension'
import { WIKI_MENTION_CLASS } from '@/modules/wiki/extensions/WikiMentionExtension'
import type {
  WikiArticle,
  WikiCategory,
  WikiVersion,
  WikiAttachment,
  WikiShareToken,
} from '@/api/wiki-types'
import type { WikiTemplate } from '@/types/wiki'

const API = API_BASE_URL
const BASE = `${API}/api/v1/wiki`

// ============================================================================
// Wire-shape Demo-Daten
// (snake_case — spiegelt Backend-API-Response, wird von wiki-adapter.ts gemappt)
// ============================================================================

const CATEGORIES: WikiCategory[] = [
  {
    id: 'wcat-001',
    tenant_id: 'tenant-001',
    name: 'Allgemein',
    parent_id: null,
    position: 0,
    created_at: daysAgo(60) + 'T08:00:00Z',
    updated_at: daysAgo(30) + 'T08:00:00Z',
  },
  {
    id: 'wcat-002',
    tenant_id: 'tenant-001',
    name: 'Prozesse & Workflows',
    parent_id: null,
    position: 1,
    created_at: daysAgo(55) + 'T09:00:00Z',
    updated_at: daysAgo(20) + 'T09:00:00Z',
  },
  {
    id: 'wcat-003',
    tenant_id: 'tenant-001',
    name: 'IT & Infrastruktur',
    parent_id: null,
    position: 2,
    created_at: daysAgo(50) + 'T10:00:00Z',
    updated_at: daysAgo(10) + 'T10:00:00Z',
  },
  {
    id: 'wcat-004',
    tenant_id: 'tenant-001',
    name: 'Onboarding',
    parent_id: 'wcat-001',
    position: 0,
    created_at: daysAgo(45) + 'T08:30:00Z',
    updated_at: daysAgo(5) + 'T08:30:00Z',
  },
  {
    id: 'wcat-005',
    tenant_id: 'tenant-001',
    name: 'Datenschutz & DSGVO',
    parent_id: null,
    position: 3,
    created_at: daysAgo(40) + 'T11:00:00Z',
    updated_at: daysAgo(7) + 'T11:00:00Z',
  },
]

// TipTap-JSON-Content als Record<string, unknown> (Wire-Shape erwartet JSONB)
function tiptapDoc(text: string): Record<string, unknown> {
  return {
    type: 'doc',
    content: [
      {
        type: 'paragraph',
        content: [{ type: 'text', text }],
      },
    ],
  }
}

// Inline demo diagram (data URL) so the rich-block showcase article (wart-003)
// renders a real figure without a backend upload pipeline.
const DEMO_FLOW_SVG =
  "<svg xmlns='http://www.w3.org/2000/svg' width='520' height='150'>" +
  "<rect width='520' height='150' fill='#f8fafc'/>" +
  "<rect x='20' y='52' width='110' height='46' rx='8' fill='#e0e7ff'/>" +
  "<rect x='205' y='52' width='110' height='46' rx='8' fill='#dbeafe'/>" +
  "<rect x='390' y='52' width='110' height='46' rx='8' fill='#dcfce7'/>" +
  "<text x='75' y='80' font-family='sans-serif' font-size='13' text-anchor='middle' fill='#3730a3'>Angebot</text>" +
  "<text x='260' y='80' font-family='sans-serif' font-size='13' text-anchor='middle' fill='#1e40af'>Auftrag</text>" +
  "<text x='445' y='80' font-family='sans-serif' font-size='13' text-anchor='middle' fill='#166534'>Rechnung</text>" +
  "<line x1='130' y1='75' x2='205' y2='75' stroke='#94a3b8' stroke-width='2'/>" +
  "<line x1='315' y1='75' x2='390' y2='75' stroke='#94a3b8' stroke-width='2'/>" +
  '</svg>'
const DEMO_FLOW_DATA_URL = 'data:image/svg+xml,' + encodeURIComponent(DEMO_FLOW_SVG)

// ----------------------------------------------------------------------------
// Block-document builders (Phase B) — produce the `{ rows }` JSONB the shared
// document engine consumes. IDs come from a load-time counter (stable per
// session, deterministic across reloads since seeds build in the same order).
// ----------------------------------------------------------------------------

let _seedBlockId = 0
const sbid = (p: string): string => `${p}-${(_seedBlockId += 1)}`
type SeedBlock = Record<string, unknown> & { id: string; type: string }
type SeedRow = { id: string; columns: { id: string; width: number; blocks: SeedBlock[] }[] }

const h = (level: 1 | 2, text: string): SeedBlock => ({ id: sbid('b'), type: 'heading', level, text })
const p = (html: string): SeedBlock => ({ id: sbid('b'), type: 'text', html })
const ul = (items: string[]): SeedBlock => ({ id: sbid('b'), type: 'bullet', items })
const ol = (items: string[]): SeedBlock => ({ id: sbid('b'), type: 'bullet', items, ordered: true })
const callout = (
  variant: 'info' | 'success' | 'warning' | 'recommendation',
  html: string,
  title?: string,
): SeedBlock => ({ id: sbid('b'), type: 'callout', variant, html, ...(title ? { title } : {}) })
const img = (url: string, caption?: string, alt?: string): SeedBlock => ({
  id: sbid('b'),
  type: 'image',
  url,
  ...(caption ? { caption } : {}),
  ...(alt ? { alt } : {}),
})
const hr = (): SeedBlock => ({ id: sbid('b'), type: 'divider' })
const toggle = (title: string, html: string, open = false): SeedBlock => ({
  id: sbid('b'),
  type: 'toggle',
  title,
  html,
  open,
})
const code = (language: string, source: string): SeedBlock => ({
  id: sbid('b'),
  type: 'code',
  language,
  code: source,
})

/** One full-width row holding the given blocks. */
const row = (...blocks: SeedBlock[]): SeedRow => ({
  id: sbid('r'),
  columns: [{ id: sbid('c'), width: 1, blocks }],
})
/** A two-column row (text beside text/image), each column a block list. */
const cols = (left: SeedBlock[], right: SeedBlock[], widths: [number, number] = [1, 1]): SeedRow => ({
  id: sbid('r'),
  columns: [
    { id: sbid('c'), width: widths[0], blocks: left },
    { id: sbid('c'), width: widths[1], blocks: right },
  ],
})
/** Assemble a block document in the `{ rows }` JSONB shape. */
const doc = (...rows: SeedRow[]): Record<string, unknown> => ({ rows })

const ARTICLES: WikiArticle[] = [
  {
    id: 'wart-001',
    tenant_id: 'tenant-001',
    title: 'Willkommen im Cosmi-Wiki',
    slug: 'willkommen-im-cosmi-wiki',
    content: doc(
      row(h(1, 'Willkommen im Cosmi-Wiki')),
      row(
        p(
          '<p>Dies ist die zentrale Wissensbasis des Unternehmens — ein ruhiger Ort für alles, ' +
            'was das Team wissen muss: Prozesse, Anleitungen und Entscheidungen, an einem Ort und ' +
            'immer aktuell.</p>',
        ),
      ),
      row(
        p(
          '<p>Neu im Team? Starte mit <a data-wiki-link data-article-id="wart-002" ' +
            `data-slug="onboarding-neue-mitarbeitende" class="${WIKI_LINK_CLASS}">` +
            'Onboarding neuer Mitarbeitender</a>. Bei Fragen hilft dir ' +
            `<span data-mention-id="${IDS.users.julia}" class="${WIKI_MENTION_CLASS}">@Julia</span> ` +
            'gerne weiter.</p>',
        ),
      ),
      row(h(2, 'Was du hier findest')),
      row(
        p(
          '<p>Das Wiki ist nach Bereichen geordnet. Jeder Artikel hat eine verantwortliche Person ' +
            'und wird regelmässig geprüft, damit du dich auf den Inhalt verlassen kannst.</p>',
        ),
      ),
      row(
        ul([
          'Onboarding — die ersten Tage und Wochen',
          'Prozesse & Workflows — wie wir arbeiten',
          'IT & Infrastruktur — Tools, Zugänge, Notfälle',
        ]),
      ),
      cols(
        [
          callout(
            'info',
            '<p>Schreibe für die Person, die zum ersten Mal hier landet — eine klare Überschrift ' +
              'und kurze Absätze reichen oft schon.</p>',
            'Für Leser schreiben',
          ),
        ],
        [
          callout(
            'recommendation',
            '<p>Verlinke verwandte Artikel, statt sie zu kopieren. So bleibt Wissen an einer ' +
              'Stelle aktuell.</p>',
            'Statt kopieren: verlinken',
          ),
        ],
      ),
      row(h(2, 'So schreibst du einen guten Artikel')),
      row(
        ol([
          'Beginne mit dem Ergebnis, nicht mit dem Kontext.',
          'Verlinke verwandte Artikel statt sie zu kopieren.',
          'Halte den Artikel aktuell — markiere ihn als geprüft.',
        ]),
      ),
      row(h(2, 'Häufige Fragen')),
      row(
        toggle(
          'Wer darf Artikel bearbeiten?',
          '<p>Jedes Teammitglied mit Schreibrechten. Die verantwortliche Person prüft grössere ' +
            'Änderungen — kleine Korrekturen kannst du direkt vornehmen.</p>',
          true,
        ),
      ),
      row(
        toggle(
          'Wie oft wird ein Artikel geprüft?',
          '<p>Mindestens alle drei Monate. Nach der Prüfung markierst du den Artikel als ' +
            '<strong>geprüft</strong>, damit alle sehen, dass der Inhalt aktuell ist.</p>',
        ),
      ),
    ),
    author_id: CURRENT_USER.id,
    category_id: 'wcat-001',
    published: true,
    tags: ['Onboarding', 'Übersicht'],
    icon: 'Rocket',
    cover_url: 'grad:aurora',
    view_count: 342,
    created_at: daysAgo(58) + 'T09:00:00Z',
    updated_at: daysAgo(5) + 'T10:30:00Z',
  },
  {
    id: 'wart-002',
    tenant_id: 'tenant-001',
    title: 'Onboarding neuer Mitarbeitender',
    slug: 'onboarding-neue-mitarbeitende',
    content: doc(
      row(h(1, 'Onboarding neuer Mitarbeitender')),
      row(
        p(
          '<p>Diese Anleitung begleitet neue Teammitglieder durch die ersten Tage — von den ' +
            'Zugängen über die Systeme bis zu den ersten Aufgaben.</p>',
        ),
      ),
      row(h(2, 'Vor dem ersten Tag')),
      row(
        p('<p>Die Teamleitung legt rechtzeitig alle Zugänge an und bereitet den Arbeitsplatz vor.</p>'),
      ),
      row(ul(['E-Mail-Konto und Kalender', 'Zugang zu Cosmi und den Modulen', 'Hardware (Laptop, Headset)'])),
      row(h(2, 'Erste Woche')),
      row(
        p(
          '<p>Der Fokus liegt auf Kennenlernen — Menschen, Prozesse, Werkzeuge. ' +
            '<strong>Tag 1</strong> dient dem Ankommen: Begrüssung, Arbeitsplatz einrichten, Team ' +
            'kennenlernen. <strong>Tag 2–3</strong> führen in Cosmi ein: Kontakte, Aufgaben, ' +
            'Kalender und Kommunikation.</p>',
        ),
      ),
      row(
        callout(
          'recommendation',
          '<p>Plane für jede neue Person eine Patin oder einen Paten ein — das senkt die ' +
            'Einstiegshürde spürbar.</p>',
          'Patenschaft',
        ),
      ),
      row(h(2, 'Erster Monat')),
      row(
        p(
          '<p>Eigenständige Aufgaben übernehmen, Feedback-Gespräch nach vier Wochen: Was lief gut, ' +
            'wo gibt es offene Fragen, was braucht die Person noch?</p>',
        ),
      ),
    ),
    author_id: IDS.users.julia,
    category_id: 'wcat-004',
    published: true,
    tags: ['HR', 'Checkliste'],
    icon: 'Users',
    view_count: 158,
    created_at: daysAgo(45) + 'T08:00:00Z',
    updated_at: daysAgo(3) + 'T14:00:00Z',
  },
  {
    id: 'wart-003',
    tenant_id: 'tenant-001',
    title: 'Angebots- und Rechnungsprozess',
    slug: 'angebots-und-rechnungsprozess',
    content: doc(
      row(h(1, 'Angebots- und Rechnungsprozess')),
      row(
        p(
          '<p>Dieser Artikel beschreibt den vollständigen Prozess von der Angebotserstellung ' +
            'bis zur Rechnungsstellung inklusive Mahnwesen und DATEV-Export.</p>',
        ),
      ),
      row(img(DEMO_FLOW_DATA_URL, 'Die drei Phasen im Überblick')),
      row(h(2, 'Angebot erstellen')),
      row(
        p(
          '<p>Jedes Angebot startet aus einer Opportunity im CRM. Pflichtfelder sind Kunde, ' +
            'Positionen und Gültigkeitsdauer.</p>',
        ),
      ),
      cols(
        [callout('info', '<p>Angebote sind 30 Tage gültig, sofern nicht anders vereinbart.</p>', 'Gültigkeit')],
        [
          callout(
            'warning',
            '<p>Rabatte über 15 % müssen vor dem Versand von der Teamleitung freigegeben werden.</p>',
            'Rabatt-Freigabe',
          ),
        ],
      ),
      row(h(2, 'Auftrag & Rechnung')),
      row(
        p(
          '<p>Nach Auftragsbestätigung wird die Rechnung automatisch aus dem Angebot erzeugt. ' +
            'Die Rechnungsnummer folgt dem Schema:</p>' +
            '<pre><code>function invoiceNumber(year, seq) {\n' +
            '  return `R-${year}-${String(seq).padStart(4, "0")}`\n}</code></pre>',
        ),
      ),
      row(h(2, 'Mahnwesen')),
      row(
        p(
          '<p>Die erste Mahnung erfolgt 14 Tage nach Fälligkeit, die zweite nach weiteren 10 Tagen. ' +
            'Danach übernimmt das Inkasso.</p>',
        ),
      ),
      row(hr()),
      row(
        callout(
          'recommendation',
          '<p>Prüfe vor dem DATEV-Export immer die Kostenstellen-Zuordnung.</p>',
          'Vor dem Export',
        ),
      ),
    ),
    author_id: IDS.users.markus,
    category_id: 'wcat-002',
    published: true,
    tags: ['Finanzen', 'DATEV', 'Prozess'],
    icon: 'Receipt',
    cover_url: 'grad:sunset',
    view_count: 97,
    created_at: daysAgo(40) + 'T10:00:00Z',
    updated_at: daysAgo(10) + 'T16:00:00Z',
  },
  {
    id: 'wart-004',
    tenant_id: 'tenant-001',
    title: 'Backup & Disaster Recovery',
    slug: 'backup-disaster-recovery',
    content: doc(
      row(h(1, 'Backup & Disaster Recovery')),
      row(
        p(
          '<p>Diese Seite fasst die Backup-Strategie, die Wiederherstellungszeiten und die ' +
            'Schritte im Notfall zusammen.</p>',
        ),
      ),
      row(h(2, 'Strategie')),
      row(
        ul([
          'Tägliche inkrementelle Backups, wöchentliches Vollbackup',
          'Aufbewahrung 30 Tage, georedundant in der EU',
          'Monatlicher Wiederherstellungs-Test',
        ]),
      ),
      row(h(2, 'Zielzeiten')),
      row(
        p(
          '<p><strong>RTO</strong> (Wiederherstellungszeit): 4 Stunden. ' +
            '<strong>RPO</strong> (max. Datenverlust): 24 Stunden.</p>',
        ),
      ),
      row(
        callout(
          'warning',
          '<p>Im Notfall zuerst die IT-Teamleitung informieren — niemals eigenständig auf das ' +
            'Produktivsystem zurückspielen.</p>',
          'Notfall',
        ),
      ),
      row(h(2, 'Wiederherstellung (Staging)')),
      row(
        p('<p>Das jüngste Vollbackup in die Staging-Datenbank einspielen — nur nach Freigabe:</p>'),
      ),
      row(
        code(
          'bash',
          '# Letztes Vollbackup auf Staging zurückspielen\n' +
            'export PGPASSWORD="$STAGING_PW"\n' +
            'latest=$(ls -t /backups/full/*.dump | head -n1)\n' +
            'pg_restore --clean --no-owner \\\n' +
            '  --host staging.db.internal --username kmuhub \\\n' +
            '  --dbname cosmi "$latest"\n' +
            'echo "Restore abgeschlossen: $latest"',
        ),
      ),
    ),
    author_id: IDS.users.thomas,
    category_id: 'wcat-003',
    published: true,
    tags: ['IT', 'Notfall'],
    view_count: 64,
    created_at: daysAgo(30) + 'T11:00:00Z',
    updated_at: daysAgo(8) + 'T09:15:00Z',
  },
  {
    id: 'wart-005',
    tenant_id: 'tenant-001',
    title: 'DSGVO: Einwilligungen & Auskunftsrechte',
    slug: 'dsgvo-einwilligungen-auskunftsrechte',
    content: doc(
      row(h(1, 'DSGVO: Einwilligungen & Auskunftsrechte')),
      row(
        p(
          '<p>Überblick über die gesetzlichen Pflichten gemäß DSGVO — von der ' +
            'Einwilligungsverwaltung bis zur fristgerechten Auskunft.</p>',
        ),
      ),
      row(h(2, 'Pflichten im Überblick')),
      row(
        ul([
          'Einwilligungen dokumentiert und widerrufbar halten',
          'Auskunftsanfragen innerhalb von 30 Tagen beantworten',
          'Löschanfragen prüfen und protokollieren',
        ]),
      ),
      row(
        callout(
          'info',
          '<p>Cosmi protokolliert jede Einwilligung mit Zeitstempel und Zweck — die Nachweise ' +
            'findest du im Kontakt unter „Consent".</p>',
          'Nachweise in Cosmi',
        ),
      ),
    ),
    author_id: IDS.users.sarah,
    category_id: 'wcat-005',
    published: true,
    tags: ['DSGVO', 'Recht'],
    icon: 'ShieldCheck',
    view_count: 211,
    created_at: daysAgo(25) + 'T13:00:00Z',
    updated_at: daysAgo(2) + 'T11:00:00Z',
  },
  {
    id: 'wart-006',
    tenant_id: 'tenant-001',
    title: 'CRM: Kontakte und Deals verwalten',
    slug: 'crm-kontakte-deals',
    content: doc(
      row(h(1, 'CRM: Kontakte und Deals verwalten')),
      row(p('<p>So pflegst du das CRM-Modul sauber — von der Kontaktanlage bis zur Deal-Pipeline.</p>')),
      row(h(2, 'Kontakte')),
      row(
        ul([
          'Pflichtfelder: Name, Firma, primäre E-Mail',
          'Tags für Branche und Segment vergeben',
          'Dubletten vor dem Anlegen prüfen',
        ]),
      ),
      row(h(2, 'Deals & Pipeline')),
      row(
        p(
          '<p>Jeder Deal durchläuft die Phasen von „Erstkontakt" bis „Abschluss". Aktivitäten ' +
            'halten die Historie fest.</p>',
        ),
      ),
    ),
    author_id: IDS.users.laura,
    category_id: 'wcat-002',
    published: true,
    tags: ['CRM', 'Vertrieb'],
    view_count: 129,
    created_at: daysAgo(20) + 'T08:30:00Z',
    updated_at: hoursAgo(36),
  },
  {
    id: 'wart-007',
    tenant_id: 'tenant-001',
    title: 'Infrastruktur-Übersicht (intern)',
    slug: 'infrastruktur-uebersicht-intern',
    content: doc(
      row(h(1, 'Infrastruktur-Übersicht (intern)')),
      row(
        p(
          '<p>Interne Übersicht der Server, Dienste und Netzwerkkonfiguration — nur für ' +
            'IT-Administratoren.</p>',
        ),
      ),
      row(
        callout(
          'warning',
          '<p>Dieser Artikel ist als Entwurf nicht veröffentlicht und enthält sensible ' +
            'Zugangsdetails.</p>',
          'Vertraulich',
        ),
      ),
    ),
    author_id: IDS.users.thomas,
    category_id: 'wcat-003',
    published: false,
    tags: ['IT', 'Intern'],
    view_count: 18,
    created_at: daysAgo(15) + 'T16:00:00Z',
    updated_at: daysAgo(1) + 'T17:45:00Z',
  },
  {
    // Empty draft — exercises the joy-moment empty canvas (WP-5)
    id: 'wart-008',
    tenant_id: 'tenant-001',
    title: 'Reisekostenrichtlinie (Entwurf)',
    slug: 'reisekostenrichtlinie-entwurf',
    content: doc(),
    author_id: CURRENT_USER.id,
    category_id: 'wcat-001',
    published: false,
    tags: ['Entwurf'],
    icon: 'Briefcase',
    view_count: 3,
    created_at: daysAgo(2) + 'T10:00:00Z',
    updated_at: daysAgo(2) + 'T10:00:00Z',
  },
]

const VERSIONS: Record<string, WikiVersion[]> = {
  'wart-001': [
    {
      id: 'wver-001-1',
      article_id: 'wart-001',
      version_number: 1,
      content: tiptapDoc('Erste Version des Willkommensartikels.'),
      changed_by: CURRENT_USER.id,
      changed_at: daysAgo(58) + 'T09:00:00Z',
      change_note: 'Erstentwurf angelegt',
    },
    {
      id: 'wver-001-2',
      article_id: 'wart-001',
      version_number: 2,
      content: tiptapDoc('Überarbeitet mit Abteilungsstruktur.'),
      changed_by: IDS.users.julia,
      changed_at: daysAgo(20) + 'T11:00:00Z',
      change_note: 'Abteilungsstruktur ergänzt',
    },
    {
      id: 'wver-001-3',
      article_id: 'wart-001',
      version_number: 3,
      content: tiptapDoc(
        'Dies ist die zentrale Wissensbasis des Unternehmens. Hier finden Sie alle wichtigen Prozesse, Anleitungen und Informationen.',
      ),
      changed_by: CURRENT_USER.id,
      changed_at: daysAgo(5) + 'T10:30:00Z',
      change_note: 'Einleitung neu formuliert',
    },
  ],
  'wart-003': [
    {
      id: 'wver-003-1',
      article_id: 'wart-003',
      version_number: 1,
      content: tiptapDoc('Ursprünglicher Prozessentwurf.'),
      changed_by: IDS.users.markus,
      changed_at: daysAgo(40) + 'T10:00:00Z',
      change_note: 'Erstentwurf des Prozesses',
    },
    {
      id: 'wver-003-2',
      article_id: 'wart-003',
      version_number: 2,
      content: tiptapDoc(
        'Beschreibt den vollständigen Prozess von der Angebotserstellung bis zur Rechnungsstellung inkl. Mahnwesen und DATEV-Export.',
      ),
      changed_by: IDS.users.markus,
      changed_at: daysAgo(10) + 'T16:00:00Z',
      change_note: 'Mahnwesen und DATEV-Export ergänzt',
    },
  ],
}

// Inline SVG demo blob so an image attachment shows a real thumbnail/preview
// without a backend upload pipeline.
const DEMO_IMAGE_SVG =
  "<svg xmlns='http://www.w3.org/2000/svg' width='200' height='130'>" +
  "<rect width='200' height='130' fill='#eef2ff'/>" +
  "<rect x='18' y='22' width='70' height='34' rx='6' fill='#6366f1'/>" +
  "<rect x='112' y='22' width='70' height='34' rx='6' fill='#8b5cf6'/>" +
  "<rect x='65' y='80' width='70' height='34' rx='6' fill='#0ea5e9'/>" +
  "<line x1='53' y1='56' x2='100' y2='80' stroke='#475569' stroke-width='2'/>" +
  "<line x1='147' y1='56' x2='100' y2='80' stroke='#475569' stroke-width='2'/>" +
  "<text x='100' y='126' font-family='sans-serif' font-size='11' text-anchor='middle' fill='#334155'>Onboarding-Ablauf</text>" +
  '</svg>'
const DEMO_IMAGE_DATA_URL = 'data:image/svg+xml,' + encodeURIComponent(DEMO_IMAGE_SVG)

const ATTACHMENTS: Record<string, WikiAttachment[]> = {
  'wart-002': [
    {
      id: 'watt-001',
      article_id: 'wart-002',
      file_ref: 'onboarding-checkliste-2026.pdf',
      mime: 'application/pdf',
      size: 142300,
      uploaded_by: IDS.users.julia,
      created_at: daysAgo(44) + 'T09:00:00Z',
    },
    {
      id: 'watt-003',
      article_id: 'wart-002',
      file_ref: 'onboarding-ablauf.svg',
      mime: 'image/svg+xml',
      size: 4200,
      uploaded_by: IDS.users.julia,
      created_at: daysAgo(44) + 'T09:05:00Z',
      data_url: DEMO_IMAGE_DATA_URL,
    },
  ],
  'wart-004': [
    {
      id: 'watt-002',
      article_id: 'wart-004',
      file_ref: 'backup-runbook-v3.pdf',
      mime: 'application/pdf',
      size: 89100,
      uploaded_by: IDS.users.thomas,
      created_at: daysAgo(28) + 'T12:00:00Z',
    },
  ],
}

// Active share tokens (stateful — created via POST, revoked via DELETE).
const SHARE_TOKENS: WikiShareToken[] = [
  {
    id: 'wshare-seed-1',
    article_id: 'wart-001',
    token: 'demo-share-token-welcome',
    expires_at: null,
    permissions: ['read'],
    created_at: daysAgo(6) + 'T08:00:00Z',
  },
]

// Templates (WP-5) — reusable content scaffolds, stateful via MSW. Icons are
// lucide names from the wiki identity palette so the picker/preview line up.
const TEMPLATES: WikiTemplate[] = [
  {
    id: 'wtpl-001',
    name: 'Meeting-Protokoll',
    description: 'Standardvorlage für Meeting-Notizen',
    content:
      '<h2>Meeting: [Titel]</h2><p><strong>Datum:</strong> [Datum]<br><strong>Teilnehmer:</strong> [Namen]</p>' +
      '<h3>Agenda</h3><ol><li>[Punkt 1]</li><li>[Punkt 2]</li></ol>' +
      '<h3>Beschlüsse</h3><ul><li>[Beschluss]</li></ul>' +
      '<h3>Nächste Schritte</h3><ul><li>[Aktion] — [Verantwortlich] — [Frist]</li></ul>',
    icon: 'Calendar',
    category: 'meetings',
  },
  {
    id: 'wtpl-002',
    name: 'Post-Mortem',
    description: 'Analyse nach einem Vorfall oder Problem',
    content:
      '<h2>Post-Mortem: [Vorfall]</h2><p><strong>Datum:</strong> [Datum]<br><strong>Schweregrad:</strong> [Kritisch/Hoch/Mittel]</p>' +
      '<h3>Zusammenfassung</h3><p>[Was ist passiert?]</p>' +
      '<h3>Timeline</h3><ul><li>[HH:MM] — [Ereignis]</li></ul>' +
      '<h3>Root Cause</h3><p>[Ursache]</p>' +
      '<div data-callout="" data-variant="recommendation" class="wiki-callout"><p>[Wichtigste Massnahme zuerst]</p></div>' +
      '<h3>Lessons Learned</h3><ul><li>[Erkenntnis]</li></ul>',
    icon: 'ShieldCheck',
    category: 'incidents',
  },
  {
    id: 'wtpl-003',
    name: 'How-To Anleitung',
    description: 'Schritt-für-Schritt-Anleitung für Prozesse',
    content:
      '<h2>How-To: [Titel]</h2><p>[Kurzbeschreibung — was wird erreicht?]</p>' +
      '<h3>Voraussetzungen</h3><ul><li>[Voraussetzung 1]</li></ul>' +
      '<h3>Schritte</h3><ol><li>[Schritt 1]</li><li>[Schritt 2]</li><li>[Schritt 3]</li></ol>' +
      '<div data-callout="" data-variant="tip" class="wiki-callout"><p>[Praktischer Tipp für diesen Prozess]</p></div>',
    icon: 'GraduationCap',
    category: 'howto',
  },
  {
    id: 'wtpl-004',
    name: 'Projekt-Steckbrief',
    description: 'Kompakte Übersicht für ein neues Projekt',
    content:
      '<h2>[Projektname]</h2><p>[Ein Satz: worum geht es?]</p>' +
      '<h3>Ziel</h3><p>[Was soll erreicht werden?]</p>' +
      '<h3>Beteiligte</h3><ul><li>[Rolle] — [Person]</li></ul>' +
      '<h3>Meilensteine</h3><ul><li>[Datum] — [Meilenstein]</li></ul>',
    icon: 'Target',
    category: 'projects',
  },
]

// ============================================================================
// Helpers
// ============================================================================

function paginate<T>(items: T[], page: number, pageSize: number): { items: T[]; total: number } {
  const start = (page - 1) * pageSize
  return { items: items.slice(start, start + pageSize), total: items.length }
}

/** Collect searchable plain text from a block document's rows. */
function rowsText(rows: unknown[]): string {
  const parts: string[] = []
  for (const row of rows) {
    const columns = (row as Record<string, unknown>)?.columns
    if (!Array.isArray(columns)) continue
    for (const col of columns) {
      const blocks = (col as Record<string, unknown>)?.blocks
      if (!Array.isArray(blocks)) continue
      for (const b of blocks as Record<string, unknown>[]) {
        if (typeof b.text === 'string') parts.push(b.text)
        if (typeof b.title === 'string') parts.push(b.title)
        if (typeof b.html === 'string') parts.push(b.html.replace(/<[^>]*>/g, ' '))
        if (typeof b.caption === 'string') parts.push(b.caption)
        if (Array.isArray(b.items)) parts.push((b.items as string[]).join(' '))
      }
    }
  }
  return parts.join(' ')
}

/** Best-effort searchable plain text from the JSONB content field. */
function contentText(content: unknown): string {
  if (typeof content === 'string') return content
  if (!content || typeof content !== 'object') return ''
  const c = content as Record<string, unknown>
  if (Array.isArray(c.rows)) return rowsText(c.rows)
  if (typeof c.html === 'string') return c.html.replace(/<[^>]*>/g, ' ')
  if (typeof c.plain === 'string') return c.plain
  // walk a TipTap doc collecting text nodes
  const parts: string[] = []
  const walk = (node: Record<string, unknown>) => {
    if (typeof node.text === 'string') parts.push(node.text)
    for (const child of (node.content as Array<Record<string, unknown>>) ?? []) walk(child)
  }
  walk(c)
  return parts.join(' ')
}

/** Append a new version snapshot for an article (newest version_number wins). */
function appendVersion(
  articleId: string,
  content: Record<string, unknown>,
  changeNote?: string,
): void {
  const list = (VERSIONS[articleId] ??= [])
  const nextNumber = list.reduce((max, v) => Math.max(max, v.version_number), 0) + 1
  list.push({
    id: `wver-${articleId}-${nextNumber}-${Date.now()}`,
    article_id: articleId,
    version_number: nextNumber,
    content,
    changed_by: CURRENT_USER.id,
    changed_at: new Date().toISOString(),
    change_note: changeNote?.trim() || undefined,
  })
}

// ============================================================================
// Handlers
// ============================================================================

export const wikiHandlers = [
  // --- Categories ---

  http.get(`${BASE}/categories`, () => {
    return HttpResponse.json(CATEGORIES)
  }),

  http.post(`${BASE}/categories`, async ({ request }) => {
    const body = (await request.json()) as { name: string; parent_id?: string; position?: number }
    const newCat: WikiCategory = {
      id: `wcat-${Date.now()}`,
      tenant_id: 'tenant-001',
      name: body.name,
      parent_id: body.parent_id ?? null,
      position: body.position ?? CATEGORIES.length,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    CATEGORIES.push(newCat)
    return HttpResponse.json(newCat, { status: 201 })
  }),

  http.patch(`${BASE}/categories/:id`, async ({ params, request }) => {
    const category = CATEGORIES.find((c) => c.id === params.id)
    if (!category) return HttpResponse.json({ error: 'category not found' }, { status: 404 })
    const body = (await request.json()) as {
      name?: string
      parent_id?: string | null
      position?: number
    }
    if (body.name !== undefined) category.name = body.name
    if (body.parent_id !== undefined) category.parent_id = body.parent_id
    if (body.position !== undefined) category.position = body.position
    category.updated_at = new Date().toISOString()
    return HttpResponse.json(category)
  }),

  http.delete(`${BASE}/categories/:id`, ({ params }) => {
    const idx = CATEGORIES.findIndex((c) => c.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'category not found' }, { status: 404 })
    CATEGORIES.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // --- Articles list ---

  http.get(`${BASE}/articles`, ({ request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 20)
    const categoryId = url.searchParams.get('category_id')
    const authorId = url.searchParams.get('author_id')
    const publishedParam = url.searchParams.get('published')
    const searchQuery = url.searchParams.get('search')

    let filtered = [...ARTICLES]

    if (categoryId) {
      filtered = filtered.filter((a) => a.category_id === categoryId)
    }
    if (authorId) {
      filtered = filtered.filter((a) => a.author_id === authorId)
    }
    if (publishedParam !== null) {
      const wantPublished = publishedParam === 'true'
      filtered = filtered.filter((a) => a.published === wantPublished)
    }
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      filtered = filtered.filter(
        (a) =>
          a.title.toLowerCase().includes(q) ||
          // walk the JSONB content to plain text so TipTap docs match too
          contentText(a.content).toLowerCase().includes(q),
      )
    }

    const result = paginate(filtered, page, pageSize)
    return HttpResponse.json({ articles: result.items, total: result.total })
  }),

  // --- Article detail ---

  http.get(`${BASE}/articles/:id`, ({ params }) => {
    const article = ARTICLES.find((a) => a.id === params.id)
    if (!article) return HttpResponse.json({ error: 'article not found' }, { status: 404 })
    // Reading an article detail increments its view counter.
    article.view_count = (article.view_count ?? 0) + 1
    return HttpResponse.json(article)
  }),

  // --- Create article ---

  http.post(`${BASE}/articles`, async ({ request }) => {
    const body = (await request.json()) as Partial<WikiArticle>
    const newArticle: WikiArticle = {
      id: `wart-${Date.now()}`,
      tenant_id: 'tenant-001',
      title: body.title ?? 'Neuer Artikel',
      slug: (body.title ?? 'neuer-artikel').toLowerCase().replace(/\s+/g, '-').replace(/[^\w-]/g, ''),
      content: body.content ?? { rows: [] },
      author_id: CURRENT_USER.id,
      category_id: body.category_id ?? null,
      published: body.published ?? false,
      tags: body.tags ?? [],
      icon: body.icon ?? undefined,
      cover_url: body.cover_url ?? undefined,
      view_count: 0,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    ARTICLES.push(newArticle)
    return HttpResponse.json(newArticle, { status: 201 })
  }),

  // --- Update article ---

  http.put(`${BASE}/articles/:id`, async ({ params, request }) => {
    const article = ARTICLES.find((a) => a.id === params.id)
    if (!article) return HttpResponse.json({ error: 'article not found' }, { status: 404 })
    const body = (await request.json()) as Partial<WikiArticle> & { change_note?: string }
    if (body.title !== undefined) article.title = body.title
    if (body.content !== undefined) {
      article.content = body.content
      // A content edit creates a new version snapshot (newest = current).
      appendVersion(article.id, body.content, body.change_note)
    }
    if (body.category_id !== undefined) article.category_id = body.category_id
    if (body.published !== undefined) article.published = body.published
    if (body.tags !== undefined) article.tags = body.tags
    if (body.icon !== undefined) article.icon = body.icon ?? undefined
    if (body.cover_url !== undefined) article.cover_url = body.cover_url ?? undefined
    article.updated_at = new Date().toISOString()
    return HttpResponse.json(article)
  }),

  // --- Delete article ---

  http.delete(`${BASE}/articles/:id`, ({ params }) => {
    const idx = ARTICLES.findIndex((a) => a.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'article not found' }, { status: 404 })
    ARTICLES.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // --- Search ---

  http.get(`${BASE}/search`, ({ request }) => {
    const url = new URL(request.url)
    const q = (url.searchParams.get('q') ?? '').toLowerCase()
    const limit = Number(url.searchParams.get('limit') ?? 20)

    const results = ARTICLES.filter(
      (a) =>
        a.title.toLowerCase().includes(q) ||
        contentText(a.content).toLowerCase().includes(q),
    ).slice(0, limit)

    return HttpResponse.json({ articles: results })
  }),

  // --- Versions ---

  http.get(`${BASE}/articles/:id/versions`, ({ params }) => {
    const versions = VERSIONS[params.id as string] ?? []
    return HttpResponse.json(versions)
  }),

  http.get(`${BASE}/versions/:versionId`, ({ params }) => {
    for (const versions of Object.values(VERSIONS)) {
      const v = versions.find((ver) => ver.id === params.versionId)
      if (v) return HttpResponse.json(v)
    }
    return HttpResponse.json({ error: 'version not found' }, { status: 404 })
  }),

  http.post(`${BASE}/articles/:id/versions/:versionId/restore`, ({ params }) => {
    const article = ARTICLES.find((a) => a.id === params.id)
    const versions = VERSIONS[params.id as string] ?? []
    const version = versions.find((v) => v.id === params.versionId)
    if (!article || !version) {
      return HttpResponse.json({ error: 'not found' }, { status: 404 })
    }
    article.content = version.content
    article.updated_at = new Date().toISOString()
    // Restoring produces a new current version equal to the restored content.
    appendVersion(article.id, version.content, `Wiederhergestellt aus v${version.version_number}`)
    return HttpResponse.json(article)
  }),

  // --- Attachments ---

  http.get(`${BASE}/articles/:id/attachments`, ({ params }) => {
    const attachments = ATTACHMENTS[params.id as string] ?? []
    return HttpResponse.json(attachments)
  }),

  http.post(`${BASE}/articles/:id/attachments`, async ({ params, request }) => {
    const body = (await request.json()) as {
      file_ref: string
      mime: string
      size: number
      data_url?: string
    }
    const attachment: WikiAttachment = {
      id: `watt-${Date.now()}`,
      article_id: params.id as string,
      file_ref: body.file_ref,
      mime: body.mime,
      size: body.size,
      uploaded_by: CURRENT_USER.id,
      created_at: new Date().toISOString(),
      data_url: body.data_url,
    }
    if (!ATTACHMENTS[params.id as string]) {
      ATTACHMENTS[params.id as string] = []
    }
    ATTACHMENTS[params.id as string].push(attachment)
    return HttpResponse.json(attachment, { status: 201 })
  }),

  http.delete(`${BASE}/attachments/:attachmentId`, ({ params }) => {
    for (const [articleId, attachments] of Object.entries(ATTACHMENTS)) {
      const idx = attachments.findIndex((a) => a.id === params.attachmentId)
      if (idx !== -1) {
        ATTACHMENTS[articleId].splice(idx, 1)
        return new HttpResponse(null, { status: 204 })
      }
    }
    return HttpResponse.json({ error: 'attachment not found' }, { status: 404 })
  }),

  // --- Share tokens (stateful) ---

  http.get(`${BASE}/articles/:id/share`, ({ params }) => {
    const tokens = SHARE_TOKENS.filter((t) => t.article_id === params.id)
    return HttpResponse.json(tokens)
  }),

  http.post(`${BASE}/articles/:id/share`, async ({ params, request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      expires_at?: string
      permissions?: string[]
    }
    const token = {
      id: `wshare-${Date.now()}`,
      article_id: params.id as string,
      token: `demo-share-token-${Math.random().toString(36).slice(2)}`,
      expires_at: body.expires_at ?? null,
      permissions: body.permissions ?? ['read'],
      created_at: new Date().toISOString(),
    }
    SHARE_TOKENS.push(token)
    return HttpResponse.json(token, { status: 201 })
  }),

  http.delete(`${BASE}/share/:tokenId`, ({ params }) => {
    const idx = SHARE_TOKENS.findIndex((t) => t.id === params.tokenId)
    if (idx === -1) return HttpResponse.json({ error: 'token not found' }, { status: 404 })
    SHARE_TOKENS.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // --- Templates (WP-5, stateful) ---

  http.get(`${BASE}/templates`, () => {
    return HttpResponse.json(TEMPLATES)
  }),

  http.post(`${BASE}/templates`, async ({ request }) => {
    const body = (await request.json()) as Partial<WikiTemplate>
    const template: WikiTemplate = {
      id: `wtpl-${Date.now()}`,
      name: body.name ?? 'Neue Vorlage',
      description: body.description ?? '',
      content: body.content ?? '<p></p>',
      icon: body.icon ?? 'FileText',
      category: body.category ?? 'custom',
    }
    TEMPLATES.push(template)
    return HttpResponse.json(template, { status: 201 })
  }),

  http.put(`${BASE}/templates/:id`, async ({ params, request }) => {
    const template = TEMPLATES.find((t) => t.id === params.id)
    if (!template) return HttpResponse.json({ error: 'template not found' }, { status: 404 })
    const body = (await request.json()) as Partial<WikiTemplate>
    if (body.name !== undefined) template.name = body.name
    if (body.description !== undefined) template.description = body.description
    if (body.content !== undefined) template.content = body.content
    if (body.icon !== undefined) template.icon = body.icon
    if (body.category !== undefined) template.category = body.category
    return HttpResponse.json(template)
  }),

  http.delete(`${BASE}/templates/:id`, ({ params }) => {
    const idx = TEMPLATES.findIndex((t) => t.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'template not found' }, { status: 404 })
    TEMPLATES.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),
]
