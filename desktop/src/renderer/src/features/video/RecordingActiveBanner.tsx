/**
 * RecordingActiveBanner -- Fixed top-stripe shown while a recording is active.
 *
 * Uses useIsRecording from LiveKit to detect active recording state.
 * Renders a red persistent banner with a REC indicator and an optional stop button.
 * The stop button is only shown when an activeRecordingId is provided (i.e. the local
 * user started the recording and can stop it).
 *
 * Design: transform/opacity only for motion (no layout animations per motion rules).
 */
import { useCallback } from 'react'
import { useIsRecording } from '@livekit/components-react'
import { useTranslation } from 'react-i18next'

import { useStopRecording } from '@/api/hooks/useVideo'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface RecordingActiveBannerProps {
  /** Recording ID returned by startRecording — null if the local user did not start it. */
  activeRecordingId: string | null
  /** Called after the recording is successfully stopped. */
  onStop?: () => void
  /** Optional CSS class for the container. */
  className?: string
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function RecordingActiveBanner({
  activeRecordingId,
  onStop,
  className,
}: RecordingActiveBannerProps) {
  const { t } = useTranslation()
  const isRecording = useIsRecording()
  const stopRecording = useStopRecording()

  const handleStop = useCallback(async () => {
    if (!activeRecordingId) return
    try {
      await stopRecording.mutateAsync(activeRecordingId)
      onStop?.()
    } catch {
      // Error is handled by TanStack Query error boundary
    }
  }, [activeRecordingId, stopRecording, onStop])

  if (!isRecording) return null

  return (
    <div
      role="status"
      aria-label={t('features.video.recordingBanner.title')}
      className={cn(
        'flex items-center justify-between gap-3 bg-red-600 px-4 py-1.5 text-sm font-medium text-white',
        className,
      )}
    >
      <div className="flex items-center gap-2">
        {/* REC indicator — pulsing dot */}
        <span
          aria-hidden="true"
          className="h-2.5 w-2.5 animate-pulse rounded-full bg-white"
        />
        <span className="tracking-wide uppercase text-xs">
          {t('features.video.recordingBanner.status')}
        </span>
      </div>

      {activeRecordingId && (
        <button
          onClick={handleStop}
          disabled={stopRecording.isPending}
          className="rounded px-2 py-0.5 text-xs font-semibold uppercase tracking-wide transition-opacity hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white disabled:opacity-50"
          aria-label={t('features.video.recordingBanner.stop')}
        >
          {t('features.video.recordingBanner.stop')}
        </button>
      )}
    </div>
  )
}
