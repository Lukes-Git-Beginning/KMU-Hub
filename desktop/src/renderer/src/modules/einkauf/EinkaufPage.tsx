import { useState, useMemo, useCallback } from 'react'
import {
  Search,
  Plus,
  ShoppingCart,
  Truck,
  BookOpen,
  Mail,
  Phone,
  Eye,
  Edit,
  Trash2,
  X,
  PackageCheck,
  CalendarDays,
  Clock,
  ChevronRight,
  Check,
  Circle,
  UserCircle,
  CreditCard,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useEinkaufStore,
  type PurchaseOrder,
  type Supplier,
} from '@/stores/einkauf'
import { ItemActions, ConfirmDialog, EmptyState, DetailPanel } from '@/components/shared'
import { formatAmount } from '@/lib/format'

type TabKey = 'bestellungen' | 'lieferanten' | 'katalog'
type StatusFilter = PurchaseOrder['status'] | 'all'

const orderStatusLabels: Record<string, string> = {
  draft: 'Entwurf',
  sent: 'Bestellt',
  confirmed: 'Bestaetigt',
  partial: 'Teillieferung',
  received: 'Geliefert',
  cancelled: 'Storniert',
}

const orderStatusColors: Record<string, string> = {
  draft: 'bg-secondary text-muted-foreground',
  sent: 'bg-info-light text-info',
  confirmed: 'bg-primary-light text-primary',
  partial: 'bg-warning-light text-warning',
  received: 'bg-success-light text-success',
  cancelled: 'bg-error-light text-error',
}

const STATUS_TIMELINE: PurchaseOrder['status'][] = [
  'draft',
  'sent',
  'confirmed',
  'partial',
  'received',
]

const PAYMENT_TERMS_OPTIONS = [
  '30 Tage netto',
  '60 Tage netto',
  '14 Tage 2% Skonto',
  '45 Tage netto',
  'Vorkasse',
  'Rechnung',
]

const STATUS_FILTER_OPTIONS: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'Alle' },
  { value: 'draft', label: 'Entwurf' },
  { value: 'sent', label: 'Bestellt' },
  { value: 'confirmed', label: 'Bestaetigt' },
  { value: 'partial', label: 'Teillieferung' },
  { value: 'received', label: 'Geliefert' },
  { value: 'cancelled', label: 'Storniert' },
]

interface NewOrderItem {
  name: string
  quantity: number
  unitPrice: number
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function EinkaufPage() {
  const { purchaseOrders, suppliers, purchaseOrderItems } = useEinkaufStore()

  // Tab & search
  const [tab, setTab] = useState<TabKey>('bestellungen')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')

  // Detail panels
  const [selectedOrder, setSelectedOrder] = useState<PurchaseOrder | null>(null)
  const [selectedSupplier, setSelectedSupplier] = useState<Supplier | null>(null)

  // Dialogs
  const [confirmDelete, setConfirmDelete] = useState<PurchaseOrder | null>(null)
  const [showNewOrderDialog, setShowNewOrderDialog] = useState(false)
  const [showNewSupplierDialog, setShowNewSupplierDialog] = useState(false)
  const [showWareneingangDialog, setShowWareneingangDialog] = useState<PurchaseOrder | null>(null)

  // New order form state
  const [newOrderSupplierId, setNewOrderSupplierId] = useState('')
  const [newOrderItems, setNewOrderItems] = useState<NewOrderItem[]>([
    { name: '', quantity: 1, unitPrice: 0 },
  ])
  const [newOrderDate, setNewOrderDate] = useState('')
  const [newOrderNotes, setNewOrderNotes] = useState('')

  // New supplier form state
  const [newSupName, setNewSupName] = useState('')
  const [newSupContact, setNewSupContact] = useState('')
  const [newSupEmail, setNewSupEmail] = useState('')
  const [newSupPhone, setNewSupPhone] = useState('')
  const [newSupPayment, setNewSupPayment] = useState(PAYMENT_TERMS_OPTIONS[0])

  // Wareneingang state
  const [receivedQtys, setReceivedQtys] = useState<Record<string, number>>({})
  const [partialDelivery, setPartialDelivery] = useState(false)

  // ---------------------------------------------------------------------------
  // Derived data
  // ---------------------------------------------------------------------------

  const filteredOrders = useMemo(() => {
    let list = purchaseOrders
    if (statusFilter !== 'all') {
      list = list.filter((o) => o.status === statusFilter)
    }
    if (search) {
      const q = search.toLowerCase()
      list = list.filter(
        (o) =>
          o.orderNumber.toLowerCase().includes(q) ||
          o.supplierName.toLowerCase().includes(q),
      )
    }
    return list
  }, [purchaseOrders, search, statusFilter])

  const filteredSuppliers = useMemo(() => {
    if (!search) return suppliers.filter((s) => s.isActive)
    const q = search.toLowerCase()
    return suppliers.filter(
      (s) =>
        s.isActive &&
        (s.name.toLowerCase().includes(q) ||
          s.contactName.toLowerCase().includes(q) ||
          s.email.toLowerCase().includes(q)),
    )
  }, [suppliers, search])

  const activeOrderCount = purchaseOrders.filter(
    (o) => o.status !== 'received' && o.status !== 'cancelled',
  ).length

  const totalOpen = purchaseOrders
    .filter((o) => o.status !== 'received' && o.status !== 'cancelled')
    .reduce((sum, o) => sum + o.total, 0)

  const activeSupplierCount = suppliers.filter((s) => s.isActive).length

  const newOrderTotal = useMemo(
    () => newOrderItems.reduce((s, i) => s + i.quantity * i.unitPrice, 0),
    [newOrderItems],
  )

  // Items for a given order
  const getOrderItems = useCallback(
    (orderId: string) => purchaseOrderItems.filter((i) => i.orderId === orderId),
    [purchaseOrderItems],
  )

  // Orders for a given supplier
  const getSupplierOrders = useCallback(
    (supplierId: string) => purchaseOrders.filter((o) => o.supplierId === supplierId),
    [purchaseOrders],
  )

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------

  const getOrderActions = (order: PurchaseOrder) => [
    {
      label: 'Details anzeigen',
      icon: Eye,
      onClick: () => {
        setSelectedSupplier(null)
        setSelectedOrder(order)
      },
    },
    {
      label: 'Wareneingang buchen',
      icon: PackageCheck,
      onClick: () => openWareneingang(order),
    },
    { label: 'Bearbeiten', icon: Edit, onClick: () => toast.info(`Bestellung ${order.orderNumber} bearbeiten`) },
    { separator: true as const, label: '', onClick: () => {} },
    { label: 'Stornieren', icon: Trash2, variant: 'destructive' as const, onClick: () => setConfirmDelete(order) },
  ]

  const handleCancelOrder = (order: PurchaseOrder) => {
    setConfirmDelete(null)
    toast.success(`Bestellung ${order.orderNumber} wurde storniert`)
  }

  // -- New order dialog helpers --
  const resetNewOrderForm = () => {
    setNewOrderSupplierId('')
    setNewOrderItems([{ name: '', quantity: 1, unitPrice: 0 }])
    setNewOrderDate('')
    setNewOrderNotes('')
  }

  const addOrderItemRow = () => {
    setNewOrderItems((prev) => [...prev, { name: '', quantity: 1, unitPrice: 0 }])
  }

  const removeOrderItemRow = (idx: number) => {
    setNewOrderItems((prev) => prev.filter((_, i) => i !== idx))
  }

  const updateOrderItem = (idx: number, field: keyof NewOrderItem, value: string | number) => {
    setNewOrderItems((prev) =>
      prev.map((item, i) => (i === idx ? { ...item, [field]: value } : item)),
    )
  }

  const handleSaveOrder = () => {
    if (!newOrderSupplierId) {
      toast.error('Bitte Lieferant auswaehlen')
      return
    }
    if (newOrderItems.some((i) => !i.name.trim())) {
      toast.error('Alle Positionen muessen einen Namen haben')
      return
    }
    const sup = suppliers.find((s) => s.id === newOrderSupplierId)
    const nr = `PO-2026-${String(purchaseOrders.length + 1).padStart(3, '0')}`
    toast.success(`Bestellung ${nr} bei ${sup?.name ?? 'Lieferant'} erstellt (CHF ${newOrderTotal.toLocaleString('de-CH', { minimumFractionDigits: 2 })})`)
    resetNewOrderForm()
    setShowNewOrderDialog(false)
  }

  // -- New supplier dialog helpers --
  const resetNewSupplierForm = () => {
    setNewSupName('')
    setNewSupContact('')
    setNewSupEmail('')
    setNewSupPhone('')
    setNewSupPayment(PAYMENT_TERMS_OPTIONS[0])
  }

  const handleSaveSupplier = () => {
    if (!newSupName.trim()) {
      toast.error('Bitte Name eingeben')
      return
    }
    toast.success(`Lieferant "${newSupName}" wurde angelegt`)
    resetNewSupplierForm()
    setShowNewSupplierDialog(false)
  }

  // -- Wareneingang --
  const openWareneingang = (order: PurchaseOrder) => {
    const items = getOrderItems(order.id)
    const initial: Record<string, number> = {}
    items.forEach((i) => {
      initial[i.id] = 0
    })
    setReceivedQtys(initial)
    setPartialDelivery(false)
    setShowWareneingangDialog(order)
  }

  const handleSaveWareneingang = () => {
    if (!showWareneingangDialog) return
    toast.success(`Wareneingang fuer Bestellung ${showWareneingangDialog.orderNumber} gebucht`)
    setShowWareneingangDialog(null)
  }

  // ---------------------------------------------------------------------------
  // Render helpers
  // ---------------------------------------------------------------------------


  const formatDate = (d: string) =>
    new Date(d.includes('T') ? d : d + 'T00:00:00').toLocaleDateString('de-CH')

  /** Status timeline for detail panel */
  const renderTimeline = (currentStatus: PurchaseOrder['status']) => {
    if (currentStatus === 'cancelled') {
      return (
        <div className="flex items-center gap-2 text-xs text-error">
          <X className="h-3.5 w-3.5" />
          <span>Bestellung storniert</span>
        </div>
      )
    }
    const currentIdx = STATUS_TIMELINE.indexOf(currentStatus)
    return (
      <div className="flex items-center gap-1">
        {STATUS_TIMELINE.map((s, idx) => {
          const reached = idx <= currentIdx
          return (
            <div key={s} className="flex items-center gap-1">
              <div className="flex flex-col items-center">
                <div
                  className={`flex h-5 w-5 items-center justify-center rounded-full ${
                    reached ? 'bg-primary text-primary-foreground' : 'bg-secondary text-muted-foreground'
                  }`}
                >
                  {reached ? <Check className="h-3 w-3" /> : <Circle className="h-3 w-3" />}
                </div>
                <span
                  className={`mt-1 text-[9px] leading-tight ${
                    reached ? 'font-medium text-foreground' : 'text-muted-foreground'
                  }`}
                >
                  {orderStatusLabels[s]}
                </span>
              </div>
              {idx < STATUS_TIMELINE.length - 1 && (
                <div
                  className={`mx-0.5 h-px w-4 ${
                    idx < currentIdx ? 'bg-primary' : 'bg-border'
                  }`}
                />
              )}
            </div>
          )
        })}
      </div>
    )
  }

  // ---------------------------------------------------------------------------
  // JSX
  // ---------------------------------------------------------------------------

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div>
          <h1 className="text-foreground">Einkauf</h1>
          <p className="text-sm text-muted-foreground">
            {activeOrderCount} offene Bestellungen · CHF {formatAmount(totalOpen)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {tab === 'lieferanten' && (
            <button
              onClick={() => {
                resetNewSupplierForm()
                setShowNewSupplierDialog(true)
              }}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              <Plus className="h-4 w-4" />
              Lieferant anlegen
            </button>
          )}
          <button
            onClick={() => {
              resetNewOrderForm()
              setShowNewOrderDialog(true)
            }}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            Neue Bestellung
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'bestellungen' as const, label: `Bestellungen (${purchaseOrders.length})` },
          { key: 'lieferanten' as const, label: `Lieferanten (${activeSupplierCount})` },
          { key: 'katalog' as const, label: 'Katalog' },
        ]).map((t) => (
          <button
            key={t.key}
            onClick={() => {
              setTab(t.key)
              setSearch('')
              setStatusFilter('all')
            }}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key
                ? 'border-primary text-primary font-medium'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Search + status filter */}
      {tab !== 'katalog' && (
        <div className="flex flex-wrap items-center gap-3 mb-4">
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder={
                tab === 'bestellungen' ? 'Bestellung suchen...' : 'Lieferant suchen...'
              }
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
          {tab === 'bestellungen' && (
            <div className="flex items-center gap-1.5 flex-wrap">
              {STATUS_FILTER_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => setStatusFilter(opt.value)}
                  className={`rounded-full px-2.5 py-1 text-xs font-medium transition-colors ${
                    statusFilter === opt.value
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-secondary text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ====================== BESTELLUNGEN TAB ====================== */}
      {tab === 'bestellungen' && (
        <>
          {filteredOrders.length === 0 ? (
            <EmptyState
              icon={ShoppingCart}
              title="Keine Bestellungen gefunden"
              description={
                search || statusFilter !== 'all'
                  ? 'Passe deine Suche oder den Filter an'
                  : 'Erstelle deine erste Bestellung'
              }
            />
          ) : (
            <div className="overflow-x-auto rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-card">
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Bestellnr.</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Lieferant</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground">Betrag (CHF)</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Lieferdatum</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Erstellt am</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground"></th>
                  </tr>
                </thead>
                <tbody>
                  {filteredOrders.map((order) => (
                    <tr
                      key={order.id}
                      onClick={() => {
                        setSelectedSupplier(null)
                        setSelectedOrder(order)
                      }}
                      className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer"
                    >
                      <td className="px-4 py-3 font-medium text-foreground font-mono text-xs">
                        {order.orderNumber}
                      </td>
                      <td className="px-4 py-3 text-foreground">{order.supplierName}</td>
                      <td className="px-4 py-3">
                        <span
                          className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                            orderStatusColors[order.status] ?? 'bg-secondary text-muted-foreground'
                          }`}
                        >
                          {orderStatusLabels[order.status] ?? order.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right text-foreground tabular-nums">
                        {formatAmount(order.total)}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {order.expectedDelivery ? formatDate(order.expectedDelivery) : '--'}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {formatDate(order.createdAt)}
                      </td>
                      <td className="px-4 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                        <ItemActions items={getOrderActions(order)} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* ====================== LIEFERANTEN TAB ====================== */}
      {tab === 'lieferanten' && (
        <>
          {filteredSuppliers.length === 0 ? (
            <EmptyState
              icon={Truck}
              title="Keine Lieferanten gefunden"
              description={search ? 'Passe deine Suche an' : 'Fuege deinen ersten Lieferanten hinzu'}
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {filteredSuppliers.map((supplier) => {
                const supOrders = getSupplierOrders(supplier.id)
                return (
                  <div
                    key={supplier.id}
                    onClick={() => {
                      setSelectedOrder(null)
                      setSelectedSupplier(supplier)
                    }}
                    className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer"
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                          <Truck className="h-5 w-5 text-primary" />
                        </div>
                        <div>
                          <h4 className="text-sm font-medium text-foreground">{supplier.name}</h4>
                          <p className="text-xs text-muted-foreground">{supplier.contactName}</p>
                        </div>
                      </div>
                      <div onClick={(e) => e.stopPropagation()}>
                        <ItemActions
                          items={[
                            {
                              label: 'Details anzeigen',
                              icon: Eye,
                              onClick: () => {
                                setSelectedOrder(null)
                                setSelectedSupplier(supplier)
                              },
                            },
                            { label: 'Bearbeiten', icon: Edit, onClick: () => toast.info(`${supplier.name} bearbeiten`) },
                            { separator: true as const, label: '', onClick: () => {} },
                            {
                              label: 'Deaktivieren',
                              variant: 'destructive' as const,
                              onClick: () => toast.info(`${supplier.name} deaktiviert`),
                            },
                          ]}
                        />
                      </div>
                    </div>
                    <div className="space-y-1.5 text-xs text-muted-foreground mb-3">
                      <div className="flex items-center gap-2">
                        <Mail className="h-3 w-3" />
                        <span>{supplier.email}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <Phone className="h-3 w-3" />
                        <span>{supplier.phone}</span>
                      </div>
                    </div>
                    <div className="flex items-center justify-between border-t border-border-muted pt-3">
                      <span className="rounded-full bg-secondary px-2 py-0.5 text-xs text-muted-foreground">
                        {supplier.paymentTerms}
                      </span>
                      <span className="text-xs text-muted-foreground">
                        {supOrders.length} Bestellung{supOrders.length !== 1 ? 'en' : ''}
                      </span>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </>
      )}

      {/* ====================== KATALOG TAB ====================== */}
      {tab === 'katalog' && (
        <EmptyState
          icon={BookOpen}
          title="Lieferantenkataloge"
          description="Lieferantenkataloge werden hier angezeigt. Diese Funktion wird in einer zukuenftigen Version verfuegbar sein."
        />
      )}

      {/* ====================== ORDER DETAIL PANEL ====================== */}
      <DetailPanel
        open={!!selectedOrder}
        onClose={() => setSelectedOrder(null)}
        title={selectedOrder?.orderNumber ?? ''}
        subtitle={selectedOrder?.supplierName}
        badge={
          selectedOrder ? (
            <span
              className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                orderStatusColors[selectedOrder.status] ?? 'bg-secondary text-muted-foreground'
              }`}
            >
              {orderStatusLabels[selectedOrder.status]}
            </span>
          ) : undefined
        }
        width="w-[440px]"
        footer={
          selectedOrder && selectedOrder.status !== 'received' && selectedOrder.status !== 'cancelled' ? (
            <div className="flex items-center gap-2">
              <button
                onClick={() => {
                  if (selectedOrder) openWareneingang(selectedOrder)
                }}
                className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <PackageCheck className="h-4 w-4" />
                Wareneingang buchen
              </button>
              <button
                onClick={() => toast.info(`Bestellung ${selectedOrder.orderNumber} bearbeiten`)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                Bearbeiten
              </button>
            </div>
          ) : undefined
        }
      >
        {selectedOrder && (
          <div className="space-y-5">
            {/* Summary */}
            <div className="grid grid-cols-2 gap-3">
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">Betrag</p>
                <p className="text-lg font-semibold text-foreground tabular-nums">
                  CHF {formatAmount(selectedOrder.total)}
                </p>
              </div>
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">Positionen</p>
                <p className="text-lg font-semibold text-foreground">{selectedOrder.itemCount}</p>
              </div>
            </div>

            {/* Dates */}
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm">
                <CalendarDays className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">Lieferdatum:</span>
                <span className="text-foreground">
                  {selectedOrder.expectedDelivery ? formatDate(selectedOrder.expectedDelivery) : '--'}
                </span>
              </div>
              <div className="flex items-center gap-2 text-sm">
                <Clock className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">Erstellt:</span>
                <span className="text-foreground">{formatDate(selectedOrder.createdAt)}</span>
              </div>
            </div>

            {/* Status timeline */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">
                Status-Verlauf
              </h4>
              {renderTimeline(selectedOrder.status)}
            </div>

            {/* Order items */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                Bestellte Positionen
              </h4>
              <div className="rounded-lg border border-border divide-y divide-border-muted">
                {getOrderItems(selectedOrder.id).length === 0 ? (
                  <p className="px-3 py-4 text-xs text-muted-foreground text-center">
                    Keine Positionen vorhanden
                  </p>
                ) : (
                  getOrderItems(selectedOrder.id).map((item) => (
                    <div key={item.id} className="flex items-center justify-between px-3 py-2.5">
                      <div>
                        <p className="text-sm text-foreground">{item.itemName}</p>
                        <p className="text-xs text-muted-foreground">
                          {item.quantity} Stk. x CHF {item.unitPrice.toFixed(2)}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="text-sm font-medium text-foreground tabular-nums">
                          CHF {formatAmount(item.quantity * item.unitPrice)}
                        </p>
                        {item.receivedQuantity > 0 && (
                          <p className="text-[10px] text-success">
                            {item.receivedQuantity}/{item.quantity} erhalten
                          </p>
                        )}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Supplier link */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                Lieferant
              </h4>
              <button
                onClick={() => {
                  const sup = suppliers.find((s) => s.id === selectedOrder.supplierId)
                  if (sup) {
                    setSelectedOrder(null)
                    setTimeout(() => setSelectedSupplier(sup), 200)
                  }
                }}
                className="flex w-full items-center justify-between rounded-lg border border-border p-3 hover:bg-secondary/50 transition-colors"
              >
                <div className="flex items-center gap-2">
                  <Truck className="h-4 w-4 text-primary" />
                  <span className="text-sm text-foreground">{selectedOrder.supplierName}</span>
                </div>
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              </button>
            </div>
          </div>
        )}
      </DetailPanel>

      {/* ====================== SUPPLIER DETAIL PANEL ====================== */}
      <DetailPanel
        open={!!selectedSupplier}
        onClose={() => setSelectedSupplier(null)}
        title={selectedSupplier?.name ?? ''}
        subtitle={selectedSupplier?.contactName}
        width="w-[420px]"
        footer={
          selectedSupplier ? (
            <div className="flex items-center gap-2">
              <button
                onClick={() => toast.info(`${selectedSupplier.name} bearbeiten`)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                Bearbeiten
              </button>
            </div>
          ) : undefined
        }
      >
        {selectedSupplier && (
          <div className="space-y-5">
            {/* Contact info */}
            <div className="space-y-3">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                  <UserCircle className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">{selectedSupplier.contactName}</p>
                  <p className="text-xs text-muted-foreground">Kontaktperson</p>
                </div>
              </div>

              <div className="rounded-lg border border-border divide-y divide-border-muted">
                <div className="flex items-center gap-3 px-3 py-2.5">
                  <Mail className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm text-foreground">{selectedSupplier.email}</span>
                </div>
                <div className="flex items-center gap-3 px-3 py-2.5">
                  <Phone className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm text-foreground">{selectedSupplier.phone}</span>
                </div>
                <div className="flex items-center gap-3 px-3 py-2.5">
                  <CreditCard className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm text-foreground">{selectedSupplier.paymentTerms}</span>
                </div>
              </div>
            </div>

            {/* Status */}
            <div className="flex items-center gap-2">
              <span
                className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                  selectedSupplier.isActive
                    ? 'bg-success-light text-success'
                    : 'bg-error-light text-error'
                }`}
              >
                {selectedSupplier.isActive ? 'Aktiv' : 'Inaktiv'}
              </span>
            </div>

            {/* Recent orders */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                Bestellungen ({getSupplierOrders(selectedSupplier.id).length})
              </h4>
              <div className="rounded-lg border border-border divide-y divide-border-muted">
                {getSupplierOrders(selectedSupplier.id).length === 0 ? (
                  <p className="px-3 py-4 text-xs text-muted-foreground text-center">
                    Keine Bestellungen vorhanden
                  </p>
                ) : (
                  getSupplierOrders(selectedSupplier.id).map((order) => (
                    <button
                      key={order.id}
                      onClick={() => {
                        setSelectedSupplier(null)
                        setTimeout(() => setSelectedOrder(order), 200)
                      }}
                      className="flex w-full items-center justify-between px-3 py-2.5 hover:bg-secondary/50 transition-colors"
                    >
                      <div className="text-left">
                        <p className="text-sm font-mono text-foreground">{order.orderNumber}</p>
                        <p className="text-xs text-muted-foreground">
                          {formatDate(order.createdAt)} · CHF {formatAmount(order.total)}
                        </p>
                      </div>
                      <span
                        className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                          orderStatusColors[order.status] ?? 'bg-secondary text-muted-foreground'
                        }`}
                      >
                        {orderStatusLabels[order.status]}
                      </span>
                    </button>
                  ))
                )}
              </div>
            </div>
          </div>
        )}
      </DetailPanel>

      {/* ====================== NEUE BESTELLUNG DIALOG ====================== */}
      {showNewOrderDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-lg rounded-xl bg-card border border-border shadow-xl mx-4 max-h-[85vh] flex flex-col">
            {/* Header */}
            <div className="flex items-center justify-between border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <ShoppingCart className="h-5 w-5 text-primary" />
                <h2 className="text-base font-semibold text-foreground">Neue Bestellung</h2>
              </div>
              <button
                onClick={() => setShowNewOrderDialog(false)}
                className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Body */}
            <div className="flex-1 overflow-y-auto p-5 space-y-4">
              {/* Supplier */}
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  Lieferant <span className="text-destructive">*</span>
                </label>
                <select
                  value={newOrderSupplierId}
                  onChange={(e) => setNewOrderSupplierId(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  <option value="">Lieferant auswaehlen...</option>
                  {suppliers
                    .filter((s) => s.isActive)
                    .map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.name}
                      </option>
                    ))}
                </select>
              </div>

              {/* Positionen */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium text-foreground">Positionen</label>
                  <button
                    onClick={addOrderItemRow}
                    className="flex items-center gap-1 rounded-lg border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-secondary transition-colors"
                  >
                    <Plus className="h-3 w-3" />
                    Position
                  </button>
                </div>
                <div className="space-y-2">
                  {newOrderItems.map((item, idx) => (
                    <div
                      key={idx}
                      className="flex items-start gap-2 rounded-lg border border-border-muted p-2.5"
                    >
                      <div className="flex-1 space-y-2">
                        <input
                          type="text"
                          placeholder="Artikelname"
                          value={item.name}
                          onChange={(e) => updateOrderItem(idx, 'name', e.target.value)}
                          className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                        />
                        <div className="flex gap-2">
                          <div className="flex-1">
                            <label className="text-[10px] text-muted-foreground">Menge</label>
                            <input
                              type="number"
                              min={1}
                              value={item.quantity}
                              onChange={(e) =>
                                updateOrderItem(idx, 'quantity', Math.max(1, parseInt(e.target.value) || 1))
                              }
                              className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                            />
                          </div>
                          <div className="flex-1">
                            <label className="text-[10px] text-muted-foreground">Einzelpreis (CHF)</label>
                            <input
                              type="number"
                              min={0}
                              step={0.01}
                              value={item.unitPrice}
                              onChange={(e) =>
                                updateOrderItem(idx, 'unitPrice', Math.max(0, parseFloat(e.target.value) || 0))
                              }
                              className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                            />
                          </div>
                        </div>
                      </div>
                      {newOrderItems.length > 1 && (
                        <button
                          onClick={() => removeOrderItemRow(idx)}
                          className="mt-1 rounded p-1 text-muted-foreground hover:text-error hover:bg-error-light transition-colors"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {/* Total */}
              <div className="flex items-center justify-between rounded-lg bg-secondary/50 px-3 py-2">
                <span className="text-sm font-medium text-muted-foreground">Total</span>
                <span className="text-base font-semibold text-foreground tabular-nums">
                  CHF {formatAmount(newOrderTotal)}
                </span>
              </div>

              {/* Expected delivery */}
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">Erwartetes Lieferdatum</label>
                <input
                  type="date"
                  value={newOrderDate}
                  onChange={(e) => setNewOrderDate(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>

              {/* Notes */}
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">Notizen</label>
                <textarea
                  value={newOrderNotes}
                  onChange={(e) => setNewOrderNotes(e.target.value)}
                  rows={3}
                  placeholder="Optionale Anmerkungen..."
                  className="w-full resize-none rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
            </div>

            {/* Footer */}
            <div className="flex items-center justify-end gap-2 border-t border-border px-5 py-4">
              <button
                onClick={() => setShowNewOrderDialog(false)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleSaveOrder}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                Bestellung erstellen
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ====================== LIEFERANT ANLEGEN DIALOG ====================== */}
      {showNewSupplierDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-xl bg-card border border-border shadow-xl mx-4">
            {/* Header */}
            <div className="flex items-center justify-between border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <Truck className="h-5 w-5 text-primary" />
                <h2 className="text-base font-semibold text-foreground">Lieferant anlegen</h2>
              </div>
              <button
                onClick={() => setShowNewSupplierDialog(false)}
                className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Body */}
            <div className="p-5 space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  Name <span className="text-destructive">*</span>
                </label>
                <input
                  type="text"
                  value={newSupName}
                  onChange={(e) => setNewSupName(e.target.value)}
                  placeholder="Firmenname"
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">Kontaktperson</label>
                <input
                  type="text"
                  value={newSupContact}
                  onChange={(e) => setNewSupContact(e.target.value)}
                  placeholder="Vor- und Nachname"
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">Email</label>
                  <input
                    type="email"
                    value={newSupEmail}
                    onChange={(e) => setNewSupEmail(e.target.value)}
                    placeholder="email@firma.ch"
                    className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">Telefon</label>
                  <input
                    type="tel"
                    value={newSupPhone}
                    onChange={(e) => setNewSupPhone(e.target.value)}
                    placeholder="+41 ..."
                    className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">Zahlungsbedingungen</label>
                <select
                  value={newSupPayment}
                  onChange={(e) => setNewSupPayment(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  {PAYMENT_TERMS_OPTIONS.map((pt) => (
                    <option key={pt} value={pt}>
                      {pt}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            {/* Footer */}
            <div className="flex items-center justify-end gap-2 border-t border-border px-5 py-4">
              <button
                onClick={() => setShowNewSupplierDialog(false)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleSaveSupplier}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                Lieferant speichern
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ====================== WARENEINGANG DIALOG ====================== */}
      {showWareneingangDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-lg rounded-xl bg-card border border-border shadow-xl mx-4 max-h-[80vh] flex flex-col">
            {/* Header */}
            <div className="flex items-center justify-between border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <PackageCheck className="h-5 w-5 text-primary" />
                <div>
                  <h2 className="text-base font-semibold text-foreground">Wareneingang buchen</h2>
                  <p className="text-xs text-muted-foreground">
                    {showWareneingangDialog.orderNumber} — {showWareneingangDialog.supplierName}
                  </p>
                </div>
              </div>
              <button
                onClick={() => setShowWareneingangDialog(null)}
                className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Body */}
            <div className="flex-1 overflow-y-auto p-5 space-y-4">
              <div className="rounded-lg border border-border divide-y divide-border-muted">
                <div className="grid grid-cols-[1fr_80px_80px] gap-2 px-3 py-2 bg-secondary/30">
                  <span className="text-[10px] font-medium text-muted-foreground uppercase">Artikel</span>
                  <span className="text-[10px] font-medium text-muted-foreground uppercase text-right">Bestellt</span>
                  <span className="text-[10px] font-medium text-muted-foreground uppercase text-right">Erhalten</span>
                </div>
                {getOrderItems(showWareneingangDialog.id).map((item) => (
                  <div key={item.id} className="grid grid-cols-[1fr_80px_80px] gap-2 items-center px-3 py-2.5">
                    <div>
                      <p className="text-sm text-foreground">{item.itemName}</p>
                      <p className="text-[10px] text-muted-foreground">
                        Bereits erhalten: {item.receivedQuantity}
                      </p>
                    </div>
                    <p className="text-sm text-muted-foreground text-right tabular-nums">{item.quantity}</p>
                    <input
                      type="number"
                      min={0}
                      max={item.quantity - item.receivedQuantity}
                      value={receivedQtys[item.id] ?? 0}
                      onChange={(e) =>
                        setReceivedQtys((prev) => ({
                          ...prev,
                          [item.id]: Math.max(0, parseInt(e.target.value) || 0),
                        }))
                      }
                      className="w-full rounded-lg border border-border bg-card px-2 py-1 text-sm text-foreground text-right tabular-nums focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    />
                  </div>
                ))}
              </div>

              {/* Partial delivery checkbox */}
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={partialDelivery}
                  onChange={(e) => setPartialDelivery(e.target.checked)}
                  className="h-4 w-4 rounded border-border text-primary focus:ring-focus-ring"
                />
                <span className="text-sm text-foreground">Teillieferung (Bestellung bleibt offen)</span>
              </label>
            </div>

            {/* Footer */}
            <div className="flex items-center justify-end gap-2 border-t border-border px-5 py-4">
              <button
                onClick={() => setShowWareneingangDialog(null)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleSaveWareneingang}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                Wareneingang buchen
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ====================== CONFIRM CANCEL ORDER ====================== */}
      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title="Bestellung stornieren?"
        description={`Bestellung ${confirmDelete?.orderNumber} bei ${confirmDelete?.supplierName} wird storniert. Diese Aktion kann nicht rueckgaengig gemacht werden.`}
        confirmLabel="Stornieren"
        variant="destructive"
        onConfirm={() => confirmDelete && handleCancelOrder(confirmDelete)}
      />
    </div>
  )
}
