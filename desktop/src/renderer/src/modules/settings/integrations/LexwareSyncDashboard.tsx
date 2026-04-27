/**
 * Lexware sync dashboard component.
 *
 * Displays connection status, sync status cards per entity type,
 * webhook status indicator, manual sync trigger, sync history table,
 * and collapsible field mapping editor. Shown as a dialog when admin
 * clicks "Konfigurieren" on a connected Lexware integration card.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
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
  CheckCircle,
  XCircle,
  AlertTriangle,
  Clock,
  Unlink,
  Webhook,
} from 'lucide-react'
import {
  useLexwareConnectionStatus,
  useLexwareDisconnect,
  useLexwareSyncStatus,
  useLexwareSyncLogs,
  useLexwareTriggerSync,
} from '@/api/hooks/useLexware'
import type {
  LexwareEntitySyncStatus,
  LexwareSyncLogEntry,
} from '@/api/lexware-types'
import { LexwareFieldMappingEditor } from './LexwareFieldMappingEditor'
import { formatDate } from '@/lib/format'

interface LexwareSyncDashboardProps {
  isOpen: boolean
  onClose: () => void
}

export function LexwareSyncDashboard({
  isOpen,
  onClose,
}: LexwareSyncDashboardProps) {
  const { t } = useTranslation()
  const { data: connection } = useLexwareConnectionStatus()
  const disconnect = useLexwareDisconnect()
  const { data: syncStatus, refetch: refetchStatus } = useLexwareSyncStatus()
  const { data: syncLogs } = useLexwareSyncLogs(20)
  const triggerSync = useLexwareTriggerSync()

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

   
  useEffect(() => {
    if (!syncStatus) return
    const anyRunning =
      syncStatus.contact_sync?.status === 'running' ||
      syncStatus.invoice_sync?.status === 'running' ||
      syncStatus.quote_sync?.status === 'running'
    if (!anyRunning && isSyncing) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync local editable state from prop
      setIsSyncing(false)
    }
  }, [syncStatus, isSyncing])

  const handleSync = async () => {
    setIsSyncing(true)
    await triggerSync.mutateAsync()
  }

  const handleDisconnect = async () => {
    if (
      !confirm(t('settings.integrations.lexware.sync.confirmDisconnect'))
    )
      return
    await disconnect.mutateAsync()
    onClose()
  }

  // Derive webhook status from connection data
  const webhookActive = connection?.connected === true

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t('settings.integrations.lexware.sync.title')}</DialogTitle>
        </DialogHeader>

        {/* Connection header */}
        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div>
            <p className="text-sm font-medium">
              {t('settings.integrations.lexware.sync.connectedWith')}
            </p>
            {connection?.connected_at && (
              <p className="text-xs text-muted-foreground">
                {t('settings.integrations.lexware.sync.connectedSince', {
                  date: formatDate(connection.connected_at),
                })}
              </p>
            )}
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleDisconnect}
            disabled={disconnect.isPending}
            className="text-destructive hover:text-destructive"
          >
            {disconnect.isPending ? (
              <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
            ) : (
              <Unlink className="h-3.5 w-3.5 mr-1" />
            )}
            {t('settings.integrations.lexware.sync.disconnect')}
          </Button>
        </div>

        {/* Sync status cards */}
        <div className="grid grid-cols-2 gap-3">
          <SyncStatusCard
            icon={<Users className="h-4 w-4" />}
            name={t('settings.integrations.lexware.sync.entityContacts')}
            status={syncStatus?.contact_sync}
          />
          <SyncStatusCard
            icon={<FileText className="h-4 w-4" />}
            name={t('settings.integrations.lexware.sync.entityInvoices')}
            status={syncStatus?.invoice_sync}
          />
          <SyncStatusCard
            icon={<Receipt className="h-4 w-4" />}
            name={t('settings.integrations.lexware.sync.entityQuotes')}
            status={syncStatus?.quote_sync}
          />

          {/* Webhook status indicator */}
          <div className="rounded-md border border-border p-3 space-y-1">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">
                  <Webhook className="h-4 w-4" />
                </span>
                <span className="text-sm font-medium">Webhook</span>
              </div>
              {webhookActive ? (
                <Badge
                  variant="outline"
                  className="text-[10px] border-success/30 text-success"
                >
                  <CheckCircle className="h-2.5 w-2.5 mr-0.5" />
                  {t('settings.integrations.lexware.sync.webhookActive')}
                </Badge>
              ) : (
                <Badge
                  variant="outline"
                  className="text-[10px] border-destructive/30 text-destructive"
                >
                  <XCircle className="h-2.5 w-2.5 mr-0.5" />
                  {t('settings.integrations.lexware.sync.webhookInactive')}
                </Badge>
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              {webhookActive
                ? t('settings.integrations.lexware.sync.webhookActiveDesc')
                : t('settings.integrations.lexware.sync.webhookInactiveDesc')}
            </p>
          </div>
        </div>

        {/* Manual sync button + field mapping toggle */}
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
            {t('settings.integrations.lexware.sync.syncNow')}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowFieldMappings(!showFieldMappings)}
          >
            {showFieldMappings
              ? t('settings.integrations.lexware.sync.fieldMappingsHide')
              : t('settings.integrations.lexware.sync.fieldMappingsShow')}
          </Button>
        </div>

        {/* Field mapping editor (collapsible) */}
        {showFieldMappings && (
          <div className="rounded-md border border-border p-4">
            <LexwareFieldMappingEditor entityType="contact" />
          </div>
        )}

        {/* Sync history */}
        {syncLogs && syncLogs.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-sm font-medium">{t('settings.integrations.lexware.sync.history')}</h4>
            <div className="rounded-md border border-border overflow-hidden">
              <table className="w-full text-xs">
                <thead>
                  <tr className="bg-muted/50">
                    <th className="text-left px-3 py-2 font-medium">{t('settings.integrations.lexware.sync.col.type')}</th>
                    <th className="text-left px-3 py-2 font-medium">{t('settings.integrations.lexware.sync.col.status')}</th>
                    <th className="text-right px-3 py-2 font-medium">
                      {t('settings.integrations.lexware.sync.col.processed')}
                    </th>
                    <th className="text-right px-3 py-2 font-medium">
                      {t('settings.integrations.lexware.sync.col.errors')}
                    </th>
                    <th className="text-left px-3 py-2 font-medium">
                      {t('settings.integrations.lexware.sync.col.started')}
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

function SyncStatusCard({
  icon,
  name,
  status,
}: {
  icon: React.ReactNode
  name: string
  status?: LexwareEntitySyncStatus
}) {
  const { t } = useTranslation()
  const statusColor =
    status?.status === 'running'
      ? 'text-info'
      : status?.status === 'completed'
        ? 'text-success'
        : status?.status === 'failed'
          ? 'text-destructive'
          : 'text-muted-foreground'

  return (
    <div className="rounded-md border border-border p-3 space-y-1">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground">{icon}</span>
          <span className="text-sm font-medium">{name}</span>
        </div>
        {status?.status === 'running' ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin text-info" />
        ) : (
          <span className={`text-xs ${statusColor}`}>
            {status?.status === 'completed'
              ? t('settings.integrations.lexware.sync.statusOk')
              : status?.status === 'failed'
                ? t('settings.integrations.lexware.sync.statusError')
                : status?.status === 'running'
                  ? t('settings.integrations.lexware.sync.statusRunning')
                  : t('settings.integrations.lexware.sync.statusWaiting')}
          </span>
        )}
      </div>
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>
          {t('settings.integrations.lexware.sync.itemsSynced', { count: status?.items_synced ?? 0 })}
          {(status?.items_failed ?? 0) > 0 && (
            <span className="text-destructive ml-1">
              ({t('settings.integrations.lexware.sync.itemsFailed', { count: status!.items_failed })})
            </span>
          )}
        </span>
        {status?.last_sync_at && (
          <span className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            {formatRelativeTime(status.last_sync_at, t)}
          </span>
        )}
      </div>
    </div>
  )
}

function SyncLogRow({ log }: { log: LexwareSyncLogEntry }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)

  const SYNC_TYPE_LABELS: Record<string, string> = {
    contact_full: t('settings.integrations.lexware.sync.syncTypeLabels.contactFull'),
    contact_delta: t('settings.integrations.lexware.sync.syncTypeLabels.contactDelta'),
    invoice_push: t('settings.integrations.lexware.sync.syncTypeLabels.invoicePush'),
    quote_push: t('settings.integrations.lexware.sync.syncTypeLabels.quotePush'),
    credit_note_push: t('settings.integrations.lexware.sync.syncTypeLabels.creditNotePush'),
    webhook_event: t('settings.integrations.lexware.sync.syncTypeLabels.webhookEvent'),
  }

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
            <span className="text-destructive">{log.items_failed}</span>
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
          <td colSpan={5} className="px-3 py-2 bg-error-light">
            <p className="text-xs text-destructive">
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
  status: LexwareSyncLogEntry['status']
}) {
  const { t } = useTranslation()
  switch (status) {
    case 'completed':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-success/30 text-success"
        >
          <CheckCircle className="h-2.5 w-2.5 mr-0.5" />
          {t('settings.integrations.lexware.sync.statusOk')}
        </Badge>
      )
    case 'failed':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-destructive/30 text-destructive"
        >
          <XCircle className="h-2.5 w-2.5 mr-0.5" />
          {t('settings.integrations.lexware.sync.statusError')}
        </Badge>
      )
    case 'partial':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-warning/30 text-warning"
        >
          <AlertTriangle className="h-2.5 w-2.5 mr-0.5" />
          {t('settings.integrations.lexware.sync.statusPartial')}
        </Badge>
      )
    case 'running':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-info/30 text-info"
        >
          <Loader2 className="h-2.5 w-2.5 mr-0.5 animate-spin" />
          {t('settings.integrations.lexware.sync.statusRunning')}
        </Badge>
      )
  }
}

function formatRelativeTime(isoDate: string, t: (key: string, opts?: object) => string): string {
  const diff = Date.now() - new Date(isoDate).getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return t('settings.integrations.lexware.sync.justNow')
  if (minutes < 60) return t('settings.integrations.lexware.sync.minutesAgo', { count: minutes })
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return t('settings.integrations.lexware.sync.hoursAgo', { count: hours })
  const days = Math.floor(hours / 24)
  return t('settings.integrations.lexware.sync.daysAgo', { count: days })
}
