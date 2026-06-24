import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysAgo, hoursAgo, minutesAgo } from '../data/date-helpers'
import {
  mockProjects,
  mockProjectStatuses,
  mockTasks,
  mockTaskComments,
  mockEntityTasks,
} from '../data/projects'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Mutable in-memory stores (deep-cloned from seed data so demo interactions
// — Kanban DnD, status edits, comments, timer, time entries — persist for the
// session without mutating the shared seed objects).
// ---------------------------------------------------------------------------

type AnyRecord = Record<string, unknown>

const clone = <T,>(value: T): T => JSON.parse(JSON.stringify(value)) as T

const projects: AnyRecord[] = clone(mockProjects.projects)
// Seed two reusable project templates so the templates manager has content.
projects.push(
  {
    id: 'prj-tpl-sprint',
    name: 'Sprint-Projekt',
    project_key: 'TPL',
    description: 'Vorlage für zweiwöchige Entwicklungs-Sprints mit Standard-Workflow.',
    status: 'active',
    priority: 'normal',
    is_template: true,
    owner_id: IDS.users.stefan,
    owner_name: 'Stefan Vogel',
    progress: 0,
    member_count: 1,
    task_count: 0,
    completed_task_count: 0,
    created_at: daysAgo(40),
    updated_at: daysAgo(40),
  },
  {
    id: 'prj-tpl-onboarding',
    name: 'Kunden-Onboarding',
    project_key: 'ONB',
    description: 'Vorlage für die Einführung neuer Kunden inkl. Setup-Checkliste.',
    status: 'active',
    priority: 'normal',
    is_template: true,
    owner_id: IDS.users.sarah,
    owner_name: 'Sarah Klein',
    progress: 0,
    member_count: 1,
    task_count: 0,
    completed_task_count: 0,
    created_at: daysAgo(25),
    updated_at: daysAgo(25),
  },
)
const tasks: AnyRecord[] = clone(mockTasks.tasks)
const statusesByProject: Record<string, AnyRecord[]> = clone(mockProjectStatuses)
const comments: AnyRecord[] = clone(mockTaskComments.comments)
const entityTasks: AnyRecord[] = clone(mockEntityTasks.tasks)

const findProject = (id: string) => projects.find((p) => p.id === id)
const findTask = (id: string) => tasks.find((t) => t.id === id)

// Demo team — used to seed project members and time-entry authors.
const TEAM = [
  { id: IDS.users.stefan, first_name: 'Stefan', last_name: 'Vogel' },
  { id: IDS.users.julia, first_name: 'Julia', last_name: 'Weber' },
  { id: IDS.users.thomas, first_name: 'Thomas', last_name: 'Braun' },
  { id: IDS.users.markus, first_name: 'Markus', last_name: 'Fischer' },
  { id: IDS.users.felix, first_name: 'Felix', last_name: 'Schmidt' },
  { id: IDS.users.laura, first_name: 'Laura', last_name: 'Richter' },
  { id: IDS.users.sarah, first_name: 'Sarah', last_name: 'Klein' },
]
const userName = (userId: string) => {
  const u = TEAM.find((t) => t.id === userId)
  return u ? `${u.first_name} ${u.last_name}` : 'Stefan Vogel'
}

// Project members — owner + a deterministic slice of the team per project.
const membersByProject: Record<string, AnyRecord[]> = {}
for (const p of projects) {
  const ownerId = p.owner_id as string
  const count = Math.max(1, Math.min((p.member_count as number) ?? 3, TEAM.length))
  const ordered = [
    ...TEAM.filter((t) => t.id === ownerId),
    ...TEAM.filter((t) => t.id !== ownerId),
  ].slice(0, count)
  membersByProject[p.id as string] = ordered.map((u, i) => ({
    project_id: p.id,
    user_id: u.id,
    role: i === 0 ? 'owner' : i === 1 ? 'member' : 'member',
    added_at: daysAgo(60 - i * 3),
    first_name: u.first_name,
    last_name: u.last_name,
  }))
}

// Per-user project view preferences (keyed by project id).
const preferenceByProject: Record<string, AnyRecord> = {}

// Task dependencies — seed a small graph so the dependency view has content.
const dependencies: AnyRecord[] = [
  {
    id: 'dep-001',
    source_task_id: 'tsk-002',
    target_task_id: 'tsk-003',
    dependency_type: 'blocked_by',
    created_by: IDS.users.stefan,
    created_at: daysAgo(9),
  },
  {
    id: 'dep-002',
    source_task_id: 'tsk-002',
    target_task_id: 'tsk-005',
    dependency_type: 'blocks',
    created_by: IDS.users.stefan,
    created_at: daysAgo(7),
  },
  {
    id: 'dep-003',
    source_task_id: 'tsk-004',
    target_task_id: 'tsk-001',
    dependency_type: 'relates_to',
    created_by: IDS.users.julia,
    created_at: daysAgo(5),
  },
]

// Entity links — seed for a couple of tasks (mirrors entity-tasks seed).
const entityLinks: AnyRecord[] = [
  {
    id: 'lnk-001',
    task_id: 'tsk-018',
    entity_type: 'deal',
    entity_id: IDS.deals.webRedesign,
    entity_display_name: 'Web Redesign Projekt',
    created_by: IDS.users.julia,
    created_at: daysAgo(8),
  },
  {
    id: 'lnk-002',
    task_id: 'tsk-033',
    entity_type: 'company',
    entity_id: IDS.companies.helvetiaSoftware,
    entity_display_name: 'Helvetia Software AG',
    created_by: IDS.users.laura,
    created_at: daysAgo(14),
  },
]

// Task files — seed for tsk-002 so the attachments view has content.
const files: AnyRecord[] = [
  {
    id: 'tfile-001',
    task_id: 'tsk-002',
    filename: 'plugin-api-spec-v2.pdf',
    mime_type: 'application/pdf',
    file_size: 482_133,
    storage_key: 'demo/plugin-api-spec-v2.pdf',
    thumbnail_key: '',
    uploaded_by: IDS.users.stefan,
    uploaded_by_name: 'Stefan Vogel',
    created_at: daysAgo(8),
  },
  {
    id: 'tfile-002',
    task_id: 'tsk-002',
    filename: 'sandbox-benchmark.xlsx',
    mime_type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    file_size: 91_204,
    storage_key: 'demo/sandbox-benchmark.xlsx',
    thumbnail_key: '',
    uploaded_by: IDS.users.thomas,
    uploaded_by_name: 'Thomas Braun',
    created_at: daysAgo(5),
  },
]

// Custom field values per task (Record<string, string>).
const customFieldsByTask: Record<string, Record<string, string>> = {
  'tsk-002': { 'cf-effort': 'XL', 'cf-team': 'Platform' },
}

// Time entries — seed for tsk-002 so time tracking has content.
const timeEntries: AnyRecord[] = [
  {
    id: 'te-001',
    task_id: 'tsk-002',
    user_id: IDS.users.stefan,
    user_name: 'Stefan Vogel',
    started_at: daysAgo(2),
    ended_at: daysAgo(2),
    duration_seconds: 7_200,
    description: 'Sandbox-Isolation Design',
    is_manual: false,
    created_at: daysAgo(2),
    updated_at: daysAgo(2),
  },
  {
    id: 'te-002',
    task_id: 'tsk-002',
    user_id: IDS.users.thomas,
    user_name: 'Thomas Braun',
    started_at: daysAgo(1),
    ended_at: daysAgo(1),
    duration_seconds: 5_400,
    description: 'WASI-Preview-2 Evaluation',
    is_manual: true,
    billed: false,
    created_at: daysAgo(1),
    updated_at: daysAgo(1),
  },
  // Further unbilled hours across Cosmi v2.0 tasks so the project-level
  // "Stunden abrechnen" dialog has a realistic list to invoice.
  { id: 'te-003', task_id: 'tsk-001', user_id: IDS.users.julia, user_name: 'Julia Weber', started_at: daysAgo(3), ended_at: daysAgo(3), duration_seconds: 21_600, description: 'Design-System Komponenten umgesetzt', is_manual: false, billed: false, created_at: daysAgo(3), updated_at: daysAgo(3) },
  { id: 'te-004', task_id: 'tsk-001', user_id: IDS.users.julia, user_name: 'Julia Weber', started_at: daysAgo(4), ended_at: daysAgo(4), duration_seconds: 9_000, description: 'Storybook-Dokumentation', is_manual: true, billed: false, created_at: daysAgo(4), updated_at: daysAgo(4) },
  { id: 'te-005', task_id: 'tsk-003', user_id: IDS.users.felix, user_name: 'Felix Schmidt', started_at: daysAgo(4), ended_at: daysAgo(4), duration_seconds: 14_400, description: 'Dashboard-Widget-Grid implementiert', is_manual: false, billed: false, created_at: daysAgo(4), updated_at: daysAgo(4) },
  { id: 'te-006', task_id: 'tsk-004', user_id: IDS.users.thomas, user_name: 'Thomas Braun', started_at: daysAgo(5), ended_at: daysAgo(5), duration_seconds: 18_000, description: 'REST-Endpoints + Tests', is_manual: false, billed: false, created_at: daysAgo(5), updated_at: daysAgo(5) },
  { id: 'te-007', task_id: 'tsk-005', user_id: IDS.users.markus, user_name: 'Markus Fischer', started_at: daysAgo(6), ended_at: daysAgo(6), duration_seconds: 12_600, description: 'CI/CD-Pipeline eingerichtet', is_manual: true, billed: false, created_at: daysAgo(6), updated_at: daysAgo(6) },
  { id: 'te-008', task_id: 'tsk-001', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', started_at: daysAgo(8), ended_at: daysAgo(8), duration_seconds: 7_200, description: 'Design-Review (bereits abgerechnet)', is_manual: false, billed: true, created_at: daysAgo(8), updated_at: daysAgo(8) },
]

// Active timer (null = none running).
let activeTimer: AnyRecord | null = null

// ---------------------------------------------------------------------------
// ID helpers — Date.now() is allowed in MSW handlers (runs in browser).
// ---------------------------------------------------------------------------

const newId = (prefix: string) => `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1e4)}`
const nowIso = () => new Date().toISOString()

// ---------------------------------------------------------------------------
// Synthesised task activity log — derived per task so the view never empties.
// ---------------------------------------------------------------------------

function activitiesForTask(taskId: string): AnyRecord[] {
  const task = findTask(taskId)
  if (!task) return []
  const reporter = (task.reporter_id as string) ?? IDS.users.stefan
  const assignee = (task.assignee_id as string) ?? IDS.users.stefan
  const out: AnyRecord[] = [
    {
      id: `${taskId}-act-1`,
      task_id: taskId,
      actor_id: reporter,
      actor_name: userName(reporter),
      action: 'created',
      field_name: '',
      old_value: '',
      new_value: '',
      created_at: task.created_at,
    },
    {
      id: `${taskId}-act-2`,
      task_id: taskId,
      actor_id: reporter,
      actor_name: userName(reporter),
      action: 'assigned',
      field_name: 'assignee',
      old_value: '',
      new_value: userName(assignee),
      created_at: hoursAgo(48),
    },
    {
      id: `${taskId}-act-3`,
      task_id: taskId,
      actor_id: assignee,
      actor_name: userName(assignee),
      action: 'status_changed',
      field_name: 'status',
      old_value: 'Offen',
      new_value: (task.status_name as string) ?? 'In Bearbeitung',
      created_at: hoursAgo(6),
    },
  ]
  return out
}

export const workHandlers = [
  // ── Projects ────────────────────────────────────────────────────────────

  // GET /api/v1/projects — list (status, search, templates_only, page)
  http.get(`${API}/api/v1/projects`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    const search = url.searchParams.get('search') ?? ''
    const templatesOnly = url.searchParams.get('templates_only') === 'true'
    const includeArchived = url.searchParams.get('include_archived') === 'true'
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 50)

    let filtered = projects
    if (!includeArchived) {
      filtered = filtered.filter((p) => p.status !== 'archived')
    }
    if (templatesOnly) {
      filtered = filtered.filter((p) => p.is_template === true)
    } else {
      filtered = filtered.filter((p) => p.is_template !== true)
    }
    if (status) {
      filtered = filtered.filter((p) => p.status === status)
    }
    if (search) {
      const q = search.toLowerCase()
      filtered = filtered.filter(
        (p) =>
          String(p.name).toLowerCase().includes(q) ||
          String(p.description).toLowerCase().includes(q) ||
          String(p.project_key).toLowerCase().includes(q),
      )
    }

    const start = (page - 1) * pageSize
    const paged = filtered.slice(start, start + pageSize)

    return HttpResponse.json({
      projects: paged,
      total: filtered.length,
      page,
      page_size: pageSize,
    })
  }),

  // GET /api/v1/projects/:id — detail (wrapped in { project })
  http.get(`${API}/api/v1/projects/:id`, ({ params }) => {
    const project = findProject(params.id as string)
    if (!project) {
      return HttpResponse.json({ error: 'Project not found' }, { status: 404 })
    }
    return HttpResponse.json({ project })
  }),

  // POST /api/v1/projects — create
  http.post(`${API}/api/v1/projects`, async ({ request }) => {
    const body = (await request.json()) as AnyRecord
    const project: AnyRecord = {
      id: newId('prj'),
      status: 'active',
      priority: 'normal',
      is_template: false,
      owner_id: IDS.users.stefan,
      owner_name: 'Stefan Vogel',
      progress: 0,
      member_count: 1,
      task_count: 0,
      completed_task_count: 0,
      ...body,
      created_at: nowIso(),
      updated_at: nowIso(),
    }
    projects.unshift(project)
    statusesByProject[project.id as string] = [
      { id: `${project.id}-st-1`, name: 'Offen', color: '#94a3b8', sort_order: 0, is_default: true, is_closed: false },
      { id: `${project.id}-st-2`, name: 'In Bearbeitung', color: '#3b82f6', sort_order: 1, is_default: false, is_closed: false },
      { id: `${project.id}-st-3`, name: 'Review', color: '#a78bfa', sort_order: 2, is_default: false, is_closed: false },
      { id: `${project.id}-st-4`, name: 'Erledigt', color: '#22c55e', sort_order: 3, is_default: false, is_closed: true },
    ]
    membersByProject[project.id as string] = [
      { project_id: project.id, user_id: IDS.users.stefan, role: 'owner', added_at: nowIso(), first_name: 'Stefan', last_name: 'Vogel' },
    ]
    return HttpResponse.json({ project }, { status: 201 })
  }),

  // PUT /api/v1/projects/:id — update
  http.put(`${API}/api/v1/projects/:id`, async ({ params, request }) => {
    const project = findProject(params.id as string)
    if (!project) {
      return HttpResponse.json({ error: 'Project not found' }, { status: 404 })
    }
    const body = (await request.json()) as AnyRecord
    Object.assign(project, body, { updated_at: nowIso() })
    return HttpResponse.json({ project })
  }),

  // POST /api/v1/projects/:id/archive — archive
  http.post(`${API}/api/v1/projects/:id/archive`, ({ params }) => {
    const project = findProject(params.id as string)
    if (!project) {
      return HttpResponse.json({ error: 'Project not found' }, { status: 404 })
    }
    project.status = 'archived'
    project.updated_at = nowIso()
    return HttpResponse.json({ project })
  }),

  // POST /api/v1/projects/:id/template — save as template
  http.post(`${API}/api/v1/projects/:id/template`, async ({ params, request }) => {
    const source = findProject(params.id as string)
    const body = (await request.json().catch(() => ({}))) as AnyRecord
    const template: AnyRecord = {
      ...(source ? clone(source) : {}),
      id: newId('prj-tpl'),
      name: (body.template_name as string) || `${source?.name ?? 'Projekt'} (Vorlage)`,
      is_template: true,
      progress: 0,
      task_count: 0,
      completed_task_count: 0,
      created_at: nowIso(),
      updated_at: nowIso(),
    }
    projects.unshift(template)
    return HttpResponse.json({ project: template }, { status: 201 })
  }),

  // POST /api/v1/projects/from-template — create from template
  http.post(`${API}/api/v1/projects/from-template`, async ({ request }) => {
    const body = (await request.json()) as AnyRecord
    const tpl = findProject(body.template_id as string)
    const project: AnyRecord = {
      ...(tpl ? clone(tpl) : {}),
      id: newId('prj'),
      name: body.name,
      project_key: body.project_key,
      is_template: false,
      status: 'active',
      progress: 0,
      member_count: 1,
      task_count: 0,
      completed_task_count: 0,
      owner_id: IDS.users.stefan,
      owner_name: 'Stefan Vogel',
      created_at: nowIso(),
      updated_at: nowIso(),
    }
    projects.unshift(project)
    statusesByProject[project.id as string] = clone(
      statusesByProject[body.template_id as string] ?? statusesByProject[projects[1]?.id as string] ?? [],
    ).map((s, i) => ({ ...s, id: `${project.id}-st-${i + 1}` }))
    return HttpResponse.json({ project }, { status: 201 })
  }),

  // ── Project members ───────────────────────────────────────────────────────

  // GET /api/v1/projects/:id/members
  http.get(`${API}/api/v1/projects/:id/members`, ({ params }) => {
    const members = membersByProject[params.id as string] ?? []
    return HttpResponse.json({ members })
  }),

  // POST /api/v1/projects/:id/members — add member
  http.post(`${API}/api/v1/projects/:id/members`, async ({ params, request }) => {
    const projectId = params.id as string
    const body = (await request.json()) as AnyRecord
    const userId = body.user_id as string
    const u = TEAM.find((t) => t.id === userId)
    const member: AnyRecord = {
      project_id: projectId,
      user_id: userId,
      role: (body.role as string) ?? 'member',
      added_at: nowIso(),
      first_name: u?.first_name ?? 'Neues',
      last_name: u?.last_name ?? 'Mitglied',
    }
    const list = (membersByProject[projectId] ??= [])
    if (!list.some((m) => m.user_id === userId)) list.push(member)
    return HttpResponse.json({ member }, { status: 201 })
  }),

  // PUT /api/v1/projects/:id/members/:userId — update role
  http.put(`${API}/api/v1/projects/:id/members/:userId`, async ({ params, request }) => {
    const list = membersByProject[params.id as string] ?? []
    const member = list.find((m) => m.user_id === params.userId)
    if (!member) return HttpResponse.json({ error: 'Member not found' }, { status: 404 })
    const body = (await request.json()) as AnyRecord
    if (body.role) member.role = body.role
    return HttpResponse.json({ member })
  }),

  // DELETE /api/v1/projects/:id/members/:userId — remove member
  http.delete(`${API}/api/v1/projects/:id/members/:userId`, ({ params }) => {
    const projectId = params.id as string
    const list = membersByProject[projectId] ?? []
    membersByProject[projectId] = list.filter((m) => m.user_id !== params.userId)
    return new HttpResponse(null, { status: 200 })
  }),

  // ── Project statuses ──────────────────────────────────────────────────────

  // GET /api/v1/projects/:id/statuses
  http.get(`${API}/api/v1/projects/:id/statuses`, ({ params }) => {
    const statuses = statusesByProject[params.id as string]
    if (!statuses) {
      return HttpResponse.json({ statuses: [] })
    }
    return HttpResponse.json({ statuses })
  }),

  // POST /api/v1/projects/:id/statuses — add status
  http.post(`${API}/api/v1/projects/:id/statuses`, async ({ params, request }) => {
    const projectId = params.id as string
    const body = (await request.json()) as AnyRecord
    const list = (statusesByProject[projectId] ??= [])
    const status: AnyRecord = {
      id: newId(`${projectId}-st`),
      project_id: projectId,
      name: (body.name as string) || 'Neue Spalte',
      color: (body.color as string) || '#64748b',
      sort_order: list.length,
      is_default: body.is_default ?? false,
      is_closed: body.is_closed ?? false,
      created_at: nowIso(),
    }
    list.push(status)
    return HttpResponse.json({ status }, { status: 201 })
  }),

  // POST /api/v1/projects/:id/statuses/reorder
  http.post(`${API}/api/v1/projects/:id/statuses/reorder`, async ({ params, request }) => {
    const projectId = params.id as string
    const body = (await request.json()) as { status_ids?: string[] }
    const list = statusesByProject[projectId] ?? []
    if (body.status_ids) {
      const order = new Map(body.status_ids.map((id, i) => [id, i]))
      list.sort((a, b) => (order.get(a.id as string) ?? 0) - (order.get(b.id as string) ?? 0))
      list.forEach((s, i) => (s.sort_order = i))
    }
    return new HttpResponse(null, { status: 200 })
  }),

  // PUT /api/v1/project-statuses/:id — update status
  http.put(`${API}/api/v1/project-statuses/:id`, async ({ params, request }) => {
    const statusId = params.id as string
    const body = (await request.json()) as AnyRecord
    for (const list of Object.values(statusesByProject)) {
      const s = list.find((x) => x.id === statusId)
      if (s) {
        Object.assign(s, body)
        return new HttpResponse(null, { status: 200 })
      }
    }
    return new HttpResponse(null, { status: 200 })
  }),

  // DELETE /api/v1/project-statuses/:id — delete status
  http.delete(`${API}/api/v1/project-statuses/:id`, ({ params }) => {
    const statusId = params.id as string
    for (const key of Object.keys(statusesByProject)) {
      statusesByProject[key] = statusesByProject[key].filter((s) => s.id !== statusId)
    }
    return new HttpResponse(null, { status: 200 })
  }),

  // ── Project view preferences ───────────────────────────────────────────────

  // GET /api/v1/projects/:id/preferences
  http.get(`${API}/api/v1/projects/:id/preferences`, ({ params }) => {
    const projectId = params.id as string
    // No stored view_type → leave it unset so the user's personal default view
    // (work settings) applies on first open. Set once the user picks a view.
    const pref = preferenceByProject[projectId] ?? {
      user_id: IDS.users.stefan,
      project_id: projectId,
      list_group_by: 'status',
      list_sort_by: 'sort_order',
      list_sort_desc: false,
    }
    return HttpResponse.json(pref)
  }),

  // PUT /api/v1/projects/:id/preferences
  http.put(`${API}/api/v1/projects/:id/preferences`, async ({ params, request }) => {
    const projectId = params.id as string
    const body = (await request.json()) as AnyRecord
    preferenceByProject[projectId] = {
      user_id: IDS.users.stefan,
      project_id: projectId,
      ...preferenceByProject[projectId],
      ...body,
    }
    return new HttpResponse(null, { status: 200 })
  }),

  // ── Tasks ───────────────────────────────────────────────────────────────

  // GET /api/v1/tasks — list with filters
  http.get(`${API}/api/v1/tasks`, ({ request }) => {
    const url = new URL(request.url)
    const projectId = url.searchParams.get('project_id')
    const assigneeId = url.searchParams.get('assignee_id')
    const statusId = url.searchParams.get('status_id')
    const parentId = url.searchParams.get('parent_task_id')
    const priority = url.searchParams.get('priority')
    const search = (url.searchParams.get('search') ?? '').toLowerCase()

    let filtered = tasks.filter((t) => !t.parent_task_id)

    if (parentId) {
      filtered = tasks.filter((t) => t.parent_task_id === parentId)
    }
    if (projectId) {
      filtered = filtered.filter((t) => t.project_id === projectId)
    }
    if (assigneeId) {
      const mine = filtered.filter((t) => t.assignee_id === assigneeId)
      // The demo user id rarely matches the seeded assignees, so show Stefan's
      // tasks as the baseline "my tasks" — merged with anything actually assigned
      // to the requester (e.g. freshly created standalone tasks) so those appear
      // too instead of collapsing the list.
      const demoFallback = filtered.filter(
        (t) => t.assignee_id === IDS.users.stefan && t.assignee_id !== assigneeId,
      )
      filtered = [...mine, ...demoFallback]
    }
    if (statusId) {
      filtered = filtered.filter((t) => t.status_id === statusId)
    }
    if (priority) {
      filtered = filtered.filter((t) => t.priority === priority)
    }
    if (search) {
      filtered = filtered.filter(
        (t) =>
          String(t.title).toLowerCase().includes(search) ||
          String(t.description).toLowerCase().includes(search),
      )
    }

    return HttpResponse.json({ tasks: filtered, total: filtered.length })
  }),

  // GET /api/v1/tasks/:id — detail (wrapped in { task })
  http.get(`${API}/api/v1/tasks/:id`, ({ params }) => {
    const task = findTask(params.id as string)
    if (!task) {
      return HttpResponse.json({ error: 'Task not found' }, { status: 404 })
    }
    return HttpResponse.json({ task })
  }),

  // POST /api/v1/tasks — create
  http.post(`${API}/api/v1/tasks`, async ({ request }) => {
    const body = (await request.json()) as AnyRecord
    const project = findProject(body.project_id as string)
    const status = project
      ? (statusesByProject[project.id as string] ?? []).find((s) => s.id === body.status_id) ??
        (statusesByProject[project.id as string] ?? [])[0]
      : undefined
    const task: AnyRecord = {
      id: newId('tsk'),
      status_name: status?.name ?? 'Offen',
      status_color: status?.color ?? '#94a3b8',
      project_name: project?.name ?? '',
      project_key: project?.project_key ?? '',
      tags: [],
      subtask_count: 0,
      completed_subtask_count: 0,
      comment_count: 0,
      sort_order: tasks.length,
      ...body,
      created_at: nowIso(),
      updated_at: nowIso(),
    }
    tasks.push(task)
    return HttpResponse.json({ task }, { status: 201 })
  }),

  // PUT /api/v1/tasks/:id — update
  http.put(`${API}/api/v1/tasks/:id`, async ({ params, request }) => {
    const task = findTask(params.id as string)
    if (!task) {
      return HttpResponse.json({ error: 'Task not found' }, { status: 404 })
    }
    const body = (await request.json()) as AnyRecord
    if (body.status_id) {
      const list = statusesByProject[task.project_id as string] ?? []
      const status = list.find((s) => s.id === body.status_id)
      if (status) {
        task.status_name = status.name
        task.status_color = status.color
      }
    }
    if (body.assignee_id) {
      task.assignee_name = userName(body.assignee_id as string)
    }
    // Reassign to another project: pull the destination project's key/name and
    // mint a fresh task number so the moved task renders correctly grouped.
    if (body.project_id && body.project_id !== task.project_id) {
      const project = findProject(body.project_id as string)
      if (project) {
        task.project_name = project.name
        task.project_key = project.project_key
        const maxNumber = tasks
          .filter((t) => t.project_id === project.id)
          .reduce((max, t) => Math.max(max, (t.task_number as number) ?? 0), 0)
        task.task_number = maxNumber + 1
        const list = statusesByProject[project.id as string] ?? []
        if (list.length > 0) {
          task.status_id = list[0].id
          task.status_name = list[0].name
          task.status_color = list[0].color
        }
      }
    }
    Object.assign(task, body, { updated_at: nowIso() })
    return HttpResponse.json({ task })
  }),

  // DELETE /api/v1/tasks/:id — delete
  http.delete(`${API}/api/v1/tasks/:id`, ({ params }) => {
    const idx = tasks.findIndex((t) => t.id === params.id)
    if (idx === -1) {
      return HttpResponse.json({ error: 'Task not found' }, { status: 404 })
    }
    tasks.splice(idx, 1)
    return new HttpResponse(null, { status: 200 })
  }),

  // POST /api/v1/tasks/:id/move — Kanban status change
  http.post(`${API}/api/v1/tasks/:id/move`, async ({ params, request }) => {
    const task = findTask(params.id as string)
    if (!task) {
      return HttpResponse.json({ error: 'Task not found' }, { status: 404 })
    }
    const body = (await request.json()) as { status_id?: string; sort_order?: number }
    if (body.status_id) {
      const list = statusesByProject[task.project_id as string] ?? []
      const status = list.find((s) => s.id === body.status_id)
      task.status_id = body.status_id
      if (status) {
        task.status_name = status.name
        task.status_color = status.color
      }
    }
    if (typeof body.sort_order === 'number') task.sort_order = body.sort_order
    task.updated_at = nowIso()
    return HttpResponse.json({ task })
  }),

  // GET /api/v1/tasks/:id/subtasks
  http.get(`${API}/api/v1/tasks/:id/subtasks`, ({ params }) => {
    const parentId = params.id as string
    const subtasks = tasks.filter((t) => t.parent_task_id === parentId)
    return HttpResponse.json({ tasks: subtasks })
  }),

  // ── Task dependencies ───────────────────────────────────────────────────

  // GET /api/v1/tasks/:id/dependencies
  http.get(`${API}/api/v1/tasks/:id/dependencies`, ({ params }) => {
    const taskId = params.id as string
    const deps = dependencies.filter(
      (d) => d.source_task_id === taskId || d.target_task_id === taskId,
    )
    return HttpResponse.json({ dependencies: deps })
  }),

  // POST /api/v1/tasks/:id/dependencies
  http.post(`${API}/api/v1/tasks/:id/dependencies`, async ({ params, request }) => {
    const body = (await request.json()) as AnyRecord
    const dependency: AnyRecord = {
      id: newId('dep'),
      source_task_id: params.id as string,
      target_task_id: body.target_task_id,
      dependency_type: body.dependency_type ?? 'relates_to',
      created_by: IDS.users.stefan,
      created_at: nowIso(),
    }
    dependencies.push(dependency)
    return HttpResponse.json({ dependency }, { status: 201 })
  }),

  // DELETE /api/v1/task-dependencies/:id
  http.delete(`${API}/api/v1/task-dependencies/:id`, ({ params }) => {
    const idx = dependencies.findIndex((d) => d.id === params.id)
    if (idx !== -1) dependencies.splice(idx, 1)
    return new HttpResponse(null, { status: 200 })
  }),

  // ── Task comments ─────────────────────────────────────────────────────────

  // GET /api/v1/tasks/:id/comments
  http.get(`${API}/api/v1/tasks/:id/comments`, ({ params }) => {
    const taskId = params.id as string
    const list = comments.filter((c) => c.task_id === taskId)
    return HttpResponse.json({ comments: list, total: list.length })
  }),

  // POST /api/v1/tasks/:id/comments — add comment
  http.post(`${API}/api/v1/tasks/:id/comments`, async ({ params, request }) => {
    const body = (await request.json()) as AnyRecord
    let quotedPreview: string | undefined
    if (body.quoted_comment_id) {
      const quoted = comments.find((c) => c.id === body.quoted_comment_id)
      quotedPreview = quoted ? String(quoted.content).slice(0, 120) : undefined
    }
    const comment: AnyRecord = {
      id: newId('tcm'),
      task_id: params.id,
      author_id: IDS.users.stefan,
      author_name: 'Stefan Vogel',
      content: (body.content as string) || '',
      quoted_comment_id: body.quoted_comment_id,
      quoted_content_preview: quotedPreview,
      created_at: nowIso(),
      updated_at: nowIso(),
    }
    comments.push(comment)
    const task = findTask(params.id as string)
    if (task) task.comment_count = ((task.comment_count as number) ?? 0) + 1
    return HttpResponse.json({ comment }, { status: 201 })
  }),

  // PUT /api/v1/task-comments/:id — edit comment
  http.put(`${API}/api/v1/task-comments/:id`, async ({ params, request }) => {
    const comment = comments.find((c) => c.id === params.id)
    if (!comment) return new HttpResponse(null, { status: 200 })
    const body = (await request.json()) as AnyRecord
    comment.content = body.content ?? comment.content
    comment.updated_at = nowIso()
    return new HttpResponse(null, { status: 200 })
  }),

  // DELETE /api/v1/task-comments/:id
  http.delete(`${API}/api/v1/task-comments/:id`, ({ params }) => {
    const idx = comments.findIndex((c) => c.id === params.id)
    if (idx !== -1) {
      const [removed] = comments.splice(idx, 1)
      const task = findTask(removed.task_id as string)
      if (task) task.comment_count = Math.max(0, ((task.comment_count as number) ?? 1) - 1)
    }
    return new HttpResponse(null, { status: 200 })
  }),

  // ── Task entity links (CRM cross-module) ────────────────────────────────────

  // GET /api/v1/tasks/:id/links
  http.get(`${API}/api/v1/tasks/:id/links`, ({ params }) => {
    const taskId = params.id as string
    const links = entityLinks.filter((l) => l.task_id === taskId)
    return HttpResponse.json({ links })
  }),

  // POST /api/v1/tasks/:id/links — link CRM entity
  http.post(`${API}/api/v1/tasks/:id/links`, async ({ params, request }) => {
    const body = (await request.json()) as AnyRecord
    const link: AnyRecord = {
      id: newId('lnk'),
      task_id: params.id,
      entity_type: body.entity_type,
      entity_id: body.entity_id,
      entity_display_name: (body.entity_display_name as string) ?? 'Verknüpfter Datensatz',
      created_by: IDS.users.stefan,
      created_at: nowIso(),
    }
    entityLinks.push(link)
    return HttpResponse.json({ link }, { status: 201 })
  }),

  // DELETE /api/v1/task-links/:id
  http.delete(`${API}/api/v1/task-links/:id`, ({ params }) => {
    const idx = entityLinks.findIndex((l) => l.id === params.id)
    if (idx !== -1) entityLinks.splice(idx, 1)
    return new HttpResponse(null, { status: 200 })
  }),

  // ── Entity tasks (cross-module) ─────────────────────────────────────────

  // GET /api/v1/entity-tasks
  http.get(`${API}/api/v1/entity-tasks`, ({ request }) => {
    const url = new URL(request.url)
    const entityType = url.searchParams.get('entity_type')
    const entityId = url.searchParams.get('entity_id')

    if (entityType && entityId) {
      const linked = entityLinks.filter(
        (l) => l.entity_type === entityType && l.entity_id === entityId,
      )
      const linkedTaskIds = new Set(linked.map((l) => l.task_id))
      const fromLinks = tasks.filter((t) => linkedTaskIds.has(t.id))
      const fromSeed = entityTasks
        .filter((et) => et.entity_type === entityType && et.entity_id === entityId)
        .map((et) => findTask(et.task_id as string))
        .filter(Boolean) as AnyRecord[]
      const merged = [...fromLinks]
      for (const t of fromSeed) if (!merged.some((m) => m.id === t.id)) merged.push(t)
      return HttpResponse.json({ tasks: merged, total: merged.length })
    }

    return HttpResponse.json({ tasks: [], total: 0 })
  }),

  // ── Task activities ─────────────────────────────────────────────────────

  // GET /api/v1/tasks/:id/activities
  http.get(`${API}/api/v1/tasks/:id/activities`, ({ params }) => {
    const list = activitiesForTask(params.id as string)
    return HttpResponse.json({ activities: list, total: list.length })
  }),

  // ── Task files ──────────────────────────────────────────────────────────

  // GET /api/v1/tasks/:id/files
  http.get(`${API}/api/v1/tasks/:id/files`, ({ params }) => {
    const taskId = params.id as string
    const list = files.filter((f) => f.task_id === taskId)
    return HttpResponse.json({ files: list })
  }),

  // POST /api/v1/tasks/:id/files — attach file metadata
  http.post(`${API}/api/v1/tasks/:id/files`, async ({ params, request }) => {
    const body = (await request.json()) as AnyRecord
    const file: AnyRecord = {
      id: newId('tfile'),
      task_id: params.id,
      filename: body.filename,
      mime_type: body.mime_type,
      file_size: body.file_size,
      storage_key: body.storage_key,
      thumbnail_key: body.thumbnail_key ?? '',
      uploaded_by: IDS.users.stefan,
      uploaded_by_name: 'Stefan Vogel',
      created_at: nowIso(),
    }
    files.push(file)
    return HttpResponse.json({ file }, { status: 201 })
  }),

  // DELETE /api/v1/task-files/:id
  http.delete(`${API}/api/v1/task-files/:id`, ({ params }) => {
    const idx = files.findIndex((f) => f.id === params.id)
    if (idx !== -1) files.splice(idx, 1)
    return new HttpResponse(null, { status: 200 })
  }),

  // ── Task custom fields ──────────────────────────────────────────────────

  // GET /api/v1/tasks/:id/custom-fields
  http.get(`${API}/api/v1/tasks/:id/custom-fields`, ({ params }) => {
    const values = customFieldsByTask[params.id as string] ?? {}
    return HttpResponse.json({ custom_fields: values })
  }),

  // PUT /api/v1/tasks/:id/custom-fields
  http.put(`${API}/api/v1/tasks/:id/custom-fields`, async ({ params, request }) => {
    const body = (await request.json()) as { values?: Array<{ field_id: string; value: string }> }
    const map: Record<string, string> = {}
    for (const v of body.values ?? []) map[v.field_id] = v.value
    customFieldsByTask[params.id as string] = map
    return new HttpResponse(null, { status: 200 })
  }),

  // ── Search ──────────────────────────────────────────────────────────────

  // GET /api/v1/work/search
  http.get(`${API}/api/v1/work/search`, ({ request }) => {
    const url = new URL(request.url)
    const query = (url.searchParams.get('q') || '').toLowerCase()
    if (!query) {
      return HttpResponse.json({ tasks: [], total: 0 })
    }
    const results = tasks.filter(
      (t) =>
        String(t.title).toLowerCase().includes(query) ||
        String(t.description).toLowerCase().includes(query) ||
        (Array.isArray(t.tags) && (t.tags as string[]).some((tag) => tag.toLowerCase().includes(query))),
    )
    return HttpResponse.json({ tasks: results, total: results.length })
  }),

  // ── Timer ───────────────────────────────────────────────────────────────

  // GET /api/v1/timer/active
  http.get(`${API}/api/v1/timer/active`, () => {
    return HttpResponse.json({ timer: activeTimer })
  }),

  // POST /api/v1/tasks/:id/timer/start
  http.post(`${API}/api/v1/tasks/:id/timer/start`, ({ params }) => {
    const taskId = params.id as string
    const task = findTask(taskId)
    let stoppedEntry: AnyRecord | undefined
    if (activeTimer) {
      // Auto-stop the previous timer into a completed entry.
      const startedAt = activeTimer.started_at as string
      const duration = Math.max(
        60,
        Math.floor((Date.now() - new Date(startedAt).getTime()) / 1000),
      )
      stoppedEntry = {
        id: newId('te'),
        task_id: activeTimer.task_id,
        user_id: IDS.users.stefan,
        user_name: 'Stefan Vogel',
        started_at: startedAt,
        ended_at: nowIso(),
        duration_seconds: duration,
        description: '',
        is_manual: false,
        created_at: nowIso(),
        updated_at: nowIso(),
      }
      timeEntries.push(stoppedEntry)
    }
    activeTimer = {
      id: newId('timer'),
      task_id: taskId,
      task_title: (task?.title as string) ?? '',
      user_id: IDS.users.stefan,
      started_at: nowIso(),
    }
    const entry: AnyRecord = {
      id: activeTimer.id,
      task_id: taskId,
      user_id: IDS.users.stefan,
      user_name: 'Stefan Vogel',
      started_at: activeTimer.started_at,
      duration_seconds: 0,
      description: '',
      is_manual: false,
      created_at: nowIso(),
      updated_at: nowIso(),
    }
    return HttpResponse.json({ entry, stopped_entry: stoppedEntry }, { status: 201 })
  }),

  // POST /api/v1/timer/stop
  http.post(`${API}/api/v1/timer/stop`, () => {
    if (!activeTimer) {
      return HttpResponse.json({ error: 'No active timer' }, { status: 404 })
    }
    const startedAt = activeTimer.started_at as string
    const duration = Math.max(
      60,
      Math.floor((Date.now() - new Date(startedAt).getTime()) / 1000),
    )
    const entry: AnyRecord = {
      id: newId('te'),
      task_id: activeTimer.task_id,
      user_id: IDS.users.stefan,
      user_name: 'Stefan Vogel',
      started_at: startedAt,
      ended_at: nowIso(),
      duration_seconds: duration,
      description: '',
      is_manual: false,
      created_at: nowIso(),
      updated_at: nowIso(),
    }
    timeEntries.push(entry)
    activeTimer = null
    return HttpResponse.json({ entry })
  }),

  // GET /api/v1/projects/:id/team-utilization — capacity/utilization roll-up
  // derived from the project's real members (design preview data).
  http.get(`${API}/api/v1/projects/:id/team-utilization`, ({ params }) => {
    const projectId = params.id as string
    const members = membersByProject[projectId] ?? []
    const ROLES = ['Frontend Lead', 'Backend Dev', 'DevOps', 'UI/UX Design', 'QA Engineer', 'Projektleitung', 'Fullstack Dev']
    const weeks = ['KW 4', 'KW 5', 'KW 6', 'KW 7', 'KW 8', 'KW 9']
    const monthsLabels = ['Dez 2025', 'Jan 2026', 'Feb 2026']
    const gen = (target: number, labels: string[], i: number, mult = 1) =>
      labels.map((label, j) => ({
        label,
        hours:
          Math.round(target * mult * (0.6 + Math.abs(Math.sin((i + 1) * 2.7 + j * 1.3)) * 0.8) * 10) / 10,
      }))
    const team = members.map((m, i) => {
      const name = `${m.first_name} ${m.last_name}`
      const weeklyTarget = i % 5 === 2 ? 32 : i % 5 === 4 ? 24 : 40
      const rate = 120 + (i % 5) * 10
      return {
        member: {
          id: m.user_id,
          name,
          role: ROLES[i % ROLES.length],
          avatarInitial: name.charAt(0),
          weeklyTarget,
          rate,
        },
        weeklyData: gen(weeklyTarget, weeks, i),
        monthlyData: gen(weeklyTarget, monthsLabels, i, 4.33),
      }
    })
    return HttpResponse.json({ team })
  }),

  // GET /api/v1/projects/:id/guest-overview — milestones + status updates for
  // the read-only guest project view (derived per project, design preview).
  http.get(`${API}/api/v1/projects/:id/guest-overview`, ({ params }) => {
    const projectId = params.id as string
    const project = findProject(projectId)
    const members = membersByProject[projectId] ?? []
    const authorAt = (i: number) => {
      const m = members[i % Math.max(members.length, 1)]
      return m ? `${m.first_name} ${m.last_name}` : 'Team'
    }
    const startMs = project?.start_date ? new Date(project.start_date as string).getTime() : Date.now()
    const endMs = project?.end_date ? new Date(project.end_date as string).getTime() : Date.now()
    const total = project?.task_count ? (project.task_count as number) : 1
    const done = (project?.completed_task_count as number) ?? 0
    const milestoneTitles = [
      'Konzept & Setup',
      'Design-Phase',
      'Entwicklung Kern',
      'Integration',
      'QA & Testing',
      'Go-Live',
    ]
    const completedCount = Math.round((done / Math.max(total, 1)) * milestoneTitles.length)
    const milestones = milestoneTitles.map((title, i) => {
      const status = i < completedCount ? 'completed' : i === completedCount ? 'in-progress' : 'upcoming'
      const dueMs = startMs + ((endMs - startMs) * (i + 1)) / milestoneTitles.length
      return {
        id: `ms-${i + 1}`,
        title,
        dueDate: new Date(dueMs).toISOString(),
        status,
        progress: status === 'completed' ? 100 : status === 'in-progress' ? 60 : 0,
      }
    })
    const statusUpdates = [
      { id: 'su1', date: daysAgo(2), author: authorAt(0), text: 'Kernfunktionen umgesetzt; Performance-Benchmarks deutlich verbessert.', type: 'update' },
      { id: 'su2', date: daysAgo(4), author: authorAt(1), text: 'API-Endpoints implementiert und getestet, Authentifizierung steht.', type: 'update' },
      { id: 'su3', date: daysAgo(7), author: authorAt(0), text: `Meilenstein „${milestoneTitles[Math.max(completedCount - 1, 0)]}" erfolgreich abgeschlossen.`, type: 'milestone' },
      { id: 'su4', date: daysAgo(9), author: authorAt(2), text: 'Offener Punkt: einzelne Datenfelder migrationsbedürftig — im nächsten Sprint adressiert.', type: 'risk' },
    ]
    return HttpResponse.json({ milestones, statusUpdates })
  }),

  // ── Time entries ──────────────────────────────────────────────────────────

  // GET /api/v1/projects/:id/time-entries?billed=false — project-level roll-up
  // of tracked hours (joined task→project), used by "Stunden abrechnen".
  http.get(`${API}/api/v1/projects/:id/time-entries`, ({ params, request }) => {
    const projectId = params.id as string
    const url = new URL(request.url)
    const billed = url.searchParams.get('billed') // 'false' | 'true' | null
    const projectTaskIds = new Set(
      tasks.filter((t) => t.project_id === projectId).map((t) => t.id),
    )
    let entries = timeEntries.filter((e) => projectTaskIds.has(e.task_id))
    if (billed === 'false') entries = entries.filter((e) => !e.billed)
    else if (billed === 'true') entries = entries.filter((e) => e.billed)
    const result = entries.map((e) => {
      const task = findTask(e.task_id as string)
      return {
        id: e.id,
        date: e.started_at,
        task: task?.title ?? '—',
        person: e.user_name,
        hours: Math.round(((e.duration_seconds as number) / 3600) * 10) / 10,
        description: e.description,
      }
    })
    return HttpResponse.json({ entries: result })
  }),

  // GET /api/v1/tasks/:id/time-entries
  http.get(`${API}/api/v1/tasks/:id/time-entries`, ({ params, request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 20)
    const taskId = params.id as string
    const all = timeEntries.filter((e) => e.task_id === taskId)
    const start = (page - 1) * pageSize
    return HttpResponse.json({ entries: all.slice(start, start + pageSize), total: all.length })
  }),

  // POST /api/v1/tasks/:id/time-entries — manual entry
  http.post(`${API}/api/v1/tasks/:id/time-entries`, async ({ params, request }) => {
    const body = (await request.json()) as AnyRecord
    const startedAt = (body.started_at as string) ?? minutesAgo(60)
    const duration = (body.duration_seconds as number) ?? 3_600
    const entry: AnyRecord = {
      id: newId('te'),
      task_id: params.id,
      user_id: IDS.users.stefan,
      user_name: 'Stefan Vogel',
      started_at: startedAt,
      ended_at: new Date(new Date(startedAt).getTime() + duration * 1000).toISOString(),
      duration_seconds: duration,
      description: (body.description as string) ?? '',
      is_manual: true,
      created_at: nowIso(),
      updated_at: nowIso(),
    }
    timeEntries.push(entry)
    return HttpResponse.json({ entry }, { status: 201 })
  }),

  // GET /api/v1/tasks/:id/time-summary
  http.get(`${API}/api/v1/tasks/:id/time-summary`, ({ params }) => {
    const taskId = params.id as string
    const all = timeEntries.filter((e) => e.task_id === taskId)
    const total = all.reduce((sum, e) => sum + ((e.duration_seconds as number) ?? 0), 0)
    return HttpResponse.json({
      summary: { task_id: taskId, total_duration_seconds: total, entry_count: all.length },
    })
  }),

  // PUT /api/v1/time-entries/:id
  http.put(`${API}/api/v1/time-entries/:id`, async ({ params, request }) => {
    const entry = timeEntries.find((e) => e.id === params.id)
    if (!entry) return HttpResponse.json({ error: 'Entry not found' }, { status: 404 })
    const body = (await request.json()) as AnyRecord
    Object.assign(entry, body, { updated_at: nowIso() })
    return HttpResponse.json({ entry })
  }),

  // DELETE /api/v1/time-entries/:id
  http.delete(`${API}/api/v1/time-entries/:id`, ({ params }) => {
    const idx = timeEntries.findIndex((e) => e.id === params.id)
    if (idx !== -1) timeEntries.splice(idx, 1)
    return new HttpResponse(null, { status: 200 })
  }),
]
