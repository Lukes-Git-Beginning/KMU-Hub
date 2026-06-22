import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, Play, ArrowRight } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { useEmailRules, useCreateRule, useDeleteRule, useApplyRules } from '@/api/hooks/useEmail'
import type { EmailLabelInfo, EmailFolderInfo, EmailRuleInfo } from '@/api/email-types'

interface RulesDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  labels: EmailLabelInfo[]
  folders: EmailFolderInfo[]
}

const inputCls =
  'rounded-md border border-border bg-background px-2.5 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-focus-ring'

export function RulesDialog({ open, onOpenChange, labels, folders }: RulesDialogProps) {
  const { t } = useTranslation()
  const { data: rulesData } = useEmailRules()
  const rules = rulesData?.rules ?? []
  const createRule = useCreateRule()
  const deleteRule = useDeleteRule()
  const applyRules = useApplyRules()

  const [form, setForm] = useState<Omit<EmailRuleInfo, 'id'>>({
    name: '',
    field: 'subject',
    op: 'contains',
    value: '',
    action_type: 'label',
    action_target: labels[0]?.id ?? '',
  })

  const targetOptions =
    form.action_type === 'label'
      ? labels.map((l) => ({ value: l.id, label: l.name }))
      : folders.map((f) => ({ value: f.id, label: f.name }))

  const labelName = (id: string) => labels.find((l) => l.id === id)?.name ?? id
  const folderName = (id: string) => folders.find((f) => f.id === id)?.name ?? id

  const addRule = () => {
    if (!form.name.trim() || !form.value.trim() || !form.action_target) return
    createRule.mutate(form)
    setForm({ ...form, name: '', value: '' })
  }

  const handleApply = () => {
    applyRules.mutate(undefined, {
      onSuccess: (res) => toast.success(t('mails.rules.applied', { count: res.affected })),
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>{t('mails.rules.title', { defaultValue: 'Regeln & Filter' })}</DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 pr-1">
          {/* Existing rules */}
          <div className="space-y-2">
            {rules.length === 0 && (
              <p className="text-sm text-muted-foreground py-2">
                {t('mails.rules.empty', { defaultValue: 'Noch keine Regeln angelegt.' })}
              </p>
            )}
            {rules.map((rule) => (
              <div key={rule.id} className="flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-2">
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-foreground truncate">{rule.name}</p>
                  <p className="flex flex-wrap items-center gap-1 text-xs text-muted-foreground">
                    <span>
                      {t(`mails.rules.field.${rule.field}`)} {t('mails.rules.contains', { defaultValue: 'enthält' })} „{rule.value}"
                    </span>
                    <ArrowRight className="h-3 w-3" />
                    <span>
                      {rule.action_type === 'label'
                        ? `${t('mails.rules.action.label', { defaultValue: 'Label' })}: ${labelName(rule.action_target)}`
                        : `${t('mails.rules.action.move', { defaultValue: 'Verschieben' })}: ${folderName(rule.action_target)}`}
                    </span>
                  </p>
                </div>
                <button
                  onClick={() => deleteRule.mutate(rule.id)}
                  className="rounded-md p-1.5 text-muted-foreground hover:text-error hover:bg-secondary transition-colors"
                  aria-label={t('common.delete')}
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))}
          </div>

          {/* New rule form */}
          <div className="rounded-lg border border-dashed border-border p-3 space-y-2.5">
            <input
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder={t('mails.rules.namePlaceholder', { defaultValue: 'Regelname' })}
              className={`${inputCls} w-full`}
            />
            <div className="flex flex-wrap items-center gap-2 text-sm">
              <span className="text-muted-foreground">{t('mails.rules.when', { defaultValue: 'Wenn' })}</span>
              <select value={form.field} onChange={(e) => setForm({ ...form, field: e.target.value as EmailRuleInfo['field'] })} className={inputCls}>
                <option value="subject">{t('mails.rules.field.subject')}</option>
                <option value="from">{t('mails.rules.field.from')}</option>
              </select>
              <span className="text-muted-foreground">{t('mails.rules.contains', { defaultValue: 'enthält' })}</span>
              <input
                value={form.value}
                onChange={(e) => setForm({ ...form, value: e.target.value })}
                placeholder={t('mails.rules.valuePlaceholder', { defaultValue: 'Text' })}
                className={`${inputCls} flex-1 min-w-[120px]`}
              />
            </div>
            <div className="flex flex-wrap items-center gap-2 text-sm">
              <span className="text-muted-foreground">{t('mails.rules.then', { defaultValue: 'dann' })}</span>
              <select
                value={form.action_type}
                onChange={(e) => {
                  const action_type = e.target.value as EmailRuleInfo['action_type']
                  setForm({
                    ...form,
                    action_type,
                    action_target: action_type === 'label' ? labels[0]?.id ?? '' : folders[0]?.id ?? '',
                  })
                }}
                className={inputCls}
              >
                <option value="label">{t('mails.rules.action.label', { defaultValue: 'Label setzen' })}</option>
                <option value="move">{t('mails.rules.action.move', { defaultValue: 'Verschieben nach' })}</option>
              </select>
              <select value={form.action_target} onChange={(e) => setForm({ ...form, action_target: e.target.value })} className={`${inputCls} flex-1 min-w-[120px]`}>
                {targetOptions.map((o) => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
            </div>
            <Button size="sm" variant="outline" onClick={addRule} disabled={!form.name.trim() || !form.value.trim()}>
              <Plus className="h-4 w-4 mr-1.5" />
              {t('mails.rules.add', { defaultValue: 'Regel hinzufügen' })}
            </Button>
          </div>
        </div>

        {/* Footer */}
        <div className="flex justify-between gap-2 pt-3 border-t border-border">
          <Button variant="outline" onClick={handleApply} disabled={rules.length === 0 || applyRules.isPending}>
            <Play className="h-4 w-4 mr-1.5" />
            {t('mails.rules.apply', { defaultValue: 'Regeln jetzt anwenden' })}
          </Button>
          <Button onClick={() => onOpenChange(false)}>{t('common.close', { defaultValue: 'Schließen' })}</Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
