/**
 * CustomFieldRow — Single row in the custom fields config table.
 *
 * Shows field name, type icon, required badge, and edit/delete actions.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Type,
  Hash,
  Calendar,
  ChevronDown,
  CheckSquare,
  Link2,
  GripVertical,
  Pencil,
  Trash2,
  Check,
  X,
} from 'lucide-react'
import type { CustomFieldDefinition, CustomFieldType } from '@/stores/contacts'

// ---------------------------------------------------------------------------
// Type config
// ---------------------------------------------------------------------------

const typeConfig: Record<CustomFieldType, { icon: typeof Type; labelKey: string; color: string }> = {
  text: { icon: Type, labelKey: 'kontakte.customField.type.text', color: 'text-blue-500' },
  number: { icon: Hash, labelKey: 'kontakte.customField.type.number', color: 'text-emerald-500' },
  date: { icon: Calendar, labelKey: 'kontakte.customField.type.date', color: 'text-orange-500' },
  dropdown: { icon: ChevronDown, labelKey: 'kontakte.customField.type.dropdown', color: 'text-violet-500' },
  checkbox: { icon: CheckSquare, labelKey: 'kontakte.customField.type.checkbox', color: 'text-pink-500' },
  url: { icon: Link2, labelKey: 'kontakte.customField.type.url', color: 'text-cyan-500' },
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface CustomFieldRowProps {
  field: CustomFieldDefinition
  onUpdate: (id: string, updates: Partial<CustomFieldDefinition>) => void
  onDelete: (id: string) => void
}

export function CustomFieldRow({ field, onUpdate, onDelete }: CustomFieldRowProps) {
  const { t } = useTranslation()
  const [isEditing, setIsEditing] = useState(false)
  const [editName, setEditName] = useState(field.name)
  const [editRequired, setEditRequired] = useState(field.required)
  const [editOptions, setEditOptions] = useState(field.options?.join(', ') ?? '')

  const tc = typeConfig[field.type]
  const Icon = tc.icon

  const handleSave = () => {
    if (!editName.trim()) return
    const updates: Partial<CustomFieldDefinition> = {
      name: editName.trim(),
      required: editRequired,
    }
    if (field.type === 'dropdown') {
      updates.options = editOptions
        .split(',')
        .map((o) => o.trim())
        .filter(Boolean)
    }
    onUpdate(field.id, updates)
    setIsEditing(false)
  }

  const handleCancel = () => {
    setEditName(field.name)
    setEditRequired(field.required)
    setEditOptions(field.options?.join(', ') ?? '')
    setIsEditing(false)
  }

  if (isEditing) {
    return (
      <div className="rounded-lg border border-primary/30 bg-primary/5 p-3 space-y-2">
        <div className="flex items-center gap-2">
          <Icon className={`h-4 w-4 shrink-0 ${tc.color}`} />
          <input
            type="text"
            value={editName}
            onChange={(e) => setEditName(e.target.value)}
            className="h-8 flex-1 rounded-md border border-border bg-transparent px-2 text-sm outline-none focus:border-primary"
            autoFocus
            onKeyDown={(e) => { if (e.key === 'Enter') handleSave(); if (e.key === 'Escape') handleCancel() }}
          />
          <span className="text-[10px] font-medium text-muted-foreground uppercase">{t(tc.labelKey)}</span>
        </div>

        {field.type === 'dropdown' && (
          <div className="pl-6">
            <label className="text-[11px] text-muted-foreground">{t('kontakte.customField.optionsCommaSeparated')}</label>
            <input
              type="text"
              value={editOptions}
              onChange={(e) => setEditOptions(e.target.value)}
              placeholder="Option 1, Option 2, Option 3"
              className="mt-0.5 h-8 w-full rounded-md border border-border bg-transparent px-2 text-sm outline-none focus:border-primary"
            />
          </div>
        )}

        <div className="flex items-center justify-between pl-6">
          <label className="flex items-center gap-2 text-xs text-muted-foreground cursor-pointer">
            <input
              type="checkbox"
              checked={editRequired}
              onChange={(e) => setEditRequired(e.target.checked)}
              className="rounded accent-primary"
            />
            {t('kontakte.customField.requiredField')}
          </label>
          <div className="flex items-center gap-1">
            <button onClick={handleCancel} className="rounded-md p-1.5 text-muted-foreground hover:bg-accent transition-colors">
              <X className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={handleSave}
              disabled={!editName.trim()}
              className="rounded-md p-1.5 text-primary hover:bg-primary/10 transition-colors disabled:opacity-50"
            >
              <Check className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex items-center gap-2 rounded-lg border border-border px-3 py-2.5 hover:bg-accent/30 transition-colors group">
      <GripVertical className="h-3.5 w-3.5 shrink-0 text-muted-foreground/40 cursor-grab" />
      <Icon className={`h-4 w-4 shrink-0 ${tc.color}`} />
      <span className="text-sm font-medium text-foreground flex-1">{field.name}</span>
      <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
        {t(tc.labelKey)}
      </span>
      {field.required && (
        <span className="rounded-full bg-warning-light px-2 py-0.5 text-[10px] font-medium text-warning">
          {t('kontakte.customField.required')}
        </span>
      )}
      {field.type === 'dropdown' && field.options && (
        <span className="text-[10px] text-muted-foreground">
          {t('kontakte.customField.optionsCount', { count: field.options.length })}
        </span>
      )}
      <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
        <button
          onClick={() => setIsEditing(true)}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
        >
          <Pencil className="h-3 w-3" />
        </button>
        <button
          onClick={() => onDelete(field.id)}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-error-light hover:text-error transition-colors"
        >
          <Trash2 className="h-3 w-3" />
        </button>
      </div>
    </div>
  )
}
