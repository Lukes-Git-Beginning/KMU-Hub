import { useState, useMemo, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Plus,
  Package,
  MapPin,
  ArrowRightLeft,
  Warehouse,
  Truck,
  Store,
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
  Hash,
  Link2,
  ClipboardCheck,
  AlertTriangle,
  Paperclip,
  X,
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
  useItemAttachments, useUploadItemAttachment, useDeleteItemAttachment,
} from '@/api/hooks/useInventar'
import { useAvatarSrc } from '@/api/hooks/useAvatarSrc'
import type { InventarItem, InventoryLocation, InventurSession, InventurCount, ItemAttachment } from '@/api/inventar-types'
import { ItemActions, ConfirmDialog, EmptyState, DetailPanel, PageHeader } from '@/components/shared'
import { formatCurrency, formatDate, formatDateTime } from '@/lib/format'

type TabKey = 'artikel' | 'lagerorte' | 'bewegungen' | 'inventur'
type StatusFilter = 'all' | 'ok' | 'warning' | 'critical'
type MovementTypeLocal = 'in' | 'out' | 'transfer' | 'adjustment'

const movementTypeKeys: Record<string, string> = {
  in: 'inventar.movementType.in',
  out: 'inventar.movementType.out',
  transfer: 'inventar.movementType.transfer',
  adjustment: 'inventar.movementType.adjustment',
}

const movementTypeColors: Record<string, string> = {
  in: 'bg-success-light text-success',
  out: 'bg-error-light text-error',
  transfer: 'bg-info-light text-info',
  adjustment: 'bg-warning-light text-warning',
}

const locationTypeKeys: Record<string, string> = {
  warehouse: 'inventar.locationType.warehouse',
  store: 'inventar.locationType.store',
  vehicle: 'inventar.locationType.vehicle',
}

const locationTypeIcons: Record<string, typeof Warehouse> = {
  warehouse: Warehouse,
  store: Store,
  vehicle: Truck,
}

const movementTypeIcons: Record<string, typeof ArrowDownToLine> = {
  in: ArrowDownToLine,
  out: ArrowUpFromLine,
  transfer: RefreshCw,
  adjustment: ClipboardEdit,
}

const inventurStatusKeys: Record<string, string> = {
  open: 'inventar.inventurStatus.open',
  counting: 'inventar.inventurStatus.counting',
  review: 'inventar.inventurStatus.review',
  completed: 'inventar.inventurStatus.completed',
}

const inventurStatusColors: Record<string, string> = {
  open: 'bg-info-light text-info',
  counting: 'bg-warning-light text-warning',
  review: 'bg-[#fff3e0] text-[#e65100] dark:bg-[#e65100]/20 dark:text-[#ffab40]',
  completed: 'bg-success-light text-success',
}

const CURRENCY_OPTIONS = ['EUR', 'CHF', 'USD']

const UNIT_OPTIONS = ['Stück', 'kg', 'Meter', 'Liter', 'Packung', 'Rolle']

function getStockStatus(item: InventarItem): 'ok' | 'warning' | 'critical' {
  if (item.quantity <= item.min_quantity) return 'critical'
  if (item.quantity < item.min_quantity * 2) return 'warning'
  return 'ok'
}

const stockStatusLabelKeys: Record<string, string> = {
  critical: 'inventar.status.critical',
  warning: 'inventar.status.warning',
  ok: 'inventar.status.ok',
}

function getStockStatusDisplay(item: InventarItem): { color: string; labelKey: string; dotColor: string } {
  const status = getStockStatus(item)
  if (status === 'critical') return { color: 'bg-error', labelKey: stockStatusLabelKeys.critical, dotColor: 'bg-error' }
  if (status === 'warning') return { color: 'bg-warning', labelKey: stockStatusLabelKeys.warning, dotColor: 'bg-warning' }
  return { color: 'bg-success', labelKey: stockStatusLabelKeys.ok, dotColor: 'bg-success' }
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
            {t('inventar.barcode.hint')}
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
  const [form, setForm] = useState<ArtikelFormData>({
    name: initial?.name ?? '',
    sku: initial?.sku ?? '',
    category: initial?.category ?? '',
    minStock: initial?.min_quantity ?? 0,
    unit: initial?.unit ?? 'Stück',
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
                {UNIT_OPTIONS.map((u) => (
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

  const recordMovementMutation = useRecordMovement()
  const adjustStockMutation = useAdjustStock()

  const handleSave = () => {
    if (!item) return
    if (menge <= 0) {
      toast.error(t('inventar.bewegung.errorMenge'))
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
    { key: 'adjustment', label: t('inventar.movementType.adjustment'), icon: ClipboardEdit },
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

// ─── Stock Bar Visual ─────────────────────────────────────────────
function StockBar({ current, min }: { current: number; min: number }) {
  const { t } = useTranslation()
  const max = Math.max(min * 3, current * 1.2, 1)
  const pct = Math.min((current / max) * 100, 100)
  const minPct = Math.min((min / max) * 100, 100)
  const status = current <= min ? 'bg-error' : current < min * 2 ? 'bg-warning' : 'bg-success'

  return (
    <div className="space-y-1">
      <div className="relative h-2 w-full rounded-full bg-secondary overflow-hidden">
        <div
          className={`absolute inset-y-0 left-0 rounded-full ${status} transition-all`}
          style={{ width: `${pct}%` }}
        />
        {/* Min threshold marker */}
        <div
          className="absolute inset-y-0 w-0.5 bg-foreground/30"
          style={{ left: `${minPct}%` }}
          title={t('inventar.stockBar.minThreshold', { min })}
        />
      </div>
      <div className="flex justify-between text-[10px] text-muted-foreground">
        <span>0</span>
        <span>{t('inventar.detail.stockBarMin', { min, unit: '' }).trim()}</span>
        <span>{Math.round(max)}</span>
      </div>
    </div>
  )
}

// ─── Inventur Session Card ────────────────────────────────────────
function InventurSessionCard({ session, allItems }: { session: InventurSession; allItems: InventarItem[] }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)

  const bookMutation = useBookInventurDifferences()

  const totalItems = session.counts.length
  const itemsWithDiff = session.counts.filter((c) => c.counted !== null && c.counted !== c.expected).length
  const totalDiff = session.counts.reduce((sum, c) => {
    if (c.counted === null) return sum
    return sum + (c.counted - c.expected)
  }, 0)

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
                  const diff = count.counted !== null ? count.counted - count.expected : null
                  const diffColor = diff === null
                    ? 'text-muted-foreground'
                    : diff === 0
                      ? 'text-success'
                      : diff < 0
                        ? 'text-error'
                        : 'text-info'
                  const matchedItem = allItems.find(i => i.id === count.item_id)
                  return (
                    <tr key={count.item_id} className="border-b border-border-muted last:border-0">
                      <td className="px-4 py-2 text-foreground">{matchedItem?.name ?? count.item_id}</td>
                      <td className="px-4 py-2 text-muted-foreground font-mono text-xs">{matchedItem?.sku ?? '—'}</td>
                      <td className="px-4 py-2 text-right text-muted-foreground tabular-nums">{count.expected}</td>
                      <td className="px-4 py-2 text-right text-foreground tabular-nums">
                        {count.counted !== null ? count.counted : '—'}
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

          {/* Actions for review status */}
          {session.status === 'review' && (
            <div className="flex justify-end p-3 border-t border-border">
              <button
                onClick={() => bookMutation.mutate({ sessionId: session.id }, {
                  onSuccess: () => toast.success(t('inventar.inventur.bookDifferencesSuccess', { name: session.name }))
                })}
                className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <ClipboardCheck className="h-4 w-4" />
                {t('inventar.inventur.bookDifferences')}
              </button>
            </div>
          )}
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

  const [tab, setTab] = useState<TabKey>('artikel')
  const [search, setSearch] = useState('')
  const [categoryFilter, setCategoryFilter] = useState('all')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [confirmDelete, setConfirmDelete] = useState<InventarItem | null>(null)

  // Detail panel
  const [selectedItem, setSelectedItem] = useState<InventarItem | null>(null)

  // Dialogs
  const [artikelDialogOpen, setArtikelDialogOpen] = useState(false)
  const [editItem, setEditItem] = useState<InventarItem | null>(null)
  const [bewegungDialogOpen, setBewegungDialogOpen] = useState(false)
  const [bewegungItem, setBewegungItem] = useState<InventarItem | null>(null)
  const [showBarcodeScanner, setShowBarcodeScanner] = useState(false)

  // selectedItem movements (conditional)
  const selectedItemMovementsQuery = useInventarMovements(selectedItem?.id ?? '', { page: 1, page_size: 5 })

  const deleteItemMutation = useDeleteInventarItem()

  // Get unique categories from items
  const allCategories = useMemo(() => {
    const cats = new Set(items.map((i) => i.category).filter(Boolean) as string[])
    return ['all', ...Array.from(cats).sort()]
  }, [items])

  // Filtered items with category + status + search
  const filteredItems = useMemo(() => {
    let result = [...items].sort((a, b) => a.name.localeCompare(b.name))

    if (categoryFilter !== 'all') {
      result = result.filter((i) => i.category === categoryFilter)
    }

    if (statusFilter !== 'all') {
      result = result.filter((i) => getStockStatus(i) === statusFilter)
    }

    if (search) {
      const q = search.toLowerCase()
      result = result.filter(
        (item) =>
          item.name.toLowerCase().includes(q) ||
          item.sku.toLowerCase().includes(q) ||
          (item.category ?? '').toLowerCase().includes(q) ||
          (item.location ?? '').toLowerCase().includes(q),
      )
    }

    return result
  }, [items, search, categoryFilter, statusFilter])

  const filteredLocations = useMemo(() => {
    if (!search) return locations
    const q = search.toLowerCase()
    return locations.filter(
      (loc) => loc.name.toLowerCase().includes(q) || loc.address.toLowerCase().includes(q),
    )
  }, [locations, search])

  const lowStockCount = items.filter((i) => i.quantity <= i.min_quantity).length
  const warningCount = items.filter((i) => getStockStatus(i) === 'warning').length

  // Items at a specific location (detail panel)
  const getLocationItems = useCallback(
    (locationName: string) => items.filter((i) => i.location === locationName),
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
      setSelectedItem(found)
      setTab('artikel')
      setShowBarcodeScanner(false)
      toast.success(t('inventar.barcode.found', { name: found.name }))
    } else {
      toast.error(t('inventar.barcode.notFound'))
    }
  }

  const getItemActions = (item: InventarItem) => [
    { label: t('inventar.actions.showDetails'), icon: Eye, onClick: () => setSelectedItem(item) },
    { label: t('common.edit'), icon: Edit, onClick: () => openArtikelDialog(item) },
    { label: t('inventar.actions.movement'), icon: ArrowRightLeft, onClick: () => openBewegungDialog(item), separator: true },
    { separator: true as const, label: '', onClick: () => {} },
    { label: t('common.delete'), icon: Trash2, variant: 'destructive' as const, onClick: () => setConfirmDelete(item) },
  ]

  const handleDelete = (item: InventarItem) => {
    setConfirmDelete(null)
    if (selectedItem?.id === item.id) setSelectedItem(null)
    deleteItemMutation.mutate(item.id, {
      onSuccess: () => toast.success(t('inventar.delete.success', { name: item.name })),
      onError: () => toast.error(t('common.error')),
    })
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <PageHeader
        title={t('inventar.page.title')}
        description={`${t('inventar.page.descriptionBase', { count: items.length })}${lowStockCount > 0 ? ` · ${t('inventar.page.descriptionCritical', { count: lowStockCount })}` : ''}${warningCount > 0 ? ` · ${t('inventar.page.descriptionWarning', { count: warningCount })}` : ''}${lowStockCount === 0 && warningCount === 0 ? ` · ${t('inventar.page.descriptionOk')}` : ''}`}
        icon={Warehouse}
        moduleId="inventar"
        className="mb-6"
        actions={
          <button
            onClick={() => openArtikelDialog()}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            {t('inventar.page.addArticle')}
          </button>
        }
      />

      {/* Tabs with badges */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'artikel' as const, label: t('inventar.tab.artikel'), count: items.length },
          { key: 'lagerorte' as const, label: t('inventar.tab.lagerorte'), count: locations.length },
          { key: 'bewegungen' as const, label: t('inventar.tab.bewegungen'), count: selectedItem ? (selectedItemMovementsQuery.data?.total ?? 0) : 0 },
          { key: 'inventur' as const, label: t('inventar.tab.inventur'), count: inventurSessions.length },
        ]).map((tabItem) => (
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

        {/* Barcode Scanner button */}
        <button
          onClick={() => setShowBarcodeScanner(true)}
          className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title={t('inventar.search.barcodeScan')}
        >
          <ScanBarcode className="h-4 w-4" />
          <span className="hidden sm:inline">{t('inventar.search.barcodeScan')}</span>
        </button>

        {/* Category + Status filters (Artikel tab only) */}
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
          </>
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
                          onClick={() => setSelectedItem(item)}
                          className={`border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer ${
                            isSelected ? 'bg-primary-light/30' : ''
                          }`}
                        >
                          <td className="px-4 py-3">
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
                          <td className="px-4 py-3 font-medium text-foreground">{item.name}</td>
                          <td className="px-4 py-3 text-muted-foreground font-mono text-xs">{item.sku}</td>
                          <td className="px-4 py-3">
                            <span className="rounded-full bg-secondary px-2 py-0.5 text-xs text-muted-foreground">{item.category ?? ''}</span>
                          </td>
                          <td className="px-4 py-3 text-right text-foreground tabular-nums">
                            <span className="inline-flex items-center gap-1.5">
                              {isCritical && (
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
                          <td className="px-4 py-3 text-right text-muted-foreground tabular-nums">
                            {item.min_quantity}
                          </td>
                          <td className="px-4 py-3 text-muted-foreground">{item.location ?? '—'}</td>
                          <td className="px-4 py-3 text-right text-foreground tabular-nums">
                            {formatCurrency(item.price ?? 0, item.currency ?? 'EUR')}
                          </td>
                          <td className="px-4 py-3 text-right" onClick={(e) => e.stopPropagation()}>
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
                const locItems = getLocationItems(loc.name)
                const criticalInLoc = locItems.filter((i) => getStockStatus(i) === 'critical').length
                return (
                  <div key={loc.id} className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)]">
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
                        {criticalInLoc > 0 && (
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
          {!selectedItem ? (
            <EmptyState
              icon={ArrowRightLeft}
              title={t('inventar.empty.bewegungen.title')}
              description={t('inventar.empty.bewegungen.selectItem')}
            />
          ) : selectedItemMovementsQuery.isLoading ? (
            <div className="space-y-2">
              {[1,2,3].map(n => <Skeleton key={n} className="h-12 w-full" />)}
            </div>
          ) : (selectedItemMovementsQuery.data?.movements ?? []).length === 0 ? (
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
                  {(selectedItemMovementsQuery.data?.movements ?? []).map((mov) => {
                    const MIcon = movementTypeIcons[mov.movement_type]
                    return (
                      <tr key={mov.id} className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors">
                        <td className="px-4 py-3 text-muted-foreground whitespace-nowrap">
                          {formatDate(mov.created_at)}{' '}
                          <span className="text-xs">{formatDateTime(mov.created_at, { timeStyle: 'short' })}</span>
                        </td>
                        <td className="px-4 py-3 font-medium text-foreground">{selectedItem.name}</td>
                        <td className="px-4 py-3">
                          <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${movementTypeColors[mov.movement_type] ?? 'bg-secondary text-muted-foreground'}`}>
                            {MIcon && <MIcon className="h-3 w-3" />}
                            {movementTypeKeys[mov.movement_type] ? t(movementTypeKeys[mov.movement_type]) : mov.movement_type}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right text-foreground tabular-nums">
                          {mov.quantity > 0 ? `+${mov.quantity}` : mov.quantity}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">
                          {mov.location_from && mov.location_to
                            ? `${mov.location_from} → ${mov.location_to}`
                            : mov.location_from ?? mov.location_to ?? '—'}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground font-mono text-xs">{mov.reference ?? '—'}</td>
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
            <button
              onClick={() => toast.success(t('inventar.inventur.newSessionSuccess'))}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-4 w-4" />
              {t('inventar.inventur.newSession')}
            </button>
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

      {/* ─── DETAIL PANEL (slide-over) ──────────────────────── */}
      <DetailPanel
        open={!!selectedItem}
        onClose={() => setSelectedItem(null)}
        title={selectedItem?.name}
        subtitle={selectedItem ? `${selectedItem.sku} · ${selectedItem.category ?? ''}` : undefined}
        badge={
          selectedItem ? (
            <span className={`ml-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
              getStockStatus(selectedItem) === 'critical'
                ? 'bg-error-light text-error'
                : getStockStatus(selectedItem) === 'warning'
                  ? 'bg-warning-light text-warning'
                  : 'bg-success-light text-success'
            }`}>
              {t(getStockStatusDisplay(selectedItem).labelKey)}
            </span>
          ) : undefined
        }
        width="w-[380px]"
        footer={
          selectedItem ? (
            <div className="flex gap-2">
              <button
                onClick={() => {
                  openArtikelDialog(selectedItem)
                }}
                className="flex-1 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('common.edit')}
              </button>
              <button
                onClick={() => {
                  openBewegungDialog(selectedItem)
                }}
                className="flex-1 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                {t('inventar.detail.buttonMovement')}
              </button>
            </div>
          ) : undefined
        }
      >
        {selectedItem && (
          <div className="space-y-5">
            {/* Nachbestell-Banner (critical stock) */}
            {selectedItem.quantity <= selectedItem.min_quantity && (
              <div className="rounded-lg border border-error/30 bg-error-light p-3">
                <div className="flex items-start gap-2">
                  <AlertTriangle className="h-4 w-4 text-error mt-0.5 shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-error">
                      {t('inventar.detail.criticalBanner.title')}
                    </p>
                    <p className="text-xs text-error/80 mt-0.5">
                      {t('inventar.detail.criticalBanner.desc', { current: selectedItem.quantity, min: selectedItem.min_quantity })}
                    </p>
                  </div>
                </div>
                <button
                  onClick={() => toast.success(t('inventar.detail.criticalBanner.orderSuccess', { name: selectedItem.name }))}
                  className="mt-2 w-full flex items-center justify-center gap-2 rounded-lg bg-error px-3 py-2 text-sm text-white hover:bg-error/90 transition-colors"
                >
                  <ShoppingCart className="h-4 w-4" />
                  {t('inventar.detail.criticalBanner.button')}
                </button>
              </div>
            )}

            {/* Basic info */}
            <div className="space-y-3">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('inventar.detail.details')}</h4>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldName')}</p>
                  <p className="text-sm text-foreground font-medium">{selectedItem.name}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">SKU</p>
                  <p className="text-sm text-foreground font-mono">{selectedItem.sku}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldCategory')}</p>
                  <p className="text-sm text-foreground">{selectedItem.category ?? '—'}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldUnit')}</p>
                  <p className="text-sm text-foreground">{selectedItem.unit}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldPrice')}</p>
                  <p className="text-sm text-foreground">{formatCurrency(selectedItem.price ?? 0, selectedItem.currency ?? 'EUR')}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldLastMovement')}</p>
                  <p className="text-sm text-foreground">{formatDate(selectedItem.updated_at)}</p>
                </div>
              </div>
              {selectedItem.barcode && (
                <div>
                  <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldBarcode')}</p>
                  <p className="text-sm text-foreground font-mono">{selectedItem.barcode}</p>
                </div>
              )}
            </div>

            {/* Stock visual bar */}
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('inventar.detail.stockBar.title')}</h4>
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <div className="flex items-baseline justify-between mb-2">
                  <span className="text-2xl font-semibold text-foreground tabular-nums">
                    {selectedItem.quantity}
                  </span>
                  <span className="text-sm text-muted-foreground">
                    Min: {selectedItem.min_quantity} {selectedItem.unit}
                  </span>
                </div>
                <StockBar current={selectedItem.quantity} min={selectedItem.min_quantity} />
              </div>
            </div>

            {/* Chargen & Seriennummern */}
            {(selectedItem.batch_number || (selectedItem.serial_numbers && selectedItem.serial_numbers.length > 0)) && (
              <div className="space-y-2">
                <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('inventar.detail.chargenTitle')}</h4>
                <div className="rounded-lg border border-border p-3 space-y-2">
                  {selectedItem.batch_number && (
                    <div className="flex items-center gap-2">
                      <Hash className="h-3.5 w-3.5 text-muted-foreground" />
                      <span className="text-xs text-muted-foreground">{t('inventar.detail.chargeLabel')}</span>
                      <span className="text-sm text-foreground font-mono">{selectedItem.batch_number}</span>
                    </div>
                  )}
                  {selectedItem.serial_numbers && selectedItem.serial_numbers.length > 0 && (
                    <div>
                      <div className="flex items-center gap-2 mb-1.5">
                        <Hash className="h-3.5 w-3.5 text-muted-foreground" />
                        <span className="text-xs text-muted-foreground">{t('inventar.detail.serialLabel')}</span>
                      </div>
                      <div className="flex flex-wrap gap-1">
                        {selectedItem.serial_numbers.map((sn) => (
                          <span
                            key={sn}
                            className="inline-flex items-center rounded-md bg-secondary px-2 py-0.5 text-xs font-mono text-foreground"
                          >
                            {sn}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Belegkette / Einkauf-Anbindung */}
            {selectedItem.linked_purchase_order && (
              <div className="space-y-2">
                <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('inventar.detail.einkaufTitle')}</h4>
                <div className="rounded-lg border border-border p-3">
                  <div className="flex items-center gap-2 mb-2">
                    <Link2 className="h-4 w-4 text-primary" />
                    <span className="text-sm text-foreground">
                      {t('inventar.detail.linkedOrder')} <span className="font-mono font-medium">{selectedItem.linked_purchase_order}</span>
                    </span>
                  </div>
                  <button
                    onClick={() => toast.info(`Wechsle zu Bestellung ${selectedItem.linked_purchase_order}`)}
                    className="w-full flex items-center justify-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                  >
                    <Link2 className="h-3.5 w-3.5" />
                    {t('inventar.detail.toOrder')}
                  </button>
                </div>
              </div>
            )}

            {/* Lagerorte */}
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('inventar.detail.lagerortTitle')}</h4>
              <div className="rounded-lg border border-border p-3">
                <div className="flex items-center gap-2">
                  {(() => {
                    const loc = locations.find((l) => l.id === selectedItem.location_id)
                    const LIcon = loc ? locationTypeIcons[loc.type] || Warehouse : Warehouse
                    return (
                      <>
                        <LIcon className="h-4 w-4 text-primary" />
                        <div className="flex-1">
                          <p className="text-sm font-medium text-foreground">{selectedItem.location ?? '—'}</p>
                          {loc && <p className="text-xs text-muted-foreground">{loc.address}</p>}
                        </div>
                        <span className="text-sm tabular-nums text-foreground font-medium">
                          {selectedItem.quantity} {selectedItem.unit}
                        </span>
                      </>
                    )
                  })()}
                </div>
              </div>
            </div>

            {/* Letzte Bewegungen */}
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('inventar.detail.lastMovements')}</h4>
              {(() => {
                const itemMovements = selectedItemMovementsQuery.data?.movements?.slice(0, 5) ?? []
                if (itemMovements.length === 0) {
                  return (
                    <p className="text-xs text-muted-foreground py-2">{t('inventar.detail.noMovements')}</p>
                  )
                }
                return (
                  <div className="space-y-1">
                    {itemMovements.map((mov) => {
                      const MIcon = movementTypeIcons[mov.movement_type]
                      return (
                        <div key={mov.id} className="flex items-center gap-2 rounded-md border border-border-muted p-2">
                          <span className={`flex h-6 w-6 items-center justify-center rounded-md ${movementTypeColors[mov.movement_type]}`}>
                            {MIcon && <MIcon className="h-3 w-3" />}
                          </span>
                          <div className="flex-1 min-w-0">
                            <p className="text-xs font-medium text-foreground">
                              {movementTypeKeys[mov.movement_type] ? t(movementTypeKeys[mov.movement_type]) : mov.movement_type}
                              {mov.quantity > 0 ? ` +${mov.quantity}` : ` ${mov.quantity}`}
                            </p>
                            <p className="text-[10px] text-muted-foreground truncate">
                              {mov.reference ?? '—'} · {mov.performed_by ?? '—'}
                            </p>
                          </div>
                          <span className="text-[10px] text-muted-foreground whitespace-nowrap">
                            {formatDateTime(mov.created_at)}
                          </span>
                        </div>
                      )
                    })}
                  </div>
                )
              })()}
            </div>

            {/* Anhänge */}
            <ItemAttachmentsSection itemId={selectedItem.id} />
          </div>
        )}
      </DetailPanel>

      {/* ─── DIALOGS ────────────────────────────────────────── */}
      <ArtikelDialog
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

// ─── Item Attachments ─────────────────────────────────────────────
// Browser-direct upload (scope=inventar) + register against the item.

function ItemAttachmentsSection({ itemId }: { itemId: string }) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)
  const attachmentsQuery = useItemAttachments(itemId)
  const uploadMut = useUploadItemAttachment()
  const deleteMut = useDeleteItemAttachment()
  const attachments = attachmentsQuery.data?.attachments ?? []

  const handleFile = (file: File | undefined) => {
    if (!file) return
    uploadMut.mutate(
      { itemId, file },
      { onError: () => toast.error(t('inventar.detail.attachmentUploadError')) },
    )
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          {t('inventar.detail.attachments')}
        </h4>
        <input
          ref={fileRef}
          type="file"
          className="hidden"
          onChange={(e) => {
            handleFile(e.target.files?.[0])
            e.target.value = ''
          }}
        />
        <button
          type="button"
          onClick={() => fileRef.current?.click()}
          disabled={uploadMut.isPending}
          className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors disabled:opacity-50"
        >
          <Paperclip className="h-3 w-3" />
          {t('inventar.detail.attachmentAdd')}
        </button>
      </div>
      {attachments.length === 0 ? (
        <p className="text-xs text-muted-foreground py-2">{t('inventar.detail.attachmentsEmpty')}</p>
      ) : (
        <div className="space-y-1">
          {attachments.map((att) => (
            <ItemAttachmentRow
              key={att.id}
              attachment={att}
              onRemove={() => deleteMut.mutate({ id: att.id, itemId })}
              removeLabel={t('inventar.detail.attachmentRemove')}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function ItemAttachmentRow({
  attachment,
  onRemove,
  removeLabel,
}: {
  attachment: ItemAttachment
  onRemove: () => void
  removeLabel: string
}) {
  const src = useAvatarSrc(attachment.object_key)
  return (
    <div className="flex items-center gap-2 rounded-md border border-border-muted p-2">
      <Paperclip className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
      {src ? (
        <a
          href={src}
          target="_blank"
          rel="noopener noreferrer"
          className="flex-1 min-w-0 truncate text-xs text-primary hover:underline"
        >
          {attachment.name}
        </a>
      ) : (
        <span className="flex-1 min-w-0 truncate text-xs text-foreground">{attachment.name}</span>
      )}
      <button
        type="button"
        onClick={onRemove}
        aria-label={removeLabel}
        title={removeLabel}
        className="shrink-0 rounded p-0.5 text-muted-foreground hover:text-error transition-colors"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}
