/**
 * Reusable integration platform card component.
 *
 * Displays platform name, description, connection status, and provides
 * configure/toggle actions. Designed for reuse in Phase 18 (Bexio) and
 * Phase 19 (Abacus/RmA) -- no Teams/Slack-specific logic inside.
 */
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Settings2 } from 'lucide-react'

export type ConnectionStatus = 'connected' | 'paused' | 'disconnected'

export interface IntegrationCardProps {
  /** Platform display name (e.g., "Microsoft Teams") */
  name: string
  /** Short description of the integration */
  description: string
  /** Platform icon/logo as a React node */
  icon: React.ReactNode
  /** Current connection status */
  status: ConnectionStatus
  /** Whether the integration is active (toggle state) */
  isActive: boolean
  /** Fired when admin clicks "Konfigurieren" */
  onConfigure: () => void
  /** Fired when admin toggles the enable/disable switch */
  onToggle: (enabled: boolean) => void
  /** Whether the toggle mutation is pending */
  isToggling?: boolean
  /** Show a "Demnachst" badge for future integrations */
  comingSoon?: boolean
}

const STATUS_CONFIG: Record<
  ConnectionStatus,
  { label: string; dotClass: string; badgeClass: string }
> = {
  connected: {
    label: 'Verbunden',
    dotClass: 'bg-green-500',
    badgeClass: 'border-green-500/30 text-green-600 dark:text-green-400',
  },
  paused: {
    label: 'Pausiert',
    dotClass: 'bg-yellow-500',
    badgeClass: 'border-yellow-500/30 text-yellow-600 dark:text-yellow-400',
  },
  disconnected: {
    label: 'Nicht verbunden',
    dotClass: 'bg-gray-400',
    badgeClass: 'border-border text-muted-foreground',
  },
}

export function IntegrationCard({
  name,
  description,
  icon,
  status,
  isActive,
  onConfigure,
  onToggle,
  isToggling = false,
  comingSoon = false,
}: IntegrationCardProps) {
  const statusCfg = STATUS_CONFIG[status]

  return (
    <div className="rounded-lg border border-border bg-card p-4 flex flex-col gap-3">
      {/* Header: icon + name + status badge */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-muted">
            {icon}
          </div>
          <div>
            <h4 className="text-sm font-medium text-foreground">{name}</h4>
            <p className="text-xs text-muted-foreground mt-0.5">
              {description}
            </p>
          </div>
        </div>
        {comingSoon ? (
          <Badge variant="secondary" className="text-xs shrink-0">
            Demnachst
          </Badge>
        ) : (
          <Badge variant="outline" className={`text-xs shrink-0 ${statusCfg.badgeClass}`}>
            <span
              className={`mr-1.5 inline-block h-1.5 w-1.5 rounded-full ${statusCfg.dotClass}`}
            />
            {statusCfg.label}
          </Badge>
        )}
      </div>

      {/* Actions row */}
      {!comingSoon && (
        <div className="flex items-center justify-between pt-1 border-t border-border-muted">
          <Button
            variant="outline"
            size="sm"
            onClick={onConfigure}
          >
            <Settings2 className="h-3.5 w-3.5 mr-1.5" />
            Konfigurieren
          </Button>

          {status !== 'disconnected' && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">
                {isActive ? 'Aktiv' : 'Inaktiv'}
              </span>
              <Switch
                checked={isActive}
                onCheckedChange={onToggle}
                disabled={isToggling}
              />
            </div>
          )}
        </div>
      )}
    </div>
  )
}
