/**
 * Dashboard page -- the home screen after login.
 *
 * Renders a personalizable widget grid via WidgetContainer.
 * Users can toggle edit mode to drag, resize, add, and remove widgets.
 * Layout persists to localStorage via the dashboard Zustand store.
 */
import { useEffect } from 'react'
import { Settings, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useDashboardStore } from '@/stores/dashboard'
import WidgetContainer from '@/components/widgets/WidgetContainer'

export default function DashboardPage() {
  const isEditing = useDashboardStore((s) => s.isEditing)
  const toggleEditing = useDashboardStore((s) => s.toggleEditing)
  const resetToDefaults = useDashboardStore((s) => s.resetToDefaults)
  const ensureDefaults = useDashboardStore((s) => s.ensureDefaults)

  // Load default widgets if this is the first visit
  useEffect(() => {
    ensureDefaults()
  }, [ensureDefaults])

  return (
    <div className="h-full overflow-auto">
      <div className="px-6 py-6">
        {/* Page header */}
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">Dashboard</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              Willkommen bei KMU Hub
            </p>
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
