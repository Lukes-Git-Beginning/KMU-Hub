/**
 * Single source of truth for the demo/mock-mode flag.
 *
 * Deliberately a leaf module with NO imports: module adapters that branch on
 * mock-vs-real (mock-exit) can read the flag without pulling the mock-handler
 * graph into their import/type graph. `demo-mode.ts` re-exports this so existing
 * `import { DEMO_MODE } from '@/mocks/demo-mode'` call-sites keep working.
 */
export const DEMO_MODE = import.meta.env.RENDERER_VITE_DEMO_MODE === 'true'
