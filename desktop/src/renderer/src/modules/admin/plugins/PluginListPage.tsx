/**
 * Main plugin administration page.
 *
 * Tabbed layout with:
 * - Installierte Plugins: installed plugins with enable/disable/uninstall
 * - Verfuegbare Plugins: manifest marketplace list with install action
 * - Validierungsregeln: CRUD editor for validation rules
 * - Workflow-Regeln: CRUD editor for workflow rules
 * - Branchenvorlagen: gallery of industry templates
 * - Ausfuehrungsprotokolle: execution log viewer
 */
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Puzzle,
  Power,
  PowerOff,
  Download,
  Loader2,
  Eye,
  Shield,
} from 'lucide-react'
import {
  usePluginManifests,
  usePluginInstallations,
  useInstallPlugin,
  useEnablePlugin,
  useDisablePlugin,
} from '@/api/hooks/usePlugins'
import type { PluginInstallation, InstallationStatus } from '@/api/plugin-types'
import { PluginDetailDialog } from './PluginDetailDialog'
import { PermissionApprovalDialog } from './PermissionApprovalDialog'
import { ValidationRulesEditor } from './ValidationRulesEditor'
import { WorkflowRulesEditor } from './WorkflowRulesEditor'
import { IndustryTemplateGallery } from './IndustryTemplateGallery'
import { ExecutionLogViewer } from './ExecutionLogViewer'

// ---------------------------------------------------------------------------
// Status badge helper
// ---------------------------------------------------------------------------

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
      return <Badge variant="outline">Deinstalliert</Badge>
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

function pluginTypeBadge(type: string) {
  if (type === 'wasm') {
    return (
      <Badge variant="secondary" className="text-xs">
        WASM
      </Badge>
    )
  }
  return (
    <Badge variant="outline" className="text-xs">
      Config
    </Badge>
  )
}

// ---------------------------------------------------------------------------
// Main page component
// ---------------------------------------------------------------------------

export default function PluginListPage() {
  const { data: manifests, isLoading: manifestsLoading } = usePluginManifests()
  const { data: installations, isLoading: installationsLoading } =
    usePluginInstallations()
  const installPlugin = useInstallPlugin()
  const enablePlugin = useEnablePlugin()
  const disablePlugin = useDisablePlugin()

  const [detailDialogOpen, setDetailDialogOpen] = useState(false)
  const [selectedInstallation, setSelectedInstallation] =
    useState<PluginInstallation | null>(null)
  const [permissionDialogOpen, setPermissionDialogOpen] = useState(false)
  const [permissionInstallation, setPermissionInstallation] =
    useState<PluginInstallation | null>(null)

  // Set of already-installed manifest IDs
  const installedManifestIds = new Set(
    (installations ?? [])
      .filter((i) => i.status !== 'uninstalled')
      .map((i) => i.manifest_id),
  )

  const openDetail = (installation: PluginInstallation) => {
    setSelectedInstallation(installation)
    setDetailDialogOpen(true)
  }

  const openPermissions = (installation: PluginInstallation) => {
    setPermissionInstallation(installation)
    setPermissionDialogOpen(true)
  }

  const handleInstall = (manifestId: string) => {
    installPlugin.mutate({ manifest_id: manifestId })
  }

  return (
    <div className="p-6 max-w-6xl">
      {/* Header */}
      <div className="flex items-center gap-3 mb-1">
        <Puzzle className="h-5 w-5 text-muted-foreground" />
        <h1 className="text-lg font-semibold text-foreground">
          Plugin-Verwaltung
        </h1>
      </div>
      <p className="text-sm text-muted-foreground mb-6">
        Plugins installieren, konfigurieren und verwalten
      </p>

      {/* Tabs */}
      <Tabs defaultValue="installed">
        <TabsList className="mb-4">
          <TabsTrigger value="installed">
            Installiert
            {installations && installations.length > 0 && (
              <span className="ml-1.5 text-xs text-muted-foreground">
                ({installations.filter((i) => i.status !== 'uninstalled').length})
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger value="available">Verfuegbar</TabsTrigger>
          <TabsTrigger value="validation">Validierung</TabsTrigger>
          <TabsTrigger value="workflows">Workflows</TabsTrigger>
          <TabsTrigger value="templates">Vorlagen</TabsTrigger>
          <TabsTrigger value="logs">Protokolle</TabsTrigger>
        </TabsList>

        {/* Installed Plugins */}
        <TabsContent value="installed">
          {installationsLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : installations && installations.filter((i) => i.status !== 'uninstalled').length > 0 ? (
            <div className="border border-border rounded-md overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Plugin</TableHead>
                    <TableHead>Version</TableHead>
                    <TableHead>Typ</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Aktionen</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {installations
                    .filter((i) => i.status !== 'uninstalled')
                    .map((installation) => (
                      <TableRow key={installation.id}>
                        <TableCell>
                          <div>
                            <p className="text-sm font-medium">
                              {installation.manifest_name}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {installation.manifest_slug}
                            </p>
                          </div>
                        </TableCell>
                        <TableCell className="text-sm tabular-nums">
                          v{installation.manifest_version}
                        </TableCell>
                        <TableCell>
                          {pluginTypeBadge(installation.plugin_type)}
                        </TableCell>
                        <TableCell>
                          {statusBadge(installation.status)}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-1">
                            {installation.status === 'pending_approval' && (
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => openPermissions(installation)}
                              >
                                <Shield className="h-3.5 w-3.5 mr-1" />
                                Genehmigen
                              </Button>
                            )}

                            {installation.status === 'active' && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() =>
                                  disablePlugin.mutate(installation.id)
                                }
                                disabled={disablePlugin.isPending}
                              >
                                <PowerOff className="h-3.5 w-3.5" />
                              </Button>
                            )}

                            {installation.status === 'disabled' && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() =>
                                  enablePlugin.mutate(installation.id)
                                }
                                disabled={enablePlugin.isPending}
                              >
                                <Power className="h-3.5 w-3.5" />
                              </Button>
                            )}

                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openDetail(installation)}
                            >
                              <Eye className="h-3.5 w-3.5" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="rounded-md border border-border p-8 text-center">
              <Puzzle className="h-8 w-8 text-muted-foreground mx-auto mb-3" />
              <p className="text-sm text-muted-foreground">
                Keine Plugins installiert. Wechseln Sie zum Tab
                &quot;Verfuegbar&quot; um Plugins zu installieren.
              </p>
            </div>
          )}
        </TabsContent>

        {/* Available Plugins (Manifests) */}
        <TabsContent value="available">
          {manifestsLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : manifests && manifests.length > 0 ? (
            <div className="border border-border rounded-md overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Plugin</TableHead>
                    <TableHead>Autor</TableHead>
                    <TableHead>Version</TableHead>
                    <TableHead>Typ</TableHead>
                    <TableHead>Berechtigungen</TableHead>
                    <TableHead className="text-right">Aktion</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {manifests.map((manifest) => {
                    const isInstalled = installedManifestIds.has(manifest.id)
                    return (
                      <TableRow key={manifest.id}>
                        <TableCell>
                          <div>
                            <p className="text-sm font-medium">
                              {manifest.name}
                            </p>
                            <p className="text-xs text-muted-foreground line-clamp-1">
                              {manifest.description}
                            </p>
                          </div>
                        </TableCell>
                        <TableCell className="text-sm">
                          {manifest.author}
                        </TableCell>
                        <TableCell className="text-sm tabular-nums">
                          v{manifest.version}
                        </TableCell>
                        <TableCell>
                          {pluginTypeBadge(manifest.plugin_type)}
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-wrap gap-1">
                            {manifest.permissions
                              .slice(0, 3)
                              .map((p) => (
                                <Badge
                                  key={p}
                                  variant="outline"
                                  className="text-xs font-mono"
                                >
                                  {p}
                                </Badge>
                              ))}
                            {manifest.permissions.length > 3 && (
                              <Badge variant="outline" className="text-xs">
                                +{manifest.permissions.length - 3}
                              </Badge>
                            )}
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          {isInstalled ? (
                            <Badge
                              variant="outline"
                              className="text-xs text-green-700"
                            >
                              Installiert
                            </Badge>
                          ) : (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleInstall(manifest.id)}
                              disabled={installPlugin.isPending}
                            >
                              {installPlugin.isPending ? (
                                <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
                              ) : (
                                <Download className="h-3.5 w-3.5 mr-1" />
                              )}
                              Installieren
                            </Button>
                          )}
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="rounded-md border border-border p-8 text-center">
              <Puzzle className="h-8 w-8 text-muted-foreground mx-auto mb-3" />
              <p className="text-sm text-muted-foreground">
                Keine Plugin-Manifeste verfuegbar.
              </p>
            </div>
          )}
        </TabsContent>

        {/* Validation Rules */}
        <TabsContent value="validation">
          <ValidationRulesEditor />
        </TabsContent>

        {/* Workflow Rules */}
        <TabsContent value="workflows">
          <WorkflowRulesEditor />
        </TabsContent>

        {/* Industry Templates */}
        <TabsContent value="templates">
          <IndustryTemplateGallery />
        </TabsContent>

        {/* Execution Logs */}
        <TabsContent value="logs">
          <ExecutionLogViewer />
        </TabsContent>
      </Tabs>

      {/* Detail dialog */}
      <PluginDetailDialog
        isOpen={detailDialogOpen}
        onClose={() => setDetailDialogOpen(false)}
        installation={selectedInstallation}
        onOpenPermissions={openPermissions}
      />

      {/* Permission approval dialog */}
      <PermissionApprovalDialog
        isOpen={permissionDialogOpen}
        onClose={() => setPermissionDialogOpen(false)}
        installation={permissionInstallation}
      />
    </div>
  )
}
