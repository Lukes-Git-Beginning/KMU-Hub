/**
 * QuickActions widget -- grid of shortcut buttons for common actions.
 *
 * Pure navigation component -- no API calls needed.
 * Each button has an icon and label, navigating to the relevant module.
 */
import { memo } from 'react'
import { useTranslation } from 'react-i18next'
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
  { labelKey: 'dashboard.quickActions.newContact', icon: UserPlus, path: '/crm/contacts' },
  { labelKey: 'dashboard.quickActions.newDeal', icon: DollarSign, path: '/crm/deals' },
  { labelKey: 'dashboard.quickActions.newMessage', icon: MessageSquarePlus, path: '/chat' },
  { labelKey: 'dashboard.quickActions.newActivity', icon: PlusCircle, path: '/crm/activities' },
  { labelKey: 'common.search', icon: Search, path: '/crm/search' },
  { labelKey: 'nav.notifications', icon: Bell, path: '/notifications' },
] as const

function QuickActions(_props: WidgetProps) {
  const { t } = useTranslation()
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
            <span className="text-xs font-medium">{t(action.labelKey)}</span>
          </button>
        )
      })}
    </div>
  )
}

export default memo(QuickActions)
