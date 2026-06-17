/**
 * Admin settings page for configuring role-based default dashboards.
 *
 * Allows admins to configure which widgets appear in each role's
 * default dashboard and save the configuration to the server.
 * Non-admin users are redirected away (guarded by route).
 */
import { useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
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

const ROLES: { key: RoleKey; labelKey: string; descriptionKey: string }[] = [
  { key: 'admin', labelKey: 'settings.dashboard.roles.admin', descriptionKey: 'settings.dashboard.roles.adminDesc' },
  { key: 'manager', labelKey: 'settings.dashboard.roles.manager', descriptionKey: 'settings.dashboard.roles.managerDesc' },
  { key: 'member', labelKey: 'settings.dashboard.roles.member', descriptionKey: 'settings.dashboard.roles.memberDesc' },
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
  const { t } = useTranslation()
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
          {isActive && <Badge variant="secondary" className="text-xs">{t('common.active')}</Badge>}
        </div>
        <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">{description}</p>
      </div>
    </button>
  )
}

/** Role tab content with widget selector and save controls. */
function RoleDefaultEditor({ role }: { role: RoleKey }) {
  const { t } = useTranslation()
  const { data: defaults, isLoading } = useDashboardDefaults(role)
  const saveMutation = useSaveDashboardDefaults()
  const dashboardLayouts = useDashboardStore((s) => s.personalLayouts)
  const dashboardActiveWidgets = useDashboardStore((s) => s.personalActiveWidgets)

  // Local state for editing the widget set
  const [localWidgets, setLocalWidgets] = useState<string[] | null>(null)
  const [saveSuccess, setSaveSuccess] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  // Effective active widgets (local edits take priority over server data)
  const activeWidgets = useMemo(
    () => localWidgets ?? (defaults?.active_widgets as string[] | undefined) ?? [],
    [localWidgets, defaults?.active_widgets]
  )

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
      setSaveError(t('settings.dashboard.saveError'))
    }
  }, [role, defaults, localWidgets, activeWidgets, saveMutation, t])

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
      setSaveError(t('settings.dashboard.saveError'))
    }
  }, [role, dashboardLayouts, dashboardActiveWidgets, saveMutation, t])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <p className="text-sm text-muted-foreground">{t('settings.dashboard.loadingLayout')}</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Widget selector */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">{t('settings.dashboard.widgets')}</CardTitle>
          <CardDescription>
            {t('settings.dashboard.widgetsDescription')}
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
          {t('common.save')}
        </Button>
        <Button variant="outline" onClick={handleCopyCurrentLayout} disabled={saveMutation.isPending}>
          <RotateCcw className="mr-2 h-4 w-4" />
          {t('settings.dashboard.saveCurrentAsDefault')}
        </Button>

        {saveSuccess && (
          <span className="flex items-center gap-1 text-sm text-success">
            <Check className="h-4 w-4" />
            {t('common.saved')}
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
  const { t } = useTranslation()
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
          <h1 className="text-2xl font-semibold text-foreground">{t('settings.dashboard.title')}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {t('settings.dashboard.description')}
          </p>
        </div>

        <Tabs defaultValue="admin">
          <TabsList>
            {ROLES.map((role) => (
              <TabsTrigger key={role.key} value={role.key}>
                {t(role.labelKey)}
              </TabsTrigger>
            ))}
          </TabsList>

          {ROLES.map((role) => (
            <TabsContent key={role.key} value={role.key}>
              <p className="mb-4 text-sm text-muted-foreground">{t(role.descriptionKey)}</p>
              <RoleDefaultEditor role={role.key} />
            </TabsContent>
          ))}
        </Tabs>
      </div>
    </div>
  )
}
