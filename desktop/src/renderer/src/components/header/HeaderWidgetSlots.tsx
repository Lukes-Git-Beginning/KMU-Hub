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
    <div className="hidden md:flex gap-2 w-full">
      {slots.map((widgetId, i) => {
        const def = widgetId ? headerWidgetRegistry[widgetId] : null

        // Progressive hiding when window shrinks:
        // 1st to go: slot 1 (weather)  — needs xl (1280px+)
        // 2nd to go: slot 2 (pomodoro) — needs lg (1024px+)
        // Always visible (from md): slot 0 (next-meeting)
        const responsiveClass =
          i === 1 ? 'hidden xl:flex' : i === 2 ? 'hidden lg:flex' : 'flex'

        if (!def) {
          // Empty slot — subtle placeholder
          return (
            <div
              key={`empty-${i}`}
              className={`${responsiveClass} h-9 flex-1 items-center justify-center rounded-lg border border-dashed border-border/20 bg-secondary/30`}
            />
          )
        }
        const Widget = def.component
        return (
          <div
            key={widgetId}
            className={`${responsiveClass} h-9 flex-1 items-center justify-center rounded-lg border border-border/30 bg-secondary/20`}
          >
            <Widget />
          </div>
        )
      })}
    </div>
  )
}
