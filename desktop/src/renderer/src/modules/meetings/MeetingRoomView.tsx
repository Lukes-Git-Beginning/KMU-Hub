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
  Settings,
  MonitorOff,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib'
import type { Meeting } from '@/stores/meetings'

type SidebarTab = 'participants' | 'chat' | 'settings'

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
  const [sidebarTab, setSidebarTab] = useState<SidebarTab | null>(null)
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
    <div className="dark fixed inset-0 z-[60] flex flex-col bg-background">
      {/* Top bar */}
      <div className="flex items-center justify-between px-4 py-2 bg-card/80">
        <div className="flex items-center gap-3">
          <h3 className="text-sm font-medium text-foreground">{meeting.title}</h3>
          <span className="text-xs text-muted-foreground">{meeting.room}</span>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            {formatTime(elapsed)}
          </div>
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Users className="h-3.5 w-3.5" />
            {participants.length}
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-muted-foreground hover:text-foreground hover:bg-muted"
            onClick={() => setIsMaximized(!isMaximized)}
          >
            {isMaximized ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
          </Button>
        </div>
      </div>

      {/* Screen share banner */}
      {isScreenSharing && (
        <div className="flex items-center justify-center gap-2 bg-primary/10 border-b border-primary/20 px-4 py-1.5">
          <Monitor className="h-4 w-4 text-primary" />
          <span className="text-xs font-medium text-primary">Du teilst deinen Bildschirm</span>
          <button
            onClick={() => setIsScreenSharing(false)}
            className="ml-2 rounded-md bg-red-600 px-2 py-0.5 text-xs font-medium text-white hover:bg-red-700 transition-colors"
          >
            Freigabe beenden
          </button>
        </div>
      )}

      {/* Main content */}
      <div className="flex flex-1 overflow-hidden">
        {/* Video grid */}
        <div className="flex-1 p-3">
          <div className={cn('grid h-full gap-2', gridClass)}>
            {participants.map((p, i) => (
              <div
                key={p.id}
                className="relative flex items-center justify-center rounded-xl bg-card overflow-hidden"
              >
                {/* Camera placeholder */}
                <div className="flex flex-col items-center gap-2">
                  <div className="flex h-16 w-16 items-center justify-center rounded-full bg-muted text-xl font-medium text-foreground">
                    {p.initials}
                  </div>
                </div>

                {/* Name + mute overlay */}
                <div className="absolute bottom-2 left-2 flex items-center gap-1.5 rounded-md bg-black/50 px-2 py-1">
                  <span className="text-xs text-white">{p.name}</span>
                  {i === 0 && isMuted && <MicOff className="h-3 w-3 text-red-400" />}
                </div>

                {/* Hand raise indicator */}
                {((i === 0 && isHandRaised) || i === 2) && (
                  <div className="absolute top-2 right-2 flex h-7 w-7 items-center justify-center rounded-full bg-yellow-500/90 shadow-sm animate-bounce">
                    <Hand className="h-4 w-4 text-white" />
                  </div>
                )}

                {/* Active speaker ring */}
                {i === 0 && (
                  <div className="absolute inset-0 rounded-xl border-2 border-primary/60 pointer-events-none" />
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Side panel with tabs */}
        {sidebarTab !== null && (
          <div className="w-72 flex flex-col border-l border-border bg-card/60">
            {/* Sidebar tab headers */}
            <div className="flex items-center border-b border-border">
              {([
                { key: 'participants' as SidebarTab, label: 'Teilnehmer', icon: Users },
                { key: 'chat' as SidebarTab, label: 'Chat', icon: MessageSquare },
                { key: 'settings' as SidebarTab, label: 'Einstellungen', icon: Settings },
              ]).map((tab) => (
                <button
                  key={tab.key}
                  onClick={() => setSidebarTab(tab.key)}
                  className={cn(
                    'flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-medium border-b-2 transition-colors',
                    sidebarTab === tab.key
                      ? 'border-primary text-primary'
                      : 'border-transparent text-muted-foreground hover:text-foreground'
                  )}
                >
                  <tab.icon className="h-3.5 w-3.5" />
                  {tab.label}
                </button>
              ))}
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 mx-1 text-muted-foreground hover:text-foreground hover:bg-muted shrink-0"
                onClick={() => setSidebarTab(null)}
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            </div>

            {/* Participants */}
            {sidebarTab === 'participants' && (
              <div className="flex-1 overflow-y-auto p-3 space-y-1">
                {participants.map((p, i) => (
                  <div key={p.id} className="flex items-center gap-2 rounded-md p-2 hover:bg-muted/50">
                    <span className="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-xs font-medium text-foreground">
                      {p.initials}
                    </span>
                    <span className="flex-1 text-sm text-foreground/90">{p.name}</span>
                    {i === 0 && isMuted && <MicOff className="h-3.5 w-3.5 text-red-400" />}
                    {((i === 0 && isHandRaised) || i === 2) && <Hand className="h-3.5 w-3.5 text-yellow-500" />}
                  </div>
                ))}
              </div>
            )}

            {/* Chat */}
            {sidebarTab === 'chat' && (
              <div className="flex flex-1 flex-col">
                <div className="flex-1 overflow-y-auto p-3">
                  <p className="text-xs text-muted-foreground/70 text-center mt-8">
                    Noch keine Nachrichten
                  </p>
                </div>
                <div className="border-t border-border p-2">
                  <input
                    type="text"
                    placeholder="Nachricht..."
                    className="w-full rounded-md bg-muted px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground/70 focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
              </div>
            )}

            {/* Settings */}
            {sidebarTab === 'settings' && (
              <div className="flex-1 overflow-y-auto p-3 space-y-4">
                <div>
                  <label className="text-xs text-muted-foreground mb-1 block">Mikrofon</label>
                  <select className="w-full rounded-md border border-border bg-muted/50 px-2 py-1.5 text-xs text-foreground">
                    <option>Standard-Mikrofon</option>
                    <option>Headset (USB)</option>
                  </select>
                </div>
                <div>
                  <label className="text-xs text-muted-foreground mb-1 block">Lautsprecher</label>
                  <select className="w-full rounded-md border border-border bg-muted/50 px-2 py-1.5 text-xs text-foreground">
                    <option>Standard-Lautsprecher</option>
                    <option>Headset (USB)</option>
                  </select>
                </div>
                <div>
                  <label className="text-xs text-muted-foreground mb-1 block">Kamera</label>
                  <select className="w-full rounded-md border border-border bg-muted/50 px-2 py-1.5 text-xs text-foreground">
                    <option>Integrierte Webcam</option>
                    <option>HD Pro Webcam</option>
                  </select>
                </div>
                <div>
                  <label className="text-xs text-muted-foreground mb-1 block">Layout</label>
                  <div className="flex gap-1">
                    {(['grid', 'speaker', 'sidebar'] as const).map((l) => (
                      <button
                        key={l}
                        className={cn(
                          'flex-1 rounded-md py-1.5 text-xs font-medium transition-colors',
                          l === 'grid' ? 'bg-primary/15 text-primary' : 'bg-muted/50 text-muted-foreground hover:bg-muted'
                        )}
                      >
                        {l === 'grid' ? 'Raster' : l === 'speaker' ? 'Sprecher' : 'Seite'}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Bottom toolbar */}
      <div className="flex items-center justify-center gap-2 px-4 py-3 bg-card/80">
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
          active={sidebarTab === 'chat'}
          onClick={() => setSidebarTab(sidebarTab === 'chat' ? null : 'chat')}
        />
        <ToolbarButton
          icon={Users}
          label="Teilnehmer"
          active={sidebarTab === 'participants'}
          onClick={() => setSidebarTab(sidebarTab === 'participants' ? null : 'participants')}
        />

        <div className="mx-2 h-8 w-px bg-border" />

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
            ? 'bg-muted text-foreground hover:bg-secondary-hover'
            : 'text-muted-foreground hover:bg-muted hover:text-foreground'
      )}
      title={label}
    >
      <Icon className="h-5 w-5" />
      <span className="hidden sm:block">{label}</span>
    </button>
  )
}
