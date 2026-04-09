import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type CallPhase = 'idle' | 'dialing' | 'on_call' | 'wrap_up'

interface DialerState {
  callPhase: CallPhase
  activeContactId: string | null
  activeSessionId: string | null
  callStartedAt: number | null
  selectedOutcomeId: string | null
  callbackAt: string | null
  draftNotes: string
  activeCampaignId: string | null
  isAddContactsDialogOpen: boolean
  isWrapUpSubmitting: boolean

  startDialing: (contactId: string, campaignId: string) => void
  onCallConnected: (sessionId: string) => void
  onHangUp: () => void
  selectOutcome: (outcomeId: string | null) => void
  setCallbackAt: (iso: string | null) => void
  setDraftNotes: (text: string) => void
  completeWrapUp: () => void
  setActiveCampaign: (id: string | null) => void
  setAddContactsDialogOpen: (open: boolean) => void
  setWrapUpSubmitting: (submitting: boolean) => void
  reset: () => void
}

const initialState = {
  callPhase: 'idle' as CallPhase,
  activeContactId: null,
  activeSessionId: null,
  callStartedAt: null,
  selectedOutcomeId: null,
  callbackAt: null,
  draftNotes: '',
  activeCampaignId: null,
  isAddContactsDialogOpen: false,
  isWrapUpSubmitting: false,
}

export const useDialerStore = create<DialerState>()(
  persist(
    (set) => ({
      ...initialState,

      startDialing: (contactId, campaignId) =>
        set({
          callPhase: 'dialing',
          activeContactId: contactId,
          activeCampaignId: campaignId,
          draftNotes: '',
          selectedOutcomeId: null,
          callbackAt: null,
        }),

      onCallConnected: (sessionId) =>
        set({
          callPhase: 'on_call',
          activeSessionId: sessionId,
          callStartedAt: Date.now(),
        }),

      onHangUp: () =>
        set({ callPhase: 'wrap_up' }),

      selectOutcome: (outcomeId) =>
        set({ selectedOutcomeId: outcomeId }),

      setCallbackAt: (iso) =>
        set({ callbackAt: iso }),

      setDraftNotes: (text) =>
        set({ draftNotes: text }),

      completeWrapUp: () =>
        set({
          callPhase: 'idle',
          activeContactId: null,
          activeSessionId: null,
          callStartedAt: null,
          selectedOutcomeId: null,
          callbackAt: null,
          draftNotes: '',
          isWrapUpSubmitting: false,
        }),

      setActiveCampaign: (id) =>
        set({ activeCampaignId: id }),

      setAddContactsDialogOpen: (open) =>
        set({ isAddContactsDialogOpen: open }),

      setWrapUpSubmitting: (submitting) =>
        set({ isWrapUpSubmitting: submitting }),

      reset: () => set(initialState),
    }),
    {
      name: 'cosmi-dialer',
      partialize: (state) => ({ activeCampaignId: state.activeCampaignId }),
    }
  )
)
