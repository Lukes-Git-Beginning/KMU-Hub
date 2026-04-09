import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import CallTimer from './CallTimer'
import CallNotesEditor from './CallNotesEditor'
import type { CampaignContact, CallOutcome } from '@/api/dialer-client'

interface WrapUpPanelProps {
  contact: CampaignContact
  callDurationStart: number | null
  selectedOutcome: CallOutcome | null
  notes: string
  onNotesChange: (notes: string) => void
  callbackAt: string | null
  onCallbackAtChange: (date: string | null) => void
  onComplete: () => void
  onSkipOutcome: () => void
  isSubmitting?: boolean
}

export default function WrapUpPanel({
  contact,
  callDurationStart,
  selectedOutcome,
  notes,
  onNotesChange,
  callbackAt,
  onCallbackAtChange,
  onComplete,
  onSkipOutcome,
  isSubmitting,
}: WrapUpPanelProps) {
  const { t } = useTranslation()
  const [frozenDuration] = useState(
    () => (callDurationStart ? Math.floor((Date.now() - callDurationStart) / 1000) : 0),
  )

  return (
    <div className="space-y-5 animate-fade-up">
      {/* Contact summary */}
      <div className="space-y-1">
        <h3 className="text-lg font-bold text-foreground">{contact.contact_name}</h3>
        {contact.contact_company && (
          <p className="text-sm text-muted-foreground">{contact.contact_company}</p>
        )}
        <p className="font-mono text-sm text-muted-foreground">{contact.contact_phone}</p>
      </div>

      {/* Duration */}
      <div className="flex items-center gap-2">
        <span className="text-xs text-muted-foreground">{t('dialer.dashboard.avgDuration')}:</span>
        <CallTimer startedAt={callDurationStart} static staticSeconds={frozenDuration} className="text-base" />
      </div>

      {/* Callback date (shows when outcome is callback) */}
      {selectedOutcome?.is_callback && (
        <div className="space-y-1.5 animate-fade-up">
          <Label>{t('dialer.workspace.wrapUp.callbackDate')}</Label>
          <Input
            type="datetime-local"
            value={callbackAt ?? ''}
            onChange={(e) => onCallbackAtChange(e.target.value || null)}
          />
        </div>
      )}

      {/* Notes */}
      <CallNotesEditor value={notes} onChange={onNotesChange} />

      {/* Actions */}
      <div className="space-y-2 pt-2">
        <Button
          size="lg"
          className="w-full"
          onClick={onComplete}
          disabled={!selectedOutcome || isSubmitting}
          loading={isSubmitting}
        >
          {t('dialer.workspace.wrapUp.complete')}
        </Button>
        <button
          onClick={onSkipOutcome}
          className="w-full text-center text-xs text-muted-foreground hover:text-foreground transition-colors py-1"
        >
          {t('dialer.workspace.wrapUp.completeWithoutOutcome')}
        </button>
      </div>
    </div>
  )
}
