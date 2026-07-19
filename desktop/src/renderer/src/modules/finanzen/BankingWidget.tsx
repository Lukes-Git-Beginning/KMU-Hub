/**
 * BankingWidget — FinAPI bank connection placeholder + transaction matching.
 *
 * Shows bank account connection card, recent transactions, and
 * automatic invoice-to-payment matching UI.
 * Mock data for design — backend: FinAPI integration.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Landmark,
  Link2,
  RefreshCw,
  Check,
  X,
  ArrowUpRight,
  ArrowDownRight,
  Search,
  CheckCircle2,
  HelpCircle,
  Zap,
  Loader2,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/format'
import {
  useBankAccounts,
  useBankTransactions,
  useMatchTransaction,
  useRejectMatch,
} from '@/api/hooks/useFinance'
import type { BankAccount, BankTransaction } from '@/types/finance-types'
import { BankTransactionDetailPanel } from './BankTransactionDetailPanel'
import { BankConnectDialog } from './BankConnectDialog'
import { useHasCapability } from '@/hooks/useCapability'
import { useAmountsVisible, maskedAmount } from './lib/amounts-visibility'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function BankingWidget() {
  const { t } = useTranslation()
  const [filter, setFilter] = useState<'all' | 'matched' | 'suggested' | 'unmatched'>('all')

  const { data: accounts = [], isLoading: accountsLoading } = useBankAccounts()
  const { data: transactions = [], isLoading: txLoading } = useBankTransactions()
  const matchMutation = useMatchTransaction()
  const rejectMutation = useRejectMatch()
  const [syncing, setSyncing] = useState(false)
  const [detailTx, setDetailTx] = useState<BankTransaction | null>(null)
  const [connectAcc, setConnectAcc] = useState<BankAccount | null>(null)

  // RBAC R-3
  const canSettings    = useHasCapability('finance:settings:manage')
  const canBook        = useHasCapability('finance:incoming:book')
  const amountsVisible = useAmountsVisible()

  const connectedAccount = accounts.find((a) => a.connected)

  const handleSync = () => {
    setSyncing(true)
    setTimeout(() => {
      setSyncing(false)
      toast.success(t('finanzen.banking.balanceUpdated'))
    }, 2000)
  }

  const handleAcceptMatch = (txId: string) => {
    matchMutation.mutate(
      { id: txId },
      {
        onSuccess: () => toast.success(t('finanzen.banking.matchConfirmed')),
        onError: (err) => toast.error(err.message),
      },
    )
  }

  const handleRejectMatch = (txId: string) => {
    rejectMutation.mutate(txId, {
      onSuccess: () => toast.success(t('finanzen.banking.matchRejected')),
      onError: (err) => toast.error(err.message),
    })
  }

  const filteredTx = transactions.filter((tx) => {
    if (filter === 'all') return true
    return tx.matchStatus === filter
  })

  const matchedCount = transactions.filter((t) => t.matchStatus === 'matched').length
  const suggestedCount = transactions.filter((t) => t.matchStatus === 'suggested').length
  const unmatchedCount = transactions.filter((t) => t.matchStatus === 'unmatched').length

  const formatEUR = (v: number) =>
    new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' }).format(v)

  if (accountsLoading || txLoading) {
    return (
      <div className="flex items-center justify-center py-16 text-muted-foreground">
        <Loader2 className="h-5 w-5 animate-spin" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Bank accounts */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {accounts.map((acc) => (
          <div
            key={acc.id}
            className={`rounded-lg border p-4 ${
              acc.connected ? 'border-success/30 bg-success-light/10' : 'border-border'
            }`}
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${
                  acc.connected ? 'bg-success-light' : 'bg-secondary'
                }`}>
                  <Landmark className={`h-5 w-5 ${acc.connected ? 'text-success' : 'text-muted-foreground'}`} />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">{acc.bankName}</p>
                  <p className="text-[10px] text-muted-foreground font-mono">{acc.iban}</p>
                </div>
              </div>
              {acc.connected ? (
                <CheckCircle2 className="h-4 w-4 text-success" />
              ) : (
                /* Connect → settings:manage */
                canSettings && (
                  <button
                    onClick={() => setConnectAcc(acc)}
                    className="flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
                  >
                    <Link2 className="h-3 w-3" />
                    {t('finanzen.banking.connect')}
                  </button>
                )
              )}
            </div>

            {acc.connected && (
              <div className="flex items-end justify-between">
                <div>
                  <p className="text-[10px] text-muted-foreground">{t('finanzen.banking.balance')}</p>
                  <p
                    className="text-lg font-semibold text-foreground"
                    title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}
                  >
                    {maskedAmount(amountsVisible, formatEUR(acc.balance))}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {acc.lastSync && (
                    <span className="text-[9px] text-muted-foreground">
                      {t('finanzen.banking.lastSync')}: {formatDate(acc.lastSync)}
                    </span>
                  )}
                  {/* Sync → incoming:book */}
                  {canBook && (
                    <button
                      onClick={handleSync}
                      disabled={syncing}
                      className="rounded-md border border-border p-1.5 text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
                    >
                      <RefreshCw className={`h-3.5 w-3.5 ${syncing ? 'animate-spin' : ''}`} />
                    </button>
                  )}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* FinAPI info */}
      <div className="flex items-start gap-2 rounded-md bg-secondary/50 px-3 py-2">
        <Zap className="h-3.5 w-3.5 text-muted-foreground mt-0.5 shrink-0" />
        <p className="text-[11px] text-muted-foreground">
          {t('finanzen.banking.finapiInfo')}
        </p>
      </div>

      {/* Transaction matching */}
      {connectedAccount && (
        <>
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium text-foreground">{t('finanzen.banking.transactionMatching')}</h3>
            <div className="flex items-center gap-1.5">
              {([
                ['all', `${t('finanzen.banking.filterAll')} (${transactions.length})`],
                ['matched', `${t('finanzen.banking.filterMatched')} (${matchedCount})`],
                ['suggested', `${t('finanzen.banking.filterSuggested')} (${suggestedCount})`],
                ['unmatched', `${t('finanzen.banking.filterUnmatched')} (${unmatchedCount})`],
              ] as const).map(([key, label]) => (
                <button
                  key={key}
                  onClick={() => setFilter(key)}
                  className={`rounded-md px-2 py-1 text-[10px] transition-colors ${
                    filter === key
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-secondary text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>

          <div className="rounded-lg border border-border overflow-hidden">
            <div className="grid grid-cols-[80px_1fr_120px_90px_100px_70px] gap-2 items-center bg-secondary/30 px-3 py-2 text-[10px] font-medium text-muted-foreground">
              <span>{t('finanzen.banking.date')}</span>
              <span>{t('finanzen.banking.description')}</span>
              <span>{t('finanzen.banking.counterpart')}</span>
              <span className="text-right">{t('finanzen.banking.amount')}</span>
              <span>{t('finanzen.banking.assignment')}</span>
              <span />
            </div>
            {filteredTx.map((tx) => (
              <div
                key={tx.id}
                role="button"
                tabIndex={0}
                onClick={() => setDetailTx(tx)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setDetailTx(tx) }
                }}
                className={`grid cursor-pointer grid-cols-[80px_1fr_120px_90px_100px_70px] gap-2 items-center px-3 py-2.5 border-t border-border transition-colors hover:bg-accent/30 focus-visible:bg-accent/30 focus-visible:outline-none ${
                  tx.matchStatus === 'suggested' ? 'bg-warning-light/10' : ''
                }`}
              >
                <span className="text-xs text-muted-foreground">
                  {formatDate(tx.date)}
                </span>
                <span className="text-xs text-foreground truncate">{tx.description}</span>
                <span className="text-xs text-muted-foreground truncate">{tx.counterpart}</span>
                <span
                  className={`text-xs font-medium text-right flex items-center justify-end gap-1 ${
                    tx.type === 'credit' ? 'text-success' : 'text-foreground'
                  }`}
                  title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}
                >
                  {tx.type === 'credit' ? (
                    <ArrowDownRight className="h-3 w-3" />
                  ) : (
                    <ArrowUpRight className="h-3 w-3 text-muted-foreground" />
                  )}
                  {maskedAmount(amountsVisible, formatEUR(Math.abs(tx.amount)))}
                </span>
                <div>
                  {tx.matchStatus === 'matched' && (
                    <span className="flex items-center gap-1 text-[10px] text-success font-medium">
                      <CheckCircle2 className="h-3 w-3" />
                      {tx.matchedInvoice}
                    </span>
                  )}
                  {tx.matchStatus === 'suggested' && (
                    <span className="flex items-center gap-1 text-[10px] text-warning font-medium">
                      <HelpCircle className="h-3 w-3" />
                      {tx.matchedInvoice}?
                    </span>
                  )}
                  {tx.matchStatus === 'unmatched' && (
                    <span className="text-[10px] text-muted-foreground">—</span>
                  )}
                </div>
                <div className="flex items-center gap-1">
                  {/* Match accept/reject → incoming:book */}
                  {tx.matchStatus === 'suggested' && canBook && (
                    <>
                      <button
                        onClick={(e) => { e.stopPropagation(); handleAcceptMatch(tx.id) }}
                        className="rounded p-1 text-success hover:bg-success-light transition-colors"
                        title={t('finanzen.banking.confirmMatch')}
                      >
                        <Check className="h-3 w-3" />
                      </button>
                      <button
                        onClick={(e) => { e.stopPropagation(); handleRejectMatch(tx.id) }}
                        className="rounded p-1 text-muted-foreground hover:bg-accent transition-colors"
                        title={t('finanzen.banking.rejectMatch')}
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </>
                  )}
                  {tx.matchStatus === 'unmatched' && tx.type === 'credit' && canBook && (
                    <button
                      onClick={(e) => { e.stopPropagation(); setDetailTx(tx) }}
                      className="rounded p-1 text-primary hover:bg-primary/10 transition-colors"
                      title={t('finanzen.banking.manualMatch')}
                    >
                      <Search className="h-3 w-3" />
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      {detailTx && (
        <BankTransactionDetailPanel transaction={detailTx} onClose={() => setDetailTx(null)} />
      )}

      {connectAcc && (
        <BankConnectDialog account={connectAcc} open={!!connectAcc} onClose={() => setConnectAcc(null)} />
      )}
    </div>
  )
}
