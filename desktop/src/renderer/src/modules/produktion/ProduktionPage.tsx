import { useState, useMemo } from 'react'
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
  PauseCircle,
  Trash2,
  Cpu,
  Wrench,
  CircleDot,
  ListChecks,
  AlertTriangle,
  Gauge,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useProduktionStore,
  type ProductionOrder,
  type BOM,
  type QualityCheck,
  type BomItem,
  type WorkStep,
  type Machine,
  type MachineBooking,
} from '@/stores/produktion'
import { DetailPanel } from '@/components/shared'
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

type TabKey = 'aufträge' | 'stücklisten' | 'qualität' | 'maschinen'
type StatusFilter = 'all' | 'planned' | 'in_progress' | 'paused' | 'completed' | 'cancelled'

const orderStatusLabels: Record<string, string> = {
  planned: 'Geplant',
  in_progress: 'In Produktion',
  paused: 'Pausiert',
  completed: 'Abgeschlossen',
  cancelled: 'Storniert',
}

const orderStatusColors: Record<string, string> = {
  planned: 'bg-secondary text-muted-foreground',
  in_progress: 'bg-info-light text-info',
  paused: 'bg-warning-light text-warning',
  completed: 'bg-success-light text-success',
  cancelled: 'bg-error-light text-error',
}

const progressBarColors: Record<string, string> = {
  planned: 'bg-muted-foreground/40',
  in_progress: 'bg-info',
  paused: 'bg-warning',
  completed: 'bg-success',
  cancelled: 'bg-error',
}

const statusFilterOptions: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'Alle' },
  { value: 'planned', label: 'Geplant' },
  { value: 'in_progress', label: 'In Produktion' },
  { value: 'paused', label: 'Pausiert' },
  { value: 'completed', label: 'Abgeschlossen' },
  { value: 'cancelled', label: 'Storniert' },
]

const workStepStatusLabels: Record<string, string> = {
  pending: 'Ausstehend',
  in_progress: 'In Arbeit',
  completed: 'Erledigt',
  skipped: 'Uebersprungen',
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

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-CH')
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
  const { orders, boms, qualityChecks, workSteps, machines, machineBookings } = useProduktionStore()

  const [tab, setTab] = useState<TabKey>('aufträge')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [expandedBom, setExpandedBom] = useState<string | null>(null)

  // Detail panel
  const [selectedOrder, setSelectedOrder] = useState<ProductionOrder | null>(null)

  // Dialogs
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
      return (
        o.orderNr.toLowerCase().includes(q) ||
        o.product.toLowerCase().includes(q)
      )
    })
  }, [orders, search, statusFilter])

  const bomMap = useMemo(() => {
    const map = new Map<string, BOM>()
    boms.forEach((b) => map.set(b.id, b))
    return map
  }, [boms])

  const toggleBom = (id: string) => {
    setExpandedBom(expandedBom === id ? null : id)
  }

  const handleOpenQualityCheckFromDetail = (orderId: string) => {
    setQualityCheckOrderId(orderId)
    setSelectedOrder(null)
    setShowQualityCheck(true)
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div>
          <h1 className="text-foreground">Produktion</h1>
          <p className="text-sm text-muted-foreground">
            {activeOrders.length} aktive Aufträge &middot; {boms.length} Stücklisten
          </p>
        </div>
        <div className="flex gap-2">
          {tab === 'stücklisten' && (
            <button
              onClick={() => setShowNewBom(true)}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <Plus className="h-4 w-4" />
              Neue Stückliste
            </button>
          )}
          {tab === 'qualität' && (
            <button
              onClick={() => { setQualityCheckOrderId(''); setShowQualityCheck(true) }}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <Plus className="h-4 w-4" />
              Qualitätsprüfung
            </button>
          )}
          <button
            onClick={() => setShowNewOrder(true)}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            Neuer Auftrag
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'aufträge' as const, label: `Aufträge (${activeOrders.length} aktiv)` },
          { key: 'stücklisten' as const, label: `Stücklisten (${boms.length})` },
          { key: 'qualität' as const, label: `Qualität (${qualityChecks.length})` },
          { key: 'maschinen' as const, label: `Maschinen (${machines.length})` },
        ]).map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key ? 'border-primary text-primary font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* ============================== */}
      {/* Aufträge Tab                   */}
      {/* ============================== */}
      {tab === 'aufträge' && (
        <>
          {/* Search + Filter bar */}
          <div className="flex items-center gap-3 mb-4 flex-wrap">
            <div className="relative flex-1 max-w-sm min-w-[200px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Auftrag suchen..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="flex items-center gap-1.5">
              {statusFilterOptions.map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => setStatusFilter(opt.value)}
                  className={`rounded-lg px-3 py-1.5 text-xs transition-colors ${
                    statusFilter === opt.value
                      ? 'bg-primary text-primary-foreground'
                      : 'border border-border text-muted-foreground hover:bg-secondary'
                  }`}
                >
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
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Auftrags-Nr</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Produkt</th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">Menge</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground min-w-[160px]">Fortschritt</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Start</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Fällig</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredOrders.map((order) => {
                    const daysLeft = getDaysRemaining(order.dueDate)
                    return (
                      <tr
                        key={order.id}
                        onClick={() => setSelectedOrder(order)}
                        className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer"
                      >
                        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{order.orderNr}</td>
                        <td className="px-4 py-3 text-foreground font-medium">{order.product}</td>
                        <td className="px-4 py-3 text-xs text-muted-foreground text-right">{order.quantity.toLocaleString('de-CH')}</td>
                        <td className="px-4 py-3">
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${orderStatusColors[order.status]}`}>
                            {orderStatusLabels[order.status] ?? order.status}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <div className="flex-1 h-2 rounded-full bg-secondary overflow-hidden max-w-[120px]">
                              <div
                                className={`h-full rounded-full transition-all ${progressBarColors[order.status] ?? 'bg-primary'}`}
                                style={{ width: `${order.progress}%` }}
                              />
                            </div>
                            <span className="text-xs text-muted-foreground tabular-nums w-8 text-right">{order.progress}%</span>
                          </div>
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground">
                          {formatDate(order.startDate)}
                        </td>
                        <td className="px-4 py-3">
                          <span className={`text-xs ${
                            order.status === 'completed' || order.status === 'cancelled'
                              ? 'text-muted-foreground'
                              : daysLeft < 0
                                ? 'text-error font-medium'
                                : daysLeft <= 3
                                  ? 'text-warning font-medium'
                                  : 'text-muted-foreground'
                          }`}>
                            {formatDate(order.dueDate)}
                            {order.status !== 'completed' && order.status !== 'cancelled' && daysLeft < 0 && (
                              <span className="ml-1">({Math.abs(daysLeft)}d überfällig)</span>
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
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <Factory className="h-10 w-10 mb-3 opacity-40" />
                <p className="text-sm font-medium">Keine Aufträge gefunden</p>
                <p className="text-xs mt-1">{search || statusFilter !== 'all' ? 'Passe deine Filter an' : 'Erstelle einen neuen Auftrag'}</p>
              </div>
            )}
          </div>
        </>
      )}

      {/* ============================== */}
      {/* Stücklisten Tab                */}
      {/* ============================== */}
      {tab === 'stücklisten' && (
        <div className="space-y-3">
          {boms.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <ClipboardList className="h-10 w-10 mb-3 opacity-40" />
              <p className="text-sm font-medium">Keine Stücklisten vorhanden</p>
              <p className="text-xs mt-1">Erstelle eine neue Stückliste</p>
            </div>
          ) : (
            boms.map((bom) => {
              const usedByOrders = orders.filter((o) => o.bomId === bom.id)
              return (
                <div
                  key={bom.id}
                  className="rounded-lg border border-border bg-card overflow-hidden transition-shadow hover:shadow-[var(--shadow-card-hover)]"
                >
                  <button
                    onClick={() => toggleBom(bom.id)}
                    className="w-full flex items-center justify-between p-4 text-left"
                  >
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                        <Package className="h-5 w-5 text-primary" />
                      </div>
                      <div>
                        <h4 className="text-sm font-medium text-foreground">{bom.productName}</h4>
                        <div className="flex items-center gap-2 mt-0.5">
                          <span className="text-xs text-muted-foreground">SKU: {bom.sku}</span>
                          <span className="text-xs text-muted-foreground">&middot; v{bom.version}</span>
                          <span className="text-xs text-muted-foreground">&middot; {bom.items.length} Positionen</span>
                          {usedByOrders.length > 0 && (
                            <span className="text-xs text-muted-foreground">&middot; {usedByOrders.length} Aufträge</span>
                          )}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {bom.active ? (
                        <span className="rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium">
                          Aktiv
                        </span>
                      ) : (
                        <span className="rounded-full bg-secondary text-muted-foreground px-2 py-0.5 text-[10px] font-medium">
                          Inaktiv
                        </span>
                      )}
                      {expandedBom === bom.id ? (
                        <ChevronDown className="h-4 w-4 text-muted-foreground" />
                      ) : (
                        <ChevronRight className="h-4 w-4 text-muted-foreground" />
                      )}
                    </div>
                  </button>

                  {expandedBom === bom.id && (
                    <div className="border-t border-border-muted px-4 pb-4">
                      <table className="w-full text-sm mt-3">
                        <thead>
                          <tr>
                            <th className="pb-2 text-left text-xs font-medium text-muted-foreground w-8">#</th>
                            <th className="pb-2 text-left text-xs font-medium text-muted-foreground">Material</th>
                            <th className="pb-2 text-right text-xs font-medium text-muted-foreground">Menge</th>
                            <th className="pb-2 text-left text-xs font-medium text-muted-foreground pl-3">Einheit</th>
                          </tr>
                        </thead>
                        <tbody>
                          {bom.items.map((item, idx) => (
                            <tr key={idx} className="border-t border-border-muted">
                              <td className="py-2 text-xs text-muted-foreground">{idx + 1}</td>
                              <td className="py-2 text-xs text-foreground">{item.materialName}</td>
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

      {/* ============================== */}
      {/* Qualität Tab                   */}
      {/* ============================== */}
      {tab === 'qualität' && (
        <div className="rounded-lg border border-border bg-card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border">
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Datum</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Auftrags-Nr</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Prüfer</th>
                  <th className="px-4 py-3 text-center text-xs font-medium text-muted-foreground">Bestanden</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">Mängel</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Notizen</th>
                </tr>
              </thead>
              <tbody>
                {qualityChecks.map((check) => (
                  <tr
                    key={check.id}
                    className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors"
                  >
                    <td className="px-4 py-3 text-xs text-muted-foreground">
                      {formatDate(check.date)}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{check.orderNr}</td>
                    <td className="px-4 py-3 text-xs text-foreground">{check.inspector}</td>
                    <td className="px-4 py-3 text-center">
                      {check.passed ? (
                        <CheckCircle2 className="h-4 w-4 text-success mx-auto" />
                      ) : (
                        <XCircle className="h-4 w-4 text-error mx-auto" />
                      )}
                    </td>
                    <td className="px-4 py-3 text-xs text-muted-foreground text-right">{check.defectsFound}</td>
                    <td className="px-4 py-3 text-xs text-muted-foreground max-w-[250px] truncate">
                      {check.notes || '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {qualityChecks.length === 0 && (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <ShieldCheck className="h-10 w-10 mb-3 opacity-40" />
              <p className="text-sm font-medium">Keine Qualitätsprüfungen</p>
              <p className="text-xs mt-1">Prüfungen werden hier aufgelistet</p>
            </div>
          )}
        </div>
      )}

      {/* ============================== */}
      {/* Maschinen Tab                  */}
      {/* ============================== */}
      {tab === 'maschinen' && (
        <MaschinenbelegungChart machines={machines} bookings={machineBookings} />
      )}

      {/* ============================== */}
      {/* Order Detail Panel             */}
      {/* ============================== */}
      {selectedOrder && (
        <OrderDetailPanel
          order={selectedOrder}
          bom={bomMap.get(selectedOrder.bomId)}
          qualityChecks={qualityChecks.filter((qc) => qc.orderId === selectedOrder.id)}
          workSteps={workSteps.filter((ws) => ws.orderId === selectedOrder.id)}
          onClose={() => setSelectedOrder(null)}
          onAddQualityCheck={() => handleOpenQualityCheckFromDetail(selectedOrder.id)}
        />
      )}

      {/* ============================== */}
      {/* Neuer Auftrag Dialog           */}
      {/* ============================== */}
      <NewOrderDialog
        open={showNewOrder}
        onOpenChange={setShowNewOrder}
        boms={boms}
      />

      {/* ============================== */}
      {/* Stückliste erstellen Dialog    */}
      {/* ============================== */}
      <NewBomDialog
        open={showNewBom}
        onOpenChange={setShowNewBom}
      />

      {/* ============================== */}
      {/* Qualitätsprüfung Dialog        */}
      {/* ============================== */}
      <QualityCheckDialog
        open={showQualityCheck}
        onOpenChange={setShowQualityCheck}
        orders={orders}
        preselectedOrderId={qualityCheckOrderId}
      />
    </div>
  )
}

// ============================================================
// Order Detail Panel
// ============================================================

function OrderDetailPanel({
  order,
  bom,
  qualityChecks,
  workSteps,
  onClose,
  onAddQualityCheck,
}: {
  order: ProductionOrder
  bom: BOM | undefined
  qualityChecks: QualityCheck[]
  workSteps: WorkStep[]
  onClose: () => void
  onAddQualityCheck: () => void
}) {
  const daysLeft = getDaysRemaining(order.dueDate)
  const totalDays = Math.ceil(
    (new Date(order.dueDate).getTime() - new Date(order.startDate).getTime()) / (1000 * 60 * 60 * 24),
  )

  const handleStatusChange = (newStatus: string) => {
    toast.success(`Status von ${order.orderNr} auf "${orderStatusLabels[newStatus]}" geändert`)
  }

  // --- 8.14 Material Availability ---
  const materialAvailability = useMemo(() => {
    if (!bom) return null
    if (order.status !== 'planned' && order.status !== 'in_progress') return null

    const results = bom.items.map((item, idx) => ({
      ...item,
      availability: getMaterialAvailability(item.materialName, idx),
    }))
    const available = results.filter((r) => r.availability === 'green').length
    return { results, available, total: results.length }
  }, [bom, order.status])

  // --- 8.15 Work Steps ---
  const sortedWorkSteps = useMemo(() => {
    return [...workSteps].sort((a, b) => a.stepNr - b.stepNr)
  }, [workSteps])

  const completedSteps = sortedWorkSteps.filter((s) => s.status === 'completed').length
  const totalDuration = sortedWorkSteps.reduce((sum, s) => sum + s.durationMinutes, 0)

  // --- 8.17 Scrap tracking ---
  const showScrap = order.status !== 'planned'

  return (
    <DetailPanel open={true} title="Auftragsdetails" subtitle={order.orderNr} onClose={onClose}>
      <div className="space-y-5">
        {/* Header info */}
        <div className="flex items-start justify-between">
          <div>
            <h3 className="text-base font-medium text-foreground">{order.product}</h3>
            <p className="text-xs text-muted-foreground mt-0.5 font-mono">{order.orderNr}</p>
          </div>
          <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-medium ${orderStatusColors[order.status]}`}>
            {orderStatusLabels[order.status]}
          </span>
        </div>

        {/* Large progress bar */}
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <span className="text-xs font-medium text-foreground">Fortschritt</span>
            <span className="text-sm font-semibold text-foreground tabular-nums">{order.progress}%</span>
          </div>
          <div className="h-3 rounded-full bg-secondary overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${progressBarColors[order.status] ?? 'bg-primary'}`}
              style={{ width: `${order.progress}%` }}
            />
          </div>
        </div>

        {/* Key info grid */}
        <div className="grid grid-cols-2 gap-3 text-xs">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Package className="h-3.5 w-3.5 shrink-0" />
            <div>
              <p className="text-[10px] text-muted-foreground">Menge</p>
              <p className="text-foreground font-medium">{order.quantity.toLocaleString('de-CH')} Stk</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <Calendar className="h-3.5 w-3.5 shrink-0" />
            <div>
              <p className="text-[10px] text-muted-foreground">Startdatum</p>
              <p className="text-foreground">{formatDate(order.startDate)}</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <Clock className="h-3.5 w-3.5 shrink-0" />
            <div>
              <p className="text-[10px] text-muted-foreground">Fälligkeitsdatum</p>
              <p className={`${
                order.status !== 'completed' && order.status !== 'cancelled' && daysLeft < 0
                  ? 'text-error font-medium'
                  : 'text-foreground'
              }`}>
                {formatDate(order.dueDate)}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <AlertCircle className="h-3.5 w-3.5 shrink-0" />
            <div>
              <p className="text-[10px] text-muted-foreground">Verbleibend</p>
              <p className={`font-medium ${
                order.status === 'completed'
                  ? 'text-success'
                  : order.status === 'cancelled'
                    ? 'text-muted-foreground'
                    : daysLeft < 0
                      ? 'text-error'
                      : daysLeft <= 3
                        ? 'text-warning'
                        : 'text-foreground'
              }`}>
                {order.status === 'completed'
                  ? 'Abgeschlossen'
                  : order.status === 'cancelled'
                    ? 'Storniert'
                    : daysLeft < 0
                      ? `${Math.abs(daysLeft)} Tage überfällig`
                      : daysLeft === 0
                        ? 'Heute fällig'
                        : `${daysLeft} Tage`}
              </p>
            </div>
          </div>
        </div>

        {/* Timeline info */}
        <div className="rounded-md border border-border p-3">
          <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
            <span>{formatDate(order.startDate)}</span>
            <span>{totalDays} Tage Laufzeit</span>
            <span>{formatDate(order.dueDate)}</span>
          </div>
          <div className="h-1.5 rounded-full bg-secondary overflow-hidden">
            <div
              className="h-full rounded-full bg-primary/60 transition-all"
              style={{
                width: `${Math.min(100, Math.max(0, ((totalDays - Math.max(0, daysLeft)) / totalDays) * 100))}%`,
              }}
            />
          </div>
        </div>

        {/* ============================== */}
        {/* 8.14 Material Availability     */}
        {/* ============================== */}
        {materialAvailability && bom && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              Materialverfuegbarkeit
            </h4>
            <div className="rounded-md border border-border p-3 space-y-3">
              {/* Summary */}
              <div className="flex items-center gap-2">
                <Package className="h-4 w-4 text-primary" />
                <span className="text-xs font-medium text-foreground">
                  {materialAvailability.available} von {materialAvailability.total} Materialien verfuegbar
                </span>
              </div>

              {/* Per-item list */}
              <div className="space-y-1">
                {materialAvailability.results.map((item, idx) => {
                  const colorMap = {
                    green: { dot: 'bg-success', text: 'text-success', bg: 'bg-success-light', label: 'Verfuegbar' },
                    yellow: { dot: 'bg-warning', text: 'text-warning', bg: 'bg-warning-light', label: 'Knapp' },
                    red: { dot: 'bg-error', text: 'text-error', bg: 'bg-error-light', label: 'Fehlt' },
                  }
                  const c = colorMap[item.availability]
                  return (
                    <div key={idx} className="flex items-center justify-between py-1">
                      <div className="flex items-center gap-2">
                        <span className={`h-2 w-2 rounded-full ${c.dot} shrink-0`} />
                        <span className="text-xs text-foreground">{item.materialName}</span>
                      </div>
                      <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${c.bg} ${c.text}`}>
                        {c.label}
                      </span>
                    </div>
                  )
                })}
              </div>
            </div>
          </section>
        )}

        {/* Associated BOM */}
        {bom && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              Stückliste
            </h4>
            <div className="rounded-md border border-border p-3">
              <div className="flex items-center gap-2">
                <ClipboardList className="h-4 w-4 text-primary" />
                <div>
                  <p className="text-xs font-medium text-foreground">{bom.productName}</p>
                  <p className="text-[10px] text-muted-foreground">
                    SKU: {bom.sku} &middot; v{bom.version} &middot; {bom.items.length} Positionen
                  </p>
                </div>
              </div>
            </div>
          </section>
        )}

        {/* ============================== */}
        {/* 8.15 Work Steps / Arbeitsgaenge*/}
        {/* ============================== */}
        {sortedWorkSteps.length > 0 && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              Arbeitsgaenge
            </h4>
            <div className="rounded-md border border-border p-3 space-y-3">
              {/* Summary bar */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <ListChecks className="h-4 w-4 text-primary" />
                  <span className="text-xs font-medium text-foreground">
                    {completedSteps} von {sortedWorkSteps.length} Schritten erledigt
                  </span>
                </div>
                <span className="text-xs text-muted-foreground">
                  Gesamt: {formatDuration(totalDuration)}
                </span>
              </div>

              {/* Progress bar */}
              <div className="h-1.5 rounded-full bg-secondary overflow-hidden">
                <div
                  className="h-full rounded-full bg-success transition-all"
                  style={{ width: sortedWorkSteps.length > 0 ? `${(completedSteps / sortedWorkSteps.length) * 100}%` : '0%' }}
                />
              </div>

              {/* Steps list */}
              <div className="space-y-1">
                {sortedWorkSteps.map((step) => (
                  <div
                    key={step.id}
                    className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2"
                  >
                    <div className="flex items-center gap-2.5 min-w-0">
                      <span className="text-[10px] text-muted-foreground font-mono w-4 shrink-0 text-right">
                        {step.stepNr}
                      </span>
                      <div className="min-w-0">
                        <p className="text-xs text-foreground font-medium truncate">{step.name}</p>
                        <div className="flex items-center gap-2 mt-0.5">
                          <span className="text-[10px] text-muted-foreground">{formatDuration(step.durationMinutes)}</span>
                          {step.assignee && (
                            <span className="text-[10px] text-muted-foreground">&middot; {step.assignee}</span>
                          )}
                        </div>
                      </div>
                    </div>
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium shrink-0 ml-2 ${workStepStatusColors[step.status]}`}>
                      {workStepStatusLabels[step.status]}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </section>
        )}

        {/* ============================== */}
        {/* 8.17 Ausschuss-Tracking        */}
        {/* ============================== */}
        {showScrap && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              Ausschuss
            </h4>
            <div className="rounded-md border border-border p-3 space-y-3">
              {order.scrapRate === 0 ? (
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-success" />
                  <span className="text-xs text-muted-foreground">Kein Ausschuss erfasst</span>
                </div>
              ) : (
                <>
                  {/* Stats row */}
                  <div className="flex items-center gap-4">
                    <div className="flex items-center gap-2">
                      <Gauge className="h-4 w-4 text-muted-foreground" />
                      <div>
                        <p className="text-[10px] text-muted-foreground">Ausschussrate</p>
                        <p className={`text-sm font-semibold tabular-nums ${order.scrapRate > 5 ? 'text-error' : 'text-foreground'}`}>
                          {order.scrapRate}%
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Trash2 className="h-4 w-4 text-muted-foreground" />
                      <div>
                        <p className="text-[10px] text-muted-foreground">Ausschuss / Produziert</p>
                        <p className="text-sm font-medium text-foreground tabular-nums">
                          {order.scrapQuantity} / {order.quantity} Stk
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Scrap rate bar */}
                  <div>
                    <div className="h-2.5 rounded-full bg-success/30 overflow-hidden">
                      <div
                        className="h-full rounded-full bg-error transition-all"
                        style={{ width: `${Math.min(100, order.scrapRate)}%` }}
                      />
                    </div>
                    <div className="flex items-center justify-between mt-1">
                      <span className="text-[10px] text-muted-foreground">0%</span>
                      <span className="text-[10px] text-muted-foreground">100%</span>
                    </div>
                  </div>

                  {/* Warning if above target */}
                  {order.scrapRate > 5 && (
                    <div className="flex items-center gap-2 rounded-md bg-error-light px-3 py-2">
                      <AlertTriangle className="h-3.5 w-3.5 text-error shrink-0" />
                      <span className="text-xs text-error font-medium">
                        Ausschussrate ueber Zielwert (5%)
                      </span>
                    </div>
                  )}
                </>
              )}
            </div>
          </section>
        )}

        {/* Quality checks for this order */}
        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
            Qualitätsprüfungen ({qualityChecks.length})
          </h4>
          {qualityChecks.length === 0 ? (
            <p className="text-xs text-muted-foreground">Keine Prüfungen vorhanden</p>
          ) : (
            <div className="space-y-1.5">
              {qualityChecks.map((qc) => (
                <div key={qc.id} className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2 text-xs">
                  <div className="flex items-center gap-2">
                    {qc.passed ? (
                      <CheckCircle2 className="h-3.5 w-3.5 text-success" />
                    ) : (
                      <XCircle className="h-3.5 w-3.5 text-error" />
                    )}
                    <div>
                      <p className="text-foreground">{qc.inspector}</p>
                      <p className="text-[10px] text-muted-foreground">
                        {formatDate(qc.date)}
                        {qc.defectsFound > 0 && ` · ${qc.defectsFound} Mängel`}
                      </p>
                    </div>
                  </div>
                  <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${qc.passed ? 'bg-success-light text-success' : 'bg-error-light text-error'}`}>
                    {qc.passed ? 'Bestanden' : 'Nicht bestanden'}
                  </span>
                </div>
              ))}
            </div>
          )}
        </section>

        {/* Actions */}
        <div className="flex gap-2 pt-2">
          {order.status !== 'completed' && order.status !== 'cancelled' && (
            <>
              {order.status === 'in_progress' && (
                <button
                  onClick={() => handleStatusChange('paused')}
                  className="flex items-center gap-1.5 flex-1 justify-center rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
                >
                  <PauseCircle className="h-3.5 w-3.5" />
                  Pausieren
                </button>
              )}
              {order.status === 'paused' && (
                <button
                  onClick={() => handleStatusChange('in_progress')}
                  className="flex items-center gap-1.5 flex-1 justify-center rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
                >
                  <PlayCircle className="h-3.5 w-3.5" />
                  Fortsetzen
                </button>
              )}
              {order.status === 'planned' && (
                <button
                  onClick={() => handleStatusChange('in_progress')}
                  className="flex items-center gap-1.5 flex-1 justify-center rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
                >
                  <PlayCircle className="h-3.5 w-3.5" />
                  Starten
                </button>
              )}
              <button
                onClick={onAddQualityCheck}
                className="flex items-center gap-1.5 flex-1 justify-center rounded-lg bg-primary py-2 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <ShieldCheck className="h-3.5 w-3.5" />
                Qualitätsprüfung
              </button>
            </>
          )}
          {order.status === 'completed' && (
            <div className="flex items-center gap-2 w-full justify-center py-2 text-xs text-success">
              <CheckCircle2 className="h-4 w-4" />
              Auftrag abgeschlossen
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

function NewOrderDialog({
  open,
  onOpenChange,
  boms,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  boms: BOM[]
}) {
  const [bomId, setBomId] = useState('')
  const [quantity, setQuantity] = useState('')
  const [startDate, setStartDate] = useState(new Date().toISOString().split('T')[0])
  const [dueDate, setDueDate] = useState(() => {
    const d = new Date()
    d.setDate(d.getDate() + 14)
    return d.toISOString().split('T')[0]
  })
  const [notes, setNotes] = useState('')

  const activeBoms = boms.filter((b) => b.active)

  const handleSave = () => {
    if (!bomId) {
      toast.error('Bitte wähle eine Stückliste aus')
      return
    }
    if (!quantity || parseInt(quantity) <= 0) {
      toast.error('Bitte gib eine gültige Menge ein')
      return
    }
    if (!startDate || !dueDate) {
      toast.error('Bitte gib Start- und Fälligkeitsdatum ein')
      return
    }

    const selectedBom = boms.find((b) => b.id === bomId)
    toast.success(`Auftrag für "${selectedBom?.productName}" mit ${quantity} Stk erstellt`)
    setBomId('')
    setQuantity('')
    setNotes('')
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle>Neuer Produktionsauftrag</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 mt-2">
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">Stückliste / Produkt</Label>
            <Select value={bomId} onValueChange={setBomId}>
              <SelectTrigger>
                <SelectValue placeholder="Stückliste auswählen..." />
              </SelectTrigger>
              <SelectContent>
                {activeBoms.map((b) => (
                  <SelectItem key={b.id} value={b.id}>
                    {b.productName} ({b.sku})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label className="text-sm font-medium">Menge</Label>
            <Input
              type="number"
              min="1"
              placeholder="z.B. 50"
              value={quantity}
              onChange={(e) => setQuantity(e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">Startdatum</Label>
              <Input
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">Fälligkeitsdatum</Label>
              <Input
                type="date"
                value={dueDate}
                onChange={(e) => setDueDate(e.target.value)}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label className="text-sm font-medium">Notizen</Label>
            <Textarea
              placeholder="Optionale Anmerkungen zum Auftrag..."
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
            />
          </div>

          <div className="flex gap-2 pt-2">
            <button
              onClick={() => onOpenChange(false)}
              className="flex-1 rounded-lg border border-border py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={handleSave}
              className="flex-1 rounded-lg bg-primary py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              Auftrag erstellen
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ============================================================
// Neue Stückliste Dialog
// ============================================================

function NewBomDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [productName, setProductName] = useState('')
  const [sku, setSku] = useState('')
  const [version, setVersion] = useState('1.0')
  const [materials, setMaterials] = useState<BomItem[]>([
    { materialName: '', quantity: 1, unit: 'Stk' },
  ])

  const addMaterial = () => {
    setMaterials([...materials, { materialName: '', quantity: 1, unit: 'Stk' }])
  }

  const removeMaterial = (idx: number) => {
    if (materials.length <= 1) return
    setMaterials(materials.filter((_, i) => i !== idx))
  }

  const updateMaterial = (idx: number, field: keyof BomItem, value: string | number) => {
    setMaterials(materials.map((m, i) => (i === idx ? { ...m, [field]: value } : m)))
  }

  const handleSave = () => {
    if (!productName.trim()) {
      toast.error('Bitte gib einen Produktnamen ein')
      return
    }
    if (!sku.trim()) {
      toast.error('Bitte gib eine SKU ein')
      return
    }
    const validMaterials = materials.filter((m) => m.materialName.trim())
    if (validMaterials.length === 0) {
      toast.error('Mindestens ein Material ist erforderlich')
      return
    }

    toast.success(`Stückliste "${productName}" mit ${validMaterials.length} Positionen erstellt`)
    setProductName('')
    setSku('')
    setVersion('1.0')
    setMaterials([{ materialName: '', quantity: 1, unit: 'Stk' }])
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>Neue Stückliste</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 mt-2">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">Produktname</Label>
              <Input
                placeholder="z.B. Schaltschrank Pro"
                value={productName}
                onChange={(e) => setProductName(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">SKU</Label>
              <Input
                placeholder="z.B. SS-PRO-01"
                value={sku}
                onChange={(e) => setSku(e.target.value)}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label className="text-sm font-medium">Version</Label>
            <Input
              placeholder="1.0"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              className="max-w-[120px]"
            />
          </div>

          {/* Materials list */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <Label className="text-sm font-medium">Materialien</Label>
              <button
                onClick={addMaterial}
                className="flex items-center gap-1 rounded-lg bg-primary px-2.5 py-1 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Plus className="h-3 w-3" />
                Position
              </button>
            </div>
            <div className="space-y-2 max-h-[240px] overflow-y-auto">
              {materials.map((mat, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground w-5 shrink-0 text-right">{idx + 1}.</span>
                  <Input
                    placeholder="Materialname"
                    value={mat.materialName}
                    onChange={(e) => updateMaterial(idx, 'materialName', e.target.value)}
                    className="flex-1"
                  />
                  <Input
                    type="number"
                    min="0"
                    step="0.1"
                    value={mat.quantity}
                    onChange={(e) => updateMaterial(idx, 'quantity', parseFloat(e.target.value) || 0)}
                    className="w-20"
                  />
                  <Select
                    value={mat.unit}
                    onValueChange={(v) => updateMaterial(idx, 'unit', v)}
                  >
                    <SelectTrigger className="w-20">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Stk">Stk</SelectItem>
                      <SelectItem value="kg">kg</SelectItem>
                      <SelectItem value="m">m</SelectItem>
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

          <div className="flex gap-2 pt-2">
            <button
              onClick={() => onOpenChange(false)}
              className="flex-1 rounded-lg border border-border py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={handleSave}
              className="flex-1 rounded-lg bg-primary py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              Stückliste erstellen
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ============================================================
// Qualitätsprüfung Dialog
// ============================================================

function QualityCheckDialog({
  open,
  onOpenChange,
  orders,
  preselectedOrderId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  orders: ProductionOrder[]
  preselectedOrderId: string
}) {
  const [orderId, setOrderId] = useState(preselectedOrderId || '')
  const [inspector, setInspector] = useState('')
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const [passed, setPassed] = useState(true)
  const [defects, setDefects] = useState('')
  const [notes, setNotes] = useState('')

  // Sync preselected order when dialog opens
  const activeOrders = orders.filter(
    (o) => o.status === 'in_progress' || o.status === 'paused' || o.status === 'completed',
  )

  const handleSave = () => {
    const selectedOrderId = orderId || preselectedOrderId
    if (!selectedOrderId) {
      toast.error('Bitte wähle einen Auftrag aus')
      return
    }
    if (!inspector) {
      toast.error('Bitte wähle einen Prüfer aus')
      return
    }
    if (!date) {
      toast.error('Bitte gib ein Datum ein')
      return
    }

    const selectedOrder = orders.find((o) => o.id === selectedOrderId)
    toast.success(
      `Qualitätsprüfung für ${selectedOrder?.orderNr ?? 'Auftrag'} erfasst — ${passed ? 'Bestanden' : 'Nicht bestanden'}`,
    )
    setOrderId('')
    setInspector('')
    setPassed(true)
    setDefects('')
    setNotes('')
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle>Qualitätsprüfung erfassen</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 mt-2">
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">Auftrag</Label>
            <Select
              value={orderId || preselectedOrderId}
              onValueChange={setOrderId}
            >
              <SelectTrigger>
                <SelectValue placeholder="Auftrag auswählen..." />
              </SelectTrigger>
              <SelectContent>
                {activeOrders.map((o) => (
                  <SelectItem key={o.id} value={o.id}>
                    {o.orderNr} — {o.product}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">Prüfer</Label>
              <Select value={inspector} onValueChange={setInspector}>
                <SelectTrigger>
                  <SelectValue placeholder="Prüfer auswählen..." />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="Werner Stöckli">Werner Stöckli</SelectItem>
                  <SelectItem value="Irene Graf">Irene Graf</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">Datum</Label>
              <Input
                type="date"
                value={date}
                onChange={(e) => setDate(e.target.value)}
              />
            </div>
          </div>

          {/* Pass/Fail toggle */}
          <div className="rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                {passed ? (
                  <CheckCircle2 className="h-5 w-5 text-success" />
                ) : (
                  <XCircle className="h-5 w-5 text-error" />
                )}
                <div>
                  <p className="text-sm font-medium text-foreground">
                    {passed ? 'Bestanden' : 'Nicht bestanden'}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {passed ? 'Alle Prüfkriterien erfüllt' : 'Mängel festgestellt'}
                  </p>
                </div>
              </div>
              <Switch checked={passed} onCheckedChange={setPassed} />
            </div>
          </div>

          {/* Defects count (only if not passed) */}
          {!passed && (
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">Anzahl Mängel</Label>
              <Input
                type="number"
                min="0"
                placeholder="0"
                value={defects}
                onChange={(e) => setDefects(e.target.value)}
              />
            </div>
          )}

          <div className="space-y-1.5">
            <Label className="text-sm font-medium">Notizen</Label>
            <Textarea
              placeholder="Prüfbericht, Bemerkungen, Massnahmen..."
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
            />
          </div>

          <div className="flex gap-2 pt-2">
            <button
              onClick={() => onOpenChange(false)}
              className="flex-1 rounded-lg border border-border py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={handleSave}
              className="flex-1 rounded-lg bg-primary py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              Prüfung speichern
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
