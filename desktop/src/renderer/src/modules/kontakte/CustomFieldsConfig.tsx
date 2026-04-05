/**
 * CustomFieldsConfig — Admin panel for managing custom field definitions.
 *
 * Accessible from KontaktePage (e.g. settings/gear button).
 * CRUD operations via useContactsStore.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Settings2,
  Plus,
  Type,
  Hash,
  Calendar,
  ChevronDown,
  CheckSquare,
  Link2,
  X,
} from 'lucide-react'
import { toast } from 'sonner'
import { useContactsStore, type CustomFieldType } from '@/stores/contacts'
import { ConfirmDialog } from '@/components/shared'
import { CustomFieldRow } from './CustomFieldRow'
import { CustomFieldPreview } from './CustomFieldPreview'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Type options
// ---------------------------------------------------------------------------

const fieldTypes: { value: CustomFieldType; icon: typeof Type; labelKey: string; descriptionKey: string }[] = [
  { value: 'text', icon: Type, labelKey: 'kontakte.customField.type.text', descriptionKey: 'kontakte.customField.typeDesc.text' },
  { value: 'number', icon: Hash, labelKey: 'kontakte.customField.type.number', descriptionKey: 'kontakte.customField.typeDesc.number' },
  { value: 'date', icon: Calendar, labelKey: 'kontakte.customField.type.date', descriptionKey: 'kontakte.customField.typeDesc.date' },
  { value: 'dropdown', icon: ChevronDown, labelKey: 'kontakte.customField.type.dropdown', descriptionKey: 'kontakte.customField.typeDesc.dropdown' },
  { value: 'checkbox', icon: CheckSquare, labelKey: 'kontakte.customField.type.checkbox', descriptionKey: 'kontakte.customField.typeDesc.checkbox' },
  { value: 'url', icon: Link2, labelKey: 'kontakte.customField.type.url', descriptionKey: 'kontakte.customField.typeDesc.url' },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface CustomFieldsConfigProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CustomFieldsConfig({ open, onOpenChange }: CustomFieldsConfigProps) {
  const { t } = useTranslation()
  const fields = useContactsStore((s) => s.customFieldDefinitions)
  const addField = useContactsStore((s) => s.addCustomFieldDefinition)
  const updateField = useContactsStore((s) => s.updateCustomFieldDefinition)
  const deleteField = useContactsStore((s) => s.deleteCustomFieldDefinition)

  const [showAddForm, setShowAddForm] = useState(false)
  const [newName, setNewName] = useState('')
  const [newType, setNewType] = useState<CustomFieldType>('text')
  const [newRequired, setNewRequired] = useState(false)
  const [newOptions, setNewOptions] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null)

  const handleAdd = () => {
    if (!newName.trim()) return
    const field: { name: string; type: CustomFieldType; required: boolean; options?: string[] } = {
      name: newName.trim(),
      type: newType,
      required: newRequired,
    }
    if (newType === 'dropdown') {
      field.options = newOptions.split(',').map((o) => o.trim()).filter(Boolean)
    }
    addField(field)
    toast.success(t('kontakte.customField.fieldCreated', { name: newName.trim() }))
    setNewName('')
    setNewType('text')
    setNewRequired(false)
    setNewOptions('')
    setShowAddForm(false)
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    deleteField(deleteTarget.id)
    toast.success(t('kontakte.customField.fieldDeleted', { name: deleteTarget.name }))
    setDeleteTarget(null)
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Settings2 className="h-5 w-5 text-muted-foreground" />
              {t('kontakte.customField.title')}
            </DialogTitle>
            <DialogDescription>
              {t('kontakte.customField.description')}
            </DialogDescription>
          </DialogHeader>

          <div className="grid grid-cols-[1fr_280px] gap-6 pt-2">
            {/* Left: Field list */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                  {t('kontakte.customField.fieldsDefined', { count: fields.length })}
                </span>
                <button
                  onClick={() => setShowAddForm(true)}
                  className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
                >
                  <Plus className="h-3.5 w-3.5" />
                  {t('kontakte.customField.newField')}
                </button>
              </div>

              {/* Add form */}
              {showAddForm && (
                <div className="rounded-lg border border-primary/30 bg-primary/5 p-3 space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-foreground">{t('kontakte.customField.newField')}</span>
                    <button onClick={() => setShowAddForm(false)} className="rounded-md p-1 text-muted-foreground hover:text-foreground">
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>

                  {/* Name */}
                  <input
                    type="text"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    placeholder={t('kontakte.customField.fieldName')}
                    className="h-8 w-full rounded-md border border-border bg-transparent px-2 text-sm outline-none focus:border-primary"
                    autoFocus
                    onKeyDown={(e) => { if (e.key === 'Enter') handleAdd() }}
                  />

                  {/* Type grid */}
                  <div className="grid grid-cols-3 gap-1.5">
                    {fieldTypes.map((ft) => {
                      const FtIcon = ft.icon
                      const isActive = newType === ft.value
                      return (
                        <button
                          key={ft.value}
                          onClick={() => setNewType(ft.value)}
                          className={`flex items-center gap-1.5 rounded-md border px-2 py-1.5 text-xs transition-colors ${
                            isActive
                              ? 'border-primary bg-primary/10 text-primary font-medium'
                              : 'border-border text-muted-foreground hover:bg-accent'
                          }`}
                        >
                          <FtIcon className="h-3 w-3" />
                          {t(ft.labelKey)}
                        </button>
                      )
                    })}
                  </div>

                  {/* Dropdown options */}
                  {newType === 'dropdown' && (
                    <input
                      type="text"
                      value={newOptions}
                      onChange={(e) => setNewOptions(e.target.value)}
                      placeholder={t('kontakte.customField.optionsCommaSeparated')}
                      className="h-8 w-full rounded-md border border-border bg-transparent px-2 text-sm outline-none focus:border-primary"
                    />
                  )}

                  {/* Required + Submit */}
                  <div className="flex items-center justify-between">
                    <label className="flex items-center gap-2 text-xs text-muted-foreground cursor-pointer">
                      <input
                        type="checkbox"
                        checked={newRequired}
                        onChange={(e) => setNewRequired(e.target.checked)}
                        className="rounded accent-primary"
                      />
                      {t('kontakte.customField.requiredField')}
                    </label>
                    <button
                      onClick={handleAdd}
                      disabled={!newName.trim()}
                      className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                    >
                      {t('kontakte.customField.add')}
                    </button>
                  </div>
                </div>
              )}

              {/* Field list */}
              {fields.length === 0 ? (
                <div className="rounded-lg border border-dashed border-border p-8 text-center">
                  <Settings2 className="mx-auto h-8 w-8 text-muted-foreground/30" />
                  <p className="mt-2 text-sm text-muted-foreground">{t('kontakte.customField.noFieldsDefined')}</p>
                  <p className="text-xs text-muted-foreground/70">
                    {t('kontakte.customField.createFieldHint')}
                  </p>
                </div>
              ) : (
                <div className="space-y-1.5">
                  {fields
                    .sort((a, b) => a.sortOrder - b.sortOrder)
                    .map((field) => (
                      <CustomFieldRow
                        key={field.id}
                        field={field}
                        onUpdate={updateField}
                        onDelete={(id) => setDeleteTarget({ id, name: field.name })}
                      />
                    ))}
                </div>
              )}
            </div>

            {/* Right: Preview */}
            <CustomFieldPreview fields={fields} />
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title={t('kontakte.customField.deleteTitle')}
        description={t('kontakte.customField.deleteDescription', { name: deleteTarget?.name })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </>
  )
}
