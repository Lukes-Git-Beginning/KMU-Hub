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
  X,
  Save,
} from 'lucide-react'
import { toast } from 'sonner'
import { useMailsStore, type Email } from '@/stores/mails'

export type ComposeMode = 'compose' | 'reply' | 'reply-all' | 'forward'

interface ComposeModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  mode?: ComposeMode
  replyTo?: Email | null
  prefillTo?: string
  prefillName?: string
}

const defaultSignature = '\n\n--\nMit freundlichen Gruessen\nDarien\nKMU Hub AG'

const contactSuggestions = [
  { name: 'Anna Mueller', email: 'anna.mueller@kmuhub.ch' },
  { name: 'Michael Berg', email: 'michael.berg@kmuhub.ch' },
  { name: 'Sarah Klein', email: 'sarah@designstudio.ch' },
  { name: 'Thomas Weber', email: 'thomas.weber@abc-gmbh.ch' },
  { name: 'Lisa Schmidt', email: 'lisa.schmidt@kmuhub.ch' },
  { name: 'Peter Koch', email: 'peter.koch@kmuhub.ch' },
  { name: 'Jonas Diaz', email: 'jonas.diaz@kmuhub.ch' },
  { name: 'Eva Brunner', email: 'eva@brunner-partner.ch' },
  { name: 'Markus Steiner', email: 'markus@steiner-bau.ch' },
  { name: 'Claudia Frei', email: 'claudia.frei@techventures.at' },
]

export function ComposeModal({
  open,
  onOpenChange,
  mode = 'compose',
  replyTo,
  prefillTo,
  prefillName,
}: ComposeModalProps) {
  const { sendEmail, saveDraft } = useMailsStore()

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
      setBody(`${defaultSignature}\n\n---\nAm ${replyTo.date} um ${replyTo.time} schrieb ${replyTo.from.name}:\n\n${replyTo.body}`)
    } else if (mode === 'reply-all' && replyTo) {
      setTo([replyTo.from.email])
      setCc(replyTo.cc.filter((e) => e !== 'darien@kmuhub.ch'))
      setShowCcBcc(replyTo.cc.length > 0)
      setBcc([])
      setSubject(`RE: ${replyTo.subject.replace(/^(RE: |FW: )+/i, '')}`)
      setBody(`${defaultSignature}\n\n---\nAm ${replyTo.date} um ${replyTo.time} schrieb ${replyTo.from.name}:\n\n${replyTo.body}`)
    } else if (mode === 'forward' && replyTo) {
      setTo([])
      setCc([])
      setBcc([])
      setSubject(`FW: ${replyTo.subject.replace(/^(RE: |FW: )+/i, '')}`)
      setBody(`${defaultSignature}\n\n--- Weitergeleitete Nachricht ---\nVon: ${replyTo.from.name} <${replyTo.from.email}>\nDatum: ${replyTo.date} ${replyTo.time}\nBetreff: ${replyTo.subject}\n\n${replyTo.body}`)
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

  const filteredSuggestions = (input: string, exclude: string[]) =>
    input.length > 0
      ? contactSuggestions.filter(
          (c) =>
            !exclude.includes(c.email) &&
            (c.name.toLowerCase().includes(input.toLowerCase()) ||
              c.email.toLowerCase().includes(input.toLowerCase()))
        )
      : []

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

  const handleSend = () => {
    if (to.length === 0 || !subject.trim()) return
    sendEmail({
      from: { name: 'Du', email: 'darien@kmuhub.ch', initials: 'DK' },
      to,
      cc,
      bcc,
      subject: subject.trim(),
      preview: body.split('\n')[0].slice(0, 100),
      body,
      attachments: [],
      signature: defaultSignature,
    })
    toast.success('E-Mail gesendet')
    onOpenChange(false)
  }

  const handleSaveDraft = () => {
    saveDraft({
      from: { name: 'Du', email: 'darien@kmuhub.ch', initials: 'DK' },
      to,
      cc,
      bcc,
      subject: subject.trim() || '(Kein Betreff)',
      preview: body.split('\n')[0].slice(0, 100),
      body,
      attachments: [],
      signature: defaultSignature,
    })
    toast.success('Entwurf gespeichert')
    onOpenChange(false)
  }

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
          {/* To */}
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

          {/* Cc/Bcc toggle */}
          {!showCcBcc && (
            <button
              onClick={() => setShowCcBcc(true)}
              className="text-xs text-primary hover:underline ml-1"
            >
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

          {/* Subject */}
          <div className="space-y-1">
            <Input
              placeholder="Betreff"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              className="border-0 border-b border-border rounded-none px-1 focus-visible:ring-0 font-medium"
            />
          </div>

          {/* Formatting toolbar */}
          <div className="flex items-center gap-0.5 border-b border-border pb-2">
            {[Bold, Italic, Underline].map((Icon, i) => (
              <button
                key={i}
                className="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
              >
                <Icon className="h-4 w-4" />
              </button>
            ))}
            <span className="mx-1 h-4 w-px bg-border" />
            {[List, ListOrdered].map((Icon, i) => (
              <button
                key={i}
                className="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
              >
                <Icon className="h-4 w-4" />
              </button>
            ))}
            <span className="mx-1 h-4 w-px bg-border" />
            <button className="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors">
              <Link className="h-4 w-4" />
            </button>
          </div>

          {/* Body */}
          <Textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder="Nachricht schreiben..."
            className="min-h-[200px] border-0 resize-none focus-visible:ring-0 px-1"
          />
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between pt-3 border-t border-border">
          <div className="flex items-center gap-2">
            <Button onClick={handleSend} disabled={to.length === 0}>
              <Send className="mr-1.5 h-4 w-4" />
              Senden
            </Button>
            <Button variant="outline" size="icon" onClick={handleSaveDraft} title="Als Entwurf speichern">
              <Save className="h-4 w-4" />
            </Button>
          </div>
          <div className="flex items-center gap-1">
            <button className="rounded p-1.5 text-muted-foreground hover:bg-secondary transition-colors" title="Datei anhaengen">
              <Paperclip className="h-4 w-4" />
            </button>
            <button
              onClick={() => onOpenChange(false)}
              className="rounded p-1.5 text-muted-foreground hover:text-red-500 transition-colors"
              title="Verwerfen"
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function RecipientField({
  label,
  recipients,
  input,
  onInputChange,
  suggestions,
  onAdd,
  onRemove,
  onKeyDown,
}: {
  label: string
  recipients: string[]
  input: string
  onInputChange: (v: string) => void
  suggestions: typeof contactSuggestions
  onAdd: (email: string) => void
  onRemove: (email: string) => void
  onKeyDown: (e: React.KeyboardEvent) => void
}) {
  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <span className="text-xs text-muted-foreground w-6 shrink-0">{label}</span>
        <div className="flex flex-1 flex-wrap items-center gap-1 rounded-md border border-border px-2 py-1 min-h-[34px]">
          {recipients.map((email) => (
            <span
              key={email}
              className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary"
            >
              {email}
              <button onClick={() => onRemove(email)} className="rounded-full hover:bg-primary/20 p-0.5">
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
          <input
            type="text"
            value={input}
            onChange={(e) => onInputChange(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder={recipients.length === 0 ? 'Empfaenger...' : ''}
            className="flex-1 min-w-[120px] bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
          />
        </div>
      </div>
      {suggestions.length > 0 && (
        <div className="ml-8 max-h-28 overflow-y-auto rounded-md border bg-card p-1">
          {suggestions.slice(0, 5).map((c) => (
            <button
              key={c.email}
              onClick={() => onAdd(c.email)}
              className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-secondary"
            >
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                {c.name.split(' ').map((n) => n[0]).join('')}
              </span>
              <span className="text-foreground">{c.name}</span>
              <span className="text-xs text-muted-foreground ml-auto">{c.email}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

