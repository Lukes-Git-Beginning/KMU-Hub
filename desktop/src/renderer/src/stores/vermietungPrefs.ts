/**
 * Zustand store for FE-only vermietung preferences.
 * Fields not backed by the API (renterType, weeklyRate, currency, etc.) are persisted here.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface RentalPrefs {
  renterType?: 'employee' | 'customer'
  pickupLocation?: string
  returnLocation?: string
  currency?: string
}

interface ObjectPrefs {
  weeklyRate?: number
  currency?: string
  serialNumber?: string
}

interface VermietungPrefsStore {
  rentalPrefs: Record<string, RentalPrefs>
  objectPrefs: Record<string, ObjectPrefs>
  setRentalPref: (id: string, prefs: Partial<RentalPrefs>) => void
  setObjectPref: (id: string, prefs: Partial<ObjectPrefs>) => void
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useVermietungPrefsStore = create<VermietungPrefsStore>()(
  persist(
    (set) => ({
      rentalPrefs: {},
      objectPrefs: {},
      setRentalPref: (id, prefs) =>
        set((state) => ({
          rentalPrefs: {
            ...state.rentalPrefs,
            [id]: { ...(state.rentalPrefs[id] ?? {}), ...prefs },
          },
        })),
      setObjectPref: (id, prefs) =>
        set((state) => ({
          objectPrefs: {
            ...state.objectPrefs,
            [id]: { ...(state.objectPrefs[id] ?? {}), ...prefs },
          },
        })),
    }),
    {
      name: 'vermietung-prefs',
    },
  ),
)
