import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Calendar,
  ChevronDown,
  ChevronUp,
  Clock,
  FolderOpen,
  ListChecks,
  MapPin,
  Paperclip,
  Plus,
  Repeat,
  Bell,
  Users,
  Video,
  X,
  FileText,
} from 'lucide-react'
import type { Meeting, AgendaItem } from '@/stores/meetings'

interface MeetingFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  meeting?: Meeting | null
  onSubmit: (data: Omit<Meeting, 'id'>) => void
}

const rooms = ['Konferenzraum A', 'Konferenzraum B', 'Huddle Space', 'Besprechungsraum', 'Remote']
const durations = [15, 30, 45, 60, 90, 120]
const recurrenceOptions = [
  { value: 'none', label: 'Keine Wiederholung' },
  { value: 'daily', label: 'Täglich' },
  { value: 'weekly', label: 'Wöchentlich' },
  { value: 'monthly', label: 'Monatlich' },
]
const reminderOptions = [
  { value: 'none', label: 'Keine Erinnerung' },
  { value: '15min', label: '15 Minuten vorher' },
  { value: '30min', label: '30 Minuten vorher' },
  { value: '1h', label: '1 Stunde vorher' },
]
const projects = ['Website Relaunch', 'Mobile App', 'CRM Integration', 'Security Audit', 'Finanzen', 'Allgemein']

const PRESET_COLORS = [
  '#3B82F6', // blue
  '#8B5CF6', // violet
  '#10B981', // emerald
  '#F59E0B', // amber
  '#EF4444', // red
  '#EC4899', // pink
  '#6B7280', // gray
  '#06B6D4', // cyan
]

const availableParticipants = [
  { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
  { id: 'p2', name: 'Michael Berg', initials: 'MB' },
  { id: 'p3', name: 'Sarah Klein', initials: 'SK' },
  { id: 'p4', name: 'Lisa Schmidt', initials: 'LS' },
  { id: 'p5', name: 'Peter Koch', initials: 'PK' },
  { id: 'p6', name: 'Jonas Diaz', initials: 'JD' },
  { id: 'p7', name: 'Thomas Weber', initials: 'TW' },
  { id: 'p8', name: 'Eva Brunner', initials: 'EB' },
]

export function MeetingFormDialog({ open, onOpenChange, meeting, onSubmit }: MeetingFormDialogProps) {
  const isEdit = !!meeting

  const [title, setTitle] = useState('')
  const [date, setDate] = useState('')
  const [startTime, setStartTime] = useState('09:00')
  const [duration, setDuration] = useState(30)
  const [room, setRoom] = useState('Remote')
  const [isVideoCall, setIsVideoCall] = useState(true)
  const [recurrence, setRecurrence] = useState<'none' | 'daily' | 'weekly' | 'monthly'>('none')
  const [reminder, setReminder] = useState<'15min' | '30min' | '1h' | 'none'>('15min')
  const [description, setDescription] = useState('')
  const [project, setProject] = useState('')
  const [color, setColor] = useState(PRESET_COLORS[0])
  const [selectedParticipants, setSelectedParticipants] = useState<string[]>([])
  const [showExtras, setShowExtras] = useState(false)
  const [participantSearch, setParticipantSearch] = useState('')
  const [agendaItems, setAgendaItems] = useState<AgendaItem[]>([])
  const [newAgendaText, setNewAgendaText] = useState('')

  useEffect(() => {
    if (meeting) {
      setTitle(meeting.title)
      setDate(meeting.date)
      setStartTime(meeting.startTime)
      setDuration(meeting.duration)
      setRoom(meeting.room)
      setIsVideoCall(meeting.isVideoCall)
      setRecurrence(meeting.recurrence)
      setReminder(meeting.reminder)
      setDescription(meeting.description)
      setProject(meeting.project)
      setColor(meeting.color || PRESET_COLORS[0])
      setSelectedParticipants(meeting.participants.map((p) => p.id))
      setAgendaItems(meeting.agenda.map((a) => ({ ...a })))
      setShowExtras(meeting.recurrence !== 'none' || !!meeting.description || meeting.agenda.length > 0)
    } else {
      setTitle('')
      setDate(new Date().toISOString().split('T')[0])
      setStartTime('09:00')
      setDuration(30)
      setRoom('Remote')
      setIsVideoCall(true)
      setRecurrence('none')
      setReminder('15min')
      setDescription('')
      setProject('')
      setColor(PRESET_COLORS[0])
      setSelectedParticipants([])
      setShowExtras(false)
      setAgendaItems([])
      setNewAgendaText('')
    }
  }, [meeting, open])

  const filteredParticipants = availableParticipants.filter(
    (p) =>
      !selectedParticipants.includes(p.id) &&
      p.name.toLowerCase().includes(participantSearch.toLowerCase())
  )

  const handleAddAgenda = () => {
    const text = newAgendaText.trim()
    if (!text) return
    setAgendaItems((prev) => [...prev, { id: `a${Date.now()}`, text, done: false }])
    setNewAgendaText('')
  }

  const handleRemoveAgenda = (id: string) => {
    setAgendaItems((prev) => prev.filter((a) => a.id !== id))
  }

  const handleReorderAgenda = (id: string, direction: 'up' | 'down') => {
    setAgendaItems((prev) => {
      const idx = prev.findIndex((a) => a.id === id)
      if (idx === -1) return prev
      const swapIdx = direction === 'up' ? idx - 1 : idx + 1
      if (swapIdx < 0 || swapIdx >= prev.length) return prev
      const next = [...prev]
      ;[next[idx], next[swapIdx]] = [next[swapIdx], next[idx]]
      return next
    })
  }

  const handleSubmit = () => {
    if (!title.trim()) return
    const participants = availableParticipants.filter((p) =>
      selectedParticipants.includes(p.id)
    )
    onSubmit({
      title: title.trim(),
      status: meeting?.status || 'scheduled',
      project: project || 'Allgemein',
      color,
      date,
      startTime,
      duration,
      room,
      isVideoCall,
      recurrence,
      reminder,
      description,
      participants,
      organizerId: meeting?.organizerId || selectedParticipants[0] || '',
      agenda: agendaItems,
      notes: meeting?.notes || '',
      files: meeting?.files || [],
      whiteboardLink: meeting?.whiteboardLink || '',
      projectLink: project.toLowerCase().replace(/\s+/g, '-'),
    })
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Meeting bearbeiten' : 'Neues Meeting'}</DialogTitle>
        </DialogHeader>

        <div className="space-y-5 py-2">
          {/* ── Title ── */}
          <div className="space-y-1.5">
            <Label className="flex items-center gap-1.5">
              <Video className="h-3.5 w-3.5 text-[var(--muted)]" />
              Titel *
            </Label>
            <Input
              placeholder="z.B. Sprint Planning"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              autoFocus
            />
          </div>

          {/* ── Color Picker ── */}
          <div className="space-y-1.5">
            <Label className="text-xs">Farbe</Label>
            <div className="flex items-center gap-2">
              {PRESET_COLORS.map((c) => (
                <button
                  key={c}
                  onClick={() => setColor(c)}
                  className={`h-7 w-7 rounded-full transition-all ${
                    color === c
                      ? 'ring-2 ring-offset-2 ring-primary scale-110'
                      : 'hover:scale-105'
                  }`}
                  style={{ backgroundColor: c }}
                  title={c}
                />
              ))}
            </div>
          </div>

          {/* ── Date + Time + Duration ── */}
          <div>
            <Label className="flex items-center gap-1.5 mb-1.5">
              <Calendar className="h-3.5 w-3.5 text-[var(--muted)]" />
              Termin
            </Label>
            <div className="grid grid-cols-3 gap-3">
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
              <div className="relative">
                <Clock className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-[var(--muted)] pointer-events-none" />
                <Input
                  type="time"
                  value={startTime}
                  onChange={(e) => setStartTime(e.target.value)}
                  className="pl-8"
                />
              </div>
              <Select value={String(duration)} onValueChange={(v) => setDuration(Number(v))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {durations.map((d) => (
                    <SelectItem key={d} value={String(d)}>
                      {d >= 60 ? `${d / 60} Std` : `${d} Min`}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* ── Room + Video ── */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="flex items-center gap-1.5">
                <MapPin className="h-3.5 w-3.5 text-[var(--muted)]" />
                Raum
              </Label>
              <Select value={room} onValueChange={setRoom}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {rooms.map((r) => (
                    <SelectItem key={r} value={r}>{r}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-end gap-3 pb-0.5">
              <div className="flex items-center gap-2">
                <Switch checked={isVideoCall} onCheckedChange={setIsVideoCall} id="video-toggle" />
                <Label htmlFor="video-toggle" className="cursor-pointer flex items-center gap-1.5">
                  <Video className="h-3.5 w-3.5 text-[var(--muted)]" />
                  Video-Call
                </Label>
              </div>
            </div>
          </div>

          {/* ── Participants ── */}
          <div className="space-y-1.5">
            <Label className="flex items-center gap-1.5">
              <Users className="h-3.5 w-3.5 text-[var(--muted)]" />
              Teilnehmer
            </Label>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {selectedParticipants.map((pId) => {
                const p = availableParticipants.find((x) => x.id === pId)
                if (!p) return null
                return (
                  <span
                    key={pId}
                    className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary"
                  >
                    {p.name}
                    <button
                      onClick={() => setSelectedParticipants((s) => s.filter((id) => id !== pId))}
                      className="ml-0.5 rounded-full hover:bg-primary/20 p-0.5"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </span>
                )
              })}
            </div>
            <Input
              placeholder="Teilnehmer suchen..."
              value={participantSearch}
              onChange={(e) => setParticipantSearch(e.target.value)}
              className="text-sm"
            />
            {participantSearch && filteredParticipants.length > 0 && (
              <div className="mt-1 max-h-32 overflow-y-auto rounded-md border bg-card p-1">
                {filteredParticipants.map((p) => (
                  <button
                    key={p.id}
                    onClick={() => {
                      setSelectedParticipants((s) => [...s, p.id])
                      setParticipantSearch('')
                    }}
                    className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-secondary"
                  >
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                      {p.initials}
                    </span>
                    {p.name}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* ── Project ── */}
          <div className="space-y-1.5">
            <Label className="flex items-center gap-1.5">
              <FolderOpen className="h-3.5 w-3.5 text-[var(--muted)]" />
              Projekt
            </Label>
            <Select value={project} onValueChange={setProject}>
              <SelectTrigger><SelectValue placeholder="Projekt zuordnen..." /></SelectTrigger>
              <SelectContent>
                {projects.map((p) => (
                  <SelectItem key={p} value={p}>{p}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* ── Expandable Extras ── */}
          <button
            onClick={() => setShowExtras(!showExtras)}
            className="flex items-center gap-1 text-sm text-primary hover:underline"
          >
            {showExtras ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
            {showExtras ? 'Weniger Optionen' : 'Weitere Optionen'}
          </button>

          {showExtras && (
            <div className="space-y-4 rounded-lg border border-border p-3">
              {/* Recurrence + Reminder */}
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label className="flex items-center gap-1.5">
                    <Repeat className="h-3.5 w-3.5 text-[var(--muted)]" />
                    Wiederholung
                  </Label>
                  <Select value={recurrence} onValueChange={(v) => setRecurrence(v as typeof recurrence)}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {recurrenceOptions.map((o) => (
                        <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label className="flex items-center gap-1.5">
                    <Bell className="h-3.5 w-3.5 text-[var(--muted)]" />
                    Erinnerung
                  </Label>
                  <Select value={reminder} onValueChange={(v) => setReminder(v as typeof reminder)}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {reminderOptions.map((o) => (
                        <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Description */}
              <div className="space-y-1.5">
                <Label className="flex items-center gap-1.5">
                  <FileText className="h-3.5 w-3.5 text-[var(--muted)]" />
                  Beschreibung
                </Label>
                <Textarea
                  placeholder="Kontext, Ziele, Vorbereitung..."
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                />
              </div>

              {/* ── Agenda Editor ── */}
              <div className="space-y-1.5">
                <Label className="flex items-center gap-1.5">
                  <ListChecks className="h-3.5 w-3.5 text-[var(--muted)]" />
                  Agenda
                </Label>
                {agendaItems.length > 0 && (
                  <div className="space-y-1 mb-2">
                    {agendaItems.map((item, idx) => (
                      <div
                        key={item.id}
                        className="group flex items-center gap-2 rounded-md border border-border px-2.5 py-1.5 text-sm"
                      >
                        <span className="text-xs text-[var(--muted)] w-4 text-center shrink-0">
                          {idx + 1}.
                        </span>
                        <span className="flex-1 text-[var(--body)] truncate">
                          {item.text}
                        </span>
                        <div className="hidden group-hover:flex items-center gap-0.5">
                          {idx > 0 && (
                            <button
                              onClick={() => handleReorderAgenda(item.id, 'up')}
                              className="rounded p-0.5 text-[var(--muted)] hover:text-[var(--body)]"
                            >
                              <ChevronUp className="h-3.5 w-3.5" />
                            </button>
                          )}
                          {idx < agendaItems.length - 1 && (
                            <button
                              onClick={() => handleReorderAgenda(item.id, 'down')}
                              className="rounded p-0.5 text-[var(--muted)] hover:text-[var(--body)]"
                            >
                              <ChevronDown className="h-3.5 w-3.5" />
                            </button>
                          )}
                          <button
                            onClick={() => handleRemoveAgenda(item.id)}
                            className="rounded p-0.5 text-[var(--muted)] hover:text-red-500"
                          >
                            <X className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <Input
                    value={newAgendaText}
                    onChange={(e) => setNewAgendaText(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleAddAgenda()}
                    placeholder="Neuen Agenda-Punkt hinzufügen..."
                    className="text-sm"
                  />
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-8 w-8 shrink-0"
                    onClick={handleAddAgenda}
                    disabled={!newAgendaText.trim()}
                    type="button"
                  >
                    <Plus className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              {/* Files placeholder */}
              <div className="space-y-1.5">
                <Label className="flex items-center gap-1.5">
                  <Paperclip className="h-3.5 w-3.5 text-[var(--muted)]" />
                  Dateien
                </Label>
                <button className="flex items-center gap-2 rounded-lg border border-dashed border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors w-full">
                  <Paperclip className="h-4 w-4" />
                  Dateien anhängen...
                </button>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSubmit} disabled={!title.trim()}>
            <Plus className="mr-1.5 h-4 w-4" />
            {isEdit ? 'Speichern' : 'Meeting erstellen'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
