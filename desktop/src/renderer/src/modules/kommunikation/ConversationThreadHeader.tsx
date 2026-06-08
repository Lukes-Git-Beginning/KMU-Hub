import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Mail,
  MessageSquare,
  Bell,
  MoreHorizontal,
  UserPlus,
  Tag,
  Archive,
  Trash2,
  MailOpen,
  MailCheck,
  Forward,
  ListTodo,
  Clock,
  Hand,
  Check,
  Plus,
  X,
  ChevronDown,
  Star,
  Phone,
  Video,
} from 'lucide-react'
import { toast } from 'sonner'
import type { InboxMessage, InboxChannel } from '@/api/inbox-types'
import type { ConversationStatus } from '@/types/communication'
import {
  useMarkRead,
  useMarkUnread,
  useToggleStar,
  useArchiveMessage,
  useAssignMessage,
  useClaimMessage,
  useSnoozeMessage,
} from '@/api/hooks/useInbox'
import { useEmployees } from '@/api/hooks/hr-hooks'
import { useMeetingsStore } from '@/stores/meetings'
import { useKommunikationStore } from '@/stores/kommunikation'
import { useInboxStatus } from '@/stores/inboxStatus'
import { useInboxTags, SUGGESTED_TAGS } from '@/stores/inboxTags'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { SnoozePopover } from './SnoozePopover'
import { ForwardDialog } from './ForwardDialog'

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

const channelConfig: Record<InboxChannel, { icon: typeof Mail; color: string; bg: string }> = {
  email: { icon: Mail, color: 'text-blue-500', bg: 'bg-blue-50 dark:bg-blue-950/30' },
  chat: { icon: MessageSquare, color: 'text-green-500', bg: 'bg-green-50 dark:bg-green-950/30' },
  notification: { icon: Bell, color: 'text-orange-500', bg: 'bg-orange-50 dark:bg-orange-950/30' },
}

const STATUS_ORDER: ConversationStatus[] = ['open', 'pending', 'resolved', 'closed']
const statusConfig: Record<ConversationStatus, { labelKey: string; dot: string }> = {
  open: { labelKey: 'kommunikation.filter.open', dot: 'bg-success' },
  pending: { labelKey: 'kommunikation.filter.pending', dot: 'bg-warning' },
  resolved: { labelKey: 'kommunikation.filter.resolved', dot: 'bg-blue-500' },
  closed: { labelKey: 'kommunikation.filter.closed', dot: 'bg-muted-foreground' },
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ConversationThreadHeaderProps {
  message: InboxMessage
}

export function ConversationThreadHeader({ message: msg }: ConversationThreadHeaderProps) {
  const { t } = useTranslation()
  const setSelectedConversation = useKommunikationStore((s) => s.setSelectedConversation)
  const markRead = useMarkRead()
  const markUnread = useMarkUnread()
  const toggleStar = useToggleStar()
  const archiveMsg = useArchiveMessage()
  const assignMsg = useAssignMessage()
  const claimMsg = useClaimMessage()
  const snoozeMsg = useSnoozeMessage()
  const startCall = useMeetingsStore((s) => s.startCall)

  const { data: employeesData } = useEmployees()
  const employees = employeesData?.employees ?? []

  const status = useInboxStatus((s) => s.statuses[msg.id] ?? 'open')
  const setStatus = useInboxStatus((s) => s.setStatus)

  const baseTags = msg.tags
  const tags = useInboxTags((s) => s.overrides[msg.id] ?? baseTags)
  const addTag = useInboxTags((s) => s.addTag)
  const removeTag = useInboxTags((s) => s.removeTag)

  const [actionsOpen, setActionsOpen] = useState(false)
  const [assignOpen, setAssignOpen] = useState(false)
  const [statusOpen, setStatusOpen] = useState(false)
  const [tagOpen, setTagOpen] = useState(false)
  const [newTag, setNewTag] = useState('')
  const [forwardOpen, setForwardOpen] = useState(false)

  const ch = channelConfig[msg.channel]
  const ChannelIcon = ch.icon

  const handleToggleRead = () => {
    if (msg.is_read) {
      markUnread.mutate(msg.id)
      toast.success(t('kommunikation.header.markedUnread'))
    } else {
      markRead.mutate(msg.id)
      toast.success(t('kommunikation.header.markedRead'))
    }
    setActionsOpen(false)
  }

  const handleArchive = () => {
    archiveMsg.mutate(msg.id)
    setSelectedConversation(null)
    setActionsOpen(false)
  }

  const handleSnooze = (iso: string) => {
    snoozeMsg.mutate({ id: msg.id, snoozeUntil: iso })
    setSelectedConversation(null)
    toast.success(t('kommunikation.snooze.snoozed'))
  }

  const handleClaim = () => {
    claimMsg.mutate(msg.id)
  }

  const handleAssign = (assigneeId: string) => {
    assignMsg.mutate({ id: msg.id, assigneeId })
    setAssignOpen(false)
  }

  const employeeLabel = (userId: string) => {
    const emp = employees.find((e) => e.userId === userId)
    return emp?.userName || emp?.userEmail || userId
  }

  const suggestions = SUGGESTED_TAGS.filter((s) => !tags.includes(s))

  // Bridge into the video module (same path as Team/Kontakte call buttons).
  const handleCall = (kind: 'audio' | 'video') => {
    startCall(msg.crm_contact_id ?? msg.sender_id ?? msg.id, msg.sender_name)
    toast.success(t(kind === 'video' ? 'kommunikation.call.startingVideo' : 'kommunikation.call.startingAudio', { name: msg.sender_name }))
  }

  return (
    <div className="border-b border-border px-4 py-3">
      {/* Row 1: Subject + actions */}
      <div className="flex items-center gap-2">
        {/* Channel badge */}
        <div className={`flex h-6 w-6 shrink-0 items-center justify-center rounded ${ch.bg}`}>
          <ChannelIcon className={`h-3.5 w-3.5 ${ch.color}`} />
        </div>

        {/* Subject */}
        <h2 className="text-sm font-semibold text-foreground truncate flex-1">
          {msg.subject}
        </h2>

        {/* Status selector */}
        <Popover open={statusOpen} onOpenChange={setStatusOpen}>
          <PopoverTrigger asChild>
            <button
              className="flex items-center gap-1.5 rounded-full border border-border px-2 py-1 text-[11px] font-medium text-foreground hover:bg-accent transition-colors"
              title={t('kommunikation.status.change')}
            >
              <span className={`h-1.5 w-1.5 rounded-full ${statusConfig[status].dot}`} />
              {t(statusConfig[status].labelKey)}
              <ChevronDown className="h-3 w-3 text-muted-foreground" />
            </button>
          </PopoverTrigger>
          <PopoverContent className="w-40 p-1" align="end" sideOffset={4}>
            {STATUS_ORDER.map((s) => (
              <button
                key={s}
                onClick={() => {
                  setStatus(msg.id, s)
                  setStatusOpen(false)
                  toast.success(t('kommunikation.status.changed', { status: t(statusConfig[s].labelKey) }))
                }}
                className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors hover:bg-accent"
              >
                <span className={`h-1.5 w-1.5 rounded-full ${statusConfig[s].dot}`} />
                {t(statusConfig[s].labelKey)}
                {status === s && <Check className="ml-auto h-3.5 w-3.5 text-primary" />}
              </button>
            ))}
          </PopoverContent>
        </Popover>

        {/* Audio call */}
        <button
          onClick={() => handleCall('audio')}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          title={t('kommunikation.call.audio')}
        >
          <Phone className="h-4 w-4" />
        </button>

        {/* Video call */}
        <button
          onClick={() => handleCall('video')}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          title={t('kommunikation.call.video')}
        >
          <Video className="h-4 w-4" />
        </button>

        {/* Snooze */}
        <SnoozePopover onSnooze={handleSnooze}>
          <button
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            title={t('kommunikation.snooze.remindLater')}
          >
            <Clock className="h-4 w-4" />
          </button>
        </SnoozePopover>

        {/* Star toggle */}
        <button
          onClick={() => toggleStar.mutate(msg.id)}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          title={msg.is_starred ? t('kommunikation.header.removeStar') : t('kommunikation.header.addStar')}
        >
          <Star className={`h-4 w-4 ${msg.is_starred ? 'fill-warning text-warning' : ''}`} />
        </button>

        {/* Actions menu */}
        <Popover open={actionsOpen} onOpenChange={setActionsOpen}>
          <PopoverTrigger asChild>
            <button className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors">
              <MoreHorizontal className="h-4 w-4" />
            </button>
          </PopoverTrigger>
          <PopoverContent className="w-48 p-1" align="end" sideOffset={4}>
            <button
              onClick={handleToggleRead}
              className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors hover:bg-accent"
            >
              {msg.is_read ? <MailOpen className="h-3.5 w-3.5" /> : <MailCheck className="h-3.5 w-3.5" />}
              {msg.is_read ? t('kommunikation.header.markUnread') : t('kommunikation.header.markRead')}
            </button>
            <button
              onClick={() => { setForwardOpen(true); setActionsOpen(false) }}
              className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors hover:bg-accent"
            >
              <Forward className="h-3.5 w-3.5" />
              {t('kommunikation.header.forward')}
            </button>
            <button
              onClick={() => { toast.success(t('kommunikation.header.taskCreated')); setActionsOpen(false) }}
              className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors hover:bg-accent"
            >
              <ListTodo className="h-3.5 w-3.5" />
              {t('kommunikation.header.createTask')}
            </button>
            <div className="my-1 border-t border-border" />
            <button
              onClick={handleArchive}
              className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors hover:bg-accent"
            >
              <Archive className="h-3.5 w-3.5" />
              {t('kommunikation.header.archive')}
            </button>
            <button
              onClick={() => {
                // Mock-first: no hard-delete RPC — archive instead (backend-gaps.md)
                archiveMsg.mutate(msg.id)
                setSelectedConversation(null)
                setActionsOpen(false)
                toast.success(t('kommunikation.header.messageArchived'))
              }}
              className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-error transition-colors hover:bg-error/10"
            >
              <Trash2 className="h-3.5 w-3.5" />
              {t('common.delete')}
            </button>
          </PopoverContent>
        </Popover>
      </div>

      {/* Row 2: Sender info + assigned + tags */}
      <div className="flex items-center gap-2 mt-1.5 flex-wrap">
        <span className="text-xs text-muted-foreground">
          {msg.sender_name}
          {msg.sender_email && <span className="text-muted-foreground/60"> · {msg.sender_email}</span>}
        </span>

        {/* Assign / assignee */}
        <Popover open={assignOpen} onOpenChange={setAssignOpen}>
          <PopoverTrigger asChild>
            <button className="flex items-center gap-1 rounded bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground hover:bg-accent transition-colors">
              <UserPlus className="h-2.5 w-2.5" />
              {msg.assigned_to ? employeeLabel(msg.assigned_to) : t('kommunikation.assign.assign')}
            </button>
          </PopoverTrigger>
          <PopoverContent className="w-52 p-1 max-h-64 overflow-y-auto" align="start" sideOffset={4}>
            <p className="px-2.5 py-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
              {t('kommunikation.assign.assignTo')}
            </p>
            {employees.length === 0 && (
              <p className="px-2.5 py-2 text-xs text-muted-foreground">{t('kommunikation.assign.noUsers')}</p>
            )}
            {employees.map((emp) => (
              <button
                key={emp.userId}
                onClick={() => handleAssign(emp.userId)}
                className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors hover:bg-accent"
              >
                <span className="truncate">{emp.userName || emp.userEmail || emp.userId}</span>
                {msg.assigned_to === emp.userId && <Check className="ml-auto h-3.5 w-3.5 text-primary" />}
              </button>
            ))}
          </PopoverContent>
        </Popover>

        {/* Claim (only when unassigned) */}
        {!msg.assigned_to && (
          <button
            onClick={handleClaim}
            className="flex items-center gap-1 rounded bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary hover:bg-primary/20 transition-colors"
            title={t('kommunikation.assign.claimHint')}
          >
            <Hand className="h-2.5 w-2.5" />
            {t('kommunikation.assign.claim')}
          </button>
        )}

        {/* Tags */}
        {tags.map((tag) => (
          <span
            key={tag}
            className="group/tag flex items-center gap-0.5 rounded bg-secondary/70 px-1.5 py-0.5 text-[10px] text-muted-foreground"
          >
            {tag}
            <button
              onClick={() => removeTag(msg.id, baseTags, tag)}
              className="text-muted-foreground/50 hover:text-error transition-colors"
              title={t('kommunikation.tags.remove')}
            >
              <X className="h-2.5 w-2.5" />
            </button>
          </span>
        ))}

        {/* Add tag */}
        <Popover open={tagOpen} onOpenChange={setTagOpen}>
          <PopoverTrigger asChild>
            <button className="flex items-center gap-0.5 rounded px-1 py-0.5 text-[10px] text-muted-foreground/60 hover:text-foreground hover:bg-accent transition-colors">
              <Tag className="h-2.5 w-2.5" />
              {t('kommunikation.tags.add')}
            </button>
          </PopoverTrigger>
          <PopoverContent className="w-56 p-2" align="start" sideOffset={4}>
            <div className="flex gap-1.5">
              <input
                type="text"
                value={newTag}
                onChange={(e) => setNewTag(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newTag.trim()) {
                    addTag(msg.id, baseTags, newTag)
                    setNewTag('')
                  }
                }}
                placeholder={t('kommunikation.tags.placeholder')}
                className="flex-1 rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
              />
              <button
                onClick={() => { if (newTag.trim()) { addTag(msg.id, baseTags, newTag); setNewTag('') } }}
                disabled={!newTag.trim()}
                className="rounded-md bg-primary px-2 py-1 text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
              >
                <Plus className="h-3.5 w-3.5" />
              </button>
            </div>
            {suggestions.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-1">
                {suggestions.slice(0, 8).map((s) => (
                  <button
                    key={s}
                    onClick={() => addTag(msg.id, baseTags, s)}
                    className="rounded bg-secondary/70 px-1.5 py-0.5 text-[10px] text-muted-foreground hover:bg-accent transition-colors"
                  >
                    + {s}
                  </button>
                ))}
              </div>
            )}
          </PopoverContent>
        </Popover>
      </div>

      <ForwardDialog message={msg} open={forwardOpen} onOpenChange={setForwardOpen} />
    </div>
  )
}
