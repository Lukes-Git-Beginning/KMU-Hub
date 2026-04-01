/**
 * IntegrationCard — single card in the integration grid.
 *
 * Shows icon, name, description (2-line clamp), connection status badge,
 * and chevron. Clickable to open drill-down panel.
 */
import { ChevronRight, Loader2 } from 'lucide-react'
import type { IntegrationDefinition } from './integration-registry'
import type { IntegrationConnectionStatus } from '@/stores/integrations'

const STATUS_CONFIG: Record<
  IntegrationConnectionStatus,
  { label: string; dotClass: string; textClass: string }
> = {
  connected: { label: 'Verbunden', dotClass: 'bg-green-500', textClass: 'text-green-700 dark:text-green-400' },
  disconnected: { label: 'Nicht verbunden', dotClass: 'bg-muted-foreground/40', textClass: 'text-muted-foreground' },
  syncing: { label: 'Synchronisiert...', dotClass: 'bg-blue-500', textClass: 'text-blue-600 dark:text-blue-400' },
  error: { label: 'Fehler', dotClass: 'bg-red-500', textClass: 'text-red-600 dark:text-red-400' },
}

interface IntegrationCardProps {
  definition: IntegrationDefinition
  status: IntegrationConnectionStatus
  onClick: () => void
}

export function IntegrationCard({ definition, status, onClick }: IntegrationCardProps) {
  const Icon = definition.icon
  const cfg = STATUS_CONFIG[status]

  if (definition.comingSoon) {
    return (
      <div className="rounded-lg border border-border bg-card/50 p-4 opacity-50 cursor-not-allowed">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-secondary">
            <Icon className={`h-5 w-5 ${definition.iconColor}`} />
          </div>
          <div className="min-w-0 flex-1">
            <h3 className="text-sm font-medium text-foreground">{definition.name}</h3>
            <p className="text-xs text-muted-foreground line-clamp-2 mt-0.5">{definition.description}</p>
          </div>
        </div>
        <div className="mt-3 flex items-center justify-between">
          <span className="text-[10px] text-muted-foreground italic">Demnächst verfügbar</span>
        </div>
      </div>
    )
  }

  return (
    <button
      onClick={onClick}
      className="w-full rounded-lg border border-border bg-card p-4 text-left hover:border-primary/40 hover:bg-secondary/30 transition-all group"
    >
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-secondary">
          <Icon className={`h-5 w-5 ${definition.iconColor}`} />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-medium text-foreground">{definition.name}</h3>
          <p className="text-xs text-muted-foreground line-clamp-2 mt-0.5">{definition.description}</p>
        </div>
      </div>
      <div className="mt-3 flex items-center justify-between">
        <span className={`flex items-center gap-1.5 text-[11px] font-medium ${cfg.textClass}`}>
          {status === 'syncing' ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <span className={`h-2 w-2 rounded-full ${cfg.dotClass}`} />
          )}
          {cfg.label}
        </span>
        <ChevronRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
      </div>
    </button>
  )
}
