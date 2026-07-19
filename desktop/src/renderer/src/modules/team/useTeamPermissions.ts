/**
 * team-module capability checks (RBAC R-3) — single source of truth for all
 * team-module gating so list, panel, dialogs, and payroll answer identically.
 *
 * Ownership: `team:employee:edit` and `team:employee:read` are scopeable.
 * `scope=own` passes only when the employee's userId matches the signed-in
 * account. `scope=team` is treated like `all` — no team model yet (documented
 * gap, R3-BRIEFING §1.4 / team≈all).
 */
import { useHasCapability, useScopedCapability } from '@/hooks/useCapability'

/** Checks for the member list and per-row actions. */
export function useTeamListCan() {
  const canCreate = useHasCapability('team:employee:create')
  const canDeactivate = useHasCapability('team:employee:deactivate')
  return { canCreate, canDeactivate }
}

/**
 * Edit-gate for a specific employee profile.
 * @param employeeUserId The employee's auth-account userId (NOT the HR employee.id).
 *                       Pass null/undefined when unknown (denies while loading).
 */
export function useEmployeeEditCan(employeeUserId: string | null | undefined): boolean {
  // scope=own → only passes if employeeUserId === signed-in user id.
  // scope=team/all → passes unconditionally (hook handles it).
  // NOTE: team≈all is a documented gap — no team model exists yet (R3-BRIEFING §1.4).
  return useScopedCapability('team:employee:edit', employeeUserId)
}

/** Approval actions for leave requests. */
export function useTeamAbsenceCan() {
  const canApprove = useHasCapability('team:absence:approve')
  return { canApprove }
}

/** Payroll run actions (distinct from payroll view which is tab-gated). */
export function useTeamPayrollCan() {
  const canView = useHasCapability('team:payroll:view')
  const canRun = useHasCapability('team:payroll:run')
  return { canView, canRun }
}

/** Personnel documents (personalakte tab). */
export function useTeamDocumentsCan() {
  const canView = useHasCapability('team:documents:view')
  const canEdit = useHasCapability('team:documents:edit')
  return { canView, canEdit }
}

/** Salary / payroll master data. */
export function useTeamSalaryCan() {
  const canView = useHasCapability('team:salary:view')
  const canEdit = useHasCapability('team:salary:edit')
  return { canView, canEdit }
}
