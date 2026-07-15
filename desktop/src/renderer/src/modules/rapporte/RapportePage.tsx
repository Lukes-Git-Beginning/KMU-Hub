import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Plus,
  ClipboardList,
  Clock,
  Users,
  Camera,
  AlertTriangle,
  Trash2,
  Ruler,
  FileText,
  ChevronDown,
  ChevronRight,
  Thermometer,
  CheckCircle2,
  XCircle,
  Send,
  Wind,
  Droplets,
  ClipboardCheck,
  Download,
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
import {
  useRapporteList,
  useReportStats,
  useCreateReport,
  useUpdateReport,
  useDeleteReport,
  useSubmitReport,
} from '@/api/hooks/useRapporte'
import { adaptWorkReport } from '@/api/rapporte-adapter'
import SignatureCanvas from './SignatureCanvas'
import SketchCanvas from './SketchCanvas'
import { EmptyState, PageHeader, SortMenu, type SortDirection, type SortFieldOption } from '@/components/shared'
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
import { formatCurrency } from '@/lib/format'
import { useRapportePrefsStore } from '@/stores/rapportePrefs'
import { useRapporteTenantStore } from '@/stores/rapporteTenant'
import { ReportDetailModal } from './ReportDetailModal'
import {
  weatherIcons,
  weatherLabelKeys,
  projectColors,
  approvalBadgeStyles,
  approvalLabelKeys,
  calcNetHours,
  calcNetMinutes,
  formatDate,
} from './rapporte-shared'
import { buildReportsCsv, downloadBlob, csvDateStamp } from './rapporte-export'

type TabKey = 'tagesberichte' | 'aufmass' | 'vorlagen'
type DateFilter = 'week' | 'month' | 'all'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatDateShort(dateStr: string): string {
  return formatDate(dateStr)
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

// ============================================================
// Main Component
// ============================================================

export default function RapportePage() {
  const { t } = useTranslation()

  // ---------------------------------------------------------------------------
  // Server state — reports (API hooks)
  // ---------------------------------------------------------------------------

  const { data: reportsData, isLoading: reportsLoading, isError: reportsError } = useRapporteList()
  const { data: statsData } = useReportStats()

  const deleteReportMutation = useDeleteReport()
  const submitReportMutation = useSubmitReport()
  const updateReportMutation = useUpdateReport()

  // Map wire shape → FieldReport for the UI
  const reports: FieldReport[] = useMemo(
    () => (reportsData?.reports ?? []).map(adaptWorkReport),
    [reportsData],
  )

  // ---------------------------------------------------------------------------
  // Client state — measurements + templates (no API endpoints yet)
  // ---------------------------------------------------------------------------

  const { measurements, templates, deleteMeasurement } = useRapporteStore()

  // ---------------------------------------------------------------------------
  // UI state
  // ---------------------------------------------------------------------------

  const [tab, setTab] = useState<TabKey>('tagesberichte')
  const [search, setSearch] = useState('')
  const [projectFilter, setProjectFilter] = useState<string>('all')
  // Personal pref (settings panel) seeds the initial date filter.
  const [dateFilter, setDateFilter] = useState<DateFilter>(() => useRapportePrefsStore.getState().defaultDateFilter)
  const [sortField, setSortField] = useState('date')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')
  const statsCurrency = useRapporteTenantStore((s) => s.currency)

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
    const dir = sortDir === 'asc' ? 1 : -1
    let list = [...reports].sort((a, b) => {
      switch (sortField) {
        case 'project':
          return a.projectName.localeCompare(b.projectName) * dir
        case 'author':
          return a.author.localeCompare(b.author) * dir
        case 'status':
          return a.approvalStatus.localeCompare(b.approvalStatus) * dir
        default:
          return a.date.localeCompare(b.date) * dir
      }
    })
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
  }, [reports, projectFilter, dateFilter, search, sortField, sortDir])

  // Stats — prefer server stats when available, fall back to client-computed values
  const reportsThisWeek = reports.filter((r) => isThisWeek(r.date)).length
  const totalHours = reports.reduce((sum, r) => sum + calcNetMinutes(r.workStart, r.workEnd, r.breakMinutes), 0)
  const totalHoursFormatted = `${Math.floor(totalHours / 60)}h`
  const activeProjects = statsData
    ? statsData.approved_count + statsData.submitted_count
    : new Set(reports.map((r) => r.projectId)).size

  // Mock material cost (no API field yet)
  const materialCostMock = 48_750

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------

  const handleDeleteReport = (id: string) => {
    deleteReportMutation.mutate(id, {
      onSuccess: () => {
        setSelectedReport(null)
        toast.success(t('rapporte.report.deleted'))
      },
      onError: () => toast.error(t('rapporte.report.deleteError')),
    })
  }

  const handleDeleteMeasurement = (id: string) => {
    deleteMeasurement(id)
    toast.success(t('rapporte.measurement.deleted'))
  }

  const handleUseTemplate = (template: ReportTemplate) => {
    setTemplatePrefill(template)
    setShowNewReport(true)
  }

  const handleUpdateReport = (id: string, updates: Partial<FieldReport>) => {
    // Map FieldReport partial back to UpdateReportInput for the API
    // Only fields the API understands are forwarded; approval transitions use dedicated endpoints
    if (updates.approvalStatus === 'submitted') {
      submitReportMutation.mutate(id, {
        onSuccess: (data) => {
          if (selectedReport && selectedReport.id === id) {
            setSelectedReport({ ...selectedReport, approvalStatus: data.report.status as FieldReport['approvalStatus'] })
          }
        },
        onError: () => toast.error(t('rapporte.detail.submitError')),
      })
    } else {
      updateReportMutation.mutate(
        { id, title: updates.projectName, description: updates.notes },
        {
          onSuccess: () => {
            if (selectedReport && selectedReport.id === id) {
              setSelectedReport({ ...selectedReport, ...updates })
            }
          },
          onError: () => toast.error(t('rapporte.detail.updateError')),
        },
      )
    }
  }

  const toggleMeasurement = (id: string) => {
    setExpandedMeasurement(expandedMeasurement === id ? null : id)
  }

  const handleExportReports = () => {
    downloadBlob(
      new Blob(['﻿' + buildReportsCsv(filteredReports)], { type: 'text/csv;charset=utf-8' }),
      `rapporte-${csvDateStamp()}.csv`,
    )
    toast.success(t('rapporte.export.success', { count: filteredReports.length }))
  }

  const sortOptions: SortFieldOption[] = [
    { value: 'date', label: t('rapporte.sort.date') },
    { value: 'project', label: t('rapporte.sort.project') },
    { value: 'author', label: t('rapporte.sort.author') },
    { value: 'status', label: t('rapporte.sort.status') },
  ]

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  if (reportsLoading) {
    return (
      <div className="flex-1 flex items-center justify-center p-6">
        <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
      </div>
    )
  }

  if (reportsError) {
    return (
      <div className="flex-1 flex items-center justify-center p-6">
        <p className="text-sm text-error">{t('common.errorLoading')}</p>
      </div>
    )
  }

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
            {formatCurrency(materialCostMock, statsCurrency)}
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
            <SortMenu
              options={sortOptions}
              field={sortField}
              direction={sortDir}
              onChange={(field, direction) => { setSortField(field); setSortDir(direction) }}
              triggerClassName="py-2"
            />
            <button
              onClick={handleExportReports}
              disabled={filteredReports.length === 0}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors disabled:opacity-50"
            >
              <Download className="h-4 w-4" />
              <span className="hidden sm:inline">{t('rapporte.export.button')}</span>
            </button>
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
                    role="button"
                    tabIndex={0}
                    aria-label={`${report.projectName} ${formatDate(report.date)}`}
                    onClick={() => setSelectedReport(report)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        setSelectedReport(report)
                      }
                    }}
                    className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
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
      {/* Report Detail Modal            */}
      {/* ============================== */}
      <ReportDetailModal
        report={selectedReport}
        onClose={() => setSelectedReport(null)}
        onDelete={handleDeleteReport}
        onUpdate={handleUpdateReport}
      />

      {/* ============================== */}
      {/* Neuer Tagesbericht Dialog      */}
      {/* ============================== */}
      <NewReportDialog
        key={showNewReport ? (templatePrefill?.id ?? 'new') : 'closed'}
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
  const createReport = useCreateReport()
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const [projectId, setProjectId] = useState('')
  const [weather, setWeather] = useState<WeatherType>('sunny')
  const [temperature, setTemperature] = useState('10')
  // Personal default shift (settings panel) seeds new reports.
  const [workStart, setWorkStart] = useState(() => useRapportePrefsStore.getState().defaultWorkStart)
  const [workEnd, setWorkEnd] = useState(() => useRapportePrefsStore.getState().defaultWorkEnd)
  const [breakMin, setBreakMin] = useState(() => String(useRapportePrefsStore.getState().defaultBreakMinutes))
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

  const resetForm = () => {
    const prefs = useRapportePrefsStore.getState()
    setDate(new Date().toISOString().split('T')[0])
    setProjectId('')
    setWeather('sunny')
    setTemperature('10')
    setWorkStart(prefs.defaultWorkStart)
    setWorkEnd(prefs.defaultWorkEnd)
    setBreakMin(String(prefs.defaultBreakMinutes))
    setWorkers([{ name: '', role: '', hours: 8 }])
    setActivities([{ description: '', category: '' }])
    setMaterials([{ article: '', quantity: 0, unit: 'Stk' }])
    setNotes('')
    setPhotos([])
    setSignatureDataUrl(null)
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
    const activitiesSummary = validActivities.map((a) => a.description).join('; ')

    createReport.mutate(
      {
        // Use project name as title (no dedicated project FK in API yet)
        title: project?.name ?? projectId,
        description: activitiesSummary,
        author_id: validWorkers[0]?.name ?? 'current-user',
        report_date: date,
      },
      {
        onSuccess: () => {
          toast.success(t('rapporte.dialog.reportCreated', { name: project?.name ?? 'Projekt' }))
          resetForm()
          onOpenChange(false)
        },
        onError: () => toast.error(t('rapporte.dialog.reportCreateError')),
      },
    )
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
  const { addMeasurement } = useRapporteStore()
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

    // Persist to store so the Aufmass tab reflects the new measurement
    addMeasurement({
      id: `ms-${Date.now()}`,
      name: name.trim(),
      projectName: project?.name ?? projectId,
      author: '',
      createdAt: new Date().toISOString().split('T')[0],
      positions: validPositions.map((p, idx) => {
        const l = parseFloat(p.length) || 0
        const w = parseFloat(p.width) || 0
        const h = parseFloat(p.height) || 0
        return {
          id: `mp-${Date.now()}-${idx}`,
          label: p.label,
          length: l,
          width: w,
          height: h,
          area: l * w,
          volume: l * w * h,
        }
      }),
    })

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
