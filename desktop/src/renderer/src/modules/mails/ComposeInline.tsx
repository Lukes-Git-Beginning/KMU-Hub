import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Send,
  Paperclip,
  Save,
  Trash2,
  Maximize2,
  FileText,
} from 'lucide-react'
import { toast } from 'sonner'
import { useMailsStore, type ComposeMode } from '@/stores/mails'
import { useSendEmail, useSaveDraft, useReplyEmail, useForwardEmail } from '@/api/hooks/useEmail'
import type { EmailMessageInfo, EmailAddress } from '@/api/email-types'
import { LazyRichTextEditor as RichTextEditor } from '@/components/shared/RichTextEditor'
import { EmailTemplateDialog } from './EmailTemplateDialog'
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

interface ComposeInlineProps {
  mode?: ComposeMode
  replyTo?: EmailMessageInfo | null
  prefillTo?: string
  accountId: string
  // contactId opts the send into the backend marketing-consent check (UWG §7).
  // Pass it only when composing outreach to a known CRM contact.
  contactId?: string
  onClose: () => void
}

export function ComposeInline({
  mode = 'compose',
  replyTo,
  prefillTo,
  accountId,
  contactId,
  onClose,
}: ComposeInlineProps) {
  const { t } = useTranslation()
  const { setComposeDraft } = useMailsStore()
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

  useEffect(() => {
    if (mode === 'reply' && replyTo) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setTo([replyTo.from.email])
      setCc([])
      setBcc([])
      setSubject(`RE: ${replyTo.subject.replace(/^(RE: |FW: )+/i, '')}`)
      setBody(
        buildReplyHtml(
          signature,
          t('mails.compose.replyHeader', { date: replyTo.date, sender: replyTo.from.name || replyTo.from.email }),
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
          t('mails.compose.replyHeader', { date: replyTo.date, sender: replyTo.from.name || replyTo.from.email }),
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
          t('mails.compose.forwardHeader', { name: replyTo.from.name || '', email: replyTo.from.email, date: replyTo.date }),
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
  }, [mode, replyTo, prefillTo, signature])

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
          onSuccess: () => { toast.success(t('mails.toast.emailSent')); onClose() },
          onError: (err) => toast.error(t('mails.toast.sendFailed', { error: err.message })),
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
          onSuccess: () => { toast.success(t('mails.toast.emailForwarded')); onClose() },
          onError: (err) => toast.error(t('mails.toast.forwardFailed', { error: err.message })),
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
          contact_id: contactId,
        },
        {
          onSuccess: () => { toast.success(t('mails.toast.emailSent')); onClose() },
          onError: (err) => toast.error(t('mails.toast.sendFailed', { error: err.message })),
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
        subject: subject.trim() || t('mails.compose.noSubject'),
        body_html: body,
        body_text: stripHtml(body),
        in_reply_to_message_id: replyTo?.id,
      },
      {
        onSuccess: () => { toast.success(t('mails.toast.draftSaved')); onClose() },
        onError: (err) => toast.error(t('mails.toast.saveFailed', { error: err.message })),
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

  const modeTitle: Record<ComposeMode, string> = {
    compose: t('mails.compose.newEmail'),
    reply: t('mails.compose.reply'),
    'reply-all': t('mails.compose.replyAll'),
    forward: t('mails.compose.forward'),
  }

  const toSuggestions = filteredSuggestions(toInput, to, allContacts)
  const ccSuggestions = filteredSuggestions(ccInput, cc, allContacts)
  const bccSuggestions = filteredSuggestions(bccInput, bcc, allContacts)

  return (
    <>
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-6 py-3">
          <h3 className="text-sm font-medium text-foreground">
            {modeTitle[mode]}
          </h3>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setTemplateOpen(true)}
              className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
              title={t('mails.compose.insertTemplate')}
            >
              <FileText className="h-4 w-4" />
            </button>
            <button
              onClick={() => {
                setComposeDraft({
                  to,
                  cc,
                  bcc,
                  subject,
                  body,
                  mode,
                  replyToMessageId: replyTo?.id,
                  accountId,
                })
                window.electronAPI?.compose.openWindow()
                onClose()
              }}
              className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
              title={t('mails.compose.openAsWindow')}
            >
              <Maximize2 className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* Form */}
        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-3">
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
              {t('mails.compose.addCcBcc')}
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

          <Input
            placeholder={t('mails.compose.subject')}
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            className="border-0 border-b border-border rounded-none px-1 focus-visible:ring-0 font-medium"
          />

          <RichTextEditor
            key={editorVersion}
            content={body}
            onChange={setBody}
            placeholder={t('mails.compose.messagePlaceholder')}
            compact
            showFooter={false}
            minHeight="180px"
            maxHeight="400px"
            className="border-0 rounded-none"
          />
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-3 border-t border-border">
          <div className="flex items-center gap-2">
            <Button
              onClick={handleSend}
              disabled={to.length === 0 || isSending}
            >
              <Send className="mr-1.5 h-4 w-4" />
              {isSending ? t('mails.compose.sending') : t('mails.compose.send')}
            </Button>
            <Button
              variant="outline"
              size="icon"
              onClick={handleSaveDraft}
              disabled={saveDraft.isPending}
              title={t('mails.compose.saveAsDraft')}
            >
              <Save className="h-4 w-4" />
            </Button>
          </div>
          <div className="flex items-center gap-1">
            <button
              className="rounded p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
              title={t('mails.compose.attachFile')}
            >
              <Paperclip className="h-4 w-4" />
            </button>
            <button
              onClick={onClose}
              className="rounded p-1.5 text-muted-foreground hover:text-red-500 transition-colors"
              title={t('mails.compose.discard')}
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      <EmailTemplateDialog
        open={templateOpen}
        onOpenChange={setTemplateOpen}
        onSelect={handleTemplateSelect}
      />
    </>
  )
}
