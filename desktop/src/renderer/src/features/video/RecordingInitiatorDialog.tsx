/**
 * RecordingInitiatorDialog -- Pre-recording consent dialog for the recording initiator.
 *
 * Shown to the person who clicks "Start recording" before the recording actually begins.
 * The initiator must confirm that they understand all participants will be notified and
 * must consent. Uses Radix AlertDialog (non-dismissible) matching the pattern established
 * by RecordingConsentDialog.
 *
 * On confirm: calls onConfirm() which should invoke confirmInitiatorConsent then startRecording.
 * On cancel: calls onCancel() which should close the dialog without starting.
 */
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
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface RecordingInitiatorDialogProps {
  /** Whether the dialog is open. */
  open: boolean
  /** Called when the initiator confirms — should start the recording. */
  onConfirm: () => void
  /** Called when the initiator cancels — should close the dialog. */
  onCancel: () => void
  /** Whether a mutation is in flight (disables buttons). */
  isLoading?: boolean
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function RecordingInitiatorDialog({
  open,
  onConfirm,
  onCancel,
  isLoading = false,
}: RecordingInitiatorDialogProps) {
  const { t } = useTranslation()

  return (
    <AlertDialog open={open}>
      <AlertDialogContent
        className="sm:max-w-md"
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
                aria-hidden="true"
              >
                <circle cx="12" cy="12" r="10" />
                <circle cx="12" cy="12" r="4" fill="currentColor" />
              </svg>
            </span>
            {t('features.video.recordingInitiator.title')}
          </AlertDialogTitle>
          <AlertDialogDescription className="space-y-3 pt-2 text-left">
            <p>{t('features.video.recordingInitiator.intro')}</p>
            {/* trusted: i18n-rendered */}
            <p className="text-xs text-muted-foreground">
              {t('features.video.recordingInitiator.legalNote')}
            </p>
          </AlertDialogDescription>
        </AlertDialogHeader>

        <AlertDialogFooter className="flex-row gap-3 sm:justify-end">
          <Button
            variant="outline"
            onClick={onCancel}
            disabled={isLoading}
            className={cn('flex-1 sm:flex-none')}
          >
            {t('features.video.recordingInitiator.cancel')}
          </Button>

          <Button
            onClick={onConfirm}
            disabled={isLoading}
            className={cn(
              'flex-1 bg-red-600 text-white hover:bg-red-700 sm:flex-none',
              'focus-visible:ring-red-500',
            )}
          >
            {t('features.video.recordingInitiator.confirm')}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
