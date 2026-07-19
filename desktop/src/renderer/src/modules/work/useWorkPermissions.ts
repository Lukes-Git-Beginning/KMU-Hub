/**
 * work-module capability checks (RBAC R-3) — one place that translates the
 * gating convention onto tasks and projects so every view (list, kanban,
 * detail, my-tasks) answers "may I act on THIS object?" identically.
 *
 * Ownership: a task counts as "own" for assignee, creator or reporter; a
 * project for its owner. Completing a task is allowed via `work:task:edit`
 * OR `work:task:be_assigned` on one's own assigned task (Asana commenter
 * pattern: assigned people may tick off their work without edit rights).
 */
import { useHasCapability, useScopedCapability } from '@/hooks/useCapability'

export interface TaskOwnership {
  assignee_id?: string | null
  created_by?: string | null
  reporter_id?: string | null
}

export interface TaskCan {
  edit: boolean
  delete: boolean
  /** Tick off / reopen (edit OR own assigned via be_assigned). */
  complete: boolean
  comment: boolean
  /** "Mir zuweisen" quick action. */
  assignSelf: boolean
}

export function useTaskCan(task: TaskOwnership | null | undefined): TaskCan {
  const owners = [task?.assignee_id, task?.created_by, task?.reporter_id]
  const edit = useScopedCapability('work:task:edit', ...owners)
  const del = useScopedCapability('work:task:delete', ...owners)
  const beAssigned = useScopedCapability('work:task:be_assigned', task?.assignee_id)
  const comment = useScopedCapability('work:task:comment', ...owners)
  const assignSelf = useHasCapability('work:task:be_assigned')
  return { edit, delete: del, complete: edit || beAssigned, comment, assignSelf }
}

export interface ProjectCan {
  edit: boolean
  delete: boolean
  manageMembers: boolean
}

/** Accepts any project shape — generated API types may lack owner_id, in which
 *  case an `own`-scoped grant denies (safe default until the field lands). */
export function useProjectCan(
  project: { owner_id?: string | null; name?: string } | null | undefined,
): ProjectCan {
  const edit = useScopedCapability('work:project:edit', project?.owner_id)
  const del = useScopedCapability('work:project:delete', project?.owner_id)
  const manageMembers = useHasCapability('work:project:manage_members')
  return { edit, delete: del, manageMembers }
}
