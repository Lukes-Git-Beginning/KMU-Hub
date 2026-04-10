import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { server } from '@/test/handlers'
import { ContactFormDialog, type ContactFormData } from '../ContactFormDialog'

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

describe('ContactFormDialog', () => {
  let onSubmit: ReturnType<typeof vi.fn>
  let onOpenChange: ReturnType<typeof vi.fn>

  beforeEach(() => {
    onSubmit = vi.fn()
    onOpenChange = vi.fn()
  })

  it('renders create form with empty fields', () => {
    renderWithProviders(
      <ContactFormDialog
        open={true}
        onOpenChange={onOpenChange}
        onSubmit={onSubmit}
      />,
    )

    expect(screen.getByText('crm.contacts.newTitle')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Max')).toHaveValue('')
    expect(screen.getByPlaceholderText('Muster')).toHaveValue('')
  })

  it('renders edit form with initial data', () => {
    renderWithProviders(
      <ContactFormDialog
        open={true}
        onOpenChange={onOpenChange}
        onSubmit={onSubmit}
        isEdit
        initialData={{
          firstName: 'Anna',
          lastName: 'Beispiel',
          email: 'anna@firma.de',
        }}
      />,
    )

    expect(screen.getByText('crm.contacts.editTitle')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Max')).toHaveValue('Anna')
    expect(screen.getByPlaceholderText('Muster')).toHaveValue('Beispiel')
    expect(screen.getByPlaceholderText('max@firma.de')).toHaveValue('anna@firma.de')
  })

  it('submits new contact with filled fields', async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <ContactFormDialog
        open={true}
        onOpenChange={onOpenChange}
        onSubmit={onSubmit}
      />,
    )

    await user.type(screen.getByPlaceholderText('Max'), 'Peter')
    await user.type(screen.getByPlaceholderText('Muster'), 'Meier')
    await user.type(screen.getByPlaceholderText('max@firma.de'), 'peter@meier.de')

    await user.click(screen.getByRole('button', { name: 'common.create' }))

    expect(onSubmit).toHaveBeenCalledTimes(1)
    const submitted = onSubmit.mock.calls[0][0] as ContactFormData
    expect(submitted.firstName).toBe('Peter')
    expect(submitted.lastName).toBe('Meier')
    expect(submitted.email).toBe('peter@meier.de')
  })

  it('prevents submission without required fields', async () => {
    renderWithProviders(
      <ContactFormDialog
        open={true}
        onOpenChange={onOpenChange}
        onSubmit={onSubmit}
      />,
    )

    // Submit button should be disabled without first/last name
    const submitBtn = screen.getByRole('button', { name: 'common.create' })
    expect(submitBtn).toBeDisabled()
  })

  it('closes dialog on cancel', async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <ContactFormDialog
        open={true}
        onOpenChange={onOpenChange}
        onSubmit={onSubmit}
      />,
    )

    await user.click(screen.getByRole('button', { name: 'common.cancel' }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})

describe('Contact API integration', () => {
  it('POST /api/v1/contacts creates a contact', async () => {
    let capturedBody: Record<string, unknown> | null = null
    server.use(
      http.post('http://localhost:8080/api/v1/contacts', async ({ request }) => {
        capturedBody = (await request.json()) as Record<string, unknown>
        return HttpResponse.json(
          { id: 'ct-new', ...capturedBody },
          { status: 201 },
        )
      }),
    )

    const res = await fetch('http://localhost:8080/api/v1/contacts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        first_name: 'Neu',
        last_name: 'Kontakt',
        email: 'neu@firma.de',
      }),
    })

    expect(res.status).toBe(201)
    const data = await res.json()
    expect(data.id).toBe('ct-new')
    expect(data.first_name).toBe('Neu')
  })

  it('DELETE /api/v1/contacts/:id returns 204', async () => {
    const res = await fetch('http://localhost:8080/api/v1/contacts/ct-001', {
      method: 'DELETE',
    })
    expect(res.status).toBe(204)
  })
})
