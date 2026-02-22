/**
 * QuickActionsBar — row of shortcut buttons below the dashboard greeting.
 *
 * Module-aware: filters actions based on active business profile.
 */
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
  label: string
  icon: React.ElementType
  route: string
  moduleId?: string
}

const allActions: QuickAction[] = [
  { id: 'contact', label: 'Neuer Kontakt', icon: UserPlus, route: '/kontakte' },
  { id: 'invoice', label: 'Neue Rechnung', icon: Receipt, route: '/finanzen', moduleId: 'finanzen' },
  { id: 'timer', label: 'Timer starten', icon: Timer, route: '/zeiterfassung', moduleId: 'zeiterfassung' },
  { id: 'project', label: 'Neues Projekt', icon: FolderPlus, route: '/work' },
  { id: 'meeting', label: 'Neues Meeting', icon: Video, route: '/meetings' },
  { id: 'message', label: 'Neue Nachricht', icon: MessageSquare, route: '/chat' },
  { id: 'event', label: 'Neuer Termin', icon: CalendarPlus, route: '/calendar' },
  { id: 'document', label: 'Neues Dokument', icon: FileText, route: '/dokumente' },
]

export function QuickActionsBar() {
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
    <div className="flex flex-wrap gap-2 mb-6">
      {visibleActions.map((action) => (
        <button
          key={action.id}
          onClick={() => navigate(action.route)}
          className="inline-flex items-center gap-2 rounded-lg border border-border/60 bg-card px-3 py-2 text-sm font-medium text-foreground hover:bg-muted hover:border-border transition-colors"
        >
          <action.icon className="h-4 w-4 text-muted-foreground" />
          {action.label}
        </button>
      ))}
    </div>
  )
}
