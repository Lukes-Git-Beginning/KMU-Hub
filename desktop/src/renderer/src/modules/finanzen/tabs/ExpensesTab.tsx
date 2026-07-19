/**
 * Ausgaben-Tab für den Buchhaltungs-Hub (finanzen P2).
 *
 * Datenquelle: MSW-Handler über `useFinanceLedger` (stateful Mock, swap-ready).
 * Jede Ausgabe trägt ein SKR-Sachkonto (Kontierung) + optional einen Beleg.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Receipt, Search, Paperclip } from 'lucide-react'
import { toast } from 'sonner'
import {
  useExpenses,
  useApproveExpense,
  useRejectExpense,
  useDeleteExpense,
} from '@/api/hooks/useFinanceLedger'
import type { Expense } from '@/stores/finance'
import { ItemActions, ConfirmDialog, EmptyState } from '@/components/shared'
import { formatCurrency } from '@/lib/format'
import { useFinanceTenantStore } from '@/stores/financeTenant'
import { formatAccount } from '../lib/skr-accounts'
import { ExpenseFormDialog } from '../ExpenseFormDialog'
import { ReceiptPreviewDialog } from '../ReceiptPreviewDialog'
import { ExpenseDetailPanel } from '../ExpenseDetailPanel'
import { useHasCapability } from '@/hooks/useCapability'
import { useAmountsVisible } from '../lib/amounts-visibility'

export function ExpensesTab() {
  const { t, i18n } = useTranslation()
  const { data: expenses = [] } = useExpenses()
  const approveExpense = useApproveExpense()
  const rejectExpense = useRejectExpense()
  const deleteExpense = useDeleteExpense()
  const framework = useFinanceTenantStore((s) => s.chartFramework)

  const [search, setSearch] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [editExpense, setEditExpense] = useState<Expense | null>(null)
  const [detailExpense, setDetailExpense] = useState<Expense | null>(null)
  const [previewExpense, setPreviewExpense] = useState<Expense | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<{ id: string; label: string } | null>(null)

  // RBAC R-3
  const canBook    = useHasCapability('finance:incoming:book')
  const canReview  = useHasCapability('finance:incoming:review')
  const amountsVisible = useAmountsVisible()

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return expenses
    return expenses.filter(
      (e) =>
        e.description.toLowerCase().includes(q) ||
        e.supplier.toLowerCase().includes(q) ||
        e.category.toLowerCase().includes(q) ||
        (e.account ?? '').includes(q),
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

  const openNew = () => {
    setEditExpense(null)
    setShowForm(true)
  }
  const openEdit = (exp: Expense) => {
    setEditExpense(exp)
    setShowForm(true)
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
        {/* Neu → incoming:book */}
        {canBook && (
          <button
            type="button"
            onClick={openNew}
            className="flex items-center gap-2 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            {t('buchhaltung.newExpense')}
          </button>
        )}
      </div>

      {filtered.length === 0 ? (
        <EmptyState
          icon={Receipt}
          title={t('buchhaltung.empty.expensesTitle')}
          description={t('buchhaltung.empty.expensesDesc')}
          action={canBook ? { label: t('buchhaltung.newExpense'), onClick: openNew } : undefined}
        />
      ) : (
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <div className="grid grid-cols-[1fr_110px_120px_100px_130px_96px_40px] gap-3 border-b border-border bg-secondary/40 px-4 py-2.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
            <span>{t('buchhaltung.table.description')}</span>
            <span>{t('buchhaltung.table.category')}</span>
            <span>{t('buchhaltung.table.account')}</span>
            <span className="text-right tabular-nums">{t('buchhaltung.table.amount')}</span>
            <span>{t('buchhaltung.table.supplier')}</span>
            <span>{t('common.status')}</span>
            <span />
          </div>
          {filtered.map((exp) => (
            <div
              key={exp.id}
              role="button"
              tabIndex={0}
              onClick={() => setDetailExpense(exp)}
              onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setDetailExpense(exp) } }}
              className="grid cursor-pointer grid-cols-[1fr_110px_120px_100px_130px_96px_40px] items-center gap-3 border-b border-border-muted px-4 py-3 transition-colors last:border-b-0 hover:bg-secondary/40 focus-visible:bg-secondary/40 focus-visible:outline-none"
            >
              <div className="flex min-w-0 items-center gap-2">
                <div className="min-w-0">
                  <p className="truncate text-sm text-foreground">{exp.description}</p>
                  <p className="text-[11px] text-muted-foreground">
                    {new Date(exp.date).toLocaleDateString(i18n.language)}
                    {exp.project && ` · ${exp.project}`}
                  </p>
                </div>
                {exp.receipt && (
                  <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); setPreviewExpense(exp) }}
                    aria-label={t('buchhaltung.actions.viewReceipt')}
                    title={exp.receiptName ?? t('buchhaltung.form.receipt')}
                    className="shrink-0 rounded-md p-1 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                  >
                    <Paperclip className="h-3.5 w-3.5" aria-hidden="true" />
                  </button>
                )}
              </div>
              <span className="truncate text-xs text-muted-foreground">
                {t(`buchhaltung.categories.${exp.category}`, { defaultValue: exp.category })}
              </span>
              {exp.account ? (
                <span
                  className="truncate font-mono text-xs text-foreground"
                  title={formatAccount(exp.account, framework)}
                >
                  {exp.account}
                </span>
              ) : (
                /* Unkategorisiert → incoming:book */
                canBook ? (
                  <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); openEdit(exp) }}
                    className="w-fit text-xs text-warning underline-offset-2 hover:underline"
                  >
                    {t('buchhaltung.table.uncategorized')}
                  </button>
                ) : (
                  <span className="text-xs text-muted-foreground">{t('buchhaltung.table.uncategorized')}</span>
                )
              )}
              <span
                className="text-right text-sm font-medium tabular-nums text-error"
                title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}
              >
                {amountsVisible ? `−${formatCurrency(exp.amount)}` : '•••'}
              </span>
              <span className="truncate text-xs text-muted-foreground">{exp.supplier}</span>
              <span className={`inline-flex w-fit items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${statusColors[exp.status]}`}>
                {statusLabels[exp.status]}
              </span>
              <div onClick={(e) => e.stopPropagation()}>
              <ItemActions
                items={[
                  {
                    label: t('common.details'),
                    onClick: () => setDetailExpense(exp),
                  },
                  // Edit → incoming:book
                  ...(canBook ? [{
                    label: t('buchhaltung.actions.editExpense'),
                    onClick: () => openEdit(exp),
                  }] : []),
                  // Approve/Reject → incoming:review
                  ...(exp.status === 'pending' && canReview
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
                  // Delete → incoming:book
                  ...(canBook ? [{
                    label: t('common.delete'),
                    variant: 'destructive' as const,
                    onClick: () => setConfirmDelete({ id: exp.id, label: exp.description }),
                  }] : []),
                ]}
              />
              </div>
            </div>
          ))}
        </div>
      )}

      <ExpenseFormDialog open={showForm} onOpenChange={setShowForm} editExpense={editExpense} />

      <ReceiptPreviewDialog expense={previewExpense} onClose={() => setPreviewExpense(null)} />

      {detailExpense && (
        <ExpenseDetailPanel
          expense={detailExpense}
          onClose={() => setDetailExpense(null)}
          onEdit={() => {
            openEdit(detailExpense)
            setDetailExpense(null)
          }}
        />
      )}

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
