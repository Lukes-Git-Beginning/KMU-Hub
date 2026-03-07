import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

const API = 'http://localhost:8080'

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

const mockUser = {
  id: 'usr-001',
  email: 'test@firma.de',
  first_name: 'Max',
  last_name: 'Muster',
  roles: ['admin'],
}

const mockTokens = {
  access_token: 'mock-access-token',
  refresh_token: 'mock-refresh-token',
  user: mockUser,
}

const mockContacts = {
  contacts: [
    {
      id: 'ct-001',
      first_name: 'Anna',
      last_name: 'Beispiel',
      email: 'anna@firma.ch',
      phone: '+41 44 123 45 67',
      company_name: 'Beispiel AG',
      tags: [],
      created_at: '2026-01-15T10:00:00Z',
    },
    {
      id: 'ct-002',
      first_name: 'Hans',
      last_name: 'Test',
      email: 'hans@test.ch',
      phone: '+41 79 999 88 77',
      company_name: 'Test GmbH',
      tags: [],
      created_at: '2026-02-01T08:30:00Z',
    },
  ],
  total: 2,
  page: 1,
  page_size: 20,
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const handlers = [
  // Auth
  http.post(`${API}/api/v1/auth/login`, async ({ request }) => {
    const body = (await request.json()) as { email: string; password: string }
    if (body.email === 'test@firma.de' && body.password === 'correct') {
      return HttpResponse.json(mockTokens)
    }
    if (body.email === '2fa@firma.de') {
      return HttpResponse.json({
        requires_two_factor: true,
        pending_token: 'pending-2fa-token',
      })
    }
    return HttpResponse.json({ message: 'Invalid credentials' }, { status: 401 })
  }),

  http.post(`${API}/api/v1/auth/logout`, () => {
    return HttpResponse.json({ success: true })
  }),

  http.get(`${API}/api/v1/auth/me`, () => {
    return HttpResponse.json({ user: mockUser })
  }),

  http.post(`${API}/api/v1/auth/2fa/validate-login`, async ({ request }) => {
    const body = (await request.json()) as { pending_token: string; code: string }
    if (body.code === '123456') {
      return HttpResponse.json({
        access_token: 'mock-2fa-access-token',
        refresh_token: 'mock-2fa-refresh-token',
        user: mockUser,
      })
    }
    return HttpResponse.json({ message: 'Invalid code' }, { status: 401 })
  }),

  // Contacts
  http.get(`${API}/api/v1/contacts`, () => {
    return HttpResponse.json(mockContacts)
  }),

  http.post(`${API}/api/v1/contacts`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({
      id: 'ct-new',
      ...body,
      created_at: new Date().toISOString(),
    }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/contacts/:id`, async ({ request, params }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({
      id: params.id,
      ...body,
    })
  }),

  http.delete(`${API}/api/v1/contacts/:id`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // Finance - Invoices
  http.post(`${API}/api/v1/finance/invoices`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({
      id: 'inv-new',
      number: 'INV-2026-001',
      ...body,
      status: 'draft',
      created_at: new Date().toISOString(),
    }, { status: 201 })
  }),
]

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

export const server = setupServer(...handlers)
export { mockUser, mockTokens, mockContacts }
