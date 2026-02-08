/**
 * Persistent banner displayed when the app is offline.
 *
 * Shows an amber "Offline" banner while disconnected, and a brief green
 * "Wieder verbunden" (reconnected) banner when coming back online.
 * Both transitions use smooth slide animations.
 */
import { WifiOff, Wifi } from 'lucide-react'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'

export function OfflineBanner() {
  const { isOnline, wasOffline } = useOnlineStatus()

  // Show reconnected banner briefly after coming back online
  if (isOnline && wasOffline) {
    return (
      <div
        className="flex items-center justify-center gap-2 bg-emerald-600 px-4 py-2 text-sm font-medium text-white transition-all duration-300"
        role="status"
        aria-live="polite"
      >
        <Wifi className="h-4 w-4" />
        <span>Wieder verbunden</span>
      </div>
    )
  }

  // Show offline banner when disconnected
  if (!isOnline) {
    return (
      <div
        className="flex items-center justify-center gap-2 bg-amber-500 px-4 py-2 text-sm font-medium text-amber-950 transition-all duration-300"
        role="alert"
        aria-live="assertive"
      >
        <WifiOff className="h-4 w-4" />
        <span>
          Offline &mdash; Daten werden aus dem Cache angezeigt. Aenderungen sind
          nicht moeglich.
        </span>
      </div>
    )
  }

  // Online and not recently reconnected: render nothing
  return null
}
