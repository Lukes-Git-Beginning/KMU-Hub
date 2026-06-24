/**
 * Lightweight fetch wrapper for Work Labels API endpoints.
 *
 * Backend: GET/POST/PUT/DELETE /api/v1/work/labels
 *          PUT /api/v1/tasks/{id}/labels  (set task label assignments)
 *
 * Wire-shape: response.JSON (encoding/json) — no protojson here.
 *   ListLabels  → { labels: LabelProto[] }
 *   CreateLabel → { label: LabelProto }
 *   UpdateLabel → { label: LabelProto }
 *   DeleteLabel → 204 No Content
 *   SetTaskLabels → 204 No Content
 *
 * LabelProto { id: string, name: string, color: string }
 *
 * RBAC (seeded in migration 000147):
 *   read   → work_labels:read
 *   write  → work_labels:write
 *   delete → work_labels:delete
 */
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface WorkLabel {
  id: string
  name: string
  color: string
}

// ---------------------------------------------------------------------------
// API functions
// ---------------------------------------------------------------------------

const BASE = '/api/v1/work/labels'
const TASKS_BASE = '/api/v1/tasks'

export function listWorkLabels() {
  return authenticatedRequest<{ labels: WorkLabel[] }>({
    method: 'GET',
    path: BASE,
  })
}

export function createWorkLabel(body: { name: string; color: string }) {
  return authenticatedRequest<{ label: WorkLabel }>({
    method: 'POST',
    path: BASE,
    body,
  })
}

export function updateWorkLabel(id: string, body: { name: string; color: string }) {
  return authenticatedRequest<{ label: WorkLabel }>({
    method: 'PUT',
    path: `${BASE}/${id}`,
    body,
  })
}

export function deleteWorkLabel(id: string) {
  return authenticatedRequest<Record<string, never>>({
    method: 'DELETE',
    path: `${BASE}/${id}`,
  })
}

/**
 * Replaces all label assignments for a task atomically.
 * PUT /api/v1/tasks/{taskId}/labels → 204
 */
export function setTaskLabels(taskId: string, labelIds: string[]) {
  return authenticatedRequest<Record<string, never>>({
    method: 'PUT',
    path: `${TASKS_BASE}/${taskId}/labels`,
    body: { label_ids: labelIds },
  })
}
