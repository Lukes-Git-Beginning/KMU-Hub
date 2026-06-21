import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS, CURRENT_USER } from '../data/shared-ids'
import { daysAgo, hoursAgo } from '../data/date-helpers'
import type {
  WikiArticle,
  WikiCategory,
  WikiVersion,
  WikiAttachment,
  WikiShareToken,
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

// Inline demo diagram (data URL) so the rich-block showcase article (wart-003)
// renders a real figure without a backend upload pipeline.
const DEMO_FLOW_SVG =
  "<svg xmlns='http://www.w3.org/2000/svg' width='520' height='150'>" +
  "<rect width='520' height='150' fill='#f8fafc'/>" +
  "<rect x='20' y='52' width='110' height='46' rx='8' fill='#e0e7ff'/>" +
  "<rect x='205' y='52' width='110' height='46' rx='8' fill='#dbeafe'/>" +
  "<rect x='390' y='52' width='110' height='46' rx='8' fill='#dcfce7'/>" +
  "<text x='75' y='80' font-family='sans-serif' font-size='13' text-anchor='middle' fill='#3730a3'>Angebot</text>" +
  "<text x='260' y='80' font-family='sans-serif' font-size='13' text-anchor='middle' fill='#1e40af'>Auftrag</text>" +
  "<text x='445' y='80' font-family='sans-serif' font-size='13' text-anchor='middle' fill='#166534'>Rechnung</text>" +
  "<line x1='130' y1='75' x2='205' y2='75' stroke='#94a3b8' stroke-width='2'/>" +
  "<line x1='315' y1='75' x2='390' y2='75' stroke='#94a3b8' stroke-width='2'/>" +
  '</svg>'
const DEMO_FLOW_DATA_URL = 'data:image/svg+xml,' + encodeURIComponent(DEMO_FLOW_SVG)

const ARTICLES: WikiArticle[] = [
  {
    id: 'wart-001',
    tenant_id: 'tenant-001',
    title: 'Willkommen im Cosmi-Wiki',
    slug: 'willkommen-im-cosmi-wiki',
    content: {
      html:
        '<p>Dies ist die zentrale Wissensbasis des Unternehmens — ein ruhiger Ort für alles, ' +
        'was das Team wissen muss: Prozesse, Anleitungen und Entscheidungen, an einem Ort und ' +
        'immer aktuell.</p>' +
        '<h2>Was du hier findest</h2>' +
        '<p>Das Wiki ist nach Bereichen geordnet. Jeder Artikel hat eine verantwortliche Person ' +
        'und wird regelmässig geprüft, damit du dich auf den Inhalt verlassen kannst.</p>' +
        '<ul><li><strong>Onboarding</strong> — die ersten Tage und Wochen</li>' +
        '<li><strong>Prozesse &amp; Workflows</strong> — wie wir arbeiten</li>' +
        '<li><strong>IT &amp; Infrastruktur</strong> — Tools, Zugänge, Notfälle</li></ul>' +
        '<h2>So schreibst du einen guten Artikel</h2>' +
        '<p>Schreibe für die Person, die zum ersten Mal hier landet. Eine klare Überschrift, ' +
        'kurze Absätze und eine Liste mit konkreten Schritten reichen oft schon aus.</p>' +
        '<h3>Drei einfache Regeln</h3>' +
        '<ol><li>Beginne mit dem Ergebnis, nicht mit dem Kontext.</li>' +
        '<li>Verlinke verwandte Artikel statt sie zu kopieren.</li>' +
        '<li>Halte den Artikel aktuell — markiere ihn als geprüft.</li></ol>',
    } as Record<string, unknown>,
    author_id: CURRENT_USER.id,
    category_id: 'wcat-001',
    published: true,
    tags: ['Onboarding', 'Übersicht'],
    view_count: 342,
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
    tags: ['HR', 'Checkliste'],
    view_count: 158,
    created_at: daysAgo(45) + 'T08:00:00Z',
    updated_at: daysAgo(3) + 'T14:00:00Z',
  },
  {
    id: 'wart-003',
    tenant_id: 'tenant-001',
    title: 'Angebots- und Rechnungsprozess',
    slug: 'angebots-und-rechnungsprozess',
    content: {
      html:
        '<p>Dieser Artikel beschreibt den vollständigen Prozess von der Angebotserstellung ' +
        'bis zur Rechnungsstellung inklusive Mahnwesen und DATEV-Export.</p>' +
        '<figure data-figure="" class="wiki-figure"><img src="' +
        DEMO_FLOW_DATA_URL +
        '" class="wiki-figure-img"><figcaption class="wiki-figure-caption">Die drei Phasen im Überblick</figcaption></figure>' +
        '<h2>Angebot erstellen</h2>' +
        '<p>Jedes Angebot startet aus einer Opportunity im CRM. Pflichtfelder sind Kunde, ' +
        'Positionen und Gültigkeitsdauer.</p>' +
        '<div data-callout="" data-variant="info" class="wiki-callout"><p>Angebote sind 30 Tage ' +
        'gültig, sofern nicht anders vereinbart.</p></div>' +
        '<div data-callout="" data-variant="warning" class="wiki-callout"><p>Rabatte über 15 % ' +
        'müssen vor dem Versand von der Teamleitung freigegeben werden.</p></div>' +
        '<h2>Auftrag &amp; Rechnung</h2>' +
        '<p>Nach Auftragsbestätigung wird die Rechnung automatisch aus dem Angebot erzeugt. ' +
        'Die Rechnungsnummer folgt dem Schema unten.</p>' +
        '<pre class="wiki-code"><code class="language-typescript">function invoiceNumber(year: number, seq: number): string {\n  return `R-${year}-${String(seq).padStart(4, "0")}`\n}</code></pre>' +
        '<div data-callout="" data-variant="tip" class="wiki-callout"><p>Tipp: Mit dem Kürzel ' +
        '<strong>/code</strong> fügst du im Editor jederzeit einen Codeblock ein.</p></div>' +
        '<details data-details="" class="wiki-details" open><summary class="wiki-details-summary">' +
        'Mahnwesen im Detail</summary><div data-details-content="" class="wiki-details-body">' +
        '<p>Die erste Mahnung erfolgt 14 Tage nach Fälligkeit, die zweite nach weiteren 10 Tagen. ' +
        'Danach übernimmt das Inkasso.</p></div></details>' +
        '<hr>' +
        '<div data-callout="" data-variant="recommendation" class="wiki-callout"><p>Empfehlung: ' +
        'Prüfe vor dem DATEV-Export immer die Kostenstellen-Zuordnung.</p></div>',
    } as Record<string, unknown>,
    author_id: IDS.users.markus,
    category_id: 'wcat-002',
    published: true,
    tags: ['Finanzen', 'DATEV', 'Prozess'],
    view_count: 97,
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
    tags: ['IT', 'Notfall'],
    view_count: 64,
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
    tags: ['DSGVO', 'Recht'],
    view_count: 211,
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
    tags: ['CRM', 'Vertrieb'],
    view_count: 129,
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
    tags: ['IT', 'Intern'],
    view_count: 18,
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
      changed_by: CURRENT_USER.id,
      changed_at: daysAgo(58) + 'T09:00:00Z',
      change_note: 'Erstentwurf angelegt',
    },
    {
      id: 'wver-001-2',
      article_id: 'wart-001',
      version_number: 2,
      content: tiptapDoc('Überarbeitet mit Abteilungsstruktur.'),
      changed_by: IDS.users.julia,
      changed_at: daysAgo(20) + 'T11:00:00Z',
      change_note: 'Abteilungsstruktur ergänzt',
    },
    {
      id: 'wver-001-3',
      article_id: 'wart-001',
      version_number: 3,
      content: tiptapDoc(
        'Dies ist die zentrale Wissensbasis des Unternehmens. Hier finden Sie alle wichtigen Prozesse, Anleitungen und Informationen.',
      ),
      changed_by: CURRENT_USER.id,
      changed_at: daysAgo(5) + 'T10:30:00Z',
      change_note: 'Einleitung neu formuliert',
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
      change_note: 'Erstentwurf des Prozesses',
    },
    {
      id: 'wver-003-2',
      article_id: 'wart-003',
      version_number: 2,
      content: tiptapDoc(
        'Beschreibt den vollständigen Prozess von der Angebotserstellung bis zur Rechnungsstellung inkl. Mahnwesen und DATEV-Export.',
      ),
      changed_by: IDS.users.markus,
      changed_at: daysAgo(10) + 'T16:00:00Z',
      change_note: 'Mahnwesen und DATEV-Export ergänzt',
    },
  ],
}

// Inline SVG demo blob so an image attachment shows a real thumbnail/preview
// without a backend upload pipeline.
const DEMO_IMAGE_SVG =
  "<svg xmlns='http://www.w3.org/2000/svg' width='200' height='130'>" +
  "<rect width='200' height='130' fill='#eef2ff'/>" +
  "<rect x='18' y='22' width='70' height='34' rx='6' fill='#6366f1'/>" +
  "<rect x='112' y='22' width='70' height='34' rx='6' fill='#8b5cf6'/>" +
  "<rect x='65' y='80' width='70' height='34' rx='6' fill='#0ea5e9'/>" +
  "<line x1='53' y1='56' x2='100' y2='80' stroke='#475569' stroke-width='2'/>" +
  "<line x1='147' y1='56' x2='100' y2='80' stroke='#475569' stroke-width='2'/>" +
  "<text x='100' y='126' font-family='sans-serif' font-size='11' text-anchor='middle' fill='#334155'>Onboarding-Ablauf</text>" +
  '</svg>'
const DEMO_IMAGE_DATA_URL = 'data:image/svg+xml,' + encodeURIComponent(DEMO_IMAGE_SVG)

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
    {
      id: 'watt-003',
      article_id: 'wart-002',
      file_ref: 'onboarding-ablauf.svg',
      mime: 'image/svg+xml',
      size: 4200,
      uploaded_by: IDS.users.julia,
      created_at: daysAgo(44) + 'T09:05:00Z',
      data_url: DEMO_IMAGE_DATA_URL,
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

// Active share tokens (stateful — created via POST, revoked via DELETE).
const SHARE_TOKENS: WikiShareToken[] = [
  {
    id: 'wshare-seed-1',
    article_id: 'wart-001',
    token: 'demo-share-token-welcome',
    expires_at: null,
    permissions: ['read'],
    created_at: daysAgo(6) + 'T08:00:00Z',
  },
]

// ============================================================================
// Helpers
// ============================================================================

function paginate<T>(items: T[], page: number, pageSize: number): { items: T[]; total: number } {
  const start = (page - 1) * pageSize
  return { items: items.slice(start, start + pageSize), total: items.length }
}

/** Best-effort searchable plain text from the JSONB content field. */
function contentText(content: unknown): string {
  if (typeof content === 'string') return content
  if (!content || typeof content !== 'object') return ''
  const c = content as Record<string, unknown>
  if (typeof c.html === 'string') return c.html.replace(/<[^>]*>/g, ' ')
  if (typeof c.plain === 'string') return c.plain
  // walk a TipTap doc collecting text nodes
  const parts: string[] = []
  const walk = (node: Record<string, unknown>) => {
    if (typeof node.text === 'string') parts.push(node.text)
    for (const child of (node.content as Array<Record<string, unknown>>) ?? []) walk(child)
  }
  walk(c)
  return parts.join(' ')
}

/** Append a new version snapshot for an article (newest version_number wins). */
function appendVersion(
  articleId: string,
  content: Record<string, unknown>,
  changeNote?: string,
): void {
  const list = (VERSIONS[articleId] ??= [])
  const nextNumber = list.reduce((max, v) => Math.max(max, v.version_number), 0) + 1
  list.push({
    id: `wver-${articleId}-${nextNumber}-${Date.now()}`,
    article_id: articleId,
    version_number: nextNumber,
    content,
    changed_by: CURRENT_USER.id,
    changed_at: new Date().toISOString(),
    change_note: changeNote?.trim() || undefined,
  })
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

  http.patch(`${BASE}/categories/:id`, async ({ params, request }) => {
    const category = CATEGORIES.find((c) => c.id === params.id)
    if (!category) return HttpResponse.json({ error: 'category not found' }, { status: 404 })
    const body = (await request.json()) as {
      name?: string
      parent_id?: string | null
      position?: number
    }
    if (body.name !== undefined) category.name = body.name
    if (body.parent_id !== undefined) category.parent_id = body.parent_id
    if (body.position !== undefined) category.position = body.position
    category.updated_at = new Date().toISOString()
    return HttpResponse.json(category)
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
          // walk the JSONB content to plain text so TipTap docs match too
          contentText(a.content).toLowerCase().includes(q),
      )
    }

    const result = paginate(filtered, page, pageSize)
    return HttpResponse.json({ articles: result.items, total: result.total })
  }),

  // --- Article detail ---

  http.get(`${BASE}/articles/:id`, ({ params }) => {
    const article = ARTICLES.find((a) => a.id === params.id)
    if (!article) return HttpResponse.json({ error: 'article not found' }, { status: 404 })
    // Reading an article detail increments its view counter.
    article.view_count = (article.view_count ?? 0) + 1
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
      author_id: CURRENT_USER.id,
      category_id: body.category_id ?? null,
      published: body.published ?? false,
      tags: body.tags ?? [],
      view_count: 0,
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
    const body = (await request.json()) as Partial<WikiArticle> & { change_note?: string }
    if (body.title !== undefined) article.title = body.title
    if (body.content !== undefined) {
      article.content = body.content
      // A content edit creates a new version snapshot (newest = current).
      appendVersion(article.id, body.content, body.change_note)
    }
    if (body.category_id !== undefined) article.category_id = body.category_id
    if (body.published !== undefined) article.published = body.published
    if (body.tags !== undefined) article.tags = body.tags
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
        a.title.toLowerCase().includes(q) ||
        contentText(a.content).toLowerCase().includes(q),
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
    // Restoring produces a new current version equal to the restored content.
    appendVersion(article.id, version.content, `Wiederhergestellt aus v${version.version_number}`)
    return HttpResponse.json(article)
  }),

  // --- Attachments ---

  http.get(`${BASE}/articles/:id/attachments`, ({ params }) => {
    const attachments = ATTACHMENTS[params.id as string] ?? []
    return HttpResponse.json(attachments)
  }),

  http.post(`${BASE}/articles/:id/attachments`, async ({ params, request }) => {
    const body = (await request.json()) as {
      file_ref: string
      mime: string
      size: number
      data_url?: string
    }
    const attachment: WikiAttachment = {
      id: `watt-${Date.now()}`,
      article_id: params.id as string,
      file_ref: body.file_ref,
      mime: body.mime,
      size: body.size,
      uploaded_by: CURRENT_USER.id,
      created_at: new Date().toISOString(),
      data_url: body.data_url,
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

  // --- Share tokens (stateful) ---

  http.get(`${BASE}/articles/:id/share`, ({ params }) => {
    const tokens = SHARE_TOKENS.filter((t) => t.article_id === params.id)
    return HttpResponse.json(tokens)
  }),

  http.post(`${BASE}/articles/:id/share`, async ({ params, request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      expires_at?: string
      permissions?: string[]
    }
    const token = {
      id: `wshare-${Date.now()}`,
      article_id: params.id as string,
      token: `demo-share-token-${Math.random().toString(36).slice(2)}`,
      expires_at: body.expires_at ?? null,
      permissions: body.permissions ?? ['read'],
      created_at: new Date().toISOString(),
    }
    SHARE_TOKENS.push(token)
    return HttpResponse.json(token, { status: 201 })
  }),

  http.delete(`${BASE}/share/:tokenId`, ({ params }) => {
    const idx = SHARE_TOKENS.findIndex((t) => t.id === params.tokenId)
    if (idx === -1) return HttpResponse.json({ error: 'token not found' }, { status: 404 })
    SHARE_TOKENS.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),
]
