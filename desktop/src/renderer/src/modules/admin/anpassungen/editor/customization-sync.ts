/**
 * Cross-window customization sync (Modul-Editor v1, E-2b).
 *
 * The editor runs in its own OS window with its own JS heap, so a deploy there
 * does not automatically reach the main Cosmi window. We bridge it with a
 * BroadcastChannel: on deploy the editor window posts the applied payload, and
 * the main window merges it into its own (mock) tenant layer and re-applies the
 * live label overlay — so a rename lands in the sidebar instantly.
 *
 * With Luke's real backend this becomes a plain refetch (the DB is the shared
 * source), but the channel is harmless to keep for instant UX.
 */
import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { CustomizationDraftPayload, CustomizationDraft } from '@/api/customization-types'
import { applyDraftToTenant, resolveLabelOverrides } from '@/mocks/data/customization'
import { applyDraftCustomFields } from '@/mocks/data/custom-fields'
import { ingestRemoteDraft, type DeploySnapshot } from '@/mocks/data/customization-drafts'
import { applyLabelOverlay } from '@/i18n/useLabelOverlay'
import { i18n } from '@/i18n/i18n'

const CHANNEL_NAME = 'cosmi:customization'

interface DeployMessage {
  type: 'deployed'
  payload: CustomizationDraftPayload
  locale: string
  /** The draft record + rollback snapshot, mirrored so the main-window hub can
      list and roll back a deploy done in the editor window (mock; Luke's DB
      replaces this with a shared table + refetch). */
  draft?: CustomizationDraft
  snapshot?: DeploySnapshot
}

/** A draft saved/scheduled in the editor window, mirrored for the hub list. */
interface DraftMirrorMessage {
  type: 'draft-mirror'
  draft: CustomizationDraft
}

/** Hub → an ALREADY OPEN editor window: load this draft (see stashDraftForEditor). */
interface ResumeDraftMessage {
  type: 'resume-draft'
  draft: CustomizationDraft
}

type ChannelMessage = DeployMessage | DraftMirrorMessage | ResumeDraftMessage

let channel: BroadcastChannel | null = null
function getChannel(): BroadcastChannel | null {
  if (typeof BroadcastChannel === 'undefined') return null
  if (!channel) channel = new BroadcastChannel(CHANNEL_NAME)
  return channel
}

/** Editor window → tell other windows a deploy landed (payload + draft mirror). */
export function publishCustomizationDeploy(
  payload: CustomizationDraftPayload,
  draft?: CustomizationDraft,
  snapshot?: DeploySnapshot,
): void {
  const msg: DeployMessage = { type: 'deployed', payload, locale: i18n.language, draft, snapshot }
  getChannel()?.postMessage(msg)
}

/** Editor window → mirror a saved/scheduled draft (no live apply) to the hub. */
export function publishDraftMirror(draft: CustomizationDraft): void {
  const msg: DraftMirrorMessage = { type: 'draft-mirror', draft }
  getChannel()?.postMessage(msg)
}

// ── Continue a saved draft (Darien 2026-08-05) ────────────────────────────────
//
// The pencil in the rollout list used to open a BLANK editor: the record was
// there, its changes were not. The editor window does not exist yet at that
// moment, so a channel message would have nobody to receive it — the draft is
// therefore handed over through localStorage, which both windows share (same
// origin) and which survives the window launch. Luke's backend turns this into
// "open the editor with ?draft=<id>" and a plain GET.

const RESUME_KEY = 'cosmi:customization:resume-draft'

/**
 * Hub → editor: hand over the draft to continue. Two paths on purpose, because
 * only one editor window exists per module: a window that is about to OPEN reads
 * the storage handover, a window that is ALREADY open (and just gets focused)
 * would never read it, so it also gets a message.
 */
export function stashDraftForEditor(draft: CustomizationDraft): void {
  try {
    localStorage.setItem(RESUME_KEY, JSON.stringify(draft))
  } catch {
    // Private mode / quota: the editor simply opens empty, as it did before.
  }
  getChannel()?.postMessage({ type: 'resume-draft', draft } satisfies ResumeDraftMessage)
}

/**
 * Editor window: an open editor is asked to continue a draft. Ignores drafts for
 * other modules; the caller decides what to do with unsaved work.
 */
export function useResumeDraftListener(
  moduleKey: string,
  onResume: (draft: CustomizationDraft) => void,
): void {
  const latest = useRef(onResume)
  useEffect(() => {
    latest.current = onResume
  })
  useEffect(() => {
    const ch = getChannel()
    if (!ch) return
    const onMessage = (e: MessageEvent<ChannelMessage>): void => {
      if (e.data?.type !== 'resume-draft' || e.data.draft.moduleKey !== moduleKey) return
      latest.current(e.data.draft)
    }
    ch.addEventListener('message', onMessage)
    return () => ch.removeEventListener('message', onMessage)
  }, [moduleKey])
}

/**
 * Editor → read and CONSUME the handover (one-shot, so a later plain open starts
 * clean). Returns null when nothing was handed over for this module.
 */
export function takeStashedDraft(moduleKey: string): CustomizationDraft | null {
  try {
    const raw = localStorage.getItem(RESUME_KEY)
    if (!raw) return null
    localStorage.removeItem(RESUME_KEY)
    const draft = JSON.parse(raw) as CustomizationDraft
    return draft?.moduleKey === moduleKey ? draft : null
  } catch {
    return null
  }
}

/**
 * Main-window listener: converge the local mock tenant layer with a deploy that
 * happened in another window and refresh the live label overlay + queries.
 * Mount ONCE in the main shell (not in the editor window).
 */
export function useCustomizationSyncListener(): void {
  const queryClient = useQueryClient()
  useEffect(() => {
    const ch = getChannel()
    if (!ch) return
    const refresh = (): void => {
      void queryClient.invalidateQueries({
        predicate: (q) => {
          const key = JSON.stringify(q.queryKey).toLowerCase()
          return key.includes('customization') || key.includes('label') || key.includes('valueset')
        },
      })
    }
    const onMessage = (e: MessageEvent<ChannelMessage>): void => {
      if (e.data?.type === 'draft-mirror') {
        // Saved/scheduled draft — list it in the hub, no live apply.
        ingestRemoteDraft(e.data.draft)
        refresh()
        return
      }
      if (e.data?.type !== 'deployed') return
      applyDraftToTenant(e.data.payload)
      // Custom fields live in their own store (E-3c) — apply the snapshot here too.
      applyDraftCustomFields(e.data.payload.customFields ?? {})
      // Mirror the draft record + rollback snapshot so the hub can list/roll it back.
      if (e.data.draft) ingestRemoteDraft(e.data.draft, e.data.snapshot)
      // Live label refresh (sidebar, headings) — direct i18n overlay, no query.
      applyLabelOverlay(e.data.locale, resolveLabelOverrides(e.data.locale))
      // Value-set-, field- and draft-list-driven UI refreshes on refetch (the
      // 'customization' predicate also matches ['customization','fields'|'drafts',…]).
      refresh()
    }
    ch.addEventListener('message', onMessage)
    return () => ch.removeEventListener('message', onMessage)
  }, [queryClient])
}
