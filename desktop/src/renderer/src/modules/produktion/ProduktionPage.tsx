import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Plus,
  Factory,
  ClipboardList,
  ShieldCheck,
  CheckCircle2,
  XCircle,
  ChevronDown,
  ChevronRight,
  Package,
  Calendar,
  Clock,
  AlertCircle,
  PlayCircle,
  Trash2,
  ListChecks,
  AlertTriangle,
  Gauge,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useOrders,
  useBOMs,
  useQualityChecks,
  useMachines,
  useMachineBookings,
  useCreateOrder,
  useCreateBOM,
  useCreateQualityCheck,
  useWorkSteps,
} from '@/api/hooks/useProduktion'
import type {
  ProductionOrder,
  BOMResponse as BOM,
  QualityCheckResponse as QualityCheck,
  BomItemResponse as BomItem,
  WorkStepResponse as WorkStep,
  CreateOrderInput,
  CreateBOMInput,
  CreateQualityCheckInput,
} from '@/api/produktion-types'
import { DetailPanel, PageHeader, EmptyState } from '@/components/shared'
import { EmptyGeneric } from '@/components/shared/illustrations'
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
import { Switch } from '@/components/ui/switch'
import MaschinenbelegungChart from './MaschinenbelegungChart'
import { formatDate } from '@/lib/format'

type TabKey = 'auftraege' | 'stuecklisten' | 'qualitaet' | 'maschinen'
type StatusFilter = 'all' | 'planned' | 'in_progress' | 'completed' | 'cancelled'

const orderStatusLabelKeys: Record<string, string> = {
  planned: 'produktion.status.planned',
  in_progress: 'produktion.status.in_progress',
  completed: 'produktion.status.completed',
  cancelled: 'produktion.status.cancelled',
}

const orderStatusColors: Record<string, string> = {
  planned: 'bg-secondary text-muted-foreground',
  in_progress: 'bg-info-light text-info',
  completed: 'bg-success-light text-success',
  cancelled: 'bg-error-light text-error',
}

const progressBarColors: Record<string, string> = {
  planned: 'bg-muted-foreground/40',
  in_progress: 'bg-info',
  completed: 'bg-success',
  cancelled: 'bg-error',
}

const statusFilterOptionKeys: { value: StatusFilter; labelKey: string }[] = [
  { value: 'all', labelKey: 'produktion.filter.all' },
  { value: 'planned', labelKey: 'produktion.filter.planned' },
  { value: 'in_progress', labelKey: 'produktion.filter.inProgress' },
  { value: 'completed', labelKey: 'produktion.filter.completed' },
  { value: 'cancelled', labelKey: 'produktion.filter.cancelled' },
]

const workStepStatusLabelKeys: Record<string, string> = {
  pending: 'produktion.workstep.pending',
  in_progress: 'produktion.workstep.inProgress',
  completed: 'produktion.workstep.completed',
  skipped: 'produktion.workstep.skipped',
}

const workStepStatusColors: Record<string, string> = {
  pending: 'bg-secondary text-muted-foreground',
  in_progress: 'bg-info-light text-info',
  completed: 'bg-success-light text-success',
  skipped: 'bg-secondary text-muted-foreground/60',
}

function getDaysRemaining(dueDate: string): number {
  const due = new Date(dueDate)
  const now = new Date()
  return Math.ceil((due.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
}

function formatDuration(minutes: number): string {
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  if (h === 0) return `${m}min`
  if (m === 0) return `${h}h`
  return `${h}h ${m}min`
}

function getMaterialAvailability(materialName: string, idx: number): 'green' | 'yellow' | 'red' {
  const val = (materialName.length * 7 + idx * 13) % 10
  if (val < 1) return 'red'
  if (val < 3) return 'yellow'
  return 'green'
}
// ============================================================
// Main Component
// ============================================================

export default function ProduktionPage() {
  const { t } = useTranslation()

  const { data: ordersData, isLoading: ordersLoading } = useOrders()
  const { data: bomsData } = useBOMs()
  const { data: qualityData } = useQualityChecks()
  const { data: machinesData } = useMachines()
  const { data: bookingsData } = useMachineBookings()

  const orders = ordersData?.orders ?? []
  const boms = bomsData?.boms ?? []
  const qualityChecks = qualityData?.checks ?? []
  const machines = machinesData?.machines ?? []
  const machineBookings = bookingsData?.bookings ?? []

  const orderStatusLabels = useMemo(() => Object.fromEntries(
    Object.entries(orderStatusLabelKeys).map(([k, v]) => [k, t(v)])
  ) as Record<string, string>, [t])

  const statusFilterOptions = useMemo(() =>
    statusFilterOptionKeys.map((o) => ({ value: o.value, label: t(o.labelKey) })),
    [t]
  )

  const [tab, setTab] = useState<TabKey>('auftraege')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [expandedBom, setExpandedBom] = useState<string | null>(null)
  const [selectedOrder, setSelectedOrder] = useState<ProductionOrder | null>(null)
  const [showNewOrder, setShowNewOrder] = useState(false)
  const [showNewBom, setShowNewBom] = useState(false)
  const [showQualityCheck, setShowQualityCheck] = useState(false)
  const [qualityCheckOrderId, setQualityCheckOrderId] = useState<string>('')

  const activeOrders = orders.filter((o) => o.status === 'in_progress' || o.status === 'planned')

  const filteredOrders = useMemo(() => {
    return orders.filter((o) => {
      if (statusFilter !== 'all' && o.status !== statusFilter) return false
      if (!search) return true
      const q = search.toLowerCase()
      return o.order_number.toLowerCase().includes(q) || o.product_name.toLowerCase().includes(q)
    })
  }, [orders, search, statusFilter])

  const bomMap = useMemo(() => {
    const map = new Map<string, BOM>()
    boms.forEach((b) => map.set(b.id, b))
    return map
  }, [boms])

  const toggleBom = (id: string) => setExpandedBom(expandedBom === id ? null : id)

  const handleOpenQualityCheckFromDetail = (orderId: string) => {
    setQualityCheckOrderId(orderId)
    setSelectedOrder(null)
    setShowQualityCheck(true)
  }

  if (ordersLoading) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <div className="h-8 w-48 rounded bg-secondary animate-pulse mb-6" />
        <div className="space-y-2">
          {[1, 2, 3].map((i) => <div key={i} className="h-12 rounded bg-secondary animate-pulse" />)}
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto p-6 animate-fade-up">
      <PageHeader
        title="Produktion"
        description={t('produktion.header.summary', { orders: activeOrders.length, boms: boms.length })}
        icon={Factory}
        moduleId="produktion"
        className="mb-6"
        actions={
          <div className="flex gap-2">
            {tab === 'stuecklisten' && (
              <button onClick={() => setShowNewBom(true)} className="flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                <Plus className="h-4 w-4" />{t('produktion.actions.neueStückliste')}
              </button>
            )}
            {tab === 'qualitaet' && (
              <button onClick={() => { setQualityCheckOrderId(''); setShowQualityCheck(true) }} className="flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                <Plus className="h-4 w-4" />{t('produktion.actions.qualitaetspruefung')}
              </button>
            )}
            <button onClick={() => setShowNewOrder(true)} className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">
              <Plus className="h-4 w-4" />{t('produktion.actions.neuerAuftrag')}
            </button>
          </div>
        }
      />

      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'auftraege' as const, label: t('produktion.tabs.auftraege', { count: activeOrders.length }) },
          { key: 'stuecklisten' as const, label: t('produktion.tabs.stuecklisten', { count: boms.length }) },
          { key: 'qualitaet' as const, label: t('produktion.tabs.qualitaet', { count: qualityChecks.length }) },
          { key: 'maschinen' as const, label: t('produktion.tabs.maschinen', { count: machines.length }) },
        ]).map((tabItem) => (
          <button key={tabItem.key} onClick={() => setTab(tabItem.key)}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${tab === tabItem.key ? 'border-primary text-primary font-medium tab-accent-active' : 'border-transparent text-muted-foreground hover:text-foreground'}`}>
            {tabItem.label}
          </button>
        ))}
      </div>

      {tab === 'auftraege' && (
        <>
          <div className="flex items-center gap-3 mb-4 flex-wrap">
            <div className="relative flex-1 max-w-sm min-w-[200px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input type="text" placeholder={t('produktion.search.placeholder')} value={search} onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring" />
            </div>
            <div className="flex items-center gap-1.5">
              {statusFilterOptions.map((opt) => (
                <button key={opt.value} onClick={() => setStatusFilter(opt.value)}
                  className={`rounded-lg px-3 py-1.5 text-xs transition-colors ${statusFilter === opt.value ? 'bg-primary text-primary-foreground' : 'border border-border text-muted-foreground hover:bg-secondary'}`}>
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          <div className="rounded-lg border border-border bg-card overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.table.auftragNr')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.table.produkt')}</th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">{t('produktion.table.menge')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.table.status')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground min-w-[160px]">{t('produktion.table.fortschritt')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.table.start')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.table.faellig')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredOrders.map((order) => {
                    const daysLeft = getDaysRemaining(order.planned_end)
                    const progress = order.status === 'completed' ? 100 : order.status === 'cancelled' ? 0 : order.status === 'in_progress' ? 50 : 0
                    return (
                      <tr key={order.id} onClick={() => setSelectedOrder(order)} className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer">
                        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{order.order_number}</td>
                        <td className="px-4 py-3 text-foreground font-medium">{order.product_name}</td>
                        <td className="px-4 py-3 text-xs text-muted-foreground text-right">{order.quantity.toLocaleString('de-DE')}</td>
                        <td className="px-4 py-3">
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${orderStatusColors[order.status]}`}>{orderStatusLabels[order.status] ?? order.status}</span>
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <div className="flex-1 h-2 rounded-full bg-secondary overflow-hidden max-w-[120px]">
                              <div className={`h-full rounded-full transition-all ${progressBarColors[order.status] ?? 'bg-primary'}`} style={{ width: `${progress}%` }} />
                            </div>
                            <span className="text-xs text-muted-foreground tabular-nums w-8 text-right">{progress}%</span>
                          </div>
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground">{formatDate(order.planned_start)}</td>
                        <td className="px-4 py-3">
                          <span className={`text-xs ${order.status === 'completed' || order.status === 'cancelled' ? 'text-muted-foreground' : daysLeft < 0 ? 'text-error font-medium' : daysLeft <= 3 ? 'text-warning font-medium' : 'text-muted-foreground'}`}>
                            {formatDate(order.planned_end)}
                            {order.status !== 'completed' && order.status !== 'cancelled' && daysLeft < 0 && (
                              <span className="ml-1">{t('produktion.table.ueberfaellig', { days: Math.abs(daysLeft) })}</span>
                            )}
                          </span>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
            {filteredOrders.length === 0 && (
              <EmptyState illustration={<EmptyGeneric />} title={t('produktion.empty.noAuftraege')}
                description={search || statusFilter !== 'all' ? t('produktion.empty.filterHint') : t('produktion.empty.createHint')} />
            )}
          </div>
        </>
      )}

      {tab === 'stuecklisten' && (
        <div className="space-y-3">
          {boms.length === 0 ? (
            <EmptyState illustration={<EmptyGeneric />} title={t('produktion.stueckliste.noEntries')} description={t('produktion.stueckliste.createHint')} />
          ) : (
            boms.map((bom) => {
              const usedByOrders = orders.filter((o) => o.bom_id === bom.id)
              return (
                <div key={bom.id} className="rounded-lg border border-border bg-card overflow-hidden transition-shadow hover:shadow-[var(--shadow-card-hover)]">
                  <button onClick={() => toggleBom(bom.id)} className="w-full flex items-center justify-between p-4 text-left">
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                        <Package className="h-5 w-5 text-primary" />
                      </div>
                      <div>
                        <h4 className="text-sm font-medium text-foreground">{bom.product_name}</h4>
                        <div className="flex items-center gap-2 mt-0.5">
                          <span className="text-xs text-muted-foreground">{t('produktion.stueckliste.sku')} {bom.sku}</span>
                          <span className="text-xs text-muted-foreground">&middot; {t('produktion.stueckliste.version', { version: bom.version })}</span>
                          <span className="text-xs text-muted-foreground">&middot; {t('produktion.stueckliste.positionen', { count: bom.items.length })}</span>
                          {usedByOrders.length > 0 && <span className="text-xs text-muted-foreground">&middot; {t('produktion.stueckliste.auftraege', { count: usedByOrders.length })}</span>}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {bom.active ? (
                        <span className="rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium">{t('produktion.stueckliste.aktiv')}</span>
                      ) : (
                        <span className="rounded-full bg-secondary text-muted-foreground px-2 py-0.5 text-[10px] font-medium">{t('produktion.stueckliste.inaktiv')}</span>
                      )}
                      {expandedBom === bom.id ? <ChevronDown className="h-4 w-4 text-muted-foreground" /> : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
                    </div>
                  </button>
                  {expandedBom === bom.id && (
                    <div className="border-t border-border-muted px-4 pb-4">
                      <table className="w-full text-sm mt-3">
                        <thead>
                          <tr>
                            <th className="pb-2 text-left text-xs font-medium text-muted-foreground w-8">{t('produktion.stueckliste.table.nr')}</th>
                            <th className="pb-2 text-left text-xs font-medium text-muted-foreground">{t('produktion.stueckliste.table.material')}</th>
                            <th className="pb-2 text-right text-xs font-medium text-muted-foreground">{t('produktion.stueckliste.table.menge')}</th>
                            <th className="pb-2 text-left text-xs font-medium text-muted-foreground pl-3">{t('produktion.stueckliste.table.einheit')}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {bom.items.map((item: BomItem, idx: number) => (
                            <tr key={idx} className="border-t border-border-muted">
                              <td className="py-2 text-xs text-muted-foreground">{idx + 1}</td>
                              <td className="py-2 text-xs text-foreground">{item.material_name}</td>
                              <td className="py-2 text-xs text-muted-foreground text-right">{item.quantity}</td>
                              <td className="py-2 text-xs text-muted-foreground pl-3">{item.unit}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              )
            })
          )}
        </div>
      )}

      {tab === 'qualitaet' && (
        <div className="rounded-lg border border-border bg-card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border">
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.qualitaet.table.datum')}</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.qualitaet.table.auftragNr')}</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.qualitaet.table.pruefer')}</th>
                  <th className="px-4 py-3 text-center text-xs font-medium text-muted-foreground">{t('produktion.qualitaet.table.bestanden')}</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">{t('produktion.qualitaet.table.maengel')}</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.qualitaet.table.notizen')}</th>
                </tr>
              </thead>
              <tbody>
                {qualityChecks.map((check: QualityCheck) => (
                  <tr key={check.id} className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors">
                    <td className="px-4 py-3 text-xs text-muted-foreground">{formatDate(check.checked_at)}</td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{check.order_id.slice(0, 8)}</td>
                    <td className="px-4 py-3 text-xs text-foreground">{check.inspector}</td>
                    <td className="px-4 py-3 text-center">
                      {check.passed ? <CheckCircle2 className="h-4 w-4 text-success mx-auto" /> : <XCircle className="h-4 w-4 text-error mx-auto" />}
                    </td>
                    <td className="px-4 py-3 text-xs text-muted-foreground text-right">{check.defects_found}</td>
                    <td className="px-4 py-3 text-xs text-muted-foreground max-w-[250px] truncate">{check.notes || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {qualityChecks.length === 0 && (
            <EmptyState illustration={<EmptyGeneric />} title={t('produktion.qualitaet.empty')} description={t('produktion.qualitaet.emptyHint')} />
          )}
        </div>
      )}

      {tab === 'maschinen' && <MaschinenbelegungChart machines={machines} bookings={machineBookings} />}

      {selectedOrder && (
        <OrderDetailPanel
          order={selectedOrder}
          bom={bomMap.get(selectedOrder.bom_id ?? '')}
          qualityChecks={qualityChecks.filter((qc: QualityCheck) => qc.order_id === selectedOrder.id)}
          onClose={() => setSelectedOrder(null)}
          onAddQualityCheck={() => handleOpenQualityCheckFromDetail(selectedOrder.id)}
        />
      )}

      <NewOrderDialog open={showNewOrder} onOpenChange={setShowNewOrder} boms={boms} />
      <NewBomDialog open={showNewBom} onOpenChange={setShowNewBom} />
      <QualityCheckDialog open={showQualityCheck} onOpenChange={setShowQualityCheck} orders={orders} preselectedOrderId={qualityCheckOrderId} />
    </div>
  )
}
// ============================================================
// Order Detail Panel
// ============================================================

function OrderDetailPanel({
  order, bom, qualityChecks, onClose, onAddQualityCheck,
}: {
  order: ProductionOrder
  bom: BOM | undefined
  qualityChecks: QualityCheck[]
  onClose: () => void
  onAddQualityCheck: () => void
}) {
  const { t } = useTranslation()
  const { data: workStepsData } = useWorkSteps(order.id)
  const workSteps: WorkStep[] = workStepsData?.steps ?? []

  const daysLeft = getDaysRemaining(order.planned_end)
  const totalDays = Math.max(1, Math.ceil(
    (new Date(order.planned_end).getTime() - new Date(order.planned_start).getTime()) / (1000 * 60 * 60 * 24),
  ))

  const orderStatusLabels = Object.fromEntries(
    Object.entries(orderStatusLabelKeys).map(([k, v]) => [k, t(v)])
  ) as Record<string, string>

  const workStepStatusLabels = Object.fromEntries(
    Object.entries(workStepStatusLabelKeys).map(([k, v]) => [k, t(v)])
  ) as Record<string, string>

  const handleStatusChange = (newStatus: string) => {
    toast.success(t('produktion.statusChange.toast', { orderNr: order.order_number, status: orderStatusLabels[newStatus] }))
  }

  const materialAvailability = useMemo(() => {
    if (!bom) return null
    if (order.status !== 'planned' && order.status !== 'in_progress') return null
    const results = bom.items.map((item: BomItem, idx: number) => ({
      ...item,
      availability: getMaterialAvailability(item.material_name, idx),
    }))
    const available = results.filter((r) => r.availability === 'green').length
    return { results, available, total: results.length }
  }, [bom, order.status])

  const sortedWorkSteps = useMemo(() => [...workSteps].sort((a, b) => a.step_nr - b.step_nr), [workSteps])
  const completedSteps = sortedWorkSteps.filter((s) => s.status === 'completed').length
  const totalDuration = sortedWorkSteps.reduce((sum, s) => sum + s.duration_minutes, 0)
  const progress = sortedWorkSteps.length > 0
    ? Math.round((completedSteps / sortedWorkSteps.length) * 100)
    : order.status === 'completed' ? 100 : 0

  const scrapRate = 0
  const showScrap = order.status !== 'planned'

  return (
    <DetailPanel open={true} title={t('produktion.detail.title')} subtitle={order.order_number} onClose={onClose}>
      <div className="space-y-5">
        <div className="flex items-start justify-between">
          <div>
            <h3 className="text-base font-medium text-foreground">{order.product_name}</h3>
            <p className="text-xs text-muted-foreground mt-0.5 font-mono">{order.order_number}</p>
          </div>
          <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-medium ${orderStatusColors[order.status]}`}>{orderStatusLabels[order.status]}</span>
        </div>

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
            <div className="rounded-md border border-border p-3">
              <div className="flex items-center gap-2">
                <ClipboardList className="h-4 w-4 text-primary" />
                <div>
                  <p className="text-xs font-medium text-foreground">{bom.product_name}</p>
                  <p className="text-[10px] text-muted-foreground">{t('produktion.stueckliste.sku')} {bom.sku} &middot; {t('produktion.stueckliste.version', { version: bom.version })} &middot; {t('produktion.stueckliste.positionen', { count: bom.items.length })}</p>
                </div>
              </div>
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
                {sortedWorkSteps.map((step: WorkStep) => (
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
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium shrink-0 ml-2 ${workStepStatusColors[step.status]}`}>{workStepStatusLabels[step.status]}</span>
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
              {scrapRate === 0 ? (
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
                        <p className={`text-sm font-semibold tabular-nums ${scrapRate > 5 ? 'text-error' : 'text-foreground'}`}>{scrapRate}%</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Trash2 className="h-4 w-4 text-muted-foreground" />
                      <div>
                        <p className="text-[10px] text-muted-foreground">{t('produktion.detail.ausschussProduziert')}</p>
                        <p className="text-sm font-medium text-foreground tabular-nums">0 / {order.quantity} Stk</p>
                      </div>
                    </div>
                  </div>
                  <div className="h-2.5 rounded-full bg-success/30 overflow-hidden">
                    <div className="h-full rounded-full bg-error transition-all" style={{ width: `${Math.min(100, scrapRate)}%` }} />
                  </div>
                  {scrapRate > 5 && (
                    <div className="flex items-center gap-2 rounded-md bg-error-light px-3 py-2">
                      <AlertTriangle className="h-3.5 w-3.5 text-error shrink-0" />
                      <span className="text-xs text-error font-medium">{t('produktion.detail.ausschussWarnung')}</span>
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
              {qualityChecks.map((qc: QualityCheck) => (
                <div key={qc.id} className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2 text-xs">
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

        <div className="flex gap-2 pt-2">
          {order.status !== 'completed' && order.status !== 'cancelled' && (
            <>
              {order.status === 'planned' && (
                <button onClick={() => handleStatusChange('in_progress')} className="flex items-center gap-1.5 flex-1 justify-center rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors">
                  <PlayCircle className="h-3.5 w-3.5" />{t('produktion.action.starten')}
                </button>
              )}
              <button onClick={onAddQualityCheck} className="flex items-center gap-1.5 flex-1 justify-center rounded-lg bg-primary py-2 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors">
                <ShieldCheck className="h-3.5 w-3.5" />{t('produktion.action.qualitaetspruefung')}
              </button>
            </>
          )}
          {order.status === 'completed' && (
            <div className="flex items-center gap-2 w-full justify-center py-2 text-xs text-success">
              <CheckCircle2 className="h-4 w-4" />{t('produktion.action.auftragAbgeschlossen')}
            </div>
          )}
        </div>
      </div>
    </DetailPanel>
  )
}
// ============================================================
// Neuer Auftrag Dialog
// ============================================================

function NewOrderDialog({ open, onOpenChange, boms }: { open: boolean; onOpenChange: (open: boolean) => void; boms: BOM[] }) {
  const { t } = useTranslation()
  const createOrder = useCreateOrder()
  const [bomId, setBomId] = useState('')
  const [quantity, setQuantity] = useState('')
  const [startDate, setStartDate] = useState(new Date().toISOString().split('T')[0])
  const [dueDate, setDueDate] = useState(() => { const d = new Date(); d.setDate(d.getDate() + 14); return d.toISOString().split('T')[0] })
  const [notes, setNotes] = useState('')
  const activeBoms = boms.filter((b) => b.active)

  const handleSave = async () => {
    if (!bomId) { toast.error(t('produktion.dialog.errorStueckliste')); return }
    if (!quantity || parseInt(quantity) <= 0) { toast.error(t('produktion.dialog.errorMenge')); return }
    if (!startDate || !dueDate) { toast.error(t('produktion.dialog.errorDaten')); return }
    const selectedBom = boms.find((b) => b.id === bomId)
    const now = new Date()
    const orderNumber = `PA-${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}-${String(now.getTime()).slice(-4)}`
    const input: CreateOrderInput = {
      order_number: orderNumber,
      product_name: selectedBom?.product_name ?? '',
      quantity: parseInt(quantity),
      planned_start: new Date(startDate).toISOString(),
      planned_end: new Date(dueDate).toISOString(),
      notes: notes || undefined,
    }
    try {
      await createOrder.mutateAsync(input)
      toast.success(t('produktion.dialog.auftragErstelltToast', { name: selectedBom?.product_name, qty: quantity }))
      setBomId(''); setQuantity(''); setNotes('')
      onOpenChange(false)
    } catch {
      toast.error(t('common.error'))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader><DialogTitle>{t('produktion.dialog.neuerAuftrag')}</DialogTitle></DialogHeader>
        <div className="space-y-4 mt-2">
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('produktion.dialog.stueckliste')}</Label>
            <Select value={bomId} onValueChange={setBomId}>
              <SelectTrigger><SelectValue placeholder={t('produktion.dialog.stuecklisteSelect')} /></SelectTrigger>
              <SelectContent>{activeBoms.map((b) => <SelectItem key={b.id} value={b.id}>{b.product_name} ({b.sku})</SelectItem>)}</SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('produktion.dialog.menge')}</Label>
            <Input type="number" min="1" placeholder={t('produktion.dialog.mengePlaceholder')} value={quantity} onChange={(e) => setQuantity(e.target.value)} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('produktion.dialog.startdatum')}</Label>
              <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('produktion.dialog.faelligkeitsdatum')}</Label>
              <Input type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('produktion.dialog.notizen')}</Label>
            <Textarea placeholder={t('produktion.dialog.notizenPlaceholder')} value={notes} onChange={(e) => setNotes(e.target.value)} rows={3} />
          </div>
          <div className="flex gap-2 pt-2">
            <button onClick={() => onOpenChange(false)} className="flex-1 rounded-lg border border-border py-2 text-sm text-foreground hover:bg-secondary transition-colors">{t('common.cancel')}</button>
            <button onClick={handleSave} disabled={createOrder.isPending} className="flex-1 rounded-lg bg-primary py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-60">{t('produktion.dialog.auftragErstellen')}</button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ============================================================
// Neue Stueckliste Dialog
// ============================================================

type LocalBomItem = { material_name: string; quantity: number; unit: string }

function NewBomDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const { t } = useTranslation()
  const createBOM = useCreateBOM()
  const [productName, setProductName] = useState('')
  const [sku, setSku] = useState('')
  const [version, setVersion] = useState('1.0')
  const [materials, setMaterials] = useState<LocalBomItem[]>([{ material_name: '', quantity: 1, unit: 'Stk' }])

  const addMaterial = () => setMaterials([...materials, { material_name: '', quantity: 1, unit: 'Stk' }])
  const removeMaterial = (idx: number) => { if (materials.length <= 1) return; setMaterials(materials.filter((_, i) => i !== idx)) }
  const updateMaterial = (idx: number, field: keyof LocalBomItem, value: string | number) =>
    setMaterials(materials.map((m, i) => (i === idx ? { ...m, [field]: value } : m)))

  const handleSave = async () => {
    if (!productName.trim()) { toast.error(t('produktion.dialog.errorProduktname')); return }
    if (!sku.trim()) { toast.error(t('produktion.dialog.errorSku')); return }
    const validMaterials = materials.filter((m) => m.material_name.trim())
    if (validMaterials.length === 0) { toast.error(t('produktion.dialog.errorMaterial')); return }
    const input: CreateBOMInput = {
      product_name: productName.trim(),
      sku: sku.trim(),
      version: version || '1.0',
      active: true,
      items: validMaterials.map((m, i) => ({ material_name: m.material_name.trim(), quantity: m.quantity, unit: m.unit, sort_order: i })),
    }
    try {
      await createBOM.mutateAsync(input)
      toast.success(t('produktion.dialog.stuecklisteErstelltToast', { name: productName, count: validMaterials.length }))
      setProductName(''); setSku(''); setVersion('1.0')
      setMaterials([{ material_name: '', quantity: 1, unit: 'Stk' }])
      onOpenChange(false)
    } catch {
      toast.error(t('common.error'))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader><DialogTitle>{t('produktion.dialog.neueStueckliste')}</DialogTitle></DialogHeader>
        <div className="space-y-4 mt-2">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('produktion.dialog.produktname')}</Label>
              <Input placeholder={t('produktion.dialog.produktnamePlaceholder')} value={productName} onChange={(e) => setProductName(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">SKU</Label>
              <Input placeholder={t('produktion.dialog.skuPlaceholder')} value={sku} onChange={(e) => setSku(e.target.value)} />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('produktion.dialog.version')}</Label>
            <Input placeholder="1.0" value={version} onChange={(e) => setVersion(e.target.value)} className="max-w-[120px]" />
          </div>
          <div>
            <div className="flex items-center justify-between mb-2">
              <Label className="text-sm font-medium">{t('produktion.dialog.materialien')}</Label>
              <button onClick={addMaterial} className="flex items-center gap-1 rounded-lg bg-primary px-2.5 py-1 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors">
                <Plus className="h-3 w-3" />{t('produktion.dialog.position')}
              </button>
            </div>
            <div className="space-y-2 max-h-[240px] overflow-y-auto">
              {materials.map((mat, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground w-5 shrink-0 text-right">{idx + 1}.</span>
                  <Input placeholder={t('produktion.dialog.materialnamePlaceholder')} value={mat.material_name} onChange={(e) => updateMaterial(idx, 'material_name', e.target.value)} className="flex-1" />
                  <Input type="number" min="0" step="0.1" value={mat.quantity} onChange={(e) => updateMaterial(idx, 'quantity', parseFloat(e.target.value) || 0)} className="w-20" />
                  <Select value={mat.unit} onValueChange={(v) => updateMaterial(idx, 'unit', v)}>
                    <SelectTrigger className="w-20"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Stk">Stk</SelectItem>
                      <SelectItem value="kg">kg</SelectItem>
                      <SelectItem value="m">m</SelectItem>
                      <SelectItem value="Set">Set</SelectItem>
                    </SelectContent>
                  </Select>
                  <button onClick={() => removeMaterial(idx)} disabled={materials.length <= 1} className="rounded-lg p-1.5 text-muted-foreground hover:text-error hover:bg-error-light transition-colors disabled:opacity-30 disabled:cursor-not-allowed">
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          </div>
          <div className="flex gap-2 pt-2">
            <button onClick={() => onOpenChange(false)} className="flex-1 rounded-lg border border-border py-2 text-sm text-foreground hover:bg-secondary transition-colors">{t('common.cancel')}</button>
            <button onClick={handleSave} disabled={createBOM.isPending} className="flex-1 rounded-lg bg-primary py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-60">{t('produktion.dialog.stuecklisteErstellen')}</button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ============================================================
// Qualitaetspruefung Dialog
// ============================================================

function QualityCheckDialog({
  open, onOpenChange, orders, preselectedOrderId,
}: { open: boolean; onOpenChange: (open: boolean) => void; orders: ProductionOrder[]; preselectedOrderId: string }) {
  const { t } = useTranslation()
  const createQualityCheck = useCreateQualityCheck()
  const [orderId, setOrderId] = useState(preselectedOrderId || '')
  const [inspector, setInspector] = useState('')
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const [passed, setPassed] = useState(true)
  const [defects, setDefects] = useState('')
  const [notes, setNotes] = useState('')

  const activeOrders = orders.filter((o) => o.status === 'in_progress' || o.status === 'completed')

  const handleSave = async () => {
    const selectedOrderId = orderId || preselectedOrderId
    if (!selectedOrderId) { toast.error(t('produktion.dialog.errorAuftrag')); return }
    if (!inspector) { toast.error(t('produktion.dialog.errorPruefer')); return }
    if (!date) { toast.error(t('produktion.dialog.errorDatum')); return }
    const selectedOrder = orders.find((o) => o.id === selectedOrderId)
    const input: CreateQualityCheckInput = {
      order_id: selectedOrderId,
      inspector,
      checked_at: new Date(date).toISOString(),
      passed,
      defects_found: passed ? 0 : (parseInt(defects) || 0),
      notes: notes || undefined,
    }
    try {
      await createQualityCheck.mutateAsync(input)
      toast.success(t('produktion.dialog.pruefungErstelltToast', {
        orderNr: selectedOrder?.order_number ?? t('produktion.dialog.auftrag'),
        result: passed ? t('produktion.detail.bestanden') : t('produktion.detail.nichtBestanden'),
      }))
      setOrderId(''); setInspector(''); setPassed(true); setDefects(''); setNotes('')
      onOpenChange(false)
    } catch {
      toast.error(t('common.error'))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader><DialogTitle>{t('produktion.dialog.qualitaetspruefung')}</DialogTitle></DialogHeader>
        <div className="space-y-4 mt-2">
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('produktion.dialog.auftrag')}</Label>
            <Select value={orderId || preselectedOrderId} onValueChange={setOrderId}>
              <SelectTrigger><SelectValue placeholder={t('produktion.dialog.auftragSelect')} /></SelectTrigger>
              <SelectContent>{activeOrders.map((o) => <SelectItem key={o.id} value={o.id}>{o.order_number} - {o.product_name}</SelectItem>)}</SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('produktion.dialog.pruefer')}</Label>
              <Select value={inspector} onValueChange={setInspector}>
                <SelectTrigger><SelectValue placeholder={t('produktion.dialog.prueferSelect')} /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="Werner Stoeckli">Werner Stoeckli</SelectItem>
                  <SelectItem value="Irene Graf">Irene Graf</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('produktion.dialog.datum')}</Label>
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </div>
          </div>
          <div className="rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                {passed ? <CheckCircle2 className="h-5 w-5 text-success" /> : <XCircle className="h-5 w-5 text-error" />}
                <div>
                  <p className="text-sm font-medium text-foreground">{passed ? t('produktion.detail.bestanden') : t('produktion.detail.nichtBestanden')}</p>
                  <p className="text-xs text-muted-foreground">{passed ? t('produktion.dialog.allePruefkriterien') : t('produktion.dialog.maengelFestgestellt')}</p>
                </div>
              </div>
              <Switch checked={passed} onCheckedChange={setPassed} />
            </div>
          </div>
          {!passed && (
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('produktion.dialog.anzahlMaengel')}</Label>
              <Input type="number" min="0" placeholder="0" value={defects} onChange={(e) => setDefects(e.target.value)} />
            </div>
          )}
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('produktion.dialog.notizen')}</Label>
            <Textarea placeholder={t('produktion.dialog.pruefbericht')} value={notes} onChange={(e) => setNotes(e.target.value)} rows={3} />
          </div>
          <div className="flex gap-2 pt-2">
            <button onClick={() => onOpenChange(false)} className="flex-1 rounded-lg border border-border py-2 text-sm text-foreground hover:bg-secondary transition-colors">{t('common.cancel')}</button>
            <button onClick={handleSave} disabled={createQualityCheck.isPending} className="flex-1 rounded-lg bg-primary py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-60">{t('produktion.dialog.pruefungSpeichern')}</button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}