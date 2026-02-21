import { useState } from 'react'
import {
  Mail,
  MessageCircle,
  Globe,
  Headphones,
  Users,
  ArrowUp,
  AlertTriangle,
  MoreHorizontal,
  UserPlus,
  CheckCircle2,
  Clock,
  XCircle,
  Tag,
} from 'lucide-react'
import { toast } from 'sonner'
import type { Conversation, CommunicationChannel, ConversationStatus } from '@/types/communication'
import { useKommunikationStore } from '@/stores/kommunikation'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

const channelConfig: Record<CommunicationChannel, { icon: typeof Mail; color: string; bg: string; label: string }> = {
  email: { icon: Mail, color: 'text-blue-500', bg: 'bg-blue-50 dark:bg-blue-950/30', label: 'E-Mail' },
  teams: { icon: Users, color: 'text-violet-500', bg: 'bg-violet-50 dark:bg-violet-950/30', label: 'Teams' },
  whatsapp: { icon: MessageCircle, color: 'text-green-500', bg: 'bg-green-50 dark:bg-green-950/30', label: 'WhatsApp' },
  widget: { icon: Globe, color: 'text-orange-500', bg: 'bg-orange-50 dark:bg-orange-950/30', label: 'Widget' },
  portal: { icon: Headphones, color: 'text-teal-500', bg: 'bg-teal-50 dark:bg-teal-950/30', label: 'Portal' },
}

const statusOptions: { id: ConversationStatus; label: string; icon: typeof CheckCircle2; color: string }[] = [
  { id: 'open', label: 'Offen', icon: Clock, color: 'text-success' },
  { id: 'pending', label: 'Wartend', icon: Clock, color: 'text-warning' },
  { id: 'resolved', label: 'Geloest', icon: CheckCircle2, color: 'text-blue-500' },
  { id: 'closed', label: 'Geschlossen', icon: XCircle, color: 'text-muted-foreground' },
]

const priorityLabels: Record<string, { icon?: typeof ArrowUp; color: string; label: string }> = {
  urgent: { icon: AlertTriangle, color: 'text-destructive', label: 'Dringend' },
  high: { icon: ArrowUp, color: 'text-warning', label: 'Hoch' },
  normal: { color: '', label: 'Normal' },
  low: { color: '', label: 'Niedrig' },
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ConversationThreadHeaderProps {
  conversation: Conversation
}

export function ConversationThreadHeader({ conversation: conv }: ConversationThreadHeaderProps) {
  const updateStatus = useKommunikationStore((s) => s.updateStatus)
  const addTag = useKommunikationStore((s) => s.addTag)
  const removeTag = useKommunikationStore((s) => s.removeTag)
  const [tagInput, setTagInput] = useState('')
  const [statusOpen, setStatusOpen] = useState(false)

  const ch = channelConfig[conv.channel]
  const ChannelIcon = ch.icon
  const prio = priorityLabels[conv.priority]
  const PrioIcon = prio?.icon

  const handleStatusChange = (status: ConversationStatus) => {
    updateStatus(conv.id, status)
    setStatusOpen(false)
    toast.success(`Status: ${statusOptions.find((s) => s.id === status)?.label}`)
  }

  const handleAddTag = () => {
    const tag = tagInput.trim().toLowerCase()
    if (!tag) return
    addTag(conv.id, tag)
    setTagInput('')
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
          {conv.subject}
        </h2>

        {/* Priority badge */}
        {PrioIcon && (
          <span className={`flex items-center gap-1 text-[11px] font-medium ${prio.color}`}>
            <PrioIcon className="h-3.5 w-3.5" />
            {prio.label}
          </span>
        )}

        {/* Status dropdown */}
        <Popover open={statusOpen} onOpenChange={setStatusOpen}>
          <PopoverTrigger asChild>
            <button className={`flex items-center gap-1 rounded-full px-2.5 py-1 text-[11px] font-medium transition-colors hover:opacity-80 ${
              conv.status === 'open' ? 'bg-success/15 text-success' :
              conv.status === 'pending' ? 'bg-warning/15 text-warning' :
              conv.status === 'resolved' ? 'bg-blue-500/15 text-blue-500' :
              'bg-muted text-muted-foreground'
            }`}>
              {statusOptions.find((s) => s.id === conv.status)?.label ?? conv.status}
            </button>
          </PopoverTrigger>
          <PopoverContent className="w-40 p-1" align="end" sideOffset={4}>
            {statusOptions.map((opt) => {
              const Icon = opt.icon
              return (
                <button
                  key={opt.id}
                  onClick={() => handleStatusChange(opt.id)}
                  className={`flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors hover:bg-accent ${
                    conv.status === opt.id ? 'bg-accent font-medium' : ''
                  }`}
                >
                  <Icon className={`h-3.5 w-3.5 ${opt.color}`} />
                  {opt.label}
                </button>
              )
            })}
          </PopoverContent>
        </Popover>

        {/* More menu placeholder */}
        <button className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors">
          <MoreHorizontal className="h-4 w-4" />
        </button>
      </div>

      {/* Row 2: Contact info + assigned + tags */}
      <div className="flex items-center gap-2 mt-1.5 flex-wrap">
        <span className="text-xs text-muted-foreground">
          {conv.contactName}
          {conv.contactCompany && <span className="text-muted-foreground/60"> · {conv.contactCompany}</span>}
        </span>

        {conv.assignedTo && (
          <span className="flex items-center gap-1 rounded bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground">
            <UserPlus className="h-2.5 w-2.5" />
            {conv.assignedTo}
          </span>
        )}

        {/* Tags */}
        {conv.tags.map((tag) => (
          <span
            key={tag}
            className="group flex items-center gap-0.5 rounded bg-secondary/70 px-1.5 py-0.5 text-[10px] text-muted-foreground"
          >
            {tag}
            <button
              onClick={() => removeTag(conv.id, tag)}
              className="hidden group-hover:inline-flex h-3 w-3 items-center justify-center rounded-full hover:bg-destructive/20 hover:text-destructive transition-colors"
            >
              ×
            </button>
          </span>
        ))}

        {/* Add tag */}
        <div className="flex items-center gap-0.5">
          <Tag className="h-2.5 w-2.5 text-muted-foreground/50" />
          <input
            type="text"
            value={tagInput}
            onChange={(e) => setTagInput(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') handleAddTag() }}
            placeholder="+Tag"
            className="w-12 bg-transparent text-[10px] text-muted-foreground outline-none placeholder:text-muted-foreground/40 focus:w-20 transition-all"
          />
        </div>
      </div>
    </div>
  )
}
