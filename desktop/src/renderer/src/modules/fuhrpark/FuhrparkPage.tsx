import { useState, useMemo, useCallback } from 'react'
import {
  Search,
  Plus,
  Car,
  Wrench,
  Fuel,
  AlertTriangle,
  Gauge,
  User,
  X,
  TrendingUp,
  Droplets,
  FileText,
  MapPin,
  Navigation,
  RefreshCw,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { toast } from 'sonner'
import { DetailPanel, EmptyState } from '@/components/shared'
import { useFuhrparkStore, type Vehicle, type MaintenanceRecord, type FuelRecord } from '@/stores/fuhrpark'

// ---------------------------------------------------------------------------
// Types & Constants
// ---------------------------------------------------------------------------

type TabKey = 'fahrzeuge' | 'wartung' | 'tankprotokoll' | 'tracking'
type DialogKey = 'addVehicle' | 'addMaintenance' | 'addFuel' | null

const vehicleTypeLabels: Record<string, string> = {
  car: 'PKW',
  van: 'Lieferwagen',
  truck: 'LKW',
}

const vehicleTypeColors: Record<string, string> = {
  car: 'bg-info-light text-info',
  van: 'bg-warning-light text-warning',
  truck: 'bg-primary-light text-primary',
}

const maintenanceTypeLabels: Record<string, string> = {
  service: 'Service',
  repair: 'Reparatur',
  inspection: 'Pruefung',
  tires: 'Reifen',
}

const maintenanceTypeColors: Record<string, string> = {
  service: 'bg-info-light text-info',
  repair: 'bg-warning-light text-warning',
  inspection: 'bg-success-light text-success',
  tires: 'bg-primary-light text-primary',
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getDateStatus(dateStr: string): 'overdue' | 'soon' | 'ok' {
  const date = new Date(dateStr)
  const now = new Date()
  const diffDays = Math.floor((date.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
  if (diffDays < 0) return 'overdue'
  if (diffDays <= 30) return 'soon'
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

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-CH')
}

function formatCHF(val: number): string {
  return val.toLocaleString('de-CH', { minimumFractionDigits: 2 })
}

function formatKm(val: number): string {
  return val.toLocaleString('de-CH')
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
  driving: 'Unterwegs',
  parked: 'Geparkt',
  unknown: 'Unbekannt',
}

const trackingStatusTextColor: Record<string, string> = {
  driving: 'text-success',
  parked: 'text-muted-foreground',
  unknown: 'text-warning',
}

function formatRelativeTime(timestamp: string): string {
  const now = new Date()
  const then = new Date(timestamp)
  const diffMs = now.getTime() - then.getTime()
  const diffMin = Math.floor(diffMs / 60000)

  if (diffMin < 1) return 'gerade eben'
  if (diffMin < 60) return `vor ${diffMin} Min`
  const diffHrs = Math.floor(diffMin / 60)
  if (diffHrs < 24) return `vor ${diffHrs} Std`
  const diffDays = Math.floor(diffHrs / 24)
  return `vor ${diffDays} Tag${diffDays > 1 ? 'en' : ''}`
}

function formatTime(timestamp: string): string {
  return new Date(timestamp).toLocaleTimeString('de-CH', { hour: '2-digit', minute: '2-digit' })
}

// ---------------------------------------------------------------------------
// Dialog: Fahrzeug hinzufuegen
// ---------------------------------------------------------------------------

interface AddVehicleDialogProps {
  onClose: () => void
  onSave: (v: Vehicle) => void
}

function AddVehicleDialog({ onClose, onSave }: AddVehicleDialogProps) {
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="w-full max-w-lg rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-base font-semibold text-foreground">Fahrzeug hinzufuegen</h2>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4">
          {/* Row: Kennzeichen + Typ */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Kennzeichen *</label>
              <input
                type="text"
                value={plate}
                onChange={(e) => setPlate(e.target.value)}
                placeholder="ZH 123 456"
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Typ</label>
              <select
                value={type}
                onChange={(e) => setType(e.target.value as 'car' | 'van' | 'truck')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                <option value="car">PKW</option>
                <option value="van">Lieferwagen</option>
                <option value="truck">LKW</option>
              </select>
            </div>
          </div>

          {/* Row: Marke + Modell */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Marke *</label>
              <input
                type="text"
                value={make}
                onChange={(e) => setMake(e.target.value)}
                placeholder="VW"
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Modell *</label>
              <input
                type="text"
                value={model}
                onChange={(e) => setModel(e.target.value)}
                placeholder="Caddy Cargo"
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Row: Jahrgang + km-Stand */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Jahrgang</label>
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
              <label className="text-xs font-medium text-muted-foreground">km-Stand</label>
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
            <label className="text-xs font-medium text-muted-foreground">Fahrer zuweisen</label>
            <input
              type="text"
              value={driver}
              onChange={(e) => setDriver(e.target.value)}
              placeholder="Name des Fahrers"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Row: Pruefung + Versicherung */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Naechste Pruefung</label>
              <input
                type="date"
                value={nextInspection}
                onChange={(e) => setNextInspection(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Versicherungsablauf</label>
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
        <div className="mt-6 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            Abbrechen
          </button>
          <button
            onClick={handleSave}
            disabled={!canSave}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:pointer-events-none"
          >
            Speichern
          </button>
        </div>
      </div>
    </div>
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
    { value: 'service', label: 'Service' },
    { value: 'repair', label: 'Reparatur' },
    { value: 'inspection', label: 'Pruefung' },
    { value: 'tires', label: 'Reifen' },
  ]

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="w-full max-w-lg rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-base font-semibold text-foreground">Wartung eintragen</h2>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4">
          {/* Fahrzeug */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Fahrzeug</label>
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
            <label className="text-xs font-medium text-muted-foreground">Typ</label>
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
              <label className="text-xs font-medium text-muted-foreground">Datum</label>
              <input
                type="date"
                value={date}
                onChange={(e) => setDate(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">km-Stand</label>
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
            <label className="text-xs font-medium text-muted-foreground">Kosten (CHF)</label>
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
            <label className="text-xs font-medium text-muted-foreground">Notizen</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
              placeholder="Beschreibung der Wartung..."
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
            />
          </div>
        </div>

        <div className="mt-6 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            Abbrechen
          </button>
          <button
            onClick={handleSave}
            disabled={!vehicleId}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:pointer-events-none"
          >
            Speichern
          </button>
        </div>
      </div>
    </div>
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-base font-semibold text-foreground">Tanken eintragen</h2>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4">
          {/* Fahrzeug */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Fahrzeug</label>
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
            <label className="text-xs font-medium text-muted-foreground">Datum</label>
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
              <label className="text-xs font-medium text-muted-foreground">Liter</label>
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
              <label className="text-xs font-medium text-muted-foreground">Kosten (CHF)</label>
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
            <label className="text-xs font-medium text-muted-foreground">km-Stand</label>
            <input
              type="number"
              value={mileage}
              onChange={(e) => setMileage(Number(e.target.value))}
              min={0}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
        </div>

        <div className="mt-6 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            Abbrechen
          </button>
          <button
            onClick={handleSave}
            disabled={!vehicleId || liters <= 0}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:pointer-events-none"
          >
            Speichern
          </button>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Vehicle Detail Panel Content
// ---------------------------------------------------------------------------

interface VehicleDetailContentProps {
  vehicle: Vehicle
  maintenanceRecords: MaintenanceRecord[]
  fuelRecords: FuelRecord[]
  onAddMaintenance: () => void
  onAddFuel: () => void
}

function VehicleDetailContent({
  vehicle,
  maintenanceRecords,
  fuelRecords,
  onAddMaintenance,
  onAddFuel,
}: VehicleDetailContentProps) {
  const inspectionStatus = getDateStatus(vehicle.nextInspection)
  const insuranceStatus = getDateStatus(vehicle.insuranceExpiry)

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
          <p className="text-xs text-muted-foreground">Jahrgang {vehicle.year}</p>
        </div>
        <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-medium ${vehicleTypeColors[vehicle.type] ?? 'bg-secondary text-muted-foreground'}`}>
          {vehicleTypeLabels[vehicle.type] ?? vehicle.type}
        </span>
      </div>

      {/* Driver + Mileage */}
      <section className="space-y-2">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Fahrzeugdaten</h4>
        <div className="space-y-1.5 text-xs">
          <div className="flex items-center gap-2 text-muted-foreground">
            <User className="h-3.5 w-3.5 shrink-0" />
            <span className="text-foreground">{vehicle.currentDriver || 'Kein Fahrer zugewiesen'}</span>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <Gauge className="h-3.5 w-3.5 shrink-0" />
            <span className="text-foreground">{formatKm(vehicle.mileage)} km</span>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <TrendingUp className="h-3.5 w-3.5 shrink-0" />
            <span className="text-foreground">
              Gesamtkosten: CHF {formatCHF(totalMaintenanceCost + totalFuelCost)}
            </span>
          </div>
        </div>
      </section>

      {/* Inspection + Insurance with traffic light */}
      <section className="space-y-2">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Status</h4>
        <div className="space-y-2">
          <div className="flex items-center justify-between rounded-md bg-secondary/50 px-3 py-2">
            <div className="flex items-center gap-2">
              <span className={`h-2.5 w-2.5 rounded-full ${statusDotColor(inspectionStatus)}`} />
              <span className="text-xs text-muted-foreground">Naechste Pruefung</span>
            </div>
            <span className={`text-xs ${statusTextColor(inspectionStatus)}`}>
              {inspectionStatus === 'overdue' && <AlertTriangle className="inline h-3 w-3 mr-1" />}
              {formatDate(vehicle.nextInspection)}
            </span>
          </div>
          <div className="flex items-center justify-between rounded-md bg-secondary/50 px-3 py-2">
            <div className="flex items-center gap-2">
              <span className={`h-2.5 w-2.5 rounded-full ${statusDotColor(insuranceStatus)}`} />
              <span className="text-xs text-muted-foreground">Versicherungsablauf</span>
            </div>
            <span className={`text-xs ${statusTextColor(insuranceStatus)}`}>
              {insuranceStatus === 'overdue' && <AlertTriangle className="inline h-3 w-3 mr-1" />}
              {formatDate(vehicle.insuranceExpiry)}
            </span>
          </div>
        </div>
      </section>

      {/* Recent Maintenance */}
      <section className="space-y-2">
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
          Letzte Wartungen
        </h4>
        {vehicleMaintenance.length === 0 ? (
          <p className="text-xs text-muted-foreground italic">Keine Eintraege vorhanden</p>
        ) : (
          <div className="space-y-1.5">
            {vehicleMaintenance.map((r) => (
              <div key={r.id} className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2">
                <div className="flex items-center gap-2 min-w-0">
                  <span className={`shrink-0 rounded-full px-2 py-0.5 text-[9px] font-medium ${maintenanceTypeColors[r.type] ?? 'bg-secondary text-muted-foreground'}`}>
                    {maintenanceTypeLabels[r.type] ?? r.type}
                  </span>
                  <span className="text-xs text-muted-foreground truncate">{r.notes || '—'}</span>
                </div>
                <div className="text-right shrink-0 ml-2">
                  <p className="text-xs font-medium text-foreground">CHF {formatCHF(r.cost)}</p>
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
          Letzte Tankungen
        </h4>
        {vehicleFuel.length === 0 ? (
          <p className="text-xs text-muted-foreground italic">Keine Eintraege vorhanden</p>
        ) : (
          <div className="space-y-1.5">
            {vehicleFuel.map((r) => (
              <div key={r.id} className="flex items-center justify-between rounded-md border border-border-muted px-3 py-2">
                <div className="flex items-center gap-2">
                  <Droplets className="h-3.5 w-3.5 text-info shrink-0" />
                  <span className="text-xs text-muted-foreground">{r.liters.toLocaleString('de-CH', { minimumFractionDigits: 1 })} L</span>
                </div>
                <div className="text-right">
                  <p className="text-xs font-medium text-foreground">CHF {formatCHF(r.cost)}</p>
                  <p className="text-[10px] text-muted-foreground">{formatDate(r.date)}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Action Buttons */}
      <div className="flex gap-2 pt-1">
        <button
          onClick={onAddMaintenance}
          className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
        >
          <Wrench className="h-3.5 w-3.5" />
          Wartung eintragen
        </button>
        <button
          onClick={onAddFuel}
          className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
        >
          <Fuel className="h-3.5 w-3.5" />
          Tanken eintragen
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main Page Component
// ---------------------------------------------------------------------------

export default function FuhrparkPage() {
  const { vehicles, maintenanceRecords, fuelRecords, vehicleRoutes, refreshTracking } = useFuhrparkStore()

  const [tab, setTab] = useState<TabKey>('fahrzeuge')
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<'all' | 'car' | 'van' | 'truck'>('all')
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all')
  const [dialog, setDialog] = useState<DialogKey>(null)
  const [selectedVehicle, setSelectedVehicle] = useState<Vehicle | null>(null)
  const [dialogPreselectedVehicleId, setDialogPreselectedVehicleId] = useState<string | undefined>()
  const [expandedRouteId, setExpandedRouteId] = useState<string | null>(null)
  const [trackingRefreshing, setTrackingRefreshing] = useState(false)

  // Filtered vehicles with search + type + status filters
  const filteredVehicles = useMemo(() => {
    return vehicles.filter((v) => {
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
  }, [vehicles, search, typeFilter, statusFilter])

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

  // Counts for tab badges
  const activeVehicleCount = vehicles.filter((v) => v.isActive).length
  const urgentCount = vehicles.filter((v) => {
    const i = getDateStatus(v.nextInspection)
    const ins = getDateStatus(v.insuranceExpiry)
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

  const handleSaveVehicle = useCallback(
    (vehicle: Vehicle) => {
      useFuhrparkStore.setState((state) => ({
        vehicles: [...state.vehicles, vehicle],
      }))
      setDialog(null)
      toast.success(`Fahrzeug ${vehicle.licensePlate} wurde hinzugefuegt`)
    },
    []
  )

  const handleSaveMaintenance = useCallback(
    (record: MaintenanceRecord) => {
      useFuhrparkStore.setState((state) => ({
        maintenanceRecords: [...state.maintenanceRecords, record],
      }))
      setDialog(null)
      toast.success(`Wartung fuer ${record.vehiclePlate} eingetragen`)
    },
    []
  )

  const handleSaveFuel = useCallback(
    (record: FuelRecord) => {
      useFuhrparkStore.setState((state) => ({
        fuelRecords: [...state.fuelRecords, record],
      }))
      setDialog(null)
      toast.success(`Tankung fuer ${record.vehiclePlate} eingetragen`)
    },
    []
  )

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
    setTimeout(() => {
      refreshTracking()
      setTrackingRefreshing(false)
    }, 800)
  }, [refreshTracking])

  // Search placeholder based on tab
  const searchPlaceholder =
    tab === 'fahrzeuge'
      ? 'Fahrzeug, Fahrer suchen...'
      : tab === 'wartung'
        ? 'Wartung suchen...'
        : tab === 'tracking'
          ? 'Fahrzeug, Fahrer, Ort suchen...'
          : 'Tankprotokoll suchen...'

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div>
          <h1 className="text-foreground">Fuhrpark</h1>
          <p className="text-sm text-muted-foreground">
            {activeVehicleCount} aktive Fahrzeuge
            {urgentCount > 0 && (
              <span className="text-warning"> &middot; {urgentCount} mit faelliger Pruefung/Versicherung</span>
            )}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {tab === 'wartung' && (
            <button
              onClick={openAddMaintenanceGlobal}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              <Wrench className="h-4 w-4" />
              Wartung eintragen
            </button>
          )}
          {tab === 'tankprotokoll' && (
            <button
              onClick={openAddFuelGlobal}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              <Fuel className="h-4 w-4" />
              Tanken eintragen
            </button>
          )}
          {tab === 'tracking' && (
            <button
              onClick={handleRefreshTracking}
              disabled={trackingRefreshing}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`h-4 w-4 ${trackingRefreshing ? 'animate-spin' : ''}`} />
              Aktualisieren
            </button>
          )}
          <button
            onClick={openAddVehicle}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            Fahrzeug hinzufuegen
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'fahrzeuge' as const, label: `Fahrzeuge (${vehicles.length})`, icon: null },
          { key: 'wartung' as const, label: `Wartung (${maintenanceRecords.length})`, icon: null },
          { key: 'tankprotokoll' as const, label: `Tankprotokoll (${fuelRecords.length})`, icon: null },
          { key: 'tracking' as const, label: 'Tracking', icon: MapPin },
        ]).map((t) => (
          <button
            key={t.key}
            onClick={() => { setTab(t.key); setSearch(''); setExpandedRouteId(null) }}
            className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key
                ? 'border-primary text-primary font-medium'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t.icon && <t.icon className="h-3.5 w-3.5" />}
            {t.label}
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
              <option value="all">Alle Typen</option>
              <option value="car">PKW</option>
              <option value="van">Lieferwagen</option>
              <option value="truck">LKW</option>
            </select>

            {/* Status filter */}
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as typeof statusFilter)}
              className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              <option value="all">Alle Status</option>
              <option value="active">Aktiv</option>
              <option value="inactive">Stillgelegt</option>
            </select>
          </>
        )}
      </div>

      {/* ================================================================= */}
      {/* Fahrzeuge Tab                                                      */}
      {/* ================================================================= */}
      {tab === 'fahrzeuge' && (
        <>
          {filteredVehicles.length === 0 ? (
            <EmptyState
              icon={Car}
              title="Keine Fahrzeuge gefunden"
              description={search || typeFilter !== 'all' || statusFilter !== 'all' ? 'Passe deine Filter an' : 'Fuege ein Fahrzeug hinzu'}
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {filteredVehicles.map((vehicle) => {
                const inspectionStatus = getDateStatus(vehicle.nextInspection)
                const insuranceStatus = getDateStatus(vehicle.insuranceExpiry)

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
                          {vehicleTypeLabels[vehicle.type] ?? vehicle.type}
                        </span>
                        {!vehicle.isActive && (
                          <span className="rounded-full bg-error-light px-2 py-0.5 text-[10px] font-medium text-error">
                            Stillgelegt
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="space-y-1.5 text-xs text-muted-foreground mb-3">
                      <div className="flex items-center gap-2">
                        <User className="h-3 w-3" />
                        <span>{vehicle.currentDriver || 'Nicht zugewiesen'}</span>
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
                          Pruefung {formatDate(vehicle.nextInspection)}
                        </span>
                      </div>
                      <div className="flex items-center gap-1.5">
                        <span className={`h-2 w-2 rounded-full ${statusDotColor(insuranceStatus)}`} />
                        <span className={`text-[11px] ${statusTextColor(insuranceStatus) || 'text-muted-foreground'}`}>
                          Vers. {formatDate(vehicle.insuranceExpiry)}
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

      {/* ================================================================= */}
      {/* Wartung Tab                                                        */}
      {/* ================================================================= */}
      {tab === 'wartung' && (
        <>
          {/* Summary cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
            {(['service', 'repair', 'inspection', 'tires'] as const).map((t) => {
              const count = maintenanceRecords.filter((r) => r.type === t).length
              const totalCost = maintenanceRecords
                .filter((r) => r.type === t)
                .reduce((sum, r) => sum + r.cost, 0)
              return (
                <div key={t} className="rounded-lg border border-border bg-card p-3">
                  <div className="flex items-center gap-2 mb-1">
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${maintenanceTypeColors[t]}`}>
                      {maintenanceTypeLabels[t]}
                    </span>
                  </div>
                  <p className="text-lg font-semibold text-foreground">{count}</p>
                  <p className="text-[11px] text-muted-foreground">CHF {formatCHF(totalCost)}</p>
                </div>
              )
            })}
          </div>

          {filteredMaintenance.length === 0 ? (
            <EmptyState
              icon={Wrench}
              title="Keine Wartungseintraege"
              description={search ? 'Passe deine Suche an' : 'Wartungen werden hier aufgelistet'}
            />
          ) : (
            <div className="rounded-lg border border-border bg-card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border">
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Datum</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Fahrzeug</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Typ</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">km-Stand</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">Kosten (CHF)</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Notizen</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredMaintenance.map((record) => (
                      <tr
                        key={record.id}
                        className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors"
                      >
                        <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                          {formatDate(record.date)}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs font-medium text-foreground whitespace-nowrap">
                          {record.vehiclePlate}
                        </td>
                        <td className="px-4 py-3">
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${maintenanceTypeColors[record.type] ?? 'bg-secondary text-muted-foreground'}`}>
                            {maintenanceTypeLabels[record.type] ?? record.type}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground text-right tabular-nums">
                          {formatKm(record.mileage)}
                        </td>
                        <td className="px-4 py-3 text-xs text-foreground font-medium text-right tabular-nums">
                          {formatCHF(record.cost)}
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

      {/* ================================================================= */}
      {/* Tankprotokoll Tab                                                  */}
      {/* ================================================================= */}
      {tab === 'tankprotokoll' && (
        <>
          {/* Monthly summary */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-4">
            <div className="rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-warning-light">
                  <Fuel className="h-5 w-5 text-warning" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Kosten diesen Monat</p>
                  <p className="text-xl font-semibold text-foreground">CHF {formatCHF(monthlyFuelCost)}</p>
                </div>
              </div>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info-light">
                  <Droplets className="h-5 w-5 text-info" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Liter diesen Monat</p>
                  <p className="text-xl font-semibold text-foreground">
                    {fuelRecords
                      .filter((r) => {
                        const d = new Date(r.date)
                        return d.getMonth() === currentMonth && d.getFullYear() === currentYear
                      })
                      .reduce((sum, r) => sum + r.liters, 0)
                      .toLocaleString('de-CH', { minimumFractionDigits: 1 })} L
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
                  <p className="text-xs text-muted-foreground">Tankungen gesamt</p>
                  <p className="text-xl font-semibold text-foreground">{fuelRecords.length}</p>
                </div>
              </div>
            </div>
          </div>

          {filteredFuel.length === 0 ? (
            <EmptyState
              icon={Fuel}
              title="Kein Tankprotokoll vorhanden"
              description={search ? 'Passe deine Suche an' : 'Tankungen werden hier aufgelistet'}
            />
          ) : (
            <div className="rounded-lg border border-border bg-card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border">
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Datum</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Fahrzeug</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">Liter</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">Kosten (CHF)</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">CHF/L</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">km-Stand</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredFuel.map((record) => (
                      <tr
                        key={record.id}
                        className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors"
                      >
                        <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                          {formatDate(record.date)}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs font-medium text-foreground whitespace-nowrap">
                          {record.vehiclePlate}
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground text-right tabular-nums">
                          {record.liters.toLocaleString('de-CH', { minimumFractionDigits: 1 })}
                        </td>
                        <td className="px-4 py-3 text-xs text-foreground font-medium text-right tabular-nums">
                          {formatCHF(record.cost)}
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

      {/* ================================================================= */}
      {/* Tracking Tab                                                       */}
      {/* ================================================================= */}
      {tab === 'tracking' && (
        <>
          {/* Stats row */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
            <div className="rounded-lg border border-border bg-card p-3">
              <div className="flex items-center gap-2 mb-1">
                <span className="h-2.5 w-2.5 rounded-full bg-success" />
                <span className="text-xs text-muted-foreground">Unterwegs</span>
              </div>
              <p className="text-lg font-semibold text-foreground">{trackingDrivingCount}</p>
            </div>
            <div className="rounded-lg border border-border bg-card p-3">
              <div className="flex items-center gap-2 mb-1">
                <span className="h-2.5 w-2.5 rounded-full bg-muted-foreground/40" />
                <span className="text-xs text-muted-foreground">Geparkt</span>
              </div>
              <p className="text-lg font-semibold text-foreground">{trackingParkedCount}</p>
            </div>
            <div className="rounded-lg border border-border bg-card p-3">
              <div className="flex items-center gap-2 mb-1">
                <span className="h-2.5 w-2.5 rounded-full bg-warning" />
                <span className="text-xs text-muted-foreground">Unbekannt</span>
              </div>
              <p className="text-lg font-semibold text-foreground">{trackingUnknownCount}</p>
            </div>
            <div className="rounded-lg border border-border bg-card p-3">
              <div className="flex items-center gap-2 mb-1">
                <Navigation className="h-3.5 w-3.5 text-primary" />
                <span className="text-xs text-muted-foreground">Gesamt-km heute</span>
              </div>
              <p className="text-lg font-semibold text-foreground">{formatKm(trackingTotalKm)}</p>
            </div>
          </div>

          {/* Vehicle list */}
          {filteredRoutes.length === 0 ? (
            <EmptyState
              icon={MapPin}
              title="Keine Tracking-Daten"
              description={search ? 'Passe deine Suche an' : 'Keine Fahrzeuge mit GPS-Daten'}
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
                                {trackingStatusLabel[route.status]}
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
                              {formatRelativeTime(lastPos.timestamp)}
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
                          Routenverlauf — {formatDate(route.date)}
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
                                        Start
                                      </span>
                                    )}
                                    {isCurrent && (
                                      <span className="rounded-full bg-primary-light px-1.5 py-0.5 text-[9px] font-medium text-primary">
                                        Aktuell
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
                          <span className="text-xs text-muted-foreground">Gesamtstrecke</span>
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

      {/* ================================================================= */}
      {/* Detail Panel (vehicle)                                             */}
      {/* ================================================================= */}
      <DetailPanel
        open={!!selectedVehicle}
        onClose={() => setSelectedVehicle(null)}
        title="Fahrzeug-Details"
        subtitle={selectedVehicle ? `${selectedVehicle.make} ${selectedVehicle.model}` : undefined}
        badge={
          selectedVehicle ? (
            <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${vehicleTypeColors[selectedVehicle.type] ?? 'bg-secondary text-muted-foreground'}`}>
              {vehicleTypeLabels[selectedVehicle.type] ?? selectedVehicle.type}
            </span>
          ) : undefined
        }
        width="w-[400px]"
      >
        {selectedVehicle && (
          <VehicleDetailContent
            vehicle={selectedVehicle}
            maintenanceRecords={maintenanceRecords}
            fuelRecords={fuelRecords}
            onAddMaintenance={() => openAddMaintenanceFromPanel(selectedVehicle.id)}
            onAddFuel={() => openAddFuelFromPanel(selectedVehicle.id)}
          />
        )}
      </DetailPanel>

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
    </div>
  )
}
