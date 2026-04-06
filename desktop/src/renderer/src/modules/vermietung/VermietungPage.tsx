import { useState, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Plus,
  Wrench,
  DoorOpen,
  Car,
  Hammer,
  MapPin,
  CalendarDays,
  ChevronLeft,
  ChevronRight,
  X,
  Package,
  CheckCircle2,
  Clock,
  AlertTriangle,
  BarChart3,
  Edit as EditIcon,
  Trash2,
  Eye,
  CalendarPlus,
  ChevronDown,
  Filter,
  ClipboardCheck,
  Building2,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useVermietungStore,
  type RentalObject,
  type RentalObjectType,
  type Reservation,
} from '@/stores/vermietung'
import { ItemActions, ConfirmDialog, EmptyState, DetailPanel, PageHeader } from '@/components/shared'
import { formatCurrency } from '@/lib/format'
import ZustandsprotokollDialog from './ZustandsprotokollDialog'

// ============================================================
// Types
// ============================================================

type TabKey = 'objekte' | 'reservierungen' | 'kalender'
type ReservationFilter = 'all' | 'active' | 'upcoming' | 'completed'

// ============================================================
// Constants
// ============================================================

const WEEKDAYS = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']

const OBJECT_TYPE_CONFIG: Record<
  RentalObjectType,
  { labelKey: string; icon: typeof Wrench; color: string; bg: string; badgeBg: string }
> = {
  gerät: { labelKey: 'vermietung.objectType.gerät', icon: Wrench, color: 'text-amber-600 dark:text-amber-400', bg: 'bg-amber-100 dark:bg-amber-900/30', badgeBg: 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300' },
  raum: { labelKey: 'vermietung.objectType.raum', icon: DoorOpen, color: 'text-blue-600 dark:text-blue-400', bg: 'bg-blue-100 dark:bg-blue-900/30', badgeBg: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300' },
  fahrzeug: { labelKey: 'vermietung.objectType.fahrzeug', icon: Car, color: 'text-green-600 dark:text-green-400', bg: 'bg-green-100 dark:bg-green-900/30', badgeBg: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300' },
  werkzeug: { labelKey: 'vermietung.objectType.werkzeug', icon: Hammer, color: 'text-orange-600 dark:text-orange-400', bg: 'bg-orange-100 dark:bg-orange-900/30', badgeBg: 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300' },
}

const STATUS_CONFIG: Record<
  RentalObject['status'],
  { labelKey: string; dot: string; text: string }
> = {
  available: { labelKey: 'vermietung.status.available', dot: 'bg-success', text: 'text-success' },
  reserved: { labelKey: 'vermietung.status.reserved', dot: 'bg-info', text: 'text-info' },
  maintenance: { labelKey: 'vermietung.status.maintenance', dot: 'bg-warning', text: 'text-warning' },
}

const RESERVATION_STATUS_CONFIG: Record<
  Reservation['status'],
  { labelKey: string; bg: string }
> = {
  active: { labelKey: 'vermietung.reservationStatus.active', bg: 'bg-success-light text-success' },
  upcoming: { labelKey: 'vermietung.reservationStatus.upcoming', bg: 'bg-info-light text-info' },
  completed: { labelKey: 'vermietung.reservationStatus.completed', bg: 'bg-secondary text-muted-foreground' },
  cancelled: { labelKey: 'vermietung.reservationStatus.cancelled', bg: 'bg-error-light text-error' },
}

const DEPOSIT_STATUS_CONFIG: Record<
  NonNullable<Reservation['depositStatus']>,
  { labelKey: string; bg: string }
> = {
  none: { labelKey: 'vermietung.depositStatus.none', bg: 'bg-secondary text-muted-foreground' },
  collected: { labelKey: 'vermietung.depositStatus.collected', bg: 'bg-success-light text-success' },
  returned: { labelKey: 'vermietung.depositStatus.returned', bg: 'bg-info-light text-info' },
}

const CURRENCY_OPTIONS = ['EUR', 'CHF', 'USD'] as const

// ============================================================
// Helpers
// ============================================================

function getWeekDates(offset: number): string[] {
  const today = new Date()
  const dayOfWeek = today.getDay()
  const monday = new Date(today)
  monday.setDate(today.getDate() - ((dayOfWeek + 6) % 7) + offset * 7)
  const dates: string[] = []
  for (let i = 0; i < 7; i++) {
    const d = new Date(monday)
    d.setDate(monday.getDate() + i)
    dates.push(d.toISOString().split('T')[0])
  }
  return dates
}

function getKW(dateStr: string): number {
  const date = new Date(dateStr + 'T00:00:00')
  const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()))
  const dayNum = d.getUTCDay() || 7
  d.setUTCDate(d.getUTCDate() + 4 - dayNum)
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1))
  return Math.ceil(((d.getTime() - yearStart.getTime()) / 86400000 + 1) / 7)
}

function formatDateRange(dates: string[]): string {
  if (dates.length === 0) return ''
  const first = new Date(dates[0] + 'T00:00:00')
  const last = new Date(dates[dates.length - 1] + 'T00:00:00')
  const opts: Intl.DateTimeFormatOptions = { day: '2-digit', month: 'short' }
  return `${first.toLocaleDateString('de-DE', opts)} – ${last.toLocaleDateString('de-DE', opts)} ${last.getFullYear()}`
}

function formatDate(dateStr: string): string {
  return new Date(dateStr + 'T00:00:00').toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

function isToday(dateStr: string): boolean {
  return dateStr === new Date().toISOString().split('T')[0]
}

function daysBetween(start: string, end: string): number {
  const s = new Date(start + 'T00:00:00')
  const e = new Date(end + 'T00:00:00')
  return Math.max(1, Math.round((e.getTime() - s.getTime()) / 86400000) + 1)
}

function dateInRange(date: string, start: string, end: string): boolean {
  return date >= start && date <= end
}

/** Compute rental price breakdown given days and rates. */
function computeRentalPrice(
  totalDays: number,
  dailyRate: number,
  weeklyRate?: number,
): { weeks: number; remainingDays: number; total: number; breakdown: string; currency?: string } {
  if (weeklyRate && totalDays >= 7) {
    const weeks = Math.floor(totalDays / 7)
    const remainingDays = totalDays % 7
    const total = weeks * weeklyRate + remainingDays * dailyRate
    const parts: string[] = [`${weeks} Woche${weeks > 1 ? 'n' : ''}`]
    if (remainingDays > 0) {
      parts.push(`${remainingDays} Tag${remainingDays > 1 ? 'e' : ''}`)
    }
    return { weeks, remainingDays, total, breakdown: parts.join(' + ') }
  }
  return {
    weeks: 0,
    remainingDays: totalDays,
    total: totalDays * dailyRate,
    breakdown: `${totalDays} Tag${totalDays > 1 ? 'e' : ''}`,
  }
}

// ============================================================
// Sub-components
// ============================================================

function ObjectDialog({
  open,
  onClose,
  initial,
}: {
  open: boolean
  onClose: () => void
  initial?: RentalObject | null
}) {
  const { t } = useTranslation()
  const { addObject, updateObject } = useVermietungStore()
  const isEdit = !!initial

  const [name, setName] = useState(initial?.name ?? '')
  const [type, setType] = useState<RentalObjectType>(initial?.type ?? 'gerät')
  const [location, setLocation] = useState(initial?.location ?? '')
  const [dailyRate, setDailyRate] = useState(initial?.dailyRate?.toString() ?? '')
  const [weeklyRate, setWeeklyRate] = useState(initial?.weeklyRate?.toString() ?? '')
  const [currency, setCurrency] = useState(initial?.currency ?? 'EUR')
  const [deposit, setDeposit] = useState(initial?.deposit?.toString() ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [serialNumber, setSerialNumber] = useState(initial?.serialNumber ?? '')

  const handleSave = () => {
    if (!name.trim()) { toast.error(t('vermietung.objectDialog.errorName')); return }
    if (!location.trim()) { toast.error(t('vermietung.objectDialog.errorStandort')); return }
    if (!dailyRate || Number(dailyRate) <= 0) { toast.error(t('vermietung.objectDialog.errorDailyRate')); return }

    const obj: RentalObject = {
      id: initial?.id ?? `obj-${Date.now()}`,
      name: name.trim(),
      type,
      description: description.trim(),
      location: location.trim(),
      serialNumber: serialNumber.trim() || undefined,
      dailyRate: Number(dailyRate),
      weeklyRate: weeklyRate ? Number(weeklyRate) : undefined,
      currency,
      deposit: deposit ? Number(deposit) : undefined,
      status: initial?.status ?? 'available',
    }

    if (isEdit) {
      updateObject(obj.id, obj)
      toast.success(t('vermietung.objectDialog.updateSuccess', { name: obj.name }))
    } else {
      addObject(obj)
      toast.success(t('vermietung.objectDialog.createSuccess', { name: obj.name }))
    }
    onClose()
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-lg font-semibold text-foreground">
            {isEdit ? t('vermietung.objectDialog.titleEdit') : t('vermietung.objectDialog.titleNew')}
          </h2>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4">
          {/* Name */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vermietung.objectDialog.labelName')} <span className="text-destructive">*</span></label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('vermietung.objectDialog.placeholderName')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Typ */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vermietung.objectDialog.labelTyp')} <span className="text-destructive">*</span></label>
            <div className="grid grid-cols-2 gap-2">
              {(Object.entries(OBJECT_TYPE_CONFIG) as [RentalObjectType, typeof OBJECT_TYPE_CONFIG['gerät']][]).map(([key, cfg]) => {
                const Icon = cfg.icon
                const isActive = type === key
                return (
                  <button
                    key={key}
                    onClick={() => setType(key)}
                    className={`flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors ${
                      isActive
                        ? 'border-primary bg-primary-light text-primary font-medium'
                        : 'border-border text-muted-foreground hover:bg-secondary'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    {t(cfg.labelKey)}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Standort */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vermietung.objectDialog.labelStandort')} <span className="text-destructive">*</span></label>
            <input
              type="text"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              placeholder={t('vermietung.objectDialog.placeholderStandort')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Currency */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vermietung.objectDialog.labelCurrency')}</label>
            <div className="flex gap-2">
              {CURRENCY_OPTIONS.map((cur) => (
                <button
                  key={cur}
                  onClick={() => setCurrency(cur)}
                  className={`rounded-lg border px-3 py-2 text-sm transition-colors ${
                    currency === cur
                      ? 'border-primary bg-primary-light text-primary font-medium'
                      : 'border-border text-muted-foreground hover:bg-secondary'
                  }`}
                >
                  {cur}
                </button>
              ))}
            </div>
          </div>

          {/* Rates */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('vermietung.objectDialog.labelDailyRate', { currency })} <span className="text-destructive">*</span></label>
              <input
                type="number"
                min={0}
                step={0.01}
                value={dailyRate}
                onChange={(e) => setDailyRate(e.target.value)}
                placeholder="0.00"
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('vermietung.objectDialog.labelWeeklyRate', { currency })} <span className="text-xs text-muted-foreground font-normal">{t('common.optional')}</span></label>
              <input
                type="number"
                min={0}
                step={0.01}
                value={weeklyRate}
                onChange={(e) => setWeeklyRate(e.target.value)}
                placeholder="0.00"
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
              />
            </div>
          </div>

          {/* Kaution */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vermietung.objectDialog.labelDeposit', { currency })} <span className="text-xs text-muted-foreground font-normal">{t('common.optional')}</span></label>
            <input
              type="number"
              min={0}
              step={0.01}
              value={deposit}
              onChange={(e) => setDeposit(e.target.value)}
              placeholder="0.00"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
            />
          </div>

          {/* Beschreibung */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vermietung.objectDialog.labelDescription')}</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t('vermietung.objectDialog.placeholderDescription')}
              rows={2}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
            />
          </div>

          {/* Seriennummer */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vermietung.objectDialog.labelSerial')} <span className="text-xs text-muted-foreground font-normal">{t('common.optional')}</span></label>
            <input
              type="text"
              value={serialNumber}
              onChange={(e) => setSerialNumber(e.target.value)}
              placeholder={t('vermietung.objectDialog.placeholderSerial')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring font-mono"
            />
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-6">
          <button onClick={onClose} className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors">
            {t('common.cancel')}
          </button>
          <button onClick={handleSave} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">
            {isEdit ? t('common.save') : t('vermietung.objectDialog.buttonCreate')}
          </button>
        </div>
      </div>
    </div>
  )
}

function ReservationDialog({
  open,
  onClose,
  objects,
  preSelectedObjectId,
  preSelectedDate,
}: {
  open: boolean
  onClose: () => void
  objects: RentalObject[]
  preSelectedObjectId?: string
  preSelectedDate?: string
}) {
  const { t } = useTranslation()
  const { addReservation } = useVermietungStore()

  const [objectId, setObjectId] = useState(preSelectedObjectId ?? '')
  const [startDate, setStartDate] = useState(preSelectedDate ?? '')
  const [endDate, setEndDate] = useState(preSelectedDate ?? '')
  const [renter, setRenter] = useState('')
  const [renterType, setRenterType] = useState<'employee' | 'customer'>('employee')
  const [pickupLocation, setPickupLocation] = useState('')
  const [returnLocation, setReturnLocation] = useState('')
  const [notes, setNotes] = useState('')

  const selectedObj = objects.find((o) => o.id === objectId)

  // Auto-fill locations when object changes
  const handleObjectChange = (id: string) => {
    setObjectId(id)
    const obj = objects.find((o) => o.id === id)
    if (obj) {
      setPickupLocation(obj.location)
      setReturnLocation(obj.location)
    }
  }

  const handleSave = () => {
    if (!objectId) { toast.error(t('vermietung.reservationDialog.errorObjekt')); return }
    if (!startDate || !endDate) { toast.error(t('vermietung.reservationDialog.errorDates')); return }
    if (startDate > endDate) { toast.error(t('vermietung.reservationDialog.errorDateOrder')); return }
    if (!renter.trim()) { toast.error(t('vermietung.reservationDialog.errorMieter')); return }

    const days = daysBetween(startDate, endDate)
    const pricing = selectedObj ? computeRentalPrice(days, selectedObj.dailyRate, selectedObj.weeklyRate) : null
    const objCurrency = selectedObj?.currency ?? 'EUR'

    const res: Reservation = {
      // eslint-disable-next-line react-hooks/purity -- Date.now() used for unique ID generation
      id: `res-${Date.now()}`,
      objectId,
      objectName: selectedObj?.name ?? '',
      startDate,
      endDate,
      renter: renter.trim(),
      renterType,
      notes: notes.trim(),
      status: startDate <= new Date().toISOString().split('T')[0] ? 'active' : 'upcoming',
      pickupLocation: pickupLocation.trim() || selectedObj?.location || '',
      returnLocation: returnLocation.trim() || selectedObj?.location || '',
      currency: objCurrency,
      totalPrice: pricing?.total,
      depositAmount: selectedObj?.deposit,
      depositStatus: selectedObj?.deposit ? 'none' : undefined,
    }

    addReservation(res)
    toast.success(t('vermietung.reservationDialog.createSuccess', { name: res.objectName }), {
      description: `${formatDate(startDate)} – ${formatDate(endDate)} (${renter.trim()})`,
    })
    onClose()
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-lg font-semibold text-foreground">{t('vermietung.reservationDialog.title')}</h2>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4">
          {/* Object */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vermietung.reservationDialog.labelObjekt')} <span className="text-destructive">*</span></label>
            <select
              value={objectId}
              onChange={(e) => handleObjectChange(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              <option value="">{t('vermietung.reservationDialog.selectObjekt')}</option>
              {objects.map((obj) => (
                <option key={obj.id} value={obj.id}>
                  {obj.name} ({t(OBJECT_TYPE_CONFIG[obj.type].labelKey)})
                </option>
              ))}
            </select>
          </div>

          {/* Dates */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('vermietung.reservationDialog.labelVon')} <span className="text-destructive">*</span></label>
              <input
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('vermietung.reservationDialog.labelBis')} <span className="text-destructive">*</span></label>
              <input
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Renter */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vermietung.reservationDialog.labelMieter')} <span className="text-destructive">*</span></label>
            <input
              type="text"
              value={renter}
              onChange={(e) => setRenter(e.target.value)}
              placeholder={t('vermietung.reservationDialog.placeholderMieter')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Renter Type */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vermietung.reservationDialog.labelMieterTyp')}</label>
            <div className="grid grid-cols-2 gap-2">
              <button
                onClick={() => setRenterType('employee')}
                className={`rounded-lg border px-3 py-2 text-sm transition-colors ${
                  renterType === 'employee'
                    ? 'border-primary bg-primary-light text-primary font-medium'
                    : 'border-border text-muted-foreground hover:bg-secondary'
                }`}
              >
                {t('vermietung.reservationDialog.mieterEmployee')}
              </button>
              <button
                onClick={() => setRenterType('customer')}
                className={`rounded-lg border px-3 py-2 text-sm transition-colors ${
                  renterType === 'customer'
                    ? 'border-primary bg-primary-light text-primary font-medium'
                    : 'border-border text-muted-foreground hover:bg-secondary'
                }`}
              >
                {t('vermietung.reservationDialog.mieterCustomer')}
              </button>
            </div>
          </div>

          {/* Locations */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('vermietung.reservationDialog.labelAbholung')}</label>
              <input
                type="text"
                value={pickupLocation}
                onChange={(e) => setPickupLocation(e.target.value)}
                placeholder={t('vermietung.reservationDialog.placeholderAbholung')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('vermietung.reservationDialog.labelRueckgabe')}</label>
              <input
                type="text"
                value={returnLocation}
                onChange={(e) => setReturnLocation(e.target.value)}
                placeholder={t('vermietung.reservationDialog.placeholderRueckgabe')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vermietung.reservationDialog.labelNotizen')}</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder={t('vermietung.reservationDialog.placeholderNotizen')}
              rows={2}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
            />
          </div>

          {/* Price Calculation (9.12) */}
          {selectedObj && startDate && endDate && startDate <= endDate && (
            <div className="rounded-lg border border-border bg-secondary/30 p-3 space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.reservationDialog.priceCalc')}</h4>
              {(() => {
                const days = daysBetween(startDate, endDate)
                const objCurrency = selectedObj.currency ?? 'EUR'
                const pricing = computeRentalPrice(days, selectedObj.dailyRate, selectedObj.weeklyRate)

                return (
                  <div className="space-y-1">
                    {pricing.weeks > 0 ? (
                      <>
                        <p className="text-xs text-muted-foreground tabular-nums">
                          {pricing.weeks} Woche{pricing.weeks > 1 ? 'n' : ''} &times; {formatCurrency(selectedObj.weeklyRate!, objCurrency)}/Woche
                          = {formatCurrency(pricing.weeks * selectedObj.weeklyRate!, objCurrency)}
                        </p>
                        {pricing.remainingDays > 0 && (
                          <p className="text-xs text-muted-foreground tabular-nums">
                            + {pricing.remainingDays} Tag{pricing.remainingDays > 1 ? 'e' : ''} &times; {formatCurrency(selectedObj.dailyRate, objCurrency)}/Tag
                            = {formatCurrency(pricing.remainingDays * selectedObj.dailyRate, objCurrency)}
                          </p>
                        )}
                      </>
                    ) : (
                      <p className="text-xs text-muted-foreground tabular-nums">
                        {days} Tag{days > 1 ? 'e' : ''} &times; {formatCurrency(selectedObj.dailyRate, objCurrency)}/Tag
                        = {formatCurrency(pricing.total, objCurrency)}
                      </p>
                    )}
                    <div className="border-t border-border-muted pt-1.5 mt-1.5">
                      <p className="text-sm font-semibold text-foreground tabular-nums">
                        {t('vermietung.reservationDialog.total', { amount: formatCurrency(pricing.total, objCurrency) })}
                      </p>
                    </div>
                  </div>
                )
              })()}
            </div>
          )}

          {/* Deposit Notice (9.13) */}
          {selectedObj?.deposit && (
            <div className="rounded-lg border border-amber-200 dark:border-amber-800/50 bg-amber-50 dark:bg-amber-900/20 p-3">
              <p className="text-xs font-medium text-amber-700 dark:text-amber-400">
                {t('vermietung.reservationDialog.depositNotice', { amount: formatCurrency(selectedObj.deposit, selectedObj.currency ?? 'EUR') })}
              </p>
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 mt-6">
          <button onClick={onClose} className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors">
            {t('common.cancel')}
          </button>
          <button onClick={handleSave} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">
            {t('vermietung.reservationDialog.buttonCreate')}
          </button>
        </div>
      </div>
    </div>
  )
}

// ============================================================
// Main Page
// ============================================================

export default function VermietungPage() {
  const { t } = useTranslation()
  const { objects, reservations, zustandsprotokolle, deleteObject, cancelReservation } = useVermietungStore()

  // State
  const [tab, setTab] = useState<TabKey>('objekte')
  const [search, setSearch] = useState('')
  const [weekOffset, setWeekOffset] = useState(0)
  const [reservationFilter, setReservationFilter] = useState<ReservationFilter>('all')
  const [hoveredCell, setHoveredCell] = useState<string | null>(null)

  // Detail panel
  const [selectedObject, setSelectedObject] = useState<RentalObject | null>(null)

  // Dialogs
  const [objectDialogOpen, setObjectDialogOpen] = useState(false)
  const [editObject, setEditObject] = useState<RentalObject | null>(null)
  const [reservationDialogOpen, setReservationDialogOpen] = useState(false)
  const [preSelectedObjectId, setPreSelectedObjectId] = useState<string | undefined>()
  const [preSelectedDate, setPreSelectedDate] = useState<string | undefined>()
  const [confirmDelete, setConfirmDelete] = useState<RentalObject | null>(null)
  const [confirmCancel, setConfirmCancel] = useState<Reservation | null>(null)
  const [zustandsprotokollReservation, setZustandsprotokollReservation] = useState<Reservation | null>(null)

  // Derived data
  const weekDates = useMemo(() => getWeekDates(weekOffset), [weekOffset])
  const kw = useMemo(() => getKW(weekDates[0]), [weekDates])
  const dateRange = useMemo(() => formatDateRange(weekDates), [weekDates])

  const availableCount = objects.filter((o) => o.status === 'available').length
  const reservedCount = objects.filter((o) => o.status === 'reserved').length
  const maintenanceCount = objects.filter((o) => o.status === 'maintenance').length
  const activeReservations = reservations.filter((r) => r.status === 'active' || r.status === 'upcoming').length
  const utilization = objects.length > 0 ? Math.round(((reservedCount + maintenanceCount) / objects.length) * 100) : 0

  // Filtered objects
  const filteredObjects = useMemo(() => {
    if (!search) return objects
    const q = search.toLowerCase()
    return objects.filter(
      (o) =>
        o.name.toLowerCase().includes(q) ||
        o.location.toLowerCase().includes(q) ||
        OBJECT_TYPE_CONFIG[o.type].label.toLowerCase().includes(q) ||
        (o.serialNumber && o.serialNumber.toLowerCase().includes(q)),
    )
  }, [objects, search])

  // Filtered reservations
  const filteredReservations = useMemo(() => {
    let result = [...reservations].sort((a, b) => b.startDate.localeCompare(a.startDate))

    if (reservationFilter !== 'all') {
      result = result.filter((r) => r.status === reservationFilter)
    }

    if (search) {
      const q = search.toLowerCase()
      result = result.filter(
        (r) =>
          r.objectName.toLowerCase().includes(q) ||
          r.renter.toLowerCase().includes(q) ||
          r.notes.toLowerCase().includes(q),
      )
    }

    return result
  }, [reservations, reservationFilter, search])

  // Reservation lookup for calendar
  const getReservationsForObjectAndDate = useCallback(
    (objectId: string, date: string) => {
      return reservations.filter(
        (r) =>
          r.objectId === objectId &&
          (r.status === 'active' || r.status === 'upcoming') &&
          dateInRange(date, r.startDate, r.endDate),
      )
    },
    [reservations],
  )

  // Object reservations for detail panel
  const getObjectReservations = useCallback(
    (objectId: string) =>
      reservations
        .filter((r) => r.objectId === objectId)
        .sort((a, b) => b.startDate.localeCompare(a.startDate))
        .slice(0, 5),
    [reservations],
  )

  // Next reservation for an object
  const getNextReservation = useCallback(
    (objectId: string) => {
      const today = new Date().toISOString().split('T')[0]
      return reservations
        .filter((r) => r.objectId === objectId && (r.status === 'active' || r.status === 'upcoming') && r.endDate >= today)
        .sort((a, b) => a.startDate.localeCompare(b.startDate))[0]
    },
    [reservations],
  )

  // Active reservation for an object (current renter)
  const getActiveReservation = useCallback(
    (objectId: string) => {
      const today = new Date().toISOString().split('T')[0]
      return reservations.find(
        (r) => r.objectId === objectId && r.status === 'active' && dateInRange(today, r.startDate, r.endDate),
      )
    },
    [reservations],
  )

  // Check if a reservation has zustandsprotokoll(e)
  const hasZustandsprotokoll = useCallback(
    (reservationId: string) => zustandsprotokolle.some((z) => z.reservationId === reservationId),
    [zustandsprotokolle],
  )

  // Handlers
  const openObjectDialog = (obj?: RentalObject) => {
    setEditObject(obj ?? null)
    setObjectDialogOpen(true)
  }

  const openReservationDialog = (objectId?: string, date?: string) => {
    setPreSelectedObjectId(objectId)
    setPreSelectedDate(date)
    setReservationDialogOpen(true)
  }

  const handleDeleteObject = (obj: RentalObject) => {
    deleteObject(obj.id)
    setConfirmDelete(null)
    if (selectedObject?.id === obj.id) setSelectedObject(null)
    toast.success(t('vermietung.delete.success', { name: obj.name }))
  }

  const handleCancelReservation = (res: Reservation) => {
    cancelReservation(res.id)
    setConfirmCancel(null)
    toast.success(t('vermietung.cancel.success', { name: res.objectName }))
  }

  const handleCalendarCellClick = useCallback(
    (objectId: string, date: string) => {
      const existing = getReservationsForObjectAndDate(objectId, date)
      if (existing.length > 0) {
        const res = existing[0]
        toast.info(`${res.objectName}: ${res.renter}`, {
          description: `${formatDate(res.startDate)} – ${formatDate(res.endDate)}`,
        })
      } else {
        openReservationDialog(objectId, date)
      }
    },
    [getReservationsForObjectAndDate],
  )

  const getObjectActions = (obj: RentalObject) => [
    { label: t('vermietung.actions.showDetails'), icon: Eye, onClick: () => setSelectedObject(obj) },
    { label: t('common.edit'), icon: EditIcon, onClick: () => openObjectDialog(obj) },
    { label: t('vermietung.actions.reservieren'), icon: CalendarPlus, onClick: () => openReservationDialog(obj.id) },
    { separator: true as const, label: '', onClick: () => {} },
    { label: t('common.delete'), icon: Trash2, variant: 'destructive' as const, onClick: () => setConfirmDelete(obj) },
  ]

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <PageHeader
        title={t('vermietung.page.title')}
        description={t('vermietung.page.description', { count: objects.length, reservations: activeReservations })}
        icon={Building2}
        moduleId="vermietung"
        className="mb-6"
        actions={
          <div className="flex items-center gap-2">
            <button
              onClick={() => openReservationDialog()}
              className="flex items-center gap-2 rounded-xl border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <CalendarPlus className="h-4 w-4" />
              {t('vermietung.page.buttonReservierung')}
            </button>
            <button
              onClick={() => openObjectDialog()}
              className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-4 w-4" />
              {t('vermietung.page.buttonObjektAnlegen')}
            </button>
          </div>
        }
      />

      {/* ---- KPI Row ---- */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        {[
          {
            label: t('vermietung.kpi.available'),
            value: `${availableCount}`,
            icon: CheckCircle2,
            color: 'text-success',
            bg: 'bg-success-light',
          },
          {
            label: t('vermietung.kpi.reserved'),
            value: `${reservedCount}`,
            icon: Clock,
            color: 'text-info',
            bg: 'bg-info-light',
          },
          {
            label: t('vermietung.kpi.maintenance'),
            value: `${maintenanceCount}`,
            icon: AlertTriangle,
            color: 'text-warning',
            bg: 'bg-warning-light',
          },
          {
            label: t('vermietung.kpi.utilization'),
            value: `${utilization}%`,
            icon: BarChart3,
            color: utilization >= 70 ? 'text-success' : utilization >= 40 ? 'text-warning' : 'text-muted-foreground',
            bg: utilization >= 70 ? 'bg-success-light' : utilization >= 40 ? 'bg-warning-light' : 'bg-secondary',
          },
        ].map((stat) => {
          const Icon = stat.icon
          return (
            <div key={stat.label} className="rounded-xl border border-border bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
              <div className="flex items-center justify-between mb-2">
                <p className="text-xs text-muted-foreground">{stat.label}</p>
                <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${stat.bg}`}>
                  <Icon className={`h-4 w-4 ${stat.color}`} />
                </div>
              </div>
              <p className={`text-xl font-semibold ${stat.color}`}>{stat.value}</p>
            </div>
          )
        })}
      </div>

      {/* ---- Tabs ---- */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'objekte' as const, label: t('vermietung.tab.objekte', { count: objects.length }), icon: Package },
          { key: 'reservierungen' as const, label: t('vermietung.tab.reservierungen', { count: reservations.length }), icon: CalendarDays },
          { key: 'kalender' as const, label: t('vermietung.tab.kalender'), icon: CalendarDays },
        ]).map((tabItem) => {
          const Icon = tabItem.icon
          return (
            <button
              key={tabItem.key}
              onClick={() => { setTab(tabItem.key); setSearch('') }}
              className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors ${
                tab === tabItem.key
                  ? 'border-primary text-primary font-medium tab-accent-active'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Icon className="h-3.5 w-3.5" />
              {tabItem.label}
            </button>
          )
        })}
      </div>

      {/* ============================================================ */}
      {/* OBJEKTE TAB                                                   */}
      {/* ============================================================ */}
      {tab === 'objekte' && (
        <>
          {/* Search */}
          <div className="flex flex-wrap items-center gap-3 mb-4">
            <div className="relative flex-1 min-w-[200px] max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder={t('vermietung.objekte.search')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {filteredObjects.length === 0 ? (
            <EmptyState
              icon={Package}
              title={t('vermietung.objekte.empty.title')}
              description={search ? t('vermietung.objekte.empty.descFilter') : t('vermietung.objekte.empty.descEmpty')}
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {filteredObjects.map((obj) => {
                const typeCfg = OBJECT_TYPE_CONFIG[obj.type]
                const statusCfg = STATUS_CONFIG[obj.status]
                const TypeIcon = typeCfg.icon
                const nextRes = getNextReservation(obj.id)
                const activeRes = getActiveReservation(obj.id)
                const isSelected = selectedObject?.id === obj.id

                return (
                  <div
                    key={obj.id}
                    onClick={() => setSelectedObject(obj)}
                    className={`rounded-lg border bg-card p-4 transition-shadow cursor-pointer hover:shadow-[var(--shadow-card-hover)] ${
                      isSelected ? 'border-primary ring-1 ring-primary/20' : 'border-border'
                    }`}
                  >
                    {/* Top row: icon + type badge + actions */}
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex items-center gap-3">
                        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${typeCfg.bg}`}>
                          <TypeIcon className={`h-5 w-5 ${typeCfg.color}`} />
                        </div>
                        <div className="min-w-0">
                          <h4 className="text-sm font-semibold text-foreground truncate">{obj.name}</h4>
                          <div className="flex items-center gap-1.5 text-xs text-muted-foreground mt-0.5">
                            <MapPin className="h-3 w-3 flex-shrink-0" />
                            <span className="truncate">{obj.location}</span>
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${typeCfg.badgeBg}`}>
                          {t(typeCfg.labelKey)}
                        </span>
                        <ItemActions items={getObjectActions(obj)} />
                      </div>
                    </div>

                    {/* Status */}
                    <div className="flex items-center gap-2 mb-3">
                      <span className={`h-2 w-2 rounded-full ${statusCfg.dot}`} />
                      <span className={`text-xs font-medium ${statusCfg.text}`}>{t(statusCfg.labelKey)}</span>
                      {obj.status === 'reserved' && activeRes && (
                        <span className="text-xs text-muted-foreground">
                          &middot; {activeRes.renter} bis {formatDate(activeRes.endDate)}
                        </span>
                      )}
                    </div>

                    {/* Pricing */}
                    <div className="flex items-center gap-3 mb-2">
                      <span className="text-sm font-medium text-foreground tabular-nums">{formatCurrency(obj.dailyRate, obj.currency)}{t('vermietung.objekte.dailyRate')}</span>
                      {obj.weeklyRate && (
                        <span className="text-xs text-muted-foreground tabular-nums">{formatCurrency(obj.weeklyRate, obj.currency)}{t('vermietung.objekte.weeklyRate')}</span>
                      )}
                      {obj.deposit && (
                        <span className="text-[10px] text-muted-foreground tabular-nums">{t('vermietung.objekte.deposit', { amount: formatCurrency(obj.deposit, obj.currency) })}</span>
                      )}
                    </div>

                    {/* Serial number */}
                    {obj.serialNumber && (
                      <p className="text-[11px] text-muted-foreground font-mono mb-2">{obj.serialNumber}</p>
                    )}

                    {/* Next reservation or available */}
                    <div className="border-t border-border-muted pt-2 mt-1">
                      {nextRes ? (
                        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                          <CalendarDays className="h-3 w-3 flex-shrink-0" />
                          <span>
                            {t('vermietung.objekte.nextReservation', { start: formatDate(nextRes.startDate), end: formatDate(nextRes.endDate), renter: nextRes.renter })}
                          </span>
                        </div>
                      ) : (
                        <div className="flex items-center gap-1.5 text-xs text-success">
                          <CheckCircle2 className="h-3 w-3 flex-shrink-0" />
                          <span>{t('vermietung.objekte.noReservations')}</span>
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </>
      )}

      {/* ============================================================ */}
      {/* RESERVIERUNGEN TAB                                            */}
      {/* ============================================================ */}
      {tab === 'reservierungen' && (
        <>
          {/* Search + Filter */}
          <div className="flex flex-wrap items-center gap-3 mb-4">
            <div className="relative flex-1 min-w-[200px] max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder={t('vermietung.reservierungen.search')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="relative">
              <select
                value={reservationFilter}
                onChange={(e) => setReservationFilter(e.target.value as ReservationFilter)}
                className="appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                <option value="all">{t('vermietung.reservierungen.allStatus')}</option>
                <option value="active">{t('vermietung.reservationStatus.active')}</option>
                <option value="upcoming">{t('vermietung.reservationStatus.upcoming')}</option>
                <option value="completed">{t('vermietung.reservationStatus.completed')}</option>
              </select>
              <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            </div>
            {reservationFilter !== 'all' && (
              <button
                onClick={() => setReservationFilter('all')}
                className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <Filter className="h-3.5 w-3.5" />
                {t('common.resetFilters')}
              </button>
            )}
          </div>

          {filteredReservations.length === 0 ? (
            <EmptyState
              icon={CalendarDays}
              title={t('vermietung.reservierungen.empty.title')}
              description={search || reservationFilter !== 'all' ? t('vermietung.reservierungen.empty.descFilter') : t('vermietung.reservierungen.empty.descEmpty')}
            />
          ) : (
            <div className="overflow-x-auto rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-card">
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vermietung.reservierungen.table.objekt')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vermietung.reservierungen.table.mieter')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vermietung.reservierungen.table.zeitraum')}</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('vermietung.reservierungen.table.dauer')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vermietung.reservierungen.table.status')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vermietung.reservierungen.table.kaution')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vermietung.reservierungen.table.abholung')}</th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vermietung.reservierungen.table.rueckgabe')}</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground w-12"></th>
                  </tr>
                </thead>
                <tbody>
                  {filteredReservations.map((res) => {
                    const statusCfg = RESERVATION_STATUS_CONFIG[res.status]
                    const days = daysBetween(res.startDate, res.endDate)
                    return (
                      <tr
                        key={res.id}
                        className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors"
                      >
                        <td className="px-4 py-3 font-medium text-foreground">{res?.objectName ?? ''}</td>
                        <td className="px-4 py-3">
                          <div className="flex flex-col">
                            <span className="text-foreground">{res.renter}</span>
                            <span className="text-[11px] text-muted-foreground">
                              {res.renterType === 'employee' ? t('vermietung.reservierungen.mieterEmployee') : t('vermietung.reservierungen.mieterCustomer')}
                            </span>
                          </div>
                        </td>
                        <td className="px-4 py-3 text-muted-foreground whitespace-nowrap">
                          {formatDate(res.startDate)} – {formatDate(res.endDate)}
                        </td>
                        <td className="px-4 py-3 text-right text-foreground tabular-nums">
                          {days} {days === 1 ? t('vermietung.reservierungen.duration.day') : t('vermietung.reservierungen.duration.days')}
                        </td>
                        <td className="px-4 py-3">
                          <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${statusCfg.bg}`}>
                            {t(statusCfg.labelKey)}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          {res.depositStatus ? (
                            <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${DEPOSIT_STATUS_CONFIG[res.depositStatus].bg}`}>
                              {t(DEPOSIT_STATUS_CONFIG[res.depositStatus].labelKey)}
                              {res.depositAmount ? ` (${formatCurrency(res.depositAmount, res.currency)})` : ''}
                            </span>
                          ) : (
                            <span className="text-xs text-muted-foreground">–</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">{res.pickupLocation}</td>
                        <td className="px-4 py-3 text-muted-foreground">{res.returnLocation}</td>
                        <td className="px-4 py-3 text-right">
                          {(res.status === 'active' || res.status === 'upcoming') && (
                            <ItemActions
                              items={[
                                {
                                  label: t('vermietung.reservierungen.actions.zustandsprotokoll'),
                                  icon: ClipboardCheck,
                                  onClick: () => setZustandsprotokollReservation(res),
                                },
                                {
                                  label: t('vermietung.reservierungen.actions.stornieren'),
                                  icon: X,
                                  variant: 'destructive' as const,
                                  onClick: () => setConfirmCancel(res),
                                },
                              ]}
                            />
                          )}
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

      {/* ============================================================ */}
      {/* KALENDER TAB (Weekly Grid)                                    */}
      {/* ============================================================ */}
      {tab === 'kalender' && (
        <>
          {/* Week Navigation */}
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <button
                onClick={() => setWeekOffset((o) => o - 1)}
                className="flex h-8 w-8 items-center justify-center rounded-lg border border-border hover:bg-secondary transition-colors"
              >
                <ChevronLeft className="h-4 w-4 text-foreground" />
              </button>
              <div className="text-center min-w-[200px]">
                <span className="text-sm font-semibold text-foreground">{t('vermietung.kalender.kw', { kw })}</span>
                <span className="mx-2 text-muted-foreground">|</span>
                <span className="text-sm text-muted-foreground">{dateRange}</span>
              </div>
              <button
                onClick={() => setWeekOffset((o) => o + 1)}
                className="flex h-8 w-8 items-center justify-center rounded-lg border border-border hover:bg-secondary transition-colors"
              >
                <ChevronRight className="h-4 w-4 text-foreground" />
              </button>
              {weekOffset !== 0 && (
                <button
                  onClick={() => setWeekOffset(0)}
                  className="ml-1 rounded-lg border border-border px-3 py-1 text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                >
                  {t('vermietung.kalender.today')}
                </button>
              )}
            </div>
          </div>

          {/* Calendar Grid */}
          {objects.length === 0 ? (
            <EmptyState
              icon={CalendarDays}
              title={t('vermietung.kalender.empty.title')}
              description={t('vermietung.kalender.empty.desc')}
            />
          ) : (
            <div className="rounded-lg border border-border overflow-hidden">
              {/* Grid Header */}
              <div className="grid border-b border-border bg-card" style={{ gridTemplateColumns: '200px repeat(7, 1fr)' }}>
                <div className="flex items-center gap-2 px-4 py-3 border-r border-border">
                  <Package className="h-4 w-4 text-muted-foreground" />
                  <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.kalender.objectColumn')}</span>
                </div>
                {weekDates.map((date, i) => {
                  const today = isToday(date)
                  const dateObj = new Date(date + 'T00:00:00')
                  return (
                    <div
                      key={date}
                      className={`flex flex-col items-center justify-center py-3 ${i < 6 ? 'border-r border-border' : ''} ${
                        today ? 'bg-primary/5' : ''
                      }`}
                    >
                      <span className={`text-xs font-semibold ${today ? 'text-primary' : i >= 5 ? 'text-muted-foreground' : 'text-foreground'}`}>
                        {WEEKDAYS[i]}
                      </span>
                      <span className={`text-[11px] ${today ? 'text-primary font-medium' : 'text-muted-foreground'}`}>
                        {dateObj.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })}
                      </span>
                      {today && <div className="mt-1 h-0.5 w-4 rounded-full bg-primary" />}
                    </div>
                  )
                })}
              </div>

              {/* Grid Body — Object Rows */}
              {objects.map((obj, objIdx) => {
                const typeCfg = OBJECT_TYPE_CONFIG[obj.type]
                const TypeIcon = typeCfg.icon

                return (
                  <div
                    key={obj.id}
                    className={`grid ${objIdx < objects.length - 1 ? 'border-b border-border-muted' : ''}`}
                    style={{ gridTemplateColumns: '200px repeat(7, 1fr)' }}
                  >
                    {/* Object Name Cell */}
                    <div className="flex items-center gap-2.5 px-3 py-2.5 border-r border-border bg-card/50">
                      <div className={`flex h-7 w-7 items-center justify-center rounded-md ${typeCfg.bg} flex-shrink-0`}>
                        <TypeIcon className={`h-3.5 w-3.5 ${typeCfg.color}`} />
                      </div>
                      <div className="min-w-0">
                        <p className="text-xs font-medium text-foreground truncate">{obj.name}</p>
                        <p className="text-[10px] text-muted-foreground truncate">{obj.location}</p>
                      </div>
                    </div>

                    {/* Day Cells */}
                    {weekDates.map((date, dayIdx) => {
                      const cellKey = `${obj.id}-${date}`
                      const cellReservations = getReservationsForObjectAndDate(obj.id, date)
                      const isMaintenance = obj.status === 'maintenance'
                      const hasReservation = cellReservations.length > 0
                      const today = isToday(date)
                      const isHovered = hoveredCell === cellKey

                      // Check if this is the start of a reservation block
                      const startsHere = cellReservations.find((r) => r.startDate === date)
                      // Check if reservation continues from previous day
                      const _continuesFromPrev = hasReservation && !startsHere

                      return (
                        <div
                          key={date}
                          className={`relative flex items-center justify-center px-0.5 py-2 cursor-pointer transition-colors ${
                            dayIdx < 6 ? 'border-r border-border-muted' : ''
                          } ${today ? 'bg-primary/[0.02]' : ''} ${
                            !hasReservation && !isMaintenance && isHovered ? 'bg-secondary/60' : ''
                          }`}
                          onClick={() => handleCalendarCellClick(obj.id, date)}
                          onMouseEnter={() => setHoveredCell(cellKey)}
                          onMouseLeave={() => setHoveredCell(null)}
                        >
                          {isMaintenance ? (
                            <div className="w-full h-8 rounded-md bg-warning/10 border border-warning/20 flex items-center justify-center">
                              <AlertTriangle className="h-3 w-3 text-warning" />
                            </div>
                          ) : hasReservation ? (
                            <div
                              className={`w-full h-8 flex items-center px-1.5 transition-shadow ${
                                startsHere ? 'rounded-l-md' : ''
                              } ${
                                cellReservations.some((r) => r.endDate === date) ? 'rounded-r-md' : ''
                              } ${
                                isHovered ? 'shadow-sm ring-1 ring-info/30' : ''
                              } bg-info/10 border-y border-info/20 ${
                                startsHere ? 'border-l border-info/20' : ''
                              } ${
                                cellReservations.some((r) => r.endDate === date) ? 'border-r border-info/20' : ''
                              }`}
                            >
                              {startsHere && (
                                <span className="text-[10px] font-medium text-info truncate">
                                  {startsHere.renter}
                                </span>
                              )}
                            </div>
                          ) : (
                            <div
                              className={`w-full h-8 rounded-md border border-dashed transition-colors ${
                                isHovered
                                  ? 'border-primary/40 bg-primary/5'
                                  : 'border-transparent'
                              }`}
                            >
                              {isHovered && (
                                <div className="flex items-center justify-center h-full">
                                  <Plus className="h-3 w-3 text-primary/40" />
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      )
                    })}
                  </div>
                )
              })}

              {/* Legend */}
              <div className="flex items-center gap-5 px-4 py-2.5 border-t border-border bg-card/30">
                <span className="text-[11px] text-muted-foreground font-medium mr-1">{t('vermietung.kalender.legend.title')}</span>
                <div className="flex items-center gap-1.5">
                  <div className="h-4 w-8 rounded bg-info/10 border border-info/20" />
                  <span className="text-[11px] text-muted-foreground">{t('vermietung.kalender.legend.reserved')}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <div className="h-4 w-8 rounded bg-warning/10 border border-warning/20 flex items-center justify-center">
                    <AlertTriangle className="h-2.5 w-2.5 text-warning" />
                  </div>
                  <span className="text-[11px] text-muted-foreground">{t('vermietung.kalender.legend.maintenance')}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <div className="h-4 w-8 rounded border border-dashed border-border-muted" />
                  <span className="text-[11px] text-muted-foreground">{t('vermietung.kalender.legend.free')}</span>
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {/* ============================================================ */}
      {/* DETAIL PANEL                                                  */}
      {/* ============================================================ */}
      <DetailPanel
        open={!!selectedObject}
        onClose={() => setSelectedObject(null)}
        title={selectedObject?.name}
        subtitle={selectedObject ? `${OBJECT_TYPE_CONFIG[selectedObject.type].label} · ${selectedObject.location}` : undefined}
        badge={
          selectedObject ? (
            <span className={`ml-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
              STATUS_CONFIG[selectedObject.status].dot === 'bg-success'
                ? 'bg-success-light text-success'
                : STATUS_CONFIG[selectedObject.status].dot === 'bg-info'
                  ? 'bg-info-light text-info'
                  : 'bg-warning-light text-warning'
            }`}>
              {t(STATUS_CONFIG[selectedObject.status].labelKey)}
            </span>
          ) : undefined
        }
        width="w-[380px]"
        footer={
          selectedObject ? (
            <div className="flex gap-2">
              <button
                onClick={() => openObjectDialog(selectedObject)}
                className="flex-1 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('common.edit')}
              </button>
              <button
                onClick={() => {
                  openReservationDialog(selectedObject.id)
                  setSelectedObject(null)
                }}
                className="flex-1 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                {t('vermietung.detail.buttonReservieren')}
              </button>
            </div>
          ) : undefined
        }
      >
        {selectedObject && (
          <div className="space-y-5">
            {/* Basic info */}
            <div className="space-y-3">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.detail.details')}</h4>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <p className="text-xs text-muted-foreground">{t('vermietung.detail.fieldName')}</p>
                  <p className="text-sm text-foreground font-medium">{selectedObject.name}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('vermietung.detail.fieldTyp')}</p>
                  <p className="text-sm text-foreground">{t(OBJECT_TYPE_CONFIG[selectedObject.type].labelKey)}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('vermietung.detail.fieldStandort')}</p>
                  <p className="text-sm text-foreground">{selectedObject.location}</p>
                </div>
                {selectedObject.serialNumber && (
                  <div>
                    <p className="text-xs text-muted-foreground">{t('vermietung.detail.fieldSerial')}</p>
                    <p className="text-sm text-foreground font-mono">{selectedObject.serialNumber}</p>
                  </div>
                )}
              </div>
              {selectedObject.description && (
                <div>
                  <p className="text-xs text-muted-foreground">{t('vermietung.detail.fieldDescription')}</p>
                  <p className="text-sm text-foreground">{selectedObject.description}</p>
                </div>
              )}
            </div>

            {/* Pricing */}
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.detail.preise')}</h4>
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <div className="flex items-baseline justify-between">
                  <div>
                    <span className="text-lg font-semibold text-foreground tabular-nums">{formatCurrency(selectedObject.dailyRate, selectedObject.currency)}</span>
                    <span className="text-xs text-muted-foreground ml-1">{t('vermietung.detail.perDay')}</span>
                  </div>
                  {selectedObject.weeklyRate && (
                    <div>
                      <span className="text-sm font-medium text-muted-foreground tabular-nums">{formatCurrency(selectedObject.weeklyRate, selectedObject.currency)}</span>
                      <span className="text-xs text-muted-foreground ml-1">{t('vermietung.detail.perWeek')}</span>
                    </div>
                  )}
                </div>
                {selectedObject.deposit && (
                  <div className="mt-2 pt-2 border-t border-border-muted flex items-baseline justify-between">
                    <span className="text-xs text-muted-foreground">{t('vermietung.detail.kaution')}</span>
                    <span className="text-sm font-medium text-foreground tabular-nums">
                      {formatCurrency(selectedObject.deposit, selectedObject.currency)}
                    </span>
                  </div>
                )}
              </div>
            </div>

            {/* Current status */}
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.detail.currentStatus')}</h4>
              <div className="rounded-lg border border-border p-3">
                <div className="flex items-center gap-2">
                  <span className={`h-2.5 w-2.5 rounded-full ${STATUS_CONFIG[selectedObject.status].dot}`} />
                  <span className={`text-sm font-medium ${STATUS_CONFIG[selectedObject.status].text}`}>
                    {t(STATUS_CONFIG[selectedObject.status].labelKey)}
                  </span>
                </div>
                {(() => {
                  const activeRes = getActiveReservation(selectedObject.id)
                  if (activeRes) {
                    return (
                      <div className="mt-2 pt-2 border-t border-border-muted">
                        <p className="text-xs text-muted-foreground">{t('vermietung.detail.activeReservation')}</p>
                        <p className="text-sm text-foreground font-medium">{activeRes.renter}</p>
                        <p className="text-xs text-muted-foreground">
                          {formatDate(activeRes.startDate)} – {formatDate(activeRes.endDate)}
                        </p>
                      </div>
                    )
                  }
                  return null
                })()}
              </div>
            </div>

            {/* Last 5 reservations */}
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.detail.lastReservations')}</h4>
              {(() => {
                const objectReservations = getObjectReservations(selectedObject.id)
                if (objectReservations.length === 0) {
                  return <p className="text-xs text-muted-foreground py-2">{t('vermietung.detail.noReservations')}</p>
                }
                return (
                  <div className="space-y-1">
                    {objectReservations.map((res) => {
                      const statusCfg = RESERVATION_STATUS_CONFIG[res.status]
                      const hasProtokoll = hasZustandsprotokoll(res.id)
                      return (
                        <div key={res.id} className="flex items-center gap-2 rounded-md border border-border-muted p-2">
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-1.5">
                              <p className="text-xs font-medium text-foreground">{res.renter}</p>
                              {hasProtokoll && (
                                <span title="Zustandsprotokoll vorhanden" className="flex items-center">
                                  <ClipboardCheck className="h-3 w-3 text-success" />
                                </span>
                              )}
                            </div>
                            <p className="text-[10px] text-muted-foreground">
                              {formatDate(res.startDate)} – {formatDate(res.endDate)}
                            </p>
                          </div>
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium whitespace-nowrap ${statusCfg.bg}`}>
                            {t(statusCfg.labelKey)}
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

      {/* ============================================================ */}
      {/* DIALOGS                                                       */}
      {/* ============================================================ */}
      <ObjectDialog
        open={objectDialogOpen}
        onClose={() => {
          setObjectDialogOpen(false)
          setEditObject(null)
        }}
        initial={editObject}
      />

      <ReservationDialog
        open={reservationDialogOpen}
        onClose={() => {
          setReservationDialogOpen(false)
          setPreSelectedObjectId(undefined)
          setPreSelectedDate(undefined)
        }}
        objects={objects}
        preSelectedObjectId={preSelectedObjectId}
        preSelectedDate={preSelectedDate}
      />

      {/* Confirm Delete Object */}
      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title={t('vermietung.confirm.deleteTitle')}
        description={t('vermietung.confirm.deleteDesc', { name: confirmDelete?.name ?? '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={() => confirmDelete && handleDeleteObject(confirmDelete)}
      />

      {/* Confirm Cancel Reservation */}
      <ConfirmDialog
        open={!!confirmCancel}
        onOpenChange={() => setConfirmCancel(null)}
        title={t('vermietung.confirm.cancelTitle')}
        description={t('vermietung.confirm.cancelDesc', { renter: confirmCancel?.renter ?? '', object: confirmCancel?.objectName ?? '' })}
        confirmLabel={t('vermietung.confirm.cancelLabel')}
        variant="destructive"
        onConfirm={() => confirmCancel && handleCancelReservation(confirmCancel)}
      />

      {/* Zustandsprotokoll Dialog (9.14) */}
      <ZustandsprotokollDialog
        open={!!zustandsprotokollReservation}
        onClose={() => setZustandsprotokollReservation(null)}
        reservation={zustandsprotokollReservation}
        object={zustandsprotokollReservation ? objects.find((o) => o.id === zustandsprotokollReservation.objectId) ?? null : null}
      />
    </div>
  )
}
