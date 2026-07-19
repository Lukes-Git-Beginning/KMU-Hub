import { useState, useMemo, useCallback, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Plus,
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
  Download,
} from 'lucide-react'
import { useCapabilitySet, useHasCapability } from '@/hooks/useCapability'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'
import { toast } from 'sonner'
import {
  useObjectsList,
  useRentalsList,
  useCreateObject,
  useUpdateObject,
  useDeleteObject,
  useCreateRental,
  useDeleteRental,
} from '@/api/hooks/useVermietung'
import type { RentalObject, Rental } from '@/api/vermietung-types'
import { useVermietungPrefsStore } from '@/stores/vermietungPrefs'
import { useVermietungViewPrefsStore } from '@/stores/vermietungViewPrefs'
import { useVermietungTenantStore, VERMIETUNG_CURRENCY_OPTIONS } from '@/stores/vermietungTenant'
import { ItemActions, ConfirmDialog, EmptyState, PageHeader, SortMenu, type SortDirection, type SortFieldOption } from '@/components/shared'
import { formatCurrency } from '@/lib/format'
import ZustandsprotokollDialog from './ZustandsprotokollDialog'
import { ObjectDetailModal } from './ObjectDetailModal'
import { RentalDetailModal } from './RentalDetailModal'
import {
  WEEKDAYS,
  OBJECT_TYPE_CONFIG,
  getTypeCfg,
  STATUS_CONFIG,
  RESERVATION_STATUS_CONFIG,
  DEPOSIT_STATUS_CONFIG,
  computeObjectStatus,
  computeDepositStatus,
  isOverdue,
  getWeekDates,
  getKW,
  formatDateRange,
  formatDate,
  isToday,
  daysBetween,
  dateInRange,
  shiftDate,
  computeRentalPrice,
} from './vermietung-shared'
import { buildObjectsCsv, buildRentalsCsv, downloadCsv, csvDateStamp } from './vermietung-export'

// ============================================================
// Types
// ============================================================

type TabKey = 'objekte' | 'reservierungen' | 'kalender'
type ReservationFilter = 'all' | 'active' | 'reserved' | 'completed'

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
  const createObjectMut = useCreateObject()
  const updateObjectMut = useUpdateObject()
  const { objectPrefs, setObjectPref } = useVermietungPrefsStore()
  // Tenant default seeds the currency for new objects (settings panel).
  const defaultCurrency = useVermietungTenantStore((s) => s.defaultCurrency)
  const isEdit = !!initial

  const [name, setName] = useState(initial?.name ?? '')
  const [type, setType] = useState<string>(initial?.category ?? 'gerät')
  const [location, setLocation] = useState(initial?.location ?? '')
  const [dailyRate, setDailyRate] = useState(initial?.daily_rate?.toString() ?? '')
  const [weeklyRate, setWeeklyRate] = useState(
    initial ? (objectPrefs[initial.id]?.weeklyRate?.toString() ?? '') : '',
  )
  const [currency, setCurrency] = useState(
    initial ? (objectPrefs[initial.id]?.currency ?? 'EUR') : defaultCurrency,
  )
  const [deposit, setDeposit] = useState(initial?.deposit?.toString() ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [serialNumber, setSerialNumber] = useState(
    initial ? (objectPrefs[initial.id]?.serialNumber ?? '') : '',
  )

  const handleSave = () => {
    if (!name.trim()) { toast.error(t('vermietung.objectDialog.errorName')); return }
    if (!location.trim()) { toast.error(t('vermietung.objectDialog.errorStandort')); return }
    if (!dailyRate || Number(dailyRate) <= 0) { toast.error(t('vermietung.objectDialog.errorDailyRate')); return }

    const input = {
      name: name.trim(),
      description: description.trim() || undefined,
      category: type,
      daily_rate: Number(dailyRate),
      deposit: deposit ? Number(deposit) : 0,
      location: location.trim() || undefined,
    }

    if (isEdit && initial) {
      updateObjectMut.mutate({ id: initial.id, ...input }, {
        onSuccess: () => {
          setObjectPref(initial.id, {
            weeklyRate: weeklyRate ? Number(weeklyRate) : undefined,
            currency,
            serialNumber: serialNumber.trim() || undefined,
          })
          toast.success(t('vermietung.objectDialog.updateSuccess', { name: name.trim() }))
          onClose()
        },
        onError: () => toast.error('Fehler beim Speichern'),
      })
    } else {
      createObjectMut.mutate(input, {
        onSuccess: (data) => {
          if (data?.object?.id) {
            setObjectPref(data.object.id, {
              weeklyRate: weeklyRate ? Number(weeklyRate) : undefined,
              currency,
              serialNumber: serialNumber.trim() || undefined,
            })
          }
          toast.success(t('vermietung.objectDialog.createSuccess', { name: name.trim() }))
          onClose()
        },
        onError: () => toast.error('Fehler beim Erstellen'),
      })
    }
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated max-h-[90vh] overflow-y-auto">
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
              {(Object.entries(OBJECT_TYPE_CONFIG) as [string, typeof OBJECT_TYPE_CONFIG['gerät']][]).map(([key, cfg]) => {
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
              {VERMIETUNG_CURRENCY_OPTIONS.map((cur) => (
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
  rentals,
  preSelectedObjectId,
  preSelectedDate,
}: {
  open: boolean
  onClose: () => void
  objects: RentalObject[]
  rentals: Rental[]
  preSelectedObjectId?: string
  preSelectedDate?: string
}) {
  const { t } = useTranslation()
  const createRentalMut = useCreateRental()
  const { objectPrefs, setRentalPref } = useVermietungPrefsStore()
  // Tenant policies: preparation buffer between rentals + deposit requirement.
  const bufferDays = useVermietungTenantStore((s) => s.bufferDays)
  const requireDeposit = useVermietungTenantStore((s) => s.requireDeposit)

  const [objectId, setObjectId] = useState(preSelectedObjectId ?? '')
  const [startDate, setStartDate] = useState(preSelectedDate ?? '')
  const [endDate, setEndDate] = useState(preSelectedDate ?? '')
  const [renter, setRenter] = useState('')
  const [renterType, setRenterType] = useState<'employee' | 'customer'>('employee')
  const [pickupLocation, setPickupLocation] = useState('')
  const [returnLocation, setReturnLocation] = useState('')
  const [notes, setNotes] = useState('')

  const selectedObj = objects.find((o) => o.id === objectId)
  const objCurrency = objectPrefs[objectId]?.currency ?? 'EUR'
  const objWeeklyRate = objectPrefs[objectId]?.weeklyRate

  // Auto-fill locations when object changes
  const handleObjectChange = (id: string) => {
    setObjectId(id)
    const obj = objects.find((o) => o.id === id)
    if (obj) {
      setPickupLocation(obj.location ?? '')
      setReturnLocation(obj.location ?? '')
    }
  }

  // Availability: conflicts including the tenant preparation buffer.
  const objRentals = rentals.filter((r) => r.object_id === objectId && (r.status === 'active' || r.status === 'reserved'))
  const hasConflict =
    !!startDate &&
    !!endDate &&
    startDate <= endDate &&
    objRentals.some(
      (r) =>
        !(shiftDate(r.end_date.slice(0, 10), bufferDays) < startDate ||
          shiftDate(r.start_date.slice(0, 10), -bufferDays) > endDate),
    )

  const handleSave = () => {
    if (!objectId) { toast.error(t('vermietung.reservationDialog.errorObjekt')); return }
    if (!startDate || !endDate) { toast.error(t('vermietung.reservationDialog.errorDates')); return }
    if (startDate > endDate) { toast.error(t('vermietung.reservationDialog.errorDateOrder')); return }
    if (!renter.trim()) { toast.error(t('vermietung.reservationDialog.errorMieter')); return }
    if (hasConflict) { toast.error(t('vermietung.reservationDialog.errorConflict')); return }

    const days = daysBetween(startDate, endDate)
    const pricing = selectedObj
      ? computeRentalPrice(days, selectedObj.daily_rate, objWeeklyRate)
      : null

    createRentalMut.mutate(
      {
        object_id: objectId,
        renter_name: renter.trim(),
        start_date: new Date(startDate + 'T00:00:00').toISOString(),
        end_date: new Date(endDate + 'T23:59:59').toISOString(),
        total_price: pricing?.total ?? 0,
        deposit_paid: false,
        notes: notes.trim() || undefined,
      },
      {
        onSuccess: (data) => {
          const rentalId = data?.rental?.id
          if (rentalId) {
            setRentalPref(rentalId, {
              renterType,
              pickupLocation: pickupLocation.trim() || undefined,
              returnLocation: returnLocation.trim() || undefined,
              currency: objCurrency,
            })
          }
          toast.success(
            t('vermietung.reservationDialog.createSuccess', { name: selectedObj?.name ?? '' }),
            { description: `${formatDate(startDate)} – ${formatDate(endDate)} (${renter.trim()})` },
          )
          onClose()
        },
        onError: () => toast.error('Fehler beim Erstellen'),
      },
    )
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated max-h-[90vh] overflow-y-auto">
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
                  {obj.name} ({t(getTypeCfg(obj.category).labelKey)})
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

          {/* Availability conflict (respects tenant buffer) */}
          {hasConflict && (
            <div className="flex items-start gap-2 rounded-lg border border-error/30 bg-error-light p-3">
              <AlertTriangle className="h-4 w-4 text-error mt-0.5 shrink-0" />
              <div className="min-w-0">
                <p className="text-sm font-medium text-error">{t('vermietung.reservationDialog.conflictTitle')}</p>
                <p className="text-xs text-error/80 mt-0.5">
                  {bufferDays > 0
                    ? t('vermietung.reservationDialog.conflictDescBuffer', { days: bufferDays })
                    : t('vermietung.reservationDialog.conflictDesc')}
                </p>
              </div>
            </div>
          )}

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

          {/* Price Calculation */}
          {selectedObj && startDate && endDate && startDate <= endDate && (
            <div className="rounded-lg border border-border bg-secondary/30 p-3 space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.reservationDialog.priceCalc')}</h4>
              {(() => {
                const days = daysBetween(startDate, endDate)
                const pricing = computeRentalPrice(days, selectedObj.daily_rate, objWeeklyRate)

                return (
                  <div className="space-y-1">
                    {pricing.weeks > 0 ? (
                      <>
                        <p className="text-xs text-muted-foreground tabular-nums">
                          {pricing.weeks} Woche{pricing.weeks > 1 ? 'n' : ''} &times; {formatCurrency(objWeeklyRate!, objCurrency)}/Woche
                          = {formatCurrency(pricing.weeks * objWeeklyRate!, objCurrency)}
                        </p>
                        {pricing.remainingDays > 0 && (
                          <p className="text-xs text-muted-foreground tabular-nums">
                            + {pricing.remainingDays} Tag{pricing.remainingDays > 1 ? 'e' : ''} &times; {formatCurrency(selectedObj.daily_rate, objCurrency)}/Tag
                            = {formatCurrency(pricing.remainingDays * selectedObj.daily_rate, objCurrency)}
                          </p>
                        )}
                      </>
                    ) : (
                      <p className="text-xs text-muted-foreground tabular-nums">
                        {days} Tag{days > 1 ? 'e' : ''} &times; {formatCurrency(selectedObj.daily_rate, objCurrency)}/Tag
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

          {/* Deposit Notice */}
          {selectedObj?.deposit ? (
            <div className="rounded-lg border border-amber-200 dark:border-amber-800/50 bg-amber-50 dark:bg-amber-900/20 p-3">
              <p className="text-xs font-medium text-amber-700 dark:text-amber-400">
                {t('vermietung.reservationDialog.depositNotice', { amount: formatCurrency(selectedObj.deposit, objCurrency) })}
                {requireDeposit ? ` ${t('vermietung.reservationDialog.depositRequired')}` : ''}
              </p>
            </div>
          ) : null}
        </div>

        <div className="flex justify-end gap-2 mt-6">
          <button onClick={onClose} className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors">
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            disabled={hasConflict}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
          >
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

  // Data
  const objectsQuery = useObjectsList({ page: 1, page_size: 200 })
  const rentalsQuery = useRentalsList({ page: 1, page_size: 500 })
  const objects: RentalObject[] = objectsQuery.data?.objects ?? []
  const rentals: Rental[] = rentalsQuery.data?.rentals ?? []

  const deleteObjectMut = useDeleteObject()
  const deleteRentalMut = useDeleteRental()
  const { objectPrefs, rentalPrefs } = useVermietungPrefsStore()
  // Personal view prefs (settings panel)
  const showKpis = useVermietungViewPrefsStore((s) => s.showKpis)

  // State
  const [tab, setTab] = useState<TabKey>(() => useVermietungViewPrefsStore.getState().defaultTab)
  const [search, setSearch] = useState('')
  const [weekOffset, setWeekOffset] = useState(0)
  const [reservationFilter, setReservationFilter] = useState<ReservationFilter>('all')
  const [hoveredCell, setHoveredCell] = useState<string | null>(null)
  const [rentalSortField, setRentalSortField] = useState('start')
  const [rentalSortDir, setRentalSortDir] = useState<SortDirection>('desc')

  // Detail modals + back-chain (object → rental → back)
  const [selectedObject, setSelectedObject] = useState<RentalObject | null>(null)
  const [selectedRental, setSelectedRental] = useState<Rental | null>(null)
  const [rentalBackObject, setRentalBackObject] = useState<RentalObject | null>(null)

  // Dialogs
  const [objectDialogOpen, setObjectDialogOpen] = useState(false)
  const [editObject, setEditObject] = useState<RentalObject | null>(null)
  const [reservationDialogOpen, setReservationDialogOpen] = useState(false)
  const [preSelectedObjectId, setPreSelectedObjectId] = useState<string | undefined>()
  const [preSelectedDate, setPreSelectedDate] = useState<string | undefined>()
  const [confirmDelete, setConfirmDelete] = useState<RentalObject | null>(null)
  const [confirmCancel, setConfirmCancel] = useState<Rental | null>(null)
  const [zustandsprotokollReservation, setZustandsprotokollReservation] = useState<Rental | null>(null)

  // Keep the rental modal in sync after mutations (start/end/deposit refetch the list).
  const liveSelectedRental = selectedRental
    ? (rentals.find((r) => r.id === selectedRental.id) ?? selectedRental)
    : null

  // ── RBAC R-3 capability checks (default hidden per gating convention) ──
  const { has: capHas, ready: capReady } = useCapabilitySet()
  const canCreateObject = useHasCapability('vermietung:object:create')
  const canEditObject = useHasCapability('vermietung:object:edit')
  const canDeleteObject = useHasCapability('vermietung:object:delete')
  const canCreateRental = useHasCapability('vermietung:rental:create')
  const canCancelRental = useHasCapability('vermietung:rental:cancel')
  const canCreateInspection = useHasCapability('vermietung:inspection:create')
  const canExportVermietung = useHasCapability('vermietung:export:run')

  // Tab capability map (objekte→object:read, reservierungen+kalender→rental:read)
  const TAB_CAPABILITY: Record<TabKey, string> = {
    objekte: 'vermietung:object:read',
    reservierungen: 'vermietung:rental:read',
    kalender: 'vermietung:rental:read',
  }
  const visibleTabs = (Object.keys(TAB_CAPABILITY) as TabKey[]).filter(
    (key) => !capReady || capHas(TAB_CAPABILITY[key]),
  )

  // Deduplicate: kalender shares the same key as reservierungen — filter keeps both only once per entry;
  // if rental:read is denied both kalender and reservierungen are hidden together.
  useEffect(() => {
    if (!capReady) return
    if (visibleTabs.length > 0 && !visibleTabs.includes(tab)) setTab(visibleTabs[0])
  }, [capReady, visibleTabs, tab])

  // All hooks before any early returns
  const filteredObjects = useMemo(() => {
    if (!search) return objects
    const q = search.toLowerCase()
    return objects.filter(
      (o) =>
        o.name.toLowerCase().includes(q) ||
        (o.location ?? '').toLowerCase().includes(q),
    )
  }, [objects, search])

  const filteredRentals = useMemo(() => {
    const dir = rentalSortDir === 'asc' ? 1 : -1
    let result = [...rentals].sort((a, b) => {
      switch (rentalSortField) {
        case 'objekt': {
          const an = objects.find((o) => o.id === a.object_id)?.name ?? ''
          const bn = objects.find((o) => o.id === b.object_id)?.name ?? ''
          return an.localeCompare(bn) * dir
        }
        case 'mieter':
          return a.renter_name.localeCompare(b.renter_name) * dir
        case 'dauer':
          return (
            (daysBetween(a.start_date.slice(0, 10), a.end_date.slice(0, 10)) -
              daysBetween(b.start_date.slice(0, 10), b.end_date.slice(0, 10))) * dir
          )
        case 'status':
          return a.status.localeCompare(b.status) * dir
        default:
          return a.start_date.localeCompare(b.start_date) * dir
      }
    })

    if (reservationFilter !== 'all') {
      result = result.filter((r) => r.status === reservationFilter)
    }

    if (search) {
      const q = search.toLowerCase()
      result = result.filter((r) => {
        const objName = objects.find((o) => o.id === r.object_id)?.name ?? ''
        return (
          objName.toLowerCase().includes(q) ||
          r.renter_name.toLowerCase().includes(q) ||
          (r.notes ?? '').toLowerCase().includes(q)
        )
      })
    }

    return result
  }, [rentals, reservationFilter, search, objects, rentalSortField, rentalSortDir])

  const getRentalsForObjectAndDate = useCallback(
    (objectId: string, date: string) => {
      return rentals.filter(
        (r) =>
          r.object_id === objectId &&
          (r.status === 'active' || r.status === 'reserved') &&
          dateInRange(date, r.start_date.slice(0, 10), r.end_date.slice(0, 10)),
      )
    },
    [rentals],
  )

  const getObjectRentals = useCallback(
    (objectId: string) =>
      rentals
        .filter((r) => r.object_id === objectId)
        .sort((a, b) => b.start_date.localeCompare(a.start_date))
        .slice(0, 5),
    [rentals],
  )

  const getNextRental = useCallback(
    (objectId: string) => {
      const today = new Date().toISOString().slice(0, 10)
      return rentals
        .filter(
          (r) =>
            r.object_id === objectId &&
            (r.status === 'active' || r.status === 'reserved') &&
            r.end_date.slice(0, 10) >= today,
        )
        .sort((a, b) => a.start_date.localeCompare(b.start_date))[0]
    },
    [rentals],
  )

  const getActiveRental = useCallback(
    (objectId: string) => {
      const today = new Date().toISOString().slice(0, 10)
      return rentals.find(
        (r) =>
          r.object_id === objectId &&
          r.status === 'active' &&
          dateInRange(today, r.start_date.slice(0, 10), r.end_date.slice(0, 10)),
      )
    },
    [rentals],
  )

  // Loading / Error states (after all hooks)
  if (objectsQuery.isLoading || rentalsQuery.isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center p-6">
        <div className="text-sm text-muted-foreground">{t('vermietung.loading')}</div>
      </div>
    )
  }
  if (objectsQuery.isError) {
    return (
      <div className="flex-1 p-6">
        <div className="rounded-lg border border-error/30 bg-error/10 p-4 text-sm text-error">{t('vermietung.errorBanner')}</div>
      </div>
    )
  }

  // moduleEmpty: no readable tab after permissions load
  if (capReady && visibleTabs.length === 0) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <PageHeader
          title={t('vermietung.page.title')}
          description={t('rbac.gate.moduleEmpty')}
          icon={Building2}
          moduleId="vermietung"
          className="mb-6"
          actions={<RestrictedModeBadge module="vermietung" />}
        />
        <EmptyState icon={Package} title={t('rbac.gate.moduleEmpty')} description={t('rbac.gate.noPermission')} />
      </div>
    )
  }

  // Derived data
  const weekDates = getWeekDates(weekOffset)
  const kw = getKW(weekDates[0])
  const dateRange = formatDateRange(weekDates)

  const availableCount = objects.filter((o) => computeObjectStatus(o, rentals) === 'available').length
  const reservedCount = objects.filter((o) => computeObjectStatus(o, rentals) === 'reserved').length
  const maintenanceCount = objects.filter((o) => computeObjectStatus(o, rentals) === 'maintenance').length
  const activeReservations = rentals.filter((r) => r.status === 'active' || r.status === 'reserved').length
  const utilization = objects.length > 0 ? Math.round(((reservedCount + maintenanceCount) / objects.length) * 100) : 0
  const objectsById = new Map(objects.map((o) => [o.id, o]))

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

  const openRentalDetail = (rental: Rental, backToObject?: RentalObject | null) => {
    setRentalBackObject(backToObject ?? null)
    setSelectedObject(null)
    setSelectedRental(rental)
  }

  const handleDeleteObject = (obj: RentalObject) => {
    deleteObjectMut.mutate(obj.id, {
      onSuccess: () => {
        if (selectedObject?.id === obj.id) setSelectedObject(null)
        toast.success(t('vermietung.delete.success', { name: obj.name }))
        setConfirmDelete(null)
      },
      onError: () => toast.error('Fehler beim Löschen'),
    })
  }

  const handleCancelReservation = (res: Rental) => {
    deleteRentalMut.mutate(res.id, {
      onSuccess: () => {
        const objName = objects.find((o) => o.id === res.object_id)?.name ?? res.id
        toast.success(t('vermietung.cancel.success', { name: objName }))
        setConfirmCancel(null)
        if (selectedRental?.id === res.id) setSelectedRental(null)
      },
      onError: () => toast.error('Fehler beim Stornieren'),
    })
  }

  const handleCalendarCellClick = (objectId: string, date: string) => {
    const existing = getRentalsForObjectAndDate(objectId, date)
    if (existing.length > 0) {
      // Belegter Slot → Reservierungs-Detailfenster (Detail öffnen ist immer erlaubt)
      openRentalDetail(existing[0])
    } else if (canCreateRental) {
      // Freier Slot → Neue Reservierung (nur mit rental:create)
      openReservationDialog(objectId, date)
    }
    // ohne Recht: Klick auf freie Zelle ignoriert
  }

  const handleExportObjects = () => {
    downloadCsv(buildObjectsCsv(filteredObjects), `vermietung-objekte-${csvDateStamp()}.csv`)
    toast.success(t('vermietung.export.objectsSuccess', { count: filteredObjects.length }))
  }

  const handleExportRentals = () => {
    downloadCsv(buildRentalsCsv(filteredRentals, objectsById), `vermietung-reservierungen-${csvDateStamp()}.csv`)
    toast.success(t('vermietung.export.rentalsSuccess', { count: filteredRentals.length }))
  }

  const getObjectActions = (obj: RentalObject) => [
    { label: t('vermietung.actions.showDetails'), icon: Eye, onClick: () => setSelectedObject(obj) },
    ...(canEditObject ? [{ label: t('common.edit'), icon: EditIcon, onClick: () => openObjectDialog(obj) }] : []),
    ...(canCreateRental ? [{ label: t('vermietung.actions.reservieren'), icon: CalendarPlus, onClick: () => openReservationDialog(obj.id) }] : []),
    ...(canDeleteObject ? [
      { separator: true as const, label: '', onClick: () => {} },
      { label: t('common.delete'), icon: Trash2, variant: 'destructive' as const, onClick: () => setConfirmDelete(obj) },
    ] : []),
  ]

  const rentalSortOptions: SortFieldOption[] = [
    { value: 'start', label: t('vermietung.sort.start') },
    { value: 'objekt', label: t('vermietung.sort.objekt') },
    { value: 'mieter', label: t('vermietung.sort.mieter') },
    { value: 'dauer', label: t('vermietung.sort.dauer') },
    { value: 'status', label: t('vermietung.sort.status') },
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
            <RestrictedModeBadge module="vermietung" />
            {canCreateRental && (
              <button
                onClick={() => openReservationDialog()}
                className="flex items-center gap-2 rounded-xl border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
              >
                <CalendarPlus className="h-4 w-4" />
                {t('vermietung.page.buttonReservierung')}
              </button>
            )}
            {canCreateObject && (
              <button
                onClick={() => openObjectDialog()}
                className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Plus className="h-4 w-4" />
                {t('vermietung.page.buttonObjektAnlegen')}
              </button>
            )}
          </div>
        }
      />

      {/* ---- KPI Row (personal pref) ---- */}
      {showKpis && (
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
      )}

      {/* ---- Tabs ---- */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'objekte' as const, label: t('vermietung.tab.objekte', { count: objects.length }), icon: Package },
          { key: 'reservierungen' as const, label: t('vermietung.tab.reservierungen', { count: rentals.length }), icon: CalendarDays },
          { key: 'kalender' as const, label: t('vermietung.tab.kalender'), icon: CalendarDays },
        ]).filter((tabItem) => visibleTabs.includes(tabItem.key)).map((tabItem) => {
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
          {/* Search + Export */}
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
            {canExportVermietung && (
              <button
                onClick={handleExportObjects}
                disabled={filteredObjects.length === 0}
                className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors disabled:opacity-50"
              >
                <Download className="h-4 w-4" />
                <span className="hidden sm:inline">{t('vermietung.export.button')}</span>
              </button>
            )}
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
                const typeCfg = getTypeCfg(obj.category)
                const objStatus = computeObjectStatus(obj, rentals)
                const statusCfg = STATUS_CONFIG[objStatus]
                const TypeIcon = typeCfg.icon
                const nextRental = getNextRental(obj.id)
                const activeRental = getActiveRental(obj.id)
                const isSelected = selectedObject?.id === obj.id
                const objWeeklyRate = objectPrefs[obj.id]?.weeklyRate
                const objCurrency = objectPrefs[obj.id]?.currency ?? 'EUR'
                const objSerial = objectPrefs[obj.id]?.serialNumber

                return (
                  <div
                    key={obj.id}
                    role="button"
                    tabIndex={0}
                    aria-label={obj.name}
                    onClick={() => setSelectedObject(obj)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        setSelectedObject(obj)
                      }
                    }}
                    className={`rounded-lg border bg-card p-4 transition-shadow cursor-pointer hover:shadow-[var(--shadow-card-hover)] focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring ${
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
                            <span className="truncate">{obj.location ?? ''}</span>
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
                      {objStatus === 'reserved' && activeRental && (
                        <span className="text-xs text-muted-foreground">
                          &middot; {activeRental.renter_name} bis {formatDate(activeRental.end_date)}
                        </span>
                      )}
                    </div>

                    {/* Pricing */}
                    <div className="flex items-center gap-3 mb-2">
                      <span className="text-sm font-medium text-foreground tabular-nums">{formatCurrency(obj.daily_rate, objCurrency)}{t('vermietung.objekte.dailyRate')}</span>
                      {objWeeklyRate && (
                        <span className="text-xs text-muted-foreground tabular-nums">{formatCurrency(objWeeklyRate, objCurrency)}{t('vermietung.objekte.weeklyRate')}</span>
                      )}
                      {obj.deposit > 0 && (
                        <span className="text-[10px] text-muted-foreground tabular-nums">{t('vermietung.objekte.deposit', { amount: formatCurrency(obj.deposit, objCurrency) })}</span>
                      )}
                    </div>

                    {/* Serial number */}
                    {objSerial && (
                      <p className="text-[11px] text-muted-foreground font-mono mb-2">{objSerial}</p>
                    )}

                    {/* Next rental or available */}
                    <div className="border-t border-border-muted pt-2 mt-1">
                      {nextRental ? (
                        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                          <CalendarDays className="h-3 w-3 flex-shrink-0" />
                          <span>
                            {t('vermietung.objekte.nextReservation', { start: formatDate(nextRental.start_date), end: formatDate(nextRental.end_date), renter: nextRental.renter_name })}
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
          {/* Search + Filter + Sort + Export */}
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
                <option value="reserved">{t('vermietung.reservationStatus.upcoming')}</option>
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
            <SortMenu
              options={rentalSortOptions}
              field={rentalSortField}
              direction={rentalSortDir}
              onChange={(field, direction) => { setRentalSortField(field); setRentalSortDir(direction) }}
              triggerClassName="py-2"
            />
            {canExportVermietung && (
              <button
                onClick={handleExportRentals}
                disabled={filteredRentals.length === 0}
                className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors disabled:opacity-50"
              >
                <Download className="h-4 w-4" />
                <span className="hidden sm:inline">{t('vermietung.export.button')}</span>
              </button>
            )}
          </div>

          {filteredRentals.length === 0 ? (
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
                  {filteredRentals.map((r) => {
                    const statusCfg = RESERVATION_STATUS_CONFIG[r.status] ?? RESERVATION_STATUS_CONFIG['cancelled']
                    const startStr = r.start_date.slice(0, 10)
                    const endStr = r.end_date.slice(0, 10)
                    const days = daysBetween(startStr, endStr)
                    const objName = objects.find((o) => o.id === r.object_id)?.name ?? r.object_id
                    const obj = objects.find((o) => o.id === r.object_id)
                    const depositStatus = computeDepositStatus(r)
                    const depositCfg = DEPOSIT_STATUS_CONFIG[depositStatus]
                    const depositAmount = obj?.deposit ?? 0
                    const rCurrency = rentalPrefs[r.id]?.currency ?? objectPrefs[r.object_id]?.currency ?? 'EUR'
                    const pickupLoc = rentalPrefs[r.id]?.pickupLocation ?? obj?.location ?? ''
                    const returnLoc = rentalPrefs[r.id]?.returnLocation ?? obj?.location ?? ''
                    const renterType = rentalPrefs[r.id]?.renterType ?? 'customer'
                    const overdue = isOverdue(r)

                    return (
                      <tr
                        key={r.id}
                        role="button"
                        tabIndex={0}
                        aria-label={`${objName} · ${r.renter_name}`}
                        onClick={() => openRentalDetail(r)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault()
                            openRentalDetail(r)
                          }
                        }}
                        className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer focus:outline-none focus-visible:bg-secondary/50"
                      >
                        <td className="px-4 py-3 font-medium text-foreground">{objName}</td>
                        <td className="px-4 py-3">
                          <div className="flex flex-col">
                            <span className="text-foreground">{r.renter_name}</span>
                            <span className="text-[11px] text-muted-foreground">
                              {renterType === 'employee' ? t('vermietung.reservierungen.mieterEmployee') : t('vermietung.reservierungen.mieterCustomer')}
                            </span>
                          </div>
                        </td>
                        <td className="px-4 py-3 text-muted-foreground whitespace-nowrap">
                          {formatDate(startStr)} – {formatDate(endStr)}
                        </td>
                        <td className="px-4 py-3 text-right text-foreground tabular-nums">
                          {days} {days === 1 ? t('vermietung.reservierungen.duration.day') : t('vermietung.reservierungen.duration.days')}
                        </td>
                        <td className="px-4 py-3">
                          <span className="flex items-center gap-1.5">
                            <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${statusCfg.bg}`}>
                              {t(statusCfg.labelKey)}
                            </span>
                            {overdue && (
                              <span className="inline-flex items-center gap-0.5 rounded-full bg-error-light px-2 py-0.5 text-[10px] font-medium text-error">
                                <AlertTriangle className="h-2.5 w-2.5" />
                                {t('vermietung.rentalDetail.overdue')}
                              </span>
                            )}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          {depositAmount > 0 ? (
                            <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${depositCfg.bg}`}>
                              {t(depositCfg.labelKey)}
                              {` (${formatCurrency(depositAmount, rCurrency)})`}
                            </span>
                          ) : (
                            <span className="text-xs text-muted-foreground">–</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">{pickupLoc}</td>
                        <td className="px-4 py-3 text-muted-foreground">{returnLoc}</td>
                        <td className="px-4 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                          {(r.status === 'active' || r.status === 'reserved') && (canCreateInspection || canCancelRental) && (
                            <ItemActions
                              items={[
                                ...(canCreateInspection ? [{
                                  label: t('vermietung.reservierungen.actions.zustandsprotokoll'),
                                  icon: ClipboardCheck,
                                  onClick: () => setZustandsprotokollReservation(r),
                                }] : []),
                                ...(canCancelRental ? [{
                                  label: t('vermietung.reservierungen.actions.stornieren'),
                                  icon: X,
                                  variant: 'destructive' as const,
                                  onClick: () => setConfirmCancel(r),
                                }] : []),
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
                  const todayCell = isToday(date)
                  const dateObj = new Date(date + 'T00:00:00')
                  return (
                    <div
                      key={date}
                      className={`flex flex-col items-center justify-center py-3 ${i < 6 ? 'border-r border-border' : ''} ${
                        todayCell ? 'bg-primary/5' : ''
                      }`}
                    >
                      <span className={`text-xs font-semibold ${todayCell ? 'text-primary' : i >= 5 ? 'text-muted-foreground' : 'text-foreground'}`}>
                        {WEEKDAYS[i]}
                      </span>
                      <span className={`text-[11px] ${todayCell ? 'text-primary font-medium' : 'text-muted-foreground'}`}>
                        {dateObj.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })}
                      </span>
                      {todayCell && <div className="mt-1 h-0.5 w-4 rounded-full bg-primary" />}
                    </div>
                  )
                })}
              </div>

              {/* Grid Body — Object Rows */}
              {objects.map((obj, objIdx) => {
                const typeCfg = getTypeCfg(obj.category)
                const TypeIcon = typeCfg.icon
                const isMaintenance = computeObjectStatus(obj, rentals) === 'maintenance'

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
                        <p className="text-[10px] text-muted-foreground truncate">{obj.location ?? ''}</p>
                      </div>
                    </div>

                    {/* Day Cells */}
                    {weekDates.map((date, dayIdx) => {
                      const cellKey = `${obj.id}-${date}`
                      const cellRentals = getRentalsForObjectAndDate(obj.id, date)
                      const hasRental = cellRentals.length > 0
                      const todayCell = isToday(date)
                      const isHovered = hoveredCell === cellKey

                      // Check if this is the start of a rental block
                      const startsHere = cellRentals.find((r) => r.start_date.slice(0, 10) === date)

                      // Free cell is only clickable/interactive when rental:create is granted
                      const freeCellClickable = !hasRental && !isMaintenance && canCreateRental

                      return (
                        <div
                          key={date}
                          role={hasRental || freeCellClickable ? 'button' : undefined}
                          tabIndex={hasRental || freeCellClickable ? 0 : undefined}
                          aria-label={hasRental ? `${obj.name}: ${cellRentals[0].renter_name}` : `${obj.name} ${date}`}
                          className={`relative flex items-center justify-center px-0.5 py-2 transition-colors focus:outline-none focus-visible:bg-secondary/60 ${
                            hasRental || freeCellClickable ? 'cursor-pointer' : 'cursor-default'
                          } ${
                            dayIdx < 6 ? 'border-r border-border-muted' : ''
                          } ${todayCell ? 'bg-primary/[0.02]' : ''} ${
                            !hasRental && !isMaintenance && isHovered && canCreateRental ? 'bg-secondary/60' : ''
                          }`}
                          onClick={() => handleCalendarCellClick(obj.id, date)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' || e.key === ' ') {
                              e.preventDefault()
                              handleCalendarCellClick(obj.id, date)
                            }
                          }}
                          onMouseEnter={() => (hasRental || freeCellClickable) && setHoveredCell(cellKey)}
                          onMouseLeave={() => setHoveredCell(null)}
                        >
                          {isMaintenance ? (
                            <div className="w-full h-8 rounded-md bg-warning/10 border border-warning/20 flex items-center justify-center">
                              <AlertTriangle className="h-3 w-3 text-warning" />
                            </div>
                          ) : hasRental ? (
                            <div
                              className={`w-full h-8 flex items-center px-1.5 transition-shadow ${
                                startsHere ? 'rounded-l-md' : ''
                              } ${
                                cellRentals.some((r) => r.end_date.slice(0, 10) === date) ? 'rounded-r-md' : ''
                              } ${
                                isHovered ? 'shadow-sm ring-1 ring-info/30' : ''
                              } bg-info/10 border-y border-info/20 ${
                                startsHere ? 'border-l border-info/20' : ''
                              } ${
                                cellRentals.some((r) => r.end_date.slice(0, 10) === date) ? 'border-r border-info/20' : ''
                              }`}
                            >
                              {startsHere && (
                                <span className="text-[10px] font-medium text-info truncate">
                                  {startsHere.renter_name}
                                </span>
                              )}
                            </div>
                          ) : (
                            <div
                              className={`w-full h-8 rounded-md border border-dashed transition-colors ${
                                isHovered && canCreateRental
                                  ? 'border-primary/40 bg-primary/5'
                                  : 'border-transparent'
                              }`}
                            >
                              {isHovered && canCreateRental && (
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
      {/* DETAIL MODALS (Cosmi-Fenster)                                 */}
      {/* ============================================================ */}
      <ObjectDetailModal
        object={selectedObject}
        rentals={rentals}
        objectPrefs={objectPrefs}
        getActiveRental={getActiveRental}
        getObjectRentals={getObjectRentals}
        onClose={() => setSelectedObject(null)}
        onEdit={openObjectDialog}
        onReserve={(id) => { openReservationDialog(id); setSelectedObject(null) }}
        onRentalClick={(rental) => openRentalDetail(rental, selectedObject)}
      />

      <RentalDetailModal
        rental={liveSelectedRental}
        object={liveSelectedRental ? objectsById.get(liveSelectedRental.object_id) : undefined}
        rentalPrefs={liveSelectedRental ? rentalPrefs[liveSelectedRental.id] : undefined}
        onClose={() => {
          setSelectedRental(null)
          setRentalBackObject(null)
        }}
        onBack={
          rentalBackObject
            ? () => {
                setSelectedRental(null)
                setSelectedObject(rentalBackObject)
                setRentalBackObject(null)
              }
            : undefined
        }
        onOpenProtokoll={(rental) => {
          setSelectedRental(null)
          setRentalBackObject(null)
          setZustandsprotokollReservation(rental)
        }}
        onCancel={(rental) => setConfirmCancel(rental)}
      />

      {/* ============================================================ */}
      {/* DIALOGS                                                       */}
      {/* ============================================================ */}
      <ObjectDialog
        key={objectDialogOpen ? (editObject?.id ?? 'new') : 'closed'}
        open={objectDialogOpen}
        onClose={() => {
          setObjectDialogOpen(false)
          setEditObject(null)
        }}
        initial={editObject}
      />

      <ReservationDialog
        key={reservationDialogOpen ? `${preSelectedObjectId ?? 'none'}-${preSelectedDate ?? 'none'}` : 'closed'}
        open={reservationDialogOpen}
        onClose={() => {
          setReservationDialogOpen(false)
          setPreSelectedObjectId(undefined)
          setPreSelectedDate(undefined)
        }}
        objects={objects}
        rentals={rentals}
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

      {/* Confirm Cancel Rental */}
      <ConfirmDialog
        open={!!confirmCancel}
        onOpenChange={() => setConfirmCancel(null)}
        title={t('vermietung.confirm.cancelTitle')}
        description={t('vermietung.confirm.cancelDesc', {
          renter: confirmCancel?.renter_name ?? '',
          object: objects.find((o) => o.id === confirmCancel?.object_id)?.name ?? '',
        })}
        confirmLabel={t('vermietung.confirm.cancelLabel')}
        variant="destructive"
        onConfirm={() => confirmCancel && handleCancelReservation(confirmCancel)}
      />

      {/* Zustandsprotokoll Dialog */}
      <ZustandsprotokollDialog
        open={!!zustandsprotokollReservation}
        onClose={() => setZustandsprotokollReservation(null)}
        reservation={zustandsprotokollReservation}
      />
    </div>
  )
}
