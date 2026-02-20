/**
 * Privacy settings tab: GDPR data export requests and deletion info.
 *
 * Users can request a data export (Art. 20 DSGVO) and see request status.
 * Admins see a link to the full GDPR erasure management page.
 * Wired to useGDPRExports and useRequestExport hooks from useSecurity.
 */
import { Link } from 'react-router-dom'
import {
  Download,
  FileText,
  Clock,
  CheckCircle,
  XCircle,
  AlertTriangle,
  ExternalLink,
} from 'lucide-react'
import { FormattedMessage, FormattedDate } from 'react-intl'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useAuthStore } from '@/stores/auth'
import { useGDPRExports, useRequestExport } from '@/api/hooks/useSecurity'

/** Map export status to badge variant and icon. */
const STATUS_CONFIG: Record<
  string,
  { icon: typeof Clock; badgeClass: string; labelId: string }
> = {
  pending: {
    icon: Clock,
    badgeClass: 'bg-yellow-100 text-yellow-800',
    labelId: 'gdpr.exportPending',
  },
  approved: {
    icon: CheckCircle,
    badgeClass: 'bg-blue-100 text-blue-800',
    labelId: 'gdpr.exportApproved',
  },
  processing: {
    icon: Clock,
    badgeClass: 'bg-blue-100 text-blue-800',
    labelId: 'gdpr.exportProcessing',
  },
  ready: {
    icon: Download,
    badgeClass: 'bg-green-100 text-green-800',
    labelId: 'gdpr.exportReady',
  },
  denied: {
    icon: XCircle,
    badgeClass: 'bg-red-100 text-red-800',
    labelId: 'gdpr.exportDenied',
  },
}

export function PrivacySettingsTab() {
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin')

  const { data: exports, isLoading } = useGDPRExports()
  const requestExport = useRequestExport()

  const handleRequestExport = () => {
    requestExport.mutate(undefined, {
      onSuccess: () => toast.success(<FormattedMessage id="gdpr.exportProcessing" />),
      onError: () => toast.error(<FormattedMessage id="common.error" />),
    })
  }

  return (
    <div className="mx-auto max-w-2xl px-6 py-6 space-y-8">
      <div>
        <h2 className="text-2xl font-semibold text-foreground">
          <FormattedMessage id="settings.privacy.title" />
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          <FormattedMessage id="settings.privacy.subtitle" />
        </p>
      </div>

      {/* Data Export */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <FileText className="h-5 w-5" />
            <FormattedMessage id="gdpr.exportTitle" />
          </CardTitle>
          <CardDescription>
            <FormattedMessage id="gdpr.exportDescription" />
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Button
            onClick={handleRequestExport}
            disabled={requestExport.isPending}
          >
            <Download className="mr-2 h-4 w-4" />
            <FormattedMessage id="gdpr.requestExport" />
          </Button>

          {/* Export requests list */}
          {isLoading ? (
            <p className="text-sm text-muted-foreground">
              <FormattedMessage id="common.loading" />
            </p>
          ) : (exports ?? []).length > 0 ? (
            <div className="space-y-2">
              {(exports ?? []).map((exp) => {
                const config = STATUS_CONFIG[exp.status] ?? STATUS_CONFIG.pending
                const StatusIcon = config.icon
                return (
                  <div
                    key={exp.id}
                    className="flex items-center gap-3 rounded-lg border border-border p-3"
                  >
                    <StatusIcon className="h-4 w-4 text-muted-foreground shrink-0" />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-foreground">
                        <FormattedMessage id={config.labelId} />
                      </p>
                      <p className="text-xs text-muted-foreground">
                        <FormattedDate
                          value={exp.requested_at}
                          year="numeric"
                          month="short"
                          day="numeric"
                          hour="numeric"
                          minute="numeric"
                        />
                      </p>
                    </div>
                    <Badge variant="secondary" className={config.badgeClass}>
                      <FormattedMessage id={config.labelId} />
                    </Badge>
                    {exp.status === 'ready' && exp.download_token && (
                      <Button
                        variant="outline"
                        size="sm"
                        asChild
                      >
                        <a href={`/api/v1/gdpr/exports/${exp.id}/download?token=${exp.download_token}`} download>
                          <Download className="mr-1 h-3 w-3" />
                          <FormattedMessage id="gdpr.downloadExport" />
                        </a>
                      </Button>
                    )}
                  </div>
                )
              })}
            </div>
          ) : null}
        </CardContent>
      </Card>

      {/* Data Deletion Info */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <AlertTriangle className="h-5 w-5" />
            <FormattedMessage id="gdpr.deleteTitle" />
          </CardTitle>
          <CardDescription>
            <FormattedMessage id="gdpr.deleteDescription" />
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            <FormattedMessage id="settings.privacy.deletionInfo" />
          </p>

          {isAdmin && (
            <Link
              to="/admin/security"
              className="inline-flex items-center gap-1 text-sm text-primary hover:underline"
            >
              <FormattedMessage id="settings.privacy.adminErasureLink" />
              <ExternalLink className="h-3 w-3" />
            </Link>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
