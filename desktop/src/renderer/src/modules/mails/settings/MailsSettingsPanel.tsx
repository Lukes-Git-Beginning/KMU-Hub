import { useTranslation } from 'react-i18next'
import { sanitizeHtmlStrict } from '@/lib/sanitize'
import { SlidersHorizontal, Server } from 'lucide-react'
import { ChevronDown } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { Switch } from '@/components/ui/switch'
import { useSettingsStore } from '@/stores/settings'
import { useMailPrefsStore } from '@/stores/mailPrefs'
import { useMailTenantStore } from '@/stores/mailTenant'
import { useEmailAccounts } from '@/api/hooks/useEmail'

const inputCls =
  'w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring'

function ToggleRow({
  label,
  hint,
  checked,
  onChange,
}: {
  label: string
  hint: string
  checked: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-lg border border-border bg-card px-3.5 py-3">
      <div className="min-w-0">
        <p className="text-sm text-foreground">{label}</p>
        <p className="text-xs text-muted-foreground">{hint}</p>
      </div>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  )
}

// ─── Personal ────────────────────────────────────────────────────
function PersonalSection() {
  const { t } = useTranslation()
  const { mail, updateMail } = useSettingsStore()
  const defaultAccountId = useMailPrefsStore((s) => s.defaultAccountId)
  const setDefaultAccount = useMailPrefsStore((s) => s.setDefaultAccount)
  const desktopNotifications = useMailPrefsStore((s) => s.desktopNotifications)
  const setDesktopNotifications = useMailPrefsStore((s) => s.setDesktopNotifications)
  const conversationView = useMailPrefsStore((s) => s.conversationView)
  const setConversationView = useMailPrefsStore((s) => s.setConversationView)
  const { data: accountsData } = useEmailAccounts()
  const accounts = accountsData?.accounts ?? []

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('mails.settings.personal.defaultAccount', { defaultValue: 'Standard-Konto' })}
        </label>
        <div className="relative w-72 max-w-full">
          <select
            value={defaultAccountId || accounts[0]?.id || ''}
            onChange={(e) => setDefaultAccount(e.target.value)}
            className={`${inputCls} appearance-none pr-8 cursor-pointer`}
          >
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>{a.email_address}</option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">
          {t('mails.settings.personal.defaultAccountHint', { defaultValue: 'Konto, mit dem neue E-Mails standardmäßig gesendet werden.' })}
        </p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('mails.settings.personal.signature', { defaultValue: 'Signatur' })}
        </label>
        <textarea
          value={mail.signature}
          onChange={(e) => updateMail({ signature: e.target.value })}
          rows={4}
          className={`${inputCls} resize-none font-mono`}
          placeholder={t('mails.settings.personal.signaturePlaceholder', { defaultValue: 'Mit freundlichen Grüßen…' })}
        />
        <div className="rounded-lg border border-border bg-card p-3">
          <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider mb-1.5">
            {t('mails.settings.personal.preview', { defaultValue: 'Vorschau' })}
          </p>
          <div className="text-sm text-foreground" dangerouslySetInnerHTML={{ __html: sanitizeHtmlStrict(mail.signature) }} />
        </div>
      </div>

      <ToggleRow
        label={t('mails.settings.personal.desktopNotif', { defaultValue: 'Desktop-Benachrichtigung bei neuer E-Mail' })}
        hint={t('mails.settings.personal.desktopNotifHint', { defaultValue: 'Zeigt einen Hinweis, wenn eine neue Nachricht eintrifft.' })}
        checked={desktopNotifications}
        onChange={setDesktopNotifications}
      />
      <ToggleRow
        label={t('mails.settings.personal.conversationView', { defaultValue: 'Konversationsansicht' })}
        hint={t('mails.settings.personal.conversationViewHint', { defaultValue: 'Zusammengehörige E-Mails als Thread bündeln.' })}
        checked={conversationView}
        onChange={setConversationView}
      />
    </div>
  )
}

// ─── Tenant ──────────────────────────────────────────────────────
function TenantSection() {
  const { t } = useTranslation()
  const { mail, updateMail } = useSettingsStore()
  const showRetentionBadges = useMailTenantStore((s) => s.showRetentionBadges)
  const setShowRetentionBadges = useMailTenantStore((s) => s.setShowRetentionBadges)
  const loadExternalImages = useMailTenantStore((s) => s.loadExternalImages)
  const setLoadExternalImages = useMailTenantStore((s) => s.setLoadExternalImages)

  return (
    <div className="space-y-5">
      <div className="space-y-3">
        <label className="text-sm font-medium text-foreground">
          {t('mails.settings.tenant.server', { defaultValue: 'Server (IMAP / SMTP)' })}
        </label>
        <div className="space-y-1.5">
          <span className="text-xs text-muted-foreground">{t('mails.settings.tenant.username', { defaultValue: 'Benutzername' })}</span>
          <input value={mail.username} onChange={(e) => updateMail({ username: e.target.value })} className={inputCls} />
        </div>
        <div className="grid grid-cols-[1fr_100px] gap-3">
          <div className="space-y-1.5">
            <span className="text-xs text-muted-foreground">IMAP</span>
            <input value={mail.imapHost} onChange={(e) => updateMail({ imapHost: e.target.value })} className={inputCls} />
          </div>
          <div className="space-y-1.5">
            <span className="text-xs text-muted-foreground">{t('mails.settings.tenant.port', { defaultValue: 'Port' })}</span>
            <input type="number" value={mail.imapPort} onChange={(e) => updateMail({ imapPort: Number(e.target.value) })} className={inputCls} />
          </div>
        </div>
        <div className="grid grid-cols-[1fr_100px] gap-3">
          <div className="space-y-1.5">
            <span className="text-xs text-muted-foreground">SMTP</span>
            <input value={mail.smtpHost} onChange={(e) => updateMail({ smtpHost: e.target.value })} className={inputCls} />
          </div>
          <div className="space-y-1.5">
            <span className="text-xs text-muted-foreground">{t('mails.settings.tenant.port', { defaultValue: 'Port' })}</span>
            <input type="number" value={mail.smtpPort} onChange={(e) => updateMail({ smtpPort: Number(e.target.value) })} className={inputCls} />
          </div>
        </div>
      </div>

      <ToggleRow
        label={t('mails.settings.tenant.autoReply', { defaultValue: 'Automatische Antwort' })}
        hint={t('mails.settings.tenant.autoReplyHint', { defaultValue: 'Abwesenheitsnotiz für eingehende E-Mails.' })}
        checked={mail.autoReplyEnabled}
        onChange={(v) => updateMail({ autoReplyEnabled: v })}
      />
      {mail.autoReplyEnabled && (
        <textarea
          value={mail.autoReplyMessage}
          onChange={(e) => updateMail({ autoReplyMessage: e.target.value })}
          rows={3}
          className={`${inputCls} resize-none`}
          placeholder={t('mails.settings.tenant.autoReplyPlaceholder', { defaultValue: 'Ich bin derzeit nicht erreichbar…' })}
        />
      )}

      <ToggleRow
        label={t('mails.settings.tenant.retentionBadges', { defaultValue: 'DSGVO-Aufbewahrungs-Hinweise anzeigen' })}
        hint={t('mails.settings.tenant.retentionBadgesHint', { defaultValue: 'Markiert geschäftliche E-Mails mit gesetzlicher Aufbewahrungsfrist.' })}
        checked={showRetentionBadges}
        onChange={setShowRetentionBadges}
      />
      <ToggleRow
        label={t('mails.settings.tenant.externalImages', { defaultValue: 'Externe Bilder automatisch laden' })}
        hint={t('mails.settings.tenant.externalImagesHint', { defaultValue: 'Aus Datenschutzgründen standardmäßig deaktiviert.' })}
        checked={loadExternalImages}
        onChange={setLoadExternalImages}
      />
    </div>
  )
}

/**
 * MailsSettingsPanel — the "Mail" entry of the module-settings overlay.
 * Personal prefs (default account, signature, notifications) and tenant config
 * (server, compliance) follow the shared ModuleSettingsShell scope model.
 */
export function MailsSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'mails.settings.personal.title',
      descriptionKey: 'mails.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <PersonalSection />,
    },
    {
      id: 'server',
      titleKey: 'mails.settings.tenant.title',
      descriptionKey: 'mails.settings.tenant.desc',
      scope: 'tenant',
      icon: Server,
      children: <TenantSection />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="mail"
      titleKey="mails.settings.title"
      descriptionKey="mails.settings.subtitle"
      sections={sections}
    />
  )
}
