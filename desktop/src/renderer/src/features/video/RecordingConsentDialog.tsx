/**
 * RecordingConsentDialog -- DSGVO-compliant recording consent popup.
 *
 * Shows when a recording is initiated and asks the participant to
 * consent. Uses Radix AlertDialog which cannot be dismissed without
 * responding (no close button, no click-outside dismiss).
 *
 * Accept: participant's video and audio are included in the recording.
 * Decline: participant is blurred (video) and muted (audio) in the
 * recording -- this is a valid choice, not an error.
 *
 * The consent response is sent to the backend via useSetRecordingConsent
 * which stores it and notifies the recording initiator.
 */
import { useCallback, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { useSetRecordingConsent } from '@/api/hooks/useVideo'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface RecordingConsentDialogProps {
  /** Recording ID for the consent response. */
  recordingId: string
  /** Whether the dialog is open. */
  open: boolean
  /** Called after the user responds (consented or declined). */
  onRespond: (consented: boolean) => void
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function RecordingConsentDialog({
  recordingId,
  open,
  onRespond,
}: RecordingConsentDialogProps) {
  const { t } = useTranslation()
  const setConsent = useSetRecordingConsent()
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleResponse = useCallback(
    async (consented: boolean) => {
      setIsSubmitting(true)
      try {
        await setConsent.mutateAsync({ recordingId, consented })
        onRespond(consented)
      } catch {
        // Retry is possible -- keep dialog open
        setIsSubmitting(false)
      }
    },
    [recordingId, setConsent, onRespond],
  )

  return (
    <AlertDialog open={open}>
      <AlertDialogContent
        className="sm:max-w-md"
        // Prevent dismiss via Escape or click-outside
        onEscapeKeyDown={(e) => e.preventDefault()}
        onPointerDownOutside={(e) => e.preventDefault()}
      >
        <AlertDialogHeader>
          <AlertDialogTitle className="flex items-center gap-2">
            {/* Recording icon */}
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-4 w-4 text-red-600 dark:text-red-400"
              >
                <circle cx="12" cy="12" r="10" />
                <circle cx="12" cy="12" r="4" fill="currentColor" />
              </svg>
            </span>
            {t('features.video.recordingConsent.title')}
          </AlertDialogTitle>
          <AlertDialogDescription className="space-y-3 pt-2 text-left">
            <p>
              {t('features.video.recordingConsent.intro')}
            </p>
            <p
              dangerouslySetInnerHTML={{
                __html: t('features.video.recordingConsent.acceptInfo'),
              }}
            />
            <p
              dangerouslySetInnerHTML={{
                __html: t('features.video.recordingConsent.declineInfo'),
              }}
            />
            <p className="text-xs text-muted-foreground">
              {t('features.video.recordingConsent.legalNote')}
            </p>
          </AlertDialogDescription>
        </AlertDialogHeader>

        <AlertDialogFooter className="flex-row gap-3 sm:justify-end">
          {/* Decline -- gray, valid choice */}
          <Button
            variant="outline"
            onClick={() => handleResponse(false)}
            disabled={isSubmitting}
            className={cn(
              'flex-1 sm:flex-none',
            )}
          >
            {t('features.video.recordingConsent.decline')}
          </Button>

          {/* Accept -- green accent */}
          <Button
            onClick={() => handleResponse(true)}
            disabled={isSubmitting}
            className={cn(
              'flex-1 bg-green-600 text-white hover:bg-green-700 sm:flex-none',
              'focus-visible:ring-green-500',
            )}
          >
            {t('features.video.recordingConsent.accept')}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
