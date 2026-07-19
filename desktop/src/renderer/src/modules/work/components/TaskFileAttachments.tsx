/**
 * Task file attachments list with upload capability.
 *
 * Displays attached files with icon, filename, size, and uploader.
 * Supports file upload via button (file picker) with progress indication.
 * Downloads via presigned MinIO URLs.
 */
import { useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Paperclip,
  FileText,
  Image,
  FileSpreadsheet,
  File,
  Download,
  Trash2,
  Upload,
  Loader2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  useTaskFiles,
  useAttachFile,
  useRemoveFile,
  getFileDownloadUrl,
  formatFileSize,
  type TaskFile,
} from '@/api/hooks/useTaskFiles'
import { useAuthStore } from '@/stores/auth'

interface TaskFileAttachmentsProps {
  taskId: string
  canEdit?: boolean
}

function getFileIcon(mimeType?: string) {
  if (!mimeType) return File
  if (mimeType.startsWith('image/')) return Image
  if (mimeType.includes('spreadsheet') || mimeType.includes('csv') || mimeType.includes('excel'))
    return FileSpreadsheet
  if (mimeType.includes('pdf') || mimeType.includes('document') || mimeType.includes('text'))
    return FileText
  return File
}

function formatRelativeTime(dateStr: string | undefined, t: (key: string, opts?: Record<string, unknown>) => string): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffDays === 0) return t('work.time.today')
  if (diffDays === 1) return t('work.time.yesterday')
  if (diffDays < 7) return t('work.time.daysAgo', { count: diffDays })
  return date.toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: '2-digit',
  })
}

export default function TaskFileAttachments({
  taskId,
  canEdit = true,
}: TaskFileAttachmentsProps) {
  const { t } = useTranslation()
  const currentUserId = useAuthStore((s) => s.user?.id)
  const { data, isLoading } = useTaskFiles(taskId)
  const attachFile = useAttachFile()
  const removeFile = useRemoveFile()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [downloading, setDownloading] = useState<string | null>(null)

  const files = data?.files ?? []

  async function handleFileSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const selectedFiles = e.target.files
    if (!selectedFiles || selectedFiles.length === 0) return

    for (const file of Array.from(selectedFiles)) {
      try {
        await attachFile.mutateAsync({ taskId, file })
      } catch {
        // Error handled by mutation state
      }
    }

    // Reset input
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  async function handleDownload(file: TaskFile) {
    if (!file.id) return
    setDownloading(file.id)
    try {
      const url = await getFileDownloadUrl(file.id)
      if (url) {
        window.open(url, '_blank')
      }
    } catch {
      // Error downloading
    } finally {
      setDownloading(null)
    }
  }

  async function handleRemove(fileId: string) {
    try {
      await removeFile.mutateAsync({ fileId, taskId })
    } catch {
      // Error handled by mutation state
    }
  }

  return (
    <div>
      {/* Upload area */}
      {canEdit && (
        <div className="mb-3">
          <input
            ref={fileInputRef}
            type="file"
            className="hidden"
            onChange={handleFileSelect}
            multiple
          />
          <Button
            variant="outline"
            size="sm"
            className="w-full gap-2"
            onClick={() => fileInputRef.current?.click()}
            disabled={attachFile.isPending}
          >
            {attachFile.isPending ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                {t('work.files.uploading')}
              </>
            ) : (
              <>
                <Upload className="h-4 w-4" />
                {t('work.files.attach')}
              </>
            )}
          </Button>
        </div>
      )}

      {/* File list */}
      {isLoading ? (
        <div className="py-4 text-center text-sm text-muted-foreground">
          {t('work.files.loading')}
        </div>
      ) : files.length === 0 ? (
        <div className="flex flex-col items-center py-4 text-center">
          <Paperclip className="h-6 w-6 text-muted-foreground/30" />
          <p className="mt-1 text-xs text-muted-foreground">
            {t('work.files.empty')}
          </p>
        </div>
      ) : (
        <div className="space-y-1">
          {files.map((file) => {
            const Icon = getFileIcon(file.mime_type)
            const isOwn = file.uploaded_by === currentUserId
            const isDownloading = downloading === file.id

            return (
              <div
                key={file.id}
                className="group flex items-center gap-2 rounded-md border border-transparent px-2 py-1.5 transition-colors hover:bg-accent/50 hover:border-border"
              >
                <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                <div className="flex-1 min-w-0">
                  <button
                    type="button"
                    className="text-sm font-medium truncate block w-full text-left hover:underline cursor-pointer"
                    onClick={() => handleDownload(file)}
                    title={file.filename}
                  >
                    {file.filename}
                  </button>
                  <p className="text-xs text-muted-foreground">
                    {formatFileSize(file.file_size)}
                    {file.uploaded_by_name && (
                      <> &middot; {file.uploaded_by_name}</>
                    )}
                    {file.created_at && (
                      <> &middot; {formatRelativeTime(file.created_at, t)}</>
                    )}
                  </p>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    type="button"
                    className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                    onClick={() => handleDownload(file)}
                    title={t('common.download')}
                    disabled={isDownloading}
                  >
                    {isDownloading ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <Download className="h-3.5 w-3.5" />
                    )}
                  </button>
                  {isOwn && canEdit && (
                    <button
                      type="button"
                      className="rounded p-1 text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                      onClick={() => file.id && handleRemove(file.id)}
                      title={t('work.dependencies.remove')}
                      disabled={removeFile.isPending}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
