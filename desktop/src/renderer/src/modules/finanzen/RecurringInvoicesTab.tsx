import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Repeat, Play, Pause, Zap, CalendarClock, TrendingUp } from 'lucide-react'
import { toast } from 'sonner'
import { ItemActions, ConfirmDialog, EmptyState } from '@/components/shared'
import {
  useRecurringInvoices,
  usePauseRecurringInvoice,
  useGenerateRecurringInvoice,
  useDeleteRecurringInvoice,
} from '@/api/hooks/useFinance'
import { formatMoney, formatEUR } from '@/stores/finance'
import { formatDate } from '@/lib/format'
import type { RecurringInvoice, RecurringStatus } from '@/types/finance-types'
import { RecurringInvoiceDialog } from './RecurringInvoiceDialog'
import { RecurringDetailPanel } from './RecurringDetailPanel'
import { useHasCapability } from '@/hooks/useCapability'
import { useAmountsVisible, maskedAmount } from './lib/amounts-visibility'

const STATUS_STYLES: Record<RecurringStatus, string> = {
  active: 'bg-success-light text-success',
  paused: 'bg-warning/15 text-warning',
  ended: 'bg-secondary text-muted-foreground',
}

/** Normalise a per-invoice gross to an approximate monthly amount. */
const MONTHLY_FACTOR: Record<RecurringInvoice['interval'], number> = {
  weekly: 4.333,
  monthly: 1,
  quarterly: 1 / 3,
  yearly: 1 / 12,
}

export function RecurringInvoicesTab() {
  const { t } = useTranslation()
  const { data, isLoading } = useRecurringInvoices()
  const pauseRec = usePauseRecurringInvoice()
  const generateRec = useGenerateRecurringInvoice()
  const deleteRec = useDeleteRecurringInvoice()

  const [showForm, setShowForm] = useState(false)
  const [editRec, setEditRec] = useState<RecurringInvoice | null>(null)
  const [detailRec, setDetailRec] = useState<RecurringInvoice | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<RecurringInvoice | null>(null)

  // RBAC R-3
  const canCreateInvoice = useHasCapability('finance:invoice:create')
  const canEditInvoice   = useHasCapability('finance:invoice:edit')
  const canDeleteInvoice = useHasCapability('finance:invoice:delete')
  const amountsVisible   = useAmountsVisible()

  const recurring = data?.recurring ?? []

  const stats = useMemo(() => {
    const active = recurring.filter((r) => r.status === 'active')
    const mrr = active.reduce((sum, r) => {
      const gross = Number(r.tax_breakdown?.gross_total ?? 0)
      return sum + gross * MONTHLY_FACTOR[r.interval]
    }, 0)
    return { activeCount: active.length, mrr }
  }, [recurring])

  const handleGenerate = (r: RecurringInvoice) => {
    generateRec.mutate(r.id, {
      onSuccess: () => toast.success(t('finanzen.recurring.generated', { title: r.title })),
      onError: (err) => toast.error(err.message),
    })
  }

  const handleTogglePause = (r: RecurringInvoice) => {
    const paused = r.status === 'active'
    pauseRec.mutate(
      { id: r.id, paused },
      {
        onSuccess: () =>
          toast.success(paused ? t('finanzen.recurring.paused') : t('finanzen.recurring.resumed')),
        onError: (err) => toast.error(err.message),
      },
    )
  }

  const getActions = (r: RecurringInvoice) => {
    const actions: { label: string; onClick: () => void; variant?: 'destructive'; separator?: true }[] = []
    actions.push({ label: t('common.details'), onClick: () => setDetailRec(r) })
    if (r.status !== 'ended') {
      // Sofort erzeugen → invoice:create
      if (canCreateInvoice) {
        actions.push({ label: t('finanzen.recurring.generateNow'), onClick: () => handleGenerate(r) })
      }
      // Pause/Resume → invoice:edit
      if (canEditInvoice) {
        actions.push({
          label: r.status === 'active' ? t('finanzen.recurring.pause') : t('finanzen.recurring.resume'),
          onClick: () => handleTogglePause(r),
        })
      }
    }
    // Edit → invoice:edit
    if (canEditInvoice) {
      actions.push({ label: t('common.edit'), onClick: () => { setEditRec(r); setShowForm(true) } })
    }
    // Delete → invoice:delete
    if (canDeleteInvoice) {
      actions.push({ separator: true as const, label: '', onClick: () => {} })
      actions.push({
        label: t('common.delete'),
        variant: 'destructive' as const,
        onClick: () => setConfirmDelete(r),
      })
    }
    return actions
  }

  if (isLoading) {
    return (
      <div className="space-y-3 py-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-16 rounded-xl bg-secondary/50 animate-shimmer" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-5 animate-fade-up">
      {/* Header row */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="grid grid-cols-2 gap-3">
          <div className="rounded-xl border border-border bg-card px-4 py-3">
            <div className="flex items-center gap-2 text-muted-foreground">
              <Repeat className="h-4 w-4" />
              <span className="text-xs">{t('finanzen.recurring.activeSchedules')}</span>
            </div>
            <p className="mt-1 text-xl font-semibold text-foreground">{stats.activeCount}</p>
          </div>
          <div className="rounded-xl border border-border bg-card px-4 py-3">
            <div className="flex items-center gap-2 text-muted-foreground">
              <TrendingUp className="h-4 w-4" />
              <span className="text-xs">{t('finanzen.recurring.mrr')}</span>
            </div>
            <p
              className="mt-1 text-xl font-semibold text-foreground stat-accent"
              title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}
            >
              {maskedAmount(amountsVisible, formatEUR(stats.mrr))}
            </p>
          </div>
        </div>
        {/* Neues Abo → invoice:create */}
        {canCreateInvoice && (
          <button
            onClick={() => { setEditRec(null); setShowForm(true) }}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Repeat className="h-4 w-4" />
            {t('finanzen.recurring.createTitle')}
          </button>
        )}
      </div>

      {recurring.length === 0 ? (
        <EmptyState
          icon={Repeat}
          title={t('finanzen.recurring.emptyTitle')}
          description={t('finanzen.recurring.emptyDescription')}
          action={canCreateInvoice ? { label: t('finanzen.recurring.createTitle'), onClick: () => { setEditRec(null); setShowForm(true) } } : undefined}
        />
      ) : (
        <div className="rounded-xl border border-border bg-card overflow-hidden">
          <div className="grid grid-cols-[1fr_110px_130px_110px_120px_40px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
            <span>{t('finanzen.recurring.scheduleCol')}</span>
            <span>{t('finanzen.recurring.intervalCol')}</span>
            <span>{t('finanzen.recurring.nextRun')}</span>
            <span className="text-right">{t('finanzen.recurring.perInvoice')}</span>
            <span>{t('common.status')}</span>
            <span />
          </div>
          {recurring.map((r) => {
            const gross = Number(r.tax_breakdown?.gross_total ?? 0)
            return (
              <div
                key={r.id}
                role="button"
                tabIndex={0}
                onClick={() => setDetailRec(r)}
                onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setDetailRec(r) } }}
                className="grid cursor-pointer grid-cols-[1fr_110px_130px_110px_120px_40px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors focus-visible:bg-secondary/40 focus-visible:outline-none"
              >
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium text-foreground">{r.title}</p>
                  <p className="text-[11px] text-muted-foreground truncate">
                    {r.customer.name}
                    {r.generated_count > 0 && (
                      <span className="text-muted-foreground"> · {t('finanzen.recurring.generatedCount', { count: r.generated_count })}</span>
                    )}
                  </p>
                </div>
                <span className="flex items-center gap-1 text-xs text-muted-foreground">
                  <Repeat className="h-3 w-3" />
                  {t(`finanzen.recurring.intervals.${r.interval}`)}
                </span>
                <span className="flex items-center gap-1 text-xs text-muted-foreground">
                  <CalendarClock className="h-3 w-3" />
                  {r.status === 'ended' ? '--' : formatDate(r.next_run)}
                </span>
                <span
                  className="text-sm font-medium text-foreground text-right"
                  title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}
                >
                  {maskedAmount(amountsVisible, formatMoney(gross, r.currency ?? 'EUR'))}
                </span>
                <span className={`inline-flex w-fit items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${STATUS_STYLES[r.status]}`}>
                  {r.status === 'active' && <Play className="h-2.5 w-2.5" />}
                  {r.status === 'paused' && <Pause className="h-2.5 w-2.5" />}
                  {t(`finanzen.recurring.status.${r.status}`)}
                </span>
                <div onClick={(e) => e.stopPropagation()}>
                  <ItemActions items={getActions(r)} />
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Quick generate hint */}
      {recurring.some((r) => r.status === 'active') && (
        <p className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
          <Zap className="h-3 w-3" />
          {t('finanzen.recurring.generateHint')}
        </p>
      )}

      <RecurringInvoiceDialog open={showForm} onOpenChange={setShowForm} editRecurring={editRec} />

      {detailRec && (
        <RecurringDetailPanel
          recurring={detailRec}
          onClose={() => setDetailRec(null)}
          onEdit={() => { setEditRec(detailRec); setShowForm(true); setDetailRec(null) }}
        />
      )}

      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title={t('finanzen.recurring.deleteTitle')}
        description={t('finanzen.recurring.deleteDescription', { title: confirmDelete?.title })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={() => {
          if (!confirmDelete) return
          deleteRec.mutate(confirmDelete.id, {
            onSuccess: () => toast.success(t('finanzen.recurring.deleted')),
            onError: (err) => toast.error(err.message),
          })
          setConfirmDelete(null)
        }}
      />
    </div>
  )
}
