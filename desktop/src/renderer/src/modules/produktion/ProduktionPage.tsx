import { useState, useMemo, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Plus,
  Factory,
  CheckCircle2,
  XCircle,
  ChevronDown,
  ChevronRight,
  Package,
  Trash2,
  Download,
} from 'lucide-react'
import { toast } from 'sonner'
import { useQueries } from '@tanstack/react-query'
import {
  useOrders,
  useBOMs,
  useQualityChecks,
  useMachines,
  useMachineBookings,
  useCreateOrder,
  useCreateBOM,
  useCreateQualityCheck,
  useCreateMachine,
} from '@/api/hooks/useProduktion'
import { listWorkSteps } from '@/api/produktion-client'
import type {
  ProductionOrder,
  BOMResponse as BOM,
  QualityCheckResponse as QualityCheck,
  BomItemResponse as BomItem,
  CreateOrderInput,
  CreateBOMInput,
  CreateQualityCheckInput,
} from '@/api/produktion-types'
import { PageHeader, EmptyState, SortMenu, type SortDirection } from '@/components/shared'
import { EmptyGeneric } from '@/components/shared/illustrations'
import { useCapabilitySet, useHasCapability } from '@/hooks/useCapability'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'
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
import {
  OrderDetailModal,
  BomDetailModal,
  QualityCheckDetailModal,
  MachineDetailModal,
} from './ProduktionDetailModals'
import {
  orderStatusLabelKeys,
  orderStatusColors,
  progressBarColors,
  priorityLabelKeys,
  priorityColors,
  PRIORITY_VALUES,
  getDaysRemaining,
  orderNumberById,
  generateOrderNumber,
} from './produktion-shared'
import { buildOrdersCsv, downloadBlob, csvDateStamp } from './produktion-export'
import { useProduktionPrefsStore } from '@/stores/produktionPrefs'
import { useProduktionTenantStore } from '@/stores/produktionTenant'
import { formatDate } from '@/lib/format'

type TabKey = 'auftraege' | 'stuecklisten' | 'qualitaet' | 'maschinen'
type StatusFilter = 'all' | 'planned' | 'in_progress' | 'completed' | 'cancelled'

const statusFilterOptionKeys: { value: StatusFilter; labelKey: string }[] = [
  { value: 'all', labelKey: 'produktion.filter.all' },
  { value: 'planned', labelKey: 'produktion.filter.planned' },
  { value: 'in_progress', labelKey: 'produktion.filter.inProgress' },
  { value: 'completed', labelKey: 'produktion.filter.completed' },
  { value: 'cancelled', labelKey: 'produktion.filter.cancelled' },
]

/** Detail navigation chain (order ↔ bom / qc / machine) rendered as DetailModal. */
type DetailTarget =
  | { type: 'order'; id: string }
  | { type: 'bom'; id: string }
  | { type: 'qc'; id: string }
  | { type: 'machine'; id: string }

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

  const orders = useMemo(() => ordersData?.orders ?? [], [ordersData])
  const boms = useMemo(() => bomsData?.boms ?? [], [bomsData])
  const qualityChecks = qualityData?.checks ?? []
  const machines = machinesData?.machines ?? []
  const machineBookings = bookingsData?.bookings ?? []

  const defaultTab = useProduktionPrefsStore((s) => s.defaultTab)
  const defaultStatusFilter = useProduktionPrefsStore((s) => s.defaultStatusFilter)

  // RBAC R-3 capability checks (default hidden per gating convention)
  const { has: capHas, ready: capReady } = useCapabilitySet()
  const canCreateOrder = useHasCapability('produktion:order:create')
  const canCreateBom = useHasCapability('produktion:bom:create')
  const canCreateQuality = useHasCapability('produktion:quality:create')
  const canManageMachine = useHasCapability('produktion:machine:manage')
  const canExport = useHasCapability('produktion:export:run')

  // RBAC tab gating — while permissions load keep all tabs visible (no flicker)
  const TAB_CAPABILITY: Record<TabKey, string> = {
    auftraege: 'produktion:order:read',
    stuecklisten: 'produktion:bom:read',
    qualitaet: 'produktion:quality:read',
    maschinen: 'produktion:machine:read',
  }
  const visibleTabs = (Object.keys(TAB_CAPABILITY) as TabKey[]).filter(
    (key) => !capReady || capHas(TAB_CAPABILITY[key]),
  )

  const orderStatusLabels = useMemo(() => Object.fromEntries(
    Object.entries(orderStatusLabelKeys).map(([k, v]) => [k, t(v)])
  ) as Record<string, string>, [t])

  const priorityLabels = useMemo(() => Object.fromEntries(
    Object.entries(priorityLabelKeys).map(([k, v]) => [Number(k), t(v)])
  ) as Record<number, string>, [t])

  const statusFilterOptions = useMemo(() =>
    statusFilterOptionKeys.map((o) => ({ value: o.value, label: t(o.labelKey) })),
    [t]
  )

  const [tab, setTab] = useState<TabKey>(defaultTab)

  useEffect(() => {
    if (!capReady) return
    if (visibleTabs.length > 0 && !visibleTabs.includes(tab)) setTab(visibleTabs[0])
  }, [capReady, visibleTabs, tab])

  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>(defaultStatusFilter)
  const [sortField, setSortField] = useState('plannedEnd')
  const [sortDir, setSortDir] = useState<SortDirection>('asc')
  const [expandedBom, setExpandedBom] = useState<string | null>(null)
  const [detailStack, setDetailStack] = useState<DetailTarget[]>([])
  const [showNewOrder, setShowNewOrder] = useState(false)
  const [showNewBom, setShowNewBom] = useState(false)
  const [showNewMachine, setShowNewMachine] = useState(false)
  const [showQualityCheck, setShowQualityCheck] = useState(false)
  const [qualityCheckOrderId, setQualityCheckOrderId] = useState<string>('')

  const currentDetail = detailStack[detailStack.length - 1]
  const pushDetail = (target: DetailTarget) => setDetailStack((s) => [...s, target])
  const goBack = () => setDetailStack((s) => s.slice(0, -1))
  const closeDetail = () => setDetailStack([])

  const activeOrders = orders.filter((o) => o.status === 'in_progress' || o.status === 'planned')

  // Real progress for running orders: completed steps / total steps. The list
  // used to hard-code 50 % for every in_progress order — fetch the (few)
  // running orders' steps instead so the bar reflects reality.
  const inProgressOrders = useMemo(
    () => orders.filter((o) => o.status === 'in_progress'),
    [orders],
  )
  const stepQueries = useQueries({
    queries: inProgressOrders.map((o) => ({
      queryKey: ['produktion', 'worksteps', o.id],
      queryFn: () => listWorkSteps(o.id),
    })),
  })
  const progressById = new Map<string, number>()
  inProgressOrders.forEach((o, i) => {
    const steps = stepQueries[i]?.data?.steps ?? []
    if (steps.length > 0) {
      progressById.set(o.id, Math.round((steps.filter((s) => s.status === 'completed').length / steps.length) * 100))
    }
  })
  orders.forEach((o) => {
    if (!progressById.has(o.id)) {
      progressById.set(o.id, o.status === 'completed' ? 100 : 0)
    }
  })

  const sortOptions = useMemo(() => [
    { value: 'orderNumber', label: t('produktion.sort.orderNumber') },
    { value: 'product', label: t('produktion.sort.product') },
    { value: 'quantity', label: t('produktion.sort.quantity') },
    { value: 'priority', label: t('produktion.sort.priority') },
    { value: 'plannedStart', label: t('produktion.sort.plannedStart') },
    { value: 'plannedEnd', label: t('produktion.sort.plannedEnd') },
  ], [t])

  const filteredOrders = useMemo(() => {
    const filtered = orders.filter((o) => {
      if (statusFilter !== 'all' && o.status !== statusFilter) return false
      if (!search) return true
      const q = search.toLowerCase()
      return o.order_number.toLowerCase().includes(q) || o.product_name.toLowerCase().includes(q)
    })
    const dir = sortDir === 'asc' ? 1 : -1
    return [...filtered].sort((a, b) => {
      switch (sortField) {
        case 'orderNumber': return dir * a.order_number.localeCompare(b.order_number)
        case 'product': return dir * a.product_name.localeCompare(b.product_name)
        case 'quantity': return dir * (a.quantity - b.quantity)
        case 'priority': return dir * (a.priority - b.priority)
        case 'plannedStart': return dir * a.planned_start.localeCompare(b.planned_start)
        default: return dir * a.planned_end.localeCompare(b.planned_end)
      }
    })
  }, [orders, search, statusFilter, sortField, sortDir])

  const bomMap = useMemo(() => {
    const map = new Map<string, BOM>()
    boms.forEach((b) => map.set(b.id, b))
    return map
  }, [boms])

  const orderNumbers = useMemo(() => orderNumberById(orders), [orders])

  const toggleBom = (id: string) => setExpandedBom(expandedBom === id ? null : id)

  const handleOpenQualityCheckFromDetail = (orderId: string) => {
    setQualityCheckOrderId(orderId)
    closeDetail()
    setShowQualityCheck(true)
  }

  const handleExportOrdersCsv = () => {
    const csv = buildOrdersCsv(filteredOrders, orderStatusLabels, priorityLabels, progressById)
    const blob = new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' })
    downloadBlob(blob, `produktionsauftraege-${csvDateStamp()}.csv`)
    toast.success(t('produktion.export.csvExported'))
  }

  // Resolve the current detail target against live query data so mutations
  // (status change, QC create) update the open modal.
  const detailOrder = currentDetail?.type === 'order' ? orders.find((o) => o.id === currentDetail.id) : undefined
  const detailBom = currentDetail?.type === 'bom' ? boms.find((b) => b.id === currentDetail.id) : undefined
  const detailQc = currentDetail?.type === 'qc' ? qualityChecks.find((c) => c.id === currentDetail.id) : undefined
  const detailMachine = currentDetail?.type === 'machine' ? machines.find((m) => m.id === currentDetail.id) : undefined

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

  // Custom role with module visibility but no read grant on any tab
  if (capReady && visibleTabs.length === 0) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <PageHeader
          title="Produktion"
          description={t('rbac.gate.moduleEmpty')}
          icon={Factory}
          moduleId="produktion"
          className="mb-6"
          actions={<RestrictedModeBadge module="produktion" />}
        />
        <EmptyState icon={Factory} title={t('rbac.gate.moduleEmpty')} description={t('rbac.gate.noPermission')} />
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
          <div className="flex items-center gap-2">
            <RestrictedModeBadge module="produktion" />
            {tab === 'stuecklisten' && canCreateBom && (
              <button onClick={() => setShowNewBom(true)} className="flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                <Plus className="h-4 w-4" />{t('produktion.actions.neueStückliste')}
              </button>
            )}
            {tab === 'qualitaet' && canCreateQuality && (
              <button onClick={() => { setQualityCheckOrderId(''); setShowQualityCheck(true) }} className="flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                <Plus className="h-4 w-4" />{t('produktion.actions.qualitaetspruefung')}
              </button>
            )}
            {tab === 'maschinen' && canManageMachine && (
              <button onClick={() => setShowNewMachine(true)} className="flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                <Plus className="h-4 w-4" />{t('produktion.actions.neueMaschine')}
              </button>
            )}
            {canCreateOrder && (
              <button onClick={() => setShowNewOrder(true)} className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">
                <Plus className="h-4 w-4" />{t('produktion.actions.neuerAuftrag')}
              </button>
            )}
          </div>
        }
      />

      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'auftraege' as const, label: t('produktion.tabs.auftraege', { count: activeOrders.length }) },
          { key: 'stuecklisten' as const, label: t('produktion.tabs.stuecklisten', { count: boms.length }) },
          { key: 'qualitaet' as const, label: t('produktion.tabs.qualitaet', { count: qualityChecks.length }) },
          { key: 'maschinen' as const, label: t('produktion.tabs.maschinen', { count: machines.length }) },
        ]).filter((tabItem) => visibleTabs.includes(tabItem.key)).map((tabItem) => (
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
            <div className="flex items-center gap-2 ml-auto">
              <SortMenu
                options={sortOptions}
                field={sortField}
                direction={sortDir}
                onChange={(f, d) => { setSortField(f); setSortDir(d) }}
              />
              {canExport && (
                <button
                  onClick={handleExportOrdersCsv}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
                >
                  <Download className="h-4 w-4" />
                  <span className="hidden md:inline">{t('produktion.export.csvButton')}</span>
                </button>
              )}
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
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.table.prioritaet')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.table.status')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground min-w-[160px]">{t('produktion.table.fortschritt')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.table.start')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('produktion.table.faellig')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredOrders.map((order) => {
                    const daysLeft = getDaysRemaining(order.planned_end)
                    const progress = progressById.get(order.id) ?? 0
                    return (
                      <tr
                        key={order.id}
                        role="button"
                        tabIndex={0}
                        onClick={() => pushDetail({ type: 'order', id: order.id })}
                        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); pushDetail({ type: 'order', id: order.id }) } }}
                        className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                      >
                        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{order.order_number}</td>
                        <td className="px-4 py-3 text-foreground font-medium">{order.product_name}</td>
                        <td className="px-4 py-3 text-xs text-muted-foreground text-right">{order.quantity.toLocaleString('de-DE')}</td>
                        <td className="px-4 py-3">
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${priorityColors[order.priority] ?? priorityColors[3]}`}>
                            {priorityLabels[order.priority] ?? `P${order.priority}`}
                          </span>
                        </td>
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
                  <div className="flex items-center">
                    <div
                      role="button"
                      tabIndex={0}
                      onClick={() => pushDetail({ type: 'bom', id: bom.id })}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); pushDetail({ type: 'bom', id: bom.id }) } }}
                      className="flex flex-1 items-center gap-3 p-4 text-left cursor-pointer hover:bg-secondary/30 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                    >
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light shrink-0">
                        <Package className="h-5 w-5 text-primary" />
                      </div>
                      <div className="min-w-0">
                        <h4 className="text-sm font-medium text-foreground">{bom.product_name}</h4>
                        <div className="flex items-center gap-2 mt-0.5 flex-wrap">
                          <span className="text-xs text-muted-foreground">{t('produktion.stueckliste.sku')} {bom.sku}</span>
                          <span className="text-xs text-muted-foreground">&middot; {t('produktion.stueckliste.version', { version: bom.version })}</span>
                          <span className="text-xs text-muted-foreground">&middot; {t('produktion.stueckliste.positionen', { count: bom.items.length })}</span>
                          {usedByOrders.length > 0 && <span className="text-xs text-muted-foreground">&middot; {t('produktion.stueckliste.auftraege', { count: usedByOrders.length })}</span>}
                        </div>
                      </div>
                      <div className="ml-auto flex items-center gap-2 shrink-0">
                        {bom.active ? (
                          <span className="rounded-full bg-success-light text-success px-2 py-0.5 text-[10px] font-medium">{t('produktion.stueckliste.aktiv')}</span>
                        ) : (
                          <span className="rounded-full bg-secondary text-muted-foreground px-2 py-0.5 text-[10px] font-medium">{t('produktion.stueckliste.inaktiv')}</span>
                        )}
                      </div>
                    </div>
                    <button
                      onClick={(e) => { e.stopPropagation(); toggleBom(bom.id) }}
                      aria-label={t('produktion.stueckliste.togglePositionen')}
                      className="p-4 text-muted-foreground hover:text-foreground transition-colors"
                    >
                      {expandedBom === bom.id ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                    </button>
                  </div>
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
                  <tr
                    key={check.id}
                    role="button"
                    tabIndex={0}
                    onClick={() => pushDetail({ type: 'qc', id: check.id })}
                    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); pushDetail({ type: 'qc', id: check.id }) } }}
                    className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                  >
                    <td className="px-4 py-3 text-xs text-muted-foreground">{formatDate(check.checked_at)}</td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{orderNumbers.get(check.order_id) ?? check.order_id.slice(0, 8)}</td>
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

      {tab === 'maschinen' && (
        <MaschinenbelegungChart
          machines={machines}
          bookings={machineBookings}
          orderNumbers={orderNumbers}
          onSelectOrder={(orderId) => pushDetail({ type: 'order', id: orderId })}
          onSelectMachine={(machineId) => pushDetail({ type: 'machine', id: machineId })}
        />
      )}

      {detailOrder && (
        <OrderDetailModal
          order={detailOrder}
          bom={bomMap.get(detailOrder.bom_id ?? '')}
          qualityChecks={qualityChecks.filter((qc: QualityCheck) => qc.order_id === detailOrder.id)}
          onClose={closeDetail}
          onBack={detailStack.length > 1 ? goBack : undefined}
          onOpenBom={(bomId) => pushDetail({ type: 'bom', id: bomId })}
          onOpenQualityCheck={(checkId) => pushDetail({ type: 'qc', id: checkId })}
          onAddQualityCheck={() => handleOpenQualityCheckFromDetail(detailOrder.id)}
        />
      )}
      {detailBom && (
        <BomDetailModal
          bom={detailBom}
          orders={orders}
          onClose={closeDetail}
          onBack={detailStack.length > 1 ? goBack : undefined}
          onOpenOrder={(orderId) => pushDetail({ type: 'order', id: orderId })}
        />
      )}
      {detailQc && (
        <QualityCheckDetailModal
          check={detailQc}
          order={orders.find((o) => o.id === detailQc.order_id)}
          onClose={closeDetail}
          onBack={detailStack.length > 1 ? goBack : undefined}
          onOpenOrder={(orderId) => pushDetail({ type: 'order', id: orderId })}
        />
      )}
      {detailMachine && (
        <MachineDetailModal
          machine={detailMachine}
          bookings={machineBookings}
          orderNumbers={orderNumbers}
          onClose={closeDetail}
          onBack={detailStack.length > 1 ? goBack : undefined}
          onOpenOrder={(orderId) => pushDetail({ type: 'order', id: orderId })}
        />
      )}

      <NewOrderDialog key={showNewOrder ? 'order-open' : 'order-closed'} open={showNewOrder} onOpenChange={setShowNewOrder} boms={boms} />
      <NewBomDialog key={showNewBom ? 'bom-open' : 'bom-closed'} open={showNewBom} onOpenChange={setShowNewBom} />
      <NewMachineDialog key={showNewMachine ? 'machine-open' : 'machine-closed'} open={showNewMachine} onOpenChange={setShowNewMachine} />
      <QualityCheckDialog
        key={showQualityCheck ? `qc-${qualityCheckOrderId || 'new'}` : 'qc-closed'}
        open={showQualityCheck}
        onOpenChange={setShowQualityCheck}
        orders={orders}
        preselectedOrderId={qualityCheckOrderId}
      />
    </div>
  )
}

// ============================================================
// Neuer Auftrag Dialog
// ============================================================

function NewOrderDialog({ open, onOpenChange, boms }: { open: boolean; onOpenChange: (open: boolean) => void; boms: BOM[] }) {
  const { t } = useTranslation()
  const createOrder = useCreateOrder()
  const orderNumberPrefix = useProduktionTenantStore((s) => s.orderNumberPrefix)
  const defaultPriority = useProduktionTenantStore((s) => s.defaultPriority)
  const defaultLeadDays = useProduktionTenantStore((s) => s.defaultLeadDays)
  const [bomId, setBomId] = useState('')
  const [quantity, setQuantity] = useState('')
  const [priority, setPriority] = useState(String(defaultPriority))
  const [startDate, setStartDate] = useState(new Date().toISOString().split('T')[0])
  const [dueDate, setDueDate] = useState(() => {
    const d = new Date()
    d.setDate(d.getDate() + defaultLeadDays)
    return d.toISOString().split('T')[0]
  })
  const [notes, setNotes] = useState('')
  const activeBoms = boms.filter((b) => b.active)

  const handleSave = async () => {
    if (!bomId) { toast.error(t('produktion.dialog.errorStueckliste')); return }
    if (!quantity || parseInt(quantity) <= 0) { toast.error(t('produktion.dialog.errorMenge')); return }
    if (!startDate || !dueDate) { toast.error(t('produktion.dialog.errorDaten')); return }
    const selectedBom = boms.find((b) => b.id === bomId)
    const input: CreateOrderInput = {
      order_number: generateOrderNumber(orderNumberPrefix),
      product_name: selectedBom?.product_name ?? '',
      quantity: parseInt(quantity),
      planned_start: new Date(startDate).toISOString(),
      planned_end: new Date(dueDate).toISOString(),
      priority: parseInt(priority),
      notes: notes || undefined,
      bom_id: bomId,
    }
    try {
      await createOrder.mutateAsync(input)
      toast.success(t('produktion.dialog.auftragErstelltToast', { name: selectedBom?.product_name, qty: quantity }))
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
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('produktion.dialog.menge')}</Label>
              <Input type="number" min="1" placeholder={t('produktion.dialog.mengePlaceholder')} value={quantity} onChange={(e) => setQuantity(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">{t('produktion.dialog.prioritaet')}</Label>
              <Select value={priority} onValueChange={setPriority}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {PRIORITY_VALUES.map((p) => (
                    <SelectItem key={p} value={String(p)}>{t(priorityLabelKeys[p])}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
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
// Neue Maschine Dialog
// ============================================================

function NewMachineDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const { t } = useTranslation()
  const createMachine = useCreateMachine()
  const [name, setName] = useState('')
  const [type, setType] = useState('')
  const [notes, setNotes] = useState('')

  const handleSave = async () => {
    if (!name.trim()) { toast.error(t('produktion.machineDialog.errorName')); return }
    try {
      await createMachine.mutateAsync({ name: name.trim(), type: type.trim() || undefined, notes: notes.trim() || undefined })
      toast.success(t('produktion.machineDialog.createdToast', { name: name.trim() }))
      onOpenChange(false)
    } catch {
      toast.error(t('common.error'))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[420px]">
        <DialogHeader><DialogTitle>{t('produktion.machineDialog.title')}</DialogTitle></DialogHeader>
        <div className="space-y-4 mt-2">
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('produktion.machineDialog.name')}</Label>
            <Input placeholder={t('produktion.machineDialog.namePlaceholder')} value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('produktion.machineDialog.type')}</Label>
            <Input placeholder={t('produktion.machineDialog.typePlaceholder')} value={type} onChange={(e) => setType(e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">{t('produktion.dialog.notizen')}</Label>
            <Textarea value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} />
          </div>
          <div className="flex gap-2 pt-2">
            <button onClick={() => onOpenChange(false)} className="flex-1 rounded-lg border border-border py-2 text-sm text-foreground hover:bg-secondary transition-colors">{t('common.cancel')}</button>
            <button onClick={handleSave} disabled={createMachine.isPending} className="flex-1 rounded-lg bg-primary py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-60">{t('produktion.machineDialog.create')}</button>
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
    if (!inspector.trim()) { toast.error(t('produktion.dialog.errorPruefer')); return }
    if (!date) { toast.error(t('produktion.dialog.errorDatum')); return }
    const selectedOrder = orders.find((o) => o.id === selectedOrderId)
    const input: CreateQualityCheckInput = {
      order_id: selectedOrderId,
      inspector: inspector.trim(),
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
              <Input placeholder={t('produktion.dialog.prueferPlaceholder')} value={inspector} onChange={(e) => setInspector(e.target.value)} />
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
