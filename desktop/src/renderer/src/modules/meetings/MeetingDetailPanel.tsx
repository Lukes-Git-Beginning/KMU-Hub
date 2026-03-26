import { useState, useEffect, useCallback, useRef } from 'react'
import {
  Calendar,
  Clock,
  MapPin,
  Video,
  Users,
  FileText,
  FolderOpen,
  Presentation,
  Pencil,
  Trash2,
  X,
  Link2,
  Crown,
  Plus,
  ChevronUp,
  ChevronDown,
  CheckSquare,
  Square,
  ListChecks,
  StickyNote,
  Repeat,
  Check,
  ExternalLink,
  Mail,
  Send,
} from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { LazyRichTextEditor as RichTextEditor } from '@/components/shared/RichTextEditor'
import { useMeetingsStore } from '@/stores/meetings'
import type { Meeting } from '@/stores/meetings'

interface MeetingDetailPanelProps {
  meeting: Meeting | null
  open: boolean
  onClose: () => void
  onEdit: (meeting: Meeting) => void
  onDelete: (meetingId: string) => void
  onJoin: (meetingId: string) => void
}

const statusConfig = {
  live: { label: 'Live', dot: 'bg-destructive' },
  scheduled: { label: 'Geplant', dot: 'bg-blue-500' },
  past: { label: 'Vergangen', dot: 'bg-gray-400' },
  cancelled: { label: 'Abgesagt', dot: 'bg-gray-400' },
}

const recurrenceLabels: Record<string, string> = {
  none: 'Einmalig',
  daily: 'Täglich',
  weekly: 'Wöchentlich',
  monthly: 'Monatlich',
}

const reminderLabels: Record<string, string> = {
  none: 'Keine',
  '15min': '15 Min vorher',
  '30min': '30 Min vorher',
  '1h': '1 Std vorher',
}

type TabId = 'details' | 'agenda' | 'notes'

/** Darken a hex color by a given amount (0-1) */
function darkenColor(hex: string, amount: number): string {
  const c = hex.replace('#', '')
  const num = parseInt(c, 16)
  const r = Math.max(0, Math.round(((num >> 16) & 0xff) * (1 - amount)))
  const g = Math.max(0, Math.round(((num >> 8) & 0xff) * (1 - amount)))
  const b = Math.max(0, Math.round((num & 0xff) * (1 - amount)))
  return `#${((r << 16) | (g << 8) | b).toString(16).padStart(6, '0')}`
}

export function MeetingDetailPanel({
  meeting,
  open,
  onClose,
  onEdit,
  onDelete,
  onJoin,
}: MeetingDetailPanelProps) {
  const [activeTab, setActiveTab] = useState<TabId>('details')
  const [newAgendaText, setNewAgendaText] = useState('')
  const [notesValue, setNotesValue] = useState('')
  const notesTimeoutRef = useRef<ReturnType<typeof setTimeout>>()

  const toggleAgendaItem = useMeetingsStore((s) => s.toggleAgendaItem)
  const addAgendaItem = useMeetingsStore((s) => s.addAgendaItem)
  const removeAgendaItem = useMeetingsStore((s) => s.removeAgendaItem)
  const reorderAgendaItem = useMeetingsStore((s) => s.reorderAgendaItem)
  const updateNotes = useMeetingsStore((s) => s.updateNotes)
  const updateMeeting = useMeetingsStore((s) => s.updateMeeting)

   
  useEffect(() => {
     
    if (meeting) setNotesValue(meeting.notes)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentionally depending on meeting.id and meeting.notes only
  }, [meeting?.id, meeting?.notes])

   
  useEffect(() => {
     
    setActiveTab('details')
    setNewAgendaText('')
  }, [meeting?.id])

  // ESC to close
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    },
    [onClose]
  )

  useEffect(() => {
    if (open) {
      document.addEventListener('keydown', handleKeyDown)
      return () => document.removeEventListener('keydown', handleKeyDown)
    }
  }, [open, handleKeyDown])

  if (!open || !meeting) return null

  const status = statusConfig[meeting.status]
  const agendaDone = (meeting.agenda ?? []).filter((a) => a.done).length
  const agendaTotal = (meeting.agenda?.length ?? 0)
  const handleAddAgenda = () => {
    const text = newAgendaText.trim()
    if (!text) return
    addAgendaItem(meeting.id, text)
    setNewAgendaText('')
  }

  const handleNotesChange = (value: string) => {
    setNotesValue(value)
    if (notesTimeoutRef.current) clearTimeout(notesTimeoutRef.current)
    notesTimeoutRef.current = setTimeout(() => {
      updateNotes(meeting.id, value)
    }, 500)
  }

  const handleCopyLink = () => {
    navigator.clipboard.writeText(`https://meet.kmuhub.ch/${meeting.id}`)
  }

  const tabs: { id: TabId; label: string; icon: typeof ListChecks }[] = [
    { id: 'details', label: 'Details', icon: FileText },
    { id: 'agenda', label: 'Agenda', icon: ListChecks },
    { id: 'notes', label: 'Notizen', icon: StickyNote },
  ]

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/20 transition-opacity"
        onClick={onClose}
      />

      {/* Panel */}
      <div className="fixed right-0 top-0 z-50 flex h-full w-[420px] flex-col border-l shadow-xl animate-in slide-in-from-right duration-200 bg-[var(--card)]">
        {/* ── Gradient Header ── */}
        <div
          className="shrink-0 px-5 pt-4 pb-5"
          style={{
            background: `linear-gradient(135deg, ${meeting.color}, ${darkenColor(meeting.color, 0.3)})`,
          }}
        >
          {/* Top row: close + actions */}
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-1.5">
              <button
                onClick={() => onEdit(meeting)}
                className="rounded-md p-1.5 text-white/70 hover:text-white transition-colors"
                title="Bearbeiten"
              >
                <Pencil className="h-4 w-4" />
              </button>
              <button
                onClick={() => onDelete(meeting.id)}
                className="rounded-md p-1.5 text-white/70 hover:text-red-300 transition-colors"
                title="Löschen"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
            <button
              onClick={onClose}
              className="rounded-md p-1.5 text-white/70 hover:text-white transition-colors"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          {/* Icon + Title + Status */}
          <div className="flex items-start gap-3">
            <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-white/20 text-white">
              <Video className="h-6 w-6" />
            </div>
            <div className="min-w-0 flex-1">
              <h2 className="text-base font-semibold text-white leading-tight truncate">
                {meeting.title}
              </h2>
              {meeting.project && (
                <p className="text-sm text-white/70 truncate mt-0.5">{meeting.project}</p>
              )}
              <div className="flex items-center gap-2 mt-2">
                <span className="inline-flex items-center gap-1.5 rounded-full bg-white/20 px-2.5 py-0.5 text-xs font-medium text-white">
                  <span className={`inline-block h-1.5 w-1.5 rounded-full ${status.dot}`} />
                  {status.label}
                </span>
                {meeting.isVideoCall && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-white/20 px-2 py-0.5 text-xs text-white">
                    <Video className="h-3 w-3" />
                    Video
                  </span>
                )}
              </div>
            </div>
          </div>

          {/* Participant avatars */}
          <div className="flex items-center gap-1.5 mt-4">
            {(meeting.participants ?? []).map((p) => (
              <div
                key={p.id}
                className="relative group"
                title={`${p.name}${p.id === meeting.organizerId ? ' (Organisator)' : ''}`}
              >
                <span className="flex h-8 w-8 items-center justify-center rounded-full border-2 border-white/30 bg-white/20 text-[10px] font-medium text-white">
                  {p.initials}
                </span>
                {p.id === meeting.organizerId && (
                  <Crown className="absolute -top-1 -right-1 h-3.5 w-3.5 text-yellow-300 fill-yellow-300" />
                )}
              </div>
            ))}
            <span className="ml-1 text-xs text-white/60">
              {(meeting.participants?.length ?? 0)} Teilnehmer
            </span>
          </div>

          {/* Quick actions */}
          <div className="flex gap-2 mt-4">
            {meeting.status === 'live' && (
              <button
                onClick={() => onJoin(meeting.id)}
                className="flex items-center gap-1.5 rounded-lg bg-white/25 px-3 py-1.5 text-xs font-medium text-white hover:bg-white/35 transition-colors"
              >
                <Video className="h-3.5 w-3.5" />
                Beitreten
              </button>
            )}
            {meeting.status === 'scheduled' && (
              <button
                onClick={() => onEdit(meeting)}
                className="flex items-center gap-1.5 rounded-lg bg-white/20 px-3 py-1.5 text-xs text-white hover:bg-white/30 transition-colors"
              >
                <Pencil className="h-3.5 w-3.5" />
                Bearbeiten
              </button>
            )}
            <button
              onClick={handleCopyLink}
              className="flex items-center gap-1.5 rounded-lg bg-white/20 px-3 py-1.5 text-xs text-white hover:bg-white/30 transition-colors"
            >
              <Link2 className="h-3.5 w-3.5" />
              Link kopieren
            </button>
          </div>
        </div>

        {/* ── Tab Bar ── */}
        <div className="shrink-0 flex border-b">
          {tabs.map((tab) => {
            const Icon = tab.icon
            const isActive = activeTab === tab.id
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex-1 flex items-center justify-center gap-1.5 py-2.5 text-xs font-medium transition-colors border-b-2 ${
                  isActive
                    ? 'border-primary text-primary'
                    : 'border-transparent text-[var(--muted)] hover:text-[var(--body)]'
                }`}
              >
                <Icon className="h-3.5 w-3.5" />
                {tab.label}
                {tab.id === 'agenda' && agendaTotal > 0 && (
                  <span className={`ml-0.5 text-[10px] ${isActive ? 'text-primary/70' : 'text-[var(--muted)]'}`}>
                    {agendaDone}/{agendaTotal}
                  </span>
                )}
              </button>
            )
          })}
        </div>

        {/* ── Tab Content ── */}
        <div className="flex-1 overflow-y-auto p-4">
          {activeTab === 'details' && <DetailsTab meeting={meeting} onUpdateMeeting={updateMeeting} />}
          {activeTab === 'agenda' && (
            <AgendaTab
              meeting={meeting}
              newText={newAgendaText}
              onNewTextChange={setNewAgendaText}
              onAdd={handleAddAgenda}
              onToggle={(agendaId) => toggleAgendaItem(meeting.id, agendaId)}
              onRemove={(agendaId) => removeAgendaItem(meeting.id, agendaId)}
              onReorder={(agendaId, dir) => reorderAgendaItem(meeting.id, agendaId, dir)}
            />
          )}
          {activeTab === 'notes' && (
            <NotesTab
              value={notesValue}
              onChange={handleNotesChange}
              isPast={meeting.status === 'past'}
            />
          )}
        </div>
      </div>
    </>
  )
}

/* ═══════════════════════════════════════════════════════════════
   Details Tab
   ═══════════════════════════════════════════════════════════════ */

function DetailsTab({ meeting, onUpdateMeeting }: { meeting: Meeting; onUpdateMeeting: (id: string, updates: Partial<Meeting>) => void }) {
  const handleAddToCalendar = () => {
    onUpdateMeeting(meeting.id, { calendarEventId: `cal-${meeting.id}` })
    toast.success('Meeting wurde dem Kalender hinzugefuegt')
  }

  const handleSendInvitations = () => {
    onUpdateMeeting(meeting.id, { invitationsSent: true })
    toast.success(`Einladungen an ${(meeting.participants?.length ?? 0)} Teilnehmer versendet (inkl. .ics Kalendereinladung)`)
  }

  return (
    <div className="space-y-4">
      {/* Date / Time / Location */}
      <div className="space-y-2.5">
        <div className="flex items-center gap-2.5 text-sm">
          <Calendar className="h-4 w-4 text-[var(--muted)] shrink-0" />
          <span className="text-[var(--body)]">
            {new Date(meeting.date).toLocaleDateString('de-CH', {
              weekday: 'long',
              day: 'numeric',
              month: 'long',
              year: 'numeric',
            })}
          </span>
        </div>
        <div className="flex items-center gap-2.5 text-sm">
          <Clock className="h-4 w-4 text-[var(--muted)] shrink-0" />
          <span className="text-[var(--body)]">
            {meeting.startTime} Uhr &middot;{' '}
            {meeting.duration >= 60
              ? `${meeting.duration / 60} Std`
              : `${meeting.duration} Min`}
          </span>
        </div>
        <div className="flex items-center gap-2.5 text-sm">
          <MapPin className="h-4 w-4 text-[var(--muted)] shrink-0" />
          <span className="text-[var(--body)]">{meeting.room}</span>
        </div>
        {meeting.recurrence !== 'none' && (
          <div className="flex items-center gap-2.5 text-sm">
            <Repeat className="h-4 w-4 text-primary shrink-0" />
            <span className="text-[var(--body)]">
              Wiederholt sich {recurrenceLabels[meeting.recurrence].toLowerCase()}
            </span>
          </div>
        )}
        <div className="flex items-center gap-2.5 text-sm">
          <Clock className="h-4 w-4 text-[var(--muted)] shrink-0" />
          <span className="text-[var(--body)]">
            Erinnerung: {reminderLabels[meeting.reminder]}
          </span>
        </div>
      </div>

      <Separator />

      {/* Project */}
      {meeting.project && (
        <>
          <div>
            <h4 className="mb-1.5 text-xs font-medium uppercase text-[var(--muted)]">Projekt</h4>
            <div className="flex items-center gap-2 text-sm text-primary">
              <FolderOpen className="h-4 w-4" />
              {meeting.project}
            </div>
          </div>
          <Separator />
        </>
      )}

      {/* ── Kalender-Synchronisation (10.18) ── */}
      <div>
        <h4 className="mb-2 text-xs font-medium uppercase text-[var(--muted)]">
          <Calendar className="mr-1 inline h-3.5 w-3.5" />
          Kalender
        </h4>
        {meeting.calendarEventId ? (
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <Check className="h-4 w-4 text-success shrink-0" />
              <span className="text-success font-medium">Im Kalender eingetragen</span>
            </div>
            <button className="flex items-center gap-1.5 text-xs text-primary hover:underline transition-colors">
              <ExternalLink className="h-3 w-3" />
              Im Kalender oeffnen
            </button>
          </div>
        ) : (
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Calendar className="h-4 w-4 shrink-0" />
              <span>Nicht im Kalender</span>
            </div>
            <Button variant="outline" size="sm" onClick={handleAddToCalendar} className="w-full">
              <Calendar className="mr-1.5 h-4 w-4" />
              Zum Kalender hinzufuegen
            </Button>
          </div>
        )}
      </div>

      <Separator />

      {/* ── Einladungs-E-Mails (10.19) ── */}
      <div>
        <h4 className="mb-2 text-xs font-medium uppercase text-[var(--muted)]">
          <Mail className="mr-1 inline h-3.5 w-3.5" />
          Einladungen
        </h4>
        {meeting.invitationsSent ? (
          <div className="space-y-2">
            <span className="inline-flex items-center gap-1.5 rounded-full bg-success-light px-2.5 py-1 text-xs font-medium text-success">
              <Check className="h-3 w-3" />
              Einladungen versendet
            </span>
            <div className="space-y-1.5 mt-2">
              {(meeting.participants ?? []).map((p) => (
                <div key={p.id} className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-[9px] font-medium text-primary">
                      {p.initials}
                    </span>
                    <span className="text-xs text-[var(--body)]">{p.name}</span>
                  </div>
                  <span className="inline-flex items-center gap-1 text-[10px] text-success">
                    <Check className="h-2.5 w-2.5" />
                    Gesendet
                  </span>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div className="space-y-2">
            <span className="inline-flex items-center gap-1.5 rounded-full bg-warning-light px-2.5 py-1 text-xs font-medium text-warning-foreground">
              <Mail className="h-3 w-3" />
              Einladungen ausstehend
            </span>
            <div className="space-y-1.5 mt-2">
              {(meeting.participants ?? []).map((p) => (
                <div key={p.id} className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-[9px] font-medium text-primary">
                      {p.initials}
                    </span>
                    <span className="text-xs text-[var(--body)]">{p.name}</span>
                  </div>
                  <span className="text-[10px] text-muted-foreground">Ausstehend</span>
                </div>
              ))}
            </div>
            <Button variant="outline" size="sm" onClick={handleSendInvitations} className="w-full mt-1">
              <Send className="mr-1.5 h-4 w-4" />
              Einladungen senden
            </Button>
          </div>
        )}
      </div>

      <Separator />

      {/* Participants list */}
      <div>
        <h4 className="mb-2 text-xs font-medium uppercase text-[var(--muted)]">
          <Users className="mr-1 inline h-3.5 w-3.5" />
          Teilnehmer ({(meeting.participants?.length ?? 0)})
        </h4>
        <div className="space-y-1.5">
          {(meeting.participants ?? []).map((p) => (
            <div key={p.id} className="flex items-center gap-2">
              <span className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                {p.initials}
              </span>
              <span className="text-sm text-[var(--body)]">{p.name}</span>
              {p.id === meeting.organizerId && (
                <span className="inline-flex items-center gap-0.5 rounded-full bg-warning-light px-1.5 py-0.5 text-[10px] font-medium text-warning-foreground">
                  <Crown className="h-2.5 w-2.5" />
                  Organisator
                </span>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Description */}
      {meeting.description && (
        <>
          <Separator />
          <div>
            <h4 className="mb-1.5 text-xs font-medium uppercase text-[var(--muted)]">
              Beschreibung
            </h4>
            <p className="text-sm text-[var(--body)] whitespace-pre-wrap">
              {meeting.description}
            </p>
          </div>
        </>
      )}

      {/* Files */}
      {meeting.files.length > 0 && (
        <>
          <Separator />
          <div>
            <h4 className="mb-2 text-xs font-medium uppercase text-[var(--muted)]">
              <FileText className="mr-1 inline h-3.5 w-3.5" />
              Dateien ({meeting.files.length})
            </h4>
            <div className="space-y-1.5">
              {meeting.files.map((f) => (
                <div
                  key={f.id}
                  className="flex items-center justify-between rounded-md border border-border px-3 py-2 text-sm hover:bg-secondary/50 cursor-pointer"
                >
                  <span className="text-[var(--body)]">{f.name}</span>
                  <span className="text-xs text-[var(--muted)]">{f.size}</span>
                </div>
              ))}
            </div>
          </div>
        </>
      )}

      {/* Whiteboard */}
      <Separator />
      <div>
        <h4 className="mb-2 text-xs font-medium uppercase text-[var(--muted)]">
          <Presentation className="mr-1 inline h-3.5 w-3.5" />
          Whiteboard
        </h4>
        <Button variant="outline" size="sm" className="w-full">
          <Presentation className="mr-1.5 h-4 w-4" />
          Whiteboard starten
        </Button>
      </div>
    </div>
  )
}

/* ═══════════════════════════════════════════════════════════════
   Agenda Tab
   ═══════════════════════════════════════════════════════════════ */

interface AgendaTabProps {
  meeting: Meeting
  newText: string
  onNewTextChange: (text: string) => void
  onAdd: () => void
  onToggle: (agendaId: string) => void
  onRemove: (agendaId: string) => void
  onReorder: (agendaId: string, direction: 'up' | 'down') => void
}

function AgendaTab({
  meeting,
  newText,
  onNewTextChange,
  onAdd,
  onToggle,
  onRemove,
  onReorder,
}: AgendaTabProps) {
  const doneCount = (meeting.agenda ?? []).filter((a) => a.done).length
  const total = (meeting.agenda?.length ?? 0)
  const progress = total > 0 ? (doneCount / total) * 100 : 0

  return (
    <div className="space-y-4">
      {/* Progress */}
      {total > 0 && (
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <span className="text-xs text-[var(--muted)]">Fortschritt</span>
            <span className="text-xs font-medium text-[var(--body)]">
              {doneCount}/{total} Punkte erledigt
            </span>
          </div>
          <div className="h-1.5 w-full rounded-full bg-secondary overflow-hidden">
            <div
              className="h-full rounded-full bg-primary transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
      )}

      {/* Agenda items */}
      <div className="space-y-1">
        {(meeting.agenda ?? []).map((item, idx) => (
          <div
            key={item.id}
            className="group flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-secondary/50 transition-colors"
          >
            <button
              onClick={() => onToggle(item.id)}
              className="shrink-0 text-[var(--muted)] hover:text-primary transition-colors"
            >
              {item.done ? (
                <CheckSquare className="h-4 w-4 text-primary" />
              ) : (
                <Square className="h-4 w-4" />
              )}
            </button>
            <span
              className={`flex-1 text-sm ${
                item.done
                  ? 'line-through text-[var(--muted)]'
                  : 'text-[var(--body)]'
              }`}
            >
              {item.text}
            </span>
            <div className="hidden group-hover:flex items-center gap-0.5">
              {idx > 0 && (
                <button
                  onClick={() => onReorder(item.id, 'up')}
                  className="rounded p-0.5 text-[var(--muted)] hover:text-[var(--body)] transition-colors"
                  title="Nach oben"
                >
                  <ChevronUp className="h-3.5 w-3.5" />
                </button>
              )}
              {idx < (meeting.agenda?.length ?? 0) - 1 && (
                <button
                  onClick={() => onReorder(item.id, 'down')}
                  className="rounded p-0.5 text-[var(--muted)] hover:text-[var(--body)] transition-colors"
                  title="Nach unten"
                >
                  <ChevronDown className="h-3.5 w-3.5" />
                </button>
              )}
              <button
                onClick={() => onRemove(item.id)}
                className="rounded p-0.5 text-[var(--muted)] hover:text-destructive transition-colors"
                title="Entfernen"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        ))}
      </div>

      {/* Add new item */}
      <div className="flex items-center gap-2">
        <input
          type="text"
          value={newText}
          onChange={(e) => onNewTextChange(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && onAdd()}
          placeholder="Neuen Punkt hinzufügen..."
          className="flex-1 rounded-md border border-border bg-transparent px-3 py-1.5 text-sm text-[var(--body)] placeholder:text-[var(--muted)] focus:outline-none focus:ring-1 focus:ring-primary"
        />
        <Button
          variant="outline"
          size="icon"
          className="h-8 w-8 shrink-0"
          onClick={onAdd}
          disabled={!newText.trim()}
        >
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      {/* Empty state */}
      {total === 0 && (
        <div className="py-8 text-center">
          <ListChecks className="mx-auto h-8 w-8 text-[var(--muted)] mb-2" />
          <p className="text-sm text-[var(--muted)]">
            Noch keine Agenda-Punkte
          </p>
          <p className="text-xs text-[var(--muted)] mt-1">
            Fuege Punkte über das Eingabefeld hinzu
          </p>
        </div>
      )}
    </div>
  )
}

/* ═══════════════════════════════════════════════════════════════
   Notes Tab
   ═══════════════════════════════════════════════════════════════ */

function NotesTab({
  value,
  onChange,
  isPast,
}: {
  value: string
  onChange: (value: string) => void
  isPast: boolean
}) {
  return (
    <div className="h-full space-y-3">
      <div className="flex items-center gap-2">
        <FileText className="h-4 w-4 text-[var(--muted)]" />
        <h4 className="text-sm font-medium text-[var(--body)]">Protokoll / Notizen</h4>
      </div>
      <RichTextEditor
        content={value}
        onChange={onChange}
        placeholder="Meeting-Notizen hier erfassen..."
        editable={!isPast}
      />
      <div className="flex items-center justify-between">
        <p className="text-[10px] text-[var(--muted)]">
          {isPast ? 'Nur-Lese-Ansicht (vergangenes Meeting)' : 'Aenderungen werden automatisch gespeichert'}
        </p>
        <p className="text-[10px] text-[var(--muted)]">
          Zuletzt bearbeitet: {new Date().toLocaleDateString('de-CH', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })}
        </p>
      </div>
    </div>
  )
}
