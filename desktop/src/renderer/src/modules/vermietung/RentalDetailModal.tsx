import { useTranslation } from 'react-i18next'
import {
  ArrowUpFromLine,
  ArrowDownToLine,
  CalendarDays,
  ClipboardCheck,
  MapPin,
  StickyNote,
  X,
  AlertTriangle,
  BadgeEuro,
} from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import {
  useInspectionsList,
  useStartRental,
  useEndRental,
  useUpdateRental,
} from '@/api/hooks/useVermietung'
import type { Rental, RentalObject } from '@/api/vermietung-types'
import { formatCurrency, formatDateTime } from '@/lib/format'
import {
  RESERVATION_STATUS_CONFIG,
  computeDepositStatus,
  DEPOSIT_STATUS_CONFIG,
  isOverdue,
  formatDate,
  daysBetween,
  getTypeCfg,
} from './vermietung-shared'
import { useHasCapability } from '@/hooks/useCapability'

interface RentalPrefsEntry {
  renterType?: 'employee' | 'customer'
  pickupLocation?: string
  returnLocation?: string
  currency?: string
}

/**
 * RentalDetailModal — zentriertes Cosmi-Detailfenster für eine Reservierung.
 * Zeitraum/Mieter/Preis/Kaution, Status-Lifecycle mit echten Aktionen
 * (Ausgeben → Zurücknehmen, Markt-Muster Rentman/Booqable), Übergabe-/
 * Rücknahme-Protokolle (Inspections) und Storno.
 */
export function RentalDetailModal({
  rental,
  object,
  rentalPrefs,
  onClose,
  onBack,
  onOpenProtokoll,
  onCancel,
}: {
  rental: Rental | null
  object: RentalObject | undefined
  rentalPrefs: RentalPrefsEntry | undefined
  onClose: () => void
  /** Set when opened from the object modal (back-navigation chain). */
  onBack?: () => void
  /** Opens the ZustandsprotokollDialog for this rental. */
  onOpenProtokoll: (rental: Rental) => void
  /** Opens the cancel confirmation. */
  onCancel: (rental: Rental) => void
}) {
  const { t } = useTranslation()
  const inspectionsQuery = useInspectionsList(rental?.id ?? '')
  const startMut = useStartRental()
  const endMut = useEndRental()
  const updateMut = useUpdateRental()

  // RBAC: fine-grained action gates for this modal
  const canEditRental = useHasCapability('vermietung:rental:edit')
  const canCancelRental = useHasCapability('vermietung:rental:cancel')
  const canHandoverRental = useHasCapability('vermietung:rental:handover')
  const canCreateInspection = useHasCapability('vermietung:inspection:create')

  const inspections = inspectionsQuery.data?.inspections ?? []
  const statusCfg = rental
    ? (RESERVATION_STATUS_CONFIG[rental.status] ?? RESERVATION_STATUS_CONFIG['cancelled'])
    : RESERVATION_STATUS_CONFIG['cancelled']
  const overdue = rental ? isOverdue(rental) : false

  const startStr = rental?.start_date.slice(0, 10) ?? ''
  const endStr = rental?.end_date.slice(0, 10) ?? ''
  const days = rental ? daysBetween(startStr, endStr) : 0
  const currency = rentalPrefs?.currency ?? 'EUR'
  const depositStatus = rental ? computeDepositStatus(rental) : 'none'
  const depositCfg = DEPOSIT_STATUS_CONFIG[depositStatus]
  const depositAmount = object?.deposit ?? 0
  const TypeIcon = object ? getTypeCfg(object.category).icon : CalendarDays

  const handleStart = () => {
    if (!rental) return
    startMut.mutate(rental.id, {
      onSuccess: () => toast.success(t('vermietung.rentalDetail.startSuccess', { name: object?.name ?? '' })),
      onError: () => toast.error(t('common.error')),
    })
  }

  const handleEnd = () => {
    if (!rental) return
    endMut.mutate(rental.id, {
      onSuccess: () => toast.success(t('vermietung.rentalDetail.endSuccess', { name: object?.name ?? '' })),
      onError: () => toast.error(t('common.error')),
    })
  }

  const handleMarkDepositPaid = () => {
    if (!rental) return
    updateMut.mutate(
      { id: rental.id, deposit_paid: true },
      {
        onSuccess: () => toast.success(t('vermietung.rentalDetail.depositSuccess')),
        onError: () => toast.error(t('common.error')),
      },
    )
  }

  return (
    <DetailModal
      open={!!rental}
      onClose={onClose}
      onBack={onBack}
      title={object?.name ?? rental?.renter_name}
      subtitle={rental ? `${formatDate(startStr)} – ${formatDate(endStr)} · ${rental.renter_name}` : undefined}
      badge={
        rental ? (
          <span className="ml-2 flex items-center gap-1.5">
            <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${statusCfg.bg}`}>
              {t(statusCfg.labelKey)}
            </span>
            {overdue && (
              <span className="inline-flex items-center gap-1 rounded-full bg-error-light px-2 py-0.5 text-xs font-medium text-error">
                <AlertTriangle className="h-3 w-3" />
                {t('vermietung.rentalDetail.overdue')}
              </span>
            )}
          </span>
        ) : undefined
      }
      footer={(() => {
        if (!rental) return undefined
        if (rental.status !== 'reserved' && rental.status !== 'active') return undefined
        // Compute which buttons are visible
        const showCancel = canCancelRental
        const showProtokoll = canCreateInspection
        const showStart = rental.status === 'reserved' && canHandoverRental
        const showEnd = rental.status === 'active' && canHandoverRental
        // If all actions are hidden, suppress footer entirely
        if (!showCancel && !showProtokoll && !showStart && !showEnd) return undefined
        return (
          <div className="flex gap-2">
            {showCancel && (
              <button
                onClick={() => onCancel(rental)}
                className="flex items-center justify-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-error transition-colors"
              >
                <X className="h-4 w-4" />
                {t('vermietung.reservierungen.actions.stornieren')}
              </button>
            )}
            {showProtokoll && (
              <button
                onClick={() => rental && onOpenProtokoll(rental)}
                className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <ClipboardCheck className="h-4 w-4" />
                {t('vermietung.reservierungen.actions.zustandsprotokoll')}
              </button>
            )}
            {showStart && (
              <button
                onClick={handleStart}
                disabled={startMut.isPending}
                className="flex-1 flex items-center justify-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
              >
                <ArrowUpFromLine className="h-4 w-4" />
                {t('vermietung.rentalDetail.buttonStart')}
              </button>
            )}
            {showEnd && (
              <button
                onClick={handleEnd}
                disabled={endMut.isPending}
                className="flex-1 flex items-center justify-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
              >
                <ArrowDownToLine className="h-4 w-4" />
                {t('vermietung.rentalDetail.buttonEnd')}
              </button>
            )}
          </div>
        )
      })()}
    >
      {rental && (
        <div className="space-y-5">
          {/* Meta grid */}
          <div className="space-y-3">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.rentalDetail.details')}</h4>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
              <div>
                <p className="text-xs text-muted-foreground">{t('vermietung.reservierungen.table.objekt')}</p>
                <p className="flex items-center gap-1.5 text-sm text-foreground font-medium">
                  <TypeIcon className="h-3.5 w-3.5 text-muted-foreground" />
                  {object?.name ?? '—'}
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t('vermietung.reservierungen.table.mieter')}</p>
                <p className="text-sm text-foreground">{rental.renter_name}</p>
                <p className="text-[11px] text-muted-foreground">
                  {(rentalPrefs?.renterType ?? 'customer') === 'employee'
                    ? t('vermietung.reservierungen.mieterEmployee')
                    : t('vermietung.reservierungen.mieterCustomer')}
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t('vermietung.reservierungen.table.zeitraum')}</p>
                <p className="text-sm text-foreground">{formatDate(startStr)} – {formatDate(endStr)}</p>
                <p className="text-[11px] text-muted-foreground">
                  {days} {days === 1 ? t('vermietung.reservierungen.duration.day') : t('vermietung.reservierungen.duration.days')}
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t('vermietung.reservierungen.table.abholung')}</p>
                <p className="flex items-center gap-1 text-sm text-foreground">
                  <MapPin className="h-3 w-3 text-muted-foreground" />
                  {rentalPrefs?.pickupLocation ?? object?.location ?? '—'}
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t('vermietung.reservierungen.table.rueckgabe')}</p>
                <p className="flex items-center gap-1 text-sm text-foreground">
                  <MapPin className="h-3 w-3 text-muted-foreground" />
                  {rentalPrefs?.returnLocation ?? object?.location ?? '—'}
                </p>
              </div>
            </div>
          </div>

          {/* Preis + Kaution */}
          <div className="space-y-2">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.rentalDetail.pricing')}</h4>
            <div className="rounded-lg border border-border bg-secondary/30 p-3 space-y-2">
              <div className="flex items-baseline justify-between">
                <span className="text-xs text-muted-foreground">{t('vermietung.rentalDetail.totalPrice')}</span>
                <span className="text-lg font-semibold text-foreground tabular-nums">
                  {formatCurrency(rental.total_price, currency)}
                </span>
              </div>
              {depositAmount > 0 && (
                <div className="flex items-center justify-between border-t border-border-muted pt-2">
                  <span className="text-xs text-muted-foreground">{t('vermietung.detail.kaution')}</span>
                  <span className="flex items-center gap-2">
                    <span className="text-sm font-medium text-foreground tabular-nums">
                      {formatCurrency(depositAmount, currency)}
                    </span>
                    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${depositCfg.bg}`}>
                      {t(depositCfg.labelKey)}
                    </span>
                    {!rental.deposit_paid && rental.status !== 'completed' && rental.status !== 'cancelled' && canEditRental && (
                      <button
                        onClick={handleMarkDepositPaid}
                        disabled={updateMut.isPending}
                        className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors disabled:opacity-50"
                      >
                        <BadgeEuro className="h-3 w-3" />
                        {t('vermietung.rentalDetail.markDepositPaid')}
                      </button>
                    )}
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* Notizen */}
          {rental.notes && (
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.rentalDetail.notes')}</h4>
              <div className="flex items-start gap-2 rounded-lg border border-border p-3">
                <StickyNote className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                <p className="text-sm text-foreground">{rental.notes}</p>
              </div>
            </div>
          )}

          {/* Zustandsprotokolle (Inspections) */}
          <div className="space-y-2">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {t('vermietung.rentalDetail.inspections')}
            </h4>
            {inspections.length === 0 ? (
              <p className="text-xs text-muted-foreground py-2">{t('vermietung.rentalDetail.noInspections')}</p>
            ) : (
              <div className="space-y-1">
                {inspections.map((insp) => (
                  <div key={insp.id} className="flex items-center gap-2 rounded-md border border-border-muted p-2">
                    <span className={`flex h-6 w-6 items-center justify-center rounded-md ${insp.kind === 'handover' ? 'bg-info-light text-info' : 'bg-success-light text-success'}`}>
                      <ClipboardCheck className="h-3 w-3" />
                    </span>
                    <div className="flex-1 min-w-0">
                      <p className="text-xs font-medium text-foreground">
                        {insp.kind === 'handover'
                          ? t('vermietung.rentalDetail.inspectionHandover')
                          : t('vermietung.rentalDetail.inspectionReturn')}
                      </p>
                      <p className="text-[10px] text-muted-foreground truncate">
                        {insp.notes || '—'}{insp.performed_by ? ` · ${insp.performed_by}` : ''}
                      </p>
                    </div>
                    <span className="text-[10px] text-muted-foreground whitespace-nowrap">
                      {formatDateTime(insp.created_at)}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </DetailModal>
  )
}
