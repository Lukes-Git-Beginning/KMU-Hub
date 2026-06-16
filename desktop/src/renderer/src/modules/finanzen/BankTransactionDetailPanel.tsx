/**
 * BankTransactionDetailPanel — Detailansicht einer Banktransaktion (P2.5e-Fix).
 *
 * Zeigt alle Felder (Datum, Art, Betrag, Gegenkonto, Status, zugeordnete
 * Rechnung) und bietet kontextabhängige Aktionen: vorgeschlagene Zuordnung
 * bestätigen/ablehnen oder bei offenen Eingängen manuell eine offene Rechnung
 * zuordnen. Nutzt das shared `DetailModal`.
 */
import { useTranslation } from 'react-i18next'
import { ArrowDownRight, ArrowUpRight, CheckCircle2, HelpCircle, FileText, ChevronRight } from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { formatDate } from '@/lib/format'
import type { BankTransaction } from '@/types/finance-types'
import { useMatchTransaction, useRejectMatch, useInvoices } from '@/api/hooks/useFinance'

const STATUS_LABEL_KEYS: Record<BankTransaction['matchStatus'], string> = {
  matched: 'finanzen.banking.filterMatched',
  suggested: 'finanzen.banking.filterSuggested',
  unmatched: 'finanzen.banking.filterUnmatched',
}
const STATUS_STYLES: Record<BankTransaction['matchStatus'], string> = {
  matched: 'bg-success-light text-success',
  suggested: 'bg-warning/15 text-warning',
  unmatched: 'bg-secondary text-muted-foreground',
}

interface Props {
  transaction: BankTransaction
  onClose: () => void
}

export function BankTransactionDetailPanel({ transaction: tx, onClose }: Props) {
  const { t } = useTranslation()
  const matchMutation = useMatchTransaction()
  const rejectMutation = useRejectMatch()
  const { data: invoicesData } = useInvoices()

  const formatEUR = (v: number) =>
    new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' }).format(v)

  // Offene Rechnungen als Zuordnungs-Kandidaten (nur bei Eingängen relevant).
  const openInvoices = (invoicesData?.invoices ?? []).filter(
    (inv) => inv.status === 'sent' || inv.status === 'overdue',
  )

  const handleConfirm = () =>
    matchMutation.mutate(
      { id: tx.id },
      {
        onSuccess: () => { toast.success(t('finanzen.banking.matchConfirmed')); onClose() },
        onError: (err) => toast.error(err.message),
      },
    )
  const handleReject = () =>
    rejectMutation.mutate(tx.id, {
      onSuccess: () => { toast.success(t('finanzen.banking.matchRejected')); onClose() },
      onError: (err) => toast.error(err.message),
    })
  const handleAssign = (invoiceNumber: string) =>
    matchMutation.mutate(
      { id: tx.id, invoiceNumber },
      {
        onSuccess: () => { toast.success(t('finanzen.banking.matchConfirmed')); onClose() },
        onError: (err) => toast.error(err.message),
      },
    )

  const isCredit = tx.type === 'credit'
  const rows: { label: string; value: React.ReactNode }[] = [
    { label: t('finanzen.banking.date'), value: formatDate(tx.date) },
    {
      label: t('finanzen.banking.type'),
      value: (
        <span className={`inline-flex items-center gap-1 ${isCredit ? 'text-success' : 'text-foreground'}`}>
          {isCredit ? <ArrowDownRight className="h-3.5 w-3.5" /> : <ArrowUpRight className="h-3.5 w-3.5 text-muted-foreground" />}
          {isCredit ? t('finanzen.banking.credit') : t('finanzen.banking.debit')}
        </span>
      ),
    },
    { label: t('finanzen.banking.counterpart'), value: tx.counterpart },
  ]

  return (
    <DetailModal open={true} title={t('finanzen.banking.transactionDetail')} onClose={onClose}>
      <div className="space-y-4">
        {/* Header: Beschreibung + Betrag */}
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="truncate text-base font-medium text-foreground">{tx.description}</h3>
            <p className="text-xs text-muted-foreground">{tx.counterpart}</p>
          </div>
          <span className={`shrink-0 text-base font-semibold ${isCredit ? 'text-success' : 'text-foreground'}`}>
            {isCredit ? '+' : '−'}{formatEUR(Math.abs(tx.amount))}
          </span>
        </div>

        {/* Key info */}
        <div className="grid grid-cols-2 gap-3 text-xs">
          {rows.map((row) => (
            <div key={row.label}>
              <p className="text-[10px] text-muted-foreground">{row.label}</p>
              <p className="text-foreground">{row.value}</p>
            </div>
          ))}
          <div>
            <p className="text-[10px] text-muted-foreground">{t('finanzen.banking.assignment')}</p>
            <span className={`mt-0.5 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${STATUS_STYLES[tx.matchStatus]}`}>
              {tx.matchStatus === 'matched' && <CheckCircle2 className="h-3 w-3" />}
              {tx.matchStatus === 'suggested' && <HelpCircle className="h-3 w-3" />}
              {t(STATUS_LABEL_KEYS[tx.matchStatus])}
            </span>
          </div>
        </div>

        {/* Zugeordnete Rechnung */}
        {tx.matchedInvoice && (
          <div className="rounded-md border border-border px-3 py-2">
            <p className="text-[10px] text-muted-foreground">{t('finanzen.banking.assignedInvoice')}</p>
            <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
              <FileText className="h-3.5 w-3.5 text-primary" />
              <span className="font-mono text-primary">{tx.matchedInvoice}</span>
            </p>
          </div>
        )}

        {/* Aktionen: vorgeschlagene Zuordnung bestätigen/ablehnen */}
        {tx.matchStatus === 'suggested' && (
          <div className="flex flex-wrap gap-2 border-t border-border pt-4">
            <Button size="sm" onClick={handleConfirm} disabled={matchMutation.isPending}>
              <CheckCircle2 className="mr-1.5 h-4 w-4" />
              {t('finanzen.banking.confirmMatch')}
            </Button>
            <Button variant="outline" size="sm" onClick={handleReject} disabled={rejectMutation.isPending}>
              {t('finanzen.banking.rejectMatch')}
            </Button>
          </div>
        )}

        {/* Manuelle Zuordnung: offene Rechnungen wählen (nur Eingänge) */}
        {tx.matchStatus === 'unmatched' && isCredit && (
          <section className="border-t border-border pt-4">
            <h4 className="mb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              {t('finanzen.banking.assignInvoice')}
            </h4>
            {openInvoices.length === 0 ? (
              <p className="text-xs text-muted-foreground">{t('finanzen.banking.noOpenInvoices')}</p>
            ) : (
              <div className="overflow-hidden rounded-md border border-border">
                {openInvoices.map((inv, idx) => (
                  <button
                    key={inv.id}
                    type="button"
                    onClick={() => handleAssign(inv.invoice_number)}
                    disabled={matchMutation.isPending}
                    className={`group flex w-full items-center justify-between px-3 py-2 text-left text-xs transition-colors hover:bg-secondary/50 disabled:opacity-50 ${
                      idx > 0 ? 'border-t border-border-muted' : ''
                    }`}
                  >
                    <span className="flex min-w-0 items-center gap-2">
                      <span className="font-mono text-primary">{inv.invoice_number}</span>
                      <span className="truncate text-[10px] text-muted-foreground">{inv.customer_name ?? inv.customer?.name}</span>
                    </span>
                    <span className="flex shrink-0 items-center gap-1">
                      <span className="font-medium text-foreground">{formatEUR(Number(inv.total_gross ?? 0))}</span>
                      <ChevronRight className="h-3.5 w-3.5 text-muted-foreground transition-[color,transform] group-hover:translate-x-0.5 group-hover:text-primary" />
                    </span>
                  </button>
                ))}
              </div>
            )}
          </section>
        )}
      </div>
    </DetailModal>
  )
}
