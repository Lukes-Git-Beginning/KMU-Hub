/**
 * Admin page for managing GDPR data export requests.
 *
 * Shows all user export requests with status filtering.
 * Admins can approve or deny pending requests (deny requires a reason).
 * Wired to useGDPRExports, useApproveExport, useDenyExport hooks.
 */
import { useState, useCallback } from 'react'
import { Navigate } from 'react-router-dom'
import {
  FileText,
  CheckCircle,
  XCircle,
  Download,
  X,
  Clock,
  Info,
} from 'lucide-react'
import { FormattedMessage, FormattedDate, useIntl } from 'react-intl'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth'
import { useGDPRExports, useApproveExport, useDenyExport } from '@/api/hooks/useSecurity'

type StatusFilter = 'all' | 'pending' | 'approved' | 'ready'

const FILTER_OPTIONS: { key: StatusFilter; labelId: string }[] = [
  { key: 'all', labelId: 'gdpr.admin.filterAll' },
  { key: 'pending', labelId: 'gdpr.admin.filterPending' },
  { key: 'approved', labelId: 'gdpr.admin.filterApproved' },
  { key: 'ready', labelId: 'gdpr.admin.filterReady' },
]

const STATUS_BADGE: Record<string, { className: string; labelId: string }> = {
  pending: { className: 'bg-warning-light text-warning', labelId: 'gdpr.exportPending' },
  approved: { className: 'bg-info-light text-info', labelId: 'gdpr.exportApproved' },
  processing: { className: 'bg-info-light text-info', labelId: 'gdpr.exportProcessing' },
  ready: { className: 'bg-success-light text-success', labelId: 'gdpr.exportReady' },
  denied: { className: 'bg-error-light text-error', labelId: 'gdpr.exportDenied' },
}

export default function GDPRExportPage() {
  const intl = useIntl()
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin')

  const { data: exports, isLoading } = useGDPRExports()
  const approveExport = useApproveExport()
  const denyExport = useDenyExport()

  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [denyId, setDenyId] = useState<string | null>(null)
  const [denyReason, setDenyReason] = useState('')
  const [exportFormats, setExportFormats] = useState<Record<string, string>>({})

  const getExportFormat = (id: string) => exportFormats[id] ?? 'json'
  const setExportFormat = (id: string, fmt: string) =>
    setExportFormats((prev) => ({ ...prev, [id]: fmt }))

  const allExports = exports ?? []
  const filteredExports = allExports.filter((exp) => {
    if (statusFilter === 'all') return true
    return exp.status === statusFilter
  })

  const pendingCount = allExports.filter((e) => e.status === 'pending').length
  const readyCount = allExports.filter((e) => e.status === 'ready').length

  const handleApprove = useCallback(
    (id: string) => {
      approveExport.mutate({ id }, {
        onSuccess: () => toast.success(intl.formatMessage({ id: 'gdpr.exportApproved' })),
        onError: () => toast.error(intl.formatMessage({ id: 'common.error' })),
      })
    },
    [approveExport, intl],
  )

  const handleDeny = useCallback(() => {
    if (!denyId || !denyReason.trim()) return
    denyExport.mutate(
      { id: denyId, note: denyReason.trim() },
      {
        onSuccess: () => {
          toast.success(intl.formatMessage({ id: 'gdpr.exportDenied' }))
          setDenyId(null)
          setDenyReason('')
        },
        onError: () => toast.error(intl.formatMessage({ id: 'common.error' })),
      },
    )
  }, [denyId, denyReason, denyExport, intl])

  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div className="flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-success-light">
            <FileText className="h-6 w-6 text-success" />
          </div>
          <div>
            <h1 className="text-foreground">
              <FormattedMessage id="gdpr.admin.title" />
            </h1>
            <div className="flex items-center gap-2 mt-1">
              {pendingCount > 0 && (
                <span className="rounded-full bg-warning-light px-2.5 py-0.5 text-xs font-medium text-warning">
                  {pendingCount} Pending
                </span>
              )}
              {readyCount > 0 && (
                <span className="rounded-full bg-success-light px-2.5 py-0.5 text-xs font-medium text-success">
                  {readyCount} Ready
                </span>
              )}
              <span className="rounded-full bg-secondary px-2.5 py-0.5 text-xs font-medium text-muted-foreground">
                {allExports.length} Total
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Scope Info */}
      <div className="flex items-start gap-2 rounded-lg bg-info-light/50 px-4 py-3 mb-6">
        <Info className="h-4 w-4 text-info mt-0.5 shrink-0" />
        <div className="text-xs text-info">
          <p className="font-medium mb-1">Enthaltene Module im Export:</p>
          <p>CRM, E-Mails, Chat, Kalender, Dokumente, Helpdesk, Rechnungen, Projekte, Formulare, Zeiterfassung</p>
        </div>
      </div>

      {/* Status Filter Tabs */}
      <div className="flex items-center gap-1 rounded-lg border border-border p-1 mb-6 w-fit">
        {FILTER_OPTIONS.map((opt) => (
          <button
            key={opt.key}
            onClick={() => setStatusFilter(opt.key)}
            className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
              statusFilter === opt.key
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-secondary'
            }`}
          >
            <FormattedMessage id={opt.labelId} />
          </button>
        ))}
      </div>

      {/* Exports Table */}
      <div className="rounded-lg border border-border bg-card overflow-hidden glass-surface">
        <div className="overflow-x-auto">
          {isLoading ? (
            <div className="py-12 text-center">
              <p className="text-sm text-muted-foreground">
                <FormattedMessage id="common.loading" />
              </p>
            </div>
          ) : filteredExports.length === 0 ? (
            <div className="py-12 text-center">
              <FileText className="mx-auto h-10 w-10 text-muted-foreground/40 mb-2" />
              <p className="text-sm text-muted-foreground">
                <FormattedMessage id="common.noResults" />
              </p>
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border">
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                    <FormattedMessage id="gdpr.admin.requestedBy" />
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                    <FormattedMessage id="common.status" />
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                    <FormattedMessage id="gdpr.admin.requestedAt" />
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                    <FormattedMessage id="gdpr.admin.reviewedBy" />
                  </th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">
                    <FormattedMessage id="common.actions" />
                  </th>
                </tr>
              </thead>
              <tbody>
                {filteredExports.map((exp) => {
                  const badge = STATUS_BADGE[exp.status] ?? STATUS_BADGE.pending
                  return (
                    <tr
                      key={exp.id}
                      className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors"
                    >
                      <td className="px-4 py-3 text-sm text-foreground">
                        {exp.user_id}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${badge.className}`}>
                          <FormattedMessage id={badge.labelId} />
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-muted-foreground">
                        <FormattedDate
                          value={exp.requested_at}
                          year="numeric"
                          month="short"
                          day="numeric"
                        />
                      </td>
                      <td className="px-4 py-3 text-sm text-muted-foreground">
                        {exp.reviewed_by ?? '-'}
                      </td>
                      <td className="px-4 py-3 text-right">
                        {exp.status === 'pending' && (
                          <div className="flex justify-end items-center gap-2">
                            <select
                              value={getExportFormat(exp.id)}
                              onChange={(e) => setExportFormat(exp.id, e.target.value)}
                              className="h-7 rounded-md border border-border bg-transparent px-1.5 text-[10px] outline-none focus:border-primary"
                            >
                              <option value="json">JSON</option>
                              <option value="csv">CSV</option>
                              <option value="pdf">PDF</option>
                            </select>
                            <button
                              onClick={() => handleApprove(exp.id)}
                              disabled={approveExport.isPending}
                              className="flex items-center gap-1 rounded-lg border border-border px-2.5 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors disabled:opacity-40"
                            >
                              <CheckCircle className="h-3 w-3 text-success" />
                              <FormattedMessage id="gdpr.admin.approve" />
                            </button>
                            <button
                              onClick={() => setDenyId(exp.id)}
                              className="flex items-center gap-1 rounded-lg border border-border px-2.5 py-1.5 text-xs text-error hover:bg-error-light transition-colors"
                            >
                              <XCircle className="h-3 w-3" />
                              <FormattedMessage id="gdpr.admin.deny" />
                            </button>
                          </div>
                        )}
                        {exp.status === 'ready' && exp.download_token && (
                          <div className="flex justify-end items-center gap-2">
                            <span className="flex items-center gap-1 text-[10px] text-warning">
                              <Clock className="h-3 w-3" />
                              Ablauf in 30 Tagen
                            </span>
                            <a
                              href={`/api/v1/gdpr/exports/${exp.id}/download?token=${exp.download_token}`}
                              download
                              className="inline-flex items-center gap-1 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
                            >
                              <Download className="h-3 w-3" />
                              <FormattedMessage id="gdpr.downloadExport" />
                            </a>
                          </div>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Deny Reason Dialog */}
      {denyId !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => { setDenyId(null); setDenyReason('') }} />
          <div className="relative z-10 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated">
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-lg font-semibold text-foreground">
                <FormattedMessage id="gdpr.admin.deny" />
              </h2>
              <button
                onClick={() => { setDenyId(null); setDenyReason('') }}
                className="rounded-lg p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <p className="text-sm text-muted-foreground mb-4">
              <FormattedMessage id="gdpr.admin.denyReason" />
            </p>

            <input
              value={denyReason}
              onChange={(e) => setDenyReason(e.target.value)}
              placeholder="Reason for denial..."
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              autoFocus
            />

            <div className="flex justify-end gap-2 mt-6">
              <button
                onClick={() => { setDenyId(null); setDenyReason('') }}
                className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <FormattedMessage id="common.cancel" />
              </button>
              <button
                onClick={handleDeny}
                disabled={!denyReason.trim() || denyExport.isPending}
                className="rounded-lg bg-error px-4 py-2 text-sm text-white hover:bg-error/90 transition-colors disabled:opacity-50"
              >
                <FormattedMessage id="gdpr.admin.deny" />
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
