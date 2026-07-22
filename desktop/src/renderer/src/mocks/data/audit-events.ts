/**
 * Live audit-event store (RBAC R-5) — stateful in-memory sink for RBAC,
 * user-lifecycle and vendor-access mutations. The 50 static seeds in
 * handlers/security.ts keep sequence 1..50; live events count upward from 51
 * so the merged list stays collision-free. Chain verification runs over the
 * seeds only (documented in R5-BRIEFING §8) — live entries are session-scoped
 * and reset on reload, mirroring what Luke's server-side interceptor will
 * persist for real (backend-gaps §RBAC R-5).
 *
 * Taxonomy (action keys — keep in sync with backend-gaps):
 *   role.assigned | role.revoked | role.definition_created |
 *   role.definition_updated | role.definition_deleted |
 *   permission.override_set | permission.override_removed   (reserved for R-6)
 *   user.invited | user.deactivated | user.reactivated | user.offboarded |
 *   user.view_as | vendor_access.requested | vendor_access.approved |
 *   vendor_access.declined | vendor_access.counter_proposed |
 *   vendor_access.granted | vendor_access.revoked | vendor_access.expired |
 *   vendor_access.completed | setting.changed
 *   customization.label_set | customization.label_removed |
 *   customization.valueset_updated                           (Customization v1.0)
 *   customization.draft_saved | customization.draft_deleted |
 *   customization.deploy_live | customization.deploy_scheduled |
 *   customization.rolled_back                                 (Modul-Editor v1)
 */
import type { AuditEntry } from '@/api/security-types'
import { getDemoSessionUserId } from './rbac'
import { displayUserName } from './shared-ids'

export interface WriteAuditEventInput {
  action: string
  /** Human-readable target (user display name, role name, …). */
  target: string
  targetType: string
  /** Old/new snapshot for the delta panel; omit when not applicable. */
  oldValue?: unknown
  newValue?: unknown
  /** Defaults to the demo session user. */
  actorId?: string
  /** Overrides display-name resolution (e.g. "Luke S. (Zentria)"). */
  actorName?: string
  ip?: string
  extraDetails?: Record<string, unknown>
}

let liveSeq = 50
const liveEvents: AuditEntry[] = []

export function writeAuditEvent(input: WriteAuditEventInput): AuditEntry {
  liveSeq += 1
  const actorId = input.actorId ?? getDemoSessionUserId()
  const entry: AuditEntry = {
    id: `aud-live-${String(liveSeq).padStart(3, '0')}`,
    sequence_num: liveSeq,
    timestamp: new Date().toISOString(),
    user_id: actorId,
    user_name: input.actorName ?? displayUserName(actorId),
    action: input.action,
    target: input.target,
    target_type: input.targetType,
    details: {
      ...(input.oldValue !== undefined ? { old_value: input.oldValue } : {}),
      ...(input.newValue !== undefined ? { new_value: input.newValue } : {}),
      ...(input.extraDetails ?? {}),
    },
    ip_address: input.ip ?? '192.168.1.100',
    user_agent: 'Cosmi Desktop/1.0',
    result: 'success',
  }
  liveEvents.unshift(entry)
  return entry
}

/** Newest first — prepend to the static seeds when serving the merged list. */
export function listLiveAuditEvents(): AuditEntry[] {
  return liveEvents
}
