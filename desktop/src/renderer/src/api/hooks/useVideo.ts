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
  createCall,
  joinCall,
  endCall,
  startRecording,
  stopRecording,
  setRecordingConsent,
} from '../video-client'
import type { CreateCallRequest, RecordingStatus } from '../video-types'

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

/** Stop an active recording. */
export function useStopRecording() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (callId: string) => stopRecording(callId),
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
