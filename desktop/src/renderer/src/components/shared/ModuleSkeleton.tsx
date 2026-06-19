import { Skeleton } from './Skeleton'

/**
 * ModuleSkeleton — modul-individuelles Lade-Gerüst (statt Spinner).
 *
 * Statt eines generischen Platzhalters spiegelt jedes Modul über ein
 * deklaratives Preset SEINE echte Top-Level-Struktur wider: Header (mit/ohne
 * farbigem Icon), Tab-Leiste, KPI-Reihe (boxed/plain), Toolbar und den primären
 * Inhaltstyp (Tabelle/Karten/Liste/Dashboard/Kalender/Split/Board/Detail).
 *
 * So „springt" das Layout beim Wechsel vom Gerüst zum geladenen Modul nicht mehr.
 * Das Preset wird per `lazyRoute(Component, '<key>')` in App.tsx gesetzt; der
 * Suspense-Fallback rendert dann das passende Gerüst. Liegt im Haupt-Bundle →
 * erscheint sofort, nicht erst nach dem Lazy-Load des Moduls.
 *
 * Neue Module: einen Eintrag in PRESETS ergänzen (Struktur am echten
 * Seiten-Layout ablesen) und den Key im Route-Eintrag setzen.
 */

type ContentType =
  | 'table'
  | 'cards'
  | 'list'
  | 'dashboard'
  | 'chart'
  | 'calendar'
  | 'board'
  | 'detail'
  | 'split'
  | 'sidebarList'

interface SkelConfig {
  /** PageHeader (Icon-Quadrat + Titel + Untertitel). Default false. */
  header?: boolean
  /** Primär-Action-Button(s) oben rechts. Default true wenn header. */
  headerAction?: boolean
  /** Tab-/Sub-Nav-Leiste unter dem Header. Anzahl Tabs. */
  tabs?: number
  /** KPI-Reihe. Anzahl Kennzahlen. */
  kpis?: number
  /** KPI-Stil: umrandete Karten vs. freistehende Zahl+Label. Default 'boxed'. */
  kpiStyle?: 'boxed' | 'plain'
  /** Spaltenzahl der KPI-Karten. Default min(count, 4). */
  kpiCols?: number
  /** KPI-Reihe VOR der Tab-Leiste rendern (z.B. team-Abteilungskarten). */
  kpisBeforeTabs?: boolean
  /** Toolbar-Zeile: false | 'search' | 'searchFilters'. */
  toolbar?: false | 'search' | 'searchFilters'
  /** Toolbar zusätzlich mit Ansichts-Umschalter rechts. */
  viewToggle?: boolean
  /** Primärer Inhaltstyp. */
  content: ContentType
  /** Tabellen-Spalten ODER Karten-Grid-Spalten. */
  columns?: number
  /** Inhalt zentriert + schmal (z.B. notifications max-w-3xl). */
  narrow?: boolean
}

/* ----------------------------- Primitives ------------------------------ */

function HeaderSkel({ icon = true, action = true }: { icon?: boolean; action?: boolean }) {
  return (
    <div className="mb-6 flex items-start justify-between gap-4">
      <div className="flex min-w-0 items-center gap-3">
        {icon && <Skeleton className="h-10 w-10 shrink-0 rounded-xl" />}
        <div className="space-y-2">
          <Skeleton className="h-7 w-48" />
          <Skeleton className="h-4 w-72" />
        </div>
      </div>
      {action && (
        <div className="flex shrink-0 items-center gap-2">
          <Skeleton className="h-9 w-36 rounded-xl" />
        </div>
      )}
    </div>
  )
}

/** Tab-Leiste: horizontale Label-Reihe mit Unterstrich (border-b). */
function TabBarSkel({ count }: { count: number }) {
  const widths = ['w-16', 'w-20', 'w-14', 'w-24', 'w-16', 'w-20', 'w-14', 'w-16', 'w-20', 'w-14']
  const shown = Math.min(count, 8)
  return (
    <div className="mb-6 flex items-center gap-5 border-b border-border">
      {Array.from({ length: shown }).map((_, i) => (
        <Skeleton key={i} className={`mb-2.5 h-3.5 ${widths[i % widths.length]}`} />
      ))}
    </div>
  )
}

/** KPI-Reihe als umrandete Karten (Icon-Quadrat + Label + Wert). */
function KpiBoxedSkel({ count, cols: colsProp }: { count: number; cols?: number }) {
  const cols = colsProp ?? Math.min(count, 4)
  return (
    <div
      className="mb-6 grid gap-4"
      style={{ gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))` }}
    >
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-3">
            <Skeleton className="h-10 w-10 shrink-0 rounded-xl" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-5 w-12" />
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}

/** KPI-Reihe freistehend (Zahl + Label, kein Rahmen) — wie eine StatsBar. */
function KpiPlainSkel({ count }: { count: number }) {
  return (
    <div className="mb-6 flex flex-wrap gap-x-10 gap-y-4 border-b border-border pb-5">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="space-y-2">
          <Skeleton className="h-3 w-20" />
          <Skeleton className="h-6 w-16" />
        </div>
      ))}
    </div>
  )
}

function ToolbarSkel({
  kind,
  viewToggle = false,
}: {
  kind: 'search' | 'searchFilters'
  viewToggle?: boolean
}) {
  return (
    <div className="mb-4 flex flex-wrap items-center gap-3">
      <Skeleton className="h-9 w-64 rounded-lg" />
      {kind === 'searchFilters' && (
        <>
          <Skeleton className="h-9 w-32 rounded-lg" />
          <Skeleton className="h-9 w-32 rounded-lg" />
        </>
      )}
      {viewToggle && <Skeleton className="ml-auto h-9 w-20 rounded-lg" />}
    </div>
  )
}

/* --------------------------- Content bodies ---------------------------- */

function TableBody({ columns = 6 }: { columns?: number }) {
  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <div className="flex items-center gap-4 border-b border-border bg-card px-4 py-3">
        {Array.from({ length: columns }).map((_, i) => (
          <Skeleton key={i} className={`h-3 ${i === 0 ? 'flex-[1.5]' : 'flex-1'}`} />
        ))}
      </div>
      <div className="divide-y divide-border-muted">
        {Array.from({ length: 8 }).map((_, r) => (
          <div key={r} className="flex items-center gap-4 px-4 py-3.5">
            {Array.from({ length: columns }).map((_, c) => (
              <Skeleton key={c} className={`h-3.5 ${c === 0 ? 'flex-[1.5]' : 'flex-1'}`} />
            ))}
          </div>
        ))}
      </div>
    </div>
  )
}

function CardGrid({ columns = 3, count = 6 }: { columns?: number; count?: number }) {
  return (
    <div
      className="grid gap-4"
      style={{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }}
    >
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center gap-3">
            <Skeleton className="h-10 w-10 shrink-0 rounded-full" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-3.5 w-2/3" />
              <Skeleton className="h-3 w-1/2" />
            </div>
          </div>
          <div className="mt-4 space-y-2">
            <Skeleton className="h-3 w-full" />
            <Skeleton className="h-3 w-4/5" />
          </div>
        </div>
      ))}
    </div>
  )
}

/** Volle Breite, Listen-Zeilen (Avatar + zwei Zeilen + Meta rechts). */
function ListRows({ rows = 8 }: { rows?: number }) {
  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card">
      <div className="divide-y divide-border-muted">
        {Array.from({ length: rows }).map((_, i) => (
          <div key={i} className="flex items-center gap-3 px-4 py-3.5">
            <Skeleton className="h-9 w-9 shrink-0 rounded-full" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-3.5 w-1/3" />
              <Skeleton className="h-3 w-1/2" />
            </div>
            <Skeleton className="h-5 w-16 shrink-0 rounded-full" />
            <Skeleton className="hidden h-3 w-20 shrink-0 sm:block" />
          </div>
        ))}
      </div>
    </div>
  )
}

/** Ein großer Inhalts-/Chart-Block (z.B. finanzen-Statusbalken, infra-Status). */
function ChartBody() {
  return (
    <div className="rounded-lg border border-border bg-card p-5">
      <Skeleton className="mb-4 h-4 w-40" />
      <Skeleton className="h-56 w-full rounded-md" />
    </div>
  )
}

function DashboardBody() {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      {['h-28', 'h-28', 'h-28', 'h-64 lg:col-span-2', 'h-64', 'h-48', 'h-48 lg:col-span-2'].map(
        (h, i) => (
          <div key={i} className={`rounded-lg border border-border bg-card p-4 ${h}`}>
            <Skeleton className="mb-3 h-4 w-1/3" />
            <Skeleton className="h-[60%] w-full" />
          </div>
        ),
      )}
    </div>
  )
}

function BoardBody() {
  const cardsPerCol = [3, 2, 4, 2]
  return (
    <div className="flex gap-4 overflow-hidden">
      {cardsPerCol.map((n, col) => (
        <div key={col} className="flex w-72 shrink-0 flex-col gap-3">
          <div className="flex items-center justify-between px-1">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-4 w-6 rounded-full" />
          </div>
          {Array.from({ length: n }).map((_, i) => (
            <div key={i} className="space-y-2 rounded-lg border border-border bg-card p-3">
              <Skeleton className="h-3.5 w-3/4" />
              <Skeleton className="h-3 w-1/2" />
              <div className="flex items-center gap-2 pt-1">
                <Skeleton className="h-5 w-5 rounded-full" />
                <Skeleton className="h-3 w-16" />
              </div>
            </div>
          ))}
        </div>
      ))}
    </div>
  )
}

function CalendarBody() {
  return (
    <>
      <div className="mb-4 flex items-center gap-3">
        <Skeleton className="h-8 w-8 rounded-md" />
        <Skeleton className="h-8 w-8 rounded-md" />
        <Skeleton className="h-6 w-40" />
        <div className="ml-auto flex gap-2">
          <Skeleton className="h-8 w-20 rounded-md" />
          <Skeleton className="h-8 w-20 rounded-md" />
        </div>
      </div>
      <div className="overflow-hidden rounded-lg border border-border">
        <div className="grid grid-cols-7 border-b border-border bg-secondary/40">
          {Array.from({ length: 7 }).map((_, i) => (
            <div key={i} className="px-2 py-2.5">
              <Skeleton className="mx-auto h-3 w-10" />
            </div>
          ))}
        </div>
        <div className="grid grid-cols-7">
          {Array.from({ length: 7 * 5 }).map((_, i) => (
            <div key={i} className="min-h-[84px] border-b border-r border-border-muted p-2">
              <Skeleton className="mb-2 h-3 w-5" />
              {i % 3 === 0 && <Skeleton className="mb-1 h-4 w-full rounded" />}
              {i % 4 === 0 && <Skeleton className="h-4 w-2/3 rounded" />}
            </div>
          ))}
        </div>
      </div>
    </>
  )
}

function DetailBody() {
  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div className="flex items-center gap-4">
        <Skeleton className="h-16 w-16 rounded-full" />
        <div className="space-y-2">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-32" />
        </div>
      </div>
      <div className="flex gap-8 border-b border-border pb-5">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="space-y-2">
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-5 w-12" />
          </div>
        ))}
      </div>
      {Array.from({ length: 3 }).map((_, s) => (
        <div key={s} className="space-y-3 rounded-lg border border-border bg-card p-5">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-3.5 w-full" />
          <Skeleton className="h-3.5 w-5/6" />
          <Skeleton className="h-3.5 w-2/3" />
        </div>
      ))}
    </div>
  )
}

/** Schmale Navigations-/Ordner-Spalte (Suche + Einträge). */
function SidebarCol({ width = 'w-56' }: { width?: string }) {
  return (
    <div className={`${width} hidden shrink-0 space-y-1.5 border-r border-border pr-4 md:block`}>
      <Skeleton className="mb-3 h-9 w-full rounded-lg" />
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="flex items-center gap-2.5 rounded-md p-2">
          <Skeleton className="h-4 w-4 shrink-0 rounded" />
          <Skeleton className={`h-3 ${i % 2 ? 'w-2/3' : 'w-4/5'}`} />
        </div>
      ))}
    </div>
  )
}

/** Split-Layout: schmale Listen-Spalte + Detail-Fläche (mails/komm/wiki). */
function SplitBody() {
  return (
    <div className="flex h-full gap-4 overflow-hidden">
      <div className="w-72 shrink-0 space-y-2 rounded-lg border border-border bg-card p-3">
        <Skeleton className="mb-2 h-8 w-full rounded-md" />
        {Array.from({ length: 9 }).map((_, i) => (
          <div key={i} className="flex items-center gap-2.5 rounded-md p-2">
            <Skeleton className="h-8 w-8 shrink-0 rounded-full" />
            <div className="flex-1 space-y-1.5">
              <Skeleton className="h-3 w-2/3" />
              <Skeleton className="h-2.5 w-full" />
            </div>
          </div>
        ))}
      </div>
      <div className="flex-1 space-y-4 rounded-lg border border-border bg-card p-5">
        <div className="flex items-center gap-3 border-b border-border pb-4">
          <Skeleton className="h-10 w-10 rounded-full" />
          <div className="space-y-2">
            <Skeleton className="h-4 w-40" />
            <Skeleton className="h-3 w-24" />
          </div>
        </div>
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className={`flex ${i % 2 ? 'justify-end' : ''}`}>
            <Skeleton className={`h-12 ${i % 2 ? 'w-1/2' : 'w-2/3'} rounded-lg`} />
          </div>
        ))}
      </div>
    </div>
  )
}

/* -------------------------- Config renderer ---------------------------- */

function renderInnerContent(cfg: SkelConfig) {
  switch (cfg.content) {
    case 'table':
      return <TableBody columns={cfg.columns} />
    case 'cards':
      return <CardGrid columns={cfg.columns ?? 3} />
    case 'list':
      return <ListRows />
    case 'dashboard':
      return <DashboardBody />
    case 'chart':
      return <ChartBody />
    case 'calendar':
      return <CalendarBody />
    case 'board':
      return <BoardBody />
    case 'detail':
      return <DetailBody />
    default:
      return <TableBody columns={cfg.columns} />
  }
}

function ConfigBody({ cfg }: { cfg: SkelConfig }) {
  // Sidebar-Module: schmale Nav-Spalte links + (Toolbar + Inhalt) rechts, kein Header.
  if (cfg.content === 'split') {
    return <SplitBody />
  }
  if (cfg.content === 'sidebarList') {
    return (
      <div className="flex h-full gap-4 overflow-hidden">
        <SidebarCol />
        <div className="min-w-0 flex-1">
          {cfg.toolbar && <ToolbarSkel kind={cfg.toolbar} viewToggle={cfg.viewToggle} />}
          {cfg.columns && cfg.columns > 1 ? <CardGrid columns={cfg.columns} /> : <ListRows />}
        </div>
      </div>
    )
  }

  const kpis = cfg.kpis ? (
    cfg.kpiStyle === 'plain' ? (
      <KpiPlainSkel count={cfg.kpis} />
    ) : (
      <KpiBoxedSkel count={cfg.kpis} cols={cfg.kpiCols} />
    )
  ) : null

  const inner = (
    <>
      {cfg.header && <HeaderSkel action={cfg.headerAction ?? true} />}
      {cfg.kpisBeforeTabs && kpis}
      {cfg.tabs ? <TabBarSkel count={cfg.tabs} /> : null}
      {!cfg.kpisBeforeTabs && kpis}
      {cfg.toolbar && <ToolbarSkel kind={cfg.toolbar} viewToggle={cfg.viewToggle} />}
      {renderInnerContent(cfg)}
    </>
  )

  return cfg.narrow ? <div className="mx-auto max-w-3xl">{inner}</div> : inner
}

/* ------------------------------ Presets -------------------------------- */

const LIST_DEFAULT: SkelConfig = { header: true, content: 'table' }

const PRESETS = {
  // Generische Archetypen (Rückwärtskompatibilität + Fallbacks)
  list: LIST_DEFAULT,
  board: { header: true, content: 'board' },
  dashboard: { header: true, content: 'dashboard' },
  calendar: { header: true, toolbar: false, content: 'calendar' },
  split: { content: 'split' },
  detail: { content: 'detail' },

  // Modul-spezifisch — Struktur am echten Default-Tab abgelesen
  finanzen: { header: true, tabs: 8, kpis: 8, kpiCols: 4, content: 'chart' },
  helpdesk: { header: true, tabs: 3, toolbar: 'searchFilters', content: 'table', columns: 7 },
  team: {
    header: true,
    kpis: 6,
    kpiCols: 6,
    kpisBeforeTabs: true,
    tabs: 8,
    toolbar: 'search',
    viewToggle: true,
    content: 'cards',
    columns: 3,
  },
  vertraege: { header: true, tabs: 4, kpis: 4, toolbar: 'search', content: 'table', columns: 7 },
  formulare: { header: true, tabs: 3, kpis: 3, toolbar: 'search', content: 'cards', columns: 3 },
  inventar: { header: true, tabs: 4, toolbar: 'searchFilters', content: 'table', columns: 7 },
  einkauf: { header: true, tabs: 4, toolbar: 'searchFilters', content: 'table', columns: 6 },
  fuhrpark: { header: true, tabs: 5, toolbar: 'searchFilters', content: 'cards', columns: 3 },
  produktion: { header: true, tabs: 4, toolbar: 'searchFilters', content: 'table', columns: 6 },
  vermietung: { header: true, kpis: 4, tabs: 3, toolbar: 'search', content: 'cards', columns: 3 },
  rapporte: { header: true, kpis: 4, tabs: 3, toolbar: 'searchFilters', content: 'list' },
  zeiterfassung: { header: true, tabs: 5, toolbar: 'search', content: 'list' },
  infrastruktur: { header: true, tabs: 7, kpis: 4, content: 'chart' },
  adminhub: { header: true, tabs: 4, content: 'dashboard' },
  notifications: { header: true, headerAction: false, tabs: 2, content: 'list', narrow: true },
  video: { header: true, tabs: 3, content: 'cards', columns: 3 },
  meetings: { header: true, tabs: 4, toolbar: 'search', viewToggle: true, content: 'cards', columns: 3 },
  automatisierung: { kpis: 3, kpiStyle: 'plain', kpisBeforeTabs: true, tabs: 3, content: 'table', columns: 4 },
  dialer: { header: true, toolbar: 'searchFilters', content: 'cards', columns: 2 },

  // Sidebar-/Split-Module (kein Header)
  dokumente: { content: 'sidebarList', toolbar: 'search', viewToggle: true, columns: 3 },
  kontakte: { content: 'sidebarList', toolbar: 'search', viewToggle: true, columns: 1 },
  mails: { content: 'split' },
  kommunikation: { content: 'split' },
  wiki: { content: 'split' },

  // CRM-Unterseiten
  leads: { toolbar: 'searchFilters', content: 'list' },
  pipeline: { header: true, toolbar: 'search', content: 'table', columns: 7 },
  aktivitaeten: { header: true, tabs: 6, content: 'list' },
} satisfies Record<string, SkelConfig>

export type ModuleSkeletonVariant = keyof typeof PRESETS

export function ModuleSkeleton({ variant = 'list' }: { variant?: ModuleSkeletonVariant }) {
  const cfg = (PRESETS[variant] ?? LIST_DEFAULT) as SkelConfig
  return (
    <div className="h-full overflow-hidden p-6" aria-busy="true" aria-live="polite">
      <ConfigBody cfg={cfg} />
    </div>
  )
}
