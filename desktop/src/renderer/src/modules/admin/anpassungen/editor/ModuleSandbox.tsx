/**
 * ModuleSandbox (Modul-Editor v1, E-2) — renders a pilot module read-only inside
 * an isolated context so it never touches the live app.
 *
 * Isolation (IST-EDITOR §Achse 1/5, refined by E-2 QA):
 *   - own QueryClient      → sandbox fetches don't pollute the live cache
 *   - ambient app router   → the module reuses the app's router that flows through
 *                            the Radix portal. NOTE: React Router v6 forbids nesting
 *                            a <Router> inside the root RouterProvider, so a
 *                            MemoryRouter is NOT usable for the in-app overlay (it
 *                            stays the path for the future real-window mode).
 *   - pointer-events: none → view-only preview (v1 edits via the trio panel, not
 *                            by clicking the canvas) → sidesteps store/navigation
 *                            side-effects (R-2/R-4)
 *   - ErrorBoundary        → a module that dislikes isolation degrades to a
 *                            "preview unavailable" note instead of crashing the editor
 *
 * Labels update live because the module reads t() from the shared i18n instance,
 * which the DraftConfigProvider re-overlays on every draft change (ICU-Live-Fix).
 */
import { Component, Suspense, useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { EyeOff } from 'lucide-react'
import { STALE_TIME } from '@/lib/constants'
import { i18n } from '@/i18n/i18n'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import {
  EditorSurfaceProvider,
  type EditorSurfaceValue,
  type EditorFocusSection,
} from '@/components/customization/EditorSurface'
import { useDraftConfig } from './DraftConfigProvider'
import type { EditorModuleDef } from './editorModules'

/** Stable fallback so a sandbox without a listener never re-memoises the surface. */
const noopReport = (): void => {}

class SandboxErrorBoundary extends Component<
  { fallback: ReactNode; children: ReactNode },
  { hasError: boolean }
> {
  state = { hasError: false }
  static getDerivedStateFromError(): { hasError: boolean } {
    return { hasError: true }
  }
  render(): ReactNode {
    return this.state.hasError ? this.props.fallback : this.props.children
  }
}

export function ModuleSandbox({
  module,
  focusSection = null,
  focusNonce = 0,
  onContextChange,
  fitToWidth = true,
}: {
  module: EditorModuleDef
  /** Left-rail section the user selected — the module navigates itself there. */
  focusSection?: EditorFocusSection | null
  focusNonce?: number
  /** The module reporting where the user now is — the rail follows it back. */
  onContextChange?: (section: EditorFocusSection | null) => void
  /**
   * Shrink the preview until it fits the canvas (Darien 2026-08-09: „Zugewiesen
   * an, SLA und Erstellt am sieht man gar nicht" — the list was wider than the
   * canvas and scrolled off to the right, which is the worst place to hide
   * columns while someone is configuring columns). Off = the module renders at
   * its true size and the canvas scrolls.
   */
  fitToWidth?: boolean
}): ReactNode {
  const { t } = useTranslation()
  const {
    labels,
    setDraftLabel,
    valueSets,
    moduleAreas,
    patchDraftModuleArea,
    patchDraftModuleAreas,
    valueSetMigrations,
    customFields,
  } = useDraftConfig()
  const [sandboxClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { staleTime: STALE_TIME, retry: 1, refetchOnWindowFocus: false },
        },
      }),
  )
  const { Component: ModulePage } = module

  // Edit-in-place surface: EditableText elements inside the module become
  // click-to-rename, writing to the draft in the active locale.
  const surface = useMemo<EditorSurfaceValue>(
    () => ({
      editing: true,
      setLabel: (key, value) => setDraftLabel(i18n.language, key, value),
      isDraft: (key) => Boolean(labels[i18n.language]?.[key]),
      valueSets,
      moduleAreas,
      // Column widths are dragged inside the preview, so the write path has to
      // reach the draft from the module — same idea as setLabel for rename.
      setAreaLayout: (areaKey, patch) => patchDraftModuleArea(module.key, areaKey, patch),
      setAreaLayouts: (patches, focusKey) => patchDraftModuleAreas(module.key, patches, focusKey),
      valueSetMigrations,
      customFields,
      focusSection,
      focusNonce,
      reportContext: onContextChange ?? noopReport,
    }),
    [
      labels,
      setDraftLabel,
      valueSets,
      moduleAreas,
      patchDraftModuleArea,
      patchDraftModuleAreas,
      module.key,
      valueSetMigrations,
      customFields,
      focusSection,
      focusNonce,
      onContextChange,
    ],
  )

  const fallback = (
    <div className="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground">
      <EyeOff className="h-6 w-6" aria-hidden="true" />
      <p className="text-sm">{t('customization.editor.previewUnavailable')}</p>
    </div>
  )

  // How much the preview has to shrink to fit. Measured, not guessed: the module
  // decides its own minimum width (a wide table, a fixed sidebar), and that
  // changes when columns are toggled.
  const canvasRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)
  const [scale, setScale] = useState(1)
  useEffect(() => {
    if (!fitToWidth) {
      setScale(1)
      return
    }
    const canvas = canvasRef.current
    const content = contentRef.current
    if (!canvas || !content) return
    const measure = (): void => {
      const available = canvas.clientWidth
      // `transform: scale` does not change layout, so scrollWidth stays the
      // module's natural width no matter what the current scale is — which keeps
      // this measurement from chasing its own tail.
      const needed = content.scrollWidth
      if (available <= 0 || needed <= 0) return
      // Never shrink below 60%: past that it stops being a preview you can judge.
      const next = Math.min(1, Math.max(0.6, available / needed))
      // Ignore sub-pixel noise, otherwise the observer keeps re-triggering.
      if (Math.abs(next - scale) > 0.01) setScale(next)
    }
    measure()
    const observer = new ResizeObserver(measure)
    observer.observe(canvas)
    observer.observe(content)
    return () => observer.disconnect()
  }, [fitToWidth, scale])

  return (
    <div ref={canvasRef} className="relative h-full w-full overflow-auto bg-muted/20">
      {/* The window chrome (title "Sandbox-Vorschau · nicht live" + amber banner)
          already frames this as a preview, so no in-canvas label is needed. */}
      <div
        ref={contentRef}
        // w-max + min-w-full: the module lays itself out at its natural width
        // (that is what gets measured), never narrower than the canvas.
        className="min-h-full w-max min-w-full"
        style={
          scale < 1
            ? {
                // transform, not width/zoom: GPU-composited, and every ratio the
                // module measures itself (column drags read pixel shares) stays
                // proportional, so nothing shifts meaning.
                transform: `scale(${scale})`,
                transformOrigin: 'top left',
              }
            : undefined
        }
      >
        {/* Navigable preview: you walk the real module (state-based tabs + detail
            modals work) and edit EditableText elements in place. */}
        <SandboxErrorBoundary fallback={fallback}>
          <QueryClientProvider client={sandboxClient}>
            <EditorSurfaceProvider value={surface}>
              <Suspense fallback={<ModuleLoadingFallback />}>
                <ModulePage />
              </Suspense>
            </EditorSurfaceProvider>
          </QueryClientProvider>
        </SandboxErrorBoundary>
      </div>
    </div>
  )
}
