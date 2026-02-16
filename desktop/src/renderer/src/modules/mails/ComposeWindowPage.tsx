import { useState, useEffect } from 'react'
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
import { useMailsStore } from '@/stores/mails'
import type { ComposeMode } from '@/stores/mails'
import { useSendEmail, useSaveDraft, useReplyEmail, useForwardEmail } from '@/api/hooks/useEmail'
import type { EmailAddress } from '@/api/email-types'
import { RecipientField, defaultSignature, filteredSuggestions } from './compose-shared'

export default function ComposeWindowPage() {
  const { composeDraft, setComposeDraft } = useMailsStore()
  const sendEmail = useSendEmail()
  const saveDraftMutation = useSaveDraft()
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
  const [mode, setMode] = useState<ComposeMode>('compose')
  const [replyToMessageId, setReplyToMessageId] = useState<string | undefined>(undefined)
  const [accountId, setAccountId] = useState<string>('')

  // Load draft state on mount
  useEffect(() => {
    if (composeDraft) {
      setTo(composeDraft.to)
      setCc(composeDraft.cc)
      setBcc(composeDraft.bcc)
      setSubject(composeDraft.subject)
      setBody(composeDraft.body)
      setMode(composeDraft.mode)
      setReplyToMessageId(composeDraft.replyToMessageId)
      setAccountId(composeDraft.accountId ?? '')
      setShowCcBcc(composeDraft.cc.length > 0 || composeDraft.bcc.length > 0)
      setComposeDraft(null)
    }
  }, [])

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

  const closeWindow = () => {
    window.electronAPI?.window.close()
  }

  const toAddresses = (emails: string[]): EmailAddress[] =>
    emails.map((e) => ({ name: '', email: e }))

  const handleSend = () => {
    if (to.length === 0 || !subject.trim() || !accountId) return

    if ((mode === 'reply' || mode === 'reply-all') && replyToMessageId) {
      replyEmail.mutate({
        account_id: accountId,
        original_message_id: replyToMessageId,
        body_html: `<p>${body.replace(/\n/g, '<br>')}</p>`,
        body_text: body,
        reply_all: mode === 'reply-all',
      }, {
        onSuccess: () => { toast.success('E-Mail gesendet'); setTimeout(closeWindow, 300) },
        onError: (err) => toast.error(`Senden fehlgeschlagen: ${err.message}`),
      })
    } else if (mode === 'forward' && replyToMessageId) {
      forwardEmail.mutate({
        account_id: accountId,
        original_message_id: replyToMessageId,
        to: toAddresses(to),
        body_html: `<p>${body.replace(/\n/g, '<br>')}</p>`,
        body_text: body,
      }, {
        onSuccess: () => { toast.success('E-Mail weitergeleitet'); setTimeout(closeWindow, 300) },
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
        onSuccess: () => { toast.success('E-Mail gesendet'); setTimeout(closeWindow, 300) },
        onError: (err) => toast.error(`Senden fehlgeschlagen: ${err.message}`),
      })
    }
  }

  const handleSaveDraft = () => {
    if (!accountId) return
    saveDraftMutation.mutate({
      account_id: accountId,
      to: toAddresses(to),
      cc: cc.length > 0 ? toAddresses(cc) : undefined,
      bcc: bcc.length > 0 ? toAddresses(bcc) : undefined,
      subject: subject.trim() || '(Kein Betreff)',
      body_html: `<p>${body.replace(/\n/g, '<br>')}</p>`,
      body_text: body,
      in_reply_to_message_id: replyToMessageId,
    }, {
      onSuccess: () => { toast.success('Entwurf gespeichert'); setTimeout(closeWindow, 300) },
      onError: (err) => toast.error(`Speichern fehlgeschlagen: ${err.message}`),
    })
  }

  const handleDiscard = () => {
    setComposeDraft(null)
    closeWindow()
  }

  const isSending = sendEmail.isPending || replyEmail.isPending || forwardEmail.isPending

  const modeTitle: Record<ComposeMode, string> = {
    compose: 'Neue E-Mail',
    reply: 'Antworten',
    'reply-all': 'Allen antworten',
    forward: 'Weiterleiten',
  }

  const toSuggestions = filteredSuggestions(toInput, to)
  const ccSuggestions = filteredSuggestions(ccInput, cc)
  const bccSuggestions = filteredSuggestions(bccInput, bcc)

  return (
    <div className="flex flex-col h-screen bg-background text-foreground">
      <div className="flex items-center justify-between px-4 py-2 border-b border-border bg-card" style={{ WebkitAppRegion: 'drag' } as React.CSSProperties}>
        <h2 className="text-sm font-medium">{modeTitle[mode]}</h2>
      </div>

      <div className="flex-1 overflow-y-auto px-5 py-4 space-y-3" style={{ WebkitAppRegion: 'no-drag' } as React.CSSProperties}>
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

        <Input
          placeholder="Betreff"
          value={subject}
          onChange={(e) => setSubject(e.target.value)}
          className="border-0 border-b border-border rounded-none px-1 focus-visible:ring-0 font-medium"
        />

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
          className="min-h-[200px] flex-1 border-0 resize-none focus-visible:ring-0 px-1"
        />
      </div>

      <div className="flex items-center justify-between px-5 py-3 border-t border-border bg-card">
        <div className="flex items-center gap-2">
          <Button onClick={handleSend} disabled={to.length === 0 || isSending}>
            <Send className="mr-1.5 h-4 w-4" />
            {isSending ? 'Sende...' : 'Senden'}
          </Button>
          <Button variant="outline" size="icon" onClick={handleSaveDraft} disabled={saveDraftMutation.isPending} title="Als Entwurf speichern">
            <Save className="h-4 w-4" />
          </Button>
        </div>
        <div className="flex items-center gap-1">
          <button className="rounded p-1.5 text-muted-foreground hover:bg-secondary transition-colors" title="Datei anhaengen">
            <Paperclip className="h-4 w-4" />
          </button>
          <button onClick={handleDiscard} className="rounded p-1.5 text-muted-foreground hover:text-red-500 transition-colors" title="Verwerfen">
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  )
}
