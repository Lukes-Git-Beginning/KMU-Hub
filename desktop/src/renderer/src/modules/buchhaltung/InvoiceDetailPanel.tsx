import { FileText, Calendar, CreditCard, Clock, CheckCircle2, AlertCircle, XCircle, User } from 'lucide-react'
import { DetailPanel } from '@/components/shared'
import { type Invoice, calcLineTotal, calcInvoiceSubtotal, calcInvoiceTax, calcInvoiceTotal, calcPaidAmount, calcRemainingAmount } from '@/stores/finance'

function formatCHF(n: number): string {
  return new Intl.NumberFormat('de-CH', { style: 'currency', currency: 'CHF' }).format(n)
}

const statusConfig: Record<string, { label: string; color: string; bg: string; icon: typeof CheckCircle2 }> = {
  draft: { label: 'Entwurf', color: 'text-muted-foreground', bg: 'bg-secondary', icon: FileText },
  sent: { label: 'Gesendet', color: 'text-info', bg: 'bg-info-light', icon: Clock },
  paid: { label: 'Bezahlt', color: 'text-success', bg: 'bg-success-light', icon: CheckCircle2 },
  overdue: { label: 'Überfällig', color: 'text-error', bg: 'bg-error-light', icon: AlertCircle },
  cancelled: { label: 'Storniert', color: 'text-muted-foreground', bg: 'bg-secondary', icon: XCircle },
}

interface InvoiceDetailPanelProps {
  invoice: Invoice
  onClose: () => void
  onEdit: () => void
  onRecordPayment: () => void
}

export function InvoiceDetailPanel({ invoice, onClose, onEdit, onRecordPayment }: InvoiceDetailPanelProps) {
  const status = statusConfig[invoice.status]
  const StatusIcon = status.icon
  const subtotal = calcInvoiceSubtotal(invoice.items)
  const tax = calcInvoiceTax(invoice.items)
  const total = calcInvoiceTotal(invoice.items)
  const paid = calcPaidAmount(invoice)
  const remaining = calcRemainingAmount(invoice)
  const isQuote = invoice.type === 'quote'

  return (
    <DetailPanel open={true} title={isQuote ? 'Angebots-Details' : 'Rechnungs-Details'} onClose={onClose}>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div>
            <h3 className="text-base font-medium text-foreground font-mono">{invoice.number}</h3>
            <p className="text-xs text-muted-foreground">{invoice.client}</p>
          </div>
          <span className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-[10px] font-medium ${status.bg} ${status.color}`}>
            <StatusIcon className="h-3 w-3" />
            {status.label}
          </span>
        </div>

        {/* Key info */}
        <div className="grid grid-cols-2 gap-3 text-xs">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Calendar className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">Datum</p>
              <p className="text-foreground">{new Date(invoice.date).toLocaleDateString('de-CH')}</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">Fällig</p>
              <p className="text-foreground">{new Date(invoice.dueDate).toLocaleDateString('de-CH')}</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <User className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">Kunde</p>
              <p className="text-foreground">{invoice.client}</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <CreditCard className="h-3.5 w-3.5" />
            <div>
              <p className="text-[10px] text-muted-foreground">Bedingungen</p>
              <p className="text-foreground">{invoice.paymentTerms}</p>
            </div>
          </div>
        </div>

        {/* Line items */}
        <section>
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">Positionen</h4>
          <div className="rounded-md border border-border overflow-hidden">
            {invoice.items.map((item, idx) => (
              <div key={item.id} className={`flex items-center justify-between px-3 py-2 text-xs ${idx > 0 ? 'border-t border-border-muted' : ''}`}>
                <div className="min-w-0 flex-1">
                  <p className="text-foreground truncate">{item.description}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {item.quantity} × {formatCHF(item.unitPrice)}
                    {item.discount > 0 && ` · ${item.discount}% Rabatt`}
                    {item.vatRate > 0 && ` · ${item.vatRate}% MwSt`}
                  </p>
                </div>
                <span className="text-foreground font-medium ml-3">{formatCHF(calcLineTotal(item))}</span>
              </div>
            ))}
          </div>
        </section>

        {/* Totals */}
        <div className="space-y-1.5 text-xs">
          <div className="flex justify-between text-muted-foreground">
            <span>Zwischensumme</span>
            <span>{formatCHF(subtotal)}</span>
          </div>
          <div className="flex justify-between text-muted-foreground">
            <span>MwSt</span>
            <span>{formatCHF(tax)}</span>
          </div>
          <div className="flex justify-between font-medium text-sm text-foreground border-t border-border pt-1.5">
            <span>Gesamt</span>
            <span>{formatCHF(total)}</span>
          </div>
          {!isQuote && paid > 0 && (
            <>
              <div className="flex justify-between text-success">
                <span>Bezahlt</span>
                <span>{formatCHF(paid)}</span>
              </div>
              {remaining > 0 && (
                <div className="flex justify-between font-medium text-warning">
                  <span>Offen</span>
                  <span>{formatCHF(remaining)}</span>
                </div>
              )}
            </>
          )}
        </div>

        {/* Payment history */}
        {invoice.payments.length > 0 && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">Zahlungshistorie</h4>
            <div className="space-y-1.5">
              {invoice.payments.map((p, idx) => (
                <div key={idx} className="flex items-center justify-between rounded-md bg-success/5 px-3 py-2 text-xs">
                  <div>
                    <p className="text-foreground">{p.method}</p>
                    <p className="text-[10px] text-muted-foreground">
                      {new Date(p.date).toLocaleDateString('de-CH')}
                      {p.reference && ` · ${p.reference}`}
                    </p>
                  </div>
                  <span className="text-success font-medium">{formatCHF(p.amount)}</span>
                </div>
              ))}
            </div>
          </section>
        )}

        {/* Notes */}
        {invoice.notes && (
          <section>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1">Notizen</h4>
            <p className="text-xs text-muted-foreground">{invoice.notes}</p>
          </section>
        )}

        {/* Actions */}
        <div className="flex gap-2">
          {(invoice.status === 'draft') && (
            <button
              onClick={onEdit}
              className="flex-1 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              Bearbeiten
            </button>
          )}
          {!isQuote && invoice.status !== 'paid' && invoice.status !== 'cancelled' && (
            <button
              onClick={onRecordPayment}
              className="flex-1 rounded-lg bg-primary py-2 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              Zahlung erfassen
            </button>
          )}
        </div>
      </div>
    </DetailPanel>
  )
}
