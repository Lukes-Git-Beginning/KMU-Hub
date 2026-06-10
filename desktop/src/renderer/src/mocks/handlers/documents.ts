import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysAgo, hoursAgo } from '../data/date-helpers'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Folder tree
// ---------------------------------------------------------------------------

const folders = [
  { id: IDS.folders.root, name: 'Meine Dateien', parent_id: null, space_type: 'personal', file_count: 0, created_at: daysAgo(90) },
  { id: IDS.folders.projekte, name: 'Projekte', parent_id: IDS.folders.root, space_type: 'personal', file_count: 8, created_at: daysAgo(60) },
  { id: IDS.folders.verträge, name: 'Verträge', parent_id: IDS.folders.root, space_type: 'personal', file_count: 5, created_at: daysAgo(45) },
  { id: IDS.folders.rechnungen, name: 'Rechnungen', parent_id: IDS.folders.root, space_type: 'personal', file_count: 12, created_at: daysAgo(30) },
  { id: IDS.folders.marketing, name: 'Marketing', parent_id: IDS.folders.root, space_type: 'personal', file_count: 6, created_at: daysAgo(20) },
  { id: IDS.folders.vorlagen, name: 'Vorlagen', parent_id: IDS.folders.root, space_type: 'personal', file_count: 4, created_at: daysAgo(80) },
  { id: IDS.folders.personal, name: 'Personal', parent_id: IDS.folders.root, space_type: 'personal', file_count: 3, created_at: daysAgo(50) },
  // Team space — own folders so the sidebar sections differ per space
  { id: 'fld-team-root', name: 'Team-Dateien', parent_id: null, space_type: 'team', file_count: 1, created_at: daysAgo(70) },
  { id: 'fld-team-vertrieb', name: 'Vertrieb', parent_id: 'fld-team-root', space_type: 'team', file_count: 1, created_at: daysAgo(40) },
  { id: 'fld-team-intern', name: 'Intern', parent_id: 'fld-team-root', space_type: 'team', file_count: 1, created_at: daysAgo(35) },
  // Project space
  { id: 'fld-project-techvision', name: 'Projekt TechVision', parent_id: null, space_type: 'project', file_count: 1, created_at: daysAgo(25) },
]

// ---------------------------------------------------------------------------
// Tags
// ---------------------------------------------------------------------------

const tags = [
  { id: 't1', name: 'Wichtig', color: '#ef4444' },
  { id: 't2', name: 'Vertrag', color: '#3b82f6' },
  { id: 't3', name: 'Design', color: '#ec4899' },
  { id: 't4', name: 'Vorlage', color: '#f59e0b' },
  { id: 't5', name: 'DSGVO', color: '#10b981' },
]

// ---------------------------------------------------------------------------
// Files (25)
// ---------------------------------------------------------------------------

interface MockFile {
  id: string
  filename: string
  name: string
  mime_type: string
  file_size: number
  folder_id: string
  created_by: string
  created_by_name: string
  created_at: string
  updated_at: string
  tags: { id: string; name: string; color: string }[]
  is_favorite?: boolean
}

const files: MockFile[] = [
  { id: 'file-001', filename: 'Projektplan_KMU_Hub_v2.pdf', name: 'Projektplan_KMU_Hub_v2.pdf', mime_type: 'application/pdf', file_size: 2450000, folder_id: IDS.folders.projekte, created_by: IDS.users.stefan, created_by_name: 'Stefan Vogel', created_at: daysAgo(5), updated_at: hoursAgo(2), tags: [tags[0]], is_favorite: true },
  { id: 'file-002', filename: 'Sprint_Backlog_Q1.xlsx', name: 'Sprint_Backlog_Q1.xlsx', mime_type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', file_size: 185000, folder_id: IDS.folders.projekte, created_by: IDS.users.elena, created_by_name: 'Sarah Beck', created_at: daysAgo(12), updated_at: daysAgo(1), tags: [] },
  { id: 'file-003', filename: 'API_Dokumentation.docx', name: 'API_Dokumentation.docx', mime_type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', file_size: 890000, folder_id: IDS.folders.projekte, created_by: IDS.users.markus, created_by_name: 'Markus Weber', created_at: daysAgo(8), updated_at: daysAgo(3), tags: [] },
  { id: 'file-004', filename: 'Architektur_Diagramm.png', name: 'Architektur_Diagramm.png', mime_type: 'image/png', file_size: 3200000, folder_id: IDS.folders.projekte, created_by: IDS.users.lena, created_by_name: 'Lena Braun', created_at: daysAgo(15), updated_at: daysAgo(15), tags: [] },
  { id: 'file-005', filename: 'Vertrag_Gruber_Maschinenbau.pdf', name: 'Vertrag_Gruber_Maschinenbau.pdf', mime_type: 'application/pdf', file_size: 540000, folder_id: IDS.folders.verträge, created_by: IDS.users.thomas, created_by_name: 'Thomas Meier', created_at: daysAgo(30), updated_at: daysAgo(30), tags: [tags[1]] },
  { id: 'file-006', filename: 'SLA_Helvetia_Software.pdf', name: 'SLA_Helvetia_Software.pdf', mime_type: 'application/pdf', file_size: 320000, folder_id: IDS.folders.verträge, created_by: IDS.users.thomas, created_by_name: 'Thomas Meier', created_at: daysAgo(25), updated_at: daysAgo(25), tags: [tags[1]] },
  { id: 'file-007', filename: 'NDA_Rhein_Consulting.pdf', name: 'NDA_Rhein_Consulting.pdf', mime_type: 'application/pdf', file_size: 180000, folder_id: IDS.folders.verträge, created_by: IDS.users.julia, created_by_name: 'Julia Hofmann', created_at: daysAgo(20), updated_at: daysAgo(20), tags: [] },
  { id: 'file-008', filename: 'RE-2026-078.pdf', name: 'RE-2026-078.pdf', mime_type: 'application/pdf', file_size: 95000, folder_id: IDS.folders.rechnungen, created_by: IDS.users.michael, created_by_name: 'Petra Zimmermann', created_at: daysAgo(7), updated_at: daysAgo(7), tags: [] },
  { id: 'file-009', filename: 'RE-2026-079.pdf', name: 'RE-2026-079.pdf', mime_type: 'application/pdf', file_size: 88000, folder_id: IDS.folders.rechnungen, created_by: IDS.users.michael, created_by_name: 'Petra Zimmermann', created_at: daysAgo(5), updated_at: daysAgo(5), tags: [] },
  { id: 'file-010', filename: 'Brandbook_2026.pdf', name: 'Brandbook_2026.pdf', mime_type: 'application/pdf', file_size: 15600000, folder_id: IDS.folders.marketing, created_by: IDS.users.lena, created_by_name: 'Lena Braun', created_at: daysAgo(40), updated_at: daysAgo(10), tags: [tags[2]], is_favorite: true },
  { id: 'file-011', filename: 'Social_Media_Kalender.xlsx', name: 'Social_Media_Kalender.xlsx', mime_type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', file_size: 210000, folder_id: IDS.folders.marketing, created_by: IDS.users.anna, created_by_name: 'Julia Hofmann', created_at: daysAgo(14), updated_at: daysAgo(2), tags: [] },
  { id: 'file-012', filename: 'Angebot_Vorlage.docx', name: 'Angebot_Vorlage.docx', mime_type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', file_size: 125000, folder_id: IDS.folders.vorlagen, created_by: IDS.users.sophie, created_by_name: 'Sophie Lang', created_at: daysAgo(60), updated_at: daysAgo(15), tags: [tags[3]] },
  { id: 'file-013', filename: 'Rechnung_Vorlage.docx', name: 'Rechnung_Vorlage.docx', mime_type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', file_size: 118000, folder_id: IDS.folders.vorlagen, created_by: IDS.users.sophie, created_by_name: 'Sophie Lang', created_at: daysAgo(60), updated_at: daysAgo(15), tags: [tags[3]] },
  { id: 'file-014', filename: 'Arbeitsvertrag_Muster.pdf', name: 'Arbeitsvertrag_Muster.pdf', mime_type: 'application/pdf', file_size: 290000, folder_id: IDS.folders.personal, created_by: IDS.users.nina, created_by_name: 'Elena Schuster', created_at: daysAgo(90), updated_at: daysAgo(30), tags: [] },
  { id: 'file-015', filename: 'Testbericht_v2.pdf', name: 'Testbericht_v2.pdf', mime_type: 'application/pdf', file_size: 780000, folder_id: IDS.folders.projekte, created_by: IDS.users.felix, created_by_name: 'Felix Krause', created_at: daysAgo(3), updated_at: daysAgo(1), tags: [] },
  { id: 'file-016', filename: 'Meeting_Notes_Sprint42.md', name: 'Meeting_Notes_Sprint42.md', mime_type: 'text/markdown', file_size: 12000, folder_id: IDS.folders.projekte, created_by: IDS.users.elena, created_by_name: 'Sarah Beck', created_at: daysAgo(2), updated_at: daysAgo(2), tags: [] },
  { id: 'file-017', filename: 'Logo_TechVision_RGB.svg', name: 'Logo_TechVision_RGB.svg', mime_type: 'image/svg+xml', file_size: 45000, folder_id: IDS.folders.marketing, created_by: IDS.users.lena, created_by_name: 'Lena Braun', created_at: daysAgo(120), updated_at: daysAgo(120), tags: [tags[2]] },
  { id: 'file-018', filename: 'Datenschutzerklaerung.pdf', name: 'Datenschutzerklaerung.pdf', mime_type: 'application/pdf', file_size: 340000, folder_id: IDS.folders.verträge, created_by: IDS.users.nina, created_by_name: 'Elena Schuster', created_at: daysAgo(50), updated_at: daysAgo(10), tags: [tags[4]] },
  { id: 'file-019', filename: 'Onboarding_Checkliste.pdf', name: 'Onboarding_Checkliste.pdf', mime_type: 'application/pdf', file_size: 156000, folder_id: IDS.folders.personal, created_by: IDS.users.nina, created_by_name: 'Elena Schuster', created_at: daysAgo(45), updated_at: daysAgo(5), tags: [] },
  { id: 'file-020', filename: 'Budget_2026.xlsx', name: 'Budget_2026.xlsx', mime_type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', file_size: 420000, folder_id: IDS.folders.rechnungen, created_by: IDS.users.michael, created_by_name: 'Petra Zimmermann', created_at: daysAgo(60), updated_at: daysAgo(8), tags: [tags[0]], is_favorite: true },
  { id: 'file-021', filename: 'Wartungsvertrag_Bavaria_Elektro.pdf', name: 'Wartungsvertrag_Bavaria_Elektro.pdf', mime_type: 'application/pdf', file_size: 410000, folder_id: IDS.folders.verträge, created_by: IDS.users.thomas, created_by_name: 'Thomas Meier', created_at: daysAgo(18), updated_at: daysAgo(18), tags: [tags[1]] },
  { id: 'file-022', filename: 'RE-2026-080.pdf', name: 'RE-2026-080.pdf', mime_type: 'application/pdf', file_size: 92000, folder_id: IDS.folders.rechnungen, created_by: IDS.users.michael, created_by_name: 'Petra Zimmermann', created_at: daysAgo(3), updated_at: daysAgo(3), tags: [] },
  { id: 'file-023', filename: 'Flyer_Messe_2026.pdf', name: 'Flyer_Messe_2026.pdf', mime_type: 'application/pdf', file_size: 8900000, folder_id: IDS.folders.marketing, created_by: IDS.users.lena, created_by_name: 'Lena Braun', created_at: daysAgo(10), updated_at: daysAgo(5), tags: [tags[2]] },
  { id: 'file-024', filename: 'Vertrag_Vorlage.docx', name: 'Vertrag_Vorlage.docx', mime_type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', file_size: 134000, folder_id: IDS.folders.vorlagen, created_by: IDS.users.sophie, created_by_name: 'Sophie Lang', created_at: daysAgo(55), updated_at: daysAgo(12), tags: [tags[3]] },
  { id: 'file-025', filename: 'Gehaltsabrechnung_Muster.pdf', name: 'Gehaltsabrechnung_Muster.pdf', mime_type: 'application/pdf', file_size: 210000, folder_id: IDS.folders.personal, created_by: IDS.users.nina, created_by_name: 'Elena Schuster', created_at: daysAgo(40), updated_at: daysAgo(20), tags: [] },
  // Team/project space files
  { id: 'file-026', filename: 'Team_Meeting_Protokoll_Juni.docx', name: 'Team_Meeting_Protokoll_Juni.docx', mime_type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', file_size: 96000, folder_id: 'fld-team-root', created_by: IDS.users.markus, created_by_name: 'Markus Weber', created_at: daysAgo(4), updated_at: daysAgo(4), tags: [] },
  { id: 'file-027', filename: 'Pitch_Deck_Vertrieb.pdf', name: 'Pitch_Deck_Vertrieb.pdf', mime_type: 'application/pdf', file_size: 4200000, folder_id: 'fld-team-vertrieb', created_by: IDS.users.thomas, created_by_name: 'Thomas Meier', created_at: daysAgo(9), updated_at: daysAgo(6), tags: [] },
  { id: 'file-028', filename: 'Onboarding_Leitfaden_Intern.pdf', name: 'Onboarding_Leitfaden_Intern.pdf', mime_type: 'application/pdf', file_size: 380000, folder_id: 'fld-team-intern', created_by: IDS.users.nina, created_by_name: 'Elena Schuster', created_at: daysAgo(22), updated_at: daysAgo(22), tags: [] },
  { id: 'file-029', filename: 'Prozessanalyse_TechVision_Onsite.pdf', name: 'Prozessanalyse_TechVision_Onsite.pdf', mime_type: 'application/pdf', file_size: 1850000, folder_id: 'fld-project-techvision', created_by: IDS.users.stefan, created_by_name: 'Stefan Vogel', created_at: daysAgo(12), updated_at: daysAgo(7), tags: [tags[0]] },
]

// ---------------------------------------------------------------------------
// Versions & shares (in-memory, mutated by the handlers below)
// ---------------------------------------------------------------------------

interface MockVersion {
  id: string
  file_id: string
  version_number: number
  version_label: string | null
  storage_key: string
  file_size: number
  created_by: string
  created_by_name: string
  created_at: string
}

const versionsByFile = new Map<string, MockVersion[]>()

function getVersions(file: MockFile): MockVersion[] {
  let versions = versionsByFile.get(file.id)
  if (!versions) {
    versions = [
      {
        id: `${file.id}-v1`,
        file_id: file.id,
        version_number: 1,
        version_label: 'Initiale Version',
        storage_key: `demo/${file.id}/v1`,
        file_size: file.file_size,
        created_by: file.created_by,
        created_by_name: file.created_by_name,
        created_at: file.created_at,
      },
    ]
    versionsByFile.set(file.id, versions)
  }
  return versions
}

const DEMO_USER_NAMES: Record<string, string> = {
  [IDS.users.stefan]: 'Stefan Vogel',
  [IDS.users.elena]: 'Sarah Beck',
  [IDS.users.markus]: 'Markus Weber',
  [IDS.users.lena]: 'Lena Braun',
  [IDS.users.thomas]: 'Thomas Meier',
  [IDS.users.julia]: 'Julia Hofmann',
  [IDS.users.michael]: 'Petra Zimmermann',
  [IDS.users.sophie]: 'Sophie Lang',
  [IDS.users.nina]: 'Elena Schuster',
  [IDS.users.felix]: 'Felix Krause',
}

/** Build a tiny but VALID single-page PDF so the viewer's iframe renders a
 *  real page in demo mode (Chromium PDF viewer needs correct xref offsets). */
function buildDemoPdf(title: string): string {
  const esc = (s: string) =>
    s.replace(/[\\()]/g, (c) => `\\${c}`).replace(/[^\x20-\xFF]/g, '?')
  const stream =
    `BT /F1 20 Tf 64 770 Td (${esc(title)}) Tj ET ` +
    `BT /F1 11 Tf 64 742 Td (Cosmi Demo-Modus - Platzhalter-Vorschau) Tj ET`
  const objects = [
    '<</Type/Catalog/Pages 2 0 R>>',
    '<</Type/Pages/Kids[3 0 R]/Count 1>>',
    '<</Type/Page/Parent 2 0 R/MediaBox[0 0 595 842]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>',
    `<</Length ${stream.length}>>\nstream\n${stream}\nendstream`,
    '<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>',
  ]
  let pdf = '%PDF-1.4\n'
  const offsets: number[] = []
  objects.forEach((obj, i) => {
    offsets.push(pdf.length)
    pdf += `${i + 1} 0 obj\n${obj}\nendobj\n`
  })
  const xrefPos = pdf.length
  pdf += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`
  for (const off of offsets) pdf += `${String(off).padStart(10, '0')} 00000 n \n`
  pdf += `trailer\n<</Size ${objects.length + 1}/Root 1 0 R>>\nstartxref\n${xrefPos}\n%%EOF`
  return pdf
}

/** SVG placeholder so image files render an actual picture in demo mode. */
function buildDemoImageSvg(filename: string): string {
  const safe = filename.replace(/[<>&"']/g, '')
  return (
    `<svg xmlns="http://www.w3.org/2000/svg" width="800" height="500">` +
    `<rect width="800" height="500" fill="#eef2f1"/>` +
    `<circle cx="190" cy="170" r="46" fill="#d3e3de"/>` +
    `<path d="M0 420 L240 240 L420 360 L560 280 L800 430 L800 500 L0 500 Z" fill="#c4d8d2"/>` +
    `<text x="40" y="62" font-family="sans-serif" font-size="26" fill="#3b5f55">${safe}</text>` +
    `<text x="40" y="92" font-family="sans-serif" font-size="15" fill="#7a948c">Cosmi Demo-Vorschau</text>` +
    `</svg>`
  )
}

/** Demo download/preview URLs as blob: URLs — Chromium blocks data: URLs in
 *  iframes, blob: works (the demo interceptor runs in the same renderer).
 *  Cached per file+name so repeated previews don't leak object URLs. */
const demoFileUrlCache = new Map<string, string>()

function getDemoFileUrl(file: MockFile): string {
  const cacheKey = `${file.id}:${file.filename}`
  const cached = demoFileUrlCache.get(cacheKey)
  if (cached) return cached
  let blob: Blob
  if (file.mime_type === 'application/pdf') {
    const pdf = buildDemoPdf(file.filename)
    const bytes = Uint8Array.from(pdf, (c) => c.charCodeAt(0) & 0xff)
    blob = new Blob([bytes], { type: 'application/pdf' })
  } else if (file.mime_type.startsWith('image/')) {
    blob = new Blob([buildDemoImageSvg(file.filename)], { type: 'image/svg+xml' })
  } else {
    blob = new Blob(
      [`Cosmi Demo-Modus — Platzhalter-Inhalt für "${file.filename}".`],
      { type: 'text/plain;charset=utf-8' },
    )
  }
  const url = URL.createObjectURL(blob)
  demoFileUrlCache.set(cacheKey, url)
  return url
}

interface MockActivity {
  id: string
  file_id: string
  action: string
  actor_id: string
  actor_name: string
  detail: string | null
  created_at: string
}

const activitiesByFile = new Map<string, MockActivity[]>()
let activitySeq = 0

function recordActivity(
  fileId: string,
  action: string,
  detail: string | null = null,
  actor: { id: string; name: string } = { id: IDS.users.markus, name: 'Markus Weber' },
  createdAt: string = new Date().toISOString(),
) {
  const list = activitiesByFile.get(fileId) ?? []
  list.push({
    id: `act-${fileId}-${activitySeq++}`,
    file_id: fileId,
    action,
    actor_id: actor.id,
    actor_name: actor.name,
    detail,
    created_at: createdAt,
  })
  activitiesByFile.set(fileId, list)
}

function getActivities(file: MockFile): MockActivity[] {
  if (!activitiesByFile.has(file.id)) {
    recordActivity(
      file.id,
      'uploaded',
      null,
      { id: file.created_by, name: file.created_by_name },
      file.created_at,
    )
    if (file.updated_at !== file.created_at) {
      recordActivity(
        file.id,
        'version_created',
        null,
        { id: file.created_by, name: file.created_by_name },
        file.updated_at,
      )
    }
  }
  return activitiesByFile.get(file.id) ?? []
}

interface MockShare {
  id: string
  entity_type: string
  entity_id: string
  shared_with_user_id: string
  shared_with_user_name: string
  permission: string
  shared_by: string
  shared_by_name: string
  created_at: string
}

const shares: MockShare[] = [
  {
    id: 'share-001',
    entity_type: 'file',
    entity_id: 'file-005',
    shared_with_user_id: IDS.users.julia,
    shared_with_user_name: 'Julia Hofmann',
    permission: 'read',
    shared_by: IDS.users.thomas,
    shared_by_name: 'Thomas Meier',
    created_at: daysAgo(10),
  },
]

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const documentHandlers = [
  // List folders (filter by parent_id and/or space_type)
  http.get(`${API}/api/v1/documents/folders`, ({ request }) => {
    const url = new URL(request.url)
    const parentId = url.searchParams.get('parent_id')
    const spaceType = url.searchParams.get('space_type')
    let filtered = folders
    if (spaceType) filtered = filtered.filter((f) => f.space_type === spaceType)
    if (parentId) filtered = filtered.filter((f) => f.parent_id === parentId)
    return HttpResponse.json({ folders: filtered, total: filtered.length })
  }),

  // Folder detail
  http.get(`${API}/api/v1/documents/folders/:id`, ({ params }) => {
    const folder = folders.find((f) => f.id === params.id)
    if (!folder) {
      return HttpResponse.json({ error: 'Folder not found' }, { status: 404 })
    }
    return HttpResponse.json({ folder })
  }),

  // Create folder
  http.post(`${API}/api/v1/documents/folders`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newFolder = {
      id: `fld-${Date.now()}`,
      name: body.name || 'Neuer Ordner',
      parent_id: body.parent_id || IDS.folders.root,
      file_count: 0,
      created_at: new Date().toISOString(),
    }
    return HttpResponse.json({ folder: newFolder }, { status: 201 })
  }),

  // Delete folder
  http.delete(`${API}/api/v1/documents/folders/:id`, ({ params }) => {
    const exists = folders.some((f) => f.id === params.id)
    if (!exists) {
      return HttpResponse.json({ error: 'Folder not found' }, { status: 404 })
    }
    return new HttpResponse(null, { status: 204 })
  }),

  // Folder breadcrumb path
  http.get(`${API}/api/v1/documents/folders/:id/path`, ({ params }) => {
    const folder = folders.find((f) => f.id === params.id)
    if (!folder) {
      return HttpResponse.json({ error: 'Folder not found' }, { status: 404 })
    }
    // Build path from root to target
    const path: typeof folders = []
    let current = folder
    while (current) {
      path.unshift(current)
      current = folders.find((f) => f.id === current!.parent_id) as typeof current
    }
    // Client expects { segments: [{ id, name }] } (was { path } — broke breadcrumbs)
    return HttpResponse.json({ segments: path.map((f) => ({ id: f.id, name: f.name })) })
  }),

  // Init user folder — return root
  http.post(`${API}/api/v1/documents/folders/init/user`, () => {
    const root = folders.find((f) => f.id === IDS.folders.root)
    return HttpResponse.json({ folder: root })
  }),

  // Init team folder
  http.post(`${API}/api/v1/documents/folders/init/team`, () => {
    const teamRoot = folders.find((f) => f.id === 'fld-team-root')
    return HttpResponse.json({ folder: teamRoot })
  }),

  // List files (filter by folder_id)
  http.get(`${API}/api/v1/documents/files`, ({ request }) => {
    const url = new URL(request.url)
    const folderId = url.searchParams.get('folder_id')
    const tagId = url.searchParams.get('tag_id')
    const page = parseInt(url.searchParams.get('page') || '1')
    const perPage = parseInt(url.searchParams.get('per_page') || '50')
    const sortField = url.searchParams.get('sort_field') || 'date'
    const sortDir = url.searchParams.get('sort_dir') || 'desc'

    let filtered = [...files]
    if (folderId) {
      filtered = filtered.filter((f) => f.folder_id === folderId)
    }
    if (tagId) {
      filtered = filtered.filter((f) => f.tags.some((t) => t.id === tagId))
    }
    if (url.searchParams.get('is_favorite') === 'true') {
      filtered = filtered.filter((f) => f.is_favorite === true)
    }

    const dir = sortDir === 'asc' ? 1 : -1
    filtered.sort((a, b) => {
      switch (sortField) {
        case 'name':
          return dir * a.filename.localeCompare(b.filename, undefined, { sensitivity: 'base' })
        case 'size':
          return dir * (a.file_size - b.file_size)
        case 'type':
          return dir * a.mime_type.localeCompare(b.mime_type)
        default:
          return dir * (new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime())
      }
    })

    const start = (page - 1) * perPage
    const paginated = filtered.slice(start, start + perPage)

    return HttpResponse.json({ files: paginated, total: filtered.length, page, per_page: perPage })
  }),

  // File detail
  http.get(`${API}/api/v1/documents/files/:id`, ({ params }) => {
    const file = files.find((f) => f.id === params.id)
    if (!file) {
      return HttpResponse.json({ error: 'File not found' }, { status: 404 })
    }
    return HttpResponse.json({ file })
  }),

  // Delete file
  http.delete(`${API}/api/v1/documents/files/:id`, ({ params }) => {
    const exists = files.some((f) => f.id === params.id)
    if (!exists) {
      return HttpResponse.json({ error: 'File not found' }, { status: 404 })
    }
    return new HttpResponse(null, { status: 204 })
  }),

  // List tags
  http.get(`${API}/api/v1/documents/tags`, () => {
    return HttpResponse.json({ tags })
  }),

  // Search files by name
  http.get(`${API}/api/v1/documents/search`, ({ request }) => {
    const url = new URL(request.url)
    const q = (url.searchParams.get('q') || '').toLowerCase()
    if (!q) {
      return HttpResponse.json({ files: [], total: 0 })
    }
    const results = files.filter((f) => f.name.toLowerCase().includes(q))
    return HttpResponse.json({ files: results, total: results.length })
  }),

  // Shared with me — empty in demo
  http.get(`${API}/api/v1/documents/shares/shared-with-me`, () => {
    return HttpResponse.json({ shares: [], total: 0 })
  }),

  // Virtual files — empty in demo
  http.get(`${API}/api/v1/documents/virtual`, () => {
    return HttpResponse.json({ files: [], total: 0 })
  }),

  // Update file (rename, favorite, folder)
  http.put(`${API}/api/v1/documents/files/:id`, async ({ params, request }) => {
    const file = files.find((f) => f.id === params.id)
    if (!file) {
      return HttpResponse.json({ error: 'File not found' }, { status: 404 })
    }
    const body = (await request.json()) as {
      filename?: string
      folder_id?: string
      is_favorite?: boolean
    }
    if (typeof body.filename === 'string' && body.filename) {
      getActivities(file) // seed before recording
      recordActivity(file.id, 'renamed', body.filename)
      file.filename = body.filename
      file.name = body.filename
    }
    if (typeof body.folder_id === 'string' && body.folder_id) file.folder_id = body.folder_id
    if (typeof body.is_favorite === 'boolean') file.is_favorite = body.is_favorite
    file.updated_at = new Date().toISOString()
    return HttpResponse.json({})
  }),

  // Copy file into target folder
  http.post(`${API}/api/v1/documents/files/:id/copy`, async ({ params, request }) => {
    const src = files.find((f) => f.id === params.id)
    if (!src) {
      return HttpResponse.json({ error: 'File not found' }, { status: 404 })
    }
    const body = (await request.json()) as { target_folder_id?: string }
    const copy: MockFile = {
      ...src,
      id: `file-copy-${Date.now()}`,
      folder_id: body.target_folder_id ?? src.folder_id,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      is_favorite: false,
    }
    files.push(copy)
    getActivities(src)
    recordActivity(src.id, 'copied', folders.find((f) => f.id === copy.folder_id)?.name ?? null)
    return HttpResponse.json({ file: copy }, { status: 201 })
  }),

  // Move file into target folder
  http.post(`${API}/api/v1/documents/files/:id/move`, async ({ params, request }) => {
    const file = files.find((f) => f.id === params.id)
    if (!file) {
      return HttpResponse.json({ error: 'File not found' }, { status: 404 })
    }
    const body = (await request.json()) as { target_folder_id?: string }
    if (body.target_folder_id) {
      getActivities(file)
      recordActivity(file.id, 'moved', folders.find((f) => f.id === body.target_folder_id)?.name ?? null)
      file.folder_id = body.target_folder_id
    }
    file.updated_at = new Date().toISOString()
    return HttpResponse.json({})
  }),

  // Download URL — data: URL so demo downloads save a real (placeholder) file
  http.get(`${API}/api/v1/documents/files/:id/download`, ({ params }) => {
    const file = files.find((f) => f.id === params.id)
    if (!file) {
      return HttpResponse.json({ error: 'File not found' }, { status: 404 })
    }
    // Mime-appropriate placeholder so the viewer shows a real preview in demo
    const url = getDemoFileUrl(file)
    // The viewer also resolves this URL for previews — only log a download
    // if the last entry isn't already a fresh one (keeps the trail readable)
    const acts = getActivities(file)
    const last = acts[acts.length - 1]
    const fresh = last && last.action === 'downloaded' &&
      Date.now() - new Date(last.created_at).getTime() < 10 * 60 * 1000
    if (!fresh) recordActivity(file.id, 'downloaded')
    return HttpResponse.json({ url })
  }),

  // Upload (multipart) — creates a real entry in the demo file list
  http.post(`${API}/api/v1/documents/files/upload`, async ({ request }) => {
    const formData = await request.formData()
    const uploaded = formData.get('file')
    const folderId = formData.get('folder_id')
    if (!(uploaded instanceof File)) {
      return HttpResponse.json({ error: 'No file' }, { status: 400 })
    }
    const newFile: MockFile = {
      id: `file-upload-${Date.now()}`,
      filename: uploaded.name,
      name: uploaded.name,
      mime_type: uploaded.type || 'application/octet-stream',
      file_size: uploaded.size,
      folder_id: typeof folderId === 'string' && folderId ? folderId : IDS.folders.root,
      created_by: IDS.users.markus,
      created_by_name: 'Markus Weber',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      tags: [],
    }
    files.push(newFile)
    return HttpResponse.json({ file: newFile }, { status: 201 })
  }),

  // List versions
  http.get(`${API}/api/v1/documents/files/:id/versions`, ({ params }) => {
    const file = files.find((f) => f.id === params.id)
    if (!file) {
      return HttpResponse.json({ error: 'File not found' }, { status: 404 })
    }
    const versions = [...getVersions(file)].sort((a, b) => b.version_number - a.version_number)
    return HttpResponse.json({ versions })
  }),

  // Create version
  http.post(`${API}/api/v1/documents/files/:id/versions`, async ({ params, request }) => {
    const file = files.find((f) => f.id === params.id)
    if (!file) {
      return HttpResponse.json({ error: 'File not found' }, { status: 404 })
    }
    const body = (await request.json()) as { version_label?: string }
    const versions = getVersions(file)
    const next = Math.max(...versions.map((v) => v.version_number)) + 1
    const version: MockVersion = {
      id: `${file.id}-v${next}`,
      file_id: file.id,
      version_number: next,
      version_label: body.version_label ?? null,
      storage_key: `demo/${file.id}/v${next}`,
      file_size: file.file_size,
      created_by: IDS.users.markus,
      created_by_name: 'Markus Weber',
      created_at: new Date().toISOString(),
    }
    versions.push(version)
    file.updated_at = version.created_at
    getActivities(file)
    recordActivity(file.id, 'version_created', body.version_label ?? null)
    return HttpResponse.json({ version }, { status: 201 })
  }),

  // Revert to version — recorded as a new version on top
  http.post(`${API}/api/v1/documents/files/:id/versions/:n/revert`, ({ params }) => {
    const file = files.find((f) => f.id === params.id)
    if (!file) {
      return HttpResponse.json({ error: 'File not found' }, { status: 404 })
    }
    const versions = getVersions(file)
    const source = versions.find((v) => v.version_number === Number(params.n))
    if (!source) {
      return HttpResponse.json({ error: 'Version not found' }, { status: 404 })
    }
    const next = Math.max(...versions.map((v) => v.version_number)) + 1
    const version: MockVersion = {
      ...source,
      id: `${file.id}-v${next}`,
      version_number: next,
      version_label: `Wiederhergestellt aus Version ${source.version_number}`,
      created_by: IDS.users.markus,
      created_by_name: 'Markus Weber',
      created_at: new Date().toISOString(),
    }
    versions.push(version)
    getActivities(file)
    recordActivity(file.id, 'reverted', `Version ${source.version_number}`)
    return HttpResponse.json({ version }, { status: 201 })
  }),

  // List shares for an entity
  http.get(`${API}/api/v1/documents/shares`, ({ request }) => {
    const url = new URL(request.url)
    const entityType = url.searchParams.get('entity_type')
    const entityId = url.searchParams.get('entity_id')
    const filtered = shares.filter(
      (s) =>
        (!entityType || s.entity_type === entityType) &&
        (!entityId || s.entity_id === entityId),
    )
    return HttpResponse.json({ shares: filtered })
  }),

  // Create share
  http.post(`${API}/api/v1/documents/shares`, async ({ request }) => {
    const body = (await request.json()) as {
      entity_type: string
      entity_id: string
      shared_with_user_id: string
      permission: string
    }
    const share: MockShare = {
      id: `share-${Date.now()}`,
      entity_type: body.entity_type,
      entity_id: body.entity_id,
      shared_with_user_id: body.shared_with_user_id,
      shared_with_user_name: DEMO_USER_NAMES[body.shared_with_user_id] ?? 'Teammitglied',
      permission: body.permission,
      shared_by: IDS.users.markus,
      shared_by_name: 'Markus Weber',
      created_at: new Date().toISOString(),
    }
    shares.push(share)
    if (body.entity_type === 'file') {
      const file = files.find((f) => f.id === body.entity_id)
      if (file) {
        getActivities(file)
        recordActivity(file.id, 'shared', share.shared_with_user_name)
      }
    }
    return HttpResponse.json({ share }, { status: 201 })
  }),

  // Remove share
  http.delete(`${API}/api/v1/documents/shares/:id`, ({ params }) => {
    const idx = shares.findIndex((s) => s.id === params.id)
    if (idx === -1) {
      return HttpResponse.json({ error: 'Share not found' }, { status: 404 })
    }
    shares.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // Create tag
  http.post(`${API}/api/v1/documents/tags`, async ({ request }) => {
    const body = (await request.json()) as { name?: string; color?: string }
    const tag = {
      id: `t-${Date.now()}`,
      name: body.name ?? 'Neues Tag',
      color: body.color ?? '#94a3b8',
    }
    tags.push(tag)
    return HttpResponse.json({ tag }, { status: 201 })
  }),

  // Tag / untag a file
  http.post(`${API}/api/v1/documents/files/:fileId/tags/:tagId`, ({ params }) => {
    const file = files.find((f) => f.id === params.fileId)
    const tag = tags.find((t) => t.id === params.tagId)
    if (!file || !tag) {
      return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    }
    if (!file.tags.some((t) => t.id === tag.id)) file.tags.push(tag)
    return HttpResponse.json({})
  }),

  http.delete(`${API}/api/v1/documents/files/:fileId/tags/:tagId`, ({ params }) => {
    const file = files.find((f) => f.id === params.fileId)
    if (!file) {
      return HttpResponse.json({ error: 'File not found' }, { status: 404 })
    }
    file.tags = file.tags.filter((t) => t.id !== params.tagId)
    return new HttpResponse(null, { status: 204 })
  }),

  // Entity links — empty in demo
  http.get(`${API}/api/v1/documents/files/:id/links`, () => {
    return HttpResponse.json({ links: [] })
  }),

  // File activity (audit trail) — newest first
  http.get(`${API}/api/v1/documents/files/:id/activity`, ({ params }) => {
    const file = files.find((f) => f.id === params.id)
    if (!file) {
      return HttpResponse.json({ error: 'File not found' }, { status: 404 })
    }
    const activities = [...getActivities(file)].sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
    )
    return HttpResponse.json({ activities })
  }),

  // WOPI token (OnlyOffice) — static demo token
  http.post(`${API}/api/v1/documents/files/:id/wopi-token`, () => {
    return HttpResponse.json({ access_token: 'demo-wopi-token', access_token_ttl: 3600 })
  }),
]
