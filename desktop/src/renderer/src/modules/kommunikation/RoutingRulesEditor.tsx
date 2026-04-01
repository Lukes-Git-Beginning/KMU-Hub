/**
 * Routing rules editor with AND/OR condition builder and rule testing.
 *
 * Allows creating/editing rules with nested condition trees,
 * action configuration, and a test panel to validate rules
 * against sample messages.
 */
import { useState } from 'react'
import {
  Plus,
  Trash2,
  GripVertical,
  Play,
  ChevronDown,
  ChevronRight,
  Power,
  X,
} from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { ConfirmDialog } from '@/components/shared'
import type {
  RoutingRule,
  Condition,
  RuleAction,
  RuleActionType,
  InboxChannel,
  InboxMessage,
} from '@/api/inbox-types'
import {
  useRoutingRules,
  useCreateRoutingRule,
  useUpdateRoutingRule,
  useDeleteRoutingRule,
  useTestRoutingRule,
} from '@/api/hooks/useInbox'

// ---------------------------------------------------------------------------
// Field / operator options
// ---------------------------------------------------------------------------

const FIELD_OPTIONS = [
  { value: 'channel', label: 'Kanal' },
  { value: 'sender_name', label: 'Absender' },
  { value: 'sender_email', label: 'Absender-E-Mail' },
  { value: 'subject', label: 'Betreff' },
  { value: 'preview', label: 'Inhalt' },
  { value: 'tags', label: 'Tags' },
  { value: 'crm_contact_id', label: 'CRM-Verknüpfung' },
]

const OPERATOR_OPTIONS = [
  { value: 'equals', label: 'ist gleich' },
  { value: 'contains', label: 'enthaelt' },
  { value: 'starts_with', label: 'beginnt mit' },
  { value: 'in', label: 'ist in' },
  { value: 'exists', label: 'existiert' },
  { value: 'not_equals', label: 'ist nicht gleich' },
]

const ACTION_TYPES: { value: RuleActionType; label: string }[] = [
  { value: 'route_to_team', label: 'An Team-Postfach weiterleiten' },
  { value: 'assign_to', label: 'Person zuweisen' },
  { value: 'add_tags', label: 'Tags hinzufügen' },
  { value: 'auto_reply', label: 'Automatische Antwort' },
]

// ---------------------------------------------------------------------------
// Condition builder sub-components
// ---------------------------------------------------------------------------

function _generateId(): string {
  return Math.random().toString(36).slice(2, 10)
}

function emptyLeafCondition(): Condition {
  return { field: 'channel', operator: 'equals', value: '' }
}

function emptyAndGroup(): Condition {
  return { and: [emptyLeafCondition()] }
}

interface ConditionRowProps {
  condition: Condition
  onChange: (c: Condition) => void
  onRemove: () => void
  depth: number
}

function ConditionRow({
  condition,
  onChange,
  onRemove,
  depth,
}: ConditionRowProps) {
  // Leaf condition (has field)
  if (condition.field !== undefined) {
    return (
      <div className="flex items-center gap-2">
        <select
          value={condition.field}
          onChange={(e) => onChange({ ...condition, field: e.target.value })}
          className="rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
        >
          {FIELD_OPTIONS.map((f) => (
            <option key={f.value} value={f.value}>
              {f.label}
            </option>
          ))}
        </select>
        <select
          value={condition.operator}
          onChange={(e) => onChange({ ...condition, operator: e.target.value })}
          className="rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
        >
          {OPERATOR_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        {condition.operator !== 'exists' && (
          <input
            type="text"
            value={condition.value ?? ''}
            onChange={(e) => onChange({ ...condition, value: e.target.value })}
            placeholder="Wert"
            className="flex-1 rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
          />
        )}
        <button
          onClick={onRemove}
          className="rounded p-1 text-muted-foreground hover:text-red-500 hover:bg-secondary transition-colors"
          title="Entfernen"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>
    )
  }

  // Group condition (AND or OR)
  const isAnd = !!condition.and
  const children = isAnd ? condition.and! : condition.or!
  const groupType = isAnd ? 'and' : 'or'
  const borderColor = depth % 2 === 0 ? 'border-blue-200 dark:border-blue-800' : 'border-purple-200 dark:border-purple-800'

  const updateChild = (idx: number, child: Condition) => {
    const newChildren = [...children]
    newChildren[idx] = child
    onChange(isAnd ? { and: newChildren } : { or: newChildren })
  }

  const removeChild = (idx: number) => {
    const newChildren = children.filter((_, i) => i !== idx)
    if (newChildren.length === 0) {
      onRemove()
      return
    }
    onChange(isAnd ? { and: newChildren } : { or: newChildren })
  }

  const addCondition = () => {
    const newChildren = [...children, emptyLeafCondition()]
    onChange(isAnd ? { and: newChildren } : { or: newChildren })
  }

  const addGroup = () => {
    const newChildren = [...children, emptyAndGroup()]
    onChange(isAnd ? { and: newChildren } : { or: newChildren })
  }

  const toggleGroupType = () => {
    onChange(isAnd ? { or: children } : { and: children })
  }

  return (
    <div
      className={`rounded-lg border-l-2 pl-3 py-2 space-y-2 ${borderColor}`}
    >
      <div className="flex items-center gap-2">
        <button
          onClick={toggleGroupType}
          className="rounded-md border border-border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground hover:bg-secondary transition-colors"
        >
          {groupType === 'and' ? 'UND' : 'ODER'}
        </button>
        <div className="flex-1" />
        <button
          onClick={onRemove}
          className="rounded p-1 text-muted-foreground hover:text-red-500 hover:bg-secondary transition-colors"
          title="Gruppe entfernen"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      {children.map((child, idx) => (
        <ConditionRow
          key={idx}
          condition={child}
          onChange={(c) => updateChild(idx, c)}
          onRemove={() => removeChild(idx)}
          depth={depth + 1}
        />
      ))}

      <div className="flex gap-2">
        <button
          onClick={addCondition}
          className="flex items-center gap-1 rounded-md px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
        >
          <Plus className="h-3 w-3" />
          Bedingung
        </button>
        <button
          onClick={addGroup}
          className="flex items-center gap-1 rounded-md px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
        >
          <Plus className="h-3 w-3" />
          Gruppe
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Rule editor dialog
// ---------------------------------------------------------------------------

interface RuleEditorProps {
  rule: Partial<RoutingRule> | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

function RuleEditorDialog({ rule, open, onOpenChange }: RuleEditorProps) {
  const isEdit = !!rule?.id
  const [name, setName] = useState(rule?.name ?? '')
  const [channel, setChannel] = useState<InboxChannel | ''>(
    rule?.channel ?? '',
  )
  const [priority, setPriority] = useState(rule?.priority ?? 100)
  const [isActive, setIsActive] = useState(rule?.is_active ?? true)
  const [conditions, setConditions] = useState<Condition>(
    rule?.conditions ?? emptyAndGroup(),
  )
  const [actions, setActions] = useState<RuleAction[]>(rule?.actions ?? [])

  // Test panel
  const [testExpanded, setTestExpanded] = useState(false)
  const [testChannel, setTestChannel] = useState<InboxChannel>('email')
  const [testSender, setTestSender] = useState('')
  const [testSubject, setTestSubject] = useState('')
  const [testResult, setTestResult] = useState<boolean | null>(null)

  const createRule = useCreateRoutingRule()
  const updateRule = useUpdateRoutingRule()
  const testRule = useTestRoutingRule()

  // Sync form state when rule changes
  useState(() => {
    if (rule) {
      setName(rule.name ?? '')
      setChannel(rule.channel ?? '')
      setPriority(rule.priority ?? 100)
      setIsActive(rule.is_active ?? true)
      setConditions(rule.conditions ?? emptyAndGroup())
      setActions(rule.actions ?? [])
    }
  })

  const handleSave = () => {
    const data: Partial<RoutingRule> = {
      name,
      channel: channel || undefined,
      conditions,
      actions,
      priority,
      is_active: isActive,
    }

    if (isEdit) {
      updateRule.mutate(
        { id: rule!.id!, ...data },
        { onSuccess: () => onOpenChange(false) },
      )
    } else {
      createRule.mutate(data, { onSuccess: () => onOpenChange(false) })
    }
  }

  const handleTest = () => {
    const testMessage: Partial<InboxMessage> = {
      channel: testChannel,
      sender_name: testSender,
      subject: testSubject,
    }
    testRule.mutate(
      { conditions, message: testMessage },
      {
        onSuccess: (result) => {
          setTestResult(result.matches)
        },
      },
    )
  }

  const addAction = () => {
    setActions([
      ...actions,
      { type: 'route_to_team', config: {} },
    ])
  }

  const updateAction = (idx: number, action: RuleAction) => {
    const newActions = [...actions]
    newActions[idx] = action
    setActions(newActions)
  }

  const removeAction = (idx: number) => {
    setActions(actions.filter((_, i) => i !== idx))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? 'Regel bearbeiten' : 'Neue Regel erstellen'}
          </DialogTitle>
          <DialogDescription>
            Definiere Bedingungen und Aktionen für die automatische
            Nachrichtenverarbeitung.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Basic fields */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs font-medium text-foreground">
                Name
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div>
              <label className="text-xs font-medium text-foreground">
                Kanal-Filter
              </label>
              <select
                value={channel}
                onChange={(e) =>
                  setChannel(e.target.value as InboxChannel | '')
                }
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                <option value="">Alle Kanäle</option>
                <option value="email">E-Mail</option>
                <option value="chat">Chat</option>
                <option value="notification">Benachrichtigungen</option>
              </select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs font-medium text-foreground">
                Prioritaet
              </label>
              <input
                type="number"
                value={priority}
                onChange={(e) => setPriority(Number(e.target.value))}
                min={1}
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="flex items-end pb-1">
              <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer">
                <input
                  type="checkbox"
                  checked={isActive}
                  onChange={(e) => setIsActive(e.target.checked)}
                  className="accent-primary h-4 w-4"
                />
                Aktiv
              </label>
            </div>
          </div>

          {/* Condition builder */}
          <div>
            <label className="text-xs font-medium text-foreground">
              Bedingungen
            </label>
            <div className="mt-1">
              <ConditionRow
                condition={conditions}
                onChange={setConditions}
                onRemove={() => setConditions(emptyAndGroup())}
                depth={0}
              />
            </div>
          </div>

          {/* Actions */}
          <div>
            <label className="text-xs font-medium text-foreground">
              Aktionen
            </label>
            <div className="mt-1 space-y-2">
              {actions.map((action, idx) => (
                <div
                  key={idx}
                  className="flex items-start gap-2 rounded-md border border-border p-2"
                >
                  <select
                    value={action.type}
                    onChange={(e) =>
                      updateAction(idx, {
                        ...action,
                        type: e.target.value as RuleActionType,
                        config: {},
                      })
                    }
                    className="rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
                  >
                    {ACTION_TYPES.map((a) => (
                      <option key={a.value} value={a.value}>
                        {a.label}
                      </option>
                    ))}
                  </select>

                  {/* Config input per action type */}
                  <div className="flex-1">
                    {action.type === 'route_to_team' && (
                      <input
                        type="text"
                        value={(action.config.team_inbox_id as string) ?? ''}
                        onChange={(e) =>
                          updateAction(idx, {
                            ...action,
                            config: { team_inbox_id: e.target.value },
                          })
                        }
                        placeholder="Team-Postfach-ID"
                        className="w-full rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
                      />
                    )}
                    {action.type === 'assign_to' && (
                      <input
                        type="text"
                        value={(action.config.user_id as string) ?? ''}
                        onChange={(e) =>
                          updateAction(idx, {
                            ...action,
                            config: { user_id: e.target.value },
                          })
                        }
                        placeholder="Benutzer-ID"
                        className="w-full rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
                      />
                    )}
                    {action.type === 'add_tags' && (
                      <input
                        type="text"
                        value={(action.config.tags as string) ?? ''}
                        onChange={(e) =>
                          updateAction(idx, {
                            ...action,
                            config: { tags: e.target.value },
                          })
                        }
                        placeholder="Tags (kommagetrennt)"
                        className="w-full rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
                      />
                    )}
                    {action.type === 'auto_reply' && (
                      <textarea
                        value={(action.config.body as string) ?? ''}
                        onChange={(e) =>
                          updateAction(idx, {
                            ...action,
                            config: { body: e.target.value },
                          })
                        }
                        placeholder="Antworttext"
                        rows={2}
                        className="w-full rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring resize-none"
                      />
                    )}
                  </div>

                  <button
                    onClick={() => removeAction(idx)}
                    className="rounded p-1 text-muted-foreground hover:text-red-500 hover:bg-secondary transition-colors"
                    title="Aktion entfernen"
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
              <button
                onClick={addAction}
                className="flex items-center gap-1 rounded-md px-2 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
              >
                <Plus className="h-3.5 w-3.5" />
                Aktion hinzufügen
              </button>
            </div>
          </div>

          {/* Test panel */}
          <div className="border-t border-border pt-3">
            <button
              onClick={() => setTestExpanded(!testExpanded)}
              className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors"
            >
              {testExpanded ? (
                <ChevronDown className="h-3.5 w-3.5" />
              ) : (
                <ChevronRight className="h-3.5 w-3.5" />
              )}
              Regel testen
            </button>
            {testExpanded && (
              <div className="mt-2 space-y-2 rounded-lg border border-border p-3">
                <div className="grid grid-cols-3 gap-2">
                  <div>
                    <label className="text-[10px] text-muted-foreground">
                      Kanal
                    </label>
                    <select
                      value={testChannel}
                      onChange={(e) =>
                        setTestChannel(e.target.value as InboxChannel)
                      }
                      className="mt-0.5 w-full rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
                    >
                      <option value="email">E-Mail</option>
                      <option value="chat">Chat</option>
                      <option value="notification">Benachrichtigung</option>
                    </select>
                  </div>
                  <div>
                    <label className="text-[10px] text-muted-foreground">
                      Absender
                    </label>
                    <input
                      type="text"
                      value={testSender}
                      onChange={(e) => setTestSender(e.target.value)}
                      className="mt-0.5 w-full rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
                    />
                  </div>
                  <div>
                    <label className="text-[10px] text-muted-foreground">
                      Betreff
                    </label>
                    <input
                      type="text"
                      value={testSubject}
                      onChange={(e) => setTestSubject(e.target.value)}
                      className="mt-0.5 w-full rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
                    />
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={handleTest}
                    disabled={testRule.isPending}
                    className="flex items-center gap-1 rounded-md bg-secondary px-3 py-1.5 text-xs text-foreground hover:bg-secondary/80 transition-colors disabled:opacity-50"
                  >
                    <Play className="h-3 w-3" />
                    Testen
                  </button>
                  {testResult !== null && (
                    <span
                      className={`text-xs font-medium ${
                        testResult
                          ? 'text-green-600 dark:text-green-400'
                          : 'text-red-600 dark:text-red-400'
                      }`}
                    >
                      {testResult ? 'Trifft zu' : 'Trifft nicht zu'}
                    </span>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Save / Cancel */}
          <div className="flex justify-end gap-2 pt-2 border-t border-border">
            <button
              onClick={() => onOpenChange(false)}
              className="rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={handleSave}
              disabled={
                !name.trim() ||
                createRule.isPending ||
                updateRule.isPending
              }
              className="rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
            >
              {isEdit ? 'Speichern' : 'Erstellen'}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Main RoutingRulesEditor component
// ---------------------------------------------------------------------------

export function RoutingRulesEditor() {
  const { data: rules, isLoading } = useRoutingRules()
  const deleteRule = useDeleteRoutingRule()
  const updateRule = useUpdateRoutingRule()

  const [editorOpen, setEditorOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<Partial<RoutingRule> | null>(
    null,
  )
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)

  const handleCreate = () => {
    setEditingRule(null)
    setEditorOpen(true)
  }

  const handleEdit = (rule: RoutingRule) => {
    setEditingRule(rule)
    setEditorOpen(true)
  }

  const handleToggleActive = (rule: RoutingRule) => {
    updateRule.mutate({
      id: rule.id,
      is_active: !rule.is_active,
    })
  }

  const handleDelete = () => {
    if (deleteConfirmId) {
      deleteRule.mutate(deleteConfirmId)
      setDeleteConfirmId(null)
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-foreground">Routing-Regeln</h3>
        <button
          onClick={handleCreate}
          className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Plus className="h-3.5 w-3.5" />
          Neue Regel
        </button>
      </div>

      {/* Rules list */}
      <div className="space-y-1">
        {isLoading && (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => (
              <div
                key={i}
                className="h-12 animate-pulse rounded-md bg-secondary"
              />
            ))}
          </div>
        )}
        {rules?.map((rule) => (
          <div
            key={rule.id}
            className="flex items-center gap-3 rounded-md border border-border px-3 py-2 hover:bg-secondary/50 transition-colors"
          >
            <GripVertical className="h-4 w-4 shrink-0 text-muted-foreground" />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-foreground truncate">
                  {rule.name}
                </span>
                <span className="text-[10px] text-muted-foreground">
                  P{rule.priority}
                </span>
                {rule.channel && (
                  <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-secondary-foreground">
                    {rule.channel}
                  </span>
                )}
              </div>
              <p className="text-[10px] text-muted-foreground">
                {rule.actions.length} Aktion
                {rule.actions.length !== 1 ? 'en' : ''}
              </p>
            </div>
            <button
              onClick={() => handleToggleActive(rule)}
              className={`rounded p-1.5 transition-colors ${
                rule.is_active
                  ? 'text-green-500 hover:bg-green-50 dark:hover:bg-green-950/30'
                  : 'text-muted-foreground hover:bg-secondary'
              }`}
              title={rule.is_active ? 'Deaktivieren' : 'Aktivieren'}
            >
              <Power className="h-4 w-4" />
            </button>
            <button
              onClick={() => handleEdit(rule)}
              className="rounded-md px-2 py-1 text-xs text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
            >
              Bearbeiten
            </button>
            <button
              onClick={() => setDeleteConfirmId(rule.id)}
              className="rounded p-1.5 text-muted-foreground hover:text-red-500 hover:bg-secondary transition-colors"
              title="Löschen"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        ))}
        {!isLoading && (!rules || rules.length === 0) && (
          <p className="text-xs text-muted-foreground py-4 text-center">
            Noch keine Routing-Regeln erstellt.
          </p>
        )}
      </div>

      <RuleEditorDialog
        rule={editingRule}
        open={editorOpen}
        onOpenChange={setEditorOpen}
      />

      <ConfirmDialog
        open={!!deleteConfirmId}
        onOpenChange={(open) => !open && setDeleteConfirmId(null)}
        title="Regel löschen?"
        description="Diese Regel wird dauerhaft gelöscht."
        confirmLabel="Löschen"
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}
