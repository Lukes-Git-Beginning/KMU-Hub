import { useState, useMemo } from 'react'
import {
  Plus,
  FileBarChart,
  FilePlus,
  CalendarClock,
  ArrowUpRight,
  ArrowDownRight,
  TrendingUp,
  Filter,
  Download,
  FileText,
  Clock,
  Mail,
  X,
  Loader2,
  Eye,
  BarChart3,
  Calendar,
  ChevronRight,
  ArrowLeftRight,
  Landmark,
  Table2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useBerichteStore } from '@/stores/berichte'
import { PageHeader } from '@/components/shared'
import { EmptyState } from '@/components/shared/EmptyState'
import { EmptyReports } from '@/components/shared/illustrations'

/* ---------- constants ---------- */

type TabKey = 'dashboard' | 'erstellen' | 'geplant' | 'datev'

const scheduleLabels: Record<string, string> = {
  daily: 'Taeglich',
  weekly: 'Woechentlich',
  monthly: 'Monatlich',
  quarterly: 'Quartalsweise',
}

const scheduleColors: Record<string, string> = {
  daily: 'bg-info-light text-info',
  weekly: 'bg-warning-light text-warning',
  monthly: 'bg-primary-light text-primary',
  quarterly: 'bg-success-light text-success',
}

const TICKET_PRIORITY_DATA = [
  { label: 'Kritisch', value: 14, color: 'bg-error', track: 'bg-error/20' },
  { label: 'Hoch', value: 38, color: 'bg-warning', track: 'bg-warning/20' },
  { label: 'Mittel', value: 62, color: 'bg-info', track: 'bg-info/20' },
  { label: 'Niedrig', value: 27, color: 'bg-muted-foreground/60', track: 'bg-secondary' },
]

/* ---------- helpers ---------- */

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString('de-CH', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
    })
  } catch {
    return '—'
  }
}

function formatDateTime(iso: string): string {
  try {
    const d = new Date(iso)
    return `${d.toLocaleDateString('de-CH', { day: '2-digit', month: '2-digit', year: 'numeric' })} ${d.toLocaleTimeString('de-CH', { hour: '2-digit', minute: '2-digit' })}`
  } catch {
    return '—'
  }
}

function computeNextRun(lastRun: string, schedule: string): string {
  try {
    const d = new Date(lastRun)
    switch (schedule) {
      case 'daily':
        d.setDate(d.getDate() + 1)
        break
      case 'weekly':
        d.setDate(d.getDate() + 7)
        break
      case 'monthly':
        d.setMonth(d.getMonth() + 1)
        break
      case 'quarterly':
        d.setMonth(d.getMonth() + 3)
        break
    }
    return formatDate(d.toISOString())
  } catch {
    return '—'
  }
}

/* ---------- component ---------- */

export default function BerichtePage() {
  const { kpis, chartData, scheduledReports, savedReports, modules, kpiDrilldownData, comparisonChartData, datevBWA, datevSuSa } = useBerichteStore()

  /* --- tab state --- */
  const [tab, setTab] = useState<TabKey>('dashboard')

  /* --- dashboard state --- */
  const [moduleFilter, setModuleFilter] = useState<string>('all')
  const [drilldownKpi, setDrilldownKpi] = useState<string | null>(null)
  const [showComparison, setShowComparison] = useState(false)
  const [datevSubTab, setDatevSubTab] = useState<'bwa' | 'susa'>('bwa')

  /* --- erstellen state --- */
  const [reportName, setReportName] = useState('')
  const [selectedModule, setSelectedModule] = useState('')
  const [dateFrom, setDateFrom] = useState('2026-02-01')
  const [dateTo, setDateTo] = useState('2026-02-15')
  const [format, setFormat] = useState<'pdf' | 'excel'>('pdf')
  const [isGenerating, setIsGenerating] = useState(false)

  /* --- geplant state --- */
  const [localToggles, setLocalToggles] = useState<Record<string, boolean>>(() => {
    const init: Record<string, boolean> = {}
    scheduledReports.forEach((r) => {
      init[r.id] = r.active
    })
    return init
  })
  const [showScheduleDialog, setShowScheduleDialog] = useState(false)

  /* --- schedule dialog state --- */
  const [schedDialogReport, setSchedDialogReport] = useState('')
  const [schedDialogSchedule, setSchedDialogSchedule] = useState<'daily' | 'weekly' | 'monthly'>('weekly')
  const [schedDialogEmail, setSchedDialogEmail] = useState('')
  const [schedDialogRecipients, setSchedDialogRecipients] = useState<string[]>([])

  /* --- derived --- */
  const activeScheduled = scheduledReports.filter((r) => localToggles[r.id] ?? r.active).length

  const filteredKpis = useMemo(() => {
    if (moduleFilter === 'all') return kpis
    return kpis.filter((k) => k.moduleId === moduleFilter)
  }, [kpis, moduleFilter])

  const chartMax = useMemo(() => Math.max(...chartData.map((b) => b.value), 1), [chartData])
  const ticketMax = useMemo(
    () => Math.max(...TICKET_PRIORITY_DATA.map((d) => d.value), 1),
    [],
  )

  /* --- handlers --- */

  const handleNewReport = () => {
    setTab('erstellen')
    setReportName('')
    setSelectedModule('')
  }

  const handleGenerate = () => {
    if (!reportName.trim()) {
      toast.error('Bitte einen Berichtsnamen eingeben')
      return
    }
    if (!selectedModule) {
      toast.error('Bitte ein Modul auswaehlen')
      return
    }
    setIsGenerating(true)
    setTimeout(() => {
      setIsGenerating(false)
      toast.success(`Bericht "${reportName}" wurde als ${format.toUpperCase()} generiert`)
      setReportName('')
      setSelectedModule('')
    }, 2200)
  }

  const handleToggleScheduled = (id: string) => {
    setLocalToggles((prev) => {
      const next = { ...prev, [id]: !prev[id] }
      toast.info(next[id] ? 'Geplanter Bericht aktiviert' : 'Geplanter Bericht deaktiviert')
      return next
    })
  }

  const handleAddRecipient = () => {
    const email = schedDialogEmail.trim()
    if (!email) return
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      toast.error('Bitte eine gueltige E-Mail-Adresse eingeben')
      return
    }
    if (schedDialogRecipients.includes(email)) {
      toast.error('Diese E-Mail ist bereits in der Liste')
      return
    }
    setSchedDialogRecipients((prev) => [...prev, email])
    setSchedDialogEmail('')
  }

  const handleRemoveRecipient = (email: string) => {
    setSchedDialogRecipients((prev) => prev.filter((e) => e !== email))
  }

  const handleSaveScheduled = () => {
    if (!schedDialogReport) {
      toast.error('Bitte einen Bericht auswaehlen')
      return
    }
    if (schedDialogRecipients.length === 0) {
      toast.error('Bitte mindestens einen Empfaenger hinzufuegen')
      return
    }
    toast.success('Geplanter Bericht wurde gespeichert')
    setShowScheduleDialog(false)
    setSchedDialogReport('')
    setSchedDialogSchedule('weekly')
    setSchedDialogEmail('')
    setSchedDialogRecipients([])
  }

  const openScheduleDialog = () => {
    setSchedDialogReport('')
    setSchedDialogSchedule('weekly')
    setSchedDialogEmail('')
    setSchedDialogRecipients([])
    setShowScheduleDialog(true)
  }

  /* ---------- render ---------- */
  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* ---- Header ---- */}
      <PageHeader
        title="Berichte"
        description={`${kpis.length} KPIs · ${activeScheduled} geplante Berichte aktiv`}
        icon={BarChart3}
        moduleId="berichte"
        actions={
          <button
            onClick={handleNewReport}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
          >
            <Plus className="h-4 w-4" />
            Neuer Bericht
          </button>
        }
        className="mb-6"
      />

      {/* ---- Tabs ---- */}
      <div className="mb-6 flex items-center gap-4 border-b border-border">
        {(
          [
            { key: 'dashboard' as const, label: 'Dashboard', icon: BarChart3 },
            { key: 'erstellen' as const, label: 'Erstellen', icon: FilePlus },
            { key: 'geplant' as const, label: `Geplant (${scheduledReports.length})`, icon: CalendarClock },
            { key: 'datev' as const, label: 'DATEV', icon: Landmark },
          ] as const
        ).map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key
                ? 'border-primary font-medium text-primary tab-accent-active'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <t.icon className="h-3.5 w-3.5" />
            {t.label}
          </button>
        ))}
      </div>

      {/* ================================================================ */}
      {/*  DASHBOARD TAB                                                    */}
      {/* ================================================================ */}
      {tab === 'dashboard' && (
        <>
          {/* Module filter */}
          <div className="mb-5 flex items-center gap-2">
            <Filter className="h-4 w-4 text-muted-foreground" />
            <select
              value={moduleFilter}
              onChange={(e) => setModuleFilter(e.target.value)}
              className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              <option value="all">Alle Module</option>
              {modules.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name}
                </option>
              ))}
            </select>
            {moduleFilter !== 'all' && (
              <button
                onClick={() => setModuleFilter('all')}
                className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-secondary"
              >
                Filter zuruecksetzen
              </button>
            )}
          </div>

          {/* KPI Cards */}
          <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 animate-fade-up" style={{ animationDelay: '100ms' }}>
            {filteredKpis.length === 0 && (
              <div className="col-span-full">
                <EmptyState
                  illustration={<EmptyReports />}
                  title="Keine KPIs fuer dieses Modul vorhanden"
                />
              </div>
            )}
            {filteredKpis.map((kpi) => {
              const isPositive = kpi.changePercent >= 0
              // For "Reaktionszeit" and "Ausschussquote" a decrease is good
              const isGood =
                kpi.id === 'kpi-4' || kpi.id === 'kpi-6'
                  ? !isPositive
                  : isPositive
              const isActive = drilldownKpi === kpi.id
              return (
                <button
                  key={kpi.id}
                  onClick={() => setDrilldownKpi(isActive ? null : kpi.id)}
                  className={`group rounded-xl border bg-card p-4 text-left transition-all duration-200 hover:border-primary/30 hover:-translate-y-0.5 hover:shadow-md ${
                    isActive ? 'border-primary ring-1 ring-primary/20' : 'border-border'
                  }`}
                >
                  <div className="mb-1 flex items-center justify-between">
                    <p className="text-xs text-muted-foreground">{kpi.label}</p>
                    <div className="flex items-center gap-1.5">
                      <div
                        className={`flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[10px] font-medium ${
                          isGood
                            ? 'bg-success-light text-success'
                            : 'bg-error-light text-error'
                        }`}
                      >
                        {isPositive ? (
                          <ArrowUpRight className="h-3 w-3" />
                        ) : (
                          <ArrowDownRight className="h-3 w-3" />
                        )}
                        {isPositive ? '+' : ''}
                        {kpi.changePercent}%
                      </div>
                      <ChevronRight className={`h-3.5 w-3.5 text-muted-foreground transition-transform ${isActive ? 'rotate-90' : ''}`} />
                    </div>
                  </div>
                  <div className="mb-1 flex items-baseline gap-2">
                    <span className="text-2xl font-semibold text-foreground">{kpi.value}</span>
                    {kpi.unit && (
                      <span className="text-sm text-muted-foreground">{kpi.unit}</span>
                    )}
                  </div>
                  <p className="text-[10px] text-muted-foreground">vs. Vormonat</p>
                </button>
              )
            })}
          </div>

          {/* KPI Drill-Down Panel */}
          {drilldownKpi && kpiDrilldownData[drilldownKpi] && (
            <div className="mb-8 rounded-lg border border-primary/20 bg-card p-5 animate-in fade-in slide-in-from-top-2 duration-200">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-sm font-medium text-foreground">
                  {kpis.find((k) => k.id === drilldownKpi)?.label} — Details
                </h3>
                <button
                  onClick={() => setDrilldownKpi(null)}
                  className="rounded-lg p-1 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border">
                      <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">Datum</th>
                      <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">Bezeichnung</th>
                      <th className="px-3 py-2 text-right text-xs font-medium text-muted-foreground">Wert</th>
                      <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {kpiDrilldownData[drilldownKpi].map((row, i) => (
                      <tr key={i} className="border-b border-border/40 last:border-0">
                        <td className="px-3 py-2 text-xs text-muted-foreground">{row.date}</td>
                        <td className="px-3 py-2 text-sm text-foreground">{row.label}</td>
                        <td className="px-3 py-2 text-right text-sm font-medium text-foreground">{row.amount}</td>
                        <td className="px-3 py-2">
                          <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
                            {row.status}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Comparison toggle */}
          <div className="mb-4 flex items-center gap-3">
            <button
              onClick={() => setShowComparison(!showComparison)}
              className={`flex items-center gap-2 rounded-lg border px-3 py-1.5 text-sm transition-colors ${
                showComparison
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-border text-muted-foreground hover:bg-secondary'
              }`}
            >
              <ArrowLeftRight className="h-3.5 w-3.5" />
              Periodenvergleich
            </button>
          </div>

          {/* Charts row */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2 animate-fade-up" style={{ animationDelay: '200ms' }}>
            {/* Vertical bar chart: Umsatzverlauf (with optional comparison) */}
            <div className="rounded-xl border border-border bg-card p-6">
              <div className="mb-5 flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-medium text-foreground">Umsatzverlauf</h3>
                  <p className="text-xs text-muted-foreground">
                    {showComparison ? 'Aktuell vs. Vorjahr (Tsd. EUR)' : 'Letzte 6 Monate in Tsd. EUR'}
                  </p>
                </div>
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary-light">
                  <TrendingUp className="h-4 w-4 text-primary" />
                </div>
              </div>

              {showComparison ? (
                <>
                  <div className="flex items-end gap-3 h-48">
                    {comparisonChartData.map((bar, idx) => {
                      const compMax = Math.max(...comparisonChartData.flatMap(b => [b.current, b.previous]), 1)
                      const curPct = Math.max(6, (bar.current / compMax) * 100)
                      const prevPct = Math.max(6, (bar.previous / compMax) * 100)
                      return (
                        <div key={idx} className="group/bar flex flex-1 flex-col items-center gap-1">
                          <div className="flex items-end gap-0.5 w-full h-full">
                            <div className="flex-1 flex flex-col items-center justify-end h-full">
                              <span className="text-[9px] font-medium text-muted-foreground opacity-0 group-hover/bar:opacity-100 transition-opacity">{bar.previous}k</span>
                              <div className="w-full rounded-t bg-muted-foreground/30" style={{ height: `${prevPct}%` }} />
                            </div>
                            <div className="flex-1 flex flex-col items-center justify-end h-full">
                              <span className="text-[9px] font-medium text-primary opacity-0 group-hover/bar:opacity-100 transition-opacity">{bar.current}k</span>
                              <div className="w-full rounded-t bg-primary" style={{ height: `${curPct}%` }} />
                            </div>
                          </div>
                          <span className="text-[10px] text-muted-foreground">{bar.label}</span>
                        </div>
                      )
                    })}
                  </div>
                  {/* Legend */}
                  <div className="mt-3 flex items-center justify-center gap-4">
                    <div className="flex items-center gap-1.5">
                      <div className="h-2.5 w-2.5 rounded-sm bg-muted-foreground/30" />
                      <span className="text-[10px] text-muted-foreground">Vorjahr</span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <div className="h-2.5 w-2.5 rounded-sm bg-primary" />
                      <span className="text-[10px] text-muted-foreground">Aktuell</span>
                    </div>
                  </div>
                  {/* Difference summary */}
                  <div className="mt-3 grid grid-cols-3 gap-2">
                    {(() => {
                      const totalCurr = comparisonChartData.reduce((s, b) => s + b.current, 0)
                      const totalPrev = comparisonChartData.reduce((s, b) => s + b.previous, 0)
                      const diff = totalCurr - totalPrev
                      const diffPct = ((diff / totalPrev) * 100).toFixed(1)
                      return (
                        <>
                          <div className="rounded-lg bg-muted/50 p-2 text-center">
                            <p className="text-[10px] text-muted-foreground">Aktuell</p>
                            <p className="text-sm font-semibold text-foreground">{totalCurr}k</p>
                          </div>
                          <div className="rounded-lg bg-muted/50 p-2 text-center">
                            <p className="text-[10px] text-muted-foreground">Vorjahr</p>
                            <p className="text-sm font-semibold text-foreground">{totalPrev}k</p>
                          </div>
                          <div className="rounded-lg bg-muted/50 p-2 text-center">
                            <p className="text-[10px] text-muted-foreground">Differenz</p>
                            <p className={`text-sm font-semibold ${diff >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                              {diff >= 0 ? '+' : ''}{diff}k ({diff >= 0 ? '+' : ''}{diffPct}%)
                            </p>
                          </div>
                        </>
                      )
                    })()}
                  </div>
                </>
              ) : (
                <div className="flex items-end gap-3 h-48">
                  {chartData.map((bar, idx) => {
                    const pct = Math.max(6, (bar.value / chartMax) * 100)
                    const isLast = idx === chartData.length - 1
                    return (
                      <div key={idx} className="group/bar flex flex-1 flex-col items-center gap-1">
                        <span className="text-[10px] font-medium text-muted-foreground opacity-0 transition-opacity group-hover/bar:opacity-100">
                          {bar.value}k
                        </span>
                        <div
                          className={`w-full rounded-t transition-all duration-500 ${
                            isLast ? 'bg-primary' : 'bg-primary/50'
                          } group-hover/bar:bg-primary`}
                          style={{
                            height: `${pct}%`,
                            animationDelay: `${idx * 80}ms`,
                          }}
                        />
                        <span
                          className={`text-[10px] ${
                            isLast ? 'font-medium text-primary' : 'text-muted-foreground'
                          }`}
                        >
                          {bar.label}
                        </span>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>

            {/* Horizontal bar chart: Tickets nach Prioritaet */}
            <div className="rounded-xl border border-border bg-card p-6">
              <div className="mb-5 flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-medium text-foreground">Tickets nach Prioritaet</h3>
                  <p className="text-xs text-muted-foreground">Aktuelle offene Tickets</p>
                </div>
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-warning-light">
                  <BarChart3 className="h-4 w-4 text-warning" />
                </div>
              </div>
              <div className="space-y-4">
                {TICKET_PRIORITY_DATA.map((item) => {
                  const pct = Math.max(4, (item.value / ticketMax) * 100)
                  return (
                    <div key={item.label}>
                      <div className="mb-1.5 flex items-center justify-between">
                        <span className="text-xs font-medium text-foreground">{item.label}</span>
                        <span className="text-xs text-muted-foreground">{item.value} Tickets</span>
                      </div>
                      <div className={`h-3 w-full overflow-hidden rounded-full ${item.track}`}>
                        <div
                          className={`h-3 rounded-full ${item.color} transition-all duration-700`}
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        </>
      )}

      {/* ================================================================ */}
      {/*  ERSTELLEN TAB                                                    */}
      {/* ================================================================ */}
      {tab === 'erstellen' && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {/* Create form */}
          <div className="lg:col-span-2">
            <div className="rounded-xl border border-border bg-card p-6">
              {/* Header */}
              <div className="mb-6 flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                  <FilePlus className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <h3 className="text-sm font-medium text-foreground">Bericht erstellen</h3>
                  <p className="text-xs text-muted-foreground">
                    Waehle Modul, Zeitraum und Format
                  </p>
                </div>
              </div>

              <div className="space-y-5">
                {/* Report Name */}
                <div>
                  <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                    Berichtsname
                  </label>
                  <input
                    type="text"
                    placeholder="z.B. Monatsbericht Februar"
                    value={reportName}
                    onChange={(e) => setReportName(e.target.value)}
                    className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>

                {/* Module Dropdown */}
                <div>
                  <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                    Modul
                  </label>
                  <select
                    value={selectedModule}
                    onChange={(e) => setSelectedModule(e.target.value)}
                    className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  >
                    <option value="">Modul auswaehlen...</option>
                    {modules.map((mod) => (
                      <option key={mod.id} value={mod.id}>
                        {mod.name}
                      </option>
                    ))}
                  </select>
                </div>

                {/* Date Range */}
                <div>
                  <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                    Zeitraum
                  </label>
                  <div className="flex items-center gap-2">
                    <div className="relative flex-1">
                      <Calendar className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                      <input
                        type="date"
                        value={dateFrom}
                        onChange={(e) => setDateFrom(e.target.value)}
                        className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                      />
                    </div>
                    <span className="text-xs text-muted-foreground">bis</span>
                    <div className="relative flex-1">
                      <Calendar className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                      <input
                        type="date"
                        value={dateTo}
                        onChange={(e) => setDateTo(e.target.value)}
                        className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                      />
                    </div>
                  </div>
                </div>

                {/* Format Radio */}
                <div>
                  <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                    Ausgabeformat
                  </label>
                  <div className="flex gap-3">
                    {(['pdf', 'excel'] as const).map((f) => (
                      <button
                        key={f}
                        onClick={() => setFormat(f)}
                        className={`flex items-center gap-2 rounded-lg border px-4 py-2 text-sm transition-colors ${
                          format === f
                            ? 'border-primary bg-primary-light text-primary'
                            : 'border-border text-muted-foreground hover:bg-secondary'
                        }`}
                      >
                        {f === 'pdf' ? (
                          <FileText className="h-4 w-4" />
                        ) : (
                          <FileBarChart className="h-4 w-4" />
                        )}
                        {f.toUpperCase()}
                      </button>
                    ))}
                  </div>
                </div>

                {/* Preview Section */}
                <div className="rounded-lg border border-dashed border-border bg-secondary/30 p-6">
                  {isGenerating ? (
                    <div className="flex flex-col items-center justify-center gap-3 py-4">
                      <Loader2 className="h-8 w-8 animate-spin text-primary" />
                      <p className="text-sm font-medium text-foreground">
                        Vorschau wird generiert...
                      </p>
                      <p className="text-xs text-muted-foreground">
                        Daten werden ausgewertet und formatiert
                      </p>
                    </div>
                  ) : (
                    <div className="flex flex-col items-center justify-center gap-2 py-4 text-center">
                      <Eye className="h-8 w-8 text-muted-foreground/40" />
                      <p className="text-sm text-muted-foreground">
                        Vorschau verfuegbar nach Generierung
                      </p>
                      <p className="text-xs text-muted-foreground/70">
                        Felder ausfuellen und &quot;Bericht generieren&quot; klicken
                      </p>
                    </div>
                  )}
                </div>

                {/* Generate Button */}
                <button
                  onClick={handleGenerate}
                  disabled={isGenerating}
                  className="flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:opacity-60"
                >
                  {isGenerating ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Download className="h-4 w-4" />
                  )}
                  {isGenerating ? 'Wird generiert...' : 'Bericht generieren'}
                </button>
              </div>
            </div>
          </div>

          {/* Saved Reports sidebar */}
          <div className="lg:col-span-1">
            <div className="rounded-lg border border-border bg-card p-4">
              <div className="mb-4 flex items-center gap-2">
                <FileText className="h-4 w-4 text-muted-foreground" />
                <h3 className="text-sm font-medium text-foreground">Gespeicherte Berichte</h3>
              </div>

              {savedReports.length === 0 ? (
                <EmptyState
                  illustration={<EmptyReports />}
                  title="Noch keine Berichte gespeichert"
                />
              ) : (
                <div className="space-y-3">
                  {savedReports.map((report) => (
                    <div
                      key={report.id}
                      className="rounded-lg border border-border-muted p-3 transition-colors hover:border-primary/30"
                    >
                      <div className="mb-1 flex items-start justify-between gap-2">
                        <h4 className="text-sm font-medium leading-tight text-foreground">
                          {report.name}
                        </h4>
                      </div>
                      <p className="mb-2 line-clamp-2 text-xs text-muted-foreground">
                        {report.description}
                      </p>
                      <div className="mb-2 flex flex-wrap items-center gap-2">
                        <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
                          {report.module}
                        </span>
                        <span className="text-[10px] text-muted-foreground/60">
                          {report.createdBy}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] text-muted-foreground/60">
                          {formatDate(report.createdAt)}
                        </span>
                        <button
                          onClick={() =>
                            toast.info(`Bericht "${report.name}" wird geoeffnet...`)
                          }
                          className="flex items-center gap-1 rounded-lg border border-border px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                        >
                          <Eye className="h-3 w-3" />
                          Oeffnen
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* ================================================================ */}
      {/*  GEPLANT TAB                                                      */}
      {/* ================================================================ */}
      {tab === 'geplant' && (
        <>
          {/* Action bar */}
          <div className="mb-4 flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              {activeScheduled} von {scheduledReports.length} Berichten aktiv
            </p>
            <button
              onClick={openScheduleDialog}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
            >
              <Plus className="h-4 w-4" />
              Neuer geplanter Bericht
            </button>
          </div>

          {/* Table */}
          <div className="overflow-hidden rounded-lg border border-border bg-card">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                      Name
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                      Zeitplan
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                      Empfaenger
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                      Letzter Lauf
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                      Naechster Lauf
                    </th>
                    <th className="px-4 py-3 text-center text-xs font-medium text-muted-foreground">
                      Aktiv
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {scheduledReports.map((report) => {
                    const isActive = localToggles[report.id] ?? report.active
                    return (
                      <tr
                        key={report.id}
                        className="border-b border-border-muted transition-colors last:border-0 hover:bg-secondary/50"
                      >
                        <td className="px-4 py-3">
                          <div>
                            <span className="font-medium text-foreground">{report.name}</span>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                              scheduleColors[report.schedule] ??
                              'bg-secondary text-muted-foreground'
                            }`}
                          >
                            {scheduleLabels[report.schedule] ?? report.schedule}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex flex-wrap gap-1">
                            {report.recipients.map((email) => (
                              <span
                                key={email}
                                className="rounded-full bg-secondary px-2 py-0.5 text-xs text-muted-foreground"
                              >
                                {email}
                              </span>
                            ))}
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                            <Clock className="h-3 w-3" />
                            {report.lastRun ? formatDateTime(report.lastRun) : '—'}
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span className="text-xs text-muted-foreground">
                            {isActive
                              ? computeNextRun(report.lastRun, report.schedule)
                              : '— (deaktiviert)'}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex justify-center">
                            <button
                              onClick={() => handleToggleScheduled(report.id)}
                              className={`flex h-6 w-10 items-center rounded-full p-0.5 transition-colors ${
                                isActive ? 'bg-primary' : 'bg-secondary'
                              }`}
                              aria-label={isActive ? 'Deaktivieren' : 'Aktivieren'}
                            >
                              <div
                                className={`h-5 w-5 rounded-full bg-white shadow transition-transform ${
                                  isActive ? 'translate-x-4' : 'translate-x-0'
                                }`}
                              />
                            </button>
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
            {scheduledReports.length === 0 && (
              <EmptyState
                illustration={<EmptyReports />}
                title="Keine geplanten Berichte"
                description="Erstelle einen geplanten Bericht"
              />
            )}
          </div>
        </>
      )}

      {/* ================================================================ */}
      {/*  DATEV TAB                                                         */}
      {/* ================================================================ */}
      {tab === 'datev' && (
        <>
          <div className="mb-4 flex items-center justify-between">
            <div className="flex gap-1">
              {([
                { key: 'bwa' as const, label: 'BWA', icon: BarChart3 },
                { key: 'susa' as const, label: 'Summen & Salden', icon: Table2 },
              ]).map((st) => (
                <button
                  key={st.key}
                  onClick={() => setDatevSubTab(st.key)}
                  className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors ${
                    datevSubTab === st.key
                      ? 'bg-primary/10 text-primary font-medium'
                      : 'text-muted-foreground hover:bg-secondary'
                  }`}
                >
                  <st.icon className="h-3.5 w-3.5" />
                  {st.label}
                </button>
              ))}
            </div>
            <button
              onClick={() => toast.success('DATEV-Export wird generiert...')}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
            >
              <Download className="h-4 w-4" />
              DATEV Export
            </button>
          </div>

          {datevSubTab === 'bwa' && (
            <div className="rounded-lg border border-border bg-card overflow-hidden">
              <div className="border-b border-border px-4 py-3">
                <h3 className="text-sm font-medium text-foreground">Betriebswirtschaftliche Auswertung (BWA)</h3>
                <p className="text-xs text-muted-foreground">Zeitraum: Februar 2026</p>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border bg-muted/30">
                      <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground w-8">Pos.</th>
                      <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Bezeichnung</th>
                      <th className="px-4 py-2.5 text-right text-xs font-medium text-muted-foreground">Aktueller Monat</th>
                      <th className="px-4 py-2.5 text-right text-xs font-medium text-muted-foreground">Vormonat</th>
                      <th className="px-4 py-2.5 text-right text-xs font-medium text-muted-foreground">Kumuliert</th>
                    </tr>
                  </thead>
                  <tbody>
                    {datevBWA.map((row) => {
                      const isSummary = ['3', '5', '9', '11'].includes(row.position)
                      return (
                        <tr key={row.position} className={`border-b border-border/40 last:border-0 ${isSummary ? 'bg-muted/20 font-medium' : ''}`}>
                          <td className="px-4 py-2 text-xs text-muted-foreground">{row.position}</td>
                          <td className={`px-4 py-2 text-sm ${isSummary ? 'text-foreground font-medium' : 'text-foreground'}`}>{row.label}</td>
                          <td className={`px-4 py-2 text-right text-sm ${row.currentMonth < 0 ? 'text-red-600 dark:text-red-400' : 'text-foreground'}`}>
                            {row.currentMonth.toLocaleString('de-DE')} EUR
                          </td>
                          <td className={`px-4 py-2 text-right text-sm ${row.previousMonth < 0 ? 'text-red-600 dark:text-red-400' : 'text-muted-foreground'}`}>
                            {row.previousMonth.toLocaleString('de-DE')} EUR
                          </td>
                          <td className={`px-4 py-2 text-right text-sm ${row.yearToDate < 0 ? 'text-red-600 dark:text-red-400' : 'text-foreground'}`}>
                            {row.yearToDate.toLocaleString('de-DE')} EUR
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {datevSubTab === 'susa' && (
            <div className="rounded-lg border border-border bg-card overflow-hidden">
              <div className="border-b border-border px-4 py-3">
                <h3 className="text-sm font-medium text-foreground">Summen- und Saldenliste</h3>
                <p className="text-xs text-muted-foreground">Zeitraum: Februar 2026</p>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border bg-muted/30">
                      <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground w-16">Konto</th>
                      <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Bezeichnung</th>
                      <th className="px-4 py-2.5 text-right text-xs font-medium text-muted-foreground">Aktueller Monat</th>
                      <th className="px-4 py-2.5 text-right text-xs font-medium text-muted-foreground">Vormonat</th>
                      <th className="px-4 py-2.5 text-right text-xs font-medium text-muted-foreground">Kumuliert</th>
                    </tr>
                  </thead>
                  <tbody>
                    {datevSuSa.map((row) => (
                      <tr key={row.position} className="border-b border-border/40 last:border-0 hover:bg-muted/20">
                        <td className="px-4 py-2 text-xs font-mono text-muted-foreground">{row.position}</td>
                        <td className="px-4 py-2 text-sm text-foreground">{row.label}</td>
                        <td className={`px-4 py-2 text-right text-sm ${row.currentMonth < 0 ? 'text-red-600 dark:text-red-400' : 'text-foreground'}`}>
                          {row.currentMonth.toLocaleString('de-DE')} EUR
                        </td>
                        <td className={`px-4 py-2 text-right text-sm ${row.previousMonth < 0 ? 'text-red-600 dark:text-red-400' : 'text-muted-foreground'}`}>
                          {row.previousMonth.toLocaleString('de-DE')} EUR
                        </td>
                        <td className={`px-4 py-2 text-right text-sm ${row.yearToDate < 0 ? 'text-red-600 dark:text-red-400' : 'text-foreground'}`}>
                          {row.yearToDate.toLocaleString('de-DE')} EUR
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}

      {/* ================================================================ */}
      {/*  DIALOG: Neuer geplanter Bericht                                  */}
      {/* ================================================================ */}
      {showScheduleDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="mx-4 w-full max-w-lg rounded-xl border border-border bg-card shadow-xl">
            {/* Dialog header */}
            <div className="flex items-center justify-between border-b border-border px-6 py-4">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
                  <CalendarClock className="h-4 w-4 text-primary" />
                </div>
                <div>
                  <h2 className="text-sm font-semibold text-foreground">
                    Neuer geplanter Bericht
                  </h2>
                  <p className="text-xs text-muted-foreground">
                    Automatischen Berichtsversand einrichten
                  </p>
                </div>
              </div>
              <button
                onClick={() => setShowScheduleDialog(false)}
                className="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Dialog body */}
            <div className="space-y-5 px-6 py-5">
              {/* Select report */}
              <div>
                <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                  Bericht auswaehlen
                </label>
                <select
                  value={schedDialogReport}
                  onChange={(e) => setSchedDialogReport(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  <option value="">Bericht auswaehlen...</option>
                  {savedReports.map((r) => (
                    <option key={r.id} value={r.id}>
                      {r.name}
                    </option>
                  ))}
                </select>
              </div>

              {/* Schedule radio */}
              <div>
                <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                  Zeitplan
                </label>
                <div className="flex gap-2">
                  {(
                    [
                      { value: 'daily', label: 'Taeglich' },
                      { value: 'weekly', label: 'Woechentlich' },
                      { value: 'monthly', label: 'Monatlich' },
                    ] as const
                  ).map((opt) => (
                    <button
                      key={opt.value}
                      onClick={() => setSchedDialogSchedule(opt.value)}
                      className={`flex-1 rounded-lg border px-3 py-2 text-sm transition-colors ${
                        schedDialogSchedule === opt.value
                          ? 'border-primary bg-primary-light text-primary'
                          : 'border-border text-muted-foreground hover:bg-secondary'
                      }`}
                    >
                      {opt.label}
                    </button>
                  ))}
                </div>
              </div>

              {/* Recipients */}
              <div>
                <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                  Empfaenger
                </label>
                <div className="flex gap-2">
                  <div className="relative flex-1">
                    <Mail className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                    <input
                      type="email"
                      placeholder="email@firma.ch"
                      value={schedDialogEmail}
                      onChange={(e) => setSchedDialogEmail(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          e.preventDefault()
                          handleAddRecipient()
                        }
                      }}
                      className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    />
                  </div>
                  <button
                    onClick={handleAddRecipient}
                    className="rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                  >
                    Hinzufuegen
                  </button>
                </div>

                {/* Recipient chips */}
                {schedDialogRecipients.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-1.5">
                    {schedDialogRecipients.map((email) => (
                      <span
                        key={email}
                        className="flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-xs text-foreground"
                      >
                        {email}
                        <button
                          onClick={() => handleRemoveRecipient(email)}
                          className="ml-0.5 rounded-full p-0.5 transition-colors hover:bg-error-light hover:text-error"
                        >
                          <X className="h-2.5 w-2.5" />
                        </button>
                      </span>
                    ))}
                  </div>
                )}
                {schedDialogRecipients.length === 0 && (
                  <p className="mt-1.5 text-[10px] text-muted-foreground/60">
                    Mindestens ein Empfaenger erforderlich
                  </p>
                )}
              </div>
            </div>

            {/* Dialog footer */}
            <div className="flex items-center justify-end gap-3 border-t border-border px-6 py-4">
              <button
                onClick={() => setShowScheduleDialog(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary"
              >
                Abbrechen
              </button>
              <button
                onClick={handleSaveScheduled}
                className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
              >
                <CalendarClock className="h-4 w-4" />
                Speichern
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
