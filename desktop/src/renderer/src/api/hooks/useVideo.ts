/**
 * TanStack Query hooks for video calls and recordings.
 *
 * Provides queries for active calls and recordings, plus mutations
 * for call lifecycle (create, join, end) and recording management.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listActiveCalls,
  getCall,
  listRecordings,
  getRecordingConsent,
  getRecordingConsents,
  tagRecordingWithConsents,
  updateRecordingMetadata,
  listRecordingsByMeeting,
  createCall,
  joinCall,
  endCall,
  startRecording,
  stopRecording,
  setRecordingConsent,
  confirmInitiatorConsent,
  getMeetingChatHistory,
  sendMeetingChatMessage,
  getMeetingCoHosts,
  promoteCoHost,
  demoteCoHost,
  muteParticipant,
  muteAllParticipants,
  kickParticipant,
  setMeetingLock,
} from '../video-client'
import type { CreateCallRequest, RecordingStatus, ConsentSnapshotEntry, UpdateRecordingMetadataRequest, SendMeetingChatMessageRequest } from '../video-types'

// Terminal states — polling stops when one of these is reached.
const RECORDING_TERMINAL_STATES: RecordingStatus[] = ['completed', 'failed', 'deleted']

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

/** List all currently active calls. */
export function useActiveCalls() {
  return useQuery({
    queryKey: ['calls', 'active'],
    queryFn: listActiveCalls,
  })
}

/** Get a single call by ID. */
export function useCall(callId: string) {
  return useQuery({
    queryKey: ['calls', callId],
    queryFn: () => getCall(callId),
    enabled: !!callId,
  })
}

/** List recordings filtered by call or meeting. */
export function useRecordings(params: { callId?: string; meetingId?: string }) {
  return useQuery({
    queryKey: ['recordings', params],
    queryFn: () =>
      listRecordings({
        call_id: params.callId,
        meeting_id: params.meetingId,
      }),
    enabled: !!params.callId || !!params.meetingId,
  })
}

/** Get recording consent status. */
export function useRecordingConsent(recordingId: string) {
  return useQuery({
    queryKey: ['recordings', recordingId, 'consent'],
    queryFn: () => getRecordingConsent(recordingId),
    enabled: !!recordingId,
  })
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

/** Create a new call (1:1 or group). */
export function useCreateCall() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: CreateCallRequest) => createCall(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['calls'] })
    },
  })
}

/** Join an existing call and receive a LiveKit token. */
export function useJoinCall() {
  return useMutation({
    mutationFn: (callId: string) => joinCall(callId),
  })
}

/** End an active call. */
export function useEndCall() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (callId: string) => endCall(callId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['calls'] })
    },
  })
}

/**
 * Confirm that the recording initiator has acknowledged the pre-recording consent dialog.
 * Must be called before startRecording in the initiator flow (R2-P0.4).
 */
export function useConfirmInitiatorConsent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (recordingId: string) => confirmInitiatorConsent(recordingId),
    onSuccess: (_data, recordingId) => {
      // Invalidate the specific recording entry AND the global recordings list
      // so any poll/list that was open before consent re-fetches the stamped state.
      queryClient.invalidateQueries({ queryKey: ['recordings', recordingId] })
      queryClient.invalidateQueries({ queryKey: ['recordings'] })
    },
  })
}

/** Start recording the current call (requires DSGVO consent). */
export function useStartRecording() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (callId: string) => startRecording(callId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['recordings'] })
    },
  })
}

/** Stop an active recording. Caller must pass the recording id, not the call id —
 * the backend route is /recordings/{id}/stop. */
export function useStopRecording() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (recordingId: string) => stopRecording(recordingId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['recordings'] })
    },
  })
}

/**
 * Poll recording status for a given call until it reaches a terminal state
 * (completed, failed, deleted). Refetches every 3 s while still processing.
 *
 * Added in Welle 3 / Sprint 1 for the Egress-Webhook fix (R2-P0 Batch A).
 */
export function useRecordingStatus(callId: string) {
  return useQuery({
    queryKey: ['recordings', 'status', callId],
    queryFn: () => listRecordings({ call_id: callId }),
    enabled: !!callId,
    refetchInterval: (query) => {
      const recordings = query.state.data
      if (!recordings || recordings.length === 0) return 3_000
      const allTerminal = recordings.every((r) =>
        RECORDING_TERMINAL_STATES.includes(r.status),
      )
      return allTerminal ? false : 3_000
    },
    select: (recordings) => ({
      recordings,
      isProcessing: recordings.some(
        (r) => !RECORDING_TERMINAL_STATES.includes(r.status),
      ),
      latestStatus: recordings[0]?.status ?? null,
    }),
  })
}

/** Set recording consent (DSGVO compliance). */
export function useSetRecordingConsent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      recordingId,
      consented,
    }: {
      recordingId: string
      consented: boolean
    }) => setRecordingConsent(recordingId, consented),
    onSuccess: (_data, variables) => {
      // Invalidate the specific consent entry AND the global recordings list.
      // The list view shows a consent-status indicator that must reflect the updated state.
      queryClient.invalidateQueries({
        queryKey: ['recordings', variables.recordingId, 'consent'],
      })
      queryClient.invalidateQueries({ queryKey: ['recordings'] })
    },
  })
}

// ---------------------------------------------------------------------------
// S1.7 Recording Tagging Hooks (Welle 1.B)
// ---------------------------------------------------------------------------

/**
 * Get the consent snapshot + live consent rows for a recording.
 * Returns the frozen JSONB snapshot enriched with up-to-date consent status.
 */
export function useRecordingConsents(recordingId: string) {
  return useQuery({
    queryKey: ['recordings', recordingId, 'consents'],
    queryFn: () => getRecordingConsents(recordingId),
    enabled: !!recordingId,
  })
}

/**
 * Overwrite the consent snapshot on an existing recording.
 * Intended for administrative corrections or late-joiner updates.
 */
export function useTagRecordingWithConsents() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      recordingId,
      snapshot,
    }: {
      recordingId: string
      snapshot: ConsentSnapshotEntry[]
    }) => tagRecordingWithConsents(recordingId, snapshot),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['recordings', variables.recordingId, 'consents'],
      })
    },
  })
}

/**
 * Update mutable metadata on a recording (file_url, file_size_bytes, duration_seconds, status).
 * Primarily used by internal tooling; the egress webhook calls this automatically.
 */
export function useUpdateRecordingMetadata() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      recordingId,
      metadata,
    }: {
      recordingId: string
      metadata: UpdateRecordingMetadataRequest
    }) => updateRecordingMetadata(recordingId, metadata),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['recordings', variables.recordingId],
      })
    },
  })
}

/**
 * Poll for an active recording for a given meeting.
 *
 * Used by MeetingRoomView (no LiveKit context) and MeetingLobby to detect
 * whether recording is in progress without relying on the LiveKit useIsRecording
 * hook. Polls every 10 seconds while the component is mounted.
 *
 * Returns the first recording with status === 'active', or null when no active
 * recording exists. Stops polling once the recording reaches a terminal state.
 */
export function useActiveMeetingRecording(meetingId: string | undefined) {
  return useQuery({
    queryKey: ['recordings', 'active', meetingId],
    queryFn: () => listRecordings({ meeting_id: meetingId }),
    enabled: !!meetingId,
    refetchInterval: 10_000,
    select: (recordings) =>
      recordings.find((r) => r.status === 'active') ?? null,
  })
}

/**
 * List all recordings for a specific meeting with optional pagination.
 * Distinct from useRecordings which filters by call_id or meeting_id via the general endpoint.
 */
export function useListRecordingsByMeeting(
  meetingId: string,
  opts?: { page?: number; pageSize?: number },
) {
  return useQuery({
    queryKey: ['recordings', 'by-meeting', meetingId, opts],
    queryFn: () =>
      listRecordingsByMeeting(meetingId, {
        page: opts?.page,
        page_size: opts?.pageSize,
      }),
    enabled: !!meetingId,
  })
}

// ---------------------------------------------------------------------------
// Meeting Chat hooks (Wave 2)
// ---------------------------------------------------------------------------

/**
 * Fetch the persisted chat history for a meeting once on join.
 * Live messages during the session arrive via LiveKit useChat DataChannel.
 * This query is fetched once (staleTime: Infinity) — no mid-session refetch.
 */
export function useMeetingChatHistory(meetingId: string) {
  return useQuery({
    queryKey: ['meetings', meetingId, 'chat'],
    queryFn: () => getMeetingChatHistory(meetingId),
    enabled: !!meetingId,
    staleTime: Infinity,
    select: (data) => data.messages,
  })
}

/**
 * Persist a chat message to the backend (fire-and-forget from the UI perspective).
 * The live delivery to peers happens via LiveKit useChat; this mutation just ensures
 * the message is stored in the DB for future reference.
 */
export function useSaveMeetingChatMessage(meetingId: string) {
  return useMutation({
    mutationFn: (req: SendMeetingChatMessageRequest) =>
      sendMeetingChatMessage(meetingId, req),
  })
}

// ---------------------------------------------------------------------------
// Meeting Co-Host hooks (Wave 3)
// ---------------------------------------------------------------------------


/**
 * Fetch the list of co-host user IDs for a meeting.
 * Refetches every 30 s while the component is mounted so the roster stays
 * in sync when the organizer promotes / demotes during the call.
 */
export function useMeetingCoHosts(meetingId: string) {
  return useQuery({
    queryKey: ['meetings', meetingId, 'cohosts'],
    queryFn: async () => {
      const res = await getMeetingCoHosts(meetingId)
      return res.user_ids ?? []
    },
    enabled: !!meetingId,
    refetchInterval: 30_000,
  })
}

/** Promote a participant to co-host (organizer only). */
export function usePromoteCoHost(meetingId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => promoteCoHost(meetingId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['meetings', meetingId, 'cohosts'] })
    },
  })
}

/** Demote a co-host back to regular participant (organizer only). */
export function useDemoteCoHost(meetingId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => demoteCoHost(meetingId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['meetings', meetingId, 'cohosts'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Meeting Moderation hooks (Wave 3)
// ---------------------------------------------------------------------------

/** Mute a specific participant server-side (host/co-host only). */
export function useMuteParticipant(meetingId: string) {
  return useMutation({
    mutationFn: (targetUserId: string) => muteParticipant(meetingId, targetUserId),
  })
}

/** Mute every participant except the caller (host/co-host only). */
export function useMuteAllParticipants(meetingId: string) {
  return useMutation({
    mutationFn: () => muteAllParticipants(meetingId),
  })
}

/** Kick a participant from the room (host/co-host only). */
export function useKickParticipant(meetingId: string) {
  return useMutation({
    mutationFn: (targetUserId: string) => kickParticipant(meetingId, targetUserId),
  })
}

/**
 * Lock or unlock the meeting room (host/co-host only).
 * `locked: true` prevents new participants from joining.
 */
export function useSetMeetingLock(meetingId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (locked: boolean) => setMeetingLock(meetingId, locked),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['meetings', meetingId] })
    },
  })
}
