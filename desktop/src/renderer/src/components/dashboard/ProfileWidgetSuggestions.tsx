/**
 * ProfileWidgetSuggestions — recommends dashboard widgets based on
 * the active business profile.
 *
 * Dismissable suggestion cards: "Widget X hinzufügen".
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  Truck,
  FileCheck,
  Clock,
  Ticket,
  Rocket,
  MessageSquare,
  Receipt,
  Users,
  X,
  Plus,
} from 'lucide-react'
import { useProfileStore } from '../../stores/profile'
import { useDashboardStore, type WidgetId } from '../../stores/dashboard'
import { moduleHsl, moduleHslBg } from '../../components/layout/sidebar/nav-items'
import type { BusinessProfileId } from '../../config/business-profiles'
import { useCapabilitySet } from '@/hooks/useCapability'
import { moduleViewKey, resolveModuleKey } from '@/config/capabilities'
import { widgetRegistry } from '@/components/widgets/WidgetRegistry'

interface WidgetSuggestion {
  id: string
  labelKey: string
  descriptionKey: string
  icon: React.ElementType
  /** nav-items module ID for color lookup */
  moduleId: string
  /** real dashboard widget added when the user clicks "+" */
  widgetId: WidgetId
}

const profileSuggestions: Record<string, WidgetSuggestion[]> = {
  handwerk: [
    { id: 'fuhrpark-status', labelKey: 'dashboard.suggestions.fleetStatus', descriptionKey: 'dashboard.suggestions.fleetStatusDesc', icon: Truck, moduleId: 'fuhrpark', widgetId: 'cross-module-overview' },
    { id: 'aktive-rapporte', labelKey: 'dashboard.suggestions.activeReports', descriptionKey: 'dashboard.suggestions.activeReportsDesc', icon: FileCheck, moduleId: 'rapporte', widgetId: 'my-tasks' },
    { id: 'zeiterfassung-quick', labelKey: 'dashboard.suggestions.timeTracking', descriptionKey: 'dashboard.suggestions.timeTrackingDesc', icon: Clock, moduleId: 'zeiterfassung', widgetId: 'time-clock' },
  ],
  it_tech: [
    { id: 'offene-tickets', labelKey: 'dashboard.suggestions.openTickets', descriptionKey: 'dashboard.suggestions.openTicketsDesc', icon: Ticket, moduleId: 'helpdesk', widgetId: 'open-tickets' },
    { id: 'sprint-status', labelKey: 'dashboard.suggestions.sprintStatus', descriptionKey: 'dashboard.suggestions.sprintStatusDesc', icon: Rocket, moduleId: 'projects', widgetId: 'my-tasks' },
    { id: 'nachrichten-quick', labelKey: 'dashboard.suggestions.messages', descriptionKey: 'dashboard.suggestions.messagesDesc', icon: MessageSquare, moduleId: 'chat', widgetId: 'unread-messages' },
  ],
  dienstleistung: [
    { id: 'nächste-termine', labelKey: 'dashboard.suggestions.nextAppointments', descriptionKey: 'dashboard.suggestions.nextAppointmentsDesc', icon: Clock, moduleId: 'calendar', widgetId: 'calendar-upcoming' },
    { id: 'offene-rechnungen', labelKey: 'dashboard.suggestions.openInvoices', descriptionKey: 'dashboard.suggestions.openInvoicesDesc', icon: Receipt, moduleId: 'finance', widgetId: 'kpi-revenue' },
    { id: 'kontakte-quick', labelKey: 'dashboard.suggestions.contacts', descriptionKey: 'dashboard.suggestions.contactsDesc', icon: Users, moduleId: 'contacts', widgetId: 'recent-contacts' },
  ],
}

// Map business profile IDs to suggestion categories
function getSuggestionsForProfile(profileId: BusinessProfileId | null): WidgetSuggestion[] {
  if (!profileId) return []
  if (profileId.includes('handwerk') || profileId.includes('bau')) return profileSuggestions.handwerk
  if (profileId.includes('it') || profileId.includes('tech') || profileId.includes('agentur')) return profileSuggestions.it_tech
  if (profileId.includes('dienstleist') || profileId.includes('beratung') || profileId.includes('kanzlei')) return profileSuggestions.dienstleistung
  // Default: show general suggestions
  return profileSuggestions.dienstleistung
}

export function ProfileWidgetSuggestions() {
  const { t } = useTranslation()
  const { businessProfileId } = useProfileStore()
  const addWidget = useDashboardStore((s) => s.addWidget)
  const [dismissed, setDismissed] = useState<Set<string>>(new Set())
  const [allDismissed, setAllDismissed] = useState(false)

  // RBAC: never suggest a widget whose module the role cannot see (R-3).
  const { has: hasCapability, ready: rbacReady } = useCapabilitySet()
  const isSuggestionVisible = (s: WidgetSuggestion): boolean => {
    const rbacModule = resolveModuleKey(widgetRegistry[s.widgetId]?.module ?? s.moduleId)
    return !rbacModule || hasCapability(moduleViewKey(rbacModule))
  }

  const suggestions = getSuggestionsForProfile(businessProfileId)
  const visible = suggestions.filter((s) => !dismissed.has(s.id) && isSuggestionVisible(s))

  if (!rbacReady || allDismissed || visible.length === 0 || !businessProfileId) return null

  return (
    <div className="mb-6">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          {t('dashboard.suggestions.title')}
        </h3>
        <button
          onClick={() => setAllDismissed(true)}
          className="text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          {t('dashboard.suggestions.hideAll')}
        </button>
      </div>
      <div className="flex flex-wrap gap-3">
        {visible.map((suggestion) => (
          <div
            key={suggestion.id}
            className="group relative flex items-center gap-3 rounded-lg border border-dashed border-border/60 bg-card/50 px-4 py-3 hover:border-border hover:bg-card transition-colors"
          >
            <div
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg"
              style={{ backgroundColor: moduleHslBg(suggestion.moduleId), color: moduleHsl(suggestion.moduleId) }}
            >
              <suggestion.icon className="h-4.5 w-4.5" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">{t(suggestion.labelKey)}</p>
              <p className="text-xs text-muted-foreground">{t(suggestion.descriptionKey)}</p>
            </div>
            <button
              onClick={() => {
                addWidget(suggestion.widgetId, 'personal')
                toast.success(t('dashboard.suggestions.added', { name: t(suggestion.labelKey) }))
                setDismissed((prev) => new Set(prev).add(suggestion.id))
              }}
              aria-label={t('dashboard.suggestions.add')}
              className="ml-2 rounded-md bg-primary/10 p-1.5 text-primary hover:bg-primary/20 transition-colors"
            >
              <Plus className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={() => setDismissed((prev) => new Set(prev).add(suggestion.id))}
              className="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-muted text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity hover:bg-muted/80"
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
