import { useState, useMemo, useCallback, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Plus,
  Package,
  MapPin,
  ArrowRightLeft,
  Warehouse,
  Edit,
  Trash2,
  Eye,
  ArrowDownToLine,
  ArrowUpFromLine,
  RefreshCw,
  ClipboardEdit,
  ChevronDown,
  Filter,
  ScanBarcode,
  ShoppingCart,
  ClipboardCheck,
  Download,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { Skeleton } from '@/components/ui/skeleton'
import {
  useInventarItems, useInventarMovements,
  useInventarLocations, useInventurSessions,
  useCreateInventarItem, useUpdateInventarItem, useDeleteInventarItem,
  useAdjustStock, useRecordMovement, useBookInventurDifferences,
  useUpsertInventurCount, useUpdateInventurSessionStatus,
} from '@/api/hooks/useInventar'
import type { InventarItem, InventoryLocation, InventurSession, InventurCount } from '@/api/inventar-types'
import { ItemActions, ConfirmDialog, EmptyState, PageHeader, SortMenu, type SortDirection, type SortFieldOption } from '@/components/shared'
import { useCapabilitySet, useHasCapability } from '@/hooks/useCapability'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'
import { formatCurrency, formatDate, formatDateTime } from '@/lib/format'
import { useInventarPrefsStore } from '@/stores/inventarPrefs'
import { useInventarTenantStore, INVENTAR_UNIT_OPTIONS } from '@/stores/inventarTenant'
import {
  getStockStatus,
  getStockStatusDisplay,
  movementTypeKeys,
  movementTypeColors,
  movementTypeIcons,
  locationTypeKeys,
  locationTypeIcons,
  inventurStatusKeys,
  inventurStatusColors,
} from './inventar-shared'
import { ItemDetailModal } from './ItemDetailModal'
import { LocationDetailModal } from './LocationDetailModal'
import { NewInventurDialog } from './NewInventurDialog'
import { buildItemsCsv, buildMovementsCsv, buildInventurCsv, downloadCsv, csvDateStamp } from './inventar-export'

type TabKey = 'artikel' | 'lagerorte' | 'bewegungen' | 'inventur'
type StatusFilter = 'all' | 'ok' | 'warning' | 'critical'
type MovementTypeLocal = 'in' | 'out' | 'transfer' | 'adjustment'

const CURRENCY_OPTIONS = ['EUR', 'CHF', 'USD']

const BARCODE_FORMAT_LABEL_KEYS: Record<string, string> = {
  ean13: 'inventar.settings.tenant.barcode.ean13',
  code128: 'inventar.settings.tenant.barcode.code128',
  qr: 'inventar.settings.tenant.barcode.qr',
}

// ─── Barcode Scanner Dialog ─────────────────────────────────────
function BarcodeScannerDialog({
  open,
  onClose,
  onScan,
}: {
  open: boolean
  onClose: () => void
  onScan: (barcode: string) => void
}) {
  const { t } = useTranslation()
  const [barcode, setBarcode] = useState('')
  const barcodeFormat = useInventarTenantStore((s) => s.barcodeFormat)

  const handleSubmit = () => {
    const trimmed = barcode.trim()
    if (!trimmed) {
      toast.error(t('inventar.barcode.errorEmpty'))
      return
    }
    onScan(trimmed)
    setBarcode('')
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleSubmit()
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-sm">
        <div className="p-6">
        <DialogHeader className="mb-4">
          <div className="flex items-center gap-2">
            <ScanBarcode className="h-5 w-5 text-primary" />
            <DialogTitle className="text-lg font-semibold text-foreground">{t('inventar.barcode.title')}</DialogTitle>
          </div>
          <DialogDescription className="sr-only">{t('inventar.barcode.title')}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <input
            type="text"
            value={barcode}
            onChange={(e) => setBarcode(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t('inventar.barcode.placeholder')}
            autoFocus
            className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring font-mono"
          />
          <p className="text-xs text-muted-foreground">
            {t('inventar.barcode.hint')}{' '}
            {t('inventar.barcode.formatHint', { format: t(BARCODE_FORMAT_LABEL_KEYS[barcodeFormat]) })}
          </p>
        </div>

        <DialogFooter className="mt-5">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSubmit}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            {t('common.search')}
          </button>
        </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ─── Artikel Dialog ───────────────────────────────────────────────
interface ArtikelFormData {
  name: string
  sku: string
  category: string
  minStock: number
  unit: string
  currency: string
  batchNumber: string
  serialNumbers: string
}

function ArtikelDialog({
  open,
  onClose,
  initial,
  onSuccess,
}: {
  open: boolean
  onClose: () => void
  initial?: InventarItem | null
  onSuccess?: () => void
}) {
  const { t } = useTranslation()
  // Tenant defaults (unit / min stock) seed new items — see inventarTenant store.
  const defaultUnit = useInventarTenantStore((s) => s.defaultUnit)
  const defaultMinStock = useInventarTenantStore((s) => s.defaultMinStock)
  const [form, setForm] = useState<ArtikelFormData>({
    name: initial?.name ?? '',
    sku: initial?.sku ?? '',
    category: initial?.category ?? '',
    minStock: Number(initial?.min_quantity ?? defaultMinStock),
    unit: initial?.unit ?? defaultUnit,
    currency: initial?.currency ?? 'EUR',
    batchNumber: initial?.batch_number ?? '',
    serialNumbers: initial?.serial_numbers?.join(', ') ?? '',
  })

  const isEdit = !!initial

  const createMutation = useCreateInventarItem()
  const updateMutation = useUpdateInventarItem()

  const handleSave = () => {
    if (!form.name.trim()) {
      toast.error(t('inventar.artikel.errorName'))
      return
    }
    if (!form.sku.trim()) {
      toast.error(t('inventar.artikel.errorSku'))
      return
    }
    const payload = {
      name: form.name,
      sku: form.sku,
      category: form.category || undefined,
      min_quantity: form.minStock,
      unit: form.unit,
      currency: form.currency,
      batch_number: form.batchNumber || undefined,
      serial_numbers: form.serialNumbers ? form.serialNumbers.split(',').map(s => s.trim()).filter(Boolean) : [],
    }
    if (isEdit && initial) {
      updateMutation.mutate({ id: initial.id, ...payload }, {
        onSuccess: () => {
          toast.success(t('inventar.artikel.saveSuccess', { name: form.name }))
          onSuccess?.()
          onClose()
        },
        onError: () => toast.error(t('common.error')),
      })
    } else {
      createMutation.mutate({ ...payload, unit: form.unit }, {
        onSuccess: () => {
          toast.success(t('inventar.artikel.createSuccess', { name: form.name }))
          onSuccess?.()
          onClose()
        },
        onError: () => toast.error(t('common.error')),
      })
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-md max-h-[90vh] overflow-y-auto">
        <div className="p-6">
        <DialogHeader className="mb-5">
          <DialogTitle className="text-lg font-semibold text-foreground">
            {isEdit ? t('inventar.artikel.dialogTitleEdit') : t('inventar.artikel.dialogTitleNew')}
          </DialogTitle>
          <DialogDescription className="sr-only">{t('inventar.artikel.dialogTitleEdit')}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Name */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('inventar.artikel.labelName')} <span className="text-destructive">*</span>
            </label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder={t('inventar.artikel.placeholderName')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* SKU */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('inventar.artikel.labelSku')} <span className="text-destructive">*</span>
            </label>
            <input
              type="text"
              value={form.sku}
              onChange={(e) => setForm((f) => ({ ...f, sku: e.target.value }))}
              placeholder={t('inventar.artikel.placeholderSku')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring font-mono"
            />
          </div>

          {/* Kategorie */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('inventar.artikel.labelCategory')}</label>
            <input
              type="text"
              value={form.category}
              onChange={(e) => setForm((f) => ({ ...f, category: e.target.value }))}
              placeholder={t('inventar.artikel.placeholderCategory')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Mindestbestand + Einheit */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('inventar.artikel.labelMinStock')}</label>
              <input
                type="number"
                min={0}
                value={form.minStock}
                onChange={(e) => setForm((f) => ({ ...f, minStock: Number(e.target.value) }))}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('inventar.artikel.labelUnit')}</label>
              <select
                value={form.unit}
                onChange={(e) => setForm((f) => ({ ...f, unit: e.target.value }))}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                {INVENTAR_UNIT_OPTIONS.map((u) => (
                  <option key={u} value={u}>{u}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Waehrung */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('inventar.artikel.labelCurrency')}</label>
            <select
              value={form.currency}
              onChange={(e) => setForm((f) => ({ ...f, currency: e.target.value }))}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              {CURRENCY_OPTIONS.map((c) => (
                <option key={c} value={c}>{c}</option>
              ))}
            </select>
          </div>

          {/* Chargennummer */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('inventar.artikel.labelBatchNumber')} <span className="text-xs text-muted-foreground font-normal">({t('common.optional')})</span>
            </label>
            <input
              type="text"
              value={form.batchNumber}
              onChange={(e) => setForm((f) => ({ ...f, batchNumber: e.target.value }))}
              placeholder={t('inventar.artikel.placeholderBatch')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring font-mono"
            />
          </div>

          {/* Seriennummern */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('inventar.artikel.labelSerialNumbers')} <span className="text-xs text-muted-foreground font-normal">({t('inventar.artikel.serialNumbersHint')})</span>
            </label>
            <input
              type="text"
              value={form.serialNumbers}
              onChange={(e) => setForm((f) => ({ ...f, serialNumbers: e.target.value }))}
              placeholder={t('inventar.artikel.placeholderSerial')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring font-mono"
            />
          </div>
        </div>

        {/* Actions */}
        <DialogFooter className="mt-6">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            {isEdit ? t('common.save') : t('inventar.artikel.buttonCreate')}
          </button>
        </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ─── Bestandsbewegung Dialog ──────────────────────────────────────
function BewegungDialog({
  open,
  onClose,
  item,
  locations,
}: {
  open: boolean
  onClose: () => void
  item: InventarItem | null
  locations: { id: string; name: string }[]
}) {
  const { t } = useTranslation()
  const [typ, setTyp] = useState<MovementTypeLocal>('in')
  const [menge, setMenge] = useState<number>(1)
  const [lagerort, setLagerort] = useState(locations[0]?.name ?? '')
  const [referenz, setReferenz] = useState('')
  const [notizen, setNotizen] = useState('')
  const allowNegativeStock = useInventarTenantStore((s) => s.allowNegativeStock)
  // RBAC: corrections rewrite stock without a movement reason — own fine switch.
  const canAdjust = useHasCapability('inventar:movement:adjust')

  const recordMovementMutation = useRecordMovement()
  const adjustStockMutation = useAdjustStock()

  const handleSave = () => {
    if (!item) return
    if (menge <= 0) {
      toast.error(t('inventar.bewegung.errorMenge'))
      return
    }
    // Tenant setting: block outgoing movements below zero unless allowed.
    if (!allowNegativeStock && typ === 'out' && menge > Number(item.quantity)) {
      toast.error(t('inventar.bewegung.errorNegative', { current: item.quantity, unit: item.unit }))
      return
    }
    const label = t(movementTypeKeys[typ])
    if (typ === 'adjustment') {
      adjustStockMutation.mutate(
        { itemId: item.id, delta: menge, reason: referenz || 'Manuelle Anpassung' },
        {
          onSuccess: () => {
            toast.success(t('inventar.bewegung.success', { label, menge, unit: item.unit, name: item.name }))
            onClose()
          },
          onError: () => toast.error(t('common.error')),
        }
      )
    } else {
      recordMovementMutation.mutate(
        { itemId: item.id, movement_type: typ as 'in' | 'out' | 'transfer', quantity: menge, reason: referenz || label },
        {
          onSuccess: () => {
            toast.success(t('inventar.bewegung.success', { label, menge, unit: item.unit, name: item.name }))
            onClose()
          },
          onError: () => toast.error(t('common.error')),
        }
      )
    }
  }

  const typeOptions: { key: MovementTypeLocal; label: string; icon: typeof ArrowDownToLine }[] = [
    { key: 'in', label: t('inventar.movementType.in'), icon: ArrowDownToLine },
    { key: 'out', label: t('inventar.movementType.out'), icon: ArrowUpFromLine },
    { key: 'transfer', label: t('inventar.movementType.transfer'), icon: RefreshCw },
    ...(canAdjust ? [{ key: 'adjustment' as const, label: t('inventar.movementType.adjustment'), icon: ClipboardEdit }] : []),
  ]

  return (
    <Dialog open={open && !!item} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-md">
        <div className="p-6">
        <DialogHeader className="mb-5">
          <DialogTitle className="text-lg font-semibold text-foreground">{t('inventar.bewegung.dialogTitle')}</DialogTitle>
          <DialogDescription className="text-xs text-muted-foreground mt-0.5">{item?.name} ({item?.sku})</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Typ radio */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('inventar.bewegung.labelTyp')}</label>
            <div className="grid grid-cols-2 gap-2">
              {typeOptions.map((opt) => {
                const Icon = opt.icon
                const isActive = typ === opt.key
                return (
                  <button
                    key={opt.key}
                    onClick={() => setTyp(opt.key)}
                    className={`flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors ${
                      isActive
                        ? 'border-primary bg-primary-light text-primary font-medium'
                        : 'border-border text-muted-foreground hover:bg-secondary'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    {opt.label}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Menge */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('inventar.bewegung.labelMenge', { unit: item?.unit ?? 'Stk' })}</label>
            <input
              type="number"
              min={1}
              value={menge}
              onChange={(e) => setMenge(Number(e.target.value))}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
            />
          </div>

          {/* Lagerort */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('inventar.bewegung.labelLagerort')}</label>
            <select
              value={lagerort}
              onChange={(e) => setLagerort(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              {locations.map((loc) => (
                <option key={loc.id} value={loc.name}>{loc.name}</option>
              ))}
            </select>
          </div>

          {/* Referenz */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('inventar.bewegung.labelReferenz')} <span className="text-xs text-muted-foreground font-normal">({t('common.optional')})</span></label>
            <input
              type="text"
              value={referenz}
              onChange={(e) => setReferenz(e.target.value)}
              placeholder={t('inventar.bewegung.placeholderReferenz')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Notizen */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('inventar.bewegung.labelNotizen')} <span className="text-xs text-muted-foreground font-normal">({t('common.optional')})</span></label>
            <textarea
              value={notizen}
              onChange={(e) => setNotizen(e.target.value)}
              placeholder={t('inventar.bewegung.placeholderNotizen')}
              rows={3}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
            />
          </div>
        </div>

        {/* Actions */}
        <DialogFooter className="mt-6">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            {t('inventar.bewegung.buttonErfassen')}
          </button>
        </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ─── Inventur Count Input ─────────────────────────────────────────
// Editable "Ist" cell while a session is open/counting. Commits on blur/Enter.
function CountInput({
  count,
  onCommit,
}: {
  count: InventurCount
  onCommit: (counted: number) => void
}) {
  const { t } = useTranslation()
  const [value, setValue] = useState(count.counted === null ? '' : String(count.counted))

  const commit = () => {
    if (value === '') return
    const n = Number(value)
    if (!Number.isFinite(n) || n < 0) return
    if (count.counted !== null && Number(count.counted) === n) return
    onCommit(n)
  }

  return (
    <input
      type="number"
      min={0}
      value={value}
      placeholder={t('inventar.inventur.countPlaceholder')}
      onChange={(e) => setValue(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => { if (e.key === 'Enter') (e.target as HTMLInputElement).blur() }}
      onClick={(e) => e.stopPropagation()}
      className="w-20 rounded-md border border-border bg-card px-2 py-1 text-right text-sm text-foreground tabular-nums placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
    />
  )
}

// ─── Inventur Session Card ────────────────────────────────────────
function InventurSessionCard({ session, allItems }: { session: InventurSession; allItems: InventarItem[] }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)

  const bookMutation = useBookInventurDifferences()
  const upsertMutation = useUpsertInventurCount()
  const statusMutation = useUpdateInventurSessionStatus()

  // RBAC: counting is warehouse work, booking differences mutates stock.
  const canCount = useHasCapability('inventar:inventur:count')
  const canBook = useHasCapability('inventar:inventur:book')
  const canExport = useHasCapability('inventar:export:run')

  const totalItems = session.counts.length
  // protojson serializes int64 as a JSON string (proto3 spec); coerce before comparing/subtracting.
  const itemsWithDiff = session.counts.filter((c) => c.counted !== null && Number(c.counted) !== Number(c.expected)).length
  const totalDiff = session.counts.reduce((sum, c) => {
    if (c.counted === null) return sum
    return sum + (Number(c.counted) - Number(c.expected))
  }, 0)
  const countedItems = session.counts.filter((c) => c.counted !== null).length
  const uncountedItems = totalItems - countedItems
  const isCounting = session.status === 'open' || session.status === 'counting'

  const itemsById = useMemo(() => new Map(allItems.map((i) => [i.id, i])), [allItems])

  const handleCommitCount = (count: InventurCount, counted: number) => {
    upsertMutation.mutate(
      { sessionId: session.id, item_id: count.item_id, counted },
      {
        onError: () => toast.error(t('common.error')),
      },
    )
    // Market lifecycle (Zoho/weclapp): first recorded count moves open → counting.
    if (session.status === 'open') {
      statusMutation.mutate({ id: session.id, status: 'counting' })
    }
  }

  const handleFinishCounting = () => {
    statusMutation.mutate(
      { id: session.id, status: 'review' },
      {
        onSuccess: () => toast.success(t('inventar.inventur.finishCountingSuccess', { name: session.name })),
        onError: () => toast.error(t('common.error')),
      },
    )
  }

  const handleExportList = () => {
    downloadCsv(
      buildInventurCsv(session, itemsById),
      `inventur-zaehlliste-${session.date}-${csvDateStamp()}.csv`,
    )
    toast.success(t('inventar.inventur.exportListSuccess', { name: session.name }))
  }

  return (
    <div className="rounded-lg border border-border bg-card overflow-hidden">
      {/* Header */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between p-4 hover:bg-secondary/50 transition-colors text-left"
      >
        <div className="flex items-center gap-3 min-w-0">
          <ClipboardCheck className="h-5 w-5 text-primary shrink-0" />
          <div className="min-w-0">
            <h4 className="text-sm font-medium text-foreground truncate">{session.name}</h4>
            <p className="text-xs text-muted-foreground">
              {formatDate(session.date)} · {session.created_by ?? ''}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0 ml-3">
          <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${inventurStatusColors[session.status]}`}>
            {t(inventurStatusKeys[session.status])}
          </span>
          <ChevronDown className={`h-4 w-4 text-muted-foreground transition-transform ${expanded ? 'rotate-180' : ''}`} />
        </div>
      </button>

      {/* Expanded content */}
      {expanded && (
        <div className="border-t border-border">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-secondary/30">
                  <th className="px-4 py-2 text-left font-medium text-muted-foreground">{t('inventar.inventur.table.artikel')}</th>
                  <th className="px-4 py-2 text-left font-medium text-muted-foreground">SKU</th>
                  <th className="px-4 py-2 text-right font-medium text-muted-foreground">{t('inventar.inventur.table.soll')}</th>
                  <th className="px-4 py-2 text-right font-medium text-muted-foreground">{t('inventar.inventur.table.ist')}</th>
                  <th className="px-4 py-2 text-right font-medium text-muted-foreground">{t('inventar.inventur.table.differenz')}</th>
                </tr>
              </thead>
              <tbody>
                {session.counts.map((count: InventurCount) => {
                  const diff = count.counted !== null ? Number(count.counted) - Number(count.expected) : null
                  const diffColor = diff === null
                    ? 'text-muted-foreground'
                    : diff === 0
                      ? 'text-success'
                      : diff < 0
                        ? 'text-error'
                        : 'text-info'
                  const matchedItem = itemsById.get(count.item_id)
                  return (
                    <tr key={count.item_id} className="border-b border-border-muted last:border-0">
                      <td className="px-4 py-2 text-foreground">{matchedItem?.name ?? count.item_id}</td>
                      <td className="px-4 py-2 text-muted-foreground font-mono text-xs">{matchedItem?.sku ?? '—'}</td>
                      <td className="px-4 py-2 text-right text-muted-foreground tabular-nums">{count.expected}</td>
                      <td className="px-4 py-2 text-right text-foreground tabular-nums">
                        {isCounting && canCount ? (
                          <CountInput
                            key={`${count.item_id}-${count.counted ?? 'null'}`}
                            count={count}
                            onCommit={(counted) => handleCommitCount(count, counted)}
                          />
                        ) : (
                          count.counted !== null ? count.counted : '—'
                        )}
                      </td>
                      <td className={`px-4 py-2 text-right font-medium tabular-nums ${diffColor}`}>
                        {diff !== null ? (diff > 0 ? `+${diff}` : diff === 0 ? '0' : diff) : '—'}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
              {/* Summary row */}
              <tfoot>
                <tr className="border-t border-border bg-secondary/30">
                  <td colSpan={2} className="px-4 py-2 text-sm font-medium text-foreground">
                    {t('inventar.inventur.table.total', { count: totalItems })}
                  </td>
                  <td className="px-4 py-2 text-right text-xs text-muted-foreground">
                    {itemsWithDiff > 0 ? t('inventar.inventur.table.withDiff', { count: itemsWithDiff }) : t('inventar.inventur.table.noDiff')}
                  </td>
                  <td className="px-4 py-2" />
                  <td className={`px-4 py-2 text-right font-medium tabular-nums ${
                    totalDiff === 0 ? 'text-success' : totalDiff < 0 ? 'text-error' : 'text-info'
                  }`}>
                    {totalDiff > 0 ? `+${totalDiff}` : totalDiff}
                  </td>
                </tr>
              </tfoot>
            </table>
          </div>

          {/* Card actions */}
          <div className="flex items-center justify-between gap-3 p-3 border-t border-border">
            <div className="flex items-center gap-3 min-w-0">
              {canExport && (
                <button
                  onClick={handleExportList}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                >
                  <Download className="h-3.5 w-3.5" />
                  {t('inventar.inventur.exportList')}
                </button>
              )}
              {isCounting && uncountedItems > 0 && (
                <span className="truncate text-xs text-muted-foreground">
                  {t('inventar.inventur.uncountedHint', { count: uncountedItems })}
                </span>
              )}
            </div>
            <div className="flex items-center gap-2 shrink-0">
              {isCounting && canCount && (
                <button
                  onClick={handleFinishCounting}
                  disabled={countedItems === 0 || statusMutation.isPending}
                  className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
                >
                  <ClipboardCheck className="h-4 w-4" />
                  {t('inventar.inventur.finishCounting')}
                </button>
              )}
              {session.status === 'review' && canBook && (
                <button
                  onClick={() => bookMutation.mutate({ sessionId: session.id }, {
                    onSuccess: () => toast.success(t('inventar.inventur.bookDifferencesSuccess', { name: session.name }))
                  })}
                  className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
                >
                  <ClipboardCheck className="h-4 w-4" />
                  {t('inventar.inventur.bookDifferences')}
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// ─── Main Page ────────────────────────────────────────────────────
export default function InventarPage() {
  const { t } = useTranslation()

  const itemsQuery = useInventarItems({ page: 1, page_size: 200 })
  const locationsQuery = useInventarLocations({ page: 1, page_size: 100 })
  const inventurSessionsQuery = useInventurSessions()

  const items: InventarItem[] = itemsQuery.data?.items ?? []
  const locations: InventoryLocation[] = locationsQuery.data?.locations ?? []
  const inventurSessions: InventurSession[] = inventurSessionsQuery.data?.sessions ?? []

  // Personal prefs (settings panel): default tab, table density, warning display.
  const density = useInventarPrefsStore((s) => s.density)
  const showMinStockWarnings = useInventarPrefsStore((s) => s.showMinStockWarnings)

  // RBAC R-3 capability checks (default hidden per gating convention)
  const { has: capHas, ready: capReady } = useCapabilitySet()
  const canCreateItem = useHasCapability('inventar:item:create')
  const canEditItem = useHasCapability('inventar:item:edit')
  const canDeleteItem = useHasCapability('inventar:item:delete')
  const canRecordMovement = useHasCapability('inventar:movement:create')
  const canCreateInventur = useHasCapability('inventar:inventur:create')
  const canExport = useHasCapability('inventar:export:run')

  const [tab, setTab] = useState<TabKey>(() => useInventarPrefsStore.getState().defaultTab)
  const [search, setSearch] = useState('')
  const [categoryFilter, setCategoryFilter] = useState('all')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [sortField, setSortField] = useState('name')
  const [sortDir, setSortDir] = useState<SortDirection>('asc')
  const [confirmDelete, setConfirmDelete] = useState<InventarItem | null>(null)

  // Detail modals (item / location) + back-chain (location → item → back)
  const [selectedItem, setSelectedItem] = useState<InventarItem | null>(null)
  const [selectedLocation, setSelectedLocation] = useState<InventoryLocation | null>(null)
  const [itemBackTarget, setItemBackTarget] = useState<InventoryLocation | null>(null)

  // Movements tab has its own subject (independent of the detail modal).
  const [movementsItemId, setMovementsItemId] = useState<string>('')

  // Dialogs
  const [artikelDialogOpen, setArtikelDialogOpen] = useState(false)
  const [editItem, setEditItem] = useState<InventarItem | null>(null)
  const [bewegungDialogOpen, setBewegungDialogOpen] = useState(false)
  const [bewegungItem, setBewegungItem] = useState<InventarItem | null>(null)
  const [showBarcodeScanner, setShowBarcodeScanner] = useState(false)
  const [inventurDialogOpen, setInventurDialogOpen] = useState(false)

  const movementsItem = items.find((i) => i.id === movementsItemId) ?? null
  const movementsQuery = useInventarMovements(movementsItemId, { page: 1, page_size: 50 })

  const deleteItemMutation = useDeleteInventarItem()

  // RBAC tab gating (tab = read-key rule); while permissions load keep all
  // tabs visible to avoid flicker (finance pattern).
  const TAB_CAPABILITY: Record<TabKey, string> = {
    artikel: 'inventar:item:read',
    lagerorte: 'inventar:location:read',
    bewegungen: 'inventar:movement:read',
    inventur: 'inventar:inventur:read',
  }
  const visibleTabs = (Object.keys(TAB_CAPABILITY) as TabKey[]).filter(
    (key) => !capReady || capHas(TAB_CAPABILITY[key]),
  )
  useEffect(() => {
    if (!capReady) return
    if (visibleTabs.length > 0 && !visibleTabs.includes(tab)) setTab(visibleTabs[0])
  }, [capReady, visibleTabs, tab])

  // Get unique categories from items
  const allCategories = useMemo(() => {
    const cats = new Set(items.map((i) => i.category).filter(Boolean) as string[])
    return ['all', ...Array.from(cats).sort()]
  }, [items])

  // Filtered + sorted items (category + status + search + SortMenu)
  const filteredItems = useMemo(() => {
    const dir = sortDir === 'asc' ? 1 : -1
    const result = [...items].sort((a, b) => {
      switch (sortField) {
        case 'stock':
          return (Number(a.quantity) - Number(b.quantity)) * dir
        case 'category':
          return (a.category ?? '').localeCompare(b.category ?? '') * dir
        case 'location':
          return (a.location ?? '').localeCompare(b.location ?? '') * dir
        case 'price':
          return ((a.price ?? 0) - (b.price ?? 0)) * dir
        default:
          return a.name.localeCompare(b.name) * dir
      }
    })

    let filtered = result
    if (categoryFilter !== 'all') {
      filtered = filtered.filter((i) => i.category === categoryFilter)
    }

    if (statusFilter !== 'all') {
      filtered = filtered.filter((i) => getStockStatus(i) === statusFilter)
    }

    if (search) {
      const q = search.toLowerCase()
      filtered = filtered.filter(
        (item) =>
          item.name.toLowerCase().includes(q) ||
          item.sku.toLowerCase().includes(q) ||
          (item.category ?? '').toLowerCase().includes(q) ||
          (item.location ?? '').toLowerCase().includes(q),
      )
    }

    return filtered
  }, [items, search, categoryFilter, statusFilter, sortField, sortDir])

  const filteredLocations = useMemo(() => {
    if (!search) return locations
    const q = search.toLowerCase()
    return locations.filter(
      (loc) => loc.name.toLowerCase().includes(q) || loc.address.toLowerCase().includes(q),
    )
  }, [locations, search])

  const lowStockCount = items.filter((i) => Number(i.quantity) <= Number(i.min_quantity)).length
  const warningCount = items.filter((i) => getStockStatus(i) === 'warning').length

  // Items at a specific location (location cards + modal)
  const getLocationItems = useCallback(
    (location: InventoryLocation) =>
      items.filter((i) => i.location_id === location.id || i.location === location.name),
    [items],
  )

  // Loading guard AFTER all hooks (rules-of-hooks: hook count must be stable)
  if (itemsQuery.isLoading) {
    return (
      <div className="flex-1 overflow-y-auto p-6 space-y-4">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  const cellPad = density === 'compact' ? 'px-4 py-2' : 'px-4 py-3'

  const openItemDetail = (item: InventarItem) => {
    setSelectedItem(item)
    setMovementsItemId(item.id)
  }

  const openArtikelDialog = (item?: InventarItem) => {
    setEditItem(item ?? null)
    setArtikelDialogOpen(true)
  }

  const openBewegungDialog = (item: InventarItem) => {
    setBewegungItem(item)
    setBewegungDialogOpen(true)
  }

  const handleBarcodeScan = (barcode: string) => {
    const found = items.find((i) => i.barcode === barcode || i.sku === barcode)
    if (found) {
      openItemDetail(found)
      setTab('artikel')
      setShowBarcodeScanner(false)
      toast.success(t('inventar.barcode.found', { name: found.name }))
    } else {
      toast.error(t('inventar.barcode.notFound'))
    }
  }

  const handleExportItems = () => {
    downloadCsv(buildItemsCsv(filteredItems), `inventar-artikel-${csvDateStamp()}.csv`)
    toast.success(t('inventar.export.itemsSuccess', { count: filteredItems.length }))
  }

  const handleExportMovements = () => {
    if (!movementsItem) return
    downloadCsv(
      buildMovementsCsv(movementsQuery.data?.movements ?? [], movementsItem.name),
      `inventar-bewegungen-${movementsItem.sku}-${csvDateStamp()}.csv`,
    )
    toast.success(t('inventar.export.movementsSuccess', { name: movementsItem.name }))
  }

  const getItemActions = (item: InventarItem) => [
    { label: t('inventar.actions.showDetails'), icon: Eye, onClick: () => openItemDetail(item) },
    ...(canEditItem ? [{ label: t('common.edit'), icon: Edit, onClick: () => openArtikelDialog(item) }] : []),
    ...(canRecordMovement
      ? [{ label: t('inventar.actions.movement'), icon: ArrowRightLeft, onClick: () => openBewegungDialog(item), separator: true }]
      : []),
    ...(canDeleteItem
      ? [
          { separator: true as const, label: '', onClick: () => {} },
          { label: t('common.delete'), icon: Trash2, variant: 'destructive' as const, onClick: () => setConfirmDelete(item) },
        ]
      : []),
  ]

  const handleDelete = (item: InventarItem) => {
    setConfirmDelete(null)
    if (selectedItem?.id === item.id) setSelectedItem(null)
    deleteItemMutation.mutate(item.id, {
      onSuccess: () => toast.success(t('inventar.delete.success', { name: item.name })),
      onError: () => toast.error(t('common.error')),
    })
  }

  const sortOptions: SortFieldOption[] = [
    { value: 'name', label: t('inventar.sort.name') },
    { value: 'stock', label: t('inventar.sort.stock') },
    { value: 'category', label: t('inventar.sort.category') },
    { value: 'location', label: t('inventar.sort.location') },
    { value: 'price', label: t('inventar.sort.price') },
  ]

  const headerWarnings = showMinStockWarnings
    ? `${lowStockCount > 0 ? ` · ${t('inventar.page.descriptionCritical', { count: lowStockCount })}` : ''}${warningCount > 0 ? ` · ${t('inventar.page.descriptionWarning', { count: warningCount })}` : ''}${lowStockCount === 0 && warningCount === 0 ? ` · ${t('inventar.page.descriptionOk')}` : ''}`
    : ''

  // Custom role with module visibility but no read grant on any tab: honest
  // hint instead of an empty tab skeleton (rbac gating convention).
  if (capReady && visibleTabs.length === 0) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <PageHeader
          title={t('inventar.page.title')}
          description={t('rbac.gate.moduleEmpty')}
          icon={Warehouse}
          moduleId="inventar"
          className="mb-6"
          actions={<RestrictedModeBadge module="inventar" />}
        />
        <EmptyState icon={Package} title={t('rbac.gate.moduleEmpty')} description={t('rbac.gate.noPermission')} />
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <PageHeader
        title={t('inventar.page.title')}
        description={`${t('inventar.page.descriptionBase', { count: items.length })}${headerWarnings}`}
        icon={Warehouse}
        moduleId="inventar"
        className="mb-6"
        actions={
          <div className="flex items-center gap-2">
            <RestrictedModeBadge module="inventar" />
            {canCreateItem && (
              <button
                onClick={() => openArtikelDialog()}
                className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Plus className="h-4 w-4" />
                {t('inventar.page.addArticle')}
              </button>
            )}
          </div>
        }
      />

      {/* Tabs with badges */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'artikel' as const, label: t('inventar.tab.artikel'), count: items.length },
          { key: 'lagerorte' as const, label: t('inventar.tab.lagerorte'), count: locations.length },
          { key: 'bewegungen' as const, label: t('inventar.tab.bewegungen'), count: movementsItemId ? (movementsQuery.data?.total ?? 0) : 0 },
          { key: 'inventur' as const, label: t('inventar.tab.inventur'), count: inventurSessions.length },
        ]).filter((tabItem) => visibleTabs.includes(tabItem.key)).map((tabItem) => (
          <button
            key={tabItem.key}
            onClick={() => { setTab(tabItem.key); setSearch('') }}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === tabItem.key ? 'border-primary text-primary font-medium tab-accent-active' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {tabItem.label} ({tabItem.count})
          </button>
        ))}
      </div>

      {/* Search + Filters */}
      <div className="flex flex-wrap items-center gap-3 mb-4">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder={
              tab === 'artikel' ? t('inventar.search.artikel')
                : tab === 'lagerorte' ? t('inventar.search.lagerorte')
                  : tab === 'bewegungen' ? t('inventar.search.bewegungen')
                    : t('inventar.search.inventur')
            }
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
        </div>

        {/* Barcode Scanner button (opens the item detail → needs item:read) */}
        {capHas('inventar:item:read') && (
          <button
            onClick={() => setShowBarcodeScanner(true)}
            className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
            title={t('inventar.search.barcodeScan')}
          >
            <ScanBarcode className="h-4 w-4" />
            <span className="hidden sm:inline">{t('inventar.search.barcodeScan')}</span>
          </button>
        )}

        {/* Category + Status filters + Sort + Export (Artikel tab only) */}
        {tab === 'artikel' && (
          <>
            <div className="relative">
              <select
                value={categoryFilter}
                onChange={(e) => setCategoryFilter(e.target.value)}
                className="appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                {allCategories.map((cat) => (
                  <option key={cat} value={cat}>
                    {cat === 'all' ? t('inventar.search.allCategories') : cat}
                  </option>
                ))}
              </select>
              <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            </div>

            <div className="relative">
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
                className="appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                <option value="all">{t('inventar.search.allStatus')}</option>
                <option value="ok">{t('inventar.status.ok')}</option>
                <option value="warning">{t('inventar.status.warning')}</option>
                <option value="critical">{t('inventar.status.critical')}</option>
              </select>
              <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            </div>

            {(categoryFilter !== 'all' || statusFilter !== 'all') && (
              <button
                onClick={() => { setCategoryFilter('all'); setStatusFilter('all') }}
                className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <Filter className="h-3.5 w-3.5" />
                {t('common.resetFilters')}
              </button>
            )}

            <SortMenu
              options={sortOptions}
              field={sortField}
              direction={sortDir}
              onChange={(field, direction) => { setSortField(field); setSortDir(direction) }}
              triggerClassName="py-2"
            />

            {canExport && (
              <button
                onClick={handleExportItems}
                disabled={filteredItems.length === 0}
                className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors disabled:opacity-50"
              >
                <Download className="h-4 w-4" />
                <span className="hidden sm:inline">{t('inventar.export.button')}</span>
              </button>
            )}
          </>
        )}

        {/* Movements export (Bewegungen tab) */}
        {tab === 'bewegungen' && canExport && movementsItem && (movementsQuery.data?.movements?.length ?? 0) > 0 && (
          <button
            onClick={handleExportMovements}
            className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          >
            <Download className="h-4 w-4" />
            <span className="hidden sm:inline">{t('inventar.export.button')}</span>
          </button>
        )}
      </div>

      {/* ─── ARTIKEL TAB ────────────────────────────────────── */}
      {tab === 'artikel' && (
        <>
          {filteredItems.length === 0 ? (
            <EmptyState
              icon={Package}
              title={t('inventar.empty.artikel.title')}
              description={
                search || categoryFilter !== 'all' || statusFilter !== 'all'
                  ? t('inventar.empty.artikel.descFilter')
                  : t('inventar.empty.artikel.descEmpty')
              }
            />
          ) : (
            <TooltipProvider delayDuration={300}>
              <div className="overflow-x-auto rounded-lg border border-border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border bg-card">
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground w-12">{t('inventar.table.status')}</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('inventar.table.name')}</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('inventar.table.sku')}</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('inventar.table.category')}</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('inventar.table.stock')}</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('inventar.table.minStock')}</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('inventar.table.location')}</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('inventar.table.price')}</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground w-12"></th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredItems.map((item) => {
                      const status = getStockStatusDisplay(item)
                      const isSelected = selectedItem?.id === item.id
                      const isCritical = getStockStatus(item) === 'critical'
                      return (
                        <tr
                          key={item.id}
                          role="button"
                          tabIndex={0}
                          aria-label={item.name}
                          onClick={() => openItemDetail(item)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' || e.key === ' ') {
                              e.preventDefault()
                              openItemDetail(item)
                            }
                          }}
                          className={`border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer focus:outline-none focus-visible:bg-secondary/50 ${
                            isSelected ? 'bg-primary-light/30' : ''
                          }`}
                        >
                          <td className={cellPad}>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className={`inline-block h-2.5 w-2.5 rounded-full ${status.dotColor}`} />
                              </TooltipTrigger>
                              <TooltipContent>
                                <p className="text-xs">{t('inventar.tooltip.stockInfo', { current: item.quantity, min: item.min_quantity })}</p>
                                <p className="text-xs font-medium">{t(status.labelKey)}</p>
                              </TooltipContent>
                            </Tooltip>
                          </td>
                          <td className={`${cellPad} font-medium text-foreground`}>{item.name}</td>
                          <td className={`${cellPad} text-muted-foreground font-mono text-xs`}>{item.sku}</td>
                          <td className={cellPad}>
                            <span className="rounded-full bg-secondary px-2 py-0.5 text-xs text-muted-foreground">{item.category ?? ''}</span>
                          </td>
                          <td className={`${cellPad} text-right text-foreground tabular-nums`}>
                            <span className="inline-flex items-center gap-1.5">
                              {isCritical && showMinStockWarnings && (
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <ShoppingCart className="h-3.5 w-3.5 text-error" />
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    <p className="text-xs">{t('inventar.tooltip.reorderRecommended')}</p>
                                  </TooltipContent>
                                </Tooltip>
                              )}
                              {item.quantity} {item.unit}
                            </span>
                          </td>
                          <td className={`${cellPad} text-right text-muted-foreground tabular-nums`}>
                            {item.min_quantity}
                          </td>
                          <td className={`${cellPad} text-muted-foreground`}>{item.location ?? '—'}</td>
                          <td className={`${cellPad} text-right text-foreground tabular-nums`}>
                            {formatCurrency(item.price ?? 0, item.currency ?? 'EUR')}
                          </td>
                          <td className={`${cellPad} text-right`} onClick={(e) => e.stopPropagation()}>
                            <ItemActions items={getItemActions(item)} />
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            </TooltipProvider>
          )}
        </>
      )}

      {/* ─── LAGERORTE TAB ──────────────────────────────────── */}
      {tab === 'lagerorte' && (
        <>
          {filteredLocations.length === 0 ? (
            <EmptyState
              icon={MapPin}
              title={t('inventar.empty.lagerorte.title')}
              description={search ? t('inventar.empty.lagerorte.descFilter') : t('inventar.empty.lagerorte.descEmpty')}
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {filteredLocations.map((loc) => {
                const Icon = locationTypeIcons[loc.type] || Warehouse
                const locItems = getLocationItems(loc)
                const criticalInLoc = locItems.filter((i) => getStockStatus(i) === 'critical').length
                return (
                  <div
                    key={loc.id}
                    role="button"
                    tabIndex={0}
                    aria-label={loc.name}
                    onClick={() => setSelectedLocation(loc)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        setSelectedLocation(loc)
                      }
                    }}
                    className="rounded-lg border border-border bg-card p-4 cursor-pointer transition-shadow hover:shadow-[var(--shadow-card-hover)] focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                          <Icon className="h-5 w-5 text-primary" />
                        </div>
                        <div>
                          <h4 className="text-sm font-medium text-foreground">{loc.name}</h4>
                          <p className="text-xs text-muted-foreground">{loc.address}</p>
                        </div>
                      </div>
                    </div>

                    {/* Items summary */}
                    {locItems.length > 0 && (
                      <div className="mb-3 space-y-1">
                        {locItems.slice(0, 3).map((li) => {
                          const st = getStockStatusDisplay(li)
                          return (
                            <div key={li.id} className="flex items-center gap-2 text-xs text-muted-foreground">
                              <span className={`h-1.5 w-1.5 rounded-full ${st.dotColor}`} />
                              <span className="truncate flex-1">{li.name}</span>
                              <span className="tabular-nums">{li.quantity} {li.unit}</span>
                            </div>
                          )
                        })}
                        {locItems.length > 3 && (
                          <p className="text-xs text-muted-foreground/60 pl-3.5">
                            {t('inventar.lagerort.moreItems', { count: locItems.length - 3 })}
                          </p>
                        )}
                      </div>
                    )}

                    <div className="flex items-center justify-between border-t border-border-muted pt-3">
                      <div className="flex items-center gap-2">
                        <span className="rounded-full bg-secondary px-2 py-0.5 text-xs text-muted-foreground">
                          {t(locationTypeKeys[loc.type])}
                        </span>
                        {criticalInLoc > 0 && showMinStockWarnings && (
                          <span className="rounded-full bg-error-light px-2 py-0.5 text-xs text-error">
                            {t('inventar.lagerort.criticalCount', { count: criticalInLoc })}
                          </span>
                        )}
                      </div>
                      <span className="text-sm text-foreground font-medium">{t('inventar.lagerort.itemCount', { count: locItems.length })}</span>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </>
      )}

      {/* ─── BEWEGUNGEN TAB ─────────────────────────────────── */}
      {tab === 'bewegungen' && (
        <>
          {/* Artikel-Auswahl — der Tab hat ein eigenes Subjekt, unabhängig vom Detail-Fenster */}
          <div className="mb-4 flex items-center gap-3">
            <label className="text-sm text-muted-foreground shrink-0">{t('inventar.bewegungen.subjectLabel')}</label>
            <div className="relative">
              <select
                value={movementsItemId}
                onChange={(e) => setMovementsItemId(e.target.value)}
                className="appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                <option value="">{t('inventar.bewegungen.subjectPlaceholder')}</option>
                {items.map((i) => (
                  <option key={i.id} value={i.id}>{i.name} ({i.sku})</option>
                ))}
              </select>
              <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            </div>
          </div>

          {!movementsItem ? (
            <EmptyState
              icon={ArrowRightLeft}
              title={t('inventar.empty.bewegungen.title')}
              description={t('inventar.empty.bewegungen.selectItem')}
            />
          ) : movementsQuery.isLoading ? (
            <div className="space-y-2">
              {[1,2,3].map(n => <Skeleton key={n} className="h-12 w-full" />)}
            </div>
          ) : (movementsQuery.data?.movements ?? []).length === 0 ? (
            <EmptyState
              icon={ArrowRightLeft}
              title={t('inventar.empty.bewegungen.title')}
              description={search ? t('inventar.empty.bewegungen.descFilter') : t('inventar.empty.bewegungen.descEmpty')}
            />
          ) : (
            <div className="overflow-x-auto rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-card">
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('inventar.bewegungen.table.datum')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('inventar.bewegungen.table.artikel')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('inventar.bewegungen.table.typ')}</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('inventar.bewegungen.table.menge')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('inventar.bewegungen.table.vonNach')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('inventar.bewegungen.table.referenz')}</th>
                  </tr>
                </thead>
                <tbody>
                  {(movementsQuery.data?.movements ?? []).map((mov) => {
                    const MIcon = movementTypeIcons[mov.movement_type]
                    return (
                      <tr key={mov.id} className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors">
                        <td className={`${cellPad} text-muted-foreground whitespace-nowrap`}>
                          {formatDate(mov.created_at)}{' '}
                          <span className="text-xs">{formatDateTime(mov.created_at, { timeStyle: 'short' })}</span>
                        </td>
                        <td className={`${cellPad} font-medium text-foreground`}>{movementsItem.name}</td>
                        <td className={cellPad}>
                          <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${movementTypeColors[mov.movement_type] ?? 'bg-secondary text-muted-foreground'}`}>
                            {MIcon && <MIcon className="h-3 w-3" />}
                            {movementTypeKeys[mov.movement_type] ? t(movementTypeKeys[mov.movement_type]) : mov.movement_type}
                          </span>
                        </td>
                        <td className={`${cellPad} text-right text-foreground tabular-nums`}>
                          {Number(mov.quantity) > 0 ? `+${mov.quantity}` : mov.quantity}
                        </td>
                        <td className={`${cellPad} text-muted-foreground`}>
                          {mov.location_from && mov.location_to
                            ? `${mov.location_from} → ${mov.location_to}`
                            : mov.location_from ?? mov.location_to ?? '—'}
                        </td>
                        <td className={`${cellPad} text-muted-foreground font-mono text-xs`}>{mov.reference ?? '—'}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* ─── INVENTUR TAB ──────────────────────────────────── */}
      {tab === 'inventur' && (
        <>
          <div className="flex items-center justify-between mb-4">
            <p className="text-sm text-muted-foreground">
              {inventurSessions.length !== 1
                ? t('inventar.inventur.sessionCountPlural', { count: inventurSessions.length })
                : t('inventar.inventur.sessionCount', { count: inventurSessions.length })}
            </p>
            {canCreateInventur && (
              <button
                onClick={() => setInventurDialogOpen(true)}
                className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Plus className="h-4 w-4" />
                {t('inventar.inventur.newSession')}
              </button>
            )}
          </div>

          {inventurSessions.length === 0 ? (
            <EmptyState
              icon={ClipboardCheck}
              title={t('inventar.empty.inventur.title')}
              description={t('inventar.empty.inventur.desc')}
            />
          ) : (
            <div className="space-y-3">
              {inventurSessions.map((session) => (
                <InventurSessionCard key={session.id} session={session} allItems={items} />
              ))}
            </div>
          )}
        </>
      )}

      {/* ─── DETAIL MODALS (Cosmi-Fenster) ──────────────────── */}
      <ItemDetailModal
        item={selectedItem}
        locations={locations}
        onClose={() => {
          setSelectedItem(null)
          setItemBackTarget(null)
        }}
        onBack={
          itemBackTarget
            ? () => {
                setSelectedItem(null)
                setSelectedLocation(itemBackTarget)
                setItemBackTarget(null)
              }
            : undefined
        }
        onEdit={(item) => openArtikelDialog(item)}
        onMovement={(item) => openBewegungDialog(item)}
      />

      <LocationDetailModal
        location={selectedLocation}
        items={selectedLocation ? getLocationItems(selectedLocation) : []}
        onClose={() => setSelectedLocation(null)}
        onItemClick={(item) => {
          setItemBackTarget(selectedLocation)
          setSelectedLocation(null)
          openItemDetail(item)
        }}
      />

      {/* ─── DIALOGS ────────────────────────────────────────── */}
      <ArtikelDialog
        key={artikelDialogOpen ? (editItem?.id ?? 'new') : 'closed'}
        open={artikelDialogOpen}
        onClose={() => {
          setArtikelDialogOpen(false)
          setEditItem(null)
        }}
        initial={editItem}
      />

      <BewegungDialog
        open={bewegungDialogOpen}
        onClose={() => {
          setBewegungDialogOpen(false)
          setBewegungItem(null)
        }}
        item={bewegungItem}
        locations={locations.map((l) => ({ id: l.id, name: l.name }))}
      />

      <BarcodeScannerDialog
        open={showBarcodeScanner}
        onClose={() => setShowBarcodeScanner(false)}
        onScan={handleBarcodeScan}
      />

      <NewInventurDialog
        open={inventurDialogOpen}
        onClose={() => setInventurDialogOpen(false)}
        items={items}
        locations={locations}
      />

      {/* Confirm Delete */}
      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title={t('inventar.confirm.deleteTitle')}
        description={t('inventar.confirm.deleteDesc', { name: confirmDelete?.name ?? '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={() => confirmDelete && handleDelete(confirmDelete)}
      />
    </div>
  )
}
