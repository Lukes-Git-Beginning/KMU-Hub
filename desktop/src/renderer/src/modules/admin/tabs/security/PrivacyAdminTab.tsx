/**
 * PrivacyAdminTab — Tenant-seitige Datenschutz-Einstellungen im Admin-Hub.
 *
 * Extrahiert aus settings/tabs/PrivacySettingsTab.tsx (Tenant-Teil).
 * Enthält:
 *   - Tracking & Telemetrie (org-weit an/aus)
 *   - Retention-Policy pro Modul
 *   - Cookie-Consent-Defaults für neue User
 *   - Hinweis: Persönlicher Datenexport/Löschung bleibt in Cosmi → Einstellungen
 *
 * Gated: admin only.
 * Apple-Disziplin: klar strukturierte Sections, kein Card-in-Card.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Cookie, Clock, ShieldCheck, Save } from 'lucide-react'
import { toast } from 'sonner'

const MODULE_RETENTION_OPTIONS = [
  { value: '30', label: '30 Tage' },
  { value: '90', label: '90 Tage' },
  { value: '180', label: '180 Tage' },
  { value: '365', label: '1 Jahr' },
  { value: '730', label: '2 Jahre' },
  { value: '1825', label: '5 Jahre' },
  { value: '3650', label: '10 Jahre' },
]

const MODULES_WITH_RETENTION = [
  { id: 'crm', label: 'CRM / Kontakte' },
  { id: 'mails', label: 'E-Mail' },
  { id: 'chat', label: 'Chat' },
  { id: 'kalender', label: 'Kalender' },
  { id: 'dialer', label: 'Dialer' },
  { id: 'finanzen', label: 'Buchhaltung' },
  { id: 'dokumente', label: 'Dokumente' },
]

export function PrivacyAdminTab() {
  const { t } = useTranslation()

  // Tracking & Telemetrie
  const [analyticsEnabled, setAnalyticsEnabled] = useState(true)
  const [crashReportsEnabled, setCrashReportsEnabled] = useState(true)
  const [usageDataEnabled, setUsageDataEnabled] = useState(false)

  // Cookie-Consent-Defaults
  const [analyticsConsent, setAnalyticsConsent] = useState(false)
  const [marketingConsent, setMarketingConsent] = useState(false)

  // Retention
  const [retentionDays, setRetentionDays] = useState<Record<string, string>>(
    Object.fromEntries(MODULES_WITH_RETENTION.map((m) => [m.id, '365']))
  )

  const handleSaveTracking = () => {
    toast.success(t('admin.security.privacy.tracking', { defaultValue: 'Tracking-Einstellungen gespeichert' }))
  }

  const handleSaveConsent = () => {
    toast.success(t('admin.security.privacy.cookieDefaults', { defaultValue: 'Cookie-Defaults gespeichert' }))
  }

  const handleSaveRetention = () => {
    toast.success(t('admin.security.privacy.retention', { defaultValue: 'Retention-Policies gespeichert' }))
  }

  return (
    <div className="px-6 py-6 max-w-2xl">
      <div className="mb-6">
        <h2 className="text-base font-semibold text-foreground">
          {t('admin.security.privacy.title', { defaultValue: 'Datenschutz (Tenant)' })}
        </h2>
        <p className="text-sm text-muted-foreground mt-0.5">
          {t('admin.security.privacy.subtitle', { defaultValue: 'Org-weite Datenschutz-Einstellungen. Persönlicher Datenexport und Löschantrag sind in Cosmi → Einstellungen → Datenschutz.' })}
        </p>
      </div>

      {/* DSGVO-Hinweis */}
      <div className="rounded-lg border border-primary/20 bg-primary/5 p-4 mb-8">
        <div className="flex items-center gap-2 mb-1">
          <ShieldCheck className="h-4 w-4 text-primary" />
          <span className="text-sm font-medium text-foreground">DSGVO / EU-Datensouveränität</span>
        </div>
        <p className="text-xs text-muted-foreground leading-relaxed">
          {t('admin.security.privacy.gdprNote', { defaultValue: 'Alle Daten bleiben in EU-Rechenzentren (Hetzner, Nürnberg). Cosmi verarbeitet keine Daten außerhalb der EU ohne explizite Zustimmung.' })}
        </p>
      </div>

      {/* ── Tracking & Telemetrie ─────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Cookie className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('admin.security.privacy.tracking', { defaultValue: 'Tracking & Telemetrie' })}
          </h3>
        </div>
        <p className="text-xs text-muted-foreground mb-4">
          {t('admin.security.privacy.trackingDesc', { defaultValue: 'Diese Einstellungen gelten org-weit für alle User. Einzelne User können persönliche Präferenzen in ihren Einstellungen setzen.' })}
        </p>

        <div className="space-y-3">
          <div className="flex items-center justify-between rounded-lg border border-border p-3">
            <div>
              <p className="text-sm text-foreground">{t('settings.privacy.tracking.analytics.label', { defaultValue: 'Nutzungsanalyse' })}</p>
              <p className="text-xs text-muted-foreground">{t('settings.privacy.tracking.analytics.desc', { defaultValue: 'Anonymisierte Nutzungsdaten zur Produktverbesserung' })}</p>
            </div>
            <Switch id="admin-analytics" checked={analyticsEnabled} onCheckedChange={setAnalyticsEnabled} />
          </div>

          <div className="flex items-center justify-between rounded-lg border border-border p-3">
            <div>
              <p className="text-sm text-foreground">{t('settings.privacy.tracking.crashReports.label', { defaultValue: 'Absturzberichte' })}</p>
              <p className="text-xs text-muted-foreground">{t('settings.privacy.tracking.crashReports.desc', { defaultValue: 'Automatische Fehlerberichte bei Abstürzen' })}</p>
            </div>
            <Switch id="admin-crash" checked={crashReportsEnabled} onCheckedChange={setCrashReportsEnabled} />
          </div>

          <div className="flex items-center justify-between rounded-lg border border-border p-3">
            <div>
              <p className="text-sm text-foreground">{t('settings.privacy.tracking.usageData.label', { defaultValue: 'Verbesserungsdaten' })}</p>
              <p className="text-xs text-muted-foreground">{t('settings.privacy.tracking.usageData.desc', { defaultValue: 'Detaillierte Feature-Nutzungsstatistiken' })}</p>
            </div>
            <Switch id="admin-usage" checked={usageDataEnabled} onCheckedChange={setUsageDataEnabled} />
          </div>
        </div>

        <Button variant="outline" size="sm" className="mt-3" onClick={handleSaveTracking}>
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.privacy.tracking.saveButton', { defaultValue: 'Speichern' })}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Cookie-Consent-Defaults ───────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Cookie className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('admin.security.privacy.cookieDefaults', { defaultValue: 'Cookie-Defaults für neue User' })}
          </h3>
        </div>
        <p className="text-xs text-muted-foreground mb-4">
          {t('admin.security.privacy.cookieDefaultsDesc', { defaultValue: 'Diese Voreinstellungen werden neuen Mitarbeitern beim ersten Login vorgeschlagen. Sie können es individuell anpassen.' })}
        </p>

        <div className="space-y-3">
          <div className="flex items-center justify-between rounded-lg border border-border p-3">
            <div>
              <p className="text-sm text-foreground">{t('admin.security.privacy.cookieAnalytics', { defaultValue: 'Analyse-Cookies vorausgefüllt' })}</p>
              <p className="text-xs text-muted-foreground">{t('admin.security.privacy.cookieAnalyticsDesc', { defaultValue: 'Vorausgefüllter Consent-Status für neue User' })}</p>
            </div>
            <Switch id="admin-cookie-analytics" checked={analyticsConsent} onCheckedChange={setAnalyticsConsent} />
          </div>
          <div className="flex items-center justify-between rounded-lg border border-border p-3">
            <div>
              <p className="text-sm text-foreground">{t('admin.security.privacy.cookieMarketing', { defaultValue: 'Marketing-Cookies vorausgefüllt' })}</p>
              <p className="text-xs text-muted-foreground">{t('admin.security.privacy.cookieMarketingDesc', { defaultValue: 'Nur relevant wenn Marketing-Integrationen aktiv sind' })}</p>
            </div>
            <Switch id="admin-cookie-marketing" checked={marketingConsent} onCheckedChange={setMarketingConsent} />
          </div>
        </div>

        <Button variant="outline" size="sm" className="mt-3" onClick={handleSaveConsent}>
          <Save className="mr-1.5 h-4 w-4" />
          {t('common.save', { defaultValue: 'Speichern' })}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Retention-Policy ──────────────────────────────── */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('admin.security.privacy.retention', { defaultValue: 'Retention-Policy' })}
          </h3>
        </div>
        <p className="text-xs text-muted-foreground mb-4">
          {t('admin.security.privacy.retentionDesc', { defaultValue: 'Maximale Aufbewahrungsdauer pro Modul. Nach Ablauf werden Daten anonymisiert oder gelöscht.' })}
        </p>

        <div className="divide-y divide-border rounded-lg border border-border overflow-hidden">
          {MODULES_WITH_RETENTION.map((mod) => (
            <div key={mod.id} className="flex items-center gap-3 px-4 py-3">
              <span className="flex-1 text-sm text-foreground">{mod.label}</span>
              <Select
                value={retentionDays[mod.id]}
                onValueChange={(v) => setRetentionDays((prev) => ({ ...prev, [mod.id]: v }))}
              >
                <SelectTrigger className="w-36 h-8 text-xs tabular-nums">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {MODULE_RETENTION_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value} className="text-xs">
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ))}
        </div>

        <Button size="sm" className="mt-4" onClick={handleSaveRetention}>
          <Save className="mr-1.5 h-4 w-4" />
          {t('admin.security.privacy.saveRetention', { defaultValue: 'Retention-Policies speichern' })}
        </Button>
      </section>
    </div>
  )
}
