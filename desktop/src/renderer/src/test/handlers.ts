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

const mockCompanies = {
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

const mockPipelineStages = {
  stages: [
    { id: 'stg-lead', name: 'Lead', color: '#94a3b8', probability: 10, sort_order: 1 },
    { id: 'stg-qual', name: 'Qualifiziert', color: '#60a5fa', probability: 25, sort_order: 2 },
    { id: 'stg-prop', name: 'Angebot', color: '#a78bfa', probability: 50, sort_order: 3 },
    { id: 'stg-won', name: 'Gewonnen', color: '#22c55e', probability: 100, sort_order: 4, is_won: true },
  ],
}

const mockDeals = {
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

const mockChannels = {
  channels: [
    { id: 'ch-001', name: 'allgemein', is_dm: false, is_private: false },
    { id: 'ch-002', name: 'development', is_dm: false, is_private: false },
  ],
}

const mockDMs = {
  channels: [
    { id: 'dm-001', name: 'Anna Beispiel', is_dm: true, other_user_id: 'usr-002' },
  ],
}

const mockUnread = {
  unread_counts: { 'ch-001': 3, 'dm-001': 1 },
}

const mockMessages = {
  messages: [
    {
      id: 'msg-001',
      content: 'Hallo Team!',
      channel_id: 'ch-001',
      sender_id: 'usr-001',
      sender_name: 'Max Muster',
      created_at: '2026-03-06T10:00:00Z',
      reply_count: 0,
    },
  ],
  has_more: false,
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

  // Companies
  http.get(`${API}/api/v1/companies`, () => {
    return HttpResponse.json(mockCompanies)
  }),

  http.post(`${API}/api/v1/companies`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ id: 'co-new', ...body, created_at: new Date().toISOString() }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/companies/:id`, async ({ request, params }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ id: params.id, ...body })
  }),

  http.delete(`${API}/api/v1/companies/:id`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // Deals
  http.get(`${API}/api/v1/deals`, () => {
    return HttpResponse.json(mockDeals)
  }),

  http.post(`${API}/api/v1/deals`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ id: 'dl-new', ...body, created_at: new Date().toISOString() }, { status: 201 })
  }),

  http.patch(`${API}/api/v1/deals/:id`, async ({ request, params }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ id: params.id, ...body })
  }),

  http.delete(`${API}/api/v1/deals/:id`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // Pipeline Stages
  http.get(`${API}/api/v1/pipeline-stages`, () => {
    return HttpResponse.json(mockPipelineStages)
  }),

  // Channels
  http.get(`${API}/api/v1/channels`, () => {
    return HttpResponse.json(mockChannels)
  }),

  http.get(`${API}/api/v1/channels/dm`, () => {
    return HttpResponse.json(mockDMs)
  }),

  http.get(`${API}/api/v1/channels/unread`, () => {
    return HttpResponse.json(mockUnread)
  }),

  http.post(`${API}/api/v1/channels`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({
      channel: { id: 'ch-new', ...body },
    }, { status: 201 })
  }),

  http.get(`${API}/api/v1/channels/:id/messages`, () => {
    return HttpResponse.json(mockMessages)
  }),

  http.post(`${API}/api/v1/channels/:id/messages`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({
      id: 'msg-new',
      ...body,
      sender_id: 'usr-001',
      sender_name: 'Max Muster',
      created_at: new Date().toISOString(),
    }, { status: 201 })
  }),
]

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

export const server = setupServer(...handlers)
export {
  mockUser,
  mockTokens,
  mockContacts,
  mockCompanies,
  mockDeals,
  mockPipelineStages,
  mockChannels,
  mockDMs,
  mockUnread,
  mockMessages,
}
