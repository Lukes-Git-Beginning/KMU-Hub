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
import { ChevronDown, ChevronUp, Paperclip, Plus, X } from 'lucide-react'
import type { Meeting } from '@/stores/meetings'

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
  { value: 'daily', label: 'Taeglich' },
  { value: 'weekly', label: 'Woechentlich' },
  { value: 'monthly', label: 'Monatlich' },
]
const reminderOptions = [
  { value: 'none', label: 'Keine Erinnerung' },
  { value: '15min', label: '15 Minuten vorher' },
  { value: '30min', label: '30 Minuten vorher' },
  { value: '1h', label: '1 Stunde vorher' },
]
const projects = ['Website Relaunch', 'Mobile App', 'CRM Integration', 'Security Audit', 'Finanzen', 'Allgemein']

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
  const [selectedParticipants, setSelectedParticipants] = useState<string[]>([])
  const [showExtras, setShowExtras] = useState(false)
  const [participantSearch, setParticipantSearch] = useState('')

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
      setSelectedParticipants(meeting.participants.map((p) => p.id))
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
      setSelectedParticipants([])
      setShowExtras(false)
    }
  }, [meeting, open])

  const filteredParticipants = availableParticipants.filter(
    (p) =>
      !selectedParticipants.includes(p.id) &&
      p.name.toLowerCase().includes(participantSearch.toLowerCase())
  )

  const handleSubmit = () => {
    if (!title.trim()) return
    const participants = availableParticipants.filter((p) =>
      selectedParticipants.includes(p.id)
    )
    onSubmit({
      title: title.trim(),
      status: meeting?.status || 'scheduled',
      project: project || 'Allgemein',
      date,
      startTime,
      duration,
      room,
      isVideoCall,
      recurrence,
      reminder,
      description,
      participants,
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

        <div className="space-y-4 py-2">
          {/* Title */}
          <div className="space-y-1.5">
            <Label>Titel *</Label>
            <Input
              placeholder="z.B. Sprint Planning"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              autoFocus
            />
          </div>

          {/* Date + Time + Duration */}
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1.5">
              <Label>Datum</Label>
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>Uhrzeit</Label>
              <Input type="time" value={startTime} onChange={(e) => setStartTime(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>Dauer</Label>
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

          {/* Room + Video */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Raum</Label>
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
                <Label htmlFor="video-toggle" className="cursor-pointer">Video-Call</Label>
              </div>
            </div>
          </div>

          {/* Participants */}
          <div className="space-y-1.5">
            <Label>Teilnehmer</Label>
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

          {/* Project */}
          <div className="space-y-1.5">
            <Label>Projekt</Label>
            <Select value={project} onValueChange={setProject}>
              <SelectTrigger><SelectValue placeholder="Projekt zuordnen..." /></SelectTrigger>
              <SelectContent>
                {projects.map((p) => (
                  <SelectItem key={p} value={p}>{p}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Expandable extras */}
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
                  <Label>Wiederholung</Label>
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
                  <Label>Erinnerung</Label>
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
                <Label>Beschreibung</Label>
                <Textarea
                  placeholder="Agenda, Notizen..."
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                />
              </div>

              {/* Files placeholder */}
              <div className="space-y-1.5">
                <Label>Dateien</Label>
                <button className="flex items-center gap-2 rounded-lg border border-dashed border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors w-full">
                  <Paperclip className="h-4 w-4" />
                  Dateien anhaengen...
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
