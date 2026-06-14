import { useState, useEffect, useCallback } from 'react'
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
import { Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  useCreateRecurringInvoice,
  useUpdateRecurringInvoice,
} from '@/api/hooks/useFinance'
import { formatMoney, calcLineTotal, calcInvoiceSubtotal, calcInvoiceTax, calcInvoiceTotal } from '@/stores/finance'
import type {
  RecurringInvoice,
  RecurringInterval,
  TaxMode,
  Currency,
} from '@/types/finance-types'

interface LineItemDraft {
  key: string
  description: string
  quantity: string
  unit_price: string
  tax_rate: string
}

const INTERVALS: RecurringInterval[] = ['weekly', 'monthly', 'quarterly', 'yearly']
const CURRENCIES: Currency[] = ['EUR', 'CHF', 'USD']

function emptyItem(): LineItemDraft {
  return { key: String(Date.now() + Math.floor(performance.now())), description: '', quantity: '1', unit_price: '0', tax_rate: '19' }
}

interface RecurringInvoiceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  editRecurring?: RecurringInvoice | null
}

export function RecurringInvoiceDialog({ open, onOpenChange, editRecurring }: RecurringInvoiceDialogProps) {
  const { t } = useTranslation()
  const createRec = useCreateRecurringInvoice()
  const updateRec = useUpdateRecurringInvoice()

  const [title, setTitle] = useState('')
  const [customerName, setCustomerName] = useState('')
  const [customerEmail, setCustomerEmail] = useState('')
  const [customerAddress, setCustomerAddress] = useState('')
  const [interval, setInterval] = useState<RecurringInterval>('monthly')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [paymentTermsDays, setPaymentTermsDays] = useState('14')
  const [taxMode, setTaxMode] = useState<TaxMode>('standard')
  const [currency, setCurrency] = useState<Currency>('EUR')
  const [notes, setNotes] = useState('')
  const [items, setItems] = useState<LineItemDraft[]>([emptyItem()])

  const resetForm = useCallback(() => {
    setTitle('')
    setCustomerName('')
    setCustomerEmail('')
    setCustomerAddress('')
    setInterval('monthly')
    setStartDate(new Date().toISOString().split('T')[0])
    setEndDate('')
    setPaymentTermsDays('14')
    setTaxMode('standard')
    setCurrency('EUR')
    setNotes('')
    setItems([emptyItem()])
  }, [])

  useEffect(() => {
    if (!open) return
    if (editRecurring) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop
      setTitle(editRecurring.title)
      setCustomerName(editRecurring.customer.name)
      setCustomerEmail(editRecurring.customer.email)
      setCustomerAddress(editRecurring.customer.address)
      setInterval(editRecurring.interval)
      setStartDate(editRecurring.start_date)
      setEndDate(editRecurring.end_date ?? '')
      setPaymentTermsDays(String(editRecurring.payment_terms_days))
      setTaxMode(editRecurring.tax_mode)
      setCurrency(editRecurring.currency ?? 'EUR')
      setNotes(editRecurring.notes ?? '')
      setItems(
        editRecurring.line_items.map((li, idx) => ({
          key: li.id || String(idx),
          description: li.description,
          quantity: li.quantity,
          unit_price: li.unit_price,
          tax_rate: li.tax_rate,
        })),
      )
    } else {
      resetForm()
    }
  }, [open, editRecurring, resetForm])

  const updateItem = (idx: number, patch: Partial<LineItemDraft>) => {
    setItems((prev) => prev.map((it, i) => (i === idx ? { ...it, ...patch } : it)))
  }
  const addItem = () => setItems((prev) => [...prev, emptyItem()])
  const removeItem = (idx: number) => setItems((prev) => (prev.length <= 1 ? prev : prev.filter((_, i) => i !== idx)))

  const effectiveItems = items.map((i) => ({
    quantity: i.quantity,
    unit_price: i.unit_price,
    tax_rate: taxMode === 'kleinunternehmer' ? '0' : i.tax_rate,
  }))
  const subtotal = calcInvoiceSubtotal(effectiveItems)
  const tax = calcInvoiceTax(effectiveItems)
  const total = calcInvoiceTotal(effectiveItems)

  const handleSave = () => {
    if (!title.trim() || !customerName.trim()) {
      toast.error(t('finanzen.recurring.titleCustomerRequired'))
      return
    }
    const validItems = items.filter((i) => i.description.trim() && Number(i.unit_price) > 0)
    if (validItems.length === 0) return

    const payload = {
      title: title.trim(),
      customer: {
        name: customerName.trim(),
        address: customerAddress.trim(),
        email: customerEmail.trim(),
      },
      line_items: validItems.map((i, idx) => ({
        position: idx + 1,
        description: i.description.trim(),
        quantity: i.quantity,
        unit_price: i.unit_price,
        tax_rate: taxMode === 'kleinunternehmer' ? '0' : i.tax_rate,
      })),
      tax_mode: taxMode,
      currency,
      interval,
      start_date: startDate,
      end_date: endDate || undefined,
      payment_terms_days: Number(paymentTermsDays),
      notes: notes.trim() || undefined,
    }

    if (editRecurring) {
      updateRec.mutate(
        { id: editRecurring.id, ...payload },
        {
          onSuccess: () => {
            toast.success(t('finanzen.recurring.updated'))
            onOpenChange(false)
          },
          onError: (err) => toast.error(err.message),
        },
      )
    } else {
      createRec.mutate(payload, {
        onSuccess: () => {
          toast.success(t('finanzen.recurring.created'))
          onOpenChange(false)
        },
        onError: (err) => toast.error(err.message),
      })
    }
  }

  const isPending = createRec.isPending || updateRec.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>
            {editRecurring ? t('finanzen.recurring.editTitle') : t('finanzen.recurring.createTitle')}
          </DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 py-2">
          {/* Title */}
          <div className="space-y-1.5">
            <Label>{t('finanzen.recurring.title')} *</Label>
            <Input
              placeholder={t('finanzen.recurring.titlePlaceholder')}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
          </div>

          {/* Customer */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('finanzen.customer')} *</Label>
              <Input
                placeholder={t('finanzen.invoiceForm.companyPerson')}
                value={customerName}
                onChange={(e) => setCustomerName(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>{t('finanzen.invoiceForm.email')}</Label>
              <Input
                type="email"
                placeholder="email@firma.de"
                value={customerEmail}
                onChange={(e) => setCustomerEmail(e.target.value)}
              />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label>{t('finanzen.invoiceForm.address')}</Label>
            <Textarea
              placeholder={t('finanzen.invoiceForm.addressPlaceholder')}
              value={customerAddress}
              onChange={(e) => setCustomerAddress(e.target.value)}
              rows={2}
            />
          </div>

          {/* Schedule */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <div className="space-y-1.5">
              <Label>{t('finanzen.recurring.interval')}</Label>
              <Select value={interval} onValueChange={(v) => setInterval(v as RecurringInterval)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {INTERVALS.map((iv) => (
                    <SelectItem key={iv} value={iv}>{t(`finanzen.recurring.intervals.${iv}`)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('finanzen.recurring.startDate')}</Label>
              <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('finanzen.recurring.endDate')}</Label>
              <Input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('finanzen.invoiceForm.paymentTermsDays')}</Label>
              <Input
                type="number"
                min={1}
                value={paymentTermsDays}
                onChange={(e) => setPaymentTermsDays(e.target.value)}
              />
            </div>
          </div>

          {/* Tax & currency */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('finanzen.invoiceForm.taxTreatment')}</Label>
              <Select value={taxMode} onValueChange={(v) => setTaxMode(v as TaxMode)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="standard">Standard (19% / 7%)</SelectItem>
                  <SelectItem value="reverse_charge">Reverse Charge</SelectItem>
                  <SelectItem value="kleinunternehmer">Kleinunternehmer</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('finanzen.invoiceForm.currency')}</Label>
              <Select value={currency} onValueChange={(v) => setCurrency(v as Currency)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {CURRENCIES.map((c) => (
                    <SelectItem key={c} value={c}>{c}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Line items */}
          <div className="space-y-2">
            <Label>{t('finanzen.lineItems.positions')}</Label>
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
                    placeholder={t('finanzen.lineItems.descriptionPlaceholder')}
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
                  {taxMode !== 'kleinunternehmer' ? (
                    <Input
                      type="number"
                      min={0}
                      step={0.1}
                      value={item.tax_rate}
                      onChange={(e) => updateItem(idx, { tax_rate: e.target.value })}
                      className="h-7 text-xs"
                    />
                  ) : (
                    <span className="text-xs text-muted-foreground px-1">0%</span>
                  )}
                  <span className="text-xs text-foreground text-right font-medium">
                    {formatMoney(calcLineTotal(item.quantity, item.unit_price), currency)}
                  </span>
                  <button
                    onClick={() => removeItem(idx)}
                    disabled={items.length <= 1}
                    className="rounded p-0.5 text-muted-foreground hover:text-destructive disabled:opacity-30"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
              <button
                onClick={addItem}
                className="flex items-center gap-1.5 w-full px-3 py-2 text-xs text-primary hover:bg-primary/5 transition-colors border-t border-border-muted"
              >
                <Plus className="h-3.5 w-3.5" />
                {t('finanzen.lineItems.addPosition')}
              </button>
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
                <span>{t('finanzen.recurring.perInvoice')}</span>
                <span>{formatMoney(total, currency)}</span>
              </div>
            </div>
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <Label>{t('finanzen.notes')}</Label>
            <Textarea
              placeholder={t('finanzen.invoiceForm.notesPlaceholder')}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={2}
            />
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 pt-3 border-t border-border">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSave} disabled={!title.trim() || !customerName.trim() || isPending}>
            {isPending ? t('finanzen.saving') : editRecurring ? t('common.save') : t('common.create')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
