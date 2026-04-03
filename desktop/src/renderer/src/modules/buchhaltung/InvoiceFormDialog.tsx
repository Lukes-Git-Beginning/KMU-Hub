import { useState, useEffect } from 'react'
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
import { useFinanceStore, type Invoice, type LineItem, calcLineTotal, calcInvoiceSubtotal, calcInvoiceTax, calcInvoiceTotal } from '@/stores/finance'
import { formatCurrency } from '@/lib/format'

interface InvoiceFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  editInvoice?: Invoice | null
  defaultType?: 'invoice' | 'quote'
}

const VAT_RATES = [
  { value: '0', label: '0%' },
  { value: '7', label: '7% (Ermäßigt)' },
  { value: '19', label: '19% (Standard)' },
]

const PAYMENT_TERMS = [
  '10 Tage netto',
  '30 Tage netto',
  '60 Tage netto',
  '50% Anzahlung, 50% bei Abnahme',
  'Sofort fällig',
]

export function InvoiceFormDialog({ open, onOpenChange, editInvoice, defaultType = 'invoice' }: InvoiceFormDialogProps) {
  const { addInvoice, updateInvoice, nextInvoiceNum } = useFinanceStore()

  const [type, setType] = useState<'invoice' | 'quote'>(defaultType)
  const [client, setClient] = useState('')
  const [clientEmail, setClientEmail] = useState('')
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const [dueDate, setDueDate] = useState('')
  const [paymentTerms, setPaymentTerms] = useState('30 Tage netto')
  const [notes, setNotes] = useState('')
  const [items, setItems] = useState<LineItem[]>([
    { id: '1', description: '', quantity: 1, unitPrice: 0, vatRate: 19, discount: 0 },
  ])

  useEffect(() => {
    if (!open) return
    if (editInvoice) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setType(editInvoice.type)
      setClient(editInvoice.client)
      setClientEmail(editInvoice.clientEmail ?? '')
      setDate(editInvoice.date)
      setDueDate(editInvoice.dueDate)
      setPaymentTerms(editInvoice.paymentTerms)
      setNotes(editInvoice.notes ?? '')
      setItems(editInvoice.items.length > 0 ? editInvoice.items : [{ id: '1', description: '', quantity: 1, unitPrice: 0, vatRate: 19, discount: 0 }])
    } else {
      setType(defaultType)
      setClient('')
      setClientEmail('')
      setDate(new Date().toISOString().split('T')[0])
      const due = new Date()
      due.setDate(due.getDate() + 30)
      setDueDate(due.toISOString().split('T')[0])
      setPaymentTerms('30 Tage netto')
      setNotes('')
      setItems([{ id: '1', description: '', quantity: 1, unitPrice: 0, vatRate: 19, discount: 0 }])
    }
  }, [open, editInvoice, defaultType])

  const updateItem = (idx: number, updates: Partial<LineItem>) => {
    setItems((prev) => prev.map((item, i) => (i === idx ? { ...item, ...updates } : item)))
  }

  const addItem = () => {
    setItems((prev) => [...prev, { id: String(Date.now()), description: '', quantity: 1, unitPrice: 0, vatRate: 19, discount: 0 }])
  }

  const removeItem = (idx: number) => {
    if (items.length <= 1) return
    setItems((prev) => prev.filter((_, i) => i !== idx))
  }

  const handleSave = () => {
    if (!client.trim()) return
    const validItems = items.filter((i) => i.description.trim() && i.unitPrice > 0)
    if (validItems.length === 0) return

    const prefix = type === 'quote' ? 'QUO' : 'INV'
    const num = editInvoice ? editInvoice.number : `${prefix}-2026-${String(nextInvoiceNum + 1).padStart(3, '0')}`

    if (editInvoice) {
      updateInvoice(editInvoice.id, {
        type, client: client.trim(), clientEmail: clientEmail.trim() || undefined,
        date, dueDate, paymentTerms, notes: notes.trim() || undefined, items: validItems,
      })
      toast.success(`${type === 'quote' ? 'Angebot' : 'Rechnung'} aktualisiert`)
    } else {
      addInvoice({
        number: num, type, client: client.trim(), clientEmail: clientEmail.trim() || undefined,
        date, dueDate, status: 'draft', paymentTerms, notes: notes.trim() || undefined, items: validItems,
      })
      toast.success(`${type === 'quote' ? 'Angebot' : 'Rechnung'} ${num} erstellt`)
    }
    onOpenChange(false)
  }

  const subtotal = calcInvoiceSubtotal(items)
  const tax = calcInvoiceTax(items)
  const total = calcInvoiceTotal(items)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>
            {editInvoice ? `${type === 'quote' ? 'Angebot' : 'Rechnung'} bearbeiten` : `${type === 'quote' ? 'Neues Angebot' : 'Neue Rechnung'}`}
          </DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 py-2">
          {/* Type & Client */}
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1.5">
              <Label>Typ</Label>
              <Select value={type} onValueChange={(v) => setType(v as typeof type)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="invoice">Rechnung</SelectItem>
                  <SelectItem value="quote">Angebot</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Kunde *</Label>
              <Input placeholder="Firma / Person" value={client} onChange={(e) => setClient(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>E-Mail</Label>
              <Input type="email" placeholder="email@firma.de" value={clientEmail} onChange={(e) => setClientEmail(e.target.value)} />
            </div>
          </div>

          {/* Dates & Terms */}
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1.5">
              <Label>Datum</Label>
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>Fällig am</Label>
              <Input type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>Zahlungsbedingungen</Label>
              <Select value={paymentTerms} onValueChange={setPaymentTerms}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {PAYMENT_TERMS.map((t) => <SelectItem key={t} value={t}>{t}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Line Items */}
          <div className="space-y-2">
            <Label>Positionen</Label>
            <div className="rounded-lg border border-border overflow-hidden">
              {/* Header */}
              <div className="grid grid-cols-[1fr_60px_80px_70px_50px_80px_32px] gap-2 px-3 py-2 text-[10px] font-medium text-muted-foreground bg-secondary/30 uppercase tracking-wider">
                <span>Beschreibung</span>
                <span>Menge</span>
                <span>Preis</span>
                <span>MwSt</span>
                <span>Rabatt</span>
                <span className="text-right">Total</span>
                <span />
              </div>
              {/* Items */}
              {items.map((item, idx) => (
                <div key={item.id} className="grid grid-cols-[1fr_60px_80px_70px_50px_80px_32px] gap-2 px-3 py-1.5 border-t border-border-muted items-center">
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
                    onChange={(e) => updateItem(idx, { quantity: Number(e.target.value) })}
                    className="h-7 text-xs"
                  />
                  <Input
                    type="number"
                    min={0}
                    step={0.01}
                    value={item.unitPrice}
                    onChange={(e) => updateItem(idx, { unitPrice: Number(e.target.value) })}
                    className="h-7 text-xs"
                  />
                  <select
                    value={item.vatRate}
                    onChange={(e) => updateItem(idx, { vatRate: Number(e.target.value) })}
                    className="h-7 rounded border border-input-border bg-input-background px-1 text-[10px] text-foreground outline-none"
                  >
                    {VAT_RATES.map((r) => <option key={r.value} value={r.value}>{r.label}</option>)}
                  </select>
                  <Input
                    type="number"
                    min={0}
                    max={100}
                    value={item.discount}
                    onChange={(e) => updateItem(idx, { discount: Number(e.target.value) })}
                    className="h-7 text-xs"
                  />
                  <span className="text-xs text-foreground text-right font-medium">
                    {formatCurrency(calcLineTotal(item))}
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
              {/* Add item */}
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
            <div className="w-60 space-y-1.5 text-xs">
              <div className="flex justify-between text-muted-foreground">
                <span>Zwischensumme</span>
                <span>{formatCurrency(subtotal)}</span>
              </div>
              <div className="flex justify-between text-muted-foreground">
                <span>MwSt</span>
                <span>{formatCurrency(tax)}</span>
              </div>
              <div className="flex justify-between font-medium text-sm text-foreground border-t border-border pt-1.5">
                <span>Gesamt</span>
                <span>{formatCurrency(total)}</span>
              </div>
            </div>
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <Label>Notizen</Label>
            <Textarea placeholder="Zusätzliche Informationen..." value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} />
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 pt-3 border-t border-border">
          <Button variant="outline" onClick={() => onOpenChange(false)}>Abbrechen</Button>
          <Button onClick={handleSave} disabled={!client.trim()}>
            {editInvoice ? 'Speichern' : 'Erstellen'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
