/**
 * OfflineBanner — shows queued offline mutations and allows manual drain.
 *
 * Displayed when:
 *   - Device is offline (always visible while offline)
 *   - Device is online but queue is non-empty (user may have just reconnected)
 *
 * Hides automatically once the queue is empty and the device is online.
 */
import { useEffect, useState, useCallback } from 'react'
import { peek, drain, getDeadLetter, type QueuedMutation } from '@/api/offline-queue'
import { useAuthStore } from '@/stores/auth'
import { API_BASE_URL } from '@/lib/constants'
import { cn } from '@/lib'

interface OfflineBannerState {
  isOnline: boolean
  queuedCount: number
  deadLetterCount: number
  isDraining: boolean
  lastDrainResult: { replayed: number; failed: number } | null
}

export function OfflineBanner() {
  const [state, setState] = useState<OfflineBannerState>({
    isOnline: navigator.onLine,
    queuedCount: 0,
    deadLetterCount: 0,
    isDraining: false,
    lastDrainResult: null,
  })

  const refreshCounts = useCallback(async () => {
    const [queue, deadLetter] = await Promise.all([peek(), getDeadLetter()])
    setState((prev) => ({
      ...prev,
      isOnline: navigator.onLine,
      queuedCount: queue.length,
      deadLetterCount: deadLetter.length,
    }))
  }, [])

  useEffect(() => {
    refreshCounts()

    const handleOnline = () => setState((prev) => ({ ...prev, isOnline: true }))
    const handleOffline = () => setState((prev) => ({ ...prev, isOnline: false }))

    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)

    // Poll queue count every 10s while banner is mounted.
    const interval = setInterval(refreshCounts, 10_000)

    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
      clearInterval(interval)
    }
  }, [refreshCounts])

  const handleDrain = useCallback(async () => {
    if (state.isDraining || !state.isOnline) return

    setState((prev) => ({ ...prev, isDraining: true, lastDrainResult: null }))

    const { accessToken } = useAuthStore.getState()
    const result = await drain(API_BASE_URL, () => accessToken)

    await refreshCounts()

    setState((prev) => ({
      ...prev,
      isDraining: false,
      lastDrainResult: { replayed: result.replayed, failed: result.failed },
    }))
  }, [state.isDraining, state.isOnline, refreshCounts])

  // Only show when offline OR when there are queued/dead-letter items.
  const visible = !state.isOnline || state.queuedCount > 0 || state.deadLetterCount > 0
  if (!visible) return null

  return (
    <div
      role="status"
      aria-live="polite"
      className={cn(
        'fixed bottom-4 left-1/2 z-50 -translate-x-1/2',
        'flex items-center gap-3 rounded-lg border px-4 py-3 shadow-lg',
        'text-sm font-medium',
        'transition-all duration-200',
        !state.isOnline
          ? 'border-destructive/30 bg-destructive/10 text-destructive'
          : 'border-warning/30 bg-warning/10 text-warning-foreground',
      )}
    >
      {/* Status indicator */}
      <span
        className={cn(
          'h-2 w-2 rounded-full',
          !state.isOnline ? 'bg-destructive' : 'bg-amber-400',
        )}
        aria-hidden
      />

      {/* Message */}
      <span>
        {!state.isOnline ? (
          <>
            Offline
            {state.queuedCount > 0 && (
              <> — {state.queuedCount} {state.queuedCount === 1 ? 'Aktion' : 'Aktionen'} in Warteschlange</>
            )}
          </>
        ) : (
          <>
            {state.queuedCount > 0 && (
              <>{state.queuedCount} {state.queuedCount === 1 ? 'Aktion' : 'Aktionen'} ausstehend</>
            )}
            {state.deadLetterCount > 0 && (
              <> · {state.deadLetterCount} fehlgeschlagen</>
            )}
          </>
        )}
      </span>

      {/* Drain button — only when online and queue non-empty */}
      {state.isOnline && state.queuedCount > 0 && (
        <button
          onClick={handleDrain}
          disabled={state.isDraining}
          className={cn(
            'ml-1 rounded-md border border-current/30 px-2.5 py-1 text-xs font-semibold',
            'transition-opacity hover:opacity-80 disabled:opacity-50',
          )}
          aria-label="Ausstehende Aktionen jetzt senden"
        >
          {state.isDraining ? 'Wird gesendet…' : 'Jetzt senden'}
        </button>
      )}

      {/* Result feedback */}
      {state.lastDrainResult && (
        <span className="text-xs opacity-70">
          {state.lastDrainResult.replayed > 0 && `${state.lastDrainResult.replayed} gesendet`}
          {state.lastDrainResult.failed > 0 && ` · ${state.lastDrainResult.failed} fehlgeschlagen`}
        </span>
      )}
    </div>
  )
}
