/**
 * HoursToInvoiceDialog (Work module) — Convert project time entries to a real
 * draft invoice.
 *
 * Tracked hours come from MSW (GET /projects/:id/time-entries?billed=false via
 * useProjectTimeEntries). Selecting entries and confirming creates a *draft*
 * invoice through useCreateInvoice — it appears in finanzen → Rechnungen for
 * the bookkeeping team to review. This is the "work generates the deliverable
 * → bill it → draft goes to accounting" flow.
 */
import { useState, useMemo, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Clock,
  FileText,
  Check,
  Calculator,
  Loader2,
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
import { useProjectTimeEntries } from '@/api/hooks/useProjects'
import { useCreateInvoice } from '@/api/hooks/useFinance'

const DEFAULT_HOURLY_RATE = 120

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface HoursToInvoiceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  projectName?: string
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function HoursToInvoiceDialog({
  open,
  onOpenChange,
  projectId,
  projectName,
}: HoursToInvoiceDialogProps) {
  const { t } = useTranslation()
  const { data: entries = [], isLoading } = useProjectTimeEntries(projectId, false)
  const createInvoice = useCreateInvoice()

  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [rates, setRates] = useState<Record<string, number>>({})
  const [customerName, setCustomerName] = useState('')

  // On open: preselect all unbilled entries and prefill the customer with the
  // project name (editable). Keyed on open + entry identity so reopening resets.
  useEffect(() => {
    if (!open) return
    setSelectedIds(new Set(entries.map((e) => e.id)))
    setCustomerName((prev) => prev || projectName || '')
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional: only on open / data arrival
  }, [open, entries.length])

  const toggleEntry = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleAll = () => {
    if (selectedIds.size === entries.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(entries.map((e) => e.id)))
    }
  }

  const updateRate = (id: string, value: string) => {
    const num = Number(value)
    if (!isNaN(num) && num >= 0) {
      setRates((prev) => ({ ...prev, [id]: num }))
    }
  }

  const selectedEntries = useMemo(
    () => entries.filter((e) => selectedIds.has(e.id)),
    [entries, selectedIds]
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
    const invoiceDate = new Date().toISOString().split('T')[0]
    createInvoice.mutate(
      {
        customer: {
          name: customerName.trim() || projectName || '',
          address: '',
          email: '',
        },
        tax_mode: 'standard',
        currency: 'EUR',
        invoice_date: invoiceDate,
        payment_terms_days: 30,
        notes: t('work.hoursInvoice.note', { project: projectName ?? '' }),
        line_items: selectedEntries.map((e, i) => ({
          position: i + 1,
          description: e.description ? `${e.task} — ${e.description}` : e.task,
          quantity: e.hours.toFixed(2),
          unit_price: String(rates[e.id] ?? DEFAULT_HOURLY_RATE),
          tax_rate: '19',
        })),
      },
      {
        onSuccess: (data) => {
          toast.success(
            t('work.hoursInvoice.created', {
              number: data.invoice?.invoice_number ?? '',
            })
          )
          onOpenChange(false)
        },
        onError: () => toast.error(t('work.hoursInvoice.createError')),
      }
    )
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
          {/* Customer for the generated draft invoice */}
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground shrink-0">
              {t('work.hoursInvoice.customer')}:
            </span>
            <input
              type="text"
              value={customerName}
              onChange={(e) => setCustomerName(e.target.value)}
              placeholder={t('work.hoursInvoice.customerPlaceholder')}
              className="flex-1 rounded-md border border-border bg-transparent px-2.5 py-1.5 text-sm text-foreground outline-none placeholder:text-input-placeholder focus:border-primary"
            />
          </div>

          {/* Select all toggle */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Calculator className="h-4 w-4" />
              <span>
                {t('work.invoice.entriesSelected', { selected: selectedIds.size, total: entries.length })}
              </span>
            </div>
            {entries.length > 0 && (
              <button
                onClick={toggleAll}
                className="text-xs text-primary hover:underline"
              >
                {selectedIds.size === entries.length
                  ? t('work.invoice.deselectAll')
                  : t('work.invoice.selectAll')}
              </button>
            )}
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
            {isLoading ? (
              <div className="flex items-center justify-center border-t border-border py-10 text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
              </div>
            ) : entries.length === 0 ? (
              <div className="border-t border-border py-10 text-center text-xs text-muted-foreground">
                {t('work.hoursInvoice.emptyHint')}
              </div>
            ) : (
              entries.map((entry) => {
                const isSelected = selectedIds.has(entry.id)
                const rate = rates[entry.id] ?? DEFAULT_HOURLY_RATE
                const lineTotal = entry.hours * rate

                return (
                  <div
                    key={entry.id}
                    role="button"
                    tabIndex={0}
                    className={`grid grid-cols-[28px_80px_1fr_110px_70px_80px_90px] gap-2 items-center px-3 py-2 border-t border-border transition-colors cursor-pointer focus-visible:outline-none focus-visible:bg-accent/40 ${
                      isSelected ? 'bg-primary/5' : 'hover:bg-accent/30'
                    }`}
                    onClick={() => toggleEntry(entry.id)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        toggleEntry(entry.id)
                      }
                    }}
                  >
                    {/* Checkbox */}
                    <div
                      className={`flex h-4 w-4 items-center justify-center rounded-sm border transition-colors ${
                        isSelected ? 'border-primary bg-primary' : 'border-border'
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
              })
            )}
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
              disabled={selectedIds.size === 0 || !customerName.trim() || createInvoice.isPending}
              onClick={handleCreateInvoice}
              className="gap-1.5"
            >
              {createInvoice.isPending ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <FileText className="h-3.5 w-3.5" />
              )}
              {t('work.invoice.createInvoice')}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
