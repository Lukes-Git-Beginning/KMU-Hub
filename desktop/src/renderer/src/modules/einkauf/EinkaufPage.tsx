import { useState, useMemo, useEffect } from 'react'
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
  PackageCheck,
  ShieldCheck,
  ScrollText,
  ScanBarcode,
  ChevronDown,
  ChevronUp,
  FileDown,
  Ban,
} from 'lucide-react'
import { toast } from 'sonner'
import type { Supplier, PurchaseOrder, CatalogItem } from '@/api/einkauf-types'
import {
  useSuppliers,
  usePOs,
  useCatalogItems,
  useFrameworkContracts,
  useCreatePO,
  useAddPOLine,
  useCreateSupplier,
  useDeletePO,
  useCancelPO,
  useDeleteSupplier,
} from '@/api/hooks/useEinkauf'
import { ItemActions, ConfirmDialog, EmptyState, PageHeader, SortMenu } from '@/components/shared'
import type { SortDirection } from '@/components/shared/SortMenu'
import { useCapabilitySet, useHasCapability } from '@/hooks/useCapability'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'
import { formatAmount, formatCurrency, formatDate } from '@/lib/format'
import { useEinkaufPrefsStore, type EinkaufTab, type EinkaufStatusFilter } from '@/stores/einkaufPrefs'
import { useEinkaufTenantStore } from '@/stores/einkaufTenant'
import {
  orderStatusLabels,
  orderStatusColors,
  contractStatusColors,
  contractStatusLabels,
  PAYMENT_TERMS_OPTIONS,
  canReceiveGoods,
  isOpenOrder,
} from './einkauf-shared'
import { OrderDetailModal, SupplierDetailModal } from './EinkaufDetailModals'
import {
  EditOrderDialog,
  EditSupplierDialog,
  CartDialog,
  NewCallDialog,
  ContractCallsList,
  WareneingangDialog,
  type CartEntry,
} from './EinkaufDialogs'
import { buildPOsCsv, buildSuppliersCsv, csvDateStamp, downloadBlob } from './einkauf-export'

type TabKey = EinkaufTab

interface NewOrderItem {
  name: string
  quantity: number
  unitPrice: number
}

const STATUS_FILTER_KEYS: EinkaufStatusFilter[] = [
  'all', 'draft', 'submitted', 'sent', 'partially_received', 'received', 'cancelled',
]

const CURRENCY_OPTIONS = ['EUR', 'CHF', 'USD']

/** Keyboard affordance for clickable rows/cards (role=button). */
function rowKeyHandler(open: () => void) {
  return (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      open()
    }
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function EinkaufPage() {
  const { t } = useTranslation()

  // RBAC R-3 capability checks (default hidden per gating convention)
  const { has: capHas, ready: capReady } = useCapabilitySet()
  const canCreatePO = useHasCapability('einkauf:po:create')
  const canEditPO = useHasCapability('einkauf:po:edit')
  const canDeletePO = useHasCapability('einkauf:po:delete')
  const canCancelPO = useHasCapability('einkauf:po:cancel')
  const canReceivePO = useHasCapability('einkauf:po:receive')
  const canCreateSupplier = useHasCapability('einkauf:supplier:create')
  const canEditSupplier = useHasCapability('einkauf:supplier:edit')
  const canDeactivateSupplier = useHasCapability('einkauf:supplier:deactivate')
  const canExport = useHasCapability('einkauf:export:run')
  const canCallContract = useHasCapability('einkauf:contract:call')

  const STATUS_FILTER_OPTIONS: { value: EinkaufStatusFilter; label: string }[] = STATUS_FILTER_KEYS.map((key) => ({
    value: key,
    label: key === 'all' ? t('einkauf.statusFilter.all') : t(orderStatusLabels[key] ?? key),
  }))

  // Settings (personal prefs seed the initial view; tenant policies feed dialogs)
  const approvalThreshold = useEinkaufTenantStore((s) => s.approvalThreshold)
  const tenantCurrency = useEinkaufTenantStore((s) => s.currency)
  const defaultPaymentTerms = useEinkaufTenantStore((s) => s.defaultPaymentTerms)
  const poNumberPrefix = useEinkaufTenantStore((s) => s.poNumberPrefix)

  // Tab & search
  const [tab, setTab] = useState<TabKey>(() => useEinkaufPrefsStore.getState().defaultTab)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<EinkaufStatusFilter>(
    () => useEinkaufPrefsStore.getState().defaultStatusFilter,
  )

  // RBAC tab gating — while permissions load keep all tabs visible to avoid flicker
  const TAB_CAPABILITY: Record<TabKey, string> = {
    bestellungen: 'einkauf:po:read',
    lieferanten: 'einkauf:supplier:read',
    katalog: 'einkauf:catalog:read',
    'rahmenverträge': 'einkauf:contract:read',
  }
  const visibleTabs = (Object.keys(TAB_CAPABILITY) as TabKey[]).filter(
    (key) => !capReady || capHas(TAB_CAPABILITY[key]),
  )
  useEffect(() => {
    if (!capReady) return
    if (visibleTabs.length > 0 && !visibleTabs.includes(tab)) setTab(visibleTabs[0])
  }, [capReady, visibleTabs, tab])

  // Sorting
  const [orderSort, setOrderSort] = useState<{ field: string; direction: SortDirection }>({
    field: 'created',
    direction: 'desc',
  })
  const [supplierSort, setSupplierSort] = useState<{ field: string; direction: SortDirection }>({
    field: 'name',
    direction: 'asc',
  })

  // Detail modals + back chain
  const [selectedOrder, setSelectedOrder] = useState<PurchaseOrder | null>(null)
  const [selectedSupplier, setSelectedSupplier] = useState<Supplier | null>(null)
  const [orderBackTarget, setOrderBackTarget] = useState<Supplier | null>(null)
  const [supplierBackTarget, setSupplierBackTarget] = useState<PurchaseOrder | null>(null)

  // Dialogs
  const [confirmCancel, setConfirmCancel] = useState<PurchaseOrder | null>(null)
  const [confirmDeactivate, setConfirmDeactivate] = useState<Supplier | null>(null)
  const [showNewOrderDialog, setShowNewOrderDialog] = useState(false)
  const [showNewSupplierDialog, setShowNewSupplierDialog] = useState(false)
  const [wareneingangOrder, setWareneingangOrder] = useState<PurchaseOrder | null>(null)
  const [editOrder, setEditOrder] = useState<PurchaseOrder | null>(null)
  const [editSupplier, setEditSupplier] = useState<Supplier | null>(null)
  const [callContract, setCallContract] = useState<import('@/api/einkauf-types').FrameworkContract | null>(null)

  // Warenkorb (cart snapshots the catalog item so search filters can't drop it)
  const [cart, setCart] = useState<Record<string, CartEntry>>({})
  const [showCartDialog, setShowCartDialog] = useState(false)
  const cartEntries = useMemo(() => Object.values(cart), [cart])
  const cartCount = cartEntries.length

  // New order form state
  const [newOrderSupplierId, setNewOrderSupplierId] = useState('')
  const [newOrderItems, setNewOrderItems] = useState<NewOrderItem[]>([
    { name: '', quantity: 1, unitPrice: 0 },
  ])
  const [newOrderDate, setNewOrderDate] = useState('')
  const [newOrderNotes, setNewOrderNotes] = useState('')
  const [newOrderCurrency, setNewOrderCurrency] = useState(tenantCurrency)

  // New supplier form state
  const [newSupName, setNewSupName] = useState('')
  const [newSupContact, setNewSupContact] = useState('')
  const [newSupEmail, setNewSupEmail] = useState('')
  const [newSupPhone, setNewSupPhone] = useState('')
  const [newSupPayment, setNewSupPayment] = useState(defaultPaymentTerms)

  // Katalog state
  const [catalogSearch, setCatalogSearch] = useState('')
  const [catalogCategory, setCatalogCategory] = useState<string>('all')

  // API data
  const { data: posData, isLoading: posLoading } = usePOs()
  const purchaseOrders = useMemo(() => posData?.pos ?? [], [posData])

  const { data: suppliersData, isLoading: suppliersLoading } = useSuppliers()
  const suppliers = useMemo(() => suppliersData?.suppliers ?? [], [suppliersData])

  const { data: catalogData, isLoading: catalogLoading } = useCatalogItems({
    search: catalogSearch || undefined,
    category: catalogCategory !== 'all' ? catalogCategory : undefined,
  })
  const catalogItems = useMemo(() => catalogData?.catalog_items ?? [], [catalogData])

  const { data: contractsData, isLoading: contractsLoading } = useFrameworkContracts()
  const frameworkContracts = contractsData?.contracts ?? []

  // Rahmenverträge state
  const [expandedContract, setExpandedContract] = useState<string | null>(null)

  // ---------------------------------------------------------------------------
  // Derived data
  // ---------------------------------------------------------------------------

  const supplierNameById = useMemo(
    () => new Map(suppliers.map((s) => [s.id, s.name])),
    [suppliers],
  )

  const filteredOrders = useMemo(() => {
    let list = purchaseOrders
    if (statusFilter !== 'all') {
      list = list.filter((o) => o.status === statusFilter)
    }
    if (search) {
      const q = search.toLowerCase()
      list = list.filter(
        (o) =>
          o.po_number.toLowerCase().includes(q) ||
          (supplierNameById.get(o.supplier_id) ?? '').toLowerCase().includes(q),
      )
    }
    return list
  }, [purchaseOrders, supplierNameById, search, statusFilter])

  const sortedOrders = useMemo(() => {
    const dir = orderSort.direction === 'asc' ? 1 : -1
    return [...filteredOrders].sort((a, b) => {
      switch (orderSort.field) {
        case 'number':
          return a.po_number.localeCompare(b.po_number) * dir
        case 'supplier':
          return (supplierNameById.get(a.supplier_id) ?? '').localeCompare(supplierNameById.get(b.supplier_id) ?? '') * dir
        case 'total':
          return (parseFloat(a.total_amount) - parseFloat(b.total_amount)) * dir
        case 'delivery':
          return (a.expected_delivery_date ?? '').localeCompare(b.expected_delivery_date ?? '') * dir
        default:
          return a.created_at.localeCompare(b.created_at) * dir
      }
    })
  }, [filteredOrders, orderSort, supplierNameById])

  const orderCountBySupplier = useMemo(() => {
    const map = new Map<string, number>()
    for (const o of purchaseOrders) {
      map.set(o.supplier_id, (map.get(o.supplier_id) ?? 0) + 1)
    }
    return map
  }, [purchaseOrders])

  const filteredSuppliers = useMemo(() => {
    if (!search) return suppliers
    const q = search.toLowerCase()
    return suppliers.filter(
      (s) => s.name.toLowerCase().includes(q) || s.email.toLowerCase().includes(q),
    )
  }, [suppliers, search])

  const sortedSuppliers = useMemo(() => {
    const dir = supplierSort.direction === 'asc' ? 1 : -1
    return [...filteredSuppliers].sort((a, b) => {
      switch (supplierSort.field) {
        case 'orders':
          return ((orderCountBySupplier.get(a.id) ?? 0) - (orderCountBySupplier.get(b.id) ?? 0)) * dir
        default:
          return a.name.localeCompare(b.name) * dir
      }
    })
  }, [filteredSuppliers, supplierSort, orderCountBySupplier])

  const catalogCategories = useMemo(() => {
    const cats = new Set(catalogItems.map((c) => c.category))
    return Array.from(cats).sort()
  }, [catalogItems])

  const activeOrderCount = purchaseOrders.filter((o) => isOpenOrder(o.status)).length

  const totalOpen = purchaseOrders
    .filter((o) => isOpenOrder(o.status))
    .reduce((sum, o) => sum + parseFloat(o.total_amount), 0)

  const newOrderTotal = useMemo(
    () => newOrderItems.reduce((s, i) => s + i.quantity * i.unitPrice, 0),
    [newOrderItems],
  )

  // Orders for a given supplier
  const getSupplierOrders = (supplierId: string) =>
    purchaseOrders.filter((o) => o.supplier_id === supplierId)

  // ---------------------------------------------------------------------------
  // Write mutations
  // ---------------------------------------------------------------------------
  const createPOMutation = useCreatePO()
  const addPOLineMutation = useAddPOLine()
  const createSupplierMutation = useCreateSupplier()
  const deletePOMutation = useDeletePO()
  const cancelPOMutation = useCancelPO()
  const deleteSupplierMutation = useDeleteSupplier()

  // ---------------------------------------------------------------------------
  // Modal open helpers (back chain)
  // ---------------------------------------------------------------------------

  const openOrder = (order: PurchaseOrder) => {
    setSelectedSupplier(null)
    setOrderBackTarget(null)
    setSupplierBackTarget(null)
    setSelectedOrder(order)
  }

  const openSupplier = (supplier: Supplier) => {
    setSelectedOrder(null)
    setOrderBackTarget(null)
    setSupplierBackTarget(null)
    setSelectedSupplier(supplier)
  }

  const openSupplierFromOrder = (supplier: Supplier) => {
    setSupplierBackTarget(selectedOrder)
    setSelectedOrder(null)
    setSelectedSupplier(supplier)
  }

  const openOrderFromSupplier = (order: PurchaseOrder) => {
    setOrderBackTarget(selectedSupplier)
    setSelectedSupplier(null)
    setSelectedOrder(order)
  }

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------

  const getOrderActions = (order: PurchaseOrder) => [
    {
      label: t('einkauf.action.showDetails'),
      icon: Eye,
      onClick: () => openOrder(order),
    },
    ...(canReceiveGoods(order.status) && canReceivePO
      ? [{
          label: t('einkauf.action.bookReceipt'),
          icon: PackageCheck,
          onClick: () => setWareneingangOrder(order),
        }]
      : []),
    ...(canEditPO
      ? [{ label: t('einkauf.action.edit'), icon: Edit, onClick: () => setEditOrder(order) }]
      : []),
    ...(order.status === 'draft' && canDeletePO
      ? [
          { separator: true as const, label: '', onClick: () => {} },
          {
            label: t('einkauf.action.deleteDraft'),
            icon: Trash2,
            variant: 'destructive' as const,
            onClick: () => setConfirmCancel(order),
          },
        ]
      : []),
    ...(order.status !== 'draft' && canCancelPO
      ? [
          { separator: true as const, label: '', onClick: () => {} },
          {
            label: t('einkauf.action.cancel'),
            icon: Ban,
            variant: 'destructive' as const,
            onClick: () => setConfirmCancel(order),
          },
        ]
      : []),
  ]

  const handleCancelOrder = (order: PurchaseOrder) => {
    setConfirmCancel(null)
    const isDraft = order.status === 'draft'
    const mutation = isDraft ? deletePOMutation : cancelPOMutation
    mutation.mutate(order.id, {
      onSuccess: () => {
        toast.success(
          t(isDraft ? 'einkauf.toast.draftDeleted' : 'einkauf.toast.orderCancelled', {
            orderNumber: order.po_number,
          }),
        )
        if (selectedOrder?.id === order.id) setSelectedOrder(null)
      },
      onError: () => toast.error(t('einkauf.toast.orderCancelFailed')),
    })
  }

  const handleDeactivateSupplier = (supplier: Supplier) => {
    setConfirmDeactivate(null)
    deleteSupplierMutation.mutate(supplier.id, {
      onSuccess: () => {
        toast.success(t('einkauf.toast.supplierDeactivated', { name: supplier.name }))
        if (selectedSupplier?.id === supplier.id) setSelectedSupplier(null)
      },
      onError: () => toast.error(t('einkauf.toast.supplierDeactivateFailed')),
    })
  }

  // -- New order dialog helpers --
  const resetNewOrderForm = () => {
    setNewOrderSupplierId('')
    setNewOrderItems([{ name: '', quantity: 1, unitPrice: 0 }])
    setNewOrderDate('')
    setNewOrderNotes('')
    setNewOrderCurrency(tenantCurrency)
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
    const nr = `${poNumberPrefix}-${new Date().getFullYear()}-${String(purchaseOrders.length + 1).padStart(3, '0')}`

    createPOMutation.mutate(
      {
        supplier_id: newOrderSupplierId,
        po_number: nr,
        order_date: new Date().toISOString(),
        expected_delivery_date: newOrderDate || undefined,
        currency: newOrderCurrency,
        notes: newOrderNotes || undefined,
      },
      {
        onSuccess: (data) => {
          const poId = data?.id
          if (poId) {
            // Add each line sequentially via fire-and-forget; failures are
            // surfaced as individual toasts but do not roll back the PO.
            newOrderItems.forEach((item, idx) => {
              addPOLineMutation.mutate({
                poId,
                product_name: item.name,
                quantity: String(item.quantity),
                unit_price: String(item.unitPrice),
                line_position: idx,
              })
            })
          }
          toast.success(
            t('einkauf.toast.orderCreated', {
              nr,
              supplier: sup?.name ?? t('einkauf.detail.supplierSection'),
              currency: newOrderCurrency,
              total: newOrderTotal.toLocaleString('de-DE', { minimumFractionDigits: 2 }),
            }),
          )
          resetNewOrderForm()
          setShowNewOrderDialog(false)
        },
        onError: () => {
          toast.error(t('einkauf.toast.orderCreateFailed'))
        },
      },
    )
  }

  // -- New supplier dialog helpers --
  const resetNewSupplierForm = () => {
    setNewSupName('')
    setNewSupContact('')
    setNewSupEmail('')
    setNewSupPhone('')
    setNewSupPayment(defaultPaymentTerms)
  }

  const handleSaveSupplier = () => {
    if (!newSupName.trim()) {
      toast.error(t('einkauf.validation.supplierNameRequired'))
      return
    }
    createSupplierMutation.mutate(
      {
        name: newSupName.trim(),
        email: newSupEmail.trim() || undefined,
        phone: newSupPhone.trim() || undefined,
        payment_terms: newSupPayment || undefined,
        notes: newSupContact.trim() || undefined,
      },
      {
        onSuccess: () => {
          toast.success(t('einkauf.toast.supplierCreated', { name: newSupName }))
          resetNewSupplierForm()
          setShowNewSupplierDialog(false)
        },
        onError: () => {
          toast.error(t('einkauf.toast.supplierCreateFailed'))
        },
      },
    )
  }

  // -- Warenkorb --
  const addToCart = (item: CatalogItem) => {
    const minQty = Math.max(1, parseFloat(item.min_order_qty) || 1)
    setCart((prev) => ({
      ...prev,
      [item.id]: { item, qty: (prev[item.id]?.qty ?? 0) + minQty },
    }))
    toast.success(t('einkauf.toast.addedToCart', { name: item.name }))
  }

  const updateCartQty = (itemId: string, qty: number) => {
    setCart((prev) => (prev[itemId] ? { ...prev, [itemId]: { ...prev[itemId], qty } } : prev))
  }

  const removeFromCart = (itemId: string) => {
    setCart((prev) => {
      const next = { ...prev }
      delete next[itemId]
      return next
    })
  }

  const handleCartOrdersCreated = () => {
    setCart({})
    setShowCartDialog(false)
    setTab('bestellungen')
    setStatusFilter('all')
  }

  // -- CSV exports --
  const handleExportOrdersCsv = () => {
    const statusLabelMap: Record<string, string> = {}
    for (const key of Object.keys(orderStatusLabels)) {
      statusLabelMap[key] = t(orderStatusLabels[key])
    }
    const csv = buildPOsCsv(sortedOrders, supplierNameById, statusLabelMap)
    downloadBlob(
      new Blob(['﻿', csv], { type: 'text/csv;charset=utf-8' }),
      `bestellungen-${csvDateStamp()}.csv`,
    )
    toast.success(t('einkauf.toast.csvExported', { count: sortedOrders.length }))
  }

  const handleExportSuppliersCsv = () => {
    const csv = buildSuppliersCsv(sortedSuppliers, orderCountBySupplier)
    downloadBlob(
      new Blob(['﻿', csv], { type: 'text/csv;charset=utf-8' }),
      `lieferanten-${csvDateStamp()}.csv`,
    )
    toast.success(t('einkauf.toast.csvExported', { count: sortedSuppliers.length }))
  }

  // ---------------------------------------------------------------------------
  // JSX
  // ---------------------------------------------------------------------------

  const wareneingangSupplierName = wareneingangOrder
    ? supplierNameById.get(wareneingangOrder.supplier_id) ?? wareneingangOrder.supplier_id
    : ''

  // moduleEmpty: custom role with module:view but no read grant on any tab
  if (capReady && visibleTabs.length === 0) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <PageHeader
          title={t('einkauf.page.title')}
          description={t('rbac.gate.moduleEmpty')}
          icon={ShoppingCart}
          moduleId="einkauf"
          className="mb-6"
          actions={<RestrictedModeBadge module="einkauf" />}
        />
        <EmptyState icon={ShoppingCart} title={t('rbac.gate.moduleEmpty')} description={t('rbac.gate.noPermission')} />
      </div>
    )
  }

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
            <RestrictedModeBadge module="einkauf" />
            {tab === 'lieferanten' && canCreateSupplier && (
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
            {cartCount > 0 && canCreatePO && (
              <button
                onClick={() => setShowCartDialog(true)}
                className="relative flex items-center gap-2 rounded-xl border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <ShoppingCart className="h-4 w-4" />
                {t('einkauf.action.openCart')}
                <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
                  {cartCount}
                </span>
              </button>
            )}
            {canCreatePO && (
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
            )}
          </div>
        }
      />

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'bestellungen' as const, label: t('einkauf.tab.bestellungen', { count: purchaseOrders.length }) },
          { key: 'lieferanten' as const, label: t('einkauf.tab.lieferanten', { count: suppliers.length }) },
          { key: 'katalog' as const, label: t('einkauf.tab.katalog', { count: catalogData?.total ?? catalogItems.length }) },
          { key: 'rahmenverträge' as const, label: t('einkauf.tab.rahmenvertraege', { count: contractsData?.total ?? frameworkContracts.length }) },
        ]).filter((tabItem) => visibleTabs.includes(tabItem.key)).map((tabItem) => (
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

      {/* Search + status filter + sort + export (bestellungen / lieferanten) */}
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
          <div className="ml-auto flex items-center gap-2">
            {tab === 'bestellungen' ? (
              <SortMenu
                options={[
                  { value: 'created', label: t('einkauf.sort.created') },
                  { value: 'number', label: t('einkauf.sort.number') },
                  { value: 'supplier', label: t('einkauf.sort.supplier') },
                  { value: 'total', label: t('einkauf.sort.total') },
                  { value: 'delivery', label: t('einkauf.sort.delivery') },
                ]}
                field={orderSort.field}
                direction={orderSort.direction}
                onChange={(field, direction) => setOrderSort({ field, direction })}
              />
            ) : (
              <SortMenu
                options={[
                  { value: 'name', label: t('einkauf.sort.name') },
                  { value: 'orders', label: t('einkauf.sort.orderCount') },
                ]}
                field={supplierSort.field}
                direction={supplierSort.direction}
                onChange={(field, direction) => setSupplierSort({ field, direction })}
              />
            )}
            {canExport && (
              <button
                onClick={tab === 'bestellungen' ? handleExportOrdersCsv : handleExportSuppliersCsv}
                className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
                title={t('einkauf.action.exportCsv')}
              >
                <FileDown className="h-4 w-4" />
                <span className="hidden md:inline">{t('einkauf.action.exportCsv')}</span>
              </button>
            )}
          </div>
        </div>
      )}

      {/* ====================== BESTELLUNGEN TAB ====================== */}
      {tab === 'bestellungen' && (
        <>
          {posLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-6 w-6 border-2 border-primary border-t-transparent" />
            </div>
          ) : sortedOrders.length === 0 ? (
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
                  {sortedOrders.map((order) => (
                    <tr
                      key={order.id}
                      role="button"
                      tabIndex={0}
                      onClick={() => openOrder(order)}
                      onKeyDown={rowKeyHandler(() => openOrder(order))}
                      className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                    >
                      <td className="px-4 py-3 font-medium text-foreground font-mono text-xs">
                        {order.po_number}
                      </td>
                      <td className="px-4 py-3 text-foreground">
                        {supplierNameById.get(order.supplier_id) ?? order.supplier_id}
                      </td>
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
                        {order.expected_delivery_date
                          ? formatDate(order.expected_delivery_date + (order.expected_delivery_date.includes('T') ? '' : 'T00:00:00'))
                          : '—'}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {formatDate(order.created_at)}
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
          {suppliersLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-6 w-6 border-2 border-primary border-t-transparent" />
            </div>
          ) : sortedSuppliers.length === 0 ? (
            <EmptyState
              icon={Truck}
              title={t('einkauf.empty.noSuppliers.title')}
              description={search ? t('einkauf.empty.noSuppliers.descriptionFilter') : t('einkauf.empty.noSuppliers.descriptionEmpty')}
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {sortedSuppliers.map((supplier) => {
                const orderCount = orderCountBySupplier.get(supplier.id) ?? 0
                return (
                  <div
                    key={supplier.id}
                    role="button"
                    tabIndex={0}
                    onClick={() => openSupplier(supplier)}
                    onKeyDown={rowKeyHandler(() => openSupplier(supplier))}
                    className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
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
                              onClick: () => openSupplier(supplier),
                            },
                            ...(canEditSupplier
                              ? [{ label: t('einkauf.action.edit'), icon: Edit, onClick: () => setEditSupplier(supplier) }]
                              : []),
                            ...(canDeactivateSupplier
                              ? [
                                  { separator: true as const, label: '', onClick: () => {} },
                                  {
                                    label: t('einkauf.action.deactivate'),
                                    variant: 'destructive' as const,
                                    onClick: () => setConfirmDeactivate(supplier),
                                  },
                                ]
                              : []),
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
                        {t('einkauf.supplier.orderCountLabel', { count: orderCount })}
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
          ) : catalogItems.length === 0 ? (
            <EmptyState
              icon={BookOpen}
              title={t('einkauf.empty.noCatalog.title')}
              description={t('einkauf.empty.noCatalog.description')}
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {catalogItems.map((item) => (
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
                      <span>{supplierNameById.get(item.supplier_id) ?? item.supplier_id}</span>
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
                    {canCreatePO && (
                      <button
                        onClick={() => addToCart(item)}
                        disabled={!item.available}
                        className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        <ShoppingCart className="h-3.5 w-3.5" />
                        {t('einkauf.action.addToCart')}
                      </button>
                    )}
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
                            <span>{supplierNameById.get(contract.supplier_id) ?? contract.supplier_id}</span>
                            <span>
                              {contract.start_date ? formatDate(contract.start_date + 'T00:00:00') : '—'}
                              {' – '}
                              {contract.end_date ? formatDate(contract.end_date + 'T00:00:00') : '—'}
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

                        {/* Bisherige Abrufe */}
                        <ContractCallsList contractId={contract.id} currency={contract.currency} />

                        {canCallContract && (
                          <div className="flex items-center justify-end px-4 py-3 border-t border-border-muted">
                            <button
                              onClick={() => setCallContract(contract)}
                              disabled={contract.status !== 'active'}
                              title={contract.status !== 'active' ? t('einkauf.contract.notActiveHint') : undefined}
                              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                              <Plus className="h-3.5 w-3.5" />
                              {t('einkauf.action.newCall')}
                            </button>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </>
      )}

      {/* ====================== DETAIL MODALS (Back-Kette) ====================== */}
      <OrderDetailModal
        order={selectedOrder}
        suppliers={suppliers}
        onClose={() => {
          setSelectedOrder(null)
          setOrderBackTarget(null)
        }}
        onBack={
          orderBackTarget
            ? () => {
                const target = orderBackTarget
                setSelectedOrder(null)
                setOrderBackTarget(null)
                setSelectedSupplier(target)
              }
            : undefined
        }
        onOpenSupplier={openSupplierFromOrder}
        onBookReceipt={(order) => setWareneingangOrder(order)}
        onEdit={(order) => setEditOrder(order)}
        onCancel={(order) => setConfirmCancel(order)}
      />

      <SupplierDetailModal
        supplier={selectedSupplier}
        orders={selectedSupplier ? getSupplierOrders(selectedSupplier.id) : []}
        onClose={() => {
          setSelectedSupplier(null)
          setSupplierBackTarget(null)
        }}
        onBack={
          supplierBackTarget
            ? () => {
                const target = supplierBackTarget
                setSelectedSupplier(null)
                setSupplierBackTarget(null)
                setSelectedOrder(target)
              }
            : undefined
        }
        onOpenOrder={openOrderFromSupplier}
        onEdit={(supplier) => setEditSupplier(supplier)}
        onDeactivate={(supplier) => setConfirmDeactivate(supplier)}
      />

      {/* ====================== ACTION DIALOGS ====================== */}
      <EditOrderDialog
        key={editOrder ? `edit-po-${editOrder.id}` : 'edit-po-closed'}
        order={editOrder}
        onClose={() => setEditOrder(null)}
      />

      <EditSupplierDialog
        key={editSupplier ? `edit-sup-${editSupplier.id}` : 'edit-sup-closed'}
        supplier={editSupplier}
        onClose={() => setEditSupplier(null)}
      />

      <CartDialog
        open={showCartDialog}
        entries={cartEntries}
        suppliers={suppliers}
        poCount={purchaseOrders.length}
        onClose={() => setShowCartDialog(false)}
        onUpdateQty={updateCartQty}
        onRemove={removeFromCart}
        onCreated={handleCartOrdersCreated}
      />

      <NewCallDialog
        key={callContract ? `call-${callContract.id}` : 'call-closed'}
        contract={callContract}
        onClose={() => setCallContract(null)}
      />

      <WareneingangDialog
        key={wareneingangOrder ? `we-${wareneingangOrder.id}` : 'we-closed'}
        order={wareneingangOrder}
        supplierName={wareneingangSupplierName}
        onClose={() => setWareneingangOrder(null)}
      />

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

              {/* Approval notice in new order dialog */}
              {approvalThreshold > 0 && newOrderTotal > approvalThreshold && (
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
                  {(PAYMENT_TERMS_OPTIONS.includes(newSupPayment) ? PAYMENT_TERMS_OPTIONS : [newSupPayment, ...PAYMENT_TERMS_OPTIONS]).map((pt) => (
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

      {/* ====================== CONFIRM CANCEL / DELETE ORDER ====================== */}
      <ConfirmDialog
        open={!!confirmCancel}
        onOpenChange={() => setConfirmCancel(null)}
        title={
          confirmCancel?.status === 'draft'
            ? t('einkauf.confirmDeleteDraft.title')
            : t('einkauf.confirmCancel.title')
        }
        description={
          confirmCancel?.status === 'draft'
            ? t('einkauf.confirmDeleteDraft.description', { orderNumber: confirmCancel?.po_number })
            : t('einkauf.confirmCancel.description', {
                orderNumber: confirmCancel?.po_number,
                supplierName: confirmCancel
                  ? supplierNameById.get(confirmCancel.supplier_id) ?? confirmCancel.supplier_id
                  : '',
              })
        }
        confirmLabel={
          confirmCancel?.status === 'draft'
            ? t('einkauf.confirmDeleteDraft.confirm')
            : t('einkauf.confirmCancel.confirm')
        }
        variant="destructive"
        onConfirm={() => confirmCancel && handleCancelOrder(confirmCancel)}
      />

      {/* ====================== CONFIRM DEACTIVATE SUPPLIER ====================== */}
      <ConfirmDialog
        open={!!confirmDeactivate}
        onOpenChange={() => setConfirmDeactivate(null)}
        title={t('einkauf.confirmDeactivate.title')}
        description={t('einkauf.confirmDeactivate.description', { name: confirmDeactivate?.name })}
        confirmLabel={t('einkauf.confirmDeactivate.confirm')}
        variant="destructive"
        onConfirm={() => confirmDeactivate && handleDeactivateSupplier(confirmDeactivate)}
      />
    </div>
  )
}
