import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
  UserRound,
  Handshake,
} from 'lucide-react'
import type { Meeting, AgendaItem } from '@/stores/meetings'
import { useContacts } from '@/api/hooks/useContacts'
import { useDeals } from '@/api/hooks/useDeals'
import { useCreateMeeting, useUpdateMeeting } from '@/api/hooks/useMeetings'

interface MeetingFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  meeting?: Meeting | null
  /** Seeds the title when creating a new meeting (e.g. a call-back from Video). */
  presetTitle?: string
  onSubmit: (data: Omit<Meeting, 'id'>) => void
}

const rooms = ['Konferenzraum A', 'Konferenzraum B', 'Huddle Space', 'Besprechungsraum', 'Remote']
const durations = [15, 30, 45, 60, 90, 120]
const recurrenceOptionKeys = [
  { value: 'none', key: 'meetings.recurrence.none' },
  { value: 'daily', key: 'meetings.recurrence.daily' },
  { value: 'weekly', key: 'meetings.recurrence.weekly' },
  { value: 'monthly', key: 'meetings.recurrence.monthly' },
  { value: 'custom', key: 'meetings.recurrence.custom' },
]

const customIntervalUnitKeys = [
  { value: 'days', key: 'meetings.interval.days' },
  { value: 'weeks', key: 'meetings.interval.weeks' },
  { value: 'months', key: 'meetings.interval.months' },
]
const reminderOptionKeys = [
  { value: 'none', key: 'meetings.reminder.none' },
  { value: '15min', key: 'meetings.reminder.15min' },
  { value: '30min', key: 'meetings.reminder.30min' },
  { value: '1h', key: 'meetings.reminder.1h' },
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
  { id: 'p1', name: 'Anna Müller', initials: 'AM' },
  { id: 'p2', name: 'Michael Berg', initials: 'MB' },
  { id: 'p3', name: 'Sarah Klein', initials: 'SK' },
  { id: 'p4', name: 'Lisa Schmidt', initials: 'LS' },
  { id: 'p5', name: 'Peter Koch', initials: 'PK' },
  { id: 'p6', name: 'Jonas Diaz', initials: 'JD' },
  { id: 'p7', name: 'Thomas Weber', initials: 'TW' },
  { id: 'p8', name: 'Eva Brunner', initials: 'EB' },
]

export function MeetingFormDialog({ open, onOpenChange, meeting, presetTitle, onSubmit }: MeetingFormDialogProps) {
  const { t } = useTranslation()
  const isEdit = !!meeting

  const createMeeting = useCreateMeeting()
  const updateMeeting = useUpdateMeeting()
  const isMutating = createMeeting.isPending || updateMeeting.isPending

  const [title, setTitle] = useState('')
  const [date, setDate] = useState('')
  const [startTime, setStartTime] = useState('09:00')
  const [duration, setDuration] = useState(30)
  const [room, setRoom] = useState('Remote')
  const [isVideoCall, setIsVideoCall] = useState(true)
  const [recurrence, setRecurrence] = useState<'none' | 'daily' | 'weekly' | 'monthly'>('none')
  const [recurrenceDisplay, setRecurrenceDisplay] = useState<string>('none')
  const [customInterval, setCustomInterval] = useState(2)
  const [customUnit, setCustomUnit] = useState<'days' | 'weeks' | 'months'>('weeks')
  const [reminder, setReminder] = useState<'15min' | '30min' | '1h' | 'none'>('15min')
  const [description, setDescription] = useState('')
  const [project, setProject] = useState('')
  const [color, setColor] = useState(PRESET_COLORS[0])
  const [selectedParticipants, setSelectedParticipants] = useState<string[]>([])
  const [showExtras, setShowExtras] = useState(false)
  const [participantSearch, setParticipantSearch] = useState('')
  const [agendaItems, setAgendaItems] = useState<AgendaItem[]>([])
  const [newAgendaText, setNewAgendaText] = useState('')
  const [addToCalendar, setAddToCalendar] = useState(true)
  const [sendInvitations, setSendInvitations] = useState(true)
  const [contactId, setContactId] = useState<string>('')
  const [dealId, setDealId] = useState<string>('')

  // CRM data for pickers
  const { data: contactsData } = useContacts({ page_size: 200 })
  const { data: dealsData } = useDeals({ page_size: 200 })
  const contacts = contactsData?.contacts ?? []
  const deals = dealsData?.deals ?? []


  useEffect(() => {
    if (meeting) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setTitle(meeting.title)
      setDate(meeting.date)
      setStartTime(meeting.startTime)
      setDuration(meeting.duration)
      setRoom(meeting.room)
      setIsVideoCall(meeting.isVideoCall)
      setRecurrence(meeting.recurrence)
      setRecurrenceDisplay(meeting.recurrence)
      setCustomInterval(2)
      setCustomUnit('weeks')
      setReminder(meeting.reminder)
      setDescription(meeting.description)
      setProject(meeting.project)
      setColor(meeting.color || PRESET_COLORS[0])
      setSelectedParticipants(meeting.participants.map((p) => p.id))
      setAgendaItems(meeting.agenda.map((a) => ({ ...a })))
      setShowExtras(meeting.recurrence !== 'none' || !!meeting.description || meeting.agenda.length > 0)
      setAddToCalendar(!!meeting.calendarEventId)
      setSendInvitations(!!meeting.invitationsSent)
      setContactId(meeting.contact_id ?? '')
      setDealId(meeting.deal_id ?? '')
    } else {
      setTitle(presetTitle ?? '')
      setDate(new Date().toISOString().split('T')[0])
      setStartTime('09:00')
      setDuration(30)
      setRoom('Remote')
      setIsVideoCall(true)
      setRecurrence('none')
      setRecurrenceDisplay('none')
      setCustomInterval(2)
      setCustomUnit('weeks')
      setReminder('15min')
      setDescription('')
      setProject('')
      setColor(PRESET_COLORS[0])
      setSelectedParticipants([])
      setShowExtras(false)
      setAgendaItems([])
      setNewAgendaText('')
      setAddToCalendar(true)
      setSendInvitations(true)
      setContactId('')
      setDealId('')
    }
  }, [meeting, open, presetTitle])

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

  const handleRecurrenceChange = (value: string) => {
    setRecurrenceDisplay(value)
    if (value === 'custom') {
      // Map custom to the closest standard recurrence for storage
      const unitMap: Record<string, 'daily' | 'weekly' | 'monthly'> = {
        days: 'daily',
        weeks: 'weekly',
        months: 'monthly',
      }
      setRecurrence(unitMap[customUnit])
    } else {
      setRecurrence(value as 'none' | 'daily' | 'weekly' | 'monthly')
    }
  }

  const handleSubmit = () => {
    if (!title.trim()) return
    const participants = availableParticipants.filter((p) =>
      selectedParticipants.includes(p.id)
    )
    // Resolve custom recurrence to closest standard value
    let finalRecurrence = recurrence
    if (recurrenceDisplay === 'custom') {
      const unitMap: Record<string, 'daily' | 'weekly' | 'monthly'> = {
        days: 'daily',
        weeks: 'weekly',
        months: 'monthly',
      }
      finalRecurrence = unitMap[customUnit]
    }

    // Build the shared local-store payload (keeps MeetingsPage in sync).
    const newMeetingId = meeting?.id || `m${Date.now()}`
    const localPayload: Omit<Meeting, 'id'> = {
      title: title.trim(),
      status: meeting?.status || 'scheduled',
      project: project || 'Allgemein',
      color,
      date,
      startTime,
      duration,
      room,
      isVideoCall,
      recurrence: finalRecurrence,
      reminder,
      description,
      participants,
      organizerId: meeting?.organizerId || selectedParticipants[0] || '',
      agenda: agendaItems,
      notes: meeting?.notes || '',
      files: meeting?.files || [],
      whiteboardLink: meeting?.whiteboardLink || '',
      projectLink: project.toLowerCase().replace(/\s+/g, '-'),
      calendarEventId: addToCalendar ? (meeting?.calendarEventId || `cal-${newMeetingId}`) : undefined,
      invitationsSent: sendInvitations ? true : (meeting?.invitationsSent || false),
      contact_id: contactId !== '' ? contactId : null,
      deal_id: dealId !== '' ? dealId : null,
    }

    // Build the ISO timestamps the backend expects.
    const scheduledStart = `${date}T${startTime}:00`
    const scheduledEnd = new Date(
      new Date(scheduledStart).getTime() + duration * 60_000,
    ).toISOString()

    const agendaText = agendaItems.map((a) => a.text).join('\n') || undefined

    // lean: local-store meetings have numeric IDs like "m1" (not UUIDs); skip
    //   the backend PUT for those — they are display-only until R3-E3 wires a
    //   real server-side edit. Trigger: when local meetings are removed in favour
    //   of backend-only state.
    const isBackendMeeting = isEdit && meeting && !/^m\d+$/.test(meeting.id)

    if (isBackendMeeting) {
      updateMeeting.mutate(
        {
          id: meeting.id,
          data: {
            title: title.trim(),
            description: description || undefined,
            agenda: agendaText,
            scheduled_start: scheduledStart,
            scheduled_end: scheduledEnd,
            contact_id: contactId !== '' ? contactId : null,
            deal_id: dealId !== '' ? dealId : null,
          },
        },
        {
          onSuccess: () => {
            onSubmit(localPayload)
            onOpenChange(false)
          },
          onError: (err) => {
            toast.error(t('meetings.toast.updateFailed', { error: (err as Error).message }))
          },
        },
      )
    } else if (isEdit) {
      // Local-store-only meeting: update only the local store (no backend call).
      onSubmit(localPayload)
      onOpenChange(false)
    } else {
      createMeeting.mutate(
        {
          title: title.trim(),
          description: description || undefined,
          agenda: agendaText,
          scheduled_start: scheduledStart,
          scheduled_end: scheduledEnd,
          // lean: mock participant IDs (p1..p8) are not real UUIDs — backend
          //   would reject them. Send empty until a real user-picker is wired.
          //   Trigger: when team-member picker replaces the mock list.
          attendee_ids: [],
          // lean: a brand-new meeting has no pre-existing calendar event to link;
          //   calendar_event_id stays undefined on create.
          calendar_event_id: undefined,
          contact_id: contactId !== '' ? contactId : null,
          deal_id: dealId !== '' ? dealId : null,
        },
        {
          onSuccess: () => {
            onSubmit(localPayload)
            onOpenChange(false)
          },
          onError: (err) => {
            toast.error(t('meetings.toast.createFailed', { error: (err as Error).message }))
          },
        },
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? t('meetings.form.editTitle') : t('meetings.form.newTitle')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-5 py-2">
          {/* ── Title ── */}
          <div className="space-y-1.5">
            <Label className="flex items-center gap-1.5">
              <Video className="h-3.5 w-3.5 text-[var(--muted)]" />
              {t('meetings.form.title')} *
            </Label>
            <Input
              placeholder={t('meetings.form.titlePlaceholder')}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              autoFocus
            />
          </div>

          {/* ── Color Picker ── */}
          <div className="space-y-1.5">
            <Label className="text-xs">{t('meetings.form.color')}</Label>
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
              {t('meetings.form.appointment')}
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
                      {d >= 60 ? `${d / 60} ${t('meetings.time.hours')}` : `${d} ${t('meetings.time.minutes')}`}
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
                {t('meetings.form.room')}
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
                  {t('meetings.form.videoCall')}
                </Label>
              </div>
            </div>
          </div>

          {/* ── Participants ── */}
          <div className="space-y-1.5">
            <Label className="flex items-center gap-1.5">
              <Users className="h-3.5 w-3.5 text-[var(--muted)]" />
              {t('meetings.form.participants')}
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
              placeholder={t('meetings.form.searchParticipants')}
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
              {t('meetings.form.project')}
            </Label>
            <Select value={project} onValueChange={setProject}>
              <SelectTrigger><SelectValue placeholder={t('meetings.form.assignProject')} /></SelectTrigger>
              <SelectContent>
                {projects.map((p) => (
                  <SelectItem key={p} value={p}>{p}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* ── CRM: Kontakt + Deal ── */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="flex items-center gap-1.5">
                <UserRound className="h-3.5 w-3.5 text-[var(--muted)]" />
                {t('meetings.form.contact')}
              </Label>
              <Select value={contactId || 'none'} onValueChange={(v) => setContactId(v === 'none' ? '' : v)}>
                <SelectTrigger>
                  <SelectValue placeholder={t('meetings.form.contactPlaceholder')} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">{t('meetings.form.crmNone')}</SelectItem>
                  {contacts.filter((c) => c.id).map((c) => {
                    const label = `${c.firstName ?? ''} ${c.lastName ?? ''}`.trim() || c.id!
                    return (
                      <SelectItem key={c.id} value={c.id!}>
                        {label}
                      </SelectItem>
                    )
                  })}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label className="flex items-center gap-1.5">
                <Handshake className="h-3.5 w-3.5 text-[var(--muted)]" />
                {t('meetings.form.deal')}
              </Label>
              <Select value={dealId || 'none'} onValueChange={(v) => setDealId(v === 'none' ? '' : v)}>
                <SelectTrigger>
                  <SelectValue placeholder={t('meetings.form.dealPlaceholder')} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">{t('meetings.form.crmNone')}</SelectItem>
                  {deals.filter((d) => d.id).map((d) => (
                    <SelectItem key={d.id} value={d.id!}>
                      {d.name ?? d.id}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* ── Expandable Extras ── */}
          <button
            onClick={() => setShowExtras(!showExtras)}
            className="flex items-center gap-1 text-sm text-primary hover:underline"
          >
            {showExtras ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
            {showExtras ? t('meetings.form.lessOptions') : t('meetings.form.moreOptions')}
          </button>

          {showExtras && (
            <div className="space-y-4 rounded-lg border border-border p-3">
              {/* Recurrence + Reminder */}
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label className="flex items-center gap-1.5">
                    <Repeat className="h-3.5 w-3.5 text-[var(--muted)]" />
                    {t('meetings.form.recurrence')}
                  </Label>
                  <Select value={recurrenceDisplay} onValueChange={handleRecurrenceChange}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {recurrenceOptionKeys.map((o) => (
                        <SelectItem key={o.value} value={o.value}>{t(o.key)}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {recurrenceDisplay === 'custom' && (
                    <div className="mt-2 space-y-2">
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-muted-foreground whitespace-nowrap">{t('meetings.recurrence.every')}</span>
                        <Input
                          type="number"
                          min={1}
                          max={99}
                          value={customInterval}
                          onChange={(e) => setCustomInterval(Math.max(1, Number(e.target.value)))}
                          className="w-16 text-sm"
                        />
                        <Select value={customUnit} onValueChange={(v) => {
                          setCustomUnit(v as typeof customUnit)
                          const unitMap: Record<string, 'daily' | 'weekly' | 'monthly'> = { days: 'daily', weeks: 'weekly', months: 'monthly' }
                          setRecurrence(unitMap[v])
                        }}>
                          <SelectTrigger className="w-24"><SelectValue /></SelectTrigger>
                          <SelectContent>
                            {customIntervalUnitKeys.map((u) => (
                              <SelectItem key={u.value} value={u.value}>{t(u.key)}</SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <p className="text-[10px] text-muted-foreground">
                        {t('meetings.recurrence.nextAppointments', { count: Math.min(customInterval * 3, 12) })}
                      </p>
                    </div>
                  )}
                  {recurrenceDisplay !== 'none' && recurrenceDisplay !== 'custom' && (
                    <p className="mt-1 text-[10px] text-muted-foreground">
                      {t('meetings.recurrence.autoCreate')}
                    </p>
                  )}
                </div>
                <div className="space-y-1.5">
                  <Label className="flex items-center gap-1.5">
                    <Bell className="h-3.5 w-3.5 text-[var(--muted)]" />
                    {t('meetings.form.reminder')}
                  </Label>
                  <Select value={reminder} onValueChange={(v) => setReminder(v as typeof reminder)}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {reminderOptionKeys.map((o) => (
                        <SelectItem key={o.value} value={o.value}>{t(o.key)}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Description */}
              <div className="space-y-1.5">
                <Label className="flex items-center gap-1.5">
                  <FileText className="h-3.5 w-3.5 text-[var(--muted)]" />
                  {t('meetings.form.description')}
                </Label>
                <Textarea
                  placeholder={t('meetings.form.descriptionPlaceholder')}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                />
              </div>

              {/* ── Agenda Editor ── */}
              <div className="space-y-1.5">
                <Label className="flex items-center gap-1.5">
                  <ListChecks className="h-3.5 w-3.5 text-[var(--muted)]" />
                  {t('meetings.form.agenda')}
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
                    placeholder={t('meetings.form.addAgendaItem')}
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
                  {t('meetings.form.files')}
                </Label>
                <button className="flex items-center gap-2 rounded-lg border border-dashed border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors w-full">
                  <Paperclip className="h-4 w-4" />
                  {t('meetings.form.attachFiles')}
                </button>
              </div>

              {/* Calendar + Invitation checkboxes */}
              <div className="space-y-3 pt-1">
                <div className="flex items-center gap-2">
                  <Switch checked={addToCalendar} onCheckedChange={setAddToCalendar} id="cal-toggle" />
                  <Label htmlFor="cal-toggle" className="cursor-pointer flex items-center gap-1.5 text-sm">
                    <Calendar className="h-3.5 w-3.5 text-[var(--muted)]" />
                    {t('meetings.form.addToCalendar')}
                  </Label>
                </div>
                <div className="flex items-center gap-2">
                  <Switch checked={sendInvitations} onCheckedChange={setSendInvitations} id="invite-toggle" />
                  <Label htmlFor="invite-toggle" className="cursor-pointer flex items-center gap-1.5 text-sm">
                    <Users className="h-3.5 w-3.5 text-[var(--muted)]" />
                    {t('meetings.form.inviteByEmail')}
                  </Label>
                </div>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSubmit} disabled={!title.trim() || isMutating}>
            <Plus className="mr-1.5 h-4 w-4" />
            {isMutating
              ? t('common.saving')
              : isEdit
                ? t('common.save')
                : t('meetings.form.createMeeting')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
