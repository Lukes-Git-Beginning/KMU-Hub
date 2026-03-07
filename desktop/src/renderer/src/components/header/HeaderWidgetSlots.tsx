/**
 * Header widget slots — 3 fixed, equal-size mini-widget slots in the header bar.
 *
 * Always renders exactly 3 slots. Each slot shows its assigned widget
 * or stays empty. Widget assignment is read from useUIStore.headerWidgets.
 */
import { useUIStore } from '@/stores/ui'
import { headerWidgetRegistry } from './header-widgets'

export function HeaderWidgetSlots() {
  const headerWidgets = useUIStore((s) => s.headerWidgets)

  // Always 3 slots — fill from store, pad with null
  const slots = [
    headerWidgets?.[0] ?? null,
    headerWidgets?.[1] ?? null,
    headerWidgets?.[2] ?? null,
  ]

  return (
    <div className="hidden md:grid grid-cols-3 gap-2 w-full">
      {slots.map((widgetId, i) => {
        const def = widgetId ? headerWidgetRegistry[widgetId] : null
        if (!def) {
          // Empty slot — subtle placeholder
          return (
            <div
              key={`empty-${i}`}
              className="flex h-9 items-center justify-center rounded-lg border border-dashed border-border/20 bg-secondary/30"
            />
          )
        }
        const Widget = def.component
        return (
          <div
            key={widgetId}
            className="flex h-9 items-center justify-center rounded-lg border border-border/30 bg-secondary/20"
          >
            <Widget />
          </div>
        )
      })}
    </div>
  )
}
