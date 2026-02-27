import { useState } from 'react'
import {
  Video,
  Plus,
  Clock,
  Calendar,
  Search,
  Phone,
  ExternalLink,
  Copy,
  Pencil,
  Trash2,
  Ban,
  LayoutGrid,
  List,
  MapPin,
  Users,
  Repeat,
  Mail,
} from 'lucide-react'
import { toast } from 'sonner'
import { useMeetingsStore, type Meeting } from '@/stores/meetings'
import { ItemActions, ConfirmDialog, EmptyState, PageHeader, type ActionItem } from '@/components/shared'
import { MeetingFormDialog } from './MeetingFormDialog'
import { MeetingDetailPanel } from './MeetingDetailPanel'
import { MeetingRoomView } from './MeetingRoomView'

type FilterTab = 'all' | 'live' | 'scheduled' | 'past'
type ViewMode = 'grid' | 'timeline'

// ── Helpers ──────────────────────────────────────────────────────

function getRelativeTime(date: string, startTime: string, duration: number, status: Meeting['status']): string | null {
  const now = new Date()
  const meetingStart = new Date(`${date}T${startTime}:00`)
  const meetingEnd = new Date(meetingStart.getTime() + duration * 60_000)
  const diffMs = meetingStart.getTime() - now.getTime()

  if (status === 'live') {
    const elapsedMin = Math.round((now.getTime() - meetingStart.getTime()) / 60_000)
    if (elapsedMin <= 0) return 'Startet jetzt'
    return `Läuft seit ${elapsedMin} Min`
  }

  if (status === 'cancelled' || status === 'past') return null
  if (diffMs < 0) return null

  const diffMin = Math.round(diffMs / 60_000)
  if (diffMin < 60) return `in ${diffMin} Min`
  const diffH = Math.round(diffMin / 60)
  if (diffH < 24) return `in ${diffH}h`
  const diffD = Math.round(diffH / 24)
  return `in ${diffD}d`
}

function groupByDate(meetings: Meeting[]): { label: string; date: string; meetings: Meeting[] }[] {
  const groups: Record<string, Meeting[]> = {}
  for (const m of meetings) {
    if (!groups[m.date]) groups[m.date] = []
    groups[m.date].push(m)
  }

  const today = new Date().toISOString().split('T')[0]
  const tomorrow = new Date(Date.now() + 86_400_000).toISOString().split('T')[0]

  return Object.entries(groups)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([date, items]) => {
      let label: string
      if (date === today) label = 'Heute'
      else if (date === tomorrow) label = 'Morgen'
      else label = new Date(date).toLocaleDateString('de-CH', { weekday: 'long', day: 'numeric', month: 'long' })
      return { label, date, meetings: items.sort((a, b) => a.startTime.localeCompare(b.startTime)) }
    })
}

// ── Main Component ───────────────────────────────────────────────

export default function MeetingsPage() {
  const { meetings, addMeeting, updateMeeting, deleteMeeting, cancelMeeting, duplicateMeeting } =
    useMeetingsStore()

  const [filter, setFilter] = useState<FilterTab>('all')
  const [search, setSearch] = useState('')
  const [viewMode, setViewMode] = useState<ViewMode>('grid')

  const [formOpen, setFormOpen] = useState(false)
  const [editMeeting, setEditMeeting] = useState<Meeting | null>(null)
  const [selectedMeetingId, setSelectedMeetingId] = useState<string | null>(null)
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)
  const [cancelConfirmId, setCancelConfirmId] = useState<string | null>(null)
  const [meetingRoomId, setMeetingRoomId] = useState<string | null>(null)

  const filtered = meetings.filter((m) => {
    if (filter !== 'all' && m.status !== filter) return false
    if (search && !m.title.toLowerCase().includes(search.toLowerCase())) return false
    return true
  })

  const liveMeetings = filtered.filter((m) => m.status === 'live')
  const scheduledMeetings = filtered.filter((m) => m.status === 'scheduled')
  const pastMeetings = filtered.filter((m) => m.status === 'past')

  const tabs: { key: FilterTab; label: string; count: number }[] = [
    { key: 'all', label: 'Alle', count: filtered.length },
    { key: 'live', label: 'Live', count: meetings.filter((m) => m.status === 'live').length },
    { key: 'scheduled', label: 'Geplant', count: meetings.filter((m) => m.status === 'scheduled').length },
    { key: 'past', label: 'Vergangen', count: meetings.filter((m) => m.status === 'past').length },
  ]

  const selectedMeeting = meetings.find((m) => m.id === selectedMeetingId) || null
  const meetingRoomMeeting = meetings.find((m) => m.id === meetingRoomId)
  const deleteTarget = meetings.find((m) => m.id === deleteConfirmId)
  const cancelTarget = meetings.find((m) => m.id === cancelConfirmId)

  const handleCreateSubmit = (data: Omit<Meeting, 'id'>) => {
    if (editMeeting) {
      updateMeeting(editMeeting.id, data)
      toast.success('Meeting aktualisiert')
    } else {
      addMeeting(data)
      toast.success('Meeting erstellt')
    }
    setEditMeeting(null)
  }

  const handleDelete = () => {
    if (deleteConfirmId) {
      deleteMeeting(deleteConfirmId)
      toast.success('Meeting gelöscht')
      setDeleteConfirmId(null)
      if (selectedMeetingId === deleteConfirmId) setSelectedMeetingId(null)
    }
  }

  const handleCancel = () => {
    if (cancelConfirmId) {
      cancelMeeting(cancelConfirmId)
      toast.success('Meeting abgesagt')
      setCancelConfirmId(null)
    }
  }

  const getMeetingActions = (m: Meeting): ActionItem[] => {
    const actions: ActionItem[] = []
    if (m.status === 'scheduled') {
      actions.push({
        label: 'Bearbeiten',
        icon: Pencil,
        onClick: () => { setEditMeeting(m); setFormOpen(true) },
      })
    }
    actions.push({
      label: 'Duplizieren',
      icon: Copy,
      onClick: () => { duplicateMeeting(m.id); toast.success('Meeting dupliziert') },
    })
    if (m.status !== 'past' && m.status !== 'cancelled') {
      actions.push({
        label: 'Absagen',
        icon: Ban,
        onClick: () => setCancelConfirmId(m.id),
        separator: true,
      })
    }
    actions.push({
      label: 'Löschen',
      icon: Trash2,
      variant: 'destructive',
      onClick: () => setDeleteConfirmId(m.id),
      separator: m.status === 'past' || m.status === 'cancelled',
    })
    return actions
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <PageHeader
        title="Meetings"
        description="Plane und verwalte deine Meetings"
        icon={Video}
        moduleId="meetings"
        actions={
          <button
            onClick={() => { setEditMeeting(null); setFormOpen(true) }}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            Neues Meeting
          </button>
        }
        className="mb-6"
      />

      {/* Search + Filters + View Toggle */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 mb-6 animate-fade-up" style={{ animationDelay: '50ms' }}>
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Meeting suchen..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
        </div>
        <div className="flex gap-1 rounded-lg border border-border bg-card p-1">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setFilter(tab.key)}
              className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
                filter === tab.key
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground hover:bg-secondary'
              }`}
            >
              {tab.label}
              {tab.key === 'live' && tab.count > 0 && (
                <span className="ml-1.5 inline-flex h-5 w-5 items-center justify-center rounded-full bg-error text-[10px] text-white">
                  {tab.count}
                </span>
              )}
            </button>
          ))}
        </div>
        <div className="flex gap-1 rounded-lg border border-border bg-card p-1">
          <button
            onClick={() => setViewMode('grid')}
            className={`rounded-md p-1.5 transition-colors ${viewMode === 'grid' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground hover:bg-secondary'}`}
            title="Grid-Ansicht"
          >
            <LayoutGrid className="h-4 w-4" />
          </button>
          <button
            onClick={() => setViewMode('timeline')}
            className={`rounded-md p-1.5 transition-colors ${viewMode === 'timeline' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground hover:bg-secondary'}`}
            title="Timeline-Ansicht"
          >
            <List className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* ── GRID VIEW ─────────────────────────────────────── */}
      {viewMode === 'grid' && (
        <>
          {liveMeetings.length > 0 && (filter === 'all' || filter === 'live') && (
            <section className="mb-8 animate-fade-up">
              <div className="flex items-center gap-2 mb-3">
                <span className="relative flex h-2.5 w-2.5">
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-error opacity-75" />
                  <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-error" />
                </span>
                <h3 className="text-sm font-medium text-foreground">Live jetzt</h3>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {liveMeetings.map((m) => (
                  <MeetingCard
                    key={m.id}
                    meeting={m}
                    actions={getMeetingActions(m)}
                    onJoin={() => setMeetingRoomId(m.id)}
                    onDetails={() => setSelectedMeetingId(m.id)}
                    onAdvanced={() => setSelectedMeetingId(m.id)}
                  />
                ))}
              </div>
            </section>
          )}

          {scheduledMeetings.length > 0 && (filter === 'all' || filter === 'scheduled') && (
            <section className="mb-8 animate-fade-up" style={{ animationDelay: '100ms' }}>
              <h3 className="text-sm font-medium text-muted-foreground mb-3">Geplante Meetings</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                {scheduledMeetings.map((m) => (
                  <MeetingCard
                    key={m.id}
                    meeting={m}
                    actions={getMeetingActions(m)}
                    onDetails={() => setSelectedMeetingId(m.id)}
                    onAdvanced={() => setSelectedMeetingId(m.id)}
                  />
                ))}
              </div>
            </section>
          )}

          {pastMeetings.length > 0 && (filter === 'all' || filter === 'past') && (
            <section className="mb-8 animate-fade-up" style={{ animationDelay: '200ms' }}>
              <h3 className="text-sm font-medium text-muted-foreground mb-3">Vergangene Meetings</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                {pastMeetings.map((m) => (
                  <MeetingCard
                    key={m.id}
                    meeting={m}
                    actions={getMeetingActions(m)}
                    onDetails={() => setSelectedMeetingId(m.id)}
                    onAdvanced={() => setSelectedMeetingId(m.id)}
                  />
                ))}
              </div>
            </section>
          )}
        </>
      )}

      {/* ── TIMELINE VIEW ─────────────────────────────────── */}
      {viewMode === 'timeline' && (
        <div className="space-y-6 animate-fade-up">
          {groupByDate(filtered).map((group) => (
            <section key={group.date}>
              <h3 className="text-sm font-semibold text-foreground mb-3 sticky top-0 bg-[var(--background)] py-1 z-[1]">
                {group.label}
              </h3>
              <div className="relative pl-6 border-l-2 border-border space-y-3">
                {group.meetings.map((m) => (
                  <TimelineRow
                    key={m.id}
                    meeting={m}
                    actions={getMeetingActions(m)}
                    onJoin={() => setMeetingRoomId(m.id)}
                    onDetails={() => setSelectedMeetingId(m.id)}
                    onAdvanced={() => setSelectedMeetingId(m.id)}
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
      )}

      {filtered.length === 0 && (
        <EmptyState
          icon={Video}
          title="Keine Meetings gefunden"
          description={search ? 'Versuche einen anderen Suchbegriff' : 'Erstelle dein erstes Meeting'}
          action={!search ? { label: 'Neues Meeting', onClick: () => { setEditMeeting(null); setFormOpen(true) } } : undefined}
        />
      )}

      {/* Form Dialog */}
      <MeetingFormDialog
        open={formOpen}
        onOpenChange={(open) => { setFormOpen(open); if (!open) setEditMeeting(null) }}
        meeting={editMeeting}
        onSubmit={handleCreateSubmit}
      />

      {/* Detail Panel */}
      <MeetingDetailPanel
        meeting={selectedMeeting}
        open={!!selectedMeetingId}
        onClose={() => setSelectedMeetingId(null)}
        onEdit={(m) => { setSelectedMeetingId(null); setEditMeeting(m); setFormOpen(true) }}
        onDelete={(id) => { setSelectedMeetingId(null); setDeleteConfirmId(id) }}
        onJoin={(id) => { setSelectedMeetingId(null); setMeetingRoomId(id) }}
      />

      {/* Meeting Room */}
      {meetingRoomMeeting && (
        <MeetingRoomView
          meeting={meetingRoomMeeting}
          open={!!meetingRoomId}
          onLeave={() => { setMeetingRoomId(null); toast.info('Meeting verlassen') }}
        />
      )}

      {/* Delete Confirm */}
      <ConfirmDialog
        open={!!deleteConfirmId}
        onOpenChange={(open) => !open && setDeleteConfirmId(null)}
        title="Meeting löschen?"
        description={`"${deleteTarget?.title}" wird unwiderruflich gelöscht.`}
        confirmLabel="Löschen"
        variant="destructive"
        onConfirm={handleDelete}
      />

      {/* Cancel Confirm */}
      <ConfirmDialog
        open={!!cancelConfirmId}
        onOpenChange={(open) => !open && setCancelConfirmId(null)}
        title="Meeting absagen?"
        description={`"${cancelTarget?.title}" wird abgesagt. Teilnehmer werden benachrichtigt.`}
        confirmLabel="Absagen"
        variant="warning"
        onConfirm={handleCancel}
      />
    </div>
  )
}

// ── Meeting Card (Grid View) ─────────────────────────────────────

function MeetingCard({
  meeting,
  actions,
  onJoin,
  onDetails,
  onAdvanced,
}: {
  meeting: Meeting
  actions: ActionItem[]
  onJoin?: () => void
  onDetails: () => void
  onAdvanced: () => void
}) {
  const isLive = meeting.status === 'live'
  const isPast = meeting.status === 'past'
  const isCancelled = meeting.status === 'cancelled'
  const relTime = getRelativeTime(meeting.date, meeting.startTime, meeting.duration, meeting.status)

  return (
    <div
      className={`rounded-xl border bg-card p-4 transition-all duration-200 hover:shadow-[var(--shadow-card-hover)] hover:-translate-y-0.5 cursor-pointer ${
        isLive ? 'border-error/50 shadow-[var(--shadow-card)]' : 'border-border'
      } ${isPast || isCancelled ? 'opacity-70' : ''}`}
      style={{ borderTop: `3px solid ${meeting.color}` }}
      onClick={onDetails}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2 min-w-0">
          <div
            className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg"
            style={{ backgroundColor: `${meeting.color}15`, color: meeting.color }}
          >
            <Video className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <h4 className="text-sm font-medium text-foreground truncate">{meeting.title}</h4>
            <p className="text-xs text-muted-foreground">{meeting.project}</p>
          </div>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {relTime && (
            <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
              isLive
                ? 'bg-error/10 text-error'
                : 'bg-primary/10 text-primary'
            }`}>
              {relTime}
            </span>
          )}
          <ItemActions items={actions} advancedLabel="Erweiterte Optionen" onAdvanced={onAdvanced} />
        </div>
      </div>

      <p className="text-xs text-text-body mb-3 line-clamp-2">{meeting.description}</p>

      <div className="flex items-center gap-3 text-xs text-muted-foreground mb-3 flex-wrap">
        <span className="flex items-center gap-1">
          <Calendar className="h-3 w-3" />
          {new Date(meeting.date).toLocaleDateString('de-CH')}
        </span>
        <span className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          {meeting.startTime} Uhr
        </span>
        <span className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          {meeting.duration >= 60 ? `${meeting.duration / 60} Std` : `${meeting.duration} Min`}
        </span>
        {meeting.recurrence !== 'none' && (
          <span className="flex items-center gap-1 text-primary">
            <Repeat className="h-3 w-3" />
            {meeting.recurrence === 'daily' ? 'Tgl.' : meeting.recurrence === 'weekly' ? 'Wchtl.' : 'Mtl.'}
          </span>
        )}
      </div>

      {(meeting.agenda?.length ?? 0) > 0 && (
        <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground mb-3">
          <div className="h-1 flex-1 rounded-full bg-border overflow-hidden">
            <div
              className="h-full rounded-full bg-primary transition-all"
              style={{ width: `${((meeting.agenda ?? []).filter((a) => a.done).length / (meeting.agenda?.length ?? 0)) * 100}%` }}
            />
          </div>
          <span>{(meeting.agenda ?? []).filter((a) => a.done).length}/{(meeting.agenda?.length ?? 0)}</span>
        </div>
      )}

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="flex -space-x-2">
            {(meeting.participants ?? []).slice(0, 4).map((p, i) => (
              <div
                key={i}
                className="flex h-7 w-7 items-center justify-center rounded-full border-2 border-card bg-primary-light text-[10px] font-medium text-primary"
                title={p.name}
              >
                {p.initials}
              </div>
            ))}
            {(meeting.participants?.length ?? 0) > 4 && (
              <div className="flex h-7 w-7 items-center justify-center rounded-full border-2 border-card bg-secondary text-[10px] font-medium text-muted-foreground">
                +{(meeting.participants?.length ?? 0) - 4}
              </div>
            )}
          </div>
          {meeting.invitationsSent && (
            <span title="Einladungen versendet" className="text-emerald-500">
              <Mail className="h-3.5 w-3.5" />
            </span>
          )}
        </div>

        {isLive && onJoin && (
          <button
            onClick={(e) => { e.stopPropagation(); onJoin() }}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Phone className="h-3 w-3" />
            Beitreten
          </button>
        )}
        {!isLive && !isPast && !isCancelled && (
          <button
            onClick={(e) => { e.stopPropagation(); onDetails() }}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <ExternalLink className="h-3 w-3" />
            Details
          </button>
        )}
        {isCancelled && <span className="text-xs text-red-500 font-medium">Abgesagt</span>}
      </div>
    </div>
  )
}

// ── Timeline Row ──────────────────────────────────────────────────

function TimelineRow({
  meeting,
  actions,
  onJoin,
  onDetails,
  onAdvanced,
}: {
  meeting: Meeting
  actions: ActionItem[]
  onJoin?: () => void
  onDetails: () => void
  onAdvanced: () => void
}) {
  const isLive = meeting.status === 'live'
  const isPast = meeting.status === 'past'
  const isCancelled = meeting.status === 'cancelled'
  const relTime = getRelativeTime(meeting.date, meeting.startTime, meeting.duration, meeting.status)

  return (
    <div
      className={`relative flex items-center gap-4 rounded-xl border bg-card px-4 py-3 cursor-pointer transition-all duration-200 hover:shadow-[var(--shadow-card-hover)] hover:-translate-y-0.5 ${
        isLive ? 'border-error/50' : 'border-border'
      } ${isPast || isCancelled ? 'opacity-70' : ''}`}
      style={{ borderLeft: `3px solid ${meeting.color}` }}
      onClick={onDetails}
    >
      {/* Time dot on the timeline */}
      <div
        className="absolute -left-[calc(1.5rem+5px)] h-2.5 w-2.5 rounded-full border-2 border-card"
        style={{ backgroundColor: meeting.color }}
      />

      {/* Time */}
      <div className="w-14 shrink-0 text-center">
        <span className="text-sm font-medium text-foreground">{meeting.startTime}</span>
        <p className="text-[10px] text-muted-foreground">
          {meeting.duration >= 60 ? `${meeting.duration / 60} Std` : `${meeting.duration} Min`}
        </p>
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <h4 className="text-sm font-medium text-foreground truncate">{meeting.title}</h4>
          {relTime && (
            <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium ${
              isLive ? 'bg-error/10 text-error' : 'bg-primary/10 text-primary'
            }`}>
              {relTime}
            </span>
          )}
        </div>
        <div className="flex items-center gap-3 text-xs text-muted-foreground mt-0.5">
          <span className="flex items-center gap-1">
            <MapPin className="h-3 w-3" />
            {meeting.room}
          </span>
          <span className="flex items-center gap-1">
            <Users className="h-3 w-3" />
            {(meeting.participants?.length ?? 0)}
          </span>
          <span className="text-muted-foreground">{meeting.project}</span>
          {meeting.recurrence !== 'none' && (
            <span className="flex items-center gap-1 text-primary">
              <Repeat className="h-3 w-3" />
            </span>
          )}
          {meeting.invitationsSent && (
            <span className="text-emerald-500" title="Einladungen versendet">
              <Mail className="h-3 w-3" />
            </span>
          )}
        </div>
      </div>

      {/* Participants */}
      <div className="hidden sm:flex -space-x-2 shrink-0">
        {(meeting.participants ?? []).slice(0, 3).map((p, i) => (
          <div
            key={i}
            className="flex h-7 w-7 items-center justify-center rounded-full border-2 border-card bg-primary-light text-[10px] font-medium text-primary"
            title={p.name}
          >
            {p.initials}
          </div>
        ))}
        {(meeting.participants?.length ?? 0) > 3 && (
          <div className="flex h-7 w-7 items-center justify-center rounded-full border-2 border-card bg-secondary text-[10px] font-medium text-muted-foreground">
            +{(meeting.participants?.length ?? 0) - 3}
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2 shrink-0">
        {isLive && onJoin && (
          <button
            onClick={(e) => { e.stopPropagation(); onJoin() }}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Phone className="h-3 w-3" />
            Beitreten
          </button>
        )}
        {isCancelled && <span className="text-xs text-red-500 font-medium">Abgesagt</span>}
        <ItemActions items={actions} advancedLabel="Erweiterte Optionen" onAdvanced={onAdvanced} />
      </div>
    </div>
  )
}
