import { create } from 'zustand'

export interface NavigationIntent {
  type: 'compose-email' | 'start-call' | 'send-message' | 'open-contact'
  data: Record<string, string>
}

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
