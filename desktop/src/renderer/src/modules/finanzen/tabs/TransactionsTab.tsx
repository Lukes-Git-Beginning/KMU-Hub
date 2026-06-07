/**
 * Transaktionen-Tab fuer den Buchhaltungs-Hub.
 *
 * Migriert aus dem alten `buchhaltung`-Modul (Sprint 1A).
 * Datenquelle: `useFinanceStore` (Zustand-Mock). Backend-Anbindung folgt
 * in einem spaeteren Sprint via TanStack Query (siehe `_DEPRECATED.md`).
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowDownRight, ArrowUpRight, Search } from 'lucide-react'
import { toast } from 'sonner'
import { useTransactions, useDeleteTransaction } from '@/api/hooks/useFinanceLedger'
import { ItemActions, ConfirmDialog, EmptyState } from '@/components/shared'
import { formatCurrency } from '@/lib/format'
import { Receipt } from 'lucide-react'

type TypeFilter = 'all' | 'income' | 'expense'

export function TransactionsTab() {
  const { t, i18n } = useTranslation()
  const { data: transactions = [] } = useTransactions()
  const deleteTransaction = useDeleteTransaction()

  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('all')
  const [confirmDelete, setConfirmDelete] = useState<{ id: string; label: string } | null>(null)

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    return transactions.filter((tx) => {
      if (typeFilter !== 'all' && tx.type !== typeFilter) return false
      if (!q) return true
      return (
        tx.description.toLowerCase().includes(q) ||
        tx.category.toLowerCase().includes(q) ||
        (tx.reference ?? '').toLowerCase().includes(q)
      )
    })
  }, [transactions, search, typeFilter])

  const totalIncome = useMemo(
    () => transactions.filter((tx) => tx.type === 'income').reduce((sum, tx) => sum + tx.amount, 0),
    [transactions],
  )
  const totalExpense = useMemo(
    () => transactions.filter((tx) => tx.type === 'expense').reduce((sum, tx) => sum + Math.abs(tx.amount), 0),
    [transactions],
  )
  const balance = totalIncome - totalExpense

  const filters: { key: TypeFilter; labelKey: string; defaultValue: string }[] = [
    { key: 'all', labelKey: 'buchhaltung.filter.all', defaultValue: 'Alle' },
    { key: 'income', labelKey: 'buchhaltung.filter.income', defaultValue: 'Einnahmen' },
    { key: 'expense', labelKey: 'buchhaltung.filter.expense', defaultValue: 'Ausgaben' },
  ]

  return (
    <div>
      {/* Inline-Stats statt 3-Card-Grid */}
      <dl className="mb-6 flex flex-wrap items-end gap-x-10 gap-y-3 border-b border-border pb-5 text-sm">
        <div className="min-w-[140px]">
          <dt className="text-[11px] uppercase tracking-wider text-muted-foreground">
            {t('buchhaltung.stats.income', { defaultValue: 'Einnahmen' })}
          </dt>
          <dd className="mt-0.5 text-2xl font-semibold leading-none text-success tabular-nums">
            {formatCurrency(totalIncome)}
          </dd>
        </div>
        <div className="min-w-[140px]">
          <dt className="text-[11px] uppercase tracking-wider text-muted-foreground">
            {t('buchhaltung.stats.expense', { defaultValue: 'Ausgaben' })}
          </dt>
          <dd className="mt-0.5 text-2xl font-semibold leading-none text-error tabular-nums">
            −{formatCurrency(totalExpense)}
          </dd>
        </div>
        <div className="min-w-[140px]">
          <dt className="text-[11px] uppercase tracking-wider text-muted-foreground">
            {t('buchhaltung.stats.balance', { defaultValue: 'Saldo' })}
          </dt>
          <dd className={`mt-0.5 text-2xl font-semibold leading-none tabular-nums ${balance >= 0 ? 'text-foreground' : 'text-error'}`}>
            {formatCurrency(balance)}
          </dd>
        </div>
      </dl>

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
          <input
            type="search"
            name="transactions-search"
            autoComplete="off"
            aria-label={t('buchhaltung.search.transactions', { defaultValue: 'Transaktionen durchsuchen' })}
            placeholder={t('buchhaltung.search.transactions', { defaultValue: 'Transaktionen durchsuchen' })}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
          />
        </div>
        <div className="flex items-center gap-1" role="group" aria-label={t('buchhaltung.filter.type', { defaultValue: 'Typ-Filter' })}>
          {filters.map((f) => (
            <button
              key={f.key}
              type="button"
              onClick={() => setTypeFilter(f.key)}
              aria-pressed={typeFilter === f.key}
              className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
                typeFilter === f.key
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-secondary text-muted-foreground hover:text-foreground'
              }`}
            >
              {t(f.labelKey, { defaultValue: f.defaultValue })}
            </button>
          ))}
        </div>
      </div>

      {filtered.length === 0 ? (
        <EmptyState
          icon={Receipt}
          title={t('buchhaltung.empty.transactionsTitle', { defaultValue: 'Keine Transaktionen' })}
          description={t('buchhaltung.empty.transactionsDesc', { defaultValue: 'Sobald Buchungen vorliegen, erscheinen sie hier.' })}
        />
      ) : (
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <div className="grid grid-cols-[1fr_140px_140px_120px_40px] gap-3 border-b border-border bg-secondary/40 px-4 py-2.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
            <span>{t('buchhaltung.table.description')}</span>
            <span>{t('buchhaltung.table.category')}</span>
            <span className="text-right tabular-nums">{t('buchhaltung.table.amount')}</span>
            <span className="text-right tabular-nums">{t('buchhaltung.table.date')}</span>
            <span />
          </div>
          {filtered.map((tx) => (
            <div
              key={tx.id}
              className="grid grid-cols-[1fr_140px_140px_120px_40px] items-center gap-3 border-b border-border-muted px-4 py-3 last:border-b-0"
            >
              <div className="flex min-w-0 items-center gap-3">
                <div
                  className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${
                    tx.type === 'income' ? 'bg-success-light' : 'bg-error-light'
                  }`}
                  aria-hidden="true"
                >
                  {tx.type === 'income' ? (
                    <ArrowUpRight className="h-4 w-4 text-success" />
                  ) : (
                    <ArrowDownRight className="h-4 w-4 text-error" />
                  )}
                </div>
                <div className="min-w-0">
                  <p className="truncate text-sm text-foreground">{tx.description}</p>
                  {tx.reference && (
                    <p className="truncate text-[11px] text-muted-foreground">{tx.reference}</p>
                  )}
                </div>
              </div>
              <span className="truncate text-xs text-muted-foreground">{tx.category}</span>
              <span
                className={`text-right text-sm font-medium tabular-nums ${
                  tx.type === 'income' ? 'text-success' : 'text-error'
                }`}
              >
                {tx.type === 'income' ? '+' : '−'}
                {formatCurrency(Math.abs(tx.amount))}
              </span>
              <span className="text-right text-xs text-muted-foreground tabular-nums">
                {new Date(tx.date).toLocaleDateString(i18n.language)}
              </span>
              <ItemActions
                items={[
                  {
                    label: t('common.delete'),
                    variant: 'destructive' as const,
                    onClick: () => setConfirmDelete({ id: tx.id, label: tx.description }),
                  },
                ]}
              />
            </div>
          ))}
        </div>
      )}

      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title={t('buchhaltung.delete.transactionTitle', { defaultValue: 'Transaktion löschen' })}
        description={t('buchhaltung.delete.transactionDescription', {
          defaultValue: 'Möchtest du „{{label}}" wirklich löschen?',
          label: confirmDelete?.label ?? '',
        })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={() => {
          if (confirmDelete) {
            deleteTransaction.mutate(confirmDelete.id)
            toast.success(t('buchhaltung.toast.deleted'))
            setConfirmDelete(null)
          }
        }}
      />
    </div>
  )
}
