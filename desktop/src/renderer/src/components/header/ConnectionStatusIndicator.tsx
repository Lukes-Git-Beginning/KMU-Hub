/**
 * Connection status indicator for the header.
 *
 * Combines browser online/offline status with WebSocket connection state
 * to show a more accurate picture:
 *   - Green dot + pulse: connected (WS open)
 *   - Amber dot + pulse: reconnecting (WS connecting or browser offline with token)
 *   - Red dot: disconnected (offline or WS failed)
 */
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { useWSConnectionState } from '@/hooks/useWebSocket'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

type IndicatorStatus = 'connected' | 'reconnecting' | 'disconnected'

function deriveStatus(isOnline: boolean, wsState: string): IndicatorStatus {
  if (!isOnline) return 'disconnected'
  if (wsState === 'connected') return 'connected'
  if (wsState === 'connecting') return 'reconnecting'
  return 'disconnected'
}

const STATUS_CONFIG: Record<IndicatorStatus, { color: string; pulse: boolean; label: string; tooltip: string }> = {
  connected: {
    color: 'bg-success',
    pulse: false,
    label: 'Verbunden',
    tooltip: 'Verbunden mit dem Server',
  },
  reconnecting: {
    color: 'bg-warning',
    pulse: true,
    label: 'Verbindung wird hergestellt',
    tooltip: 'Verbindung wird hergestellt...',
  },
  disconnected: {
    color: 'bg-destructive',
    pulse: false,
    label: 'Nicht verbunden',
    tooltip: 'Keine Verbindung zum Server',
  },
}

export function ConnectionStatusIndicator() {
  const { isOnline } = useOnlineStatus()
  const wsState = useWSConnectionState()
  const status = deriveStatus(isOnline, wsState)
  const config = STATUS_CONFIG[status]

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="relative inline-flex h-2.5 w-2.5" role="status" aria-label={config.label}>
          {config.pulse && (
            <span
              className={`absolute inset-0 rounded-full ${config.color} opacity-75 animate-ping`}
            />
          )}
          <span className={`relative inline-block h-2.5 w-2.5 rounded-full ${config.color}`} />
        </span>
      </TooltipTrigger>
      <TooltipContent>{config.tooltip}</TooltipContent>
    </Tooltip>
  )
}
