import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Trash2, AlertTriangle } from 'lucide-react'
import { toast } from 'sonner'
import { useCreateCreditNote, useInvoices } from '@/api/hooks/useFinance'
import { formatMoney, calcLineTotal, calcInvoiceSubtotal, calcInvoiceTax, calcInvoiceTotal } from '@/stores/finance'
import type { Invoice, TaxMode } from '@/types/finance-types'

interface LineItemDraft {
  key: string
  description: string
  quantity: string
  unit_price: string
  tax_rate: string
}

interface CreditNoteDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  preselectedInvoice?: Invoice | null
  /** When true, the dialog issues a full credit note that cancels the invoice. */
  isStorno?: boolean
}

export function CreditNoteDialog({
  open,
  onOpenChange,
  preselectedInvoice,
  isStorno = false,
}: CreditNoteDialogProps) {
  const { t } = useTranslation()
  const createCreditNote = useCreateCreditNote()
  const { data: invoicesData } = useInvoices()

  const [selectedInvoiceId, setSelectedInvoiceId] = useState('')
  const [taxMode, setTaxMode] = useState<TaxMode>('standard')
  const [reason, setReason] = useState('')
  const [items, setItems] = useState<LineItemDraft[]>([])

  // Filter to invoices that can have credit notes (sent, paid, overdue)
  const eligibleInvoices = (invoicesData?.invoices ?? []).filter(
    (inv) => inv.status === 'sent' || inv.status === 'paid' || inv.status === 'overdue',
  )

  // Find the selected invoice from the list or preselected
  const selectedInvoice =
    preselectedInvoice ??
    eligibleInvoices.find((inv) => inv.id === selectedInvoiceId)

  function populateFromInvoice(inv: Invoice) {
    setTaxMode(inv.tax_mode ?? 'standard')
    // List payloads carry `items` (description/quantity/unit_price/total) while
    // the detail endpoint normalises to `line_items`. Accept both shapes so the
    // dialog works whether opened from the list or from a detail view.
    const raw = inv as unknown as {
      line_items?: Array<Record<string, unknown>>
      items?: Array<Record<string, unknown>>
      tax_rate?: number | string
    }
    const source = raw.line_items ?? raw.items ?? []
    setItems(
      source.map((li, idx) => ({
        key: String(li.id ?? idx),
        description: String(li.description ?? ''),
        quantity: String(li.quantity ?? '1'),
        unit_price: String(li.unit_price ?? '0'),
        tax_rate: String(li.tax_rate ?? raw.tax_rate ?? '19'),
      })),
    )
  }

   
  useEffect(() => {
    if (!open) return
    if (preselectedInvoice) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from dialog props
      setSelectedInvoiceId(preselectedInvoice.id)
      populateFromInvoice(preselectedInvoice)
      setReason(
        isStorno
          ? t('finanzen.creditNote.stornoReasonDefault', { number: preselectedInvoice.invoice_number })
          : '',
      )
    } else {
      setSelectedInvoiceId('')
      setTaxMode('standard')
      setReason('')
      setItems([])
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- t intentionally omitted
  }, [open, preselectedInvoice, isStorno])

  const handleInvoiceSelect = (invoiceId: string) => {
    setSelectedInvoiceId(invoiceId)
    const inv = eligibleInvoices.find((i) => i.id === invoiceId)
    if (inv) {
      populateFromInvoice(inv)
    }
  }

  const updateItem = (idx: number, updates: Partial<LineItemDraft>) => {
    setItems((prev) =>
      prev.map((item, i) => (i === idx ? { ...item, ...updates } : item)),
    )
  }

  const removeItem = (idx: number) => {
    setItems((prev) => prev.filter((_, i) => i !== idx))
  }

  const handleSave = () => {
    if (!selectedInvoice) return
    if (!reason.trim()) {
      toast.error(t('finanzen.creditNote.reasonRequired'))
      return
    }
    const validItems = items.filter(
      (i) => i.description.trim() && Number(i.unit_price) > 0,
    )
    if (validItems.length === 0) return

    createCreditNote.mutate(
      {
        original_invoice_id: selectedInvoice.id,
        line_items: validItems.map((i, idx) => ({
          position: idx + 1,
          description: i.description.trim(),
          quantity: i.quantity,
          unit_price: i.unit_price,
          tax_rate: i.tax_rate,
        })),
        tax_mode: taxMode,
        reason: reason.trim(),
        is_storno: isStorno,
      },
      {
        onSuccess: () => {
          toast.success(
            isStorno
              ? t('finanzen.creditNote.stornoDone', { number: selectedInvoice.invoice_number })
              : t('finanzen.creditNote.created'),
          )
          onOpenChange(false)
        },
        onError: (err) => toast.error(err.message),
      },
    )
  }

  const effectiveItems = items.map((i) => ({
    quantity: i.quantity,
    unit_price: i.unit_price,
    tax_rate: taxMode === 'kleinunternehmer' ? '0' : i.tax_rate,
  }))
  const subtotal = calcInvoiceSubtotal(effectiveItems)
  const tax = calcInvoiceTax(effectiveItems)
  const total = calcInvoiceTotal(effectiveItems)
  const currency = selectedInvoice?.currency ?? 'EUR'

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>
            {isStorno ? t('finanzen.creditNote.stornoTitle') : t('finanzen.creditNote.createTitle')}
          </DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 py-2">
          {/* Storno notice */}
          {isStorno && selectedInvoice && (
            <div className="flex items-start gap-2 rounded-lg border border-error/30 bg-error/5 p-3 text-xs text-error">
              <AlertTriangle className="h-4 w-4 mt-0.5 shrink-0" />
              <span>{t('finanzen.creditNote.stornoNotice', { number: selectedInvoice.invoice_number })}</span>
            </div>
          )}

          {/* Invoice selection */}
          {!preselectedInvoice && (
            <div className="space-y-1.5">
              <Label>{t('finanzen.creditNote.originalInvoice')} *</Label>
              <Select value={selectedInvoiceId} onValueChange={handleInvoiceSelect}>
                <SelectTrigger>
                  <SelectValue placeholder={t('finanzen.creditNote.selectInvoice')} />
                </SelectTrigger>
                <SelectContent>
                  {eligibleInvoices.map((inv) => (
                    <SelectItem key={inv.id} value={inv.id}>
                      {inv.invoice_number} - {inv.customer?.name ?? ''} ({formatMoney(inv.tax_breakdown?.gross_total ?? inv.total_gross ?? 0, inv.currency)})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          {selectedInvoice && (
            <>
              {/* Customer info (read-only) */}
              <div className="rounded-md bg-secondary/50 p-3 text-xs space-y-1">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">{t('finanzen.invoice')}</span>
                  <span className="font-medium text-foreground font-mono">
                    {selectedInvoice.invoice_number}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">{t('finanzen.customer')}</span>
                  <span className="text-foreground">{selectedInvoice.customer?.name ?? ''}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">{t('finanzen.invoiceAmount')}</span>
                  <span className="font-medium text-foreground">
                    {formatMoney(selectedInvoice.tax_breakdown?.gross_total ?? selectedInvoice.total_gross ?? 0, currency)}
                  </span>
                </div>
              </div>

              {/* Reason */}
              <div className="space-y-1.5">
                <Label>{t('finanzen.creditNote.reason')} *</Label>
                <Textarea
                  placeholder={t('finanzen.creditNote.reasonPlaceholder')}
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  rows={2}
                />
              </div>

              {/* Line items (adjustable) */}
              <div className="space-y-2">
                <Label>{t('finanzen.creditNote.lineItemsLabel')}</Label>
                <div className="rounded-lg border border-border overflow-hidden">
                  <div className="grid grid-cols-[1fr_70px_90px_80px_90px_32px] gap-2 px-3 py-2 text-[10px] font-medium text-muted-foreground bg-secondary/30 uppercase tracking-wider">
                    <span>{t('finanzen.lineItems.description')}</span>
                    <span>{t('finanzen.lineItems.quantity')}</span>
                    <span>{t('finanzen.lineItems.unitPrice')}</span>
                    <span>{t('finanzen.lineItems.vat')}</span>
                    <span className="text-right">{t('finanzen.lineItems.total')}</span>
                    <span />
                  </div>
                  {items.map((item, idx) => (
                    <div
                      key={item.key}
                      className="grid grid-cols-[1fr_70px_90px_80px_90px_32px] gap-2 px-3 py-1.5 border-t border-border-muted items-center"
                    >
                      <Input
                        value={item.description}
                        onChange={(e) => updateItem(idx, { description: e.target.value })}
                        className="h-7 text-xs"
                      />
                      <Input
                        type="number"
                        min={0.01}
                        step={0.5}
                        value={item.quantity}
                        onChange={(e) => updateItem(idx, { quantity: e.target.value })}
                        className="h-7 text-xs"
                      />
                      <Input
                        type="number"
                        min={0}
                        step={0.01}
                        value={item.unit_price}
                        onChange={(e) => updateItem(idx, { unit_price: e.target.value })}
                        className="h-7 text-xs"
                      />
                      <span className="text-[10px] text-muted-foreground px-1">
                        {item.tax_rate}%
                      </span>
                      <span className="text-xs text-foreground text-right font-medium">
                        {formatMoney(calcLineTotal(item.quantity, item.unit_price), currency)}
                      </span>
                      <button
                        onClick={() => removeItem(idx)}
                        className="rounded p-0.5 text-muted-foreground hover:text-destructive"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>

              {/* Totals */}
              <div className="flex justify-end">
                <div className="w-64 space-y-1.5 text-xs">
                  <div className="flex justify-between text-muted-foreground">
                    <span>{t('finanzen.totals.subtotalNet')}</span>
                    <span>{formatMoney(subtotal, currency)}</span>
                  </div>
                  {taxMode !== 'kleinunternehmer' && (
                    <div className="flex justify-between text-muted-foreground">
                      <span>{t('finanzen.lineItems.vat')}</span>
                      <span>{formatMoney(tax, currency)}</span>
                    </div>
                  )}
                  <div className="flex justify-between font-medium text-sm text-foreground border-t border-border pt-1.5">
                    <span>{t('finanzen.creditNote.creditAmount')}</span>
                    <span>{formatMoney(total, currency)}</span>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 pt-3 border-t border-border">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            onClick={handleSave}
            variant={isStorno ? 'destructive' : 'default'}
            disabled={
              !selectedInvoice || !reason.trim() || items.length === 0 || createCreditNote.isPending
            }
          >
            {createCreditNote.isPending
              ? t('finanzen.creditNote.creating')
              : isStorno
                ? t('finanzen.creditNote.stornoConfirm')
                : t('finanzen.creditNote.createTitle')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
