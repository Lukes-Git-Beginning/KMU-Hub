import { useState, useEffect } from 'react'
import {
  Mic,
  MicOff,
  Video,
  VideoOff,
  Monitor,
  Hand,
  MessageSquare,
  Presentation,
  PhoneOff,
  Users,
  X,
  Maximize2,
  Minimize2,
  Clock,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib'
import type { Meeting } from '@/stores/meetings'

interface MeetingRoomViewProps {
  meeting: Meeting
  open: boolean
  onLeave: () => void
}

export function MeetingRoomView({ meeting, open, onLeave }: MeetingRoomViewProps) {
  const [isMuted, setIsMuted] = useState(false)
  const [isCameraOff, setIsCameraOff] = useState(false)
  const [isScreenSharing, setIsScreenSharing] = useState(false)
  const [isHandRaised, setIsHandRaised] = useState(false)
  const [showChat, setShowChat] = useState(false)
  const [showParticipants, setShowParticipants] = useState(false)
  const [isMaximized, setIsMaximized] = useState(false)
  const [elapsed, setElapsed] = useState(0)

  useEffect(() => {
    if (!open) {
      setElapsed(0)
      return
    }
    const interval = setInterval(() => setElapsed((e) => e + 1), 1000)
    return () => clearInterval(interval)
  }, [open])

  if (!open) return null

  const formatTime = (seconds: number) => {
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = seconds % 60
    if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
    return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  }

  const participants = meeting.participants.length > 0 ? meeting.participants : [
    { id: 'self', name: 'Du', initials: 'DU' },
  ]

  // Grid layout based on participant count
  const gridClass =
    participants.length <= 2
      ? 'grid-cols-1 sm:grid-cols-2'
      : participants.length <= 4
        ? 'grid-cols-2'
        : participants.length <= 6
          ? 'grid-cols-2 sm:grid-cols-3'
          : 'grid-cols-3 sm:grid-cols-4'

  return (
    <div className="fixed inset-0 z-[60] flex flex-col bg-gray-900">
      {/* Top bar */}
      <div className="flex items-center justify-between px-4 py-2 bg-gray-800/80">
        <div className="flex items-center gap-3">
          <h3 className="text-sm font-medium text-white">{meeting.title}</h3>
          <span className="text-xs text-gray-400">{meeting.room}</span>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5 text-xs text-gray-300">
            <Clock className="h-3.5 w-3.5" />
            {formatTime(elapsed)}
          </div>
          <div className="flex items-center gap-1.5 text-xs text-gray-400">
            <Users className="h-3.5 w-3.5" />
            {participants.length}
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-gray-400 hover:text-white hover:bg-gray-700"
            onClick={() => setIsMaximized(!isMaximized)}
          >
            {isMaximized ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
          </Button>
        </div>
      </div>

      {/* Main content */}
      <div className="flex flex-1 overflow-hidden">
        {/* Video grid */}
        <div className="flex-1 p-3">
          <div className={cn('grid h-full gap-2', gridClass)}>
            {participants.map((p, i) => (
              <div
                key={p.id}
                className="relative flex items-center justify-center rounded-xl bg-gray-800 overflow-hidden"
              >
                {/* Camera placeholder */}
                <div className="flex flex-col items-center gap-2">
                  <div className="flex h-16 w-16 items-center justify-center rounded-full bg-gray-700 text-xl font-medium text-white">
                    {p.initials}
                  </div>
                  <span className="text-sm text-gray-300">{p.name}</span>
                </div>

                {/* Name overlay */}
                <div className="absolute bottom-2 left-2 flex items-center gap-1.5 rounded-md bg-black/50 px-2 py-1">
                  <span className="text-xs text-white">{p.name}</span>
                  {i === 0 && isMuted && <MicOff className="h-3 w-3 text-red-400" />}
                </div>

                {/* Active speaker indicator */}
                {i === 0 && (
                  <div className="absolute inset-0 rounded-xl border-2 border-primary/60 pointer-events-none" />
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Side panel: Participants or Chat */}
        {(showParticipants || showChat) && (
          <div className="w-72 flex flex-col border-l border-gray-700 bg-gray-800/60">
            <div className="flex items-center justify-between px-3 py-2 border-b border-gray-700">
              <span className="text-sm font-medium text-white">
                {showParticipants ? 'Teilnehmer' : 'Chat'}
              </span>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 text-gray-400 hover:text-white hover:bg-gray-700"
                onClick={() => {
                  setShowParticipants(false)
                  setShowChat(false)
                }}
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            </div>

            {showParticipants && (
              <div className="flex-1 overflow-y-auto p-3 space-y-2">
                {participants.map((p) => (
                  <div key={p.id} className="flex items-center gap-2 rounded-md p-2 hover:bg-gray-700/50">
                    <span className="flex h-8 w-8 items-center justify-center rounded-full bg-gray-700 text-xs font-medium text-white">
                      {p.initials}
                    </span>
                    <span className="text-sm text-gray-200">{p.name}</span>
                  </div>
                ))}
              </div>
            )}

            {showChat && (
              <div className="flex flex-1 flex-col">
                <div className="flex-1 overflow-y-auto p-3">
                  <p className="text-xs text-gray-500 text-center mt-8">
                    Noch keine Nachrichten
                  </p>
                </div>
                <div className="border-t border-gray-700 p-2">
                  <input
                    type="text"
                    placeholder="Nachricht..."
                    className="w-full rounded-md bg-gray-700 px-3 py-1.5 text-sm text-white placeholder:text-gray-500 focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Bottom toolbar */}
      <div className="flex items-center justify-center gap-2 px-4 py-3 bg-gray-800/80">
        <ToolbarButton
          icon={isMuted ? MicOff : Mic}
          label={isMuted ? 'Stumm' : 'Mikrofon'}
          active={!isMuted}
          danger={isMuted}
          onClick={() => setIsMuted(!isMuted)}
        />
        <ToolbarButton
          icon={isCameraOff ? VideoOff : Video}
          label={isCameraOff ? 'Kamera aus' : 'Kamera'}
          active={!isCameraOff}
          danger={isCameraOff}
          onClick={() => setIsCameraOff(!isCameraOff)}
        />
        <ToolbarButton
          icon={Monitor}
          label="Bildschirm"
          active={isScreenSharing}
          onClick={() => setIsScreenSharing(!isScreenSharing)}
        />
        <ToolbarButton
          icon={Hand}
          label="Hand"
          active={isHandRaised}
          onClick={() => setIsHandRaised(!isHandRaised)}
        />
        <ToolbarButton
          icon={Presentation}
          label="Whiteboard"
          onClick={() => {}}
        />
        <ToolbarButton
          icon={MessageSquare}
          label="Chat"
          active={showChat}
          onClick={() => { setShowChat(!showChat); setShowParticipants(false) }}
        />
        <ToolbarButton
          icon={Users}
          label="Teilnehmer"
          active={showParticipants}
          onClick={() => { setShowParticipants(!showParticipants); setShowChat(false) }}
        />

        <div className="mx-2 h-8 w-px bg-gray-700" />

        <button
          onClick={onLeave}
          className="flex items-center gap-2 rounded-full bg-red-600 px-5 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors"
        >
          <PhoneOff className="h-4 w-4" />
          Verlassen
        </button>
      </div>
    </div>
  )
}

function ToolbarButton({
  icon: Icon,
  label,
  active,
  danger,
  onClick,
}: {
  icon: React.ElementType
  label: string
  active?: boolean
  danger?: boolean
  onClick: () => void
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'flex flex-col items-center gap-1 rounded-lg px-3 py-2 text-xs transition-colors',
        danger
          ? 'bg-red-500/20 text-red-400 hover:bg-red-500/30'
          : active
            ? 'bg-gray-700 text-white hover:bg-gray-600'
            : 'text-gray-400 hover:bg-gray-700 hover:text-white'
      )}
      title={label}
    >
      <Icon className="h-5 w-5" />
      <span className="hidden sm:block">{label}</span>
    </button>
  )
}
