import { useState } from 'react'
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
  label: string
  icon: typeof Mail
  color: string
  description: string
}

const channelOptions: ChannelOption[] = [
  { id: 'email', label: 'E-Mail', icon: Mail, color: 'text-blue-500', description: 'E-Mail Konversation starten' },
  { id: 'teams', label: 'Teams', icon: Users, color: 'text-violet-500', description: 'Microsoft Teams Nachricht' },
  { id: 'whatsapp', label: 'WhatsApp', icon: MessageCircle, color: 'text-green-500', description: 'WhatsApp Nachricht senden' },
  { id: 'widget', label: 'Widget', icon: Globe, color: 'text-orange-500', description: 'Website-Chat starten' },
  { id: 'portal', label: 'Portal', icon: Headphones, color: 'text-teal-500', description: 'Kundenportal-Nachricht' },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface NewConversationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function NewConversationDialog({ open, onOpenChange }: NewConversationDialogProps) {
  const [channel, setChannel] = useState<CommunicationChannel>('email')
  const [recipient, setRecipient] = useState('')
  const [subject, setSubject] = useState('')
  const [message, setMessage] = useState('')

  const handleSend = () => {
    if (!recipient.trim() || !subject.trim() || !message.trim()) return
    toast.success('Konversation erstellt (Mock)')
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
          <DialogTitle>Neue Konversation</DialogTitle>
          <DialogDescription>
            Starte eine neue externe Konversation über einen beliebigen Kanal.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Channel selection */}
          <div>
            <label className="text-xs font-medium text-foreground mb-1.5 block">Kanal</label>
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
                    <span className="text-[10px]">{ch.label}</span>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Recipient */}
          <div>
            <label className="text-xs font-medium text-foreground mb-1 block">Empfänger</label>
            <input
              type="text"
              value={recipient}
              onChange={(e) => setRecipient(e.target.value)}
              placeholder={channel === 'email' ? 'name@firma.de' : 'Name oder Kontakt suchen...'}
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary transition-colors"
            />
          </div>

          {/* Subject */}
          <div>
            <label className="text-xs font-medium text-foreground mb-1 block">Betreff</label>
            <input
              type="text"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="Betreff eingeben..."
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary transition-colors"
            />
          </div>

          {/* Message */}
          <div>
            <label className="text-xs font-medium text-foreground mb-1 block">Nachricht</label>
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Nachricht schreiben..."
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
              Abbrechen
            </button>
            <button
              onClick={handleSend}
              disabled={!recipient.trim() || !subject.trim() || !message.trim()}
              className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              Senden
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
