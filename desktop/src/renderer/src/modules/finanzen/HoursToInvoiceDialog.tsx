/**
 * HoursToInvoiceDialog — Convert time entries into invoice line items.
 *
 * Select time entries, set hourly rate, preview line items, then generate invoice.
 * Mock data for design — backend swap: real time entries from Zeiterfassung API.
 */
import { useState, useMemo } from 'react'
import {
  Clock,
  FileText,
  ArrowRight,
  Check,
  Calculator,
  Calendar,
  FolderKanban,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface TimeEntry {
  id: string
  date: string
  project: string
  task: string
  employee: string
  hours: number
  description: string
  billed: boolean
}

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

// TODO: Replace mock data with API call — Backend needed: GET /api/v1/time-entries?billed=false
// Fetch unbilled time entries from Zeiterfassung service, with project/employee filtering
const mockTimeEntries: TimeEntry[] = [
  { id: 'te1', date: '2026-02-18', project: 'Website Redesign', task: 'Frontend-Entwicklung', employee: 'Anna Müller', hours: 4.5, description: 'React-Komponenten implementiert', billed: false },
  { id: 'te2', date: '2026-02-18', project: 'Website Redesign', task: 'Design-Review', employee: 'Anna Müller', hours: 1.5, description: 'Figma-Prototyp reviewt', billed: false },
  { id: 'te3', date: '2026-02-17', project: 'CRM Integration', task: 'API-Entwicklung', employee: 'Thomas Fischer', hours: 6.0, description: 'REST-Endpoints implementiert', billed: false },
  { id: 'te4', date: '2026-02-17', project: 'CRM Integration', task: 'Testing', employee: 'Thomas Fischer', hours: 2.0, description: 'Integration Tests geschrieben', billed: false },
  { id: 'te5', date: '2026-02-16', project: 'Website Redesign', task: 'Deployment', employee: 'Max Schmidt', hours: 3.0, description: 'Staging-Deployment + DNS', billed: false },
  { id: 'te6', date: '2026-02-15', project: 'Mobile App', task: 'UI-Development', employee: 'Sara Weber', hours: 5.5, description: 'React Native Screens', billed: false },
  { id: 'te7', date: '2026-02-15', project: 'CRM Integration', task: 'Datenbank-Migration', employee: 'Thomas Fischer', hours: 3.5, description: 'PostgreSQL Migrations', billed: false },
  { id: 'te8', date: '2026-02-14', project: 'Website Redesign', task: 'Performance', employee: 'Anna Müller', hours: 2.0, description: 'Lighthouse-Optimierung', billed: false },
  { id: 'te9', date: '2026-02-13', project: 'Beratung', task: 'Workshop', employee: 'Max Schmidt', hours: 4.0, description: 'Onboarding-Workshop beim Kunden', billed: true },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface HoursToInvoiceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function HoursToInvoiceDialog({
  open,
  onOpenChange,
}: HoursToInvoiceDialogProps) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [hourlyRate, setHourlyRate] = useState('150')
  const [groupBy, setGroupBy] = useState<'project' | 'task' | 'date'>('project')
  const [step, setStep] = useState<'select' | 'preview'>('select')

  const unbilledEntries = mockTimeEntries.filter((e) => !e.billed)

  const toggleEntry = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleAll = () => {
    if (selectedIds.size === unbilledEntries.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(unbilledEntries.map((e) => e.id)))
    }
  }

  const selectedEntries = useMemo(() => unbilledEntries.filter((e) => selectedIds.has(e.id)), [unbilledEntries, selectedIds])
  const totalHours = selectedEntries.reduce((sum, e) => sum + e.hours, 0)
  const rate = Number(hourlyRate) || 0
  const totalAmount = totalHours * rate

  // Group selected entries into invoice line items
  const lineItems = useMemo(() => {
    const groups = new Map<string, { label: string; hours: number; entries: TimeEntry[] }>()

    for (const entry of selectedEntries) {
      const key =
        groupBy === 'project'
          ? entry.project
          : groupBy === 'task'
            ? `${entry.project} — ${entry.task}`
            : entry.date

      const existing = groups.get(key)
      if (existing) {
        existing.hours += entry.hours
        existing.entries.push(entry)
      } else {
        groups.set(key, {
          label: key,
          hours: entry.hours,
          entries: [entry],
        })
      }
    }

    return Array.from(groups.values())
  }, [selectedEntries, groupBy])

  const formatCHF = (v: number) =>
    new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'CHF' }).format(v)

  const handleCreateInvoice = () => {
    toast.success(`Rechnung mit ${lineItems.length} Positionen erstellt (${formatCHF(totalAmount)})`)
    onOpenChange(false)
    setStep('select')
    setSelectedIds(new Set())
  }

  const handleClose = () => {
    onOpenChange(false)
    setTimeout(() => {
      setStep('select')
    }, 200)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Clock className="h-5 w-5 text-primary" />
            Stunden zu Rechnung
          </DialogTitle>
        </DialogHeader>

        {step === 'select' && (
          <div className="space-y-4">
            {/* Hourly rate */}
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2">
                <Calculator className="h-4 w-4 text-muted-foreground" />
                <span className="text-xs text-muted-foreground">Stundensatz:</span>
                <div className="relative">
                  <input
                    type="number"
                    value={hourlyRate}
                    onChange={(e) => setHourlyRate(e.target.value)}
                    className="w-24 rounded-md border border-border bg-transparent px-2 py-1.5 text-sm text-foreground text-right outline-none focus:border-primary"
                    min={0}
                    step={5}
                  />
                  <span className="absolute right-2 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground">
                    CHF/h
                  </span>
                </div>
              </div>
              <div className="flex-1" />
              <button
                onClick={toggleAll}
                className="text-xs text-primary hover:underline"
              >
                {selectedIds.size === unbilledEntries.length ? 'Alle abwählen' : 'Alle wählen'}
              </button>
            </div>

            {/* Time entries list */}
            <div className="rounded-md border border-border overflow-hidden">
              <div className="grid grid-cols-[24px_80px_1fr_120px_60px_60px] gap-2 items-center bg-secondary/50 px-3 py-2 text-[10px] font-medium text-muted-foreground">
                <span />
                <span>Datum</span>
                <span>Projekt / Aufgabe</span>
                <span>Mitarbeiter</span>
                <span className="text-right">Stunden</span>
                <span className="text-right">Betrag</span>
              </div>
              {unbilledEntries.map((entry) => {
                const isSelected = selectedIds.has(entry.id)
                return (
                  <button
                    key={entry.id}
                    onClick={() => toggleEntry(entry.id)}
                    className={`grid w-full grid-cols-[24px_80px_1fr_120px_60px_60px] gap-2 items-center px-3 py-2 text-left border-t border-border transition-colors ${
                      isSelected ? 'bg-primary/5' : 'hover:bg-accent/30'
                    }`}
                  >
                    <div
                      className={`flex h-4 w-4 items-center justify-center rounded-sm border ${
                        isSelected
                          ? 'border-primary bg-primary'
                          : 'border-border'
                      }`}
                    >
                      {isSelected && <Check className="h-3 w-3 text-white" />}
                    </div>
                    <span className="text-xs text-muted-foreground">
                      {new Date(entry.date).toLocaleDateString('de-DE')}
                    </span>
                    <div className="min-w-0">
                      <p className="text-xs font-medium text-foreground truncate">{entry.project}</p>
                      <p className="text-[10px] text-muted-foreground truncate">{entry.task}</p>
                    </div>
                    <span className="text-xs text-muted-foreground truncate">{entry.employee}</span>
                    <span className="text-xs text-foreground text-right font-mono">
                      {entry.hours.toFixed(1)}h
                    </span>
                    <span className="text-xs text-foreground text-right">
                      {formatCHF(entry.hours * rate)}
                    </span>
                  </button>
                )
              })}
            </div>

            {/* Summary + actions */}
            <div className="flex items-center justify-between rounded-md bg-secondary/50 px-4 py-3">
              <div className="text-xs text-muted-foreground">
                <span className="font-medium text-foreground">{selectedIds.size}</span> Einträge |{' '}
                <span className="font-medium text-foreground">{totalHours.toFixed(1)}h</span> |{' '}
                <span className="font-medium text-primary">{formatCHF(totalAmount)}</span>
              </div>
              <button
                onClick={() => setStep('preview')}
                disabled={selectedIds.size === 0}
                className="flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                Vorschau
                <ArrowRight className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        )}

        {step === 'preview' && (
          <div className="space-y-4">
            {/* Group by selector */}
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">Gruppierung:</span>
              {([
                ['project', 'Projekt', FolderKanban],
                ['task', 'Aufgabe', FileText],
                ['date', 'Datum', Calendar],
              ] as const).map(([key, label, Icon]) => (
                <button
                  key={key}
                  onClick={() => setGroupBy(key)}
                  className={`flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs transition-colors ${
                    groupBy === key
                      ? 'border-primary bg-primary/5 text-primary font-medium'
                      : 'border-border text-muted-foreground hover:bg-accent'
                  }`}
                >
                  <Icon className="h-3 w-3" />
                  {label}
                </button>
              ))}
            </div>

            {/* Line items preview */}
            <div className="rounded-md border border-border overflow-hidden">
              <div className="grid grid-cols-[2rem_1fr_60px_80px_80px] gap-2 items-center bg-secondary/50 px-3 py-2 text-[10px] font-medium text-muted-foreground">
                <span>Pos.</span>
                <span>Beschreibung</span>
                <span className="text-right">Stunden</span>
                <span className="text-right">Einzelpreis</span>
                <span className="text-right">Gesamt</span>
              </div>
              {lineItems.map((item, i) => (
                <div
                  key={item.label}
                  className="grid grid-cols-[2rem_1fr_60px_80px_80px] gap-2 items-center px-3 py-2.5 border-t border-border"
                >
                  <span className="text-xs text-muted-foreground">{i + 1}</span>
                  <div className="min-w-0">
                    <p className="text-xs font-medium text-foreground truncate">{item.label}</p>
                    <p className="text-[10px] text-muted-foreground">
                      {item.entries.length} Eintrag{item.entries.length > 1 ? 'e' : ''}
                    </p>
                  </div>
                  <span className="text-xs text-foreground text-right font-mono">
                    {item.hours.toFixed(1)}h
                  </span>
                  <span className="text-xs text-muted-foreground text-right">
                    {formatCHF(rate)}/h
                  </span>
                  <span className="text-xs font-medium text-foreground text-right">
                    {formatCHF(item.hours * rate)}
                  </span>
                </div>
              ))}
            </div>

            {/* Totals */}
            <div className="flex justify-end">
              <div className="w-56 space-y-1.5 text-xs">
                <div className="flex justify-between text-muted-foreground">
                  <span>Stunden gesamt</span>
                  <span className="font-mono">{totalHours.toFixed(1)}h</span>
                </div>
                <div className="flex justify-between text-muted-foreground">
                  <span>Stundensatz</span>
                  <span>{formatCHF(rate)}</span>
                </div>
                <div className="flex justify-between font-medium text-sm text-foreground border-t border-border pt-1.5">
                  <span>Netto-Betrag</span>
                  <span>{formatCHF(totalAmount)}</span>
                </div>
              </div>
            </div>

            {/* Actions */}
            <div className="flex justify-between pt-1">
              <button
                onClick={() => setStep('select')}
                className="h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
              >
                Zurück
              </button>
              <button
                onClick={handleCreateInvoice}
                className="flex items-center gap-1.5 h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                <FileText className="h-3.5 w-3.5" />
                Rechnung erstellen
              </button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
