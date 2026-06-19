/**
 * Trigger selector component for the automation wizard.
 *
 * Shows available triggers grouped by module with search filter.
 * Uses useTriggerDefinitions() hook to load catalog from backend.
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Search,
  Users,
  Briefcase,
  Mail,
  Receipt,
  UserCheck,
  Calendar,
  LifeBuoy,
  Settings,
  Zap,
} from 'lucide-react'
import { useTriggerDefinitions } from '@/api/hooks/useAutomation'
import { useAutomatisierungStore } from '@/stores/automatisierung'
import type { TriggerDefinition } from '@/api/automation-types'

// ---------------------------------------------------------------------------
// Module icons and labels
// ---------------------------------------------------------------------------

const MODULE_CONFIG: Record<
  string,
  { icon: React.ElementType; labelKey: string; color: string }
> = {
  crm: { icon: Users, labelKey: 'automatisierung.modules.crm', color: 'text-blue-500' },
  work: { icon: Briefcase, labelKey: 'automatisierung.modules.work', color: 'text-purple-500' },
  email: { icon: Mail, labelKey: 'automatisierung.modules.email', color: 'text-green-500' },
  finance: { icon: Receipt, labelKey: 'automatisierung.modules.finance', color: 'text-warning-foreground' },
  hr: { icon: UserCheck, labelKey: 'automatisierung.modules.hr', color: 'text-rose-500' },
  calendar: { icon: Calendar, labelKey: 'automatisierung.modules.calendar', color: 'text-cyan-500' },
  support: { icon: LifeBuoy, labelKey: 'automatisierung.modules.support', color: 'text-orange-500' },
  system: { icon: Settings, labelKey: 'automatisierung.modules.system', color: 'text-slate-500' },
}

function getModuleConfig(module: string) {
  return (
    MODULE_CONFIG[module] ?? {
      icon: Zap,
      labelKey: module,
      color: 'text-muted-foreground',
    }
  )
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function TriggerSelector() {
  const { t } = useTranslation()
  const { data, isLoading } = useTriggerDefinitions()
  const { draftWorkflow, updateDraftTrigger } = useAutomatisierungStore()
  const [search, setSearch] = useState('')

  const triggers = useMemo(() => data?.triggers ?? [], [data?.triggers])
  // Selection is tracked by event_type (unique); multiple triggers share type 'event'.
  const selectedEvent =
    (draftWorkflow?.trigger_config?.event_type as string) ?? draftWorkflow?.trigger_type ?? ''

  // Group by module
  const grouped = useMemo(() => {
    const filtered = search
      ? triggers.filter(
          (t) =>
            t.name.toLowerCase().includes(search.toLowerCase()) ||
            t.description.toLowerCase().includes(search.toLowerCase()) ||
            t.module.toLowerCase().includes(search.toLowerCase()),
        )
      : triggers

    const groups: Record<string, TriggerDefinition[]> = {}
    for (const t of filtered) {
      const mod = t.module || 'other'
      if (!groups[mod]) groups[mod] = []
      groups[mod].push(t)
    }
    return groups
  }, [triggers, search])

  const handleSelect = (trigger: TriggerDefinition) => {
    updateDraftTrigger(trigger.type, { event_type: trigger.event_type })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t('automatisierung.trigger.searchPlaceholder')}
          className="w-full rounded-md border border-border bg-background pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
        />
      </div>

      {/* Grouped triggers */}
      {Object.entries(grouped).map(([module, moduleTriggers]) => {
        const config = getModuleConfig(module)
        const Icon = config.icon
        return (
          <div key={module}>
            <h3 className="flex items-center gap-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
              <Icon className={`h-4 w-4 ${config.color}`} />
              {t(config.labelKey)}
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
              {moduleTriggers.map((trigger) => {
                const isSelected = selectedEvent === trigger.event_type
                return (
                  <button
                    key={trigger.event_type}
                    onClick={() => handleSelect(trigger)}
                    className={`flex flex-col items-start gap-1 rounded-lg border p-3 text-left transition-colors ${
                      isSelected
                        ? 'border-primary bg-primary/5 ring-1 ring-primary'
                        : 'border-border hover:border-primary/50 hover:bg-secondary/50'
                    }`}
                  >
                    <div className="flex items-center gap-2">
                      <Icon
                        className={`h-4 w-4 ${isSelected ? 'text-primary' : config.color}`}
                      />
                      <span className="text-sm font-medium text-foreground">
                        {trigger.name}
                      </span>
                    </div>
                    <p className="text-xs text-muted-foreground line-clamp-2">
                      {trigger.description}
                    </p>
                  </button>
                )
              })}
            </div>
          </div>
        )
      })}

      {Object.keys(grouped).length === 0 && (
        <div className="py-8 text-center text-sm text-muted-foreground">
          {t('automatisierung.trigger.noResults')}
        </div>
      )}
    </div>
  )
}
