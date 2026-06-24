/**
 * TanStack Query hooks for Work Labels.
 *
 * Replaces the mock-first stores/workSettings + stores/taskLabels
 * with real backend calls.
 *
 * Endpoints:
 *   GET    /api/v1/work/labels           → list tenant label taxonomy
 *   POST   /api/v1/work/labels           → create label
 *   PUT    /api/v1/work/labels/{id}      → update label
 *   DELETE /api/v1/work/labels/{id}      → delete label
 *   PUT    /api/v1/tasks/{id}/labels     → set task label assignments (replace-all)
 *
 * RBAC guards (seeded in migration 000147):
 *   read   → work_labels:read
 *   write  → work_labels:write
 *   delete → work_labels:delete
 *
 * Wire-shape note:
 *   ListLabels  → { labels: WorkLabel[] }   (response.JSON, not protojson)
 *   CreateLabel → { label: WorkLabel }
 *   UpdateLabel → { label: WorkLabel }
 *   DeleteLabel → 204 (returns {} from authenticatedRequest)
 *   SetTaskLabels → 204
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listWorkLabels,
  createWorkLabel,
  updateWorkLabel,
  deleteWorkLabel,
  setTaskLabels,
} from '../work-labels-client'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const workLabelKeys = {
  all: ['work', 'labels'] as const,
  list: () => ['work', 'labels', 'list'] as const,
}

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

/**
 * Fetches the tenant-wide label taxonomy.
 *
 * Returns data as { labels: WorkLabel[] } — callers access `data?.labels ?? []`.
 */
export function useWorkLabels() {
  return useQuery({
    queryKey: workLabelKeys.list(),
    queryFn: listWorkLabels,
  })
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

export function useCreateWorkLabel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name: string; color: string }) => createWorkLabel(body),
    onSuccess: () => qc.invalidateQueries({ queryKey: workLabelKeys.all }),
  })
}

export function useUpdateWorkLabel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: { id: string; name: string; color: string }) =>
      updateWorkLabel(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: workLabelKeys.all }),
  })
}

export function useDeleteWorkLabel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteWorkLabel(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: workLabelKeys.all }),
  })
}

/**
 * Atomically replaces all label assignments for a task.
 *
 * Does NOT invalidate workLabelKeys — the taxonomy itself does not change,
 * only the per-task assignment. Callers that need to reflect the new
 * assignments should invalidate the relevant task query themselves.
 *
 * Usage:
 *   const { mutate: setLabels } = useSetTaskLabels()
 *   setLabels({ taskId: 'uuid', labelIds: ['uuid-1', 'uuid-2'] })
 */
export function useSetTaskLabels() {
  return useMutation({
    mutationFn: ({ taskId, labelIds }: { taskId: string; labelIds: string[] }) =>
      setTaskLabels(taskId, labelIds),
  })
}
