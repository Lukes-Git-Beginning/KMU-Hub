/**
 * PipelineStagesEditor — tenant-wide pipeline stage configuration.
 *
 * Embedded in the CRM settings panel. Lists stages with inline name/probability
 * editing, colour swatches, up/down reordering and add/delete. Backed by the
 * pipeline-stage hooks (mock handlers in mocks/handlers/crm.ts).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, ChevronUp, ChevronDown, Trophy, XCircle } from 'lucide-react'
import { toast } from 'sonner'
import {
  usePipelineStages,
  useCreatePipelineStage,
  useUpdatePipelineStage,
  useDeletePipelineStage,
  useReorderPipelineStages,
} from '@/api/hooks/usePipelineStages'
import { ConfirmDialog } from '@/components/shared'

const STAGE_COLORS = [
  '#6B7280', '#3B82F6', '#06B6D4', '#10B981',
  '#F59E0B', '#F97316', '#EF4444', '#8B5CF6', '#EC4899',
]

interface Stage {
  id: string
  name: string
  order: number
  probability: number
  color: string
  is_won: boolean
  is_lost: boolean
  dealCount?: number
}

export function PipelineStagesEditor() {
  const { t } = useTranslation()
  const { data, isLoading } = usePipelineStages()
  const createStage = useCreatePipelineStage()
  const updateStage = useUpdatePipelineStage()
  const deleteStage = useDeletePipelineStage()
  const reorder = useReorderPipelineStages()

  const [adding, setAdding] = useState(false)
  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState(STAGE_COLORS[0])
  const [newProb, setNewProb] = useState(20)
  const [deleteTarget, setDeleteTarget] = useState<Stage | null>(null)

  const stages: Stage[] = [...(data?.stages ?? [])].sort((a, b) => a.order - b.order)

  const handleAdd = () => {
    if (!newName.trim()) return
    createStage.mutate(
      { name: newName.trim(), color: newColor, probability: newProb, is_won: false, is_lost: false },
      {
        onSuccess: () => {
          toast.success(t('crm.settings.pipeline.added', { name: newName.trim() }))
          setNewName(''); setNewColor(STAGE_COLORS[0]); setNewProb(20); setAdding(false)
        },
      },
    )
  }

  const move = (index: number, dir: -1 | 1) => {
    const ids = stages.map((s) => s.id)
    const j = index + dir
    if (j < 0 || j >= ids.length) return
    const tmp = ids[index]; ids[index] = ids[j]; ids[j] = tmp
    reorder.mutate(ids)
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    const name = deleteTarget.name
    deleteStage.mutate(deleteTarget.id, {
      onSuccess: () => toast.success(t('crm.settings.pipeline.deleted', { name })),
    })
    setDeleteTarget(null)
  }

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
  }

  return (
    <div className="space-y-3">
      <div className="space-y-1.5">
        {stages.map((stage, index) => (
          <div
            key={stage.id}
            className="flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-2"
          >
            {/* Reorder */}
            <div className="flex flex-col">
              <button
                onClick={() => move(index, -1)}
                disabled={index === 0}
                aria-label={t('crm.settings.pipeline.moveUp')}
                className="text-muted-foreground transition-colors hover:text-foreground disabled:opacity-30"
              >
                <ChevronUp className="h-3.5 w-3.5" />
              </button>
              <button
                onClick={() => move(index, 1)}
                disabled={index === stages.length - 1}
                aria-label={t('crm.settings.pipeline.moveDown')}
                className="text-muted-foreground transition-colors hover:text-foreground disabled:opacity-30"
              >
                <ChevronDown className="h-3.5 w-3.5" />
              </button>
            </div>

            {/* Colour */}
            <ColorPopover
              value={stage.color}
              onChange={(color) => updateStage.mutate({ id: stage.id, color })}
            />

            {/* Name */}
            <input
              defaultValue={stage.name}
              key={`${stage.id}-${stage.name}`}
              onBlur={(e) => {
                const v = e.target.value.trim()
                if (v && v !== stage.name) updateStage.mutate({ id: stage.id, name: v })
              }}
              className="min-w-0 flex-1 rounded-md border border-transparent bg-transparent px-1.5 py-1 text-sm text-foreground outline-none hover:border-border focus:border-primary"
            />

            {/* Won/Lost markers */}
            {stage.is_won && <Trophy className="h-3.5 w-3.5 shrink-0 text-success" aria-label={t('crm.settings.pipeline.won')} />}
            {stage.is_lost && <XCircle className="h-3.5 w-3.5 shrink-0 text-error" aria-label={t('crm.settings.pipeline.lost')} />}

            {/* Probability */}
            <div className="flex shrink-0 items-center gap-1">
              <input
                type="number"
                min={0}
                max={100}
                defaultValue={stage.probability}
                key={`${stage.id}-${stage.probability}`}
                onBlur={(e) => {
                  const v = Math.max(0, Math.min(100, Number(e.target.value)))
                  if (v !== stage.probability) updateStage.mutate({ id: stage.id, probability: v })
                }}
                className="w-14 rounded-md border border-border bg-transparent px-1.5 py-1 text-right text-sm tabular-nums outline-none focus:border-primary"
              />
              <span className="text-xs text-muted-foreground">%</span>
            </div>

            {/* Deal count */}
            <span className="w-16 shrink-0 text-right text-[11px] text-muted-foreground">
              {t('crm.settings.pipeline.dealCount', { count: stage.dealCount ?? 0 })}
            </span>

            {/* Delete */}
            <button
              onClick={() => setDeleteTarget(stage)}
              aria-label={t('common.delete')}
              className="shrink-0 text-muted-foreground transition-colors hover:text-error"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        ))}
      </div>

      {/* Add */}
      {adding ? (
        <div className="space-y-2 rounded-lg border border-primary/30 bg-primary/5 p-3">
          <div className="flex items-center gap-2">
            <ColorPopover value={newColor} onChange={setNewColor} />
            <input
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder={t('crm.settings.pipeline.namePlaceholder')}
              autoFocus
              onKeyDown={(e) => { if (e.key === 'Enter') handleAdd() }}
              className="min-w-0 flex-1 rounded-md border border-border bg-transparent px-2 py-1 text-sm outline-none focus:border-primary"
            />
            <input
              type="number"
              min={0}
              max={100}
              value={newProb}
              onChange={(e) => setNewProb(Math.max(0, Math.min(100, Number(e.target.value))))}
              className="w-14 rounded-md border border-border bg-transparent px-1.5 py-1 text-right text-sm tabular-nums outline-none focus:border-primary"
            />
            <span className="text-xs text-muted-foreground">%</span>
          </div>
          <div className="flex justify-end gap-2">
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
              {t('crm.settings.pipeline.addButton')}
            </button>
          </div>
        </div>
      ) : (
        <button
          onClick={() => setAdding(true)}
          className="flex w-full items-center justify-center gap-1.5 rounded-lg border border-dashed border-border py-2 text-xs font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
        >
          <Plus className="h-3.5 w-3.5" />
          {t('crm.settings.pipeline.newStage')}
        </button>
      )}

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title={t('crm.settings.pipeline.deleteTitle')}
        description={t('crm.settings.pipeline.deleteDescription', { name: deleteTarget?.name ?? '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}

function ColorPopover({ value, onChange }: { value: string; onChange: (c: string) => void }) {
  const [open, setOpen] = useState(false)
  return (
    <div className="relative shrink-0">
      <button
        onClick={() => setOpen((v) => !v)}
        aria-label="Farbe"
        className="h-5 w-5 rounded-full border border-border"
        style={{ backgroundColor: value }}
      />
      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute left-0 top-7 z-20 grid grid-cols-3 gap-1.5 rounded-lg border border-border bg-card p-2 shadow-lg">
            {STAGE_COLORS.map((c) => (
              <button
                key={c}
                onClick={() => { onChange(c); setOpen(false) }}
                className="h-5 w-5 rounded-full border border-border transition-transform hover:scale-110"
                style={{ backgroundColor: c }}
              />
            ))}
          </div>
        </>
      )}
    </div>
  )
}
