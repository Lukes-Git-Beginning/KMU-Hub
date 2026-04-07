import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Save, Mail, Server } from 'lucide-react'
import { toast } from 'sonner'
import { useSettingsStore } from '@/stores/settings'

export function MailSettingsTab() {
  const { t } = useTranslation()
  const { mail, updateMail } = useSettingsStore()

  const [imapHost, setImapHost] = useState(mail.imapHost)
  const [imapPort, setImapPort] = useState(mail.imapPort)
  const [smtpHost, setSmtpHost] = useState(mail.smtpHost)
  const [smtpPort, setSmtpPort] = useState(mail.smtpPort)
  const [username, setUsername] = useState(mail.username)
  const [signature, setSignature] = useState(mail.signature)
  const [autoReplyEnabled, setAutoReplyEnabled] = useState(mail.autoReplyEnabled)
  const [autoReplyMessage, setAutoReplyMessage] = useState(mail.autoReplyMessage)

  const handleSaveServer = () => {
    updateMail({ imapHost, imapPort, smtpHost, smtpPort, username })
    toast.success(t('settings.mail.serverSaved'))
  }

  const handleSaveSignature = () => {
    updateMail({ signature })
    toast.success(t('settings.mail.signatureSaved'))
  }

  const handleSaveAutoReply = () => {
    updateMail({ autoReplyEnabled, autoReplyMessage })
    toast.success(t('settings.mail.autoReplySaved'))
  }

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">{t('settings.mail.title')}</h2>
      <p className="text-sm text-muted-foreground mb-6">{t('settings.mail.subtitle')}</p>

      {/* Server config */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Server className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.mail.server.title')}</h3>
        </div>

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label>{t('settings.mail.username')}</Label>
            <Input value={username} onChange={(e) => setUsername(e.target.value)} />
          </div>

          <div className="grid grid-cols-[1fr_100px] gap-3">
            <div className="space-y-1.5">
              <Label>{t('settings.mail.imapServer')}</Label>
              <Input value={imapHost} onChange={(e) => setImapHost(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('settings.mail.port')}</Label>
              <Input type="number" value={imapPort} onChange={(e) => setImapPort(Number(e.target.value))} />
            </div>
          </div>

          <div className="grid grid-cols-[1fr_100px] gap-3">
            <div className="space-y-1.5">
              <Label>{t('settings.mail.smtpServer')}</Label>
              <Input value={smtpHost} onChange={(e) => setSmtpHost(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('settings.mail.port')}</Label>
              <Input type="number" value={smtpPort} onChange={(e) => setSmtpPort(Number(e.target.value))} />
            </div>
          </div>
        </div>

        <Button onClick={handleSaveServer} className="mt-4" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.mail.server.saveButton')}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* Signature */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Mail className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.mail.signature.title')}</h3>
        </div>

        <div className="space-y-3">
          <textarea
            value={signature}
            onChange={(e) => setSignature(e.target.value)}
            rows={4}
            className="w-full rounded-lg border border-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none font-mono"
            placeholder={t('settings.mail.signature.placeholder')}
          />

          <div className="rounded-lg border border-border bg-card p-3">
            <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider mb-1.5">{t('settings.mail.signature.preview')}</p>
            <div
              className="text-sm text-foreground"
              dangerouslySetInnerHTML={{ __html: signature }}
            />
          </div>
        </div>

        <Button onClick={handleSaveSignature} className="mt-4" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.mail.signature.saveButton')}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* Auto-reply */}
      <section>
        <h3 className="text-sm font-medium text-foreground mb-4">{t('settings.mail.autoReply.title')}</h3>

        <div className="flex items-center justify-between rounded-lg border border-border bg-card p-4 mb-4">
          <div>
            <p className="text-sm text-foreground">{t('settings.mail.autoReply.label')}</p>
            <p className="text-xs text-muted-foreground">{t('settings.mail.autoReply.desc')}</p>
          </div>
          <Switch checked={autoReplyEnabled} onCheckedChange={setAutoReplyEnabled} />
        </div>

        {autoReplyEnabled && (
          <div className="space-y-1.5">
            <Label>{t('settings.mail.autoReply.messageLabel')}</Label>
            <textarea
              value={autoReplyMessage}
              onChange={(e) => setAutoReplyMessage(e.target.value)}
              rows={3}
              className="w-full rounded-lg border border-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
            />
          </div>
        )}

        <Button onClick={handleSaveAutoReply} className="mt-4" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.mail.autoReply.saveButton')}
        </Button>
      </section>
    </div>
  )
}
