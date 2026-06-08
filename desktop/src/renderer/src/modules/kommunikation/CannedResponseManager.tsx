/**
 * Canned-response management (tenant scope, Phase 5).
 *
 * CRUD over the shared canned-response store (useKommunikationStore). Backend
 * canned-response CRUD does not exist yet (see backend-gaps.md) — the store is
 * the mock backing and swaps to API hooks later.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Pencil, Trash2, Check } from 'lucide-react'
import { useKommunikationStore } from '@/stores/kommunikation'
import type { CannedResponse } from '@/types/communication'

type Draft = { title: string; content: string; category: string; shortcut: string }

const EMPTY: Draft = { title: '', content: '', category: 'allgemein', shortcut: '' }

export function CannedResponseManager() {
  const { t } = useTranslation()
  const responses = useKommunikationStore((s) => s.cannedResponses)
  const add = useKommunikationStore((s) => s.addCannedResponse)
  const update = useKommunikationStore((s) => s.updateCannedResponse)
  const remove = useKommunikationStore((s) => s.deleteCannedResponse)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [draft, setDraft] = useState<Draft>(EMPTY)

  const startEdit = (r: CannedResponse) => {
    setEditingId(r.id)
    setCreating(false)
    setDraft({ title: r.title, content: r.content, category: r.category, shortcut: r.shortcut ?? '' })
  }

  const startCreate = () => {
    setCreating(true)
    setEditingId(null)
    setDraft(EMPTY)
  }

  const cancel = () => {
    setCreating(false)
    setEditingId(null)
    setDraft(EMPTY)
  }

  const save = () => {
    if (!draft.title.trim() || !draft.content.trim()) return
    const payload = {
      title: draft.title.trim(),
      content: draft.content.trim(),
      category: draft.category.trim() || 'allgemein',
      shortcut: draft.shortcut.trim() || undefined,
    }
    if (editingId) {
      update(editingId, payload)
    } else {
      add({ ...payload, createdBy: t('kommunikation.thread.you') })
    }
    cancel()
  }

  const renderForm = () => (
    <div className="space-y-2 rounded-lg border border-primary/30 bg-primary/5 p-3">
      <div className="grid grid-cols-2 gap-2">
        <input
          value={draft.title}
          onChange={(e) => setDraft({ ...draft, title: e.target.value })}
          placeholder={t('kommunikation.canned.titlePlaceholder')}
          className="rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
        />
        <input
          value={draft.shortcut}
          onChange={(e) => setDraft({ ...draft, shortcut: e.target.value })}
          placeholder={t('kommunikation.canned.shortcutPlaceholder')}
          className="rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
        />
      </div>
      <textarea
        value={draft.content}
        onChange={(e) => setDraft({ ...draft, content: e.target.value })}
        rows={2}
        placeholder={t('kommunikation.canned.contentPlaceholder')}
        className="w-full resize-none rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
      />
      <div className="flex justify-end gap-2">
        <button onClick={cancel} className="rounded-md px-2.5 py-1 text-xs text-muted-foreground hover:bg-accent transition-colors">
          {t('common.cancel')}
        </button>
        <button
          onClick={save}
          disabled={!draft.title.trim() || !draft.content.trim()}
          className="flex items-center gap-1 rounded-md bg-primary px-2.5 py-1 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
        >
          <Check className="h-3.5 w-3.5" />
          {t('common.save')}
        </button>
      </div>
    </div>
  )

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-foreground">{t('kommunikation.canned.title')}</h3>
        {!creating && (
          <button
            onClick={startCreate}
            className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-3.5 w-3.5" />
            {t('kommunikation.canned.new')}
          </button>
        )}
      </div>

      {creating && renderForm()}

      <div className="space-y-1">
        {responses.map((r) =>
          editingId === r.id ? (
            <div key={r.id}>{renderForm()}</div>
          ) : (
            <div
              key={r.id}
              className="flex items-start gap-3 rounded-md border border-border px-3 py-2 hover:bg-secondary/50 transition-colors"
            >
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-foreground truncate">{r.title}</span>
                  {r.shortcut && (
                    <span className="rounded bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground">{r.shortcut}</span>
                  )}
                </div>
                <p className="truncate text-[11px] text-muted-foreground">{r.content}</p>
              </div>
              <button
                onClick={() => startEdit(r)}
                className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
                title={t('common.edit')}
              >
                <Pencil className="h-3.5 w-3.5" />
              </button>
              <button
                onClick={() => remove(r.id)}
                className="rounded p-1 text-muted-foreground hover:text-error hover:bg-secondary transition-colors"
                title={t('common.delete')}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ),
        )}
        {responses.length === 0 && !creating && (
          <p className="py-3 text-center text-xs text-muted-foreground">{t('kommunikation.canned.empty')}</p>
        )}
      </div>
    </div>
  )
}
