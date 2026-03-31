import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import {
  mockContacts,
  mockCompanies,
  mockDeals,
  mockPipelineStages,
  mockActivities,
  getContactById,
  getCompanyById,
  getDealById,
} from '../data/contacts'

// ---------------------------------------------------------------------------
// Helper: transform snake_case contact to camelCase (matches OpenAPI schema)
// ---------------------------------------------------------------------------
function contactToCamel(c: Record<string, unknown>) {
  return {
    id: c.id,
    firstName: c.first_name,
    lastName: c.last_name,
    email: c.email,
    phone: c.phone,
    companyName: c.company_name,
    companyId: c.company_id,
    title: c.title,
    tags: c.tags,
    notes: c.notes,
    address: c.address,
    createdAt: c.created_at,
    updatedAt: c.updated_at ?? c.created_at,
  }
}

// ---------------------------------------------------------------------------
// Helper: enrich activity with resolved entity names
// ---------------------------------------------------------------------------
function enrichActivity(a: Record<string, unknown>) {
  const contact = a.contact_id ? getContactById(a.contact_id as string) : null
  const company = a.company_id ? getCompanyById(a.company_id as string) : null
  const deal = a.deal_id ? getDealById(a.deal_id as string) : null

  return {
    ...a,
    activity_type: a.type,
    contact_name: contact ? `${contact.first_name} ${contact.last_name}` : undefined,
    company_name: company ? company.name : undefined,
    deal_name: deal ? deal.name : undefined,
  }
}

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Helper: basic search filter across string fields
// ---------------------------------------------------------------------------
function matchesSearch<T extends Record<string, unknown>>(
  item: T,
  query: string,
  fields: (keyof T)[],
): boolean {
  const q = query.toLowerCase()
  return fields.some((f) => {
    const val = item[f]
    return typeof val === 'string' && val.toLowerCase().includes(q)
  })
}

// ---------------------------------------------------------------------------
// CRM Handlers
// ---------------------------------------------------------------------------
export const crmHandlers = [
  // ---- Contacts ---------------------------------------------------------

  http.get(`${API}/api/v1/contacts`, ({ request }) => {
    const url = new URL(request.url)
    const search = url.searchParams.get('search') ?? url.searchParams.get('q') ?? ''
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 50)

    let filtered = mockContacts.contacts
    if (search) {
      filtered = filtered.filter((c) =>
        matchesSearch(c, search, ['first_name', 'last_name', 'email', 'company_name', 'title']),
      )
    }

    const start = (page - 1) * pageSize
    const paged = filtered.slice(start, start + pageSize)

    return HttpResponse.json({
      contacts: paged.map((c) => contactToCamel(c as unknown as Record<string, unknown>)),
      total: filtered.length,
      page,
      page_size: pageSize,
    })
  }),

  http.get(`${API}/api/v1/contacts/:id`, ({ params }) => {
    const contact = getContactById(params.id as string)
    if (!contact) {
      return HttpResponse.json({ error: 'Contact not found' }, { status: 404 })
    }
    return HttpResponse.json(contactToCamel(contact as unknown as Record<string, unknown>))
  }),

  http.post(`${API}/api/v1/contacts`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const created = {
      id: `ct-new-${Date.now()}`,
      ...body,
      created_at: new Date().toISOString(),
    }
    return HttpResponse.json(created, { status: 201 })
  }),

  http.patch(`${API}/api/v1/contacts/:id`, async ({ params, request }) => {
    const existing = getContactById(params.id as string)
    if (!existing) {
      return HttpResponse.json({ error: 'Contact not found' }, { status: 404 })
    }
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ ...existing, ...body, updated_at: new Date().toISOString() })
  }),

  http.delete(`${API}/api/v1/contacts/:id`, ({ params }) => {
    const existing = getContactById(params.id as string)
    if (!existing) {
      return HttpResponse.json({ error: 'Contact not found' }, { status: 404 })
    }
    return new HttpResponse(null, { status: 204 })
  }),

  // ---- Companies --------------------------------------------------------

  http.get(`${API}/api/v1/companies`, ({ request }) => {
    const url = new URL(request.url)
    const search = url.searchParams.get('search') ?? url.searchParams.get('q') ?? ''
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 50)

    let filtered = mockCompanies.companies
    if (search) {
      filtered = filtered.filter((c) =>
        matchesSearch(c, search, ['name', 'industry', 'email']),
      )
    }

    const start = (page - 1) * pageSize
    const paged = filtered.slice(start, start + pageSize)

    return HttpResponse.json({
      companies: paged,
      total: filtered.length,
      page,
      page_size: pageSize,
    })
  }),

  http.get(`${API}/api/v1/companies/:id`, ({ params }) => {
    const company = getCompanyById(params.id as string)
    if (!company) {
      return HttpResponse.json({ error: 'Company not found' }, { status: 404 })
    }
    return HttpResponse.json(company)
  }),

  http.post(`${API}/api/v1/companies`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const created = {
      id: `co-new-${Date.now()}`,
      ...body,
      createdAt: new Date().toISOString(),
    }
    return HttpResponse.json(created, { status: 201 })
  }),

  http.patch(`${API}/api/v1/companies/:id`, async ({ params, request }) => {
    const existing = getCompanyById(params.id as string)
    if (!existing) {
      return HttpResponse.json({ error: 'Company not found' }, { status: 404 })
    }
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ ...existing, ...body, updatedAt: new Date().toISOString() })
  }),

  http.delete(`${API}/api/v1/companies/:id`, ({ params }) => {
    const existing = getCompanyById(params.id as string)
    if (!existing) {
      return HttpResponse.json({ error: 'Company not found' }, { status: 404 })
    }
    return new HttpResponse(null, { status: 204 })
  }),

  // ---- Deals ------------------------------------------------------------

  http.get(`${API}/api/v1/deals`, ({ request }) => {
    const url = new URL(request.url)
    const search = url.searchParams.get('search') ?? url.searchParams.get('q') ?? ''
    const stageId = url.searchParams.get('stage_id') ?? ''
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 50)

    let filtered = mockDeals.deals
    if (search) {
      filtered = filtered.filter((d) =>
        matchesSearch(d, search, ['name', 'contactName', 'companyName', 'stageName']),
      )
    }
    if (stageId) {
      filtered = filtered.filter((d) => d.stageId === stageId)
    }

    const start = (page - 1) * pageSize
    const paged = filtered.slice(start, start + pageSize)

    return HttpResponse.json({
      deals: paged,
      total: filtered.length,
      page,
      page_size: pageSize,
    })
  }),

  http.get(`${API}/api/v1/deals/:id`, ({ params }) => {
    const deal = getDealById(params.id as string)
    if (!deal) {
      return HttpResponse.json({ error: 'Deal not found' }, { status: 404 })
    }
    return HttpResponse.json(deal)
  }),

  http.post(`${API}/api/v1/deals`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const created = {
      id: `dl-new-${Date.now()}`,
      ...body,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json(created, { status: 201 })
  }),

  http.patch(`${API}/api/v1/deals/:id`, async ({ params, request }) => {
    const existing = getDealById(params.id as string)
    if (!existing) {
      return HttpResponse.json({ error: 'Deal not found' }, { status: 404 })
    }
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ ...existing, ...body, updated_at: new Date().toISOString() })
  }),

  http.delete(`${API}/api/v1/deals/:id`, ({ params }) => {
    const existing = getDealById(params.id as string)
    if (!existing) {
      return HttpResponse.json({ error: 'Deal not found' }, { status: 404 })
    }
    return new HttpResponse(null, { status: 204 })
  }),

  // ---- Pipeline Stages --------------------------------------------------

  http.get(`${API}/api/v1/pipeline-stages`, () => {
    // Enrich each stage with dealCount and totalValue from deals data
    const stages = mockPipelineStages.stages.map((stage) => {
      const stageDeals = mockDeals.deals.filter((d) => d.stageId === stage.id)
      return {
        ...stage,
        dealCount: stageDeals.length,
        totalValue: stageDeals.reduce((sum, d) => sum + d.value, 0),
      }
    })
    return HttpResponse.json({ stages })
  }),

  // ---- Activities -------------------------------------------------------

  http.get(`${API}/api/v1/activities`, ({ request }) => {
    const url = new URL(request.url)
    const search = url.searchParams.get('search') ?? url.searchParams.get('q') ?? ''
    const type = url.searchParams.get('type') ?? ''
    const contactId = url.searchParams.get('contact_id') ?? ''
    const dealId = url.searchParams.get('deal_id') ?? ''
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 50)

    let filtered = mockActivities.activities
    if (search) {
      filtered = filtered.filter((a) =>
        matchesSearch(a, search, ['subject', 'description']),
      )
    }
    if (type) {
      filtered = filtered.filter((a) => a.type === type)
    }
    if (contactId) {
      filtered = filtered.filter((a) => a.contact_id === contactId)
    }
    if (dealId) {
      filtered = filtered.filter((a) => a.deal_id === dealId)
    }

    // Sort by created_at descending when requested
    const sortBy = url.searchParams.get('sort_by')
    const sortDesc = url.searchParams.get('sort_desc') === 'true'
    if (sortBy === 'created_at') {
      filtered = [...filtered].sort((a, b) => {
        const diff = new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
        return sortDesc ? -diff : diff
      })
    }

    const start = (page - 1) * pageSize
    const paged = filtered.slice(start, start + pageSize)

    return HttpResponse.json({
      activities: paged.map((a) => enrichActivity(a as unknown as Record<string, unknown>)),
      total: filtered.length,
      page,
      page_size: pageSize,
    })
  }),

  http.get(`${API}/api/v1/activities/:id`, ({ params }) => {
    const activity = mockActivities.activities.find((a) => a.id === params.id)
    if (!activity) {
      return HttpResponse.json({ error: 'Activity not found' }, { status: 404 })
    }
    return HttpResponse.json(enrichActivity(activity as unknown as Record<string, unknown>))
  }),

  http.post(`${API}/api/v1/activities`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const created = {
      id: `act-new-${Date.now()}`,
      ...body,
      created_at: new Date().toISOString(),
    }
    return HttpResponse.json(created, { status: 201 })
  }),

  // ---- Global Search ----------------------------------------------------

  http.get(`${API}/api/v1/search`, ({ request }) => {
    const url = new URL(request.url)
    const query = url.searchParams.get('q') ?? url.searchParams.get('query') ?? ''

    if (!query) {
      return HttpResponse.json({ contacts: [], companies: [], deals: [] })
    }

    const matchedContacts = mockContacts.contacts.filter((c) =>
      matchesSearch(c, query, ['first_name', 'last_name', 'email', 'company_name']),
    )
    const matchedCompanies = mockCompanies.companies.filter((c) =>
      matchesSearch(c, query, ['name', 'industry']),
    )
    const matchedDeals = mockDeals.deals.filter((d) =>
      matchesSearch(d, query, ['name', 'contactName', 'companyName']),
    )

    return HttpResponse.json({
      contacts: matchedContacts.slice(0, 10).map((c) => contactToCamel(c as unknown as Record<string, unknown>)),
      companies: matchedCompanies.slice(0, 10),
      deals: matchedDeals.slice(0, 10),
    })
  }),
]
