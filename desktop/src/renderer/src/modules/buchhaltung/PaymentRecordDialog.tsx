import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toast } from 'sonner'
import { useFinanceStore, type Invoice, calcRemainingAmount } from '@/stores/finance'
import { formatCurrency } from '@/lib/format'

const PAYMENT_METHODS = [
  'Banküberweisung',
  'Kreditkarte',
  'Twint',
  'PayPal',
  'Bar',
  'Lastschrift',
]

interface PaymentRecordDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  invoice: Invoice | null
}

export function PaymentRecordDialog({ open, onOpenChange, invoice }: PaymentRecordDialogProps) {
  const { recordPayment } = useFinanceStore()

  const remaining = invoice ? calcRemainingAmount(invoice) : 0
  const [amount, setAmount] = useState(remaining)
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const [method, setMethod] = useState('Banküberweisung')
  const [reference, setReference] = useState('')

  // Reset when opening
  useEffect(() => {
    if (open && invoice) {
      setAmount(calcRemainingAmount(invoice))
    }
  }, [open, invoice])

  if (!invoice) return null

  const handleRecord = () => {
    if (amount <= 0) return
    recordPayment(invoice.id, {
      date,
      amount,
      method,
      reference: reference.trim() || undefined,
    })
    toast.success(`Zahlung von ${formatCurrency(amount)} erfasst`)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Zahlung erfassen</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="rounded-md bg-secondary/50 p-3 text-xs space-y-1">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Rechnung</span>
              <span className="font-medium text-foreground">{invoice.number}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Kunde</span>
              <span className="text-foreground">{invoice.client}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Offener Betrag</span>
              <span className="font-medium text-primary">{formatCurrency(remaining)}</span>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label>Betrag (CHF)</Label>
            <Input
              autoFocus
              type="number"
              min={0.01}
              step={0.01}
              value={amount}
              onChange={(e) => setAmount(Number(e.target.value))}
            />
          </div>

          <div className="space-y-1.5">
            <Label>Datum</Label>
            <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </div>

          <div className="space-y-1.5">
            <Label>Zahlungsart</Label>
            <Select value={method} onValueChange={setMethod}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {PAYMENT_METHODS.map((m) => <SelectItem key={m} value={m}>{m}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label>Referenz / Beleg-Nr. (optional)</Label>
            <Input placeholder="z.B. BELEG-043" value={reference} onChange={(e) => setReference(e.target.value)} />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Abbrechen</Button>
          <Button onClick={handleRecord} disabled={amount <= 0}>Zahlung erfassen</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
