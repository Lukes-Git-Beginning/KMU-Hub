import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Pencil, Eye, Trash2, ChevronLeft } from 'lucide-react'
import { toast } from 'sonner'
import { useBookingPages, useDeleteBookingPage } from '@/api/hooks/useBookingPages'
import type { BookingPage } from '@/api/booking-client'
import { BookingPageEditor } from './BookingPageEditor'
import { BookingPagePreview } from './BookingPagePreview'

type PanelMode = 'list' | 'editor' | 'preview'

export function BookingPagesPanel() {
  const { t } = useTranslation()

  const [mode, setMode] = useState<PanelMode>('list')
  const [editorPage, setEditorPage] = useState<BookingPage | undefined>(undefined)
  const [previewPage, setPreviewPage] = useState<BookingPage | null>(null)
  const [includeInactive, setIncludeInactive] = useState(false)

  const { data, isLoading } = useBookingPages(includeInactive)
  const pages = data?.pages ?? []
  const deleteMutation = useDeleteBookingPage()

  function handleCreate() {
    setEditorPage(undefined)
    setMode('editor')
  }

  function handleEdit(page: BookingPage) {
    setEditorPage(page)
    setMode('editor')
  }

  function handlePreview(page: BookingPage) {
    setPreviewPage(page)
    setMode('preview')
  }

  function handleSave(page: BookingPage) {
    // After successful save, return to list; query is already invalidated by mutation
    setMode('list')
    setEditorPage(undefined)
  }

  function handleCancel() {
    setMode('list')
    setEditorPage(undefined)
  }

  async function handleDelete(page: BookingPage) {
    const confirmed = window.confirm(
      t('kalender.booking.pages.panel.deleteConfirm', { slug: page.slug }),
    )
    if (!confirmed) return
    try {
      await deleteMutation.mutateAsync(page.id)
      toast.success(t('kalender.booking.pages.panel.deletedSuccess'))
    } catch {
      // error handled by mutation
    }
  }

  // ---------------------------------------------------------------------------
  // Editor mode
  // ---------------------------------------------------------------------------
  if (mode === 'editor') {
    return (
      <div>
        <div className="mb-4 flex items-center gap-3">
          <button
            onClick={handleCancel}
            className="flex items-center gap-1.5 text-xs text-primary hover:underline"
          >
            <ChevronLeft className="h-3.5 w-3.5" />
            {t('kalender.booking.pages.panel.backToList')}
          </button>
        </div>
        <BookingPageEditor
          page={editorPage}
          onSave={handleSave}
          onCancel={handleCancel}
        />
      </div>
    )
  }

  // ---------------------------------------------------------------------------
  // Preview mode
  // ---------------------------------------------------------------------------
  if (mode === 'preview' && previewPage) {
    return (
      <div>
        <div className="mb-4 flex items-center gap-3">
          <button
            onClick={() => setMode('list')}
            className="flex items-center gap-1.5 text-xs text-primary hover:underline"
          >
            <ChevronLeft className="h-3.5 w-3.5" />
            {t('kalender.booking.pages.panel.backToList')}
          </button>
        </div>
        <BookingPagePreview page={previewPage} />
      </div>
    )
  }

  // ---------------------------------------------------------------------------
  // List mode
  // ---------------------------------------------------------------------------
  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h3 className="text-sm font-medium text-foreground">
            {t('kalender.booking.pages.panel.title')}
          </h3>
          <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer select-none">
            <input
              type="checkbox"
              checked={includeInactive}
              onChange={(e) => setIncludeInactive(e.target.checked)}
              className="rounded"
            />
            {t('kalender.booking.pages.panel.includeInactive')}
          </label>
        </div>
        <button
          onClick={handleCreate}
          className="flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Plus className="h-4 w-4" />
          {t('kalender.booking.pages.panel.newPage')}
        </button>
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
          <span className="animate-pulse">...</span>
        </div>
      )}

      {/* Empty state */}
      {!isLoading && pages.length === 0 && (
        <div className="rounded-lg border border-border bg-card p-12 text-center">
          <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-secondary">
            <Plus className="h-6 w-6 text-muted-foreground" />
          </div>
          <h4 className="mb-1 text-sm font-medium text-foreground">
            {t('kalender.booking.pages.panel.emptyTitle')}
          </h4>
          <p className="mb-4 text-xs text-muted-foreground">
            {t('kalender.booking.pages.panel.emptyDescription')}
          </p>
          <button
            onClick={handleCreate}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            {t('kalender.booking.pages.panel.emptyAction')}
          </button>
        </div>
      )}

      {/* Pages list */}
      {!isLoading && pages.length > 0 && (
        <div className="rounded-lg border border-border bg-card overflow-hidden">
          {pages.map((page, idx) => (
            <div
              key={page.id}
              className={`flex items-center justify-between gap-4 px-5 py-4 hover:bg-secondary/20 transition-colors ${
                idx > 0 ? 'border-t border-border-muted' : ''
              }`}
            >
              {/* Info */}
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium text-foreground truncate">
                    {page.company_name}
                  </p>
                  <span
                    className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium ${
                      page.active
                        ? 'bg-success/15 text-success'
                        : 'bg-muted text-muted-foreground'
                    }`}
                  >
                    {page.active
                      ? t('kalender.booking.pages.panel.activeLabel')
                      : t('kalender.booking.pages.panel.inactiveLabel')}
                  </span>
                </div>
                <p className="mt-0.5 text-xs text-muted-foreground truncate">
                  /{page.slug}
                </p>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-2 shrink-0">
                <button
                  onClick={() => handleEdit(page)}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
                >
                  <Pencil className="h-3 w-3" />
                  {t('kalender.booking.pages.panel.editButton')}
                </button>
                <button
                  onClick={() => handlePreview(page)}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
                >
                  <Eye className="h-3 w-3" />
                  {t('kalender.booking.pages.panel.previewButton')}
                </button>
                <button
                  onClick={() => handleDelete(page)}
                  disabled={deleteMutation.isPending}
                  className="flex items-center gap-1.5 rounded-lg border border-error/30 px-3 py-1.5 text-xs text-error hover:bg-error/10 transition-colors disabled:opacity-60"
                >
                  <Trash2 className="h-3 w-3" />
                  {t('kalender.booking.pages.panel.deleteButton')}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
