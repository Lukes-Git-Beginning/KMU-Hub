import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Send,
  Paperclip,
  Save,
  Trash2,
  FileText,
} from 'lucide-react'
import { toast } from 'sonner'
import { takeComposeDraft, clearComposeDraft } from '@/stores/mails'
import type { ComposeMode } from '@/stores/mails'
import { useSendEmail, useSaveDraft, useReplyEmail, useForwardEmail } from '@/api/hooks/useEmail'
import type { EmailAddress } from '@/api/email-types'
import { LazyRichTextEditor as RichTextEditor } from '@/components/shared/RichTextEditor'
import { EmailTemplateDialog } from './EmailTemplateDialog'
import {
  RecipientField,
  useEmailSignature,
  useContactSuggestions,
  filteredSuggestions,
  buildContactMap,
  buildSignatureHtml,
  stripHtml,
} from './compose-shared'

export default function ComposeWindowPage() {
  const { t } = useTranslation()
  const sendEmail = useSendEmail()
  const saveDraftMutation = useSaveDraft()
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
  const [mode, setMode] = useState<ComposeMode>('compose')
  const [replyToMessageId, setReplyToMessageId] = useState<string | undefined>(
    undefined,
  )
  const [accountId, setAccountId] = useState<string>('')
  const [editorVersion, setEditorVersion] = useState(0)
  const [templateOpen, setTemplateOpen] = useState(false)

  // Load draft state on mount
   
  useEffect(() => {
    // Consumes the handover entry, so a reload of this window starts fresh
    // instead of re-applying a draft the user already sent or discarded.
    const composeDraft = takeComposeDraft()
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
      setEditorVersion((v) => v + 1)
    } else {
      // Fresh window — set default signature
      setBody(buildSignatureHtml(signature))
      setEditorVersion((v) => v + 1)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount-only: initialize form state from composeDraft/signature once
  }, [])

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

  const closeWindow = () => {
    window.electronAPI?.window.close()
  }

  const toAddresses = (emails: string[]): EmailAddress[] =>
    emails.map((e) => ({ name: contactMap.get(e) || '', email: e }))

  const handleSend = () => {
    if (to.length === 0 || !subject.trim() || !accountId) return

    const bodyText = stripHtml(body)

    if ((mode === 'reply' || mode === 'reply-all') && replyToMessageId) {
      replyEmail.mutate(
        {
          account_id: accountId,
          original_message_id: replyToMessageId,
          body_html: body,
          body_text: bodyText,
          reply_all: mode === 'reply-all',
        },
        {
          onSuccess: () => {
            toast.success(t('mails.toast.emailSent'))
            setTimeout(closeWindow, 300)
          },
          onError: (err) =>
            toast.error(t('mails.toast.sendFailed', { error: err.message })),
        },
      )
    } else if (mode === 'forward' && replyToMessageId) {
      forwardEmail.mutate(
        {
          account_id: accountId,
          original_message_id: replyToMessageId,
          to: toAddresses(to),
          body_html: body,
          body_text: bodyText,
        },
        {
          onSuccess: () => {
            toast.success(t('mails.toast.emailForwarded'))
            setTimeout(closeWindow, 300)
          },
          onError: (err) =>
            toast.error(t('mails.toast.forwardFailed', { error: err.message })),
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
          onSuccess: () => {
            toast.success(t('mails.toast.emailSent'))
            setTimeout(closeWindow, 300)
          },
          onError: (err) =>
            toast.error(t('mails.toast.sendFailed', { error: err.message })),
        },
      )
    }
  }

  const handleSaveDraft = () => {
    if (!accountId) return
    saveDraftMutation.mutate(
      {
        account_id: accountId,
        to: toAddresses(to),
        cc: cc.length > 0 ? toAddresses(cc) : undefined,
        bcc: bcc.length > 0 ? toAddresses(bcc) : undefined,
        subject: subject.trim() || t('mails.compose.noSubject'),
        body_html: body,
        body_text: stripHtml(body),
        in_reply_to_message_id: replyToMessageId,
      },
      {
        onSuccess: () => {
          toast.success(t('mails.toast.draftSaved'))
          setTimeout(closeWindow, 300)
        },
        onError: (err) =>
          toast.error(t('mails.toast.saveFailed', { error: err.message })),
      },
    )
  }

  const handleDiscard = () => {
    clearComposeDraft()
    closeWindow()
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
      <div className="flex flex-col h-screen bg-background text-foreground">
        <div
          className="flex items-center justify-between px-4 py-2 border-b border-border bg-card"
          style={{ WebkitAppRegion: 'drag' } as React.CSSProperties}
        >
          <h2 className="text-sm font-medium">{modeTitle[mode]}</h2>
          <button
            onClick={() => setTemplateOpen(true)}
            className="rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors flex items-center gap-1"
            style={{ WebkitAppRegion: 'no-drag' } as React.CSSProperties}
            title={t('mails.compose.insertTemplate')}
          >
            <FileText className="h-3.5 w-3.5" />
            {t('mails.compose.template')}
          </button>
        </div>

        <div
          className="flex-1 overflow-y-auto px-5 py-4 space-y-3"
          style={{ WebkitAppRegion: 'no-drag' } as React.CSSProperties}
        >
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
            minHeight="200px"
            maxHeight="500px"
            className="border-0 rounded-none"
          />
        </div>

        <div className="flex items-center justify-between px-5 py-3 border-t border-border bg-card">
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
              disabled={saveDraftMutation.isPending}
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
              onClick={handleDiscard}
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
