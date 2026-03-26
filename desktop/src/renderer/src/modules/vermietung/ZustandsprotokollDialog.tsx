import { useState } from 'react'
import { Camera, Plus, Check, AlertTriangle, XCircle } from 'lucide-react'
import { toast } from 'sonner'
import type { Reservation, Zustandsprotokoll, ZustandsprotokollItem } from '@/stores/vermietung'
import SignatureCanvas from '../rapporte/SignatureCanvas'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Types & Constants
// ---------------------------------------------------------------------------

type ProtokollType = 'pickup' | 'return'

interface ChecklistRow {
  label: string
  condition: ZustandsprotokollItem['condition']
  note: string
}

const DEFAULT_CHECKLIST_LABELS = [
  'Allgemeinzustand',
  'Sauberkeit',
  'Vollstaendigkeit',
  'Beschaedigungen',
  'Funktion',
]

const CONDITION_OPTIONS: {
  value: ZustandsprotokollItem['condition']
  label: string
  icon: typeof Check
  dot: string
  activeBg: string
}[] = [
  {
    value: 'ok',
    label: 'OK',
    icon: Check,
    dot: 'bg-success',
    activeBg: 'bg-success/15 text-success border-success/40',
  },
  {
    value: 'damaged',
    label: 'Beschaedigt',
    icon: AlertTriangle,
    dot: 'bg-warning',
    activeBg: 'bg-warning/15 text-warning border-warning/40',
  },
  {
    value: 'missing',
    label: 'Fehlend',
    icon: XCircle,
    dot: 'bg-error',
    activeBg: 'bg-error/15 text-error border-error/40',
  },
]

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface ZustandsprotokollDialogProps {
  open: boolean
  onClose: () => void
  reservation: Reservation
  onSave: (z: Zustandsprotokoll) => void
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function todayISO(): string {
  const d = new Date()
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
}

function createDefaultChecklist(): ChecklistRow[] {
  return DEFAULT_CHECKLIST_LABELS.map((label) => ({
    label,
    condition: 'ok' as const,
    note: '',
  }))
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function ZustandsprotokollDialog({
  open,
  onClose,
  reservation,
  onSave,
}: ZustandsprotokollDialogProps) {
  // Form state
  const [type, setType] = useState<ProtokollType>('pickup')
  const [date, setDate] = useState(todayISO)
  const [checklist, setChecklist] = useState<ChecklistRow[]>(createDefaultChecklist)
  const [photoCount, setPhotoCount] = useState(0)
  const [notes, setNotes] = useState('')
  const [signatureDataUrl, setSignatureDataUrl] = useState<string | undefined>()
  const [createdBy, setCreatedBy] = useState('')

  // ---- Checklist helpers ----

  const updateChecklistItem = (idx: number, patch: Partial<ChecklistRow>) => {
    setChecklist((prev) =>
      prev.map((item, i) => (i === idx ? { ...item, ...patch } : item)),
    )
  }

  const addChecklistItem = () => {
    setChecklist((prev) => [...prev, { label: '', condition: 'ok', note: '' }])
  }

  const removeChecklistItem = (idx: number) => {
    setChecklist((prev) => prev.filter((_, i) => i !== idx))
  }

  // ---- Save ----

  const handleSave = () => {
    if (!createdBy.trim()) {
      toast.error('Bitte den Namen des Erstellers angeben')
      return
    }

    const protokoll: Zustandsprotokoll = {
      id: `zp-${Date.now()}`,
      reservationId: reservation.id,
      type,
      date,
      checklist: checklist
        .filter((c) => c.label.trim())
        .map((c) => ({
          label: c.label.trim(),
          condition: c.condition,
          ...(c.condition !== 'ok' && c.note.trim() ? { note: c.note.trim() } : {}),
        })),
      photoCount,
      notes: notes.trim(),
      signatureDataUrl,
      createdBy: createdBy.trim(),
    }

    onSave(protokoll)
    toast.success('Zustandsprotokoll gespeichert')
    onClose()
  }

  // ---- Render ----

  return (
    <Dialog open={open} onOpenChange={(isOpen) => { if (!isOpen) onClose() }}>
      <DialogContent className="max-w-2xl max-h-[85vh] flex flex-col gap-0 p-0 glass-elevated">
        {/* ---- Header ---- */}
        <DialogHeader className="px-6 pt-6 pb-4 border-b border-border flex-shrink-0">
          <DialogTitle>Zustandsprotokoll</DialogTitle>
          <DialogDescription>
            {reservation.objectName} &middot; {reservation.renter}
          </DialogDescription>
        </DialogHeader>

        {/* ---- Scrollable body ---- */}
        <div className="flex-1 overflow-y-auto px-6 py-5 space-y-6">
          {/* ---------- Typ toggle ---------- */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">
              Typ
            </label>
            <div className="inline-flex rounded-lg border border-border overflow-hidden">
              {([
                { key: 'pickup' as const, label: 'Abholung' },
                { key: 'return' as const, label: 'Rueckgabe' },
              ]).map((opt) => (
                <button
                  key={opt.key}
                  type="button"
                  onClick={() => setType(opt.key)}
                  className={`px-4 py-1.5 text-sm font-medium transition-colors ${
                    type === opt.key
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-card text-muted-foreground hover:bg-secondary'
                  }`}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          {/* ---------- Datum ---------- */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">
              Datum
            </label>
            <input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="w-full max-w-[200px] rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* ---------- Checkliste ---------- */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-xs font-medium text-muted-foreground">
                Checkliste
              </label>
              <button
                type="button"
                onClick={addChecklistItem}
                className="inline-flex items-center gap-1 rounded-lg border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
              >
                <Plus className="h-3 w-3" />
                Punkt
              </button>
            </div>

            <div className="rounded-lg border border-border overflow-hidden">
              {checklist.map((item, idx) => {
                const needsNote = item.condition === 'damaged' || item.condition === 'missing'

                return (
                  <div
                    key={idx}
                    className={`px-3 py-2.5 ${
                      idx % 2 === 0 ? 'bg-card' : 'bg-secondary/30'
                    } ${idx > 0 ? 'border-t border-border-muted' : ''}`}
                  >
                    <div className="flex items-center gap-3">
                      {/* Label */}
                      <input
                        type="text"
                        value={item.label}
                        onChange={(e) => updateChecklistItem(idx, { label: e.target.value })}
                        placeholder="Prüfpunkt..."
                        aria-label={`Pruefpunkt ${idx + 1}`}
                        className="flex-1 min-w-0 bg-transparent text-sm text-foreground placeholder:text-input-placeholder focus:outline-none"
                      />

                      {/* Condition toggles */}
                      <div className="flex items-center gap-1 flex-shrink-0">
                        {CONDITION_OPTIONS.map((opt) => {
                          const Icon = opt.icon
                          const isActive = item.condition === opt.value

                          return (
                            <button
                              key={opt.value}
                              type="button"
                              onClick={() => updateChecklistItem(idx, { condition: opt.value })}
                              title={opt.label}
                              aria-label={`${item.label || `Punkt ${idx + 1}`}: ${opt.label}`}
                              className={`inline-flex items-center gap-1 rounded-md border px-2 py-1 text-[11px] font-medium transition-colors ${
                                isActive
                                  ? opt.activeBg
                                  : 'border-border text-muted-foreground hover:bg-secondary'
                              }`}
                            >
                              <Icon className="h-3 w-3" />
                              {opt.label}
                            </button>
                          )
                        })}
                      </div>

                      {/* Remove button */}
                      {checklist.length > 1 && (
                        <button
                          type="button"
                          onClick={() => removeChecklistItem(idx)}
                          className="flex-shrink-0 rounded p-0.5 text-muted-foreground hover:text-error transition-colors"
                          title="Entfernen"
                          aria-label={`${item.label || `Punkt ${idx + 1}`} entfernen`}
                        >
                          <svg className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                        </button>
                      )}
                    </div>

                    {/* Conditional note input */}
                    {needsNote && (
                      <input
                        type="text"
                        value={item.note}
                        onChange={(e) => updateChecklistItem(idx, { note: e.target.value })}
                        placeholder="Anmerkung zur Abweichung..."
                        aria-label={`Anmerkung fuer ${item.label || `Punkt ${idx + 1}`}`}
                        className="mt-2 w-full rounded-md border border-border bg-card px-2.5 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                      />
                    )}
                  </div>
                )
              })}
            </div>
          </div>

          {/* ---------- Fotos (mock counter) ---------- */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">
              Fotos
            </label>
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => setPhotoCount((c) => c + 1)}
                className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
              >
                <Camera className="h-4 w-4" />
                Foto hinzufuegen
              </button>
              <span className="text-sm text-foreground font-medium tabular-nums">
                {photoCount} {photoCount === 1 ? 'Foto' : 'Fotos'}
              </span>
              {photoCount > 0 && (
                <button
                  type="button"
                  onClick={() => setPhotoCount((c) => Math.max(0, c - 1))}
                  className="text-xs text-muted-foreground hover:text-error transition-colors"
                >
                  Letztes entfernen
                </button>
              )}
            </div>
          </div>

          {/* ---------- Notizen ---------- */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">
              Notizen
            </label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
              placeholder="Allgemeine Anmerkungen zum Zustand..."
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder resize-none focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* ---------- Unterschrift ---------- */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">
              Unterschrift
            </label>
            {signatureDataUrl ? (
              <div className="space-y-2">
                <div className="rounded-lg border border-border bg-white dark:bg-card p-2 inline-block">
                  <img
                    src={signatureDataUrl}
                    alt="Unterschrift"
                    loading="lazy"
                    decoding="async"
                    className="h-[100px] object-contain"
                  />
                </div>
                <div>
                  <button
                    type="button"
                    onClick={() => setSignatureDataUrl(undefined)}
                    className="text-xs text-muted-foreground hover:text-error transition-colors"
                  >
                    Unterschrift entfernen
                  </button>
                </div>
              </div>
            ) : (
              <SignatureCanvas
                width={380}
                height={150}
                onSave={(dataUrl) => setSignatureDataUrl(dataUrl)}
              />
            )}
          </div>

          {/* ---------- Erstellt von ---------- */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">
              Erstellt von <span className="text-error">*</span>
            </label>
            <input
              type="text"
              value={createdBy}
              onChange={(e) => setCreatedBy(e.target.value)}
              placeholder="Name des Erstellers"
              className="w-full max-w-[300px] rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
        </div>

        {/* ---- Footer ---- */}
        <DialogFooter className="px-6 py-4 border-t border-border flex-shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            Abbrechen
          </button>
          <button
            type="button"
            onClick={handleSave}
            className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            Protokoll speichern
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
