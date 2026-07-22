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
import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { CustomizationDraftPayload } from '@/api/customization-types'
import { applyDraftToTenant, resolveLabelOverrides } from '@/mocks/data/customization'
import { applyLabelOverlay } from '@/i18n/useLabelOverlay'
import { i18n } from '@/i18n/i18n'

const CHANNEL_NAME = 'cosmi:customization'

interface DeployMessage {
  type: 'deployed'
  payload: CustomizationDraftPayload
  locale: string
}

let channel: BroadcastChannel | null = null
function getChannel(): BroadcastChannel | null {
  if (typeof BroadcastChannel === 'undefined') return null
  if (!channel) channel = new BroadcastChannel(CHANNEL_NAME)
  return channel
}

/** Editor window → tell other windows a deploy landed (with the payload). */
export function publishCustomizationDeploy(payload: CustomizationDraftPayload): void {
  const msg: DeployMessage = { type: 'deployed', payload, locale: i18n.language }
  getChannel()?.postMessage(msg)
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
    const onMessage = (e: MessageEvent<DeployMessage>): void => {
      if (e.data?.type !== 'deployed') return
      applyDraftToTenant(e.data.payload)
      // Live label refresh (sidebar, headings) — direct i18n overlay, no query.
      applyLabelOverlay(e.data.locale, resolveLabelOverrides(e.data.locale))
      // Value-set-driven UI refreshes on refetch.
      void queryClient.invalidateQueries({
        predicate: (q) => {
          const key = JSON.stringify(q.queryKey).toLowerCase()
          return key.includes('customization') || key.includes('label') || key.includes('valueset')
        },
      })
    }
    ch.addEventListener('message', onMessage)
    return () => ch.removeEventListener('message', onMessage)
  }, [queryClient])
}
