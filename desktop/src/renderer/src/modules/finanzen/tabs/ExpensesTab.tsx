/**
 * Ausgaben-Tab fuer den Buchhaltungs-Hub.
 *
 * Migriert aus dem alten `buchhaltung`-Modul (Sprint 1A).
 * Datenquelle: `useFinanceStore` (Zustand-Mock). Backend-Anbindung folgt
 * in einem spaeteren Sprint via TanStack Query (siehe `_DEPRECATED.md`).
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Receipt, Search } from 'lucide-react'
import { toast } from 'sonner'
import { useExpenses, useApproveExpense, useRejectExpense, useDeleteExpense } from '@/api/hooks/useFinanceLedger'
import { ItemActions, ConfirmDialog, EmptyState } from '@/components/shared'
import { formatCurrency } from '@/lib/format'
import { ExpenseFormDialog } from '../ExpenseFormDialog'

export function ExpensesTab() {
  const { t, i18n } = useTranslation()
  const { data: expenses = [] } = useExpenses()
  const approveExpense = useApproveExpense()
  const rejectExpense = useRejectExpense()
  const deleteExpense = useDeleteExpense()

  const [search, setSearch] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState<{ id: string; label: string } | null>(null)

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return expenses
    return expenses.filter(
      (e) =>
        e.description.toLowerCase().includes(q) ||
        e.supplier.toLowerCase().includes(q) ||
        e.category.toLowerCase().includes(q),
    )
  }, [expenses, search])

  const statusColors: Record<string, string> = {
    pending: 'bg-warning-light text-warning',
    approved: 'bg-success-light text-success',
    rejected: 'bg-error-light text-error',
  }

  const statusLabels: Record<string, string> = {
    pending: t('buchhaltung.expenseStatus.pending'),
    approved: t('buchhaltung.expenseStatus.approved'),
    rejected: t('buchhaltung.expenseStatus.rejected'),
  }

  return (
    <div>
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
          <input
            type="search"
            name="expenses-search"
            autoComplete="off"
            aria-label={t('buchhaltung.search.expenses', { defaultValue: 'Ausgaben durchsuchen' })}
            placeholder={t('buchhaltung.search.expenses', { defaultValue: 'Ausgaben durchsuchen' })}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
          />
        </div>
        <button
          type="button"
          onClick={() => setShowForm(true)}
          className="flex items-center gap-2 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
        >
          <Plus className="h-4 w-4" aria-hidden="true" />
          {t('buchhaltung.newExpense')}
        </button>
      </div>

      {filtered.length === 0 ? (
        <EmptyState
          icon={Receipt}
          title={t('buchhaltung.empty.expensesTitle')}
          description={t('buchhaltung.empty.expensesDesc')}
          action={{ label: t('buchhaltung.newExpense'), onClick: () => setShowForm(true) }}
        />
      ) : (
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <div className="grid grid-cols-[1fr_120px_100px_140px_100px_40px] gap-3 border-b border-border bg-secondary/40 px-4 py-2.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
            <span>{t('buchhaltung.table.description')}</span>
            <span>{t('buchhaltung.table.category')}</span>
            <span className="text-right tabular-nums">{t('buchhaltung.table.amount')}</span>
            <span>{t('buchhaltung.table.supplier')}</span>
            <span>{t('common.status')}</span>
            <span />
          </div>
          {filtered.map((exp) => (
            <div
              key={exp.id}
              className="grid grid-cols-[1fr_120px_100px_140px_100px_40px] items-center gap-3 border-b border-border-muted px-4 py-3 last:border-b-0"
            >
              <div className="min-w-0">
                <p className="truncate text-sm text-foreground">{exp.description}</p>
                <p className="text-[11px] text-muted-foreground">
                  {new Date(exp.date).toLocaleDateString(i18n.language)}
                  {exp.project && ` · ${exp.project}`}
                </p>
              </div>
              <span className="truncate text-xs text-muted-foreground">{exp.category}</span>
              <span className="text-right text-sm font-medium tabular-nums text-error">
                −{formatCurrency(exp.amount)}
              </span>
              <span className="truncate text-xs text-muted-foreground">{exp.supplier}</span>
              <span className={`inline-flex w-fit items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${statusColors[exp.status]}`}>
                {statusLabels[exp.status]}
              </span>
              <ItemActions
                items={[
                  ...(exp.status === 'pending'
                    ? [
                        {
                          label: t('buchhaltung.actions.approve'),
                          onClick: () => {
                            approveExpense.mutate(exp.id)
                            toast.success(t('buchhaltung.toast.approved'))
                          },
                        },
                        {
                          label: t('buchhaltung.actions.reject'),
                          onClick: () => {
                            rejectExpense.mutate(exp.id)
                            toast.success(t('buchhaltung.toast.rejected'))
                          },
                        },
                      ]
                    : []),
                  {
                    label: t('common.delete'),
                    variant: 'destructive' as const,
                    onClick: () => setConfirmDelete({ id: exp.id, label: exp.description }),
                  },
                ]}
              />
            </div>
          ))}
        </div>
      )}

      <ExpenseFormDialog open={showForm} onOpenChange={setShowForm} />

      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title={t('buchhaltung.delete.expenseTitle')}
        description={t('buchhaltung.delete.expenseDescription', { label: confirmDelete?.label ?? '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={() => {
          if (confirmDelete) {
            deleteExpense.mutate(confirmDelete.id)
            toast.success(t('buchhaltung.toast.deleted'))
            setConfirmDelete(null)
          }
        }}
      />
    </div>
  )
}
