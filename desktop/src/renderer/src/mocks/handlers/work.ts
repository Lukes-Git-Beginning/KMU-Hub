import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import {
  mockProjects,
  mockProjectStatuses,
  mockTasks,
  mockTaskComments,
  mockEntityTasks,
  getProjectById,
  getTaskById,
  getTasksByProject,
  getMyTasks,
} from '../data/projects'

const API = API_BASE_URL

export const workHandlers = [
  // ── Projects ────────────────────────────────────────────────────────────

  // GET /api/v1/projects — list (with optional status, search, templates_only filter)
  http.get(`${API}/api/v1/projects`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    const search = url.searchParams.get('search') ?? ''
    const templatesOnly = url.searchParams.get('templates_only') === 'true'
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 50)

    let filtered = mockProjects.projects

    if (status) {
      filtered = filtered.filter((p) => p.status === status)
    }
    if (search) {
      const q = search.toLowerCase()
      filtered = filtered.filter(
        (p) =>
          p.name.toLowerCase().includes(q) ||
          p.description.toLowerCase().includes(q) ||
          p.project_key.toLowerCase().includes(q),
      )
    }
    if (templatesOnly) {
      filtered = filtered.filter((p) => p.is_template === true)
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

  // GET /api/v1/projects/:id — detail
  http.get(`${API}/api/v1/projects/:id`, ({ params }) => {
    const project = getProjectById(params.id as string)
    if (!project) {
      return HttpResponse.json({ error: 'Project not found' }, { status: 404 })
    }
    return HttpResponse.json(project)
  }),

  // POST /api/v1/projects — create
  http.post(`${API}/api/v1/projects`, async ({ request }) => {
    const body = await request.json() as Record<string, unknown>
    const newProject = {
      id: `prj-${Date.now()}`,
      ...body,
      progress: 0,
      member_count: 1,
      task_count: 0,
      completed_task_count: 0,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json(newProject, { status: 201 })
  }),

  // PUT /api/v1/projects/:id — update
  http.put(`${API}/api/v1/projects/:id`, async ({ params, request }) => {
    const project = getProjectById(params.id as string)
    if (!project) {
      return HttpResponse.json({ error: 'Project not found' }, { status: 404 })
    }
    const body = await request.json() as Record<string, unknown>
    const updated = { ...project, ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json(updated)
  }),

  // GET /api/v1/projects/:id/statuses — list statuses for project
  http.get(`${API}/api/v1/projects/:id/statuses`, ({ params }) => {
    const statuses = mockProjectStatuses[params.id as string]
    if (!statuses) {
      return HttpResponse.json({ error: 'Project not found' }, { status: 404 })
    }
    return HttpResponse.json({ statuses })
  }),

  // POST /api/v1/projects/:id/statuses — add status
  http.post(`${API}/api/v1/projects/:id/statuses`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const newStatus = {
      id: `${params.id}-st-${Date.now()}`,
      name: body.name || 'Neue Spalte',
      color: body.color || '#64748b',
      sort_order: (mockProjectStatuses[params.id as string]?.length ?? 0),
    }
    return HttpResponse.json(newStatus, { status: 201 })
  }),

  // ── Tasks ───────────────────────────────────────────────────────────────

  // GET /api/v1/tasks — list (with filters: project_id, assignee_id, status_id)
  http.get(`${API}/api/v1/tasks`, ({ request }) => {
    const url = new URL(request.url)
    const projectId = url.searchParams.get('project_id')
    const assigneeId = url.searchParams.get('assignee_id')
    const statusId = url.searchParams.get('status_id')

    let filtered = [...mockTasks.tasks]

    if (projectId) {
      return HttpResponse.json(getTasksByProject(projectId))
    }
    if (assigneeId) {
      return HttpResponse.json(getMyTasks(assigneeId))
    }
    if (statusId) {
      filtered = filtered.filter((t) => t.status_id === statusId)
      return HttpResponse.json({ tasks: filtered, total: filtered.length, page: 1, page_size: 50 })
    }

    return HttpResponse.json(mockTasks)
  }),

  // GET /api/v1/tasks/:id — detail
  http.get(`${API}/api/v1/tasks/:id`, ({ params }) => {
    const task = getTaskById(params.id as string)
    if (!task) {
      return HttpResponse.json({ error: 'Task not found' }, { status: 404 })
    }
    return HttpResponse.json(task)
  }),

  // POST /api/v1/tasks — create
  http.post(`${API}/api/v1/tasks`, async ({ request }) => {
    const body = await request.json() as Record<string, unknown>
    const newTask = {
      id: `tsk-${Date.now()}`,
      ...body,
      subtask_count: 0,
      completed_subtask_count: 0,
      comment_count: 0,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json(newTask, { status: 201 })
  }),

  // PUT /api/v1/tasks/:id — update
  http.put(`${API}/api/v1/tasks/:id`, async ({ params, request }) => {
    const task = getTaskById(params.id as string)
    if (!task) {
      return HttpResponse.json({ error: 'Task not found' }, { status: 404 })
    }
    const body = await request.json() as Record<string, unknown>
    const updated = { ...task, ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json(updated)
  }),

  // DELETE /api/v1/tasks/:id — delete
  http.delete(`${API}/api/v1/tasks/:id`, ({ params }) => {
    const task = getTaskById(params.id as string)
    if (!task) {
      return HttpResponse.json({ error: 'Task not found' }, { status: 404 })
    }
    return new HttpResponse(null, { status: 204 })
  }),

  // ── Search ──────────────────────────────────────────────────────────────

  // GET /api/v1/work/search — task search
  http.get(`${API}/api/v1/work/search`, ({ request }) => {
    const url = new URL(request.url)
    const query = (url.searchParams.get('q') || '').toLowerCase()

    if (!query) {
      return HttpResponse.json({ tasks: [], total: 0, page: 1, page_size: 50 })
    }

    const results = mockTasks.tasks.filter(
      (t) =>
        t.title.toLowerCase().includes(query) ||
        t.description.toLowerCase().includes(query) ||
        t.tags.some((tag) => tag.toLowerCase().includes(query)),
    )

    return HttpResponse.json({ tasks: results, total: results.length, page: 1, page_size: 50 })
  }),

  // ── Entity Tasks (cross-module) ─────────────────────────────────────────

  // GET /api/v1/entity-tasks — cross-module task links
  http.get(`${API}/api/v1/entity-tasks`, ({ request }) => {
    const url = new URL(request.url)
    const entityType = url.searchParams.get('entity_type')
    const entityId = url.searchParams.get('entity_id')

    if (entityType && entityId) {
      const filtered = mockEntityTasks.tasks.filter(
        (et) => et.entity_type === entityType && et.entity_id === entityId,
      )
      return HttpResponse.json({ tasks: filtered, total: filtered.length })
    }

    return HttpResponse.json(mockEntityTasks)
  }),

  // ── Timer ───────────────────────────────────────────────────────────────

  // GET /api/v1/timer/active — active timer (none running)
  http.get(`${API}/api/v1/timer/active`, () => {
    return HttpResponse.json({ timer: null })
  }),

  // POST /api/v1/timer/stop — stop timer
  http.post(`${API}/api/v1/timer/stop`, () => {
    return HttpResponse.json({ success: true, timer: null })
  }),

  // ── Task Comments ───────────────────────────────────────────────────────

  // GET /api/v1/tasks/:id/comments — task comments
  http.get(`${API}/api/v1/tasks/:id/comments`, ({ params }) => {
    const taskId = params.id as string
    const comments = mockTaskComments.comments.filter((c) => c.task_id === taskId)
    return HttpResponse.json({ comments, total: comments.length })
  }),

  // POST /api/v1/tasks/:id/comments — add comment
  http.post(`${API}/api/v1/tasks/:id/comments`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>
    const newComment = {
      id: `tcm-${Date.now()}`,
      task_id: params.id,
      author_id: body.author_id || 'usr-001',
      author_name: body.author_name || 'Stefan Müller',
      content: body.content || '',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json(newComment, { status: 201 })
  }),

  // ── Task Files ──────────────────────────────────────────────────────────

  // GET /api/v1/tasks/:id/files — task files (empty)
  http.get(`${API}/api/v1/tasks/:id/files`, () => {
    return HttpResponse.json({ files: [], total: 0 })
  }),
]
