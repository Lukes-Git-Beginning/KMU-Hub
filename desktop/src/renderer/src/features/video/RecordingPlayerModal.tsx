/**
 * RecordingPlayerModal -- Playback modal for a completed meeting recording.
 *
 * Opens a native <video> element with a presigned download URL and shows
 * retention expiry information. The URL is fetched lazily on open via
 * useGetRecordingDownloadURL (Wave 5 / B4).
 *
 * Design rules:
 *   - Only transform/opacity animated (motion hardrule).
 *   - No emojis -- custom SVGs only.
 */
import { useCallback, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { useGetRecordingDownloadURL } from '@/api/hooks/useVideo'
import type { Recording } from '@/api/video-types'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatBytes(bytes: number | string | null): string {
  const n = Number(bytes)
  if (!n || n <= 0) return ''
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

function formatDurationSeconds(s: number | null): string {
  if (!s) return ''
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = Math.round(s % 60)
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
  return `${m}:${String(sec).padStart(2, '0')}`
}

/** Returns the number of full days until a given ISO timestamp. */
function daysUntil(isoDate: string | null): number | null {
  if (!isoDate) return null
  const ms = new Date(isoDate).getTime() - Date.now()
  if (ms < 0) return 0
  return Math.floor(ms / 86_400_000)
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface RecordingPlayerModalProps {
  recording: Recording | null
  open: boolean
  onClose: () => void
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function RecordingPlayerModal({ recording, open, onClose }: RecordingPlayerModalProps) {
  const { t } = useTranslation()
  const getDownloadURL = useGetRecordingDownloadURL()
  const videoRef = useRef<HTMLVideoElement>(null)

  // Fetch presigned URL when the modal opens
  useEffect(() => {
    if (!open || !recording?.id) return
    if (recording.status !== 'completed') return
    getDownloadURL.mutate(recording.id)
    // We intentionally only re-fetch when the recording id or open-state changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, recording?.id])

  // Pause and reset video when the modal closes
  useEffect(() => {
    if (!open && videoRef.current) {
      videoRef.current.pause()
      videoRef.current.currentTime = 0
    }
  }, [open])

  const handleClose = useCallback(() => {
    getDownloadURL.reset()
    onClose()
  }, [getDownloadURL, onClose])

  const retentionDays = daysUntil(recording?.retention_expires_at ?? null)
  const canPlay = recording?.status === 'completed' && !!getDownloadURL.data?.url
  const isProcessing = recording?.status === 'processing' || recording?.status === 'active'

  return (
    <AlertDialog open={open}>
      <AlertDialogContent
        className="sm:max-w-2xl p-0 overflow-hidden"
        onEscapeKeyDown={handleClose}
        onPointerDownOutside={handleClose}
      >
        {/* Header */}
        <AlertDialogHeader className="flex-row items-center justify-between gap-3 border-b border-border px-5 py-3">
          <AlertDialogTitle className="flex items-center gap-2 text-base">
            <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary/10">
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="text-primary"
                aria-hidden="true"
              >
                <polygon points="5 3 19 12 5 21 5 3" fill="currentColor" stroke="none" />
              </svg>
            </span>
            {t('features.video.recordingPlayer.title')}
          </AlertDialogTitle>
          <button
            onClick={handleClose}
            className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
            aria-label={t('common.close')}
          >
            <svg width="14" height="14" viewBox="0 0 12 12" fill="none" aria-hidden="true">
              <line x1="1" y1="1" x2="11" y2="11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
              <line x1="11" y1="1" x2="1" y2="11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          </button>
        </AlertDialogHeader>

        {/* Video player area */}
        <div className="bg-black">
          {canPlay ? (
            <video
              ref={videoRef}
              src={getDownloadURL.data!.url}
              controls
              autoPlay={false}
              className="h-auto max-h-[420px] w-full"
              aria-label={t('features.video.recordingPlayer.videoLabel')}
            />
          ) : (
            <div className="flex h-56 w-full flex-col items-center justify-center gap-3 text-white/60">
              {getDownloadURL.isPending ? (
                <>
                  <span className="h-6 w-6 animate-spin rounded-full border-2 border-white/30 border-t-white/80" aria-hidden="true" />
                  <p className="text-sm">{t('features.video.recordingPlayer.loading')}</p>
                </>
              ) : isProcessing ? (
                <>
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <circle cx="12" cy="12" r="10" />
                    <polyline points="12 6 12 12 16 14" />
                  </svg>
                  <p className="text-sm">{t('features.video.recordingPlayer.processing')}</p>
                </>
              ) : getDownloadURL.isError ? (
                <>
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <circle cx="12" cy="12" r="10" />
                    <line x1="12" y1="8" x2="12" y2="12" />
                    <line x1="12" y1="16" x2="12.01" y2="16" />
                  </svg>
                  <p className="text-sm">{t('features.video.recordingPlayer.error')}</p>
                </>
              ) : null}
            </div>
          )}
        </div>

        {/* Metadata footer */}
        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border bg-card px-5 py-3">
          <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
            {recording?.duration_seconds != null && (
              <span>
                {t('features.video.recordingPlayer.duration')}: {formatDurationSeconds(recording.duration_seconds)}
              </span>
            )}
            {recording?.file_size_bytes != null && (
              <span>
                {t('features.video.recordingPlayer.size')}: {formatBytes(recording.file_size_bytes)}
              </span>
            )}
            {retentionDays !== null && (
              <span
                className={cn(
                  'font-medium',
                  retentionDays <= 7 ? 'text-destructive' : retentionDays <= 30 ? 'text-amber-500' : '',
                )}
              >
                {retentionDays === 0
                  ? t('features.video.recordingPlayer.expirestoday')
                  : t('features.video.recordingPlayer.expiresIn', { days: retentionDays })}
              </span>
            )}
          </div>

          <div className="flex items-center gap-2">
            {canPlay && getDownloadURL.data?.url && (
              <Button variant="outline" size="sm" asChild>
                <a
                  href={getDownloadURL.data.url}
                  download
                  target="_blank"
                  rel="noreferrer"
                >
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="mr-1.5" aria-hidden="true">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                  </svg>
                  {t('features.video.recordingPlayer.download')}
                </a>
              </Button>
            )}
            <Button variant="ghost" size="sm" onClick={handleClose}>
              {t('common.close')}
            </Button>
          </div>
        </div>
      </AlertDialogContent>
    </AlertDialog>
  )
}