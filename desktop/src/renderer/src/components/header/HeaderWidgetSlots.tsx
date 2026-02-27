/**
 * Header widget slots — renders up to 3 configurable mini-widgets in the header bar.
 *
 * Reads active widget IDs from useUIStore.headerWidgets and renders
 * matching components from the header widget registry.
 */
import { useUIStore } from '@/stores/ui'
import { headerWidgetRegistry } from './header-widgets'

export function HeaderWidgetSlots() {
  const headerWidgets = useUIStore((s) => s.headerWidgets)

  if (!headerWidgets || headerWidgets.length === 0) return null

  return (
    <div className="hidden sm:flex items-center gap-0.5 rounded-lg border border-border/50 bg-secondary/30 px-1 py-0.5">
      {headerWidgets.map((widgetId) => {
        const def = headerWidgetRegistry[widgetId]
        if (!def) return null
        const Widget = def.component
        return <Widget key={widgetId} />
      })}
    </div>
  )
}
