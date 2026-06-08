/**
 * DefaultStatusSetEditor — tenant-wide default workflow seeded into new projects.
 *
 * Embedded in the work settings panel. Mock-first (stores/workSettings); the
 * status set a freshly created project starts with. Per-project status edits
 * still happen in the project settings dialog.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, GripVertical } from 'lucide-react'
import { toast } from 'sonner'
import { useWorkSettingsStore, type WorkDefaultStatus } from '@/stores/workSettings'
import { ConfirmDialog, ColorSwatchPicker, SWATCH_COLORS } from '@/components/shared'

export function DefaultStatusSetEditor() {
  const { t } = useTranslation()
  const statuses = useWorkSettingsStore((s) => s.defaultStatuses)
  const addStatus = useWorkSettingsStore((s) => s.addDefaultStatus)
  const updateStatus = useWorkSettingsStore((s) => s.updateDefaultStatus)
  const removeStatus = useWorkSettingsStore((s) => s.removeDefaultStatus)

  const [adding, setAdding] = useState(false)
  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState(SWATCH_COLORS[0])
  const [deleteTarget, setDeleteTarget] = useState<WorkDefaultStatus | null>(null)

  const handleAdd = () => {
    const name = newName.trim()
    if (!name) return
    addStatus(name, newColor)
    toast.success(t('work.settings.statusSet.added', { name }))
    setNewName('')
    setNewColor(SWATCH_COLORS[0])
    setAdding(false)
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    removeStatus(deleteTarget.id)
    setDeleteTarget(null)
  }

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">{t('work.settings.statusSet.hint')}</p>

      <div className="space-y-1.5">
        {statuses.map((status) => (
          <div key={status.id} className="flex items-center gap-2.5 rounded-lg border border-border bg-card px-3 py-2">
            <GripVertical className="h-3.5 w-3.5 shrink-0 text-muted-foreground/40" />
            <ColorSwatchPicker value={status.color} size={16} onChange={(color) => updateStatus(status.id, { color })} />
            <input
              defaultValue={status.name}
              key={`${status.id}-${status.name}`}
              onBlur={(e) => {
                const v = e.target.value.trim()
                if (v && v !== status.name) updateStatus(status.id, { name: v })
              }}
              className="flex-1 rounded border border-transparent bg-transparent px-1 py-0.5 text-sm text-foreground outline-none hover:border-border focus:border-primary"
            />
            <button
              onClick={() => setDeleteTarget(status)}
              aria-label={t('common.delete')}
              disabled={statuses.length <= 1}
              className="text-muted-foreground transition-colors hover:text-error disabled:opacity-30"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        ))}
      </div>

      {adding ? (
        <div className="flex items-center gap-2 rounded-lg border border-primary/30 bg-primary/5 p-3">
          <ColorSwatchPicker value={newColor} onChange={setNewColor} />
          <input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder={t('work.settings.statusSet.namePlaceholder')}
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
            {t('work.settings.statusSet.addButton')}
          </button>
        </div>
      ) : (
        <button
          onClick={() => setAdding(true)}
          className="flex items-center gap-1.5 rounded-lg border border-dashed border-border px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
        >
          <Plus className="h-3.5 w-3.5" />
          {t('work.settings.statusSet.newStatus')}
        </button>
      )}

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title={t('work.settings.statusSet.deleteTitle')}
        description={t('work.settings.statusSet.deleteDescription', { name: deleteTarget?.name ?? '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}
