/**
 * Ticket routing rules configuration.
 *
 * Auto-assigns tickets to agents based on category. Reads/writes the helpdesk
 * store (saveRoutingRules). Renders either as a standalone dialog (legacy) or as
 * embedded content for the helpdesk settings panel.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, Route, ChevronDown } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { useHelpdeskStore, type RoutingRule, MOCK_CATEGORIES } from '@/stores/helpdesk'

const AGENTS = ['Marco Hartmann', 'Sandra Bürki'] as const

// Monotonic suffix for new rule ids within a session.
let _ruleSeq = 0

interface TicketRoutingConfigProps {
  open?: boolean
  onClose?: () => void
  /** Render just the editable content (no Dialog chrome) for the settings panel. */
  embedded?: boolean
}

export function TicketRoutingConfig({ open, onClose, embedded }: TicketRoutingConfigProps) {
  const { t } = useTranslation()
  const storeRules = useHelpdeskStore((s) => s.routingRules)
  const saveRoutingRules = useHelpdeskStore((s) => s.saveRoutingRules)
  const [rules, setRules] = useState<RoutingRule[]>(() => storeRules.map((r) => ({ ...r })))

  const PRIORITY_LABELS: Record<string, string> = {
    '': t('helpdesk.routing.noChange'), low: t('helpdesk.priority.low'), medium: t('helpdesk.priority.medium'), high: t('helpdesk.priority.high'), critical: t('helpdesk.priority.critical'),
  }

  const toggleActive = (id: string) => setRules((p) => p.map((r) => r.id === id ? { ...r, active: !r.active } : r))
  const updateRule = (id: string, field: 'category' | 'assignTo' | 'priorityOverride', value: string) =>
    setRules((p) => p.map((r) => {
      if (r.id !== id) return r
      if (field === 'priorityOverride') return { ...r, priorityOverride: (value || undefined) as RoutingRule['priorityOverride'] }
      return { ...r, [field]: value }
    }))
  const deleteRule = (id: string) => setRules((p) => p.filter((r) => r.id !== id))
  const addRule = () => { _ruleSeq += 1; setRules((p) => [...p, { id: `rr-new-${_ruleSeq}`, category: 'Sonstiges', assignTo: 'Marco Hartmann', active: true }]) }

  const handleSave = () => {
    saveRoutingRules(rules)
    toast.success(t('helpdesk.routing.saved'))
    onClose?.()
  }

  const body = (
    <>
      <p className="text-sm text-muted-foreground mb-4">
        {t('helpdesk.routing.info')}
      </p>
      <div className="rounded-lg border border-border overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-secondary/30">
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.routing.active')}</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.routing.category')}</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.routing.assignTo')}</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.routing.priority')}</th>
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
                      {MOCK_CATEGORIES.map((c) => <option key={c} value={c}>{c}</option>)}
                    </select>
                    <ChevronDown className="pointer-events-none absolute right-1.5 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
                  </div>
                </td>
                <td className="px-4 py-2">
                  <div className="relative">
                    <select value={rule.assignTo} onChange={(e) => updateRule(rule.id, 'assignTo', e.target.value)} className="appearance-none rounded border border-border bg-card px-2 pr-7 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring cursor-pointer">
                      {(AGENTS.includes(rule.assignTo as (typeof AGENTS)[number]) ? AGENTS : [rule.assignTo, ...AGENTS]).map((a) => <option key={a} value={a}>{a}</option>)}
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
        <Plus className="h-4 w-4" /> {t('helpdesk.routing.newRule')}
      </button>
    </>
  )

  // Embedded (settings panel): content + save button.
  if (embedded) {
    return (
      <div>
        {body}
        <div className="mt-4 flex justify-end">
          <button onClick={handleSave} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">{t('common.save')}</button>
        </div>
      </div>
    )
  }

  // Standalone dialog (legacy).
  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose?.() }}>
      <DialogContent className="gap-0 p-0 max-w-3xl max-h-[85vh] flex flex-col overflow-hidden">
        <DialogHeader className="border-b border-border px-6 py-4 shrink-0">
          <div className="flex items-center gap-2">
            <Route className="h-4 w-4 text-primary" />
            <DialogTitle className="text-base font-semibold text-foreground">{t('helpdesk.routing.title')}</DialogTitle>
          </div>
          <DialogDescription className="sr-only">{t('helpdesk.routing.description')}</DialogDescription>
        </DialogHeader>
        <div className="flex-1 overflow-y-auto px-6 py-5">
          {body}
        </div>
        <DialogFooter className="border-t border-border px-6 py-4 shrink-0">
          <button onClick={() => onClose?.()} className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors">{t('common.cancel')}</button>
          <button onClick={handleSave} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">{t('common.save')}</button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
