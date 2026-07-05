/**
 * IncomingCallOverlay -- Fullscreen overlay for incoming calls.
 *
 * Displays a phone-app style fullscreen overlay with caller avatar,
 * name, call type, and accept/decline buttons. Uses the video store
 * incomingCall state (set by WebSocket handler on 'call.incoming').
 *
 * Accept: joins the call and navigates to the call view.
 * Decline: clears the incoming call state (caller side handles timeout).
 *
 * Includes a pulsing animation on the avatar and gentle button animations.
 */
import { useCallback, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'

import { useVideoStore } from '@/stores/video'
import { useJoinCall } from '@/api/hooks/useVideo'
import { wsManager } from '@/api/websocket'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { cn } from '@/lib'
import { useIncomingCallListener } from './useIncomingCallListener'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface IncomingCallOverlayProps {
  /** Optional CSS class for the container. */
  className?: string
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getCallTypeLabel(callType: 'one_to_one' | 'group'): string {
  return callType === 'group' ? 'Gruppenanruf' : 'Videoanruf'
}

function getInitials(name: string): string {
  return name
    .split(' ')
    .map((w) => w[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function IncomingCallOverlay({ className }: IncomingCallOverlayProps) {
  const navigate = useNavigate()
  const joinCall = useJoinCall()
  const audioRef = useRef<HTMLAudioElement | null>(null)

  // Wires 'call.incoming' / 'call.ended' WS events into the video store.
  // This component is always mounted globally (AppShell), so this is
  // effectively the app-wide listener registration point.
  useIncomingCallListener()

  const incomingCall = useVideoStore((s) => s.incomingCall)
  const setIncomingCall = useVideoStore((s) => s.setIncomingCall)
  const setActiveCall = useVideoStore((s) => s.setActiveCall)

  // Play ringtone sound when incoming call appears
  useEffect(() => {
    if (!incomingCall) return

    // Use Web Audio API to generate a simple ringtone tone
    // (actual sound file can be added later at /sounds/ringtone.mp3)
    let audioContext: AudioContext | null = null
    let oscillator: OscillatorNode | null = null
    let gainNode: GainNode | null = null
    let intervalId: ReturnType<typeof setInterval> | null = null

    try {
      audioContext = new AudioContext()
      gainNode = audioContext.createGain()
      gainNode.connect(audioContext.destination)
      gainNode.gain.value = 0

      // Ring pattern: 400Hz tone, 1s on, 2s off
      const startRing = () => {
        if (!audioContext || !gainNode) return
        oscillator = audioContext.createOscillator()
        oscillator.type = 'sine'
        oscillator.frequency.value = 400
        oscillator.connect(gainNode)
        oscillator.start()
        gainNode.gain.setValueAtTime(0.15, audioContext.currentTime)
        gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 1)
        setTimeout(() => {
          oscillator?.stop()
          oscillator?.disconnect()
          oscillator = null
        }, 1000)
      }

      startRing()
      intervalId = setInterval(startRing, 3000)
    } catch {
      // Audio API may not be available -- fail silently
    }

    return () => {
      if (intervalId) clearInterval(intervalId)
      oscillator?.stop()
      oscillator?.disconnect()
      gainNode?.disconnect()
      if (audioContext?.state !== 'closed') {
        audioContext?.close()
      }
    }
  }, [incomingCall])

  // Accept call
  const handleAccept = useCallback(async () => {
    if (!incomingCall) return
    try {
      const response = await joinCall.mutateAsync(incomingCall.callId)
      setActiveCall(
        {
          id: incomingCall.callId,
          call_type: incomingCall.callType,
          status: 'active',
          room_name: '',
          initiator_id: '',
          channel_id: null,
          created_at: new Date().toISOString(),
          ended_at: null,
          duration_seconds: null,
          participants: null,
        },
        response.token,
        response.ws_url,
        response.ice_servers,
      )
      setIncomingCall(null)
      navigate(`/video/call/${incomingCall.callId}`)
    } catch {
      // Join failed -- keep overlay visible so user can retry
    }
  }, [incomingCall, joinCall, setActiveCall, setIncomingCall, navigate])

  // Decline call
  const handleDecline = useCallback(() => {
    if (!incomingCall) return
    wsManager.send({ type: 'call.declined', message_id: incomingCall.callId })
    setIncomingCall(null)
  }, [incomingCall, setIncomingCall])

  // Don't render if no incoming call
  if (!incomingCall) return null

  return (
    <div
      className={cn(
        'fixed inset-0 z-[100] flex flex-col items-center justify-center bg-black/80 backdrop-blur-md',
        'animate-in fade-in duration-300',
        className,
      )}
    >
      {/* Hidden audio element for future ringtone file */}
      <audio ref={audioRef} loop preload="none" />

      {/* Caller info */}
      <div className="flex flex-col items-center gap-6">
        {/* Pulsing avatar */}
        <div className="relative">
          <div className="absolute -inset-3 animate-ping rounded-full bg-green-500/20" />
          <div className="absolute -inset-3 animate-pulse rounded-full bg-green-500/10" />
          <Avatar className="h-28 w-28 border-4 border-green-500/30">
            {incomingCall.callerAvatar ? (
              <AvatarImage
                src={incomingCall.callerAvatar}
                alt={incomingCall.callerName}
              />
            ) : null}
            <AvatarFallback className="bg-zinc-700 text-2xl text-white">
              {getInitials(incomingCall.callerName)}
            </AvatarFallback>
          </Avatar>
        </div>

        {/* Caller name */}
        <div className="text-center">
          <h2 className="text-2xl font-semibold text-white">
            {incomingCall.callerName}
          </h2>
          <p className="mt-1 text-sm text-zinc-400">
            {getCallTypeLabel(incomingCall.callType)}
          </p>
        </div>
      </div>

      {/* Accept / Decline buttons */}
      <div className="mt-12 flex items-center gap-10">
        {/* Decline */}
        <div className="flex flex-col items-center gap-2">
          <button
            className={cn(
              'flex h-16 w-16 items-center justify-center rounded-full bg-destructive text-white shadow-lg transition-all',
              'hover:bg-destructive/90 hover:scale-105',
              'focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-destructive/50',
            )}
            onClick={handleDecline}
            aria-label="Anruf ablehnen"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-7 w-7"
            >
              <path d="M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.42 19.42 0 0 1-3.33-2.67m-2.67-3.34a19.79 19.79 0 0 1-3.07-8.63A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91" />
              <line x1="22" x2="2" y1="2" y2="22" />
            </svg>
          </button>
          <span className="text-sm text-zinc-400">Ablehnen</span>
        </div>

        {/* Accept */}
        <div className="flex flex-col items-center gap-2">
          <button
            className={cn(
              'flex h-16 w-16 items-center justify-center rounded-full bg-green-600 text-white shadow-lg transition-all',
              'hover:bg-green-700 hover:scale-105',
              'focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-green-500/50',
              'animate-pulse',
            )}
            onClick={handleAccept}
            disabled={joinCall.isPending}
            aria-label="Anruf annehmen"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-7 w-7"
            >
              <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" />
            </svg>
          </button>
          <span className="text-sm text-zinc-400">Annehmen</span>
        </div>
      </div>
    </div>
  )
}
