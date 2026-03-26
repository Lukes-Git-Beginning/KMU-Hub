/**
 * Demo Mode Bootstrap — starts MSW browser worker to intercept all API calls
 * with realistic mock data. Activated via RENDERER_VITE_DEMO_MODE=true.
 *
 * Dynamic imports ensure MSW is tree-shaken from production bundles.
 */

export const DEMO_MODE = import.meta.env.RENDERER_VITE_DEMO_MODE === 'true'

export async function startDemoMode(): Promise<void> {
  if (!DEMO_MODE) return

  // Clear stale query cache so fresh mock data is loaded
  try {
    localStorage.removeItem('kmuhub-query-cache')
  } catch {
    // localStorage may be unavailable
  }

  const { setupWorker } = await import('msw/browser')
  const { handlers } = await import('./handlers')

  const worker = setupWorker(...handlers)
  await worker.start({
    onUnhandledRequest: 'bypass',
    quiet: false,
  })

  console.info('[Demo Mode] MSW worker started — all API calls are mocked')
}
