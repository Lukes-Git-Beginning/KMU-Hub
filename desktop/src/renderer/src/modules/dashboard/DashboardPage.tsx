import { useEffect } from 'react'
import { Pencil, Check, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { AlertsSection, ModulesGrid } from '@/components/dashboard'
import WidgetContainer from '@/components/widgets/WidgetContainer'
import { useDashboardStore } from '@/stores/dashboard'

function getGreeting(): string {
  const hour = new Date().getHours()
  if (hour < 12) return 'Guten Morgen'
  if (hour < 18) return 'Guten Tag'
  return 'Guten Abend'
}

export default function DashboardPage() {
  const isEditing = useDashboardStore((s) => s.isEditing)
  const toggleEditing = useDashboardStore((s) => s.toggleEditing)
  const resetToDefaults = useDashboardStore((s) => s.resetToDefaults)
  const ensureDefaults = useDashboardStore((s) => s.ensureDefaults)

  useEffect(() => {
    ensureDefaults()
  }, [ensureDefaults])

  return (
    <div className="h-full overflow-auto">
      <div className="p-4 md:p-8">
        {/* Greeting Header */}
        <div className="mb-8 flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">
              {getGreeting()}
            </h1>
            <p className="mt-1 text-sm text-muted-foreground">
              Willkommen im KMU Digital Hub &ndash; Ihre All-in-One Plattform
            </p>
          </div>

          {/* Dashboard edit controls */}
          <div className="flex items-center gap-2">
            {isEditing && (
              <Button
                variant="ghost"
                size="sm"
                onClick={resetToDefaults}
                className="text-muted-foreground"
              >
                <RotateCcw className="mr-1.5 h-3.5 w-3.5" />
                Zurücksetzen
              </Button>
            )}
            <Button
              variant={isEditing ? 'default' : 'outline'}
              size="sm"
              onClick={toggleEditing}
            >
              {isEditing ? (
                <>
                  <Check className="mr-1.5 h-3.5 w-3.5" />
                  Fertig
                </>
              ) : (
                <>
                  <Pencil className="mr-1.5 h-3.5 w-3.5" />
                  Dashboard anpassen
                </>
              )}
            </Button>
          </div>
        </div>

        {/* Alerts */}
        <AlertsSection />

        {/* Modules Grid */}
        <ModulesGrid />

        {/* Widget Grid */}
        <div className="mb-8">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-foreground">
              Widgets
            </h2>
            {isEditing && (
              <p className="text-xs text-muted-foreground">
                Widgets verschieben, skalieren oder entfernen
              </p>
            )}
          </div>
          <WidgetContainer />
        </div>
      </div>
    </div>
  )
}
