/**
 * Dashboard page -- the home screen after login.
 *
 * Renders a personalizable widget grid via WidgetContainer.
 * Users can toggle edit mode to drag, resize, add, and remove widgets.
 * Layout syncs to server with offline fallback to localStorage.
 */
import { useEffect } from 'react'
import { Settings, RotateCcw, Cloud, CloudOff, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useDashboardStore, type SyncStatus } from '@/stores/dashboard'
import WidgetContainer from '@/components/widgets/WidgetContainer'

/** Small sync status indicator icon. */
function SyncStatusIcon({ status }: { status: SyncStatus }) {
  switch (status) {
    case 'synced':
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <Cloud className="h-4 w-4 text-green-500" />
          </TooltipTrigger>
          <TooltipContent>Mit Server synchronisiert</TooltipContent>
        </Tooltip>
      )
    case 'syncing':
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </TooltipTrigger>
          <TooltipContent>Synchronisiere...</TooltipContent>
        </Tooltip>
      )
    case 'offline':
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <CloudOff className="h-4 w-4 text-amber-500" />
          </TooltipTrigger>
          <TooltipContent>Offline -- lokal gespeichert</TooltipContent>
        </Tooltip>
      )
    default:
      return null
  }
}

export default function DashboardPage() {
  const isEditing = useDashboardStore((s) => s.isEditing)
  const toggleEditing = useDashboardStore((s) => s.toggleEditing)
  const resetToDefaults = useDashboardStore((s) => s.resetToDefaults)
  const ensureDefaults = useDashboardStore((s) => s.ensureDefaults)
  const initFromServer = useDashboardStore((s) => s.initFromServer)
  const serverSyncStatus = useDashboardStore((s) => s.serverSyncStatus)

  // Load default widgets if this is the first visit
  useEffect(() => {
    ensureDefaults()
  }, [ensureDefaults])

  // Initialize from server on mount
  useEffect(() => {
    initFromServer()
  }, [initFromServer])

  return (
    <div className="h-full overflow-auto">
      <div className="px-6 py-6">
        {/* Page header */}
        <div className="mb-6 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div>
              <h1 className="text-2xl font-semibold text-foreground">Dashboard</h1>
              <p className="mt-1 text-sm text-muted-foreground">
                Willkommen bei KMU Hub
              </p>
            </div>
            <SyncStatusIcon status={serverSyncStatus} />
          </div>

          <div className="flex items-center gap-2">
            {isEditing && (
              <Button
                variant="outline"
                size="sm"
                onClick={resetToDefaults}
              >
                <RotateCcw className="mr-2 h-4 w-4" />
                Zuruecksetzen
              </Button>
            )}
            <Button
              variant={isEditing ? 'default' : 'outline'}
              size="sm"
              onClick={toggleEditing}
            >
              <Settings className="mr-2 h-4 w-4" />
              {isEditing ? 'Fertig' : 'Anpassen'}
            </Button>
          </div>
        </div>

        {/* Widget grid */}
        <WidgetContainer />
      </div>
    </div>
  )
}
