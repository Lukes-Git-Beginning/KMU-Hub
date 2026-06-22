/**
 * Dialogs for the real slash commands (KO-8): /umfrage (poll) and /erinnerung
 * (reminder). On submit they hand a structured payload back so the conversation
 * thread can append it as an interactive block.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import type { ConversationPoll, ConversationReminder } from '@/types/communication'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Poll
// ---------------------------------------------------------------------------

interface PollDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreate: (poll: ConversationPoll) => void
}

export function PollDialog({ open, onOpenChange, onCreate }: PollDialogProps) {
  const { t } = useTranslation()
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState<string[]>(['', ''])

  const reset = () => {
    setQuestion('')
    setOptions(['', ''])
  }

  const filledOptions = options.map((o) => o.trim()).filter(Boolean)
  const canSubmit = question.trim().length > 0 && filledOptions.length >= 2

  const handleSubmit = () => {
    if (!canSubmit) return
    onCreate({
      question: question.trim(),
      options: filledOptions.map((label, i) => ({ id: `opt-${i}`, label, votes: 0 })),
      votedOptionId: null,
    })
    reset()
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) reset(); onOpenChange(v) }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('kommunikation.poll.title')}</DialogTitle>
          <DialogDescription>{t('kommunikation.poll.description')}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="poll-question">{t('kommunikation.poll.questionLabel')}</Label>
            <Input
              id="poll-question"
              autoFocus
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              placeholder={t('kommunikation.poll.questionPlaceholder')}
            />
          </div>

          <div className="space-y-2">
            <Label>{t('kommunikation.poll.optionsLabel')}</Label>
            <div className="space-y-1.5">
              {options.map((opt, i) => (
                <div key={i} className="flex items-center gap-1.5">
                  <Input
                    value={opt}
                    onChange={(e) => setOptions((prev) => prev.map((o, j) => (j === i ? e.target.value : o)))}
                    placeholder={t('kommunikation.poll.optionPlaceholder', { n: i + 1 })}
                  />
                  {options.length > 2 && (
                    <button
                      type="button"
                      onClick={() => setOptions((prev) => prev.filter((_, j) => j !== i))}
                      className="shrink-0 rounded-md p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                      aria-label={t('kommunikation.poll.removeOption')}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>
              ))}
            </div>
            {options.length < 5 && (
              <button
                type="button"
                onClick={() => setOptions((prev) => [...prev, ''])}
                className="flex items-center gap-1 text-xs text-primary hover:underline"
              >
                <Plus className="h-3.5 w-3.5" />
                {t('kommunikation.poll.addOption')}
              </button>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button type="button" disabled={!canSubmit} onClick={handleSubmit}>
            {t('kommunikation.poll.create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Reminder
// ---------------------------------------------------------------------------

interface ReminderDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreate: (reminder: ConversationReminder) => void
  /** Stable "now" baseline (ms) so the dialog has no impure Date.now at module scope. */
  nowMs: number
}

const REMINDER_OFFSETS: Array<{ key: string; minutes: number }> = [
  { key: 'in1h', minutes: 60 },
  { key: 'in3h', minutes: 180 },
  { key: 'tomorrow', minutes: 60 * 24 },
  { key: 'in3d', minutes: 60 * 24 * 3 },
]

export function ReminderDialog({ open, onOpenChange, onCreate, nowMs }: ReminderDialogProps) {
  const { t } = useTranslation()
  const [text, setText] = useState('')
  const [offsetKey, setOffsetKey] = useState('in1h')

  const reset = () => {
    setText('')
    setOffsetKey('in1h')
  }

  const handleSubmit = () => {
    if (!text.trim()) return
    const offset = REMINDER_OFFSETS.find((o) => o.key === offsetKey) ?? REMINDER_OFFSETS[0]
    const dueAt = new Date(nowMs + offset.minutes * 60000).toISOString()
    onCreate({ text: text.trim(), dueAt })
    reset()
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) reset(); onOpenChange(v) }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('kommunikation.reminder.title')}</DialogTitle>
          <DialogDescription>{t('kommunikation.reminder.description')}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="reminder-text">{t('kommunikation.reminder.textLabel')}</Label>
            <Input
              id="reminder-text"
              autoFocus
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder={t('kommunikation.reminder.textPlaceholder')}
            />
          </div>

          <div className="space-y-2">
            <Label>{t('kommunikation.reminder.whenLabel')}</Label>
            <div className="grid grid-cols-2 gap-1.5">
              {REMINDER_OFFSETS.map((o) => (
                <button
                  key={o.key}
                  type="button"
                  onClick={() => setOffsetKey(o.key)}
                  className={`rounded-md border px-2 py-1.5 text-xs transition-colors ${
                    offsetKey === o.key
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border text-muted-foreground hover:bg-accent'
                  }`}
                >
                  {t(`kommunikation.reminder.offset.${o.key}`)}
                </button>
              ))}
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button type="button" disabled={!text.trim()} onClick={handleSubmit}>
            {t('kommunikation.reminder.create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
