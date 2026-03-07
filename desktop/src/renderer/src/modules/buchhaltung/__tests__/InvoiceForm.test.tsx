import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { InvoiceFormDialog } from '../InvoiceFormDialog'

// Mock the finance store
vi.mock('@/stores/finance', () => ({
  useFinanceStore: () => ({
    addInvoice: vi.fn(),
    updateInvoice: vi.fn(),
    nextInvoiceNum: 0,
  }),
  calcLineTotal: (item: { quantity: number; unitPrice: number; discount: number; vatRate: number }) => {
    const base = item.quantity * item.unitPrice
    const discounted = base * (1 - (item.discount || 0) / 100)
    return discounted * (1 + item.vatRate / 100)
  },
  calcInvoiceSubtotal: (items: Array<{ quantity: number; unitPrice: number; discount: number }>) =>
    items.reduce((sum, i) => sum + i.quantity * i.unitPrice * (1 - (i.discount || 0) / 100), 0),
  calcInvoiceTax: (items: Array<{ quantity: number; unitPrice: number; discount: number; vatRate: number }>) =>
    items.reduce((sum, i) => {
      const base = i.quantity * i.unitPrice * (1 - (i.discount || 0) / 100)
      return sum + base * (i.vatRate / 100)
    }, 0),
  calcInvoiceTotal: (items: Array<{ quantity: number; unitPrice: number; discount: number; vatRate: number }>) =>
    items.reduce((sum, i) => {
      const base = i.quantity * i.unitPrice * (1 - (i.discount || 0) / 100)
      return sum + base * (1 + i.vatRate / 100)
    }, 0),
}))

// Mock formatCurrency
vi.mock('@/lib/format', () => ({
  formatCurrency: (n: number) => `CHF ${n.toFixed(2)}`,
}))

// Mock sonner toast
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
}

function renderInvoiceForm(props: Partial<React.ComponentProps<typeof InvoiceFormDialog>> = {}) {
  const queryClient = createQueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <InvoiceFormDialog
          open={true}
          onOpenChange={vi.fn()}
          {...props}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('InvoiceFormDialog', () => {
  it('renders with default invoice type', () => {
    renderInvoiceForm()

    expect(screen.getByText('Neue Rechnung')).toBeInTheDocument()
    // Should have one default line item row
    expect(screen.getByPlaceholderText('Beschreibung...')).toBeInTheDocument()
    expect(screen.getByText(/Position hinzuf/)).toBeInTheDocument()
  })

  it('renders as quote form when defaultType is quote', () => {
    renderInvoiceForm({ defaultType: 'quote' })
    expect(screen.getByText('Neues Angebot')).toBeInTheDocument()
  })

  it('adds a new line item when clicking add button', async () => {
    const user = userEvent.setup()
    renderInvoiceForm()

    // Initially one line item
    const descriptions = screen.getAllByPlaceholderText('Beschreibung...')
    expect(descriptions).toHaveLength(1)

    // Click "Position hinzufuegen"
    await user.click(screen.getByText(/Position hinzuf/))

    // Now two line items
    const updatedDescriptions = screen.getAllByPlaceholderText('Beschreibung...')
    expect(updatedDescriptions).toHaveLength(2)
  })

  it('disables save button without client name', () => {
    renderInvoiceForm()

    const saveBtn = screen.getByRole('button', { name: 'Erstellen' })
    expect(saveBtn).toBeDisabled()
  })

  it('enables save button after entering client name', async () => {
    const user = userEvent.setup()
    renderInvoiceForm()

    await user.type(screen.getByPlaceholderText('Firma / Person'), 'Test AG')

    const saveBtn = screen.getByRole('button', { name: 'Erstellen' })
    expect(saveBtn).toBeEnabled()
  })

  it('displays totals section with Zwischensumme, MwSt, Gesamt', () => {
    renderInvoiceForm()

    expect(screen.getByText('Zwischensumme')).toBeInTheDocument()
    expect(screen.getAllByText('MwSt').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('Gesamt')).toBeInTheDocument()
  })
})
