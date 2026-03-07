/**
 * Audit log viewer page.
 *
 * Displays a searchable, filterable table of all security-relevant actions.
 * Supports CSV/JSON export and integrity verification of the audit chain.
 */
import { useState, useCallback, useMemo } from 'react'
import { FormattedMessage, FormattedDate, useIntl } from 'react-intl'
import { toast } from 'sonner'
import {
  Shield,
  Download,
  ShieldCheck,
  Search,
  Filter,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  useAuditLog,
  useExportAuditLog,
  useVerifyAuditChain,
} from '@/api/hooks/useSecurity'
import type { AuditFilter, AuditExportFormat } from '@/api/security-types'

const PAGE_SIZE = 25

const RESULT_OPTIONS = [
  { value: 'all', labelId: 'common.filter' },
  { value: 'success', labelId: 'audit.result.success' },
  { value: 'failure', labelId: 'audit.result.failure' },
] as const

export default function AuditLogPage() {
  const intl = useIntl()

  // Filter state
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [actionFilter, setActionFilter] = useState('')
  const [resultFilter, setResultFilter] = useState<'all' | 'success' | 'failure'>('all')
  const [page, setPage] = useState(0)

  const filter: AuditFilter = useMemo(() => ({
    date_from: dateFrom || undefined,
    date_to: dateTo || undefined,
    action: actionFilter || undefined,
    result: resultFilter === 'all' ? undefined : resultFilter,
    offset: page * PAGE_SIZE,
    limit: PAGE_SIZE,
  }), [dateFrom, dateTo, actionFilter, resultFilter, page])

  const { data, isLoading } = useAuditLog(filter)
  const exportMutation = useExportAuditLog()
  const verifyMutation = useVerifyAuditChain()

  const entries = useMemo(() => data?.entries ?? [], [data?.entries])
  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  const handleExport = useCallback(
    (format: AuditExportFormat) => {
      const { offset: _offset, limit: _limit, ...filterWithoutPaging } = filter
      exportMutation.mutate(
        { filter: filterWithoutPaging, format },
        {
          onError: (err) => {
            toast.error(err instanceof Error ? err.message : intl.formatMessage({ id: 'common.error' }))
          },
        },
      )
    },
    [filter, exportMutation, intl],
  )

  const handleVerifyChain = useCallback(() => {
    const fromSeq = entries.length > 0 ? entries[entries.length - 1].sequence_num : 0
    const toSeq = entries.length > 0 ? entries[0].sequence_num : 0
    if (fromSeq === 0 && toSeq === 0) return

    verifyMutation.mutate(
      { fromSeq, toSeq },
      {
        onSuccess: (result) => {
          if (result.valid) {
            toast.success(intl.formatMessage({ id: 'audit.chainValid' }))
          } else {
            toast.error(
              intl.formatMessage(
                { id: 'audit.chainBroken' },
                { sequence: result.broken_at_sequence ?? 0 },
              ),
            )
          }
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : intl.formatMessage({ id: 'common.error' }))
        },
      },
    )
  }, [entries, verifyMutation, intl])

  const handleResetFilters = useCallback(() => {
    setDateFrom('')
    setDateTo('')
    setActionFilter('')
    setResultFilter('all')
    setPage(0)
  }, [])

  const hasActiveFilters = dateFrom || dateTo || actionFilter || resultFilter !== 'all'

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div className="flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-[#f0e8f5] dark:bg-violet-900/30">
            <Shield className="h-6 w-6 text-violet-600 dark:text-violet-400" />
          </div>
          <div>
            <h1 className="text-foreground">
              <FormattedMessage id="audit.title" />
            </h1>
            <div className="flex items-center gap-2 mt-1">
              {total > 0 && (
                <span className="rounded-full bg-secondary px-2.5 py-0.5 text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="audit.totalEntries" values={{ count: total }} />
                </span>
              )}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => handleExport('csv')}
            disabled={exportMutation.isPending}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors disabled:opacity-50"
          >
            <Download className="h-3.5 w-3.5" />
            <FormattedMessage id="audit.exportCSV" />
          </button>
          <button
            onClick={() => handleExport('json')}
            disabled={exportMutation.isPending}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors disabled:opacity-50"
          >
            <Download className="h-3.5 w-3.5" />
            <FormattedMessage id="audit.exportJSON" />
          </button>
          <button
            onClick={handleVerifyChain}
            disabled={verifyMutation.isPending || entries.length === 0}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
          >
            <ShieldCheck className="h-3.5 w-3.5" />
            <FormattedMessage id="audit.verifyChain" />
          </button>
        </div>
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap items-end gap-3 mb-4">
        {/* Date range */}
        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">
            <FormattedMessage id="audit.filter.dateRange" />
          </label>
          <div className="flex items-center gap-1.5">
            <input
              type="date"
              value={dateFrom}
              onChange={(e) => { setDateFrom(e.target.value); setPage(0) }}
              className="h-9 w-36 rounded-lg border border-border bg-card px-3 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
            <span className="text-muted-foreground text-xs">-</span>
            <input
              type="date"
              value={dateTo}
              onChange={(e) => { setDateTo(e.target.value); setPage(0) }}
              className="h-9 w-36 rounded-lg border border-border bg-card px-3 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
        </div>

        {/* Action search */}
        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">
            <FormattedMessage id="audit.filter.actionType" />
          </label>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              type="text"
              placeholder={intl.formatMessage({ id: 'common.search' })}
              value={actionFilter}
              onChange={(e) => { setActionFilter(e.target.value); setPage(0) }}
              className="h-9 w-40 rounded-lg border border-border bg-card pl-9 pr-3 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
        </div>

        {/* Result filter */}
        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">
            <FormattedMessage id="audit.result" />
          </label>
          <Select
            value={resultFilter}
            onValueChange={(v) => { setResultFilter(v as 'all' | 'success' | 'failure'); setPage(0) }}
          >
            <SelectTrigger className="h-9 w-36 rounded-lg border-border bg-card text-sm">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {RESULT_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {intl.formatMessage({ id: opt.labelId })}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Reset */}
        {hasActiveFilters && (
          <button
            onClick={handleResetFilters}
            className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors h-9"
          >
            <Filter className="h-3.5 w-3.5" />
            <FormattedMessage id="common.resetFilters" />
          </button>
        )}
      </div>

      {/* Table */}
      <div className="rounded-lg border border-border bg-card overflow-hidden glass-surface">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="audit.timestamp" />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="audit.user" />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="audit.action" />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="audit.target" />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="audit.ip" />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="audit.result" />
                </th>
              </tr>
            </thead>
            <tbody>
              {isLoading && (
                <tr>
                  <td colSpan={6} className="text-center py-12 text-muted-foreground">
                    <FormattedMessage id="common.loading" />
                  </td>
                </tr>
              )}
              {!isLoading && entries.length === 0 && (
                <tr>
                  <td colSpan={6} className="text-center py-12">
                    <Shield className="mx-auto h-10 w-10 text-muted-foreground/40 mb-2" />
                    <p className="text-sm text-muted-foreground">
                      <FormattedMessage id="common.noResults" />
                    </p>
                  </td>
                </tr>
              )}
              {entries.map((entry) => (
                <tr
                  key={entry.id}
                  className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors"
                >
                  <td className="px-4 py-3 whitespace-nowrap text-xs text-muted-foreground">
                    <FormattedDate
                      value={entry.timestamp}
                      year="numeric"
                      month="2-digit"
                      day="2-digit"
                      hour="2-digit"
                      minute="2-digit"
                      second="2-digit"
                    />
                  </td>
                  <td className="px-4 py-3 text-sm text-foreground">
                    {entry.user_name || entry.user_id}
                  </td>
                  <td className="px-4 py-3 text-sm font-medium text-foreground">
                    {entry.action}
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">
                    {entry.target_type && (
                      <span className="rounded-full bg-secondary px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground mr-1.5">
                        {entry.target_type}
                      </span>
                    )}
                    {entry.target}
                  </td>
                  <td className="px-4 py-3 text-xs text-muted-foreground font-mono">
                    {entry.ip_address}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                        entry.result === 'success'
                          ? 'bg-success-light text-success'
                          : 'bg-error-light text-error'
                      }`}
                    >
                      <FormattedMessage
                        id={entry.result === 'success' ? 'audit.result.success' : 'audit.result.failure'}
                      />
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between mt-4">
          <p className="text-sm text-muted-foreground">
            {page * PAGE_SIZE + 1}-{Math.min((page + 1) * PAGE_SIZE, total)} / {total}
          </p>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={page === 0}
              className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-sm text-foreground hover:bg-secondary transition-colors disabled:opacity-40"
            >
              <ChevronLeft className="h-3.5 w-3.5" />
              <FormattedMessage id="common.back" />
            </button>
            <button
              onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
              disabled={page >= totalPages - 1}
              className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-sm text-foreground hover:bg-secondary transition-colors disabled:opacity-40"
            >
              <FormattedMessage id="common.next" />
              <ChevronRight className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
