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
import { RecipientField, defaultSignature, filteredSuggestions } from './compose-shared'
import type { ComposeMode } from './ComposeModal'

export default function ComposeWindowPage() {
  const { composeDraft, setComposeDraft, sendEmail, saveDraft } = useMailsStore()

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

  // Load draft state on mount
  useEffect(() => {
    if (composeDraft) {
      setTo(composeDraft.to)
      setCc(composeDraft.cc)
      setBcc(composeDraft.bcc)
      setSubject(composeDraft.subject)
      setBody(composeDraft.body)
      setMode(composeDraft.mode)
      setShowCcBcc(composeDraft.cc.length > 0 || composeDraft.bcc.length > 0)
      // Clear draft from store so it doesn't re-appear
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
    setTimeout(closeWindow, 300)
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
    setTimeout(closeWindow, 300)
  }

  const handleDiscard = () => {
    setComposeDraft(null)
    closeWindow()
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
    <div className="flex flex-col h-screen bg-background text-foreground">
      {/* Draggable title bar area */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-border bg-card" style={{ WebkitAppRegion: 'drag' } as React.CSSProperties}>
        <h2 className="text-sm font-medium">{modeTitle[mode]}</h2>
      </div>

      {/* Form */}
      <div className="flex-1 overflow-y-auto px-5 py-4 space-y-3" style={{ WebkitAppRegion: 'no-drag' } as React.CSSProperties}>
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
      <div className="flex items-center justify-between px-5 py-3 border-t border-border bg-card">
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
            onClick={handleDiscard}
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
