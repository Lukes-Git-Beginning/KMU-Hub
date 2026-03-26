import { useState } from 'react'
import { Camera, Plus, AlertTriangle, AlertCircle, AlertOctagon } from 'lucide-react'
import { toast } from 'sonner'
import type { Vehicle, DamageReport } from '@/stores/fuhrpark'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface SchadensmeldungDialogProps {
  vehicles: Vehicle[]
  preselectedVehicleId?: string
  onClose: () => void
  onSave: (report: DamageReport) => void
}

type Severity = 'minor' | 'moderate' | 'major'

const severityOptions: { value: Severity; label: string; icon: typeof AlertTriangle }[] = [
  { value: 'minor', label: 'Gering', icon: AlertCircle },
  { value: 'moderate', label: 'Mittel', icon: AlertTriangle },
  { value: 'major', label: 'Schwer', icon: AlertOctagon },
]

const severityStyles: Record<Severity, { active: string; inactive: string }> = {
  minor: {
    active: 'border-info bg-info-light text-info font-medium',
    inactive: 'border-border text-muted-foreground hover:bg-secondary',
  },
  moderate: {
    active: 'border-warning bg-warning-light text-warning font-medium',
    inactive: 'border-border text-muted-foreground hover:bg-secondary',
  },
  major: {
    active: 'border-error bg-error-light text-error font-medium',
    inactive: 'border-border text-muted-foreground hover:bg-secondary',
  },
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function SchadensmeldungDialog({
  vehicles,
  preselectedVehicleId,
  onClose,
  onSave,
}: SchadensmeldungDialogProps) {
  const [vehicleId, setVehicleId] = useState(preselectedVehicleId || vehicles[0]?.id || '')
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10))
  const [severity, setSeverity] = useState<Severity>('minor')
  const [location, setLocation] = useState('')
  const [description, setDescription] = useState('')
  const [photoCount, setPhotoCount] = useState(0)
  const [reportedBy, setReportedBy] = useState('')

  const selectedVehicle = vehicles.find((v) => v.id === vehicleId)
  const canSave = vehicleId && description.trim() && location.trim()

  const handleSave = () => {
    if (!canSave || !selectedVehicle) {
      toast.error('Bitte alle Pflichtfelder ausfuellen')
      return
    }

    const report: DamageReport = {
      id: `dmg-${Date.now()}`,
      vehicleId,
      vehiclePlate: selectedVehicle.licensePlate,
      date,
      description: description.trim(),
      severity,
      location: location.trim(),
      photoCount,
      reportedBy: reportedBy.trim(),
      status: 'open',
    }

    onSave(report)
    onClose()
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-lg">
        <div className="p-6">
        {/* Header */}
        <DialogHeader className="mb-5">
          <DialogTitle className="text-base font-semibold text-foreground">Schadensmeldung erstellen</DialogTitle>
          <DialogDescription className="sr-only">Formular fuer eine neue Schadensmeldung</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Fahrzeug */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Fahrzeug *</label>
            <select
              value={vehicleId}
              onChange={(e) => setVehicleId(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              {vehicles
                .filter((v) => v.isActive)
                .map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.licensePlate} — {v.make} {v.model}
                  </option>
                ))}
            </select>
          </div>

          {/* Row: Datum */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Datum</label>
            <input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Schweregrad */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Schweregrad</label>
            <div className="flex gap-2">
              {severityOptions.map((opt) => {
                const isActive = severity === opt.value
                const Icon = opt.icon
                return (
                  <button
                    key={opt.value}
                    onClick={() => setSeverity(opt.value)}
                    className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs transition-colors ${
                      isActive
                        ? severityStyles[opt.value].active
                        : severityStyles[opt.value].inactive
                    }`}
                  >
                    <Icon className="h-3.5 w-3.5" />
                    {opt.label}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Schadensort */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Schadensort *</label>
            <input
              type="text"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              placeholder="z.B. Rechte Seite hinten, Windschutzscheibe"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Beschreibung */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Beschreibung *</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              placeholder="Detaillierte Beschreibung des Schadens..."
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
            />
          </div>

          {/* Fotos (mock) */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Fotos</label>
            <div className="flex items-center gap-3 rounded-lg border-2 border-dashed border-border bg-secondary/30 p-4">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-secondary">
                <Camera className="h-5 w-5 text-muted-foreground" />
              </div>
              <div className="flex-1 min-w-0">
                {photoCount > 0 ? (
                  <p className="text-sm text-foreground font-medium">
                    {photoCount} Foto{photoCount !== 1 ? 's' : ''} ausgewaehlt
                  </p>
                ) : (
                  <p className="text-xs text-muted-foreground">Noch keine Fotos hinzugefuegt</p>
                )}
              </div>
              <button
                onClick={() => setPhotoCount((prev) => prev + 1)}
                className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors shrink-0"
              >
                <Plus className="h-3.5 w-3.5" />
                Fotos hinzufuegen
              </button>
            </div>
          </div>

          {/* Gemeldet von */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Gemeldet von</label>
            <input
              type="text"
              value={reportedBy}
              onChange={(e) => setReportedBy(e.target.value)}
              placeholder="Name des Meldenden"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
        </div>

        {/* Actions */}
        <DialogFooter className="mt-6">
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
            Schadensmeldung speichern
          </button>
        </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}
