import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import DashboardPage from '../DashboardPage'

// Mock dashboard store
const mockEnsureDefaults = vi.fn()
const mockToggleEditing = vi.fn()
const mockResetToDefaults = vi.fn()
let mockIsEditing = false

vi.mock('@/stores/dashboard', () => ({
  useDashboardStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      isEditing: mockIsEditing,
      toggleEditing: mockToggleEditing,
      resetToDefaults: mockResetToDefaults,
      ensureDefaults: mockEnsureDefaults,
    }),
}))

// Mock heavy child components
vi.mock('@/components/dashboard', () => ({
  AlertsSection: () => <div data-testid="alerts-section" />,
  ModulesGrid: () => <div data-testid="modules-grid" />,
}))

vi.mock('@/components/widgets/WidgetContainer', () => ({
  default: () => <div data-testid="widget-container" />,
}))

vi.mock('@/components/dashboard/QuickActionsBar', () => ({
  QuickActionsBar: () => <div data-testid="quick-actions" />,
}))

vi.mock('@/components/dashboard/ProfileWidgetSuggestions', () => ({
  ProfileWidgetSuggestions: () => <div data-testid="profile-suggestions" />,
}))

vi.mock('@/components/shared/TextReveal', () => ({
  TextReveal: ({ text }: { text: string }) => <span>{text}</span>,
}))

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
}

function renderDashboard() {
  const queryClient = createQueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('DashboardPage', () => {
  it('renders greeting based on time of day', () => {
    renderDashboard()

    // One of the three greeting keys must be present
    const greetingKeys = ['dashboard.greeting.morning', 'dashboard.greeting.afternoon', 'dashboard.greeting.evening']
    const found = greetingKeys.some((g) => screen.queryByText(g) !== null)
    expect(found).toBe(true)
  })

  it('renders welcome message', () => {
    renderDashboard()

    expect(screen.getByText('dashboard.greeting.subtitle')).toBeInTheDocument()
  })

  it('renders edit button "Dashboard anpassen"', () => {
    mockIsEditing = false
    renderDashboard()

    expect(screen.getByText('dashboard.edit.customize')).toBeInTheDocument()
  })

  it('calls ensureDefaults on mount', () => {
    renderDashboard()

    expect(mockEnsureDefaults).toHaveBeenCalled()
  })

  it('renders all section containers', () => {
    renderDashboard()

    expect(screen.getByTestId('alerts-section')).toBeInTheDocument()
    expect(screen.getByTestId('modules-grid')).toBeInTheDocument()
    expect(screen.getByTestId('widget-container')).toBeInTheDocument()
    expect(screen.getByTestId('quick-actions')).toBeInTheDocument()
    expect(screen.getByTestId('profile-suggestions')).toBeInTheDocument()
  })
})
