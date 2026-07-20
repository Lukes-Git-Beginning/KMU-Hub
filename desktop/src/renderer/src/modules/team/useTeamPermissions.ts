/**
 * team-module capability checks (RBAC R-3/R-4) — single source of truth for
 * all team-module gating so list, panel, dialogs, and payroll answer
 * identically.
 *
 * R-4: the HR data drawers resolve scope `team` as the REAL reporting line
 * (self + entire chain below via managerUserId, never up) through
 * useHrScopedCapability. Everywhere else `team` keeps the documented team≈all
 * behaviour of the generic useScopedCapability (R3-BRIEFING §1.4).
 */
import { useMemo } from 'react'
import { useAuthStore } from '@/stores/auth'
import { useCapability, useHasCapability, useScopedCapability } from '@/hooks/useCapability'
import { useEmployees } from '@/api/hooks/hr-hooks'
import {
  buildManagerMap,
  getDescendantUserIds,
  getVisibleDirectoryUserIds,
} from './reporting-line'

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

// ── R-4: reporting-line scope ───────────────────────────────────────────────

/**
 * Reporting-line shapes for the signed-in account, derived from the employee
 * list (managerUserId chain). Cached with the hr/employees query; `ready`
 * stays false until the list is loaded (pessimistic while loading).
 */
export function useHrScope(): {
  ready: boolean
  /** Drawer scope `team`: self or anywhere in the line BELOW the viewer. */
  isInTeamScope: (targetUserId: string | null | undefined) => boolean
  /** Directory neighbourhood (2 up + 2 down + self) — R4-BRIEFING §0.2. */
  isInDirectoryNeighbourhood: (targetUserId: string | null | undefined) => boolean
} {
  const userId = useAuthStore((s) => s.user?.id ?? null)
  const { data } = useEmployees()
  return useMemo(() => {
    const employees = data?.employees ?? []
    if (userId === null || employees.length === 0) {
      return {
        ready: false,
        isInTeamScope: () => false,
        isInDirectoryNeighbourhood: () => false,
      }
    }
    const managerMap = buildManagerMap(employees)
    const descendants = getDescendantUserIds(managerMap, userId)
    const neighbourhood = getVisibleDirectoryUserIds(managerMap, userId)
    return {
      ready: true,
      isInTeamScope: (target: string | null | undefined) =>
        target != null && (target === userId || descendants.has(target)),
      isInDirectoryNeighbourhood: (target: string | null | undefined) =>
        target != null && neighbourhood.has(target),
    }
  }, [data, userId])
}

/**
 * Scope-aware drawer check (R-4): may the account see/edit THIS employee's
 * drawer? `own` = self only · `team` = self + entire reporting line below ·
 * `all` = everyone. Denies while the employee list (chain) is still loading.
 */
export function useHrScopedCapability(
  key: string,
  targetUserId: string | null | undefined,
): boolean {
  const { allowed, scope } = useCapability(key)
  const userId = useAuthStore((s) => s.user?.id ?? null)
  const { ready, isInTeamScope } = useHrScope()
  if (!allowed || targetUserId == null || userId === null) return false
  if (scope === 'all') return true
  if (scope === 'own') return targetUserId === userId
  return ready && isInTeamScope(targetUserId)
}

/**
 * Directory visibility (R-4 §0.2): without `team:directory:full` the member
 * list/org chart shows only the 2-up/2-down neighbourhood. Returns a filter
 * predicate; `restricted` is true when the neighbourhood filter applies.
 */
export function useDirectoryVisibility(): {
  restricted: boolean
  isVisible: (targetUserId: string | null | undefined) => boolean
} {
  const full = useHasCapability('team:directory:full')
  const { ready, isInDirectoryNeighbourhood } = useHrScope()
  if (full) return { restricted: false, isVisible: () => true }
  return {
    restricted: true,
    isVisible: (target) => ready && isInDirectoryNeighbourhood(target),
  }
}
