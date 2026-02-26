/**
 * Table of plugin execution logs with status badges.
 *
 * Shows hook type, module, entity, duration, status, and error messages.
 * Used both standalone (in Logs tab) and inside PluginDetailDialog.
 */
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

function statusBadge(status: string) {
  switch (status) {
    case 'success':
      return (
        <Badge className="bg-green-500/15 text-green-700 border-green-500/30 hover:bg-green-500/15">
          Erfolg
        </Badge>
      )
    case 'error':
      return (
        <Badge className="bg-red-500/15 text-red-700 border-red-500/30 hover:bg-red-500/15">
          Fehler
        </Badge>
      )
    case 'skipped':
      return (
        <Badge className="bg-yellow-500/15 text-yellow-700 border-yellow-500/30 hover:bg-yellow-500/15">
          Uebersprungen
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
        Keine Ausfuehrungsprotokolle vorhanden.
      </p>
    )
  }

  return (
    <div className="border border-border rounded-md overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Hook-Typ</TableHead>
            <TableHead>Modul</TableHead>
            <TableHead>Entitaet</TableHead>
            <TableHead className="text-right">Dauer</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Zeitpunkt</TableHead>
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
              <TableCell>{statusBadge(log.status)}</TableCell>
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
