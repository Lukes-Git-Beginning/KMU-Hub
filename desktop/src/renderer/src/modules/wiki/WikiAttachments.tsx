import { useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Paperclip, Plus, Trash2, FileText, Image as ImageIcon, File, Loader2 } from 'lucide-react'
import {
  useAttachments,
  useUploadAttachment,
  useDeleteAttachment,
} from '@/api/hooks/useWiki'
import { formatDate as libFormatDate } from '@/lib/format'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatBytes(bytes: number): string {
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(1)} GB`
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(1)} MB`
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${bytes} B`
}

function iconForMime(mime: string): typeof FileText {
  if (mime.startsWith('image/')) return ImageIcon
  if (mime === 'application/pdf' || mime.startsWith('text/')) return FileText
  return File
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiAttachmentsProps {
  articleId: string
}

/**
 * Attachment panel for the article detail view. Lists attachments and supports
 * adding (file metadata only — the mock stores name/mime/size) and removing.
 */
export function WikiAttachments({ articleId }: WikiAttachmentsProps) {
  const { t } = useTranslation()
  const { data: attachments = [], isLoading } = useAttachments(articleId)
  const uploadMutation = useUploadAttachment()
  const deleteMutation = useDeleteAttachment()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handlePick = () => fileInputRef.current?.click()

  const handleFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = '' // allow re-selecting the same file
    if (!file) return
    try {
      await uploadMutation.mutateAsync({
        articleId,
        file_ref: file.name,
        mime: file.type || 'application/octet-stream',
        size: file.size,
      })
      toast.success(t('wiki.attachments.added'))
    } catch {
      toast.error(t('wiki.attachments.addError'))
    }
  }

  const handleDelete = async (attachmentId: string) => {
    try {
      await deleteMutation.mutateAsync({ attachmentId, articleId })
      toast.success(t('wiki.attachments.deleted'))
    } catch {
      toast.error(t('wiki.attachments.deleteError'))
    }
  }

  return (
    <div className="mt-6 border-t border-border pt-4">
      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        onChange={handleFile}
      />

      {/* Header */}
      <div className="mb-2 flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <Paperclip className="h-3.5 w-3.5 text-muted-foreground" />
          <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            {t('wiki.attachments.title')}
          </h3>
          <span className="text-[10px] text-muted-foreground">({attachments.length})</span>
        </div>
        <button
          onClick={handlePick}
          disabled={uploadMutation.isPending}
          className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-foreground hover:bg-accent transition-colors disabled:opacity-50"
        >
          {uploadMutation.isPending ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <Plus className="h-3 w-3" />
          )}
          {t('wiki.attachments.add')}
        </button>
      </div>

      {/* List */}
      {isLoading ? (
        <div className="space-y-1.5">
          {[1, 2].map((i) => (
            <div key={i} className="h-10 rounded-md bg-muted animate-pulse" />
          ))}
        </div>
      ) : attachments.length === 0 ? (
        <button
          onClick={handlePick}
          className="flex w-full flex-col items-center gap-1 rounded-lg border border-dashed border-border px-4 py-5 text-center transition-colors hover:bg-accent/40"
        >
          <Paperclip className="h-4 w-4 text-muted-foreground/50" />
          <span className="text-xs text-muted-foreground">{t('wiki.attachments.empty')}</span>
        </button>
      ) : (
        <div className="space-y-1">
          {attachments.map((att) => {
            const Icon = iconForMime(att.mime)
            const deleting = deleteMutation.isPending && deleteMutation.variables?.attachmentId === att.id
            return (
              <div
                key={att.id}
                className="group flex items-center gap-2.5 rounded-md border border-border bg-card/40 px-2.5 py-1.5 transition-colors hover:bg-accent/40"
              >
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-info-light text-info">
                  <Icon className="h-4 w-4" />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-xs font-medium text-foreground">{att.file_ref}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {formatBytes(att.size)} · {libFormatDate(att.created_at)}
                  </p>
                </div>
                <button
                  onClick={() => handleDelete(att.id)}
                  disabled={deleting}
                  className="shrink-0 rounded p-1 text-muted-foreground opacity-0 transition-opacity hover:bg-accent hover:text-destructive group-hover:opacity-100 focus-visible:opacity-100 disabled:opacity-100"
                  title={t('common.delete')}
                >
                  {deleting ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <Trash2 className="h-3.5 w-3.5" />
                  )}
                </button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
