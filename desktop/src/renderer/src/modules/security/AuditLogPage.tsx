/**
 * Audit log viewer page.
 *
 * Displays a searchable, filterable table of all security-relevant actions.
 * Supports CSV/JSON export and integrity verification of the audit chain.
 */
import { useState, useCallback } from 'react'
import { FormattedMessage, FormattedDate, useIntl } from 'react-intl'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
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

  const filter: AuditFilter = {
    date_from: dateFrom || undefined,
    date_to: dateTo || undefined,
    action: actionFilter || undefined,
    result: resultFilter === 'all' ? undefined : resultFilter,
    offset: page * PAGE_SIZE,
    limit: PAGE_SIZE,
  }

  const { data, isLoading } = useAuditLog(filter)
  const exportMutation = useExportAuditLog()
  const verifyMutation = useVerifyAuditChain()

  const entries = data?.entries ?? []
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
    // Verify entire chain (from first to last visible entry)
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

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">
            <FormattedMessage id="audit.title" />
          </h2>
          <p className="text-sm text-muted-foreground">
            <FormattedMessage id="audit.description" />
            {total > 0 && (
              <span className="ml-2">
                (<FormattedMessage id="audit.totalEntries" values={{ count: total }} />)
              </span>
            )}
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleExport('csv')}
            disabled={exportMutation.isPending}
          >
            <FormattedMessage id="audit.exportCSV" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleExport('json')}
            disabled={exportMutation.isPending}
          >
            <FormattedMessage id="audit.exportJSON" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleVerifyChain}
            disabled={verifyMutation.isPending || entries.length === 0}
          >
            <FormattedMessage id="audit.verifyChain" />
          </Button>
        </div>
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap items-end gap-3 rounded-md border p-3">
        <div className="space-y-1">
          <label className="text-xs text-muted-foreground">
            <FormattedMessage id="audit.filter.dateRange" />
          </label>
          <div className="flex gap-1">
            <Input
              type="date"
              value={dateFrom}
              onChange={(e) => { setDateFrom(e.target.value); setPage(0) }}
              className="h-8 w-36"
            />
            <span className="self-center text-muted-foreground">-</span>
            <Input
              type="date"
              value={dateTo}
              onChange={(e) => { setDateTo(e.target.value); setPage(0) }}
              className="h-8 w-36"
            />
          </div>
        </div>

        <div className="space-y-1">
          <label className="text-xs text-muted-foreground">
            <FormattedMessage id="audit.filter.actionType" />
          </label>
          <Input
            type="text"
            placeholder={intl.formatMessage({ id: 'common.search' })}
            value={actionFilter}
            onChange={(e) => { setActionFilter(e.target.value); setPage(0) }}
            className="h-8 w-40"
          />
        </div>

        <div className="space-y-1">
          <label className="text-xs text-muted-foreground">
            <FormattedMessage id="audit.result" />
          </label>
          <Select
            value={resultFilter}
            onValueChange={(v) => { setResultFilter(v as 'all' | 'success' | 'failure'); setPage(0) }}
          >
            <SelectTrigger className="h-8 w-32">
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

        <Button variant="ghost" size="sm" onClick={handleResetFilters}>
          <FormattedMessage id="common.resetFilters" />
        </Button>
      </div>

      {/* Table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead><FormattedMessage id="audit.timestamp" /></TableHead>
              <TableHead><FormattedMessage id="audit.user" /></TableHead>
              <TableHead><FormattedMessage id="audit.action" /></TableHead>
              <TableHead><FormattedMessage id="audit.target" /></TableHead>
              <TableHead><FormattedMessage id="audit.ip" /></TableHead>
              <TableHead><FormattedMessage id="audit.result" /></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading && (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                  <FormattedMessage id="common.loading" />
                </TableCell>
              </TableRow>
            )}
            {!isLoading && entries.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                  <FormattedMessage id="common.noResults" />
                </TableCell>
              </TableRow>
            )}
            {entries.map((entry) => (
              <TableRow key={entry.id}>
                <TableCell className="whitespace-nowrap text-xs">
                  <FormattedDate
                    value={entry.timestamp}
                    year="numeric"
                    month="2-digit"
                    day="2-digit"
                    hour="2-digit"
                    minute="2-digit"
                    second="2-digit"
                  />
                </TableCell>
                <TableCell className="text-sm">{entry.user_name || entry.user_id}</TableCell>
                <TableCell className="text-sm font-medium">{entry.action}</TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {entry.target_type && (
                    <span className="mr-1 text-xs text-muted-foreground">
                      [{entry.target_type}]
                    </span>
                  )}
                  {entry.target}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground font-mono">
                  {entry.ip_address}
                </TableCell>
                <TableCell>
                  <Badge
                    variant={entry.result === 'success' ? 'default' : 'destructive'}
                    className="text-xs"
                  >
                    <FormattedMessage
                      id={entry.result === 'success' ? 'audit.result.success' : 'audit.result.failure'}
                    />
                  </Badge>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            {page * PAGE_SIZE + 1}-{Math.min((page + 1) * PAGE_SIZE, total)} / {total}
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={page === 0}
            >
              <FormattedMessage id="common.back" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
              disabled={page >= totalPages - 1}
            >
              <FormattedMessage id="common.next" />
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
