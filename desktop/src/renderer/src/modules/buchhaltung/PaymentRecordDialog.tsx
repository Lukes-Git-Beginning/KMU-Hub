import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
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

const PAYMENT_METHOD_KEYS = [
  'buchhaltung.paymentMethod.bankTransfer',
  'buchhaltung.paymentMethod.creditCard',
  'buchhaltung.paymentMethod.twint',
  'buchhaltung.paymentMethod.paypal',
  'buchhaltung.paymentMethod.cash',
  'buchhaltung.paymentMethod.directDebit',
]

interface PaymentRecordDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  invoice: Invoice | null
}

export function PaymentRecordDialog({ open, onOpenChange, invoice }: PaymentRecordDialogProps) {
  const { t } = useTranslation()
  const { recordPayment } = useFinanceStore()

  const remaining = invoice ? calcRemainingAmount(invoice) : 0
  const [amount, setAmount] = useState(remaining)
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const [method, setMethod] = useState(PAYMENT_METHOD_KEYS[0])
  const [reference, setReference] = useState('')

  // Reset when opening
  useEffect(() => {
    if (open && invoice) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
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
    toast.success(t('buchhaltung.toast.paymentRecorded', { amount: formatCurrency(amount) }))
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('buchhaltung.actions.recordPayment')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="rounded-md bg-secondary/50 p-3 text-xs space-y-1">
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t('buchhaltung.table.invoice')}</span>
              <span className="font-medium text-foreground">{invoice.number}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t('buchhaltung.table.client')}</span>
              <span className="text-foreground">{invoice.client}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t('buchhaltung.payment.openAmount')}</span>
              <span className="font-medium text-primary">{formatCurrency(remaining)}</span>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label>{t('buchhaltung.form.amount')}</Label>
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
            <Label>{t('buchhaltung.form.date')}</Label>
            <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </div>

          <div className="space-y-1.5">
            <Label>{t('buchhaltung.payment.method')}</Label>
            <Select value={method} onValueChange={setMethod}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {PAYMENT_METHOD_KEYS.map((key) => <SelectItem key={key} value={key}>{t(key)}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label>{t('buchhaltung.payment.reference')}</Label>
            <Input placeholder={t('buchhaltung.payment.referencePlaceholder')} value={reference} onChange={(e) => setReference(e.target.value)} />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t('common.cancel')}</Button>
          <Button onClick={handleRecord} disabled={amount <= 0}>{t('buchhaltung.actions.recordPayment')}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
