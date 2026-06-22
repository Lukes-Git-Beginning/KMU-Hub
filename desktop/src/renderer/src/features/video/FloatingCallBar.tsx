/**
 * FloatingCallBar -- Compact floating mini-bar for background calls.
 *
 * Appears when the user navigates away from the call view while still
 * in an active call. Shows a duration timer + quick controls for
 * mute, camera toggle, and hang-up.
 *
 * Positioned as a fixed pill in the bottom-right corner with backdrop
 * blur and z-50 to float above all content.
 *
 * Clicking the bar (outside buttons) navigates back to the call view.
 *
 * Wave 1.5: Mic and camera buttons are now functional. They read
 * roomControls from the video store (injected by InnerCallView while
 * LiveKitRoom is mounted) and call the toggle callbacks directly.
 * When no roomControls are present (call not yet connected or already
 * disconnected) the buttons are shown as inactive but remain clickable.
 */
import { useCallback, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

import { useVideoStore } from '@/stores/video'
import { useEndCall } from '@/api/hooks/useVideo'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface FloatingCallBarProps {
  /** Optional CSS class for the container. */
  className?: string
}

// ---------------------------------------------------------------------------
// Duration formatter
// ---------------------------------------------------------------------------

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function FloatingCallBar({ className }: FloatingCallBarProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const endCallMutation = useEndCall()

  const {
    isInCall,
    isFloatingBarVisible,
    callDuration,
    activeCall,
    roomControls,
    setCallDuration,
    clearActiveCall,
  } = useVideoStore()

  // Duration timer -- increment every second
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    if (isInCall && isFloatingBarVisible) {
      timerRef.current = setInterval(() => {
        setCallDuration(useVideoStore.getState().callDuration + 1)
      }, 1000)
    }

    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current)
        timerRef.current = null
      }
    }
  }, [isInCall, isFloatingBarVisible, setCallDuration])

  // Navigate to call view
  const handleBarClick = useCallback(() => {
    if (activeCall) {
      navigate(`/video/call/${activeCall.id}`)
    }
  }, [navigate, activeCall])

  // Hang up
  const handleHangUp = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation()
      if (!activeCall) return
      try {
        await endCallMutation.mutateAsync(activeCall.id)
      } finally {
        clearActiveCall()
      }
    },
    [activeCall, endCallMutation, clearActiveCall],
  )

  // Mic toggle -- uses roomControls from store (injected by InnerCallView)
  const handleMicToggle = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation()
      if (roomControls) {
        await roomControls.toggleMic()
      }
    },
    [roomControls],
  )

  // Camera toggle -- uses roomControls from store
  const handleCamToggle = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation()
      if (roomControls) {
        await roomControls.toggleCam()
      }
    },
    [roomControls],
  )

  // Don't render if not in call or bar not visible
  if (!isInCall || !isFloatingBarVisible) return null

  const isMicOn = roomControls?.isMicEnabled ?? true
  const isCamOn = roomControls?.isCamEnabled ?? true

  return (
    <div
      className={cn(
        'fixed bottom-6 right-6 z-50 flex cursor-pointer items-center gap-3 rounded-full bg-zinc-900/90 px-4 py-2.5 shadow-lg backdrop-blur-md transition-all',
        'animate-in slide-in-from-bottom-4 fade-in duration-300',
        'ring-1 ring-zinc-700/50',
        className,
      )}
      onClick={handleBarClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') handleBarClick()
      }}
      aria-label={t('features.video.floatingBar.backToCall')}
    >
      {/* Pulsing green dot -- call active indicator */}
      <span className="relative flex h-2.5 w-2.5">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
        <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-green-500" />
      </span>

      {/* Duration timer */}
      <span className="min-w-[48px] text-center text-sm font-mono font-medium text-white tabular-nums">
        {formatDuration(callDuration)}
      </span>

      {/* Separator */}
      <div className="h-5 w-px bg-zinc-600" />

      {/* Mic toggle -- functional (1.5) */}
      <button
        className={cn(
          'flex h-8 w-8 items-center justify-center rounded-full transition-colors',
          isMicOn
            ? 'text-zinc-300 hover:bg-zinc-700 hover:text-white'
            : 'bg-red-600/20 text-red-400 hover:bg-red-600/30',
        )}
        onClick={handleMicToggle}
        title={t('features.video.floatingBar.microphone')}
        aria-label={t('features.video.floatingBar.toggleMicrophone')}
      >
        {isMicOn ? (
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-4 w-4"
            aria-hidden="true"
          >
            <path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z" />
            <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
            <line x1="12" x2="12" y1="19" y2="22" />
          </svg>
        ) : (
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-4 w-4"
            aria-hidden="true"
          >
            <line x1="2" x2="22" y1="2" y2="22" />
            <path d="M18.89 13.23A7.12 7.12 0 0 0 19 12v-2" />
            <path d="M5 10v2a7 7 0 0 0 12 5.29" />
            <path d="M15 9.34V5a3 3 0 0 0-5.68-1.33" />
            <path d="M9 9v3a3 3 0 0 0 5.12 2.12" />
            <line x1="12" x2="12" y1="19" y2="22" />
          </svg>
        )}
      </button>

      {/* Camera toggle -- functional (1.5) */}
      <button
        className={cn(
          'flex h-8 w-8 items-center justify-center rounded-full transition-colors',
          isCamOn
            ? 'text-zinc-300 hover:bg-zinc-700 hover:text-white'
            : 'bg-red-600/20 text-red-400 hover:bg-red-600/30',
        )}
        onClick={handleCamToggle}
        title={t('features.video.floatingBar.camera')}
        aria-label={t('features.video.floatingBar.toggleCamera')}
      >
        {isCamOn ? (
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-4 w-4"
            aria-hidden="true"
          >
            <path d="m16 13 5.223 3.482a.5.5 0 0 0 .777-.416V7.87a.5.5 0 0 0-.752-.432L16 10.5" />
            <rect x="2" y="6" width="14" height="12" rx="2" />
          </svg>
        ) : (
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-4 w-4"
            aria-hidden="true"
          >
            <line x1="2" x2="22" y1="2" y2="22" />
            <path d="M21 17.31V7.87a.5.5 0 0 0-.752-.432L16 10.5" />
            <path d="M11.19 6H14a2 2 0 0 1 2 2v3.19" />
            <path d="M2 8a2 2 0 0 1 2-2h0M16 16a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V8" />
          </svg>
        )}
      </button>

      {/* Hang up */}
      <button
        className="flex h-8 w-8 items-center justify-center rounded-full bg-destructive text-white transition-colors hover:bg-destructive/90"
        onClick={handleHangUp}
        title={t('features.video.floatingBar.hangUp')}
        aria-label={t('features.video.floatingBar.endCall')}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="h-4 w-4"
          aria-hidden="true"
        >
          <path d="M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.42 19.42 0 0 1-3.33-2.67m-2.67-3.34a19.79 19.79 0 0 1-3.07-8.63A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91" />
          <line x1="22" x2="2" y1="2" y2="22" />
        </svg>
      </button>
    </div>
  )
}
