/**
 * ExpenseDetailPanel — Ausgaben-Detailansicht (finanzen P2.5b).
 *
 * Öffnet beim Zeilenklick in ExpensesTab: alle Felder + Kontierung + Beleg
 * (mit Vorschau) + Genehmigen/Ablehnen/Bearbeiten. Daten kommen direkt aus der
 * Liste (kein Extra-Fetch nötig). Nutzt das shared `DetailPanel`.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Calendar, Tag, Building2, FolderOpen, BookOpen, Paperclip, CheckCircle2, XCircle } from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import type { Expense } from '@/stores/finance'
import { useApproveExpense, useRejectExpense, useExpenses } from '@/api/hooks/useFinanceLedger'
import { useFinanceTenantStore } from '@/stores/financeTenant'
import { formatCurrency, formatDate } from '@/lib/format'
import { formatAccount } from './lib/skr-accounts'
import { ReceiptPreviewDialog } from './ReceiptPreviewDialog'

interface ExpenseDetailPanelProps {
  expense: Expense
  onClose: () => void
  onEdit: () => void
}

export function ExpenseDetailPanel({ expense, onClose, onEdit }: ExpenseDetailPanelProps) {
  const { t, i18n } = useTranslation()
  const framework = useFinanceTenantStore((s) => s.chartFramework)
  const { data: allExpenses = [] } = useExpenses()
  const approve = useApproveExpense()
  const reject = useRejectExpense()
  const [showReceipt, setShowReceipt] = useState(false)

  // Lieferanten-Rollup: weitere Ausgaben desselben Lieferanten.
  const supplierExpenses = expense.supplier
    ? allExpenses
        .filter((e) => e.id !== expense.id && e.supplier === expense.supplier)
        .sort((a, b) => (a.date < b.date ? 1 : -1))
    : []
  const supplierTotal = supplierExpenses.reduce((sum, e) => sum + e.amount, 0) + expense.amount

  const statusStyles: Record<string, string> = {
    pending: 'bg-warning-light text-warning',
    approved: 'bg-success-light text-success',
    rejected: 'bg-error-light text-error',
  }
  const statusLabels: Record<string, string> = {
    pending: t('buchhaltung.expenseStatus.pending'),
    approved: t('buchhaltung.expenseStatus.approved'),
    rejected: t('buchhaltung.expenseStatus.rejected'),
  }

  const info: { icon: typeof Calendar; label: string; value: string }[] = [
    { icon: Calendar, label: t('buchhaltung.table.date', { defaultValue: 'Datum' }), value: new Date(expense.date).toLocaleDateString(i18n.language) },
    { icon: Tag, label: t('buchhaltung.table.category'), value: t(`buchhaltung.categories.${expense.category}`, { defaultValue: expense.category }) },
    { icon: Building2, label: t('buchhaltung.table.supplier'), value: expense.supplier || '–' },
    { icon: BookOpen, label: t('buchhaltung.form.account'), value: expense.account ? formatAccount(expense.account, framework) : t('buchhaltung.table.uncategorized') },
  ]
  if (expense.project) {
    info.push({ icon: FolderOpen, label: t('buchhaltung.form.project'), value: expense.project })
  }

  return (
    <DetailModal open={true} title={t('buchhaltung.expenseDetail.title')} onClose={onClose}>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="text-base font-medium text-foreground">{expense.description}</h3>
            <p className="mt-0.5 text-xl font-semibold tabular-nums text-error">−{formatCurrency(expense.amount)}</p>
          </div>
          <span className={`inline-flex shrink-0 items-center rounded-full px-2.5 py-0.5 text-[10px] font-medium ${statusStyles[expense.status]}`}>
            {statusLabels[expense.status]}
          </span>
        </div>

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

        {/* Receipt */}
        <section>
          <h4 className="mb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            {t('buchhaltung.form.receipt')}
          </h4>
          {expense.receipt ? (
            <button
              type="button"
              onClick={() => setShowReceipt(true)}
              className="flex w-full items-center gap-2 rounded-lg border border-border bg-secondary/40 px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-secondary"
            >
              <Paperclip className="h-4 w-4 shrink-0 text-muted-foreground" />
              <span className="truncate">{expense.receiptName ?? t('buchhaltung.actions.viewReceipt')}</span>
            </button>
          ) : (
            <p className="text-xs text-muted-foreground">{t('buchhaltung.expenseDetail.noReceipt')}</p>
          )}
        </section>

        {/* Lieferanten-Rollup */}
        {expense.supplier && (
          <section>
            <h4 className="mb-2 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              <Building2 className="h-3 w-3" />
              {t('buchhaltung.expenseDetail.supplierHistory', { supplier: expense.supplier })}
            </h4>
            <div className="overflow-hidden rounded-md border border-border">
              <div className="flex items-center justify-between border-b border-border-muted bg-secondary/30 px-3 py-2 text-xs">
                <span className="text-muted-foreground">
                  {t('buchhaltung.expenseDetail.supplierTotal', { count: supplierExpenses.length + 1 })}
                </span>
                <span className="font-semibold tabular-nums text-foreground">{formatCurrency(supplierTotal)}</span>
              </div>
              {supplierExpenses.length === 0 ? (
                <p className="px-3 py-2 text-[11px] text-muted-foreground">
                  {t('buchhaltung.expenseDetail.supplierOnly')}
                </p>
              ) : (
                supplierExpenses.slice(0, 5).map((e, idx) => (
                  <div
                    key={e.id}
                    className={`flex items-center justify-between px-3 py-2 text-xs ${idx > 0 ? 'border-t border-border-muted' : ''}`}
                  >
                    <span className="flex min-w-0 items-center gap-2">
                      <span className="truncate text-foreground">{e.description}</span>
                      <span className="shrink-0 text-[10px] text-muted-foreground">
                        {new Date(e.date).toLocaleDateString(i18n.language)}
                      </span>
                    </span>
                    <span className="shrink-0 font-medium tabular-nums text-error">−{formatCurrency(e.amount)}</span>
                  </div>
                ))
              )}
            </div>
          </section>
        )}

        {/* Actions */}
        <div className="flex flex-wrap gap-2 border-t border-border pt-4">
          {expense.status === 'pending' && (
            <>
              <Button size="sm" onClick={() => { approve.mutate(expense.id); toast.success(t('buchhaltung.toast.approved')); onClose() }}>
                <CheckCircle2 className="mr-1.5 h-4 w-4" />
                {t('buchhaltung.actions.approve')}
              </Button>
              <Button variant="outline" size="sm" onClick={() => { reject.mutate(expense.id); toast.success(t('buchhaltung.toast.rejected')); onClose() }}>
                <XCircle className="mr-1.5 h-4 w-4" />
                {t('buchhaltung.actions.reject')}
              </Button>
            </>
          )}
          <Button variant="outline" size="sm" onClick={onEdit}>
            {t('buchhaltung.actions.editExpense')}
          </Button>
        </div>
      </div>

      <ReceiptPreviewDialog expense={showReceipt ? expense : null} onClose={() => setShowReceipt(false)} />
    </DetailModal>
  )
}
