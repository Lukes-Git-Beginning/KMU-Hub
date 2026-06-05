/**
 * HoursToInvoiceDialog (Work module) — Convert project time entries to invoice.
 *
 * Shows a table of time entries for the current project with editable hourly
 * rates. User can select entries and generate an invoice (placeholder toast).
 * Mock data for design — backend swap: real time entries from Zeiterfassung API.
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Clock,
  FileText,
  Check,
  Calculator,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatCurrency } from '@/lib'
import { formatDate } from '@/lib/format'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface TimeEntry {
  id: string
  date: string
  task: string
  person: string
  hours: number
  description: string
}

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

// TODO: Replace with useProjectTimeEntries(projectId, { billed: false }) hook
// Current API only supports useTimeEntries(taskId) — need project-level endpoint
// Backend ticket needed: GET /api/v1/projects/{id}/time-entries?billed=false
const MOCK_TIME_ENTRIES: TimeEntry[] = [
  { id: 'te-w1', date: '2026-02-20', task: 'Frontend-Entwicklung', person: 'Anna Müller', hours: 6.0, description: 'React-Komponenten implementiert' },
  { id: 'te-w2', date: '2026-02-20', task: 'Design-Review', person: 'Anna Müller', hours: 1.5, description: 'Figma-Prototyp reviewt' },
  { id: 'te-w3', date: '2026-02-19', task: 'API-Endpoints', person: 'Thomas Fischer', hours: 5.0, description: 'REST-Endpoints aufgebaut' },
  { id: 'te-w4', date: '2026-02-19', task: 'Testing', person: 'Thomas Fischer', hours: 2.5, description: 'Integration Tests geschrieben' },
  { id: 'te-w5', date: '2026-02-18', task: 'Deployment', person: 'Max Schmidt', hours: 3.0, description: 'CI/CD Pipeline eingerichtet' },
  { id: 'te-w6', date: '2026-02-18', task: 'Datenbank-Migration', person: 'Thomas Fischer', hours: 4.0, description: 'PostgreSQL Migrations' },
  { id: 'te-w7', date: '2026-02-17', task: 'UI-Polishing', person: 'Sara Weber', hours: 5.5, description: 'Feinschliff + Animationen' },
  { id: 'te-w8', date: '2026-02-17', task: 'Performance', person: 'Anna Müller', hours: 2.0, description: 'Lighthouse-Optimierung' },
]

const DEFAULT_HOURLY_RATE = 120

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface HoursToInvoiceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectName?: string
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function HoursToInvoiceDialog({
  open,
  onOpenChange,
  projectName,
}: HoursToInvoiceDialogProps) {
  const { t } = useTranslation()
  const [selectedIds, setSelectedIds] = useState<Set<string>>(
    () => new Set(MOCK_TIME_ENTRIES.map((e) => e.id))
  )
  const [rates, setRates] = useState<Record<string, number>>(() => {
    const map: Record<string, number> = {}
    for (const entry of MOCK_TIME_ENTRIES) {
      map[entry.id] = DEFAULT_HOURLY_RATE
    }
    return map
  })

  const toggleEntry = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleAll = () => {
    if (selectedIds.size === MOCK_TIME_ENTRIES.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(MOCK_TIME_ENTRIES.map((e) => e.id)))
    }
  }

  const updateRate = (id: string, value: string) => {
    const num = Number(value)
    if (!isNaN(num) && num >= 0) {
      setRates((prev) => ({ ...prev, [id]: num }))
    }
  }

  // Selected entries with computed totals
  const selectedEntries = useMemo(
    () => MOCK_TIME_ENTRIES.filter((e) => selectedIds.has(e.id)),
    [selectedIds]
  )

  const totalHours = useMemo(
    () => selectedEntries.reduce((sum, e) => sum + e.hours, 0),
    [selectedEntries]
  )

  const totalAmount = useMemo(
    () =>
      selectedEntries.reduce(
        (sum, e) => sum + e.hours * (rates[e.id] ?? DEFAULT_HOURLY_RATE),
        0
      ),
    [selectedEntries, rates]
  )

  const handleCreateInvoice = () => {
    toast.success(t('work.invoice.creating'), {
      description: `${selectedEntries.length} Positionen, ${formatCurrency(totalAmount)}`,
    })
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Clock className="h-5 w-5 text-primary" />
            {t('work.invoice.title')}
            {projectName && (
              <span className="text-sm font-normal text-muted-foreground">
                — {projectName}
              </span>
            )}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* Select all toggle */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Calculator className="h-4 w-4" />
              <span>
                {t('work.invoice.entriesSelected', { selected: selectedIds.size, total: MOCK_TIME_ENTRIES.length })}
              </span>
            </div>
            <button
              onClick={toggleAll}
              className="text-xs text-primary hover:underline"
            >
              {selectedIds.size === MOCK_TIME_ENTRIES.length
                ? t('work.invoice.deselectAll')
                : t('work.invoice.selectAll')}
            </button>
          </div>

          {/* Time entries table */}
          <div className="rounded-md border border-border overflow-hidden">
            {/* Header */}
            <div className="grid grid-cols-[28px_80px_1fr_110px_70px_80px_90px] gap-2 items-center bg-secondary/50 px-3 py-2 text-[10px] font-medium text-muted-foreground uppercase tracking-wider">
              <span />
              <span>{t('work.invoice.date')}</span>
              <span>{t('work.invoice.task')}</span>
              <span>{t('work.utilization.employee')}</span>
              <span className="text-right">{t('work.budget.hours')}</span>
              <span className="text-right">{t('work.invoice.rate')}</span>
              <span className="text-right">{t('work.budget.total')}</span>
            </div>

            {/* Rows */}
            {MOCK_TIME_ENTRIES.map((entry) => {
              const isSelected = selectedIds.has(entry.id)
              const rate = rates[entry.id] ?? DEFAULT_HOURLY_RATE
              const lineTotal = entry.hours * rate

              return (
                <div
                  key={entry.id}
                  className={`grid grid-cols-[28px_80px_1fr_110px_70px_80px_90px] gap-2 items-center px-3 py-2 border-t border-border transition-colors cursor-pointer ${
                    isSelected ? 'bg-primary/5' : 'hover:bg-accent/30'
                  }`}
                  onClick={() => toggleEntry(entry.id)}
                >
                  {/* Checkbox */}
                  <div
                    className={`flex h-4 w-4 items-center justify-center rounded-sm border transition-colors ${
                      isSelected
                        ? 'border-primary bg-primary'
                        : 'border-border'
                    }`}
                  >
                    {isSelected && <Check className="h-3 w-3 text-white" />}
                  </div>

                  {/* Date */}
                  <span className="text-xs text-muted-foreground">
                    {formatDate(entry.date)}
                  </span>

                  {/* Task */}
                  <div className="min-w-0">
                    <p className="text-xs font-medium text-foreground truncate">
                      {entry.task}
                    </p>
                    <p className="text-[10px] text-muted-foreground truncate">
                      {entry.description}
                    </p>
                  </div>

                  {/* Person */}
                  <span className="text-xs text-muted-foreground truncate">
                    {entry.person}
                  </span>

                  {/* Hours */}
                  <span className="text-xs text-foreground text-right font-mono">
                    {entry.hours.toFixed(1)}h
                  </span>

                  {/* Rate (editable) */}
                  <div className="text-right" onClick={(e) => e.stopPropagation()}>
                    <input
                      type="number"
                      value={rate}
                      onChange={(e) => updateRate(entry.id, e.target.value)}
                      className="w-16 rounded border border-border bg-transparent px-1.5 py-0.5 text-xs text-right text-foreground outline-none focus:border-primary"
                      min={0}
                      step={5}
                    />
                  </div>

                  {/* Line total */}
                  <span className="text-xs font-medium text-foreground text-right">
                    {formatCurrency(lineTotal)}
                  </span>
                </div>
              )
            })}
          </div>

          {/* Summary row */}
          <div className="flex items-center justify-between rounded-md bg-secondary/50 px-4 py-3">
            <div className="text-xs text-muted-foreground space-x-2">
              <span>
                <span className="font-medium text-foreground">{selectedIds.size}</span> {t('work.invoice.entries')}
              </span>
              <span>|</span>
              <span>
                <span className="font-medium text-foreground">{totalHours.toFixed(1)}h</span> {t('work.invoice.total')}
              </span>
              <span>|</span>
              <span className="font-semibold text-primary">
                {formatCurrency(totalAmount)}
              </span>
            </div>
            <Button
              size="sm"
              disabled={selectedIds.size === 0}
              onClick={handleCreateInvoice}
              className="gap-1.5"
            >
              <FileText className="h-3.5 w-3.5" />
              {t('work.invoice.createInvoice')}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
