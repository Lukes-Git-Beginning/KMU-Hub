/**
 * Ticket routing rules configuration.
 *
 * Reads and writes routing rules via the Helpdesk API (useRoutingRules /
 * useCreateRoutingRule / useUpdateRoutingRule / useDeleteRoutingRule).
 * Renders either as a standalone dialog (legacy) or as embedded content for
 * the helpdesk settings panel.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, Route } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  useRoutingRules,
  useCreateRoutingRule,
  useUpdateRoutingRule,
  useDeleteRoutingRule,
} from '@/api/hooks/useHelpdesk'
import type { RoutingRule } from '@/api/helpdesk-types'

interface TicketRoutingConfigProps {
  open?: boolean
  onClose?: () => void
  /** Render just the editable content (no Dialog chrome) for the settings panel. */
  embedded?: boolean
}

export function TicketRoutingConfig({ open, onClose, embedded }: TicketRoutingConfigProps) {
  const { t } = useTranslation()
  const { data: routingRulesData, isLoading } = useRoutingRules()
  const rules: RoutingRule[] = routingRulesData ?? []

  const createMut = useCreateRoutingRule()
  const updateMut = useUpdateRoutingRule()
  const deleteMut = useDeleteRoutingRule()

  const [newRuleName, setNewRuleName] = useState('')

  const handleToggleEnabled = (rule: RoutingRule) => {
    updateMut.mutate(
      { id: rule.id, name: rule.name, enabled: !rule.enabled },
      { onError: () => toast.error(t('helpdesk.routing.ruleLoadError')) },
    )
  }

  const handleDelete = (id: string) => {
    deleteMut.mutate(id, {
      onSuccess: () => toast.success(t('helpdesk.routing.ruleDeleted')),
      onError: () => toast.error(t('helpdesk.routing.ruleLoadError')),
    })
  }

  const handleAddRule = () => {
    const name = newRuleName.trim()
    if (!name) return
    createMut.mutate(
      { name, conditions: '{}', priority: rules.length },
      {
        onSuccess: () => {
          setNewRuleName('')
          toast.success(t('helpdesk.routing.ruleCreated'))
        },
        onError: () => toast.error(t('helpdesk.routing.ruleLoadError')),
      },
    )
  }

  const body = (
    <>
      <p className="text-sm text-muted-foreground mb-4">
        {t('helpdesk.routing.info')}
      </p>

      {isLoading ? (
        <p className="text-sm text-muted-foreground py-4">{t('common.loading')}</p>
      ) : (
        <div className="rounded-lg border border-border overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-secondary/30">
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.routing.active')}</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.routing.ruleName')}</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">{t('helpdesk.routing.priority')}</th>
                <th className="px-4 py-2.5 w-12"></th>
              </tr>
            </thead>
            <tbody>
              {rules.map((rule) => (
                <tr key={rule.id} className={`border-b border-border-muted last:border-0 ${!rule.enabled ? 'opacity-50' : ''}`}>
                  <td className="px-4 py-2">
                    <button
                      onClick={() => handleToggleEnabled(rule)}
                      className={`h-5 w-9 rounded-full transition-colors ${rule.enabled ? 'bg-primary' : 'bg-secondary'}`}
                    >
                      <div className={`h-4 w-4 rounded-full bg-white transition-transform ${rule.enabled ? 'translate-x-[18px]' : 'translate-x-0.5'}`} />
                    </button>
                  </td>
                  <td className="px-4 py-2 text-xs text-foreground">{rule.name}</td>
                  <td className="px-4 py-2 text-xs text-muted-foreground">{rule.priority}</td>
                  <td className="px-4 py-2">
                    <button
                      onClick={() => handleDelete(rule.id)}
                      className="rounded p-1 text-muted-foreground hover:text-destructive transition-colors"
                      disabled={deleteMut.isPending}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="mt-3 flex items-center gap-2">
        <input
          type="text"
          value={newRuleName}
          onChange={(e) => setNewRuleName(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') handleAddRule() }}
          placeholder={t('helpdesk.routing.newRulePlaceholder')}
          className="flex-1 rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
        />
        <button
          onClick={handleAddRule}
          disabled={!newRuleName.trim() || createMut.isPending}
          className="flex items-center gap-1.5 rounded-lg border border-dashed border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Plus className="h-4 w-4" /> {t('helpdesk.routing.newRule')}
        </button>
      </div>
    </>
  )

  // Embedded (settings panel): content only.
  if (embedded) {
    return <div>{body}</div>
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
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
