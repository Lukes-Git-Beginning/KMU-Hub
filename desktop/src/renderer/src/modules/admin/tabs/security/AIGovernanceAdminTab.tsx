/**
 * AIGovernanceAdminTab — Org-weite KI-Governance im Admin-Hub.
 *
 * Extrahiert aus settings/tabs/AIGovernanceTab.tsx (Tenant-Teil).
 * Enthält:
 *   - KI global an/aus (tenant-weit)
 *   - Pro-Modul-Verbote (KI darf in Modul X grundsätzlich nicht laufen)
 *   - Org-weiter KI-Activity-Log (alle User)
 *   - Provider-Auswahl (OpenAI, Anthropic, lokales Llama)
 *   - Daten-Retention für KI-Eingaben
 *
 * Gated: admin only.
 * Hinweis: Persönlicher KI-Toggle bleibt in Cosmi → Einstellungen → KI-Governance.
 * Apple-Disziplin: strukturierte Sections, kein Card-in-Card.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Brain,
  Server,
  Activity,
  Shield,
  Ban,
  Save,
} from 'lucide-react'
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
import { toast } from 'sonner'
import { useAIStore } from '@/stores/ai'

// Module that can be globally forbidden from using AI
const AI_MODULES = [
  { id: 'email', label: 'E-Mail', descKey: 'settings.ai.module.email.desc' },
  { id: 'meetings', label: 'Meetings / LiveKit', descKey: 'settings.ai.module.meetings.desc' },
  { id: 'helpdesk', label: 'Helpdesk', descKey: 'settings.ai.module.helpdesk.desc' },
  { id: 'search', label: 'Globale Suche', descKey: 'settings.ai.module.search.desc' },
  { id: 'docs', label: 'Dokumente', descKey: 'settings.ai.module.docs.desc' },
  { id: 'crm', label: 'CRM / Kontakte', descKey: 'settings.ai.module.email.desc' },
  { id: 'dialer', label: 'Dialer', descKey: 'settings.ai.module.email.desc' },
]

const AI_PROVIDERS = [
  { value: 'openai', label: 'OpenAI (GPT-4o)' },
  { value: 'anthropic', label: 'Anthropic (Claude)' },
  { value: 'local-llama', label: 'Lokales Llama (selbst-gehostet)' },
]

const RETENTION_OPTIONS = [
  { value: '7', label: '7 Tage' },
  { value: '30', label: '30 Tage' },
  { value: '90', label: '90 Tage' },
  { value: '0', label: 'Nicht speichern' },
]

export function AIGovernanceAdminTab() {
  const { t } = useTranslation()
  const { aiEnabled, setAIEnabled } = useAIStore()

  // Tenant-weite Einstellungen (lokal bis Backend kommt)
  const [tenantAIEnabled, setTenantAIEnabled] = useState(aiEnabled)
  const [moduleBans, setModuleBans] = useState<Record<string, boolean>>({})
  const [provider, setProvider] = useState('openai')
  const [dataRetention, setDataRetention] = useState('30')

  // Mock org-activity-log
  const mockOrgLog = [
    { id: '1', user: 'Anna M.', module: 'E-Mail', action: 'Antwort vorschlagen', timestamp: new Date(Date.now() - 2 * 60000).toISOString() },
    { id: '2', user: 'Bernd S.', module: 'CRM', action: 'Kontakt-Zusammenfassung', timestamp: new Date(Date.now() - 15 * 60000).toISOString() },
    { id: '3', user: 'Carla F.', module: 'Dokumente', action: 'Text generieren', timestamp: new Date(Date.now() - 42 * 60000).toISOString() },
    { id: '4', user: 'David K.', module: 'Helpdesk', action: 'Ticket klassifizieren', timestamp: new Date(Date.now() - 3600000).toISOString() },
    { id: '5', user: 'Emma W.', module: 'E-Mail', action: 'Betreff vorschlagen', timestamp: new Date(Date.now() - 7200000).toISOString() },
  ]

  const toggleBan = (moduleId: string) => {
    setModuleBans((prev) => ({ ...prev, [moduleId]: !prev[moduleId] }))
  }

  const handleSaveMaster = () => {
    setAIEnabled(tenantAIEnabled)
    toast.success(t('admin.security.ai.master', { defaultValue: 'KI-Einstellung gespeichert' }))
  }

  const handleSaveBans = () => {
    toast.success(t('admin.security.ai.moduleBans', { defaultValue: 'Modul-Verbote gespeichert' }))
  }

  const handleSaveProvider = () => {
    toast.success(t('admin.security.ai.provider', { defaultValue: 'Provider gespeichert' }))
  }

  return (
    <div className="px-6 py-6 max-w-3xl">
      <div className="mb-6">
        <h2 className="text-base font-semibold text-foreground">
          {t('admin.security.ai.title', { defaultValue: 'KI-Governance (Tenant)' })}
        </h2>
        <p className="text-sm text-muted-foreground mt-0.5">
          {t('admin.security.ai.subtitle', { defaultValue: 'Org-weite KI-Einstellungen. Persönlicher KI-Toggle ist in Cosmi → Einstellungen → KI-Governance.' })}
        </p>
      </div>

      {/* ── Master-Toggle ─────────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Brain className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('admin.security.ai.master', { defaultValue: 'KI tenant-weit' })}
          </h3>
        </div>

        <div className="flex items-center justify-between rounded-lg border border-border p-4">
          <div>
            <p className="text-sm font-medium text-foreground">
              {t('admin.security.ai.masterLabel', { defaultValue: 'KI-Funktionen in Cosmi aktivieren' })}
            </p>
            <p className="text-xs text-muted-foreground mt-0.5">
              {tenantAIEnabled
                ? t('admin.security.ai.masterOn', { defaultValue: 'KI ist org-weit aktiviert. Einzelne User können sich abmelden.' })
                : t('admin.security.ai.masterOff', { defaultValue: 'KI ist org-weit deaktiviert. Kein User kann KI nutzen.' })
              }
            </p>
          </div>
          <Switch
            id="tenant-ai-master"
            checked={tenantAIEnabled}
            onCheckedChange={setTenantAIEnabled}
          />
        </div>

        <Button size="sm" className="mt-3" onClick={handleSaveMaster}>
          <Save className="mr-1.5 h-4 w-4" />
          {t('common.save', { defaultValue: 'Speichern' })}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Pro-Modul-Verbote ─────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Ban className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('admin.security.ai.moduleBans', { defaultValue: 'Pro-Modul-Verbote' })}
          </h3>
        </div>
        <p className="text-xs text-muted-foreground mb-4">
          {t('admin.security.ai.moduleBansDesc', { defaultValue: 'Wenn ein Modul verboten ist, wird KI dort für ALLE User blockiert — unabhängig von persönlichen Einstellungen.' })}
        </p>

        <div className={`space-y-2 ${!tenantAIEnabled ? 'opacity-50 pointer-events-none' : ''}`}>
          {AI_MODULES.map((mod) => {
            const isBanned = moduleBans[mod.id] ?? false
            return (
              <div
                key={mod.id}
                className={`flex items-center justify-between rounded-lg border p-3 transition-colors ${
                  isBanned ? 'border-error/30 bg-error/5' : 'border-border'
                }`}
              >
                <div>
                  <p className="text-sm font-medium text-foreground">{mod.label}</p>
                  <p className="text-[11px] text-muted-foreground">
                    {isBanned
                      ? t('admin.security.ai.banned', { defaultValue: 'KI verboten' })
                      : t('admin.security.ai.allowed', { defaultValue: 'KI erlaubt' })
                    }
                  </p>
                </div>
                <Switch
                  id={`ban-${mod.id}`}
                  checked={isBanned}
                  onCheckedChange={() => toggleBan(mod.id)}
                  aria-label={`KI in ${mod.label} verbieten`}
                />
              </div>
            )
          })}
        </div>

        <Button size="sm" className="mt-4" onClick={handleSaveBans} disabled={!tenantAIEnabled}>
          <Save className="mr-1.5 h-4 w-4" />
          {t('admin.security.ai.saveModuleBans', { defaultValue: 'Modul-Verbote speichern' })}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Provider ──────────────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Server className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('admin.security.ai.provider', { defaultValue: 'KI-Provider' })}
          </h3>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label htmlFor="ai-provider" className="text-xs font-medium text-muted-foreground">
              {t('admin.security.ai.providerLabel', { defaultValue: 'Anbieter' })}
            </label>
            <Select value={provider} onValueChange={setProvider}>
              <SelectTrigger id="ai-provider" className="text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {AI_PROVIDERS.map((p) => (
                  <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <label htmlFor="ai-retention" className="text-xs font-medium text-muted-foreground">
              {t('admin.security.ai.retention', { defaultValue: 'Daten-Retention KI-Eingaben' })}
            </label>
            <Select value={dataRetention} onValueChange={setDataRetention}>
              <SelectTrigger id="ai-retention" className="text-sm tabular-nums">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {RETENTION_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>{opt.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="mt-3 rounded-lg border border-border bg-secondary/30 px-4 py-3">
          <div className="grid grid-cols-2 gap-3 text-xs">
            <div>
              <p className="text-muted-foreground">Hosting</p>
              <p className="text-foreground font-medium">EU (Frankfurt)</p>
            </div>
            <div>
              <p className="text-muted-foreground">{t('settings.ai.provider.dataSharing', { defaultValue: 'Datenweitergabe' })}</p>
              <p className="text-foreground font-medium">{t('settings.ai.provider.dataSharingValue', { defaultValue: 'Kein Training mit Kundendaten' })}</p>
            </div>
          </div>
        </div>

        <Button size="sm" className="mt-4" onClick={handleSaveProvider}>
          <Save className="mr-1.5 h-4 w-4" />
          {t('admin.security.ai.saveProvider', { defaultValue: 'Provider speichern' })}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Org-Activity-Log ──────────────────────────────── */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Activity className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('admin.security.ai.activityLog', { defaultValue: 'Org-Activity-Log' })}
          </h3>
          <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
            {mockOrgLog.length} {t('admin.security.ai.entries', { defaultValue: 'Einträge' })}
          </span>
        </div>

        <div className="rounded-lg border border-border overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-secondary/30">
                <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">Zeit</th>
                <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">User</th>
                <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">Modul</th>
                <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">Aktion</th>
              </tr>
            </thead>
            <tbody>
              {mockOrgLog.map((entry) => (
                <tr key={entry.id} className="border-b border-border-muted last:border-0 hover:bg-secondary/30 transition-colors">
                  <td className="px-3 py-2 text-xs text-muted-foreground tabular-nums whitespace-nowrap">
                    {new Date(entry.timestamp).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })}
                  </td>
                  <td className="px-3 py-2 text-xs text-foreground font-medium">{entry.user}</td>
                  <td className="px-3 py-2">
                    <span className="rounded-full bg-primary-light px-2 py-0.5 text-[10px] font-medium text-primary">
                      {entry.module}
                    </span>
                  </td>
                  <td className="px-3 py-2 text-xs text-foreground">{entry.action}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <p className="mt-2 text-[10px] text-muted-foreground">
          <Shield className="inline h-3 w-3 mr-1" />
          {t('admin.security.ai.logNote', { defaultValue: 'Log-Einträge werden nach Ablauf der Daten-Retention automatisch gelöscht.' })}
        </p>
      </section>
    </div>
  )
}
