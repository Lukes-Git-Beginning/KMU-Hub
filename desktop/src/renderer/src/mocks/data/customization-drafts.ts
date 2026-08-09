/**
 * Customization drafts store (Modul-Editor v1) — stateful in-memory sink for
 * saved blueprints and scheduled tenant-wide rollouts.
 *
 * Change-Management für Config: a draft holds only the sparse deviations from
 * the live tenant layer (identical to the overlay principle in customization.ts).
 * Lifecycle: draft → scheduled → live → superseded.
 *   - "Jetzt übernehmen"     → commitDraftNow()   (draft→live immediately)
 *   - "Terminiert am Tag X"   → scheduleDraft()    (draft→scheduled; promoted by the
 *                                                   mock scheduler at scheduledAt)
 *   - "Als Entwurf speichern" → saveDraft()        (stays draft, affects no user)
 * Rollback restores the tenant snapshot captured just before promotion.
 *
 * The real backend (Luke's track 🔒, backend-gaps §Customization) persists this
 * in `tenant_customization_drafts` with a promotion cron; here the records live
 * in shared browser storage (see "Persistence" below) so they survive a restart
 * and are visible from both windows.
 */
import type {
  CustomizationDraft,
  CustomizationDraftPayload,
  DeployDraftInput,
  SaveDraftInput,
} from '@/api/customization-types'
import { writeAuditEvent } from './audit-events'
import { applyDraftToTenant, restoreTenant, snapshotTenant, type TenantSnapshot } from './customization'
import {
  applyDraftCustomFields,
  restoreCustomFields,
  snapshotCustomFields,
} from './custom-fields'
import type { CustomFieldDefinition } from './custom-fields'
import { getDemoSessionUserId } from './rbac'

let draftSeq = 0
const drafts: CustomizationDraft[] = []

/**
 * Rollback snapshots keyed by draft id: the state that existed BEFORE this draft
 * went live. Restoring it undoes exactly this deploy. Captures both the tenant
 * overlay (labels + value-sets) AND the custom-field store (E-3c: fields have
 * their own store, outside the overlay).
 */
export interface DeploySnapshot {
  tenant: TenantSnapshot
  fields: CustomFieldDefinition[]
}
const rollbackSnapshots: Record<string, DeploySnapshot> = {}

// ── Persistence (Darien 2026-08-05) ───────────────────────────────────────────
//
// "Wenn ich als Entwurf speichern drücke, dann speichert er keinen Entwurf."
// Two things made that true: the store lived only in the JS heap, so a reload or
// an app restart wiped every draft — and the editor runs in a SEPARATE window
// with its own heap, so the hub only ever learned about a draft through a live
// BroadcastChannel message it had to be listening for at that exact moment.
//
// Both go away by keeping the records in storage the windows share: whoever looks
// at the list reads the current state, no message required, and a draft survives
// a restart the way a saved thing should. Luke's backend replaces this file with
// a table; the channel stays for instant cross-window refresh.

const STORAGE_KEY = 'cosmi:customization:drafts'

interface PersistedDrafts {
  drafts: CustomizationDraft[]
  snapshots: Record<string, DeploySnapshot>
  seq: number
}

function persist(): void {
  try {
    const state: PersistedDrafts = { drafts, snapshots: rollbackSnapshots, seq: draftSeq }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch {
    // No storage (private mode, quota): fall back to the previous in-memory-only
    // behaviour rather than breaking the editor.
  }
}

/** Pull the shared state into this window — cheap, and the list is small. */
function syncFromStorage(): void {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return
    const parsed = JSON.parse(raw) as PersistedDrafts
    drafts.length = 0
    drafts.push(...(parsed.drafts ?? []))
    for (const key of Object.keys(rollbackSnapshots)) delete rollbackSnapshots[key]
    Object.assign(rollbackSnapshots, parsed.snapshots ?? {})
    draftSeq = Math.max(parsed.seq ?? 0, drafts.length)
  } catch {
    // Corrupt entry: keep whatever this window already has.
  }
}

let hydrated = false
/**
 * First read of the session: restore the records AND make the live ones true
 * again. Without the re-apply the list would claim "Live" after a restart while
 * the module had quietly fallen back to stock.
 */
function hydrateOnce(): void {
  if (hydrated) return
  hydrated = true
  syncFromStorage()
  for (const draft of drafts) {
    if (draft.status !== 'live') continue
    applyDraftToTenant(draft.payload)
    applyDraftCustomFields(draft.payload.customFields ?? {})
  }
}

/**
 * Make the deployed customization true for THIS window, at startup (Darien
 * 2026-08-06).
 *
 * Until now hydration only happened when something read the draft list — which
 * only the Anpassungen page does. Open the app and go straight to Helpdesk and
 * the module showed stock value lists: a rollout that had been live for weeks
 * looked like it had never happened, and came back the moment you visited
 * Anpassungen. Every window boots through main.tsx, so this is where it belongs.
 *
 * The backend replaces this: there the tenant layer arrives with the data.
 */
export function hydrateCustomization(): void {
  hydrateOnce()
}

function newId(): string {
  draftSeq += 1
  return `draft-${String(draftSeq).padStart(3, '0')}`
}

function summarize(payload: CustomizationDraftPayload): {
  labelCount: number
  valueSetCount: number
  fieldEntityCount: number
} {
  const labelCount = Object.values(payload.labels).reduce(
    (acc, map) => acc + Object.keys(map).length,
    0,
  )
  return {
    labelCount,
    valueSetCount: Object.keys(payload.valueSets).length,
    fieldEntityCount: Object.keys(payload.customFields ?? {}).length,
  }
}

// ── Reads ──────────────────────────────────────────────────────────────────────

/**
 * All drafts, newest first. Optionally filtered by module.
 *
 * Returns COPIES on purpose: this store mutates records in place (status, dates,
 * announcement), and React Query's structural sharing keeps the previous object
 * when the refetched one is reference-identical — so a scheduled rollout kept
 * rendering as "Entwurf" until the page was left. Copies make each refetch compare
 * by value, which is what the UI expects.
 */
export function listDrafts(moduleKey?: string): CustomizationDraft[] {
  hydrateOnce()
  // Re-read every time: the other window may have saved since the last look.
  syncFromStorage()
  const all = [...drafts]
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
    .map((d) => ({ ...d }))
  return moduleKey ? all.filter((d) => d.moduleKey === moduleKey) : all
}

export function getDraft(id: string): CustomizationDraft | undefined {
  hydrateOnce()
  syncFromStorage()
  return drafts.find((d) => d.id === id)
}

// ── Cross-window mirror (mock only; Luke's shared DB replaces this) ─────────────
//
// The editor runs in its own OS window (own JS heap → own drafts store). So the
// Anpassungen hub in the MAIN window mirrors draft records (and, for deploys, the
// rollback snapshot) via the customization-sync BroadcastChannel — otherwise the
// rollout list would be empty in the main window. With Luke's backend this is a
// plain refetch of a shared table.

/** The rollback snapshot captured when a draft went live (for mirroring). */
export function getDeploySnapshot(id: string): DeploySnapshot | undefined {
  return rollbackSnapshots[id]
}

/**
 * Upsert a draft record (and optional rollback snapshot) that was created in
 * another window. Pure mirror: no tenant mutation, no audit — the originating
 * window already did those; this only makes the record visible/rollback-able here.
 */
export function ingestRemoteDraft(draft: CustomizationDraft, snapshot?: DeploySnapshot): void {
  const idx = drafts.findIndex((d) => d.id === draft.id)
  if (idx >= 0) drafts[idx] = draft
  else drafts.unshift(draft)
  if (snapshot) rollbackSnapshots[draft.id] = snapshot
  if (draft.status === 'live') {
    for (const d of drafts) {
      if (d.id !== draft.id && d.moduleKey === draft.moduleKey && d.status === 'live') d.status = 'superseded'
    }
  }
  persist()
}

// ── Save (blueprint, affects no user) ───────────────────────────────────────────

/** Upsert a draft in status 'draft'. No tenant mutation, no user impact. */
export function saveDraft(input: SaveDraftInput): CustomizationDraft {
  const now = new Date().toISOString()
  const existing = input.id ? drafts.find((d) => d.id === input.id) : undefined

  if (existing) {
    existing.name = input.name
    existing.payload = input.payload
    existing.status = 'draft'
    existing.scheduledAt = undefined
    // undefined = "not touched here": the announcement is edited in the rollout
    // detail, and saving the draft again must not wipe it.
    if (input.announcement !== undefined) existing.announcement = input.announcement || undefined
    existing.updatedAt = now
    persist()
    return existing
  }

  const draft: CustomizationDraft = {
    id: newId(),
    moduleKey: input.moduleKey,
    name: input.name,
    status: 'draft',
    payload: input.payload,
    createdAt: now,
    updatedAt: now,
    createdBy: getDemoSessionUserId(),
    ...(input.announcement ? { announcement: input.announcement } : {}),
  }
  drafts.unshift(draft)
  persist()
  writeAuditEvent({
    action: 'customization.draft_saved',
    target: draft.name,
    targetType: 'customization_draft',
    newValue: { moduleKey: draft.moduleKey, ...summarize(draft.payload) },
  })
  return draft
}

export function deleteDraft(id: string): void {
  const idx = drafts.findIndex((d) => d.id === id)
  if (idx < 0) return
  const [removed] = drafts.splice(idx, 1)
  delete rollbackSnapshots[id]
  persist()
  writeAuditEvent({
    action: 'customization.draft_deleted',
    target: removed.name,
    targetType: 'customization_draft',
    oldValue: { moduleKey: removed.moduleKey, status: removed.status },
  })
}

// ── Promote (draft → live) ───────────────────────────────────────────────────

function promote(draft: CustomizationDraft): void {
  // Snapshot tenant overlay + field store BEFORE applying, so we can roll back.
  rollbackSnapshots[draft.id] = { tenant: snapshotTenant(), fields: snapshotCustomFields() }

  // Any previously-live draft for the same module becomes superseded.
  for (const d of drafts) {
    if (d.id !== draft.id && d.moduleKey === draft.moduleKey && d.status === 'live') {
      d.status = 'superseded'
      d.updatedAt = new Date().toISOString()
    }
  }

  const applied = applyDraftToTenant(draft.payload)
  const fieldCount = applyDraftCustomFields(draft.payload.customFields ?? {})
  draft.status = 'live'
  draft.scheduledAt = undefined
  draft.updatedAt = new Date().toISOString()

  writeAuditEvent({
    action: 'customization.deploy_live',
    target: draft.name,
    targetType: 'customization_draft',
    newValue: { moduleKey: draft.moduleKey, ...applied, fieldCount },
  })
  persist()
}

/** Deploy immediately: persist the draft (if new) and promote it to the tenant layer. */
export function commitDraftNow(input: DeployDraftInput): CustomizationDraft {
  const draft = saveDraft({
    id: input.id,
    moduleKey: input.moduleKey,
    name: input.name,
    payload: input.payload,
    announcement: input.announcement,
  })
  promote(draft)
  return draft
}

/** Schedule a tenant-wide rollout at `scheduledAt`; promoted by the mock scheduler. */
export function scheduleDraft(input: DeployDraftInput): CustomizationDraft {
  if (!input.scheduledAt) throw new Error('scheduleDraft requires scheduledAt')
  const draft = saveDraft({
    id: input.id,
    moduleKey: input.moduleKey,
    name: input.name,
    payload: input.payload,
    announcement: input.announcement,
  })
  draft.status = 'scheduled'
  draft.scheduledAt = input.scheduledAt
  draft.updatedAt = new Date().toISOString()
  writeAuditEvent({
    action: 'customization.deploy_scheduled',
    target: draft.name,
    targetType: 'customization_draft',
    newValue: { moduleKey: draft.moduleKey, scheduledAt: input.scheduledAt, ...summarize(draft.payload) },
  })
  persist()
  return draft
}

/** Single entry point for the commit dialog's three modes. */
export function deployDraft(input: DeployDraftInput): CustomizationDraft {
  switch (input.mode) {
    case 'now':
      return commitDraftNow(input)
    case 'scheduled':
      return scheduleDraft(input)
    case 'draft':
      return saveDraft({
        id: input.id,
        moduleKey: input.moduleKey,
        name: input.name,
        payload: input.payload,
        announcement: input.announcement,
      })
  }
}

// ── Rollout detail actions (Darien 2026-08-05) ─────────────────────────────────
// Everything the rollout list can do to an existing record without reopening the
// editor: move a scheduled date, take it off the calendar, edit the announcement.

/** Rewrite the announcement of an existing draft/rollout. Empty string clears it. */
export function setDraftAnnouncement(id: string, announcement: string): CustomizationDraft | undefined {
  const draft = drafts.find((d) => d.id === id)
  if (!draft) return undefined
  draft.announcement = announcement.trim() || undefined
  draft.updatedAt = new Date().toISOString()
  writeAuditEvent({
    action: 'customization.announcement_changed',
    target: draft.name,
    targetType: 'customization_draft',
    newValue: { moduleKey: draft.moduleKey, hasAnnouncement: Boolean(draft.announcement) },
  })
  persist()
  return draft
}

/** Put a saved draft on the calendar, or move an existing appointment. */
export function setDraftSchedule(id: string, scheduledAt: string): CustomizationDraft | undefined {
  const draft = drafts.find((d) => d.id === id)
  if (!draft || draft.status === 'live' || draft.status === 'superseded') return undefined
  draft.status = 'scheduled'
  draft.scheduledAt = scheduledAt
  draft.updatedAt = new Date().toISOString()
  writeAuditEvent({
    action: 'customization.deploy_scheduled',
    target: draft.name,
    targetType: 'customization_draft',
    newValue: { moduleKey: draft.moduleKey, scheduledAt },
  })
  persist()
  return draft
}

/** Take a scheduled rollout off the calendar — it stays a draft, nothing is lost. */
export function clearDraftSchedule(id: string): CustomizationDraft | undefined {
  const draft = drafts.find((d) => d.id === id)
  if (!draft || draft.status !== 'scheduled') return undefined
  draft.status = 'draft'
  draft.scheduledAt = undefined
  draft.updatedAt = new Date().toISOString()
  writeAuditEvent({
    action: 'customization.deploy_unscheduled',
    target: draft.name,
    targetType: 'customization_draft',
    newValue: { moduleKey: draft.moduleKey },
  })
  persist()
  return draft
}

/** Promote an existing draft/scheduled rollout right now (from the rollout list). */
export function promoteDraftById(id: string): CustomizationDraft | undefined {
  const draft = drafts.find((d) => d.id === id)
  if (!draft || draft.status === 'live') return undefined
  promote(draft)
  return draft
}

// ── Scheduler mock ──────────────────────────────────────────────────────────

/**
 * Promote every scheduled draft whose scheduledAt has passed. Stands in for the
 * server-side cron (Luke's track). Call on hub load / editor open. Returns the
 * drafts that went live in this pass.
 */
export function runDueScheduledDeploys(nowIso: string = new Date().toISOString()): CustomizationDraft[] {
  const due = drafts.filter(
    (d) => d.status === 'scheduled' && d.scheduledAt !== undefined && d.scheduledAt <= nowIso,
  )
  for (const draft of due) promote(draft)
  return due
}

// ── Rollback (live → superseded) ──────────────────────────────────────────────

/** Whether a deployed draft can be rolled back (its pre-deploy snapshot exists). */
export function canRollback(id: string): boolean {
  return rollbackSnapshots[id] !== undefined
}

/**
 * Roll a deployed draft back: restore the tenant snapshot captured before its
 * promotion and mark it superseded. Modul-granular in intent; the snapshot is
 * whole-tenant (acceptable for the single-draft-at-a-time demo flow).
 */
export function rollbackDeploy(id: string): void {
  const draft = drafts.find((d) => d.id === id)
  const snap = rollbackSnapshots[id]
  if (!draft || !snap) return

  restoreTenant(snap.tenant)
  restoreCustomFields(snap.fields)
  draft.status = 'superseded'
  draft.updatedAt = new Date().toISOString()
  delete rollbackSnapshots[id]
  persist()

  writeAuditEvent({
    action: 'customization.rolled_back',
    target: draft.name,
    targetType: 'customization_draft',
    oldValue: { moduleKey: draft.moduleKey, status: 'live' },
    newValue: { status: 'superseded' },
  })
}
