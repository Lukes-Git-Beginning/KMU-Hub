import { useTranslation } from 'react-i18next'
import { DetailModal } from '@/components/shared'
import type { Rental, RentalObject } from '@/api/vermietung-types'
import { formatCurrency } from '@/lib/format'
import {
  getTypeCfg,
  STATUS_CONFIG,
  RESERVATION_STATUS_CONFIG,
  computeObjectStatus,
  formatDate,
} from './vermietung-shared'
import { useHasCapability } from '@/hooks/useCapability'

interface ObjectPrefsMap {
  [id: string]: { weeklyRate?: number; currency?: string; serialNumber?: string }
}

/**
 * ObjectDetailModal — zentriertes Cosmi-Detailfenster für ein Mietobjekt
 * (ersetzt das frühere Slide-over-DetailPanel). Meta, Preise, aktueller Status
 * und die letzten Reservierungen (klickbar → RentalDetailModal).
 */
export function ObjectDetailModal({
  object,
  rentals,
  objectPrefs,
  getActiveRental,
  getObjectRentals,
  onClose,
  onEdit,
  onReserve,
  onRentalClick,
}: {
  object: RentalObject | null
  rentals: Rental[]
  objectPrefs: ObjectPrefsMap
  getActiveRental: (id: string) => Rental | undefined
  getObjectRentals: (id: string) => Rental[]
  onClose: () => void
  onEdit: (obj: RentalObject) => void
  onReserve: (id: string) => void
  /** Opens the rental detail modal (caller wires the back-chain). */
  onRentalClick: (rental: Rental) => void
}) {
  const { t } = useTranslation()
  // RBAC: checks inside the modal follow the same hook-before-early-return rule
  const canEditObject = useHasCapability('vermietung:object:edit')
  const canCreateRental = useHasCapability('vermietung:rental:create')

  const objStatus = object ? computeObjectStatus(object, rentals) : 'available'
  const statusCfg = STATUS_CONFIG[objStatus]
  const badgeClass =
    statusCfg.dot === 'bg-success'
      ? 'bg-success-light text-success'
      : statusCfg.dot === 'bg-info'
        ? 'bg-info-light text-info'
        : 'bg-warning-light text-warning'

  const objWeeklyRate = object ? objectPrefs[object.id]?.weeklyRate : undefined
  const objCurrency = object ? (objectPrefs[object.id]?.currency ?? 'EUR') : 'EUR'
  const objSerial = object ? objectPrefs[object.id]?.serialNumber : undefined
  const activeRental = object ? getActiveRental(object.id) : undefined
  const objectRentals = object ? getObjectRentals(object.id) : []

  return (
    <DetailModal
      open={!!object}
      onClose={onClose}
      title={object?.name}
      subtitle={object ? `${t(getTypeCfg(object.category).labelKey)} · ${object.location ?? ''}` : undefined}
      badge={
        object ? (
          <span className={`ml-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${badgeClass}`}>
            {t(statusCfg.labelKey)}
          </span>
        ) : undefined
      }
      footer={
        object && (canEditObject || canCreateRental) ? (
          <div className="flex gap-2">
            {canEditObject && (
              <button
                onClick={() => onEdit(object)}
                className="flex-1 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('common.edit')}
              </button>
            )}
            {canCreateRental && (
              <button
                onClick={() => onReserve(object.id)}
                className="flex-1 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                {t('vermietung.detail.buttonReservieren')}
              </button>
            )}
          </div>
        ) : undefined
      }
    >
      {object && (
        <div className="space-y-5">
          {/* Basic info */}
          <div className="space-y-3">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.detail.details')}</h4>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
              <div>
                <p className="text-xs text-muted-foreground">{t('vermietung.detail.fieldName')}</p>
                <p className="text-sm text-foreground font-medium">{object.name}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t('vermietung.detail.fieldTyp')}</p>
                <p className="text-sm text-foreground">{t(getTypeCfg(object.category).labelKey)}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t('vermietung.detail.fieldStandort')}</p>
                <p className="text-sm text-foreground">{object.location ?? ''}</p>
              </div>
              {objSerial && (
                <div>
                  <p className="text-xs text-muted-foreground">{t('vermietung.detail.fieldSerial')}</p>
                  <p className="text-sm text-foreground font-mono">{objSerial}</p>
                </div>
              )}
            </div>
            {object.description && (
              <div>
                <p className="text-xs text-muted-foreground">{t('vermietung.detail.fieldDescription')}</p>
                <p className="text-sm text-foreground">{object.description}</p>
              </div>
            )}
          </div>

          {/* Pricing */}
          <div className="space-y-2">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.detail.preise')}</h4>
            <div className="rounded-lg border border-border bg-secondary/30 p-3">
              <div className="flex items-baseline justify-between">
                <div>
                  <span className="text-lg font-semibold text-foreground tabular-nums">{formatCurrency(object.daily_rate, objCurrency)}</span>
                  <span className="text-xs text-muted-foreground ml-1">{t('vermietung.detail.perDay')}</span>
                </div>
                {objWeeklyRate && (
                  <div>
                    <span className="text-sm font-medium text-muted-foreground tabular-nums">{formatCurrency(objWeeklyRate, objCurrency)}</span>
                    <span className="text-xs text-muted-foreground ml-1">{t('vermietung.detail.perWeek')}</span>
                  </div>
                )}
              </div>
              {object.deposit > 0 && (
                <div className="mt-2 pt-2 border-t border-border-muted flex items-baseline justify-between">
                  <span className="text-xs text-muted-foreground">{t('vermietung.detail.kaution')}</span>
                  <span className="text-sm font-medium text-foreground tabular-nums">
                    {formatCurrency(object.deposit, objCurrency)}
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
                <span className={`h-2.5 w-2.5 rounded-full ${statusCfg.dot}`} />
                <span className={`text-sm font-medium ${statusCfg.text}`}>
                  {t(statusCfg.labelKey)}
                </span>
              </div>
              {activeRental && (
                <div className="mt-2 pt-2 border-t border-border-muted">
                  <p className="text-xs text-muted-foreground">{t('vermietung.detail.activeReservation')}</p>
                  <p className="text-sm text-foreground font-medium">{activeRental.renter_name}</p>
                  <p className="text-xs text-muted-foreground">
                    {formatDate(activeRental.start_date)} – {formatDate(activeRental.end_date)}
                  </p>
                </div>
              )}
            </div>
          </div>

          {/* Last 5 rentals — klickbar → RentalDetailModal */}
          <div className="space-y-2">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vermietung.detail.lastReservations')}</h4>
            {objectRentals.length === 0 ? (
              <p className="text-xs text-muted-foreground py-2">{t('vermietung.detail.noReservations')}</p>
            ) : (
              <div className="space-y-1">
                {objectRentals.map((res) => {
                  const resCfg = RESERVATION_STATUS_CONFIG[res.status] ?? RESERVATION_STATUS_CONFIG['cancelled']
                  return (
                    <div
                      key={res.id}
                      role="button"
                      tabIndex={0}
                      onClick={() => onRentalClick(res)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          onRentalClick(res)
                        }
                      }}
                      className="flex cursor-pointer items-center gap-2 rounded-md border border-border-muted p-2 hover:bg-secondary/50 transition-colors focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    >
                      <div className="flex-1 min-w-0">
                        <p className="text-xs font-medium text-foreground">{res.renter_name}</p>
                        <p className="text-[10px] text-muted-foreground">
                          {formatDate(res.start_date)} – {formatDate(res.end_date)}
                        </p>
                      </div>
                      <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium whitespace-nowrap ${resCfg.bg}`}>
                        {t(resCfg.labelKey)}
                      </span>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>
      )}
    </DetailModal>
  )
}
