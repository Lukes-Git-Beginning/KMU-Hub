/**
 * Automatisierung (Automation) main page.
 *
 * Three tabs:
 * - Meine Automatisierungen: list with enable/disable toggles, stats bar
 * - Vorlagen: template gallery with module/complexity grouping
 * - Protokoll: execution log viewer across all automations
 */
import { useState } from 'react'
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
import type { Automation } from '@/api/automation-types'
import { EmptyState } from '@/components/shared/EmptyState'
import { EmptyAutomation } from '@/components/shared/illustrations'
import { AutomationWizard } from './AutomationWizard'
import { TemplateGallery } from './TemplateGallery'
import { ExecutionLogViewer } from './ExecutionLogViewer'

// ---------------------------------------------------------------------------
// Stats bar
// ---------------------------------------------------------------------------

function StatsBar() {
  const { data: stats } = useAutomationStats()

  return (
    <div className="flex items-center gap-6 border-b border-border px-6 py-3">
      <div className="flex items-center gap-2">
        <Zap className="h-4 w-4 text-primary" />
        <span className="text-sm text-muted-foreground">Aktive Automatisierungen</span>
        <span className="text-sm font-semibold text-foreground">
          {stats?.active ?? 0}
        </span>
      </div>
      <div className="flex items-center gap-2">
        <CheckCircle className="h-4 w-4 text-green-500" />
        <span className="text-sm text-muted-foreground">Ausfuehrungen heute</span>
        <span className="text-sm font-semibold text-foreground">
          {stats?.total_executions ?? 0}
        </span>
      </div>
      <div className="flex items-center gap-2">
        <XCircle className="h-4 w-4 text-destructive" />
        <span className="text-sm text-muted-foreground">Erfolgsrate</span>
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
  onEdit,
}: {
  automation: Automation
  onEdit: (a: Automation) => void
}) {
  const enableMutation = useEnableAutomation()
  const disableMutation = useDisableAutomation()

  const handleToggle = (checked: boolean) => {
    if (checked) {
      enableMutation.mutate(automation.id)
    } else {
      disableMutation.mutate(automation.id)
    }
  }

  return (
    <tr
      className="border-b border-border hover:bg-secondary/50 transition-colors cursor-pointer"
      onClick={() => onEdit(automation)}
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
        <div onClick={(e) => e.stopPropagation()}>
          <Switch.Root
            checked={automation.is_active}
            onCheckedChange={handleToggle}
            className="relative inline-flex h-5 w-9 items-center rounded-full transition-colors data-[state=checked]:bg-primary data-[state=unchecked]:bg-secondary"
          >
            <Switch.Thumb className="block h-4 w-4 rounded-full bg-white transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0.5 shadow-sm" />
          </Switch.Root>
        </div>
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
          <span className="text-muted-foreground/50">Nie</span>
        )}
      </td>
      <td className="px-4 py-3 text-xs text-muted-foreground">
        {automation.scope === 'personal'
          ? 'Persoenlich'
          : automation.scope === 'team'
            ? 'Team'
            : 'Organisation'}
      </td>
    </tr>
  )
}

// ---------------------------------------------------------------------------
// Automation list
// ---------------------------------------------------------------------------

function AutomationList({
  onNew,
  onEdit,
}: {
  onNew: () => void
  onEdit: (a: Automation) => void
}) {
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
        title="Keine Automatisierungen"
        description="Erstelle deine erste Automation um wiederkehrende Aufgaben zu vereinfachen."
        action={{ label: 'Erste Automation erstellen', onClick: onNew }}
      />
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="border-b border-border">
            <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
              Name
            </th>
            <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
              Trigger
            </th>
            <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
              Status
            </th>
            <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
              Letzter Lauf
            </th>
            <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
              Bereich
            </th>
          </tr>
        </thead>
        <tbody>
          {automations.map((a) => (
            <AutomationRow key={a.id} automation={a} onEdit={onEdit} />
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
  const [wizardOpen, setWizardOpen] = useState(false)
  const { resetDraft, loadAutomationIntoDraft, setEditorMode } =
    useAutomatisierungStore()

  const handleNew = () => {
    resetDraft()
    setWizardOpen(true)
  }

  const handleEdit = (automation: Automation) => {
    loadAutomationIntoDraft(automation)
    setEditorMode('wizard')
    setWizardOpen(true)
  }

  const handleWizardClose = () => {
    setWizardOpen(false)
  }

  return (
    <div className="flex h-full flex-col overflow-hidden animate-fade-up">
      <StatsBar />

      <div className="flex-1 overflow-hidden">
        <Tabs.Root defaultValue="automations" className="flex h-full flex-col">
          <div className="flex items-center justify-between border-b border-border px-6">
            <Tabs.List className="flex gap-4">
              <Tabs.Trigger
                value="automations"
                className="flex items-center gap-1.5 border-b-2 border-transparent px-1 py-3 text-sm text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-primary data-[state=active]:text-foreground"
              >
                <Zap className="h-4 w-4" />
                Meine Automatisierungen
              </Tabs.Trigger>
              <Tabs.Trigger
                value="templates"
                className="flex items-center gap-1.5 border-b-2 border-transparent px-1 py-3 text-sm text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-primary data-[state=active]:text-foreground"
              >
                <LayoutTemplate className="h-4 w-4" />
                Vorlagen
              </Tabs.Trigger>
              <Tabs.Trigger
                value="log"
                className="flex items-center gap-1.5 border-b-2 border-transparent px-1 py-3 text-sm text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-primary data-[state=active]:text-foreground"
              >
                <ScrollText className="h-4 w-4" />
                Protokoll
              </Tabs.Trigger>
            </Tabs.List>
            <button
              onClick={handleNew}
              className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-3.5 w-3.5" />
              Neue Automatisierung
            </button>
          </div>

          <Tabs.Content value="automations" className="flex-1 overflow-y-auto">
            <AutomationList onNew={handleNew} onEdit={handleEdit} />
          </Tabs.Content>

          <Tabs.Content value="templates" className="flex-1 overflow-y-auto p-6">
            <TemplateGallery />
          </Tabs.Content>

          <Tabs.Content value="log" className="flex-1 overflow-y-auto p-6">
            <ExecutionLogViewer />
          </Tabs.Content>
        </Tabs.Root>
      </div>

      {/* Wizard dialog */}
      {wizardOpen && <AutomationWizard onClose={handleWizardClose} />}
    </div>
  )
}
