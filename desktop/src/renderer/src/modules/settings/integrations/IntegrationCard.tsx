/**
 * IntegrationCard — single card in the integration grid.
 *
 * Shows icon, name, description (2-line clamp), connection status badge,
 * and chevron. Clickable to open drill-down panel.
 */
import { useTranslation } from 'react-i18next'
import { ChevronRight, Loader2 } from 'lucide-react'
import type { IntegrationDefinition } from './integration-registry'
import type { IntegrationConnectionStatus } from '@/stores/integrations'

const STATUS_CONFIG: Record<
  IntegrationConnectionStatus,
  { labelKey: string; dotClass: string; textClass: string }
> = {
  connected: { labelKey: 'settings.integrations.status.connected', dotClass: 'bg-success', textClass: 'text-success' },
  disconnected: { labelKey: 'settings.integrations.status.disconnected', dotClass: 'bg-muted-foreground/40', textClass: 'text-muted-foreground' },
  syncing: { labelKey: 'settings.integrations.status.syncing', dotClass: 'bg-info', textClass: 'text-info' },
  error: { labelKey: 'settings.integrations.status.error', dotClass: 'bg-destructive', textClass: 'text-destructive' },
}

interface IntegrationCardProps {
  definition: IntegrationDefinition
  status: IntegrationConnectionStatus
  onClick: () => void
}

export function IntegrationCard({ definition, status, onClick }: IntegrationCardProps) {
  const { t } = useTranslation()
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
            <p className="text-xs text-muted-foreground line-clamp-2 mt-0.5">{t(definition.description)}</p>
          </div>
        </div>
        <div className="mt-3 flex items-center justify-between">
          <span className="text-[10px] text-muted-foreground italic">{t('settings.integrations.comingSoon')}</span>
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
          <p className="text-xs text-muted-foreground line-clamp-2 mt-0.5">{t(definition.description)}</p>
        </div>
      </div>
      <div className="mt-3 flex items-center justify-between">
        <span className={`flex items-center gap-1.5 text-[11px] font-medium ${cfg.textClass}`}>
          {status === 'syncing' ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <span className={`h-2 w-2 rounded-full ${cfg.dotClass}`} />
          )}
          {t(cfg.labelKey)}
        </span>
        <ChevronRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
      </div>
    </button>
  )
}
