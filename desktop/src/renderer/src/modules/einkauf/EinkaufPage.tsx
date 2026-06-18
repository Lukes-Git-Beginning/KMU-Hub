import { useState, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
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
  Star,
  FileText,
  Link2,
  ShieldCheck,
  Coins,
  ScrollText,
  BarChart3,
  ScanBarcode,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { toast } from 'sonner'
import { useEinkaufStore } from '@/stores/einkauf'
import type {
  Supplier,
  PurchaseOrder,
} from '@/api/einkauf-types'
import {
  useSuppliers,
  usePOs,
  useCatalogItems,
  useFrameworkContracts,
  useSupplierRatings,
} from '@/api/hooks/useEinkauf'
import { ItemActions, ConfirmDialog, EmptyState, DetailPanel, PageHeader } from '@/components/shared'
import { formatAmount, formatCurrency, formatDate } from '@/lib/format'

type TabKey = 'bestellungen' | 'lieferanten' | 'katalog' | 'rahmenverträge'
type StatusFilter = PurchaseOrder['status'] | 'all' | 'confirmed' | 'partial'

const orderStatusLabels: Record<string, string> = {
  draft: 'einkauf.orderStatus.draft',
  submitted: 'einkauf.orderStatus.submitted',
  sent: 'einkauf.orderStatus.sent',
  confirmed: 'einkauf.orderStatus.confirmed',
  partial: 'einkauf.orderStatus.partial',
  partially_received: 'einkauf.orderStatus.partial',
  received: 'einkauf.orderStatus.received',
  closed: 'einkauf.orderStatus.received',
  cancelled: 'einkauf.orderStatus.cancelled',
}

const orderStatusColors: Record<string, string> = {
  draft: 'bg-secondary text-muted-foreground',
  submitted: 'bg-info-light text-info',
  sent: 'bg-info-light text-info',
  confirmed: 'bg-primary-light text-primary',
  partial: 'bg-warning-light text-warning',
  partially_received: 'bg-warning-light text-warning',
  received: 'bg-success-light text-success',
  closed: 'bg-success-light text-success',
  cancelled: 'bg-error-light text-error',
}

type TimelineStatus = 'draft' | 'sent' | 'partially_received' | 'received'
const STATUS_TIMELINE: TimelineStatus[] = [
  'draft',
  'sent',
  'partially_received',
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

// STATUS_FILTER_OPTIONS labels are resolved with t() inside the component
const STATUS_FILTER_KEYS: StatusFilter[] = ['all', 'draft', 'submitted', 'sent', 'partially_received', 'received', 'cancelled']

const CURRENCY_OPTIONS = ['EUR', 'CHF', 'USD']

const ratingCategoryLabels: Record<string, string> = {
  quality: 'einkauf.ratingCategory.quality',
  delivery: 'einkauf.ratingCategory.delivery',
  price: 'einkauf.ratingCategory.price',
}

const contractStatusColors: Record<string, string> = {
  active: 'bg-success-light text-success',
  expired: 'bg-secondary text-muted-foreground',
  draft: 'bg-info-light text-info',
}

const contractStatusLabels: Record<string, string> = {
  active: 'einkauf.contractStatus.active',
  expired: 'einkauf.contractStatus.expired',
  draft: 'einkauf.contractStatus.draft',
}

interface NewOrderItem {
  name: string
  quantity: number
  unitPrice: number
}

// ---------------------------------------------------------------------------
// Helper: Star rating display
// ---------------------------------------------------------------------------
function StarRating({ rating, max = 5 }: { rating: number; max?: number }) {
  return (
    <div className="flex items-center gap-0.5">
      {Array.from({ length: max }, (_, i) => (
        <Star
          key={i}
          className={`h-3.5 w-3.5 ${
            i < rating ? 'fill-warning text-warning' : 'text-border'
          }`}
        />
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function EinkaufPage() {
  const { t } = useTranslation()

  const STATUS_FILTER_OPTIONS: { value: StatusFilter; label: string }[] = STATUS_FILTER_KEYS.map((key) => ({
    value: key,
    label: key === 'all' ? t('einkauf.statusFilter.all') : t(orderStatusLabels[key] ?? key),
  }))

  const { approvalThreshold } = useEinkaufStore()

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
  const [newOrderCurrency, setNewOrderCurrency] = useState('EUR')

  // New supplier form state
  const [newSupName, setNewSupName] = useState('')
  const [newSupContact, setNewSupContact] = useState('')
  const [newSupEmail, setNewSupEmail] = useState('')
  const [newSupPhone, setNewSupPhone] = useState('')
  const [newSupPayment, setNewSupPayment] = useState(PAYMENT_TERMS_OPTIONS[0])

  // Wareneingang state
  const [receivedQtys, setReceivedQtys] = useState<Record<string, number>>({})
  const [partialDelivery, setPartialDelivery] = useState(false)
  const [bookToInventory, setBookToInventory] = useState(false)

  // Katalog state
  const [catalogSearch, setCatalogSearch] = useState('')
  const [catalogCategory, setCatalogCategory] = useState<string>('all')

  // API data
  const { data: posData, isLoading: posLoading } = usePOs()
  const purchaseOrders = posData?.pos ?? []

  const { data: suppliersData, isLoading: suppliersLoading } = useSuppliers()
  const suppliers = suppliersData?.suppliers ?? []

  const { data: catalogData, isLoading: catalogLoading } = useCatalogItems({
    search: catalogSearch || undefined,
    category: catalogCategory !== 'all' ? catalogCategory : undefined,
  })
  const catalogItems = catalogData?.catalog_items ?? []

  const { data: contractsData, isLoading: contractsLoading } = useFrameworkContracts()
  const frameworkContracts = contractsData?.contracts ?? []

  // Rahmenverträge state
  const [expandedContract, setExpandedContract] = useState<string | null>(null)

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
        (o) => {
          const sup = suppliers.find(s => s.id === o.supplier_id)
          return (
            o.po_number.toLowerCase().includes(q) ||
            (sup?.name ?? '').toLowerCase().includes(q)
          )
        },
      )
    }
    return list
  }, [purchaseOrders, suppliers, search, statusFilter])

  const filteredSuppliers = useMemo(() => {
    if (!search) return suppliers
    const q = search.toLowerCase()
    return suppliers.filter(
      (s) =>
        s.name.toLowerCase().includes(q) ||
        s.email.toLowerCase().includes(q),
    )
  }, [suppliers, search])

  const catalogCategories = useMemo(() => {
    const cats = new Set(catalogItems.map((c) => c.category))
    return Array.from(cats).sort()
  }, [catalogItems])

  const filteredCatalog = useMemo(() => {
    // Catalog is already server-filtered via useCatalogItems params — local filtering as fallback
    let list = catalogItems
    if (catalogCategory !== 'all') {
      list = list.filter((c) => c.category === catalogCategory)
    }
    if (catalogSearch) {
      const q = catalogSearch.toLowerCase()
      const supMap = new Map(suppliers.map(s => [s.id, s.name]))
      list = list.filter(
        (c) =>
          c.name.toLowerCase().includes(q) ||
          c.sku.toLowerCase().includes(q) ||
          (supMap.get(c.supplier_id) ?? '').toLowerCase().includes(q),
      )
    }
    return list
  }, [catalogItems, catalogSearch, catalogCategory, suppliers])

  const activeOrderCount = purchaseOrders.filter(
    (o) => o.status !== 'received' && o.status !== 'closed' && o.status !== 'cancelled',
  ).length

  const totalOpen = purchaseOrders
    .filter((o) => o.status !== 'received' && o.status !== 'closed' && o.status !== 'cancelled')
    .reduce((sum, o) => sum + parseFloat(o.total_amount), 0)

  const activeSupplierCount = suppliers.length

  const newOrderTotal = useMemo(
    () => newOrderItems.reduce((s, i) => s + i.quantity * i.unitPrice, 0),
    [newOrderItems],
  )

  // Items for a given order (lines come embedded on the PO from API)
  const getOrderItems = useCallback(
    (orderId: string) => {
      const po = purchaseOrders.find(p => p.id === orderId)
      return po?.lines ?? []
    },
    [purchaseOrders],
  )

  // Orders for a given supplier
  const getSupplierOrders = useCallback(
    (supplierId: string) => purchaseOrders.filter((o) => o.supplier_id === supplierId),
    [purchaseOrders],
  )

  // Ratings for selected supplier (loaded via hook when supplier is selected)
  const { data: ratingsData } = useSupplierRatings(selectedSupplier?.id ?? '')
  const selectedSupplierRatings = ratingsData?.ratings ?? []

  const getSupplierAvgRating = useCallback(
    (ratings: typeof selectedSupplierRatings) => {
      if (ratings.length === 0) return 0
      return ratings.reduce((sum, r) => sum + r.rating, 0) / ratings.length
    },
    [],
  )

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------

  const getOrderActions = (order: PurchaseOrder) => [
    {
      label: t('einkauf.action.showDetails'),
      icon: Eye,
      onClick: () => {
        setSelectedSupplier(null)
        setSelectedOrder(order)
      },
    },
    {
      label: t('einkauf.action.bookReceipt'),
      icon: PackageCheck,
      onClick: () => openWareneingang(order),
    },
    { label: t('einkauf.action.edit'), icon: Edit, onClick: () => toast.info(t('einkauf.toast.editOrder', { orderNumber: order.po_number })) },
    { separator: true as const, label: '', onClick: () => {} },
    { label: t('einkauf.action.cancel'), icon: Trash2, variant: 'destructive' as const, onClick: () => setConfirmDelete(order) },
  ]

  const handleCancelOrder = (order: PurchaseOrder) => {
    setConfirmDelete(null)
    toast.success(t('einkauf.toast.orderCancelled', { orderNumber: order.po_number }))
  }

  // -- New order dialog helpers --
  const resetNewOrderForm = () => {
    setNewOrderSupplierId('')
    setNewOrderItems([{ name: '', quantity: 1, unitPrice: 0 }])
    setNewOrderDate('')
    setNewOrderNotes('')
    setNewOrderCurrency('EUR')
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
      toast.error(t('einkauf.validation.selectSupplier'))
      return
    }
    if (newOrderItems.some((i) => !i.name.trim())) {
      toast.error(t('einkauf.validation.itemsNeedName'))
      return
    }
    const sup = suppliers.find((s) => s.id === newOrderSupplierId)
    const nr = `PO-${new Date().getFullYear()}-${String(purchaseOrders.length + 1).padStart(3, '0')}`
    toast.success(t('einkauf.toast.orderCreated', { nr, supplier: sup?.name ?? t('einkauf.detail.supplierSection'), currency: newOrderCurrency, total: newOrderTotal.toLocaleString('de-DE', { minimumFractionDigits: 2 }) }))
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
      toast.error(t('einkauf.validation.supplierNameRequired'))
      return
    }
    toast.success(t('einkauf.toast.supplierCreated', { name: newSupName }))
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
    setBookToInventory(false)
    setShowWareneingangDialog(order)
  }

  const handleSaveWareneingang = () => {
    if (!showWareneingangDialog) return
    if (bookToInventory) {
      toast.success(t('einkauf.toast.wareneingangWithInventory', { orderNumber: showWareneingangDialog.po_number }))
    } else {
      toast.success(t('einkauf.toast.wareneingang', { orderNumber: showWareneingangDialog.po_number }))
    }
    setShowWareneingangDialog(null)
  }

  // -- Approval --
  const handleApproveOrder = (order: PurchaseOrder) => {
    toast.success(t('einkauf.toast.orderApproved', { orderNumber: order.po_number }))
  }

  // ---------------------------------------------------------------------------
  // Render helpers
  // ---------------------------------------------------------------------------


  const formatDateLocal = (d: string) =>
    formatDate(d.includes('T') ? d : d + 'T00:00:00')

  /** Status timeline for detail panel */
  const renderTimeline = (currentStatus: string) => {
    if (currentStatus === 'cancelled') {
      return (
        <div className="flex items-center gap-2 text-xs text-error">
          <X className="h-3.5 w-3.5" />
          <span>{t('einkauf.detail.cancelled')}</span>
        </div>
      )
    }
    const currentIdx = STATUS_TIMELINE.indexOf(currentStatus as TimelineStatus)
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
                  {orderStatusLabels[s] ? t(orderStatusLabels[s]) : s}
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
      <PageHeader
        title={t('einkauf.page.title')}
        description={t('einkauf.page.description', { count: activeOrderCount, amount: formatAmount(totalOpen) })}
        icon={ShoppingCart}
        moduleId="einkauf"
        className="mb-6"
        actions={
          <div className="flex items-center gap-2">
            {tab === 'lieferanten' && (
              <button
                onClick={() => {
                  resetNewSupplierForm()
                  setShowNewSupplierDialog(true)
                }}
                className="flex items-center gap-2 rounded-xl border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <Plus className="h-4 w-4" />
                {t('einkauf.action.addSupplier')}
              </button>
            )}
            <button
              onClick={() => {
                resetNewOrderForm()
                setShowNewOrderDialog(true)
              }}
              className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-4 w-4" />
              {t('einkauf.action.newOrder')}
            </button>
          </div>
        }
      />

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'bestellungen' as const, label: t('einkauf.tab.bestellungen', { count: purchaseOrders.length }) },
          { key: 'lieferanten' as const, label: t('einkauf.tab.lieferanten', { count: activeSupplierCount }) },
          { key: 'katalog' as const, label: t('einkauf.tab.katalog', { count: catalogData?.total ?? catalogItems.length }) },
          { key: 'rahmenverträge' as const, label: t('einkauf.tab.rahmenvertraege', { count: contractsData?.total ?? frameworkContracts.length }) },
        ]).map((tabItem) => (
          <button
            key={tabItem.key}
            onClick={() => {
              setTab(tabItem.key)
              setSearch('')
              setStatusFilter('all')
              setCatalogSearch('')
              setCatalogCategory('all')
            }}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === tabItem.key
                ? 'border-primary text-primary font-medium tab-accent-active'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {tabItem.label}
          </button>
        ))}
      </div>

      {/* Search + status filter (bestellungen / lieferanten) */}
      {(tab === 'bestellungen' || tab === 'lieferanten') && (
        <div className="flex flex-wrap items-center gap-3 mb-4">
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder={
                tab === 'bestellungen' ? t('einkauf.search.orders') : t('einkauf.search.suppliers')
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
          {posLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-6 w-6 border-2 border-primary border-t-transparent" />
            </div>
          ) : filteredOrders.length === 0 ? (
            <EmptyState
              icon={ShoppingCart}
              title={t('einkauf.empty.noOrders.title')}
              description={
                search || statusFilter !== 'all'
                  ? t('einkauf.empty.noOrders.descriptionFilter')
                  : t('einkauf.empty.noOrders.descriptionEmpty')
              }
            />
          ) : (
            <div className="overflow-x-auto rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-card">
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('einkauf.detail.orderNumber')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('einkauf.detail.supplier')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('einkauf.detail.status')}</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('einkauf.detail.amount')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('einkauf.detail.deliveryDate')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('einkauf.detail.createdAt')}</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground"></th>
                  </tr>
                </thead>
                <tbody>
                  {filteredOrders.map((order) => {
                    const sup = suppliers.find(s => s.id === order.supplier_id)
                    return (
                    <tr
                      key={order.id}
                      onClick={() => {
                        setSelectedSupplier(null)
                        setSelectedOrder(order)
                      }}
                      className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer"
                    >
                      <td className="px-4 py-3 font-medium text-foreground font-mono text-xs">
                        {order.po_number}
                      </td>
                      <td className="px-4 py-3 text-foreground">{sup?.name ?? order.supplier_id}</td>
                      <td className="px-4 py-3">
                        <span
                          className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                            orderStatusColors[order.status] ?? 'bg-secondary text-muted-foreground'
                          }`}
                        >
                          {orderStatusLabels[order.status] ? t(orderStatusLabels[order.status]) : order.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right text-foreground tabular-nums">
                        {formatCurrency(parseFloat(order.total_amount), order.currency || 'EUR')}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {order.expected_delivery_date ? formatDateLocal(order.expected_delivery_date) : '--'}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {formatDate(order.created_at)}
                      </td>
                      <td className="px-4 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                        <ItemActions items={getOrderActions(order)} />
                      </td>
                    </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* ====================== LIEFERANTEN TAB ====================== */}
      {tab === 'lieferanten' && (
        <>
          {suppliersLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-6 w-6 border-2 border-primary border-t-transparent" />
            </div>
          ) : filteredSuppliers.length === 0 ? (
            <EmptyState
              icon={Truck}
              title={t('einkauf.empty.noSuppliers.title')}
              description={search ? t('einkauf.empty.noSuppliers.descriptionFilter') : t('einkauf.empty.noSuppliers.descriptionEmpty')}
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
                          <p className="text-xs text-muted-foreground">{supplier.email}</p>
                        </div>
                      </div>
                      <div onClick={(e) => e.stopPropagation()}>
                        <ItemActions
                          items={[
                            {
                              label: t('einkauf.action.showDetails'),
                              icon: Eye,
                              onClick: () => {
                                setSelectedOrder(null)
                                setSelectedSupplier(supplier)
                              },
                            },
                            { label: t('einkauf.action.edit'), icon: Edit, onClick: () => toast.info(t('einkauf.toast.editSupplier', { name: supplier.name })) },
                            { separator: true as const, label: '', onClick: () => {} },
                            {
                              label: t('einkauf.action.deactivate'),
                              variant: 'destructive' as const,
                              onClick: () => toast.info(t('einkauf.toast.supplierDeactivated', { name: supplier.name })),
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
                        {supplier.payment_terms}
                      </span>
                      <span className="text-xs text-muted-foreground">
                        {supOrders.length !== 1
                        ? t('einkauf.supplier.orderCountPlural', { count: supOrders.length })
                        : t('einkauf.supplier.orderCount', { count: supOrders.length })
                      }
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
        <>
          {/* Katalog search + category filter */}
          <div className="flex flex-wrap items-center gap-3 mb-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder={t('einkauf.search.catalog')}
                value={catalogSearch}
                onChange={(e) => setCatalogSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <select
              value={catalogCategory}
              onChange={(e) => setCatalogCategory(e.target.value)}
              className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              <option value="all">{t('einkauf.catalog.allCategories')}</option>
              {catalogCategories.map((cat) => (
                <option key={cat} value={cat}>
                  {cat}
                </option>
              ))}
            </select>
          </div>

          {catalogLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-6 w-6 border-2 border-primary border-t-transparent" />
            </div>
          ) : filteredCatalog.length === 0 ? (
            <EmptyState
              icon={BookOpen}
              title={t('einkauf.empty.noCatalog.title')}
              description={t('einkauf.empty.noCatalog.description')}
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {filteredCatalog.map((item) => (
                <div
                  key={item.id}
                  className="rounded-lg border border-border bg-card p-4 flex flex-col"
                >
                  <div className="flex items-start justify-between mb-2">
                    <div className="flex-1 min-w-0">
                      <h4 className="text-sm font-medium text-foreground truncate">{item.name}</h4>
                      <p className="text-xs text-muted-foreground font-mono">{item.sku}</p>
                    </div>
                    <span
                      className={`ml-2 shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium ${
                        item.available
                          ? 'bg-success-light text-success'
                          : 'bg-error-light text-error'
                      }`}
                    >
                      {item.available ? t('einkauf.catalog.available') : t('einkauf.catalog.unavailable')}
                    </span>
                  </div>

                  <div className="space-y-1.5 text-xs text-muted-foreground mb-3 flex-1">
                    <div className="flex items-center gap-2">
                      <Truck className="h-3 w-3" />
                      <span>{suppliers.find(s => s.id === item.supplier_id)?.name ?? item.supplier_id}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <ScanBarcode className="h-3 w-3" />
                      <span>{t('einkauf.catalog.category', { category: item.category })}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <PackageCheck className="h-3 w-3" />
                      <span>{t('einkauf.catalog.minOrder', { qty: item.min_order_qty, unit: item.unit })}</span>
                    </div>
                  </div>

                  <div className="flex items-center justify-between border-t border-border-muted pt-3">
                    <div>
                      <p className="text-base font-semibold text-foreground tabular-nums">
                        {formatCurrency(parseFloat(item.price), item.currency)}
                      </p>
                      <p className="text-[10px] text-muted-foreground">{t('einkauf.catalog.perUnit', { unit: item.unit })}</p>
                    </div>
                    <button
                      onClick={() =>
                        toast.success(t('einkauf.toast.addedToCart', { name: item.name }))
                      }
                      className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
                    >
                      <ShoppingCart className="h-3.5 w-3.5" />
                      {t('einkauf.action.addToCart')}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {/* ====================== RAHMENVERTRAEGE TAB ====================== */}
      {tab === 'rahmenverträge' && (
        <>
          {contractsLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-6 w-6 border-2 border-primary border-t-transparent" />
            </div>
          ) : frameworkContracts.length === 0 ? (
            <EmptyState
              icon={ScrollText}
              title={t('einkauf.empty.noContracts.title')}
              description={t('einkauf.empty.noContracts.description')}
            />
          ) : (
            <div className="space-y-4">
              {frameworkContracts.map((contract) => {
                const isExpanded = expandedContract === contract.id
                const totalVal = parseFloat(contract.total_value)
                const usedVal = parseFloat(contract.used_value)
                const usagePct = totalVal > 0
                  ? Math.round((usedVal / totalVal) * 100)
                  : 0

                return (
                  <div
                    key={contract.id}
                    className="rounded-lg border border-border bg-card overflow-hidden"
                  >
                    {/* Contract header */}
                    <button
                      onClick={() => setExpandedContract(isExpanded ? null : contract.id)}
                      className="flex w-full items-center justify-between p-4 hover:bg-secondary/30 transition-colors"
                    >
                      <div className="flex items-center gap-3 text-left min-w-0">
                        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-light">
                          <ScrollText className="h-5 w-5 text-primary" />
                        </div>
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <h4 className="text-sm font-medium text-foreground truncate">{contract.title}</h4>
                            <span
                              className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium ${
                                contractStatusColors[contract.status]
                              }`}
                            >
                              {contractStatusLabels[contract.status] ? t(contractStatusLabels[contract.status]) : contract.status}
                            </span>
                          </div>
                          <div className="flex items-center gap-3 text-xs text-muted-foreground mt-0.5">
                            <span className="font-mono">{contract.contract_nr}</span>
                            <span>{suppliers.find(s => s.id === contract.supplier_id)?.name ?? contract.supplier_id}</span>
                            <span>
                              {contract.start_date ? formatDateLocal(contract.start_date) : '--'}
                              {' – '}
                              {contract.end_date ? formatDateLocal(contract.end_date) : '--'}
                            </span>
                          </div>
                        </div>
                      </div>

                      <div className="flex items-center gap-4 shrink-0 ml-4">
                        {/* Usage bar */}
                        <div className="w-32 hidden sm:block">
                          <div className="flex items-center justify-between text-[10px] text-muted-foreground mb-1">
                            <span>{t('einkauf.contract.usage')}</span>
                            <span>{usagePct}%</span>
                          </div>
                          <div className="h-2 rounded-full bg-secondary overflow-hidden">
                            <div
                              className={`h-full rounded-full transition-all ${
                                usagePct >= 90 ? 'bg-warning' : usagePct >= 70 ? 'bg-info' : 'bg-primary'
                              }`}
                              style={{ width: `${Math.min(usagePct, 100)}%` }}
                            />
                          </div>
                          <div className="flex items-center justify-between text-[10px] text-muted-foreground mt-0.5">
                            <span>{formatCurrency(usedVal, contract.currency)}</span>
                            <span>{formatCurrency(totalVal, contract.currency)}</span>
                          </div>
                        </div>
                        {isExpanded ? (
                          <ChevronUp className="h-4 w-4 text-muted-foreground" />
                        ) : (
                          <ChevronDown className="h-4 w-4 text-muted-foreground" />
                        )}
                      </div>
                    </button>

                    {/* Expanded items table */}
                    {isExpanded && (
                      <div className="border-t border-border">
                        <div className="overflow-x-auto">
                          <table className="w-full text-sm">
                            <thead>
                              <tr className="border-b border-border bg-secondary/30">
                                <th className="px-4 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">{t('einkauf.contract.article')}</th>
                                <th className="px-4 py-2 text-right text-[10px] font-medium text-muted-foreground uppercase">{t('einkauf.contract.unitPrice')}</th>
                                <th className="px-4 py-2 text-right text-[10px] font-medium text-muted-foreground uppercase">{t('einkauf.contract.agreed')}</th>
                                <th className="px-4 py-2 text-right text-[10px] font-medium text-muted-foreground uppercase">{t('einkauf.contract.called')}</th>
                                <th className="px-4 py-2 text-right text-[10px] font-medium text-muted-foreground uppercase">{t('einkauf.contract.remaining')}</th>
                              </tr>
                            </thead>
                            <tbody>
                              {(contract.items ?? []).map((item, idx) => {
                                const agreedQtyN = parseFloat(item.agreed_qty)
                                const calledQtyN = parseFloat(item.called_qty)
                                const remaining = agreedQtyN - calledQtyN
                                const callPct = agreedQtyN > 0
                                  ? Math.round((calledQtyN / agreedQtyN) * 100)
                                  : 0
                                return (
                                  <tr key={idx} className="border-b border-border-muted last:border-0">
                                    <td className="px-4 py-2.5 text-foreground">
                                      <div>
                                        <p className="text-sm">{item.name}</p>
                                        <p className="text-[10px] text-muted-foreground">{item.unit} · {t('einkauf.contract.callPct', { pct: callPct })}</p>
                                      </div>
                                    </td>
                                    <td className="px-4 py-2.5 text-right text-foreground tabular-nums">
                                      {formatCurrency(parseFloat(item.unit_price), contract.currency)}
                                    </td>
                                    <td className="px-4 py-2.5 text-right text-muted-foreground tabular-nums">
                                      {agreedQtyN.toLocaleString('de-DE')}
                                    </td>
                                    <td className="px-4 py-2.5 text-right text-foreground tabular-nums">
                                      {calledQtyN.toLocaleString('de-DE')}
                                    </td>
                                    <td className="px-4 py-2.5 text-right tabular-nums">
                                      <span className={remaining <= 0 ? 'text-error' : 'text-success'}>
                                        {remaining.toLocaleString('de-DE')}
                                      </span>
                                    </td>
                                  </tr>
                                )
                              })}
                            </tbody>
                          </table>
                        </div>

                        <div className="flex items-center justify-end px-4 py-3 border-t border-border-muted">
                          <button
                            onClick={() => toast.success(t('einkauf.toast.newCall', { title: contract.title }))}
                            className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
                          >
                            <Plus className="h-3.5 w-3.5" />
                            {t('einkauf.action.newCall')}
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

      {/* ====================== ORDER DETAIL PANEL ====================== */}
      <DetailPanel
        open={!!selectedOrder}
        onClose={() => setSelectedOrder(null)}
        title={selectedOrder?.po_number ?? ''}
        subtitle={selectedOrder ? (suppliers.find(s => s.id === selectedOrder.supplier_id)?.name ?? selectedOrder.supplier_id) : undefined}
        badge={
          selectedOrder ? (
            <span
              className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                orderStatusColors[selectedOrder.status] ?? 'bg-secondary text-muted-foreground'
              }`}
            >
              {orderStatusLabels[selectedOrder.status] ? t(orderStatusLabels[selectedOrder.status]) : selectedOrder.status}
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
                {t('einkauf.action.bookReceipt')}
              </button>
              <button
                onClick={() => toast.info(t('einkauf.toast.editOrder', { orderNumber: selectedOrder.po_number }))}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('einkauf.action.edit')}
              </button>
            </div>
          ) : undefined
        }
      >
        {selectedOrder && (
          <div className="space-y-5">
            {/* 8.4 Approval banner — shown when total exceeds threshold */}
            {parseFloat(selectedOrder.total_amount) > approvalThreshold && (
              <div className="rounded-lg p-3 flex items-center gap-3 bg-warning-light border border-warning/20">
                <ShieldCheck className="h-5 w-5 shrink-0 text-warning" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-warning font-medium">
                    {t('einkauf.detail.approvalRequired', { threshold: formatCurrency(approvalThreshold, selectedOrder.currency || 'EUR') })}
                  </p>
                </div>
                <button
                  onClick={() => handleApproveOrder(selectedOrder)}
                  className="shrink-0 rounded-lg bg-warning px-3 py-1.5 text-xs font-medium text-warning-foreground hover:opacity-90 transition-opacity"
                >
                  {t('einkauf.action.approve')}
                </button>
              </div>
            )}

            {/* Summary */}
            <div className="grid grid-cols-2 gap-3">
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">{t('einkauf.detail.amount')}</p>
                <p className="text-lg font-semibold text-foreground tabular-nums">
                  {formatCurrency(parseFloat(selectedOrder.total_amount), selectedOrder.currency || 'EUR')}
                </p>
              </div>
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">{t('einkauf.detail.positions')}</p>
                <p className="text-lg font-semibold text-foreground">{selectedOrder.lines?.length ?? 0}</p>
              </div>
            </div>

            {/* Dates */}
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm">
                <CalendarDays className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">{t('einkauf.detail.deliveryDateLabel')}</span>
                <span className="text-foreground">
                  {selectedOrder.expected_delivery_date ? formatDateLocal(selectedOrder.expected_delivery_date) : '--'}
                </span>
              </div>
              <div className="flex items-center gap-2 text-sm">
                <Clock className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">{t('einkauf.detail.createdLabel')}</span>
                <span className="text-foreground">{formatDate(selectedOrder.created_at)}</span>
              </div>
            </div>

            {/* Status timeline */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">
                {t('einkauf.detail.statusHistory')}
              </h4>
              {renderTimeline(selectedOrder.status)}
            </div>

            {/* Order items */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                {t('einkauf.detail.orderedPositions')}
              </h4>
              <div className="rounded-lg border border-border divide-y divide-border-muted">
                {getOrderItems(selectedOrder?.id ?? '').length === 0 ? (
                  <p className="px-3 py-4 text-xs text-muted-foreground text-center">
                    {t('einkauf.detail.noPositions')}
                  </p>
                ) : (
                  getOrderItems(selectedOrder?.id ?? '').map((item) => {
                    const qty = parseFloat(item.quantity)
                    const unitPrice = parseFloat(item.unit_price)
                    const receivedQty = parseFloat(item.received_quantity)
                    return (
                    <div key={item.id} className="flex items-center justify-between px-3 py-2.5">
                      <div>
                        <p className="text-sm text-foreground">{item.product_name}</p>
                        <p className="text-xs text-muted-foreground">
                          {t('einkauf.detail.qty', { qty, price: formatCurrency(unitPrice, selectedOrder.currency || 'EUR') })}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="text-sm font-medium text-foreground tabular-nums">
                          {formatCurrency(qty * unitPrice, selectedOrder.currency || 'EUR')}
                        </p>
                        {receivedQty > 0 && (
                          <p className="text-[10px] text-success">
                            {t('einkauf.detail.receivedQty', { received: receivedQty, total: qty })}
                          </p>
                        )}
                      </div>
                    </div>
                    )
                  })
                )}
              </div>
            </div>

            {/* Supplier link */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                {t('einkauf.detail.supplierSection')}
              </h4>
              <button
                onClick={() => {
                  const sup = suppliers.find((s) => s.id === selectedOrder.supplier_id)
                  if (sup) {
                    setSelectedOrder(null)
                    setTimeout(() => setSelectedSupplier(sup), 200)
                  }
                }}
                className="flex w-full items-center justify-between rounded-lg border border-border p-3 hover:bg-secondary/50 transition-colors"
              >
                <div className="flex items-center gap-2">
                  <Truck className="h-4 w-4 text-primary" />
                  <span className="text-sm text-foreground">
                    {suppliers.find(s => s.id === selectedOrder.supplier_id)?.name ?? selectedOrder.supplier_id}
                  </span>
                </div>
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              </button>
            </div>

            {/* 8.2 Belegkette (document chain) — only for non-draft, non-cancelled */}
            {selectedOrder.status !== 'draft' && selectedOrder.status !== 'cancelled' && (
              <div>
                <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                  {t('einkauf.detail.documentChain')}
                </h4>
                <div className="rounded-lg border border-border p-3 space-y-3">
                  <div className="flex items-center gap-2">
                    {/* Bestellung */}
                    <div className="flex items-center gap-1.5 rounded-lg bg-primary-light px-2.5 py-1.5">
                      <FileText className="h-3.5 w-3.5 text-primary" />
                      <span className="text-xs font-medium text-primary">{t('einkauf.detail.order')}</span>
                    </div>
                    <div className="h-px w-4 bg-border" />
                    {/* Lieferschein */}
                    <div className={`flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 ${
                      selectedOrder.status === 'partially_received' || selectedOrder.status === 'received' || selectedOrder.status === 'closed'
                        ? 'bg-success-light'
                        : 'bg-secondary'
                    }`}>
                      <Link2 className={`h-3.5 w-3.5 ${
                        selectedOrder.status === 'partially_received' || selectedOrder.status === 'received' || selectedOrder.status === 'closed'
                          ? 'text-success'
                          : 'text-muted-foreground'
                      }`} />
                      <span className={`text-xs font-medium ${
                        selectedOrder.status === 'partially_received' || selectedOrder.status === 'received' || selectedOrder.status === 'closed'
                          ? 'text-success'
                          : 'text-muted-foreground'
                      }`}>{t('einkauf.detail.deliveryNote')}</span>
                    </div>
                    <div className="h-px w-4 bg-border" />
                    {/* Eingangsrechnung */}
                    <div className="flex items-center gap-1.5 rounded-lg bg-secondary px-2.5 py-1.5">
                      <Coins className="h-3.5 w-3.5 text-muted-foreground" />
                      <span className="text-xs font-medium text-muted-foreground">{t('einkauf.detail.incomingInvoice')}</span>
                    </div>
                  </div>
                  <button
                    onClick={() => toast.info(t('einkauf.toast.assignInvoice', { orderNumber: selectedOrder.po_number }))}
                    className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors"
                  >
                    <Link2 className="h-3 w-3" />
                    {t('einkauf.action.assignInvoice')}
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </DetailPanel>

      {/* ====================== SUPPLIER DETAIL PANEL ====================== */}
      <DetailPanel
        open={!!selectedSupplier}
        onClose={() => setSelectedSupplier(null)}
        title={selectedSupplier?.name ?? ''}
        subtitle={selectedSupplier?.email}
        width="w-[420px]"
        footer={
          selectedSupplier ? (
            <div className="flex items-center gap-2">
              <button
                onClick={() => toast.info(t('einkauf.toast.editSupplier', { name: selectedSupplier.name }))}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('einkauf.action.edit')}
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
                  <p className="text-sm font-medium text-foreground">{selectedSupplier.name}</p>
                  <p className="text-xs text-muted-foreground">{t('einkauf.supplier.contactPerson')}</p>
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
                  <span className="text-sm text-foreground">{selectedSupplier.payment_terms}</span>
                </div>
              </div>
            </div>

            {/* 8.3 Supplier rating */}
            {(() => {
              const ratings = selectedSupplierRatings
              const avg = getSupplierAvgRating(ratings)
              if (ratings.length === 0) return null
              return (
                <div>
                  <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                    {t('einkauf.detail.ratings')}
                  </h4>
                  <div className="rounded-lg border border-border p-3 space-y-3">
                    {/* Average */}
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-warning-light">
                        <BarChart3 className="h-5 w-5 text-warning" />
                      </div>
                      <div>
                        <p className="text-lg font-semibold text-foreground tabular-nums">
                          {avg.toFixed(1)}
                        </p>
                        <p className="text-[10px] text-muted-foreground">{t('einkauf.detail.average')}</p>
                      </div>
                      <div className="ml-auto">
                        <StarRating rating={Math.round(avg)} />
                      </div>
                    </div>

                    {/* Category breakdown */}
                    <div className="space-y-2 border-t border-border-muted pt-3">
                      {(['quality', 'delivery', 'price'] as const).map((cat) => {
                        const catRating = ratings.find((r) => r.category === cat)
                        if (!catRating) return null
                        return (
                          <div key={cat} className="flex items-center justify-between">
                            <span className="text-xs text-muted-foreground">
                              {ratingCategoryLabels[cat] ? t(ratingCategoryLabels[cat]) : cat}
                            </span>
                            <div className="flex items-center gap-2">
                              <StarRating rating={catRating.rating} />
                              <span className="text-xs text-foreground font-medium tabular-nums w-4 text-right">
                                {catRating.rating}
                              </span>
                            </div>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                </div>
              )
            })()}

            {/* Status */}
            <div className="flex items-center gap-2">
              <span className="rounded-full px-2 py-0.5 text-[10px] font-medium bg-success-light text-success">
                {t('einkauf.supplier.active')}
              </span>
            </div>

            {/* Recent orders */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                {t('einkauf.detail.orders', { count: getSupplierOrders(selectedSupplier.id).length })}
              </h4>
              <div className="rounded-lg border border-border divide-y divide-border-muted">
                {getSupplierOrders(selectedSupplier.id).length === 0 ? (
                  <p className="px-3 py-4 text-xs text-muted-foreground text-center">
                    {t('einkauf.detail.noOrders')}
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
                        <p className="text-sm font-mono text-foreground">{order.po_number}</p>
                        <p className="text-xs text-muted-foreground">
                          {formatDate(order.created_at)} · {formatCurrency(parseFloat(order.total_amount), order.currency || 'EUR')}
                        </p>
                      </div>
                      <span
                        className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                          orderStatusColors[order.status] ?? 'bg-secondary text-muted-foreground'
                        }`}
                      >
                        {orderStatusLabels[order.status] ? t(orderStatusLabels[order.status]) : order.status}
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
      <Dialog open={showNewOrderDialog} onOpenChange={(o) => { if (!o) setShowNewOrderDialog(false) }}>
        <DialogContent className="gap-0 p-0 max-w-lg max-h-[85vh] flex flex-col">
            {/* Header */}
            <DialogHeader className="border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <ShoppingCart className="h-5 w-5 text-primary" />
                <DialogTitle className="text-base font-semibold text-foreground">{t('einkauf.newOrder.title')}</DialogTitle>
              </div>
              <DialogDescription className="sr-only">{t('einkauf.newOrder.title')}</DialogDescription>
            </DialogHeader>

            {/* Body */}
            <div className="flex-1 overflow-y-auto p-5 space-y-4">
              {/* Supplier */}
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  {t('einkauf.newOrder.supplierLabel')} <span className="text-destructive">*</span>
                </label>
                <select
                  value={newOrderSupplierId}
                  onChange={(e) => setNewOrderSupplierId(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  <option value="">{t('einkauf.newOrder.supplierPlaceholder')}</option>
                  {suppliers
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
                  <label className="text-sm font-medium text-foreground">{t('einkauf.newOrder.positions')}</label>
                  <button
                    onClick={addOrderItemRow}
                    className="flex items-center gap-1 rounded-lg border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-secondary transition-colors"
                  >
                    <Plus className="h-3 w-3" />
                    {t('einkauf.newOrder.addPosition')}
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
                          placeholder={t('einkauf.newOrder.itemNamePlaceholder')}
                          value={item.name}
                          onChange={(e) => updateOrderItem(idx, 'name', e.target.value)}
                          className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                        />
                        <div className="flex gap-2">
                          <div className="flex-1">
                            <label className="text-[10px] text-muted-foreground">{t('einkauf.newOrder.qty')}</label>
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
                            <label className="text-[10px] text-muted-foreground">{t('einkauf.newOrder.unitPrice', { currency: newOrderCurrency })}</label>
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

              {/* Total + Currency selector */}
              <div className="flex items-center justify-between rounded-lg bg-secondary/50 px-3 py-2">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-muted-foreground">{t('einkauf.newOrder.total')}</span>
                  <select
                    value={newOrderCurrency}
                    onChange={(e) => setNewOrderCurrency(e.target.value)}
                    className="rounded border border-border bg-card px-2 py-0.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  >
                    {CURRENCY_OPTIONS.map((c) => (
                      <option key={c} value={c}>{c}</option>
                    ))}
                  </select>
                </div>
                <span className="text-base font-semibold text-foreground tabular-nums">
                  {formatCurrency(newOrderTotal, newOrderCurrency)}
                </span>
              </div>

              {/* 8.4 Approval notice in new order dialog */}
              {newOrderTotal > approvalThreshold && (
                <div className="flex items-center gap-2 rounded-lg bg-warning-light border border-warning/20 px-3 py-2">
                  <ShieldCheck className="h-4 w-4 text-warning shrink-0" />
                  <p className="text-xs text-warning font-medium">
                    {t('einkauf.newOrder.approvalRequired', { threshold: formatCurrency(approvalThreshold, newOrderCurrency) })}
                  </p>
                </div>
              )}

              {/* Expected delivery */}
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">{t('einkauf.newOrder.deliveryDate')}</label>
                <input
                  type="date"
                  value={newOrderDate}
                  onChange={(e) => setNewOrderDate(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>

              {/* Notes */}
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">{t('einkauf.newOrder.notes')}</label>
                <textarea
                  value={newOrderNotes}
                  onChange={(e) => setNewOrderNotes(e.target.value)}
                  rows={3}
                  placeholder={t('einkauf.newOrder.notesPlaceholder')}
                  className="w-full resize-none rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
            </div>

            {/* Footer */}
            <DialogFooter className="border-t border-border px-5 py-4">
              <button
                onClick={() => setShowNewOrderDialog(false)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('common.cancel')}
              </button>
              <button
                onClick={handleSaveOrder}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                {t('einkauf.newOrder.create')}
              </button>
            </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ====================== LIEFERANT ANLEGEN DIALOG ====================== */}
      <Dialog open={showNewSupplierDialog} onOpenChange={(o) => { if (!o) setShowNewSupplierDialog(false) }}>
        <DialogContent className="gap-0 p-0 max-w-md">
            {/* Header */}
            <DialogHeader className="border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <Truck className="h-5 w-5 text-primary" />
                <DialogTitle className="text-base font-semibold text-foreground">{t('einkauf.newSupplier.title')}</DialogTitle>
              </div>
              <DialogDescription className="sr-only">{t('einkauf.newSupplier.title')}</DialogDescription>
            </DialogHeader>

            {/* Body */}
            <div className="p-5 space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  {t('einkauf.newSupplier.name')} <span className="text-destructive">*</span>
                </label>
                <input
                  type="text"
                  value={newSupName}
                  onChange={(e) => setNewSupName(e.target.value)}
                  placeholder={t('einkauf.newSupplier.namePlaceholder')}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">{t('einkauf.newSupplier.contact')}</label>
                <input
                  type="text"
                  value={newSupContact}
                  onChange={(e) => setNewSupContact(e.target.value)}
                  placeholder={t('einkauf.newSupplier.contactPlaceholder')}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">{t('einkauf.newSupplier.email')}</label>
                  <input
                    type="email"
                    value={newSupEmail}
                    onChange={(e) => setNewSupEmail(e.target.value)}
                    placeholder={t('einkauf.newSupplier.emailPlaceholder')}
                    className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">{t('einkauf.newSupplier.phone')}</label>
                  <input
                    type="tel"
                    value={newSupPhone}
                    onChange={(e) => setNewSupPhone(e.target.value)}
                    placeholder={t('einkauf.newSupplier.phonePlaceholder')}
                    className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">{t('einkauf.newSupplier.paymentTerms')}</label>
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
            <DialogFooter className="border-t border-border px-5 py-4">
              <button
                onClick={() => setShowNewSupplierDialog(false)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('common.cancel')}
              </button>
              <button
                onClick={handleSaveSupplier}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                {t('einkauf.newSupplier.save')}
              </button>
            </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ====================== WARENEINGANG DIALOG ====================== */}
      <Dialog open={!!showWareneingangDialog} onOpenChange={(o) => { if (!o) setShowWareneingangDialog(null) }}>
        <DialogContent className="gap-0 p-0 max-w-lg max-h-[80vh] flex flex-col">
            {/* Header */}
            <DialogHeader className="border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <PackageCheck className="h-5 w-5 text-primary" />
                <div>
                  <DialogTitle className="text-base font-semibold text-foreground">{t('einkauf.wareneingang.title')}</DialogTitle>
                  <DialogDescription className="text-xs text-muted-foreground">
                    {showWareneingangDialog?.po_number} — {showWareneingangDialog ? (suppliers.find(s => s.id === showWareneingangDialog.supplier_id)?.name ?? showWareneingangDialog.supplier_id) : ''}
                  </DialogDescription>
                </div>
              </div>
            </DialogHeader>

            {/* Body */}
            <div className="flex-1 overflow-y-auto p-5 space-y-4">
              <div className="rounded-lg border border-border divide-y divide-border-muted">
                <div className="grid grid-cols-[1fr_80px_80px] gap-2 px-3 py-2 bg-secondary/30">
                  <span className="text-[10px] font-medium text-muted-foreground uppercase">{t('einkauf.wareneingang.article')}</span>
                  <span className="text-[10px] font-medium text-muted-foreground uppercase text-right">{t('einkauf.wareneingang.ordered')}</span>
                  <span className="text-[10px] font-medium text-muted-foreground uppercase text-right">{t('einkauf.wareneingang.received')}</span>
                </div>
                {getOrderItems(showWareneingangDialog?.id ?? '').map((item) => {
                  const itemQty = parseFloat(item.quantity)
                  const itemReceived = parseFloat(item.received_quantity)
                  return (<div key={item.id} className="grid grid-cols-[1fr_80px_80px] gap-2 items-center px-3 py-2.5">
                    <div>
                      <p className="text-sm text-foreground">{item.product_name}</p>
                      <p className="text-[10px] text-muted-foreground">
                        {t('einkauf.wareneingang.alreadyReceived', { qty: itemReceived })}
                      </p>
                    </div>
                    <p className="text-sm text-muted-foreground text-right tabular-nums">{itemQty}</p>
                    <input
                      type="number"
                      min={0}
                      max={itemQty - itemReceived}
                      value={receivedQtys[item.id] ?? 0}
                      onChange={(e) =>
                        setReceivedQtys((prev) => ({
                          ...prev,
                          [item.id]: Math.max(0, parseInt(e.target.value) || 0),
                        }))
                      }
                      className="w-full rounded-lg border border-border bg-card px-2 py-1 text-sm text-foreground text-right tabular-nums focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    />
                  </div>)
                })}
              </div>

              {/* Partial delivery checkbox */}
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={partialDelivery}
                  onChange={(e) => setPartialDelivery(e.target.checked)}
                  className="h-4 w-4 rounded border-border text-primary focus:ring-focus-ring"
                />
                <span className="text-sm text-foreground">{t('einkauf.wareneingang.partialDelivery')}</span>
              </label>

              {/* 8.6 Inventar-Integration */}
              <label className="flex items-start gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={bookToInventory}
                  onChange={(e) => setBookToInventory(e.target.checked)}
                  className="h-4 w-4 mt-0.5 rounded border-border text-primary focus:ring-focus-ring"
                />
                <div>
                  <span className="text-sm text-foreground">{t('einkauf.wareneingang.bookToInventory')}</span>
                  {bookToInventory && (
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {t('einkauf.wareneingang.bookToInventoryHint')}
                    </p>
                  )}
                </div>
              </label>
            </div>

            {/* Footer */}
            <DialogFooter className="border-t border-border px-5 py-4">
              <button
                onClick={() => setShowWareneingangDialog(null)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('common.cancel')}
              </button>
              <button
                onClick={handleSaveWareneingang}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                {t('einkauf.wareneingang.book')}
              </button>
            </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ====================== CONFIRM CANCEL ORDER ====================== */}
      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title={t('einkauf.confirmCancel.title')}
        description={t('einkauf.confirmCancel.description', {
          orderNumber: confirmDelete?.po_number,
          supplierName: confirmDelete ? (suppliers.find(s => s.id === confirmDelete.supplier_id)?.name ?? confirmDelete.supplier_id) : '',
        })}
        confirmLabel={t('einkauf.confirmCancel.confirm')}
        variant="destructive"
        onConfirm={() => confirmDelete && handleCancelOrder(confirmDelete)}
      />
    </div>
  )
}
