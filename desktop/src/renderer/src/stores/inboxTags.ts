import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Mock-first overlay for inbox message tags.
 *
 * The backend exposes `tags: string[]` on InboxMessage but has no add/remove
 * tag RPC yet (see backend-gaps.md). We keep a client-side overlay keyed by
 * message id. Once a message's tags are edited the overlay holds the full
 * effective list; until then the message's base `tags` are used.
 *
 * Adapter-ready: replace `effectiveTags`/`addTag`/`removeTag` with the API
 * hooks and drop the persisted map once the backend ships tag mutations.
 */

interface InboxTagsState {
  /** Per-message full tag list override. Absent → use message base tags. */
  overrides: Record<string, string[]>
  /** Resolve the effective tags: override if present, else the base tags. */
  effectiveTags: (msgId: string, baseTags: string[]) => string[]
  addTag: (msgId: string, baseTags: string[], tag: string) => void
  removeTag: (msgId: string, baseTags: string[], tag: string) => void
}

export const useInboxTags = create<InboxTagsState>()(
  persist(
    (set, get) => ({
      overrides: {},
      effectiveTags: (msgId, baseTags) => get().overrides[msgId] ?? baseTags,
      addTag: (msgId, baseTags, tag) => {
        const clean = tag.trim()
        if (!clean) return
        set((state) => {
          const current = state.overrides[msgId] ?? baseTags
          if (current.includes(clean)) return state
          return { overrides: { ...state.overrides, [msgId]: [...current, clean] } }
        })
      },
      removeTag: (msgId, baseTags, tag) =>
        set((state) => {
          const current = state.overrides[msgId] ?? baseTags
          return { overrides: { ...state.overrides, [msgId]: current.filter((tt) => tt !== tag) } }
        }),
    }),
    { name: 'cosmi-inbox-tags' },
  ),
)

/** Known tags used as suggestions in the tag picker (mock-first). */
export const SUGGESTED_TAGS = [
  'Demo',
  'Enterprise',
  'Support',
  'Technik',
  'Lead',
  'Vertrag',
  'Angebot',
  'Feedback',
  'Dringend',
  'Intern',
]
