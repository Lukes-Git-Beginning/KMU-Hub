/**
 * TransactionDetailPanel — Transaktions-Detailansicht (finanzen P2.5b).
 *
 * Reine Inhaltsansicht (Buchungssatz-Beleg): Typ, Betrag, Datum, Kategorie,
 * Referenz. Daten aus der Liste. Nutzt das shared `DetailPanel`.
 */
import { useTranslation } from 'react-i18next'
import { ArrowDownRight, ArrowUpRight, Calendar, Tag, Hash, Activity, FileText, ChevronRight } from 'lucide-react'
import { DetailModal } from '@/components/shared'
import type { Transaction } from '@/stores/finance'
import { formatCurrency, formatDate } from '@/lib/format'
import { useInvoices } from '@/api/hooks/useFinance'
import { useTransactions } from '@/api/hooks/useFinanceLedger'
import { useFinanceDetailNavOptional } from './FinanceDetailNav'

interface TransactionDetailPanelProps {
  transaction: Transaction
  onClose: () => void
}

export function TransactionDetailPanel({ transaction: tx, onClose }: TransactionDetailPanelProps) {
  const { t, i18n } = useTranslation()
  const { data: invoicesData } = useInvoices()
  const { data: allTransactions = [] } = useTransactions()
  const nav = useFinanceDetailNavOptional()
  const isIncome = tx.type === 'income'

  // Verknüpfte Rechnung (falls die Buchung aus einem Zahlungseingang stammt).
  const linkedInvoice = tx.invoiceId
    ? (invoicesData?.invoices ?? []).find((inv) => inv.id === tx.invoiceId)
    : undefined

  // Verwandte Buchungen derselben Kategorie (neueste zuerst).
  const relatedTransactions = allTransactions
    .filter((x) => x.id !== tx.id && x.category === tx.category)
    .sort((a, b) => (a.date < b.date ? 1 : -1))
    .slice(0, 5)

  const info: { icon: typeof Calendar; label: string; value: string }[] = [
    { icon: Calendar, label: t('buchhaltung.table.date', { defaultValue: 'Datum' }), value: new Date(tx.date).toLocaleDateString(i18n.language) },
    { icon: Tag, label: t('buchhaltung.table.category'), value: tx.category },
    { icon: Activity, label: t('common.status'), value: tx.status === 'completed' ? t('buchhaltung.txStatus.completed') : t('buchhaltung.txStatus.pending') },
  ]
  if (tx.reference) {
    info.push({ icon: Hash, label: t('buchhaltung.transactionDetail.reference'), value: tx.reference })
  }

  return (
    <DetailModal open={true} title={t('buchhaltung.transactionDetail.title')} onClose={onClose}>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-center gap-3">
          <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${isIncome ? 'bg-success-light' : 'bg-error-light'}`}>
            {isIncome ? <ArrowUpRight className="h-5 w-5 text-success" /> : <ArrowDownRight className="h-5 w-5 text-error" />}
          </div>
          <div className="min-w-0">
            <h3 className="truncate text-base font-medium text-foreground">{tx.description}</h3>
            <p className={`text-xl font-semibold tabular-nums ${isIncome ? 'text-success' : 'text-error'}`}>
              {isIncome ? '+' : '−'}{formatCurrency(Math.abs(tx.amount))}
            </p>
          </div>
        </div>

        {/* Type badge */}
        <span className={`inline-flex w-fit items-center rounded-full px-2.5 py-0.5 text-[10px] font-medium ${isIncome ? 'bg-success-light text-success' : 'bg-error-light text-error'}`}>
          {isIncome ? t('buchhaltung.filter.income', { defaultValue: 'Einnahme' }) : t('buchhaltung.filter.expense', { defaultValue: 'Ausgabe' })}
        </span>

        {/* Key info */}
        <div className="grid grid-cols-2 gap-3 text-xs">
          {info.map((row) => (
            <div key={row.label} className="flex items-center gap-2 text-muted-foreground">
              <row.icon className="h-3.5 w-3.5 shrink-0" />
              <div className="min-w-0">
                <p className="text-[10px] text-muted-foreground">{row.label}</p>
                <p className="truncate text-foreground">{row.value}</p>
              </div>
            </div>
          ))}
        </div>

        {/* Verknüpfte Rechnung (klickbar) */}
        {linkedInvoice && (
          <section>
            <h4 className="mb-2 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              <FileText className="h-3 w-3" />
              {t('buchhaltung.transactionDetail.linkedInvoice')}
            </h4>
            <button
              type="button"
              disabled={!nav}
              onClick={() => nav?.open('invoice', linkedInvoice.id)}
              className="flex w-full items-center justify-between rounded-md border border-border px-3 py-2 text-left text-xs transition-colors hover:bg-secondary/50 disabled:cursor-default disabled:hover:bg-transparent"
            >
              <span className="flex min-w-0 items-center gap-2">
                <span className="font-mono text-primary">{linkedInvoice.invoice_number}</span>
                <span className="truncate text-[10px] text-muted-foreground">{linkedInvoice.customer?.name}</span>
              </span>
              {nav && <ChevronRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />}
            </button>
          </section>
        )}

        {/* Verwandte Buchungen derselben Kategorie */}
        {relatedTransactions.length > 0 && (
          <section>
            <h4 className="mb-2 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              <Tag className="h-3 w-3" />
              {t('buchhaltung.transactionDetail.relatedInCategory', { category: tx.category })}
            </h4>
            <div className="overflow-hidden rounded-md border border-border">
              {relatedTransactions.map((x, idx) => {
                const inc = x.type === 'income'
                return (
                  <div
                    key={x.id}
                    className={`flex items-center justify-between px-3 py-2 text-xs ${idx > 0 ? 'border-t border-border-muted' : ''}`}
                  >
                    <span className="flex min-w-0 items-center gap-2">
                      <span className="truncate text-foreground">{x.description}</span>
                      <span className="shrink-0 text-[10px] text-muted-foreground">
                        {new Date(x.date).toLocaleDateString(i18n.language)}
                      </span>
                    </span>
                    <span className={`shrink-0 font-medium tabular-nums ${inc ? 'text-success' : 'text-error'}`}>
                      {inc ? '+' : '−'}{formatCurrency(Math.abs(x.amount))}
                    </span>
                  </div>
                )
              })}
            </div>
          </section>
        )}
      </div>
    </DetailModal>
  )
}
