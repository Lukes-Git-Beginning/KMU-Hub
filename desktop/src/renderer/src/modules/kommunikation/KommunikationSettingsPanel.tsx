import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { LayoutGrid, MessageSquare, Inbox, Users, GitBranch, Plus, Settings2 } from 'lucide-react'
import { ModuleSettingsShell } from '@/components/shared/ModuleSettingsShell'
import type { ModuleSettingsSection } from '@/components/shared/ModuleSettingsShell'
import { useKommunikationPrefs } from '@/stores/kommunikationPrefs'
import { useTeamInboxes, useCreateTeamInbox } from '@/api/hooks/useInbox'
import type { TeamInbox } from '@/api/inbox-types'
import { TeamInboxSettings } from './TeamInboxSettings'
import { RoutingRulesEditor } from './RoutingRulesEditor'

/**
 * Module settings for the unified Kommunikation module (moduleId 'chat').
 *
 * Personal: display preferences (default area, density, enter-to-send).
 * Tenant (Modul-Leiter only, gated by ModuleSettingsShell lock):
 *   • Team-Postfächer (TeamInboxSettings)
 *   • Routing-Regeln (RoutingRulesEditor)
 */
export function KommunikationSettingsPanel() {
  const { t } = useTranslation()
  const defaultBereich = useKommunikationPrefs((s) => s.defaultBereich)
  const setDefaultBereich = useKommunikationPrefs((s) => s.setDefaultBereich)
  const density = useKommunikationPrefs((s) => s.density)
  const setDensity = useKommunikationPrefs((s) => s.setDensity)
  const enterToSend = useKommunikationPrefs((s) => s.enterToSend)
  const setEnterToSend = useKommunikationPrefs((s) => s.setEnterToSend)

  const sections: ModuleSettingsSection[] = [
    {
      id: 'display',
      titleKey: 'kommunikation.settings.display.title',
      descriptionKey: 'kommunikation.settings.display.desc',
      scope: 'personal',
      icon: LayoutGrid,
      children: (
        <div className="space-y-5">
          {/* Default area */}
          <div className="space-y-2">
            <p className="text-xs font-medium text-foreground">{t('kommunikation.settings.display.defaultArea')}</p>
            <div className="grid grid-cols-2 gap-2">
              <AreaChoice
                active={defaultBereich === 'team'}
                icon={MessageSquare}
                label={t('kommunikation.bereich.team')}
                onClick={() => setDefaultBereich('team')}
              />
              <AreaChoice
                active={defaultBereich === 'posteingang'}
                icon={Inbox}
                label={t('kommunikation.bereich.posteingang')}
                onClick={() => setDefaultBereich('posteingang')}
              />
            </div>
          </div>

          {/* Density */}
          <label className="flex items-center justify-between gap-4">
            <span className="text-xs text-foreground">{t('kommunikation.settings.display.density')}</span>
            <select
              value={density}
              onChange={(e) => setDensity(e.target.value as 'comfortable' | 'compact')}
              className="rounded border border-border bg-input-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              <option value="comfortable">{t('kommunikation.settings.display.densityComfortable')}</option>
              <option value="compact">{t('kommunikation.settings.display.densityCompact')}</option>
            </select>
          </label>

          {/* Enter to send */}
          <label className="flex items-center justify-between gap-4">
            <span className="text-xs text-foreground">{t('kommunikation.settings.display.enterToSend')}</span>
            <button
              type="button"
              role="switch"
              aria-checked={enterToSend}
              onClick={() => setEnterToSend(!enterToSend)}
              className={`flex h-5 w-9 shrink-0 items-center rounded-full px-0.5 transition-colors ${
                enterToSend ? 'justify-end bg-primary' : 'justify-start bg-border'
              }`}
            >
              <span className="h-4 w-4 rounded-full bg-white shadow-sm" />
            </button>
          </label>
        </div>
      ),
    },
    {
      id: 'teamInboxes',
      titleKey: 'kommunikation.settings.teamInboxes.title',
      descriptionKey: 'kommunikation.settings.teamInboxes.desc',
      scope: 'tenant',
      icon: Users,
      children: <TeamInboxSection />,
    },
    {
      id: 'routing',
      titleKey: 'kommunikation.settings.routing.title',
      descriptionKey: 'kommunikation.settings.routing.desc',
      scope: 'tenant',
      icon: GitBranch,
      children: <RoutingRulesEditor />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="chat"
      titleKey="kommunikation.settings.title"
      descriptionKey="kommunikation.settings.desc"
      sections={sections}
    />
  )
}

// ---------------------------------------------------------------------------
// Team inbox section
// ---------------------------------------------------------------------------

function TeamInboxSection() {
  const { t } = useTranslation()
  const { data: teams, isLoading } = useTeamInboxes()
  const createTeam = useCreateTeamInbox()
  const [editingTeam, setEditingTeam] = useState<TeamInbox | null>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)

  const openTeam = (team: TeamInbox) => {
    setEditingTeam(team)
    setSettingsOpen(true)
  }

  const handleCreate = () => {
    createTeam.mutate(
      { name: t('kommunikation.teamInbox.defaultName'), assignment_mode: 'manual', visibility: 'open' },
      { onSuccess: (team) => openTeam(team) },
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-foreground">{t('kommunikation.teamInbox.listTitle')}</h3>
        <button
          onClick={handleCreate}
          disabled={createTeam.isPending}
          className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
        >
          <Plus className="h-3.5 w-3.5" />
          {t('kommunikation.teamInbox.create')}
        </button>
      </div>

      <div className="space-y-1">
        {isLoading && (
          <div className="space-y-2">
            {[1, 2].map((i) => (
              <div key={i} className="h-14 animate-pulse rounded-md bg-secondary" />
            ))}
          </div>
        )}
        {teams?.map((team) => (
          <button
            key={team.id}
            onClick={() => openTeam(team)}
            className="flex w-full items-center gap-3 rounded-md border border-border px-3 py-2.5 text-left hover:bg-secondary/50 transition-colors"
          >
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Users className="h-4 w-4" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-foreground truncate">{team.name}</span>
                <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-secondary-foreground">
                  {team.assignment_mode === 'round_robin'
                    ? t('kommunikation.inbox.roundRobin')
                    : t('kommunikation.teamInbox.manual')}
                </span>
              </div>
              {team.description && (
                <p className="text-[11px] text-muted-foreground truncate">{team.description}</p>
              )}
            </div>
            <Settings2 className="h-4 w-4 shrink-0 text-muted-foreground" />
          </button>
        ))}
        {!isLoading && (!teams || teams.length === 0) && (
          <p className="py-4 text-center text-xs text-muted-foreground">
            {t('kommunikation.teamInbox.empty')}
          </p>
        )}
      </div>

      <TeamInboxSettings team={editingTeam} open={settingsOpen} onOpenChange={setSettingsOpen} />
    </div>
  )
}

function AreaChoice({
  active,
  icon: Icon,
  label,
  onClick,
}: {
  active: boolean
  icon: typeof MessageSquare
  label: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`flex items-center gap-2 rounded-lg border px-3 py-2.5 text-xs font-medium transition-colors ${
        active ? 'border-primary bg-primary-light text-primary' : 'border-border text-foreground hover:bg-secondary'
      }`}
    >
      <Icon className="h-4 w-4" aria-hidden="true" />
      {label}
    </button>
  )
}
