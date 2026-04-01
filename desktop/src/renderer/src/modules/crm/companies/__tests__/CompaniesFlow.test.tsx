import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { CompanyFormDialog, type CompanyFormData } from '../CompanyFormDialog'
import CompaniesListPage from '../CompaniesListPage'

// Mock sonner toast
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

// Mock company hooks at hook level (avoids openapi-fetch + MSW issues)
const mockCompaniesData = {
  companies: [
    {
      id: 'co-001',
      name: 'Beispiel AG',
      website: 'beispiel.ch',
      industry: 'IT & Software',
      contactCount: 3,
      tags: [{ id: 't1', name: 'Key Account', color: '#22c55e' }],
      createdAt: '2026-01-15T10:00:00Z',
    },
    {
      id: 'co-002',
      name: 'Test GmbH',
      website: '',
      industry: 'Beratung',
      contactCount: 0,
      tags: [],
      createdAt: '2026-02-01T08:30:00Z',
    },
  ],
  total: 2,
  page: 1,
  page_size: 20,
}

let mockUseCompaniesReturn: Record<string, unknown> = {
  data: mockCompaniesData,
  isLoading: false,
  error: null,
  refetch: vi.fn(),
}

const mockMutateAsync = vi.fn().mockResolvedValue({ id: 'co-new' })

vi.mock('@/api/hooks/useCompanies', () => ({
  useCompanies: () => mockUseCompaniesReturn,
  useCreateCompany: () => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
  }),
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
// CompanyFormDialog
// ---------------------------------------------------------------------------

describe('CompanyFormDialog', () => {
  let onSubmit: ReturnType<typeof vi.fn>
  let onOpenChange: ReturnType<typeof vi.fn>

  beforeEach(() => {
    onSubmit = vi.fn()
    onOpenChange = vi.fn()
  })

  it('renders create form with empty fields', () => {
    renderWithProviders(
      <CompanyFormDialog open={true} onOpenChange={onOpenChange} onSubmit={onSubmit} />,
    )

    expect(screen.getByText('Neues Unternehmen')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Muster AG')).toHaveValue('')
  })

  it('renders edit form with initial data', () => {
    renderWithProviders(
      <CompanyFormDialog
        open={true}
        onOpenChange={onOpenChange}
        onSubmit={onSubmit}
        isEdit
        initialData={{ name: 'Beispiel AG', website: 'beispiel.ch' }}
      />,
    )

    expect(screen.getByText('Unternehmen bearbeiten')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Muster AG')).toHaveValue('Beispiel AG')
    expect(screen.getByPlaceholderText('www.firma.ch')).toHaveValue('beispiel.ch')
  })

  it('submits new company with required fields', async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <CompanyFormDialog open={true} onOpenChange={onOpenChange} onSubmit={onSubmit} />,
    )

    await user.type(screen.getByPlaceholderText('Muster AG'), 'Neue Firma GmbH')
    await user.click(screen.getByRole('button', { name: 'Erstellen' }))

    expect(onSubmit).toHaveBeenCalledTimes(1)
    const submitted = onSubmit.mock.calls[0][0] as CompanyFormData
    expect(submitted.name).toBe('Neue Firma GmbH')
  })

  it('disables submit button without company name', () => {
    renderWithProviders(
      <CompanyFormDialog open={true} onOpenChange={onOpenChange} onSubmit={onSubmit} />,
    )

    const submitBtn = screen.getByRole('button', { name: 'Erstellen' })
    expect(submitBtn).toBeDisabled()
  })

  it('adds and removes tags', async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <CompanyFormDialog open={true} onOpenChange={onOpenChange} onSubmit={onSubmit} />,
    )

    const tagInput = screen.getByPlaceholderText('Tag hinzufügen...')
    await user.type(tagInput, 'VIP')
    await user.keyboard('{Enter}')

    expect(screen.getByText('VIP')).toBeInTheDocument()

    // Remove the tag (X button next to tag text)
    const removeBtn = screen.getByText('VIP').closest('span')!.querySelector('button')!
    await user.click(removeBtn)

    expect(screen.queryByText('VIP')).not.toBeInTheDocument()
  })

  it('closes dialog on cancel', async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <CompanyFormDialog open={true} onOpenChange={onOpenChange} onSubmit={onSubmit} />,
    )

    await user.click(screen.getByRole('button', { name: 'Abbrechen' }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})

// ---------------------------------------------------------------------------
// CompaniesListPage
// ---------------------------------------------------------------------------

describe('CompaniesListPage', () => {
  beforeEach(() => {
    mockUseCompaniesReturn = {
      data: mockCompaniesData,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    }
  })

  it('renders company list from hook data', () => {
    renderWithProviders(<CompaniesListPage />)

    expect(screen.getByText('Beispiel AG')).toBeInTheDocument()
    expect(screen.getByText('Test GmbH')).toBeInTheDocument()
  })

  it('shows empty state when no companies', () => {
    mockUseCompaniesReturn = {
      data: { companies: [], total: 0, page: 1, page_size: 20 },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    }

    renderWithProviders(<CompaniesListPage />)

    expect(screen.getByText('Keine Unternehmen gefunden')).toBeInTheDocument()
  })

  it('shows error state with retry button', () => {
    mockUseCompaniesReturn = {
      data: undefined,
      isLoading: false,
      error: new Error('Server error'),
      refetch: vi.fn(),
    }

    renderWithProviders(<CompaniesListPage />)

    expect(screen.getByText('Fehler beim Laden der Unternehmen')).toBeInTheDocument()
    expect(screen.getByText('Erneut versuchen')).toBeInTheDocument()
  })

  it('shows create dialog when clicking new company button', async () => {
    const user = userEvent.setup()
    renderWithProviders(<CompaniesListPage />)

    await user.click(screen.getByText('Neues Unternehmen'))

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Muster AG')).toBeInTheDocument()
    })
  })
})
