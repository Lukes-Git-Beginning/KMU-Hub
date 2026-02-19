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
  Clock,
  Download,
  Filter,
} from 'lucide-react'
import { FormattedMessage, FormattedDate } from 'react-intl'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
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
  pending: { className: 'bg-yellow-100 text-yellow-800', labelId: 'gdpr.exportPending' },
  approved: { className: 'bg-blue-100 text-blue-800', labelId: 'gdpr.exportApproved' },
  processing: { className: 'bg-blue-100 text-blue-800', labelId: 'gdpr.exportProcessing' },
  ready: { className: 'bg-green-100 text-green-800', labelId: 'gdpr.exportReady' },
  denied: { className: 'bg-red-100 text-red-800', labelId: 'gdpr.exportDenied' },
}

export default function GDPRExportPage() {
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin')

  const { data: exports, isLoading } = useGDPRExports()
  const approveExport = useApproveExport()
  const denyExport = useDenyExport()

  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [denyId, setDenyId] = useState<string | null>(null)
  const [denyReason, setDenyReason] = useState('')

  const filteredExports = (exports ?? []).filter((exp) => {
    if (statusFilter === 'all') return true
    return exp.status === statusFilter
  })

  const handleApprove = useCallback(
    (id: string) => {
      approveExport.mutate({ id }, {
        onSuccess: () => toast.success(<FormattedMessage id="gdpr.exportApproved" />),
        onError: () => toast.error(<FormattedMessage id="common.error" />),
      })
    },
    [approveExport]
  )

  const handleDeny = useCallback(() => {
    if (!denyId || !denyReason.trim()) return
    denyExport.mutate(
      { id: denyId, note: denyReason.trim() },
      {
        onSuccess: () => {
          toast.success(<FormattedMessage id="gdpr.exportDenied" />)
          setDenyId(null)
          setDenyReason('')
        },
        onError: () => toast.error(<FormattedMessage id="common.error" />),
      }
    )
  }, [denyId, denyReason, denyExport])

  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-4xl px-6 py-6 space-y-6">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">
            <FormattedMessage id="gdpr.admin.title" />
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            <FormattedMessage id="gdpr.admin.description" />
          </p>
        </div>

        {/* Status Filter */}
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          {FILTER_OPTIONS.map((opt) => (
            <Button
              key={opt.key}
              variant={statusFilter === opt.key ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => setStatusFilter(opt.key)}
            >
              <FormattedMessage id={opt.labelId} />
            </Button>
          ))}
        </div>

        {/* Exports Table */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <FileText className="h-5 w-5" />
              <FormattedMessage id="gdpr.admin.title" />
            </CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <p className="text-sm text-muted-foreground py-8 text-center">
                <FormattedMessage id="common.loading" />
              </p>
            ) : filteredExports.length === 0 ? (
              <p className="text-sm text-muted-foreground py-8 text-center">
                <FormattedMessage id="common.noResults" />
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-border text-left">
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="gdpr.admin.requestedBy" />
                      </th>
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="common.status" />
                      </th>
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="gdpr.admin.requestedAt" />
                      </th>
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="gdpr.admin.reviewedBy" />
                      </th>
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="common.actions" />
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {filteredExports.map((exp) => {
                      const badge = STATUS_BADGE[exp.status] ?? STATUS_BADGE.pending
                      return (
                        <tr key={exp.id}>
                          <td className="py-3 text-sm text-foreground">
                            {exp.user_id}
                          </td>
                          <td className="py-3">
                            <Badge variant="secondary" className={badge.className}>
                              <FormattedMessage id={badge.labelId} />
                            </Badge>
                          </td>
                          <td className="py-3 text-sm text-muted-foreground">
                            <FormattedDate
                              value={exp.requested_at}
                              year="numeric"
                              month="short"
                              day="numeric"
                            />
                          </td>
                          <td className="py-3 text-sm text-muted-foreground">
                            {exp.reviewed_by ?? '-'}
                          </td>
                          <td className="py-3">
                            {exp.status === 'pending' && (
                              <div className="flex gap-2">
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => handleApprove(exp.id)}
                                  disabled={approveExport.isPending}
                                >
                                  <CheckCircle className="mr-1 h-3 w-3" />
                                  <FormattedMessage id="gdpr.admin.approve" />
                                </Button>
                                <Button
                                  variant="outline"
                                  size="sm"
                                  className="text-destructive"
                                  onClick={() => setDenyId(exp.id)}
                                >
                                  <XCircle className="mr-1 h-3 w-3" />
                                  <FormattedMessage id="gdpr.admin.deny" />
                                </Button>
                              </div>
                            )}
                            {exp.status === 'ready' && exp.download_token && (
                              <Button variant="outline" size="sm" asChild>
                                <a href={`/api/v1/gdpr/exports/${exp.id}/download?token=${exp.download_token}`} download>
                                  <Download className="mr-1 h-3 w-3" />
                                  <FormattedMessage id="gdpr.downloadExport" />
                                </a>
                              </Button>
                            )}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Deny Reason Dialog */}
        <Dialog open={denyId !== null} onOpenChange={(open) => { if (!open) { setDenyId(null); setDenyReason('') } }}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                <FormattedMessage id="gdpr.admin.deny" />
              </DialogTitle>
              <DialogDescription>
                <FormattedMessage id="gdpr.admin.denyReason" />
              </DialogDescription>
            </DialogHeader>
            <div className="py-4">
              <Input
                value={denyReason}
                onChange={(e) => setDenyReason(e.target.value)}
                placeholder="Reason for denial..."
              />
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => { setDenyId(null); setDenyReason('') }}>
                <FormattedMessage id="common.cancel" />
              </Button>
              <Button
                variant="destructive"
                onClick={handleDeny}
                disabled={!denyReason.trim() || denyExport.isPending}
              >
                <FormattedMessage id="gdpr.admin.deny" />
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  )
}
