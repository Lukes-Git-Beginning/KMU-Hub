import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Mail,
  MessageCircle,
  Globe,
  Headphones,
  Users,
} from 'lucide-react'
import { toast } from 'sonner'
import type { CommunicationChannel } from '@/types/communication'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Channel options
// ---------------------------------------------------------------------------

interface ChannelOption {
  id: CommunicationChannel
  labelKey: string
  icon: typeof Mail
  color: string
  descriptionKey: string
}

const channelOptions: ChannelOption[] = [
  { id: 'email', labelKey: 'kommunikation.channels.email', icon: Mail, color: 'text-blue-500', descriptionKey: 'kommunikation.newConversation.emailDesc' },
  { id: 'teams', labelKey: 'kommunikation.channels.teams', icon: Users, color: 'text-violet-500', descriptionKey: 'kommunikation.newConversation.teamsDesc' },
  { id: 'whatsapp', labelKey: 'kommunikation.channels.whatsapp', icon: MessageCircle, color: 'text-green-500', descriptionKey: 'kommunikation.newConversation.whatsappDesc' },
  { id: 'widget', labelKey: 'kommunikation.channels.widget', icon: Globe, color: 'text-orange-500', descriptionKey: 'kommunikation.newConversation.widgetDesc' },
  { id: 'portal', labelKey: 'kommunikation.channels.portal', icon: Headphones, color: 'text-teal-500', descriptionKey: 'kommunikation.newConversation.portalDesc' },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface NewConversationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function NewConversationDialog({ open, onOpenChange }: NewConversationDialogProps) {
  const { t } = useTranslation()
  const [channel, setChannel] = useState<CommunicationChannel>('email')
  const [recipient, setRecipient] = useState('')
  const [subject, setSubject] = useState('')
  const [message, setMessage] = useState('')

  const handleSend = () => {
    if (!recipient.trim() || !subject.trim() || !message.trim()) return
    toast.success(t('kommunikation.newConversation.createdMock'))
    onOpenChange(false)
    setRecipient('')
    setSubject('')
    setMessage('')
    setChannel('email')
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('kommunikation.newConversation.title')}</DialogTitle>
          <DialogDescription>
            {t('kommunikation.newConversation.description')}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Channel selection */}
          <div>
            <label className="text-xs font-medium text-foreground mb-1.5 block">{t('kommunikation.newConversation.channel')}</label>
            <div className="grid grid-cols-5 gap-1.5">
              {channelOptions.map((ch) => {
                const Icon = ch.icon
                const isActive = channel === ch.id
                return (
                  <button
                    key={ch.id}
                    onClick={() => setChannel(ch.id)}
                    className={`flex flex-col items-center gap-1 rounded-md border px-2 py-2 text-xs transition-colors ${
                      isActive
                        ? 'border-primary bg-primary/5 text-primary'
                        : 'border-border text-muted-foreground hover:bg-accent'
                    }`}
                  >
                    <Icon className={`h-4 w-4 ${isActive ? 'text-primary' : ch.color}`} />
                    <span className="text-[10px]">{t(ch.labelKey)}</span>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Recipient */}
          <div>
            <label className="text-xs font-medium text-foreground mb-1 block">{t('kommunikation.newConversation.recipient')}</label>
            <input
              type="text"
              value={recipient}
              onChange={(e) => setRecipient(e.target.value)}
              placeholder={channel === 'email' ? t('kommunikation.newConversation.emailPlaceholder') : t('kommunikation.newConversation.contactPlaceholder')}
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary transition-colors"
            />
          </div>

          {/* Subject */}
          <div>
            <label className="text-xs font-medium text-foreground mb-1 block">{t('kommunikation.newConversation.subject')}</label>
            <input
              type="text"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder={t('kommunikation.newConversation.subjectPlaceholder')}
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary transition-colors"
            />
          </div>

          {/* Message */}
          <div>
            <label className="text-xs font-medium text-foreground mb-1 block">{t('kommunikation.newConversation.message')}</label>
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder={t('kommunikation.newConversation.messagePlaceholder')}
              rows={4}
              className="w-full rounded-md border border-border bg-transparent px-3 py-2 text-sm outline-none placeholder:text-muted-foreground focus:border-primary resize-none transition-colors"
            />
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2 pt-1">
            <button
              onClick={() => onOpenChange(false)}
              className="h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleSend}
              disabled={!recipient.trim() || !subject.trim() || !message.trim()}
              className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {t('kommunikation.reply.send')}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
