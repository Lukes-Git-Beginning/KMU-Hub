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
import { IDS } from '../data/shared-ids'

// Tenant-wide contact tag definitions — mutable so the tag manager (CRM
// settings) can create/rename/recolour/delete them in demo mode.
const mockContactTags: { id: string; name: string; color: string; entityType: string }[] = [
  { id: 'tag-c1', name: 'Entscheider', color: '#EF4444', entityType: 'contact' },
  { id: 'tag-c2', name: 'VIP', color: '#F59E0B', entityType: 'contact' },
  { id: 'tag-c3', name: 'Technik', color: '#3B82F6', entityType: 'contact' },
  { id: 'tag-c5', name: 'Partner', color: '#8B5CF6', entityType: 'contact' },
  { id: 'tag-c7', name: 'Vertrieb', color: '#F59E0B', entityType: 'contact' },
  { id: 'tag-c9', name: 'Einkauf', color: '#10B981', entityType: 'contact' },
  { id: 'tag-c12', name: 'Beratung', color: '#8B5CF6', entityType: 'contact' },
  { id: 'tag-c17', name: 'Bestandskunde', color: '#10B981', entityType: 'contact' },
  { id: 'tag-c18', name: 'Neukunde', color: '#F59E0B', entityType: 'contact' },
  { id: 'tag-c19', name: 'Interessent', color: '#6366F1', entityType: 'contact' },
]

// Report references attached to contacts (R-5a) — demo-stateful, swap-ready.
const contactReportLinks: Array<{
  id: string
  contact_id: string
  filename: string
  mime_type: string
  storage_key: string
  created_at: string
}> = []
let contactFileCounter = 0

// GDPR consent per contact (route_crm_ext.go). Demo-stateful: a grant or a
// revocation survives a reload within the session, so the panel behaves the way
// it does against the real service instead of resetting to a canned state.
interface DemoConsent {
  id: string
  contact_id: string
  consent_type: string
  granted: boolean
  legal_basis: string
  source: string
  ip_address: string | null
  notes: string
  granted_at: string | null
  revoked_at: string | null
  created_by: string | null
  created_at: string
}

const consentsByContact: Record<string, Record<string, DemoConsent>> = {}
const consentHistory: Record<string, DemoConsent[]> = {}
let consentCounter = 0

const consentHistoryKey = (contactId: string, type: string) => `${contactId}:${type}`

function makeConsent(
  contactId: string,
  consentType: string,
  fields: Partial<DemoConsent>,
): DemoConsent {
  consentCounter += 1
  return {
    id: `consent-${consentCounter}`,
    contact_id: contactId,
    consent_type: consentType,
    granted: false,
    legal_basis: 'consent',
    source: 'web_form',
    ip_address: null,
    notes: '',
    granted_at: null,
    revoked_at: null,
    created_by: null,
    created_at: new Date().toISOString(),
    ...fields,
  }
}

/**
 * Seeds a plausible starting state the first time a contact is opened, hashed
 * off the id so different contacts do not all look identical. Only the demo
 * needs this — against the real service an unseen contact simply has no records.
 */
function ensureConsentSeed(contactId: string): void {
  if (consentsByContact[contactId]) return

  const variant = [...contactId].reduce((sum, ch) => sum + ch.charCodeAt(0), 0) % 3
  const store: Record<string, DemoConsent> = {}

  if (variant === 0) {
    store.marketing_email = makeConsent(contactId, 'marketing_email', {
      granted: true,
      source: 'email_confirmation',
      granted_at: '2026-03-14T09:12:00Z',
    })
    store.newsletter = makeConsent(contactId, 'newsletter', {
      granted: true,
      source: 'web_form',
      granted_at: '2026-03-14T09:12:00Z',
    })
  } else if (variant === 1) {
    store.marketing_email = makeConsent(contactId, 'marketing_email', {
      granted: true,
      source: 'contract',
      granted_at: '2026-01-08T14:30:00Z',
    })
    store.profiling = makeConsent(contactId, 'profiling', {
      granted: false,
      source: 'web_form',
      granted_at: '2025-11-02T10:00:00Z',
      revoked_at: '2026-05-20T16:45:00Z',
    })
  }

  consentsByContact[contactId] = store
  for (const [type, record] of Object.entries(store)) {
    consentHistory[consentHistoryKey(contactId, type)] = [record]
  }
}

// GDPR right-to-erasure (deletion request) per contact (route_crm_ext.go).
// Demo-stateful: filing a request marks it pending for the session; the
// (admin-only) process step flips it to completed -- mirrors the real
// two-step flow (HandleRequestDeletion / HandleProcessDeletion).
interface DemoDeletionRequest {
  id: string
  contact_id: string
  requested_by: string | null
  reason: string
  status: 'pending' | 'processing' | 'completed'
  completed_at: string | null
  created_at: string
}

const deletionRequestsById: Record<string, DemoDeletionRequest> = {}
let deletionRequestCounter = 0

// ---------------------------------------------------------------------------
// Helper: flatten a company.address object into a string. OpenAPI defines
// CompanyInfo.address as a string ("street, zip city, country"); the mock data
// keeps it structured for readability, so normalize on the way out.
// ---------------------------------------------------------------------------
function normalizeCompany<T extends { address?: unknown }>(c: T): T {
  const addr = c.address
  if (addr && typeof addr === 'object') {
    const a = addr as { street?: string; zip?: string; city?: string; country?: string }
    const parts = [
      a.street,
      [a.zip, a.city].filter(Boolean).join(' ').trim(),
      a.country,
    ].filter(Boolean)
    return { ...c, address: parts.join(', ') }
  }
  return c
}

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

  // Update ist serverseitig PUT (echtes Gateway); Mock spiegelt das, damit der
  // FE-Hook in beiden Modi dieselbe Methode nutzt. Die OpenAPI-Spec nennt
  // faelschlich PATCH (X-3).
  http.put(`${API}/api/v1/contacts/:id`, async ({ params, request }) => {
    const existing = getContactById(params.id as string)
    if (!existing) {
      return HttpResponse.json({ error: 'Contact not found' }, { status: 404 })
    }
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ ...existing, ...body, updated_at: new Date().toISOString() })
  }),

  http.delete(`${API}/api/v1/contacts/:id`, ({ params }) => {
    const id = params.id as string
    const existing = getContactById(id)
    if (!existing) {
      return HttpResponse.json({ error: 'Contact not found' }, { status: 404 })
    }
    // Hans Müller (ct-001) stands in for a contact with call-campaign/advisory
    // history in the real service, so the GDPR-anonymization detour
    // (KontaktePage) is reachable and screenshot-testable in demo mode too.
    if (id === IDS.contacts.mueller) {
      return HttpResponse.json(
        {
          error:
            'contact is in use and cannot be deleted: call campaign history, advisory protocols reference this contact; use GDPR anonymization instead of deleting',
        },
        { status: 409 },
      )
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
      companies: paged.map(normalizeCompany),
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
    return HttpResponse.json({ company: normalizeCompany(company) })
  }),

  http.get(`${API}/api/v1/companies/:id/contacts`, ({ params }) => {
    const linked = mockContacts.contacts.filter(
      (c) => (c as unknown as Record<string, unknown>).company_id === params.id,
    )
    return HttpResponse.json({
      contacts: linked.map((c) => contactToCamel(c as unknown as Record<string, unknown>)),
      total: linked.length,
    })
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

  // Update is PUT server-side; mock mirrors it so the FE hook uses one method
  // in both modes (OpenAPI spec mislabels it PATCH, X-3).
  http.put(`${API}/api/v1/companies/:id`, async ({ params, request }) => {
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
    const contactId = url.searchParams.get('contact_id') ?? ''
    const sortBy = url.searchParams.get('sort_by') ?? 'created_at'
    const sortDesc = url.searchParams.get('sort_desc') === 'true'
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
    if (contactId) {
      filtered = filtered.filter((d) => d.contactId === contactId)
    }

    // Sort before paging
    filtered = [...filtered].sort((a, b) => {
      let cmp = 0
      const ra = a as Record<string, unknown>
      const rb = b as Record<string, unknown>
      if (sortBy === 'name') {
        cmp = String(ra.name ?? '').localeCompare(String(rb.name ?? ''), 'de')
      } else if (sortBy === 'value') {
        cmp = (Number(ra.value) || 0) - (Number(rb.value) || 0)
      } else if (sortBy === 'expected_close_date') {
        const aD = ra.expectedCloseDate ? new Date(ra.expectedCloseDate as string).getTime() : 0
        const bD = rb.expectedCloseDate ? new Date(rb.expectedCloseDate as string).getTime() : 0
        cmp = aD - bD
      } else if (sortBy === 'updated_at') {
        const aD = ra.updated_at ? new Date(ra.updated_at as string).getTime() : 0
        const bD = rb.updated_at ? new Date(rb.updated_at as string).getTime() : 0
        cmp = aD - bD
      } else {
        // created_at (default)
        const aD = ra.created_at ? new Date(ra.created_at as string).getTime() : 0
        const bD = rb.created_at ? new Date(rb.created_at as string).getTime() : 0
        cmp = aD - bD
      }
      return sortDesc ? -cmp : cmp
    })

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
    return HttpResponse.json({ deal })
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

  // Update is PUT server-side; mock mirrors it (spec mislabels PATCH, X-3).
  http.put(`${API}/api/v1/deals/:id`, async ({ params, request }) => {
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

  // Move deal to another stage (drag-and-drop). Mutates the in-memory mock
  // array in place so the change persists across refetches in the demo.
  http.post(`${API}/api/v1/deals/:id/stage`, async ({ params, request }) => {
    const deal = getDealById(params.id as string) as
      | (Record<string, unknown> & { stageId?: string; stageName?: string })
      | undefined
    if (!deal) {
      return HttpResponse.json({ error: 'Deal not found' }, { status: 404 })
    }
    const body = (await request.json()) as { stage_id?: string }
    const targetStage = mockPipelineStages.stages.find((s) => s.id === body.stage_id)
    if (body.stage_id) {
      deal.stageId = body.stage_id
      if (targetStage) deal.stageName = targetStage.name
      deal.updated_at = new Date().toISOString()
    }
    return HttpResponse.json({ deal })
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

  http.post(`${API}/api/v1/pipeline-stages`, async ({ request }) => {
    const body = (await request.json()) as {
      name: string; color: string; is_won: boolean; is_lost: boolean; probability: number
    }
    const maxOrder = mockPipelineStages.stages.reduce((m, s) => Math.max(m, s.order), 0)
    const stage = {
      id: `stage-${Date.now()}`,
      name: body.name,
      order: maxOrder + 1,
      probability: body.probability,
      color: body.color,
      is_won: body.is_won,
      is_lost: body.is_lost,
    }
    mockPipelineStages.stages.push(stage)
    return HttpResponse.json({ stage })
  }),

  // Update is PUT server-side; mock mirrors it (spec mislabels PATCH, X-3).
  http.put(`${API}/api/v1/pipeline-stages/:id`, async ({ request, params }) => {
    const stage = mockPipelineStages.stages.find((s) => s.id === params.id)
    if (!stage) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as Partial<typeof stage>
    Object.assign(stage, body)
    return HttpResponse.json({ stage })
  }),

  http.delete(`${API}/api/v1/pipeline-stages/:id`, ({ params }) => {
    const idx = mockPipelineStages.stages.findIndex((s) => s.id === params.id)
    if (idx >= 0) mockPipelineStages.stages.splice(idx, 1)
    return HttpResponse.json({ success: true })
  }),

  http.post(`${API}/api/v1/pipeline-stages/reorder`, async ({ request }) => {
    const { stage_ids } = (await request.json()) as { stage_ids: string[] }
    stage_ids.forEach((id, i) => {
      const stage = mockPipelineStages.stages.find((s) => s.id === id)
      if (stage) stage.order = i + 1
    })
    mockPipelineStages.stages.sort((a, b) => a.order - b.order)
    return HttpResponse.json({ success: true })
  }),

  // ---- Activities -------------------------------------------------------

  http.get(`${API}/api/v1/activities`, ({ request }) => {
    const url = new URL(request.url)
    const search = url.searchParams.get('search') ?? url.searchParams.get('q') ?? ''
    const type = url.searchParams.get('activity_type') ?? url.searchParams.get('type') ?? ''
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

    // Sort by requested field
    const sortBy = url.searchParams.get('sort_by') ?? 'created_at'
    const sortDesc = url.searchParams.get('sort_desc') === 'true'
    filtered = [...filtered].sort((a, b) => {
      const ra = a as Record<string, unknown>
      const rb = b as Record<string, unknown>
      let cmp = 0
      if (sortBy === 'subject') {
        cmp = String(ra.subject ?? '').localeCompare(String(rb.subject ?? ''), 'de')
      } else if (sortBy === 'due_date') {
        const aD = ra.due_date ? new Date(ra.due_date as string).getTime() : 0
        const bD = rb.due_date ? new Date(rb.due_date as string).getTime() : 0
        cmp = aD - bD
      } else {
        // created_at (default)
        cmp = new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
      }
      return sortDesc ? -cmp : cmp
    })

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
      // mock data keys off `type`; the API/form sends `activity_type`
      type: (body.activity_type as string) ?? (body.type as string) ?? 'note',
      completed: false,
      ...body,
      created_at: new Date().toISOString(),
    }
    // Persist so created follow-ups show up in list + Wiedervorlage views.
    mockActivities.activities.unshift(created as unknown as (typeof mockActivities.activities)[number])
    return HttpResponse.json(enrichActivity(created), { status: 201 })
  }),

  http.post(`${API}/api/v1/activities/:id/complete`, ({ params }) => {
    const activity = mockActivities.activities.find((a) => a.id === params.id) as
      | (Record<string, unknown> & { completed?: boolean })
      | undefined
    if (!activity) return HttpResponse.json({ error: 'Activity not found' }, { status: 404 })
    activity.completed = true
    activity.completed_at = new Date().toISOString()
    return HttpResponse.json(enrichActivity(activity))
  }),

  http.patch(`${API}/api/v1/activities/:id`, async ({ params, request }) => {
    const activity = mockActivities.activities.find((a) => a.id === params.id) as
      | Record<string, unknown>
      | undefined
    if (!activity) return HttpResponse.json({ error: 'Activity not found' }, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    Object.assign(activity, body)
    if (body.activity_type) activity.type = body.activity_type
    return HttpResponse.json(enrichActivity(activity))
  }),

  http.delete(`${API}/api/v1/activities/:id`, ({ params }) => {
    const idx = mockActivities.activities.findIndex((a) => a.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'Activity not found' }, { status: 404 })
    mockActivities.activities.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // ---- Tags -------------------------------------------------------------

  // All tag definitions used across demo contacts/companies (deduplicated by id)
  // These are returned for GET /api/v1/tags so the TagPopover can list them.
  http.get(`${API}/api/v1/tags`, () => {
    return HttpResponse.json({ tags: mockContactTags })
  }),

  // POST /api/v1/tags — create a tag definition
  http.post(`${API}/api/v1/tags`, async ({ request }) => {
    const body = (await request.json()) as { name: string; color?: string; entity_type?: string }
    const tag = {
      id: `tag-${Date.now()}`,
      name: body.name,
      color: body.color ?? '#6B7280',
      entityType: body.entity_type ?? 'contact',
    }
    mockContactTags.push(tag)
    return HttpResponse.json({ tag })
  }),

  // PATCH /api/v1/tags/:id — rename / recolour
  http.patch(`${API}/api/v1/tags/:id`, async ({ request, params }) => {
    const tag = mockContactTags.find((t) => t.id === params.id)
    if (!tag) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as { name?: string; color?: string }
    if (body.name !== undefined) tag.name = body.name
    if (body.color !== undefined) tag.color = body.color
    return HttpResponse.json({ tag })
  }),

  // DELETE /api/v1/tags/:id — remove a tag definition
  http.delete(`${API}/api/v1/tags/:id`, ({ params }) => {
    const idx = mockContactTags.findIndex((t) => t.id === params.id)
    if (idx >= 0) mockContactTags.splice(idx, 1)
    return HttpResponse.json({ success: true })
  }),

  // POST /api/v1/contacts/:id/tags — add tags
  http.post(`${API}/api/v1/contacts/:id/tags`, async ({ params }) => {
    const existing = getContactById(params.id as string)
    if (!existing) {
      return HttpResponse.json({ error: 'Contact not found' }, { status: 404 })
    }
    // Return the updated contact info (tags merged — demo doesn't mutate in-memory)
    const updated = contactToCamel(existing as unknown as Record<string, unknown>)
    return HttpResponse.json({ contact: updated })
  }),

  // DELETE /api/v1/contacts/:id/tags — remove tags
  http.delete(`${API}/api/v1/contacts/:id/tags`, async ({ params }) => {
    const existing = getContactById(params.id as string)
    if (!existing) {
      return HttpResponse.json({ error: 'Contact not found' }, { status: 404 })
    }
    // Return the updated contact info
    const updated = contactToCamel(existing as unknown as Record<string, unknown>)
    return HttpResponse.json({ contact: updated })
  }),

  // GET /api/v1/crm/contacts/:id/timeline — timeline events for a contact
  // useContactTimeline calls GET /api/v1/crm/contacts/{id}/timeline
  http.get(`${API}/api/v1/crm/contacts/:id/timeline`, ({ params, request }) => {
    const contactId = params.id as string
    const url = new URL(request.url)
    const offset = Number(url.searchParams.get('offset') ?? 0)
    const limit = Number(url.searchParams.get('limit') ?? 20)

    const contactActivities = mockActivities.activities
      .filter((a) => a.contact_id === contactId)
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())

    const events = contactActivities.slice(offset, offset + limit).map((a) => ({
      id: a.id,
      event_type: 'activity',
      occurred_at: a.created_at,
      title: (a as Record<string, unknown>).subject as string ?? (a as Record<string, unknown>).description as string ?? 'Aktivität',
      description: (a as Record<string, unknown>).description as string | undefined,
      created_by_name: 'TechVision Team',
      metadata: {
        activity_type: a.type,
      },
    }))

    return HttpResponse.json({ events, total: contactActivities.length })
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

  // ── Contact report references (R-5a) ──────────────────────────────────────
  http.get(`${API}/api/v1/contacts/:id/files`, ({ params }) => {
    const list = contactReportLinks.filter((f) => f.contact_id === params.id)
    return HttpResponse.json({ files: list })
  }),

  http.post(`${API}/api/v1/contacts/:id/files`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>
    contactFileCounter += 1
    const file = {
      id: `cfile-${contactFileCounter}`,
      contact_id: params.id as string,
      filename: String(body.filename ?? ''),
      mime_type: String(body.mime_type ?? ''),
      storage_key: String(body.storage_key ?? ''),
      created_at: new Date().toISOString(),
    }
    contactReportLinks.push(file)
    return HttpResponse.json({ file }, { status: 201 })
  }),

  // ---- GDPR consent -----------------------------------------------------
  // The real routes return the raw crmv1 protos: a summary keyed by
  // consent_type, and a { history, total } envelope.

  http.get(`${API}/api/v1/contacts/:id/consents`, ({ params }) => {
    const contactId = params.id as string
    ensureConsentSeed(contactId)
    return HttpResponse.json({
      contact_id: contactId,
      consents: consentsByContact[contactId],
    })
  }),

  http.post(`${API}/api/v1/contacts/:id/consents`, async ({ params, request }) => {
    const contactId = params.id as string
    const body = (await request.json()) as Record<string, unknown>
    const consentType = String(body.consent_type ?? '')
    if (!consentType) {
      return HttpResponse.json({ message: 'consent_type is required' }, { status: 400 })
    }

    ensureConsentSeed(contactId)
    const record = makeConsent(contactId, consentType, {
      granted: true,
      source: String(body.source ?? 'web_form'),
      legal_basis: String(body.legal_basis ?? 'consent'),
      granted_at: new Date().toISOString(),
    })

    consentsByContact[contactId][consentType] = record
    const key = consentHistoryKey(contactId, consentType)
    consentHistory[key] = [record, ...(consentHistory[key] ?? [])]
    return HttpResponse.json(record, { status: 201 })
  }),

  http.delete(`${API}/api/v1/contacts/:id/consents/:type`, ({ params }) => {
    const contactId = params.id as string
    const consentType = params.type as string
    ensureConsentSeed(contactId)

    const previous = consentsByContact[contactId][consentType]
    if (!previous) {
      return HttpResponse.json({ message: 'consent not found' }, { status: 404 })
    }

    const record = makeConsent(contactId, consentType, {
      granted: false,
      source: previous.source,
      legal_basis: previous.legal_basis,
      granted_at: previous.granted_at,
      revoked_at: new Date().toISOString(),
    })

    consentsByContact[contactId][consentType] = record
    const key = consentHistoryKey(contactId, consentType)
    consentHistory[key] = [record, ...(consentHistory[key] ?? [])]
    return HttpResponse.json(record)
  }),

  http.get(`${API}/api/v1/contacts/:id/consents/:type/history`, ({ params }) => {
    const contactId = params.id as string
    ensureConsentSeed(contactId)
    const history = consentHistory[consentHistoryKey(contactId, params.type as string)] ?? []
    return HttpResponse.json({ history, total: history.length })
  }),

  // ---- GDPR right-to-erasure (deletion request) --------------------------

  http.post(`${API}/api/v1/contacts/:id/gdpr/deletion-request`, async ({ params, request }) => {
    const contactId = params.id as string
    const existing = getContactById(contactId)
    if (!existing) {
      return HttpResponse.json({ error: 'Contact not found' }, { status: 404 })
    }
    const body = (await request.json().catch(() => ({}))) as Record<string, unknown>
    deletionRequestCounter += 1
    const deletionRequest: DemoDeletionRequest = {
      id: `del-req-${deletionRequestCounter}`,
      contact_id: contactId,
      requested_by: null,
      reason: String(body.reason ?? ''),
      status: 'pending',
      completed_at: null,
      created_at: new Date().toISOString(),
    }
    deletionRequestsById[deletionRequest.id] = deletionRequest
    return HttpResponse.json(deletionRequest, { status: 201 })
  }),

  http.post(`${API}/api/v1/gdpr/deletion-requests/:id/process`, ({ params }) => {
    const deletionRequest = deletionRequestsById[params.id as string]
    if (!deletionRequest) {
      return HttpResponse.json({ error: 'deletion request not found' }, { status: 404 })
    }
    deletionRequest.status = 'completed'
    deletionRequest.completed_at = new Date().toISOString()
    return HttpResponse.json({ status: 'completed' })
  }),
]
