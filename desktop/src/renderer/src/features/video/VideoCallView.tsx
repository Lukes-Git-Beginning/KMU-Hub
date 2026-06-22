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
import { cn } from '@/lib'

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
  onClose: () => void
}

function ParticipantRoster({ participants, localParticipant, onClose }: ParticipantRosterProps) {
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
              </span>

              {/* Status icons */}
              <div className="flex shrink-0 items-center gap-1.5">
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

  return (
    <div
      ref={ref}
      className="absolute bottom-full left-1/2 z-40 mb-2 w-72 -translate-x-1/2 rounded-xl border border-zinc-700 bg-zinc-900 p-4 shadow-xl"
    >
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
}

function InnerCallView({
  callId,
  viewMode,
  setViewMode,
  focusedParticipant,
  setFocusedParticipant,
  onLeave,
}: InnerCallViewProps) {
  const { t } = useTranslation()
  const room = useRoomContext()
  const participants = useParticipants()
  const { localParticipant, isMicrophoneEnabled, isCameraEnabled } = useLocalParticipant()

  const [showRoster, setShowRoster] = useState(false)
  const [showDeviceSelector, setShowDeviceSelector] = useState(false)

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

  return (
    <div className="relative flex h-full w-full flex-col bg-zinc-950">
      {/* DSGVO: visible recording indicator for every participant. Self-renders
          only while a recording is active (useIsRecording). The stop control
          lives in CallControls, so no activeRecordingId is needed here. */}
      <RecordingActiveBanner activeRecordingId={null} />

      {/* Main content: video area + optional roster sidebar */}
      <div className="relative flex flex-1 overflow-hidden">
        {/* Video area */}
        <div className="relative flex-1 overflow-hidden">
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
            onClick={() => setShowRoster((v) => !v)}
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
        </div>

        {/* Live participant roster sidebar (1.3) */}
        {showRoster && (
          <ParticipantRoster
            participants={participants}
            localParticipant={localParticipant}
            onClose={() => setShowRoster(false)}
          />
        )}
      </div>

      {/* Call controls bar -- device selector opens above it (1.2) */}
      <div className="relative">
        {showDeviceSelector && (
          <DeviceSelector onClose={() => setShowDeviceSelector(false)} />
        )}
        <CallControls
          callId={callId}
          onLeave={onLeave}
          onOpenDeviceSelector={() => setShowDeviceSelector((v) => !v)}
        />
      </div>

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
}: VideoCallViewProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('gallery')
  const [focusedParticipant, setFocusedParticipant] =
    useState<Participant | null>(null)

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
      />
    </LiveKitRoom>
  )
}
