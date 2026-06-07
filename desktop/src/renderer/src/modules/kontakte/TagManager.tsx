/**
 * TagManager — tenant-wide contact tag administration.
 *
 * Embedded in the CRM settings panel. List/create/rename/recolour/delete tag
 * definitions. Backed by the tag CRUD hooks (mock handlers in crm.ts).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, Tag as TagIcon } from 'lucide-react'
import { toast } from 'sonner'
import {
  useTags,
  useCreateTag,
  useUpdateTag,
  useDeleteTag,
  type TagInfo,
} from '@/api/hooks/useContactTags'
import { ConfirmDialog, ColorSwatchPicker, SWATCH_COLORS } from '@/components/shared'

export function TagManager() {
  const { t } = useTranslation()
  const { data: tags = [], isLoading } = useTags()
  const createTag = useCreateTag()
  const updateTag = useUpdateTag()
  const deleteTag = useDeleteTag()

  const [adding, setAdding] = useState(false)
  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState(SWATCH_COLORS[0])
  const [deleteTarget, setDeleteTarget] = useState<TagInfo | null>(null)

  const handleAdd = () => {
    if (!newName.trim()) return
    createTag.mutate(
      { name: newName.trim(), color: newColor },
      {
        onSuccess: () => {
          toast.success(t('crm.settings.tags.added', { name: newName.trim() }))
          setNewName(''); setNewColor(SWATCH_COLORS[0]); setAdding(false)
        },
      },
    )
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    const name = deleteTarget.name
    deleteTag.mutate(deleteTarget.id, {
      onSuccess: () => toast.success(t('crm.settings.tags.deleted', { name })),
    })
    setDeleteTarget(null)
  }

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
  }

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">{t('crm.settings.tags.count', { count: tags.length })}</p>

      {tags.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-6 text-center">
          <TagIcon className="mx-auto h-7 w-7 text-muted-foreground/30" />
          <p className="mt-2 text-sm text-muted-foreground">{t('crm.settings.tags.empty')}</p>
        </div>
      ) : (
        <div className="flex flex-wrap gap-2">
          {tags.map((tag) => (
            <div
              key={tag.id}
              className="flex items-center gap-2 rounded-full border border-border bg-card py-1 pl-1.5 pr-2"
            >
              <ColorSwatchPicker
                value={tag.color}
                size={16}
                onChange={(color) => updateTag.mutate({ id: tag.id, color })}
              />
              <input
                defaultValue={tag.name}
                key={`${tag.id}-${tag.name}`}
                onBlur={(e) => {
                  const v = e.target.value.trim()
                  if (v && v !== tag.name) updateTag.mutate({ id: tag.id, name: v })
                }}
                className="w-24 rounded border border-transparent bg-transparent px-1 py-0.5 text-xs text-foreground outline-none hover:border-border focus:border-primary"
              />
              <button
                onClick={() => setDeleteTarget(tag)}
                aria-label={t('common.delete')}
                className="text-muted-foreground transition-colors hover:text-error"
              >
                <Trash2 className="h-3 w-3" />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Add */}
      {adding ? (
        <div className="flex items-center gap-2 rounded-lg border border-primary/30 bg-primary/5 p-3">
          <ColorSwatchPicker value={newColor} onChange={setNewColor} />
          <input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder={t('crm.settings.tags.namePlaceholder')}
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
            {t('crm.settings.tags.addButton')}
          </button>
        </div>
      ) : (
        <button
          onClick={() => setAdding(true)}
          className="flex items-center gap-1.5 rounded-lg border border-dashed border-border px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
        >
          <Plus className="h-3.5 w-3.5" />
          {t('crm.settings.tags.newTag')}
        </button>
      )}

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title={t('crm.settings.tags.deleteTitle')}
        description={t('crm.settings.tags.deleteDescription', { name: deleteTarget?.name ?? '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}
