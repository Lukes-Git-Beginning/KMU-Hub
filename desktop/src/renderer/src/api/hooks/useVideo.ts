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
} from '../video-client'
import type { CreateCallRequest, RecordingStatus, ConsentSnapshotEntry, UpdateRecordingMetadataRequest } from '../video-types'

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
      queryClient.invalidateQueries({ queryKey: ['recordings', recordingId] })
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
      queryClient.invalidateQueries({
        queryKey: ['recordings', variables.recordingId, 'consent'],
      })
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
