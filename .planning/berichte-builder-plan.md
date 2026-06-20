# Report-Builder (berichte „Erstellen") — Voller Bau-Plan

> **Ziel (Darien 2026-06-20):** Den flachen „Erstellen"-Tab zu einem vollwertigen Self-Service-Report-Builder machen — alle Viz-Typen, pro-Bericht-Anpassung, modulweite Settings-Defaults, eigene Berichte speichern + Bibliothek + Dashboard-Pin. Modul soll **fertig** sein. FE-mock-first; echter Query-Executor = Luke 🔒.
> Markt-Recherche: Metabase · Looker Studio · HubSpot Custom Report Builder. Synthese unten eingearbeitet.

## Leitprinzipien (aus Markt-Recherche)
1. **Live-Preview ohne Apply** — jede Feld-/Filter-Änderung aktualisiert die Vorschau sofort.
2. **Slot-Constraints** — Dimensions nur als Gruppierung/X-Achse, Measures nur als Wert/Y-Achse. Verhindert Fehler ohne Erklärtext.
3. **Auto-Viz + Override** — Datenkombination schlägt Chart-Typ vor (1 Dim + 1 Measure → Balken; Zeit + Measure → Linie; 1 Measure solo → KPI), Galerie immer wechselbar.
4. **Relative Datums-Quick-Picks** — „Letzte 7/30/90 Tage", „Dieser Monat", „Letztes Quartal" als 1-Klick + freie Eingabe.
5. **Aggregations-Transparenz** — „Summe von Umsatz" als sichtbare Beschriftung.
6. **Setup/Style getrennt** (Looker) — Setup = *was* (Felder/Filter/Aggregation), Style = *wie* (Farbe/Achsen/Labels).

## Architektur (erweiterbar + Sub-Terminal-sicher)

### Datenquellen-Registry — `modules/berichte/report-sources/`  (analog `module-settings-registry`)
```
report-sources/
├─ types.ts            ReportSource, FieldDefinition, FieldRole, FieldDataType
├─ registry.ts         REPORT_SOURCES[] + resolveSource(id)   ← EINZIGER gemeinsamer Merge-Punkt
├─ finanzen.source.ts  ┐  je 1 Datei pro Modul — disjunkt.
├─ kontakte.source.ts  │  Neues Modul fertig → neue *.source.ts + 1 Zeile in registry.ts.
├─ work.source.ts      │  Sub-Terminal kann hier konfliktfrei erweitern.
├─ helpdesk.source.ts  │
└─ kommunikation.source.ts ┘
```
Jede Source: `{ id, module, labelKey, icon, fields: FieldDefinition[], sampleRows() }`.
`FieldDefinition = { key, labelKey, dataType: number|string|date|enum|boolean, role: dimension|measure, enumValues?, format? }`.
Reale Felder pro Modul liegen in der Ist-Recherche vor (finanzen: total_net/gross/issue_date/status/customer/category; crm: deal.value/stageName/probability/industry; work: project.progress/task counts/status/priority; helpdesk: sla mins/status/priority; kommunikation: channel/is_read/assigned_to).

### Query-Schema (typisiert) — erweitert `api/berichte-types.ts`
```ts
type VisualizationType = 'table'|'bar'|'line'|'area'|'donut'|'kpi'|'combo'|'gauge'
type AggregationFn = 'count'|'sum'|'avg'|'min'|'max'
type FilterOperator = 'eq'|'neq'|'gt'|'lt'|'between'|'contains'|'startsWith'|'isEmpty'|'in'|'notIn'|'before'|'after'|'inLastDays'
interface ReportFilter { field: string; operator: FilterOperator; value?: unknown; value2?: unknown }
interface BuilderQueryConfig {                 // wird in query_config (JSONB) gespeichert
  sourceId: string
  viz: VisualizationType
  dimensions: string[]                          // group-by, max 2
  measures: { field: string; agg: AggregationFn }[]
  filters: ReportFilter[]                       // AND-verknüpft
  dateRange?: { field: string; preset?: string; from?: string; to?: string }
  sort?: { field: string; dir: 'asc'|'desc' }
  limit?: number                                // Top-N
  options?: ReportViewOptions                   // Style-Tab (pro Bericht)
}
interface ReportViewOptions { palette?, showLegend?, legendPos?, showDataLabels?, axisXTitle?, axisYTitle?, stacked?, numberFormat? }
```
`ReportModule` erweitern: + `work`, `kommunikation` (behalte bestehende).

### Viz-Layer — `modules/berichte/components/charts/`  (recharts-Wrapper, nutzen useChartTheme + usePrefersReducedMotion)
`TableWidget · BarChartWidget · LineChartWidget · AreaChartWidget · DonutChartWidget · KpiWidget · ComboChartWidget · GaugeWidget`
+ `ChartRenderer` (dispatch nach VisualizationType) — nimmt ReportResult + ReportViewOptions.

### Builder-UI — `modules/berichte/components/builder/`  (ersetzt ReportBuilder.tsx)
`ReportBuilderShell` (links Config / rechts Live-Preview) · `SourcePicker` · `FieldPicker` (Dim/Measure getrennt, Typ-Icons, Slot-Constraint) · `VizSwitcher` (Auto-Select + Galerie) · `FilterBuilder` (typ-aware Operatoren, relative Datums, Chips) · `SummarizeBlock` (Aggregation + Group-by) · `ReportStylePanel` (Style-Tab) · `LivePreview` · `SaveReportBar`.

### Bibliothek — `modules/berichte/components/library/`
`MyReportsLibrary` (custom Definitions, Modul-Badge-Filter, Suche, SortMenu, Liste/Kachel) · öffnen → Builder-State restore · Duplizieren/Umbenennen/Löschen · „Zum Dashboard hinzufügen".

### MSW — `mocks/handlers/berichte.ts` erweitern
`POST /definitions` (create custom) · `PUT/PATCH /:id` · `DELETE /:id` · **`POST /preview`** (nimmt BuilderQueryConfig → generiert ReportResult aus source.sampleRows + wendet filter/aggregation/sort/limit an = Query-Executor-Stub). Custom Defs session-persistent.

### Settings — `BerichteSettingsPanel` erweitern
**Für-alle (tenant):** Standard-Palette · Zahlenformat de-DE (`1.234,56 €`) · Standard-Zeitraum · Standard-Granularität · Default-Export-Format. **Persönlich:** Standard-Viz-Präferenz · Standard-Ansicht Bibliothek (Liste/Kachel).

## Phasen
| # | Phase | Inhalt | DoD |
|---|-------|--------|-----|
| **E-0** | Fundament | Query-Schema-Typen + ReportModule-Erweiterung + report-sources Registry (types+registry+5 sources) | Registry importierbar, Felder typisiert |
| **E-1** | Viz + Picker + Preview | 8 Chart-Wrapper + ChartRenderer + SourcePicker + FieldPicker (Slot-Constraint) + VizSwitcher (Auto-Select) + LivePreview + MSW `/preview` | Modul→Felder→Viz→Live-Vorschau funktioniert end-to-end |
| **E-2** | Filter-Builder | typ-aware Operatoren + relative Datums-Quick-Picks + AND + Filter-Chips, Vorschau reagiert | Filter ändert Vorschau live |
| **E-3** | Aggregation/Grouping | Summarize-Block, 5 Aggregationen, max 2 Group-by, Aggregations-Label sichtbar | Sum/Avg/Count etc. greifen in Vorschau |
| **E-4** | Speichern + Bibliothek | SaveReportBar + MyReportsLibrary + CRUD-MSW + State-Restore + Dashboard-Pin | Bericht speichern→Bibliothek→öffnen→State zurück |
| **E-5** | Style-Optionen + Settings + Polish | ReportStylePanel (Palette/Achsen/Labels/Legende/Sort/Top-N) + modulweite Settings-Defaults + i18n ×4 + Screenshot-QA | Alle Optionen greifen, 0 Raw-Keys, QA grün |

## Verifikation pro Phase
gescopter `tsconfig.eNcheck.json` (nur geänderte Dateien) → `node_modules/.bin/tsc --noEmit -p ...` · Playwright-QA `scripts/qa-berichte-builder.mjs` gegen `npm run dev` :5173 · Screenshots ansehen (Raw-Keys/Layout/leere Zustände) · ein Commit/Push pro Phase.

## Wiederverwenden (nicht neu bauen)
`useChartTheme()` · `categoricalPalette()` · `usePrefersReducedMotion()` · `KPICard` · `shared/DetailModal` · `shared/SortMenu` · `ModuleSettingsShell` · TanStack-Hooks in `useBerichte.ts`.
