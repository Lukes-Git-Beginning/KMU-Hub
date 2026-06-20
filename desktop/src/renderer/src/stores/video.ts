/**
 * Video call state store (Zustand, ephemeral -- not persisted).
 *
 * Manages the active call session, LiveKit connection details,
 * floating call bar visibility, and incoming call notifications.
 * Call state is inherently transient and should not survive app restart.
 */
import { create } from 'zustand'
import type { CallSession, IceServer, IncomingCallData } from '@/api/video-types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

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

  // Incoming call
  incomingCall: IncomingCallData | null

  // Actions
  setActiveCall: (call: CallSession, token: string, wsUrl: string, iceServers?: IceServer[]) => void
  clearActiveCall: () => void
  setFloatingBarVisible: (visible: boolean) => void
  setCallDuration: (seconds: number) => void
  setIncomingCall: (call: IncomingCallData | null) => void
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
    }),

  setFloatingBarVisible: (visible: boolean) =>
    set({ isFloatingBarVisible: visible }),

  setCallDuration: (seconds: number) =>
    set({ callDuration: seconds }),

  setIncomingCall: (call: IncomingCallData | null) =>
    set({ incomingCall: call }),
}))
