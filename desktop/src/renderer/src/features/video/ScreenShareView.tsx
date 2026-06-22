/**
 * ScreenShareView -- Layout when someone is screen sharing.
 *
 * Displays the shared screen as the main area (maintaining aspect ratio)
 * with a vertical sidebar of participant video thumbnails.
 *
 * When screen sharing ends the parent (VideoCallView) reverts to
 * the previous view mode (gallery or speaker).
 */
import { useTranslation } from 'react-i18next'
import {
  VideoTrack,
  ParticipantTile,
  type TrackReferenceOrPlaceholder,
} from '@livekit/components-react'

import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ScreenShareViewProps {
  /** The screen share track reference to display in the main area. */
  screenShareTrack: TrackReferenceOrPlaceholder
  /** Camera tracks for all participants (displayed as thumbnails). */
  participantTracks: TrackReferenceOrPlaceholder[]
  /** Optional CSS class for the container. */
  className?: string
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ScreenShareView({
  screenShareTrack,
  participantTracks,
  className,
}: ScreenShareViewProps) {
  const { t } = useTranslation()
  const sharerName =
    screenShareTrack.participant.name || screenShareTrack.participant.identity

  return (
    <div className={cn('flex h-full w-full gap-2 p-2', className)}>
      {/* Main area: screen share */}
      <div className="flex flex-1 items-center justify-center overflow-hidden rounded-lg bg-black">
        <VideoTrack
          trackRef={screenShareTrack}
          className="h-full w-full object-contain"
        />
        <div className="absolute bottom-3 left-3 rounded bg-black/60 px-2 py-1 text-xs text-white backdrop-blur-sm">
          {t('features.video.screenShare.sharingLabel', { name: sharerName })}
        </div>
      </div>

      {/* Sidebar: participant thumbnails */}
      {participantTracks.length > 0 && (
        <div className="flex w-[200px] flex-shrink-0 flex-col gap-2 overflow-y-auto">
          {participantTracks.map((track) => (
            <div
              key={track.participant.identity + ':' + track.source}
              className="aspect-video w-full overflow-hidden rounded-lg"
            >
              <ParticipantTile trackRef={track} />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
