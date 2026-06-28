/**
 * Video call state store (Zustand, ephemeral -- not persisted).
 *
 * Manages the active call session, LiveKit connection details,
 * floating call bar visibility, and incoming call notifications.
 * Call state is inherently transient and should not survive app restart.
 *
 * LiveKit room controls (mic/cam toggle) are stored here so that
 * FloatingCallBar can toggle media devices without requiring access to the
 * LiveKitRoom Provider (which lives inside VideoCallView).
 */
import { create } from 'zustand'
import type { CallSession, IceServer, IncomingCallData } from '@/api/video-types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/**
 * Minimal interface for LiveKit room media controls exposed to the store.
 * Using a plain callbacks object avoids importing livekit-client Room directly
 * into the store, keeping the store framework-agnostic.
 */
export interface LiveKitRoomControls {
  isMicEnabled: boolean
  isCamEnabled: boolean
  toggleMic: () => Promise<void>
  toggleCam: () => Promise<void>
}

interface VideoState {
  // Active call
  activeCall: CallSession | null
  activeCallToken: string | null
  livekitWsUrl: string | null
  /** Per-session TURN servers for RTCConfiguration; null when TURN is off. */
  activeCallIceServers: IceServer[] | null
  isInCall: boolean
  isFloatingBarVisible: boolean
  callDuration: number // seconds since call start

  /**
   * Label of the breakout room the local user is currently in; null when in the main room.
   * Set atomically by switchBreakoutRoom, cleared by clearActiveCall.
   */
  breakoutRoomLabel: string | null

  /**
   * LiveKit room controls injected by VideoCallView (InnerCallView) so that
   * FloatingCallBar can toggle mic/cam without needing LiveKitRoom context.
   * Cleared when the call ends.
   */
  roomControls: LiveKitRoomControls | null

  // Incoming call
  incomingCall: IncomingCallData | null

  // Actions
  setActiveCall: (call: CallSession, token: string, wsUrl: string, iceServers?: IceServer[]) => void
  clearActiveCall: () => void
  setFloatingBarVisible: (visible: boolean) => void
  setCallDuration: (seconds: number) => void
  setIncomingCall: (call: IncomingCallData | null) => void
  setRoomControls: (controls: LiveKitRoomControls | null) => void
  /**
   * Atomically update the LiveKit connection credentials for a breakout room switch.
   * Does NOT reset isInCall or FloatingBar visibility — the user stays "in the call"
   * while the room transitions. The parent component syncs from these fields to
   * re-render VideoCallView with the new token, causing LiveKit to reconnect.
   */
  switchBreakoutRoom: (token: string, wsUrl: string, iceServers: IceServer[] | null, label: string | null) => void
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useVideoStore = create<VideoState>()((set) => ({
  // Defaults
  activeCall: null,
  activeCallToken: null,
  livekitWsUrl: null,
  activeCallIceServers: null,
  isInCall: false,
  isFloatingBarVisible: false,
  callDuration: 0,
  breakoutRoomLabel: null,
  roomControls: null,
  incomingCall: null,

  setActiveCall: (call: CallSession, token: string, wsUrl: string, iceServers?: IceServer[]) =>
    set({
      activeCall: call,
      activeCallToken: token,
      livekitWsUrl: wsUrl,
      activeCallIceServers: iceServers ?? null,
      isInCall: true,
      isFloatingBarVisible: true,
      callDuration: 0,
    }),

  clearActiveCall: () =>
    set({
      activeCall: null,
      activeCallToken: null,
      livekitWsUrl: null,
      activeCallIceServers: null,
      isInCall: false,
      isFloatingBarVisible: false,
      callDuration: 0,
      breakoutRoomLabel: null,
      roomControls: null,
    }),

  setFloatingBarVisible: (visible: boolean) =>
    set({ isFloatingBarVisible: visible }),

  setCallDuration: (seconds: number) =>
    set({ callDuration: seconds }),

  setIncomingCall: (call: IncomingCallData | null) =>
    set({ incomingCall: call }),

  setRoomControls: (controls: LiveKitRoomControls | null) =>
    set({ roomControls: controls }),

  switchBreakoutRoom: (
    token: string,
    wsUrl: string,
    iceServers: IceServer[] | null,
    label: string | null,
  ) =>
    set({
      activeCallToken: token,
      livekitWsUrl: wsUrl,
      activeCallIceServers: iceServers,
      breakoutRoomLabel: label,
      // isInCall and isFloatingBarVisible intentionally NOT reset — user stays in call.
    }),
}))
