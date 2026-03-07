/**
 * Ticket routing rules configuration panel.
 *
 * Auto-assigns tickets to agents based on category.
 */
import { useState } from 'react'
import { X, Plus, Trash2, Route, ChevronDown } from 'lucide-react'
import { toast } from 'sonner'

export interface RoutingRule {
  id: string
  category: string
  assignTo: string
  priorityOverride?: string
  active: boolean
}

// eslint-disable-next-line react-refresh/only-export-components
export const TICKET_CATEGORIES = [
  'Netzwerk', 'Hardware', 'Software', 'Zugang',
  'E-Mail', 'Telefonie', 'Sicherheit', 'Sonstiges',
] as const

const AGENTS = ['Marco Hartmann', 'Sandra Buerki'] as const

const INITIAL_RULES: RoutingRule[] = [
  { id: 'rr-1', category: 'Netzwerk', assignTo: 'Marco Hartmann', active: true },
  { id: 'rr-2', category: 'Hardware', assignTo: 'Sandra Buerki', active: true },
  { id: 'rr-3', category: 'Software', assignTo: 'Marco Hartmann', active: true },
  { id: 'rr-4', category: 'Sicherheit', assignTo: 'Marco Hartmann', priorityOverride: 'high', active: true },
  { id: 'rr-5', category: 'Zugang', assignTo: 'Sandra Buerki', active: true },
  { id: 'rr-6', category: 'E-Mail', assignTo: 'Sandra Buerki', active: true },
  { id: 'rr-7', category: 'Telefonie', assignTo: 'Sandra Buerki', active: true },
  { id: 'rr-8', category: 'Sonstiges', assignTo: 'Marco Hartmann', active: false },
]

const PRIORITY_LABELS: Record<string, string> = {
  '': 'Keine Aenderung', low: 'Niedrig', medium: 'Mittel', high: 'Hoch', critical: 'Kritisch',
}

interface TicketRoutingConfigProps { open: boolean; onClose: () => void }

export function TicketRoutingConfig({ open, onClose }: TicketRoutingConfigProps) {
  const [rules, setRules] = useState<RoutingRule[]>(INITIAL_RULES)
  if (!open) return null

  const toggleActive = (id: string) => setRules((p) => p.map((r) => r.id === id ? { ...r, active: !r.active } : r))
  const updateRule = (id: string, field: string, value: string) => setRules((p) => p.map((r) => r.id === id ? { ...r, [field]: value } : r))
  const deleteRule = (id: string) => setRules((p) => p.filter((r) => r.id !== id))
  const addRule = () => setRules((p) => [...p, { id: `rr-${Date.now()}`, category: 'Sonstiges', assignTo: 'Marco Hartmann', active: true }])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div className="w-full max-w-3xl max-h-[85vh] overflow-hidden rounded-xl bg-card border border-border shadow-xl flex flex-col" onClick={(e) => e.stopPropagation()}>
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-6 py-4 shrink-0">
          <div className="flex items-center gap-2">
            <Route className="h-4 w-4 text-primary" />
            <h2 className="text-base font-semibold text-foreground">Ticket-Routing konfigurieren</h2>
          </div>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5">
          <p className="text-sm text-muted-foreground mb-4">
            Tickets werden basierend auf der Kategorie automatisch dem zustaendigen Agenten zugewiesen.
          </p>
          <div className="rounded-lg border border-border overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-secondary/30">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Aktiv</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Kategorie</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Zuweisen an</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Prioritaet</th>
                  <th className="px-4 py-2.5 w-12"></th>
                </tr>
              </thead>
              <tbody>
                {rules.map((rule) => (
                  <tr key={rule.id} className={`border-b border-border-muted last:border-0 ${!rule.active ? 'opacity-50' : ''}`}>
                    <td className="px-4 py-2">
                      <button onClick={() => toggleActive(rule.id)} className={`h-5 w-9 rounded-full transition-colors ${rule.active ? 'bg-primary' : 'bg-secondary'}`}>
                        <div className={`h-4 w-4 rounded-full bg-white transition-transform ${rule.active ? 'translate-x-[18px]' : 'translate-x-0.5'}`} />
                      </button>
                    </td>
                    <td className="px-4 py-2">
                      <div className="relative">
                        <select value={rule.category} onChange={(e) => updateRule(rule.id, 'category', e.target.value)} className="appearance-none rounded border border-border bg-card px-2 pr-7 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring cursor-pointer">
                          {TICKET_CATEGORIES.map((c) => <option key={c} value={c}>{c}</option>)}
                        </select>
                        <ChevronDown className="pointer-events-none absolute right-1.5 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
                      </div>
                    </td>
                    <td className="px-4 py-2">
                      <div className="relative">
                        <select value={rule.assignTo} onChange={(e) => updateRule(rule.id, 'assignTo', e.target.value)} className="appearance-none rounded border border-border bg-card px-2 pr-7 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring cursor-pointer">
                          {AGENTS.map((a) => <option key={a} value={a}>{a}</option>)}
                        </select>
                        <ChevronDown className="pointer-events-none absolute right-1.5 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
                      </div>
                    </td>
                    <td className="px-4 py-2">
                      <div className="relative">
                        <select value={rule.priorityOverride ?? ''} onChange={(e) => updateRule(rule.id, 'priorityOverride', e.target.value)} className="appearance-none rounded border border-border bg-card px-2 pr-7 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring cursor-pointer">
                          {Object.entries(PRIORITY_LABELS).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
                        </select>
                        <ChevronDown className="pointer-events-none absolute right-1.5 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
                      </div>
                    </td>
                    <td className="px-4 py-2">
                      <button onClick={() => deleteRule(rule.id)} className="rounded p-1 text-muted-foreground hover:text-destructive transition-colors">
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <button onClick={addRule} className="mt-3 flex items-center gap-1.5 rounded-lg border border-dashed border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors">
            <Plus className="h-4 w-4" /> Neue Regel
          </button>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 border-t border-border px-6 py-4 shrink-0">
          <button onClick={onClose} className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors">Abbrechen</button>
          <button onClick={() => { toast.success('Routing-Regeln gespeichert'); onClose() }} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">Speichern</button>
        </div>
      </div>
    </div>
  )
}
