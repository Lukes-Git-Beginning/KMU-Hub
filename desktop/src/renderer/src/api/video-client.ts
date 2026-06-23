/**
 * Lightweight fetch wrapper for Video, Meeting, Presence & Reaction API endpoints.
 *
 * Follows the same pattern as calendar-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry. Once these routes are added to
 * openapi.yaml and types regenerated, hooks can migrate to the typed apiClient.
 */
import type {
  CallSession,
  CreateCallRequest,
  JoinCallResponse,
  Recording,
  RecordingConsentStatus,
  RecordingConsentsResponse,
  ConsentSnapshotEntry,
  UpdateRecordingMetadataRequest,
  RecordingsByMeetingResponse,
  Meeting,
  CreateMeetingRequest,
  UpdateMeetingRequest,
  MeetingFilter,
  MeetingSummary,
  StartMeetingResponse,
  JoinMeetingResponse,
  MeetingNotes,
  SaveNotesRequest,
  ActionItem,
  CreateActionItemRequest,
  ConvertActionItemsRequest,
  ConvertActionItemsResponse,
  UserPresence,
  PresenceConfig,
  PresenceLevel,
  ToggleReactionRequest,
  ToggleReactionResponse,
  ListReactionsApiResponse,
  ReactionSummaryApiResponse,
  MeetingChatMessage,
  MeetingChatHistoryResponse,
  SendMeetingChatMessageRequest,
} from './video-types'
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Internal fetch helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | string[] | undefined>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
}

// Convenience methods
function get<T>(path: string, params?: RequestOptions['params']): Promise<T> {
  return request<T>({ method: 'GET', path, params })
}

function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>({ method: 'POST', path, body })
}

function put<T>(path: string, body?: unknown): Promise<T> {
  return request<T>({ method: 'PUT', path, body })
}

function del<T = void>(path: string): Promise<T> {
  return request<T>({ method: 'DELETE', path })
}

// ---------------------------------------------------------------------------
// Calls
// ---------------------------------------------------------------------------

export const createCall = (req: CreateCallRequest) =>
  post<CallSession>('/api/v1/video/calls', req)

export const joinCall = (callId: string) =>
  post<JoinCallResponse>(`/api/v1/video/calls/${callId}/join`)

export const endCall = (callId: string) =>
  post<void>(`/api/v1/video/calls/${callId}/end`)

export const getCall = (callId: string) =>
  get<CallSession>(`/api/v1/video/calls/${callId}`)

export const listActiveCalls = () =>
  get<CallSession[]>('/api/v1/video/calls')

// ---------------------------------------------------------------------------
// Recordings
// ---------------------------------------------------------------------------

export const confirmInitiatorConsent = (recordingId: string) =>
  post<{ stamped: boolean }>(`/api/v1/video/recordings/${recordingId}/initiator-consent`)

export const startRecording = (callId: string) =>
  post<Recording>('/api/v1/video/recordings/start', { call_id: callId })

export const stopRecording = (recordingId: string) =>
  post<Recording>(`/api/v1/video/recordings/${recordingId}/stop`)

export const setRecordingConsent = (recordingId: string, consented: boolean) =>
  post<void>(`/api/v1/video/recordings/${recordingId}/consent`, { consented })

export const getRecordingConsent = (recordingId: string) =>
  get<RecordingConsentStatus>(`/api/v1/video/recordings/${recordingId}/consent`)

export const listRecordings = (params: { call_id?: string; meeting_id?: string }) =>
  get<Recording[]>('/api/v1/video/recordings', params)

/** Get consent snapshot + live consent rows for a recording (S1.7). */
export const getRecordingConsents = (recordingId: string) =>
  get<RecordingConsentsResponse>(`/api/v1/video/recordings/${recordingId}/consents`)

/** Overwrite the consent snapshot on a recording (S1.7). */
export const tagRecordingWithConsents = (recordingId: string, snapshot: ConsentSnapshotEntry[]) =>
  post<void>(`/api/v1/video/recordings/${recordingId}/tag-consents`, { snapshot })

/** Update mutable metadata on a recording (S1.7). */
export const updateRecordingMetadata = (recordingId: string, req: UpdateRecordingMetadataRequest) =>
  request<Recording>({ method: 'PATCH', path: `/api/v1/video/recordings/${recordingId}/metadata`, body: req })

/** Get the current status of a single recording (S1.7). */
export const getRecordingStatus = (recordingId: string) =>
  get<Recording>(`/api/v1/video/recordings/${recordingId}/status`)

/** List recordings for a specific meeting with pagination (S1.7). */
export const listRecordingsByMeeting = (meetingId: string, params?: { page?: number; page_size?: number }) =>
  get<RecordingsByMeetingResponse>(`/api/v1/video/meetings/${meetingId}/recordings`, params as RequestOptions['params'])

/** Get a presigned download URL for a completed recording (Wave 5). */
export const getRecordingDownloadURL = (recordingId: string) =>
  get<{ url: string; expires_at: string }>(`/api/v1/video/recordings/${recordingId}/download`)

// ---------------------------------------------------------------------------
// Meetings
// ---------------------------------------------------------------------------

export const createMeeting = (req: CreateMeetingRequest) =>
  post<Meeting>('/api/v1/meetings', req)

export const getMeeting = (id: string) =>
  get<Meeting>(`/api/v1/meetings/${id}`)

export const updateMeeting = (id: string, req: UpdateMeetingRequest) =>
  put<Meeting>(`/api/v1/meetings/${id}`, req)

export const deleteMeeting = (id: string) =>
  del<void>(`/api/v1/meetings/${id}`)

export const listMeetings = async (filter?: MeetingFilter): Promise<Meeting[]> => {
  // The gateway returns a bare Meeting[] (response.ProtoList). Older builds and
  // the demo mock returned a wrapped {meetings,total} envelope; unwrap defensively
  // so a stale persisted cache or mock never reaches the UI as a non-array.
  const res = await get<Meeting[] | { meetings?: Meeting[] }>(
    '/api/v1/meetings',
    filter as RequestOptions['params'],
  )
  return Array.isArray(res) ? res : (res?.meetings ?? [])
}

export const startMeeting = (id: string) =>
  post<StartMeetingResponse>(`/api/v1/meetings/${id}/start`)

/**
 * Idempotent participant join: the organizer's first call starts the meeting,
 * any invited attendee then gets a LiveKit token + TURN ice_servers for the room.
 */
export const joinMeeting = (id: string) =>
  post<JoinMeetingResponse>(`/api/v1/meetings/${id}/join`)

export const endMeeting = (id: string) =>
  post<MeetingSummary>(`/api/v1/meetings/${id}/end`)

// ---------------------------------------------------------------------------
// Meeting Notes
// ---------------------------------------------------------------------------

export const saveMeetingNotes = (meetingId: string, req: SaveNotesRequest) =>
  put<MeetingNotes>(`/api/v1/meetings/${meetingId}/notes`, req)

export const getMeetingNotes = (meetingId: string) =>
  get<MeetingNotes>(`/api/v1/meetings/${meetingId}/notes`)

export const getPreviousMeetingNotes = (meetingId: string) =>
  get<MeetingNotes>(`/api/v1/meetings/${meetingId}/previous-notes`)

// ---------------------------------------------------------------------------
// Action Items
// ---------------------------------------------------------------------------

export const createActionItem = (meetingId: string, req: CreateActionItemRequest) =>
  post<ActionItem>(`/api/v1/meetings/${meetingId}/action-items`, req)

export const listActionItems = (meetingId: string) =>
  get<ActionItem[]>(`/api/v1/meetings/${meetingId}/action-items`)

export const updateActionItem = (itemId: string, req: Partial<ActionItem>) =>
  put<ActionItem>(`/api/v1/meetings/action-items/${itemId}`, req)

export const deleteActionItem = (itemId: string) =>
  del<void>(`/api/v1/meetings/action-items/${itemId}`)

export const convertActionItemsToTasks = (
  meetingId: string,
  req: ConvertActionItemsRequest,
) => post<ConvertActionItemsResponse>(`/api/v1/meetings/${meetingId}/action-items/convert`, req)

// ---------------------------------------------------------------------------
// Presence
// ---------------------------------------------------------------------------

export const getPresence = (userId: string) =>
  get<UserPresence>(`/api/v1/video/presence/${userId}`)

export const getBulkPresence = (userIds: string[]) =>
  post<Record<string, UserPresence>>('/api/v1/video/presence/bulk', { user_ids: userIds })

export const setPresenceStatus = (status: PresenceLevel) =>
  post<void>('/api/v1/video/presence/status', { status })

export const getPresenceConfig = () =>
  get<PresenceConfig>('/api/v1/video/presence/config')

export const updatePresenceConfig = (awayTimeoutSeconds: number) =>
  put<void>('/api/v1/video/presence/config', { away_timeout_seconds: awayTimeoutSeconds })

// ---------------------------------------------------------------------------
// Reactions (chat service — /api/v1/messages)
// ---------------------------------------------------------------------------

export const toggleReaction = (req: ToggleReactionRequest) =>
  post<ToggleReactionResponse>(`/api/v1/messages/${req.message_id}/reactions`, { emoji: req.emoji })

export const listReactions = (messageId: string) =>
  get<ListReactionsApiResponse>(`/api/v1/messages/${messageId}/reactions`)

export const getReactionSummary = (messageIds: string[]) =>
  post<ReactionSummaryApiResponse>('/api/v1/messages/reactions/summary', {
    message_ids: messageIds,
  })

// ---------------------------------------------------------------------------
// Meeting Chat (Wave 2)
// ---------------------------------------------------------------------------

/**
 * Fetch the persisted chat history for a meeting (up to 100 messages, ascending by created_at).
 * Called once on join — live messages during the session come via LiveKit useChat DataChannel.
 */
export const getMeetingChatHistory = (meetingId: string) =>
  get<MeetingChatHistoryResponse>(`/api/v1/meetings/${meetingId}/chat`, { limit: 100 })

/**
 * Persist a chat message for a meeting (fire-and-forget; live delivery via useChat).
 * sender_id is resolved server-side from the JWT; sender_name is a display hint only.
 */
export const sendMeetingChatMessage = (meetingId: string, req: SendMeetingChatMessageRequest) =>
  post<MeetingChatMessage>(`/api/v1/meetings/${meetingId}/chat`, req)

// ---------------------------------------------------------------------------
// Meeting Co-Hosts (Wave 3)
// ---------------------------------------------------------------------------

/**
 * List current co-hosts for a meeting.
 * Returns an array of user IDs that have co-host privileges.
 */
export const getMeetingCoHosts = (meetingId: string) =>
  get<{ user_ids: string[] }>(`/api/v1/meetings/${meetingId}/cohosts`)

/**
 * Promote a participant to co-host.
 * Only the organizer may call this.
 */
export const promoteCoHost = (meetingId: string, userId: string) =>
  post<void>(`/api/v1/meetings/${meetingId}/cohosts`, { user_id: userId })

/**
 * Demote a co-host back to regular participant.
 * Only the organizer may call this.
 */
export const demoteCoHost = (meetingId: string, userId: string) =>
  del<void>(`/api/v1/meetings/${meetingId}/cohosts/${userId}`)

// ---------------------------------------------------------------------------
// Meeting Moderation (Wave 3)
// ---------------------------------------------------------------------------

/**
 * Mute a specific participant (server-authoritative — LiveKit mutes via RoomServiceClient).
 * Callable by host or co-host.
 */
export const muteParticipant = (meetingId: string, targetUserId: string) =>
  post<void>(`/api/v1/meetings/${meetingId}/moderation/mute`, {
    target_user_id: targetUserId,
  })

/**
 * Mute all participants except the caller.
 * Callable by host or co-host.
 */
export const muteAllParticipants = (meetingId: string) =>
  post<void>(`/api/v1/meetings/${meetingId}/moderation/mute-all`)

/**
 * Remove (kick) a participant from the meeting room.
 * Callable by host or co-host.
 */
export const kickParticipant = (meetingId: string, targetUserId: string) =>
  post<void>(`/api/v1/meetings/${meetingId}/moderation/kick`, {
    target_user_id: targetUserId,
  })

/**
 * Lock or unlock the meeting room.
 * When locked, new participants cannot join.
 * Callable by host or co-host.
 */
export const setMeetingLock = (meetingId: string, locked: boolean) =>
  post<void>(`/api/v1/meetings/${meetingId}/lock`, { locked })
