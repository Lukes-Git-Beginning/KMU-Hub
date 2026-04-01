/**
 * Detail dialog for a plugin installation.
 *
 * Shows status, permissions, settings editor, and recent execution logs
 * for a single installed plugin.
 */
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import {
  Power,
  PowerOff,
  Trash2,
  Shield,
  Loader2,
} from 'lucide-react'
import {
  usePluginSettings,
  useUpdatePluginSettings,
  useEnablePlugin,
  useDisablePlugin,
  useUninstallPlugin,
} from '@/api/hooks/usePlugins'
import type { PluginInstallation, InstallationStatus } from '@/api/plugin-types'
import { PluginSettingsEditor } from './PluginSettingsEditor'
import { ExecutionLogViewer } from './ExecutionLogViewer'

interface PluginDetailDialogProps {
  isOpen: boolean
  onClose: () => void
  installation: PluginInstallation | null
  onOpenPermissions: (installation: PluginInstallation) => void
}

function statusBadge(status: InstallationStatus) {
  switch (status) {
    case 'active':
      return (
        <Badge className="bg-green-500/15 text-green-700 border-green-500/30 hover:bg-green-500/15">
          Aktiv
        </Badge>
      )
    case 'disabled':
      return (
        <Badge className="bg-gray-500/15 text-gray-700 border-gray-500/30 hover:bg-gray-500/15">
          Deaktiviert
        </Badge>
      )
    case 'pending_approval':
      return (
        <Badge className="bg-yellow-500/15 text-yellow-700 border-yellow-500/30 hover:bg-yellow-500/15">
          Genehmigung ausstehend
        </Badge>
      )
    case 'error':
      return (
        <Badge className="bg-red-500/15 text-red-700 border-red-500/30 hover:bg-red-500/15">
          Fehler
        </Badge>
      )
    case 'uninstalled':
      return (
        <Badge variant="outline">Deinstalliert</Badge>
      )
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

export function PluginDetailDialog({
  isOpen,
  onClose,
  installation,
  onOpenPermissions,
}: PluginDetailDialogProps) {
  const enablePlugin = useEnablePlugin()
  const disablePlugin = useDisablePlugin()
  const uninstallPlugin = useUninstallPlugin()
  const updateSettings = useUpdatePluginSettings()

  const { data: settingsData, isLoading: settingsLoading } =
    usePluginSettings(installation?.id ?? '')

  if (!installation) return null

  const handleEnable = async () => {
    await enablePlugin.mutateAsync(installation.id)
  }

  const handleDisable = async () => {
    await disablePlugin.mutateAsync(installation.id)
  }

  const handleUninstall = async () => {
    await uninstallPlugin.mutateAsync(installation.id)
    onClose()
  }

  const handleSaveSettings = (values: Record<string, unknown>) => {
    updateSettings.mutate({
      installationId: installation.id,
      data: { settings: values },
    })
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {installation.manifest_name}
            <span className="text-xs font-normal text-muted-foreground">
              v{installation.manifest_version}
            </span>
          </DialogTitle>
          <DialogDescription>
            {installation.plugin_type === 'wasm' ? 'WASM-Plugin' : 'Config-Plugin'}{' '}
            &middot; Installiert am{' '}
            {new Date(installation.created_at).toLocaleDateString('de-DE')}
          </DialogDescription>
        </DialogHeader>

        {/* Status and actions */}
        <div className="flex items-center gap-2 flex-wrap">
          {statusBadge(installation.status)}

          {installation.status === 'pending_approval' && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => onOpenPermissions(installation)}
            >
              <Shield className="h-3.5 w-3.5 mr-1.5" />
              Berechtigungen prüfen
            </Button>
          )}

          {installation.status === 'disabled' && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleEnable}
              disabled={enablePlugin.isPending}
            >
              {enablePlugin.isPending ? (
                <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
              ) : (
                <Power className="h-3.5 w-3.5 mr-1.5" />
              )}
              Aktivieren
            </Button>
          )}

          {installation.status === 'active' && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleDisable}
              disabled={disablePlugin.isPending}
            >
              {disablePlugin.isPending ? (
                <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
              ) : (
                <PowerOff className="h-3.5 w-3.5 mr-1.5" />
              )}
              Deaktivieren
            </Button>
          )}

          <Button
            variant="ghost"
            size="sm"
            className="text-red-500 hover:text-red-600 ml-auto"
            onClick={handleUninstall}
            disabled={uninstallPlugin.isPending}
          >
            {uninstallPlugin.isPending ? (
              <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
            ) : (
              <Trash2 className="h-3.5 w-3.5 mr-1.5" />
            )}
            Deinstallieren
          </Button>
        </div>

        {/* Error message */}
        {installation.error_message && (
          <div className="rounded-md border border-red-500/30 bg-red-50/10 p-3">
            <p className="text-sm text-red-700 dark:text-red-400">
              {installation.error_message}
            </p>
          </div>
        )}

        <Separator />

        {/* Settings */}
        <section>
          <h3 className="text-sm font-medium mb-3">Einstellungen</h3>
          {settingsLoading ? (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Einstellungen laden...
            </div>
          ) : (
            <PluginSettingsEditor
              schema={settingsData?.schema ?? installation.settings_schema ?? {}}
              values={settingsData?.settings ?? installation.settings ?? {}}
              onSave={handleSaveSettings}
              isSaving={updateSettings.isPending}
            />
          )}
        </section>

        <Separator />

        {/* Permissions */}
        <section>
          <h3 className="text-sm font-medium mb-2">Berechtigungen</h3>
          <div className="flex flex-wrap gap-1.5">
            {(installation.granted_permissions ?? []).map((p) => (
              <Badge
                key={p}
                variant="outline"
                className="text-xs font-mono"
              >
                {p}
              </Badge>
            ))}
            {(installation.required_permissions ?? [])
              .filter(
                (p) => !(installation.granted_permissions ?? []).includes(p),
              )
              .map((p) => (
                <Badge
                  key={p}
                  className="text-xs font-mono bg-yellow-500/15 text-yellow-700 border-yellow-500/30 hover:bg-yellow-500/15"
                >
                  {p} (ausstehend)
                </Badge>
              ))}
            {(installation.required_permissions ?? []).length === 0 && (
              <p className="text-sm text-muted-foreground">
                Keine Berechtigungen erforderlich.
              </p>
            )}
          </div>
        </section>

        <Separator />

        {/* Recent execution logs */}
        <section>
          <h3 className="text-sm font-medium mb-3">
            Letzte Ausführungen
          </h3>
          <ExecutionLogViewer installationId={installation.id} limit={10} />
        </section>
      </DialogContent>
    </Dialog>
  )
}
