/**
 * QuickActions widget -- grid of shortcut buttons for common actions.
 *
 * Pure navigation component -- no API calls needed.
 * Each button has an icon and label, navigating to the relevant module.
 */
import { memo } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  UserPlus,
  DollarSign,
  MessageSquarePlus,
  PlusCircle,
  Search,
  Bell,
} from 'lucide-react'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const actions = [
  { label: 'Neuer Kontakt', icon: UserPlus, path: '/crm/contacts' },
  { label: 'Neuer Deal', icon: DollarSign, path: '/crm/deals' },
  { label: 'Neue Nachricht', icon: MessageSquarePlus, path: '/chat' },
  { label: 'Neue Aktivitaet', icon: PlusCircle, path: '/crm/activities' },
  { label: 'Suche', icon: Search, path: '/crm/search' },
  { label: 'Benachrichtigungen', icon: Bell, path: '/notifications' },
] as const

function QuickActions(_props: WidgetProps) {
  const navigate = useNavigate()

  return (
    <div className="grid grid-cols-3 gap-2 p-4">
      {actions.map((action) => {
        const Icon = action.icon
        return (
          <button
            key={action.path}
            className="flex flex-col items-center gap-1.5 rounded-lg p-2.5 text-center transition-colors hover:bg-accent/50"
            onClick={() => navigate(action.path)}
          >
            <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary/10">
              <Icon className="h-4 w-4 text-primary" />
            </div>
            <span className="text-xs font-medium">{action.label}</span>
          </button>
        )
      })}
    </div>
  )
}

export default memo(QuickActions)
