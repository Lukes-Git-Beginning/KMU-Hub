import {
  FileText,
  Calendar,
  CreditCard,
  Clock,
  CheckCircle2,
  AlertCircle,
  XCircle,
  User,
  Download,
  Send,
  Ban,
  Info,
  History,
  Shield,
} from 'lucide-react'
import { toast } from 'sonner'
import { DetailPanel } from '@/components/shared'
import { Button } from '@/components/ui/button'
import {
  useInvoice,
  usePayments,
  useSendInvoice,
  useMarkInvoicePaid,
  useCancelInvoice,
  useDownloadInvoicePDF,
} from '@/api/hooks/useFinance'
import { formatEUR } from '@/stores/finance'
import type { InvoiceStatus } from '@/types/finance-types'
import { PDFPreviewPanel } from './PDFPreviewPanel'

const statusConfig: Record<
  InvoiceStatus,
  {
    label: string
    color: string
    bg: string
    icon: typeof CheckCircle2
  }
> = {
  draft: {
    label: 'Entwurf',
    color: 'text-muted-foreground',
    bg: 'bg-secondary',
    icon: FileText,
  },
  sent: {
    label: 'Gesendet',
    color: 'text-info',
    bg: 'bg-info-light',
    icon: Clock,
  },
  paid: {
    label: 'Bezahlt',
    color: 'text-success',
    bg: 'bg-success-light',
    icon: CheckCircle2,
  },
  overdue: {
    label: 'Ueberfaellig',
    color: 'text-error',
    bg: 'bg-error-light',
    icon: AlertCircle,
  },
  cancelled: {
    label: 'Storniert',
    color: 'text-muted-foreground',
    bg: 'bg-secondary',
    icon: XCircle,
  },
}

const PAYMENT_METHOD_LABELS: Record<string, string> = {
  bank_transfer: 'Ueberweisung',
  cash: 'Barzahlung',
  credit_card: 'Kreditkarte',
  other: 'Sonstige',
}

interface InvoiceDetailPanelProps {
  invoiceId: string
  onClose: () => void
  onEdit: () => void
  onRecordPayment: () => void
}

export function InvoiceDetailPanel({
  invoiceId,
  onClose,
  onEdit,
  onRecordPayment,
}: InvoiceDetailPanelProps) {
  const { data: invoice, isLoading } = useInvoice(invoiceId)
  const { data: paymentsData } = usePayments(invoiceId)
  const sendInvoice = useSendInvoice()
  const markPaid = useMarkInvoicePaid()
  const cancelInvoice = useCancelInvoice()
  const downloadPDF = useDownloadInvoicePDF()

  if (isLoading || !invoice) {
    return (
      <DetailPanel open={true} title="Rechnungs-Details" onClose={onClose}>
        <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
          Lade...
        </div>
      </DetailPanel>
    )
  }

  const status = statusConfig[invoice.status]
  const StatusIcon = status.icon
  const grossTotal = Number(invoice.tax_breakdown.gross_total)
  const payments = paymentsData?.payments ?? []
  const totalPaid = payments.reduce((sum, p) => sum + Number(p.amount), 0)
  const remaining = Math.max(0, grossTotal - totalPaid)
  const isImmutable = invoice.status !== 'draft'

  const handleSend = () => {
    sendInvoice.mutate(invoiceId, {
      onSuccess: () => toast.success('Rechnung gesendet'),
      onError: (err) => toast.error(err.message),
    })
  }

  const handleMarkPaid = () => {
    markPaid.mutate(invoiceId, {
      onSuccess: () => toast.success('Als bezahlt markiert'),
      onError: (err) => toast.error(err.message),
    })
  }

  const handleCancel = () => {
    cancelInvoice.mutate(invoiceId, {
      onSuccess: () => toast.success('Rechnung storniert'),
      onError: (err) => toast.error(err.message),
    })
  }

  return (
    <DetailPanel open={true} title="Rechnungs-Details" onClose={onClose}>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div>
            <h3 className="text-base font-medium text-foreground font-mono">
              {invoice.invoice_number}
            </h3>
            <p className="text-xs text-muted-foreground">
              {invoice.customer.name}
            </p>
          </div>
          <span
            className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-[10px] font-medium ${status.bg} ${status.color}`}
          >
            <StatusIcon className="h-3 w-3" />
            {status.label}
          </span>
        </div>

        {/* Immutability notice */}
        {isImmutable && (
          <div className="flex items-start gap-2 rounded-lg border border-info/30 bg-info/5 p-3 text-xs text-info">
            <Info className="h-4 w-4 mt-0.5 shrink-0" />
            <span>
              Rechnung wurde versendet und kann nicht mehr bearbeitet werden.
            </span>
          </div>
        )}

        {/* Key info */}
        <div className="grid grid-cols-2 gap-3 text-xs">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Calendar className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">
                Rechnungsdatum
              </p>
              <p className="text-foreground">
                {new Date(invoice.invoice_date).toLocaleDateString('de-DE')}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">Faellig</p>
              <p className="text-foreground">
                {new Date(invoice.due_date).toLocaleDateString('de-DE')}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <User className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">Kunde</p>
              <p className="text-foreground">{invoice.customer.name}</p>
              {invoice.customer.address && (
                <p className="text-[10px] text-muted-foreground">
                  {invoice.customer.address}
                </p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <CreditCard className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">
                Zahlungsziel
              </p>
              <p className="text-foreground">
                {invoice.payment_terms ?? '--'} Tage
              </p>
            </div>
          </div>
        </div>

        {/* Line items */}
        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
            Positionen
          </h4>
          <div className="rounded-md border border-border overflow-hidden">
            {invoice.line_items.map((item, idx) => (
              <div
                key={item.id || idx}
                className={`flex items-center justify-between px-3 py-2 text-xs ${
                  idx > 0 ? 'border-t border-border-muted' : ''
                }`}
              >
                <div className="min-w-0 flex-1">
                  <p className="text-foreground truncate">
                    {item.description}
                  </p>
                  <p className="text-[10px] text-muted-foreground">
                    {item.quantity} x {formatEUR(item.unit_price)}
                    {Number(item.tax_rate) > 0 && ` | ${item.tax_rate}% MwSt`}
                  </p>
                </div>
                <span className="text-foreground font-medium ml-3">
                  {formatEUR(item.line_total)}
                </span>
              </div>
            ))}
          </div>
        </section>

        {/* Tax breakdown */}
        <div className="space-y-1.5 text-xs">
          <div className="flex justify-between text-muted-foreground">
            <span>Zwischensumme (netto)</span>
            <span>{formatEUR(invoice.tax_breakdown.subtotal)}</span>
          </div>
          {Object.entries(invoice.tax_breakdown.tax_by_rate ?? {}).map(
            ([rate, amount]) => (
              <div
                key={rate}
                className="flex justify-between text-muted-foreground"
              >
                <span>MwSt {rate}%</span>
                <span>{formatEUR(amount)}</span>
              </div>
            ),
          )}
          <div className="flex justify-between font-medium text-sm text-foreground border-t border-border pt-1.5">
            <span>Gesamtbetrag</span>
            <span>{formatEUR(invoice.tax_breakdown.gross_total)}</span>
          </div>
          {totalPaid > 0 && (
            <>
              <div className="flex justify-between text-success">
                <span>Bezahlt</span>
                <span>{formatEUR(totalPaid)}</span>
              </div>
              {remaining > 0 && (
                <div className="flex justify-between font-medium text-warning">
                  <span>Offen</span>
                  <span>{formatEUR(remaining)}</span>
                </div>
              )}
            </>
          )}
        </div>

        {/* Payment history */}
        {payments.length > 0 && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              Zahlungshistorie
            </h4>
            <div className="space-y-1.5">
              {payments.map((p) => (
                <div
                  key={p.id}
                  className="flex items-center justify-between rounded-md bg-success/5 px-3 py-2 text-xs"
                >
                  <div>
                    <p className="text-foreground">
                      {PAYMENT_METHOD_LABELS[p.method] ?? p.method}
                    </p>
                    <p className="text-[10px] text-muted-foreground">
                      {new Date(p.payment_date).toLocaleDateString('de-DE')}
                      {p.reference && ` | ${p.reference}`}
                    </p>
                  </div>
                  <span className="text-success font-medium">
                    {formatEUR(p.amount)}
                  </span>
                </div>
              ))}
            </div>
          </section>
        )}

        {/* Notes */}
        {invoice.notes && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1">
              Notizen
            </h4>
            <p className="text-xs text-muted-foreground">{invoice.notes}</p>
          </section>
        )}

        {/* PDF Preview (3.18) */}
        <PDFPreviewPanel
          invoiceNumber={invoice.invoice_number}
          customerName={invoice.customer.name}
          date={new Date(invoice.invoice_date).toLocaleDateString('de-DE')}
          amount={formatEUR(invoice.tax_breakdown.gross_total)}
          onDownload={() => downloadPDF.mutate(invoiceId)}
        />

        {/* GoBD Audit Log (3.17) */}
        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1">
            <Shield className="h-3 w-3" />
            GoBD-Aenderungsprotokoll
          </h4>
          <div className="rounded-md border border-border overflow-hidden">
            {[
              {
                action: 'Rechnung erstellt',
                user: 'Max Mueller',
                date: invoice.invoice_date,
                detail: `Nummer: ${invoice.invoice_number}`,
              },
              ...(invoice.status !== 'draft'
                ? [
                    {
                      action: 'Rechnung gesendet',
                      user: 'Max Mueller',
                      date: invoice.invoice_date,
                      detail: `An: ${invoice.customer.email}`,
                    },
                  ]
                : []),
              ...(invoice.status === 'cancelled'
                ? [
                    {
                      action: 'Rechnung storniert',
                      user: 'Max Mueller',
                      date: new Date().toISOString().split('T')[0],
                      detail: 'Storno (keine Loeschung)',
                    },
                  ]
                : []),
              ...(payments.length > 0
                ? payments.map((p) => ({
                    action: 'Zahlung erfasst',
                    user: 'System',
                    date: p.payment_date,
                    detail: `${formatEUR(p.amount)} via ${PAYMENT_METHOD_LABELS[p.method] ?? p.method}`,
                  }))
                : []),
            ].map((entry, idx) => (
              <div
                key={idx}
                className={`flex items-start gap-2 px-3 py-2 text-xs ${
                  idx > 0 ? 'border-t border-border-muted' : ''
                }`}
              >
                <History className="h-3 w-3 text-muted-foreground mt-0.5 shrink-0" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <span className="font-medium text-foreground">{entry.action}</span>
                    <span className="text-[10px] text-muted-foreground">
                      {new Date(entry.date).toLocaleDateString('de-DE')}
                    </span>
                  </div>
                  <p className="text-[10px] text-muted-foreground">
                    {entry.user} — {entry.detail}
                  </p>
                </div>
              </div>
            ))}
          </div>
          <p className="mt-1 text-[9px] text-muted-foreground flex items-center gap-1">
            <Shield className="h-2.5 w-2.5" />
            GoBD-konform: Unveraenderbar nach Versand. Storno statt Loeschung.
          </p>
        </section>

        {/* Actions */}
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => downloadPDF.mutate(invoiceId)}
            disabled={downloadPDF.isPending}
          >
            <Download className="mr-1.5 h-3.5 w-3.5" />
            PDF
          </Button>
          {invoice.status === 'draft' && (
            <>
              <Button variant="outline" size="sm" onClick={onEdit}>
                Bearbeiten
              </Button>
              <Button
                size="sm"
                onClick={handleSend}
                disabled={sendInvoice.isPending}
              >
                <Send className="mr-1.5 h-3.5 w-3.5" />
                Senden
              </Button>
            </>
          )}
          {invoice.status !== 'paid' && invoice.status !== 'cancelled' && (
            <>
              <Button size="sm" onClick={onRecordPayment}>
                <CreditCard className="mr-1.5 h-3.5 w-3.5" />
                Zahlung erfassen
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleMarkPaid}
                disabled={markPaid.isPending}
              >
                <CheckCircle2 className="mr-1.5 h-3.5 w-3.5" />
                Als bezahlt
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleCancel}
                disabled={cancelInvoice.isPending}
                className="text-error hover:text-error"
              >
                <Ban className="mr-1.5 h-3.5 w-3.5" />
                Stornieren
              </Button>
            </>
          )}
        </div>
      </div>
    </DetailPanel>
  )
}
