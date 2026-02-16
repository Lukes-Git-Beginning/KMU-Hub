/**
 * MeetingsPage -- main meeting list with status filter tabs and search.
 *
 * Adapted from Darien's design (design/brainstorm) to use real backend API
 * via TanStack Query hooks from api/hooks/useMeetings.ts.
 */
import { useState, useMemo } from 'react'
import {
  Video,
  Plus,
  Clock,
  Calendar,
  Search,
  Phone,
  ExternalLink,
  Pencil,
  Trash2,
  Ban,
  Copy,
  Users,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useMeetings,
  useCreateMeeting,
  useDeleteMeeting,
} from '@/api/hooks/useMeetings'
import { useMeetingsUIStore } from '@/stores/meetings'
import { ItemActions, ConfirmDialog, EmptyState, type ActionItem } from '@/components/shared'
import { MeetingFormDialog } from './MeetingFormDialog'
import { MeetingDetailPanel } from './MeetingDetailPanel'
import type { Meeting, MeetingStatus } from '@/api/video-types'

type FilterTab = 'all' | 'in_progress' | 'scheduled' | 'completed'

/** Map backend MeetingStatus to filter tabs. */
function statusToFilter(status: MeetingStatus): FilterTab {
  if (status === 'in_progress') return 'in_progress'
  if (status === 'scheduled') return 'scheduled'
  if (status === 'completed') return 'completed'
  return 'all'
}

/** Format ISO date to German locale display. */
function formatDate(isoDate: string): string {
  return new Date(isoDate).toLocaleDateString('de-CH', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

/** Format ISO datetime to time string. */
function formatTime(isoDate: string): string {
  return new Date(isoDate).toLocaleTimeString('de-CH', {
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** Calculate duration in minutes between two ISO datetimes. */
function durationMinutes(start: string, end: string): number {
  return Math.round((new Date(end).getTime() - new Date(start).getTime()) / 60000)
}

/** Format duration in human-readable form. */
function formatDuration(minutes: number): string {
  if (minutes >= 60) {
    const hours = Math.floor(minutes / 60)
    const remainder = minutes % 60
    return remainder > 0 ? `${hours} Std ${remainder} Min` : `${hours} Std`
  }
  return `${minutes} Min`
}

export default function MeetingsPage() {
  const { data: meetings = [], isLoading } = useMeetings()
  const deleteMutation = useDeleteMeeting()
  const createMutation = useCreateMeeting()

  const {
    selectedMeetingId,
    isFormOpen,
    editMeetingId,
    deleteConfirmId,
    selectMeeting,
    openForm,
    closeForm,
    setDeleteConfirmId,
  } = useMeetingsUIStore()

  const [filter, setFilter] = useState<FilterTab>('all')
  const [search, setSearch] = useState('')

  // Filtered meetings based on tab and search
  const filtered = useMemo(() => {
    return meetings.filter((m) => {
      if (filter !== 'all' && statusToFilter(m.status) !== filter) return false
      if (search && !m.title.toLowerCase().includes(search.toLowerCase())) return false
      return true
    })
  }, [meetings, filter, search])

  // Group meetings by status for display
  const liveMeetings = useMemo(
    () => filtered.filter((m) => m.status === 'in_progress'),
    [filtered],
  )
  const scheduledMeetings = useMemo(
    () => filtered.filter((m) => m.status === 'scheduled'),
    [filtered],
  )
  const completedMeetings = useMemo(
    () => filtered.filter((m) => m.status === 'completed' || m.status === 'cancelled'),
    [filtered],
  )

  const tabs: { key: FilterTab; label: string; count: number }[] = [
    { key: 'all', label: 'Alle', count: meetings.length },
    {
      key: 'in_progress',
      label: 'Live',
      count: meetings.filter((m) => m.status === 'in_progress').length,
    },
    {
      key: 'scheduled',
      label: 'Geplant',
      count: meetings.filter((m) => m.status === 'scheduled').length,
    },
    {
      key: 'completed',
      label: 'Vergangen',
      count: meetings.filter((m) => m.status === 'completed' || m.status === 'cancelled').length,
    },
  ]

  const selectedMeeting = meetings.find((m) => m.id === selectedMeetingId) ?? null
  const editMeeting = editMeetingId ? meetings.find((m) => m.id === editMeetingId) ?? null : null
  const deleteTarget = meetings.find((m) => m.id === deleteConfirmId)

  const handleDelete = () => {
    if (deleteConfirmId) {
      deleteMutation.mutate(deleteConfirmId, {
        onSuccess: () => {
          toast.success('Meeting geloescht')
          setDeleteConfirmId(null)
          if (selectedMeetingId === deleteConfirmId) selectMeeting(null)
        },
        onError: () => toast.error('Fehler beim Loeschen'),
      })
    }
  }

  const handleDuplicate = (meeting: Meeting) => {
    createMutation.mutate(
      {
        title: `${meeting.title} (Kopie)`,
        description: meeting.description ?? undefined,
        agenda: meeting.agenda ?? undefined,
        scheduled_start: meeting.scheduled_start,
        scheduled_end: meeting.scheduled_end,
        attendee_ids: meeting.attendees?.map((a) => a.user_id) ?? [],
      },
      {
        onSuccess: () => toast.success('Meeting dupliziert'),
        onError: () => toast.error('Fehler beim Duplizieren'),
      },
    )
  }

  const getMeetingActions = (m: Meeting): ActionItem[] => {
    const actions: ActionItem[] = []

    if (m.status === 'scheduled') {
      actions.push({
        label: 'Bearbeiten',
        icon: Pencil,
        onClick: () => openForm(m.id),
      })
    }

    actions.push({
      label: 'Duplizieren',
      icon: Copy,
      onClick: () => handleDuplicate(m),
    })

    if (m.status !== 'completed' && m.status !== 'cancelled') {
      actions.push({
        label: 'Absagen',
        icon: Ban,
        onClick: () => setDeleteConfirmId(m.id),
        separator: true,
      })
    }

    actions.push({
      label: 'Loeschen',
      icon: Trash2,
      variant: 'destructive',
      onClick: () => setDeleteConfirmId(m.id),
      separator: m.status === 'completed' || m.status === 'cancelled',
    })

    return actions
  }

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="animate-spin h-8 w-8 border-2 border-primary border-t-transparent rounded-full" />
      </div>
    )
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
          onClick={() => openForm()}
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
              {tab.key === 'in_progress' && tab.count > 0 && (
                <span className="ml-1.5 inline-flex h-5 w-5 items-center justify-center rounded-full bg-error text-[10px] text-white">
                  {tab.count}
                </span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* Live Meetings */}
      {liveMeetings.length > 0 && (filter === 'all' || filter === 'in_progress') && (
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
                onDetails={() => selectMeeting(m.id)}
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
                onDetails={() => selectMeeting(m.id)}
              />
            ))}
          </div>
        </section>
      )}

      {/* Completed / Cancelled */}
      {completedMeetings.length > 0 && (filter === 'all' || filter === 'completed') && (
        <section className="mb-8">
          <h3 className="text-sm font-medium text-muted-foreground mb-3">Vergangene Meetings</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
            {completedMeetings.map((m) => (
              <MeetingCard
                key={m.id}
                meeting={m}
                actions={getMeetingActions(m)}
                onDetails={() => selectMeeting(m.id)}
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
                  onClick: () => openForm(),
                }
              : undefined
          }
        />
      )}

      {/* Form Dialog */}
      <MeetingFormDialog
        open={isFormOpen}
        onOpenChange={(open) => {
          if (!open) closeForm()
        }}
        meeting={editMeeting}
      />

      {/* Detail Panel */}
      <MeetingDetailPanel
        meeting={selectedMeeting}
        open={!!selectedMeetingId}
        onClose={() => selectMeeting(null)}
        onEdit={(m) => {
          selectMeeting(null)
          openForm(m.id)
        }}
        onDelete={(id) => {
          selectMeeting(null)
          setDeleteConfirmId(id)
        }}
      />

      {/* Delete Confirm */}
      <ConfirmDialog
        open={!!deleteConfirmId}
        onOpenChange={(open) => !open && setDeleteConfirmId(null)}
        title="Meeting loeschen?"
        description={`"${deleteTarget?.title ?? ''}" wird unwiderruflich geloescht.`}
        confirmLabel="Loeschen"
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// MeetingCard sub-component
// ---------------------------------------------------------------------------

function MeetingCard({
  meeting,
  actions,
  onDetails,
}: {
  meeting: Meeting
  actions: ActionItem[]
  onDetails: () => void
}) {
  const isLive = meeting.status === 'in_progress'
  const isPast = meeting.status === 'completed'
  const isCancelled = meeting.status === 'cancelled'
  const attendeeCount = meeting.attendees?.length ?? 0
  const duration = durationMinutes(meeting.scheduled_start, meeting.scheduled_end)

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
            {meeting.description && (
              <p className="text-xs text-muted-foreground line-clamp-1">
                {meeting.description}
              </p>
            )}
          </div>
        </div>
        <ItemActions
          items={actions}
          advancedLabel="Erweiterte Optionen"
          onAdvanced={onDetails}
        />
      </div>

      {meeting.agenda && (
        <p className="text-xs text-text-body mb-3 line-clamp-2">{meeting.agenda}</p>
      )}

      <div className="flex items-center gap-3 text-xs text-muted-foreground mb-3">
        <span className="flex items-center gap-1">
          <Calendar className="h-3 w-3" />
          {formatDate(meeting.scheduled_start)}
        </span>
        <span className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          {formatTime(meeting.scheduled_start)} Uhr
        </span>
        <span className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          {formatDuration(duration)}
        </span>
      </div>

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <Users className="h-3.5 w-3.5" />
          <span>
            {attendeeCount} {attendeeCount === 1 ? 'Teilnehmer' : 'Teilnehmer'}
          </span>
        </div>

        {isLive && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              onDetails()
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
