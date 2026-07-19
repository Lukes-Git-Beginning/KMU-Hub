import { useState, useMemo, useCallback, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Plus,
  Car,
  Wrench,
  Fuel,
  AlertTriangle,
  Gauge,
  User,
  TrendingUp,
  Droplets,
  FileText,
  MapPin,
  Navigation,
  RefreshCw,
  ChevronDown,
  ChevronUp,
  BookOpen,
  CircleDollarSign,
  Camera,
  Truck,
  FileDown,
  FileSpreadsheet,
} from 'lucide-react'
import { toast } from 'sonner'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { DetailModal, EmptyState, PageHeader, SortMenu, type SortDirection } from '@/components/shared'
import { useCapabilitySet, useHasCapability } from '@/hooks/useCapability'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'
import { formatAmount, formatDate, formatTime } from '@/lib/format'
import type { Vehicle, MaintenanceRecord, FuelRecord, LogbookEntry, VehicleDocument, DamageReport, VehicleRoute } from '@/stores/fuhrpark'
import {
  useVehiclesList,
  useServicesList,
  useFuelLogs,
  useTripLogs,
  useDamagesList,
  useVehicleDocuments,
  useVehicleRoutes,
  useCreateVehicle,
  useScheduleService,
  useCreateFuelLog,
  useReportDamage,
  useCreateTripLog,
} from '@/api/hooks/useFuhrpark'
import type { Vehicle as ApiVehicle, VehicleService, VehicleDamage, FuelLog, TripLog } from '@/api/fuhrpark-client'
import SchadensmeldungDialog from './SchadensmeldungDialog'
import { useFuhrparkPrefsStore } from '@/stores/fuhrparkPrefs'
import { useFuhrparkTenantStore } from '@/stores/fuhrparkTenant'
import { MaintenanceDetailModal, FuelDetailModal, TripDetailModal } from './FuhrparkDetailModals'
import {
  buildLogbookCsv,
  buildLogbookPdf,
  buildVehiclesCsv,
  csvDateStamp,
  downloadBlob,
} from './fuhrpark-export'

// ---------------------------------------------------------------------------
// API → Store type adapters (keep UI types stable, map API snake_case fields)
// ---------------------------------------------------------------------------

function adaptVehicle(v: ApiVehicle): Vehicle {
  return {
    id: v.id,
    licensePlate: v.license_plate,
    make: v.make,
    model: v.model,
    year: v.year,
    type: 'car', // API has no vehicle type classification yet — default to car
    currentDriver: v.assigned_driver_id ?? '',
    // protojson serializes int64 as a JSON string (proto3 spec); coerce before storing as UI number.
    mileage: Number(v.mileage_km),
    nextInspection: v.tuev_due_date ?? '2099-01-01',
    insuranceExpiry: '2099-01-01', // not yet in API model
    isActive: v.status === 'active' || v.status === 'in_service',
  }
}

function adaptService(s: VehicleService): MaintenanceRecord {
  const typeMap: Record<string, MaintenanceRecord['type']> = {
    oil_change: 'service',
    tire_change: 'tires',
    inspection: 'inspection',
    repair: 'repair',
  }
  return {
    id: s.id,
    vehicleId: s.vehicle_id,
    vehiclePlate: s.vehicle_id, // plate not returned by API at list level — use id as placeholder
    type: typeMap[s.service_type] ?? 'service',
    date: s.scheduled_at,
    // protojson serializes int64 as a JSON string (proto3 spec); coerce before storing as UI number.
    mileage: Number(s.mileage_km ?? 0),
    cost: Number(s.cost_cents ?? 0) / 100,
    notes: s.notes ?? s.description ?? '',
  }
}

function adaptFuelLog(f: FuelLog): FuelRecord {
  return {
    id: f.id,
    vehicleId: f.vehicle_id,
    vehiclePlate: f.vehicle_id, // plate not returned at list level — use vehicle_id as placeholder
    date: f.date,
    liters: f.liters,
    // protojson serializes int64 as a JSON string (proto3 spec); coerce before storing as UI number.
    cost: Number(f.cost_cents) / 100,
    mileage: Number(f.mileage_km),
  }
}

function adaptTripLog(t: TripLog): LogbookEntry {
  return {
    id: t.id,
    vehicleId: t.vehicle_id,
    vehiclePlate: t.vehicle_id,
    date: t.date,
    startLocation: t.start_location,
    endLocation: t.end_location,
    purpose: t.purpose,
    // protojson serializes int64 as a JSON string (proto3 spec); coerce before storing as UI number.
    startKm: Number(t.start_km),
    endKm: Number(t.end_km),
    km: Number(t.km),
    isPrivate: t.is_private,
    driver: t.driver_name,
  }
}

function adaptDamage(d: VehicleDamage): DamageReport {
  const severityMap: Record<string, DamageReport['severity']> = {
    minor: 'minor',
    moderate: 'moderate',
    major: 'major',
    totalled: 'major',
  }
  const statusMap: Record<string, DamageReport['status']> = {
    reported: 'open',
    in_repair: 'in_progress',
    resolved: 'resolved',
  }
  return {
    id: d.id,
    vehicleId: d.vehicle_id,
    vehiclePlate: d.vehicle_id,
    date: d.created_at.slice(0, 10),
    description: d.description,
    severity: severityMap[d.severity] ?? 'minor',
    location: '',
    photoCount: d.photo_keys?.length ?? 0,
    reportedBy: d.reported_by ?? '',
    status: statusMap[d.status] ?? 'open',
  }
}

// ---------------------------------------------------------------------------
// Types & Constants
// ---------------------------------------------------------------------------

type TabKey = 'fahrzeuge' | 'wartung' | 'tankprotokoll' | 'tracking' | 'fahrtenbuch'
type DialogKey = 'addVehicle' | 'addMaintenance' | 'addFuel' | 'addDamage' | 'addTrip' | null

const vehicleTypeLabels: Record<string, string> = {
  car: 'fuhrpark.vehicleType.car',
  van: 'fuhrpark.vehicleType.van',
  truck: 'fuhrpark.vehicleType.truck',
}

const vehicleTypeColors: Record<string, string> = {
  car: 'bg-info-light text-info',
  van: 'bg-warning-light text-warning',
  truck: 'bg-primary-light text-primary',
}

const maintenanceTypeLabels: Record<string, string> = {
  service: 'fuhrpark.maintenanceType.service',
  repair: 'fuhrpark.maintenanceType.repair',
  inspection: 'fuhrpark.maintenanceType.inspection',
  tires: 'fuhrpark.maintenanceType.tires',
}

const maintenanceTypeColors: Record<string, string> = {
  service: 'bg-info-light text-info',
  repair: 'bg-warning-light text-warning',
  inspection: 'bg-success-light text-success',
  tires: 'bg-primary-light text-primary',
}

const documentTypeLabels: Record<string, string> = {
  registration: 'fuhrpark.documentType.registration',
  insurance: 'fuhrpark.documentType.insurance',
  tuev: 'fuhrpark.documentType.tuev',
  other: 'fuhrpark.documentType.other',
}

const documentTypeColors: Record<string, string> = {
  registration: 'bg-primary-light text-primary',
  insurance: 'bg-info-light text-info',
  tuev: 'bg-warning-light text-warning',
  other: 'bg-secondary text-muted-foreground',
}

const severityLabels: Record<string, string> = {
  minor: 'fuhrpark.severity.minor',
  moderate: 'fuhrpark.severity.moderate',
  major: 'fuhrpark.severity.major',
}

const severityColors: Record<string, string> = {
  minor: 'bg-info-light text-info',
  moderate: 'bg-warning-light text-warning',
  major: 'bg-error-light text-error',
}

const damageStatusLabels: Record<string, string> = {
  open: 'fuhrpark.damageStatus.open',
  in_progress: 'fuhrpark.damageStatus.in_progress',
  resolved: 'fuhrpark.damageStatus.resolved',
}

const damageStatusColors: Record<string, string> = {
  open: 'bg-warning-light text-warning',
  in_progress: 'bg-info-light text-info',
  resolved: 'bg-success-light text-success',
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// Lead days come from the tenant settings (default 30) — market standard
// (Vimcar/Fleetster) makes the reminder window company-configurable.
function getDateStatus(dateStr: string, leadDays = 30): 'overdue' | 'soon' | 'ok' {
  const date = new Date(dateStr)
  const now = new Date()
  const diffDays = Math.floor((date.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
  if (diffDays < 0) return 'overdue'
  if (diffDays <= leadDays) return 'soon'
  return 'ok'
}

function statusDotColor(status: 'overdue' | 'soon' | 'ok'): string {
  if (status === 'overdue') return 'bg-error'
  if (status === 'soon') return 'bg-warning'
  return 'bg-success'
}

function statusTextColor(status: 'overdue' | 'soon' | 'ok'): string {
  if (status === 'overdue') return 'text-error font-medium'
  if (status === 'soon') return 'text-warning font-medium'
  return ''
}

// formatDate / formatTime imported from @/lib/format (locale-aware)


function formatKm(val: number): string {
  return val.toLocaleString('de-DE')
}

// '2099-01-01' is the adapter placeholder for dates the API does not carry yet
// (insuranceExpiry, missing tuev_due_date) — render a dash instead of a fake date.
function formatDueDate(dateStr: string): string {
  return dateStr.startsWith('2099') ? '—' : formatDate(dateStr)
}

function generateId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
}

const trackingStatusDot: Record<string, string> = {
  driving: 'bg-success',
  parked: 'bg-muted-foreground/40',
  unknown: 'bg-warning',
}

const trackingStatusLabel: Record<string, string> = {
  driving: 'fuhrpark.trackingStatus.driving',
  parked: 'fuhrpark.trackingStatus.parked',
  unknown: 'fuhrpark.trackingStatus.unknown',
}

const trackingStatusTextColor: Record<string, string> = {
  driving: 'text-success',
  parked: 'text-muted-foreground',
  unknown: 'text-warning',
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function formatRelativeTime(timestamp: string, t: (key: string, opts?: any) => string): string {
  const now = new Date()
  const then = new Date(timestamp)
  const diffMs = now.getTime() - then.getTime()
  const diffMin = Math.floor(diffMs / 60000)

  if (diffMin < 1) return t('fuhrpark.relativeTime.justNow')
  if (diffMin < 60) return t('fuhrpark.relativeTime.minutesAgo', { min: diffMin })
  const diffHrs = Math.floor(diffMin / 60)
  if (diffHrs < 24) return t('fuhrpark.relativeTime.hoursAgo', { hrs: diffHrs })
  const diffDays = Math.floor(diffHrs / 24)
  return diffDays > 1
    ? t('fuhrpark.relativeTime.daysAgoPlural', { days: diffDays })
    : t('fuhrpark.relativeTime.daysAgo', { days: diffDays })
}

// formatTime imported from @/lib/format (locale-aware, defaults to timeStyle 'short')

// ---------------------------------------------------------------------------
// Dialog: Fahrzeug hinzufügen
// ---------------------------------------------------------------------------

interface AddVehicleDialogProps {
  onClose: () => void
  onSave: (v: Vehicle) => void
}

function AddVehicleDialog({ onClose, onSave }: AddVehicleDialogProps) {
  const { t } = useTranslation()
  const [plate, setPlate] = useState('')
  const [make, setMake] = useState('')
  const [model, setModel] = useState('')
  const [year, setYear] = useState(new Date().getFullYear())
  const [type, setType] = useState<'car' | 'van' | 'truck'>('car')
  const [driver, setDriver] = useState('')
  const [mileage, setMileage] = useState(0)
  const [nextInspection, setNextInspection] = useState('')
  const [insuranceExpiry, setInsuranceExpiry] = useState('')

  const canSave = plate.trim() && make.trim() && model.trim()

  const handleSave = () => {
    if (!canSave) return
    const vehicle: Vehicle = {
      id: generateId('veh'),
      licensePlate: plate.trim(),
      make: make.trim(),
      model: model.trim(),
      year,
      type,
      currentDriver: driver.trim(),
      mileage,
      nextInspection: nextInspection || '2027-01-01',
      insuranceExpiry: insuranceExpiry || '2027-01-01',
      isActive: true,
    }
    onSave(vehicle)
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-lg glass-elevated">
        <DialogHeader className="px-6 pt-6 pb-0">
          <DialogTitle className="text-base font-semibold text-foreground">{t('fuhrpark.addVehicle.title')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 px-6 pt-5">
          {/* Row: Kennzeichen + Typ */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addVehicle.plate')}</label>
              <input
                type="text"
                value={plate}
                onChange={(e) => setPlate(e.target.value)}
                placeholder={t('fuhrpark.addVehicle.platePlaceholder')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addVehicle.type')}</label>
              <select
                value={type}
                onChange={(e) => setType(e.target.value as 'car' | 'van' | 'truck')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                <option value="car">{t('fuhrpark.vehicleType.car')}</option>
                <option value="van">{t('fuhrpark.vehicleType.van')}</option>
                <option value="truck">{t('fuhrpark.vehicleType.truck')}</option>
              </select>
            </div>
          </div>

          {/* Row: Marke + Modell */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addVehicle.make')}</label>
              <input
                type="text"
                value={make}
                onChange={(e) => setMake(e.target.value)}
                placeholder={t('fuhrpark.addVehicle.makePlaceholder')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addVehicle.model')}</label>
              <input
                type="text"
                value={model}
                onChange={(e) => setModel(e.target.value)}
                placeholder={t('fuhrpark.addVehicle.modelPlaceholder')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Row: Jahrgang + km-Stand */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addVehicle.year')}</label>
              <input
                type="number"
                value={year}
                onChange={(e) => setYear(Number(e.target.value))}
                min={2000}
                max={2030}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addVehicle.mileage')}</label>
              <input
                type="number"
                value={mileage}
                onChange={(e) => setMileage(Number(e.target.value))}
                min={0}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Fahrer */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addVehicle.driver')}</label>
            <input
              type="text"
              value={driver}
              onChange={(e) => setDriver(e.target.value)}
              placeholder={t('fuhrpark.addVehicle.driverPlaceholder')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Row: Prüfung + Versicherung */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addVehicle.nextInspection')}</label>
              <input
                type="date"
                value={nextInspection}
                onChange={(e) => setNextInspection(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addVehicle.insuranceExpiry')}</label>
              <input
                type="date"
                value={insuranceExpiry}
                onChange={(e) => setInsuranceExpiry(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>
        </div>

        {/* Actions */}
        <DialogFooter className="px-6 py-4">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            disabled={!canSave}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:pointer-events-none"
          >
            {t('common.save')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Dialog: Wartung eintragen
// ---------------------------------------------------------------------------

interface AddMaintenanceDialogProps {
  vehicles: Vehicle[]
  preselectedVehicleId?: string
  onClose: () => void
  onSave: (r: MaintenanceRecord) => void
}

function AddMaintenanceDialog({ vehicles, preselectedVehicleId, onClose, onSave }: AddMaintenanceDialogProps) {
  const { t } = useTranslation()
  const [vehicleId, setVehicleId] = useState(preselectedVehicleId || vehicles[0]?.id || '')
  const [type, setType] = useState<'service' | 'repair' | 'inspection' | 'tires'>('service')
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10))
  const [mileage, setMileage] = useState(0)
  const [cost, setCost] = useState(0)
  const [notes, setNotes] = useState('')

  const selectedVehicle = vehicles.find((v) => v.id === vehicleId)

  const handleSave = () => {
    if (!selectedVehicle) return
    const record: MaintenanceRecord = {
      id: generateId('mnt'),
      vehicleId,
      vehiclePlate: selectedVehicle.licensePlate,
      type,
      date,
      mileage,
      cost,
      notes: notes.trim(),
    }
    onSave(record)
  }

  const typeOptions: { value: MaintenanceRecord['type']; label: string }[] = [
    { value: 'service', label: t('fuhrpark.maintenanceType.service') },
    { value: 'repair', label: t('fuhrpark.maintenanceType.repair') },
    { value: 'inspection', label: t('fuhrpark.maintenanceType.inspection') },
    { value: 'tires', label: t('fuhrpark.maintenanceType.tires') },
  ]

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-lg glass-elevated">
        <DialogHeader className="px-6 pt-6 pb-0">
          <DialogTitle className="text-base font-semibold text-foreground">{t('fuhrpark.addMaintenance.title')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 px-6 pt-5">
          {/* Fahrzeug */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addMaintenance.vehicle')}</label>
            <select
              value={vehicleId}
              onChange={(e) => setVehicleId(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              {vehicles.filter((v) => v.isActive).map((v) => (
                <option key={v.id} value={v.id}>
                  {v.licensePlate} - {v.make} {v.model}
                </option>
              ))}
            </select>
          </div>

          {/* Typ (radio) */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addMaintenance.type')}</label>
            <div className="flex gap-2">
              {typeOptions.map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => setType(opt.value)}
                  className={`rounded-lg border px-3 py-1.5 text-xs transition-colors ${
                    type === opt.value
                      ? 'border-primary bg-primary/10 text-primary font-medium'
                      : 'border-border text-muted-foreground hover:bg-secondary'
                  }`}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          {/* Row: Datum + km-Stand */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addMaintenance.date')}</label>
              <input
                type="date"
                value={date}
                onChange={(e) => setDate(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addMaintenance.mileage')}</label>
              <input
                type="number"
                value={mileage}
                onChange={(e) => setMileage(Number(e.target.value))}
                min={0}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Kosten */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addMaintenance.cost')}</label>
            <input
              type="number"
              value={cost}
              onChange={(e) => setCost(Number(e.target.value))}
              min={0}
              step={0.01}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Notizen */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addMaintenance.notes')}</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
              placeholder={t('fuhrpark.addMaintenance.notesPlaceholder')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
            />
          </div>
        </div>

        <DialogFooter className="px-6 py-4">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            disabled={!vehicleId}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:pointer-events-none"
          >
            {t('common.save')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Dialog: Tanken eintragen
// ---------------------------------------------------------------------------

interface AddFuelDialogProps {
  vehicles: Vehicle[]
  preselectedVehicleId?: string
  onClose: () => void
  onSave: (r: FuelRecord) => void
}

function AddFuelDialog({ vehicles, preselectedVehicleId, onClose, onSave }: AddFuelDialogProps) {
  const { t } = useTranslation()
  const [vehicleId, setVehicleId] = useState(preselectedVehicleId || vehicles[0]?.id || '')
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10))
  const [liters, setLiters] = useState(0)
  const [cost, setCost] = useState(0)
  const [mileage, setMileage] = useState(0)

  const selectedVehicle = vehicles.find((v) => v.id === vehicleId)

  const handleSave = () => {
    if (!selectedVehicle) return
    const record: FuelRecord = {
      id: generateId('fuel'),
      vehicleId,
      vehiclePlate: selectedVehicle.licensePlate,
      date,
      liters,
      cost,
      mileage,
    }
    onSave(record)
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-md glass-elevated">
        <DialogHeader className="px-6 pt-6 pb-0">
          <DialogTitle className="text-base font-semibold text-foreground">{t('fuhrpark.addFuel.title')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 px-6 pt-5">
          {/* Fahrzeug */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addFuel.vehicle')}</label>
            <select
              value={vehicleId}
              onChange={(e) => setVehicleId(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              {vehicles.filter((v) => v.isActive).map((v) => (
                <option key={v.id} value={v.id}>
                  {v.licensePlate} - {v.make} {v.model}
                </option>
              ))}
            </select>
          </div>

          {/* Datum */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addFuel.date')}</label>
            <input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Row: Liter + Kosten */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addFuel.liters')}</label>
              <input
                type="number"
                value={liters}
                onChange={(e) => setLiters(Number(e.target.value))}
                min={0}
                step={0.1}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addFuel.cost')}</label>
              <input
                type="number"
                value={cost}
                onChange={(e) => setCost(Number(e.target.value))}
                min={0}
                step={0.01}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* km-Stand */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addFuel.mileage')}</label>
            <input
              type="number"
              value={mileage}
              onChange={(e) => setMileage(Number(e.target.value))}
              min={0}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
        </div>

        <DialogFooter className="px-6 py-4">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            disabled={!vehicleId || liters <= 0}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:pointer-events-none"
          >
            {t('common.save')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Dialog: Fahrt eintragen (Fahrtenbuch) — wires the previously unused
// createTripLog mutation; fields follow the finanzamts-konform market layout
// (Vimcar/Fleetster): route, km readings, category, purpose, driver.
// ---------------------------------------------------------------------------

interface AddTripDialogProps {
  vehicles: Vehicle[]
  defaultCategory: 'business' | 'private'
  privateTripsEnabled: boolean
  onClose: () => void
  onSave: (trip: {
    vehicleId: string
    date: string
    startLocation: string
    endLocation: string
    purpose: string
    startKm: number
    endKm: number
    isPrivate: boolean
    driver: string
  }) => void
}

function AddTripDialog({ vehicles, defaultCategory, privateTripsEnabled, onClose, onSave }: AddTripDialogProps) {
  const { t } = useTranslation()
  const [vehicleId, setVehicleId] = useState(vehicles[0]?.id || '')
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10))
  const [startLocation, setStartLocation] = useState('')
  const [endLocation, setEndLocation] = useState('')
  const [purpose, setPurpose] = useState('')
  const [startKm, setStartKm] = useState(0)
  const [endKm, setEndKm] = useState(0)
  const [isPrivate, setIsPrivate] = useState(privateTripsEnabled && defaultCategory === 'private')
  const [driver, setDriver] = useState('')

  const km = Math.max(0, endKm - startKm)
  // Finanzamt: purpose is mandatory for business trips, end km must exceed start km
  const canSave =
    !!vehicleId && startLocation.trim() && endLocation.trim() && endKm > startKm && (isPrivate || purpose.trim())

  const handleSave = () => {
    if (!canSave) return
    onSave({
      vehicleId,
      date,
      startLocation: startLocation.trim(),
      endLocation: endLocation.trim(),
      purpose: purpose.trim(),
      startKm,
      endKm,
      isPrivate,
      driver: driver.trim(),
    })
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-lg glass-elevated">
        <DialogHeader className="px-6 pt-6 pb-0">
          <DialogTitle className="text-base font-semibold text-foreground">{t('fuhrpark.addTrip.title')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 px-6 pt-5">
          {/* Row: Fahrzeug + Datum */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addTrip.vehicle')}</label>
              <select
                value={vehicleId}
                onChange={(e) => setVehicleId(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                {vehicles.filter((v) => v.isActive).map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.licensePlate} - {v.make} {v.model}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addTrip.date')}</label>
              <input
                type="date"
                value={date}
                max={new Date().toISOString().slice(0, 10)}
                onChange={(e) => setDate(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Row: Start + Ziel */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addTrip.start')}</label>
              <input
                type="text"
                value={startLocation}
                onChange={(e) => setStartLocation(e.target.value)}
                placeholder={t('fuhrpark.addTrip.startPlaceholder')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addTrip.end')}</label>
              <input
                type="text"
                value={endLocation}
                onChange={(e) => setEndLocation(e.target.value)}
                placeholder={t('fuhrpark.addTrip.endPlaceholder')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Row: km-Stände + berechnete km */}
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addTrip.startKm')}</label>
              <input
                type="number"
                value={startKm}
                onChange={(e) => setStartKm(Number(e.target.value))}
                min={0}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addTrip.endKm')}</label>
              <input
                type="number"
                value={endKm}
                onChange={(e) => setEndKm(Number(e.target.value))}
                min={0}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addTrip.km')}</label>
              <div className="rounded-lg border border-border-muted bg-secondary/40 px-3 py-2 text-sm text-foreground tabular-nums">
                {km.toLocaleString('de-DE')} km
              </div>
            </div>
          </div>
          {endKm > 0 && endKm <= startKm && (
            <p className="text-xs text-error">{t('fuhrpark.addTrip.kmError')}</p>
          )}

          {/* Kategorie */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addTrip.category')}</label>
            <div className="flex gap-2">
              <button
                onClick={() => setIsPrivate(false)}
                className={`rounded-lg border px-3 py-1.5 text-xs transition-colors ${
                  !isPrivate
                    ? 'border-primary bg-primary/10 text-primary font-medium'
                    : 'border-border text-muted-foreground hover:bg-secondary'
                }`}
              >
                {t('fuhrpark.logbook.business')}
              </button>
              {privateTripsEnabled && (
                <button
                  onClick={() => setIsPrivate(true)}
                  className={`rounded-lg border px-3 py-1.5 text-xs transition-colors ${
                    isPrivate
                      ? 'border-primary bg-primary/10 text-primary font-medium'
                      : 'border-border text-muted-foreground hover:bg-secondary'
                  }`}
                >
                  {t('fuhrpark.logbook.private')}
                </button>
              )}
            </div>
            {!privateTripsEnabled && (
              <p className="text-xs text-muted-foreground">{t('fuhrpark.addTrip.privateDisabledHint')}</p>
            )}
          </div>

          {/* Zweck (Pflicht bei Geschäftsfahrt) */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              {isPrivate ? t('fuhrpark.addTrip.purposeOptional') : t('fuhrpark.addTrip.purpose')}
            </label>
            <input
              type="text"
              value={purpose}
              onChange={(e) => setPurpose(e.target.value)}
              placeholder={t('fuhrpark.addTrip.purposePlaceholder')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Fahrer */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t('fuhrpark.addTrip.driver')}</label>
            <input
              type="text"
              value={driver}
              onChange={(e) => setDriver(e.target.value)}
              placeholder={t('fuhrpark.addTrip.driverPlaceholder')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
        </div>

        <DialogFooter className="px-6 py-4">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            disabled={!canSave}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:pointer-events-none"
          >
            {t('common.save')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Vehicle Detail Panel Content
// ---------------------------------------------------------------------------

interface VehicleDetailContentProps {
  vehicle: Vehicle
  maintenanceRecords: MaintenanceRecord[]
  fuelRecords: FuelRecord[]
  logbookEntries: LogbookEntry[]
  vehicleDocuments: VehicleDocument[]
  damageReports: DamageReport[]
  currency: string
  reminderLeadDays: number
  onAddMaintenance: () => void
  onAddFuel: () => void
  onAddDamage: () => void
  // RBAC: action gate flags passed from parent (hooks live in FuhrparkPage)
  canCreateService: boolean
  canCreateFuel: boolean
  canCreateDamage: boolean
}

function VehicleDetailContent({
  vehicle,
  maintenanceRecords,
  fuelRecords,
  logbookEntries,
  vehicleDocuments,
  damageReports,
  currency,
  reminderLeadDays,
  onAddMaintenance,
  onAddFuel,
  onAddDamage,
  canCreateService,
  canCreateFuel,
  canCreateDamage,
}: VehicleDetailContentProps) {
  const { t } = useTranslation()
  const inspectionStatus = getDateStatus(vehicle.nextInspection, reminderLeadDays)
  const insuranceStatus = getDateStatus(vehicle.insuranceExpiry, reminderLeadDays)

  const vehicleMaintenance = maintenanceRecords
    .filter((r) => r.vehicleId === vehicle.id)
    .sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())
    .slice(0, 3)

  const vehicleFuel = fuelRecords
    .filter((r) => r.vehicleId === vehicle.id)
    .sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())
    .slice(0, 3)

  const totalFuelCost = fuelRecords
    .filter((r) => r.vehicleId === vehicle.id)
    .reduce((sum, r) => sum + r.cost, 0)

  const totalMaintenanceCost = maintenanceRecords
    .filter((r) => r.vehicleId === vehicle.id)
    .reduce((sum, r) => sum + r.cost, 0)

  // Wave 9: Logbook entries for this vehicle (last 3)
  const vehicleLogbook = logbookEntries
    .filter((e) => e.vehicleId === vehicle.id)
    .sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())
    .slice(0, 3)

  // Wave 9: Documents for this vehicle
  const vehicleDocs = vehicleDocuments
    .filter((d) => d.vehicleId === vehicle.id)
    .sort((a, b) => new Date(b.uploadDate).getTime() - new Date(a.uploadDate).getTime())

  // Wave 9: Damage reports for this vehicle
  const vehicleDamage = damageReports
    .filter((d) => d.vehicleId === vehicle.id)
    .sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())

  // Wave 9: TCO calculation
  const mockInsuranceCost = 2400 // EUR/year mock
  const vehicleAge = new Date().getFullYear() - vehicle.year
  const mockDepreciation = vehicleAge > 0 ? Math.round(30000 / Math.max(vehicleAge * 2, 1)) : 5000 // rough mock
  const tcoMaintenance = maintenanceRecords.filter((r) => r.vehicleId === vehicle.id).reduce((s, r) => s + r.cost, 0)
  const tcoFuel = fuelRecords.filter((r) => r.vehicleId === vehicle.id).reduce((s, r) => s + r.cost, 0)
  const tcoTotal = tcoMaintenance + tcoFuel + mockInsuranceCost + mockDepreciation
  const tcoCats = [
    { label: t('fuhrpark.tco.maintenance'), value: tcoMaintenance, color: 'bg-warning' },
    { label: t('fuhrpark.tco.fuel'), value: tcoFuel, color: 'bg-info' },
    { label: t('fuhrpark.tco.insurance'), value: mockInsuranceCost, color: 'bg-primary' },
    { label: t('fuhrpark.tco.depreciation'), value: mockDepreciation, color: 'bg-muted-foreground' },
  ]

  return (
    <div className="space-y-5">
      {/* License Plate - large monospace */}
      <div className="text-center">
        <div className="inline-block rounded-lg border-2 border-border bg-secondary/50 px-5 py-2.5">
          <p className="text-2xl font-bold text-foreground tracking-[0.15em] font-mono">
            {vehicle.licensePlate}
          </p>
        </div>
      </div>

      {/* Make/Model/Year + Type badge */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-medium text-foreground">
            {vehicle.make} {vehicle.model}
          </h3>
          <p className="text-xs text-muted-foreground">{t('fuhrpark.detail.modelYear', { year: vehicle.year })}</p>
        </div>
        <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-medium ${vehicleTypeColors[vehicle.type] ?? 'bg-secondary text-muted-foreground'}`}>
          {vehicleTypeLabels[vehicle.type] ? t(vehicleTypeLabels[vehicle.type]) : vehicle.type}
        </span>
      </div>

      {/* Driver + Mileage */}
      <section className="space-y-2">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{t('fuhrpark.detail.vehicleData')}</h4>
        <div className="space-y-1.5 text-xs">
          <div className="flex items-center gap-2 text-muted-foreground">
            <User className="h-3.5 w-3.5 shrink-0" />
            <span className="text-foreground">{vehicle.currentDriver || t('fuhrpark.detail.noDriverAssigned')}</span>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <Gauge className="h-3.5 w-3.5 shrink-0" />
            <span className="text-foreground">{formatKm(vehicle.mileage)} km</span>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <TrendingUp className="h-3.5 w-3.5 shrink-0" />
            <span className="text-foreground">
              {t('fuhrpark.detail.totalCost', { amount: formatAmount(totalMaintenanceCost + totalFuelCost) })}
            </span>
          </div>
        </div>
      </section>

      {/* Wave 9: TCO Dashboard */}
      <section className="space-y-2">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
          <CircleDollarSign className="h-3 w-3" />
          {t('fuhrpark.detail.tco')}
        </h4>
        <div className="grid grid-cols-2 gap-2">
          {tcoCats.map((cat) => (
            <div key={cat.label} className="rounded-md border border-border-muted bg-secondary/30 px-3 py-2">
              <p className="text-[10px] text-muted-foreground">{cat.label}</p>
              <p className="text-xs font-semibold text-foreground">{currency} {formatAmount(cat.value)}</p>
            </div>
          ))}
        </div>
        <div className="rounded-md bg-secondary/50 px-3 py-2">
          <div className="flex items-center justify-between mb-1.5">
            <span className="text-[10px] text-muted-foreground">{t('fuhrpark.detail.costDistribution')}</span>
            <span className="text-xs font-semibold text-foreground">{currency} {formatAmount(tcoTotal)}</span>
          </div>
          <div className="flex h-2.5 rounded-full overflow-hidden gap-0.5">
            {tcoCats.map((cat) => {
              const pct = tcoTotal > 0 ? (cat.value / tcoTotal) * 100 : 0
              return pct > 0 ? (
                <div
                  key={cat.label}
                  className={`${cat.color} rounded-sm`}
                  style={{ width: `${pct}%` }}
                  title={`${cat.label}: ${Math.round(pct)}%`}
                />
              ) : null
            })}
          </div>
          <div className="flex flex-wrap gap-x-3 gap-y-0.5 mt-1.5">
            {tcoCats.map((cat) => (
              <div key={cat.label} className="flex items-center gap-1">
                <span className={`h-1.5 w-1.5 rounded-full ${cat.color}`} />
                <span className="text-[9px] text-muted-foreground">{cat.label}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Inspection + Insurance with traffic light */}
      <section className="space-y-2">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{t('fuhrpark.detail.status')}</h4>
        <div className="space-y-2">
          <div className="flex items-center justify-between rounded-md bg-secondary/50 px-3 py-2">
            <div className="flex items-center gap-2">
              <span className={`h-2.5 w-2.5 rounded-full ${statusDotColor(inspectionStatus)}`} />
              <span className="text-xs text-muted-foreground">{t('fuhrpark.detail.nextInspection')}</span>
            </div>
            <span className={`text-xs ${statusTextColor(inspectionStatus)}`}>
              {inspectionStatus === 'overdue' && <AlertTriangle className="inline h-3 w-3 mr-1" />}
              {formatDueDate(vehicle.nextInspection)}
            </span>
          </div>
          <div className="flex items-center justify-between rounded-md bg-secondary/50 px-3 py-2">
            <div className="flex items-center gap-2">
              <span className={`h-2.5 w-2.5 rounded-full ${statusDotColor(insuranceStatus)}`} />
              <span className="text-xs text-muted-foreground">{t('fuhrpark.detail.insuranceExpiry')}</span>
            </div>
            <span className={`text-xs ${statusTextColor(insuranceStatus)}`}>
              {insuranceStatus === 'overdue' && <AlertTriangle className="inline h-3 w-3 mr-1" />}
              {formatDueDate(vehicle.insuranceExpiry)}
            </span>
          </div>
        </div>
      </section>

      {/* Recent Maintenance */}
      <section className="space-y-2">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
          {t('fuhrpark.detail.recentMaintenance')}
        </h4>
        {vehicleMaintenance.length === 0 ? (
          <p className="text-xs text-muted-foreground italic">{t('fuhrpark.detail.noEntries')}</p>
        ) : (
          <div className="space-y-1.5">
            {vehicleMaintenance.map((r) => (
              <div key={r.id} className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2">
                <div className="flex items-center gap-2 min-w-0">
                  <span className={`shrink-0 rounded-full px-2 py-0.5 text-[9px] font-medium ${maintenanceTypeColors[r.type] ?? 'bg-secondary text-muted-foreground'}`}>
                    {maintenanceTypeLabels[r.type] ? t(maintenanceTypeLabels[r.type]) : r.type}
                  </span>
                  <span className="text-xs text-muted-foreground truncate">{r.notes || '—'}</span>
                </div>
                <div className="text-right shrink-0 ml-2">
                  <p className="text-xs font-medium text-foreground">{currency} {formatAmount(r.cost)}</p>
                  <p className="text-[10px] text-muted-foreground">{formatDate(r.date)}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Recent Fuel */}
      <section className="space-y-2">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
          {t('fuhrpark.detail.recentFuel')}
        </h4>
        {vehicleFuel.length === 0 ? (
          <p className="text-xs text-muted-foreground italic">{t('fuhrpark.detail.noFuelEntries')}</p>
        ) : (
          <div className="space-y-1.5">
            {vehicleFuel.map((r) => (
              <div key={r.id} className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2">
                <div className="flex items-center gap-2">
                  <Droplets className="h-3.5 w-3.5 text-info shrink-0" />
                  <span className="text-xs text-muted-foreground">{r.liters.toLocaleString('de-DE', { minimumFractionDigits: 1 })} L</span>
                </div>
                <div className="text-right">
                  <p className="text-xs font-medium text-foreground">{currency} {formatAmount(r.cost)}</p>
                  <p className="text-[10px] text-muted-foreground">{formatDate(r.date)}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Wave 9: Letzte Fahrten (Logbook) */}
      <section className="space-y-2">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
          <BookOpen className="h-3 w-3" />
          {t('fuhrpark.detail.recentTrips')}
        </h4>
        {vehicleLogbook.length === 0 ? (
          <p className="text-xs text-muted-foreground italic">{t('fuhrpark.detail.noTrips')}</p>
        ) : (
          <div className="space-y-1.5">
            {vehicleLogbook.map((entry) => (
              <div key={entry.id} className="rounded-md border border-border-muted px-3 py-2">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-[10px] text-muted-foreground">{formatDate(entry.date)}</span>
                  <span className={`rounded-full px-2 py-0.5 text-[9px] font-medium ${
                    entry.isPrivate ? 'bg-warning-light text-warning' : 'bg-info-light text-info'
                  }`}>
                    {entry.isPrivate ? t('fuhrpark.logbook.private') : t('fuhrpark.logbook.business')}
                  </span>
                </div>
                <p className="text-xs text-foreground truncate">
                  {entry.startLocation} → {entry.endLocation}
                </p>
                <p className="text-[10px] text-muted-foreground mt-0.5">{entry.km} km — {entry.purpose}</p>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Wave 9: Dokumente */}
      <section className="space-y-2">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
          <FileText className="h-3 w-3" />
          {t('fuhrpark.detail.documents')}
        </h4>
        {vehicleDocs.length === 0 ? (
          <p className="text-xs text-muted-foreground italic">{t('fuhrpark.detail.noDocs')}</p>
        ) : (
          <div className="space-y-1.5">
            {vehicleDocs.map((doc) => {
              const expiryStatus = doc.expiryDate ? getDateStatus(doc.expiryDate, reminderLeadDays) : null
              return (
                <div key={doc.id} className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2">
                  <div className="flex items-center gap-2 min-w-0">
                    <FileText className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                    <div className="min-w-0">
                      <div className="flex items-center gap-1.5">
                        <span className={`shrink-0 rounded-full px-2 py-0.5 text-[9px] font-medium ${documentTypeColors[doc.type] ?? 'bg-secondary text-muted-foreground'}`}>
                          {documentTypeLabels[doc.type] ? t(documentTypeLabels[doc.type]) : doc.type}
                        </span>
                      </div>
                      <p className="text-xs text-foreground truncate mt-0.5">{doc.name}</p>
                    </div>
                  </div>
                  <div className="text-right shrink-0 ml-2">
                    <p className="text-[10px] text-muted-foreground">{formatDate(doc.uploadDate)}</p>
                    {doc.expiryDate && expiryStatus && (
                      <p className={`text-[10px] ${statusTextColor(expiryStatus)}`}>
                        {expiryStatus === 'overdue' && <AlertTriangle className="inline h-2.5 w-2.5 mr-0.5" />}
                        {t('fuhrpark.detail.docExpiry')} {formatDate(doc.expiryDate)}
                      </p>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </section>

      {/* Wave 9: Schadensmeldungen */}
      <section className="space-y-2">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
          <Camera className="h-3 w-3" />
          {t('fuhrpark.detail.damageReports')}
        </h4>
        {vehicleDamage.length === 0 ? (
          <p className="text-xs text-muted-foreground italic">{t('fuhrpark.detail.noDamage')}</p>
        ) : (
          <div className="space-y-1.5">
            {vehicleDamage.map((dmg) => (
              <div key={dmg.id} className="rounded-md border border-border-muted px-3 py-2">
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center gap-1.5">
                    <span className={`rounded-full px-2 py-0.5 text-[9px] font-medium ${severityColors[dmg.severity] ?? 'bg-secondary text-muted-foreground'}`}>
                      {severityLabels[dmg.severity] ? t(severityLabels[dmg.severity]) : dmg.severity}
                    </span>
                    <span className={`rounded-full px-2 py-0.5 text-[9px] font-medium ${damageStatusColors[dmg.status] ?? 'bg-secondary text-muted-foreground'}`}>
                      {damageStatusLabels[dmg.status] ? t(damageStatusLabels[dmg.status]) : dmg.status}
                    </span>
                  </div>
                  <span className="text-[10px] text-muted-foreground">{formatDate(dmg.date)}</span>
                </div>
                <p className="text-xs text-foreground">{dmg.description}</p>
                <div className="flex items-center gap-2 mt-1 text-[10px] text-muted-foreground">
                  <span>{dmg.location}</span>
                  {dmg.photoCount > 0 && (
                    <span className="flex items-center gap-0.5">
                      <Camera className="h-2.5 w-2.5" /> {dmg.photoCount}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Action Buttons — only render when at least one action is permitted */}
      {(canCreateService || canCreateFuel || canCreateDamage) && (
        <div className="flex gap-2 pt-1 flex-wrap">
          {canCreateService && (
            <button
              onClick={onAddMaintenance}
              className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <Wrench className="h-3.5 w-3.5" />
              {t('fuhrpark.action.addMaintenance')}
            </button>
          )}
          {canCreateFuel && (
            <button
              onClick={onAddFuel}
              className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <Fuel className="h-3.5 w-3.5" />
              {t('fuhrpark.action.addFuel')}
            </button>
          )}
          {canCreateDamage && (
            <button
              onClick={onAddDamage}
              className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <Camera className="h-3.5 w-3.5" />
              {t('fuhrpark.action.reportDamage')}
            </button>
          )}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main Page Component
// ---------------------------------------------------------------------------

export default function FuhrparkPage() {
  const { t } = useTranslation()

  // ── Module settings (personal prefs + tenant policy) ──
  const defaultTab = useFuhrparkPrefsStore((s) => s.defaultTab)
  const showTireReminder = useFuhrparkPrefsStore((s) => s.showTireReminder)
  const defaultTripCategory = useFuhrparkPrefsStore((s) => s.defaultTripCategory)
  const reminderLeadDays = useFuhrparkTenantStore((s) => s.reminderLeadDays)
  const currency = useFuhrparkTenantStore((s) => s.currency)
  const defaultFuelType = useFuhrparkTenantStore((s) => s.defaultFuelType)
  const privateTripsEnabled = useFuhrparkTenantStore((s) => s.privateTripsEnabled)

  // ── RBAC R-3 capability checks (default hidden per gating convention) ──
  const { has: capHas, ready: capReady } = useCapabilitySet()
  const canManageVehicle = useHasCapability('fuhrpark:vehicle:manage')
  const canCreateService = useHasCapability('fuhrpark:service:create')
  const canCreateFuel = useHasCapability('fuhrpark:fuel:create')
  const canCreateTrip = useHasCapability('fuhrpark:trip:create')
  const canCreateDamage = useHasCapability('fuhrpark:damage:create')
  const canExportFuhrpark = useHasCapability('fuhrpark:export:run')

  // Tab capability map (tab = read key rule)
  const TAB_CAPABILITY: Record<TabKey, string> = {
    fahrzeuge: 'fuhrpark:vehicle:read',
    wartung: 'fuhrpark:service:read',
    tankprotokoll: 'fuhrpark:fuel:read',
    fahrtenbuch: 'fuhrpark:trip:read',
    tracking: 'fuhrpark:gps:read',
  }
  const visibleTabs = (Object.keys(TAB_CAPABILITY) as TabKey[]).filter(
    (key) => !capReady || capHas(TAB_CAPABILITY[key]),
  )

  // ---------------------------------------------------------------------------
  // Selected vehicle state (needed before hook calls so useVehicleDocuments can use it)
  // ---------------------------------------------------------------------------
  const [selectedVehicle, setSelectedVehicle] = useState<Vehicle | null>(null)

  // ---------------------------------------------------------------------------
  // Queries
  // ---------------------------------------------------------------------------
  const { data: vehiclesData, isLoading: vehiclesLoading, error: vehiclesError } = useVehiclesList()
  const { data: servicesData, isLoading: servicesLoading } = useServicesList()
  const { data: fuelData, isLoading: fuelLoading } = useFuelLogs()
  const { data: tripData, isLoading: tripLoading } = useTripLogs()
  const { data: damagesData, isLoading: _damagesLoading } = useDamagesList()
  const { data: routesData, isLoading: routesLoading, refetch: refetchRoutes } = useVehicleRoutes()
  const { data: documentsData } = useVehicleDocuments(selectedVehicle?.id ?? '')

  // Adapt API data to UI store types
  const vehicles = useMemo(() => (vehiclesData?.vehicles ?? []).map(adaptVehicle), [vehiclesData])
  const maintenanceRecords = useMemo(() => (servicesData?.services ?? []).map(adaptService), [servicesData])
  const fuelRecords = useMemo(() => (fuelData?.fuel_logs ?? []).map(adaptFuelLog), [fuelData])
  const logbookEntries = useMemo(() => (tripData?.trip_logs ?? []).map(adaptTripLog), [tripData])
  const damageReports = useMemo(() => (damagesData?.damages ?? []).map(adaptDamage), [damagesData])
  const vehicleRoutes: VehicleRoute[] = useMemo(
    () =>
      (routesData?.routes ?? []).map((r) => ({
        id: `${r.vehicle_id}-${r.date}`,
        vehicleId: r.vehicle_id,
        vehicleName: r.vehicle_name,
        date: r.date,
        positions: r.positions.map((p) => ({
          lat: p.lat,
          lng: p.lng,
          address: `${p.lat.toFixed(4)}, ${p.lng.toFixed(4)}`,
          timestamp: p.recorded_at,
        })),
        dailyKm: r.daily_km,
        status: r.status,
        driver: r.driver,
      })),
    [routesData]
  )
  const vehicleDocuments: VehicleDocument[] = useMemo(
    () =>
      (documentsData?.documents ?? []).map((d) => ({
        id: d.id,
        vehicleId: d.vehicle_id,
        type: d.doc_type as VehicleDocument['type'],
        name: d.name,
        uploadDate: d.upload_date,
        expiryDate: d.expiry_date,
      })),
    [documentsData]
  )

  // ---------------------------------------------------------------------------
  // Mutations
  // ---------------------------------------------------------------------------
  const createVehicleMutation = useCreateVehicle()
  const scheduleServiceMutation = useScheduleService()
  const createFuelLogMutation = useCreateFuelLog()
  const reportDamageMutation = useReportDamage()
  const createTripLogMutation = useCreateTripLog()

  const [tab, setTab] = useState<TabKey>(defaultTab)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<'all' | 'car' | 'van' | 'truck'>('all')

  // Fallback: wenn der aktive Tab durch RBAC nicht mehr sichtbar ist, zum ersten sichtbaren wechseln
  useEffect(() => {
    if (!capReady) return
    if (visibleTabs.length > 0 && !visibleTabs.includes(tab)) setTab(visibleTabs[0])
  }, [capReady, visibleTabs, tab])
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all')
  const [dialog, setDialog] = useState<DialogKey>(null)
  const [dialogPreselectedVehicleId, setDialogPreselectedVehicleId] = useState<string | undefined>()
  const [expandedRouteId, setExpandedRouteId] = useState<string | null>(null)
  const [trackingRefreshing, setTrackingRefreshing] = useState(false)
  // Row-level detail modals (Wartung / Tanken / Fahrtenbuch)
  const [maintenanceDetail, setMaintenanceDetail] = useState<MaintenanceRecord | null>(null)
  const [fuelDetail, setFuelDetail] = useState<FuelRecord | null>(null)
  const [tripDetail, setTripDetail] = useState<LogbookEntry | null>(null)
  // Vehicle list sorting (field + direction, shared SortMenu)
  const [sortField, setSortField] = useState('plate')
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc')

  // License plate lookup — the list adapters only carry vehicle_id as the
  // plate placeholder, so tables/modals/exports resolve the real plate here.
  const plateByVehicleId = useMemo(
    () => new Map(vehicles.map((v) => [v.id, v.licensePlate])),
    [vehicles],
  )
  const plateFor = useCallback(
    (vehicleId: string, fallback: string) => plateByVehicleId.get(vehicleId) ?? fallback,
    [plateByVehicleId],
  )

  // Filtered vehicles with search + type + status filters, sorted via SortMenu
  const filteredVehicles = useMemo(() => {
    const filtered = vehicles.filter((v) => {
      // Type filter
      if (typeFilter !== 'all' && v.type !== typeFilter) return false
      // Status filter
      if (statusFilter === 'active' && !v.isActive) return false
      if (statusFilter === 'inactive' && v.isActive) return false
      // Search
      if (!search) return true
      const q = search.toLowerCase()
      return (
        v.licensePlate.toLowerCase().includes(q) ||
        v.make.toLowerCase().includes(q) ||
        v.model.toLowerCase().includes(q) ||
        v.currentDriver.toLowerCase().includes(q)
      )
    })
    const dir = sortDirection === 'asc' ? 1 : -1
    return [...filtered].sort((a, b) => {
      if (sortField === 'make') return dir * `${a.make} ${a.model}`.localeCompare(`${b.make} ${b.model}`, 'de')
      if (sortField === 'mileage') return dir * (a.mileage - b.mileage)
      if (sortField === 'inspection') return dir * a.nextInspection.localeCompare(b.nextInspection)
      return dir * a.licensePlate.localeCompare(b.licensePlate, 'de')
    })
  }, [vehicles, search, typeFilter, statusFilter, sortField, sortDirection])

  // Filtered maintenance with search
  const filteredMaintenance = useMemo(() => {
    const sorted = [...maintenanceRecords].sort(
      (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime()
    )
    if (!search) return sorted
    const q = search.toLowerCase()
    return sorted.filter(
      (r) =>
        r.vehiclePlate.toLowerCase().includes(q) ||
        r.notes.toLowerCase().includes(q) ||
        maintenanceTypeLabels[r.type]?.toLowerCase().includes(q)
    )
  }, [maintenanceRecords, search])

  // Filtered fuel with search
  const filteredFuel = useMemo(() => {
    const sorted = [...fuelRecords].sort(
      (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime()
    )
    if (!search) return sorted
    const q = search.toLowerCase()
    return sorted.filter((r) => r.vehiclePlate.toLowerCase().includes(q))
  }, [fuelRecords, search])

  // Filtered logbook with search
  const filteredLogbook = useMemo(() => {
    const sorted = [...logbookEntries].sort(
      (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime()
    )
    if (!search) return sorted
    const q = search.toLowerCase()
    return sorted.filter(
      (e) =>
        e.vehiclePlate.toLowerCase().includes(q) ||
        e.startLocation.toLowerCase().includes(q) ||
        e.endLocation.toLowerCase().includes(q) ||
        e.purpose.toLowerCase().includes(q) ||
        e.driver.toLowerCase().includes(q)
    )
  }, [logbookEntries, search])

  // Logbook stats
  const logbookBusinessKm = useMemo(
    () => logbookEntries.filter((e) => !e.isPrivate).reduce((s, e) => s + e.km, 0),
    [logbookEntries]
  )
  const logbookPrivateKm = useMemo(
    () => logbookEntries.filter((e) => e.isPrivate).reduce((s, e) => s + e.km, 0),
    [logbookEntries]
  )
  const logbookTotalKm = logbookBusinessKm + logbookPrivateKm
  const logbookPrivatePct = logbookTotalKm > 0 ? ((logbookPrivateKm / logbookTotalKm) * 100).toFixed(1) : '0.0'

  // Monthly fuel cost
  const now = new Date()
  const currentMonth = now.getMonth()
  const currentYear = now.getFullYear()
  const monthlyFuelCost = useMemo(
    () =>
      fuelRecords
        .filter((r) => {
          const d = new Date(r.date)
          return d.getMonth() === currentMonth && d.getFullYear() === currentYear
        })
        .reduce((sum, r) => sum + r.cost, 0),
    [fuelRecords, currentMonth, currentYear]
  )

  // Counts for tab badges (reminder window from tenant settings)
  const activeVehicleCount = vehicles.filter((v) => v.isActive).length
  const urgentCount = vehicles.filter((v) => {
    const i = getDateStatus(v.nextInspection, reminderLeadDays)
    const ins = getDateStatus(v.insuranceExpiry, reminderLeadDays)
    return i === 'overdue' || i === 'soon' || ins === 'overdue' || ins === 'soon'
  }).length

  // Dialog handlers
  const openAddVehicle = useCallback(() => {
    setDialog('addVehicle')
  }, [])

  const openAddMaintenanceFromPanel = useCallback((vehicleId: string) => {
    setDialogPreselectedVehicleId(vehicleId)
    setDialog('addMaintenance')
  }, [])

  const openAddFuelFromPanel = useCallback((vehicleId: string) => {
    setDialogPreselectedVehicleId(vehicleId)
    setDialog('addFuel')
  }, [])

  const openAddMaintenanceGlobal = useCallback(() => {
    setDialogPreselectedVehicleId(undefined)
    setDialog('addMaintenance')
  }, [])

  const openAddFuelGlobal = useCallback(() => {
    setDialogPreselectedVehicleId(undefined)
    setDialog('addFuel')
  }, [])

  const openAddDamageFromPanel = useCallback((vehicleId: string) => {
    setDialogPreselectedVehicleId(vehicleId)
    setDialog('addDamage')
  }, [])

  const handleSaveVehicle = useCallback(
    (vehicle: Vehicle) => {
      createVehicleMutation.mutate(
        {
          license_plate: vehicle.licensePlate,
          make: vehicle.make,
          model: vehicle.model,
          year: vehicle.year,
          fuel_type: defaultFuelType,
          mileage_km: vehicle.mileage,
          tuev_due_date: vehicle.nextInspection !== '2099-01-01' ? vehicle.nextInspection : undefined,
        },
        {
          onSuccess: () => {
            setDialog(null)
            toast.success(t('fuhrpark.toast.vehicleAdded', { plate: vehicle.licensePlate }))
          },
        }
      )
    },
    [createVehicleMutation, defaultFuelType, t]
  )

  const handleSaveMaintenance = useCallback(
    (record: MaintenanceRecord) => {
      scheduleServiceMutation.mutate(
        {
          vehicleId: record.vehicleId,
          service_type: record.type,
          scheduled_at: record.date,
          cost_cents: Math.round(record.cost * 100),
          mileage_km: record.mileage,
          notes: record.notes || undefined,
        },
        {
          onSuccess: () => {
            setDialog(null)
            toast.success(t('fuhrpark.toast.maintenanceAdded', { plate: record.vehiclePlate }))
          },
        }
      )
    },
    [scheduleServiceMutation, t]
  )

  const handleSaveFuel = useCallback(
    (record: FuelRecord) => {
      createFuelLogMutation.mutate(
        {
          vehicleId: record.vehicleId,
          data: {
            vehicle_id: record.vehicleId,
            date: record.date,
            liters: record.liters,
            cost_cents: Math.round(record.cost * 100),
            mileage_km: record.mileage,
            fuel_type: defaultFuelType,
          },
        },
        {
          onSuccess: () => {
            setDialog(null)
            toast.success(t('fuhrpark.toast.fuelAdded', { plate: record.vehiclePlate }))
          },
        }
      )
    },
    [createFuelLogMutation, defaultFuelType, t]
  )

  // Save a manual trip-log entry (wires the previously unused createTripLog)
  const handleSaveTrip = useCallback(
    (trip: {
      vehicleId: string
      date: string
      startLocation: string
      endLocation: string
      purpose: string
      startKm: number
      endKm: number
      isPrivate: boolean
      driver: string
    }) => {
      createTripLogMutation.mutate(
        {
          vehicleId: trip.vehicleId,
          data: {
            vehicle_id: trip.vehicleId,
            date: trip.date,
            start_location: trip.startLocation,
            end_location: trip.endLocation,
            purpose: trip.purpose,
            start_km: trip.startKm,
            end_km: trip.endKm,
            is_private: trip.isPrivate,
            driver_name: trip.driver,
          },
        },
        {
          onSuccess: () => {
            setDialog(null)
            toast.success(t('fuhrpark.toast.tripAdded', { km: trip.endKm - trip.startKm }))
          },
        }
      )
    },
    [createTripLogMutation, t]
  )

  // Exports — real downloads (the old getExportUrl pointed at a dead endpoint)
  const handleVehiclesCsvExport = useCallback(() => {
    const typeLabels: Record<string, string> = {
      car: t('fuhrpark.vehicleType.car'),
      van: t('fuhrpark.vehicleType.van'),
      truck: t('fuhrpark.vehicleType.truck'),
    }
    const csv = buildVehiclesCsv(filteredVehicles, typeLabels)
    downloadBlob(
      new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' }),
      `fahrzeuge-${csvDateStamp()}.csv`,
    )
    toast.success(t('fuhrpark.export.vehiclesCsvToast', { count: filteredVehicles.length }))
  }, [filteredVehicles, t])

  const handleLogbookCsvExport = useCallback(() => {
    const csv = buildLogbookCsv(filteredLogbook, plateByVehicleId, {
      business: t('fuhrpark.logbook.business'),
      privateTrip: t('fuhrpark.logbook.private'),
    })
    downloadBlob(
      new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' }),
      `fahrtenbuch-${csvDateStamp()}.csv`,
    )
    toast.success(t('fuhrpark.export.logbookCsvToast', { count: filteredLogbook.length }))
  }, [filteredLogbook, plateByVehicleId, t])

  const handleLogbookPdfExport = useCallback(() => {
    const blob = buildLogbookPdf(filteredLogbook, plateByVehicleId, {
      title: t('fuhrpark.export.logbookPdfTitle'),
      period: t('fuhrpark.export.period'),
      businessKm: t('fuhrpark.logbook.businessKm'),
      privateKm: t('fuhrpark.logbook.privateKm'),
      totalKm: t('fuhrpark.export.totalKm'),
      business: t('fuhrpark.logbook.business'),
      privateTrip: t('fuhrpark.logbook.private'),
      noEntries: t('fuhrpark.empty.noLogbook.title'),
    })
    downloadBlob(blob, `fahrtenbuch-${csvDateStamp()}.pdf`)
    toast.success(t('fuhrpark.export.logbookPdfToast'))
  }, [filteredLogbook, plateByVehicleId, t])

  // Tracking stats
  const trackingDrivingCount = vehicleRoutes.filter((r) => r.status === 'driving').length
  const trackingParkedCount = vehicleRoutes.filter((r) => r.status === 'parked').length
  const trackingUnknownCount = vehicleRoutes.filter((r) => r.status === 'unknown').length
  const trackingTotalKm = vehicleRoutes.reduce((sum, r) => sum + r.dailyKm, 0)

  // Filtered tracking routes
  const filteredRoutes = useMemo(() => {
    if (!search) return vehicleRoutes
    const q = search.toLowerCase()
    return vehicleRoutes.filter(
      (r) =>
        r.vehicleName.toLowerCase().includes(q) ||
        r.driver.toLowerCase().includes(q) ||
        r.positions.some((p) => p.address.toLowerCase().includes(q))
    )
  }, [vehicleRoutes, search])

  // Handle refresh tracking
  const handleRefreshTracking = useCallback(() => {
    setTrackingRefreshing(true)
    refetchRoutes().finally(() => setTrackingRefreshing(false))
  }, [refetchRoutes])

  // Search placeholder based on tab
  const searchPlaceholder =
    tab === 'fahrzeuge'
      ? t('fuhrpark.search.vehicles')
      : tab === 'wartung'
        ? t('fuhrpark.search.maintenance')
        : tab === 'tracking'
          ? t('fuhrpark.search.tracking')
          : tab === 'fahrtenbuch'
            ? t('fuhrpark.search.logbook')
            : t('fuhrpark.search.fuel')

  // moduleEmpty: kein sichtbarer Tab nach Permissions-Load
  if (capReady && visibleTabs.length === 0) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <PageHeader
          title={t('fuhrpark.page.title')}
          description={t('rbac.gate.moduleEmpty')}
          icon={Truck}
          moduleId="fuhrpark"
          className="mb-6"
          actions={<RestrictedModeBadge module="fuhrpark" />}
        />
        <EmptyState icon={Car} title={t('rbac.gate.moduleEmpty')} description={t('rbac.gate.noPermission')} />
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <PageHeader
        title={t('fuhrpark.page.title')}
        description={urgentCount > 0
          ? t('fuhrpark.page.descriptionWithWarning', { count: activeVehicleCount, urgent: urgentCount })
          : t('fuhrpark.page.description', { count: activeVehicleCount })
        }
        icon={Truck}
        moduleId="fuhrpark"
        className="mb-6"
        actions={
          <div className="flex items-center gap-2">
            <RestrictedModeBadge module="fuhrpark" />
            {tab === 'wartung' && canCreateService && (
              <button
                onClick={openAddMaintenanceGlobal}
                className="flex items-center gap-2 rounded-xl border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <Wrench className="h-4 w-4" />
                {t('fuhrpark.action.addMaintenance')}
              </button>
            )}
            {tab === 'tankprotokoll' && canCreateFuel && (
              <button
                onClick={openAddFuelGlobal}
                className="flex items-center gap-2 rounded-xl border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <Fuel className="h-4 w-4" />
                {t('fuhrpark.action.addFuel')}
              </button>
            )}
            {tab === 'fahrzeuge' && canExportFuhrpark && (
              <button
                onClick={handleVehiclesCsvExport}
                className="flex items-center gap-2 rounded-xl border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <FileSpreadsheet className="h-4 w-4" />
                {t('fuhrpark.action.exportVehicles')}
              </button>
            )}
            {tab === 'fahrtenbuch' && (
              <>
                {canExportFuhrpark && (
                  <button
                    onClick={handleLogbookCsvExport}
                    className="flex items-center gap-2 rounded-xl border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
                  >
                    <FileSpreadsheet className="h-4 w-4" />
                    {t('fuhrpark.action.exportCsv')}
                  </button>
                )}
                {canExportFuhrpark && (
                  <button
                    onClick={handleLogbookPdfExport}
                    className="flex items-center gap-2 rounded-xl border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
                  >
                    <FileDown className="h-4 w-4" />
                    {t('fuhrpark.action.exportPdf')}
                  </button>
                )}
                {canCreateTrip && (
                  <button
                    onClick={() => setDialog('addTrip')}
                    className="flex items-center gap-2 rounded-xl border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
                  >
                    <BookOpen className="h-4 w-4" />
                    {t('fuhrpark.action.addTrip')}
                  </button>
                )}
              </>
            )}
            {tab === 'tracking' && capHas('fuhrpark:gps:read') && (
              <button
                onClick={handleRefreshTracking}
                disabled={trackingRefreshing}
                className="flex items-center gap-2 rounded-xl border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors disabled:opacity-50"
              >
                <RefreshCw className={`h-4 w-4 ${trackingRefreshing ? 'animate-spin' : ''}`} />
                {t('fuhrpark.action.refresh')}
              </button>
            )}
            {canManageVehicle && (
              <button
                onClick={openAddVehicle}
                className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Plus className="h-4 w-4" />
                {t('fuhrpark.action.addVehicle')}
              </button>
            )}
          </div>
        }
      />

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'fahrzeuge' as const, label: t('fuhrpark.tab.fahrzeuge', { count: vehicles.length }), icon: null },
          { key: 'wartung' as const, label: t('fuhrpark.tab.wartung', { count: maintenanceRecords.length }), icon: null },
          { key: 'tankprotokoll' as const, label: t('fuhrpark.tab.tankprotokoll', { count: fuelRecords.length }), icon: null },
          { key: 'fahrtenbuch' as const, label: t('fuhrpark.tab.fahrtenbuch', { count: logbookEntries.length }), icon: BookOpen },
          { key: 'tracking' as const, label: t('fuhrpark.tab.tracking'), icon: MapPin },
        ]).filter((tabItem) => visibleTabs.includes(tabItem.key)).map((tabItem) => (
          <button
            key={tabItem.key}
            onClick={() => { setTab(tabItem.key); setSearch(''); setExpandedRouteId(null) }}
            className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === tabItem.key
                ? 'border-primary text-primary font-medium tab-accent-active'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {tabItem.icon && <tabItem.icon className="h-3.5 w-3.5" />}
            {tabItem.label}
          </button>
        ))}
      </div>

      {/* Search + Filters */}
      <div className="flex items-center gap-3 mb-4 flex-wrap">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder={searchPlaceholder}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
        </div>

        {tab === 'fahrzeuge' && (
          <>
            {/* Type filter */}
            <select
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value as typeof typeFilter)}
              className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              <option value="all">{t('fuhrpark.filter.allTypes')}</option>
              <option value="car">{t('fuhrpark.vehicleType.car')}</option>
              <option value="van">{t('fuhrpark.vehicleType.van')}</option>
              <option value="truck">{t('fuhrpark.vehicleType.truck')}</option>
            </select>

            {/* Status filter */}
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as typeof statusFilter)}
              className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              <option value="all">{t('fuhrpark.filter.allStatus')}</option>
              <option value="active">{t('fuhrpark.filter.active')}</option>
              <option value="inactive">{t('fuhrpark.filter.inactive')}</option>
            </select>

            <SortMenu
              options={[
                { value: 'plate', label: t('fuhrpark.sort.plate') },
                { value: 'make', label: t('fuhrpark.sort.make') },
                { value: 'mileage', label: t('fuhrpark.sort.mileage') },
                { value: 'inspection', label: t('fuhrpark.sort.inspection') },
              ]}
              field={sortField}
              direction={sortDirection}
              onChange={(f, d) => { setSortField(f); setSortDirection(d) }}
            />
          </>
        )}
      </div>

      {/* ================================================================= */}
      {/* Fahrzeuge Tab                                                      */}
      {/* ================================================================= */}
      {tab === 'fahrzeuge' && (
        <>
          {vehiclesLoading && (
            <div className="p-6 text-center text-muted-foreground text-sm">{t('fuhrpark.loading.vehicles')}</div>
          )}
          {vehiclesError && !vehiclesLoading && (
            <div className="p-6 text-center text-error text-sm">{t('fuhrpark.error.vehicles')}</div>
          )}
          {!vehiclesLoading && !vehiclesError && (
          <>
          {/* Wave 9: Reifenwechsel-Erinnerung Banner (toggleable via personal prefs) */}
          {showTireReminder && (
            <div className="mb-4 flex items-center gap-3 rounded-lg border border-warning/30 bg-warning-light px-4 py-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-warning/20">
                <RefreshCw className="h-4 w-4 text-warning" />
              </div>
              <div className="flex-1">
                <p className="text-sm font-medium text-foreground">{t('fuhrpark.tireReminder.title')}</p>
                <p className="text-xs text-muted-foreground">{t('fuhrpark.tireReminder.description', { count: vehicles.filter((v) => v.isActive).length })}</p>
              </div>
            </div>
          )}

          {filteredVehicles.length === 0 ? (
            <EmptyState
              icon={Car}
              title={t('fuhrpark.empty.noVehicles.title')}
              description={search || typeFilter !== 'all' || statusFilter !== 'all'
                ? t('fuhrpark.empty.noVehicles.descriptionFilter')
                : t('fuhrpark.empty.noVehicles.descriptionEmpty')
              }
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {filteredVehicles.map((vehicle) => {
                const inspectionStatus = getDateStatus(vehicle.nextInspection, reminderLeadDays)
                const insuranceStatus = getDateStatus(vehicle.insuranceExpiry, reminderLeadDays)

                return (
                  <button
                    key={vehicle.id}
                    onClick={() => setSelectedVehicle(vehicle)}
                    className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] text-left w-full"
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div>
                        <p className="text-lg font-bold text-foreground tracking-wider font-mono">
                          {vehicle.licensePlate}
                        </p>
                        <p className="text-sm text-muted-foreground">
                          {vehicle.make} {vehicle.model} &middot; {vehicle.year}
                        </p>
                      </div>
                      <div className="flex flex-col items-end gap-1.5">
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${vehicleTypeColors[vehicle.type] ?? 'bg-secondary text-muted-foreground'}`}>
                          {vehicleTypeLabels[vehicle.type] ? t(vehicleTypeLabels[vehicle.type]) : vehicle.type}
                        </span>
                        {!vehicle.isActive && (
                          <span className="rounded-full bg-error-light px-2 py-0.5 text-[10px] font-medium text-error">
                            {t('fuhrpark.filter.inactive')}
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="space-y-1.5 text-xs text-muted-foreground mb-3">
                      <div className="flex items-center gap-2">
                        <User className="h-3 w-3" />
                        <span>{vehicle.currentDriver || t('fuhrpark.vehicle.noDriver')}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <Gauge className="h-3 w-3" />
                        <span>{formatKm(vehicle.mileage)} km</span>
                      </div>
                    </div>

                    {/* Status dots row */}
                    <div className="flex items-center gap-4 border-t border-border-muted pt-2.5">
                      <div className="flex items-center gap-1.5">
                        <span className={`h-2 w-2 rounded-full ${statusDotColor(inspectionStatus)}`} />
                        <span className={`text-[11px] ${statusTextColor(inspectionStatus) || 'text-muted-foreground'}`}>
                          {t('fuhrpark.vehicle.inspection')} {formatDueDate(vehicle.nextInspection)}
                        </span>
                      </div>
                      <div className="flex items-center gap-1.5">
                        <span className={`h-2 w-2 rounded-full ${statusDotColor(insuranceStatus)}`} />
                        <span className={`text-[11px] ${statusTextColor(insuranceStatus) || 'text-muted-foreground'}`}>
                          {t('fuhrpark.vehicle.insurance')} {formatDueDate(vehicle.insuranceExpiry)}
                        </span>
                      </div>
                    </div>
                  </button>
                )
              })}
            </div>
          )}
          </>
          )}
        </>
      )}

      {/* ================================================================= */}
      {/* Wartung Tab                                                        */}
      {/* ================================================================= */}
      {tab === 'wartung' && (
        <>
          {servicesLoading && (
            <div className="p-6 text-center text-muted-foreground text-sm">{t('fuhrpark.loading.maintenance')}</div>
          )}
          {!servicesLoading && (
          <>
          {/* Summary cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
            {(['service', 'repair', 'inspection', 'tires'] as const).map((mType) => {
              const count = maintenanceRecords.filter((r) => r.type === mType).length
              const totalCost = maintenanceRecords
                .filter((r) => r.type === mType)
                .reduce((sum, r) => sum + r.cost, 0)
              return (
                <div key={mType} className="rounded-lg border border-border bg-card p-3">
                  <div className="flex items-center gap-2 mb-1">
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${maintenanceTypeColors[mType]}`}>
                      {maintenanceTypeLabels[mType] ? t(maintenanceTypeLabels[mType]) : mType}
                    </span>
                  </div>
                  <p className="text-lg font-semibold text-foreground">{count}</p>
                  <p className="text-[11px] text-muted-foreground">{currency} {formatAmount(totalCost)}</p>
                </div>
              )
            })}
          </div>

          {filteredMaintenance.length === 0 ? (
            <EmptyState
              icon={Wrench}
              title={t('fuhrpark.empty.noMaintenance.title')}
              description={search
                ? t('fuhrpark.empty.noMaintenance.descriptionFilter')
                : t('fuhrpark.empty.noMaintenance.descriptionEmpty')
              }
            />
          ) : (
            <div className="rounded-lg border border-border bg-card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border">
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('fuhrpark.table.date')}</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('fuhrpark.table.vehicle')}</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('fuhrpark.table.type')}</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">{t('fuhrpark.table.mileage')}</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">{t('fuhrpark.table.cost', { currency })}</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('fuhrpark.table.notes')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredMaintenance.map((record) => (
                      <tr
                        key={record.id}
                        role="button"
                        tabIndex={0}
                        aria-label={`${plateFor(record.vehicleId, record.vehiclePlate)} — ${formatDate(record.date)}`}
                        onClick={() => setMaintenanceDetail(record)}
                        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setMaintenanceDetail(record) } }}
                        className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer focus-visible:outline-none focus-visible:bg-secondary/50"
                      >
                        <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                          {formatDate(record.date)}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs font-medium text-foreground whitespace-nowrap">
                          {plateFor(record.vehicleId, record.vehiclePlate)}
                        </td>
                        <td className="px-4 py-3">
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${maintenanceTypeColors[record.type] ?? 'bg-secondary text-muted-foreground'}`}>
                            {maintenanceTypeLabels[record.type] ? t(maintenanceTypeLabels[record.type]) : record.type}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground text-right tabular-nums">
                          {formatKm(record.mileage)}
                        </td>
                        <td className="px-4 py-3 text-xs text-foreground font-medium text-right tabular-nums">
                          {formatAmount(record.cost)}
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground max-w-[240px] truncate">
                          {record.notes || '\u2014'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
          </>
          )}
        </>
      )}

      {/* ================================================================= */}
      {/* Tankprotokoll Tab                                                  */}
      {/* ================================================================= */}
      {tab === 'tankprotokoll' && (
        <>
          {fuelLoading && (
            <div className="p-6 text-center text-muted-foreground text-sm">{t('fuhrpark.loading.fuel')}</div>
          )}
          {!fuelLoading && (
          <>
          {/* Monthly summary */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-4">
            <div className="rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-warning-light">
                  <Fuel className="h-5 w-5 text-warning" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('fuhrpark.fuel.costThisMonth')}</p>
                  <p className="text-xl font-semibold text-foreground">{currency} {formatAmount(monthlyFuelCost)}</p>
                </div>
              </div>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info-light">
                  <Droplets className="h-5 w-5 text-info" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('fuhrpark.fuel.litersThisMonth')}</p>
                  <p className="text-xl font-semibold text-foreground">
                    {fuelRecords
                      .filter((r) => {
                        const d = new Date(r.date)
                        return d.getMonth() === currentMonth && d.getFullYear() === currentYear
                      })
                      .reduce((sum, r) => sum + r.liters, 0)
                      .toLocaleString('de-DE', { minimumFractionDigits: 1 })} L
                  </p>
                </div>
              </div>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                  <FileText className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('fuhrpark.fuel.totalRefuels')}</p>
                  <p className="text-xl font-semibold text-foreground">{fuelRecords.length}</p>
                </div>
              </div>
            </div>
          </div>

          {filteredFuel.length === 0 ? (
            <EmptyState
              icon={Fuel}
              title={t('fuhrpark.empty.noFuel.title')}
              description={search
                ? t('fuhrpark.empty.noFuel.descriptionFilter')
                : t('fuhrpark.empty.noFuel.descriptionEmpty')
              }
            />
          ) : (
            <div className="rounded-lg border border-border bg-card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border">
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('fuhrpark.table.date')}</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('fuhrpark.table.vehicle')}</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">{t('fuhrpark.table.liters')}</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">{t('fuhrpark.table.cost', { currency })}</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">{t('fuhrpark.table.pricePerLiter', { currency })}</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">{t('fuhrpark.table.mileage')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredFuel.map((record) => (
                      <tr
                        key={record.id}
                        role="button"
                        tabIndex={0}
                        aria-label={`${plateFor(record.vehicleId, record.vehiclePlate)} — ${formatDate(record.date)}`}
                        onClick={() => setFuelDetail(record)}
                        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setFuelDetail(record) } }}
                        className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer focus-visible:outline-none focus-visible:bg-secondary/50"
                      >
                        <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                          {formatDate(record.date)}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs font-medium text-foreground whitespace-nowrap">
                          {plateFor(record.vehicleId, record.vehiclePlate)}
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground text-right tabular-nums">
                          {record.liters.toLocaleString('de-DE', { minimumFractionDigits: 1 })}
                        </td>
                        <td className="px-4 py-3 text-xs text-foreground font-medium text-right tabular-nums">
                          {formatAmount(record.cost)}
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground text-right tabular-nums">
                          {record.liters > 0 ? (record.cost / record.liters).toFixed(2) : '\u2014'}
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground text-right tabular-nums">
                          {formatKm(record.mileage)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
          </>
          )}
        </>
      )}

      {/* ================================================================= */}
      {/* Fahrtenbuch Tab                                                    */}
      {/* ================================================================= */}
      {tab === 'fahrtenbuch' && (
        <>
          {tripLoading && (
            <div className="p-6 text-center text-muted-foreground text-sm">{t('fuhrpark.loading.logbook')}</div>
          )}
          {!tripLoading && (
          <>
          {/* Summary stats */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-4">
            <div className="rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info-light">
                  <BookOpen className="h-5 w-5 text-info" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('fuhrpark.logbook.businessKm')}</p>
                  <p className="text-xl font-semibold text-foreground">{formatKm(logbookBusinessKm)} km</p>
                </div>
              </div>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-warning-light">
                  <Car className="h-5 w-5 text-warning" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('fuhrpark.logbook.privateKm')}</p>
                  <p className="text-xl font-semibold text-foreground">{formatKm(logbookPrivateKm)} km</p>
                </div>
              </div>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                  <TrendingUp className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">{t('fuhrpark.logbook.privateShare')}</p>
                  <p className="text-xl font-semibold text-foreground">{logbookPrivatePct}%</p>
                </div>
              </div>
            </div>
          </div>

          {filteredLogbook.length === 0 ? (
            <EmptyState
              icon={BookOpen}
              title={t('fuhrpark.empty.noLogbook.title')}
              description={search
                ? t('fuhrpark.empty.noLogbook.descriptionFilter')
                : t('fuhrpark.empty.noLogbook.descriptionEmpty')
              }
            />
          ) : (
            <div className="rounded-lg border border-border bg-card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border">
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('fuhrpark.table.date')}</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('fuhrpark.table.vehicle')}</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('fuhrpark.table.route')}</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('fuhrpark.table.purpose')}</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">{t('fuhrpark.table.km')}</th>
                      <th className="px-4 py-3 text-center text-xs font-medium text-muted-foreground">{t('fuhrpark.table.tripType')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredLogbook.map((entry) => (
                      <tr
                        key={entry.id}
                        role="button"
                        tabIndex={0}
                        aria-label={`${entry.startLocation} → ${entry.endLocation}, ${formatDate(entry.date)}`}
                        onClick={() => setTripDetail(entry)}
                        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setTripDetail(entry) } }}
                        className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer focus-visible:outline-none focus-visible:bg-secondary/50"
                      >
                        <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                          {formatDate(entry.date)}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs font-medium text-foreground whitespace-nowrap">
                          {plateFor(entry.vehicleId, entry.vehiclePlate)}
                        </td>
                        <td className="px-4 py-3 text-xs text-foreground max-w-[240px]">
                          <span className="truncate block">{entry.startLocation} → {entry.endLocation}</span>
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground max-w-[200px] truncate">
                          {entry.purpose}
                        </td>
                        <td className="px-4 py-3 text-xs text-foreground font-medium text-right tabular-nums">
                          {formatKm(entry.km)}
                        </td>
                        <td className="px-4 py-3 text-center">
                          <span className={`inline-block rounded-full px-2 py-0.5 text-[10px] font-medium ${
                            entry.isPrivate ? 'bg-warning-light text-warning' : 'bg-info-light text-info'
                          }`}>
                            {entry.isPrivate ? t('fuhrpark.logbook.private') : t('fuhrpark.logbook.business')}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
          </>
          )}
        </>
      )}

      {/* ================================================================= */}
      {/* Tracking Tab                                                       */}
      {/* ================================================================= */}
      {tab === 'tracking' && (
        <>
          {routesLoading && (
            <div className="p-6 text-center text-muted-foreground text-sm">{t('fuhrpark.loading.tracking')}</div>
          )}
          {!routesLoading && (
          <>
          {/* Stats row */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
            <div className="rounded-lg border border-border bg-card p-3">
              <div className="flex items-center gap-2 mb-1">
                <span className="h-2.5 w-2.5 rounded-full bg-success" />
                <span className="text-xs text-muted-foreground">{t('fuhrpark.trackingStatus.driving')}</span>
              </div>
              <p className="text-lg font-semibold text-foreground">{trackingDrivingCount}</p>
            </div>
            <div className="rounded-lg border border-border bg-card p-3">
              <div className="flex items-center gap-2 mb-1">
                <span className="h-2.5 w-2.5 rounded-full bg-muted-foreground/40" />
                <span className="text-xs text-muted-foreground">{t('fuhrpark.trackingStatus.parked')}</span>
              </div>
              <p className="text-lg font-semibold text-foreground">{trackingParkedCount}</p>
            </div>
            <div className="rounded-lg border border-border bg-card p-3">
              <div className="flex items-center gap-2 mb-1">
                <span className="h-2.5 w-2.5 rounded-full bg-warning" />
                <span className="text-xs text-muted-foreground">{t('fuhrpark.trackingStatus.unknown')}</span>
              </div>
              <p className="text-lg font-semibold text-foreground">{trackingUnknownCount}</p>
            </div>
            <div className="rounded-lg border border-border bg-card p-3">
              <div className="flex items-center gap-2 mb-1">
                <Navigation className="h-3.5 w-3.5 text-primary" />
                <span className="text-xs text-muted-foreground">{t('fuhrpark.tracking.totalKmToday')}</span>
              </div>
              <p className="text-lg font-semibold text-foreground">{formatKm(trackingTotalKm)}</p>
            </div>
          </div>

          {/* Vehicle list */}
          {filteredRoutes.length === 0 ? (
            <EmptyState
              icon={MapPin}
              title={t('fuhrpark.empty.noTracking.title')}
              description={search
                ? t('fuhrpark.empty.noTracking.descriptionFilter')
                : t('fuhrpark.empty.noTracking.descriptionEmpty')
              }
            />
          ) : (
            <div className="space-y-3">
              {filteredRoutes.map((route) => {
                const vehicle = vehicles.find((v) => v.id === route.vehicleId)
                const lastPos = route.positions[route.positions.length - 1]
                const isExpanded = expandedRouteId === route.id

                return (
                  <div key={route.id} className="rounded-lg border border-border bg-card overflow-hidden">
                    {/* Card header — clickable */}
                    <button
                      onClick={() => setExpandedRouteId(isExpanded ? null : route.id)}
                      className="w-full text-left p-4 hover:bg-secondary/30 transition-colors"
                    >
                      <div className="flex items-start justify-between gap-3">
                        {/* Left: status + vehicle info */}
                        <div className="flex items-start gap-3 min-w-0">
                          <span
                            className={`mt-1 h-3 w-3 shrink-0 rounded-full ${trackingStatusDot[route.status]} ${
                              route.status === 'driving' ? 'animate-pulse' : ''
                            }`}
                          />
                          <div className="min-w-0">
                            <div className="flex items-center gap-2 flex-wrap">
                              <p className="text-sm font-medium text-foreground">
                                {route.vehicleName}
                              </p>
                              {vehicle && (
                                <span className="font-mono text-xs text-muted-foreground">
                                  {vehicle.licensePlate}
                                </span>
                              )}
                            </div>
                            <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                              <span className="flex items-center gap-1">
                                <User className="h-3 w-3" />
                                {route.driver}
                              </span>
                              <span className={`font-medium ${trackingStatusTextColor[route.status]}`}>
                                {trackingStatusLabel[route.status] ? t(trackingStatusLabel[route.status]) : route.status}
                              </span>
                            </div>
                          </div>
                        </div>

                        {/* Right: km + last update + chevron */}
                        <div className="flex items-center gap-3 shrink-0">
                          <div className="text-right">
                            <p className="text-sm font-semibold text-foreground tabular-nums">
                              {formatKm(route.dailyKm)} km
                            </p>
                            <p className="text-[11px] text-muted-foreground">
                              {formatRelativeTime(lastPos.timestamp, t)}
                            </p>
                          </div>
                          {isExpanded ? (
                            <ChevronUp className="h-4 w-4 text-muted-foreground" />
                          ) : (
                            <ChevronDown className="h-4 w-4 text-muted-foreground" />
                          )}
                        </div>
                      </div>

                      {/* Last known position */}
                      <div className="flex items-center gap-1.5 mt-2 ml-6 text-xs text-muted-foreground">
                        <MapPin className="h-3 w-3 shrink-0" />
                        <span className="truncate">{lastPos.address}</span>
                      </div>
                    </button>

                    {/* Expanded route detail */}
                    {isExpanded && (
                      <div className="border-t border-border px-4 py-4">
                        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-3">
                          {t('fuhrpark.tracking.routeHistory', { date: formatDate(route.date) })}
                        </h4>
                        <div className="relative ml-1.5">
                          {route.positions.map((pos, idx) => {
                            const isFirst = idx === 0
                            const isLast = idx === route.positions.length - 1
                            const isCurrent = isLast && route.status === 'driving'

                            // Determine stop duration if parked at same location
                            let stopDuration: string | null = null
                            if (idx < route.positions.length - 1) {
                              const next = route.positions[idx + 1]
                              if (pos.address === next.address || next.address.includes('geparkt')) {
                                const diffMin = Math.floor(
                                  (new Date(next.timestamp).getTime() - new Date(pos.timestamp).getTime()) / 60000
                                )
                                if (diffMin > 0) {
                                  stopDuration = diffMin >= 60
                                    ? `${Math.floor(diffMin / 60)} Std ${diffMin % 60} Min Stopp`
                                    : `${diffMin} Min Stopp`
                                }
                              }
                            }

                            return (
                              <div key={idx} className="flex gap-3 relative">
                                {/* Timeline line + dot */}
                                <div className="flex flex-col items-center shrink-0">
                                  <span
                                    className={`relative z-10 h-3 w-3 rounded-full border-2 ${
                                      isFirst
                                        ? 'border-success bg-success'
                                        : isCurrent
                                          ? 'border-primary bg-primary animate-pulse'
                                          : 'border-border bg-secondary'
                                    }`}
                                  />
                                  {!isLast && (
                                    <span className="w-0.5 flex-1 bg-border" />
                                  )}
                                </div>

                                {/* Content */}
                                <div className={`pb-4 min-w-0 ${isLast ? 'pb-0' : ''}`}>
                                  <div className="flex items-center gap-2">
                                    <span className="text-xs font-medium text-foreground tabular-nums">
                                      {formatTime(pos.timestamp)}
                                    </span>
                                    {isFirst && (
                                      <span className="rounded-full bg-success-light px-1.5 py-0.5 text-[9px] font-medium text-success">
                                        {t('fuhrpark.tracking.start')}
                                      </span>
                                    )}
                                    {isCurrent && (
                                      <span className="rounded-full bg-primary-light px-1.5 py-0.5 text-[9px] font-medium text-primary">
                                        {t('fuhrpark.tracking.current')}
                                      </span>
                                    )}
                                  </div>
                                  <p className="text-xs text-muted-foreground mt-0.5 truncate">
                                    {pos.address}
                                  </p>
                                  {stopDuration && (
                                    <p className="text-[10px] text-warning mt-0.5">
                                      {stopDuration}
                                    </p>
                                  )}
                                </div>
                              </div>
                            )
                          })}
                        </div>

                        {/* Total distance */}
                        <div className="mt-3 pt-3 border-t border-border-muted flex items-center justify-between">
                          <span className="text-xs text-muted-foreground">{t('fuhrpark.tracking.totalDistance')}</span>
                          <span className="text-sm font-semibold text-foreground tabular-nums">
                            {formatKm(route.dailyKm)} km
                          </span>
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
        </>
      )}

      {/* ================================================================= */}
      {/* Detail Modal (vehicle) — Cosmi window standard (was a slide-over)  */}
      {/* ================================================================= */}
      <DetailModal
        open={!!selectedVehicle}
        onClose={() => setSelectedVehicle(null)}
        title={t('fuhrpark.detail.title')}
        subtitle={selectedVehicle ? `${selectedVehicle.make} ${selectedVehicle.model}` : undefined}
        badge={
          selectedVehicle ? (
            <span className={`ml-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${vehicleTypeColors[selectedVehicle.type] ?? 'bg-secondary text-muted-foreground'}`}>
              {vehicleTypeLabels[selectedVehicle.type] ? t(vehicleTypeLabels[selectedVehicle.type]) : selectedVehicle.type}
            </span>
          ) : undefined
        }
      >
        {selectedVehicle && (
          <VehicleDetailContent
            vehicle={selectedVehicle}
            maintenanceRecords={maintenanceRecords}
            fuelRecords={fuelRecords}
            logbookEntries={logbookEntries}
            vehicleDocuments={vehicleDocuments}
            damageReports={damageReports}
            currency={currency}
            reminderLeadDays={reminderLeadDays}
            onAddMaintenance={() => openAddMaintenanceFromPanel(selectedVehicle.id)}
            onAddFuel={() => openAddFuelFromPanel(selectedVehicle.id)}
            onAddDamage={() => openAddDamageFromPanel(selectedVehicle.id)}
            canCreateService={canCreateService}
            canCreateFuel={canCreateFuel}
            canCreateDamage={canCreateDamage}
          />
        )}
      </DetailModal>

      {/* Row-level detail modals (Wartung / Tanken / Fahrtenbuch) */}
      <MaintenanceDetailModal
        record={maintenanceDetail}
        onClose={() => setMaintenanceDetail(null)}
        plate={maintenanceDetail ? plateFor(maintenanceDetail.vehicleId, maintenanceDetail.vehiclePlate) : ''}
        typeLabel={
          maintenanceDetail && maintenanceTypeLabels[maintenanceDetail.type]
            ? t(maintenanceTypeLabels[maintenanceDetail.type])
            : (maintenanceDetail?.type ?? '')
        }
        typeColor={maintenanceDetail ? (maintenanceTypeColors[maintenanceDetail.type] ?? 'bg-secondary text-muted-foreground') : ''}
        currency={currency}
      />
      <FuelDetailModal
        record={fuelDetail}
        onClose={() => setFuelDetail(null)}
        plate={fuelDetail ? plateFor(fuelDetail.vehicleId, fuelDetail.vehiclePlate) : ''}
        currency={currency}
      />
      <TripDetailModal
        entry={tripDetail}
        onClose={() => setTripDetail(null)}
        plate={tripDetail ? plateFor(tripDetail.vehicleId, tripDetail.vehiclePlate) : ''}
        businessLabel={t('fuhrpark.logbook.business')}
        privateLabel={t('fuhrpark.logbook.private')}
      />

      {/* ================================================================= */}
      {/* Dialogs                                                            */}
      {/* ================================================================= */}
      {dialog === 'addVehicle' && (
        <AddVehicleDialog
          onClose={() => setDialog(null)}
          onSave={handleSaveVehicle}
        />
      )}
      {dialog === 'addMaintenance' && (
        <AddMaintenanceDialog
          vehicles={vehicles}
          preselectedVehicleId={dialogPreselectedVehicleId}
          onClose={() => setDialog(null)}
          onSave={handleSaveMaintenance}
        />
      )}
      {dialog === 'addFuel' && (
        <AddFuelDialog
          vehicles={vehicles}
          preselectedVehicleId={dialogPreselectedVehicleId}
          onClose={() => setDialog(null)}
          onSave={handleSaveFuel}
        />
      )}
      {dialog === 'addTrip' && (
        <AddTripDialog
          vehicles={vehicles}
          defaultCategory={defaultTripCategory}
          privateTripsEnabled={privateTripsEnabled}
          onClose={() => setDialog(null)}
          onSave={handleSaveTrip}
        />
      )}
      {dialog === 'addDamage' && (
        <SchadensmeldungDialog
          vehicles={vehicles}
          preselectedVehicleId={dialogPreselectedVehicleId}
          onClose={() => setDialog(null)}
          onSave={(report: DamageReport) => {
            reportDamageMutation.mutate(
              {
                vehicleId: report.vehicleId,
                description: report.description,
                severity: report.severity === 'major' ? 'major' : report.severity,
                reported_by: report.reportedBy || undefined,
              },
              {
                onSuccess: () => {
                  setDialog(null)
                  toast.success(t('fuhrpark.toast.damageReported', { plate: report.vehiclePlate }))
                },
              }
            )
          }}
        />
      )}
    </div>
  )
}
