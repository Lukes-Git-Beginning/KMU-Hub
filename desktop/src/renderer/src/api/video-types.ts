/**
 * Video, Meeting, Presence & Reaction TypeScript types.
 *
 * Manually defined types for the Phase 8 API entities. These types mirror
 * the backend video/meeting/presence/reaction service models and will be
 * replaced by auto-generated types once the OpenAPI spec is updated.
 */

// ---------------------------------------------------------------------------
// Enums / union types
// ---------------------------------------------------------------------------

export type CallType = 'one_to_one' | 'group'
export type CallStatus = 'ringing' | 'active' | 'ended'
export type MeetingStatus = 'scheduled' | 'in_progress' | 'completed' | 'cancelled'
export type RecordingStatus = 'active' | 'processing' | 'completed' | 'failed' | 'deleted'
export type PresenceLevel = 'online' | 'away' | 'dnd' | 'in_call' | 'offline'
export type RsvpStatus = 'pending' | 'accepted' | 'declined' | 'tentative'

// ---------------------------------------------------------------------------
// Call entities
// ---------------------------------------------------------------------------

export interface CallParticipant {
  call_id: string
  user_id: string
  joined_at: string
  left_at: string | null
  has_video: boolean
  has_audio: boolean
}

export interface CallSession {
  id: string
  call_type: CallType
  status: CallStatus
  room_name: string
  initiator_id: string
  channel_id: string | null
  created_at: string
  ended_at: string | null
  duration_seconds: number | null
  participants: CallParticipant[] | null
}

export interface CreateCallRequest {
  call_type: CallType
  target_user_ids: string[]
  channel_id?: string
}

/** WebRTC ICE/TURN server (RTCIceServer shape) for relaying in NAT'd networks. */
export interface IceServer {
  urls: string[]
  username: string
  credential: string
}

export interface JoinCallResponse {
  token: string
  ws_url: string
  /** Per-session TURN credentials; empty/absent when TURN is not configured. */
  ice_servers?: IceServer[]
}

// ---------------------------------------------------------------------------
// Meeting entities
// ---------------------------------------------------------------------------

export interface MeetingAttendee {
  meeting_id: string
  user_id: string
  rsvp_status: RsvpStatus
  user_name?: string
}

export interface Meeting {
  id: string
  title: string
  description: string | null
  agenda: string | null
  organizer_id: string
  status: MeetingStatus
  scheduled_start: string
  scheduled_end: string
  actual_start: string | null
  actual_end: string | null
  room_name: string | null
  calendar_event_id: string | null
  recurring_meeting_id: string | null
  attendees: MeetingAttendee[] | null
  created_at: string
  updated_at: string
  locked: boolean
  contact_id: string | null
  deal_id: string | null
  /** AI-generated summary of the meeting's public notes (Wave 7C); null until generated. */
  ai_summary: string | null
  ai_summary_at: string | null
}

export interface CreateMeetingRequest {
  title: string
  description?: string
  agenda?: string
  scheduled_start: string
  scheduled_end: string
  attendee_ids: string[]
  calendar_event_id?: string
  recurring_meeting_id?: string
  contact_id?: string | null
  deal_id?: string | null
}

export interface UpdateMeetingRequest {
  title?: string
  description?: string
  agenda?: string
  scheduled_start?: string
  scheduled_end?: string
  contact_id?: string | null
  deal_id?: string | null
}

export interface MeetingFilter {
  organizer_id?: string
  status?: MeetingStatus
  start_after?: string
  start_before?: string
  limit?: number
  offset?: number
}

export interface MeetingSummary {
  meeting_id: string
  duration_minutes: number
  attendee_count: number
  notes_summary: string
  action_items: ActionItem[]
}

export interface StartMeetingResponse {
  meeting: Meeting
  token: string
  ws_url: string
}

/**
 * Response for the idempotent meeting join (POST /meetings/{id}/join).
 * The organizer's first call starts the meeting; any invited attendee then
 * receives a LiveKit token + per-session TURN ice_servers for the room.
 */
export interface JoinMeetingResponse {
  meeting: Meeting
  token: string
  ws_url: string
  room_name?: string
  /** Per-session TURN credentials; empty/absent when TURN is not configured. */
  ice_servers?: IceServer[]
}

// ---------------------------------------------------------------------------
// Notes entities
// ---------------------------------------------------------------------------

export interface MeetingNotes {
  id: string
  meeting_id: string
  author_id: string
  content: string
  is_private: boolean
  created_at: string
  updated_at: string
}

export interface SaveNotesRequest {
  content: string
  is_private: boolean
}

// ---------------------------------------------------------------------------
// Action item entities
// ---------------------------------------------------------------------------

export interface ActionItem {
  id: string
  meeting_id: string
  description: string
  assignee_id: string | null
  is_completed: boolean
  task_id: string | null
  sort_order: number
  created_at: string
}

export interface CreateActionItemRequest {
  description: string
  assignee_id?: string
}

export interface ConvertActionItemsRequest {
  /** IDs of the action items to convert. Must contain at least one entry. */
  action_item_ids: string[]
  project_id: string
}

export interface ConvertActionItemsResponse {
  task_ids: string[]
}

// ---------------------------------------------------------------------------
// Recording entities
// ---------------------------------------------------------------------------

export interface Recording {
  id: string
  call_id: string | null
  meeting_id: string | null
  status: RecordingStatus
  egress_id: string | null
  file_url: string | null
  // protojson serializes int64 as a JSON string (proto3 spec); encoding/json
  // emitted a number. Accept both and coerce at the call site.
  file_size_bytes: number | string | null
  duration_seconds: number | null
  retention_expires_at: string | null
  created_at: string
  started_by: string | null
}

export interface RecordingConsent {
  recording_id: string
  user_id: string
  consented: boolean
  responded_at: string
  first_name?: string
  last_name?: string
}

export interface RecordingConsentStatus {
  all_responded: boolean
  consents: RecordingConsent[]
}

/** Enriched consent status returned by GetRecordingConsents (S1.7) */
export interface RecordingConsentsResponse {
  recording_id: string
  consents: RecordingConsent[]
  all_consented: boolean
}

/** A single entry in the consent snapshot */
export interface ConsentSnapshotEntry {
  user_id: string
  display_name: string
  joined_at?: string
}

/** Request body for PATCH /recordings/{id}/metadata */
export interface UpdateRecordingMetadataRequest {
  file_url?: string
  file_size_bytes?: number
  duration_seconds?: number
  status?: RecordingStatus
}

/** Paginated list of recordings for a meeting */
export interface RecordingsByMeetingResponse {
  recordings: Recording[]
  total: number
}

// ---------------------------------------------------------------------------
// Presence entities
// ---------------------------------------------------------------------------

export interface UserPresence {
  user_id: string
  status: PresenceLevel
  manual_override: boolean
  last_activity: string | null
}

export interface PresenceConfig {
  away_timeout_seconds: number
}

// ---------------------------------------------------------------------------
// Reaction entities
// ---------------------------------------------------------------------------

export interface Reaction {
  message_id: string
  user_id: string
  emoji: string
  created_at: string
}

export interface ReactionSummary {
  message_id: string
  emoji: string
  count: number
  user_ids: string[]
  current_user_reacted: boolean
}

export interface ToggleReactionRequest {
  message_id: string
  emoji: string
}

export interface ToggleReactionResponse {
  reactions: ReactionSummary[]
  added: boolean
}

/** Shape returned by GET /api/v1/messages/:id/reactions */
export interface ReactionItem {
  message_id: string
  user_id: string
  emoji: string
  created_at: string
  first_name?: string
  last_name?: string
}

export interface ListReactionsApiResponse {
  reactions: ReactionItem[]
}

/** Shape returned by POST /api/v1/messages/reactions/summary */
export interface ReactionSummaryApiResponse {
  summaries: ReactionSummary[]
}

// ---------------------------------------------------------------------------
// Meeting chat entities (Wave 2)
// ---------------------------------------------------------------------------

/** A persisted chat message for a meeting (POST/GET /api/v1/meetings/{id}/chat). */
export interface MeetingChatMessage {
  id: string
  meeting_id: string
  sender_id: string
  /** Display hint — authoritative identity is sender_id; use this only as fallback. */
  sender_name: string
  message: string
  /** ISO-8601 string */
  created_at: string
}

export interface MeetingChatHistoryResponse {
  messages: MeetingChatMessage[]
}

export interface SendMeetingChatMessageRequest {
  message: string
  sender_name: string
}

// ---------------------------------------------------------------------------
// Incoming call data (for WebSocket push)
// ---------------------------------------------------------------------------

export interface IncomingCallData {
  callId: string
  callerName: string
  callerAvatar?: string
  callType: CallType
}

// ---------------------------------------------------------------------------
// Host / moderation entities (Wave 3)
// ---------------------------------------------------------------------------

/** Response for GET /meetings/{meetingId}/cohosts */
export interface MeetingCoHostsResponse {
  user_ids: string[]
}

/** Request body for POST /meetings/{meetingId}/cohosts */
export interface PromoteCoHostRequest {
  user_id: string
}

/** Request body for POST /meetings/{meetingId}/moderation/mute */
export interface MuteParticipantRequest {
  target_user_id: string
}

/** Request body for POST /meetings/{meetingId}/moderation/kick */
export interface KickParticipantRequest {
  target_user_id: string
}

/** Request body for POST /meetings/{meetingId}/lock */
export interface SetMeetingLockRequest {
  locked: boolean
}

// ---------------------------------------------------------------------------
// Breakout room entities (Wave 6A)
// ---------------------------------------------------------------------------

export type BreakoutRoomStatus = 'open' | 'closed'

/** A LiveKit sub-room created by the host during a meeting. */
export interface BreakoutRoom {
  id: string
  meeting_id: string
  /** LiveKit room identifier ("breakout-{meetingID}-{index}"). */
  room_name: string
  label: string
  sort_index: number
  status: BreakoutRoomStatus
  created_by: string
  closed_at: string | null
  created_at: string
}

/** Maps a participant to their assigned sub-room (host overview). */
export interface BreakoutAssignmentInfo {
  user_id: string
  breakout_room_id: string
}

/** Request body for POST /meetings/{id}/breakout-rooms */
export interface CreateBreakoutRoomsRequest {
  count: number
  labels?: string[]
}

/** Response for GET /meetings/{id}/breakout-rooms (host overview). */
export interface ListBreakoutRoomsResponse {
  rooms: BreakoutRoom[]
  assignments: BreakoutAssignmentInfo[]
}

/** Request body for POST /meetings/{id}/breakout-rooms/assign */
export interface AssignBreakoutParticipantRequest {
  target_user_id: string
  /** null = move the participant back to the main room. */
  breakout_room_id: string | null
}

/** Response for POST /meetings/{id}/breakout-rooms/join — token for the sub-room. */
export interface JoinBreakoutRoomResponse {
  token: string
  room_name: string
  ws_url: string
  /** Per-session TURN credentials; empty/absent when TURN is not configured. */
  ice_servers?: IceServer[]
  breakout_room: BreakoutRoom
}

/** Response for GET /meetings/{id}/breakout-rooms/assignment — caller's own room. */
export interface BreakoutAssignmentResponse {
  /** null = the caller is currently in the main room. */
  room: BreakoutRoom | null
}

/** Request body for POST /meetings/{id}/breakout-rooms/return */
export interface ReturnToMainRoomRequest {
  /** Defaults to the caller; a host may pull another participant back. */
  target_user_id?: string
}
