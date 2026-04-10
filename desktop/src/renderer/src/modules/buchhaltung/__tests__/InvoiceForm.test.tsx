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

    expect(screen.getByText('buchhaltung.newInvoice')).toBeInTheDocument()
    // Should have one default line item row
    expect(screen.getByPlaceholderText('buchhaltung.form.descriptionDots')).toBeInTheDocument()
    expect(screen.getByText('buchhaltung.form.addPosition')).toBeInTheDocument()
  })

  it('renders as quote form when defaultType is quote', () => {
    renderInvoiceForm({ defaultType: 'quote' })
    expect(screen.getByText('buchhaltung.newQuote')).toBeInTheDocument()
  })

  it('adds a new line item when clicking add button', async () => {
    const user = userEvent.setup()
    renderInvoiceForm()

    // Initially one line item
    const descriptions = screen.getAllByPlaceholderText('buchhaltung.form.descriptionDots')
    expect(descriptions).toHaveLength(1)

    // Click "Position hinzufügen"
    await user.click(screen.getByText('buchhaltung.form.addPosition'))

    // Now two line items
    const updatedDescriptions = screen.getAllByPlaceholderText('buchhaltung.form.descriptionDots')
    expect(updatedDescriptions).toHaveLength(2)
  })

  it('disables save button without client name', () => {
    renderInvoiceForm()

    const saveBtn = screen.getByRole('button', { name: 'common.create' })
    expect(saveBtn).toBeDisabled()
  })

  it('enables save button after entering client name', async () => {
    const user = userEvent.setup()
    renderInvoiceForm()

    await user.type(screen.getByPlaceholderText('buchhaltung.form.clientPlaceholder'), 'Test AG')

    const saveBtn = screen.getByRole('button', { name: 'common.create' })
    expect(saveBtn).toBeEnabled()
  })

  it('displays totals section with Zwischensumme, MwSt, Gesamt', () => {
    renderInvoiceForm()

    expect(screen.getByText('buchhaltung.totals.subtotal')).toBeInTheDocument()
    expect(screen.getAllByText('buchhaltung.form.vat').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('buchhaltung.totals.total')).toBeInTheDocument()
  })
})
