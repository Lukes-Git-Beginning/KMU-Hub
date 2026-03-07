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
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toast } from 'sonner'
import { useRecordPayment, useInvoice, usePayments } from '@/api/hooks/useFinance'
import { formatEUR } from '@/stores/finance'
import type { PaymentMethod } from '@/types/finance-types'

const PAYMENT_METHODS: { value: PaymentMethod; label: string }[] = [
  { value: 'bank_transfer', label: 'Ueberweisung' },
  { value: 'cash', label: 'Barzahlung' },
  { value: 'credit_card', label: 'Kreditkarte' },
  { value: 'other', label: 'Sonstige' },
]

interface PaymentRecordDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  invoiceId: string | null
}

export function PaymentRecordDialog({
  open,
  onOpenChange,
  invoiceId,
}: PaymentRecordDialogProps) {
  const recordPayment = useRecordPayment()
  const { data: invoice } = useInvoice(invoiceId ?? '')
  const { data: paymentsData } = usePayments(invoiceId ?? '')

  const grossTotal = Number(invoice?.tax_breakdown.gross_total ?? 0)
  const totalPaid = (paymentsData?.payments ?? []).reduce(
    (sum, p) => sum + Number(p.amount),
    0,
  )
  const remaining = Math.max(0, grossTotal - totalPaid)

  const [amount, setAmount] = useState('')
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const [method, setMethod] = useState<PaymentMethod>('bank_transfer')
  const [reference, setReference] = useState('')
  const [notes, setNotes] = useState('')

  useEffect(() => {
    if (open && remaining > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setAmount(remaining.toFixed(2))
      setDate(new Date().toISOString().split('T')[0])
      setMethod('bank_transfer')
      setReference('')
      setNotes('')
    }
  }, [open, remaining])

  if (!invoiceId) return null

  const handleRecord = () => {
    const amountNum = Number(amount)
    if (amountNum <= 0) return

    recordPayment.mutate(
      {
        invoiceId,
        amount: amount,
        payment_date: date,
        method,
        reference: reference.trim() || undefined,
        notes: notes.trim() || undefined,
      },
      {
        onSuccess: () => {
          toast.success(`Zahlung von ${formatEUR(amountNum)} erfasst`)
          onOpenChange(false)
        },
        onError: (err) => toast.error(err.message),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Zahlung erfassen</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Invoice summary */}
          <div className="rounded-md bg-secondary/50 p-3 text-xs space-y-1">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Rechnung</span>
              <span className="font-medium text-foreground font-mono">
                {invoice?.invoice_number ?? '--'}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Kunde</span>
              <span className="text-foreground">
                {invoice?.customer.name ?? '--'}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Offener Betrag</span>
              <span className="font-medium text-primary">
                {formatEUR(remaining)}
              </span>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label>Betrag (EUR)</Label>
            <Input
              autoFocus
              type="number"
              min={0.01}
              step={0.01}
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <Label>Datum</Label>
            <Input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <Label>Zahlungsart</Label>
            <Select
              value={method}
              onValueChange={(v) => setMethod(v as PaymentMethod)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PAYMENT_METHODS.map((m) => (
                  <SelectItem key={m.value} value={m.value}>
                    {m.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label>Referenz / Beleg-Nr. (optional)</Label>
            <Input
              placeholder="z.B. BELEG-043"
              value={reference}
              onChange={(e) => setReference(e.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <Label>Notizen (optional)</Label>
            <Textarea
              placeholder="Zusaetzliche Informationen..."
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={2}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={handleRecord}
            disabled={Number(amount) <= 0 || recordPayment.isPending}
          >
            {recordPayment.isPending ? 'Speichert...' : 'Zahlung erfassen'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
