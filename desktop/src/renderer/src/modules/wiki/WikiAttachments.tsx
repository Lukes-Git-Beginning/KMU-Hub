import { useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Paperclip, Plus, Trash2, FileText, Image as ImageIcon, File, Loader2, Download } from 'lucide-react'
import {
  useAttachments,
  useUploadAttachment,
  useDeleteAttachment,
} from '@/api/hooks/useWiki'
import { formatDate as libFormatDate } from '@/lib/format'

// Read a File as a data URL so demo uploads can be downloaded/previewed later.
// Skip very large files to keep the (in-memory) mock payload reasonable.
const MAX_INLINE_BYTES = 5 * 1024 * 1024
function readAsDataUrl(file: File): Promise<string | undefined> {
  if (file.size > MAX_INLINE_BYTES) return Promise.resolve(undefined)
  return new Promise((resolve) => {
    const reader = new FileReader()
    reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : undefined)
    reader.onerror = () => resolve(undefined)
    reader.readAsDataURL(file)
  })
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// bytes may arrive as a string — protojson serializes int64 (size) as a JSON string.
function formatBytes(bytes: number | string): string {
  const n = Number(bytes)
  if (n >= 1073741824) return `${(n / 1073741824).toFixed(1)} GB`
  if (n >= 1048576) return `${(n / 1048576).toFixed(1)} MB`
  if (n >= 1024) return `${(n / 1024).toFixed(0)} KB`
  return `${n} B`
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
      const dataUrl = await readAsDataUrl(file)
      await uploadMutation.mutateAsync({
        articleId,
        file_ref: file.name,
        mime: file.type || 'application/octet-stream',
        size: file.size,
        data_url: dataUrl,
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

  // Trigger a real download — uses the stored data URL, or a generated demo blob
  // so the button is never dead for legacy/seed attachments without a blob.
  const handleDownload = (att: { file_ref: string; mime: string; size: number | string; data_url?: string }) => {
    let href = att.data_url
    let revoke = false
    if (!href) {
      const blob = new Blob(
        [`Demo-Anhang: ${att.file_ref}\nTyp: ${att.mime}\nGröße: ${att.size} Bytes`],
        { type: 'text/plain' },
      )
      href = URL.createObjectURL(blob)
      revoke = true
    }
    const a = document.createElement('a')
    a.href = href
    a.download = att.file_ref
    document.body.appendChild(a)
    a.click()
    a.remove()
    if (revoke) setTimeout(() => URL.revokeObjectURL(href as string), 1000)
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
            const isImage = att.mime.startsWith('image/') && !!att.data_url
            const deleting = deleteMutation.isPending && deleteMutation.variables?.attachmentId === att.id
            return (
              <div
                key={att.id}
                className="group flex items-center gap-2.5 rounded-md border border-border bg-card/40 px-2.5 py-1.5 transition-colors hover:bg-accent/40"
              >
                {isImage ? (
                  <img
                    src={att.data_url}
                    alt={att.file_ref}
                    className="h-8 w-10 shrink-0 rounded-md border border-border object-cover"
                  />
                ) : (
                  <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-info-light text-info">
                    <Icon className="h-4 w-4" />
                  </span>
                )}
                <div className="min-w-0 flex-1">
                  <p className="truncate text-xs font-medium text-foreground">{att.file_ref}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {formatBytes(att.size)} · {libFormatDate(att.created_at)}
                  </p>
                </div>
                <button
                  onClick={() => handleDownload(att)}
                  className="shrink-0 rounded p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                  title={t('wiki.attachments.download')}
                  aria-label={t('wiki.attachments.download')}
                >
                  <Download className="h-3.5 w-3.5" />
                </button>
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
