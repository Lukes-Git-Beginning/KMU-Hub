/**
 * TanStack Query hooks for meetings, notes, and action items.
 *
 * Provides queries for meeting list/detail, notes, and action items,
 * plus mutations for full meeting lifecycle and task conversion.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listMeetings,
  getMeeting,
  getMeetingNotes,
  getPreviousMeetingNotes,
  listActionItems,
  createMeeting,
  updateMeeting,
  deleteMeeting,
  startMeeting,
  endMeeting,
  saveMeetingNotes,
  createActionItem,
  updateActionItem,
  deleteActionItem,
  convertActionItemsToTasks,
} from '../video-client'
import type {
  MeetingFilter,
  CreateMeetingRequest,
  UpdateMeetingRequest,
  SaveNotesRequest,
  CreateActionItemRequest,
  ActionItem,
} from '../video-types'

// ---------------------------------------------------------------------------
// Meeting queries
// ---------------------------------------------------------------------------

/** List meetings with optional filtering. */
export function useMeetings(filter?: MeetingFilter) {
  return useQuery({
    queryKey: ['meetings', filter],
    queryFn: () => listMeetings(filter),
  })
}

/** Get a single meeting by ID. */
export function useMeeting(id: string) {
  return useQuery({
    queryKey: ['meetings', id],
    queryFn: () => getMeeting(id),
    enabled: !!id,
  })
}

/** Get notes for a meeting. */
export function useMeetingNotes(meetingId: string) {
  return useQuery({
    queryKey: ['meetings', meetingId, 'notes'],
    queryFn: () => getMeetingNotes(meetingId),
    enabled: !!meetingId,
  })
}

/** Get notes from the previous occurrence of a recurring meeting. */
export function usePreviousMeetingNotes(meetingId: string) {
  return useQuery({
    queryKey: ['meetings', meetingId, 'previous-notes'],
    queryFn: () => getPreviousMeetingNotes(meetingId),
    enabled: !!meetingId,
  })
}

/** List action items for a meeting. */
export function useActionItems(meetingId: string) {
  return useQuery({
    queryKey: ['meetings', meetingId, 'action-items'],
    queryFn: () => listActionItems(meetingId),
    enabled: !!meetingId,
  })
}

// ---------------------------------------------------------------------------
// Meeting mutations
// ---------------------------------------------------------------------------

/** Create a new meeting. */
export function useCreateMeeting() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: CreateMeetingRequest) => createMeeting(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['meetings'] })
    },
  })
}

/** Update an existing meeting. */
export function useUpdateMeeting() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateMeetingRequest }) =>
      updateMeeting(id, data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['meetings'] })
      queryClient.invalidateQueries({ queryKey: ['meetings', variables.id] })
    },
  })
}

/** Delete a meeting. */
export function useDeleteMeeting() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteMeeting(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['meetings'] })
    },
  })
}

/** Start a meeting (transitions to in_progress, returns LiveKit token). */
export function useStartMeeting() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => startMeeting(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['meetings', id] })
    },
  })
}

/** End a meeting (transitions to completed, returns summary). */
export function useEndMeeting() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => endMeeting(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['meetings'] })
      queryClient.invalidateQueries({ queryKey: ['meetings', id] })
    },
  })
}

// ---------------------------------------------------------------------------
// Notes mutations
// ---------------------------------------------------------------------------

/** Save or update meeting notes. */
export function useSaveMeetingNotes() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      meetingId,
      data,
    }: {
      meetingId: string
      data: SaveNotesRequest
    }) => saveMeetingNotes(meetingId, data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['meetings', variables.meetingId, 'notes'],
      })
    },
  })
}

// ---------------------------------------------------------------------------
// Action item mutations
// ---------------------------------------------------------------------------

/** Create an action item for a meeting. */
export function useCreateActionItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      meetingId,
      data,
    }: {
      meetingId: string
      data: CreateActionItemRequest
    }) => createActionItem(meetingId, data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['meetings', variables.meetingId, 'action-items'],
      })
    },
  })
}

/** Update an action item. */
export function useUpdateActionItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      itemId,
      data,
    }: {
      itemId: string
      data: Partial<ActionItem>
    }) => updateActionItem(itemId, data),
    onSuccess: () => {
      // Invalidate all action item queries since we don't track meetingId here
      queryClient.invalidateQueries({
        queryKey: ['meetings'],
        predicate: (query) =>
          Array.isArray(query.queryKey) &&
          query.queryKey[query.queryKey.length - 1] === 'action-items',
      })
    },
  })
}

/** Delete an action item. */
export function useDeleteActionItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (itemId: string) => deleteActionItem(itemId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['meetings'],
        predicate: (query) =>
          Array.isArray(query.queryKey) &&
          query.queryKey[query.queryKey.length - 1] === 'action-items',
      })
    },
  })
}

/** Convert action items to tasks in a project. */
export function useConvertActionItemsToTasks() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      meetingId,
      projectId,
    }: {
      meetingId: string
      projectId: string
    }) => convertActionItemsToTasks(meetingId, { project_id: projectId }),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['meetings', variables.meetingId, 'action-items'],
      })
      // Also invalidate tasks since new ones were created
      queryClient.invalidateQueries({ queryKey: ['tasks'] })
    },
  })
}
