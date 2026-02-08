import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface WorkProfile {
  id: string
  name: string
  description: string
  icon: string
  color: string
  isDefault: boolean
}

interface ProfileState {
  profiles: WorkProfile[]
  activeProfileId: string

  switchProfile: (profileId: string) => void
  createProfile: (profile: Omit<WorkProfile, 'id'>) => void
  deleteProfile: (profileId: string) => void
  setDefaultProfile: (profileId: string) => void
}

const DEFAULT_PROFILES: WorkProfile[] = [
  {
    id: 'default',
    name: 'Standard',
    description: 'Standardansicht mit allen Modulen',
    icon: '\u2699\uFE0F',
    color: '#059669',
    isDefault: true,
  },
  {
    id: 'projektleitung',
    name: 'Projektleitung',
    description: 'Fokus auf Projekte, Aufgaben & Team',
    icon: '\uD83D\uDCCB',
    color: '#2563eb',
    isDefault: false,
  },
  {
    id: 'buchhaltung',
    name: 'Buchhaltung',
    description: 'Finanzen, Rechnungen & Berichte',
    icon: '\uD83D\uDCB0',
    color: '#d97706',
    isDefault: false,
  },
]

export const useProfileStore = create<ProfileState>()(
  persist(
    (set, get) => ({
      profiles: DEFAULT_PROFILES,
      activeProfileId: 'default',

      switchProfile: (profileId: string) => {
        const profile = get().profiles.find((p) => p.id === profileId)
        if (profile) {
          set({ activeProfileId: profileId })
        }
      },

      createProfile: (profile: Omit<WorkProfile, 'id'>) => {
        const newProfile: WorkProfile = {
          ...profile,
          id: `profile-${Date.now()}`,
        }
        set((state) => ({ profiles: [...state.profiles, newProfile] }))
      },

      deleteProfile: (profileId: string) => {
        if (profileId === 'default') return
        set((state) => {
          const next = state.profiles.filter((p) => p.id !== profileId)
          const activeId =
            state.activeProfileId === profileId
              ? 'default'
              : state.activeProfileId
          return { profiles: next, activeProfileId: activeId }
        })
      },

      setDefaultProfile: (profileId: string) => {
        set((state) => ({
          profiles: state.profiles.map((p) => ({
            ...p,
            isDefault: p.id === profileId,
          })),
        }))
      },
    }),
    {
      name: 'kmuhub-profiles',
    },
  ),
)
