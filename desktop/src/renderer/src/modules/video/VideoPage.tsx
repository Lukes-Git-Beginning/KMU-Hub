/**
 * Video & Calls module page.
 *
 * Placeholder page for the video/calls module showing active calls.
 * Will be expanded with call history, settings, and LiveKit integration.
 */
import { Video } from 'lucide-react'

export default function VideoPage() {
  return (
    <div className="flex h-full items-center justify-center">
      <div className="text-center">
        <Video className="mx-auto h-12 w-12 text-muted-foreground/50" />
        <h2 className="mt-4 text-lg font-semibold text-foreground">
          Video & Anrufe
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Starte einen Anruf ueber einen Chat-Channel oder plane ein Meeting.
        </p>
      </div>
    </div>
  )
}
