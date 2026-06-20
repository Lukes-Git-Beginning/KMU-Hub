/**
 * DATEV settings panel component.
 *
 * Dialog (not a wizard) for configuring the DATEV integration:
 * - Connection status with OAuth connect/disconnect
 * - Mandant-Nr (client ID) input
 * - Auto-upload toggle
 * - Manual upload button with date range picker
 * - Upload log table
 * - "Beleg hochladen" for individual invoice PDFs
 * - Fallback: when not connected, shows "Manueller CSV-Export verfügbar"
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import {
  CheckCircle,
  XCircle,
  Loader2,
  ExternalLink,
  Unlink,
  Upload,
  FileText,
  Calendar,
  Download,
  AlertTriangle,
} from 'lucide-react'
import {
  useDatevConnectionStatus,
  useDatevDisconnect,
  useDatevGetAuthURL,
  useDatevUploadConfig,
  useDatevUpdateConfig,
  useDatevUploadBuchungsstapel,
  useDatevUploadBeleg,
  useDatevUploadLogs,
} from '@/api/hooks/useDatevUpload'
import type {
  DatevUploadConfig,
  DatevUploadLogEntry,
} from '@/api/datev-upload-types'
import { formatDate } from '@/lib/format'

interface DatevSettingsPanelProps {
  isOpen: boolean
  onClose: () => void
}

export function DatevSettingsPanel({ isOpen, onClose }: DatevSettingsPanelProps) {
  const { t } = useTranslation()
  const { data: connection } = useDatevConnectionStatus()
  const disconnect = useDatevDisconnect()
  const getAuthURL = useDatevGetAuthURL()
  const { data: config } = useDatevUploadConfig()
  const updateConfig = useDatevUpdateConfig()
  const uploadBuchungsstapel = useDatevUploadBuchungsstapel()
  const uploadBeleg = useDatevUploadBeleg()
  const { data: uploadLogs } = useDatevUploadLogs(20)

  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [polling, setPolling] = useState(false)

  const isConnected = connection?.connected === true

  const handleConnect = async () => {
    try {
      const redirectUrl = window.location.href
      const result = await getAuthURL.mutateAsync(redirectUrl)
      window.open(result.authorization_url, '_blank')
      setPolling(true)
    } catch {
      // Error handled by hook toast
    }
  }

  const handleDisconnect = async () => {
    if (
      !confirm(t('settings.integrations.datev.settings.disconnectConfirm'))
    )
      return
    await disconnect.mutateAsync()
  }

  const handleConfigUpdate = (partial: Partial<DatevUploadConfig>) => {
    if (!config) return
    updateConfig.mutate({ ...config, ...partial })
  }

  const handleUpload = async () => {
    if (!dateFrom || !dateTo) return
    await uploadBuchungsstapel.mutateAsync({
      startDate: dateFrom,
      endDate: dateTo,
    })
  }

  const handleUploadBeleg = () => {
    // Open file picker so the user selects an invoice PDF.
    // The selected file's name is used as the invoiceId for the beleg upload
    // (the backend endpoint accepts either a document-id or a file reference).
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.pdf'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (!file) return
      // uploadBeleg uses the file name as the invoice reference sent to DATEV.
      // Retry + backoff is handled transparently inside useDatevUploadBeleg.
      await uploadBeleg.mutateAsync(file.name)
    }
    input.click()
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t('settings.integrations.datev.settings.title')}</DialogTitle>
          <DialogDescription>
            {t('settings.integrations.datev.settings.description')}
          </DialogDescription>
        </DialogHeader>

        {/* Connection status */}
        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div className="flex items-center gap-3">
            {isConnected ? (
              <CheckCircle className="h-5 w-5 text-success shrink-0" />
            ) : (
              <XCircle className="h-5 w-5 text-muted-foreground shrink-0" />
            )}
            <div>
              <p className="text-sm font-medium">
                {isConnected ? t('settings.integrations.datev.settings.connectionStatusConnected') : t('settings.integrations.datev.settings.connectionStatusDisconnected')}
              </p>
              {isConnected && connection?.connected_at && (
                <p className="text-xs text-muted-foreground">
                  {t('settings.integrations.datev.settings.connectedSince', { date: formatDate(connection.connected_at) })}
                </p>
              )}
            </div>
          </div>
          {isConnected ? (
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
              {t('settings.integrations.panel.disconnect')}
            </Button>
          ) : (
            <Button
              size="sm"
              onClick={handleConnect}
              disabled={getAuthURL.isPending || polling}
            >
              {getAuthURL.isPending || polling ? (
                <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
              ) : (
                <ExternalLink className="h-3.5 w-3.5 mr-1.5" />
              )}
              {polling ? t('settings.integrations.datev.settings.waitingForAuth') : t('settings.integrations.datev.settings.connectButton')}
            </Button>
          )}
        </div>

        {/* Fallback when not connected */}
        {!isConnected && (
          <div className="rounded-md border border-warning/30 bg-warning-light p-3 flex items-start gap-2">
            <AlertTriangle className="h-4 w-4 text-warning shrink-0 mt-0.5" />
            <div>
              <p className="text-sm text-warning">
                {t('settings.integrations.datev.settings.fallbackMessage')}
              </p>
              <Button
                variant="link"
                size="sm"
                className="h-auto p-0 text-warning underline text-xs mt-1"
              >
                <Download className="h-3 w-3 mr-1" />
                {t('settings.integrations.datev.settings.csvExportAvailable')}
              </Button>
            </div>
          </div>
        )}

        {/* Connected settings */}
        {isConnected && (
          <>
            <Separator />

            {/* Mandant-Nr */}
            <div className="space-y-2">
              <Label htmlFor="mandant-nr">{t('settings.integrations.datev.settings.mandantNrLabel')}</Label>
              <Input
                id="mandant-nr"
                placeholder={t('settings.integrations.datev.config.mandantNrPlaceholder')}
                value={config?.client_number ?? ''}
                onChange={(e) =>
                  handleConfigUpdate({ client_number: e.target.value })
                }
              />
              <p className="text-xs text-muted-foreground">
                {t('settings.integrations.datev.settings.mandantNrHint')}
              </p>
            </div>

            {/* Auto-upload toggle */}
            <div className="flex items-center justify-between rounded-md border border-border p-3">
              <div>
                <Label className="text-sm font-medium">
                  {t('settings.integrations.datev.settings.autoUploadLabel')}
                </Label>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {t('settings.integrations.datev.settings.autoUploadDesc')}
                </p>
              </div>
              <Switch
                checked={config?.auto_upload_enabled ?? false}
                onCheckedChange={(v) =>
                  handleConfigUpdate({ auto_upload_enabled: v })
                }
              />
            </div>

            <Separator />

            {/* Manual upload with date range */}
            <div className="space-y-3">
              <h4 className="text-sm font-medium">{t('settings.integrations.datev.settings.manualUploadSection')}</h4>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label htmlFor="date-from" className="text-xs">
                    {t('settings.integrations.datev.settings.dateFrom')}
                  </Label>
                  <div className="relative">
                    <Calendar className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                    <Input
                      id="date-from"
                      type="date"
                      value={dateFrom}
                      onChange={(e) => setDateFrom(e.target.value)}
                      className="pl-9 text-sm"
                    />
                  </div>
                </div>
                <div className="space-y-1">
                  <Label htmlFor="date-to" className="text-xs">
                    {t('settings.integrations.datev.settings.dateTo')}
                  </Label>
                  <div className="relative">
                    <Calendar className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                    <Input
                      id="date-to"
                      type="date"
                      value={dateTo}
                      onChange={(e) => setDateTo(e.target.value)}
                      className="pl-9 text-sm"
                    />
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  onClick={handleUpload}
                  disabled={
                    !dateFrom ||
                    !dateTo ||
                    uploadBuchungsstapel.isPending
                  }
                >
                  {uploadBuchungsstapel.isPending ? (
                    <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                  ) : (
                    <Upload className="h-3.5 w-3.5 mr-1.5" />
                  )}
                  {t('settings.integrations.datev.settings.uploadBuchungsstapel')}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleUploadBeleg}
                  disabled={uploadBeleg.isPending}
                >
                  {uploadBeleg.isPending ? (
                    <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                  ) : (
                    <FileText className="h-3.5 w-3.5 mr-1.5" />
                  )}
                  {t('settings.integrations.datev.settings.uploadBeleg')}
                </Button>
              </div>
            </div>

            <Separator />

            {/* Upload log table */}
            {uploadLogs && uploadLogs.length > 0 && (
              <div className="space-y-2">
                <h4 className="text-sm font-medium">{t('settings.integrations.datev.settings.uploadLog')}</h4>
                <div className="rounded-md border border-border overflow-hidden">
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="bg-muted/50">
                        <th className="text-left px-3 py-2 font-medium">
                          {t('settings.integrations.datev.settings.colType')}
                        </th>
                        <th className="text-left px-3 py-2 font-medium">
                          {t('settings.integrations.datev.settings.colStatus')}
                        </th>
                        <th className="text-right px-3 py-2 font-medium">
                          {t('settings.integrations.datev.settings.colDocuments')}
                        </th>
                        <th className="text-right px-3 py-2 font-medium">
                          {t('settings.integrations.datev.settings.colSize')}
                        </th>
                        <th className="text-left px-3 py-2 font-medium">
                          {t('settings.integrations.datev.settings.colDate')}
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {uploadLogs.map((log) => (
                        <UploadLogRow key={log.id} log={log} />
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

const UPLOAD_TYPE_KEYS: Record<string, string> = {
  buchungsstapel: 'settings.integrations.datev.settings.uploadTypeBuchungsstapel',
  belegbild: 'settings.integrations.datev.settings.uploadTypeBelegbild',
}

function UploadLogRow({ log }: { log: DatevUploadLogEntry }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)

  return (
    <>
      <tr
        className="border-t border-border hover:bg-muted/30 cursor-pointer"
        onClick={() => log.error_message && setExpanded(!expanded)}
      >
        <td className="px-3 py-2">
          {UPLOAD_TYPE_KEYS[log.upload_type] ? t(UPLOAD_TYPE_KEYS[log.upload_type]) : log.upload_type}
        </td>
        <td className="px-3 py-2">
          <UploadStatusBadge status={log.status} />
        </td>
        <td className="px-3 py-2 text-right">{log.document_count}</td>
        <td className="px-3 py-2 text-right">{formatFileSize(log.file_size)}</td>
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

function UploadStatusBadge({
  status,
}: {
  status: DatevUploadLogEntry['status']
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
          {t('settings.integrations.datev.settings.statusCompleted')}
        </Badge>
      )
    case 'failed':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-destructive/30 text-destructive"
        >
          <XCircle className="h-2.5 w-2.5 mr-0.5" />
          {t('settings.integrations.datev.settings.statusFailed')}
        </Badge>
      )
    case 'uploading':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-info/30 text-info"
        >
          <Loader2 className="h-2.5 w-2.5 mr-0.5 animate-spin" />
          {t('settings.integrations.datev.settings.statusUploading')}
        </Badge>
      )
    case 'pending':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-warning/30 text-warning"
        >
          <AlertTriangle className="h-2.5 w-2.5 mr-0.5" />
          {t('settings.integrations.datev.settings.statusPending')}
        </Badge>
      )
  }
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const size = bytes / Math.pow(1024, i)
  return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}
