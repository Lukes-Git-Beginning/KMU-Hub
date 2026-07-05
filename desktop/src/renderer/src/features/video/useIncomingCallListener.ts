/**
 * useIncomingCallListener -- Wires the 'call.incoming' / 'call.ended'
 * WebSocket events into the video store's `incomingCall` state.
 *
 * 'call.incoming' populates `incomingCall` so IncomingCallOverlay renders the
 * fullscreen ringing UI in real time (pushed from HandleCreateCall on the
 * gateway). 'call.ended' with reason 'declined' clears the overlay again when
 * the call is declined from elsewhere (e.g. another device) before this
 * client's user acts on it.
 *
 * Registration/cleanup mirrors useNotificationWebSocket
 * (api/hooks/useNotifications.ts): raw wsManager.on(...) per event type,
 * unsubscribed on unmount via the function it returns.
 *
 * Mounted from IncomingCallOverlay, which is already rendered globally and
 * unconditionally in AppShell (same always-on lifecycle as PresenceProvider).
 */
import { useEffect } from 'react'
import { toast } from 'sonner'
import { wsManager } from '@/api/websocket'
import { useVideoStore } from '@/stores/video'
import type { CallType, IncomingCallData } from '@/api/video-types'

function isCallType(value: unknown): value is CallType {
  return value === 'one_to_one' || value === 'group'
}

/** Unwraps nested-proto-JSON payloads (e.g. `{ call: {...} }`) to a flat record. */
function unwrap(raw: Record<string, unknown>): Record<string, unknown> {
  const nested = raw.call
  return nested && typeof nested === 'object' ? (nested as Record<string, unknown>) : raw
}

export function useIncomingCallListener(): void {
  const setIncomingCall = useVideoStore((s) => s.setIncomingCall)

  useEffect(() => {
    const unsubIncoming = wsManager.on('call.incoming', (raw) => {
      const data = unwrap(raw)
      const callId = data.call_id as string | undefined
      if (!callId) return

      // lean: caller_name/avatar are not yet resolved server-side (the
      // broadcast only carries caller_id) -- fall back to the raw ID until
      // a user lookup is wired into HandleCreateCall's broadcast payload.
      const call: IncomingCallData = {
        callId,
        callerName: (data.caller_name as string | undefined) ?? (data.caller_id as string | undefined) ?? '',
        callerAvatar: data.caller_avatar as string | undefined,
        callType: isCallType(data.call_type) ? data.call_type : 'one_to_one',
      }

      setIncomingCall(call)
    })

    const unsubEnded = wsManager.on('call.ended', (raw) => {
      const data = unwrap(raw)
      const callId = data.call_id as string | undefined
      const reason = data.reason as string | undefined
      if (!callId || reason !== 'declined') return

      const current = useVideoStore.getState().incomingCall
      if (current?.callId === callId) {
        setIncomingCall(null)
        toast.info('Anruf abgelehnt')
      }
    })

    return () => {
      unsubIncoming()
      unsubEnded()
    }
  }, [setIncomingCall])
}
