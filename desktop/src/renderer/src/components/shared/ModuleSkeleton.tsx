import { Skeleton } from './Skeleton'

/**
 * ModuleSkeleton — modul-individuelles Lade-Gerüst (statt Spinner).
 *
 * Statt eines generischen Platzhalters spiegelt jedes Modul über einen
 * Layout-Archetyp seine echte Struktur wider. Die Variante wird im `lazyRoute`
 * (App.tsx) pro Modul gesetzt; der Suspense-Fallback rendert dann das passende
 * Gerüst. Liegt im Haupt-Bundle → erscheint sofort, nicht erst nach dem
 * Lazy-Load des Moduls.
 *
 * Archetypen:
 * - list      Header + KPIs + Toolbar + Tabelle/Karten (Default, ~Mehrheit)
 * - board     Kanban-Spalten mit Karten (work)
 * - dashboard Widget-Grid unterschiedlicher Größen (Übersicht, berichte)
 * - calendar  Wochenraster (kalender, schichten)
 * - split     schmale Listen-Spalte + Detail-Fläche (mails, kommunikation, wiki)
 * - detail    Detail-Seite: Kopf + Sektionen (Profil, Settings, :id-Seiten)
 */
export type ModuleSkeletonVariant =
  | 'list'
  | 'board'
  | 'dashboard'
  | 'calendar'
  | 'split'
  | 'detail'

function HeaderSkel({ action = true }: { action?: boolean }) {
  return (
    <div className="mb-6 flex items-start justify-between gap-4">
      <div className="space-y-2.5">
        <Skeleton className="h-7 w-52" />
        <Skeleton className="h-4 w-80" />
      </div>
      {action && <Skeleton className="h-9 w-36 shrink-0 rounded-lg" />}
    </div>
  )
}

function KpiRowSkel({ count = 4 }: { count?: number }) {
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

function ToolbarSkel() {
  return (
    <div className="mb-4 flex flex-wrap items-center gap-3">
      <Skeleton className="h-9 w-64 rounded-lg" />
      <Skeleton className="h-9 w-32 rounded-lg" />
      <Skeleton className="h-9 w-32 rounded-lg" />
    </div>
  )
}

function ListBody() {
  return (
    <>
      <KpiRowSkel />
      <ToolbarSkel />
      <div className="overflow-hidden rounded-lg border border-border">
        <div className="flex items-center gap-4 border-b border-border bg-secondary/40 px-4 py-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className={`h-3 ${i === 0 ? 'flex-[1.5]' : 'flex-1'}`} />
          ))}
        </div>
        <div className="divide-y divide-border-muted">
          {Array.from({ length: 8 }).map((_, r) => (
            <div key={r} className="flex items-center gap-4 px-4 py-3.5">
              {Array.from({ length: 5 }).map((_, c) => (
                <Skeleton key={c} className={`h-3.5 ${c === 0 ? 'flex-[1.5]' : 'flex-1'}`} />
              ))}
            </div>
          ))}
        </div>
      </div>
    </>
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

function DashboardBody() {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      {[
        'h-28', 'h-28', 'h-28',
        'h-64 lg:col-span-2', 'h-64',
        'h-48', 'h-48 lg:col-span-2',
      ].map((h, i) => (
        <div key={i} className={`rounded-lg border border-border bg-card p-4 ${h}`}>
          <Skeleton className="mb-3 h-4 w-1/3" />
          <Skeleton className="h-[60%] w-full" />
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

function SplitBody() {
  return (
    <div className="flex gap-4 overflow-hidden" style={{ height: 'calc(100% - 3rem)' }}>
      {/* Left list column */}
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
      {/* Main content pane */}
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

const BODIES: Record<ModuleSkeletonVariant, () => JSX.Element> = {
  list: ListBody,
  board: BoardBody,
  dashboard: DashboardBody,
  calendar: CalendarBody,
  split: SplitBody,
  detail: DetailBody,
}

export function ModuleSkeleton({ variant = 'list' }: { variant?: ModuleSkeletonVariant }) {
  const Body = BODIES[variant] ?? ListBody
  return (
    <div className="h-full overflow-hidden p-6" aria-busy="true" aria-live="polite">
      <HeaderSkel action={variant !== 'detail'} />
      <Body />
    </div>
  )
}
