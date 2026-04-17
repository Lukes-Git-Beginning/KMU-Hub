/**
 * QuickActionsBar — row of shortcut buttons below the dashboard greeting.
 *
 * Module-aware: filters actions based on active business profile.
 */
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  UserPlus,
  FileText,
  Timer,
  FolderPlus,
  Video,
  MessageSquare,
  Receipt,
  CalendarPlus,
} from 'lucide-react'
import { useProfileStore } from '../../stores/profile'
import { getProfileById } from '../../config/business-profiles'

interface QuickAction {
  id: string
  labelKey: string
  icon: React.ElementType
  route: string
  moduleId?: string
}

const allActions: QuickAction[] = [
  { id: 'contact', labelKey: 'dashboard.quickActions.newContact', icon: UserPlus, route: '/kontakte' },
  { id: 'invoice', labelKey: 'dashboard.quickActions.newInvoice', icon: Receipt, route: '/finanzen', moduleId: 'finanzen' },
  { id: 'timer', labelKey: 'dashboard.quickActions.startTimer', icon: Timer, route: '/zeiterfassung', moduleId: 'zeiterfassung' },
  { id: 'project', labelKey: 'dashboard.quickActions.newProject', icon: FolderPlus, route: '/work' },
  { id: 'meeting', labelKey: 'dashboard.quickActions.newMeeting', icon: Video, route: '/meetings' },
  { id: 'message', labelKey: 'dashboard.quickActions.newMessage', icon: MessageSquare, route: '/chat' },
  { id: 'event', labelKey: 'dashboard.quickActions.newEvent', icon: CalendarPlus, route: '/calendar' },
  { id: 'document', labelKey: 'dashboard.quickActions.newDocument', icon: FileText, route: '/dokumente' },
]

export function QuickActionsBar() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { businessProfileId, devShowAllModules, enabledOptionalModules } = useProfileStore()

  // Filter actions based on business profile
  const profile = businessProfileId ? getProfileById(businessProfileId) : null
  const visibleActions = allActions.filter((action) => {
    if (!action.moduleId) return true
    if (devShowAllModules) return true
    if (!profile) return true
    // Check if module is in core or enabled optional
    const isCore = profile.defaultModules.includes(action.moduleId)
    const isOptional = enabledOptionalModules.includes(action.moduleId)
    return isCore || isOptional
  })

  return (
    <div className="grid auto-cols-fr grid-flow-col gap-3">
      {visibleActions.map((action, i) => (
        <button
          key={action.id}
          onClick={() => navigate(action.route)}
          className="flex items-center justify-center gap-2 rounded-xl border border-border/60 bg-card px-3 py-3 text-sm font-medium text-foreground hover:bg-muted hover:border-border hover:shadow-sm transition-all"
        >
          <action.icon className={`h-4 w-4 shrink-0 ${i % 2 === 0 ? 'icon-accent' : 'icon-accent-2'} text-muted-foreground`} />
          <span className="truncate">{t(action.labelKey)}</span>
        </button>
      ))}
    </div>
  )
}
