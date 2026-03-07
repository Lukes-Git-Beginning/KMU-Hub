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

// Mock import.meta.env
vi.stubEnv('RENDERER_VITE_API_URL', 'http://localhost:8080')

// Mock WebSocket manager (no real WS in tests)
vi.mock('@/api/websocket', () => ({
  wsManager: {
    connect: vi.fn(),
    disconnect: vi.fn(),
  },
}))

// Mock react-intl to return message IDs directly
vi.mock('react-intl', () => ({
  FormattedMessage: ({ id }: { id: string }) => id,
  useIntl: () => ({
    formatMessage: ({ id }: { id: string }) => id,
  }),
  IntlProvider: ({ children }: { children: React.ReactNode }) => children,
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
