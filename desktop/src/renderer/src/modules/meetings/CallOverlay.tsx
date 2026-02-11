import { useState, useEffect } from 'react'
import { Mic, MicOff, Video, VideoOff, PhoneOff, Minimize2 } from 'lucide-react'
import { cn } from '@/lib'

interface CallOverlayProps {
  contactName: string
  contactInitials?: string
  open: boolean
  onEnd: () => void
}

export function CallOverlay({ contactName, contactInitials, open, onEnd }: CallOverlayProps) {
  const [isMuted, setIsMuted] = useState(false)
  const [isCameraOff, setIsCameraOff] = useState(false)
  const [callState, setCallState] = useState<'ringing' | 'connected'>('ringing')
  const [elapsed, setElapsed] = useState(0)
  const [isMinimized, setIsMinimized] = useState(false)

  const initials = contactInitials || contactName
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)

  useEffect(() => {
    if (!open) {
      setCallState('ringing')
      setElapsed(0)
      setIsMuted(false)
      setIsCameraOff(false)
      setIsMinimized(false)
      return
    }
    // Simulate connecting after 2 seconds
    const connectTimer = setTimeout(() => setCallState('connected'), 2000)
    return () => clearTimeout(connectTimer)
  }, [open])

  useEffect(() => {
    if (!open || callState !== 'connected') return
    const interval = setInterval(() => setElapsed((e) => e + 1), 1000)
    return () => clearInterval(interval)
  }, [open, callState])

  if (!open) return null

  const formatTime = (seconds: number) => {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  }

  // Minimized bubble
  if (isMinimized) {
    return (
      <div
        className="fixed bottom-6 right-6 z-[70] flex items-center gap-3 rounded-full bg-gray-900 px-4 py-3 shadow-2xl cursor-pointer hover:bg-gray-800 transition-colors"
        onClick={() => setIsMinimized(false)}
      >
        <div className="relative">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary text-sm font-medium text-white">
            {initials}
          </div>
          <span className="absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-gray-900 bg-green-500" />
        </div>
        <div>
          <p className="text-sm font-medium text-white">{contactName}</p>
          <p className="text-xs text-gray-400">{formatTime(elapsed)}</p>
        </div>
        <button
          onClick={(e) => {
            e.stopPropagation()
            onEnd()
          }}
          className="ml-2 rounded-full bg-red-600 p-2 hover:bg-red-700 transition-colors"
        >
          <PhoneOff className="h-4 w-4 text-white" />
        </button>
      </div>
    )
  }

  // Full overlay
  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="flex w-80 flex-col items-center rounded-2xl bg-gray-900 p-8 shadow-2xl">
        {/* Minimize button */}
        <button
          onClick={() => setIsMinimized(true)}
          className="absolute top-4 right-4 text-gray-400 hover:text-white"
        >
          <Minimize2 className="h-4 w-4" />
        </button>

        {/* Avatar */}
        <div className="relative mb-4">
          <div
            className={cn(
              'flex h-24 w-24 items-center justify-center rounded-full text-2xl font-medium text-white transition-all',
              callState === 'ringing'
                ? 'bg-primary animate-pulse'
                : 'bg-primary'
            )}
          >
            {initials}
          </div>
          {callState === 'connected' && (
            <span className="absolute bottom-1 right-1 h-4 w-4 rounded-full border-2 border-gray-900 bg-green-500" />
          )}
        </div>

        {/* Name */}
        <h3 className="text-lg font-medium text-white mb-1">{contactName}</h3>

        {/* Status */}
        <p className="text-sm text-gray-400 mb-8">
          {callState === 'ringing' ? 'Klingelt...' : formatTime(elapsed)}
        </p>

        {/* Controls */}
        <div className="flex items-center gap-4">
          <button
            onClick={() => setIsMuted(!isMuted)}
            className={cn(
              'flex h-12 w-12 items-center justify-center rounded-full transition-colors',
              isMuted
                ? 'bg-red-500/20 text-red-400'
                : 'bg-gray-700 text-white hover:bg-gray-600'
            )}
          >
            {isMuted ? <MicOff className="h-5 w-5" /> : <Mic className="h-5 w-5" />}
          </button>

          <button
            onClick={() => setIsCameraOff(!isCameraOff)}
            className={cn(
              'flex h-12 w-12 items-center justify-center rounded-full transition-colors',
              isCameraOff
                ? 'bg-red-500/20 text-red-400'
                : 'bg-gray-700 text-white hover:bg-gray-600'
            )}
          >
            {isCameraOff ? <VideoOff className="h-5 w-5" /> : <Video className="h-5 w-5" />}
          </button>

          <button
            onClick={onEnd}
            className="flex h-14 w-14 items-center justify-center rounded-full bg-red-600 text-white hover:bg-red-700 transition-colors"
          >
            <PhoneOff className="h-6 w-6" />
          </button>
        </div>
      </div>
    </div>
  )
}
