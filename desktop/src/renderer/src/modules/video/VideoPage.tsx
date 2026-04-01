/**
 * Video & Calls module page.
 *
 * Full-featured page with active calls, call history, and device settings.
 * Integrates with meetings store for live meetings and call history data.
 */
import { useState } from 'react'
import {
  Video,
  Phone,
  PhoneIncoming,
  PhoneOutgoing,
  PhoneMissed,
  VideoOff,
  Settings,
  Clock,
  Users,
  Search,
  Plus,
  ExternalLink,
} from 'lucide-react'
import { useMeetingsStore, type CallHistoryEntry } from '../../stores/meetings'
import { PageHeader } from '@/components/shared'

type VideoTab = 'active' | 'history' | 'settings'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatDuration(minutes: number): string {
  if (minutes === 0) return '—'
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return h > 0 ? `${h} Std ${m} Min` : `${m} Min`
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  const today = new Date()
  const yesterday = new Date()
  yesterday.setDate(today.getDate() - 1)

  if (d.toDateString() === today.toDateString()) return 'Heute'
  if (d.toDateString() === yesterday.toDateString()) return 'Gestern'
  return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

function directionIcon(direction: CallHistoryEntry['direction']) {
  switch (direction) {
    case 'incoming':
      return <PhoneIncoming className="h-4 w-4 text-green-500" />
    case 'outgoing':
      return <PhoneOutgoing className="h-4 w-4 text-blue-500" />
    case 'missed':
      return <PhoneMissed className="h-4 w-4 text-red-500" />
  }
}

function directionLabel(direction: CallHistoryEntry['direction']): string {
  switch (direction) {
    case 'incoming':
      return 'Eingehend'
    case 'outgoing':
      return 'Ausgehend'
    case 'missed':
      return 'Verpasst'
  }
}

// ---------------------------------------------------------------------------
// Active Call Card
// ---------------------------------------------------------------------------

function ActiveCallCard({
  meeting,
}: {
  meeting: { id: string; title: string; participants: { id: string; name: string; initials: string }[]; room: string; startTime: string; duration: number; color: string }
}) {
  const { joinMeeting } = useMeetingsStore()

  return (
    <div className="rounded-xl border border-border/60 bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg" style={{ backgroundColor: meeting.color + '20', color: meeting.color }}>
            <Video className="h-5 w-5" />
          </div>
          <div>
            <h3 className="font-medium text-foreground">{meeting.title}</h3>
            <p className="text-xs text-muted-foreground">{meeting.room} — seit {meeting.startTime}</p>
          </div>
        </div>
        <span className="flex items-center gap-1.5 rounded-full bg-green-500/15 px-2.5 py-0.5 text-xs font-medium text-green-600 dark:text-green-400">
          <span className="h-1.5 w-1.5 rounded-full bg-green-500 animate-pulse" />
          Live
        </span>
      </div>

      <div className="mt-3 flex items-center gap-2">
        <div className="flex -space-x-2">
          {meeting.participants.slice(0, 4).map((p) => (
            <div
              key={p.id}
              className="flex h-7 w-7 items-center justify-center rounded-full border-2 border-card bg-muted text-[10px] font-medium text-muted-foreground"
              title={p.name}
            >
              {p.initials}
            </div>
          ))}
          {meeting.participants.length > 4 && (
            <div className="flex h-7 w-7 items-center justify-center rounded-full border-2 border-card bg-muted text-[10px] font-medium text-muted-foreground">
              +{meeting.participants.length - 4}
            </div>
          )}
        </div>
        <span className="text-xs text-muted-foreground">{meeting.participants.length} Teilnehmer</span>
      </div>

      <div className="mt-3 flex gap-2">
        <button
          onClick={() => joinMeeting(meeting.id)}
          className="flex-1 rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700 transition-colors"
        >
          Beitreten
        </button>
        <button className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted transition-colors">
          Details
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Call History Row
// ---------------------------------------------------------------------------

function CallHistoryRow({ entry }: { entry: CallHistoryEntry }) {
  return (
    <div className="flex items-center gap-4 rounded-lg px-3 py-2.5 hover:bg-muted/50 transition-colors group">
      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-muted text-sm font-medium text-muted-foreground">
        {entry.contactInitials}
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className={`font-medium text-sm ${entry.direction === 'missed' ? 'text-red-600 dark:text-red-400' : 'text-foreground'}`}>
            {entry.contactName}
          </span>
          {entry.type === 'video' ? (
            <Video className="h-3.5 w-3.5 text-muted-foreground" />
          ) : (
            <Phone className="h-3.5 w-3.5 text-muted-foreground" />
          )}
        </div>
        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
          {directionIcon(entry.direction)}
          <span>{directionLabel(entry.direction)}</span>
          <span>·</span>
          <span>{entry.startTime}</span>
          {entry.duration > 0 && (
            <>
              <span>·</span>
              <span>{formatDuration(entry.duration)}</span>
            </>
          )}
        </div>
      </div>

      <div className="text-xs text-muted-foreground">{formatDate(entry.date)}</div>

      <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
        <button className="rounded-lg p-1.5 hover:bg-muted text-muted-foreground hover:text-foreground" title="Videoanruf" aria-label="Videoanruf">
          <Video className="h-4 w-4" />
        </button>
        <button className="rounded-lg p-1.5 hover:bg-muted text-muted-foreground hover:text-foreground" title="Audioanruf" aria-label="Audioanruf">
          <Phone className="h-4 w-4" />
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Settings Tab
// ---------------------------------------------------------------------------

function SettingsTab() {
  const [audioInput, setAudioInput] = useState('default')
  const [audioOutput, setAudioOutput] = useState('default')
  const [videoInput, setVideoInput] = useState('default')
  const [background, setBackground] = useState('none')

  return (
    <div className="max-w-xl mx-auto space-y-6">
      <div>
        <h3 className="text-sm font-medium text-foreground mb-3">Audio-Einstellungen</h3>
        <div className="space-y-3">
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Mikrofon</label>
            <select
              value={audioInput}
              onChange={(e) => setAudioInput(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
            >
              <option value="default">Standard-Mikrofon</option>
              <option value="headset">Headset-Mikrofon (USB)</option>
              <option value="webcam">Webcam-Mikrofon (HD Pro)</option>
            </select>
          </div>
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Lautsprecher</label>
            <select
              value={audioOutput}
              onChange={(e) => setAudioOutput(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
            >
              <option value="default">Standard-Lautsprecher</option>
              <option value="headset">Headset (USB)</option>
              <option value="speaker">Externer Lautsprecher</option>
            </select>
          </div>
        </div>
      </div>

      <div>
        <h3 className="text-sm font-medium text-foreground mb-3">Video-Einstellungen</h3>
        <div className="space-y-3">
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Kamera</label>
            <select
              value={videoInput}
              onChange={(e) => setVideoInput(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
            >
              <option value="default">Integrierte Webcam</option>
              <option value="hd">HD Pro Webcam C920</option>
              <option value="external">Externe Kamera (USB)</option>
            </select>
          </div>

          <div className="rounded-xl border border-border bg-muted/30 aspect-video flex items-center justify-center">
            <div className="text-center text-muted-foreground">
              <VideoOff className="mx-auto h-8 w-8 mb-2 opacity-50" />
              <p className="text-xs">Kamera-Vorschau</p>
            </div>
          </div>
        </div>
      </div>

      <div>
        <h3 className="text-sm font-medium text-foreground mb-3">Hintergrund</h3>
        <div className="grid grid-cols-4 gap-2">
          {[
            { id: 'none', label: 'Keiner' },
            { id: 'blur', label: 'Unscharf' },
            { id: 'office', label: 'Büro' },
            { id: 'nature', label: 'Natur' },
          ].map((bg) => (
            <button
              key={bg.id}
              onClick={() => setBackground(bg.id)}
              className={`rounded-lg border p-3 text-xs text-center transition-colors ${
                background === bg.id
                  ? 'border-primary bg-primary/10 text-primary font-medium'
                  : 'border-border text-muted-foreground hover:bg-muted'
              }`}
            >
              {bg.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main VideoPage
// ---------------------------------------------------------------------------

export default function VideoPage() {
  const [activeTab, setActiveTab] = useState<VideoTab>('active')
  const [historySearch, setHistorySearch] = useState('')
  const { meetings, callHistory } = useMeetingsStore()

  const liveMeetings = meetings.filter((m) => m.status === 'live')

  const filteredHistory = historySearch
    ? callHistory.filter((e) =>
        e.contactName.toLowerCase().includes(historySearch.toLowerCase()),
      )
    : callHistory

  // Group history by date
  const groupedHistory = filteredHistory.reduce<Record<string, CallHistoryEntry[]>>((acc, entry) => {
    const key = formatDate(entry.date)
    if (!acc[key]) acc[key] = []
    acc[key].push(entry)
    return acc
  }, {})

  const tabs: { key: VideoTab; label: string; icon: typeof Video }[] = [
    { key: 'active', label: 'Aktive Anrufe', icon: Video },
    { key: 'history', label: 'Anrufverlauf', icon: Clock },
    { key: 'settings', label: 'Einstellungen', icon: Settings },
  ]

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="border-b border-border/60 px-6 py-4">
        <PageHeader
          title="Video & Anrufe"
          description={liveMeetings.length > 0
            ? `${liveMeetings.length} aktive${liveMeetings.length === 1 ? 'r' : ''} Anruf${liveMeetings.length === 1 ? '' : 'e'}`
            : 'Keine aktiven Anrufe'}
          icon={Video}
          moduleId="meetings"
          actions={
            <div className="flex gap-2">
              <button className="inline-flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm font-medium text-foreground hover:bg-muted transition-colors">
                <Phone className="h-4 w-4" />
                Neuer Anruf
              </button>
              <button className="inline-flex items-center gap-2 rounded-xl bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors">
                <Video className="h-4 w-4" />
                Meeting starten
              </button>
            </div>
          }
        />
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-border/60 px-6">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={`inline-flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
              activeTab === tab.key
                ? 'border-primary text-primary tab-accent-active'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <tab.icon className="h-4 w-4" />
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {activeTab === 'active' && (
          <div className="animate-fade-up">
            {liveMeetings.length > 0 ? (
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {liveMeetings.map((m) => (
                  <ActiveCallCard key={m.id} meeting={m} />
                ))}
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-20 text-center">
                <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted mb-4">
                  <Video className="h-8 w-8 text-muted-foreground/50" />
                </div>
                <h3 className="text-sm font-medium text-foreground">Keine aktiven Anrufe</h3>
                <p className="mt-1 text-sm text-muted-foreground max-w-sm">
                  Starte einen neuen Anruf oder tritt einem geplanten Meeting bei.
                </p>
                <div className="mt-4 flex gap-2">
                  <button className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors">
                    <Plus className="h-4 w-4" />
                    Neues Meeting
                  </button>
                  <button className="inline-flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted transition-colors">
                    <ExternalLink className="h-4 w-4" />
                    Meeting beitreten
                  </button>
                </div>
              </div>
            )}

            {/* Upcoming meetings */}
            {meetings.filter((m) => m.status === 'scheduled' && m.isVideoCall).length > 0 && (
              <div className="mt-8">
                <h2 className="text-sm font-medium text-muted-foreground mb-3">Geplante Video-Meetings</h2>
                <div className="space-y-1">
                  {meetings
                    .filter((m) => m.status === 'scheduled' && m.isVideoCall)
                    .slice(0, 5)
                    .map((m) => (
                      <div
                        key={m.id}
                        className="flex items-center gap-3 rounded-lg px-3 py-2.5 hover:bg-muted/50 transition-colors group"
                      >
                        <div className="h-2 w-2 rounded-full" style={{ backgroundColor: m.color }} />
                        <div className="flex-1 min-w-0">
                          <span className="text-sm font-medium text-foreground">{m.title}</span>
                          <span className="ml-2 text-xs text-muted-foreground">{m.date} · {m.startTime} · {m.duration} Min</span>
                        </div>
                        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                          <Users className="h-3.5 w-3.5" />
                          {m.participants.length}
                        </div>
                        <button
                          onClick={() => useMeetingsStore.getState().joinMeeting(m.id)}
                          className="rounded-lg bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary opacity-0 group-hover:opacity-100 transition-opacity hover:bg-primary/20"
                        >
                          Beitreten
                        </button>
                      </div>
                    ))}
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === 'history' && (
          <div className="animate-fade-up">
            <div className="mb-4 relative max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Kontakt suchen..."
                value={historySearch}
                onChange={(e) => setHistorySearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>

            {Object.keys(groupedHistory).length > 0 ? (
              <div className="space-y-4">
                {Object.entries(groupedHistory).map(([dateLabel, entries]) => (
                  <div key={dateLabel}>
                    <h3 className="text-xs font-medium text-muted-foreground mb-1 px-3">{dateLabel}</h3>
                    <div className="space-y-0.5">
                      {entries.map((entry) => (
                        <CallHistoryRow key={entry.id} entry={entry} />
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-20 text-center">
                <Clock className="h-10 w-10 text-muted-foreground/40 mb-3" />
                <p className="text-sm text-muted-foreground">
                  {historySearch ? 'Keine Ergebnisse für diese Suche.' : 'Noch keine Anrufe im Verlauf.'}
                </p>
              </div>
            )}
          </div>
        )}

        {activeTab === 'settings' && <div className="animate-fade-up"><SettingsTab /></div>}
      </div>
    </div>
  )
}
