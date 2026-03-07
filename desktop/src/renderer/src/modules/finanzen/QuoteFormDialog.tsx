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
import { useCreateQuote, useUpdateQuote, useCompanySettings } from '@/api/hooks/useFinance'
import { formatEUR, calcLineTotal, calcInvoiceSubtotal, calcInvoiceTax, calcInvoiceTotal } from '@/stores/finance'
import type { Quote, TaxMode } from '@/types/finance-types'

interface LineItemDraft {
  key: string
  position: number
  description: string
  quantity: string
  unit_price: string
  tax_rate: string
}

const TAX_RATES = [
  { value: '19', label: '19% (Standard)' },
  { value: '7', label: '7% (Ermaessigt)' },
  { value: '0', label: '0%' },
]

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

interface QuoteFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  editQuote?: Quote | null
  dealId?: string
}

export function QuoteFormDialog({
  open,
  onOpenChange,
  editQuote,
  dealId,
}: QuoteFormDialogProps) {
  const createQuote = useCreateQuote()
  const updateQuote = useUpdateQuote()
  const { data: settings } = useCompanySettings()

  const [customerName, setCustomerName] = useState('')
  const [customerAddress, setCustomerAddress] = useState('')
  const [customerEmail, setCustomerEmail] = useState('')
  const [customerUstIdNr, setCustomerUstIdNr] = useState('')
  const [taxMode, setTaxMode] = useState<TaxMode>('standard')
  const [validUntil, setValidUntil] = useState('')
  const [notes, setNotes] = useState('')
  const [items, setItems] = useState<LineItemDraft[]>([emptyItem(1)])

  const resetForm = useCallback(() => {
    setCustomerName('')
    setCustomerAddress('')
    setCustomerEmail('')
    setCustomerUstIdNr('')
    setTaxMode('standard')
    setNotes('')
    setItems([emptyItem(1)])
    const defaultDays = settings?.default_quote_validity_days ?? 30
    const d = new Date()
    d.setDate(d.getDate() + defaultDays)
    setValidUntil(d.toISOString().split('T')[0])
  }, [settings])

  useEffect(() => {
    if (!open) return
    if (editQuote) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setCustomerName(editQuote.customer.name)
      setCustomerAddress(editQuote.customer.address)
      setCustomerEmail(editQuote.customer.email)
      setCustomerUstIdNr(editQuote.customer.ust_id_nr ?? '')
      setTaxMode(editQuote.tax_mode)
      setValidUntil(editQuote.valid_until)
      setNotes(editQuote.notes ?? '')
      setItems(
        editQuote.line_items.map((li, idx) => ({
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
  }, [open, editQuote, resetForm])

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

    if (editQuote) {
      updateQuote.mutate(
        {
          id: editQuote.id,
          customer,
          tax_mode: taxMode,
          line_items: lineItems,
          valid_until: validUntil,
          notes: notes.trim() || undefined,
        },
        {
          onSuccess: () => {
            toast.success('Angebot aktualisiert')
            onOpenChange(false)
          },
          onError: (err) => toast.error(err.message),
        },
      )
    } else {
      createQuote.mutate(
        {
          customer,
          tax_mode: taxMode,
          line_items: lineItems,
          valid_until: validUntil || undefined,
          notes: notes.trim() || undefined,
          deal_id: dealId,
        },
        {
          onSuccess: () => {
            toast.success('Angebot erstellt')
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

  const isPending = createQuote.isPending || updateQuote.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>
            {editQuote ? 'Angebot bearbeiten' : 'Neues Angebot'}
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

          {/* Tax mode & validity */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Steuerbehandlung</Label>
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
              <Label>Gueltig bis</Label>
              <Input
                type="date"
                value={validUntil}
                onChange={(e) => setValidUntil(e.target.value)}
              />
            </div>
          </div>

          {/* Tax mode info banners */}
          {taxMode === 'reverse_charge' && (
            <div className="flex items-start gap-2 rounded-lg border border-info/30 bg-info/5 p-3 text-xs text-info">
              <Info className="h-4 w-4 mt-0.5 shrink-0" />
              <span>Reverse Charge -- Steuerschuldnerschaft des Leistungsempfaengers</span>
            </div>
          )}
          {taxMode === 'kleinunternehmer' && (
            <div className="flex items-start gap-2 rounded-lg border border-warning/30 bg-warning/5 p-3 text-xs text-warning">
              <Info className="h-4 w-4 mt-0.5 shrink-0" />
              <span>Kein Ausweis von Umsatzsteuer gemaess Paragraph 19 UStG</span>
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
                  <span className="text-xs text-muted-foreground">{idx + 1}</span>
                  <Input
                    placeholder="Beschreibung..."
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
                    <select
                      value={item.tax_rate}
                      onChange={(e) => updateItem(idx, { tax_rate: e.target.value })}
                      className="h-7 rounded border border-input-border bg-input-background px-1 text-[10px] text-foreground outline-none"
                    >
                      {TAX_RATES.map((r) => (
                        <option key={r.value} value={r.value}>{r.label}</option>
                      ))}
                    </select>
                  ) : (
                    <span className="text-xs text-muted-foreground px-1">0%</span>
                  )}
                  <span className="text-xs text-foreground text-right font-medium">
                    {formatEUR(calcLineTotal(item.quantity, item.unit_price))}
                  </span>
                  <button
                    onClick={() => removeItem(idx)}
                    disabled={items.length <= 1}
                    className="rounded p-0.5 text-muted-foreground hover:text-red-500 disabled:opacity-30"
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
                Position hinzufuegen
              </button>
            </div>
          </div>

          {/* Totals */}
          <div className="flex justify-end">
            <div className="w-64 space-y-1.5 text-xs">
              <div className="flex justify-between text-muted-foreground">
                <span>Zwischensumme (netto)</span>
                <span>{formatEUR(subtotal)}</span>
              </div>
              {taxMode !== 'kleinunternehmer' && (
                <div className="flex justify-between text-muted-foreground">
                  <span>MwSt</span>
                  <span>{formatEUR(tax)}</span>
                </div>
              )}
              <div className="flex justify-between font-medium text-sm text-foreground border-t border-border pt-1.5">
                <span>Gesamtbetrag</span>
                <span>{formatEUR(total)}</span>
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
            {isPending ? 'Speichert...' : editQuote ? 'Speichern' : 'Erstellen'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
