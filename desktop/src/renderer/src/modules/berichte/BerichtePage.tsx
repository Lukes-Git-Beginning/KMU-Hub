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
} from 'lucide-react'
import { toast } from 'sonner'
import { useBerichteStore } from '@/stores/berichte'

/* ---------- constants ---------- */

type TabKey = 'dashboard' | 'erstellen' | 'geplant'

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
  const { kpis, chartData, scheduledReports, savedReports, modules } = useBerichteStore()

  /* --- tab state --- */
  const [tab, setTab] = useState<TabKey>('dashboard')

  /* --- dashboard state --- */
  const [moduleFilter, setModuleFilter] = useState<string>('all')

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
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-foreground">Berichte</h1>
          <p className="text-sm text-muted-foreground">
            {kpis.length} KPIs &middot; {activeScheduled} geplante Berichte aktiv
          </p>
        </div>
        <button
          onClick={handleNewReport}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
        >
          <Plus className="h-4 w-4" />
          Neuer Bericht
        </button>
      </div>

      {/* ---- Tabs ---- */}
      <div className="mb-6 flex items-center gap-4 border-b border-border">
        {(
          [
            { key: 'dashboard' as const, label: 'Dashboard', icon: BarChart3 },
            { key: 'erstellen' as const, label: 'Erstellen', icon: FilePlus },
            { key: 'geplant' as const, label: `Geplant (${scheduledReports.length})`, icon: CalendarClock },
          ] as const
        ).map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key
                ? 'border-primary font-medium text-primary'
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
          <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filteredKpis.length === 0 && (
              <div className="col-span-full rounded-lg border border-border bg-card p-8 text-center">
                <TrendingUp className="mx-auto mb-2 h-8 w-8 text-muted-foreground/40" />
                <p className="text-sm text-muted-foreground">
                  Keine KPIs fuer dieses Modul vorhanden
                </p>
              </div>
            )}
            {filteredKpis.map((kpi) => {
              const isPositive = kpi.changePercent >= 0
              // For "Reaktionszeit" and "Ausschussquote" a decrease is good
              const isGood =
                kpi.id === 'kpi-4' || kpi.id === 'kpi-6'
                  ? !isPositive
                  : isPositive
              return (
                <div
                  key={kpi.id}
                  className="group rounded-lg border border-border bg-card p-4 transition-colors hover:border-primary/30"
                >
                  <div className="mb-1 flex items-center justify-between">
                    <p className="text-xs text-muted-foreground">{kpi.label}</p>
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
                  </div>
                  <div className="mb-1 flex items-baseline gap-2">
                    <span className="text-2xl font-semibold text-foreground">{kpi.value}</span>
                    {kpi.unit && (
                      <span className="text-sm text-muted-foreground">{kpi.unit}</span>
                    )}
                  </div>
                  <p className="text-[10px] text-muted-foreground">vs. Vormonat</p>
                </div>
              )
            })}
          </div>

          {/* Charts row */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            {/* Vertical bar chart: Umsatzverlauf */}
            <div className="rounded-lg border border-border bg-card p-6">
              <div className="mb-5 flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-medium text-foreground">Umsatzverlauf</h3>
                  <p className="text-xs text-muted-foreground">Letzte 6 Monate in Tsd. CHF</p>
                </div>
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary-light">
                  <TrendingUp className="h-4 w-4 text-primary" />
                </div>
              </div>
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
            </div>

            {/* Horizontal bar chart: Tickets nach Prioritaet */}
            <div className="rounded-lg border border-border bg-card p-6">
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
            <div className="rounded-lg border border-border bg-card p-6">
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
                <div className="py-8 text-center">
                  <FileText className="mx-auto mb-2 h-8 w-8 text-muted-foreground/30" />
                  <p className="text-xs text-muted-foreground">Noch keine Berichte gespeichert</p>
                </div>
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
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <CalendarClock className="mb-3 h-10 w-10 opacity-40" />
                <p className="text-sm font-medium">Keine geplanten Berichte</p>
                <p className="mt-1 text-xs">Erstelle einen geplanten Bericht</p>
              </div>
            )}
          </div>
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
