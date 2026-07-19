/**
 * Automatisierung (Automation) main page.
 *
 * Three tabs:
 * - Meine Automatisierungen: list with enable/disable toggles, stats bar
 * - Vorlagen: template gallery with module/complexity grouping
 * - Protokoll: execution log viewer across all automations
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import * as Tabs from '@radix-ui/react-tabs'
import * as Switch from '@radix-ui/react-switch'
import {
  Zap,
  Plus,
  LayoutTemplate,
  ScrollText,
  Clock,
  CheckCircle,
  XCircle,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import {
  useAutomations,
  useAutomationStats,
  useEnableAutomation,
  useDisableAutomation,
} from '@/api/hooks/useAutomation'
import { useAutomatisierungStore } from '@/stores/automatisierung'
import { useAutomatisierungPrefsStore } from '@/stores/automatisierungPrefs'
import type { AutomatisierungStartTab } from '@/stores/automatisierungPrefs'
import type { Automation } from '@/api/automation-types'
import { EmptyState, PageHeader } from '@/components/shared'
import { EmptyAutomation } from '@/components/shared/illustrations'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'
import { useCapabilitySet, useScopedCapability } from '@/hooks/useCapability'
import { AutomationWizard } from './AutomationWizard'
import { AutomationEditor } from './AutomationEditor'
import { AutomationDetailModal } from './AutomationDetailModal'
import { TemplateGallery } from './TemplateGallery'
import { ExecutionLogViewer } from './ExecutionLogViewer'

// ---------------------------------------------------------------------------
// Tab capability mapping (R-3)
// ---------------------------------------------------------------------------

type TabKey = AutomatisierungStartTab

const TAB_CAPABILITY: Record<TabKey, string> = {
  automations: 'automatisierung:automations:read',
  templates: 'automatisierung:automations:create',
  log: 'automatisierung:executions:read',
}

// ---------------------------------------------------------------------------
// Stats bar
// ---------------------------------------------------------------------------

function StatsBar() {
  const { t } = useTranslation()
  const { data: stats } = useAutomationStats()

  return (
    <div className="flex items-center gap-6 border-b border-border px-6 py-3">
      <div className="flex items-center gap-2">
        <Zap className="h-4 w-4 text-primary" />
        <span className="text-sm text-muted-foreground">{t('automatisierung.stats.active')}</span>
        <span className="text-sm font-semibold text-foreground">
          {stats?.active_automations ?? 0}
        </span>
      </div>
      <div className="flex items-center gap-2">
        <CheckCircle className="h-4 w-4 text-green-500" />
        <span className="text-sm text-muted-foreground">{t('automatisierung.stats.executionsToday')}</span>
        <span className="text-sm font-semibold text-foreground">
          {stats?.total_executions ?? 0}
        </span>
      </div>
      <div className="flex items-center gap-2">
        <XCircle className="h-4 w-4 text-destructive" />
        <span className="text-sm text-muted-foreground">{t('automatisierung.stats.successRate')}</span>
        <span className="text-sm font-semibold text-foreground">
          {stats?.success_rate !== undefined
            ? `${Math.round(stats.success_rate * 100)}%`
            : '--'}
        </span>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Automation list row
// ---------------------------------------------------------------------------

function AutomationRow({
  automation,
  onOpen,
}: {
  automation: Automation
  onOpen: (a: Automation) => void
}) {
  const { t } = useTranslation()
  const enableMutation = useEnableAutomation()
  const disableMutation = useDisableAutomation()

  // Scope-aware toggle: only own automations are toggleable when scope='own'
  const canToggle = useScopedCapability('automatisierung:automations:toggle', automation.owner_id)

  const handleToggle = (checked: boolean) => {
    if (checked) {
      enableMutation.mutate(automation.id)
    } else {
      disableMutation.mutate(automation.id)
    }
  }

  return (
    <tr
      role="button"
      tabIndex={0}
      aria-label={automation.name}
      className="border-b border-border hover:bg-secondary/50 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring focus-visible:ring-inset"
      onClick={() => onOpen(automation)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen(automation)
        }
      }}
    >
      <td className="px-4 py-3">
        <div>
          <div className="text-sm font-medium text-foreground">{automation.name}</div>
          {automation.description && (
            <div className="text-xs text-muted-foreground mt-0.5">
              {automation.description}
            </div>
          )}
        </div>
      </td>
      <td className="px-4 py-3">
        <span className="inline-flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-xs text-foreground">
          <Zap className="h-3 w-3" />
          {automation.trigger_type}
        </span>
      </td>
      <td className="px-4 py-3">
        {canToggle && (
          <div onClick={(e) => e.stopPropagation()}>
            <Switch.Root
              checked={automation.is_active}
              onCheckedChange={handleToggle}
              className="relative inline-flex h-5 w-9 items-center rounded-full transition-colors data-[state=checked]:bg-primary data-[state=unchecked]:bg-secondary"
            >
              <Switch.Thumb className="block h-4 w-4 rounded-full bg-white transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0.5 shadow-sm" />
            </Switch.Root>
          </div>
        )}
        {!canToggle && (
          <span
            className={`inline-block h-2 w-2 rounded-full ${automation.is_active ? 'bg-green-500' : 'bg-secondary'}`}
          />
        )}
      </td>
      <td className="px-4 py-3 text-xs text-muted-foreground">
        {automation.last_triggered_at ? (
          <span className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            {formatDistanceToNow(new Date(automation.last_triggered_at), {
              addSuffix: true,
              locale: de,
            })}
          </span>
        ) : (
          <span className="text-muted-foreground/50">{t('automatisierung.list.never')}</span>
        )}
      </td>
      <td className="px-4 py-3 text-xs text-muted-foreground">
        {automation.scope === 'personal'
          ? t('automatisierung.scope.personal')
          : automation.scope === 'team'
            ? t('automatisierung.scope.team')
            : t('automatisierung.scope.organization')}
      </td>
    </tr>
  )
}

// ---------------------------------------------------------------------------
// Automation list
// ---------------------------------------------------------------------------

function AutomationList({
  onNew,
  onOpen,
  canCreate,
}: {
  onNew: () => void
  onOpen: (a: Automation) => void
  canCreate: boolean
}) {
  const { t } = useTranslation()
  const { data, isLoading } = useAutomations()
  const automations = data?.automations ?? []

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    )
  }

  if (automations.length === 0) {
    return (
      <EmptyState
        illustration={<EmptyAutomation />}
        title={t('automatisierung.empty.title')}
        description={t('automatisierung.empty.description')}
        action={canCreate ? { label: t('automatisierung.empty.action'), onClick: onNew } : undefined}
      />
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="border-b border-border">
            <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
              {t('automatisierung.list.name')}
            </th>
            <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
              {t('automatisierung.list.trigger')}
            </th>
            <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
              {t('common.status')}
            </th>
            <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
              {t('automatisierung.list.lastRun')}
            </th>
            <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
              {t('automatisierung.list.scope')}
            </th>
          </tr>
        </thead>
        <tbody>
          {automations.map((a) => (
            <AutomationRow key={a.id} automation={a} onOpen={onOpen} />
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export default function AutomatisierungPage() {
  const { t } = useTranslation()
  const [wizardOpen, setWizardOpen] = useState(false)
  const [detailAutomation, setDetailAutomation] = useState<Automation | null>(null)
  const { resetDraft, loadAutomationIntoDraft, setEditorMode, editorMode } =
    useAutomatisierungStore()
  const startTab = useAutomatisierungPrefsStore((s) => s.startTab)

  // RBAC R-3 capability checks — all hooks BEFORE any early return
  const { has: capHas, ready: capReady } = useCapabilitySet()
  const canCreate = capHas('automatisierung:automations:create')

  const visibleTabs = (Object.keys(TAB_CAPABILITY) as TabKey[]).filter(
    (key) => !capReady || capHas(TAB_CAPABILITY[key]),
  )
  // Stable string for effect dependency (avoids new array ref each render)
  const visibleTabsKey = visibleTabs.join(',')

  // Safe initial tab: resolve at render time against visible set
  // (startTab from prefs might not be visible for this role)
  const safeStartTab = visibleTabs.includes(startTab) ? startTab : (visibleTabs[0] ?? 'automations')
  const [activeTab, setActiveTab] = useState<TabKey>(safeStartTab)

  // Fallback effect: once caps load, redirect away from an invisible tab
  useEffect(() => {
    if (!capReady) return
    const tabs = visibleTabsKey ? (visibleTabsKey.split(',') as TabKey[]) : []
    if (tabs.length > 0 && !tabs.includes(activeTab)) {
      setActiveTab(tabs[0])
    }
  }, [capReady, activeTab, visibleTabsKey])

  const handleNew = () => {
    resetDraft()
    setWizardOpen(true)
  }

  const handleEdit = (automation: Automation) => {
    loadAutomationIntoDraft(automation)
    setEditorMode('wizard')
    setWizardOpen(true)
  }

  // Row click opens the read-only detail; editing is reached from inside it.
  const handleOpenDetail = (automation: Automation) => setDetailAutomation(automation)

  const handleEditFromDetail = (automation: Automation) => {
    setDetailAutomation(null)
    handleEdit(automation)
  }

  const handleWizardClose = () => {
    setWizardOpen(false)
  }

  // moduleEmpty: caps loaded but no tab is visible at all
  if (capReady && visibleTabs.length === 0) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <PageHeader
          title={t('layout.navItems.automatisierung')}
          description={t('rbac.gate.moduleEmpty')}
          icon={Zap}
          moduleId="automatisierung"
          className="mb-6"
          actions={<RestrictedModeBadge module="automatisierung" />}
        />
        <EmptyState
          illustration={<EmptyAutomation />}
          title={t('rbac.gate.moduleEmpty')}
          description={t('rbac.gate.noPermission')}
        />
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col overflow-hidden animate-fade-up">
      <StatsBar />

      <div className="flex-1 overflow-hidden">
        <Tabs.Root
          value={activeTab}
          onValueChange={(v) => setActiveTab(v as TabKey)}
          className="flex h-full flex-col"
        >
          <div className="flex items-center justify-between border-b border-border px-6">
            <Tabs.List className="flex gap-4">
              {visibleTabs.includes('automations') && (
                <Tabs.Trigger
                  value="automations"
                  className="flex items-center gap-1.5 border-b-2 border-transparent px-1 py-3 text-sm text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-primary data-[state=active]:text-foreground"
                >
                  <Zap className="h-4 w-4" />
                  {t('automatisierung.tabs.myAutomations')}
                </Tabs.Trigger>
              )}
              {visibleTabs.includes('templates') && (
                <Tabs.Trigger
                  value="templates"
                  className="flex items-center gap-1.5 border-b-2 border-transparent px-1 py-3 text-sm text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-primary data-[state=active]:text-foreground"
                >
                  <LayoutTemplate className="h-4 w-4" />
                  {t('automatisierung.tabs.templates')}
                </Tabs.Trigger>
              )}
              {visibleTabs.includes('log') && (
                <Tabs.Trigger
                  value="log"
                  className="flex items-center gap-1.5 border-b-2 border-transparent px-1 py-3 text-sm text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-primary data-[state=active]:text-foreground"
                >
                  <ScrollText className="h-4 w-4" />
                  {t('automatisierung.tabs.log')}
                </Tabs.Trigger>
              )}
            </Tabs.List>

            <div className="flex items-center gap-2">
              <RestrictedModeBadge module="automatisierung" />
              {canCreate && (
                <button
                  onClick={handleNew}
                  className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
                >
                  <Plus className="h-3.5 w-3.5" />
                  {t('automatisierung.newAutomation')}
                </button>
              )}
            </div>
          </div>

          {visibleTabs.includes('automations') && (
            <Tabs.Content value="automations" className="flex-1 overflow-y-auto">
              <AutomationList onNew={handleNew} onOpen={handleOpenDetail} canCreate={canCreate} />
            </Tabs.Content>
          )}

          {visibleTabs.includes('templates') && (
            <Tabs.Content value="templates" className="flex-1 overflow-y-auto p-6">
              <TemplateGallery />
            </Tabs.Content>
          )}

          {visibleTabs.includes('log') && (
            <Tabs.Content value="log" className="flex-1 overflow-y-auto p-6">
              <ExecutionLogViewer />
            </Tabs.Content>
          )}
        </Tabs.Root>
      </div>

      {/* Detail modal (row click) */}
      <AutomationDetailModal
        automation={detailAutomation}
        open={!!detailAutomation}
        onClose={() => setDetailAutomation(null)}
        onEdit={handleEditFromDetail}
      />

      {/* Wizard / visual editor dialog (editorMode toggles between them) */}
      {wizardOpen &&
        (editorMode === 'editor' ? (
          <AutomationEditor onClose={handleWizardClose} />
        ) : (
          <AutomationWizard onClose={handleWizardClose} />
        ))}
    </div>
  )
}
