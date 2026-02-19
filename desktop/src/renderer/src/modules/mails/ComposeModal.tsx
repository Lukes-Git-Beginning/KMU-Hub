import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  Send,
  Paperclip,
  Bold,
  Italic,
  Underline,
  List,
  ListOrdered,
  Link,
  Save,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import type { ComposeMode } from '@/stores/mails'
import { useSendEmail, useSaveDraft, useReplyEmail, useForwardEmail } from '@/api/hooks/useEmail'
import type { EmailMessageInfo, EmailAddress } from '@/api/email-types'
import { RecipientField, defaultSignature, filteredSuggestions } from './compose-shared'

export type { ComposeMode }

interface ComposeModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  mode?: ComposeMode
  replyTo?: EmailMessageInfo | null
  prefillTo?: string
  accountId: string
}

export function ComposeModal({
  open,
  onOpenChange,
  mode = 'compose',
  replyTo,
  prefillTo,
  accountId,
}: ComposeModalProps) {
  const sendEmail = useSendEmail()
  const saveDraft = useSaveDraft()
  const replyEmail = useReplyEmail()
  const forwardEmail = useForwardEmail()

  const [to, setTo] = useState<string[]>([])
  const [toInput, setToInput] = useState('')
  const [showCcBcc, setShowCcBcc] = useState(false)
  const [cc, setCc] = useState<string[]>([])
  const [ccInput, setCcInput] = useState('')
  const [bcc, setBcc] = useState<string[]>([])
  const [bccInput, setBccInput] = useState('')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')

  useEffect(() => {
    if (!open) return

    if (mode === 'reply' && replyTo) {
      setTo([replyTo.from.email])
      setCc([])
      setBcc([])
      setSubject(`RE: ${replyTo.subject.replace(/^(RE: |FW: )+/i, '')}`)
      setBody(`${defaultSignature}\n\n---\nAm ${replyTo.date} schrieb ${replyTo.from.name || replyTo.from.email}:\n\n${replyTo.body_text || ''}`)
    } else if (mode === 'reply-all' && replyTo) {
      setTo([replyTo.from.email])
      setCc(replyTo.cc.map((a) => a.email))
      setShowCcBcc(replyTo.cc.length > 0)
      setBcc([])
      setSubject(`RE: ${replyTo.subject.replace(/^(RE: |FW: )+/i, '')}`)
      setBody(`${defaultSignature}\n\n---\nAm ${replyTo.date} schrieb ${replyTo.from.name || replyTo.from.email}:\n\n${replyTo.body_text || ''}`)
    } else if (mode === 'forward' && replyTo) {
      setTo([])
      setCc([])
      setBcc([])
      setSubject(`FW: ${replyTo.subject.replace(/^(RE: |FW: )+/i, '')}`)
      setBody(`${defaultSignature}\n\n--- Weitergeleitete Nachricht ---\nVon: ${replyTo.from.name || ''} <${replyTo.from.email}>\nDatum: ${replyTo.date}\nBetreff: ${replyTo.subject}\n\n${replyTo.body_text || ''}`)
    } else {
      setTo(prefillTo ? [prefillTo] : [])
      setCc([])
      setBcc([])
      setSubject('')
      setBody(defaultSignature)
      setShowCcBcc(false)
    }
    setToInput('')
    setCcInput('')
    setBccInput('')
  }, [open, mode, replyTo, prefillTo])

  const addRecipient = (
    email: string,
    setter: React.Dispatch<React.SetStateAction<string[]>>,
    inputSetter: React.Dispatch<React.SetStateAction<string>>
  ) => {
    setter((prev) => [...prev, email])
    inputSetter('')
  }

  const removeRecipient = (
    email: string,
    setter: React.Dispatch<React.SetStateAction<string[]>>
  ) => {
    setter((prev) => prev.filter((e) => e !== email))
  }

  const toAddresses = (emails: string[]): EmailAddress[] =>
    emails.map((e) => ({ name: '', email: e }))

  const handleSend = () => {
    if (to.length === 0 || !subject.trim()) return

    if ((mode === 'reply' || mode === 'reply-all') && replyTo) {
      replyEmail.mutate({
        account_id: accountId,
        original_message_id: replyTo.id,
        body_html: `<p>${body.replace(/\n/g, '<br>')}</p>`,
        body_text: body,
        reply_all: mode === 'reply-all',
      }, {
        onSuccess: () => { toast.success('E-Mail gesendet'); onOpenChange(false) },
        onError: (err) => toast.error(`Senden fehlgeschlagen: ${err.message}`),
      })
    } else if (mode === 'forward' && replyTo) {
      forwardEmail.mutate({
        account_id: accountId,
        original_message_id: replyTo.id,
        to: toAddresses(to),
        body_html: `<p>${body.replace(/\n/g, '<br>')}</p>`,
        body_text: body,
      }, {
        onSuccess: () => { toast.success('E-Mail weitergeleitet'); onOpenChange(false) },
        onError: (err) => toast.error(`Weiterleiten fehlgeschlagen: ${err.message}`),
      })
    } else {
      sendEmail.mutate({
        account_id: accountId,
        to: toAddresses(to),
        cc: cc.length > 0 ? toAddresses(cc) : undefined,
        bcc: bcc.length > 0 ? toAddresses(bcc) : undefined,
        subject: subject.trim(),
        body_html: `<p>${body.replace(/\n/g, '<br>')}</p>`,
        body_text: body,
      }, {
        onSuccess: () => { toast.success('E-Mail gesendet'); onOpenChange(false) },
        onError: (err) => toast.error(`Senden fehlgeschlagen: ${err.message}`),
      })
    }
  }

  const handleSaveDraft = () => {
    saveDraft.mutate({
      account_id: accountId,
      to: toAddresses(to),
      cc: cc.length > 0 ? toAddresses(cc) : undefined,
      bcc: bcc.length > 0 ? toAddresses(bcc) : undefined,
      subject: subject.trim() || '(Kein Betreff)',
      body_html: `<p>${body.replace(/\n/g, '<br>')}</p>`,
      body_text: body,
      in_reply_to_message_id: replyTo?.id,
    }, {
      onSuccess: () => { toast.success('Entwurf gespeichert'); onOpenChange(false) },
      onError: (err) => toast.error(`Speichern fehlgeschlagen: ${err.message}`),
    })
  }

  const isSending = sendEmail.isPending || replyEmail.isPending || forwardEmail.isPending

  const modeTitle = {
    compose: 'Neue E-Mail',
    reply: 'Antworten',
    'reply-all': 'Allen antworten',
    forward: 'Weiterleiten',
  }

  const toSuggestions = filteredSuggestions(toInput, to)
  const ccSuggestions = filteredSuggestions(ccInput, cc)
  const bccSuggestions = filteredSuggestions(bccInput, bcc)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>{modeTitle[mode]}</DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-3 py-2">
          <RecipientField
            label="An"
            recipients={to}
            input={toInput}
            onInputChange={setToInput}
            suggestions={toSuggestions}
            onAdd={(email) => addRecipient(email, setTo, setToInput)}
            onRemove={(email) => removeRecipient(email, setTo)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && toInput.includes('@')) {
                addRecipient(toInput, setTo, setToInput)
              }
            }}
          />

          {!showCcBcc && (
            <button onClick={() => setShowCcBcc(true)} className="text-xs text-primary hover:underline ml-1">
              Cc/Bcc hinzufuegen
            </button>
          )}

          {showCcBcc && (
            <>
              <RecipientField
                label="Cc"
                recipients={cc}
                input={ccInput}
                onInputChange={setCcInput}
                suggestions={ccSuggestions}
                onAdd={(email) => addRecipient(email, setCc, setCcInput)}
                onRemove={(email) => removeRecipient(email, setCc)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && ccInput.includes('@')) {
                    addRecipient(ccInput, setCc, setCcInput)
                  }
                }}
              />
              <RecipientField
                label="Bcc"
                recipients={bcc}
                input={bccInput}
                onInputChange={setBccInput}
                suggestions={bccSuggestions}
                onAdd={(email) => addRecipient(email, setBcc, setBccInput)}
                onRemove={(email) => removeRecipient(email, setBcc)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && bccInput.includes('@')) {
                    addRecipient(bccInput, setBcc, setBccInput)
                  }
                }}
              />
            </>
          )}

          <div className="space-y-1">
            <Input
              placeholder="Betreff"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              className="border-0 border-b border-border rounded-none px-1 focus-visible:ring-0 font-medium"
            />
          </div>

          <div className="flex items-center gap-0.5 border-b border-border pb-2">
            {[Bold, Italic, Underline].map((Icon, i) => (
              <button key={i} className="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors">
                <Icon className="h-4 w-4" />
              </button>
            ))}
            <span className="mx-1 h-4 w-px bg-border" />
            {[List, ListOrdered].map((Icon, i) => (
              <button key={i} className="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors">
                <Icon className="h-4 w-4" />
              </button>
            ))}
            <span className="mx-1 h-4 w-px bg-border" />
            <button className="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors">
              <Link className="h-4 w-4" />
            </button>
          </div>

          <Textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder="Nachricht schreiben..."
            className="min-h-[200px] border-0 resize-none focus-visible:ring-0 px-1"
          />
        </div>

        <div className="flex items-center justify-between pt-3 border-t border-border">
          <div className="flex items-center gap-2">
            <Button onClick={handleSend} disabled={to.length === 0 || isSending}>
              <Send className="mr-1.5 h-4 w-4" />
              {isSending ? 'Sende...' : 'Senden'}
            </Button>
            <Button variant="outline" size="icon" onClick={handleSaveDraft} disabled={saveDraft.isPending} title="Als Entwurf speichern">
              <Save className="h-4 w-4" />
            </Button>
          </div>
          <div className="flex items-center gap-1">
            <button className="rounded p-1.5 text-muted-foreground hover:bg-secondary transition-colors" title="Datei anhaengen">
              <Paperclip className="h-4 w-4" />
            </button>
            <button onClick={() => onOpenChange(false)} className="rounded p-1.5 text-muted-foreground hover:text-red-500 transition-colors" title="Verwerfen">
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
