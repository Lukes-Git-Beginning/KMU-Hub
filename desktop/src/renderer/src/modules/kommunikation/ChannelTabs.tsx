import { useTranslation } from 'react-i18next'
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
  labelKey: string
  icon: typeof Mail
  color: string
}

const channels: ChannelDef[] = [
  { id: 'all', labelKey: 'kommunikation.channels.all', icon: Inbox, color: 'text-foreground' },
  { id: 'email', labelKey: 'kommunikation.channels.email', icon: Mail, color: 'text-blue-500' },
  { id: 'teams', labelKey: 'kommunikation.channels.teams', icon: Users, color: 'text-violet-500' },
  { id: 'whatsapp', labelKey: 'kommunikation.channels.whatsapp', icon: MessageCircle, color: 'text-green-500' },
  { id: 'widget', labelKey: 'kommunikation.channels.widget', icon: Globe, color: 'text-orange-500' },
  { id: 'portal', labelKey: 'kommunikation.channels.portal', icon: Headphones, color: 'text-teal-500' },
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
  const { t } = useTranslation()
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
            <span>{t(ch.labelKey)}</span>
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
