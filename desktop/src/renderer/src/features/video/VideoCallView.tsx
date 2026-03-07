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
 */
import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  LiveKitRoom,
  GridLayout,
  FocusLayout,
  FocusLayoutContainer,
  CarouselLayout,
  ParticipantTile,
  RoomAudioRenderer,
  ConnectionStateToast,
  useTracks,
  useParticipants,
  useLocalParticipant,
  useRoomContext,
  isTrackReference,
  type TrackReferenceOrPlaceholder,
} from '@livekit/components-react'
import { Track, RoomEvent, type Participant } from 'livekit-client'
import '@livekit/components-styles'

import { useVideoStore } from '@/stores/video'
import { CallControls } from './CallControls'
import { ScreenShareView } from './ScreenShareView'
import { cn } from '@/lib'

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
  /** Called when the local participant leaves the call. */
  onLeave?: () => void
}

type ViewMode = 'gallery' | 'speaker'

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
  const room = useRoomContext()
  const participants = useParticipants()
  const { localParticipant } = useLocalParticipant()

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
      {/* Main video area */}
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
      </div>

      {/* View toggle */}
      {!isScreenSharing && participants.length > 1 && (
        <button
          className="absolute right-4 top-4 z-30 rounded-lg bg-zinc-800/80 p-2 text-white backdrop-blur-sm transition-colors hover:bg-zinc-700"
          onClick={() =>
            setViewMode(viewMode === 'gallery' ? 'speaker' : 'gallery')
          }
          title={
            viewMode === 'gallery'
              ? 'Zur Sprecheransicht wechseln'
              : 'Zur Galerieansicht wechseln'
          }
        >
          {viewMode === 'gallery' ? (
            // Speaker layout icon
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
            >
              <rect x="2" y="2" width="14" height="14" rx="2" />
              <rect x="18" y="2" width="4" height="6" rx="1" />
              <rect x="18" y="10" width="4" height="6" rx="1" />
            </svg>
          ) : (
            // Grid layout icon
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
            >
              <rect x="2" y="2" width="9" height="9" rx="2" />
              <rect x="13" y="2" width="9" height="9" rx="2" />
              <rect x="2" y="13" width="9" height="9" rx="2" />
              <rect x="13" y="13" width="9" height="9" rx="2" />
            </svg>
          )}
        </button>
      )}

      {/* Call controls bar */}
      <CallControls callId={callId} onLeave={onLeave} />

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

  return (
    <LiveKitRoom
      token={token}
      serverUrl={wsUrl}
      connect={true}
      audio={true}
      video={true}
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
