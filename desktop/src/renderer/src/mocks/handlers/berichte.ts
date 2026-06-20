import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { buildSimplePdf, type PdfLine } from '@/modules/finanzen/lib/mini-pdf'
import { resolveSource } from '@/modules/berichte/report-sources/registry'
import { executeQuery } from '@/modules/berichte/report-sources/query-executor'
import { isBuilderQuery } from '@/api/berichte-types'
import type {
  BuilderQueryConfig,
  DashboardKPI,
  KpiBlock,
  ReportBlock,
  ReportColumn,
  ReportDefinition,
  ReportDocument,
  ReportResult,
  ReportRow,
  ReportSchedule,
  ReportSeries,
  ReportTemplate,
} from '@/api/berichte-types'

const API = API_BASE_URL
const TENANT = 't-demo'

// ---------------------------------------------------------------------------
// Dashboard KPIs
//
// The berichte (Reports/BI) backend is not yet available; the renderer fetches
// `/api/v1/berichte/*` directly. This handler serves a representative, stateful
// demo surface so all four tabs (Dashboard / Erstellen / Geplant / DATEV) live
// in demo mode. Backend gap (real time-series + KPI + schedule service) tracked
// in backend-gaps.md. No-Code query builder + real DATEV stay 🔒 (Luke).
// ---------------------------------------------------------------------------

const DEMO_KPIS: DashboardKPI[] = [
  { id: 'kpi-umsatz-mtd', label: 'Umsatz (MTD)', value: '84.532', unit: '€', change_percent: 12.4, module_id: 'finanzen' },
  { id: 'kpi-offene-rechnungen', label: 'Offene Rechnungen', value: '23', unit: '', change_percent: -8.1, module_id: 'finanzen' },
  { id: 'kpi-neue-leads', label: 'Neue Leads', value: '47', unit: '', change_percent: 23, module_id: 'crm' },
  { id: 'kpi-gewinnrate', label: 'Gewinnrate', value: '32', unit: '%', change_percent: 4.2, module_id: 'crm' },
  { id: 'kpi-reaktionszeit', label: 'Ø Reaktionszeit', value: '2,4', unit: 'Std.', change_percent: -15, module_id: 'helpdesk' },
  { id: 'kpi-offene-tickets', label: 'Offene Tickets', value: '18', unit: '', change_percent: 6, module_id: 'helpdesk' },
  { id: 'kpi-lagerwert', label: 'Lagerwert', value: '412.900', unit: '€', change_percent: 1.8, module_id: 'inventar' },
  { id: 'kpi-ausschussquote', label: 'Ausschussquote', value: '1,2', unit: '%', change_percent: -0.4, module_id: 'produktion' },
  { id: 'kpi-aktive-projekte', label: 'Aktive Projekte', value: '9', unit: '', change_percent: null, module_id: 'cross' },
]

// ---------------------------------------------------------------------------
// Report definitions (system, published)
// ---------------------------------------------------------------------------

function def(
  id: string,
  name: string,
  description: string,
  module: ReportDefinition['module'],
  configKind: string,
): ReportDefinition {
  const now = new Date().toISOString()
  return {
    id,
    tenant_id: TENANT,
    name,
    description,
    module,
    kind: 'system',
    query_config: { kind: configKind },
    default_format: 'pdf',
    created_by: null,
    is_published: true,
    created_at: now,
    updated_at: now,
  }
}

// Session-persistent: system defs (seeded) + custom defs created via the builder.
let DEMO_DEFINITIONS: ReportDefinition[] = [
  def('def-umsatz', 'Umsatzverlauf', 'Monatlicher Umsatz der letzten 12 Monate inkl. Trend.', 'finanzen', 'revenue'),
  def('def-tickets', 'Helpdesk-Auslastung', 'Offene und gelöste Tickets pro Monat.', 'helpdesk', 'tickets'),
  def('def-lager', 'Lagerwert-Entwicklung', 'Entwicklung des Lagerwerts über das Jahr.', 'inventar', 'inventory_value'),
  def('def-leads', 'Lead-Conversion', 'Conversion-Rate neuer Leads je Monat.', 'crm', 'lead_conversion'),
  def('def-datev-bwa', 'DATEV BWA', 'Betriebswirtschaftliche Auswertung (DATEV-Format).', 'finanzen', 'datev_bwa'),
  def('def-datev-susa', 'DATEV Summen- und Saldenliste', 'Summen- und Saldenliste je Konto (DATEV-Format).', 'finanzen', 'datev_susa'),
]
let definitionCounter = 0

// ---------------------------------------------------------------------------
// Report Documents (multi-page authoring, Schicht 4) — session-persistent.
// Seeded demo documents + ones created via "Neuer Bericht". Block layout:
// rows -> columns -> blocks. Backend gap (ReportDocument persistence + snapshots)
// tracked in backend-gaps.md.
// ---------------------------------------------------------------------------

let blockSeq = 0
const bid = (p: string): string => `${p}-${(blockSeq += 1)}`

/** One full-width column row. */
function r(...blocks: ReportBlock[]): ReportRow {
  return { id: bid('row'), columns: [{ id: bid('col'), blocks }] }
}
/** Multi-column row (KPI rows, text-beside-chart). */
function rc(...columns: { width?: number; blocks: ReportBlock[] }[]): ReportRow {
  return {
    id: bid('row'),
    columns: columns.map((c) => ({ id: bid('col'), width: c.width, blocks: c.blocks })),
  }
}
function kpiBlock(
  label: string,
  value: string,
  unit: string,
  change: number | null,
  source: string,
): KpiBlock {
  return { id: bid('kpi'), type: 'kpi', label, value, unit, changePercent: change, source }
}

const docAgo = (days: number): string =>
  new Date(Date.now() - days * 86400000).toISOString()

let DEMO_DOCUMENTS: ReportDocument[] = [
  {
    id: 'doc-verkauf-q2',
    tenant_id: TENANT,
    title: 'Verkaufsbericht Q2 2026',
    description: 'Quartalsbericht Vertrieb: Pipeline, Umsatz und Empfehlungen.',
    module: 'crm',
    status: 'released',
    created_by: 'u-demo',
    created_at: docAgo(14),
    updated_at: docAgo(2),
    released_at: docAgo(2),
    settings: {
      showHeader: true,
      showFooter: true,
      showPageNumbers: true,
      headerText: 'Verkaufsbericht Q2 2026',
      palette: 'cosmi',
    },
    rows: [
      r({ id: bid('cover'), type: 'cover', title: 'Verkaufsbericht', subtitle: 'Q2 2026 · Zentria GmbH', author: 'Vertriebsleitung', showDate: true }),
      r({ id: bid('pb'), type: 'pagebreak' }),
      r({ id: bid('h'), type: 'heading', level: 1, text: 'Zusammenfassung' }),
      r({ id: bid('t'), type: 'text', html: '<p>Das zweite Quartal zeigt ein Umsatzplus von <strong>18 %</strong> gegenüber Q1. Die Region Süd trägt den größten Anteil; die Gewinnrate ist stabil bei 32 %.</p>' }),
      rc(
        { blocks: [kpiBlock('Umsatz', '142.500', '€', 18, 'finanzen')] },
        { blocks: [kpiBlock('Gewinnrate', '32', '%', 4.2, 'crm')] },
        { blocks: [kpiBlock('Neue Deals', '37', '', 12, 'crm')] },
      ),
      r({ id: bid('h'), type: 'heading', level: 2, text: 'Umsatz nach Monat' }),
      rc(
        { width: 2, blocks: [{ id: bid('t'), type: 'text', html: '<p>Der Aufwärtstrend hält seit März an. Mai war der stärkste Monat seit Jahresbeginn.</p>' }] },
        { width: 3, blocks: [{ id: bid('c'), type: 'chart', definitionId: 'def-umsatz', viz: 'line', caption: 'Monatlicher Umsatz, letzte 12 Monate' }] },
      ),
      r({ id: bid('cl'), type: 'callout', variant: 'recommendation', title: 'Empfehlung', html: '<p>Vertriebskapazität in Region Süd ausbauen und das Verkaufsmodell an den höheren Abschlussraten ausrichten.</p>' }),
    ],
  },
  {
    id: 'doc-monat-juni',
    tenant_id: TENANT,
    title: 'Monatsbericht Juni 2026',
    description: 'Management-Monatsbericht: Kennzahlen, Finanzen, Helpdesk.',
    module: 'finanzen',
    status: 'final',
    created_by: 'u-demo',
    created_at: docAgo(6),
    updated_at: docAgo(1),
    released_at: null,
    settings: {
      showHeader: true,
      showFooter: true,
      showPageNumbers: true,
      headerText: 'Monatsbericht Juni 2026',
      palette: 'cosmi',
    },
    rows: [
      r({ id: bid('cover'), type: 'cover', title: 'Monatsbericht', subtitle: 'Juni 2026', author: 'Geschäftsführung', showDate: true }),
      r({ id: bid('h'), type: 'heading', level: 1, text: 'Kennzahlen im Überblick' }),
      rc(
        { blocks: [kpiBlock('Umsatz (MTD)', '84.532', '€', 12.4, 'finanzen')] },
        { blocks: [kpiBlock('Offene Rechnungen', '23', '', -8.1, 'finanzen')] },
        { blocks: [kpiBlock('Offene Tickets', '18', '', 6, 'helpdesk')] },
      ),
      r({ id: bid('t'), type: 'text', html: '<p>Der Monat verlief planmäßig. Die offenen Rechnungen sind gegenüber Mai gesunken.</p>' }),
      r({ id: bid('h'), type: 'heading', level: 2, text: 'Helpdesk-Auslastung' }),
      r({ id: bid('c'), type: 'chart', definitionId: 'def-tickets', viz: 'bar', caption: 'Offene und gelöste Tickets pro Monat' }),
      r({ id: bid('bl'), type: 'bullet', items: ['Reaktionszeit unter Zielwert', 'Zwei Eskalationen gelöst', 'CSAT stabil bei 4,5'] }),
    ],
  },
  {
    id: 'doc-helpdesk-kw24',
    tenant_id: TENANT,
    title: 'Helpdesk-Auslastung KW 24',
    description: 'Wöchentlicher Kurzbericht Support (Entwurf).',
    module: 'helpdesk',
    status: 'draft',
    created_by: 'u-demo',
    created_at: docAgo(1),
    updated_at: docAgo(0),
    released_at: null,
    settings: { showHeader: false, showFooter: true, showPageNumbers: true },
    rows: [
      r({ id: bid('cover'), type: 'cover', title: 'Helpdesk-Auslastung', subtitle: 'KW 24 · Entwurf', showDate: true }),
      r({ id: bid('h'), type: 'heading', level: 1, text: 'Notizen' }),
      r({ id: bid('t'), type: 'text', html: '<p>Erste Stichpunkte für den Wochenbericht. Diagramme folgen.</p>' }),
    ],
  },
]
let documentCounter = 0

// ---------------------------------------------------------------------------
// Report Templates — prebuilt block structures for "Neuer Bericht aus Vorlage".
// Placeholder content; the user fills it in. Charts/tables carry a caption but
// no source yet (picked in the editor, R-1b).
// ---------------------------------------------------------------------------

const TPL_SETTINGS: ReportDocument['settings'] = {
  showHeader: true,
  showFooter: true,
  showPageNumbers: true,
}

const DEMO_TEMPLATES: ReportTemplate[] = [
  {
    id: 'tpl-monat',
    title: 'Monats-/Managementbericht',
    description: 'Deckblatt, Zusammenfassung, Kennzahlen, Finanzen, Empfehlung.',
    module: 'cross',
    icon: 'CalendarRange',
    settings: TPL_SETTINGS,
    rows: [
      r({ id: bid('cover'), type: 'cover', title: 'Monatsbericht', subtitle: 'Monat Jahr', showDate: true }),
      r({ id: bid('h'), type: 'heading', level: 1, text: 'Zusammenfassung' }),
      r({ id: bid('t'), type: 'text', html: '<p>Kernaussage des Monats in zwei bis drei Sätzen.</p>' }),
      rc(
        { blocks: [kpiBlock('Umsatz', '—', '€', null, 'finanzen')] },
        { blocks: [kpiBlock('Kosten', '—', '€', null, 'finanzen')] },
        { blocks: [kpiBlock('Ergebnis', '—', '€', null, 'finanzen')] },
      ),
      r({ id: bid('h'), type: 'heading', level: 2, text: 'Finanzen' }),
      r({ id: bid('c'), type: 'chart', caption: 'Umsatz nach Monat' }),
      r({ id: bid('cl'), type: 'callout', variant: 'recommendation', title: 'Empfehlung', html: '<p>Maßnahmen für den nächsten Monat.</p>' }),
    ],
  },
  {
    id: 'tpl-sales',
    title: 'Vertriebsbericht',
    description: 'Kennzahlen, Pipeline-Trichter, Abschlüsse, nächste Schritte.',
    module: 'crm',
    icon: 'TrendingUp',
    settings: TPL_SETTINGS,
    rows: [
      r({ id: bid('cover'), type: 'cover', title: 'Vertriebsbericht', subtitle: 'Periode', showDate: true }),
      rc(
        { blocks: [kpiBlock('Umsatz', '—', '€', null, 'crm')] },
        { blocks: [kpiBlock('Gewinnrate', '—', '%', null, 'crm')] },
        { blocks: [kpiBlock('Pipeline', '—', '€', null, 'crm')] },
      ),
      r({ id: bid('h'), type: 'heading', level: 2, text: 'Pipeline' }),
      r({ id: bid('c'), type: 'chart', caption: 'Pipeline nach Phase' }),
      r({ id: bid('tb'), type: 'table', caption: 'Abschlüsse der Periode' }),
      r({ id: bid('cl'), type: 'callout', variant: 'recommendation', title: 'Nächste Schritte', html: '<p>Top-3-Maßnahmen.</p>' }),
    ],
  },
  {
    id: 'tpl-bi',
    title: 'Analyse-Bericht',
    description: 'Kennzahlen-Übersicht, Zeitreihe, Vergleiche, Detaildaten.',
    module: 'cross',
    icon: 'BarChart3',
    settings: TPL_SETTINGS,
    rows: [
      r({ id: bid('cover'), type: 'cover', title: 'Analyse-Bericht', subtitle: 'Zeitraum', showDate: true }),
      rc(
        { blocks: [kpiBlock('Kennzahl A', '—', '', null, 'cross')] },
        { blocks: [kpiBlock('Kennzahl B', '—', '', null, 'cross')] },
        { blocks: [kpiBlock('Kennzahl C', '—', '', null, 'cross')] },
      ),
      r({ id: bid('c'), type: 'chart', caption: 'Entwicklung über Zeit' }),
      rc(
        { blocks: [{ id: bid('c'), type: 'chart', caption: 'Verteilung A' }] },
        { blocks: [{ id: bid('c'), type: 'chart', caption: 'Verteilung B' }] },
      ),
      r({ id: bid('tb'), type: 'table', caption: 'Detaildaten' }),
    ],
  },
  {
    id: 'tpl-projekt',
    title: 'Projektbericht',
    description: 'Status, Fortschritt, Budget, Meilensteine, Risiken.',
    module: 'work',
    icon: 'ClipboardList',
    settings: TPL_SETTINGS,
    rows: [
      r({ id: bid('cover'), type: 'cover', title: 'Projektbericht', subtitle: 'Projektname', showDate: true }),
      r({ id: bid('h'), type: 'heading', level: 1, text: 'Status & Fortschritt' }),
      rc(
        { blocks: [kpiBlock('Fortschritt', '—', '%', null, 'work')] },
        { blocks: [kpiBlock('Budget', '—', '€', null, 'work')] },
      ),
      r({ id: bid('t'), type: 'text', html: '<p>Wichtigste Meilensteine und ihr Stand.</p>' }),
      r({ id: bid('bl'), type: 'bullet', items: ['Risiko 1', 'Risiko 2'] }),
      r({ id: bid('cl'), type: 'callout', variant: 'recommendation', title: 'Maßnahmen', html: '<p>Empfohlene nächste Schritte.</p>' }),
    ],
  },
  {
    id: 'tpl-exec',
    title: 'Executive-Kurzbericht',
    description: 'Kernaussage, drei Kennzahlen, Top-Findings, Empfehlung.',
    module: 'cross',
    icon: 'Zap',
    settings: TPL_SETTINGS,
    rows: [
      r({ id: bid('cover'), type: 'cover', title: 'Executive Update', subtitle: 'Datum', showDate: true }),
      r({ id: bid('t'), type: 'text', html: '<p>Kernaussage zuerst: das Wichtigste in zwei Sätzen.</p>' }),
      rc(
        { blocks: [kpiBlock('Kennzahl 1', '—', '', null, 'cross')] },
        { blocks: [kpiBlock('Kennzahl 2', '—', '', null, 'cross')] },
        { blocks: [kpiBlock('Kennzahl 3', '—', '', null, 'cross')] },
      ),
      r({ id: bid('bl'), type: 'bullet', items: ['Finding 1', 'Finding 2', 'Finding 3'] }),
      r({ id: bid('cl'), type: 'callout', variant: 'recommendation', title: 'Empfehlung', html: '<p>Was als Nächstes zu tun ist.</p>' }),
    ],
  },
]

// ---------------------------------------------------------------------------
// Report results (per definition) — series for charts, columns+rows for tables
// ---------------------------------------------------------------------------

const MONTHS = ['Jan', 'Feb', 'Mär', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dez']

function monthlySeries(id: string, label: string, base: number, step: number, jitter: number): ReportSeries {
  return {
    id,
    label,
    data: MONTHS.map((m, i) => ({
      label: m,
      value: Math.round(base + step * i + Math.sin(i * 1.3) * jitter),
    })),
  }
}

function seriesToRows(series: ReportSeries, valueKey: string): Record<string, unknown>[] {
  return series.data.map((p) => ({ monat: p.label, [valueKey]: p.value }))
}

function metaFor(id: string, rowCount: number): ReportResult['meta'] {
  return { generated_at: new Date().toISOString(), row_count: rowCount, definition_id: id }
}

function buildResults(): Record<string, ReportResult> {
  const results: Record<string, ReportResult> = {}

  const umsatz = monthlySeries('umsatz', 'Umsatz (€)', 62000, 2100, 4800)
  results['def-umsatz'] = {
    columns: [
      { key: 'monat', label: 'Monat', type: 'string' },
      { key: 'wert', label: 'Umsatz', type: 'currency' },
    ],
    rows: seriesToRows(umsatz, 'wert'),
    series: [umsatz],
    totals: { wert: umsatz.data.reduce((s, p) => s + p.value, 0) },
    meta: metaFor('def-umsatz', 12),
  }

  const ticketsOpen = monthlySeries('open', 'Offen', 18, -0.4, 5)
  const ticketsSolved = monthlySeries('solved', 'Gelöst', 42, 1.1, 7)
  results['def-tickets'] = {
    columns: [
      { key: 'monat', label: 'Monat', type: 'string' },
      { key: 'offen', label: 'Offen', type: 'number' },
      { key: 'geloest', label: 'Gelöst', type: 'number' },
    ],
    rows: MONTHS.map((m, i) => ({ monat: m, offen: ticketsOpen.data[i].value, geloest: ticketsSolved.data[i].value })),
    series: [ticketsOpen, ticketsSolved],
    meta: metaFor('def-tickets', 12),
  }

  const lager = monthlySeries('lager', 'Lagerwert (€)', 392000, 2400, 9000)
  results['def-lager'] = {
    columns: [
      { key: 'monat', label: 'Monat', type: 'string' },
      { key: 'wert', label: 'Lagerwert', type: 'currency' },
    ],
    rows: seriesToRows(lager, 'wert'),
    series: [lager],
    totals: { wert: lager.data[lager.data.length - 1].value },
    meta: metaFor('def-lager', 12),
  }

  const leads = monthlySeries('conv', 'Conversion (%)', 26, 0.6, 3)
  results['def-leads'] = {
    columns: [
      { key: 'monat', label: 'Monat', type: 'string' },
      { key: 'rate', label: 'Conversion-Rate', type: 'percent' },
    ],
    rows: seriesToRows(leads, 'rate'),
    series: [leads],
    meta: metaFor('def-leads', 12),
  }

  // DATEV BWA — position / current / prior year
  const bwaCols: ReportColumn[] = [
    { key: 'position', label: 'Position', type: 'string' },
    { key: 'betrag', label: 'Aktuelles Jahr', type: 'currency' },
    { key: 'vorjahr', label: 'Vorjahr', type: 'currency' },
  ]
  const bwaRows: Record<string, unknown>[] = [
    { position: 'Umsatzerlöse', betrag: 984500, vorjahr: 902300 },
    { position: 'Materialaufwand', betrag: -312400, vorjahr: -298100 },
    { position: 'Rohertrag', betrag: 672100, vorjahr: 604200 },
    { position: 'Personalkosten', betrag: -348900, vorjahr: -321500 },
    { position: 'Raumkosten', betrag: -58200, vorjahr: -56800 },
    { position: 'Abschreibungen', betrag: -41300, vorjahr: -39900 },
    { position: 'Sonstige Kosten', betrag: -72600, vorjahr: -68400 },
    { position: 'Betriebsergebnis', betrag: 151100, vorjahr: 117600 },
  ]
  results['def-datev-bwa'] = {
    columns: bwaCols,
    rows: bwaRows,
    meta: metaFor('def-datev-bwa', bwaRows.length),
  }

  // DATEV SuSa — account / name / debit / credit / balance
  const susaCols: ReportColumn[] = [
    { key: 'konto', label: 'Konto', type: 'string' },
    { key: 'bezeichnung', label: 'Bezeichnung', type: 'string' },
    { key: 'soll', label: 'Soll', type: 'currency' },
    { key: 'haben', label: 'Haben', type: 'currency' },
    { key: 'saldo', label: 'Saldo', type: 'currency' },
  ]
  const susaRows: Record<string, unknown>[] = [
    { konto: '1200', bezeichnung: 'Bank', soll: 248300, haben: 191200, saldo: 57100 },
    { konto: '1400', bezeichnung: 'Forderungen aL+L', soll: 132900, haben: 98400, saldo: 34500 },
    { konto: '1600', bezeichnung: 'Verbindlichkeiten aL+L', soll: 72100, haben: 114800, saldo: -42700 },
    { konto: '1576', bezeichnung: 'Vorsteuer 19 %', soll: 59400, haben: 0, saldo: 59400 },
    { konto: '1776', bezeichnung: 'Umsatzsteuer 19 %', soll: 0, haben: 187100, saldo: -187100 },
    { konto: '4000', bezeichnung: 'Umsatzerlöse', soll: 0, haben: 984500, saldo: -984500 },
    { konto: '6000', bezeichnung: 'Personalkosten', soll: 348900, haben: 0, saldo: 348900 },
  ]
  results['def-datev-susa'] = {
    columns: susaCols,
    rows: susaRows,
    meta: metaFor('def-datev-susa', susaRows.length),
  }

  return results
}

const REPORT_RESULTS = buildResults()

// ---------------------------------------------------------------------------
// Export helpers (real Blob downloads — CSV text + one-page PDF)
// ---------------------------------------------------------------------------

function resultToCsv(result: ReportResult): string {
  const header = result.columns.map((c) => c.label).join(';')
  const lines = result.rows.map((row) =>
    result.columns
      .map((c) => {
        const v = row[c.key]
        return v === null || v === undefined ? '' : String(v)
      })
      .join(';'),
  )
  return [header, ...lines].join('\r\n')
}

function resultToPdf(definition: ReportDefinition, result: ReportResult): Uint8Array {
  const lines: PdfLine[] = [
    { text: 'Zentria GmbH · Cosmi', size: 9 },
    { text: definition.name, size: 20, gap: 6 },
    { text: definition.description, size: 9, gap: 8 },
  ]
  // Header row
  lines.push({ text: result.columns.map((c) => c.label).join('   '), size: 10, gap: 2 })
  for (const row of result.rows.slice(0, 30)) {
    lines.push({
      text: result.columns.map((c) => String(row[c.key] ?? '—')).join('   '),
      size: 9,
    })
  }
  lines.push({ text: '', gap: 8 })
  lines.push({ text: 'Demo-Bericht (Cosmi) — finale BI-/DATEV-Anbindung folgt.', size: 8 })
  return buildSimplePdf(lines)
}

const CONTENT_TYPES: Record<string, string> = {
  pdf: 'application/pdf',
  csv: 'text/csv;charset=utf-8',
  xlsx: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
}

function slugify(name: string): string {
  return name
    .replace(/ä/g, 'ae').replace(/ö/g, 'oe').replace(/ü/g, 'ue').replace(/ß/g, 'ss')
    .replace(/[^a-zA-Z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
}

// ---------------------------------------------------------------------------
// Schedules — stateful, session-persistent (in-memory, reset on reload)
// ---------------------------------------------------------------------------

function hoursAgo(h: number): string {
  return new Date(Date.now() - h * 3600_000).toISOString()
}

let SCHEDULES: ReportSchedule[] = [
  {
    id: 'sch-1', tenant_id: TENANT, definition_id: 'def-umsatz',
    name: 'Wöchentlicher Umsatzbericht', cron_expression: '0 8 * * MON',
    recipients: ['finanzen@zentria.tech'], format: 'pdf', params: {},
    active: true, last_run_at: hoursAgo(20), last_run_status: 'success', last_run_error: null,
    created_by: null, created_at: hoursAgo(720), updated_at: hoursAgo(20),
  },
  {
    id: 'sch-2', tenant_id: TENANT, definition_id: 'def-datev-bwa',
    name: 'Monatliche BWA', cron_expression: '0 6 1 * *',
    recipients: ['buchhaltung@zentria.tech', 'gf@zentria.tech'], format: 'xlsx', params: {},
    active: true, last_run_at: hoursAgo(54), last_run_status: 'success', last_run_error: null,
    created_by: null, created_at: hoursAgo(2160), updated_at: hoursAgo(54),
  },
  {
    id: 'sch-3', tenant_id: TENANT, definition_id: 'def-tickets',
    name: 'Täglicher Ticket-Report', cron_expression: '0 18 * * *',
    recipients: ['support@zentria.tech'], format: 'csv', params: {},
    active: false, last_run_at: hoursAgo(96), last_run_status: 'skipped', last_run_error: null,
    created_by: null, created_at: hoursAgo(480), updated_at: hoursAgo(96),
  },
]

let scheduleCounter = SCHEDULES.length

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const berichteHandlers = [
  // --- Dashboard KPIs ---
  http.get(`${API}/api/v1/berichte/kpis`, ({ request }) => {
    const url = new URL(request.url)
    const modules = url.searchParams.getAll('modules').filter(Boolean)
    const kpis =
      modules.length > 0
        ? DEMO_KPIS.filter((k) => modules.includes(String(k.module_id)))
        : DEMO_KPIS
    return HttpResponse.json({ kpis, generated_at: new Date().toISOString() })
  }),

  // --- Definitions ---
  http.get(`${API}/api/v1/berichte/definitions`, ({ request }) => {
    const url = new URL(request.url)
    const module = url.searchParams.get('module')
    const kind = url.searchParams.get('kind')
    const isPublished = url.searchParams.get('is_published')
    let defs = DEMO_DEFINITIONS
    if (module) defs = defs.filter((d) => d.module === module)
    if (kind) defs = defs.filter((d) => d.kind === kind)
    if (isPublished != null) defs = defs.filter((d) => d.is_published === (isPublished === 'true'))
    return HttpResponse.json({ definitions: defs, total: defs.length })
  }),

  http.get(`${API}/api/v1/berichte/definitions/:id`, ({ params }) => {
    const definition = DEMO_DEFINITIONS.find((d) => d.id === params.id)
    if (!definition) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json({ definition })
  }),

  // --- Create custom definition (from the builder) ---
  http.post(`${API}/api/v1/berichte/definitions`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as Record<string, unknown>
    definitionCounter += 1
    const now = new Date().toISOString()
    const definition: ReportDefinition = {
      id: `def-custom-${definitionCounter}`,
      tenant_id: TENANT,
      name: String(body.name ?? 'Neuer Bericht'),
      description: String(body.description ?? ''),
      module: (body.module as ReportDefinition['module']) ?? 'cross',
      kind: 'custom',
      query_config: (body.query_config as ReportDefinition['query_config']) ?? {},
      default_format: (body.default_format as ReportDefinition['default_format']) ?? 'pdf',
      created_by: 'u-demo',
      is_published: body.is_published != null ? Boolean(body.is_published) : false,
      created_at: now,
      updated_at: now,
    }
    DEMO_DEFINITIONS = [definition, ...DEMO_DEFINITIONS]
    return HttpResponse.json({ definition }, { status: 201 })
  }),

  // --- Update definition (rename, edit query, pin to dashboard) ---
  http.patch(`${API}/api/v1/berichte/definitions/:id`, async ({ params, request }) => {
    const body = (await request.json().catch(() => ({}))) as Record<string, unknown>
    const idx = DEMO_DEFINITIONS.findIndex((d) => d.id === params.id)
    if (idx === -1) return new HttpResponse(null, { status: 404 })
    DEMO_DEFINITIONS[idx] = {
      ...DEMO_DEFINITIONS[idx],
      ...body,
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json({ definition: DEMO_DEFINITIONS[idx] })
  }),

  // --- Delete definition ---
  http.delete(`${API}/api/v1/berichte/definitions/:id`, ({ params }) => {
    DEMO_DEFINITIONS = DEMO_DEFINITIONS.filter((d) => d.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // --- Report Templates ---
  http.get(`${API}/api/v1/berichte/templates`, () =>
    HttpResponse.json({ templates: DEMO_TEMPLATES }),
  ),

  // --- Report Documents (multi-page authoring) ---
  http.get(`${API}/api/v1/berichte/documents`, ({ request }) => {
    const url = new URL(request.url)
    const module = url.searchParams.get('module')
    const status = url.searchParams.get('status')
    const search = url.searchParams.get('search')?.toLowerCase()
    let docs = DEMO_DOCUMENTS
    if (module) docs = docs.filter((d) => d.module === module)
    if (status) docs = docs.filter((d) => d.status === status)
    if (search) docs = docs.filter((d) => d.title.toLowerCase().includes(search))
    const documents = [...docs].sort((a, b) => b.updated_at.localeCompare(a.updated_at))
    return HttpResponse.json({ documents, total: documents.length })
  }),

  http.get(`${API}/api/v1/berichte/documents/:id`, ({ params }) => {
    const document = DEMO_DOCUMENTS.find((d) => d.id === params.id)
    if (!document) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json({ document })
  }),

  http.post(`${API}/api/v1/berichte/documents`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as Record<string, unknown>
    documentCounter += 1
    const now = new Date().toISOString()
    const template = body.template_id
      ? DEMO_TEMPLATES.find((t) => t.id === body.template_id)
      : undefined
    const title = String(body.title ?? template?.title ?? 'Neuer Bericht')
    const document: ReportDocument = {
      id: `doc-custom-${documentCounter}`,
      tenant_id: TENANT,
      title,
      description: body.description ? String(body.description) : template?.description,
      module: (body.module as ReportDocument['module']) ?? template?.module ?? 'cross',
      status: 'draft',
      rows:
        (body.rows as ReportDocument['rows']) ??
        (template
          ? (JSON.parse(JSON.stringify(template.rows)) as ReportDocument['rows'])
          : [
              {
                id: 'row-1',
                columns: [
                  {
                    id: 'col-1',
                    blocks: [{ id: 'cover-1', type: 'cover', title, showDate: true }],
                  },
                ],
              },
            ]),
      settings:
        (body.settings as ReportDocument['settings']) ??
        template?.settings ??
        { showHeader: true, showFooter: true, showPageNumbers: true },
      template_id: (body.template_id as string | null) ?? null,
      created_by: 'u-demo',
      created_at: now,
      updated_at: now,
      released_at: null,
    }
    DEMO_DOCUMENTS = [document, ...DEMO_DOCUMENTS]
    return HttpResponse.json({ document }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/berichte/documents/:id`, async ({ params, request }) => {
    const body = (await request.json().catch(() => ({}))) as Record<string, unknown>
    const idx = DEMO_DOCUMENTS.findIndex((d) => d.id === params.id)
    if (idx === -1) return new HttpResponse(null, { status: 404 })
    const prev = DEMO_DOCUMENTS[idx]
    const next = { ...prev, ...body, updated_at: new Date().toISOString() } as ReportDocument
    // Stamp released_at on first release.
    if (body.status === 'released' && !prev.released_at) {
      next.released_at = new Date().toISOString()
    }
    DEMO_DOCUMENTS[idx] = next
    return HttpResponse.json({ document: next })
  }),

  http.delete(`${API}/api/v1/berichte/documents/:id`, ({ params }) => {
    DEMO_DOCUMENTS = DEMO_DOCUMENTS.filter((d) => d.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  // --- Run ---
  http.post(`${API}/api/v1/berichte/definitions/:id/run`, ({ params }) => {
    const id = String(params.id)
    // System defs use canned results; custom builder defs run through the executor
    // (same fallback as export) so newly built charts render in documents.
    let result = REPORT_RESULTS[id]
    if (!result) {
      const definition = DEMO_DEFINITIONS.find((d) => d.id === id)
      if (definition && isBuilderQuery(definition.query_config)) {
        const source = resolveSource(definition.query_config.sourceId)
        if (source) result = executeQuery(source, definition.query_config, source.sampleRows())
      }
    }
    if (!result) return new HttpResponse(null, { status: 404 })
    const now = new Date().toISOString()
    return HttpResponse.json({
      result: { ...result, meta: { ...result.meta, generated_at: now } },
      run: {
        id: `run-${id}-${Date.now()}`,
        tenant_id: TENANT,
        definition_id: id,
        schedule_id: null,
        trigger: 'manual',
        params: {},
        duration_ms: 38,
        row_count: result.rows.length,
        status: 'success',
        error: null,
        started_at: now,
        completed_at: now,
      },
    })
  }),

  // --- Export (real Blob) ---
  http.post(`${API}/api/v1/berichte/definitions/:id/export`, ({ params, request }) => {
    const id = String(params.id)
    const definition = DEMO_DEFINITIONS.find((d) => d.id === id)
    if (!definition) return new HttpResponse(null, { status: 404 })
    // System defs use canned results; custom builder defs run through the executor.
    let result = REPORT_RESULTS[id]
    if (!result && isBuilderQuery(definition.query_config)) {
      const source = resolveSource(definition.query_config.sourceId)
      if (source) result = executeQuery(source, definition.query_config, source.sampleRows())
    }
    if (!result) return new HttpResponse(null, { status: 404 })
    const url = new URL(request.url)
    const format = (url.searchParams.get('format') ?? 'pdf') as 'pdf' | 'csv' | 'xlsx'
    const filename = `${slugify(definition.name)}.${format}`
    const body: BodyInit =
      format === 'pdf' ? (resultToPdf(definition, result) as BodyInit) : resultToCsv(result)
    return new HttpResponse(body, {
      headers: {
        'Content-Type': CONTENT_TYPES[format] ?? 'application/octet-stream',
        'Content-Disposition': `attachment; filename="${filename}"`,
      },
    })
  }),

  // --- Cache invalidation (no-op demo) ---
  http.delete(`${API}/api/v1/berichte/definitions/:id/cache`, () =>
    HttpResponse.json({ evicted: 0 }),
  ),

  // --- Builder preview (query executor stub; real executor = Luke) ---
  http.post(`${API}/api/v1/berichte/preview`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { query?: BuilderQueryConfig }
    const query = body.query
    if (!query || query.kind !== 'builder') {
      return HttpResponse.json({ error: 'Invalid builder query' }, { status: 400 })
    }
    const source = resolveSource(query.sourceId)
    if (!source) {
      return HttpResponse.json({ error: `Unknown source: ${query.sourceId}` }, { status: 404 })
    }
    const result = executeQuery(source, query, source.sampleRows())
    return HttpResponse.json({ result })
  }),

  // --- Schedules ---
  http.get(`${API}/api/v1/berichte/schedules`, ({ request }) => {
    const url = new URL(request.url)
    const definitionId = url.searchParams.get('definition_id')
    const active = url.searchParams.get('active')
    let list = SCHEDULES
    if (definitionId) list = list.filter((s) => s.definition_id === definitionId)
    if (active != null) list = list.filter((s) => s.active === (active === 'true'))
    return HttpResponse.json({ schedules: list, total: list.length })
  }),

  http.post(`${API}/api/v1/berichte/schedules`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const now = new Date().toISOString()
    scheduleCounter += 1
    const schedule: ReportSchedule = {
      id: `sch-${scheduleCounter}`,
      tenant_id: TENANT,
      definition_id: String(body.definition_id ?? ''),
      name: String(body.name ?? 'Neuer Bericht'),
      cron_expression: String(body.cron_expression ?? '0 8 * * MON'),
      recipients: Array.isArray(body.recipients) ? (body.recipients as string[]) : [],
      format: (body.format as ReportSchedule['format']) ?? 'pdf',
      params: (body.params as Record<string, unknown>) ?? {},
      active: body.active != null ? Boolean(body.active) : true,
      last_run_at: null,
      last_run_status: null,
      last_run_error: null,
      created_by: null,
      created_at: now,
      updated_at: now,
    }
    SCHEDULES = [schedule, ...SCHEDULES]
    return HttpResponse.json({ schedule }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/berichte/schedules/:id`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const idx = SCHEDULES.findIndex((s) => s.id === params.id)
    if (idx === -1) return new HttpResponse(null, { status: 404 })
    SCHEDULES[idx] = { ...SCHEDULES[idx], ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json({ schedule: SCHEDULES[idx] })
  }),

  http.delete(`${API}/api/v1/berichte/schedules/:id`, ({ params }) => {
    SCHEDULES = SCHEDULES.filter((s) => s.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),

  http.post(`${API}/api/v1/berichte/schedules/:id/toggle`, async ({ params, request }) => {
    const body = (await request.json().catch(() => ({}))) as { active?: boolean }
    const idx = SCHEDULES.findIndex((s) => s.id === params.id)
    if (idx === -1) return new HttpResponse(null, { status: 404 })
    SCHEDULES[idx] = {
      ...SCHEDULES[idx],
      active: body.active != null ? Boolean(body.active) : !SCHEDULES[idx].active,
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json({ schedule: SCHEDULES[idx] })
  }),
]
