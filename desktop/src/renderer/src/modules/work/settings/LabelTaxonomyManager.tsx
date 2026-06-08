/**
 * LabelTaxonomyManager — tenant-wide task label administration.
 *
 * Embedded in the work settings panel. List/create/rename/recolour/delete label
 * definitions. P3 builds the task chip UI + filter on this taxonomy.
 * Mock-first (stores/workSettings); backend gap tracked in backend-gaps.md.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, Tags as TagsIcon } from 'lucide-react'
import { toast } from 'sonner'
import { useWorkSettingsStore, type WorkLabel } from '@/stores/workSettings'
import { ConfirmDialog, ColorSwatchPicker, SWATCH_COLORS } from '@/components/shared'

export function LabelTaxonomyManager() {
  const { t } = useTranslation()
  const labels = useWorkSettingsStore((s) => s.labels)
  const addLabel = useWorkSettingsStore((s) => s.addLabel)
  const updateLabel = useWorkSettingsStore((s) => s.updateLabel)
  const removeLabel = useWorkSettingsStore((s) => s.removeLabel)

  const [adding, setAdding] = useState(false)
  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState(SWATCH_COLORS[0])
  const [deleteTarget, setDeleteTarget] = useState<WorkLabel | null>(null)

  const handleAdd = () => {
    const name = newName.trim()
    if (!name) return
    addLabel(name, newColor)
    toast.success(t('work.settings.labels.added', { name }))
    setNewName('')
    setNewColor(SWATCH_COLORS[0])
    setAdding(false)
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    const name = deleteTarget.name
    removeLabel(deleteTarget.id)
    toast.success(t('work.settings.labels.deleted', { name }))
    setDeleteTarget(null)
  }

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">{t('work.settings.labels.count', { count: labels.length })}</p>

      {labels.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-6 text-center">
          <TagsIcon className="mx-auto h-7 w-7 text-muted-foreground/30" />
          <p className="mt-2 text-sm text-muted-foreground">{t('work.settings.labels.empty')}</p>
        </div>
      ) : (
        <div className="flex flex-wrap gap-2">
          {labels.map((label) => (
            <div
              key={label.id}
              className="flex items-center gap-2 rounded-full border border-border bg-card py-1 pl-1.5 pr-2"
            >
              <ColorSwatchPicker
                value={label.color}
                size={16}
                onChange={(color) => updateLabel(label.id, { color })}
              />
              <input
                defaultValue={label.name}
                key={`${label.id}-${label.name}`}
                onBlur={(e) => {
                  const v = e.target.value.trim()
                  if (v && v !== label.name) updateLabel(label.id, { name: v })
                }}
                className="w-24 rounded border border-transparent bg-transparent px-1 py-0.5 text-xs text-foreground outline-none hover:border-border focus:border-primary"
              />
              <button
                onClick={() => setDeleteTarget(label)}
                aria-label={t('common.delete')}
                className="text-muted-foreground transition-colors hover:text-error"
              >
                <Trash2 className="h-3 w-3" />
              </button>
            </div>
          ))}
        </div>
      )}

      {adding ? (
        <div className="flex items-center gap-2 rounded-lg border border-primary/30 bg-primary/5 p-3">
          <ColorSwatchPicker value={newColor} onChange={setNewColor} />
          <input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder={t('work.settings.labels.namePlaceholder')}
            autoFocus
            onKeyDown={(e) => { if (e.key === 'Enter') handleAdd() }}
            className="min-w-0 flex-1 rounded-md border border-border bg-transparent px-2 py-1 text-sm outline-none focus:border-primary"
          />
          <button
            onClick={() => { setAdding(false); setNewName('') }}
            className="rounded-md border border-border px-3 py-1.5 text-xs text-foreground transition-colors hover:bg-secondary"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleAdd}
            disabled={!newName.trim()}
            className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
          >
            {t('work.settings.labels.addButton')}
          </button>
        </div>
      ) : (
        <button
          onClick={() => setAdding(true)}
          className="flex items-center gap-1.5 rounded-lg border border-dashed border-border px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
        >
          <Plus className="h-3.5 w-3.5" />
          {t('work.settings.labels.newLabel')}
        </button>
      )}

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title={t('work.settings.labels.deleteTitle')}
        description={t('work.settings.labels.deleteDescription', { name: deleteTarget?.name ?? '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}
