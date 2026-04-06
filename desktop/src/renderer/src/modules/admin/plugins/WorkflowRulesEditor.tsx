/**
 * CRUD list for plugin workflow rules.
 *
 * Shows all workflow rules with enable/disable toggle, edit dialog,
 * and delete action. Supports filtering by installation.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Plus, Pencil, Trash2, Loader2 } from 'lucide-react'
import {
  useWorkflowRules,
  useCreateWorkflowRule,
  useUpdateWorkflowRule,
  useDeleteWorkflowRule,
} from '@/api/hooks/usePlugins'
import type { WorkflowRule, CreateWorkflowRuleRequest } from '@/api/plugin-types'

interface WorkflowRulesEditorProps {
  installationId?: string
}

const EMPTY_FORM: CreateWorkflowRuleRequest = {
  installation_id: '',
  name: '',
  description: '',
  trigger_event: 'contact.created',
  conditions: {},
  actions: [],
  priority: 0,
  enabled: true,
}

export function WorkflowRulesEditor({
  installationId,
}: WorkflowRulesEditorProps) {
  const { t } = useTranslation()
  const { data: rules, isLoading } = useWorkflowRules(installationId)
  const createRule = useCreateWorkflowRule()
  const updateRule = useUpdateWorkflowRule()
  const deleteRule = useDeleteWorkflowRule()

  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<WorkflowRule | null>(null)
  const [form, setForm] = useState<CreateWorkflowRuleRequest>(EMPTY_FORM)
  const [conditionsJson, setConditionsJson] = useState('{}')
  const [actionsJson, setActionsJson] = useState('[]')

  const openCreate = () => {
    setEditingRule(null)
    setForm({ ...EMPTY_FORM, installation_id: installationId ?? '' })
    setConditionsJson('{}')
    setActionsJson('[]')
    setEditDialogOpen(true)
  }

  const openEdit = (rule: WorkflowRule) => {
    setEditingRule(rule)
    setForm({
      installation_id: rule.installation_id,
      name: rule.name,
      description: rule.description,
      trigger_event: rule.trigger_event,
      conditions: rule.conditions,
      actions: rule.actions,
      priority: rule.priority,
      enabled: rule.enabled,
    })
    setConditionsJson(JSON.stringify(rule.conditions, null, 2))
    setActionsJson(JSON.stringify(rule.actions, null, 2))
    setEditDialogOpen(true)
  }

  const handleSave = async () => {
    let conditions: Record<string, unknown> = {}
    let actions: Record<string, unknown>[] = []

    try {
      conditions = JSON.parse(conditionsJson)
    } catch {
      // Keep empty object on parse error
    }

    try {
      actions = JSON.parse(actionsJson)
    } catch {
      // Keep empty array on parse error
    }

    if (editingRule) {
      await updateRule.mutateAsync({
        id: editingRule.id,
        data: {
          name: form.name,
          description: form.description,
          trigger_event: form.trigger_event,
          conditions,
          actions,
          priority: form.priority,
          enabled: form.enabled,
        },
      })
    } else {
      await createRule.mutateAsync({
        ...form,
        conditions,
        actions,
      })
    }
    setEditDialogOpen(false)
  }

  const handleToggle = (rule: WorkflowRule) => {
    updateRule.mutate({
      id: rule.id,
      data: { enabled: !rule.enabled },
    })
  }

  const handleDelete = (id: string) => {
    deleteRule.mutate(id)
  }

  const isSaving = createRule.isPending || updateRule.isPending

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {t('admin.workflows.ruleCount', { count: rules?.length ?? 0 })}
        </p>
        <Button size="sm" onClick={openCreate}>
          <Plus className="h-3.5 w-3.5 mr-1.5" />
          {t('admin.workflows.newRule')}
        </Button>
      </div>

      {rules && rules.length > 0 ? (
        <div className="border border-border rounded-md overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('admin.workflows.name')}</TableHead>
                <TableHead>{t('admin.workflows.trigger')}</TableHead>
                <TableHead className="text-center">{t('common.actions')}</TableHead>
                <TableHead className="text-center">{t('admin.workflows.active')}</TableHead>
                <TableHead className="text-right">{t('common.edit')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rules.map((rule) => (
                <TableRow key={rule.id}>
                  <TableCell>
                    <div>
                      <p className="text-sm font-medium">{rule.name}</p>
                      {rule.description && (
                        <p className="text-xs text-muted-foreground">
                          {rule.description}
                        </p>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="text-xs font-mono">
                      {rule.trigger_event}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-center text-sm tabular-nums">
                    {rule.actions?.length ?? 0}
                  </TableCell>
                  <TableCell className="text-center">
                    <Switch
                      checked={rule.enabled}
                      onCheckedChange={() => handleToggle(rule)}
                    />
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => openEdit(rule)}
                      >
                        <Pencil className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:text-red-600"
                        onClick={() => handleDelete(rule.id)}
                        disabled={deleteRule.isPending}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">
          {t('admin.workflows.noRules')}
        </p>
      )}

      {/* Create / Edit dialog */}
      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {editingRule
                ? t('admin.workflows.editRule')
                : t('admin.workflows.newRuleTitle')}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-3">
            <div className="space-y-1.5">
              <Label className="text-sm">{t('admin.workflows.name')}</Label>
              <Input
                value={form.name}
                onChange={(e) =>
                  setForm((f) => ({ ...f, name: e.target.value }))
                }
                placeholder={t('admin.workflows.namePlaceholder')}
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-sm">{t('admin.workflows.description')}</Label>
              <Input
                value={form.description}
                onChange={(e) =>
                  setForm((f) => ({ ...f, description: e.target.value }))
                }
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label className="text-sm">{t('admin.workflows.triggerEvent')}</Label>
                <select
                  className="w-full text-sm rounded-md border border-border bg-background px-3 py-2"
                  value={form.trigger_event}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      trigger_event: e.target.value,
                    }))
                  }
                >
                  <option value="contact.created">{t('admin.workflows.triggers.contactCreated')}</option>
                  <option value="contact.updated">{t('admin.workflows.triggers.contactUpdated')}</option>
                  <option value="deal.created">{t('admin.workflows.triggers.dealCreated')}</option>
                  <option value="deal.stage_changed">{t('admin.workflows.triggers.dealStageChanged')}</option>
                  <option value="deal.won">{t('admin.workflows.triggers.dealWon')}</option>
                  <option value="deal.lost">{t('admin.workflows.triggers.dealLost')}</option>
                  <option value="invoice.created">{t('admin.workflows.triggers.invoiceCreated')}</option>
                  <option value="invoice.paid">{t('admin.workflows.triggers.invoicePaid')}</option>
                </select>
              </div>

              <div className="space-y-1.5">
                <Label className="text-sm">{t('admin.workflows.priority')}</Label>
                <Input
                  type="number"
                  value={form.priority}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      priority: Number(e.target.value),
                    }))
                  }
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label className="text-sm">{t('admin.workflows.conditionsJson')}</Label>
              <Textarea
                className="font-mono text-xs"
                rows={3}
                value={conditionsJson}
                onChange={(e) => setConditionsJson(e.target.value)}
                placeholder='{"field": "status", "operator": "eq", "value": "new"}'
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-sm">{t('admin.workflows.actionsJson')}</Label>
              <Textarea
                className="font-mono text-xs"
                rows={4}
                value={actionsJson}
                onChange={(e) => setActionsJson(e.target.value)}
                placeholder='[{"type": "send_email", "template": "welcome"}]'
              />
            </div>

            <div className="flex items-center justify-between rounded-md border border-border p-3">
              <Label className="text-sm">{t('admin.workflows.active')}</Label>
              <Switch
                checked={form.enabled}
                onCheckedChange={(v) =>
                  setForm((f) => ({ ...f, enabled: v }))
                }
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setEditDialogOpen(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              size="sm"
              onClick={handleSave}
              disabled={!form.name || isSaving}
            >
              {isSaving && (
                <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
              )}
              {editingRule ? t('common.save') : t('common.create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
