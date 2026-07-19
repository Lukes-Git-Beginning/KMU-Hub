import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { LayoutGrid, MessageSquare, Inbox, Users, GitBranch, Plus, Settings2, Bell, Circle, MessageSquareReply, Plug, Webhook, ShieldCheck, Mail } from 'lucide-react'
import { ModuleSettingsShell } from '@/components/shared/ModuleSettingsShell'
import type { ModuleSettingsSection } from '@/components/shared/ModuleSettingsShell'
import { useKommunikationPrefs, type MutableChannel } from '@/stores/kommunikationPrefs'
import { usePresenceStore } from '@/stores/presence'
import type { PresenceLevel } from '@/api/video-types'
import { useHasCapability } from '@/hooks/useCapability'
import { useTeamInboxes, useCreateTeamInbox } from '@/api/hooks/useInbox'
import type { TeamInbox } from '@/api/inbox-types'
import { TeamInboxSettings } from './TeamInboxSettings'
import { RoutingRulesEditor } from './RoutingRulesEditor'
import { CannedResponseManager } from './CannedResponseManager'
import { WebhookConfig } from './WebhookConfig'
import { ChannelSettingsDialog } from './ChannelSettingsDialog'

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
  // RBAC (R-3): management-only tenant sections are dropped entirely for
  // roles without the key (no empty section shells — e.g. it_admin holds
  // settings:tenant:manage + webhook:manage but none of the other keys).
  const canRouting = useHasCapability('kommunikation:routing:manage')
  const canChannels = useHasCapability('kommunikation:channel:manage')
  const canCanned = useHasCapability('kommunikation:canned:manage')
  const canWebhooks = useHasCapability('kommunikation:webhook:manage')
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
      id: 'presence',
      titleKey: 'kommunikation.settings.presence.title',
      descriptionKey: 'kommunikation.settings.presence.desc',
      scope: 'personal',
      icon: Circle,
      children: <PresenceSection />,
    },
    {
      id: 'notifications',
      titleKey: 'kommunikation.settings.notifications.title',
      descriptionKey: 'kommunikation.settings.notifications.desc',
      scope: 'personal',
      icon: Bell,
      children: <NotificationsSection />,
    },
    {
      id: 'teamInboxes',
      titleKey: 'kommunikation.settings.teamInboxes.title',
      descriptionKey: 'kommunikation.settings.teamInboxes.desc',
      scope: 'tenant',
      icon: Users,
      children: <TeamInboxSection />,
    },
    ...(canRouting
      ? [{
          id: 'routing',
          titleKey: 'kommunikation.settings.routing.title',
          descriptionKey: 'kommunikation.settings.routing.desc',
          scope: 'tenant' as const,
          icon: GitBranch,
          children: <RoutingRulesEditor />,
        }]
      : []),
    ...(canChannels
      ? [{
          id: 'channels',
          titleKey: 'kommunikation.settings.channels.title',
          descriptionKey: 'kommunikation.settings.channels.desc',
          scope: 'tenant' as const,
          icon: Plug,
          children: <ChannelsSection />,
        }]
      : []),
    ...(canCanned
      ? [{
          id: 'canned',
          titleKey: 'kommunikation.settings.canned.title',
          descriptionKey: 'kommunikation.settings.canned.desc',
          scope: 'tenant' as const,
          icon: MessageSquareReply,
          children: <CannedResponseManager />,
        }]
      : []),
    ...(canWebhooks
      ? [{
          id: 'webhooks',
          titleKey: 'kommunikation.settings.webhooks.title',
          descriptionKey: 'kommunikation.settings.webhooks.desc',
          scope: 'tenant' as const,
          icon: Webhook,
          children: <WebhookConfig />,
        }]
      : []),
    {
      id: 'retention',
      titleKey: 'kommunikation.settings.retention.title',
      descriptionKey: 'kommunikation.settings.retention.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: (
        <div className="flex items-start gap-3 rounded-lg border border-border bg-secondary/30 p-3">
          <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
          <p className="text-xs leading-relaxed text-muted-foreground">{t('kommunikation.settings.retention.notice')}</p>
        </div>
      ),
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
  const canManageTeamInbox = useHasCapability('kommunikation:team_inbox:manage')
  const { data: teams, isLoading } = useTeamInboxes()
  const createTeam = useCreateTeamInbox()
  const [editingTeam, setEditingTeam] = useState<TeamInbox | null>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)

  const openTeam = (team: TeamInbox) => {
    if (!canManageTeamInbox) return
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
        {canManageTeamInbox && (
          <button
            onClick={handleCreate}
            disabled={createTeam.isPending}
            className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
          >
            <Plus className="h-3.5 w-3.5" />
            {t('kommunikation.teamInbox.create')}
          </button>
        )}
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
          canManageTeamInbox ? (
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
          ) : (
            <div
              key={team.id}
              className="flex w-full items-center gap-3 rounded-md border border-border px-3 py-2.5"
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
            </div>
          )
        ))}
        {!isLoading && (!teams || teams.length === 0) && (
          <p className="py-4 text-center text-xs text-muted-foreground">
            {t('kommunikation.teamInbox.empty')}
          </p>
        )}
      </div>

      {canManageTeamInbox && (
        <TeamInboxSettings team={editingTeam} open={settingsOpen} onOpenChange={setSettingsOpen} />
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Presence (own status) — personal
// ---------------------------------------------------------------------------

const STATUS_OPTIONS: { value: PresenceLevel; labelKey: string; dot: string }[] = [
  { value: 'online', labelKey: 'kommunikation.settings.presence.online', dot: 'bg-emerald-500' },
  { value: 'away', labelKey: 'kommunikation.settings.presence.away', dot: 'bg-amber-400' },
  { value: 'dnd', labelKey: 'kommunikation.settings.presence.dnd', dot: 'bg-red-500' },
]

function PresenceSection() {
  const { t } = useTranslation()
  const myStatus = usePresenceStore((s) => s.myStatus)
  const setMyStatus = usePresenceStore((s) => s.setMyStatus)

  return (
    <div className="grid grid-cols-3 gap-2">
      {STATUS_OPTIONS.map((opt) => (
        <button
          key={opt.value}
          type="button"
          onClick={() => setMyStatus(opt.value)}
          aria-pressed={myStatus === opt.value}
          className={`flex items-center gap-2 rounded-lg border px-3 py-2.5 text-xs font-medium transition-colors ${
            myStatus === opt.value ? 'border-primary bg-primary-light text-primary' : 'border-border text-foreground hover:bg-secondary'
          }`}
        >
          <span className={`h-2 w-2 rounded-full ${opt.dot}`} />
          {t(opt.labelKey)}
        </button>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Notifications (per-channel mute) — personal
// ---------------------------------------------------------------------------

const MUTE_CHANNELS: { value: MutableChannel; labelKey: string; icon: typeof Mail }[] = [
  { value: 'email', labelKey: 'kommunikation.channels.email', icon: Mail },
  { value: 'chat', labelKey: 'kommunikation.settings.notifications.chat', icon: MessageSquare },
  { value: 'notification', labelKey: 'kommunikation.settings.notifications.system', icon: Bell },
]

function NotificationsSection() {
  const { t } = useTranslation()
  const muted = useKommunikationPrefs((s) => s.mutedChannels)
  const toggle = useKommunikationPrefs((s) => s.toggleMutedChannel)

  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">{t('kommunikation.settings.notifications.hint')}</p>
      {MUTE_CHANNELS.map((c) => {
        const Icon = c.icon
        const isMuted = muted.includes(c.value)
        return (
          <label key={c.value} className="flex items-center justify-between gap-4 rounded-md border border-border px-3 py-2">
            <span className="flex items-center gap-2 text-xs text-foreground">
              <Icon className="h-3.5 w-3.5 text-muted-foreground" />
              {t(c.labelKey)}
            </span>
            <button
              type="button"
              role="switch"
              aria-checked={!isMuted}
              onClick={() => toggle(c.value)}
              className={`flex h-5 w-9 shrink-0 items-center rounded-full px-0.5 transition-colors ${
                isMuted ? 'justify-start bg-border' : 'justify-end bg-primary'
              }`}
            >
              <span className="h-4 w-4 rounded-full bg-white shadow-sm" />
            </button>
          </label>
        )
      })}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Channels (connect shell) — tenant
// ---------------------------------------------------------------------------

function ChannelsSection() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">{t('kommunikation.settings.channels.hint')}</p>
      <button
        onClick={() => setOpen(true)}
        className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
      >
        <Plug className="h-3.5 w-3.5" />
        {t('kommunikation.settings.channels.manage')}
      </button>
      <ChannelSettingsDialog open={open} onOpenChange={setOpen} />
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
