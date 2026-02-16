import { useState, useMemo, useCallback } from 'react'
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
  X,
  ArrowDownToLine,
  ArrowUpFromLine,
  RefreshCw,
  ClipboardEdit,
  ChevronDown,
  Filter,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useInventarStore, type InventoryItem } from '@/stores/inventar'
import { ItemActions, ConfirmDialog, EmptyState, DetailPanel } from '@/components/shared'

type TabKey = 'artikel' | 'lagerorte' | 'bewegungen'
type StatusFilter = 'all' | 'ok' | 'warning' | 'critical'
type MovementType = 'in' | 'out' | 'transfer' | 'adjustment'

const movementTypeLabels: Record<string, string> = {
  in: 'Eingang',
  out: 'Ausgang',
  transfer: 'Transfer',
  adjustment: 'Korrektur',
}

const movementTypeColors: Record<string, string> = {
  in: 'bg-success-light text-success',
  out: 'bg-error-light text-error',
  transfer: 'bg-info-light text-info',
  adjustment: 'bg-warning-light text-warning',
}

const locationTypeLabels: Record<string, string> = {
  warehouse: 'Lager',
  store: 'Filiale',
  vehicle: 'Fahrzeug',
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

const UNIT_OPTIONS = ['Stück', 'kg', 'Meter', 'Liter', 'Packung', 'Rolle']

function getStockStatus(item: InventoryItem): 'ok' | 'warning' | 'critical' {
  if (item.currentStock <= item.minStock) return 'critical'
  if (item.currentStock < item.minStock * 2) return 'warning'
  return 'ok'
}

function getStockStatusDisplay(item: InventoryItem): { color: string; label: string; dotColor: string } {
  const status = getStockStatus(item)
  if (status === 'critical') return { color: 'bg-error', label: 'Kritisch', dotColor: 'bg-error' }
  if (status === 'warning') return { color: 'bg-warning', label: 'Warnung', dotColor: 'bg-warning' }
  return { color: 'bg-success', label: 'OK', dotColor: 'bg-success' }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr + (dateStr.includes('T') ? '' : 'T00:00:00')).toLocaleDateString('de-CH')
}

function formatDateTime(dateStr: string): string {
  const d = new Date(dateStr)
  return `${d.toLocaleDateString('de-CH')} ${d.toLocaleTimeString('de-CH', { hour: '2-digit', minute: '2-digit' })}`
}

// ─── Artikel Dialog ───────────────────────────────────────────────
interface ArtikelFormData {
  name: string
  sku: string
  category: string
  minStock: number
  unit: string
}

function ArtikelDialog({
  open,
  onClose,
  initial,
}: {
  open: boolean
  onClose: () => void
  initial?: InventoryItem | null
}) {
  const [form, setForm] = useState<ArtikelFormData>({
    name: initial?.name ?? '',
    sku: initial?.sku ?? '',
    category: initial?.category ?? '',
    minStock: initial?.minStock ?? 0,
    unit: initial?.unit ?? 'Stück',
  })

  const isEdit = !!initial

  const handleSave = () => {
    if (!form.name.trim()) {
      toast.error('Bitte einen Artikelnamen eingeben')
      return
    }
    if (!form.sku.trim()) {
      toast.error('Bitte eine SKU eingeben')
      return
    }
    toast.success(isEdit ? `"${form.name}" wurde aktualisiert` : `"${form.name}" wurde angelegt`)
    onClose()
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-lg font-semibold text-foreground">
            {isEdit ? 'Artikel bearbeiten' : 'Neuen Artikel anlegen'}
          </h2>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4">
          {/* Name */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              Name <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="z.B. Kabelkanal 20x10mm"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* SKU */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              SKU <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={form.sku}
              onChange={(e) => setForm((f) => ({ ...f, sku: e.target.value }))}
              placeholder="z.B. KK-2010"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring font-mono"
            />
          </div>

          {/* Kategorie */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Kategorie</label>
            <input
              type="text"
              value={form.category}
              onChange={(e) => setForm((f) => ({ ...f, category: e.target.value }))}
              placeholder="z.B. Elektromaterial"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Mindestbestand + Einheit */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Mindestbestand</label>
              <input
                type="number"
                min={0}
                value={form.minStock}
                onChange={(e) => setForm((f) => ({ ...f, minStock: Number(e.target.value) }))}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Einheit</label>
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
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 mt-6">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            Abbrechen
          </button>
          <button
            onClick={handleSave}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            {isEdit ? 'Speichern' : 'Anlegen'}
          </button>
        </div>
      </div>
    </div>
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
  item: InventoryItem | null
  locations: { id: string; name: string }[]
}) {
  const [typ, setTyp] = useState<MovementType>('in')
  const [menge, setMenge] = useState<number>(1)
  const [lagerort, setLagerort] = useState(locations[0]?.name ?? '')
  const [referenz, setReferenz] = useState('')
  const [notizen, setNotizen] = useState('')

  const handleSave = () => {
    if (menge <= 0) {
      toast.error('Bitte eine gueltige Menge eingeben')
      return
    }
    const label = movementTypeLabels[typ]
    toast.success(`${label}: ${menge} ${item?.unit ?? 'Stk'} von "${item?.name}" erfasst`)
    onClose()
  }

  if (!open || !item) return null

  const typeOptions: { key: MovementType; label: string; icon: typeof ArrowDownToLine }[] = [
    { key: 'in', label: 'Eingang', icon: ArrowDownToLine },
    { key: 'out', label: 'Ausgang', icon: ArrowUpFromLine },
    { key: 'transfer', label: 'Transfer', icon: RefreshCw },
    { key: 'adjustment', label: 'Korrektur', icon: ClipboardEdit },
  ]

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated">
        <div className="flex items-center justify-between mb-5">
          <div>
            <h2 className="text-lg font-semibold text-foreground">Bestandsbewegung</h2>
            <p className="text-xs text-muted-foreground mt-0.5">{item.name} ({item.sku})</p>
          </div>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4">
          {/* Typ radio */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Typ</label>
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
            <label className="text-sm font-medium text-foreground">Menge ({item.unit})</label>
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
            <label className="text-sm font-medium text-foreground">Lagerort</label>
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
            <label className="text-sm font-medium text-foreground">Referenz <span className="text-xs text-muted-foreground font-normal">(optional)</span></label>
            <input
              type="text"
              value={referenz}
              onChange={(e) => setReferenz(e.target.value)}
              placeholder="z.B. PO-2024-032 oder Auftrag #A-450"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Notizen */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Notizen <span className="text-xs text-muted-foreground font-normal">(optional)</span></label>
            <textarea
              value={notizen}
              onChange={(e) => setNotizen(e.target.value)}
              placeholder="Zusaetzliche Informationen..."
              rows={3}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
            />
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 mt-6">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            Abbrechen
          </button>
          <button
            onClick={handleSave}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            Bewegung erfassen
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── Stock Bar Visual ─────────────────────────────────────────────
function StockBar({ current, min }: { current: number; min: number }) {
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
          title={`Mindestbestand: ${min}`}
        />
      </div>
      <div className="flex justify-between text-[10px] text-muted-foreground">
        <span>0</span>
        <span>Min: {min}</span>
        <span>{Math.round(max)}</span>
      </div>
    </div>
  )
}

// ─── Main Page ────────────────────────────────────────────────────
export default function InventarPage() {
  const { items, locations, movements } = useInventarStore()

  const [tab, setTab] = useState<TabKey>('artikel')
  const [search, setSearch] = useState('')
  const [categoryFilter, setCategoryFilter] = useState('all')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [confirmDelete, setConfirmDelete] = useState<InventoryItem | null>(null)

  // Detail panel
  const [selectedItem, setSelectedItem] = useState<InventoryItem | null>(null)

  // Dialogs
  const [artikelDialogOpen, setArtikelDialogOpen] = useState(false)
  const [editItem, setEditItem] = useState<InventoryItem | null>(null)
  const [bewegungDialogOpen, setBewegungDialogOpen] = useState(false)
  const [bewegungItem, setBewegungItem] = useState<InventoryItem | null>(null)

  // Get unique categories from items
  const allCategories = useMemo(() => {
    const cats = new Set(items.map((i) => i.category))
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
          item.category.toLowerCase().includes(q) ||
          item.locationName.toLowerCase().includes(q),
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

  const filteredMovements = useMemo(() => {
    if (!search) return movements
    const q = search.toLowerCase()
    return movements.filter(
      (mov) =>
        mov.itemName.toLowerCase().includes(q) ||
        mov.reference.toLowerCase().includes(q) ||
        mov.createdBy.toLowerCase().includes(q),
    )
  }, [movements, search])

  const lowStockCount = items.filter((i) => i.currentStock <= i.minStock).length
  const warningCount = items.filter((i) => getStockStatus(i) === 'warning').length

  // Movements for a specific item (detail panel)
  const getItemMovements = useCallback(
    (itemId: string) => movements.filter((m) => m.itemId === itemId).slice(0, 5),
    [movements],
  )

  // Items at a specific location (detail panel)
  const getLocationItems = useCallback(
    (locationName: string) => items.filter((i) => i.locationName === locationName),
    [items],
  )

  const openArtikelDialog = (item?: InventoryItem) => {
    setEditItem(item ?? null)
    setArtikelDialogOpen(true)
  }

  const openBewegungDialog = (item: InventoryItem) => {
    setBewegungItem(item)
    setBewegungDialogOpen(true)
  }

  const getItemActions = (item: InventoryItem) => [
    { label: 'Details anzeigen', icon: Eye, onClick: () => setSelectedItem(item) },
    { label: 'Bearbeiten', icon: Edit, onClick: () => openArtikelDialog(item) },
    { label: 'Bestandsbewegung', icon: ArrowRightLeft, onClick: () => openBewegungDialog(item), separator: true },
    { separator: true as const, label: '', onClick: () => {} },
    { label: 'Loeschen', icon: Trash2, variant: 'destructive' as const, onClick: () => setConfirmDelete(item) },
  ]

  const handleDelete = (item: InventoryItem) => {
    setConfirmDelete(null)
    if (selectedItem?.id === item.id) setSelectedItem(null)
    toast.success(`"${item.name}" wurde geloescht`)
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div>
          <h1 className="text-foreground">Inventar</h1>
          <p className="text-sm text-muted-foreground">
            {items.length} Artikel
            {lowStockCount > 0 && (
              <span className="text-error"> · {lowStockCount} kritisch</span>
            )}
            {warningCount > 0 && (
              <span className="text-warning"> · {warningCount} Warnung</span>
            )}
            {lowStockCount === 0 && warningCount === 0 && ' · Alle Bestaende OK'}
          </p>
        </div>
        <button
          onClick={() => openArtikelDialog()}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Plus className="h-4 w-4" />
          Artikel hinzufuegen
        </button>
      </div>

      {/* Tabs with badges */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'artikel' as const, label: 'Artikel', count: items.length },
          { key: 'lagerorte' as const, label: 'Lagerorte', count: locations.length },
          { key: 'bewegungen' as const, label: 'Bewegungen', count: movements.length },
        ]).map((t) => (
          <button
            key={t.key}
            onClick={() => { setTab(t.key); setSearch('') }}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key ? 'border-primary text-primary font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t.label} ({t.count})
          </button>
        ))}
      </div>

      {/* Search + Filters */}
      <div className="flex flex-wrap items-center gap-3 mb-4">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder={tab === 'artikel' ? 'Artikel suchen...' : tab === 'lagerorte' ? 'Lagerort suchen...' : 'Bewegung suchen...'}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
        </div>

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
                    {cat === 'all' ? 'Alle Kategorien' : cat}
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
                <option value="all">Alle Status</option>
                <option value="ok">OK</option>
                <option value="warning">Warnung</option>
                <option value="critical">Kritisch</option>
              </select>
              <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            </div>

            {(categoryFilter !== 'all' || statusFilter !== 'all') && (
              <button
                onClick={() => { setCategoryFilter('all'); setStatusFilter('all') }}
                className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <Filter className="h-3.5 w-3.5" />
                Filter zuruecksetzen
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
              title="Keine Artikel gefunden"
              description={
                search || categoryFilter !== 'all' || statusFilter !== 'all'
                  ? 'Passe deine Suche oder Filter an'
                  : 'Fuege deinen ersten Artikel hinzu'
              }
            />
          ) : (
            <TooltipProvider delayDuration={300}>
              <div className="overflow-x-auto rounded-lg border border-border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border bg-card">
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground w-12">Status</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">SKU</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">Kategorie</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground">Bestand</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground">Mindest.</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">Standort</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground">Preis</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground w-12"></th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredItems.map((item) => {
                      const status = getStockStatusDisplay(item)
                      const isSelected = selectedItem?.id === item.id
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
                                <p className="text-xs">Bestand: {item.currentStock} / Min: {item.minStock}</p>
                                <p className="text-xs font-medium">{status.label}</p>
                              </TooltipContent>
                            </Tooltip>
                          </td>
                          <td className="px-4 py-3 font-medium text-foreground">{item.name}</td>
                          <td className="px-4 py-3 text-muted-foreground font-mono text-xs">{item.sku}</td>
                          <td className="px-4 py-3">
                            <span className="rounded-full bg-secondary px-2 py-0.5 text-xs text-muted-foreground">{item.category}</span>
                          </td>
                          <td className="px-4 py-3 text-right text-foreground tabular-nums">
                            {item.currentStock} {item.unit}
                          </td>
                          <td className="px-4 py-3 text-right text-muted-foreground tabular-nums">
                            {item.minStock}
                          </td>
                          <td className="px-4 py-3 text-muted-foreground">{item.locationName}</td>
                          <td className="px-4 py-3 text-right text-foreground tabular-nums">
                            CHF {item.price.toFixed(2)}
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
              title="Keine Lagerorte gefunden"
              description={search ? 'Passe deine Suche an' : 'Erstelle deinen ersten Lagerort'}
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
                              <span className="tabular-nums">{li.currentStock} {li.unit}</span>
                            </div>
                          )
                        })}
                        {locItems.length > 3 && (
                          <p className="text-xs text-muted-foreground/60 pl-3.5">
                            +{locItems.length - 3} weitere Artikel
                          </p>
                        )}
                      </div>
                    )}

                    <div className="flex items-center justify-between border-t border-border-muted pt-3">
                      <div className="flex items-center gap-2">
                        <span className="rounded-full bg-secondary px-2 py-0.5 text-xs text-muted-foreground">
                          {locationTypeLabels[loc.type]}
                        </span>
                        {criticalInLoc > 0 && (
                          <span className="rounded-full bg-error-light px-2 py-0.5 text-xs text-error">
                            {criticalInLoc} kritisch
                          </span>
                        )}
                      </div>
                      <span className="text-sm text-foreground font-medium">{loc.itemCount} Artikel</span>
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
          {filteredMovements.length === 0 ? (
            <EmptyState
              icon={ArrowRightLeft}
              title="Keine Bewegungen gefunden"
              description={search ? 'Passe deine Suche an' : 'Es gibt noch keine Lagerbewegungen'}
            />
          ) : (
            <div className="overflow-x-auto rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-card">
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Datum</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Artikel</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Typ</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground">Menge</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Von / Nach</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Referenz</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Erstellt von</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredMovements.map((mov) => {
                    const MIcon = movementTypeIcons[mov.type]
                    return (
                      <tr key={mov.id} className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors">
                        <td className="px-4 py-3 text-muted-foreground whitespace-nowrap">
                          {new Date(mov.createdAt).toLocaleDateString('de-CH')}{' '}
                          <span className="text-xs">{new Date(mov.createdAt).toLocaleTimeString('de-CH', { hour: '2-digit', minute: '2-digit' })}</span>
                        </td>
                        <td className="px-4 py-3 font-medium text-foreground">{mov.itemName}</td>
                        <td className="px-4 py-3">
                          <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${movementTypeColors[mov.type] ?? 'bg-secondary text-muted-foreground'}`}>
                            {MIcon && <MIcon className="h-3 w-3" />}
                            {movementTypeLabels[mov.type] ?? mov.type}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right text-foreground tabular-nums">
                          {mov.quantity > 0 ? `+${mov.quantity}` : mov.quantity}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">
                          {mov.locationFrom && mov.locationTo
                            ? `${mov.locationFrom} → ${mov.locationTo}`
                            : mov.locationFrom
                              ? mov.locationFrom
                              : mov.locationTo ?? '—'}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground font-mono text-xs">{mov.reference}</td>
                        <td className="px-4 py-3 text-muted-foreground">{mov.createdBy}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* ─── DETAIL PANEL (slide-over) ──────────────────────── */}
      <DetailPanel
        open={!!selectedItem}
        onClose={() => setSelectedItem(null)}
        title={selectedItem?.name}
        subtitle={selectedItem ? `${selectedItem.sku} · ${selectedItem.category}` : undefined}
        badge={
          selectedItem ? (
            <span className={`ml-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
              getStockStatus(selectedItem) === 'critical'
                ? 'bg-error-light text-error'
                : getStockStatus(selectedItem) === 'warning'
                  ? 'bg-warning-light text-warning'
                  : 'bg-success-light text-success'
            }`}>
              {getStockStatusDisplay(selectedItem).label}
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
                Bearbeiten
              </button>
              <button
                onClick={() => {
                  openBewegungDialog(selectedItem)
                }}
                className="flex-1 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                Bestandsbewegung
              </button>
            </div>
          ) : undefined
        }
      >
        {selectedItem && (
          <div className="space-y-5">
            {/* Basic info */}
            <div className="space-y-3">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Details</h4>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <p className="text-xs text-muted-foreground">Artikel-Name</p>
                  <p className="text-sm text-foreground font-medium">{selectedItem.name}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">SKU</p>
                  <p className="text-sm text-foreground font-mono">{selectedItem.sku}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Kategorie</p>
                  <p className="text-sm text-foreground">{selectedItem.category}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Einheit</p>
                  <p className="text-sm text-foreground">{selectedItem.unit}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Preis</p>
                  <p className="text-sm text-foreground">CHF {selectedItem.price.toFixed(2)}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Letzte Bewegung</p>
                  <p className="text-sm text-foreground">{formatDate(selectedItem.lastMovement)}</p>
                </div>
              </div>
              {selectedItem.barcode && (
                <div>
                  <p className="text-xs text-muted-foreground">Barcode</p>
                  <p className="text-sm text-foreground font-mono">{selectedItem.barcode}</p>
                </div>
              )}
            </div>

            {/* Stock visual bar */}
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Bestand vs Mindestbestand</h4>
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <div className="flex items-baseline justify-between mb-2">
                  <span className="text-2xl font-semibold text-foreground tabular-nums">
                    {selectedItem.currentStock}
                  </span>
                  <span className="text-sm text-muted-foreground">
                    Min: {selectedItem.minStock} {selectedItem.unit}
                  </span>
                </div>
                <StockBar current={selectedItem.currentStock} min={selectedItem.minStock} />
              </div>
            </div>

            {/* Lagerorte */}
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Lagerort</h4>
              <div className="rounded-lg border border-border p-3">
                <div className="flex items-center gap-2">
                  {(() => {
                    const loc = locations.find((l) => l.id === selectedItem.locationId)
                    const LIcon = loc ? locationTypeIcons[loc.type] || Warehouse : Warehouse
                    return (
                      <>
                        <LIcon className="h-4 w-4 text-primary" />
                        <div className="flex-1">
                          <p className="text-sm font-medium text-foreground">{selectedItem.locationName}</p>
                          {loc && <p className="text-xs text-muted-foreground">{loc.address}</p>}
                        </div>
                        <span className="text-sm tabular-nums text-foreground font-medium">
                          {selectedItem.currentStock} {selectedItem.unit}
                        </span>
                      </>
                    )
                  })()}
                </div>
              </div>
            </div>

            {/* Letzte Bewegungen */}
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Letzte Bewegungen</h4>
              {(() => {
                const itemMovements = getItemMovements(selectedItem.id)
                if (itemMovements.length === 0) {
                  return (
                    <p className="text-xs text-muted-foreground py-2">Keine Bewegungen vorhanden</p>
                  )
                }
                return (
                  <div className="space-y-1">
                    {itemMovements.map((mov) => {
                      const MIcon = movementTypeIcons[mov.type]
                      return (
                        <div key={mov.id} className="flex items-center gap-2 rounded-md border border-border-muted p-2">
                          <span className={`flex h-6 w-6 items-center justify-center rounded-md ${movementTypeColors[mov.type]}`}>
                            {MIcon && <MIcon className="h-3 w-3" />}
                          </span>
                          <div className="flex-1 min-w-0">
                            <p className="text-xs font-medium text-foreground">
                              {movementTypeLabels[mov.type]}
                              {mov.quantity > 0 ? ` +${mov.quantity}` : ` ${mov.quantity}`}
                            </p>
                            <p className="text-[10px] text-muted-foreground truncate">
                              {mov.reference} · {mov.createdBy}
                            </p>
                          </div>
                          <span className="text-[10px] text-muted-foreground whitespace-nowrap">
                            {formatDateTime(mov.createdAt)}
                          </span>
                        </div>
                      )
                    })}
                  </div>
                )
              })()}
            </div>
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

      {/* Confirm Delete */}
      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title="Artikel loeschen?"
        description={`"${confirmDelete?.name}" wird unwiderruflich geloescht. Alle zugehoerigen Bewegungen bleiben erhalten.`}
        confirmLabel="Loeschen"
        variant="destructive"
        onConfirm={() => confirmDelete && handleDelete(confirmDelete)}
      />
    </div>
  )
}
