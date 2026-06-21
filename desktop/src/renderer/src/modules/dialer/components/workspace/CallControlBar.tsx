import { useTranslation } from 'react-i18next'
import { PhoneCall, PhoneOff, StickyNote, SkipForward, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { CallPhase } from '@/stores/dialer'

interface CallControlBarProps {
  phase: CallPhase
  onDial?: () => void
  onCancel?: () => void
  onHangUp?: () => void
  onSkip?: () => void
  onToggleNotes?: () => void
  notesOpen?: boolean
  className?: string
}

export default function CallControlBar({
  phase,
  onDial,
  onCancel,
  onHangUp,
  onSkip,
  onToggleNotes,
  notesOpen,
  className,
}: CallControlBarProps) {
  const { t } = useTranslation()

  return (
    <div className={`flex items-center justify-center gap-4 ${className ?? ''}`}>
      {phase === 'dialing' && (
        <>
          <Button
            variant="ghost"
            size="lg"
            className="gap-2 text-destructive hover:text-destructive"
            onClick={onCancel}
          >
            <X className="h-5 w-5" />
            {t('common.cancel')}
          </Button>
          <Button
            size="lg"
            className="h-14 px-8 gap-3 bg-success hover:bg-success/90 text-white text-lg font-semibold animate-scale-in-bounce shadow-lg"
            onClick={onDial}
          >
            <PhoneCall className="h-6 w-6" />
            {t('dialer.workspace.dialing.call')}
          </Button>
        </>
      )}

      {phase === 'on_call' && (
        <>
          <Button
            variant="ghost"
            size="lg"
            className={`gap-2 ${notesOpen ? 'bg-accent text-accent-foreground' : ''}`}
            onClick={onToggleNotes}
          >
            <StickyNote className="h-5 w-5" />
            {t('dialer.workspace.onCall.notes')}
          </Button>
          <Button
            size="lg"
            className="h-14 px-8 gap-3 bg-destructive hover:bg-destructive/90 text-white text-lg font-semibold shadow-lg"
            onClick={onHangUp}
          >
            <PhoneOff className="h-6 w-6" />
            {t('dialer.workspace.onCall.hangUp')}
          </Button>
          <Button
            variant="ghost"
            size="lg"
            className="gap-2"
            onClick={onSkip}
          >
            <SkipForward className="h-5 w-5" />
            {t('dialer.workspace.onCall.skip')}
          </Button>
        </>
      )}
    </div>
  )
}
