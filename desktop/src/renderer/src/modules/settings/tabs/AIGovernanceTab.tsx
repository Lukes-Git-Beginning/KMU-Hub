import { useState } from 'react'
import {
  Sparkles,
  Mail,
  MessageSquare,
  Headphones,
  Search,
  FileText,
  Shield,
  Server,
  Activity,
} from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import { useAIStore } from '@/stores/ai'

const MODULE_CONFIG = [
  { key: 'email', label: 'E-Mail-Entwürfe', description: 'KI-generierte Antwortvorschläge im E-Mail-Composer', icon: Mail },
  { key: 'meetings', label: 'Meeting-Zusammenfassungen', description: 'Automatische Zusammenfassung von Meeting-Notizen', icon: MessageSquare },
  { key: 'helpdesk', label: 'Ticket-Antwortvorschläge', description: 'KI-basierte Antwortvorschläge für Helpdesk-Tickets', icon: Headphones },
  { key: 'search', label: 'Semantische Suche', description: 'Natürlichsprachliche Suche über alle Module', icon: Search },
  { key: 'docs', label: 'Dokument-Klassifizierung', description: 'Automatische Einstufung als öffentlich/intern/vertraulich', icon: FileText },
] as const

export function AIGovernanceTab() {
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

  const enabledCount = MODULE_CONFIG.filter((m) => !moduleOptOuts[m.key]).length

  return (
    <div className="max-w-3xl">
      <h2 className="text-foreground mb-1">KI-Assistent</h2>
      <p className="text-sm text-muted-foreground mb-6">
        Konfiguriere den KI-Assistenten und steuere, welche Module KI-Funktionen nutzen duerfen.
      </p>

      {/* Master toggle */}
      <div className="rounded-lg border border-border bg-card p-5 glass-surface mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${aiEnabled ? 'bg-primary-light' : 'bg-secondary'}`}>
              <Sparkles className={`h-5 w-5 ${aiEnabled ? 'text-primary' : 'text-muted-foreground'}`} />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">KI-Assistent aktivieren</p>
              <p className="text-xs text-muted-foreground">
                {aiEnabled ? `${enabledCount} von ${MODULE_CONFIG.length} Modulen aktiv` : 'Alle KI-Funktionen deaktiviert'}
              </p>
            </div>
          </div>
          <Switch checked={aiEnabled} onCheckedChange={setAIEnabled} />
        </div>
      </div>

      {/* Module opt-outs */}
      <div className="rounded-lg border border-border bg-card p-5 glass-surface mb-6">
        <h3 className="text-sm font-medium text-foreground mb-1">Modul-Einstellungen</h3>
        <p className="text-xs text-muted-foreground mb-4">
          Aktiviere oder deaktiviere KI-Funktionen für einzelne Module.
        </p>

        <div className="space-y-3">
          {MODULE_CONFIG.map((mod) => {
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
                    <p className="text-sm font-medium text-foreground">{mod.label}</p>
                    <p className="text-[11px] text-muted-foreground">{mod.description}</p>
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

      {/* Privacy section */}
      <div className="rounded-lg border border-border bg-card p-5 glass-surface mb-6">
        <div className="flex items-center gap-2 mb-4">
          <Shield className="h-4 w-4 text-primary" />
          <h3 className="text-sm font-medium text-foreground">Datenschutz</h3>
        </div>

        <div className="space-y-4">
          {/* No training — always on, read-only */}
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-foreground">Kein Training auf Kundendaten</p>
              <p className="text-[11px] text-muted-foreground">Ihre Daten werden niemals zum Training von KI-Modellen verwendet</p>
            </div>
            <div className="flex items-center gap-2">
              <span className="rounded-full bg-success-light px-2 py-0.5 text-[10px] font-medium text-success">Immer aktiv</span>
              <Switch checked={noTrainingOnData} disabled />
            </div>
          </div>

          <div className="border-t border-border pt-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-foreground">KI-Aktivitaeten protokollieren</p>
                <p className="text-[11px] text-muted-foreground">Alle KI-Anfragen und -Antworten werden im Aktivitaetslog festgehalten</p>
              </div>
              <Switch checked={logAIActivity} onCheckedChange={setLogAIActivity} />
            </div>
          </div>
        </div>
      </div>

      {/* Provider info */}
      <div className="rounded-lg border border-border bg-card p-5 glass-surface mb-6">
        <div className="flex items-center gap-2 mb-4">
          <Server className="h-4 w-4 text-primary" />
          <h3 className="text-sm font-medium text-foreground">KI-Anbieter</h3>
        </div>

        <div className="grid grid-cols-2 gap-3">
          {[
            { label: 'Modell', value: 'GPT-4o' },
            { label: 'Hosting', value: 'EU (Frankfurt)' },
            { label: 'Datenverarbeitung', value: 'Nur in der EU' },
            { label: 'Datenweitergabe', value: 'Keine an Dritte' },
          ].map((item) => (
            <div key={item.label} className="rounded-md bg-secondary/50 px-3 py-2">
              <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">{item.label}</p>
              <p className="text-sm text-foreground mt-0.5">{item.value}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Activity log */}
      <div className="rounded-lg border border-border bg-card p-5 glass-surface">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-primary" />
            <h3 className="text-sm font-medium text-foreground">Aktivitaetslog</h3>
            <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
              {activityLog.length} Einträge
            </span>
          </div>
        </div>

        {activityLog.length === 0 ? (
          <div className="py-8 text-center">
            <Sparkles className="mx-auto h-8 w-8 text-muted-foreground/30 mb-2" />
            <p className="text-sm text-muted-foreground">Noch keine KI-Aktivitaeten protokolliert</p>
          </div>
        ) : (
          <>
            <div className="rounded-lg border border-border overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-secondary/30">
                    <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">Zeitpunkt</th>
                    <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">Modul</th>
                    <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">Aktion</th>
                    <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">Eingabe</th>
                    <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">Ergebnis</th>
                  </tr>
                </thead>
                <tbody>
                  {visibleLogs.map((entry) => (
                    <tr key={entry.id} className="border-b border-border-muted last:border-0 hover:bg-secondary/30 transition-colors">
                      <td className="px-3 py-2 text-xs text-muted-foreground whitespace-nowrap">
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
                {showAllLogs ? 'Weniger anzeigen' : `Alle ${activityLog.length} Einträge anzeigen`}
              </button>
            )}
          </>
        )}
      </div>
    </div>
  )
}
