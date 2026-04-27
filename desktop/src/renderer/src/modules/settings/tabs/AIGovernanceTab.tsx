/**
 * AIGovernanceTab — Persönliche KI-Einstellungen.
 *
 * Phase 4: Nur noch persönlicher Inhalt.
 * Org-weite KI-Governance (Tenant-Master-Toggle, Modul-Verbote,
 * Org-Activity-Log, Provider-Auswahl) ist in Admin-Hub →
 * Sicherheit → KI-Governance (AIGovernanceAdminTab).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Bot,
  Mail,
  MessageSquare,
  Headphones,
  Search,
  FileText,
  Shield,
  Activity,
} from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import { useAIStore } from '@/stores/ai'

const MODULE_CONFIG_KEYS = [
  { key: 'email', labelKey: 'settings.ai.module.email.label', descKey: 'settings.ai.module.email.desc', icon: Mail },
  { key: 'meetings', labelKey: 'settings.ai.module.meetings.label', descKey: 'settings.ai.module.meetings.desc', icon: MessageSquare },
  { key: 'helpdesk', labelKey: 'settings.ai.module.helpdesk.label', descKey: 'settings.ai.module.helpdesk.desc', icon: Headphones },
  { key: 'search', labelKey: 'settings.ai.module.search.label', descKey: 'settings.ai.module.search.desc', icon: Search },
  { key: 'docs', labelKey: 'settings.ai.module.docs.label', descKey: 'settings.ai.module.docs.desc', icon: FileText },
] as const

export function AIGovernanceTab() {
  const { t } = useTranslation()
  const {
    aiEnabled,
    setAIEnabled,
    moduleOptOuts,
    setModuleOptOut,
    noTrainingOnData,
    logAIActivity,
    setLogAIActivity,
    activityLog,
  } = useAIStore()

  const [showAllLogs, setShowAllLogs] = useState(false)
  const visibleLogs = showAllLogs ? activityLog : activityLog.slice(0, 10)
  const enabledCount = MODULE_CONFIG_KEYS.filter((m) => !moduleOptOuts[m.key]).length

  return (
    <div className="max-w-3xl">
      <h2 className="text-foreground mb-1">
        {t('settings.ai.personal.title', { defaultValue: 'KI-Einstellungen' })}
      </h2>
      <p className="text-sm text-muted-foreground mb-6">
        {t('settings.ai.subtitle', { defaultValue: 'Steuere KI-Vorschläge auf deinem Account.' })}
      </p>

      {/* Persönlicher Master-Toggle */}
      <div className="rounded-lg border border-border bg-card p-5 mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${aiEnabled ? 'bg-primary-light' : 'bg-secondary'}`}>
              <Bot className={`h-5 w-5 ${aiEnabled ? 'text-primary' : 'text-muted-foreground'}`} />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">
                {t('settings.ai.personal.toggle', { defaultValue: 'KI-Vorschläge auf meinem Account' })}
              </p>
              <p className="text-xs text-muted-foreground">
                {aiEnabled
                  ? t('settings.ai.masterToggle.activeCount', { count: enabledCount, total: MODULE_CONFIG_KEYS.length })
                  : t('settings.ai.masterToggle.allDisabled')
                }
              </p>
            </div>
          </div>
          <Switch id="personal-ai-master" checked={aiEnabled} onCheckedChange={setAIEnabled} />
        </div>
      </div>

      {/* Pro-Modul Opt-out */}
      <div className="rounded-lg border border-border bg-card p-5 mb-6">
        <h3 className="text-sm font-medium text-foreground mb-1">
          {t('settings.ai.personal.moduleOptOut', { defaultValue: 'Pro-Modul Opt-out' })}
        </h3>
        <p className="text-xs text-muted-foreground mb-4">
          {t('settings.ai.modules.subtitle', { defaultValue: 'Wähle Module wo du keine KI-Vorschläge möchtest.' })}
        </p>

        <div className="space-y-3">
          {MODULE_CONFIG_KEYS.map((mod) => {
            const Icon = mod.icon
            const enabled = aiEnabled && !moduleOptOuts[mod.key]
            return (
              <div
                key={mod.key}
                className={`flex items-center justify-between rounded-lg border p-3 transition-colors ${
                  enabled ? 'border-primary/20 bg-primary-light/20' : 'border-border'
                } ${!aiEnabled ? 'opacity-50' : ''}`}
              >
                <div className="flex items-center gap-3">
                  <Icon className={`h-4 w-4 ${enabled ? 'text-primary' : 'text-muted-foreground'}`} />
                  <div>
                    <p className="text-sm font-medium text-foreground">{t(mod.labelKey)}</p>
                    <p className="text-[11px] text-muted-foreground">{t(mod.descKey)}</p>
                  </div>
                </div>
                <Switch
                  checked={!moduleOptOuts[mod.key]}
                  onCheckedChange={(checked) => setModuleOptOut(mod.key, !checked)}
                  disabled={!aiEnabled}
                />
              </div>
            )
          })}
        </div>
      </div>

      {/* Privacy / No-Training */}
      <div className="rounded-lg border border-border bg-card p-5 mb-6">
        <div className="flex items-center gap-2 mb-4">
          <Shield className="h-4 w-4 text-primary" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.ai.privacy.title')}</h3>
        </div>
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-foreground">{t('settings.ai.privacy.noTraining.label')}</p>
              <p className="text-[11px] text-muted-foreground">{t('settings.ai.privacy.noTraining.desc')}</p>
            </div>
            <div className="flex items-center gap-2">
              <span className="rounded-full bg-success-light px-2 py-0.5 text-[10px] font-medium text-success">
                {t('settings.ai.privacy.alwaysActive')}
              </span>
              <Switch checked={noTrainingOnData} disabled />
            </div>
          </div>
          <div className="border-t border-border pt-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-foreground">{t('settings.ai.privacy.logActivity.label')}</p>
                <p className="text-[11px] text-muted-foreground">{t('settings.ai.privacy.logActivity.desc')}</p>
              </div>
              <Switch checked={logAIActivity} onCheckedChange={setLogAIActivity} />
            </div>
          </div>
        </div>
      </div>

      {/* Meine AI-Aktivität */}
      <div className="rounded-lg border border-border bg-card p-5">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-primary" />
            <h3 className="text-sm font-medium text-foreground">
              {t('settings.ai.personal.history', { defaultValue: 'Meine AI-Aktivität' })}
            </h3>
            <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
              {t('settings.ai.activityLog.entries', { count: activityLog.length })}
            </span>
          </div>
        </div>

        {activityLog.length === 0 ? (
          <div className="py-8 text-center">
            <Bot className="mx-auto h-8 w-8 text-muted-foreground/30 mb-2" />
            <p className="text-sm text-muted-foreground">{t('settings.ai.activityLog.empty')}</p>
          </div>
        ) : (
          <>
            <div className="rounded-lg border border-border overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-secondary/30">
                    <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">{t('settings.ai.activityLog.col.timestamp')}</th>
                    <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">{t('settings.ai.activityLog.col.module')}</th>
                    <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">{t('settings.ai.activityLog.col.action')}</th>
                    <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">{t('settings.ai.activityLog.col.input')}</th>
                    <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">{t('settings.ai.activityLog.col.output')}</th>
                  </tr>
                </thead>
                <tbody>
                  {visibleLogs.map((entry) => (
                    <tr key={entry.id} className="border-b border-border-muted last:border-0 hover:bg-secondary/30 transition-colors">
                      <td className="px-3 py-2 text-xs text-muted-foreground tabular-nums whitespace-nowrap">
                        {new Date(entry.timestamp).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })}
                      </td>
                      <td className="px-3 py-2">
                        <span className="rounded-full bg-primary-light px-2 py-0.5 text-[10px] font-medium text-primary">
                          {entry.module}
                        </span>
                      </td>
                      <td className="px-3 py-2 text-xs text-foreground">{entry.action}</td>
                      <td className="px-3 py-2 text-xs text-muted-foreground max-w-[160px] truncate">{entry.inputPreview}</td>
                      <td className="px-3 py-2 text-xs text-muted-foreground max-w-[160px] truncate">{entry.outputPreview}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {activityLog.length > 10 && (
              <button
                onClick={() => setShowAllLogs(!showAllLogs)}
                className="mt-3 text-xs text-primary hover:underline"
              >
                {showAllLogs
                  ? t('settings.ai.activityLog.showLess')
                  : t('settings.ai.activityLog.showAll', { count: activityLog.length })
                }
              </button>
            )}
          </>
        )}
      </div>
    </div>
  )
}
