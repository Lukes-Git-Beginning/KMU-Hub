import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ConversationStatus } from '@/types/communication'

/**
 * Mock-first overlay for inbox message status (offen/wartend/gelöst/geschlossen).
 *
 * The backend InboxMessage has no `status` field yet (see backend-gaps.md),
 * so we keep a client-side overlay keyed by message id. Messages without an
 * entry default to 'open'. This is adapter-ready: once the backend adds a
 * status field + RPC, replace `getStatus`/`setStatus` with the API hooks and
 * drop the persisted map.
 */

interface InboxStatusState {
  /** Per-message status override. Absent → 'open'. */
  statuses: Record<string, ConversationStatus>
  getStatus: (msgId: string) => ConversationStatus
  setStatus: (msgId: string, status: ConversationStatus) => void
}

export const useInboxStatus = create<InboxStatusState>()(
  persist(
    (set, get) => ({
      statuses: {},
      getStatus: (msgId) => get().statuses[msgId] ?? 'open',
      setStatus: (msgId, status) =>
        set((state) => ({ statuses: { ...state.statuses, [msgId]: status } })),
    }),
    { name: 'cosmi-inbox-status' },
  ),
)
