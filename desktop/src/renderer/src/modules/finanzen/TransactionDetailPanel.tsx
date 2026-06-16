/**
 * TransactionDetailPanel — Transaktions-Detailansicht (finanzen P2.5b).
 *
 * Reine Inhaltsansicht (Buchungssatz-Beleg): Typ, Betrag, Datum, Kategorie,
 * Referenz. Daten aus der Liste. Nutzt das shared `DetailPanel`.
 */
import { useTranslation } from 'react-i18next'
import { ArrowDownRight, ArrowUpRight, Calendar, Tag, Hash, Activity } from 'lucide-react'
import { DetailPanel } from '@/components/shared'
import type { Transaction } from '@/stores/finance'
import { formatCurrency, formatDate } from '@/lib/format'

interface TransactionDetailPanelProps {
  transaction: Transaction
  onClose: () => void
}

export function TransactionDetailPanel({ transaction: tx, onClose }: TransactionDetailPanelProps) {
  const { t, i18n } = useTranslation()
  const isIncome = tx.type === 'income'

  const info: { icon: typeof Calendar; label: string; value: string }[] = [
    { icon: Calendar, label: t('buchhaltung.table.date', { defaultValue: 'Datum' }), value: new Date(tx.date).toLocaleDateString(i18n.language) },
    { icon: Tag, label: t('buchhaltung.table.category'), value: tx.category },
    { icon: Activity, label: t('common.status'), value: tx.status === 'completed' ? t('buchhaltung.txStatus.completed') : t('buchhaltung.txStatus.pending') },
  ]
  if (tx.reference) {
    info.push({ icon: Hash, label: t('buchhaltung.transactionDetail.reference'), value: tx.reference })
  }

  return (
    <DetailPanel open={true} title={t('buchhaltung.transactionDetail.title')} onClose={onClose}>
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
      </div>
    </DetailPanel>
  )
}
