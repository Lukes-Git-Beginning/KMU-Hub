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

const PAYMENT_METHODS: { value: PaymentMethod; labelKey: string }[] = [
  { value: 'bank_transfer', labelKey: 'finanzen.paymentMethod.bankTransfer' },
  { value: 'cash', labelKey: 'finanzen.paymentMethod.cash' },
  { value: 'credit_card', labelKey: 'finanzen.paymentMethod.creditCard' },
  { value: 'other', labelKey: 'finanzen.paymentMethod.other' },
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
  const { t } = useTranslation()
  const recordPayment = useRecordPayment()
  const { data: invoice } = useInvoice(invoiceId ?? '')
  const { data: paymentsData } = usePayments(invoiceId ?? '')

  const grossTotal = Number(invoice?.tax_breakdown?.gross_total ?? invoice?.total_gross ?? 0)
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
          toast.success(t('finanzen.payment.recorded', { amount: formatEUR(amountNum) }))
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
          <DialogTitle>{t('finanzen.payment.title')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Invoice summary */}
          <div className="rounded-md bg-secondary/50 p-3 text-xs space-y-1">
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t('finanzen.invoice')}</span>
              <span className="font-medium text-foreground font-mono">
                {invoice?.invoice_number ?? '--'}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t('finanzen.customer')}</span>
              <span className="text-foreground">
                {invoice?.customer.name ?? '--'}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t('finanzen.payment.openAmount')}</span>
              <span className="font-medium text-primary">
                {formatEUR(remaining)}
              </span>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label>{t('finanzen.payment.amountEUR')}</Label>
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
            <Label>{t('finanzen.banking.date')}</Label>
            <Input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <Label>{t('finanzen.payment.paymentMethod')}</Label>
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
                    {t(m.labelKey)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label>{t('finanzen.payment.reference')}</Label>
            <Input
              placeholder="z.B. BELEG-043"
              value={reference}
              onChange={(e) => setReference(e.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <Label>{t('finanzen.payment.notesOptional')}</Label>
            <Textarea
              placeholder={t('finanzen.invoiceForm.notesPlaceholder')}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={2}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            onClick={handleRecord}
            disabled={Number(amount) <= 0 || recordPayment.isPending}
          >
            {recordPayment.isPending ? t('finanzen.saving') : t('finanzen.payment.title')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
