import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { InviteMemberDialog } from '../InviteMemberDialog'

// Mock sonner toast
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}))

// Mock HR hooks at the hook level (custom fetch client, not openapi-fetch)
const mockMutate = vi.fn()
const mockCreateEmployee = {
  mutate: mockMutate,
  isPending: false,
}

vi.mock('@/api/hooks/hr-hooks', () => ({
  useCreateEmployee: () => mockCreateEmployee,
  useEmployees: () => ({
    data: {
      employees: [
        { id: 'emp-001', firstName: 'Anna', lastName: 'Beispiel', email: 'anna@firma.de', department: 'Engineering' },
        { id: 'emp-002', firstName: 'Hans', lastName: 'Test', email: 'hans@firma.de', department: 'Marketing' },
      ],
      total: 2,
    },
    isLoading: false,
    error: null,
  }),
  useLeaveRequests: () => ({ data: { requests: [] }, isLoading: false }),
  useUpdateEmployee: () => ({ mutate: vi.fn(), isPending: false }),
}))

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
}

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = createQueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  )
}

// ---------------------------------------------------------------------------
// InviteMemberDialog
// ---------------------------------------------------------------------------

describe('InviteMemberDialog', () => {
  let onOpenChange: ReturnType<typeof vi.fn>

  beforeEach(() => {
    onOpenChange = vi.fn()
    mockMutate.mockReset()
  })

  it('renders invite form with empty fields', () => {
    renderWithProviders(
      <InviteMemberDialog open={true} onOpenChange={onOpenChange} />,
    )

    expect(screen.getByText('team.invite.title')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('team.invite.firstName')).toHaveValue('')
    expect(screen.getByPlaceholderText('team.invite.lastName')).toHaveValue('')
    expect(screen.getByPlaceholderText('team.invite.emailPlaceholder')).toHaveValue('')
  })

  it('submits invite with required fields', async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <InviteMemberDialog open={true} onOpenChange={onOpenChange} />,
    )

    await user.type(screen.getByPlaceholderText('team.invite.firstName'), 'Peter')
    await user.type(screen.getByPlaceholderText('team.invite.lastName'), 'Müller')
    await user.type(screen.getByPlaceholderText('team.invite.emailPlaceholder'), 'peter@firma.de')

    await user.click(screen.getByRole('button', { name: 'team.invite.sendInvitation' }))

    expect(mockMutate).toHaveBeenCalledTimes(1)
    const callArgs = mockMutate.mock.calls[0][0]
    expect(callArgs.firstName).toBe('Peter')
    expect(callArgs.lastName).toBe('Müller')
    expect(callArgs.email).toBe('peter@firma.de')
  })

  it('disables submit without required fields', () => {
    renderWithProviders(
      <InviteMemberDialog open={true} onOpenChange={onOpenChange} />,
    )

    const submitBtn = screen.getByRole('button', { name: 'team.invite.sendInvitation' })
    expect(submitBtn).toBeDisabled()
  })

  it('shows loading state while submitting', () => {
    mockCreateEmployee.isPending = true
    renderWithProviders(
      <InviteMemberDialog open={true} onOpenChange={onOpenChange} />,
    )

    expect(screen.getByRole('button', { name: 'team.invite.sending' })).toBeInTheDocument()
    mockCreateEmployee.isPending = false
  })

  it('closes dialog on cancel', async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <InviteMemberDialog open={true} onOpenChange={onOpenChange} />,
    )

    await user.click(screen.getByRole('button', { name: 'common.cancel' }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})
