/**
 * Widget grid container -- renders the react-grid-layout dashboard grid.
 *
 * Reads active widgets and layout from the dashboard store.
 * In edit mode, widgets are draggable and resizable. A floating
 * "Add Widget" button opens a picker dialog.
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import ReactGridLayout, { type Layout } from 'react-grid-layout'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { useDashboardStore } from '@/stores/dashboard'
import { widgetList, widgetRegistry } from './WidgetRegistry'
import { WidgetWrapper } from './WidgetWrapper'

import 'react-grid-layout/css/styles.css'

/** Debounce helper -- returns a callback that fires after delay. */
function useDebouncedLayoutUpdate(
  fn: (layout: Layout[]) => void,
  delayMs: number
): (layout: Layout[]) => void {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const fnRef = useRef(fn)
  useEffect(() => { fnRef.current = fn }, [fn])

  return useCallback(
    (layout: Layout[]) => {
      if (timerRef.current) clearTimeout(timerRef.current)
      timerRef.current = setTimeout(() => fnRef.current(layout), delayMs)
    },
    [delayMs]
  )
}

export default function WidgetContainer() {
  const layouts = useDashboardStore((s) => s.layouts)
  const activeWidgets = useDashboardStore((s) => s.activeWidgets)
  const isEditing = useDashboardStore((s) => s.isEditing)
  const updateLayout = useDashboardStore((s) => s.updateLayout)
  const addWidget = useDashboardStore((s) => s.addWidget)

  const [pickerOpen, setPickerOpen] = useState(false)

  // Track container width for responsive grid
  const containerRef = useRef<HTMLDivElement>(null)
  const [containerWidth, setContainerWidth] = useState(1200)

  useEffect(() => {
    const el = containerRef.current
    if (!el) return

    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setContainerWidth(entry.contentRect.width)
      }
    })
    observer.observe(el)
    return () => observer.disconnect()
  }, [])

  // Debounced layout save (500ms)
  const debouncedUpdateLayout = useDebouncedLayoutUpdate(
    (layout: Layout[]) => updateLayout(layout),
    500
  )

  const handleLayoutChange = useCallback(
    (newLayout: Layout[]) => {
      debouncedUpdateLayout(newLayout)
    },
    [debouncedUpdateLayout]
  )

  // Available widgets that are NOT currently active
  const availableWidgets = widgetList.filter(
    (w) => !activeWidgets.includes(w.id)
  )

  return (
    <div ref={containerRef} className="relative w-full">
      <ReactGridLayout
        className="layout"
        layout={layouts}
        cols={12}
        rowHeight={80}
        width={containerWidth}
        isDraggable={isEditing}
        isResizable={isEditing}
        draggableHandle=".widget-drag-handle"
        compactType="vertical"
        margin={[16, 16]}
        onLayoutChange={handleLayoutChange}
      >
        {activeWidgets.map((widgetId) => {
          const def = widgetRegistry[widgetId]
          if (!def) return null

          return (
            <div key={widgetId}>
              <WidgetWrapper widgetId={widgetId} isEditing={isEditing} />
            </div>
          )
        })}
      </ReactGridLayout>

      {/* Floating add button -- visible in edit mode when widgets are available */}
      {isEditing && availableWidgets.length > 0 && (
        <div className="sticky bottom-4 flex justify-center pb-4">
          <Button
            onClick={() => setPickerOpen(true)}
            className="shadow-lg"
          >
            <Plus className="mr-2 h-4 w-4" />
            Widget hinzufuegen
          </Button>
        </div>
      )}

      {/* Widget picker dialog */}
      <Dialog open={pickerOpen} onOpenChange={setPickerOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Widget hinzufuegen</DialogTitle>
            <DialogDescription>
              Waehlen Sie ein Widget fuer Ihr Dashboard aus.
            </DialogDescription>
          </DialogHeader>

          <div className="grid grid-cols-2 gap-3 pt-2">
            {widgetList.map((widget) => {
              const isActive = activeWidgets.includes(widget.id)
              const Icon = widget.icon

              return (
                <button
                  key={widget.id}
                  disabled={isActive}
                  onClick={() => {
                    addWidget(widget.id)
                    // Close picker if that was the last available widget
                    if (availableWidgets.length <= 1) {
                      setPickerOpen(false)
                    }
                  }}
                  className={`
                    flex items-start gap-3 rounded-lg border p-3 text-left transition-colors
                    ${
                      isActive
                        ? 'cursor-not-allowed border-border bg-muted/50 opacity-50'
                        : 'cursor-pointer border-border hover:border-primary/50 hover:bg-accent/50'
                    }
                  `}
                >
                  <div className="mt-0.5 rounded-md bg-primary/10 p-1.5">
                    <Icon className="h-4 w-4 text-primary" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium">{widget.name}</p>
                    <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">
                      {widget.description}
                    </p>
                    {isActive && (
                      <p className="mt-1 text-xs text-muted-foreground italic">
                        Bereits aktiv
                      </p>
                    )}
                  </div>
                </button>
              )
            })}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
