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
 * - Fallback: when not connected, shows "Manueller CSV-Export verfuegbar"
 */
import { useState } from 'react'
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
  useDatevUploadLogs,
} from '@/api/hooks/useDatev'
import type {
  DatevUploadConfig,
  DatevUploadLogEntry,
} from '@/api/datev-upload-types'

interface DatevSettingsPanelProps {
  isOpen: boolean
  onClose: () => void
}

export function DatevSettingsPanel({ isOpen, onClose }: DatevSettingsPanelProps) {
  const { data: connection } = useDatevConnectionStatus()
  const disconnect = useDatevDisconnect()
  const getAuthURL = useDatevGetAuthURL()
  const { data: config } = useDatevUploadConfig()
  const updateConfig = useDatevUpdateConfig()
  const uploadBuchungsstapel = useDatevUploadBuchungsstapel()
  const { data: uploadLogs } = useDatevUploadLogs(20)

  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [polling, setPolling] = useState(false)

  const isConnected = connection?.connected === true

  const handleConnect = async () => {
    try {
      const result = await getAuthURL.mutateAsync()
      window.open(result.authorization_url, '_blank')
      setPolling(true)
    } catch {
      // Error handled by hook toast
    }
  }

  const handleDisconnect = async () => {
    if (
      !confirm(
        'DATEV-Verbindung wirklich trennen? Upload-Einstellungen gehen verloren.',
      )
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
      date_from: dateFrom,
      date_to: dateTo,
    })
  }

  const handleUploadBeleg = () => {
    // Open file dialog for invoice PDF upload
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.pdf'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (!file) return
      // Delegate to upload mutation -- belegbild endpoint
      await uploadBuchungsstapel.mutateAsync({
        date_from: '',
        date_to: '',
        file,
        upload_type: 'belegbild',
      })
    }
    input.click()
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>DATEV-Einstellungen</DialogTitle>
          <DialogDescription>
            Konfigurieren Sie die DATEV-Anbindung fuer den automatischen oder
            manuellen Buchungsexport.
          </DialogDescription>
        </DialogHeader>

        {/* Connection status */}
        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div className="flex items-center gap-3">
            {isConnected ? (
              <CheckCircle className="h-5 w-5 text-green-500 shrink-0" />
            ) : (
              <XCircle className="h-5 w-5 text-muted-foreground shrink-0" />
            )}
            <div>
              <p className="text-sm font-medium">
                {isConnected ? 'Mit DATEV verbunden' : 'Nicht verbunden'}
              </p>
              {isConnected && connection?.connected_at && (
                <p className="text-xs text-muted-foreground">
                  Seit{' '}
                  {new Date(connection.connected_at).toLocaleDateString(
                    'de-DE',
                  )}
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
              className="text-red-500 hover:text-red-600"
            >
              {disconnect.isPending ? (
                <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
              ) : (
                <Unlink className="h-3.5 w-3.5 mr-1" />
              )}
              Trennen
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
              {polling ? 'Warte auf Autorisierung...' : 'Mit DATEV verbinden'}
            </Button>
          )}
        </div>

        {/* Fallback when not connected */}
        {!isConnected && (
          <div className="rounded-md border border-yellow-500/30 bg-yellow-50/10 p-3 flex items-start gap-2">
            <AlertTriangle className="h-4 w-4 text-yellow-600 dark:text-yellow-400 shrink-0 mt-0.5" />
            <div>
              <p className="text-sm text-yellow-700 dark:text-yellow-400">
                Ohne DATEV-Verbindung steht der manuelle CSV-Export zur
                Verfuegung.
              </p>
              <Button
                variant="link"
                size="sm"
                className="h-auto p-0 text-yellow-700 dark:text-yellow-400 underline text-xs mt-1"
              >
                <Download className="h-3 w-3 mr-1" />
                Manueller CSV-Export verfuegbar
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
              <Label htmlFor="mandant-nr">Mandant-Nr.</Label>
              <Input
                id="mandant-nr"
                placeholder="z.B. 12345"
                value={config?.client_number ?? ''}
                onChange={(e) =>
                  handleConfigUpdate({ client_number: e.target.value })
                }
              />
              <p className="text-xs text-muted-foreground">
                Die Mandantennummer aus Ihrem DATEV-Konto.
              </p>
            </div>

            {/* Auto-upload toggle */}
            <div className="flex items-center justify-between rounded-md border border-border p-3">
              <div>
                <Label className="text-sm font-medium">
                  Automatischer Upload
                </Label>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Buchungsstapel automatisch nach Export an DATEV senden
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
              <h4 className="text-sm font-medium">Manueller Upload</h4>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label htmlFor="date-from" className="text-xs">
                    Von
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
                    Bis
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
                  Buchungsstapel hochladen
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleUploadBeleg}
                  disabled={uploadBuchungsstapel.isPending}
                >
                  <FileText className="h-3.5 w-3.5 mr-1.5" />
                  Beleg hochladen
                </Button>
              </div>
            </div>

            <Separator />

            {/* Upload log table */}
            {uploadLogs && uploadLogs.length > 0 && (
              <div className="space-y-2">
                <h4 className="text-sm font-medium">Upload-Verlauf</h4>
                <div className="rounded-md border border-border overflow-hidden">
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="bg-muted/50">
                        <th className="text-left px-3 py-2 font-medium">
                          Typ
                        </th>
                        <th className="text-left px-3 py-2 font-medium">
                          Status
                        </th>
                        <th className="text-right px-3 py-2 font-medium">
                          Belege
                        </th>
                        <th className="text-right px-3 py-2 font-medium">
                          Groesse
                        </th>
                        <th className="text-left px-3 py-2 font-medium">
                          Datum
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

const UPLOAD_TYPE_LABELS: Record<string, string> = {
  buchungsstapel: 'Buchungsstapel',
  belegbild: 'Belegbild',
}

function UploadLogRow({ log }: { log: DatevUploadLogEntry }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <>
      <tr
        className="border-t border-border hover:bg-muted/30 cursor-pointer"
        onClick={() => log.error_message && setExpanded(!expanded)}
      >
        <td className="px-3 py-2">
          {UPLOAD_TYPE_LABELS[log.upload_type] || log.upload_type}
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

function UploadStatusBadge({
  status,
}: {
  status: DatevUploadLogEntry['status']
}) {
  switch (status) {
    case 'completed':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-green-500/30 text-green-600 dark:text-green-400"
        >
          <CheckCircle className="h-2.5 w-2.5 mr-0.5" />
          Fertig
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
    case 'uploading':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-blue-500/30 text-blue-600 dark:text-blue-400"
        >
          <Loader2 className="h-2.5 w-2.5 mr-0.5 animate-spin" />
          Hochladen
        </Badge>
      )
    case 'pending':
      return (
        <Badge
          variant="outline"
          className="text-[10px] border-yellow-500/30 text-yellow-600 dark:text-yellow-400"
        >
          <AlertTriangle className="h-2.5 w-2.5 mr-0.5" />
          Wartend
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
