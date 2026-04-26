/**
 * CallControls -- Bottom control bar during a video call.
 *
 * Provides buttons for:
 *   - Mute / unmute microphone
 *   - Camera on / off
 *   - Screen share toggle
 *   - Record (initiates RecordingConsentDialog flow)
 *   - Leave call (red hang-up button)
 *
 * Uses LiveKit hooks to read and toggle local track state.
 * Visual indicators: red dot when recording, highlight when sharing screen.
 */
import { useCallback, useState } from 'react'
import {
  useLocalParticipant,
  useRoomContext,
  useIsRecording,
} from '@livekit/components-react'

import { useEndCall, useStartRecording, useStopRecording } from '@/api/hooks/useVideo'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface CallControlsProps {
  /** Active call ID for API mutations. */
  callId: string
  /** Called after the local participant leaves the call. */
  onLeave?: () => void
  /** Optional CSS class for the container. */
  className?: string
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function CallControls({ callId, onLeave, className }: CallControlsProps) {
  const room = useRoomContext()
  const { localParticipant, isMicrophoneEnabled, isCameraEnabled, isScreenShareEnabled } =
    useLocalParticipant()
  const isRecording = useIsRecording()

  const endCall = useEndCall()
  const startRecording = useStartRecording()
  const stopRecording = useStopRecording()

  const [isLeaving, setIsLeaving] = useState(false)
  // Tracks the recording id returned by startRecording so stopRecording can target
  // the correct backend route (/recordings/{id}/stop). LiveKit's useIsRecording
  // tells us *that* a recording is active, but not *which* one. If the user joins
  // a call where someone else already started recording, this stays null and the
  // stop button is hidden — Sprint 3 backlog: query the active recording by call id.
  const [activeRecordingId, setActiveRecordingId] = useState<string | null>(null)

  // Mic toggle
  const handleMicToggle = useCallback(async () => {
    await localParticipant.setMicrophoneEnabled(!isMicrophoneEnabled)
  }, [localParticipant, isMicrophoneEnabled])

  // Camera toggle
  const handleCameraToggle = useCallback(async () => {
    await localParticipant.setCameraEnabled(!isCameraEnabled)
  }, [localParticipant, isCameraEnabled])

  // Screen share toggle
  const handleScreenShareToggle = useCallback(async () => {
    await localParticipant.setScreenShareEnabled(!isScreenShareEnabled)
  }, [localParticipant, isScreenShareEnabled])

  // Record toggle
  const handleRecordToggle = useCallback(async () => {
    if (isRecording && activeRecordingId) {
      await stopRecording.mutateAsync(activeRecordingId)
      setActiveRecordingId(null)
    } else if (!isRecording) {
      const recording = await startRecording.mutateAsync(callId)
      setActiveRecordingId(recording.id)
    }
  }, [isRecording, activeRecordingId, callId, startRecording, stopRecording])

  // Leave call
  const handleLeave = useCallback(async () => {
    setIsLeaving(true)
    try {
      await endCall.mutateAsync(callId)
      await room.disconnect()
    } finally {
      setIsLeaving(false)
      onLeave?.()
    }
  }, [callId, endCall, room, onLeave])

  return (
    <div
      className={cn(
        'flex items-center justify-center gap-3 border-t border-zinc-800 bg-zinc-900/95 px-6 py-3 backdrop-blur-sm',
        className,
      )}
    >
      {/* Mute / Unmute Mic */}
      <ControlButton
        active={isMicrophoneEnabled}
        onClick={handleMicToggle}
        title={isMicrophoneEnabled ? 'Mikrofon stummschalten' : 'Mikrofon einschalten'}
        aria-label={isMicrophoneEnabled ? 'Mikrofon stummschalten' : 'Mikrofon einschalten'}
      >
        {isMicrophoneEnabled ? (
          <MicIcon className="h-5 w-5" />
        ) : (
          <MicOffIcon className="h-5 w-5" />
        )}
      </ControlButton>

      {/* Camera on/off */}
      <ControlButton
        active={isCameraEnabled}
        onClick={handleCameraToggle}
        title={isCameraEnabled ? 'Kamera ausschalten' : 'Kamera einschalten'}
        aria-label={isCameraEnabled ? 'Kamera ausschalten' : 'Kamera einschalten'}
      >
        {isCameraEnabled ? (
          <CameraIcon className="h-5 w-5" />
        ) : (
          <CameraOffIcon className="h-5 w-5" />
        )}
      </ControlButton>

      {/* Screen share */}
      <ControlButton
        active={isScreenShareEnabled}
        onClick={handleScreenShareToggle}
        title={
          isScreenShareEnabled
            ? 'Bildschirmfreigabe beenden'
            : 'Bildschirm freigeben'
        }
        aria-label={
          isScreenShareEnabled
            ? 'Bildschirmfreigabe beenden'
            : 'Bildschirm freigeben'
        }
        className={isScreenShareEnabled ? 'bg-blue-600 text-white hover:bg-blue-700' : ''}
      >
        <ScreenShareIcon className="h-5 w-5" />
      </ControlButton>

      {/* Record */}
      <ControlButton
        active={isRecording}
        onClick={handleRecordToggle}
        title={isRecording ? 'Aufnahme stoppen' : 'Aufnahme starten'}
        aria-label={isRecording ? 'Aufnahme stoppen' : 'Aufnahme starten'}
        className={isRecording ? 'bg-destructive/20 text-destructive hover:bg-destructive/30' : ''}
      >
        <div className="relative">
          <RecordIcon className="h-5 w-5" />
          {isRecording && (
            <span className="absolute -right-1 -top-1 h-2.5 w-2.5 animate-pulse rounded-full bg-destructive" />
          )}
        </div>
      </ControlButton>

      {/* Separator */}
      <div className="mx-2 h-8 w-px bg-zinc-700" />

      {/* Leave call */}
      <button
        className={cn(
          'flex h-11 items-center gap-2 rounded-full bg-destructive px-5 text-sm font-medium text-white transition-colors hover:bg-destructive/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-destructive/50 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-900 disabled:opacity-50',
        )}
        onClick={handleLeave}
        disabled={isLeaving}
        title="Anruf beenden"
        aria-label="Anruf beenden"
      >
        <PhoneOffIcon className="h-5 w-5" />
        <span>Auflegen</span>
      </button>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Reusable control button
// ---------------------------------------------------------------------------

interface ControlButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  active?: boolean
}

function ControlButton({
  active,
  className,
  children,
  ...props
}: ControlButtonProps) {
  return (
    <button
      className={cn(
        'flex h-11 w-11 items-center justify-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-400 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-900',
        active
          ? 'bg-zinc-700 text-white hover:bg-zinc-600'
          : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-white',
        className,
      )}
      {...props}
    >
      {children}
    </button>
  )
}

// ---------------------------------------------------------------------------
// Inline SVG icons (avoiding extra dependencies)
// ---------------------------------------------------------------------------

function MicIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z" />
      <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
      <line x1="12" x2="12" y1="19" y2="22" />
    </svg>
  )
}

function MicOffIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <line x1="2" x2="22" y1="2" y2="22" />
      <path d="M18.89 13.23A7.12 7.12 0 0 0 19 12v-2" />
      <path d="M5 10v2a7 7 0 0 0 12 5.29" />
      <path d="M15 9.34V5a3 3 0 0 0-5.68-1.33" />
      <path d="M9 9v3a3 3 0 0 0 5.12 2.12" />
      <line x1="12" x2="12" y1="19" y2="22" />
    </svg>
  )
}

function CameraIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <path d="m16 13 5.223 3.482a.5.5 0 0 0 .777-.416V7.87a.5.5 0 0 0-.752-.432L16 10.5" />
      <rect x="2" y="6" width="14" height="12" rx="2" />
    </svg>
  )
}

function CameraOffIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <line x1="2" x2="22" y1="2" y2="22" />
      <path d="M21 17.31V7.87a.5.5 0 0 0-.752-.432L16 10.5" />
      <path d="M11.19 6H14a2 2 0 0 1 2 2v3.19" />
      <path d="M2 8a2 2 0 0 1 2-2h0M16 16a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V8" />
    </svg>
  )
}

function ScreenShareIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <path d="M13 3H4a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-3" />
      <path d="M8 21h8" />
      <path d="M12 17v4" />
      <path d="m17 8 5-5" />
      <path d="M17 3h5v5" />
    </svg>
  )
}

function RecordIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <circle cx="12" cy="12" r="10" />
      <circle cx="12" cy="12" r="4" fill="currentColor" />
    </svg>
  )
}

function PhoneOffIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <path d="M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.42 19.42 0 0 1-3.33-2.67m-2.67-3.34a19.79 19.79 0 0 1-3.07-8.63A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91" />
      <line x1="22" x2="2" y1="2" y2="22" />
    </svg>
  )
}
