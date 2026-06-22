import '@testing-library/jest-dom'
import { cleanup } from '@testing-library/react'
import { afterEach, afterAll, beforeAll, vi } from 'vitest'
import { server } from './handlers'

// Start MSW server before all tests
beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => {
  cleanup()
  server.resetHandlers()
})
afterAll(() => server.close())

// Mock window.electronAPI (Electron IPC bridge)
Object.defineProperty(window, 'electronAPI', {
  value: {
    auth: {
      getStoredTokens: vi.fn().mockResolvedValue(null),
      storeTokens: vi.fn().mockResolvedValue(undefined),
      clearTokens: vi.fn().mockResolvedValue(undefined),
    },
    platform: 'win32',
  },
  writable: true,
})

// Mock navigator.onLine
Object.defineProperty(navigator, 'onLine', {
  value: true,
  writable: true,
})

// Polyfill ResizeObserver — jsdom lacks it, but Radix UI primitives
// (e.g. Checkbox) reference it on render and throw without this stub.
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

// Polyfill scrollIntoView — jsdom lacks it, but components that auto-scroll
// (chat panels, autocomplete, time pickers) call it on render and throw without this stub.
Element.prototype.scrollIntoView = vi.fn()

// Mock import.meta.env
vi.stubEnv('RENDERER_VITE_API_URL', 'http://localhost:8080')

// Mock WebSocket manager (no real WS in tests)
vi.mock('@/api/websocket', () => ({
  wsManager: {
    connect: vi.fn(),
    disconnect: vi.fn(),
  },
}))

// Mock react-i18next to return translation keys directly
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: 'de', changeLanguage: vi.fn() },
  }),
  I18nextProvider: ({ children }: { children: React.ReactNode }) => children,
  Trans: ({ children }: { children: React.ReactNode }) => children,
}))

// Mock react-router-dom navigate
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

// Export for test assertions
export { mockNavigate }
