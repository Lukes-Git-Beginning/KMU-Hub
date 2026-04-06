/**
 * Execution log viewer for automations.
 *
 * Table with expandable row details showing trigger event, condition result,
 * and step-by-step action results. Supports status filtering and pagination.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  ChevronDown,
  ChevronRight,
  CheckCircle,
  XCircle,
  AlertTriangle,
  MinusCircle,
  Loader2,
} from 'lucide-react'
import { format } from 'date-fns'
import { de } from 'date-fns/locale'
import { useAutomationExecutions } from '@/api/hooks/useAutomation'
import { useAutomatisierungStore } from '@/stores/automatisierung'
import type {
  AutomationExecution,
  ExecutionStatus,
  ExecutionStep,
} from '@/api/automation-types'

// ---------------------------------------------------------------------------
// Status badges
// ---------------------------------------------------------------------------

const STATUS_CONFIG: Record<
  ExecutionStatus,
  { icon: React.ElementType; labelKey: string; className: string }
> = {
  completed: {
    icon: CheckCircle,
    labelKey: 'automatisierung.execution.status.completed',
    className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  },
  failed: {
    icon: XCircle,
    labelKey: 'automatisierung.execution.status.failed',
    className: 'bg-error-light text-destructive',
  },
  running: {
    icon: Loader2,
    labelKey: 'automatisierung.execution.status.running',
    className: 'bg-warning-light text-warning-foreground',
  },
  skipped: {
    icon: MinusCircle,
    labelKey: 'automatisierung.execution.status.skipped',
    className: 'bg-gray-100 text-gray-600 dark:bg-gray-900/30 dark:text-gray-400',
  },
  aborted: {
    icon: AlertTriangle,
    labelKey: 'automatisierung.execution.status.aborted',
    className: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
  },
}

function StatusBadge({ status }: { status: ExecutionStatus }) {
  const { t } = useTranslation()
  const config = STATUS_CONFIG[status] ?? STATUS_CONFIG.failed
  const Icon = config.icon

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ${config.className}`}
    >
      <Icon className={`h-3 w-3 ${status === 'running' ? 'animate-spin' : ''}`} />
      {t(config.labelKey)}
    </span>
  )
}

// ---------------------------------------------------------------------------
// Expandable row detail
// ---------------------------------------------------------------------------

function ExecutionDetail({ execution }: { execution: AutomationExecution }) {
  const { t } = useTranslation()
  return (
    <div className="bg-secondary/30 px-6 py-4 space-y-4">
      {/* Trigger event */}
      <div>
        <h4 className="text-xs font-medium text-muted-foreground mb-1">
          {t('automatisierung.execution.triggerEvent')}
        </h4>
        <pre className="rounded-md bg-background border border-border p-2 text-xs text-foreground overflow-x-auto font-mono max-h-32 overflow-y-auto">
          {JSON.stringify(execution.trigger_event, null, 2)}
        </pre>
      </div>

      {/* Condition result */}
      <div>
        <h4 className="text-xs font-medium text-muted-foreground mb-1">
          {t('automatisierung.execution.condition')}
        </h4>
        <span
          className={`text-sm font-medium ${
            execution.condition_result ? 'text-green-600' : 'text-destructive'
          }`}
        >
          {execution.condition_result ? t('automatisierung.execution.conditionMet') : t('automatisierung.execution.conditionNotMet')}
        </span>
      </div>

      {/* Steps */}
      {execution.steps.length > 0 && (
        <div>
          <h4 className="text-xs font-medium text-muted-foreground mb-2">
            {t('automatisierung.execution.actionSteps')} ({execution.steps.length})
          </h4>
          <div className="space-y-2">
            {execution.steps.map((step, idx) => (
              <StepDetail key={idx} step={step} index={idx} />
            ))}
          </div>
        </div>
      )}

      {/* Error */}
      {execution.error_message && (
        <div>
          <h4 className="text-xs font-medium text-destructive mb-1">{t('automatisierung.execution.error')}</h4>
          <p className="text-sm text-destructive">{execution.error_message}</p>
        </div>
      )}
    </div>
  )
}

function StepDetail({ step, index }: { step: ExecutionStep; index: number }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const hasError = !!step.error

  return (
    <div className="rounded-md border border-border bg-background">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center justify-between w-full px-3 py-2 text-left"
      >
        <div className="flex items-center gap-2">
          <span className="flex items-center justify-center h-5 w-5 rounded-full bg-secondary text-[10px] font-bold text-foreground">
            {index + 1}
          </span>
          <span className="text-xs font-medium text-foreground">{step.action_type}</span>
          {hasError && <XCircle className="h-3 w-3 text-destructive" />}
          <span className="text-[10px] text-muted-foreground">{step.duration_ms}ms</span>
        </div>
        {expanded ? (
          <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
        )}
      </button>
      {expanded && (
        <div className="border-t border-border px-3 py-2 space-y-2">
          <div>
            <span className="text-[10px] font-medium text-muted-foreground">{t('automatisierung.execution.input')}</span>
            <pre className="text-[11px] font-mono text-foreground mt-0.5 max-h-20 overflow-auto">
              {JSON.stringify(step.input, null, 2)}
            </pre>
          </div>
          <div>
            <span className="text-[10px] font-medium text-muted-foreground">{t('automatisierung.execution.output')}</span>
            <pre className="text-[11px] font-mono text-foreground mt-0.5 max-h-20 overflow-auto">
              {JSON.stringify(step.output, null, 2)}
            </pre>
          </div>
          {step.error && (
            <div>
              <span className="text-[10px] font-medium text-destructive">{t('automatisierung.execution.error')}</span>
              <p className="text-[11px] text-destructive mt-0.5">{step.error}</p>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Filter controls
// ---------------------------------------------------------------------------

const STATUS_OPTIONS: { value: string; labelKey: string }[] = [
  { value: '', labelKey: 'automatisierung.execution.filter.allStatus' },
  { value: 'completed', labelKey: 'automatisierung.execution.status.completed' },
  { value: 'failed', labelKey: 'automatisierung.execution.status.failed' },
  { value: 'running', labelKey: 'automatisierung.execution.status.running' },
  { value: 'skipped', labelKey: 'automatisierung.execution.status.skipped' },
  { value: 'aborted', labelKey: 'automatisierung.execution.status.aborted' },
]

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function ExecutionLogViewer({
  automationId,
}: {
  automationId?: string
}) {
  const { t } = useTranslation()
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const { executionLogFilter, setExecutionLogFilter } = useAutomatisierungStore()
  const [limit, setLimit] = useState(20)

  // Use a placeholder ID when showing all executions (page-level view)
  const queryId = automationId ?? ''
  const { data, isLoading } = useAutomationExecutions(queryId, {
    status: (executionLogFilter.status as ExecutionStatus) || undefined,
    limit,
  })

  const executions = data?.executions ?? []

  return (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex items-center gap-3">
        <select
          value={executionLogFilter.status ?? ''}
          onChange={(e) =>
            setExecutionLogFilter({
              ...executionLogFilter,
              status: e.target.value || undefined,
            })
          }
          className="rounded-md border border-border bg-background px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
        >
          {STATUS_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {t(opt.labelKey)}
            </option>
          ))}
        </select>
      </div>

      {/* Table */}
      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary" />
        </div>
      ) : executions.length === 0 ? (
        <div className="py-8 text-center text-sm text-muted-foreground">
          {t('automatisierung.execution.noExecutions')}
        </div>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-border">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-secondary/30">
                <th className="w-8 px-2 py-2" />
                <th className="px-3 py-2 text-left text-[11px] font-medium text-muted-foreground">
                  {t('automatisierung.execution.timestamp')}
                </th>
                <th className="px-3 py-2 text-left text-[11px] font-medium text-muted-foreground">
                  {t('common.status')}
                </th>
                <th className="px-3 py-2 text-left text-[11px] font-medium text-muted-foreground">
                  {t('automatisierung.execution.condition')}
                </th>
                <th className="px-3 py-2 text-left text-[11px] font-medium text-muted-foreground">
                  {t('automatisierung.execution.actions')}
                </th>
                <th className="px-3 py-2 text-left text-[11px] font-medium text-muted-foreground">
                  {t('automatisierung.execution.duration')}
                </th>
              </tr>
            </thead>
            <tbody>
              {executions.map((exec) => {
                const isExpanded = expandedId === exec.id
                return (
                  <ExecutionRow
                    key={exec.id}
                    execution={exec}
                    isExpanded={isExpanded}
                    onToggle={() =>
                      setExpandedId(isExpanded ? null : exec.id)
                    }
                  />
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Load more */}
      {executions.length >= limit && (
        <div className="flex justify-center">
          <button
            onClick={() => setLimit((l) => l + 20)}
            className="rounded-md border border-border px-4 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            {t('automatisierung.execution.loadMore')}
          </button>
        </div>
      )}
    </div>
  )
}

function ExecutionRow({
  execution,
  isExpanded,
  onToggle,
}: {
  execution: AutomationExecution
  isExpanded: boolean
  onToggle: () => void
}) {
  const { t } = useTranslation()
  return (
    <>
      <tr
        className="border-b border-border hover:bg-secondary/30 transition-colors cursor-pointer"
        onClick={onToggle}
      >
        <td className="px-2 py-2">
          {isExpanded ? (
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
          )}
        </td>
        <td className="px-3 py-2 text-xs text-foreground">
          {execution.started_at
            ? format(new Date(execution.started_at), 'dd.MM.yyyy HH:mm:ss', {
                locale: de,
              })
            : '--'}
        </td>
        <td className="px-3 py-2">
          <StatusBadge status={execution.status} />
        </td>
        <td className="px-3 py-2 text-xs text-foreground">
          {execution.condition_result ? t('common.yes') : t('common.no')}
        </td>
        <td className="px-3 py-2 text-xs text-foreground">
          {execution.steps.length}
        </td>
        <td className="px-3 py-2 text-xs text-muted-foreground">
          {execution.duration_ms}ms
        </td>
      </tr>
      {isExpanded && (
        <tr>
          <td colSpan={6} className="p-0">
            <ExecutionDetail execution={execution} />
          </td>
        </tr>
      )}
    </>
  )
}
