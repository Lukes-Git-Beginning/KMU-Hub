import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysAgo, hoursAgo } from '../data/date-helpers'
import type {
  WikiArticle,
  WikiCategory,
  WikiVersion,
  WikiAttachment,
} from '@/api/wiki-types'

const API = API_BASE_URL
const BASE = `${API}/api/v1/wiki`

// ============================================================================
// Wire-shape Demo-Daten
// (snake_case — spiegelt Backend-API-Response, wird von wiki-adapter.ts gemappt)
// ============================================================================

const CATEGORIES: WikiCategory[] = [
  {
    id: 'wcat-001',
    tenant_id: 'tenant-001',
    name: 'Allgemein',
    parent_id: null,
    position: 0,
    created_at: daysAgo(60) + 'T08:00:00Z',
    updated_at: daysAgo(30) + 'T08:00:00Z',
  },
  {
    id: 'wcat-002',
    tenant_id: 'tenant-001',
    name: 'Prozesse & Workflows',
    parent_id: null,
    position: 1,
    created_at: daysAgo(55) + 'T09:00:00Z',
    updated_at: daysAgo(20) + 'T09:00:00Z',
  },
  {
    id: 'wcat-003',
    tenant_id: 'tenant-001',
    name: 'IT & Infrastruktur',
    parent_id: null,
    position: 2,
    created_at: daysAgo(50) + 'T10:00:00Z',
    updated_at: daysAgo(10) + 'T10:00:00Z',
  },
  {
    id: 'wcat-004',
    tenant_id: 'tenant-001',
    name: 'Onboarding',
    parent_id: 'wcat-001',
    position: 0,
    created_at: daysAgo(45) + 'T08:30:00Z',
    updated_at: daysAgo(5) + 'T08:30:00Z',
  },
  {
    id: 'wcat-005',
    tenant_id: 'tenant-001',
    name: 'Datenschutz & DSGVO',
    parent_id: null,
    position: 3,
    created_at: daysAgo(40) + 'T11:00:00Z',
    updated_at: daysAgo(7) + 'T11:00:00Z',
  },
]

// TipTap-JSON-Content als Record<string, unknown> (Wire-Shape erwartet JSONB)
function tiptapDoc(text: string): Record<string, unknown> {
  return {
    type: 'doc',
    content: [
      {
        type: 'paragraph',
        content: [{ type: 'text', text }],
      },
    ],
  }
}

const ARTICLES: WikiArticle[] = [
  {
    id: 'wart-001',
    tenant_id: 'tenant-001',
    title: 'Willkommen im Cosmi-Wiki',
    slug: 'willkommen-im-cosmi-wiki',
    content: tiptapDoc(
      'Dies ist die zentrale Wissensbasis des Unternehmens. Hier finden Sie alle wichtigen Prozesse, Anleitungen und Informationen.',
    ),
    author_id: IDS.users.stefan,
    category_id: 'wcat-001',
    published: true,
    created_at: daysAgo(58) + 'T09:00:00Z',
    updated_at: daysAgo(5) + 'T10:30:00Z',
  },
  {
    id: 'wart-002',
    tenant_id: 'tenant-001',
    title: 'Onboarding neuer Mitarbeitender',
    slug: 'onboarding-neue-mitarbeitende',
    content: tiptapDoc(
      'Schritt-für-Schritt-Anleitung für den Einstieg neuer Teammitglieder: Zugänge einrichten, Systeme kennenlernen, erste Aufgaben.',
    ),
    author_id: IDS.users.julia,
    category_id: 'wcat-004',
    published: true,
    created_at: daysAgo(45) + 'T08:00:00Z',
    updated_at: daysAgo(3) + 'T14:00:00Z',
  },
  {
    id: 'wart-003',
    tenant_id: 'tenant-001',
    title: 'Angebots- und Rechnungsprozess',
    slug: 'angebots-und-rechnungsprozess',
    content: tiptapDoc(
      'Beschreibt den vollständigen Prozess von der Angebotserstellung bis zur Rechnungsstellung inkl. Mahnwesen und DATEV-Export.',
    ),
    author_id: IDS.users.markus,
    category_id: 'wcat-002',
    published: true,
    created_at: daysAgo(40) + 'T10:00:00Z',
    updated_at: daysAgo(10) + 'T16:00:00Z',
  },
  {
    id: 'wart-004',
    tenant_id: 'tenant-001',
    title: 'Backup & Disaster Recovery',
    slug: 'backup-disaster-recovery',
    content: tiptapDoc(
      'Backup-Strategie, Wiederherstellungszeiten (RTO/RPO) und Schritt-für-Schritt-Anleitung im Notfall.',
    ),
    author_id: IDS.users.thomas,
    category_id: 'wcat-003',
    published: true,
    created_at: daysAgo(30) + 'T11:00:00Z',
    updated_at: daysAgo(8) + 'T09:15:00Z',
  },
  {
    id: 'wart-005',
    tenant_id: 'tenant-001',
    title: 'DSGVO: Einwilligungen & Auskunftsrechte',
    slug: 'dsgvo-einwilligungen-auskunftsrechte',
    content: tiptapDoc(
      'Übersicht über die gesetzlichen Pflichten gemäß DSGVO: Einwilligungsverwaltung, Auskunftspflicht, Löschanfragen.',
    ),
    author_id: IDS.users.sarah,
    category_id: 'wcat-005',
    published: true,
    created_at: daysAgo(25) + 'T13:00:00Z',
    updated_at: daysAgo(2) + 'T11:00:00Z',
  },
  {
    id: 'wart-006',
    tenant_id: 'tenant-001',
    title: 'CRM: Kontakte und Deals verwalten',
    slug: 'crm-kontakte-deals',
    content: tiptapDoc(
      'Anleitung zur Pflege des CRM-Moduls: Kontakte anlegen, Deals verwalten, Pipeline-Phasen und Aktivitäten.',
    ),
    author_id: IDS.users.laura,
    category_id: 'wcat-002',
    published: true,
    created_at: daysAgo(20) + 'T08:30:00Z',
    updated_at: hoursAgo(36),
  },
  {
    id: 'wart-007',
    tenant_id: 'tenant-001',
    title: 'Infrastruktur-Übersicht (intern)',
    slug: 'infrastruktur-uebersicht-intern',
    content: tiptapDoc(
      'Interne Übersicht der Server, Dienste und Netzwerkkonfiguration — nur für IT-Administratoren.',
    ),
    author_id: IDS.users.thomas,
    category_id: 'wcat-003',
    published: false,
    created_at: daysAgo(15) + 'T16:00:00Z',
    updated_at: daysAgo(1) + 'T17:45:00Z',
  },
]

const VERSIONS: Record<string, WikiVersion[]> = {
  'wart-001': [
    {
      id: 'wver-001-1',
      article_id: 'wart-001',
      version_number: 1,
      content: tiptapDoc('Erste Version des Willkommensartikels.'),
      changed_by: IDS.users.stefan,
      changed_at: daysAgo(58) + 'T09:00:00Z',
    },
    {
      id: 'wver-001-2',
      article_id: 'wart-001',
      version_number: 2,
      content: tiptapDoc('Überarbeitet mit Abteilungsstruktur.'),
      changed_by: IDS.users.julia,
      changed_at: daysAgo(20) + 'T11:00:00Z',
    },
  ],
  'wart-003': [
    {
      id: 'wver-003-1',
      article_id: 'wart-003',
      version_number: 1,
      content: tiptapDoc('Ursprünglicher Prozessentwurf.'),
      changed_by: IDS.users.markus,
      changed_at: daysAgo(40) + 'T10:00:00Z',
    },
  ],
}

const ATTACHMENTS: Record<string, WikiAttachment[]> = {
  'wart-002': [
    {
      id: 'watt-001',
      article_id: 'wart-002',
      file_ref: 'onboarding-checkliste-2026.pdf',
      mime: 'application/pdf',
      size: 142300,
      uploaded_by: IDS.users.julia,
      created_at: daysAgo(44) + 'T09:00:00Z',
    },
  ],
  'wart-004': [
    {
      id: 'watt-002',
      article_id: 'wart-004',
      file_ref: 'backup-runbook-v3.pdf',
      mime: 'application/pdf',
      size: 89100,
      uploaded_by: IDS.users.thomas,
      created_at: daysAgo(28) + 'T12:00:00Z',
    },
  ],
}

// ============================================================================
// Helpers
// ============================================================================

function paginate<T>(items: T[], page: number, pageSize: number): { items: T[]; total: number } {
  const start = (page - 1) * pageSize
  return { items: items.slice(start, start + pageSize), total: items.length }
}

// ============================================================================
// Handlers
// ============================================================================

export const wikiHandlers = [
  // --- Categories ---

  http.get(`${BASE}/categories`, () => {
    return HttpResponse.json(CATEGORIES)
  }),

  http.post(`${BASE}/categories`, async ({ request }) => {
    const body = (await request.json()) as { name: string; parent_id?: string; position?: number }
    const newCat: WikiCategory = {
      id: `wcat-${Date.now()}`,
      tenant_id: 'tenant-001',
      name: body.name,
      parent_id: body.parent_id ?? null,
      position: body.position ?? CATEGORIES.length,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    CATEGORIES.push(newCat)
    return HttpResponse.json(newCat, { status: 201 })
  }),

  http.delete(`${BASE}/categories/:id`, ({ params }) => {
    const idx = CATEGORIES.findIndex((c) => c.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'category not found' }, { status: 404 })
    CATEGORIES.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // --- Articles list ---

  http.get(`${BASE}/articles`, ({ request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 20)
    const categoryId = url.searchParams.get('category_id')
    const authorId = url.searchParams.get('author_id')
    const publishedParam = url.searchParams.get('published')
    const searchQuery = url.searchParams.get('search')

    let filtered = [...ARTICLES]

    if (categoryId) {
      filtered = filtered.filter((a) => a.category_id === categoryId)
    }
    if (authorId) {
      filtered = filtered.filter((a) => a.author_id === authorId)
    }
    if (publishedParam !== null) {
      const wantPublished = publishedParam === 'true'
      filtered = filtered.filter((a) => a.published === wantPublished)
    }
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      filtered = filtered.filter(
        (a) =>
          a.title.toLowerCase().includes(q) ||
          (typeof a.content === 'string' && (a.content as string).toLowerCase().includes(q)),
      )
    }

    const result = paginate(filtered, page, pageSize)
    return HttpResponse.json({ articles: result.items, total: result.total })
  }),

  // --- Article detail ---

  http.get(`${BASE}/articles/:id`, ({ params }) => {
    const article = ARTICLES.find((a) => a.id === params.id)
    if (!article) return HttpResponse.json({ error: 'article not found' }, { status: 404 })
    return HttpResponse.json(article)
  }),

  // --- Create article ---

  http.post(`${BASE}/articles`, async ({ request }) => {
    const body = (await request.json()) as Partial<WikiArticle>
    const newArticle: WikiArticle = {
      id: `wart-${Date.now()}`,
      tenant_id: 'tenant-001',
      title: body.title ?? 'Neuer Artikel',
      slug: (body.title ?? 'neuer-artikel').toLowerCase().replace(/\s+/g, '-').replace(/[^\w-]/g, ''),
      content: body.content ?? tiptapDoc(''),
      author_id: IDS.users.stefan,
      category_id: body.category_id ?? null,
      published: body.published ?? false,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    ARTICLES.push(newArticle)
    return HttpResponse.json(newArticle, { status: 201 })
  }),

  // --- Update article ---

  http.put(`${BASE}/articles/:id`, async ({ params, request }) => {
    const article = ARTICLES.find((a) => a.id === params.id)
    if (!article) return HttpResponse.json({ error: 'article not found' }, { status: 404 })
    const body = (await request.json()) as Partial<WikiArticle>
    if (body.title !== undefined) article.title = body.title
    if (body.content !== undefined) article.content = body.content
    if (body.category_id !== undefined) article.category_id = body.category_id
    if (body.published !== undefined) article.published = body.published
    article.updated_at = new Date().toISOString()
    return HttpResponse.json(article)
  }),

  // --- Delete article ---

  http.delete(`${BASE}/articles/:id`, ({ params }) => {
    const idx = ARTICLES.findIndex((a) => a.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'article not found' }, { status: 404 })
    ARTICLES.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // --- Search ---

  http.get(`${BASE}/search`, ({ request }) => {
    const url = new URL(request.url)
    const q = (url.searchParams.get('q') ?? '').toLowerCase()
    const limit = Number(url.searchParams.get('limit') ?? 20)

    const results = ARTICLES.filter(
      (a) =>
        a.published &&
        (a.title.toLowerCase().includes(q) ||
          (typeof a.content === 'string' && (a.content as string).toLowerCase().includes(q))),
    ).slice(0, limit)

    return HttpResponse.json({ articles: results })
  }),

  // --- Versions ---

  http.get(`${BASE}/articles/:id/versions`, ({ params }) => {
    const versions = VERSIONS[params.id as string] ?? []
    return HttpResponse.json(versions)
  }),

  http.get(`${BASE}/versions/:versionId`, ({ params }) => {
    for (const versions of Object.values(VERSIONS)) {
      const v = versions.find((ver) => ver.id === params.versionId)
      if (v) return HttpResponse.json(v)
    }
    return HttpResponse.json({ error: 'version not found' }, { status: 404 })
  }),

  http.post(`${BASE}/articles/:id/versions/:versionId/restore`, ({ params }) => {
    const article = ARTICLES.find((a) => a.id === params.id)
    const versions = VERSIONS[params.id as string] ?? []
    const version = versions.find((v) => v.id === params.versionId)
    if (!article || !version) {
      return HttpResponse.json({ error: 'not found' }, { status: 404 })
    }
    article.content = version.content
    article.updated_at = new Date().toISOString()
    return HttpResponse.json(article)
  }),

  // --- Attachments ---

  http.get(`${BASE}/articles/:id/attachments`, ({ params }) => {
    const attachments = ATTACHMENTS[params.id as string] ?? []
    return HttpResponse.json(attachments)
  }),

  http.post(`${BASE}/articles/:id/attachments`, async ({ params, request }) => {
    const body = (await request.json()) as { file_ref: string; mime: string; size: number }
    const attachment: WikiAttachment = {
      id: `watt-${Date.now()}`,
      article_id: params.id as string,
      file_ref: body.file_ref,
      mime: body.mime,
      size: body.size,
      uploaded_by: IDS.users.stefan,
      created_at: new Date().toISOString(),
    }
    if (!ATTACHMENTS[params.id as string]) {
      ATTACHMENTS[params.id as string] = []
    }
    ATTACHMENTS[params.id as string].push(attachment)
    return HttpResponse.json(attachment, { status: 201 })
  }),

  http.delete(`${BASE}/attachments/:attachmentId`, ({ params }) => {
    for (const [articleId, attachments] of Object.entries(ATTACHMENTS)) {
      const idx = attachments.findIndex((a) => a.id === params.attachmentId)
      if (idx !== -1) {
        ATTACHMENTS[articleId].splice(idx, 1)
        return new HttpResponse(null, { status: 204 })
      }
    }
    return HttpResponse.json({ error: 'attachment not found' }, { status: 404 })
  }),

  // --- Share tokens ---

  http.post(`${BASE}/articles/:id/share`, async ({ params, request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      expires_at?: string
      permissions?: string[]
    }
    return HttpResponse.json(
      {
        id: `wshare-${Date.now()}`,
        article_id: params.id,
        token: `demo-share-token-${Math.random().toString(36).slice(2)}`,
        expires_at: body.expires_at ?? null,
        permissions: body.permissions ?? ['read'],
        created_at: new Date().toISOString(),
      },
      { status: 201 },
    )
  }),

  http.delete(`${BASE}/share/:tokenId`, () => {
    return new HttpResponse(null, { status: 204 })
  }),
]
