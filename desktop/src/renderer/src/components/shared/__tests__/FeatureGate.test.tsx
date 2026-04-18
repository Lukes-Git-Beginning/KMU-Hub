import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { FeatureGate } from '../FeatureGate'

// Mock useFeatureFlags so tests run without a real API
vi.mock('@/api/hooks/useFeatureFlags', () => ({
  useFeatureFlags: vi.fn(),
}))

import { useFeatureFlags } from '@/api/hooks/useFeatureFlags'

const mockUseFeatureFlags = vi.mocked(useFeatureFlags)

function wrap(ui: React.ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>)
}

describe('FeatureGate', () => {
  it('renders children when flag is enabled', () => {
    mockUseFeatureFlags.mockReturnValue({
      flags: { 'modules.wiki': true },
      isLoading: false,
      error: null,
      isEnabled: (key: string) => key === 'modules.wiki',
    })

    wrap(
      <FeatureGate flag="modules.wiki">
        <span data-testid="child">Wiki link</span>
      </FeatureGate>,
    )

    expect(screen.getByTestId('child')).toBeInTheDocument()
  })

  it('renders fallback when flag is disabled', () => {
    mockUseFeatureFlags.mockReturnValue({
      flags: { 'modules.wiki': false },
      isLoading: false,
      error: null,
      isEnabled: () => false,
    })

    wrap(
      <FeatureGate flag="modules.wiki" fallback={<span data-testid="fallback">Hidden</span>}>
        <span data-testid="child">Wiki link</span>
      </FeatureGate>,
    )

    expect(screen.queryByTestId('child')).not.toBeInTheDocument()
    expect(screen.getByTestId('fallback')).toBeInTheDocument()
  })

  it('renders fallback (null by default) while flags are loading', () => {
    mockUseFeatureFlags.mockReturnValue({
      flags: undefined,
      isLoading: true,
      error: null,
      isEnabled: () => false,
    })

    wrap(
      <FeatureGate flag="modules.wiki">
        <span data-testid="child">Wiki link</span>
      </FeatureGate>,
    )

    expect(screen.queryByTestId('child')).not.toBeInTheDocument()
  })

  it('renders custom fallback while flags are loading', () => {
    mockUseFeatureFlags.mockReturnValue({
      flags: undefined,
      isLoading: true,
      error: null,
      isEnabled: () => false,
    })

    wrap(
      <FeatureGate flag="modules.wiki" fallback={<span data-testid="loading-placeholder">…</span>}>
        <span data-testid="child">Wiki link</span>
      </FeatureGate>,
    )

    expect(screen.getByTestId('loading-placeholder')).toBeInTheDocument()
    expect(screen.queryByTestId('child')).not.toBeInTheDocument()
  })
})
