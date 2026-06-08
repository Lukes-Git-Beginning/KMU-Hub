import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { DealFormDialog, type DealFormData } from '../DealFormDialog'
import DealsListPage from '../DealsListPage'
import DealPipelineView from '../DealPipelineView'

// Mock sonner toast
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

// Mock data
const mockPipelineStages = {
  stages: [
    { id: 'stg-lead', name: 'Lead', color: '#94a3b8', probability: 10, sort_order: 1 },
    { id: 'stg-qual', name: 'Qualifiziert', color: '#60a5fa', probability: 25, sort_order: 2 },
    { id: 'stg-prop', name: 'Angebot', color: '#a78bfa', probability: 50, sort_order: 3 },
    { id: 'stg-won', name: 'Gewonnen', color: '#22c55e', probability: 100, sort_order: 4, is_won: true },
  ],
}

const mockDealsData = {
  deals: [
    {
      id: 'dl-001',
      name: 'CRM Lizenz ABC',
      value: 25000,
      currency: 'CHF',
      stageId: 'stg-lead',
      stageName: 'Lead',
      contactName: 'Thomas Weber',
      expectedCloseDate: '2026-06-15',
      tags: [],
    },
    {
      id: 'dl-002',
      name: 'Support Vertrag',
      value: 5000,
      currency: 'EUR',
      stageId: 'stg-prop',
      stageName: 'Angebot',
      contactName: null,
      expectedCloseDate: null,
      tags: [],
    },
  ],
  total: 2,
  page: 1,
  page_size: 20,
}

// Mock hooks at hook level
let mockUseDealsReturn: Record<string, unknown> = {
  data: mockDealsData,
  isLoading: false,
  error: null,
  refetch: vi.fn(),
}

let mockUsePipelineStagesReturn: Record<string, unknown> = {
  data: mockPipelineStages,
  isLoading: false,
  error: null,
  refetch: vi.fn(),
}

const mockCreateDealMutateAsync = vi.fn().mockResolvedValue({ id: 'dl-new' })

vi.mock('@/api/hooks/useDeals', () => ({
  useDeals: () => mockUseDealsReturn,
  useCreateDeal: () => ({
    mutateAsync: mockCreateDealMutateAsync,
    isPending: false,
  }),
  useDeleteDeal: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useMoveDealToStage: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock('@/api/hooks/usePipelineStages', () => ({
  usePipelineStages: () => mockUsePipelineStagesReturn,
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
// DealFormDialog
// ---------------------------------------------------------------------------

describe('DealFormDialog', () => {
  let onSubmit: ReturnType<typeof vi.fn>
  let onOpenChange: ReturnType<typeof vi.fn>

  beforeEach(() => {
    onSubmit = vi.fn()
    onOpenChange = vi.fn()
  })

  it('renders create form with "Neuer Deal"', () => {
    renderWithProviders(
      <DealFormDialog open={true} onOpenChange={onOpenChange} onSubmit={onSubmit} />,
    )

    expect(screen.getByText('crm.deals.newTitle')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('z.B. CRM-Lizenz ABC GmbH')).toHaveValue('')
  })

  it('renders edit form with initial data', () => {
    renderWithProviders(
      <DealFormDialog
        open={true}
        onOpenChange={onOpenChange}
        onSubmit={onSubmit}
        isEdit
        initialData={{ name: 'Bestandsdeal', value: 10000 }}
      />,
    )

    expect(screen.getByText('crm.deals.editTitle')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('z.B. CRM-Lizenz ABC GmbH')).toHaveValue('Bestandsdeal')
  })

  it('submits deal with required fields', async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <DealFormDialog open={true} onOpenChange={onOpenChange} onSubmit={onSubmit} />,
    )

    await user.type(screen.getByPlaceholderText('z.B. CRM-Lizenz ABC GmbH'), 'Neuer Deal')
    await user.click(screen.getByRole('button', { name: 'common.create' }))

    expect(onSubmit).toHaveBeenCalledTimes(1)
    const submitted = onSubmit.mock.calls[0][0] as DealFormData
    expect(submitted.name).toBe('Neuer Deal')
  })

  it('disables submit without deal name', () => {
    renderWithProviders(
      <DealFormDialog open={true} onOpenChange={onOpenChange} onSubmit={onSubmit} />,
    )

    const submitBtn = screen.getByRole('button', { name: 'common.create' })
    expect(submitBtn).toBeDisabled()
  })

  it('shows weighted value when value is entered', () => {
    renderWithProviders(
      <DealFormDialog
        open={true}
        onOpenChange={onOpenChange}
        onSubmit={onSubmit}
        initialData={{ name: 'Test', value: 20000, probability: 50 }}
      />,
    )

    expect(screen.getByText('crm.deals.weightedValue')).toBeInTheDocument()
  })
})

// ---------------------------------------------------------------------------
// DealsListPage
// ---------------------------------------------------------------------------

describe('DealsListPage', () => {
  beforeEach(() => {
    mockUseDealsReturn = {
      data: mockDealsData,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    }
    mockUsePipelineStagesReturn = {
      data: mockPipelineStages,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    }
  })

  it('renders deals table from hook data', () => {
    renderWithProviders(<DealsListPage />)

    expect(screen.getByText('CRM Lizenz ABC')).toBeInTheDocument()
    expect(screen.getByText('Support Vertrag')).toBeInTheDocument()
  })

  it('shows empty state when no deals', () => {
    mockUseDealsReturn = {
      data: { deals: [], total: 0, page: 1, page_size: 20 },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    }

    renderWithProviders(<DealsListPage />)

    expect(screen.getByText('crm.deals.noResults')).toBeInTheDocument()
  })

  it('shows error state with retry button', () => {
    mockUseDealsReturn = {
      data: undefined,
      isLoading: false,
      error: new Error('Server error'),
      refetch: vi.fn(),
    }

    renderWithProviders(<DealsListPage />)

    expect(screen.getByText('crm.deals.loadError')).toBeInTheDocument()
    expect(screen.getByText('common.retry')).toBeInTheDocument()
  })

  it('opens create dialog on button click', async () => {
    const user = userEvent.setup()
    renderWithProviders(<DealsListPage />)

    await user.click(screen.getByText('crm.deals.new'))

    await waitFor(() => {
      expect(screen.getByPlaceholderText('z.B. CRM-Lizenz ABC GmbH')).toBeInTheDocument()
    })
  })
})

// ---------------------------------------------------------------------------
// DealPipelineView
// ---------------------------------------------------------------------------

describe('DealPipelineView', () => {
  beforeEach(() => {
    mockUseDealsReturn = {
      data: mockDealsData,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    }
    mockUsePipelineStagesReturn = {
      data: mockPipelineStages,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    }
  })

  it('renders pipeline columns from stages', () => {
    renderWithProviders(<DealPipelineView />)

    expect(screen.getByText('Lead')).toBeInTheDocument()
    expect(screen.getByText('Qualifiziert')).toBeInTheDocument()
    expect(screen.getByText('Angebot')).toBeInTheDocument()
    expect(screen.getByText('Gewonnen')).toBeInTheDocument()
  })

  it('groups deals into correct stage columns', () => {
    renderWithProviders(<DealPipelineView />)

    expect(screen.getByText('CRM Lizenz ABC')).toBeInTheDocument()
    expect(screen.getByText('Support Vertrag')).toBeInTheDocument()
  })

  it('shows empty placeholder for stages with no deals', () => {
    renderWithProviders(<DealPipelineView />)

    // Stages without deals show the no-deals key
    const emptyLabels = screen.getAllByText('crm.deals.noDeals')
    expect(emptyLabels.length).toBeGreaterThanOrEqual(1)
  })
})
