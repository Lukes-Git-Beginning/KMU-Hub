/**
 * VideoCallView -- Main video call view with gallery/speaker toggle.
 *
 * Wraps @livekit/components-react LiveKitRoom and provides two layout modes:
 *   - Gallery: Equal-sized participant tiles in a responsive CSS grid.
 *   - Speaker: One large "focused" participant + sidebar thumbnails.
 *
 * When a screen share track is detected the view automatically switches
 * to ScreenShareView (shared screen as main area + sidebar thumbnails).
 *
 * Clicking a participant in gallery mode switches to speaker view with
 * that participant focused. Active-speaker events also update the focus.
 *
 * Wave 1 additions:
 *   1.2 In-call device selection (mic/cam/speaker) via MediaDeviceMenu
 *   1.3 Live participant roster via useParticipants (real join/leave + mic/cam status)
 *   1.4 Connection quality indicator per roster entry via useConnectionQualityIndicator
 *   1.5 Room controls (mic/cam toggle callbacks + state) injected into video store
 *       so FloatingCallBar can toggle without needing LiveKitRoom context
 *
 * Wave 3 additions:
 *   3.1 Host-Controls (server-authoritative): mute, kick, co-host, mute-all, lock
 *       Visible only to the meeting host (organizer or co-host).
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  LiveKitRoom,
  GridLayout,
  FocusLayout,
  FocusLayoutContainer,
  CarouselLayout,
  ParticipantTile,
  RoomAudioRenderer,
  ConnectionStateToast,
  MediaDeviceMenu,
  useTracks,
  useParticipants,
  useLocalParticipant,
  useRoomContext,
  useConnectionQualityIndicator,
  useDataChannel,
  isTrackReference,
  type TrackReferenceOrPlaceholder,
} from '@livekit/components-react'
import { Track, RoomEvent, ConnectionQuality, type Participant } from 'livekit-client'
import '@livekit/components-styles'

import { useVideoStore } from '@/stores/video'
import type { IceServer } from '@/api/video-types'
import { CallControls } from './CallControls'
import { RecordingActiveBanner } from './RecordingActiveBanner'
import { ScreenShareView } from './ScreenShareView'
import { ScreenSourcePicker } from './ScreenSourcePicker'
import { BackgroundSelector } from './BackgroundSelector'
import { InCallChat } from './InCallChat'
import { ReactionLayer, ReactionPicker, useReactionChannel } from './ReactionLayer'
import { useAuthStore } from '@/stores/auth'
import { useMeeting } from '@/api/hooks/useMeetings'
import { useMeetingCoHosts, useActiveMeetingRecording } from '@/api/hooks/useVideo'
import {
  ParticipantActions,
  GlobalHostControls,
  MeetingLockIndicator,
} from './HostControls'
import { RecordingConsentDialog } from './RecordingConsentDialog'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Hand-raise DataChannel topic & payload
// ---------------------------------------------------------------------------

const HAND_TOPIC = 'cosmi-hand'

interface HandPayload {
  type: 'hand'
  raised: boolean
  identity: string
}

// Public STUN fallback used alongside any self-hosted TURN servers so plain NAT
// traversal still works when TURN is not configured.
const DEFAULT_ICE_SERVERS: RTCIceServer[] = [{ urls: 'stun:stun.l.google.com:19302' }]

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface VideoCallViewProps {
  /** ID of the active call (used for API operations). */
  callId: string
  /** LiveKit access token for this participant. */
  token: string
  /** LiveKit server WebSocket URL. */
  wsUrl: string
  /** Per-session TURN servers from the join response; applied to RTCConfiguration. */
  iceServers?: IceServer[]
  /** Called when the local participant leaves the call. */
  onLeave?: () => void
  /**
   * Meeting ID (distinct from callId for meeting-based calls).
   * Required to enable host-controls (co-host list, moderation routes).
   * When absent, host-controls are hidden.
   */
  meetingId?: string
}

type ViewMode = 'gallery' | 'speaker'

// ---------------------------------------------------------------------------
// Connection quality SVG indicator (no emoji, custom SVG per design rules)
// ---------------------------------------------------------------------------

interface ConnectionQualityIconProps {
  participant: Participant
  className?: string
}

function ConnectionQualityIcon({ participant, className }: ConnectionQualityIconProps) {
  const { t } = useTranslation()
  const { quality } = useConnectionQualityIndicator({ participant })

  const qualityKey =
    quality === ConnectionQuality.Excellent
      ? 'excellent'
      : quality === ConnectionQuality.Good
        ? 'good'
        : quality === ConnectionQuality.Poor
          ? 'poor'
          : quality === ConnectionQuality.Lost
            ? 'lost'
            : 'unknown'

  const title = t(`features.video.connectionQuality.${qualityKey}`)

  const barColor =
    quality === ConnectionQuality.Excellent
      ? 'text-emerald-400'
      : quality === ConnectionQuality.Good
        ? 'text-yellow-400'
        : quality === ConnectionQuality.Poor
          ? 'text-orange-500'
          : 'text-red-500'

  if (quality === ConnectionQuality.Lost || quality === ConnectionQuality.Unknown) {
    return (
      <span
        className={cn('inline-flex items-center', barColor, className)}
        title={title}
        aria-label={title}
      >
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
          <line x1="1.5" y1="1.5" x2="10.5" y2="10.5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
          <line x1="10.5" y1="1.5" x2="1.5" y2="10.5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
        </svg>
      </span>
    )
  }

  const bars = [
    { height: 4, active: true },
    {
      height: 7,
      active:
        quality === ConnectionQuality.Good || quality === ConnectionQuality.Excellent,
    },
    { height: 10, active: quality === ConnectionQuality.Excellent },
  ]

  return (
    <span
      className={cn('inline-flex items-end gap-[2px]', barColor, className)}
      title={title}
      aria-label={title}
    >
      <svg width="13" height="11" viewBox="0 0 13 11" fill="none" aria-hidden="true">
        {bars.map((bar, i) => (
          <rect
            key={i}
            x={i * 4 + 0.5}
            y={11 - bar.height}
            width="3"
            height={bar.height}
            rx="0.75"
            fill="currentColor"
            opacity={bar.active ? 1 : 0.22}
          />
        ))}
      </svg>
    </span>
  )
}

// ---------------------------------------------------------------------------
// Participant roster sidebar (1.3 + 1.4)
// ---------------------------------------------------------------------------

interface ParticipantRosterProps {
  participants: Participant[]
  localParticipant: Participant
  raisedHands?: Map<string, boolean>
  onClose: () => void
  /** Host-control props -- undefined when the local user is not a host. */
  hostControls?: {
    meetingId: string
    isOrganizer: boolean
    coHosts: string[]
  }
}

function ParticipantRoster({ participants, localParticipant, raisedHands, onClose, hostControls }: ParticipantRosterProps) {
  const { t } = useTranslation()

  return (
    <div className="flex w-64 flex-shrink-0 flex-col border-l border-zinc-800 bg-zinc-900/95">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-800 px-3 py-2.5">
        <span className="text-sm font-medium text-zinc-200">
          {t('features.video.roster.title')} ({participants.length})
        </span>
        <button
          onClick={onClose}
          className="flex h-6 w-6 items-center justify-center rounded text-zinc-400 transition-colors hover:bg-zinc-700 hover:text-zinc-200"
          aria-label="Close"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
            <line x1="1" y1="1" x2="11" y2="11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <line x1="11" y1="1" x2="1" y2="11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
          </svg>
        </button>
      </div>

      {/* Participant list */}
      <div className="flex-1 overflow-y-auto py-1">
        {participants.map((p) => {
          const isLocal = p.identity === localParticipant.identity
          const isMicOn = p.isMicrophoneEnabled
          const isCamOn = p.isCameraEnabled

          const isHandRaised = raisedHands?.get(p.identity) ?? false

          return (
            <div
              key={p.identity}
              className="flex items-center gap-2.5 px-3 py-2 hover:bg-zinc-800/60"
            >
              {/* Avatar initials */}
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-zinc-700 text-xs font-medium text-zinc-200">
                {(p.name || p.identity).slice(0, 2).toUpperCase()}
              </div>

              {/* Name */}
              <span className="flex-1 truncate text-sm text-zinc-200">
                {p.name || p.identity}
                {isLocal && (
                  <span className="ml-1.5 text-xs text-zinc-500">
                    ({t('features.video.roster.you')})
                  </span>
                )}
                {!isLocal && (hostControls?.coHosts.includes(p.identity) ?? false) && (
                  <span className="ml-1.5 text-xs text-violet-400">
                    {t('features.video.hostControls.coHostBadge')}
                  </span>
                )}
              </span>

              {/* Status icons */}
              <div className="flex shrink-0 items-center gap-1.5">
                {/* Hand raised indicator (Wave 2) */}
                {isHandRaised && (
                  <span
                    className="text-amber-400"
                    title={t('features.video.handRaise.raised')}
                    aria-label={t('features.video.handRaise.raised')}
                  >
                    {/* Hand SVG (no emoji in UI chrome) */}
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                      <path d="M18 11V6a2 2 0 0 0-2-2 2 2 0 0 0-2 2" />
                      <path d="M14 10V4a2 2 0 0 0-2-2 2 2 0 0 0-2 2v2" />
                      <path d="M10 10.5V6a2 2 0 0 0-2-2 2 2 0 0 0-2 2v8" />
                      <path d="M18 8a2 2 0 1 1 4 0v6a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15" />
                    </svg>
                  </span>
                )}

                {/* Mic */}
                <span
                  className={isMicOn ? 'text-zinc-500' : 'text-red-500'}
                  title={t(
                    isMicOn
                      ? 'features.video.roster.micOn'
                      : 'features.video.roster.micOff',
                  )}
                >
                  {isMicOn ? (
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                      <path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z" />
                      <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
                      <line x1="12" x2="12" y1="19" y2="22" />
                    </svg>
                  ) : (
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                      <line x1="2" x2="22" y1="2" y2="22" />
                      <path d="M18.89 13.23A7.12 7.12 0 0 0 19 12v-2" />
                      <path d="M5 10v2a7 7 0 0 0 12 5.29" />
                      <path d="M15 9.34V5a3 3 0 0 0-5.68-1.33" />
                      <path d="M9 9v3a3 3 0 0 0 5.12 2.12" />
                      <line x1="12" x2="12" y1="19" y2="22" />
                    </svg>
                  )}
                </span>

                {/* Camera */}
                <span
                  className={isCamOn ? 'text-zinc-500' : 'text-red-500'}
                  title={t(
                    isCamOn
                      ? 'features.video.roster.camOn'
                      : 'features.video.roster.camOff',
                  )}
                >
                  {isCamOn ? (
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                      <path d="m16 13 5.223 3.482a.5.5 0 0 0 .777-.416V7.87a.5.5 0 0 0-.752-.432L16 10.5" />
                      <rect x="2" y="6" width="14" height="12" rx="2" />
                    </svg>
                  ) : (
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                      <line x1="2" x2="22" y1="2" y2="22" />
                      <path d="M21 17.31V7.87a.5.5 0 0 0-.752-.432L16 10.5" />
                      <path d="M11.19 6H14a2 2 0 0 1 2 2v3.19" />
                      <path d="M2 8a2 2 0 0 1 2-2h0M16 16a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V8" />
                    </svg>
                  )}
                </span>

                {/* Connection quality (1.4) */}
                <ConnectionQualityIcon participant={p} />

                {/* Host-controls actions (Wave 3): only for other participants when local is host */}
                {hostControls && !isLocal && (
                  <ParticipantActions
                    participantIdentity={p.identity}
                    meetingId={hostControls.meetingId}
                    isOrganizer={hostControls.isOrganizer}
                    isCoHost={hostControls.coHosts.includes(p.identity)}
                  />
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Device selector popover (1.2)
// ---------------------------------------------------------------------------

interface DeviceSelectorProps {
  onClose: () => void
}

function DeviceSelector({ onClose }: DeviceSelectorProps) {
  const { t } = useTranslation()
  const ref = useRef<HTMLDivElement>(null)
  const videoPreviewRef = useRef<HTMLVideoElement>(null)
  const { localParticipant } = useLocalParticipant()

  // Close on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose()
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [onClose])

  // Attach the active camera track to the preview video element.
  // The camera is already running inside LiveKitRoom, so MediaStream is available.
  // This also ensures enumerateDevices returns labels (permission already granted).
  useEffect(() => {
    const el = videoPreviewRef.current
    if (!el) return
    const camPub = localParticipant.getTrackPublication(Track.Source.Camera)
    const track = camPub?.track?.mediaStreamTrack
    if (!track) return
    const stream = new MediaStream([track])
    el.srcObject = stream
    el.play().catch(() => { /* autoplay guard — safe to ignore */ })
    return () => {
      el.srcObject = null
    }
  }, [localParticipant])

  return (
    <div
      ref={ref}
      className="absolute bottom-full left-1/2 z-40 mb-2 w-72 -translate-x-1/2 rounded-xl border border-zinc-700 bg-zinc-900 p-4 shadow-xl"
    >
      {/* Camera live preview thumbnail */}
      <div className="mb-3 overflow-hidden rounded-lg bg-zinc-950">
        <video
          ref={videoPreviewRef}
          muted
          autoPlay
          playsInline
          className="h-[108px] w-full object-cover"
          aria-label={t('features.video.deviceSelect.camera')}
        />
      </div>

      <div className="space-y-3">
        {/* Microphone selector */}
        <div>
          <p className="mb-1.5 text-xs font-medium text-zinc-400">
            {t('features.video.deviceSelect.microphone')}
          </p>
          <MediaDeviceMenu
            kind="audioinput"
            className="lk-button w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 text-left text-sm text-zinc-200 hover:bg-zinc-700"
          />
        </div>

        {/* Camera selector */}
        <div>
          <p className="mb-1.5 text-xs font-medium text-zinc-400">
            {t('features.video.deviceSelect.camera')}
          </p>
          <MediaDeviceMenu
            kind="videoinput"
            className="lk-button w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 text-left text-sm text-zinc-200 hover:bg-zinc-700"
          />
        </div>

        {/* Speaker selector */}
        <div>
          <p className="mb-1.5 text-xs font-medium text-zinc-400">
            {t('features.video.deviceSelect.speaker')}
          </p>
          <MediaDeviceMenu
            kind="audiooutput"
            className="lk-button w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 text-left text-sm text-zinc-200 hover:bg-zinc-700"
          />
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Inner component (must be inside LiveKitRoom to use room hooks)
// ---------------------------------------------------------------------------

interface InnerCallViewProps {
  callId: string
  viewMode: ViewMode
  setViewMode: (mode: ViewMode) => void
  focusedParticipant: Participant | null
  setFocusedParticipant: (p: Participant | null) => void
  onLeave?: () => void
  /** Meeting ID for host-controls (Wave 3). Undefined = no host features. */
  meetingId?: string
  /** Organizer user ID derived from meeting data. */
  organizerId?: string
  /** Current authenticated user ID. */
  currentUserId?: string
  /** Co-host user IDs from the server (Wave 3). */
  coHosts: string[]
  /** Whether the room is currently locked (Wave 3). */
  isLocked: boolean
  /** ID of the currently active recording for this meeting (from useActiveMeetingRecording). */
  activeMeetingRecordingId: string | null
  /** User ID of the recording initiator (null when unknown). */
  recordingInitiatorId: string | null
}

// ---------------------------------------------------------------------------
// Hand-raise hook (synced via DataChannel)
// ---------------------------------------------------------------------------

function useHandRaise() {
  const { localParticipant } = useLocalParticipant()
  // Map of participant identity → raised state
  const [raisedHands, setRaisedHands] = useState<Map<string, boolean>>(new Map())

  const onMessage = useCallback(
    (msg: { payload: Uint8Array; from?: { identity?: string } }) => {
      try {
        const decoded = new TextDecoder().decode(msg.payload)
        const parsed = JSON.parse(decoded) as HandPayload
        if (parsed.type !== 'hand') return
        setRaisedHands((prev) => {
          const next = new Map(prev)
          if (parsed.raised) {
            next.set(parsed.identity, true)
          } else {
            next.delete(parsed.identity)
          }
          return next
        })
      } catch {
        // Ignore malformed payloads
      }
    },
    [],
  )

  const { send } = useDataChannel(HAND_TOPIC, onMessage)

  const [isLocalHandRaised, setIsLocalHandRaised] = useState(false)

  const toggleHand = useCallback(() => {
    const nextRaised = !isLocalHandRaised
    setIsLocalHandRaised(nextRaised)

    // Update local state immediately
    setRaisedHands((prev) => {
      const next = new Map(prev)
      if (nextRaised) {
        next.set(localParticipant.identity, true)
      } else {
        next.delete(localParticipant.identity)
      }
      return next
    })

    // Broadcast to peers
    const payload: HandPayload = {
      type: 'hand',
      raised: nextRaised,
      identity: localParticipant.identity,
    }
    send(new TextEncoder().encode(JSON.stringify(payload)), { reliable: true })
  }, [isLocalHandRaised, localParticipant.identity, send])

  // Clear own hand on unmount (e.g. leaving call)
  useEffect(() => {
    return () => {
      if (isLocalHandRaised) {
        const payload: HandPayload = { type: 'hand', raised: false, identity: localParticipant.identity }
        send(new TextEncoder().encode(JSON.stringify(payload)), { reliable: true })
      }
    }
    // Only run on unmount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return { raisedHands, isLocalHandRaised, toggleHand }
}

function InnerCallView({
  callId,
  viewMode,
  setViewMode,
  focusedParticipant,
  setFocusedParticipant,
  onLeave,
  meetingId,
  organizerId,
  currentUserId,
  coHosts,
  isLocked,
  activeMeetingRecordingId,
  recordingInitiatorId,
}: InnerCallViewProps) {
  const { t } = useTranslation()
  const room = useRoomContext()
  const participants = useParticipants()
  const { localParticipant, isMicrophoneEnabled, isCameraEnabled, isScreenShareEnabled } = useLocalParticipant()

  const [showRoster, setShowRoster] = useState(false)
  const [showDeviceSelector, setShowDeviceSelector] = useState(false)
  const [showChat, setShowChat] = useState(false)

  // Wave 4.1: Electron screen-source picker
  const [showScreenPicker, setShowScreenPicker] = useState(false)

  // Wave 4.3: Background blur/virtual BG selector
  const [showBackgroundSelector, setShowBackgroundSelector] = useState(false)
  const [isBackgroundActive, setIsBackgroundActive] = useState(false)

  // B2: track which recordings this user has already responded to (prevent re-opening)
  const [respondedRecordingIds, setRespondedRecordingIds] = useState<Set<string>>(new Set())
  // Derive whether to show consent dialog for a non-initiator participant
  const showInCallConsent =
    !!activeMeetingRecordingId &&
    currentUserId !== recordingInitiatorId &&
    !respondedRecordingIds.has(activeMeetingRecordingId)

  // Wave 2: hand raise (synced DataChannel)
  const { raisedHands, isLocalHandRaised, toggleHand } = useHandRaise()

  // Wave 2: emoji reactions (DataChannel floats)
  const { floats, sendReaction } = useReactionChannel()

  // Wave 3: derive host role from meeting organizer + co-host list
  const isOrganizer = !!currentUserId && currentUserId === organizerId
  const isHost = isOrganizer || (!!currentUserId && coHosts.includes(currentUserId))

  const setRoomControls = useVideoStore((s) => s.setRoomControls)

  // Inject room controls into the store so FloatingCallBar can toggle mic/cam (1.5).
  // Re-run whenever the enabled state changes so the store always reflects reality.
  useEffect(() => {
    setRoomControls({
      isMicEnabled: isMicrophoneEnabled,
      isCamEnabled: isCameraEnabled,
      toggleMic: async () => {
        await localParticipant.setMicrophoneEnabled(!localParticipant.isMicrophoneEnabled)
      },
      toggleCam: async () => {
        await localParticipant.setCameraEnabled(!localParticipant.isCameraEnabled)
      },
    })
    return () => {
      setRoomControls(null)
    }
  }, [localParticipant, isMicrophoneEnabled, isCameraEnabled, setRoomControls])

  // All camera + screen share tracks
  const tracks = useTracks(
    [
      { source: Track.Source.Camera, withPlaceholder: true },
      { source: Track.Source.ScreenShare, withPlaceholder: false },
    ],
    { onlySubscribed: false },
  )

  // Detect screen share tracks
  const screenShareTracks = useMemo(
    () =>
      tracks.filter(
        (t) =>
          t.source === Track.Source.ScreenShare && isTrackReference(t),
      ),
    [tracks],
  )

  const cameraTracks = useMemo(
    () => tracks.filter((t) => t.source === Track.Source.Camera),
    [tracks],
  )

  const isScreenSharing = screenShareTracks.length > 0

  // Active speaker detection -- auto-switch to speaker view
  useEffect(() => {
    const handleActiveSpeaker = (speakers: Participant[]) => {
      if (speakers.length > 0 && speakers[0] !== localParticipant) {
        setFocusedParticipant(speakers[0])
        if (viewMode === 'gallery' && participants.length > 2) {
          setViewMode('speaker')
        }
      }
    }

    room.on(RoomEvent.ActiveSpeakersChanged, handleActiveSpeaker)
    return () => {
      room.off(RoomEvent.ActiveSpeakersChanged, handleActiveSpeaker)
    }
  }, [room, localParticipant, setFocusedParticipant, setViewMode, viewMode, participants.length])

  // Handle participant click to focus in speaker mode
  const handleParticipantClick = useCallback(
    (evt: { participant: Participant }) => {
      setFocusedParticipant(evt.participant)
      setViewMode('speaker')
    },
    [setFocusedParticipant, setViewMode],
  )

  // Wave 4.1: Screen-share picker handlers
  // Called by CallControls when screen-share button is pressed while not sharing.
  const handleScreenShareButtonClick = useCallback(() => {
    setShowScreenPicker(true)
  }, [])

  // Called by ScreenSourcePicker once the user confirms a source.
  // LiveKit ScreenShareCaptureOptions.sourceId tells livekit-client to pass
  // chromeMediaSourceId in the video constraint, which Electron's
  // setDisplayMediaRequestHandler will receive and map to the right source.
  // We also call window.electronAPI.screenshare.setSource() to cache the id
  // in the main process before getDisplayMedia fires — avoids the async race
  // inside setDisplayMediaRequestHandler on some Windows configurations.
  const handleSourceSelected = useCallback(
    async (sourceId: string) => {
      setShowScreenPicker(false)
      try {
        if (sourceId) {
          // Electron path: cache the chosen id in the main process first (when the
          // newer preload exposes setSource) so setDisplayMediaRequestHandler can
          // resolve it synchronously, then start sharing.
          if (typeof window !== 'undefined' && window.electronAPI?.screenshare?.setSource) {
            await window.electronAPI.screenshare.setSource(sourceId)
          }
          await localParticipant.setScreenShareEnabled(true, { audio: true, sourceId })
        } else {
          // Non-Electron path (web fallback): browser's native picker
          await localParticipant.setScreenShareEnabled(true, { audio: true })
        }
      } catch {
        // User may have cancelled the native dialog or no source found — safe to ignore
      }
    },
    [localParticipant],
  )

  const handleScreenPickerCancel = useCallback(() => {
    setShowScreenPicker(false)
  }, [])

  // Wave 4.3: Background selector
  const handleOpenBackgroundSelector = useCallback(() => {
    setShowBackgroundSelector((v) => !v)
    // Collapse other popovers for clean UX
    setShowDeviceSelector(false)
  }, [])

  // Called by BackgroundSelector to close itself (outside click)
  const handleBackgroundSelectorClose = useCallback(() => {
    setShowBackgroundSelector(false)
  }, [])

  // Find the focused track reference for speaker view
  const focusedTrack = useMemo((): TrackReferenceOrPlaceholder | undefined => {
    if (!focusedParticipant) {
      // Default to first remote participant or first in list
      const remote = cameraTracks.find(
        (t) => t.participant !== localParticipant,
      )
      return remote ?? cameraTracks[0]
    }
    return cameraTracks.find(
      (t) => t.participant.identity === focusedParticipant.identity,
    )
  }, [focusedParticipant, cameraTracks, localParticipant])

  // Carousel tracks (everyone except focused)
  const carouselTracks = useMemo(() => {
    if (!focusedTrack) return cameraTracks
    return cameraTracks.filter(
      (t) => t.participant.identity !== focusedTrack.participant.identity,
    )
  }, [cameraTracks, focusedTrack])

  // Dynamic grid columns based on participant count
  const gridColumns = useMemo(() => {
    const count = cameraTracks.length
    if (count <= 1) return 'grid-cols-1'
    if (count <= 2) return 'grid-cols-2'
    if (count <= 4) return 'grid-cols-2'
    if (count <= 9) return 'grid-cols-3'
    if (count <= 16) return 'grid-cols-4'
    return 'grid-cols-5'
  }, [cameraTracks.length])

  const localDisplayName = localParticipant.name ?? localParticipant.identity

  // Host-controls props for the roster (only defined when local user is host)
  const rosterHostControls = (isHost && meetingId)
    ? { meetingId, isOrganizer, coHosts }
    : undefined

  return (
    <div className="relative flex h-full w-full flex-col bg-zinc-950">
      {/* DSGVO: visible recording indicator for every participant. Self-renders
          only while a recording is active (useIsRecording). The stop control
          lives in CallControls, so no activeRecordingId is needed here. */}
      <RecordingActiveBanner activeRecordingId={activeMeetingRecordingId} />

      {/* Wave 3: Global host controls bar (mute-all, lock) + lock indicator */}
      {isHost && meetingId && (
        <div className="flex items-center justify-between border-b border-zinc-800/60 bg-zinc-900/80 px-4 py-1.5">
          <MeetingLockIndicator isLocked={isLocked} />
          <GlobalHostControls meetingId={meetingId} isLocked={isLocked} />
        </div>
      )}

      {/* Main content: video area + optional roster/chat sidebar */}
      <div className="relative flex flex-1 overflow-hidden">
        {/* Video area (reaction floats are overlaid here) */}
        <div className="relative flex-1 overflow-hidden">
          {/* Wave 2: Reaction float overlay */}
          <ReactionLayer floats={floats} />
          {isScreenSharing ? (
            <ScreenShareView
              screenShareTrack={screenShareTracks[0] as TrackReferenceOrPlaceholder}
              participantTracks={cameraTracks}
            />
          ) : viewMode === 'speaker' && focusedTrack ? (
            <FocusLayoutContainer className="h-full">
              <CarouselLayout
                tracks={carouselTracks}
                orientation="vertical"
                className="h-full w-[200px]"
              >
                <ParticipantTile
                  onParticipantClick={handleParticipantClick}
                />
              </CarouselLayout>
              <FocusLayout
                trackRef={focusedTrack}
                className="flex-1"
              />
            </FocusLayoutContainer>
          ) : (
            <GridLayout
              tracks={cameraTracks}
              className={cn('h-full', gridColumns)}
            >
              <ParticipantTile
                onParticipantClick={handleParticipantClick}
              />
            </GridLayout>
          )}

          {/* View toggle (top-right) */}
          {!isScreenSharing && participants.length > 1 && (
            <button
              className="absolute right-4 top-4 z-30 rounded-lg bg-zinc-800/80 p-2 text-white backdrop-blur-sm transition-colors hover:bg-zinc-700"
              onClick={() =>
                setViewMode(viewMode === 'gallery' ? 'speaker' : 'gallery')
              }
              title={
                viewMode === 'gallery'
                  ? t('features.video.viewToggle.toSpeaker')
                  : t('features.video.viewToggle.toGallery')
              }
            >
              {viewMode === 'gallery' ? (
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  aria-hidden="true"
                >
                  <rect x="2" y="2" width="14" height="14" rx="2" />
                  <rect x="18" y="2" width="4" height="6" rx="1" />
                  <rect x="18" y="10" width="4" height="6" rx="1" />
                </svg>
              ) : (
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  aria-hidden="true"
                >
                  <rect x="2" y="2" width="9" height="9" rx="2" />
                  <rect x="13" y="2" width="9" height="9" rx="2" />
                  <rect x="2" y="13" width="9" height="9" rx="2" />
                  <rect x="13" y="13" width="9" height="9" rx="2" />
                </svg>
              )}
            </button>
          )}

          {/* Roster toggle button (top-left) */}
          <button
            className={cn(
              'absolute left-4 top-4 z-30 rounded-lg p-2 backdrop-blur-sm transition-colors',
              showRoster
                ? 'bg-primary text-white'
                : 'bg-zinc-800/80 text-white hover:bg-zinc-700',
            )}
            onClick={() => {
              setShowRoster((v) => !v)
              // Collapse chat when opening roster (and vice versa for clean sidebar)
              if (!showRoster) setShowChat(false)
            }}
            title={t('features.video.roster.title')}
            aria-label={t('features.video.roster.title')}
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
              <circle cx="9" cy="7" r="4" />
              <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
              <path d="M16 3.13a4 4 0 0 1 0 7.75" />
            </svg>
          </button>

          {/* Chat toggle button (top-left, offset from roster) */}
          <button
            className={cn(
              'absolute left-16 top-4 z-30 rounded-lg p-2 backdrop-blur-sm transition-colors',
              showChat
                ? 'bg-primary text-white'
                : 'bg-zinc-800/80 text-white hover:bg-zinc-700',
            )}
            onClick={() => {
              setShowChat((v) => !v)
              if (!showChat) setShowRoster(false)
            }}
            title={t('features.video.chat.title')}
            aria-label={t('features.video.chat.title')}
          >
            {/* Chat bubble SVG */}
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
            </svg>
          </button>
        </div>

        {/* Live participant roster sidebar (1.3) */}
        {showRoster && (
          <ParticipantRoster
            participants={participants}
            localParticipant={localParticipant}
            raisedHands={raisedHands}
            onClose={() => setShowRoster(false)}
            hostControls={rosterHostControls}
          />
        )}

        {/* In-call chat panel (Wave 2) */}
        {showChat && (
          <InCallChat
            meetingId={callId}
            localDisplayName={localDisplayName}
            onClose={() => setShowChat(false)}
          />
        )}
      </div>

      {/* Wave 4.1: Screen source picker modal (portaled above everything) */}
      <ScreenSourcePicker
        open={showScreenPicker}
        onSelect={handleSourceSelected}
        onCancel={handleScreenPickerCancel}
      />

      {/* Call controls bar -- device selector / background selector opens above it */}
      <div className="relative">
        {showDeviceSelector && (
          <DeviceSelector onClose={() => setShowDeviceSelector(false)} />
        )}
        {/* Wave 4.3: Background selector popover */}
        {showBackgroundSelector && (
          <BackgroundSelector
            onClose={handleBackgroundSelectorClose}
            onModeChange={(m) => setIsBackgroundActive(m !== 'off')}
          />
        )}
        <CallControls
          callId={callId}
          meetingId={meetingId}
          onLeave={onLeave}
          onOpenDeviceSelector={() => setShowDeviceSelector((v) => !v)}
          onScreenShareClick={!isScreenShareEnabled ? handleScreenShareButtonClick : undefined}
          onOpenBackgroundSelector={handleOpenBackgroundSelector}
          isBackgroundActive={isBackgroundActive}
          leadingControls={
            <>
              {/* Wave 2: Hand-raise button */}
              <button
                className={cn(
                  'flex h-11 w-11 items-center justify-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-400 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-900',
                  isLocalHandRaised
                    ? 'bg-amber-500/20 text-amber-400 hover:bg-amber-500/30'
                    : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-white',
                )}
                onClick={toggleHand}
                title={t(isLocalHandRaised ? 'features.video.handRaise.lower' : 'features.video.handRaise.raise')}
                aria-label={t(isLocalHandRaised ? 'features.video.handRaise.lower' : 'features.video.handRaise.raise')}
              >
                {/* Hand SVG (no emoji in UI chrome) */}
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M18 11V6a2 2 0 0 0-2-2 2 2 0 0 0-2 2" />
                  <path d="M14 10V4a2 2 0 0 0-2-2 2 2 0 0 0-2 2v2" />
                  <path d="M10 10.5V6a2 2 0 0 0-2-2 2 2 0 0 0-2 2v8" />
                  <path d="M18 8a2 2 0 1 1 4 0v6a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15" />
                </svg>
              </button>

              {/* Wave 2: Reaction picker */}
              <ReactionPicker onSend={sendReaction} />
            </>
          }
        />
      </div>

      {/* B2: in-call DSGVO consent dialog for non-initiator participants */}
      {showInCallConsent && activeMeetingRecordingId && (
        <RecordingConsentDialog
          recordingId={activeMeetingRecordingId}
          open={showInCallConsent}
          onRespond={() => {
            setRespondedRecordingIds((prev) => new Set([...prev, activeMeetingRecordingId]))
          }}
        />
      )}

      {/* Hidden audio renderer for all remote audio tracks */}
      <RoomAudioRenderer />
      <ConnectionStateToast />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function VideoCallView({
  callId,
  token,
  wsUrl,
  iceServers,
  onLeave,
  meetingId,
}: VideoCallViewProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('gallery')
  const [focusedParticipant, setFocusedParticipant] =
    useState<Participant | null>(null)

  // Wave 3: meeting data + co-host list for host-controls
  const currentUserId = useAuthStore((s) => s.user?.id)
  const { data: meeting } = useMeeting(meetingId ?? '')
  const { data: coHosts = [] } = useMeetingCoHosts(meetingId ?? '')

  // B3: Lock state from meeting data (backend exposes meeting.locked).
  const isLocked = meeting?.locked ?? false

  // B1/B2: Active recording for this meeting - used by CallControls and consent dialog.
  const { data: activeMeetingRecording = null } = useActiveMeetingRecording(meetingId)

  const clearActiveCall = useVideoStore((s) => s.clearActiveCall)

  const handleDisconnect = useCallback(() => {
    clearActiveCall()
    onLeave?.()
  }, [clearActiveCall, onLeave])

  // Apply self-hosted TURN servers (plus a public STUN fallback) before the peer
  // connection opens, so relaying works in NAT'd networks where STUN alone fails.
  const roomOptions = useMemo(
    () => ({
      rtcConfig: {
        iceServers: [...DEFAULT_ICE_SERVERS, ...(iceServers ?? [])],
      },
    }),
    [iceServers],
  )

  return (
    <LiveKitRoom
      token={token}
      serverUrl={wsUrl}
      connect={true}
      audio={true}
      video={true}
      options={roomOptions}
      onDisconnected={handleDisconnect}
      className="h-full w-full"
      data-lk-theme="default"
    >
      <InnerCallView
        callId={callId}
        viewMode={viewMode}
        setViewMode={setViewMode}
        focusedParticipant={focusedParticipant}
        setFocusedParticipant={setFocusedParticipant}
        onLeave={onLeave}
        meetingId={meetingId}
        organizerId={meeting?.organizer_id}
        currentUserId={currentUserId}
        coHosts={coHosts}
        isLocked={isLocked}
        activeMeetingRecordingId={activeMeetingRecording?.id ?? null}
        recordingInitiatorId={activeMeetingRecording?.started_by ?? null}
      />
    </LiveKitRoom>
  )
}
