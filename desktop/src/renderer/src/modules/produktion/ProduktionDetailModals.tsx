/**
 * Detail modals for the Produktion module (shared/DetailModal — the Cosmi
 * window style, replacing the old slide-over DetailPanel).
 *
 * OrderDetailModal      — production order 360°: progress, material, BOM link,
 *                         work steps (check-off), scrap, QC list, real status
 *                         lifecycle actions (start/complete/cancel/delete) and
 *                         the Laufkarte PDF export.
 * BomDetailModal        — BOM positions, used-by orders (back-chained), active
 *                         toggle, positions CSV.
 * QualityCheckDetailModal — single inspection with link to its order.
 * MachineDetailModal    — machine status (editable) + its bookings.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Package,
  Calendar,
  Clock,
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  PlayCircle,
  ShieldCheck,
  ClipboardList,
  ListChecks,
  Gauge,
  Trash2,
  Download,
  FileText,
  Pencil,
  ChevronRight,
  Check,
  Play,
  Cpu,
} from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal, EmptyState } from '@/components/shared'
import { EmptyGeneric } from '@/components/shared/illustrations'
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
import {
  useWorkSteps,
  useUpdateWorkStep,
  useStartOrder,
  useCompleteOrder,
  useCancelOrder,
  useDeleteOrder,
  useUpdateOrder,
  useUpdateBOM,
  useUpdateMachine,
} from '@/api/hooks/useProduktion'
import type {
  ProductionOrder,
  BOMResponse,
  QualityCheckResponse,
  WorkStepResponse,
  MachineResponse,
  MachineBooking,
  MachineStatus,
} from '@/api/produktion-types'
import { formatDate } from '@/lib/format'
import {
  orderStatusLabelKeys,
  orderStatusColors,
  progressBarColors,
  workStepStatusLabelKeys,
  workStepStatusColors,
  machineStatusLabelKeys,
  machineStatusDots,
  priorityLabelKeys,
  priorityColors,
  PRIORITY_VALUES,
  getDaysRemaining,
  formatDuration,
  getMaterialAvailability,
  getScrapForOrder,
} from './produktion-shared'
import {
  buildOrderPdf,
  buildBomItemsCsv,
  downloadBlob,
  csvDateStamp,
  type OrderPdfLabels,
} from './produktion-export'
import { useProduktionTenantStore } from '@/stores/produktionTenant'

// ---------------------------------------------------------------------------
// Order detail
// ---------------------------------------------------------------------------

export function OrderDetailModal({
  order,
  bom,
  qualityChecks,
  onClose,
  onBack,
  onOpenBom,
  onOpenQualityCheck,
  onAddQualityCheck,
}: {
  order: ProductionOrder
  bom: BOMResponse | undefined
  qualityChecks: QualityCheckResponse[]
  onClose: () => void
  onBack?: () => void
  onOpenBom: (bomId: string) => void
  onOpenQualityCheck: (checkId: string) => void
  onAddQualityCheck: () => void
}) {
  const { t } = useTranslation()
  const { data: workStepsData } = useWorkSteps(order.id)
  const workSteps: WorkStepResponse[] = useMemo(
    () => workStepsData?.steps ?? [],
    [workStepsData],
  )

  const startOrder = useStartOrder()
  const completeOrder = useCompleteOrder()
  const cancelOrder = useCancelOrder()
  const deleteOrder = useDeleteOrder()
  const updateWorkStep = useUpdateWorkStep()

  const requireQc = useProduktionTenantStore((s) => s.requireQcBeforeComplete)
  const scrapWarnThreshold = useProduktionTenantStore((s) => s.scrapWarnThreshold)

  const [confirmAction, setConfirmAction] = useState<'cancel' | 'delete' | null>(null)
  const [editing, setEditing] = useState(false)

  const daysLeft = getDaysRemaining(order.planned_end)
  const totalDays = Math.max(1, Math.ceil(
    (new Date(order.planned_end).getTime() - new Date(order.planned_start).getTime()) / (1000 * 60 * 60 * 24),
  ))

  const materialAvailability = useMemo(() => {
    if (!bom) return null
    if (order.status !== 'planned' && order.status !== 'in_progress') return null
    const results = bom.items.map((item, idx) => ({
      ...item,
      availability: getMaterialAvailability(item.material_name, idx),
    }))
    const available = results.filter((r) => r.availability === 'green').length
    return { results, available, total: results.length }
  }, [bom, order.status])

  const sortedWorkSteps = useMemo(
    () => [...workSteps].sort((a, b) => a.step_nr - b.step_nr),
    [workSteps],
  )
  const completedSteps = sortedWorkSteps.filter((s) => s.status === 'completed').length
  const totalDuration = sortedWorkSteps.reduce((sum, s) => sum + s.duration_minutes, 0)
  const progress = sortedWorkSteps.length > 0
    ? Math.round((completedSteps / sortedWorkSteps.length) * 100)
    : order.status === 'completed' ? 100 : 0

  const scrap = getScrapForOrder(order, qualityChecks)
  const showScrap = order.status !== 'planned'
  const hasPassedQc = qualityChecks.some((qc) => qc.passed)
  const completeBlocked = requireQc && !hasPassedQc
  const isActive = order.status === 'planned' || order.status === 'in_progress'
  const mutating =
    startOrder.isPending || completeOrder.isPending || cancelOrder.isPending || deleteOrder.isPending

  const handleStart = async () => {
    try {
      await startOrder.mutateAsync(order.id)
      toast.success(t('produktion.statusChange.started', { orderNr: order.order_number }))
    } catch {
      toast.error(t('common.error'))
    }
  }

  const handleComplete = async () => {
    try {
      await completeOrder.mutateAsync(order.id)
      toast.success(t('produktion.statusChange.completed', { orderNr: order.order_number }))
    } catch {
      toast.error(t('common.error'))
    }
  }

  const handleCancel = async () => {
    try {
      await cancelOrder.mutateAsync(order.id)
      toast.success(t('produktion.statusChange.cancelled', { orderNr: order.order_number }))
      setConfirmAction(null)
    } catch {
      toast.error(t('common.error'))
    }
  }

  const handleDelete = async () => {
    try {
      await deleteOrder.mutateAsync(order.id)
      toast.success(t('produktion.statusChange.deleted', { orderNr: order.order_number }))
      onClose()
    } catch {
      toast.error(t('common.error'))
    }
  }

  const handleStepAdvance = async (step: WorkStepResponse) => {
    const next = step.status === 'pending' ? 'in_progress' : 'completed'
    try {
      await updateWorkStep.mutateAsync({ orderId: order.id, stepId: step.id, status: next })
      toast.success(t('produktion.workstep.updated', { name: step.name, status: t(workStepStatusLabelKeys[next]) }))
    } catch {
      toast.error(t('common.error'))
    }
  }

  const handleExportPdf = () => {
    const labels: OrderPdfLabels = {
      title: t('produktion.export.pdf.title'),
      product: t('produktion.export.pdf.product'),
      quantity: t('produktion.export.pdf.quantity'),
      status: t('produktion.export.pdf.status'),
      priority: t('produktion.export.pdf.priority'),
      plannedStart: t('produktion.export.pdf.plannedStart'),
      plannedEnd: t('produktion.export.pdf.plannedEnd'),
      bomSection: t('produktion.export.pdf.bomSection'),
      bomEmpty: t('produktion.export.pdf.bomEmpty'),
      perPiece: t('produktion.export.pdf.perPiece'),
      demand: t('produktion.export.pdf.demand'),
      stepsSection: t('produktion.export.pdf.stepsSection'),
      stepsEmpty: t('produktion.export.pdf.stepsEmpty'),
      duration: t('produktion.export.pdf.duration'),
      assignee: t('produktion.export.pdf.assignee'),
      signature: t('produktion.export.pdf.signature'),
      qualitySection: t('produktion.export.pdf.qualitySection'),
      qcPassed: t('produktion.detail.bestanden'),
      qcFailed: t('produktion.detail.nichtBestanden'),
      qcDefects: (count) => t('produktion.export.pdf.qcDefects', { count }),
      notes: t('produktion.export.pdf.notes'),
      moreLines: (count) => t('produktion.export.pdf.moreLines', { count }),
    }
    const blob = buildOrderPdf(
      order,
      bom,
      sortedWorkSteps,
      qualityChecks,
      t(orderStatusLabelKeys[order.status]),
      t(priorityLabelKeys[order.priority] ?? 'produktion.priority.p3'),
      labels,
    )
    downloadBlob(blob, `${order.order_number}.pdf`)
    toast.success(t('produktion.export.pdfExported', { orderNr: order.order_number }))
  }

  return (
    <DetailModal
      open
      onClose={onClose}
      onBack={onBack}
      title={t('produktion.detail.title')}
      subtitle={order.order_number}
      badge={
        <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-medium shrink-0 ${orderStatusColors[order.status]}`}>
          {t(orderStatusLabelKeys[order.status])}
        </span>
      }
      footer={
        <div className="flex flex-wrap gap-2">
          {confirmAction ? (
            <div className="flex w-full items-center justify-between gap-2">
              <span className="text-xs text-error font-medium">
                {confirmAction === 'cancel'
                  ? t('produktion.confirm.cancelOrder', { orderNr: order.order_number })
                  : t('produktion.confirm.deleteOrder', { orderNr: order.order_number })}
              </span>
              <div className="flex gap-2">
                <button
                  onClick={() => setConfirmAction(null)}
                  className="rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
                >
                  {t('common.cancel')}
                </button>
                <button
                  onClick={confirmAction === 'cancel' ? handleCancel : handleDelete}
                  disabled={mutating}
                  className="rounded-lg bg-error px-3 py-1.5 text-xs text-white hover:opacity-90 transition-opacity disabled:opacity-60"
                >
                  {t('common.confirm')}
                </button>
              </div>
            </div>
          ) : (
            <>
              {order.status === 'planned' && (
                <button
                  onClick={handleStart}
                  disabled={mutating}
                  className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-60"
                >
                  <PlayCircle className="h-3.5 w-3.5" />
                  {t('produktion.action.starten')}
                </button>
              )}
              {order.status === 'in_progress' && (
                <button
                  onClick={handleComplete}
                  disabled={mutating || completeBlocked}
                  title={completeBlocked ? t('produktion.action.qcRequiredHint') : undefined}
                  className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
                >
                  <CheckCircle2 className="h-3.5 w-3.5" />
                  {t('produktion.action.abschliessen')}
                </button>
              )}
              {isActive && (
                <>
                  <button
                    onClick={onAddQualityCheck}
                    className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs text-foreground hover:bg-secondary transition-colors"
                  >
                    <ShieldCheck className="h-3.5 w-3.5" />
                    {t('produktion.action.qualitaetspruefung')}
                  </button>
                  <button
                    onClick={() => setEditing((v) => !v)}
                    className={`flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs transition-colors ${editing ? 'bg-secondary text-foreground' : 'text-foreground hover:bg-secondary'}`}
                  >
                    <Pencil className="h-3.5 w-3.5" />
                    {t('produktion.action.bearbeiten')}
                  </button>
                </>
              )}
              <button
                onClick={handleExportPdf}
                className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs text-foreground hover:bg-secondary transition-colors"
              >
                <FileText className="h-3.5 w-3.5" />
                {t('produktion.action.pdfLaufkarte')}
              </button>
              {isActive && (
                <button
                  onClick={() => setConfirmAction('cancel')}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs text-error hover:bg-error-light transition-colors ml-auto"
                >
                  <XCircle className="h-3.5 w-3.5" />
                  {t('produktion.action.stornieren')}
                </button>
              )}
              {order.status === 'planned' && (
                <button
                  onClick={() => setConfirmAction('delete')}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs text-error hover:bg-error-light transition-colors"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  {t('produktion.action.loeschen')}
                </button>
              )}
            </>
          )}
        </div>
      }
    >
      <div className="space-y-5">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h3 className="text-base font-medium text-foreground">{order.product_name}</h3>
            <p className="text-xs text-muted-foreground mt-0.5 font-mono">{order.order_number}</p>
          </div>
          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium shrink-0 ${priorityColors[order.priority] ?? priorityColors[3]}`}>
            {t(priorityLabelKeys[order.priority] ?? 'produktion.priority.p3')}
          </span>
        </div>

        {editing && isActive ? (
          <OrderInlineEditForm order={order} onDone={() => setEditing(false)} />
        ) : (
          <>
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-xs font-medium text-foreground">{t('produktion.detail.fortschritt')}</span>
                <span className="text-sm font-semibold text-foreground tabular-nums">{progress}%</span>
              </div>
              <div className="h-3 rounded-full bg-secondary overflow-hidden">
                <div className={`h-full rounded-full transition-all ${progressBarColors[order.status] ?? 'bg-primary'}`} style={{ width: `${progress}%` }} />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3 text-xs">
              <div className="flex items-center gap-2 text-muted-foreground">
                <Package className="h-3.5 w-3.5 shrink-0" />
                <div>
                  <p className="text-[10px] text-muted-foreground">{t('produktion.detail.menge')}</p>
                  <p className="text-foreground font-medium">{t('produktion.detail.stueck', { qty: order.quantity.toLocaleString('de-DE') })}</p>
                </div>
              </div>
              <div className="flex items-center gap-2 text-muted-foreground">
                <Calendar className="h-3.5 w-3.5 shrink-0" />
                <div>
                  <p className="text-[10px] text-muted-foreground">{t('produktion.detail.startdatum')}</p>
                  <p className="text-foreground">{formatDate(order.planned_start)}</p>
                </div>
              </div>
              <div className="flex items-center gap-2 text-muted-foreground">
                <Clock className="h-3.5 w-3.5 shrink-0" />
                <div>
                  <p className="text-[10px] text-muted-foreground">{t('produktion.detail.faelligkeitsdatum')}</p>
                  <p className={order.status !== 'completed' && order.status !== 'cancelled' && daysLeft < 0 ? 'text-error font-medium' : 'text-foreground'}>
                    {formatDate(order.planned_end)}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2 text-muted-foreground">
                <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                <div>
                  <p className="text-[10px] text-muted-foreground">{t('produktion.detail.verbleibend')}</p>
                  <p className={`font-medium ${order.status === 'completed' ? 'text-success' : order.status === 'cancelled' ? 'text-muted-foreground' : daysLeft < 0 ? 'text-error' : daysLeft <= 3 ? 'text-warning' : 'text-foreground'}`}>
                    {order.status === 'completed' ? t('produktion.status.completed') :
                     order.status === 'cancelled' ? t('produktion.status.cancelled') :
                     daysLeft < 0 ? t('produktion.detail.tageUeberfaellig', { days: Math.abs(daysLeft) }) :
                     daysLeft === 0 ? t('produktion.detail.heuteFaellig') :
                     t('produktion.detail.tage', { days: daysLeft })}
                  </p>
                </div>
              </div>
            </div>

            <div className="rounded-md border border-border p-3">
              <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
                <span>{formatDate(order.planned_start)}</span>
                <span>{t('produktion.detail.laufzeit', { days: totalDays })}</span>
                <span>{formatDate(order.planned_end)}</span>
              </div>
              <div className="h-1.5 rounded-full bg-secondary overflow-hidden">
                <div className="h-full rounded-full bg-primary/60 transition-all"
                  style={{ width: `${Math.min(100, Math.max(0, ((totalDays - Math.max(0, daysLeft)) / totalDays) * 100))}%` }} />
              </div>
            </div>
          </>
        )}

        {materialAvailability && bom && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.detail.materialverfuegbarkeit')}</h4>
            <div className="rounded-md border border-border p-3 space-y-3">
              <div className="flex items-center gap-2">
                <Package className="h-4 w-4 text-primary" />
                <span className="text-xs font-medium text-foreground">{t('produktion.detail.materialVon', { available: materialAvailability.available, total: materialAvailability.total })}</span>
              </div>
              <div className="space-y-1">
                {materialAvailability.results.map((item, idx: number) => {
                  const colorMap = {
                    green: { dot: 'bg-success', text: 'text-success', bg: 'bg-success-light', label: t('produktion.material.available') },
                    yellow: { dot: 'bg-warning', text: 'text-warning', bg: 'bg-warning-light', label: t('produktion.material.limited') },
                    red: { dot: 'bg-error', text: 'text-error', bg: 'bg-error-light', label: t('produktion.material.missing') },
                  }
                  const c = colorMap[item.availability]
                  return (
                    <div key={idx} className="flex items-center justify-between py-1">
                      <div className="flex items-center gap-2">
                        <span className={`h-2 w-2 rounded-full ${c.dot} shrink-0`} />
                        <span className="text-xs text-foreground">{item.material_name}</span>
                      </div>
                      <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${c.bg} ${c.text}`}>{c.label}</span>
                    </div>
                  )
                })}
              </div>
            </div>
          </section>
        )}

        {bom && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.detail.stueckliste')}</h4>
            <div
              role="button"
              tabIndex={0}
              onClick={() => onOpenBom(bom.id)}
              onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onOpenBom(bom.id) } }}
              className="flex items-center justify-between rounded-md border border-border p-3 cursor-pointer hover:bg-secondary/50 transition-colors focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              <div className="flex items-center gap-2">
                <ClipboardList className="h-4 w-4 text-primary" />
                <div>
                  <p className="text-xs font-medium text-foreground">{bom.product_name}</p>
                  <p className="text-[10px] text-muted-foreground">{t('produktion.stueckliste.sku')} {bom.sku} &middot; {t('produktion.stueckliste.version', { version: bom.version })} &middot; {t('produktion.stueckliste.positionen', { count: bom.items.length })}</p>
                </div>
              </div>
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            </div>
          </section>
        )}

        {sortedWorkSteps.length > 0 && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.detail.arbeitsgaenge')}</h4>
            <div className="rounded-md border border-border p-3 space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <ListChecks className="h-4 w-4 text-primary" />
                  <span className="text-xs font-medium text-foreground">{t('produktion.detail.schritte', { completed: completedSteps, total: sortedWorkSteps.length })}</span>
                </div>
                <span className="text-xs text-muted-foreground">{t('produktion.detail.gesamt')} {formatDuration(totalDuration)}</span>
              </div>
              <div className="h-1.5 rounded-full bg-secondary overflow-hidden">
                <div className="h-full rounded-full bg-success transition-all" style={{ width: sortedWorkSteps.length > 0 ? `${(completedSteps / sortedWorkSteps.length) * 100}%` : '0%' }} />
              </div>
              <div className="space-y-1">
                {sortedWorkSteps.map((step) => (
                  <div key={step.id} className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2">
                    <div className="flex items-center gap-2.5 min-w-0">
                      <span className="text-[10px] text-muted-foreground font-mono w-4 shrink-0 text-right">{step.step_nr}</span>
                      <div className="min-w-0">
                        <p className="text-xs text-foreground font-medium truncate">{step.name}</p>
                        <div className="flex items-center gap-2 mt-0.5">
                          <span className="text-[10px] text-muted-foreground">{formatDuration(step.duration_minutes)}</span>
                          {step.assignee && <span className="text-[10px] text-muted-foreground">&middot; {step.assignee}</span>}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-1.5 shrink-0 ml-2">
                      <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${workStepStatusColors[step.status]}`}>{t(workStepStatusLabelKeys[step.status])}</span>
                      {order.status === 'in_progress' && step.status === 'pending' && (
                        <button
                          onClick={() => handleStepAdvance(step)}
                          disabled={updateWorkStep.isPending}
                          title={t('produktion.workstep.markInProgress')}
                          aria-label={t('produktion.workstep.markInProgress')}
                          className="rounded-md p-1 text-muted-foreground hover:text-info hover:bg-info-light transition-colors disabled:opacity-40"
                        >
                          <Play className="h-3.5 w-3.5" />
                        </button>
                      )}
                      {order.status === 'in_progress' && step.status === 'in_progress' && (
                        <button
                          onClick={() => handleStepAdvance(step)}
                          disabled={updateWorkStep.isPending}
                          title={t('produktion.workstep.markCompleted')}
                          aria-label={t('produktion.workstep.markCompleted')}
                          className="rounded-md p-1 text-muted-foreground hover:text-success hover:bg-success-light transition-colors disabled:opacity-40"
                        >
                          <Check className="h-3.5 w-3.5" />
                        </button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </section>
        )}

        {showScrap && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.detail.ausschuss')}</h4>
            <div className="rounded-md border border-border p-3 space-y-3">
              {scrap.quantity === 0 ? (
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-success" />
                  <span className="text-xs text-muted-foreground">{t('produktion.detail.keinAusschuss')}</span>
                </div>
              ) : (
                <>
                  <div className="flex items-center gap-4">
                    <div className="flex items-center gap-2">
                      <Gauge className="h-4 w-4 text-muted-foreground" />
                      <div>
                        <p className="text-[10px] text-muted-foreground">{t('produktion.detail.ausschussrate')}</p>
                        <p className={`text-sm font-semibold tabular-nums ${scrap.rate > scrapWarnThreshold ? 'text-error' : 'text-foreground'}`}>{scrap.rate.toLocaleString('de-DE')}%</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Trash2 className="h-4 w-4 text-muted-foreground" />
                      <div>
                        <p className="text-[10px] text-muted-foreground">{t('produktion.detail.ausschussProduziert')}</p>
                        <p className="text-sm font-medium text-foreground tabular-nums">{scrap.quantity} / {order.quantity} Stk</p>
                      </div>
                    </div>
                  </div>
                  <div className="h-2.5 rounded-full bg-success/30 overflow-hidden">
                    <div className="h-full rounded-full bg-error transition-all" style={{ width: `${Math.min(100, scrap.rate)}%` }} />
                  </div>
                  {scrap.rate > scrapWarnThreshold && (
                    <div className="flex items-center gap-2 rounded-md bg-error-light px-3 py-2">
                      <AlertTriangle className="h-3.5 w-3.5 text-error shrink-0" />
                      <span className="text-xs text-error font-medium">{t('produktion.detail.ausschussWarnung', { threshold: scrapWarnThreshold })}</span>
                    </div>
                  )}
                </>
              )}
            </div>
          </section>
        )}

        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.detail.qualitaetspruefungen', { count: qualityChecks.length })}</h4>
          {qualityChecks.length === 0 ? (
            <EmptyState illustration={<EmptyGeneric />} title={t('produktion.detail.noPruefungen')} />
          ) : (
            <div className="space-y-1.5">
              {qualityChecks.map((qc) => (
                <div
                  key={qc.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => onOpenQualityCheck(qc.id)}
                  onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onOpenQualityCheck(qc.id) } }}
                  className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2 text-xs cursor-pointer hover:bg-secondary/50 transition-colors focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  <div className="flex items-center gap-2">
                    {qc.passed ? <CheckCircle2 className="h-3.5 w-3.5 text-success" /> : <XCircle className="h-3.5 w-3.5 text-error" />}
                    <div>
                      <p className="text-foreground">{qc.inspector}</p>
                      <p className="text-[10px] text-muted-foreground">
                        {formatDate(qc.checked_at)}
                        {qc.defects_found > 0 && t('produktion.detail.maengel', { count: qc.defects_found })}
                      </p>
                    </div>
                  </div>
                  <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${qc.passed ? 'bg-success-light text-success' : 'bg-error-light text-error'}`}>
                    {qc.passed ? t('produktion.detail.bestanden') : t('produktion.detail.nichtBestanden')}
                  </span>
                </div>
              ))}
            </div>
          )}
        </section>

        {order.notes && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.detail.notizen')}</h4>
            <p className="rounded-md border border-border p-3 text-xs text-foreground whitespace-pre-wrap">{order.notes}</p>
          </section>
        )}
      </div>
    </DetailModal>
  )
}

/** Inline edit (planned: all fields; in_progress: priority + notes only). */
function OrderInlineEditForm({ order, onDone }: { order: ProductionOrder; onDone: () => void }) {
  const { t } = useTranslation()
  const updateOrder = useUpdateOrder()
  const planned = order.status === 'planned'
  const [quantity, setQuantity] = useState(String(order.quantity))
  const [startDate, setStartDate] = useState(order.planned_start.slice(0, 10))
  const [dueDate, setDueDate] = useState(order.planned_end.slice(0, 10))
  const [priority, setPriority] = useState(String(order.priority))
  const [notes, setNotes] = useState(order.notes)

  const handleSave = async () => {
    if (planned && (!quantity || parseInt(quantity) <= 0)) {
      toast.error(t('produktion.dialog.errorMenge'))
      return
    }
    try {
      await updateOrder.mutateAsync({
        id: order.id,
        priority: parseInt(priority),
        notes,
        ...(planned
          ? {
              quantity: parseInt(quantity),
              planned_start: new Date(startDate).toISOString(),
              planned_end: new Date(dueDate).toISOString(),
            }
          : {}),
      })
      toast.success(t('produktion.edit.savedToast', { orderNr: order.order_number }))
      onDone()
    } catch {
      toast.error(t('common.error'))
    }
  }

  return (
    <div className="rounded-md border border-border p-3 space-y-3">
      <p className="text-xs font-medium text-foreground">{t('produktion.edit.title')}</p>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label className="text-xs">{t('produktion.dialog.menge')}</Label>
          <Input type="number" min="1" value={quantity} disabled={!planned} onChange={(e) => setQuantity(e.target.value)} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{t('produktion.dialog.prioritaet')}</Label>
          <Select value={priority} onValueChange={setPriority}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              {PRIORITY_VALUES.map((p) => (
                <SelectItem key={p} value={String(p)}>{t(priorityLabelKeys[p])}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{t('produktion.dialog.startdatum')}</Label>
          <Input type="date" value={startDate} disabled={!planned} onChange={(e) => setStartDate(e.target.value)} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{t('produktion.dialog.faelligkeitsdatum')}</Label>
          <Input type="date" value={dueDate} disabled={!planned} onChange={(e) => setDueDate(e.target.value)} />
        </div>
      </div>
      <div className="space-y-1">
        <Label className="text-xs">{t('produktion.dialog.notizen')}</Label>
        <Textarea value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} />
      </div>
      <div className="flex gap-2 justify-end">
        <button onClick={onDone} className="rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors">
          {t('common.cancel')}
        </button>
        <button
          onClick={handleSave}
          disabled={updateOrder.isPending}
          className="rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-60"
        >
          {t('common.save')}
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// BOM detail
// ---------------------------------------------------------------------------

export function BomDetailModal({
  bom,
  orders,
  onClose,
  onBack,
  onOpenOrder,
}: {
  bom: BOMResponse
  orders: ProductionOrder[]
  onClose: () => void
  onBack?: () => void
  onOpenOrder: (orderId: string) => void
}) {
  const { t } = useTranslation()
  const updateBOM = useUpdateBOM()
  const usedByOrders = orders.filter((o) => o.bom_id === bom.id)
  const sortedItems = [...bom.items].sort((a, b) => a.sort_order - b.sort_order)

  const handleToggleActive = async () => {
    try {
      await updateBOM.mutateAsync({ id: bom.id, active: !bom.active })
      toast.success(bom.active
        ? t('produktion.bomDetail.deactivatedToast', { name: bom.product_name })
        : t('produktion.bomDetail.activatedToast', { name: bom.product_name }))
    } catch {
      toast.error(t('common.error'))
    }
  }

  const handleExportCsv = () => {
    const csv = buildBomItemsCsv(bom)
    const blob = new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' })
    downloadBlob(blob, `stueckliste-${bom.sku}-${csvDateStamp()}.csv`)
    toast.success(t('produktion.export.csvExported'))
  }

  return (
    <DetailModal
      open
      onClose={onClose}
      onBack={onBack}
      title={t('produktion.bomDetail.title')}
      subtitle={`${bom.sku} · ${t('produktion.stueckliste.version', { version: bom.version })}`}
      badge={
        bom.active ? (
          <span className="rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium shrink-0">{t('produktion.stueckliste.aktiv')}</span>
        ) : (
          <span className="rounded-full bg-secondary text-muted-foreground px-2 py-0.5 text-[10px] font-medium shrink-0">{t('produktion.stueckliste.inaktiv')}</span>
        )
      }
      footer={
        <div className="flex flex-wrap gap-2">
          <button
            onClick={handleToggleActive}
            disabled={updateBOM.isPending}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs text-foreground hover:bg-secondary transition-colors disabled:opacity-60"
          >
            {bom.active ? <XCircle className="h-3.5 w-3.5" /> : <CheckCircle2 className="h-3.5 w-3.5" />}
            {bom.active ? t('produktion.bomDetail.deactivate') : t('produktion.bomDetail.activate')}
          </button>
          <button
            onClick={handleExportCsv}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <Download className="h-3.5 w-3.5" />
            {t('produktion.bomDetail.exportCsv')}
          </button>
        </div>
      }
    >
      <div className="space-y-5">
        <div>
          <h3 className="text-base font-medium text-foreground">{bom.product_name}</h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            {t('produktion.stueckliste.sku')} {bom.sku} &middot; {t('produktion.stueckliste.positionen', { count: bom.items.length })}
          </p>
        </div>

        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.bomDetail.positionen')}</h4>
          <div className="rounded-md border border-border overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border">
                  <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground w-8">{t('produktion.stueckliste.table.nr')}</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">{t('produktion.stueckliste.table.material')}</th>
                  <th className="px-3 py-2 text-right text-xs font-medium text-muted-foreground">{t('produktion.stueckliste.table.menge')}</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground pl-3">{t('produktion.stueckliste.table.einheit')}</th>
                </tr>
              </thead>
              <tbody>
                {sortedItems.map((item, idx) => (
                  <tr key={item.id} className="border-b border-border-muted last:border-0">
                    <td className="px-3 py-2 text-xs text-muted-foreground">{idx + 1}</td>
                    <td className="px-3 py-2 text-xs text-foreground">{item.material_name}</td>
                    <td className="px-3 py-2 text-xs text-muted-foreground text-right">{item.quantity}</td>
                    <td className="px-3 py-2 text-xs text-muted-foreground pl-3">{item.unit}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.bomDetail.usedBy', { count: usedByOrders.length })}</h4>
          {usedByOrders.length === 0 ? (
            <p className="text-xs text-muted-foreground">{t('produktion.bomDetail.usedByNone')}</p>
          ) : (
            <div className="space-y-1.5">
              {usedByOrders.map((o) => (
                <div
                  key={o.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => onOpenOrder(o.id)}
                  onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onOpenOrder(o.id) } }}
                  className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2 text-xs cursor-pointer hover:bg-secondary/50 transition-colors focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="font-mono text-muted-foreground">{o.order_number}</span>
                    <span className="text-foreground truncate">{t('produktion.detail.stueck', { qty: o.quantity.toLocaleString('de-DE') })}</span>
                  </div>
                  <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium shrink-0 ${orderStatusColors[o.status]}`}>{t(orderStatusLabelKeys[o.status])}</span>
                </div>
              ))}
            </div>
          )}
        </section>

        {bom.notes && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.detail.notizen')}</h4>
            <p className="rounded-md border border-border p-3 text-xs text-foreground whitespace-pre-wrap">{bom.notes}</p>
          </section>
        )}
      </div>
    </DetailModal>
  )
}

// ---------------------------------------------------------------------------
// Quality check detail
// ---------------------------------------------------------------------------

export function QualityCheckDetailModal({
  check,
  order,
  onClose,
  onBack,
  onOpenOrder,
}: {
  check: QualityCheckResponse
  order: ProductionOrder | undefined
  onClose: () => void
  onBack?: () => void
  onOpenOrder: (orderId: string) => void
}) {
  const { t } = useTranslation()

  return (
    <DetailModal
      open
      onClose={onClose}
      onBack={onBack}
      title={t('produktion.qcDetail.title')}
      subtitle={order?.order_number ?? check.order_id}
      badge={
        <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-medium shrink-0 ${check.passed ? 'bg-success-light text-success' : 'bg-error-light text-error'}`}>
          {check.passed ? t('produktion.detail.bestanden') : t('produktion.detail.nichtBestanden')}
        </span>
      }
      footer={
        order && (
          <button
            onClick={() => onOpenOrder(order.id)}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <ClipboardList className="h-3.5 w-3.5" />
            {t('produktion.qcDetail.openOrder')}
          </button>
        )
      }
    >
      <div className="space-y-5">
        <div className="flex items-start gap-3">
          {check.passed
            ? <CheckCircle2 className="h-8 w-8 text-success shrink-0" />
            : <XCircle className="h-8 w-8 text-error shrink-0" />}
          <div>
            <h3 className="text-base font-medium text-foreground">
              {check.passed ? t('produktion.qcDetail.passedHeadline') : t('produktion.qcDetail.failedHeadline')}
            </h3>
            {order && <p className="text-xs text-muted-foreground mt-0.5">{order.product_name}</p>}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3 text-xs">
          <div>
            <p className="text-[10px] text-muted-foreground">{t('produktion.qcDetail.inspector')}</p>
            <p className="text-foreground font-medium">{check.inspector}</p>
          </div>
          <div>
            <p className="text-[10px] text-muted-foreground">{t('produktion.qcDetail.date')}</p>
            <p className="text-foreground">{formatDate(check.checked_at)}</p>
          </div>
          <div>
            <p className="text-[10px] text-muted-foreground">{t('produktion.qcDetail.defects')}</p>
            <p className={`font-medium tabular-nums ${check.defects_found > 0 ? 'text-error' : 'text-foreground'}`}>{check.defects_found}</p>
          </div>
          {order && (
            <div>
              <p className="text-[10px] text-muted-foreground">{t('produktion.qcDetail.order')}</p>
              <p className="text-foreground font-mono">{order.order_number}</p>
            </div>
          )}
        </div>

        {check.notes && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.detail.notizen')}</h4>
            <p className="rounded-md border border-border p-3 text-xs text-foreground whitespace-pre-wrap">{check.notes}</p>
          </section>
        )}
      </div>
    </DetailModal>
  )
}

// ---------------------------------------------------------------------------
// Machine detail
// ---------------------------------------------------------------------------

export function MachineDetailModal({
  machine,
  bookings,
  orderNumbers,
  onClose,
  onBack,
  onOpenOrder,
}: {
  machine: MachineResponse
  bookings: MachineBooking[]
  orderNumbers: Map<string, string>
  onClose: () => void
  onBack?: () => void
  onOpenOrder: (orderId: string) => void
}) {
  const { t } = useTranslation()
  const updateMachine = useUpdateMachine()
  const machineBookings = bookings
    .filter((b) => b.machine_id === machine.id)
    .sort((a, b) => a.starts_at.localeCompare(b.starts_at))

  const handleStatusChange = async (status: MachineStatus) => {
    if (status === machine.status) return
    try {
      await updateMachine.mutateAsync({ id: machine.id, status })
      toast.success(t('produktion.machineDetail.statusUpdated', { name: machine.name, status: t(machineStatusLabelKeys[status]) }))
    } catch {
      toast.error(t('common.error'))
    }
  }

  return (
    <DetailModal
      open
      onClose={onClose}
      onBack={onBack}
      title={t('produktion.machineDetail.title')}
      subtitle={machine.type || undefined}
      badge={
        <span className="flex items-center gap-1.5 rounded-full bg-secondary px-2.5 py-0.5 text-[10px] font-medium text-foreground shrink-0">
          <span className={`h-2 w-2 rounded-full ${machineStatusDots[machine.status]}`} />
          {t(machineStatusLabelKeys[machine.status])}
        </span>
      }
    >
      <div className="space-y-5">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light shrink-0">
            <Cpu className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h3 className="text-base font-medium text-foreground">{machine.name}</h3>
            {machine.type && <p className="text-xs text-muted-foreground mt-0.5">{machine.type}</p>}
          </div>
        </div>

        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.machineDetail.status')}</h4>
          <Select value={machine.status} onValueChange={(v) => handleStatusChange(v as MachineStatus)}>
            <SelectTrigger className="w-56" disabled={updateMachine.isPending}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(['available', 'in_use', 'maintenance'] as const).map((s) => (
                <SelectItem key={s} value={s}>{t(machineStatusLabelKeys[s])}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </section>

        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.machineDetail.bookings', { count: machineBookings.length })}</h4>
          {machineBookings.length === 0 ? (
            <p className="text-xs text-muted-foreground">{t('produktion.machineDetail.noBookings')}</p>
          ) : (
            <div className="space-y-1.5">
              {machineBookings.map((b) => (
                <div
                  key={b.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => onOpenOrder(b.production_order_id)}
                  onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onOpenOrder(b.production_order_id) } }}
                  className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2 text-xs cursor-pointer hover:bg-secondary/50 transition-colors focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  <span className="font-mono text-muted-foreground">
                    {orderNumbers.get(b.production_order_id) ?? b.production_order_id.slice(0, 8)}
                  </span>
                  <span className="text-muted-foreground">
                    {formatDate(b.starts_at)} – {formatDate(b.ends_at)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </section>

        {machine.notes && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">{t('produktion.detail.notizen')}</h4>
            <p className="rounded-md border border-border p-3 text-xs text-foreground whitespace-pre-wrap">{machine.notes}</p>
          </section>
        )}
      </div>
    </DetailModal>
  )
}
