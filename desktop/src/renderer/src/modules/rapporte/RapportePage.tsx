import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Plus,
  ClipboardList,
  Sun,
  Cloud,
  CloudRain,
  Snowflake,
  Clock,
  Users,
  Camera,
  AlertTriangle,
  Trash2,
  Ruler,
  Pencil,
  FileText,
  ChevronDown,
  ChevronRight,
  Thermometer,
  HardHat,
  Package,
  FileDown,
  CheckCircle2,
  XCircle,
  Send,
  ShieldCheck,
  Wind,
  Droplets,
  ClipboardCheck,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useRapporteStore,
  type FieldReport,
  type ReportTemplate,
  type WeatherType,
  type ReportWorker,
  type ReportActivity,
  type ReportMaterial,
} from '@/stores/rapporte'
import SignatureCanvas from './SignatureCanvas'
import SketchCanvas from './SketchCanvas'
import { DetailPanel, EmptyState, PageHeader } from '@/components/shared'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

type TabKey = 'tagesberichte' | 'aufmass' | 'vorlagen'
type DateFilter = 'week' | 'month' | 'all'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const weatherIcons: Record<WeatherType, typeof Sun> = {
  sunny: Sun,
  cloudy: Cloud,
  rainy: CloudRain,
  snowy: Snowflake,
}

const weatherLabelKeys: Record<WeatherType, string> = {
  sunny: 'rapporte.weather.sunny',
  cloudy: 'rapporte.weather.cloudy',
  rainy: 'rapporte.weather.rainy',
  snowy: 'rapporte.weather.snowy',
}

const projectColors: Record<string, string> = {
  'prj-1': 'bg-info-light text-info',
  'prj-2': 'bg-warning-light text-warning',
  'prj-3': 'bg-success-light text-success',
  'prj-4': 'bg-error-light text-error',
  'prj-5': 'bg-primary-light text-primary',
  'prj-6': 'bg-info-light text-info',
  'prj-7': 'bg-warning-light text-warning',
  'prj-8': 'bg-error-light text-error',
}

function formatDate(dateStr: string): string {
  return new Date(dateStr + 'T00:00:00').toLocaleDateString('de-DE', {
    weekday: 'short',
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

function formatDateShort(dateStr: string): string {
  return formatDate(dateStr + 'T00:00:00')
}

function calcNetHours(start: string, end: string, breakMin: number): string {
  const [sh, sm] = start.split(':').map(Number)
  const [eh, em] = end.split(':').map(Number)
  const totalMin = (eh * 60 + em) - (sh * 60 + sm) - breakMin
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  return `${h}h ${m > 0 ? m + 'min' : ''}`
}

function calcNetMinutes(start: string, end: string, breakMin: number): number {
  const [sh, sm] = start.split(':').map(Number)
  const [eh, em] = end.split(':').map(Number)
  return (eh * 60 + em) - (sh * 60 + sm) - breakMin
}

function isThisWeek(dateStr: string): boolean {
  const date = new Date(dateStr + 'T00:00:00')
  const now = new Date()
  const startOfWeek = new Date(now)
  startOfWeek.setDate(now.getDate() - now.getDay() + (now.getDay() === 0 ? -6 : 1))
  startOfWeek.setHours(0, 0, 0, 0)
  const endOfWeek = new Date(startOfWeek)
  endOfWeek.setDate(startOfWeek.getDate() + 7)
  return date >= startOfWeek && date < endOfWeek
}

function isThisMonth(dateStr: string): boolean {
  const date = new Date(dateStr + 'T00:00:00')
  const now = new Date()
  return date.getMonth() === now.getMonth() && date.getFullYear() === now.getFullYear()
}

// Mock weather details by weather type (9.11)
const weatherMockData: Record<WeatherType, { temp: number; wind: number; humidity: number }> = {
  sunny: { temp: 18, wind: 5, humidity: 45 },
  cloudy: { temp: 12, wind: 15, humidity: 65 },
  rainy: { temp: 8, wind: 25, humidity: 85 },
  snowy: { temp: -2, wind: 10, humidity: 75 },
}

// Approval status styling (9.10)
const approvalBadgeStyles: Record<FieldReport['approvalStatus'], string> = {
  draft: 'bg-secondary text-muted-foreground',
  submitted: 'bg-info-light text-info',
  approved: 'bg-success-light text-success',
  rejected: 'bg-error-light text-error',
}

const approvalLabelKeys: Record<FieldReport['approvalStatus'], string> = {
  draft: 'rapporte.approval.draft',
  submitted: 'rapporte.approval.submitted',
  approved: 'rapporte.approval.approved',
  rejected: 'rapporte.approval.rejected',
}

// ============================================================
// Main Component
// ============================================================

export default function RapportePage() {
  const { t } = useTranslation()
  const { reports, measurements, templates, deleteReport, deleteMeasurement, updateReport } = useRapporteStore()

  const [tab, setTab] = useState<TabKey>('tagesberichte')
  const [search, setSearch] = useState('')
  const [projectFilter, setProjectFilter] = useState<string>('all')
  const [dateFilter, setDateFilter] = useState<DateFilter>('all')

  // Detail panel
  const [selectedReport, setSelectedReport] = useState<FieldReport | null>(null)

  // Expanded measurements
  const [expandedMeasurement, setExpandedMeasurement] = useState<string | null>(null)

  // Dialogs
  const [showNewReport, setShowNewReport] = useState(false)
  const [showNewMeasurement, setShowNewMeasurement] = useState(false)
  const [templatePrefill, setTemplatePrefill] = useState<ReportTemplate | null>(null)

  // ---------------------------------------------------------------------------
  // Derived data
  // ---------------------------------------------------------------------------

  const uniqueProjects = useMemo(() => {
    const set = new Map<string, string>()
    reports.forEach((r) => set.set(r.projectId, r.projectName))
    return Array.from(set.entries()).map(([id, name]) => ({ id, name }))
  }, [reports])

  const filteredReports = useMemo(() => {
    let list = reports
    if (projectFilter !== 'all') {
      list = list.filter((r) => r.projectId === projectFilter)
    }
    if (dateFilter === 'week') {
      list = list.filter((r) => isThisWeek(r.date))
    } else if (dateFilter === 'month') {
      list = list.filter((r) => isThisMonth(r.date))
    }
    if (search) {
      const q = search.toLowerCase()
      list = list.filter(
        (r) =>
          r.projectName.toLowerCase().includes(q) ||
          r.author.toLowerCase().includes(q) ||
          r.activities.some((a) => a.description.toLowerCase().includes(q)),
      )
    }
    return list
  }, [reports, projectFilter, dateFilter, search])

  // Stats
  const reportsThisWeek = reports.filter((r) => isThisWeek(r.date)).length
  const totalHours = reports.reduce((sum, r) => sum + calcNetMinutes(r.workStart, r.workEnd, r.breakMinutes), 0)
  const totalHoursFormatted = `${Math.floor(totalHours / 60)}h`
  const activeProjects = new Set(reports.map((r) => r.projectId)).size

  // Mock material cost (for stats display)
  const materialCostMock = 48_750

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------

  const handleDeleteReport = (id: string) => {
    deleteReport(id)
    setSelectedReport(null)
    toast.success(t('rapporte.report.deleted'))
  }

  const handleDeleteMeasurement = (id: string) => {
    deleteMeasurement(id)
    toast.success(t('rapporte.measurement.deleted'))
  }

  const handleUseTemplate = (template: ReportTemplate) => {
    setTemplatePrefill(template)
    setShowNewReport(true)
  }

  const toggleMeasurement = (id: string) => {
    setExpandedMeasurement(expandedMeasurement === id ? null : id)
  }

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="flex-1 overflow-y-auto p-6 animate-fade-up">
      <PageHeader
        title={t('rapporte.title')}
        description={t('rapporte.description')}
        icon={ClipboardCheck}
        moduleId="rapporte"
        className="mb-6"
        actions={
          <div className="flex gap-2">
            {tab === 'aufmass' && (
              <button
                onClick={() => setShowNewMeasurement(true)}
                className="flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
              >
                <Plus className="h-4 w-4" />
                {t('rapporte.newMeasurement')}
              </button>
            )}
            <button
              onClick={() => {
                setTemplatePrefill(null)
                setShowNewReport(true)
              }}
              className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-4 w-4" />
              {t('rapporte.newReport')}
            </button>
          </div>
        }
      />

      {/* Stats Row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <div className="rounded-xl border border-border bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
          <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">{t('rapporte.stats.reportsThisWeek')}</p>
          <p className="text-2xl font-semibold text-foreground tabular-nums">{reportsThisWeek}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
          <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">{t('rapporte.stats.totalHours')}</p>
          <p className="text-2xl font-semibold text-foreground tabular-nums">{totalHoursFormatted}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
          <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">{t('rapporte.stats.materialCost')}</p>
          <p className="text-2xl font-semibold text-foreground tabular-nums">
            CHF {materialCostMock.toLocaleString('de-DE')}
          </p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
          <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">{t('rapporte.stats.activeProjects')}</p>
          <p className="text-2xl font-semibold text-foreground tabular-nums">{activeProjects}</p>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'tagesberichte' as const, label: t('rapporte.tabs.reports', { count: reports.length }) },
          { key: 'aufmass' as const, label: t('rapporte.tabs.measurements', { count: measurements.length }) },
          { key: 'vorlagen' as const, label: t('rapporte.tabs.templates', { count: templates.length }) },
        ]).map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key ? 'border-primary text-primary font-medium tab-accent-active' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* ============================== */}
      {/* Tagesberichte Tab              */}
      {/* ============================== */}
      {tab === 'tagesberichte' && (
        <>
          {/* Filter bar */}
          <div className="flex items-center gap-3 mb-4 flex-wrap">
            <div className="relative flex-1 max-w-sm min-w-[200px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder={t('rapporte.filter.searchPlaceholder')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <Select value={projectFilter} onValueChange={setProjectFilter}>
              <SelectTrigger className="w-[220px]">
                <SelectValue placeholder={t('rapporte.filter.allProjects')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t('rapporte.filter.allProjects')}</SelectItem>
                {uniqueProjects.map((p) => (
                  <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <div className="flex items-center gap-1.5">
              {([
                { value: 'week' as const, label: t('rapporte.filter.thisWeek') },
                { value: 'month' as const, label: t('rapporte.filter.thisMonth') },
                { value: 'all' as const, label: t('rapporte.filter.all') },
              ]).map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => setDateFilter(opt.value)}
                  className={`rounded-lg px-3 py-1.5 text-xs transition-colors ${
                    dateFilter === opt.value
                      ? 'bg-primary text-primary-foreground'
                      : 'border border-border text-muted-foreground hover:bg-secondary'
                  }`}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          {/* Report cards */}
          {filteredReports.length === 0 ? (
            <EmptyState
              icon={ClipboardList}
              title={t('rapporte.empty.noReports')}
              description={search || projectFilter !== 'all' || dateFilter !== 'all' ? t('rapporte.empty.adjustFilters') : t('rapporte.empty.createFirst')}
            />
          ) : (
            <div className="space-y-3">
              {filteredReports.map((report) => {
                const WeatherIcon = weatherIcons[report.weather]
                const netHours = calcNetHours(report.workStart, report.workEnd, report.breakMinutes)
                return (
                  <div
                    key={report.id}
                    onClick={() => setSelectedReport(report)}
                    className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer"
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex items-start gap-3">
                        <div className="flex flex-col items-center justify-center rounded-lg bg-secondary px-3 py-2 min-w-[70px]">
                          <span className="text-lg font-bold text-foreground tabular-nums">
                            {new Date(report.date + 'T00:00:00').getDate()}
                          </span>
                          <span className="text-[10px] text-muted-foreground uppercase">
                            {new Date(report.date + 'T00:00:00').toLocaleDateString('de-DE', { month: 'short' })}
                          </span>
                        </div>
                        <div>
                          <span className={`inline-block rounded-full px-2 py-0.5 text-[10px] font-medium mb-1 ${projectColors[report.projectId] ?? 'bg-secondary text-muted-foreground'}`}>
                            {report.projectName}
                          </span>
                          <p className="text-xs text-muted-foreground">{report.author}</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-3">
                        {/* Approval badge (9.10) */}
                        <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${approvalBadgeStyles[report.approvalStatus]}`}>
                          {report.approvalStatus === 'approved' && <CheckCircle2 className="h-3 w-3" />}
                          {report.approvalStatus === 'rejected' && <XCircle className="h-3 w-3" />}
                          {report.approvalStatus === 'submitted' && <Send className="h-3 w-3" />}
                          {t(approvalLabelKeys[report.approvalStatus])}
                        </span>
                        {/* Weather */}
                        <div className="flex items-center gap-1 text-xs text-muted-foreground">
                          <WeatherIcon className="h-4 w-4" />
                          <span>{report.temperature}°C</span>
                        </div>
                      </div>
                    </div>

                    {/* Info row */}
                    <div className="flex items-center gap-4 text-xs text-muted-foreground flex-wrap">
                      <div className="flex items-center gap-1">
                        <Clock className="h-3.5 w-3.5" />
                        <span>{report.workStart}–{report.workEnd}</span>
                        <span className="text-foreground font-medium ml-0.5">({netHours})</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <Users className="h-3.5 w-3.5" />
                        <span>{t('rapporte.report.workers', { count: report.workers.length })}</span>
                      </div>
                      {report.photos.length > 0 && (
                        <div className="flex items-center gap-1">
                          <Camera className="h-3.5 w-3.5" />
                          <span>{t('rapporte.report.photos', { count: report.photos.length })}</span>
                        </div>
                      )}
                      {report.signatureStatus === 'pending' && (
                        <span className="flex items-center gap-1 rounded-full bg-warning-light text-warning px-2 py-0.5 text-[10px] font-medium">
                          <AlertTriangle className="h-3 w-3" />
                          {t('rapporte.report.signaturePending')}
                        </span>
                      )}
                    </div>

                    {/* Activities preview */}
                    <div className="mt-2 text-xs text-muted-foreground line-clamp-2">
                      {report.activities.slice(0, 2).map((a) => a.description).join(' / ')}
                      {report.activities.length > 2 && ` ${t('rapporte.report.more', { count: report.activities.length - 2 })}`}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </>
      )}

      {/* ============================== */}
      {/* Aufmass Tab                    */}
      {/* ============================== */}
      {tab === 'aufmass' && (
        <>
          {measurements.length === 0 ? (
            <EmptyState
              icon={Ruler}
              title={t('rapporte.empty.noMeasurements')}
              description={t('rapporte.empty.createMeasurement')}
            />
          ) : (
            <div className="space-y-3">
              {measurements.map((m) => {
                const totalArea = m.positions.reduce((s, p) => s + p.area, 0)
                const totalVolume = m.positions.reduce((s, p) => s + p.volume, 0)
                const isExpanded = expandedMeasurement === m.id
                return (
                  <div
                    key={m.id}
                    className="rounded-lg border border-border bg-card overflow-hidden transition-shadow hover:shadow-[var(--shadow-card-hover)]"
                  >
                    <button
                      onClick={() => toggleMeasurement(m.id)}
                      className="w-full flex items-center justify-between p-4 text-left"
                    >
                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                          <Ruler className="h-5 w-5 text-primary" />
                        </div>
                        <div>
                          <h4 className="text-sm font-medium text-foreground">{m.name}</h4>
                          <div className="flex items-center gap-2 mt-0.5 text-xs text-muted-foreground flex-wrap">
                            <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px]">
                              {m.projectName}
                            </span>
                            <span>{m.positions.length} Position{m.positions.length !== 1 ? 'en' : ''}</span>
                            <span>&middot; {totalArea.toFixed(2)} m²</span>
                            <span>&middot; {totalVolume.toFixed(2)} m³</span>
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-3">
                        <div className="text-right text-xs text-muted-foreground hidden sm:block">
                          <p>{m.author}</p>
                          <p>{formatDateShort(m.createdAt)}</p>
                        </div>
                        {isExpanded ? (
                          <ChevronDown className="h-4 w-4 text-muted-foreground" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-muted-foreground" />
                        )}
                      </div>
                    </button>

                    {isExpanded && (
                      <div className="border-t border-border-muted px-4 pb-4">
                        <table className="w-full text-sm mt-3">
                          <thead>
                            <tr>
                              <th className="pb-2 text-left text-xs font-medium text-muted-foreground">{t('rapporte.table.position')}</th>
                              <th className="pb-2 text-right text-xs font-medium text-muted-foreground">{t('rapporte.table.length')}</th>
                              <th className="pb-2 text-right text-xs font-medium text-muted-foreground">{t('rapporte.table.width')}</th>
                              <th className="pb-2 text-right text-xs font-medium text-muted-foreground">{t('rapporte.table.height')}</th>
                              <th className="pb-2 text-right text-xs font-medium text-muted-foreground">{t('rapporte.table.area')}</th>
                              <th className="pb-2 text-right text-xs font-medium text-muted-foreground">{t('rapporte.table.volume')}</th>
                            </tr>
                          </thead>
                          <tbody>
                            {m.positions.map((pos) => (
                              <tr key={pos.id} className="border-t border-border-muted">
                                <td className="py-2 text-xs text-foreground">{pos.label}</td>
                                <td className="py-2 text-xs text-muted-foreground text-right tabular-nums">{pos.length.toFixed(2)}</td>
                                <td className="py-2 text-xs text-muted-foreground text-right tabular-nums">{pos.width.toFixed(2)}</td>
                                <td className="py-2 text-xs text-muted-foreground text-right tabular-nums">{pos.height.toFixed(2)}</td>
                                <td className="py-2 text-xs text-foreground text-right font-medium tabular-nums">{pos.area.toFixed(2)}</td>
                                <td className="py-2 text-xs text-foreground text-right font-medium tabular-nums">{pos.volume.toFixed(2)}</td>
                              </tr>
                            ))}
                            {/* Sum row */}
                            <tr className="border-t-2 border-border">
                              <td className="py-2 text-xs font-semibold text-foreground">{t('rapporte.table.sum')}</td>
                              <td className="py-2" />
                              <td className="py-2" />
                              <td className="py-2" />
                              <td className="py-2 text-xs font-semibold text-primary text-right tabular-nums">{totalArea.toFixed(2)}</td>
                              <td className="py-2 text-xs font-semibold text-primary text-right tabular-nums">{totalVolume.toFixed(2)}</td>
                            </tr>
                          </tbody>
                        </table>

                        {/* Sketch canvas (9.8) */}
                        <div className="mt-4 rounded-lg border border-border overflow-hidden" style={{ height: 400 }}>
                          <SketchCanvas />
                        </div>

                        {/* Delete button */}
                        <div className="mt-3 flex justify-end">
                          <button
                            onClick={(e) => {
                              e.stopPropagation()
                              handleDeleteMeasurement(m.id)
                            }}
                            className="flex items-center gap-1 rounded-lg px-2 py-1 text-xs text-muted-foreground hover:text-error hover:bg-error-light transition-colors"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                            {t('common.delete')}
                          </button>
                        </div>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </>
      )}

      {/* ============================== */}
      {/* Vorlagen Tab                   */}
      {/* ============================== */}
      {tab === 'vorlagen' && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {templates.map((tpl) => (
            <div
              key={tpl.id}
              className="rounded-lg border border-border bg-card p-4 flex flex-col"
            >
              <div className="flex items-start gap-3 mb-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light shrink-0">
                  <FileText className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <h4 className="text-sm font-medium text-foreground">{tpl.name}</h4>
                  <p className="text-xs text-muted-foreground mt-0.5">{tpl.description}</p>
                </div>
              </div>

              {/* Default activities */}
              <div className="mb-3">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5">{t('rapporte.template.activities')}</p>
                <div className="flex flex-wrap gap-1">
                  {tpl.defaultActivities.map((a) => (
                    <span key={a} className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
                      {a}
                    </span>
                  ))}
                </div>
              </div>

              {/* Default materials */}
              <div className="mb-4">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5">{t('rapporte.template.materials')}</p>
                <div className="flex flex-wrap gap-1">
                  {tpl.defaultMaterials.map((m) => (
                    <span key={m} className="rounded-full bg-primary-light px-2 py-0.5 text-[10px] text-primary">
                      {m}
                    </span>
                  ))}
                </div>
              </div>

              <div className="mt-auto">
                <button
                  onClick={() => handleUseTemplate(tpl)}
                  className="w-full flex items-center justify-center gap-2 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
                >
                  <Plus className="h-3.5 w-3.5" />
                  {t('rapporte.template.use')}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* ============================== */}
      {/* Report Detail Panel            */}
      {/* ============================== */}
      {selectedReport && (
        <ReportDetailPanel
          report={selectedReport}
          onClose={() => setSelectedReport(null)}
          onDelete={handleDeleteReport}
          onUpdate={(id, updates) => {
            updateReport(id, updates)
            setSelectedReport({ ...selectedReport, ...updates })
          }}
        />
      )}

      {/* ============================== */}
      {/* Neuer Tagesbericht Dialog      */}
      {/* ============================== */}
      <NewReportDialog
        open={showNewReport}
        onOpenChange={setShowNewReport}
        projects={uniqueProjects}
        templatePrefill={templatePrefill}
      />

      {/* ============================== */}
      {/* Neues Aufmass Dialog           */}
      {/* ============================== */}
      <NewMeasurementDialog
        open={showNewMeasurement}
        onOpenChange={setShowNewMeasurement}
        projects={uniqueProjects}
      />
    </div>
  )
}

// ============================================================
// Report Detail Panel
// ============================================================

function ReportDetailPanel({
  report,
  onClose,
  onDelete,
  onUpdate,
}: {
  report: FieldReport
  onClose: () => void
  onDelete: (id: string) => void
  onUpdate: (id: string, updates: Partial<FieldReport>) => void
}) {
  const { t } = useTranslation()
  const WeatherIcon = weatherIcons[report.weather]
  const netHours = calcNetHours(report.workStart, report.workEnd, report.breakMinutes)
  const [showSignaturePad, setShowSignaturePad] = useState(false)

  return (
    <DetailPanel
      open={true}
      title={formatDate(report.date)}
      subtitle={report.projectName}
      onClose={onClose}
      width="w-[480px]"
      footer={
        <div className="flex items-center gap-2">
          <button
            onClick={() => onDelete(report.id)}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:text-error hover:bg-error-light transition-colors"
          >
            <Trash2 className="h-3.5 w-3.5" />
            {t('common.delete')}
          </button>
          <button
            onClick={() => toast.success(t('rapporte.detail.pdfExportToast'), { description: t('rapporte.detail.pdfExportDescription') })}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <FileDown className="h-3.5 w-3.5" />
            {t('rapporte.detail.pdfExport')}
          </button>
        </div>
      }
    >
      <div className="space-y-5">
        {/* Header info */}
        <div className="flex items-start justify-between">
          <div>
            <span className={`inline-block rounded-full px-2.5 py-0.5 text-[10px] font-medium mb-1 ${projectColors[report.projectId] ?? 'bg-secondary text-muted-foreground'}`}>
              {report.projectName}
            </span>
            <p className="text-xs text-muted-foreground">{t('rapporte.detail.author', { name: report.author })}</p>
          </div>
          {report.signatureStatus === 'pending' && (
            <span className="flex items-center gap-1 rounded-full bg-warning-light text-warning px-2 py-0.5 text-[10px] font-medium">
              <AlertTriangle className="h-3 w-3" />
              {t('rapporte.detail.pending')}
            </span>
          )}
          {report.signatureStatus === 'signed' && (
            <span className="flex items-center gap-1 rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium">
              {t('rapporte.detail.signed')}
            </span>
          )}
        </div>

        {/* Weather block */}
        <div className="rounded-lg border border-border p-3 flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-secondary">
            <WeatherIcon className="h-5 w-5 text-foreground" />
          </div>
          <div>
            <p className="text-sm font-medium text-foreground">{t(weatherLabelKeys[report.weather])}</p>
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Thermometer className="h-3 w-3" />
              {report.temperature}°C
            </p>
          </div>
        </div>

        {/* Work time block */}
        <div className="grid grid-cols-4 gap-2">
          <div className="rounded-lg border border-border bg-secondary/30 p-2.5 text-center">
            <p className="text-[10px] text-muted-foreground mb-0.5">{t('rapporte.detail.start')}</p>
            <p className="text-sm font-medium text-foreground">{report.workStart}</p>
          </div>
          <div className="rounded-lg border border-border bg-secondary/30 p-2.5 text-center">
            <p className="text-[10px] text-muted-foreground mb-0.5">{t('rapporte.detail.end')}</p>
            <p className="text-sm font-medium text-foreground">{report.workEnd}</p>
          </div>
          <div className="rounded-lg border border-border bg-secondary/30 p-2.5 text-center">
            <p className="text-[10px] text-muted-foreground mb-0.5">{t('rapporte.detail.break')}</p>
            <p className="text-sm font-medium text-foreground">{report.breakMinutes} min</p>
          </div>
          <div className="rounded-lg border border-border bg-primary-light p-2.5 text-center">
            <p className="text-[10px] text-primary mb-0.5">{t('rapporte.detail.net')}</p>
            <p className="text-sm font-semibold text-primary">{netHours}</p>
          </div>
        </div>

        {/* Workers table */}
        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
            <HardHat className="h-3 w-3" />
            {t('rapporte.detail.workersTitle', { count: report.workers.length })}
          </h4>
          <div className="rounded-lg border border-border overflow-hidden">
            <table className="w-full text-xs">
              <thead>
                <tr className="bg-secondary/30">
                  <th className="px-3 py-2 text-left font-medium text-muted-foreground">{t('rapporte.detail.tableName')}</th>
                  <th className="px-3 py-2 text-left font-medium text-muted-foreground">{t('rapporte.detail.tableRole')}</th>
                  <th className="px-3 py-2 text-right font-medium text-muted-foreground">{t('rapporte.detail.tableHours')}</th>
                </tr>
              </thead>
              <tbody>
                {report.workers.map((w, i) => (
                  <tr key={i} className="border-t border-border-muted">
                    <td className="px-3 py-2 text-foreground">{w.name}</td>
                    <td className="px-3 py-2 text-muted-foreground">{w.role}</td>
                    <td className="px-3 py-2 text-foreground text-right tabular-nums">{w.hours}h</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Activities */}
        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
            {t('rapporte.detail.activitiesTitle', { count: report.activities.length })}
          </h4>
          <div className="space-y-1.5">
            {report.activities.map((a, i) => (
              <div key={i} className="flex items-start gap-2 rounded-md border border-border-muted px-3 py-2">
                <div className="flex-1">
                  <p className="text-xs text-foreground">{a.description}</p>
                </div>
                <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground shrink-0">
                  {a.category}
                </span>
              </div>
            ))}
          </div>
        </section>

        {/* Materials table */}
        {report.materials.length > 0 && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
              <Package className="h-3 w-3" />
              {t('rapporte.detail.materialTitle', { count: report.materials.length })}
            </h4>
            <div className="rounded-lg border border-border overflow-hidden">
              <table className="w-full text-xs">
                <thead>
                  <tr className="bg-secondary/30">
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">{t('rapporte.detail.tableArticle')}</th>
                    <th className="px-3 py-2 text-right font-medium text-muted-foreground">{t('rapporte.detail.tableQuantity')}</th>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground pl-3">{t('rapporte.detail.tableUnit')}</th>
                  </tr>
                </thead>
                <tbody>
                  {report.materials.map((m, i) => (
                    <tr key={i} className="border-t border-border-muted">
                      <td className="px-3 py-2 text-foreground">{m.article}</td>
                      <td className="px-3 py-2 text-muted-foreground text-right tabular-nums">{m.quantity}</td>
                      <td className="px-3 py-2 text-muted-foreground pl-3">{m.unit}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        )}

        {/* Photos (improved 9.7) */}
        {report.photos.length > 0 && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
              <Camera className="h-3 w-3" />
              {t('rapporte.detail.photosTitle', { count: report.photos.length })}
            </h4>
            <div className="grid grid-cols-2 gap-2">
              {report.photos.map((photo, idx) => (
                <div key={photo.id} className="rounded-lg border border-border bg-secondary/30 overflow-hidden">
                  <div className="relative aspect-[4/3] bg-secondary flex items-center justify-center">
                    <Camera className="h-8 w-8 text-muted-foreground opacity-20" />
                    <div className="absolute top-2 left-2 flex h-6 w-6 items-center justify-center rounded-full bg-foreground/70 text-[10px] font-bold text-background">
                      {idx + 1}
                    </div>
                  </div>
                  <div className="px-2 py-1.5">
                    <p className="text-[10px] text-muted-foreground line-clamp-2">{photo.caption || t('rapporte.detail.noCaption')}</p>
                  </div>
                </div>
              ))}
            </div>
          </section>
        )}

        {/* Signature section (9.6) */}
        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
            {t('rapporte.detail.signatureTitle')}
          </h4>
          {report.signatureDataUrl ? (
            <div className="rounded-lg border border-border bg-white p-3">
              <img
                src={report.signatureDataUrl}
                alt={t('rapporte.detail.signatureAlt')}
                loading="lazy"
                decoding="async"
                className="max-h-24 mx-auto"
              />
              <p className="text-[10px] text-success text-center mt-1 flex items-center justify-center gap-1">
                <CheckCircle2 className="h-3 w-3" />
                {t('rapporte.detail.signatureSigned')}
              </p>
            </div>
          ) : showSignaturePad ? (
            <div className="rounded-lg border border-border p-3 space-y-2">
              <SignatureCanvas
                onSave={(dataUrl) => {
                  onUpdate(report.id, {
                    signatureDataUrl: dataUrl,
                    signatureStatus: 'signed',
                  })
                  setShowSignaturePad(false)
                  toast.success(t('rapporte.detail.signatureSaved'))
                }}
                onCancel={() => setShowSignaturePad(false)}
              />
            </div>
          ) : (
            <button
              onClick={() => setShowSignaturePad(true)}
              className="w-full rounded-lg border-2 border-dashed border-border p-4 flex items-center justify-center gap-2 text-muted-foreground hover:bg-secondary/50 hover:border-primary/30 transition-colors"
            >
              <Pencil className="h-4 w-4" />
              <span className="text-xs font-medium">{t('rapporte.detail.signatureCapture')}</span>
            </button>
          )}
        </section>

        {/* Approval workflow (9.10) */}
        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
            <ShieldCheck className="h-3 w-3" />
            {t('rapporte.detail.approvalTitle')}
          </h4>
          {report.approvalStatus === 'draft' && (
            <button
              onClick={() => {
                onUpdate(report.id, { approvalStatus: 'submitted' })
                toast.success(t('rapporte.detail.submitted'))
              }}
              className="w-full flex items-center justify-center gap-2 rounded-lg border border-primary bg-primary-light px-3 py-2.5 text-xs font-medium text-primary hover:bg-primary hover:text-primary-foreground transition-colors"
            >
              <Send className="h-3.5 w-3.5" />
              {t('rapporte.detail.submitApproval')}
            </button>
          )}
          {report.approvalStatus === 'submitted' && (
            <div className="rounded-lg border border-info bg-info-light p-3 flex items-center gap-2">
              <Clock className="h-4 w-4 text-info shrink-0" />
              <div>
                <p className="text-xs font-medium text-info">{t('rapporte.detail.waitingApproval')}</p>
                <p className="text-[10px] text-info/70 mt-0.5">{t('rapporte.detail.waitingDescription')}</p>
              </div>
            </div>
          )}
          {report.approvalStatus === 'approved' && (
            <div className="rounded-lg border border-success bg-success-light p-3 flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-success shrink-0" />
              <div>
                <p className="text-xs font-medium text-success">{t('rapporte.detail.approved')}</p>
                {report.approvedBy && (
                  <p className="text-[10px] text-success/70 mt-0.5">{t('rapporte.detail.approvedBy', { name: report.approvedBy })}</p>
                )}
              </div>
            </div>
          )}
          {report.approvalStatus === 'rejected' && (
            <div className="space-y-2">
              <div className="rounded-lg border border-error bg-error-light p-3 flex items-start gap-2">
                <XCircle className="h-4 w-4 text-error shrink-0 mt-0.5" />
                <div>
                  <p className="text-xs font-medium text-error">{t('rapporte.detail.rejected')}</p>
                  {report.approvalComment && (
                    <p className="text-[10px] text-error/80 mt-0.5">{t('rapporte.detail.rejectedReason', { reason: report.approvalComment })}</p>
                  )}
                  {report.approvedBy && (
                    <p className="text-[10px] text-error/60 mt-0.5">{t('rapporte.detail.rejectedBy', { name: report.approvedBy })}</p>
                  )}
                </div>
              </div>
              <button
                onClick={() => {
                  onUpdate(report.id, { approvalStatus: 'submitted', approvalComment: undefined })
                  toast.success(t('rapporte.detail.resubmitted'))
                }}
                className="w-full flex items-center justify-center gap-2 rounded-lg border border-border px-3 py-2 text-xs text-foreground hover:bg-secondary transition-colors"
              >
                <Send className="h-3.5 w-3.5" />
                {t('rapporte.detail.resubmit')}
              </button>
            </div>
          )}
        </section>

        {/* Notes */}
        {report.notes && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              {t('rapporte.detail.notesTitle')}
            </h4>
            <div className="rounded-lg border border-border bg-secondary/30 p-3">
              <p className="text-xs text-foreground leading-relaxed">{report.notes}</p>
            </div>
          </section>
        )}
      </div>
    </DetailPanel>
  )
}

// ============================================================
// Neuer Tagesbericht Dialog
// ============================================================

function NewReportDialog({
  open,
  onOpenChange,
  projects,
  templatePrefill,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  projects: { id: string; name: string }[]
  templatePrefill: ReportTemplate | null
}) {
  const { t } = useTranslation()
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const [projectId, setProjectId] = useState('')
  const [weather, setWeather] = useState<WeatherType>('sunny')
  const [temperature, setTemperature] = useState('10')
  const [workStart, setWorkStart] = useState('07:00')
  const [workEnd, setWorkEnd] = useState('17:00')
  const [breakMin, setBreakMin] = useState('60')
  const [workers, setWorkers] = useState<ReportWorker[]>([
    { name: '', role: '', hours: 8 },
  ])
  const [activities, setActivities] = useState<ReportActivity[]>(
    templatePrefill
      ? templatePrefill.defaultActivities.map((a) => ({ description: '', category: a }))
      : [{ description: '', category: '' }],
  )
  const [materials, setMaterials] = useState<ReportMaterial[]>(
    templatePrefill
      ? templatePrefill.defaultMaterials.map((m) => ({ article: m, quantity: 0, unit: 'Stk' }))
      : [{ article: '', quantity: 0, unit: 'Stk' }],
  )
  const [notes, setNotes] = useState('')
  const [photos, setPhotos] = useState<{ id: string; caption: string }[]>([])
  const [signatureDataUrl, setSignatureDataUrl] = useState<string | null>(null)

  const weatherTypes: WeatherType[] = ['sunny', 'cloudy', 'rainy', 'snowy']
  const currentWeatherMock = weatherMockData[weather]

  // Photos (9.7)
  const addPhoto = () => setPhotos([...photos, { id: `ph-${Date.now()}`, caption: '' }])
  const removePhoto = (id: string) => setPhotos(photos.filter((p) => p.id !== id))
  const updatePhotoCaption = (id: string, caption: string) =>
    setPhotos(photos.map((p) => (p.id === id ? { ...p, caption } : p)))

  // Workers
  const addWorker = () => setWorkers([...workers, { name: '', role: '', hours: 8 }])
  const removeWorker = (idx: number) => {
    if (workers.length <= 1) return
    setWorkers(workers.filter((_, i) => i !== idx))
  }
  const updateWorker = (idx: number, field: keyof ReportWorker, value: string | number) => {
    setWorkers(workers.map((w, i) => (i === idx ? { ...w, [field]: value } : w)))
  }

  // Activities
  const addActivity = () => setActivities([...activities, { description: '', category: '' }])
  const removeActivity = (idx: number) => {
    if (activities.length <= 1) return
    setActivities(activities.filter((_, i) => i !== idx))
  }
  const updateActivity = (idx: number, field: keyof ReportActivity, value: string) => {
    setActivities(activities.map((a, i) => (i === idx ? { ...a, [field]: value } : a)))
  }

  // Materials
  const addMaterial = () => setMaterials([...materials, { article: '', quantity: 0, unit: 'Stk' }])
  const removeMaterial = (idx: number) => {
    if (materials.length <= 1) return
    setMaterials(materials.filter((_, i) => i !== idx))
  }
  const updateMaterial = (idx: number, field: keyof ReportMaterial, value: string | number) => {
    setMaterials(materials.map((m, i) => (i === idx ? { ...m, [field]: value } : m)))
  }

  const handleSave = () => {
    if (!projectId) {
      toast.error(t('rapporte.dialog.selectProject'))
      return
    }
    const validWorkers = workers.filter((w) => w.name.trim())
    if (validWorkers.length === 0) {
      toast.error(t('rapporte.dialog.workerRequired'))
      return
    }
    const validActivities = activities.filter((a) => a.description.trim())
    if (validActivities.length === 0) {
      toast.error(t('rapporte.dialog.activityRequired'))
      return
    }

    const project = projects.find((p) => p.id === projectId)
    toast.success(t('rapporte.dialog.reportCreated', { name: project?.name ?? 'Projekt' }))

    // Reset
    setDate(new Date().toISOString().split('T')[0])
    setProjectId('')
    setWeather('sunny')
    setTemperature('10')
    setWorkStart('07:00')
    setWorkEnd('17:00')
    setBreakMin('60')
    setWorkers([{ name: '', role: '', hours: 8 }])
    setActivities([{ description: '', category: '' }])
    setMaterials([{ article: '', quantity: 0, unit: 'Stk' }])
    setNotes('')
    setPhotos([])
    setSignatureDataUrl(null)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[620px] max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {templatePrefill ? t('rapporte.dialog.newReportTemplate', { name: templatePrefill.name }) : t('rapporte.dialog.newReportTitle')}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4 mt-2">
          {/* Date + Project */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('rapporte.dialog.date')}</Label>
              <Input
                type="date"
                value={date}
                onChange={(e) => setDate(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('rapporte.dialog.project')} <span className="text-destructive">*</span></Label>
              <Select value={projectId} onValueChange={setProjectId}>
                <SelectTrigger>
                  <SelectValue placeholder={t('rapporte.dialog.projectPlaceholder')} />
                </SelectTrigger>
                <SelectContent>
                  {projects.map((p) => (
                    <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Weather */}
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('rapporte.dialog.weather')}</Label>
            <div className="flex items-center gap-2">
              {weatherTypes.map((w) => {
                const Icon = weatherIcons[w]
                return (
                  <button
                    key={w}
                    type="button"
                    onClick={() => setWeather(w)}
                    className={`flex items-center gap-1.5 rounded-lg border px-3 py-2 text-xs transition-colors ${
                      weather === w
                        ? 'border-primary bg-primary-light text-primary font-medium'
                        : 'border-border text-muted-foreground hover:bg-secondary'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    {t(weatherLabelKeys[w])}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Temperature */}
          <div className="space-y-1.5 max-w-[160px]">
            <Label className="text-sm font-medium">{t('rapporte.dialog.temperature')}</Label>
            <Input
              type="number"
              value={temperature}
              onChange={(e) => setTemperature(e.target.value)}
            />
          </div>

          {/* Weather mock data (9.11) */}
          <div className="rounded-lg border border-border bg-secondary/30 p-3">
            <div className="flex items-center gap-4 text-xs text-muted-foreground">
              <div className="flex items-center gap-1">
                <Thermometer className="h-3.5 w-3.5" />
                <span>{currentWeatherMock.temp}°C</span>
              </div>
              <div className="flex items-center gap-1">
                <Wind className="h-3.5 w-3.5" />
                <span>{currentWeatherMock.wind} km/h</span>
              </div>
              <div className="flex items-center gap-1">
                <Droplets className="h-3.5 w-3.5" />
                <span>{currentWeatherMock.humidity}%</span>
              </div>
            </div>
            <p className="text-[10px] text-muted-foreground/60 mt-1">
              {t('rapporte.weather.dataFor', { date: date || 'heute' })}
            </p>
          </div>

          {/* Work time */}
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('rapporte.dialog.workStart')}</Label>
              <Input
                type="time"
                value={workStart}
                onChange={(e) => setWorkStart(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('rapporte.dialog.workEnd')}</Label>
              <Input
                type="time"
                value={workEnd}
                onChange={(e) => setWorkEnd(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('rapporte.dialog.breakMin')}</Label>
              <Input
                type="number"
                min="0"
                value={breakMin}
                onChange={(e) => setBreakMin(e.target.value)}
              />
            </div>
          </div>

          {/* Workers */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <Label className="text-sm font-medium">{t('rapporte.dialog.workers')} <span className="text-destructive">*</span></Label>
              <button
                onClick={addWorker}
                className="flex items-center gap-1 rounded-lg bg-primary px-2.5 py-1 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Plus className="h-3 w-3" />
                {t('rapporte.dialog.addWorker')}
              </button>
            </div>
            <div className="space-y-2 max-h-[200px] overflow-y-auto">
              {workers.map((w, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground w-5 shrink-0 text-right">{idx + 1}.</span>
                  <Input
                    placeholder="Name"
                    value={w.name}
                    onChange={(e) => updateWorker(idx, 'name', e.target.value)}
                    className="flex-1"
                  />
                  <Input
                    placeholder="Funktion"
                    value={w.role}
                    onChange={(e) => updateWorker(idx, 'role', e.target.value)}
                    className="w-28"
                  />
                  <Input
                    type="number"
                    min="0"
                    step="0.5"
                    value={w.hours}
                    onChange={(e) => updateWorker(idx, 'hours', parseFloat(e.target.value) || 0)}
                    className="w-16"
                  />
                  <span className="text-xs text-muted-foreground">h</span>
                  <button
                    onClick={() => removeWorker(idx)}
                    disabled={workers.length <= 1}
                    className="rounded-lg p-1.5 text-muted-foreground hover:text-error hover:bg-error-light transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          </div>

          {/* Activities */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <Label className="text-sm font-medium">{t('rapporte.dialog.activities')} <span className="text-destructive">*</span></Label>
              <button
                onClick={addActivity}
                className="flex items-center gap-1 rounded-lg bg-primary px-2.5 py-1 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Plus className="h-3 w-3" />
                {t('rapporte.dialog.addActivity')}
              </button>
            </div>
            <div className="space-y-2 max-h-[200px] overflow-y-auto">
              {activities.map((a, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground w-5 shrink-0 text-right">{idx + 1}.</span>
                  <Input
                    placeholder="Beschreibung"
                    value={a.description}
                    onChange={(e) => updateActivity(idx, 'description', e.target.value)}
                    className="flex-1"
                  />
                  <Input
                    placeholder="Kategorie"
                    value={a.category}
                    onChange={(e) => updateActivity(idx, 'category', e.target.value)}
                    className="w-32"
                  />
                  <button
                    onClick={() => removeActivity(idx)}
                    disabled={activities.length <= 1}
                    className="rounded-lg p-1.5 text-muted-foreground hover:text-error hover:bg-error-light transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          </div>

          {/* Materials */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <Label className="text-sm font-medium">{t('rapporte.dialog.material')}</Label>
              <button
                onClick={addMaterial}
                className="flex items-center gap-1 rounded-lg bg-primary px-2.5 py-1 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Plus className="h-3 w-3" />
                {t('rapporte.dialog.addMaterial')}
              </button>
            </div>
            <div className="space-y-2 max-h-[200px] overflow-y-auto">
              {materials.map((m, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground w-5 shrink-0 text-right">{idx + 1}.</span>
                  <Input
                    placeholder="Artikel"
                    value={m.article}
                    onChange={(e) => updateMaterial(idx, 'article', e.target.value)}
                    className="flex-1"
                  />
                  <Input
                    type="number"
                    min="0"
                    step="0.1"
                    placeholder="Menge"
                    value={m.quantity || ''}
                    onChange={(e) => updateMaterial(idx, 'quantity', parseFloat(e.target.value) || 0)}
                    className="w-20"
                  />
                  <Select
                    value={m.unit}
                    onValueChange={(v) => updateMaterial(idx, 'unit', v)}
                  >
                    <SelectTrigger className="w-20">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Stk">Stk</SelectItem>
                      <SelectItem value="m">m</SelectItem>
                      <SelectItem value="m²">m²</SelectItem>
                      <SelectItem value="m³">m³</SelectItem>
                      <SelectItem value="kg">kg</SelectItem>
                      <SelectItem value="t">t</SelectItem>
                      <SelectItem value="Liter">Liter</SelectItem>
                      <SelectItem value="Sack">Sack</SelectItem>
                      <SelectItem value="lfm">lfm</SelectItem>
                      <SelectItem value="Set">Set</SelectItem>
                    </SelectContent>
                  </Select>
                  <button
                    onClick={() => removeMaterial(idx)}
                    disabled={materials.length <= 1}
                    className="rounded-lg p-1.5 text-muted-foreground hover:text-error hover:bg-error-light transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('rapporte.dialog.notes')}</Label>
            <Textarea
              placeholder={t('rapporte.dialog.notesPlaceholder')}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
            />
          </div>

          {/* Photos (9.7) */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <Label className="text-sm font-medium">{t('rapporte.dialog.photos')}</Label>
              <button
                type="button"
                onClick={addPhoto}
                className="flex items-center gap-1 rounded-lg bg-primary px-2.5 py-1 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Camera className="h-3 w-3" />
                {t('rapporte.dialog.addPhotos')}
              </button>
            </div>
            {photos.length > 0 && (
              <div className="grid grid-cols-2 gap-2">
                {photos.map((photo) => (
                  <div key={photo.id} className="rounded-lg border border-border bg-card overflow-hidden">
                    <div className="aspect-[4/3] bg-secondary flex items-center justify-center">
                      <Camera className="h-8 w-8 text-muted-foreground opacity-20" />
                    </div>
                    <div className="p-2 space-y-1">
                      <Input
                        placeholder={t('rapporte.dialog.captionPlaceholder')}
                        value={photo.caption}
                        onChange={(e) => updatePhotoCaption(photo.id, e.target.value)}
                        className="text-xs h-7"
                      />
                      <button
                        type="button"
                        onClick={() => removePhoto(photo.id)}
                        className="flex items-center gap-1 text-[10px] text-muted-foreground hover:text-error transition-colors"
                      >
                        <Trash2 className="h-3 w-3" />
                        {t('rapporte.dialog.removePhoto')}
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
            {photos.length === 0 && (
              <p className="text-xs text-muted-foreground">{t('rapporte.dialog.noPhotos')}</p>
            )}
          </div>

          {/* Signature (9.6) */}
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('rapporte.dialog.signature')}</Label>
            {signatureDataUrl ? (
              <div className="rounded-lg border border-border bg-white p-3">
                <img src={signatureDataUrl} alt="Unterschrift" loading="lazy" decoding="async" className="max-h-20 mx-auto" />
                <div className="flex items-center justify-center gap-2 mt-2">
                  <p className="text-[10px] text-success flex items-center gap-1">
                    <CheckCircle2 className="h-3 w-3" />
                    {t('rapporte.dialog.signatureCaptured')}
                  </p>
                  <button
                    type="button"
                    onClick={() => setSignatureDataUrl(null)}
                    className="text-[10px] text-muted-foreground hover:text-error transition-colors"
                  >
                    {t('rapporte.dialog.removeSignature')}
                  </button>
                </div>
              </div>
            ) : (
              <div className="rounded-lg border border-border p-3">
                <SignatureCanvas
                  onSave={(dataUrl) => {
                    setSignatureDataUrl(dataUrl)
                    toast.success(t('rapporte.dialog.signatureCapturedToast'))
                  }}
                  onCancel={() => {}}
                />
              </div>
            )}
          </div>

          {/* Save/Cancel */}
          <div className="flex gap-2 pt-2">
            <button
              onClick={() => onOpenChange(false)}
              className="flex-1 rounded-lg border border-border py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleSave}
              className="flex-1 rounded-lg bg-primary py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              {t('rapporte.dialog.createReport')}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ============================================================
// Neues Aufmass Dialog
// ============================================================

interface PositionDraft {
  label: string
  length: string
  width: string
  height: string
}

function NewMeasurementDialog({
  open,
  onOpenChange,
  projects,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  projects: { id: string; name: string }[]
}) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [projectId, setProjectId] = useState('')
  const [positions, setPositions] = useState<PositionDraft[]>([
    { label: '', length: '', width: '', height: '' },
  ])

  const addPosition = () =>
    setPositions([...positions, { label: '', length: '', width: '', height: '' }])

  const removePosition = (idx: number) => {
    if (positions.length <= 1) return
    setPositions(positions.filter((_, i) => i !== idx))
  }

  const updatePosition = (idx: number, field: keyof PositionDraft, value: string) => {
    setPositions(positions.map((p, i) => (i === idx ? { ...p, [field]: value } : p)))
  }

  const calcArea = (p: PositionDraft): number => {
    const l = parseFloat(p.length) || 0
    const w = parseFloat(p.width) || 0
    return l * w
  }

  const calcVolume = (p: PositionDraft): number => {
    const l = parseFloat(p.length) || 0
    const w = parseFloat(p.width) || 0
    const h = parseFloat(p.height) || 0
    return l * w * h
  }

  const totalArea = positions.reduce((s, p) => s + calcArea(p), 0)
  const totalVolume = positions.reduce((s, p) => s + calcVolume(p), 0)

  const handleSave = () => {
    if (!name.trim()) {
      toast.error(t('rapporte.measurement.dialog.nameRequired'))
      return
    }
    if (!projectId) {
      toast.error(t('rapporte.measurement.dialog.projectRequired'))
      return
    }
    const validPositions = positions.filter((p) => p.label.trim())
    if (validPositions.length === 0) {
      toast.error(t('rapporte.measurement.dialog.positionRequired'))
      return
    }

    const project = projects.find((p) => p.id === projectId)
    toast.success(t('rapporte.measurement.dialog.created', { name, project: project?.name ?? 'Projekt' }))

    // Reset
    setName('')
    setProjectId('')
    setPositions([{ label: '', length: '', width: '', height: '' }])
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[620px] max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t('rapporte.measurement.dialog.title')}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 mt-2">
          {/* Name + Project */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('rapporte.measurement.dialog.roomName')} <span className="text-destructive">*</span></Label>
              <Input
                placeholder={t('rapporte.measurement.dialog.roomPlaceholder')}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('rapporte.dialog.project')} <span className="text-destructive">*</span></Label>
              <Select value={projectId} onValueChange={setProjectId}>
                <SelectTrigger>
                  <SelectValue placeholder={t('rapporte.dialog.projectPlaceholder')} />
                </SelectTrigger>
                <SelectContent>
                  {projects.map((p) => (
                    <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Positions */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <Label className="text-sm font-medium">{t('rapporte.measurement.dialog.positions')} <span className="text-destructive">*</span></Label>
              <button
                onClick={addPosition}
                className="flex items-center gap-1 rounded-lg bg-primary px-2.5 py-1 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Plus className="h-3 w-3" />
                {t('rapporte.measurement.dialog.addPosition')}
              </button>
            </div>
            <div className="space-y-3 max-h-[300px] overflow-y-auto">
              {positions.map((pos, idx) => {
                const area = calcArea(pos)
                const volume = calcVolume(pos)
                return (
                  <div key={idx} className="rounded-lg border border-border-muted p-3 space-y-2">
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-muted-foreground w-5 shrink-0 text-right font-medium">{idx + 1}.</span>
                      <Input
                        placeholder={t('rapporte.measurement.dialog.labelPlaceholder')}
                        value={pos.label}
                        onChange={(e) => updatePosition(idx, 'label', e.target.value)}
                        className="flex-1"
                      />
                      <button
                        onClick={() => removePosition(idx)}
                        disabled={positions.length <= 1}
                        className="rounded-lg p-1.5 text-muted-foreground hover:text-error hover:bg-error-light transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                    <div className="grid grid-cols-5 gap-2 pl-7">
                      <div className="space-y-0.5">
                        <label className="text-[10px] text-muted-foreground">{t('rapporte.measurement.dialog.lengthLabel')}</label>
                        <Input
                          type="number"
                          min="0"
                          step="0.01"
                          placeholder="0.00"
                          value={pos.length}
                          onChange={(e) => updatePosition(idx, 'length', e.target.value)}
                        />
                      </div>
                      <div className="space-y-0.5">
                        <label className="text-[10px] text-muted-foreground">{t('rapporte.measurement.dialog.widthLabel')}</label>
                        <Input
                          type="number"
                          min="0"
                          step="0.01"
                          placeholder="0.00"
                          value={pos.width}
                          onChange={(e) => updatePosition(idx, 'width', e.target.value)}
                        />
                      </div>
                      <div className="space-y-0.5">
                        <label className="text-[10px] text-muted-foreground">{t('rapporte.measurement.dialog.heightLabel')}</label>
                        <Input
                          type="number"
                          min="0"
                          step="0.01"
                          placeholder="0.00"
                          value={pos.height}
                          onChange={(e) => updatePosition(idx, 'height', e.target.value)}
                        />
                      </div>
                      <div className="space-y-0.5">
                        <label className="text-[10px] text-muted-foreground">{t('rapporte.measurement.dialog.areaLabel')}</label>
                        <div className="rounded-lg border border-border bg-secondary/30 px-3 py-[7px] text-sm text-foreground tabular-nums">
                          {area.toFixed(2)}
                        </div>
                      </div>
                      <div className="space-y-0.5">
                        <label className="text-[10px] text-muted-foreground">{t('rapporte.measurement.dialog.volumeLabel')}</label>
                        <div className="rounded-lg border border-border bg-secondary/30 px-3 py-[7px] text-sm text-foreground tabular-nums">
                          {volume.toFixed(2)}
                        </div>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>

            {/* Live totals */}
            <div className="flex items-center justify-end gap-4 mt-3 text-xs">
              <span className="text-muted-foreground">
                {t('rapporte.measurement.dialog.totalArea')} <span className="text-foreground font-semibold tabular-nums">{totalArea.toFixed(2)} m²</span>
              </span>
              <span className="text-muted-foreground">
                {t('rapporte.measurement.dialog.totalVolume')} <span className="text-foreground font-semibold tabular-nums">{totalVolume.toFixed(2)} m³</span>
              </span>
            </div>
          </div>

          {/* Sketch canvas (9.8) */}
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('rapporte.measurement.dialog.sketch')}</Label>
            <div className="rounded-lg border border-border overflow-hidden" style={{ height: 400 }}>
              <SketchCanvas />
            </div>
          </div>

          {/* Save/Cancel */}
          <div className="flex gap-2 pt-2">
            <button
              onClick={() => onOpenChange(false)}
              className="flex-1 rounded-lg border border-border py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleSave}
              className="flex-1 rounded-lg bg-primary py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              {t('rapporte.measurement.dialog.create')}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
