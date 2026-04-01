import { useState, useEffect, useMemo } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Send,
  Paperclip,
  Save,
  Trash2,
  FileText,
  Sparkles,
} from 'lucide-react'
import { toast } from 'sonner'
import type { ComposeMode } from '@/stores/mails'
import { useSendEmail, useSaveDraft, useReplyEmail, useForwardEmail } from '@/api/hooks/useEmail'
import type { EmailMessageInfo, EmailAddress } from '@/api/email-types'
import { LazyRichTextEditor as RichTextEditor } from '@/components/shared/RichTextEditor'
import { EmailTemplateDialog } from './EmailTemplateDialog'
import { useAIStore } from '@/stores/ai'
import {
  RecipientField,
  useEmailSignature,
  useContactSuggestions,
  filteredSuggestions,
  buildContactMap,
  buildSignatureHtml,
  buildReplyHtml,
  buildForwardHtml,
  stripHtml,
} from './compose-shared'

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

  const signature = useEmailSignature()
  const allContacts = useContactSuggestions()
  const contactMap = useMemo(() => buildContactMap(allContacts), [allContacts])

  const [to, setTo] = useState<string[]>([])
  const [toInput, setToInput] = useState('')
  const [showCcBcc, setShowCcBcc] = useState(false)
  const [cc, setCc] = useState<string[]>([])
  const [ccInput, setCcInput] = useState('')
  const [bcc, setBcc] = useState<string[]>([])
  const [bccInput, setBccInput] = useState('')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [editorVersion, setEditorVersion] = useState(0)
  const [templateOpen, setTemplateOpen] = useState(false)
  const [aiDraftLoading, setAIDraftLoading] = useState(false)
  const aiEmailEnabled = useAIStore((s) => s.isModuleEnabled('email'))

  const handleAIDraft = () => {
    setAIDraftLoading(true)
    setTimeout(() => {
      const isReply = mode === 'reply' || mode === 'reply-all'
      const mockDraft = isReply
        ? '<p>Sehr geehrter Herr/Frau [Name],</p><p>vielen Dank für Ihre Nachricht. Ich habe Ihr Anliegen geprüft und moechte Ihnen folgendes mitteilen:</p><p>[Hier Ihre Antwort einfügen]</p><p>Sollten Sie weitere Fragen haben, stehe ich Ihnen gerne zur Verfuegung.</p><p>Mit freundlichen Grüßen</p>'
        : '<p>Sehr geehrte Damen und Herren,</p><p>ich schreibe Ihnen bezueglich [Thema]. Gerne moechte ich folgendes besprechen:</p><p>1. [Punkt 1]</p><p>2. [Punkt 2]</p><p>Ich freue mich auf Ihre Rueckmeldung.</p><p>Mit freundlichen Grüßen</p>'
      setBody(mockDraft)
      setEditorVersion((v) => v + 1)
      setAIDraftLoading(false)
      useAIStore.getState().addActivityLog({
        module: 'E-Mail',
        action: isReply ? 'Antwort-Entwurf generiert' : 'Entwurf generiert',
        inputPreview: subject || '(Kein Betreff)',
        outputPreview: 'Sehr geehrte...',
      })
      toast.success('KI-Entwurf eingefuegt')
    }, 1500)
  }

  useEffect(() => {
    if (!open) return

    if (mode === 'reply' && replyTo) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setTo([replyTo.from.email])
      setCc([])
      setBcc([])
      setSubject(`RE: ${replyTo.subject.replace(/^(RE: |FW: )+/i, '')}`)
      setBody(
        buildReplyHtml(
          signature,
          `Am ${replyTo.date} schrieb ${replyTo.from.name || replyTo.from.email}:`,
          replyTo.body_html || `<p>${replyTo.body_text || ''}</p>`,
        ),
      )
    } else if (mode === 'reply-all' && replyTo) {
      setTo([replyTo.from.email])
      setCc(replyTo.cc.map((a) => a.email))
      setShowCcBcc(replyTo.cc.length > 0)
      setBcc([])
      setSubject(`RE: ${replyTo.subject.replace(/^(RE: |FW: )+/i, '')}`)
      setBody(
        buildReplyHtml(
          signature,
          `Am ${replyTo.date} schrieb ${replyTo.from.name || replyTo.from.email}:`,
          replyTo.body_html || `<p>${replyTo.body_text || ''}</p>`,
        ),
      )
    } else if (mode === 'forward' && replyTo) {
      setTo([])
      setCc([])
      setBcc([])
      setSubject(`FW: ${replyTo.subject.replace(/^(RE: |FW: )+/i, '')}`)
      setBody(
        buildForwardHtml(
          signature,
          `Weitergeleitete Nachricht von ${replyTo.from.name || ''} &lt;${replyTo.from.email}&gt; am ${replyTo.date}:`,
          replyTo.body_html || `<p>${replyTo.body_text || ''}</p>`,
        ),
      )
    } else {
      setTo(prefillTo ? [prefillTo] : [])
      setCc([])
      setBcc([])
      setSubject('')
      setBody(buildSignatureHtml(signature))
      setShowCcBcc(false)
    }
    setToInput('')
    setCcInput('')
    setBccInput('')
    setEditorVersion((v) => v + 1)
  }, [open, mode, replyTo, prefillTo, signature])

  const addRecipient = (
    email: string,
    setter: React.Dispatch<React.SetStateAction<string[]>>,
    inputSetter: React.Dispatch<React.SetStateAction<string>>,
  ) => {
    setter((prev) => [...prev, email])
    inputSetter('')
  }

  const removeRecipient = (
    email: string,
    setter: React.Dispatch<React.SetStateAction<string[]>>,
  ) => {
    setter((prev) => prev.filter((e) => e !== email))
  }

  const toAddresses = (emails: string[]): EmailAddress[] =>
    emails.map((e) => ({ name: contactMap.get(e) || '', email: e }))

  const handleSend = () => {
    if (to.length === 0 || !subject.trim()) return

    const bodyText = stripHtml(body)

    if ((mode === 'reply' || mode === 'reply-all') && replyTo) {
      replyEmail.mutate(
        {
          account_id: accountId,
          original_message_id: replyTo.id,
          body_html: body,
          body_text: bodyText,
          reply_all: mode === 'reply-all',
        },
        {
          onSuccess: () => { toast.success('E-Mail gesendet'); onOpenChange(false) },
          onError: (err) => toast.error(`Senden fehlgeschlagen: ${err.message}`),
        },
      )
    } else if (mode === 'forward' && replyTo) {
      forwardEmail.mutate(
        {
          account_id: accountId,
          original_message_id: replyTo.id,
          to: toAddresses(to),
          body_html: body,
          body_text: bodyText,
        },
        {
          onSuccess: () => { toast.success('E-Mail weitergeleitet'); onOpenChange(false) },
          onError: (err) => toast.error(`Weiterleiten fehlgeschlagen: ${err.message}`),
        },
      )
    } else {
      sendEmail.mutate(
        {
          account_id: accountId,
          to: toAddresses(to),
          cc: cc.length > 0 ? toAddresses(cc) : undefined,
          bcc: bcc.length > 0 ? toAddresses(bcc) : undefined,
          subject: subject.trim(),
          body_html: body,
          body_text: bodyText,
        },
        {
          onSuccess: () => { toast.success('E-Mail gesendet'); onOpenChange(false) },
          onError: (err) => toast.error(`Senden fehlgeschlagen: ${err.message}`),
        },
      )
    }
  }

  const handleSaveDraft = () => {
    saveDraft.mutate(
      {
        account_id: accountId,
        to: toAddresses(to),
        cc: cc.length > 0 ? toAddresses(cc) : undefined,
        bcc: bcc.length > 0 ? toAddresses(bcc) : undefined,
        subject: subject.trim() || '(Kein Betreff)',
        body_html: body,
        body_text: stripHtml(body),
        in_reply_to_message_id: replyTo?.id,
      },
      {
        onSuccess: () => { toast.success('Entwurf gespeichert'); onOpenChange(false) },
        onError: (err) => toast.error(`Speichern fehlgeschlagen: ${err.message}`),
      },
    )
  }

  const handleTemplateSelect = (tmpl: { subject: string; body: string }) => {
    setSubject(tmpl.subject)
    setBody(tmpl.body)
    setEditorVersion((v) => v + 1)
  }

  const isSending =
    sendEmail.isPending || replyEmail.isPending || forwardEmail.isPending

  const modeTitle = {
    compose: 'Neue E-Mail',
    reply: 'Antworten',
    'reply-all': 'Allen antworten',
    forward: 'Weiterleiten',
  }

  const toSuggestions = filteredSuggestions(toInput, to, allContacts)
  const ccSuggestions = filteredSuggestions(ccInput, cc, allContacts)
  const bccSuggestions = filteredSuggestions(bccInput, bcc, allContacts)

  return (
    <>
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
              contactMap={contactMap}
            />

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
                  contactMap={contactMap}
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
                  contactMap={contactMap}
                />
              </>
            )}

            <div className="flex items-center gap-2">
              <Input
                placeholder="Betreff"
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
                className="flex-1 border-0 border-b border-border rounded-none px-1 focus-visible:ring-0 font-medium"
              />
              <button
                onClick={() => setTemplateOpen(true)}
                className="shrink-0 flex items-center gap-1 rounded-md px-2 py-1.5 text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                title="Vorlage einfügen"
              >
                <FileText className="h-3.5 w-3.5" />
                Vorlage
              </button>
              {aiEmailEnabled && (
                <button
                  onClick={handleAIDraft}
                  disabled={aiDraftLoading}
                  className="shrink-0 flex items-center gap-1 rounded-md px-2 py-1.5 text-xs text-primary hover:bg-primary-light transition-colors disabled:opacity-40"
                  title="KI-Entwurf generieren"
                >
                  {aiDraftLoading ? (
                    <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                  ) : (
                    <Sparkles className="h-3.5 w-3.5" />
                  )}
                  KI-Entwurf
                </button>
              )}
            </div>

            <RichTextEditor
              key={editorVersion}
              content={body}
              onChange={setBody}
              placeholder="Nachricht schreiben..."
              compact
              showFooter={false}
              minHeight="180px"
              maxHeight="350px"
              className="border-0 rounded-none"
            />
          </div>

          <div className="flex items-center justify-between pt-3 border-t border-border">
            <div className="flex items-center gap-2">
              <Button
                onClick={handleSend}
                disabled={to.length === 0 || isSending}
              >
                <Send className="mr-1.5 h-4 w-4" />
                {isSending ? 'Sende...' : 'Senden'}
              </Button>
              <Button
                variant="outline"
                size="icon"
                onClick={handleSaveDraft}
                disabled={saveDraft.isPending}
                title="Als Entwurf speichern"
              >
                <Save className="h-4 w-4" />
              </Button>
            </div>
            <div className="flex items-center gap-1">
              <button
                className="rounded p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
                title="Datei anhaengen"
              >
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

      <EmailTemplateDialog
        open={templateOpen}
        onOpenChange={setTemplateOpen}
        onSelect={handleTemplateSelect}
      />
    </>
  )
}
