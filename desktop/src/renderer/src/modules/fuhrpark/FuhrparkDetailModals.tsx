/**
 * Row-level detail modals for the Fuhrpark tables (Wartung / Tanken /
 * Fahrtenbuch) — previously the rows were hover-only with no detail view.
 * All three use shared/DetailModal (the Cosmi window standard).
 */
import { useTranslation } from 'react-i18next'
import {
  Wrench,
  Fuel,
  BookOpen,
  Gauge,
  CalendarDays,
  User,
  Droplets,
  CircleDollarSign,
  MapPin,
  StickyNote,
} from 'lucide-react'
import { DetailModal } from '@/components/shared'
import { formatAmount, formatDate } from '@/lib/format'
import type { MaintenanceRecord, FuelRecord, LogbookEntry } from '@/stores/fuhrpark'

function MetaItem({ icon: Icon, label, value }: { icon: typeof Gauge; label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="flex items-center gap-1 text-[11px] text-muted-foreground">
        <Icon className="h-3 w-3" />
        {label}
      </p>
      <p className="mt-0.5 truncate text-sm text-foreground">{value}</p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Wartung
// ---------------------------------------------------------------------------

interface MaintenanceDetailModalProps {
  record: MaintenanceRecord | null
  onClose: () => void
  plate: string
  typeLabel: string
  typeColor: string
  currency: string
}

export function MaintenanceDetailModal({ record, onClose, plate, typeLabel, typeColor, currency }: MaintenanceDetailModalProps) {
  const { t } = useTranslation()
  if (!record) return null
  return (
    <DetailModal
      open={!!record}
      onClose={onClose}
      title={typeLabel}
      subtitle={`${plate} · ${formatDate(record.date)}`}
      badge={
        <span className={`ml-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${typeColor}`}>
          {typeLabel}
        </span>
      }
    >
      <div className="space-y-5">
        <div className="flex items-center gap-3 rounded-lg border border-border bg-secondary/30 p-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-card">
            <Wrench className="h-5 w-5 text-warning" />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-semibold text-foreground">{typeLabel}</p>
            <p className="text-xs text-muted-foreground">{plate}</p>
          </div>
          <p className="ml-auto text-sm font-semibold text-foreground tabular-nums">
            {currency} {formatAmount(record.cost)}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-x-4 gap-y-3">
          <MetaItem icon={CalendarDays} label={t('fuhrpark.table.date')} value={formatDate(record.date)} />
          <MetaItem icon={Gauge} label={t('fuhrpark.table.mileage')} value={`${record.mileage.toLocaleString('de-DE')} km`} />
          <MetaItem icon={CircleDollarSign} label={t('fuhrpark.rowDetail.cost')} value={`${currency} ${formatAmount(record.cost)}`} />
        </div>
        {record.notes && (
          <div>
            <p className="mb-1 flex items-center gap-1 text-[11px] text-muted-foreground">
              <StickyNote className="h-3 w-3" />
              {t('fuhrpark.table.notes')}
            </p>
            <p className="rounded-lg border border-border-muted bg-secondary/30 p-3 text-sm text-foreground">{record.notes}</p>
          </div>
        )}
      </div>
    </DetailModal>
  )
}

// ---------------------------------------------------------------------------
// Tanken
// ---------------------------------------------------------------------------

interface FuelDetailModalProps {
  record: FuelRecord | null
  onClose: () => void
  plate: string
  currency: string
}

export function FuelDetailModal({ record, onClose, plate, currency }: FuelDetailModalProps) {
  const { t } = useTranslation()
  if (!record) return null
  const pricePerLiter = record.liters > 0 ? record.cost / record.liters : 0
  return (
    <DetailModal
      open={!!record}
      onClose={onClose}
      title={t('fuhrpark.rowDetail.fuelTitle')}
      subtitle={`${plate} · ${formatDate(record.date)}`}
    >
      <div className="space-y-5">
        <div className="flex items-center gap-3 rounded-lg border border-border bg-secondary/30 p-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-card">
            <Fuel className="h-5 w-5 text-info" />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-semibold text-foreground">
              {record.liters.toLocaleString('de-DE', { minimumFractionDigits: 1 })} L
            </p>
            <p className="text-xs text-muted-foreground">{plate}</p>
          </div>
          <p className="ml-auto text-sm font-semibold text-foreground tabular-nums">
            {currency} {formatAmount(record.cost)}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-x-4 gap-y-3">
          <MetaItem icon={CalendarDays} label={t('fuhrpark.table.date')} value={formatDate(record.date)} />
          <MetaItem icon={Droplets} label={t('fuhrpark.table.liters')} value={`${record.liters.toLocaleString('de-DE', { minimumFractionDigits: 1 })} L`} />
          <MetaItem icon={CircleDollarSign} label={t('fuhrpark.rowDetail.pricePerLiter')} value={`${currency} ${pricePerLiter.toFixed(2)}`} />
          <MetaItem icon={Gauge} label={t('fuhrpark.table.mileage')} value={`${record.mileage.toLocaleString('de-DE')} km`} />
        </div>
      </div>
    </DetailModal>
  )
}

// ---------------------------------------------------------------------------
// Fahrtenbuch
// ---------------------------------------------------------------------------

interface TripDetailModalProps {
  entry: LogbookEntry | null
  onClose: () => void
  plate: string
  businessLabel: string
  privateLabel: string
}

export function TripDetailModal({ entry, onClose, plate, businessLabel, privateLabel }: TripDetailModalProps) {
  const { t } = useTranslation()
  if (!entry) return null
  return (
    <DetailModal
      open={!!entry}
      onClose={onClose}
      title={`${entry.startLocation} → ${entry.endLocation}`}
      subtitle={`${plate} · ${formatDate(entry.date)}`}
      badge={
        <span className={`ml-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${
          entry.isPrivate ? 'bg-warning-light text-warning' : 'bg-info-light text-info'
        }`}>
          {entry.isPrivate ? privateLabel : businessLabel}
        </span>
      }
    >
      <div className="space-y-5">
        <div className="flex items-center gap-3 rounded-lg border border-border bg-secondary/30 p-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-card">
            <BookOpen className="h-5 w-5 text-primary" />
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold text-foreground">
              {entry.startLocation} → {entry.endLocation}
            </p>
            <p className="text-xs text-muted-foreground">{entry.purpose}</p>
          </div>
          <p className="ml-auto text-sm font-semibold text-foreground tabular-nums">
            {entry.km.toLocaleString('de-DE')} km
          </p>
        </div>
        <div className="grid grid-cols-2 gap-x-4 gap-y-3">
          <MetaItem icon={CalendarDays} label={t('fuhrpark.table.date')} value={formatDate(entry.date)} />
          <MetaItem icon={User} label={t('fuhrpark.rowDetail.driver')} value={entry.driver || '—'} />
          <MetaItem icon={Gauge} label={t('fuhrpark.rowDetail.kmStart')} value={`${entry.startKm.toLocaleString('de-DE')} km`} />
          <MetaItem icon={Gauge} label={t('fuhrpark.rowDetail.kmEnd')} value={`${entry.endKm.toLocaleString('de-DE')} km`} />
          <MetaItem icon={MapPin} label={t('fuhrpark.rowDetail.route')} value={`${entry.startLocation} → ${entry.endLocation}`} />
          <MetaItem icon={StickyNote} label={t('fuhrpark.table.purpose')} value={entry.purpose || '—'} />
        </div>
      </div>
    </DetailModal>
  )
}
