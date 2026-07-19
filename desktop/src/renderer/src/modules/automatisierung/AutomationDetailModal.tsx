/**
 * AutomationDetailModal — read-only "Cosmi-Fenster" detail view for one automation.
 *
 * Opened by clicking a row in the automations table (projektweiter Standard:
 * Zeilen-Klick → zentriertes DetailModal mit allen Infos + Aktionsleiste).
 * Editing happens via the "Bearbeiten" action which hands off to the wizard.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import * as Switch from '@radix-ui/react-switch'
import { Pencil, Copy, Trash2 } from 'lucide-react'
import { formatDistanceToNow, format } from 'date-fns'
import { de } from 'date-fns/locale'
import {
  useActionDefinitions,
  useTriggerDefinitions,
  useAutomationExecutions,
  useEnableAutomation,
  useDisableAutomation,
  useCreateAutomation,
  useDeleteAutomation,
} from '@/api/hooks/useAutomation'
import type { Automation } from '@/api/automation-types'
import { DetailModal, ConfirmDialog } from '@/components/shared'
import { useHasCapability, useScopedCapability } from '@/hooks/useCapability'

const RUN_DOT: Record<string, string> = {
  completed: 'bg-green-500',
  failed: 'bg-destructive',
  running: 'bg-blue-500',
  skipped: 'bg-muted-foreground/40',
  aborted: 'bg-muted-foreground/40',
}

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
      {children}
    </div>
  )
}

export function AutomationDetailModal({
  automation,
  open,
  onClose,
  onEdit,
}: {
  automation: Automation | null
  open: boolean
  onClose: () => void
  onEdit: (a: Automation) => void
}) {
  const { t } = useTranslation()
  const { data: actionDefs } = useActionDefinitions()
  const { data: triggerDefs } = useTriggerDefinitions()
  const enableMutation = useEnableAutomation()
  const disableMutation = useDisableAutomation()
  const createMutation = useCreateAutomation()
  const deleteMutation = useDeleteAutomation()
  const [confirmDelete, setConfirmDelete] = useState(false)
  const { data: execData } = useAutomationExecutions(automation?.id ?? '')

  // RBAC R-3 — all hooks before early return (Rules of Hooks)
  const canCreate = useHasCapability('automatisierung:automations:create')
  const canEdit = useScopedCapability('automatisierung:automations:edit', automation?.owner_id)
  const canDelete = useScopedCapability('automatisierung:automations:delete', automation?.owner_id)
  const canToggle = useScopedCapability('automatisierung:automations:toggle', automation?.owner_id)

  const handleDuplicate = (a: Automation) => {
    createMutation.mutate({
      name: `${a.name} ${t('automatisierung.detail.duplicateSuffix')}`,
      description: a.description,
      scope: a.scope,
      trigger_type: a.trigger_type,
      trigger_config: a.trigger_config,
      conditions: a.conditions,
      actions: a.actions,
      max_steps: a.max_steps,
    })
    onClose()
  }

  const handleDelete = (a: Automation) => {
    deleteMutation.mutate(a.id, { onSuccess: () => { setConfirmDelete(false); onClose() } })
  }

  const scopeLabel = (scope: string) =>
    scope === 'personal'
      ? t('automatisierung.scope.personal')
      : scope === 'team'
        ? t('automatisierung.scope.team')
        : t('automatisierung.scope.organization')

  const actionName = (type: string) =>
    actionDefs?.actions.find((a) => a.type === type)?.name ?? type

  const eventType = automation?.trigger_config?.event_type as string | undefined
  const cron = automation?.trigger_config?.cron as string | undefined
  const triggerDef =
    triggerDefs?.triggers.find((tr) => tr.event_type === eventType) ??
    triggerDefs?.triggers.find((tr) => tr.type === automation?.trigger_type)
  const runs = (execData?.executions ?? []).slice(0, 5)

  return (
    <>
    <DetailModal
      open={open}
      onClose={onClose}
      title={automation?.name}
      subtitle={automation?.description || undefined}
      badge={
        automation && (
          <span
            className={`ml-1 inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ${
              automation.is_active
                ? 'bg-green-500/15 text-green-600 dark:text-green-400'
                : 'bg-secondary text-muted-foreground'
            }`}
          >
            {automation.is_active ? t('automatisierung.detail.active') : t('automatisierung.detail.inactive')}
          </span>
        )
      }
      footer={
        automation && (
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-1.5">
              {canCreate && (
                <button
                  onClick={() => handleDuplicate(automation)}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-secondary"
                >
                  <Copy className="h-3.5 w-3.5" />
                  {t('automatisierung.detail.duplicate')}
                </button>
              )}
              {canDelete && (
                <button
                  onClick={() => setConfirmDelete(true)}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-destructive transition-colors hover:bg-destructive/10"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  {t('automatisierung.detail.delete')}
                </button>
              )}
            </div>
            {canEdit && (
              <button
                onClick={() => onEdit(automation)}
                className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
              >
                <Pencil className="h-3.5 w-3.5" />
                {t('automatisierung.detail.edit')}
              </button>
            )}
          </div>
        )
      }
    >
      {automation && (
        <div className="space-y-5">
          {/* Trigger */}
          <Section label={t('automatisierung.detail.trigger')}>
            <p className="text-sm text-foreground">{triggerDef?.name ?? automation.trigger_type}</p>
            {(eventType || cron) && (
              <p className="mt-0.5 font-mono text-xs text-muted-foreground">{eventType ?? cron}</p>
            )}
          </Section>

          {/* Conditions */}
          <Section label={t('automatisierung.detail.conditions')}>
            {automation.conditions?.mode === 'expression' && automation.conditions.expression ? (
              <p className="rounded-md bg-secondary/40 px-2.5 py-1.5 font-mono text-xs text-foreground">
                {automation.conditions.expression}
              </p>
            ) : (
              <p className="text-sm text-muted-foreground">{t('automatisierung.detail.noConditions')}</p>
            )}
          </Section>

          {/* Actions */}
          <Section label={t('automatisierung.detail.actions')}>
            <ol className="space-y-1.5">
              {automation.actions.map((a, i) => (
                <li key={i} className="flex items-center gap-2.5 rounded-md border border-border bg-card px-3 py-2">
                  <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-[11px] font-medium text-primary">
                    {i + 1}
                  </span>
                  <span className="text-sm text-foreground">{actionName(a.type)}</span>
                </li>
              ))}
            </ol>
          </Section>

          {/* Details */}
          <Section label={t('automatisierung.detail.details')}>
            <dl className="space-y-2 text-sm">
              <div className="flex items-center justify-between">
                <dt className="text-muted-foreground">{t('automatisierung.list.scope')}</dt>
                <dd className="text-foreground">{scopeLabel(automation.scope)}</dd>
              </div>
              <div className="flex items-center justify-between">
                <dt className="text-muted-foreground">{t('automatisierung.detail.created')}</dt>
                <dd className="text-foreground">{format(new Date(automation.created_at), 'dd.MM.yyyy')}</dd>
              </div>
              <div className="flex items-center justify-between">
                <dt className="text-muted-foreground">{t('automatisierung.detail.updated')}</dt>
                <dd className="text-foreground">{format(new Date(automation.updated_at), 'dd.MM.yyyy')}</dd>
              </div>
              <div className="flex items-center justify-between">
                <dt className="text-muted-foreground">{t('common.status')}</dt>
                <dd>
                  {canToggle ? (
                    <Switch.Root
                      checked={automation.is_active}
                      onCheckedChange={(c) => (c ? enableMutation.mutate(automation.id) : disableMutation.mutate(automation.id))}
                      className="relative inline-flex h-5 w-9 items-center rounded-full transition-colors data-[state=checked]:bg-primary data-[state=unchecked]:bg-secondary"
                    >
                      <Switch.Thumb className="block h-4 w-4 rounded-full bg-white shadow-sm transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0.5" />
                    </Switch.Root>
                  ) : (
                    <span
                      className={`text-sm font-medium ${automation.is_active ? 'text-green-600 dark:text-green-400' : 'text-muted-foreground'}`}
                    >
                      {automation.is_active ? t('automatisierung.detail.active') : t('automatisierung.detail.inactive')}
                    </span>
                  )}
                </dd>
              </div>
            </dl>
          </Section>

          {/* Recent runs */}
          <Section label={t('automatisierung.detail.recentRuns')}>
            {runs.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t('automatisierung.detail.noRuns')}</p>
            ) : (
              <ul className="space-y-1.5">
                {runs.map((r) => (
                  <li key={r.id} className="flex items-center gap-2.5 text-xs">
                    <span className={`h-2 w-2 shrink-0 rounded-full ${RUN_DOT[r.status] ?? 'bg-muted-foreground/40'}`} />
                    <span className="text-muted-foreground">
                      {formatDistanceToNow(new Date(r.started_at), { addSuffix: true, locale: de })}
                    </span>
                    <span className="ml-auto font-mono text-muted-foreground/70">{r.duration_ms} ms</span>
                  </li>
                ))}
              </ul>
            )}
          </Section>
        </div>
      )}
    </DetailModal>
    {automation && (
      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        variant="destructive"
        title={t('automatisierung.detail.deleteTitle')}
        description={t('automatisierung.detail.deleteBody', { name: automation.name })}
        confirmLabel={t('automatisierung.detail.delete')}
        onConfirm={() => handleDelete(automation)}
      />
    )}
    </>
  )
}
