/**
 * PreJoinScreen -- Camera/microphone preview before joining a call.
 *
 * Uses the @livekit/components-react PreJoin prefab component for
 * device enumeration, camera preview, and mic level visualization.
 * Styled with Tailwind to match the KMU Hub design system.
 */
import { useCallback } from 'react'
import { PreJoin, type LocalUserChoices } from '@livekit/components-react'
import '@livekit/components-styles'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface PreJoinScreenProps {
  /** Called when the user clicks "Beitreten" with their device preferences. */
  onJoin: (audioEnabled: boolean, videoEnabled: boolean) => void
  /** Called when the user cancels joining. */
  onCancel?: () => void
  /** Optional CSS class for the container. */
  className?: string
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function PreJoinScreen({
  onJoin,
  onCancel,
  className,
}: PreJoinScreenProps) {
  const handleSubmit = useCallback(
    (choices: LocalUserChoices) => {
      onJoin(choices.audioEnabled, choices.videoEnabled)
    },
    [onJoin],
  )

  return (
    <div
      className={cn(
        'flex h-full w-full flex-col items-center justify-center gap-6 bg-zinc-950 p-8',
        className,
      )}
    >
      <h2 className="text-xl font-semibold text-white">
        Anruf beitreten
      </h2>
      <p className="text-sm text-zinc-400">
        Ueberpruefen Sie Ihre Kamera und Ihr Mikrofon bevor Sie beitreten.
      </p>

      <div className="w-full max-w-lg rounded-xl bg-zinc-900 p-6">
        <PreJoin
          onSubmit={handleSubmit}
          onError={(err) => {
            // eslint-disable-next-line no-console
            console.error('PreJoin device error:', err)
          }}
          defaults={{
            audioEnabled: true,
            videoEnabled: true,
          }}
          joinLabel="Beitreten"
          data-lk-theme="default"
        />
      </div>

      {onCancel && (
        <Button
          variant="ghost"
          className="text-zinc-400 hover:text-white"
          onClick={onCancel}
        >
          Abbrechen
        </Button>
      )}
    </div>
  )
}
