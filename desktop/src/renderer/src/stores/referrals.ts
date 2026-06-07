import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Contact referrals ("Empfohlen von") — who referred a given contact.
 *
 * Self-referential contact relationship, stored as a local overlay (contacts
 * themselves come from the backend, like favoriteIds in the contacts store).
 * MOCK-FIRST. Backend: `contacts.referred_by` (self-FK) + referrer aggregation —
 * see .planning/backend-gaps.md (P8).
 */
interface ReferralsState {
  /** contactId -> referrerContactId */
  referredBy: Record<string, string>
  setReferrer: (contactId: string, referrerId: string) => void
  clearReferrer: (contactId: string) => void
}

export const useReferralsStore = create<ReferralsState>()(
  persist(
    (set) => ({
      referredBy: {},
      setReferrer: (contactId, referrerId) =>
        set((s) => ({ referredBy: { ...s.referredBy, [contactId]: referrerId } })),
      clearReferrer: (contactId) =>
        set((s) => {
          const next = { ...s.referredBy }
          delete next[contactId]
          return { referredBy: next }
        }),
    }),
    { name: 'cosmi-contact-referrals' },
  ),
)
