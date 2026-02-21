import {
  Mail,
  MessageCircle,
  Globe,
  Headphones,
  Users,
  Inbox,
} from 'lucide-react'
import type { CommunicationChannel } from '@/types/communication'

// ---------------------------------------------------------------------------
// Channel config
// ---------------------------------------------------------------------------

interface ChannelDef {
  id: CommunicationChannel | 'all'
  label: string
  icon: typeof Mail
  color: string
}

const channels: ChannelDef[] = [
  { id: 'all', label: 'Alle', icon: Inbox, color: 'text-foreground' },
  { id: 'email', label: 'E-Mail', icon: Mail, color: 'text-blue-500' },
  { id: 'teams', label: 'Teams', icon: Users, color: 'text-violet-500' },
  { id: 'whatsapp', label: 'WhatsApp', icon: MessageCircle, color: 'text-green-500' },
  { id: 'widget', label: 'Widget', icon: Globe, color: 'text-orange-500' },
  { id: 'portal', label: 'Portal', icon: Headphones, color: 'text-teal-500' },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ChannelTabsProps {
  active: CommunicationChannel | 'all'
  onChange: (channel: CommunicationChannel | 'all') => void
  unreadCounts: Record<string, number>
}

export function ChannelTabs({ active, onChange, unreadCounts }: ChannelTabsProps) {
  return (
    <div className="flex items-center gap-1 overflow-x-auto px-3 py-2 border-b border-border">
      {channels.map((ch) => {
        const Icon = ch.icon
        const isActive = active === ch.id
        const count = ch.id === 'all'
          ? Object.values(unreadCounts).reduce((s, n) => s + n, 0)
          : unreadCounts[ch.id] ?? 0
        return (
          <button
            key={ch.id}
            onClick={() => onChange(ch.id)}
            className={`flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium whitespace-nowrap transition-colors ${
              isActive
                ? 'bg-primary/10 text-primary'
                : 'text-muted-foreground hover:bg-accent hover:text-foreground'
            }`}
          >
            <Icon className={`h-3.5 w-3.5 ${isActive ? 'text-primary' : ch.color}`} />
            <span>{ch.label}</span>
            {count > 0 && (
              <span className={`flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[10px] font-semibold ${
                isActive
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-secondary text-muted-foreground'
              }`}>
                {count}
              </span>
            )}
          </button>
        )
      })}
    </div>
  )
}
