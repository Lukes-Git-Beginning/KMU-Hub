import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { BusinessProfileId } from '@/config/business-profiles'

// ---- Work Profiles (focus views within the app) ----

export interface WorkProfile {
  id: string
  name: string
  description: string
  icon: string
  color: string
  isDefault: boolean
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
    id: 'finanzen',
    name: 'Finanzen',
    description: 'Finanzen, Rechnungen & Berichte',
    icon: '\uD83D\uDCB0',
    color: '#d97706',
    isDefault: false,
  },
]

// ---- Combined State ----

interface ProfileState {
  // Work profiles (focus views)
  profiles: WorkProfile[]
  activeProfileId: string
  switchProfile: (profileId: string) => void
  createProfile: (profile: Omit<WorkProfile, 'id'>) => void
  deleteProfile: (profileId: string) => void
  setDefaultProfile: (profileId: string) => void

  // Business profile (industry — determines which modules are visible)
  businessProfileId: BusinessProfileId | null
  devShowAllModules: boolean
  enabledOptionalModules: string[]
  setBusinessProfile: (id: BusinessProfileId | null) => void
  toggleDevShowAll: () => void
  enableModule: (moduleId: string) => void
  disableModule: (moduleId: string) => void
}

export const useProfileStore = create<ProfileState>()(
  persist(
    (set, get) => ({
      // ---- Work Profiles ----
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

      // ---- Business Profile (Industry) ----
      businessProfileId: null,
      devShowAllModules: false,
      enabledOptionalModules: [],

      setBusinessProfile: (id) => set({ businessProfileId: id, enabledOptionalModules: [] }),
      toggleDevShowAll: () => set((s) => ({ devShowAllModules: !s.devShowAllModules })),
      enableModule: (moduleId) =>
        set((s) => ({
          enabledOptionalModules: s.enabledOptionalModules.includes(moduleId)
            ? s.enabledOptionalModules
            : [...s.enabledOptionalModules, moduleId],
        })),
      disableModule: (moduleId) =>
        set((s) => ({
          enabledOptionalModules: s.enabledOptionalModules.filter((id) => id !== moduleId),
        })),
    }),
    {
      name: 'cosmi-profiles',
    },
  ),
)
