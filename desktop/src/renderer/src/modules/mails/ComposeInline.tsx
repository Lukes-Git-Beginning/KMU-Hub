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
  Maximize2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useMailsStore, type Email } from '@/stores/mails'
import { RecipientField, defaultSignature, filteredSuggestions } from './compose-shared'
import type { ComposeMode } from './ComposeModal'

interface ComposeInlineProps {
  mode?: ComposeMode
  replyTo?: Email | null
  prefillTo?: string
  onClose: () => void
}

export function ComposeInline({
  mode = 'compose',
  replyTo,
  prefillTo,
  onClose,
}: ComposeInlineProps) {
  const { sendEmail, saveDraft, setComposeDraft } = useMailsStore()

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
  }, [mode, replyTo, prefillTo])

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
    onClose()
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
    onClose()
  }

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
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-6 py-3">
        <h3 className="text-sm font-medium text-foreground">{modeTitle[mode]}</h3>
        <button
          onClick={() => {
            // Save current state to store for the new window to read
            setComposeDraft({ to, cc, bcc, subject, body, mode })
            // Open real OS window via Electron IPC
            window.electronAPI?.compose.openWindow()
            // Close inline panel
            onClose()
          }}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title="Als Fenster öffnen"
        >
          <Maximize2 className="h-4 w-4" />
        </button>
      </div>

      {/* Form */}
      <div className="flex-1 overflow-y-auto px-6 py-4 space-y-3">
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
            Cc/Bcc hinzufügen
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
        <Input
          placeholder="Betreff"
          value={subject}
          onChange={(e) => setSubject(e.target.value)}
          className="border-0 border-b border-border rounded-none px-1 focus-visible:ring-0 font-medium"
        />

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
          className="min-h-[200px] flex-1 border-0 resize-none focus-visible:ring-0 px-1"
        />
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between px-6 py-3 border-t border-border">
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
          <button className="rounded p-1.5 text-muted-foreground hover:bg-secondary transition-colors" title="Datei anhängen">
            <Paperclip className="h-4 w-4" />
          </button>
          <button
            onClick={onClose}
            className="rounded p-1.5 text-muted-foreground hover:text-red-500 transition-colors"
            title="Verwerfen"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  )
}
