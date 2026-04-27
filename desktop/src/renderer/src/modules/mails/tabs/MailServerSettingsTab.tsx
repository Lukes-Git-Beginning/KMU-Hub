/**
 * MailServerSettingsTab — Firmen-Mail-Server-Einstellungen im Mail-Modul.
 *
 * Extrahiert aus settings/tabs/ITAdminTab.tsx (email + signature Sub-Tabs).
 * Enthält:
 *   - Firmen-IMAP/SMTP-Server-Defaults (werden von neuen Mitarbeitern geerbt)
 *   - Firmen-Signatur-Template (HTML mit Platzhaltern wie {{name}}, {{position}})
 *   - Pro-Mitarbeiter-Email-Config (Connect-Status, expandable)
 *
 * Gating: admin oder it_support.
 * Apple-Disziplin: 2-Section-Layout, kein Card-in-Card.
 */
import { useState } from 'react'
import { sanitizeHtml } from '@/lib/sanitize'
import { useTranslation } from 'react-i18next'
import {
  Server,
  Mail,
  Save,
  ChevronDown,
  ChevronRight,
  Eye,
  Pencil,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { toast } from 'sonner'
import { EMPLOYEES } from '@/mocks/mock-db'

// Mock employee list
const employees = EMPLOYEES.slice(0, 8).map((e) => ({
  id: e.id,
  name: `${e.firstName} ${e.lastName}`,
  email: e.email,
  initials: e.initials,
}))

const DEFAULT_SIGNATURE_TEMPLATE = `<div style="font-family: 'Plus Jakarta Sans', sans-serif; font-size: 13px; color: #333;">
  <p style="margin: 0; font-weight: 600;">{{name}}</p>
  <p style="margin: 0; color: #666;">{{position}} | {{department}}</p>
  <p style="margin: 8px 0 0; font-size: 12px; color: #888;">
    {{company}} · {{phone}} · {{email}}
  </p>
  <p style="margin: 4px 0 0; font-size: 11px; color: #aaa;">
    {{street}}, {{plz}} {{city}}
  </p>
</div>`

type SubSection = 'server' | 'signature'

export function MailServerSettingsTab() {
  const { t } = useTranslation()
  const [activeSection, setActiveSection] = useState<SubSection>('server')

  const sections: { key: SubSection; labelKey: string; icon: typeof Server }[] = [
    { key: 'server', labelKey: 'mails.einstellungen.serverConfig', icon: Server },
    { key: 'signature', labelKey: 'mails.einstellungen.signatureTemplate', icon: Mail },
  ]

  return (
    <div className="max-w-2xl">
      <div className="mb-6">
        <h2 className="text-base font-semibold text-foreground">
          {t('mails.einstellungen.title', { defaultValue: 'Mail-Einstellungen' })}
        </h2>
        <p className="text-sm text-muted-foreground mt-0.5">
          {t('mails.einstellungen.subtitle', { defaultValue: 'Firmen-weite Server-Defaults und Signatur-Template' })}
        </p>
      </div>

      {/* Section nav */}
      <div className="flex gap-1 mb-6 border-b border-border">
        {sections.map((sec) => {
          const Icon = sec.icon
          return (
            <button
              key={sec.key}
              onClick={() => setActiveSection(sec.key)}
              className={`flex items-center gap-1.5 px-3 py-2 text-sm border-b-2 -mb-px transition-colors ${
                activeSection === sec.key
                  ? 'border-primary text-primary font-medium'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Icon className="h-3.5 w-3.5" />
              {t(sec.labelKey, { defaultValue: sec.key })}
            </button>
          )
        })}
      </div>

      {activeSection === 'server' && <ServerDefaultsSection />}
      {activeSection === 'signature' && <SignatureTemplateSection />}
    </div>
  )
}

// ── Server-Defaults ─────────────────────────────────────────────────────────

function ServerDefaultsSection() {
  const { t } = useTranslation()

  // IMAP
  const [imapHost, setImapHost] = useState('mail.firma.de')
  const [imapPort, setImapPort] = useState('993')
  const [imapSsl, setImapSsl] = useState(true)

  // SMTP
  const [smtpHost, setSmtpHost] = useState('mail.firma.de')
  const [smtpPort, setSmtpPort] = useState('587')
  const [smtpStarttls, setSmtpStarttls] = useState(true)

  // Per-employee accounts (expandable)
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const handleSave = () => {
    toast.success(t('mails.einstellungen.serverSaved', { defaultValue: 'Server-Defaults gespeichert' }))
  }

  return (
    <div className="space-y-8">
      {/* IMAP */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Server className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">IMAP</h3>
        </div>
        <div className="space-y-3">
          <div className="grid grid-cols-[1fr_120px] gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="imap-host">
                {t('mails.einstellungen.host', { defaultValue: 'Host' })}
              </Label>
              <Input
                id="imap-host"
                value={imapHost}
                onChange={(e) => setImapHost(e.target.value)}
                placeholder="imap.firma.de"
                className="font-mono text-xs"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="imap-port">Port</Label>
              <Input
                id="imap-port"
                value={imapPort}
                onChange={(e) => setImapPort(e.target.value)}
                placeholder="993"
                className="font-mono text-xs tabular-nums"
              />
            </div>
          </div>
          <div className="flex items-center justify-between rounded-lg border border-border p-3">
            <div>
              <p className="text-sm text-foreground">SSL/TLS</p>
              <p className="text-xs text-muted-foreground">
                {t('mails.einstellungen.sslDesc', { defaultValue: 'Verschlüsselte Verbindung erzwingen' })}
              </p>
            </div>
            <Switch id="imap-ssl" checked={imapSsl} onCheckedChange={setImapSsl} />
          </div>
        </div>
      </section>

      <Separator />

      {/* SMTP */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Mail className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">SMTP</h3>
        </div>
        <div className="space-y-3">
          <div className="grid grid-cols-[1fr_120px] gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="smtp-host">
                {t('mails.einstellungen.host', { defaultValue: 'Host' })}
              </Label>
              <Input
                id="smtp-host"
                value={smtpHost}
                onChange={(e) => setSmtpHost(e.target.value)}
                placeholder="smtp.firma.de"
                className="font-mono text-xs"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="smtp-port">Port</Label>
              <Input
                id="smtp-port"
                value={smtpPort}
                onChange={(e) => setSmtpPort(e.target.value)}
                placeholder="587"
                className="font-mono text-xs tabular-nums"
              />
            </div>
          </div>
          <div className="flex items-center justify-between rounded-lg border border-border p-3">
            <div>
              <p className="text-sm text-foreground">STARTTLS</p>
              <p className="text-xs text-muted-foreground">
                {t('mails.einstellungen.starttlsDesc', { defaultValue: 'Verbindung auf TLS upgraden' })}
              </p>
            </div>
            <Switch id="smtp-starttls" checked={smtpStarttls} onCheckedChange={setSmtpStarttls} />
          </div>
        </div>
      </section>

      <Button onClick={handleSave} size="sm">
        <Save className="mr-1.5 h-4 w-4" />
        {t('mails.einstellungen.saveServer', { defaultValue: 'Server-Defaults speichern' })}
      </Button>

      <Separator />

      {/* Per-Employee */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Mail className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('mails.einstellungen.userAccounts', { defaultValue: 'Mitarbeiter-Konten' })}
          </h3>
        </div>
        <div className="divide-y divide-border rounded-lg border border-border overflow-hidden">
          {employees.map((emp) => (
            <div key={emp.id}>
              <button
                onClick={() => setExpandedId(expandedId === emp.id ? null : emp.id)}
                className="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-secondary/30 transition-colors"
              >
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-light text-xs font-medium text-primary">
                  {emp.initials}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-foreground truncate">{emp.name}</p>
                  <p className="text-xs text-muted-foreground truncate">{emp.email}</p>
                </div>
                <span className="text-[10px] font-medium text-success bg-success-light rounded-full px-2 py-0.5 shrink-0">
                  {t('mails.einstellungen.accountConnected', { defaultValue: 'Verbunden' })}
                </span>
                {expandedId === emp.id
                  ? <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
                  : <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
                }
              </button>
              {expandedId === emp.id && (
                <div className="px-4 pb-3 pt-0 border-t border-border bg-secondary/10">
                  <div className="grid grid-cols-2 gap-2 mt-3 text-xs">
                    <div>
                      <p className="text-muted-foreground">IMAP</p>
                      <p className="font-mono text-foreground">{imapHost}:{imapPort}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">SMTP</p>
                      <p className="font-mono text-foreground">{smtpHost}:{smtpPort}</p>
                    </div>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}

// ── Signatur-Template ────────────────────────────────────────────────────────

function SignatureTemplateSection() {
  const { t } = useTranslation()
  const [template, setTemplate] = useState(DEFAULT_SIGNATURE_TEMPLATE)
  const [previewMode, setPreviewMode] = useState<'edit' | 'preview'>('edit')

  const placeholders = ['{{name}}', '{{position}}', '{{department}}', '{{company}}', '{{phone}}', '{{email}}', '{{street}}', '{{plz}}', '{{city}}']

  return (
    <div className="space-y-6">
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-foreground">
            {t('mails.einstellungen.signatureTemplate', { defaultValue: 'Signatur-Template' })}
          </h3>
          <div className="flex items-center gap-1 rounded-lg border border-border p-0.5">
            <button
              onClick={() => setPreviewMode('edit')}
              className={`flex items-center gap-1 rounded-md px-2 py-1 text-xs transition-colors ${
                previewMode === 'edit' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <Pencil className="h-3 w-3" />
              {t('common.edit', { defaultValue: 'Bearbeiten' })}
            </button>
            <button
              onClick={() => setPreviewMode('preview')}
              className={`flex items-center gap-1 rounded-md px-2 py-1 text-xs transition-colors ${
                previewMode === 'preview' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <Eye className="h-3 w-3" />
              {t('common.preview', { defaultValue: 'Vorschau' })}
            </button>
          </div>
        </div>

        {previewMode === 'edit' ? (
          <textarea
            value={template}
            onChange={(e) => setTemplate(e.target.value)}
            rows={12}
            className="w-full rounded-lg border border-border bg-card px-3 py-2 font-mono text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-y"
          />
        ) : (
          <div
            className="rounded-lg border border-border bg-card px-4 py-3 min-h-[200px]"
            dangerouslySetInnerHTML={{ __html: sanitizeHtml(template) }}
          />
        )}
      </div>

      {/* Placeholder reference */}
      <div>
        <p className="text-xs font-medium text-muted-foreground mb-2">
          {t('mails.einstellungen.placeholders', { defaultValue: 'Verfügbare Platzhalter' })}
        </p>
        <div className="flex flex-wrap gap-1.5">
          {placeholders.map((ph) => (
            <code
              key={ph}
              className="rounded bg-secondary px-2 py-0.5 text-[11px] font-mono text-foreground cursor-copy"
              onClick={() => {
                navigator.clipboard.writeText(ph)
                toast.success(t('mails.einstellungen.placeholderCopied', { defaultValue: 'Platzhalter kopiert' }))
              }}
            >
              {ph}
            </code>
          ))}
        </div>
      </div>

      <Button
        size="sm"
        onClick={() => toast.success(t('mails.einstellungen.signatureSaved', { defaultValue: 'Signatur-Template gespeichert' }))}
      >
        <Save className="mr-1.5 h-4 w-4" />
        {t('mails.einstellungen.saveSignature', { defaultValue: 'Template speichern' })}
      </Button>
    </div>
  )
}
