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
} from 'lucide-react'
import { toast } from 'sonner'
import { useMeetingsStore, type Meeting } from '@/stores/meetings'
import { ItemActions, ConfirmDialog, EmptyState, type ActionItem } from '@/components/shared'
import { MeetingFormDialog } from './MeetingFormDialog'
import { MeetingDetailPanel } from './MeetingDetailPanel'
import { MeetingRoomView } from './MeetingRoomView'

type FilterTab = 'all' | 'live' | 'scheduled' | 'past'

export default function MeetingsPage() {
  const { meetings, addMeeting, updateMeeting, deleteMeeting, cancelMeeting, duplicateMeeting } =
    useMeetingsStore()

  const [filter, setFilter] = useState<FilterTab>('all')
  const [search, setSearch] = useState('')

  // Dialog/panel state
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
      toast.success('Meeting geloescht')
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
        onClick: () => {
          setEditMeeting(m)
          setFormOpen(true)
        },
      })
    }

    actions.push({
      label: 'Duplizieren',
      icon: Copy,
      onClick: () => {
        duplicateMeeting(m.id)
        toast.success('Meeting dupliziert')
      },
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
      label: 'Loeschen',
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
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div>
          <h1 className="text-foreground">Meetings</h1>
          <p className="text-sm text-muted-foreground">Plane und verwalte deine Meetings</p>
        </div>
        <button
          onClick={() => {
            setEditMeeting(null)
            setFormOpen(true)
          }}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Plus className="h-4 w-4" />
          Neues Meeting
        </button>
      </div>

      {/* Search + Filters */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 mb-6">
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
      </div>

      {/* Live Meetings */}
      {liveMeetings.length > 0 && (filter === 'all' || filter === 'live') && (
        <section className="mb-8">
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

      {/* Scheduled */}
      {scheduledMeetings.length > 0 && (filter === 'all' || filter === 'scheduled') && (
        <section className="mb-8">
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

      {/* Past */}
      {pastMeetings.length > 0 && (filter === 'all' || filter === 'past') && (
        <section className="mb-8">
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

      {filtered.length === 0 && (
        <EmptyState
          icon={Video}
          title="Keine Meetings gefunden"
          description={search ? 'Versuche einen anderen Suchbegriff' : 'Erstelle dein erstes Meeting'}
          action={
            !search
              ? {
                  label: 'Neues Meeting',
                  onClick: () => {
                    setEditMeeting(null)
                    setFormOpen(true)
                  },
                }
              : undefined
          }
        />
      )}

      {/* Form Dialog */}
      <MeetingFormDialog
        open={formOpen}
        onOpenChange={(open) => {
          setFormOpen(open)
          if (!open) setEditMeeting(null)
        }}
        meeting={editMeeting}
        onSubmit={handleCreateSubmit}
      />

      {/* Detail Panel */}
      <MeetingDetailPanel
        meeting={selectedMeeting}
        open={!!selectedMeetingId}
        onClose={() => setSelectedMeetingId(null)}
        onEdit={(m) => {
          setSelectedMeetingId(null)
          setEditMeeting(m)
          setFormOpen(true)
        }}
        onDelete={(id) => {
          setSelectedMeetingId(null)
          setDeleteConfirmId(id)
        }}
        onJoin={(id) => {
          setSelectedMeetingId(null)
          setMeetingRoomId(id)
        }}
      />

      {/* Meeting Room */}
      {meetingRoomMeeting && (
        <MeetingRoomView
          meeting={meetingRoomMeeting}
          open={!!meetingRoomId}
          onLeave={() => {
            setMeetingRoomId(null)
            toast.info('Meeting verlassen')
          }}
        />
      )}

      {/* Delete Confirm */}
      <ConfirmDialog
        open={!!deleteConfirmId}
        onOpenChange={(open) => !open && setDeleteConfirmId(null)}
        title="Meeting loeschen?"
        description={`"${deleteTarget?.title}" wird unwiderruflich geloescht.`}
        confirmLabel="Loeschen"
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

  return (
    <div
      className={`rounded-lg border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer ${
        isLive ? 'border-error/50 shadow-[var(--shadow-card)]' : 'border-border'
      } ${isPast || isCancelled ? 'opacity-70' : ''}`}
      onClick={onDetails}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <div
            className={`flex h-9 w-9 items-center justify-center rounded-lg ${
              isLive ? 'bg-error/10 text-error' : 'bg-primary-light text-primary'
            }`}
          >
            <Video className="h-4 w-4" />
          </div>
          <div>
            <h4 className="text-sm font-medium text-foreground">{meeting.title}</h4>
            <p className="text-xs text-muted-foreground">{meeting.project}</p>
          </div>
        </div>
        <ItemActions
          items={actions}
          advancedLabel="Erweiterte Optionen"
          onAdvanced={onAdvanced}
        />
      </div>

      <p className="text-xs text-text-body mb-3 line-clamp-2">{meeting.description}</p>

      <div className="flex items-center gap-3 text-xs text-muted-foreground mb-3">
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
      </div>

      <div className="flex items-center justify-between">
        <div className="flex -space-x-2">
          {meeting.participants.slice(0, 4).map((p, i) => (
            <div
              key={i}
              className="flex h-7 w-7 items-center justify-center rounded-full border-2 border-card bg-primary-light text-[10px] font-medium text-primary"
              title={p.name}
            >
              {p.initials}
            </div>
          ))}
          {meeting.participants.length > 4 && (
            <div className="flex h-7 w-7 items-center justify-center rounded-full border-2 border-card bg-secondary text-[10px] font-medium text-muted-foreground">
              +{meeting.participants.length - 4}
            </div>
          )}
        </div>

        {isLive && onJoin && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              onJoin()
            }}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Phone className="h-3 w-3" />
            Beitreten
          </button>
        )}
        {!isLive && !isPast && !isCancelled && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              onDetails()
            }}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <ExternalLink className="h-3 w-3" />
            Details
          </button>
        )}
        {isCancelled && (
          <span className="text-xs text-red-500 font-medium">Abgesagt</span>
        )}
      </div>
    </div>
  )
}
