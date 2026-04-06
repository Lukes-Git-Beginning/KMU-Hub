/**
 * Table of plugin execution logs with status badges.
 *
 * Shows hook type, module, entity, duration, status, and error messages.
 * Used both standalone (in Logs tab) and inside PluginDetailDialog.
 */
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useExecutionLogs } from '@/api/hooks/usePlugins'
import { Loader2 } from 'lucide-react'

interface ExecutionLogViewerProps {
  installationId?: string
  limit?: number
}

function statusBadge(status: string, t: (key: string) => string) {
  switch (status) {
    case 'success':
      return (
        <Badge className="bg-success/15 text-success border-success/30 hover:bg-success/15">
          {t('admin.plugins.logs.statusSuccess')}
        </Badge>
      )
    case 'error':
      return (
        <Badge className="bg-destructive/15 text-destructive border-destructive/30 hover:bg-destructive/15">
          {t('admin.plugins.logs.statusError')}
        </Badge>
      )
    case 'skipped':
      return (
        <Badge className="bg-warning/15 text-warning-foreground border-warning/30 hover:bg-warning/15">
          {t('admin.plugins.logs.statusSkipped')}
        </Badge>
      )
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

export function ExecutionLogViewer({
  installationId,
  limit = 50,
}: ExecutionLogViewerProps) {
  const { t } = useTranslation()
  const { data: logs, isLoading } = useExecutionLogs(installationId, limit)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!logs || logs.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4">
        {t('admin.plugins.logs.empty')}
      </p>
    )
  }

  return (
    <div className="border border-border rounded-md overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('admin.plugins.logs.hookType')}</TableHead>
            <TableHead>{t('admin.plugins.logs.module')}</TableHead>
            <TableHead>{t('admin.plugins.logs.entity')}</TableHead>
            <TableHead className="text-right">{t('admin.plugins.logs.duration')}</TableHead>
            <TableHead>{t('common.status')}</TableHead>
            <TableHead>{t('admin.plugins.logs.timestamp')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {logs.map((log) => (
            <TableRow key={log.id}>
              <TableCell className="font-mono text-xs">
                {log.hook_type}
              </TableCell>
              <TableCell className="text-sm">{log.module}</TableCell>
              <TableCell className="text-sm text-muted-foreground">
                {log.entity_type}
                {log.entity_id ? ` #${log.entity_id.slice(0, 8)}` : ''}
              </TableCell>
              <TableCell className="text-right text-sm tabular-nums">
                {log.duration_ms}ms
              </TableCell>
              <TableCell>{statusBadge(log.status, t)}</TableCell>
              <TableCell className="text-sm text-muted-foreground">
                {new Date(log.created_at).toLocaleString('de-DE')}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
