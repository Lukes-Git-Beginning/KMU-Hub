import { useState } from 'react'
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
  StickyNote,
  Star,
} from 'lucide-react'
import { toast } from 'sonner'
import type { InboxMessage, InboxChannel } from '@/api/inbox-types'
import {
  useMarkRead,
  useMarkUnread,
  useToggleStar,
  useArchiveMessage,
  useAssignMessage,
} from '@/api/hooks/useInbox'
import { useKommunikationStore } from '@/stores/kommunikation'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

const channelConfig: Record<InboxChannel, { icon: typeof Mail; color: string; bg: string; label: string }> = {
  email: { icon: Mail, color: 'text-blue-500', bg: 'bg-blue-50 dark:bg-blue-950/30', label: 'E-Mail' },
  chat: { icon: MessageSquare, color: 'text-green-500', bg: 'bg-green-50 dark:bg-green-950/30', label: 'Chat' },
  notification: { icon: Bell, color: 'text-orange-500', bg: 'bg-orange-50 dark:bg-orange-950/30', label: 'Benachrichtigung' },
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ConversationThreadHeaderProps {
  message: InboxMessage
}

export function ConversationThreadHeader({ message: msg }: ConversationThreadHeaderProps) {
  const setSelectedConversation = useKommunikationStore((s) => s.setSelectedConversation)
  const markRead = useMarkRead()
  const markUnread = useMarkUnread()
  const toggleStar = useToggleStar()
  const archiveMsg = useArchiveMessage()
   
  const _assignMsg = useAssignMessage()

  const [actionsOpen, setActionsOpen] = useState(false)

  const ch = channelConfig[msg.channel]
  const ChannelIcon = ch.icon

  const handleToggleRead = () => {
    if (msg.is_read) {
      markUnread.mutate(msg.id)
      toast.success('Als ungelesen markiert')
    } else {
      markRead.mutate(msg.id)
      toast.success('Als gelesen markiert')
    }
    setActionsOpen(false)
  }

  const handleArchive = () => {
    archiveMsg.mutate(msg.id)
    setSelectedConversation(null)
    setActionsOpen(false)
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

        {/* Star toggle */}
        <button
          onClick={() => toggleStar.mutate(msg.id)}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          title={msg.is_starred ? 'Stern entfernen' : 'Stern vergeben'}
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
              {msg.is_read ? 'Als ungelesen markieren' : 'Als gelesen markieren'}
            </button>
            <button
              onClick={() => { toast.info('Weiterleitung geöffnet'); setActionsOpen(false) }}
              className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors hover:bg-accent"
            >
              <Forward className="h-3.5 w-3.5" />
              Weiterleiten
            </button>
            <button
              onClick={() => {
                // TODO: No internal note API exists yet
                toast.info('Interne Notizen sind noch nicht per API verfügbar')
                setActionsOpen(false)
              }}
              className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors hover:bg-accent"
            >
              <StickyNote className="h-3.5 w-3.5" />
              Notiz hinzufügen
            </button>
            <button
              onClick={() => { toast.success('Aufgabe erstellt'); setActionsOpen(false) }}
              className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors hover:bg-accent"
            >
              <ListTodo className="h-3.5 w-3.5" />
              Aufgabe erstellen
            </button>
            <div className="my-1 border-t border-border" />
            <button
              onClick={handleArchive}
              className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors hover:bg-accent"
            >
              <Archive className="h-3.5 w-3.5" />
              Archivieren
            </button>
            <button
              onClick={() => {
                // TODO: No delete API for messages — archive instead
                archiveMsg.mutate(msg.id)
                setSelectedConversation(null)
                setActionsOpen(false)
                toast.success('Nachricht archiviert')
              }}
              className="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-error transition-colors hover:bg-error/10"
            >
              <Trash2 className="h-3.5 w-3.5" />
              Löschen
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

        {msg.assigned_to && (
          <span className="flex items-center gap-1 rounded bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground">
            <UserPlus className="h-2.5 w-2.5" />
            {msg.assigned_to}
          </span>
        )}

        {/* Tags */}
        {msg.tags.map((tag) => (
          <span
            key={tag}
            className="flex items-center gap-0.5 rounded bg-secondary/70 px-1.5 py-0.5 text-[10px] text-muted-foreground"
          >
            {tag}
            {/* TODO: No tag remove API exists yet */}
          </span>
        ))}

        {/* Add tag placeholder */}
        <div className="flex items-center gap-0.5">
          <Tag className="h-2.5 w-2.5 text-muted-foreground/50" />
          <span className="text-[10px] text-muted-foreground/40">
            {/* TODO: No tag add API exists yet */}
            +Tag
          </span>
        </div>
      </div>
    </div>
  )
}
