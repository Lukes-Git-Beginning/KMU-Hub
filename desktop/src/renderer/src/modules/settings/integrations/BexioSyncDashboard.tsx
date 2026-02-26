/**
 * Bexio sync dashboard component.
 *
 * Displays connection status, sync status cards per entity type,
 * manual sync trigger, sync history table, and links to field mapping
 * editor. Shown as a dialog when admin clicks "Konfigurieren" on a
 * connected Bexio integration card.
 */
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  RefreshCw,
  Loader2,
  Users,
  FileText,
  Receipt,
  CreditCard,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Clock,
  Unlink,
} from 'lucide-react'
import {
  useBexioConnectionStatus,
  useBexioDisconnect,
  useBexioSyncStatus,
  useBexioSyncLogs,
  useBexioTriggerSync,
} from '@/api/hooks/useBexio'
import type {
  BexioEntitySyncStatus,
  BexioSyncLogEntry,
} from '@/api/bexio-types'
import { BexioFieldMappingEditor } from './BexioFieldMappingEditor'

interface BexioSyncDashboardProps {
  isOpen: boolean
  onClose: () => void
}

export function BexioSyncDashboard({
  isOpen,
  onClose,
}: BexioSyncDashboardProps) {
  const { data: connection } = useBexioConnectionStatus()
  const disconnect = useBexioDisconnect()
  const { data: syncStatus, refetch: refetchStatus } = useBexioSyncStatus()
  const { data: syncLogs } = useBexioSyncLogs(20)
  const triggerSync = useBexioTriggerSync()

  const [isSyncing, setIsSyncing] = useState(false)
  const [showFieldMappings, setShowFieldMappings] = useState(false)

  // Poll sync status while syncing
  useEffect(() => {
    if (!isSyncing) return
    const interval = setInterval(() => {
      refetchStatus()
    }, 2000)
    return () => clearInterval(interval)
  }, [isSyncing, refetchStatus])

  // Check if any sync is running
  useEffect(() => {
    if (!syncStatus) return
    const anyRunning =
      syncStatus.contact_sync?.status === 'running' ||
      syncStatus.invoice_sync?.status === 'running' ||
      syncStatus.quote_sync?.status === 'running' ||
      syncStatus.payment_poll?.status === 'running'
    if (!anyRunning && isSyncing) {
      setIsSyncing(false)
    }
  }, [syncStatus, isSyncing])

  const handleSync = async () => {
    setIsSyncing(true)
    await triggerSync.mutateAsync()
  }

  const handleDisconnect = async () => {
    if (!confirm('Bexio-Verbindung wirklich trennen? Alle Sync-Einstellungen gehen verloren.')) return
    await disconnect.mutateAsync()
    onClose()
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Bexio Synchronisierung</DialogTitle>
        </DialogHeader>

        {/* Connection header */}
        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div>
            <p className="text-sm font-medium">
              Verbunden mit: {connection?.org_name || 'Bexio'}
            </p>
            {connection?.connected_at && (
              <p className="text-xs text-muted-foreground">
                Seit{' '}
                {new Date(connection.connected_at).toLocaleDateString('de-DE')}
              </p>
            )}
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleDisconnect}
            disabled={disconnect.isPending}
            className="text-red-500 hover:text-red-600"
          >
            {disconnect.isPending ? (
              <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
            ) : (
              <Unlink className="h-3.5 w-3.5 mr-1" />
            )}
            Trennen
          </Button>
        </div>

        {/* Sync status cards */}
        <div className="grid grid-cols-2 gap-3">
          <SyncStatusCard
            icon={<Users className="h-4 w-4" />}
            name="Kontakte"
            status={syncStatus?.contact_sync}
          />
          <SyncStatusCard
            icon={<FileText className="h-4 w-4" />}
            name="Rechnungen"
            status={syncStatus?.invoice_sync}
          />
          <SyncStatusCard
            icon={<Receipt className="h-4 w-4" />}
            name="Offerten"
            status={syncStatus?.quote_sync}
          />
          <SyncStatusCard
            icon={<CreditCard className="h-4 w-4" />}
            name="Zahlungen"
            status={syncStatus?.payment_poll}
          />
        </div>

        {/* Manual sync button */}
        <div className="flex items-center gap-3">
          <Button
            onClick={handleSync}
            disabled={isSyncing || triggerSync.isPending}
          >
            {isSyncing || triggerSync.isPending ? (
              <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
            ) : (
              <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
            )}
            Jetzt synchronisieren
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowFieldMappings(!showFieldMappings)}
          >
            Feld-Zuordnungen {showFieldMappings ? 'ausblenden' : 'anzeigen'}
          </Button>
        </div>

        {/* Field mapping editor (collapsible) */}
        {showFieldMappings && (
          <div className="rounded-md border border-border p-4">
            <BexioFieldMappingEditor entityType="contact" />
          </div>
        )}

        {/* Sync history */}
        {syncLogs && syncLogs.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-sm font-medium">Sync-Verlauf</h4>
            <div className="rounded-md border border-border overflow-hidden">
              <table className="w-full text-xs">
                <thead>
                  <tr className="bg-muted/50">
                    <th className="text-left px-3 py-2 font-medium">Typ</th>
                    <th className="text-left px-3 py-2 font-medium">Status</th>
                    <th className="text-right px-3 py-2 font-medium">
                      Verarbeitet
                    </th>
                    <th className="text-right px-3 py-2 font-medium">
                      Fehler
                    </th>
                    <th className="text-left px-3 py-2 font-medium">
                      Gestartet
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {syncLogs.map((log) => (
                    <SyncLogRow key={log.id} log={log} />
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

const SYNC_TYPE_LABELS: Record<string, string> = {
  contact_full: 'Kontakte (Voll)',
  contact_delta: 'Kontakte (Delta)',
  invoice_push: 'Rechnungen',
  quote_push: 'Offerten',
  payment_poll: 'Zahlungen',
}

function SyncStatusCard({
  icon,
  name,
  status,
}: {
  icon: React.ReactNode
  name: string
  status?: BexioEntitySyncStatus
}) {
  const statusColor =
    status?.status === 'running'
      ? 'text-blue-500'
      : status?.status === 'completed'
        ? 'text-green-500'
        : status?.status === 'failed'
          ? 'text-red-500'
          : 'text-muted-foreground'

  return (
    <div className="rounded-md border border-border p-3 space-y-1">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground">{icon}</span>
          <span className="text-sm font-medium">{name}</span>
        </div>
        {status?.status === 'running' ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-500" />
        ) : (
          <span className={`text-xs ${statusColor}`}>
            {status?.status === 'completed'
              ? 'OK'
              : status?.status === 'failed'
                ? 'Fehler'
                : status?.status === 'running'
                  ? 'Laeuft'
                  : 'Wartend'}
          </span>
        )}
      </div>
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>
          {status?.items_synced ?? 0} synchronisiert
          {(status?.items_failed ?? 0) > 0 && (
            <span className="text-red-500 ml-1">
              ({status!.items_failed} Fehler)
            </span>
          )}
        </span>
        {status?.last_sync_at && (
          <span className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            {formatRelativeTime(status.last_sync_at)}
          </span>
        )}
      </div>
    </div>
  )
}

function SyncLogRow({ log }: { log: BexioSyncLogEntry }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <>
      <tr
        className="border-t border-border hover:bg-muted/30 cursor-pointer"
        onClick={() => log.error_message && setExpanded(!expanded)}
      >
        <td className="px-3 py-2">
          {SYNC_TYPE_LABELS[log.sync_type] || log.sync_type}
        </td>
        <td className="px-3 py-2">
          <SyncStatusBadge status={log.status} />
        </td>
        <td className="px-3 py-2 text-right">{log.items_processed}</td>
        <td className="px-3 py-2 text-right">
          {log.items_failed > 0 ? (
            <span className="text-red-500">{log.items_failed}</span>
          ) : (
            '0'
          )}
        </td>
        <td className="px-3 py-2">
          {new Date(log.started_at).toLocaleString('de-DE', {
            day: '2-digit',
            month: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
          })}
        </td>
      </tr>
      {expanded && log.error_message && (
        <tr className="border-t border-border">
          <td colSpan={5} className="px-3 py-2 bg-red-50/10">
            <p className="text-xs text-red-600 dark:text-red-400">
              {log.error_message}
            </p>
          </td>
        </tr>
      )}
    </>
  )
}

function SyncStatusBadge({
  status,
}: {
  status: BexioSyncLogEntry['status']
}) {
  switch (status) {
    case 'completed':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-green-500/30 text-green-600 dark:text-green-400"
        >
          <CheckCircle className="h-2.5 w-2.5 mr-0.5" />
          OK
        </Badge>
      )
    case 'failed':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-red-500/30 text-red-600 dark:text-red-400"
        >
          <XCircle className="h-2.5 w-2.5 mr-0.5" />
          Fehler
        </Badge>
      )
    case 'partial':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-yellow-500/30 text-yellow-600 dark:text-yellow-400"
        >
          <AlertTriangle className="h-2.5 w-2.5 mr-0.5" />
          Teilweise
        </Badge>
      )
    case 'running':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-blue-500/30 text-blue-600 dark:text-blue-400"
        >
          <Loader2 className="h-2.5 w-2.5 mr-0.5 animate-spin" />
          Laeuft
        </Badge>
      )
  }
}

function formatRelativeTime(isoDate: string): string {
  const diff = Date.now() - new Date(isoDate).getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return 'gerade eben'
  if (minutes < 60) return `vor ${minutes} Min.`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `vor ${hours} Std.`
  const days = Math.floor(hours / 24)
  return `vor ${days} Tag${days > 1 ? 'en' : ''}`
}
