import { useState, useEffect, useCallback } from 'react'
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
import { Plus, Trash2, Info } from 'lucide-react'
import { toast } from 'sonner'
import { useCreateInvoice, useUpdateInvoice, useCompanySettings } from '@/api/hooks/useFinance'
import { calcLineTotal, calcInvoiceSubtotal, calcInvoiceTax, calcInvoiceTotal } from '@/stores/finance'
import type { Invoice, TaxMode } from '@/types/finance-types'

interface LineItemDraft {
  key: string
  position: number
  description: string
  quantity: string
  unit_price: string
  tax_rate: string
}

// ---------------------------------------------------------------------------
// Multi-country tax rates (3.15)
// ---------------------------------------------------------------------------

type TaxCountry = 'DE' | 'AT' | 'CH'

const TAX_RATES_BY_COUNTRY: Record<TaxCountry, { value: string; label: string }[]> = {
  DE: [
    { value: '19', label: '19% (Standard)' },
    { value: '7', label: '7% (Ermaessigt)' },
    { value: '0', label: '0%' },
  ],
  AT: [
    { value: '20', label: '20% (Standard)' },
    { value: '13', label: '13% (Ermaessigt)' },
    { value: '10', label: '10% (Ermaessigt)' },
    { value: '0', label: '0%' },
  ],
  CH: [
    { value: '8.1', label: '8.1% (Normal)' },
    { value: '3.8', label: '3.8% (Beherbergung)' },
    { value: '2.6', label: '2.6% (Reduziert)' },
    { value: '0', label: '0%' },
  ],
}

const COUNTRY_LABELS: Record<TaxCountry, string> = {
  DE: 'Deutschland',
  AT: 'Oesterreich',
  CH: 'Schweiz',
}

// ---------------------------------------------------------------------------
// Currency options (3.16)
// ---------------------------------------------------------------------------

type InvoiceCurrency = 'EUR' | 'CHF' | 'USD'

const CURRENCY_OPTIONS: { value: InvoiceCurrency; label: string; symbol: string }[] = [
  { value: 'EUR', label: 'Euro (EUR)', symbol: '\u20AC' },
  { value: 'CHF', label: 'Schweizer Franken (CHF)', symbol: 'CHF' },
  { value: 'USD', label: 'US-Dollar (USD)', symbol: '$' },
]

// Legacy alias for backward compatibility
const _TAX_RATES = TAX_RATES_BY_COUNTRY.DE

function emptyItem(position: number): LineItemDraft {
  return {
    key: String(Date.now()) + position,
    position,
    description: '',
    quantity: '1',
    unit_price: '0',
    tax_rate: '19',
  }
}

interface InvoiceFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  editInvoice?: Invoice | null
  sourceQuoteId?: string
}

export function InvoiceFormDialog({
  open,
  onOpenChange,
  editInvoice,
  sourceQuoteId,
}: InvoiceFormDialogProps) {
  const createInvoice = useCreateInvoice()
  const updateInvoice = useUpdateInvoice()
  const { data: settings } = useCompanySettings()

  const [customerName, setCustomerName] = useState('')
  const [customerAddress, setCustomerAddress] = useState('')
  const [customerEmail, setCustomerEmail] = useState('')
  const [customerUstIdNr, setCustomerUstIdNr] = useState('')
  const [taxMode, setTaxMode] = useState<TaxMode>('standard')
  const [taxCountry, setTaxCountry] = useState<TaxCountry>('DE')
  const [currency, setCurrency] = useState<InvoiceCurrency>('EUR')
  const [invoiceDate, setInvoiceDate] = useState('')
  const [deliveryDate, setDeliveryDate] = useState('')
  const [paymentTermsDays, setPaymentTermsDays] = useState('30')
  const [notes, setNotes] = useState('')
  const [items, setItems] = useState<LineItemDraft[]>([emptyItem(1)])

  const resetForm = useCallback(() => {
    setCustomerName('')
    setCustomerAddress('')
    setCustomerEmail('')
    setCustomerUstIdNr('')
    setTaxMode('standard')
    setTaxCountry('DE')
    setCurrency('EUR')
    setInvoiceDate(new Date().toISOString().split('T')[0])
    setDeliveryDate('')
    setPaymentTermsDays(
      String(settings?.default_payment_terms_days ?? 30),
    )
    setNotes('')
    setItems([emptyItem(1)])
  }, [settings])

  useEffect(() => {
    if (!open) return
    if (editInvoice) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setCustomerName(editInvoice.customer.name)
      setCustomerAddress(editInvoice.customer.address)
      setCustomerEmail(editInvoice.customer.email)
      setCustomerUstIdNr(editInvoice.customer.ust_id_nr ?? '')
      setTaxMode(editInvoice.tax_mode)
      setInvoiceDate(editInvoice.invoice_date)
      setDeliveryDate(editInvoice.delivery_date ?? '')
      setPaymentTermsDays(editInvoice.payment_terms ?? '30')
      setNotes(editInvoice.notes ?? '')
      setItems(
        editInvoice.line_items.map((li, idx) => ({
          key: li.id,
          position: li.position || idx + 1,
          description: li.description,
          quantity: li.quantity,
          unit_price: li.unit_price,
          tax_rate: li.tax_rate,
        })),
      )
    } else {
      resetForm()
    }
  }, [open, editInvoice, resetForm])

  const updateItem = (idx: number, updates: Partial<LineItemDraft>) => {
    setItems((prev) =>
      prev.map((item, i) => (i === idx ? { ...item, ...updates } : item)),
    )
  }

  const addItem = () => {
    setItems((prev) => [...prev, emptyItem(prev.length + 1)])
  }

  const removeItem = (idx: number) => {
    if (items.length <= 1) return
    setItems((prev) => prev.filter((_, i) => i !== idx))
  }

  const handleSave = () => {
    if (!customerName.trim()) return
    const validItems = items.filter(
      (i) => i.description.trim() && Number(i.unit_price) > 0,
    )
    if (validItems.length === 0) return

    const lineItems = validItems.map((i, idx) => ({
      position: idx + 1,
      description: i.description.trim(),
      quantity: i.quantity,
      unit_price: i.unit_price,
      tax_rate: taxMode === 'kleinunternehmer' ? '0' : i.tax_rate,
    }))

    const customer = {
      name: customerName.trim(),
      address: customerAddress.trim(),
      email: customerEmail.trim(),
      ...(customerUstIdNr.trim() ? { ust_id_nr: customerUstIdNr.trim() } : {}),
    }

    if (editInvoice) {
      updateInvoice.mutate(
        {
          id: editInvoice.id,
          customer,
          tax_mode: taxMode,
          line_items: lineItems,
          invoice_date: invoiceDate,
          delivery_date: deliveryDate || undefined,
          payment_terms_days: Number(paymentTermsDays),
          notes: notes.trim() || undefined,
        },
        {
          onSuccess: () => {
            toast.success('Rechnung aktualisiert')
            onOpenChange(false)
          },
          onError: (err) => toast.error(err.message),
        },
      )
    } else {
      createInvoice.mutate(
        {
          customer,
          tax_mode: taxMode,
          line_items: lineItems,
          invoice_date: invoiceDate,
          delivery_date: deliveryDate || undefined,
          payment_terms_days: Number(paymentTermsDays),
          notes: notes.trim() || undefined,
          source_quote_id: sourceQuoteId,
        },
        {
          onSuccess: () => {
            toast.success('Rechnung erstellt')
            onOpenChange(false)
          },
          onError: (err) => toast.error(err.message),
        },
      )
    }
  }

  const effectiveItems = items.map((i) => ({
    quantity: i.quantity,
    unit_price: i.unit_price,
    tax_rate: taxMode === 'kleinunternehmer' ? '0' : i.tax_rate,
  }))
  const subtotal = calcInvoiceSubtotal(effectiveItems)
  const tax = calcInvoiceTax(effectiveItems)
  const total = calcInvoiceTotal(effectiveItems)

  // Currency-aware formatter (3.16)
  const formatAmount = (value: number | string): string => {
    const n = typeof value === 'string' ? Number(value) : value
    if (isNaN(n)) return `${currency} 0,00`
    return new Intl.NumberFormat('de-DE', {
      style: 'currency',
      currency: currency,
    }).format(n)
  }

  const isPending = createInvoice.isPending || updateInvoice.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>
            {editInvoice ? 'Rechnung bearbeiten' : 'Neue Rechnung'}
          </DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 py-2">
          {/* Customer */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Kunde *</Label>
              <Input
                placeholder="Firma / Person"
                value={customerName}
                onChange={(e) => setCustomerName(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>E-Mail</Label>
              <Input
                type="email"
                placeholder="email@firma.de"
                value={customerEmail}
                onChange={(e) => setCustomerEmail(e.target.value)}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Adresse</Label>
              <Textarea
                placeholder="Strasse, PLZ Ort"
                value={customerAddress}
                onChange={(e) => setCustomerAddress(e.target.value)}
                rows={2}
              />
            </div>
            <div className="space-y-1.5">
              <Label>USt-IdNr. (optional)</Label>
              <Input
                placeholder="DE123456789"
                value={customerUstIdNr}
                onChange={(e) => setCustomerUstIdNr(e.target.value)}
              />
            </div>
          </div>

          {/* Tax country + currency (3.15 + 3.16) */}
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1.5">
              <Label>Mandanten-Land</Label>
              <Select
                value={taxCountry}
                onValueChange={(v) => {
                  const country = v as TaxCountry
                  setTaxCountry(country)
                  // Auto-set currency to match country
                  if (country === 'CH') setCurrency('CHF')
                  else setCurrency('EUR')
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {(Object.entries(COUNTRY_LABELS) as [TaxCountry, string][]).map(([k, l]) => (
                    <SelectItem key={k} value={k}>{l}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Waehrung</Label>
              <Select
                value={currency}
                onValueChange={(v) => setCurrency(v as InvoiceCurrency)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {CURRENCY_OPTIONS.map((c) => (
                    <SelectItem key={c.value} value={c.value}>{c.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Zahlungsziel (Tage)</Label>
              <Input
                type="number"
                min={1}
                value={paymentTermsDays}
                onChange={(e) => setPaymentTermsDays(e.target.value)}
              />
            </div>
          </div>

          {/* Tax mode */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Steuerbehandlung</Label>
              <Select
                value={taxMode}
                onValueChange={(v) => setTaxMode(v as TaxMode)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="standard">
                    Standard ({TAX_RATES_BY_COUNTRY[taxCountry][0].label})
                  </SelectItem>
                  <SelectItem value="reverse_charge">Reverse Charge</SelectItem>
                  <SelectItem value="kleinunternehmer">Kleinunternehmer</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Rechnungsdatum</Label>
              <Input
                type="date"
                value={invoiceDate}
                onChange={(e) => setInvoiceDate(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Leistungsdatum</Label>
              <Input
                type="date"
                value={deliveryDate}
                onChange={(e) => setDeliveryDate(e.target.value)}
              />
            </div>
          </div>

          {/* Tax mode info banners */}
          {taxMode === 'reverse_charge' && (
            <div className="flex items-start gap-2 rounded-lg border border-info/30 bg-info/5 p-3 text-xs text-info">
              <Info className="h-4 w-4 mt-0.5 shrink-0" />
              <span>
                Reverse Charge -- Steuerschuldnerschaft des Leistungsempfaengers
              </span>
            </div>
          )}
          {taxMode === 'kleinunternehmer' && (
            <div className="flex items-start gap-2 rounded-lg border border-warning/30 bg-warning/5 p-3 text-xs text-warning">
              <Info className="h-4 w-4 mt-0.5 shrink-0" />
              <span>
                Kein Ausweis von Umsatzsteuer gemaess Paragraph 19 UStG
              </span>
            </div>
          )}

          {/* Line Items */}
          <div className="space-y-2">
            <Label>Positionen</Label>
            <div className="rounded-lg border border-border overflow-hidden">
              <div className="grid grid-cols-[2rem_1fr_70px_90px_80px_90px_32px] gap-2 px-3 py-2 text-[10px] font-medium text-muted-foreground bg-secondary/30 uppercase tracking-wider">
                <span>Nr.</span>
                <span>Bezeichnung</span>
                <span>Menge</span>
                <span>Einzelpreis</span>
                <span>MwSt</span>
                <span className="text-right">Gesamt</span>
                <span />
              </div>
              {items.map((item, idx) => (
                <div
                  key={item.key}
                  className="grid grid-cols-[2rem_1fr_70px_90px_80px_90px_32px] gap-2 px-3 py-1.5 border-t border-border-muted items-center"
                >
                  <span className="text-xs text-muted-foreground">
                    {idx + 1}
                  </span>
                  <Input
                    placeholder="Beschreibung..."
                    value={item.description}
                    onChange={(e) =>
                      updateItem(idx, { description: e.target.value })
                    }
                    className="h-7 text-xs"
                  />
                  <Input
                    type="number"
                    min={0.01}
                    step={0.5}
                    value={item.quantity}
                    onChange={(e) =>
                      updateItem(idx, { quantity: e.target.value })
                    }
                    className="h-7 text-xs"
                  />
                  <Input
                    type="number"
                    min={0}
                    step={0.01}
                    value={item.unit_price}
                    onChange={(e) =>
                      updateItem(idx, { unit_price: e.target.value })
                    }
                    className="h-7 text-xs"
                  />
                  {taxMode !== 'kleinunternehmer' ? (
                    <select
                      value={item.tax_rate}
                      onChange={(e) =>
                        updateItem(idx, { tax_rate: e.target.value })
                      }
                      className="h-7 rounded border border-input-border bg-input-background px-1 text-[10px] text-foreground outline-none"
                    >
                      {TAX_RATES_BY_COUNTRY[taxCountry].map((r) => (
                        <option key={r.value} value={r.value}>
                          {r.label}
                        </option>
                      ))}
                    </select>
                  ) : (
                    <span className="text-xs text-muted-foreground px-1">
                      0%
                    </span>
                  )}
                  <span className="text-xs text-foreground text-right font-medium">
                    {formatAmount(calcLineTotal(item.quantity, item.unit_price))}
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
                Position hinzufügen
              </button>
            </div>
          </div>

          {/* Totals */}
          <div className="flex justify-end">
            <div className="w-64 space-y-1.5 text-xs">
              <div className="flex justify-between text-muted-foreground">
                <span>Zwischensumme (netto)</span>
                <span>{formatAmount(subtotal)}</span>
              </div>
              {taxMode !== 'kleinunternehmer' && (
                <div className="flex justify-between text-muted-foreground">
                  <span>MwSt</span>
                  <span>{formatAmount(tax)}</span>
                </div>
              )}
              <div className="flex justify-between font-medium text-sm text-foreground border-t border-border pt-1.5">
                <span>Gesamtbetrag</span>
                <span>{formatAmount(total)}</span>
              </div>
            </div>
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <Label>Notizen</Label>
            <Textarea
              placeholder="Zusaetzliche Informationen..."
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={2}
            />
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 pt-3 border-t border-border">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={handleSave}
            disabled={!customerName.trim() || isPending}
          >
            {isPending
              ? 'Speichert...'
              : editInvoice
                ? 'Speichern'
                : 'Erstellen'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
