/**
 * Connection status indicator for the header.
 *
 * Combines browser online/offline status with WebSocket connection state
 * to show a more accurate picture:
 *   - Green dot + pulse: connected (WS open)
 *   - Amber dot + pulse: reconnecting (WS connecting or browser offline with token)
 *   - Red dot: disconnected (offline or WS failed)
 */
import { useTranslation } from 'react-i18next'
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

const STATUS_COLORS: Record<IndicatorStatus, { color: string; pulse: boolean }> = {
  connected: { color: 'bg-success', pulse: false },
  reconnecting: { color: 'bg-warning', pulse: true },
  disconnected: { color: 'bg-destructive', pulse: false },
}

export function ConnectionStatusIndicator() {
  const { t } = useTranslation()
  const { isOnline } = useOnlineStatus()
  const wsState = useWSConnectionState()
  const status = deriveStatus(isOnline, wsState)
  const { color, pulse } = STATUS_COLORS[status]

  const label = t(`header.connectionStatus.${status}.label`)
  const tooltip = t(`header.connectionStatus.${status}.tooltip`)
  const config = { color, pulse, label, tooltip }

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
