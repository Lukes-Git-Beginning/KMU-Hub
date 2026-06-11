import { create } from 'zustand'

export type NavigationIntent =
  | { type: 'compose-email'; data: Record<string, string> }
  | { type: 'start-call'; data: Record<string, string> }
  | { type: 'send-message'; data: { name: string; userId?: string; channelId?: string; contactId?: string } }
  | { type: 'open-contact'; data: Record<string, string> }
  | { type: 'open-team-modulzuteilung'; data: { moduleId?: string; userIds?: string[] } }

interface NavigationState {
  intent: NavigationIntent | null
  setIntent: (intent: NavigationIntent) => void
  consumeIntent: () => NavigationIntent | null
  clearIntent: () => void
}

export const useNavigationStore = create<NavigationState>((set, get) => ({
  intent: null,

  setIntent: (intent) => set({ intent }),

  consumeIntent: () => {
    const current = get().intent
    if (current) {
      set({ intent: null })
    }
    return current
  },

  clearIntent: () => set({ intent: null }),
}))
