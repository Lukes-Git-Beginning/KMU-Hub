/**
 * Execution log viewer for automations.
 *
 * Table with expandable row details showing trigger event, condition result,
 * and step-by-step action results. Supports status filtering and pagination.
 */
import { useState, useMemo } from 'react'
import {
  ChevronDown,
  ChevronRight,
  CheckCircle,
  XCircle,
  Clock,
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
  { icon: React.ElementType; label: string; className: string }
> = {
  completed: {
    icon: CheckCircle,
    label: 'Abgeschlossen',
    className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  },
  failed: {
    icon: XCircle,
    label: 'Fehlgeschlagen',
    className: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  },
  running: {
    icon: Loader2,
    label: 'Laeuft',
    className: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
  },
  skipped: {
    icon: MinusCircle,
    label: 'Uebersprungen',
    className: 'bg-gray-100 text-gray-600 dark:bg-gray-900/30 dark:text-gray-400',
  },
  aborted: {
    icon: AlertTriangle,
    label: 'Abgebrochen',
    className: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
  },
}

function StatusBadge({ status }: { status: ExecutionStatus }) {
  const config = STATUS_CONFIG[status] ?? STATUS_CONFIG.failed
  const Icon = config.icon

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ${config.className}`}
    >
      <Icon className={`h-3 w-3 ${status === 'running' ? 'animate-spin' : ''}`} />
      {config.label}
    </span>
  )
}

// ---------------------------------------------------------------------------
// Expandable row detail
// ---------------------------------------------------------------------------

function ExecutionDetail({ execution }: { execution: AutomationExecution }) {
  return (
    <div className="bg-secondary/30 px-6 py-4 space-y-4">
      {/* Trigger event */}
      <div>
        <h4 className="text-xs font-medium text-muted-foreground mb-1">
          Trigger-Ereignis
        </h4>
        <pre className="rounded-md bg-background border border-border p-2 text-xs text-foreground overflow-x-auto font-mono max-h-32 overflow-y-auto">
          {JSON.stringify(execution.trigger_event, null, 2)}
        </pre>
      </div>

      {/* Condition result */}
      <div>
        <h4 className="text-xs font-medium text-muted-foreground mb-1">
          Bedingung
        </h4>
        <span
          className={`text-sm font-medium ${
            execution.condition_result ? 'text-green-600' : 'text-red-600'
          }`}
        >
          {execution.condition_result ? 'Ja (Bedingung erfuellt)' : 'Nein (Bedingung nicht erfuellt)'}
        </span>
      </div>

      {/* Steps */}
      {execution.steps.length > 0 && (
        <div>
          <h4 className="text-xs font-medium text-muted-foreground mb-2">
            Aktionsschritte ({execution.steps.length})
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
          <h4 className="text-xs font-medium text-red-600 mb-1">Fehler</h4>
          <p className="text-sm text-red-600">{execution.error_message}</p>
        </div>
      )}
    </div>
  )
}

function StepDetail({ step, index }: { step: ExecutionStep; index: number }) {
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
          {hasError && <XCircle className="h-3 w-3 text-red-500" />}
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
            <span className="text-[10px] font-medium text-muted-foreground">Eingabe</span>
            <pre className="text-[11px] font-mono text-foreground mt-0.5 max-h-20 overflow-auto">
              {JSON.stringify(step.input, null, 2)}
            </pre>
          </div>
          <div>
            <span className="text-[10px] font-medium text-muted-foreground">Ausgabe</span>
            <pre className="text-[11px] font-mono text-foreground mt-0.5 max-h-20 overflow-auto">
              {JSON.stringify(step.output, null, 2)}
            </pre>
          </div>
          {step.error && (
            <div>
              <span className="text-[10px] font-medium text-red-600">Fehler</span>
              <p className="text-[11px] text-red-600 mt-0.5">{step.error}</p>
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

const STATUS_OPTIONS: { value: string; label: string }[] = [
  { value: '', label: 'Alle Status' },
  { value: 'completed', label: 'Abgeschlossen' },
  { value: 'failed', label: 'Fehlgeschlagen' },
  { value: 'running', label: 'Laeuft' },
  { value: 'skipped', label: 'Uebersprungen' },
  { value: 'aborted', label: 'Abgebrochen' },
]

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function ExecutionLogViewer({
  automationId,
}: {
  automationId?: string
}) {
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
              {opt.label}
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
          Keine Ausfuehrungen gefunden
        </div>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-border">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-secondary/30">
                <th className="w-8 px-2 py-2" />
                <th className="px-3 py-2 text-left text-[11px] font-medium text-muted-foreground">
                  Zeitpunkt
                </th>
                <th className="px-3 py-2 text-left text-[11px] font-medium text-muted-foreground">
                  Status
                </th>
                <th className="px-3 py-2 text-left text-[11px] font-medium text-muted-foreground">
                  Bedingung
                </th>
                <th className="px-3 py-2 text-left text-[11px] font-medium text-muted-foreground">
                  Aktionen
                </th>
                <th className="px-3 py-2 text-left text-[11px] font-medium text-muted-foreground">
                  Dauer
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
            Mehr laden
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
          {execution.condition_result ? 'Ja' : 'Nein'}
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
