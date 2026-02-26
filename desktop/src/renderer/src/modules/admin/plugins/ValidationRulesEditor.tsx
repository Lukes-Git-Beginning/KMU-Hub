/**
 * CRUD list for plugin validation rules.
 *
 * Shows all validation rules with enable/disable toggle, edit dialog,
 * and delete action. Supports filtering by installation.
 */
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
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
  useValidationRules,
  useCreateValidationRule,
  useUpdateValidationRule,
  useDeleteValidationRule,
} from '@/api/hooks/usePlugins'
import type { ValidationRule, CreateValidationRuleRequest } from '@/api/plugin-types'

interface ValidationRulesEditorProps {
  installationId?: string
}

const EMPTY_FORM: CreateValidationRuleRequest = {
  installation_id: '',
  name: '',
  description: '',
  entity_type: 'contact',
  field_name: '',
  rule_type: 'required',
  rule_config: {},
  error_message: '',
  priority: 0,
  enabled: true,
}

export function ValidationRulesEditor({
  installationId,
}: ValidationRulesEditorProps) {
  const { data: rules, isLoading } = useValidationRules(installationId)
  const createRule = useCreateValidationRule()
  const updateRule = useUpdateValidationRule()
  const deleteRule = useDeleteValidationRule()

  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<ValidationRule | null>(null)
  const [form, setForm] = useState<CreateValidationRuleRequest>(EMPTY_FORM)

  const openCreate = () => {
    setEditingRule(null)
    setForm({ ...EMPTY_FORM, installation_id: installationId ?? '' })
    setEditDialogOpen(true)
  }

  const openEdit = (rule: ValidationRule) => {
    setEditingRule(rule)
    setForm({
      installation_id: rule.installation_id,
      name: rule.name,
      description: rule.description,
      entity_type: rule.entity_type,
      field_name: rule.field_name,
      rule_type: rule.rule_type,
      rule_config: rule.rule_config,
      error_message: rule.error_message,
      priority: rule.priority,
      enabled: rule.enabled,
    })
    setEditDialogOpen(true)
  }

  const handleSave = async () => {
    if (editingRule) {
      await updateRule.mutateAsync({
        id: editingRule.id,
        data: {
          name: form.name,
          description: form.description,
          entity_type: form.entity_type,
          field_name: form.field_name,
          rule_type: form.rule_type,
          rule_config: form.rule_config,
          error_message: form.error_message,
          priority: form.priority,
          enabled: form.enabled,
        },
      })
    } else {
      await createRule.mutateAsync(form)
    }
    setEditDialogOpen(false)
  }

  const handleToggle = (rule: ValidationRule) => {
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
          {rules?.length ?? 0} Validierungsregel{(rules?.length ?? 0) !== 1 ? 'n' : ''}
        </p>
        <Button size="sm" onClick={openCreate}>
          <Plus className="h-3.5 w-3.5 mr-1.5" />
          Neue Regel
        </Button>
      </div>

      {rules && rules.length > 0 ? (
        <div className="border border-border rounded-md overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Entitaet</TableHead>
                <TableHead>Feld</TableHead>
                <TableHead>Typ</TableHead>
                <TableHead className="text-center">Aktiv</TableHead>
                <TableHead className="text-right">Aktionen</TableHead>
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
                    <Badge variant="outline" className="text-xs">
                      {rule.entity_type}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm font-mono">
                    {rule.field_name}
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary" className="text-xs">
                      {rule.rule_type}
                    </Badge>
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
          Keine Validierungsregeln vorhanden.
        </p>
      )}

      {/* Create / Edit dialog */}
      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>
              {editingRule
                ? 'Validierungsregel bearbeiten'
                : 'Neue Validierungsregel'}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-3">
            <div className="space-y-1.5">
              <Label className="text-sm">Name</Label>
              <Input
                value={form.name}
                onChange={(e) =>
                  setForm((f) => ({ ...f, name: e.target.value }))
                }
                placeholder="z.B. E-Mail Pflichtfeld"
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-sm">Beschreibung</Label>
              <Input
                value={form.description}
                onChange={(e) =>
                  setForm((f) => ({ ...f, description: e.target.value }))
                }
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label className="text-sm">Entitaetstyp</Label>
                <select
                  className="w-full text-sm rounded-md border border-border bg-background px-3 py-2"
                  value={form.entity_type}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, entity_type: e.target.value }))
                  }
                >
                  <option value="contact">Kontakt</option>
                  <option value="deal">Deal</option>
                  <option value="invoice">Rechnung</option>
                  <option value="quote">Offerte</option>
                </select>
              </div>

              <div className="space-y-1.5">
                <Label className="text-sm">Feldname</Label>
                <Input
                  value={form.field_name}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, field_name: e.target.value }))
                  }
                  placeholder="z.B. email"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label className="text-sm">Regeltyp</Label>
                <select
                  className="w-full text-sm rounded-md border border-border bg-background px-3 py-2"
                  value={form.rule_type}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, rule_type: e.target.value }))
                  }
                >
                  <option value="required">Pflichtfeld</option>
                  <option value="regex">Regex</option>
                  <option value="min_length">Mindestlaenge</option>
                  <option value="max_length">Maximallaenge</option>
                  <option value="range">Wertebereich</option>
                  <option value="custom">Benutzerdefiniert</option>
                </select>
              </div>

              <div className="space-y-1.5">
                <Label className="text-sm">Prioritaet</Label>
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
              <Label className="text-sm">Fehlermeldung</Label>
              <Input
                value={form.error_message}
                onChange={(e) =>
                  setForm((f) => ({ ...f, error_message: e.target.value }))
                }
                placeholder="z.B. E-Mail ist ein Pflichtfeld"
              />
            </div>

            <div className="flex items-center justify-between rounded-md border border-border p-3">
              <Label className="text-sm">Aktiv</Label>
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
              Abbrechen
            </Button>
            <Button
              size="sm"
              onClick={handleSave}
              disabled={!form.name || !form.field_name || isSaving}
            >
              {isSaving && (
                <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
              )}
              {editingRule ? 'Speichern' : 'Erstellen'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
