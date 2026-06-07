import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { emptyProtocol, type AdvisoryProtocol } from '@/lib/advisory'

/**
 * Beratungsprotokolle (advisory protocols) per contact.
 *
 * MOCK-FIRST: persisted to localStorage. A finalized protocol is immutable
 * (legal documentation, ~10-year retention). Backend target: `advisory_protocols`
 * (contact_id, all fields, immutable-after-delivery, 10y retention) —
 * see .planning/backend-gaps.md.
 */
interface AdvisoryProtocolsState {
  protocols: AdvisoryProtocol[]
  byContact: (contactId: string) => AdvisoryProtocol[]
  get: (id: string) => AdvisoryProtocol | undefined
  /** Create a blank draft for a contact and return its id. */
  createDraft: (contactId: string, advisor?: string) => string
  /** Patch a draft. No-op if the protocol is finalized (immutable). */
  update: (id: string, patch: Partial<AdvisoryProtocol>) => void
  /** Lock a protocol after the document was handed to the customer. */
  finalize: (id: string) => void
  remove: (id: string) => void
}

function nowIso(): string {
  return new Date().toISOString()
}

export const useAdvisoryProtocolsStore = create<AdvisoryProtocolsState>()(
  persist(
    (set, get) => ({
      protocols: [],

      byContact: (contactId) =>
        get()
          .protocols.filter((p) => p.contactId === contactId)
          .sort((a, b) => (b.date || b.createdAt).localeCompare(a.date || a.createdAt)),

      get: (id) => get().protocols.find((p) => p.id === id),

      createDraft: (contactId, advisor = '') => {
        const id = crypto.randomUUID()
        const ts = nowIso()
        const draft = { ...emptyProtocol(contactId, id, advisor), createdAt: ts, updatedAt: ts }
        set((s) => ({ protocols: [...s.protocols, draft] }))
        return id
      },

      update: (id, patch) =>
        set((s) => ({
          protocols: s.protocols.map((p) =>
            p.id === id && p.status !== 'finalized'
              ? { ...p, ...patch, id: p.id, contactId: p.contactId, updatedAt: nowIso() }
              : p,
          ),
        })),

      finalize: (id) =>
        set((s) => ({
          protocols: s.protocols.map((p) =>
            p.id === id ? { ...p, status: 'finalized', updatedAt: nowIso() } : p,
          ),
        })),

      remove: (id) => set((s) => ({ protocols: s.protocols.filter((p) => p.id !== id) })),
    }),
    { name: 'cosmi-advisory-protocols' },
  ),
)
