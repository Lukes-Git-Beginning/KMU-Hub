/**
 * WorkCustomFieldsManager — tenant-wide custom-field definitions for tasks.
 *
 * Embedded in the work settings panel. Define fields (name + type) applied to
 * tasks. Mock-first (stores/workSettings); backend gap tracked in backend-gaps.md.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Type, Hash, Calendar, ChevronDown, CheckSquare, Trash2, X } from 'lucide-react'
import { toast } from 'sonner'
import { useWorkSettingsStore, type WorkCustomField, type WorkFieldType } from '@/stores/workSettings'
import { ConfirmDialog } from '@/components/shared'

const FIELD_TYPES: { value: WorkFieldType; icon: typeof Type; labelKey: string }[] = [
  { value: 'text', icon: Type, labelKey: 'work.settings.fields.type.text' },
  { value: 'number', icon: Hash, labelKey: 'work.settings.fields.type.number' },
  { value: 'date', icon: Calendar, labelKey: 'work.settings.fields.type.date' },
  { value: 'dropdown', icon: ChevronDown, labelKey: 'work.settings.fields.type.dropdown' },
  { value: 'checkbox', icon: CheckSquare, labelKey: 'work.settings.fields.type.checkbox' },
]

function typeIcon(type: WorkFieldType) {
  return FIELD_TYPES.find((ft) => ft.value === type)?.icon ?? Type
}

export function WorkCustomFieldsManager() {
  const { t } = useTranslation()
  const fields = useWorkSettingsStore((s) => s.customFields)
  const addField = useWorkSettingsStore((s) => s.addCustomField)
  const removeField = useWorkSettingsStore((s) => s.removeCustomField)

  const [adding, setAdding] = useState(false)
  const [newName, setNewName] = useState('')
  const [newType, setNewType] = useState<WorkFieldType>('text')
  const [newRequired, setNewRequired] = useState(false)
  const [newOptions, setNewOptions] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<WorkCustomField | null>(null)

  const handleAdd = () => {
    const name = newName.trim()
    if (!name) return
    addField({
      name,
      type: newType,
      required: newRequired,
      options: newType === 'dropdown' ? newOptions.split(',').map((o) => o.trim()).filter(Boolean) : undefined,
    })
    toast.success(t('work.settings.fields.added', { name }))
    setNewName('')
    setNewType('text')
    setNewRequired(false)
    setNewOptions('')
    setAdding(false)
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    const name = deleteTarget.name
    removeField(deleteTarget.id)
    toast.success(t('work.settings.fields.deleted', { name }))
    setDeleteTarget(null)
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">{t('work.settings.fields.count', { count: fields.length })}</span>
        {!adding && (
          <button
            onClick={() => setAdding(true)}
            className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          >
            <Plus className="h-3.5 w-3.5" />
            {t('work.settings.fields.newField')}
          </button>
        )}
      </div>

      {adding && (
        <div className="space-y-3 rounded-lg border border-primary/30 bg-primary/5 p-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-foreground">{t('work.settings.fields.newField')}</span>
            <button onClick={() => setAdding(false)} className="rounded-md p-1 text-muted-foreground hover:text-foreground">
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
          <input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder={t('work.settings.fields.namePlaceholder')}
            autoFocus
            onKeyDown={(e) => { if (e.key === 'Enter') handleAdd() }}
            className="h-8 w-full rounded-md border border-border bg-transparent px-2 text-sm outline-none focus:border-primary"
          />
          <div className="grid grid-cols-3 gap-1.5 sm:grid-cols-5">
            {FIELD_TYPES.map((ft) => {
              const Icon = ft.icon
              const active = newType === ft.value
              return (
                <button
                  key={ft.value}
                  onClick={() => setNewType(ft.value)}
                  className={`flex items-center gap-1.5 rounded-md border px-2 py-1.5 text-xs transition-colors ${
                    active ? 'border-primary bg-primary/10 font-medium text-primary' : 'border-border text-muted-foreground hover:bg-secondary'
                  }`}
                >
                  <Icon className="h-3 w-3" />
                  {t(ft.labelKey)}
                </button>
              )
            })}
          </div>
          {newType === 'dropdown' && (
            <input
              value={newOptions}
              onChange={(e) => setNewOptions(e.target.value)}
              placeholder={t('work.settings.fields.optionsCommaSeparated')}
              className="h-8 w-full rounded-md border border-border bg-transparent px-2 text-sm outline-none focus:border-primary"
            />
          )}
          <div className="flex items-center justify-between">
            <label className="flex cursor-pointer items-center gap-2 text-xs text-muted-foreground">
              <input type="checkbox" checked={newRequired} onChange={(e) => setNewRequired(e.target.checked)} className="rounded accent-primary" />
              {t('work.settings.fields.required')}
            </label>
            <button
              onClick={handleAdd}
              disabled={!newName.trim()}
              className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
            >
              {t('work.settings.fields.add')}
            </button>
          </div>
        </div>
      )}

      {fields.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-6 text-center">
          <Type className="mx-auto h-7 w-7 text-muted-foreground/30" />
          <p className="mt-2 text-sm text-muted-foreground">{t('work.settings.fields.empty')}</p>
        </div>
      ) : (
        <div className="space-y-1.5">
          {fields
            .slice()
            .sort((a, b) => a.sortOrder - b.sortOrder)
            .map((field) => {
              const Icon = typeIcon(field.type)
              return (
                <div key={field.id} className="flex items-center gap-2.5 rounded-lg border border-border bg-card px-3 py-2">
                  <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                  <span className="text-sm text-foreground">{field.name}</span>
                  <span className="rounded bg-secondary px-1.5 py-0.5 text-[11px] text-muted-foreground">
                    {t(`work.settings.fields.type.${field.type}`)}
                  </span>
                  {field.required && (
                    <span className="rounded bg-primary-light px-1.5 py-0.5 text-[11px] text-primary">
                      {t('work.settings.fields.requiredBadge')}
                    </span>
                  )}
                  <button
                    onClick={() => setDeleteTarget(field)}
                    aria-label={t('common.delete')}
                    className="ml-auto text-muted-foreground transition-colors hover:text-error"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              )
            })}
        </div>
      )}

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title={t('work.settings.fields.deleteTitle')}
        description={t('work.settings.fields.deleteDescription', { name: deleteTarget?.name ?? '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}
