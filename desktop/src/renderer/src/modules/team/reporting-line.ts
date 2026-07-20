/**
 * Reporting-line resolver (RBAC R-4) — pure helpers that turn the employee
 * list (`managerUserId` edges) into the two HR visibility shapes:
 *
 * 1. Drawer scope `team` = self + the ENTIRE line below the viewer (all
 *    levels, never up) — Darien decision R4-BRIEFING §0.3.
 * 2. Directory neighbourhood = 2 levels up + 2 levels down + self; the full
 *    company list needs `team:directory:full` — R4-BRIEFING §0.2.
 *
 * Only the team/HR module resolves `team` this way; every other module keeps
 * the documented team≈all behaviour of the generic useScopedCapability.
 */

export interface ReportingEdge {
  userId: string | null
  managerUserId?: string | null
}

/** userId → managerUserId (only rows where both sides are present). */
export function buildManagerMap(employees: ReportingEdge[]): Map<string, string> {
  const map = new Map<string, string>()
  for (const e of employees) {
    if (e.userId && e.managerUserId) map.set(e.userId, e.managerUserId)
  }
  return map
}

/** All userIds whose manager chain reaches `viewerUserId` (any depth, cycle-safe). */
export function getDescendantUserIds(
  managerMap: Map<string, string>,
  viewerUserId: string,
): Set<string> {
  const children = new Map<string, string[]>()
  for (const [child, manager] of managerMap) {
    const list = children.get(manager)
    if (list) list.push(child)
    else children.set(manager, [child])
  }
  const result = new Set<string>()
  const queue = [viewerUserId]
  while (queue.length > 0) {
    const current = queue.pop() as string
    for (const child of children.get(current) ?? []) {
      if (!result.has(child)) {
        result.add(child)
        queue.push(child)
      }
    }
  }
  return result
}

/**
 * The 2-up/2-down neighbourhood around the viewer (plus the viewer): direct
 * manager + their manager, everyone at most two chain steps below, and the
 * viewer's siblings (same direct manager) — without siblings a member with no
 * reports would only ever see their two bosses, not their own team.
 */
export function getVisibleDirectoryUserIds(
  managerMap: Map<string, string>,
  viewerUserId: string,
): Set<string> {
  const visible = new Set<string>([viewerUserId])

  // 2 levels up (cycle-safe by depth bound).
  let up: string | undefined = managerMap.get(viewerUserId)
  for (let depth = 0; depth < 2 && up; depth++) {
    visible.add(up)
    up = managerMap.get(up)
  }

  const children = new Map<string, string[]>()
  for (const [child, manager] of managerMap) {
    const list = children.get(manager)
    if (list) list.push(child)
    else children.set(manager, [child])
  }

  // 2 levels down: children + grandchildren.
  for (const child of children.get(viewerUserId) ?? []) {
    visible.add(child)
    for (const grandchild of children.get(child) ?? []) visible.add(grandchild)
  }

  // Siblings: everyone reporting to the same direct manager.
  const directManager = managerMap.get(viewerUserId)
  if (directManager) {
    for (const sibling of children.get(directManager) ?? []) visible.add(sibling)
  }
  return visible
}
