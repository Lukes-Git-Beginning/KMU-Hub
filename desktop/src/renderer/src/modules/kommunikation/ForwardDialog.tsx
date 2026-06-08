/**
 * Forward dialog (mock-first).
 *
 * Lets the user forward an inbox message to a colleague or external address
 * with an optional note. There is no Forward RPC in the backend yet
 * (see backend-gaps.md) — on submit we show a success toast. The form shape
 * (recipient + note) is adapter-ready for the future ForwardMessage RPC.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Forward } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import type { InboxMessage } from '@/api/inbox-types'

interface ForwardDialogProps {
  message: InboxMessage
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ForwardDialog({ message, open, onOpenChange }: ForwardDialogProps) {
  const { t } = useTranslation()
  const [recipient, setRecipient] = useState('')
  const [note, setNote] = useState('')

  const handleForward = () => {
    if (!recipient.trim()) return
    // Mock-first: no Forward RPC yet (backend-gaps.md).
    toast.success(t('kommunikation.forward.sent', { recipient: recipient.trim() }))
    setRecipient('')
    setNote('')
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Forward className="h-4 w-4" />
            {t('kommunikation.forward.title')}
          </DialogTitle>
          <DialogDescription>
            {t('kommunikation.forward.description', { subject: message.subject })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          <div>
            <label className="text-xs font-medium text-foreground">
              {t('kommunikation.forward.recipient')}
            </label>
            <input
              type="text"
              value={recipient}
              onChange={(e) => setRecipient(e.target.value)}
              placeholder={t('kommunikation.forward.recipientPlaceholder')}
              className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          <div>
            <label className="text-xs font-medium text-foreground">
              {t('kommunikation.forward.note')}
            </label>
            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              rows={3}
              placeholder={t('kommunikation.forward.notePlaceholder')}
              className="mt-1 w-full resize-none rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          <div className="flex justify-end gap-2 pt-1">
            <button
              onClick={() => onOpenChange(false)}
              className="rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleForward}
              disabled={!recipient.trim()}
              className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
            >
              <Forward className="h-3.5 w-3.5" />
              {t('kommunikation.forward.submit')}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
