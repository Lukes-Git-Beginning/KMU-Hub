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
import { Component, Suspense, useMemo, useState } from 'react'
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
}: {
  module: EditorModuleDef
  /** Left-rail section the user selected — the module navigates itself there. */
  focusSection?: EditorFocusSection | null
  focusNonce?: number
}): ReactNode {
  const { t } = useTranslation()
  const { labels, setDraftLabel, valueSets, moduleAreas, valueSetMigrations, customFields } = useDraftConfig()
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
      valueSetMigrations,
      customFields,
      focusSection,
      focusNonce,
    }),
    [labels, setDraftLabel, valueSets, moduleAreas, valueSetMigrations, customFields, focusSection, focusNonce],
  )

  const fallback = (
    <div className="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground">
      <EyeOff className="h-6 w-6" aria-hidden="true" />
      <p className="text-sm">{t('customization.editor.previewUnavailable')}</p>
    </div>
  )

  return (
    <div className="relative h-full w-full overflow-auto bg-muted/20">
      {/* The window chrome (title "Sandbox-Vorschau · nicht live" + amber banner)
          already frames this as a preview, so no in-canvas label is needed. */}
      <div className="min-h-full">
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
