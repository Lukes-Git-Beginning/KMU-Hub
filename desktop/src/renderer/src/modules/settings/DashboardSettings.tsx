/**
 * Admin settings page for configuring role-based default dashboards.
 *
 * Allows admins to configure which widgets appear in each role's
 * default dashboard and save the configuration to the server.
 * Non-admin users are redirected away (guarded by route).
 */
import { useState, useCallback } from 'react'
import { Navigate } from 'react-router-dom'
import { Save, RotateCcw, Check, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { useAuthStore } from '@/stores/auth'
import { useDashboardStore } from '@/stores/dashboard'
import { useDashboardDefaults, useSaveDashboardDefaults } from '@/api/hooks/useDashboard'
import { widgetList } from '@/components/widgets/WidgetRegistry'

type RoleKey = 'admin' | 'manager' | 'member'

const ROLES: { key: RoleKey; label: string; description: string }[] = [
  { key: 'admin', label: 'Admin', description: 'Administratoren sehen Pipeline und Aktivitaeten.' },
  { key: 'manager', label: 'Manager', description: 'Manager fokussieren auf Pipeline und Kommunikation.' },
  { key: 'member', label: 'Mitarbeiter', description: 'Mitarbeiter sehen Nachrichten und eigene Aktivitaeten.' },
]

/** Widget toggle card for the widget selector. */
function WidgetToggle({
  widgetId,
  name,
  description,
  isActive,
  onToggle,
}: {
  widgetId: string
  name: string
  description: string
  isActive: boolean
  onToggle: (id: string) => void
}) {
  const widget = widgetList.find((w) => w.id === widgetId)
  const Icon = widget?.icon

  return (
    <button
      type="button"
      onClick={() => onToggle(widgetId)}
      className={`
        flex items-start gap-3 rounded-lg border p-3 text-left transition-colors w-full
        ${
          isActive
            ? 'border-primary/50 bg-primary/5'
            : 'border-border hover:border-muted-foreground/30'
        }
      `}
    >
      <div className={`mt-0.5 rounded-md p-1.5 ${isActive ? 'bg-primary/10' : 'bg-muted'}`}>
        {Icon && <Icon className={`h-4 w-4 ${isActive ? 'text-primary' : 'text-muted-foreground'}`} />}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <p className="text-sm font-medium">{name}</p>
          {isActive && <Badge variant="secondary" className="text-xs">Aktiv</Badge>}
        </div>
        <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">{description}</p>
      </div>
    </button>
  )
}

/** Role tab content with widget selector and save controls. */
function RoleDefaultEditor({ role }: { role: RoleKey }) {
  const { data: defaults, isLoading } = useDashboardDefaults(role)
  const saveMutation = useSaveDashboardDefaults()
  const dashboardLayouts = useDashboardStore((s) => s.layouts)
  const dashboardActiveWidgets = useDashboardStore((s) => s.activeWidgets)

  // Local state for editing the widget set
  const [localWidgets, setLocalWidgets] = useState<string[] | null>(null)
  const [saveSuccess, setSaveSuccess] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  // Effective active widgets (local edits take priority over server data)
  const activeWidgets = localWidgets ?? (defaults?.active_widgets as string[] | undefined) ?? []

  const toggleWidget = useCallback((widgetId: string) => {
    setLocalWidgets((prev) => {
      const current = prev ?? (defaults?.active_widgets as string[] | undefined) ?? []
      if (current.includes(widgetId)) {
        return current.filter((id) => id !== widgetId)
      }
      return [...current, widgetId]
    })
    setSaveSuccess(false)
    setSaveError(null)
  }, [defaults])

  const handleSave = useCallback(async () => {
    setSaveSuccess(false)
    setSaveError(null)

    // Build layout from defaults or generate a simple grid layout
    const currentLayout = (defaults?.layout as Array<Record<string, unknown>> | undefined) ?? []
    const widgetsToSave = localWidgets ?? activeWidgets

    // Filter layout to only include active widgets, add missing ones
    const layoutMap = new Map(currentLayout.map((l) => [l.i as string, l]))
    const newLayout: Array<Record<string, unknown>> = []
    let col = 0
    let row = 0

    for (const widgetId of widgetsToSave) {
      if (layoutMap.has(widgetId)) {
        newLayout.push(layoutMap.get(widgetId)!)
      } else {
        // Generate a position for newly added widgets
        const widgetDef = widgetList.find((w) => w.id === widgetId)
        newLayout.push({
          i: widgetId,
          x: col,
          y: row,
          w: widgetDef?.defaultSize.w ?? 4,
          h: widgetDef?.defaultSize.h ?? 3,
        })
        col += widgetDef?.defaultSize.w ?? 4
        if (col >= 12) {
          col = 0
          row += 4
        }
      }
    }

    try {
      await saveMutation.mutateAsync({
        role,
        layout: newLayout,
        active_widgets: widgetsToSave,
      })
      setLocalWidgets(null)
      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 3000)
    } catch {
      setSaveError('Speichern fehlgeschlagen. Bitte erneut versuchen.')
    }
  }, [role, defaults, localWidgets, activeWidgets, saveMutation])

  const handleCopyCurrentLayout = useCallback(async () => {
    setSaveSuccess(false)
    setSaveError(null)

    try {
      await saveMutation.mutateAsync({
        role,
        layout: dashboardLayouts.map((l) => ({
          i: l.i,
          x: l.x,
          y: l.y,
          w: l.w,
          h: l.h,
        })),
        active_widgets: dashboardActiveWidgets,
      })
      setLocalWidgets(null)
      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 3000)
    } catch {
      setSaveError('Speichern fehlgeschlagen. Bitte erneut versuchen.')
    }
  }, [role, dashboardLayouts, dashboardActiveWidgets, saveMutation])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <p className="text-sm text-muted-foreground">Lade Standard-Layout...</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Widget selector */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Widgets</CardTitle>
          <CardDescription>
            Waehlen Sie die Widgets aus, die im Standard-Dashboard fuer diese Rolle angezeigt werden.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-3">
            {widgetList.map((widget) => (
              <WidgetToggle
                key={widget.id}
                widgetId={widget.id}
                name={widget.name}
                description={widget.description}
                isActive={activeWidgets.includes(widget.id)}
                onToggle={toggleWidget}
              />
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Actions */}
      <div className="flex items-center gap-3">
        <Button onClick={handleSave} disabled={saveMutation.isPending}>
          <Save className="mr-2 h-4 w-4" />
          Speichern
        </Button>
        <Button variant="outline" onClick={handleCopyCurrentLayout} disabled={saveMutation.isPending}>
          <RotateCcw className="mr-2 h-4 w-4" />
          Aktuelle Ansicht als Standard speichern
        </Button>

        {saveSuccess && (
          <span className="flex items-center gap-1 text-sm text-green-600">
            <Check className="h-4 w-4" />
            Gespeichert
          </span>
        )}
        {saveError && (
          <span className="flex items-center gap-1 text-sm text-destructive">
            <AlertCircle className="h-4 w-4" />
            {saveError}
          </span>
        )}
      </div>
    </div>
  )
}

export default function DashboardSettings() {
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin')

  // Redirect non-admins
  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-4xl px-6 py-6">
        <div className="mb-6">
          <h1 className="text-2xl font-semibold text-foreground">Dashboard-Einstellungen</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Konfigurieren Sie die Standard-Dashboards fuer jede Benutzerrolle.
            Benutzer koennen ihr Dashboard individuell anpassen; hier legen Sie die Ausgangsbasis fest.
          </p>
        </div>

        <Tabs defaultValue="admin">
          <TabsList>
            {ROLES.map((role) => (
              <TabsTrigger key={role.key} value={role.key}>
                {role.label}
              </TabsTrigger>
            ))}
          </TabsList>

          {ROLES.map((role) => (
            <TabsContent key={role.key} value={role.key}>
              <p className="mb-4 text-sm text-muted-foreground">{role.description}</p>
              <RoleDefaultEditor role={role.key} />
            </TabsContent>
          ))}
        </Tabs>
      </div>
    </div>
  )
}
