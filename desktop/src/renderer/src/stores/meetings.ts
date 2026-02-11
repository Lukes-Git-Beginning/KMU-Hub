/**
 * Meetings UI state store (Zustand, ephemeral -- not persisted).
 *
 * Manages UI-only state for the meetings module: selected meeting,
 * form dialog visibility, active meeting room. Actual meeting data
 * is managed by TanStack Query (useMeetings/useMeeting hooks).
 */
import { create } from 'zustand'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MeetingsUIState {
  // Selection / navigation
  selectedMeetingId: string | null
  isFormOpen: boolean
  editMeetingId: string | null
  activeMeetingRoomId: string | null
  deleteConfirmId: string | null

  // Actions
  selectMeeting: (id: string | null) => void
  openForm: (editId?: string | null) => void
  closeForm: () => void
  setActiveMeetingRoom: (id: string | null) => void
  setDeleteConfirmId: (id: string | null) => void
  reset: () => void
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useMeetingsUIStore = create<MeetingsUIState>()((set) => ({
  selectedMeetingId: null,
  isFormOpen: false,
  editMeetingId: null,
  activeMeetingRoomId: null,
  deleteConfirmId: null,

  selectMeeting: (id) => set({ selectedMeetingId: id }),

  openForm: (editId = null) => set({ isFormOpen: true, editMeetingId: editId }),

  closeForm: () => set({ isFormOpen: false, editMeetingId: null }),

  setActiveMeetingRoom: (id) => set({ activeMeetingRoomId: id }),

  setDeleteConfirmId: (id) => set({ deleteConfirmId: id }),

  reset: () =>
    set({
      selectedMeetingId: null,
      isFormOpen: false,
      editMeetingId: null,
      activeMeetingRoomId: null,
      deleteConfirmId: null,
    }),
}))
