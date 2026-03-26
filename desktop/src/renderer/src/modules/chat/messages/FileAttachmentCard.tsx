/**
 * Displays an attached file in the message area.
 *
 * Images show a thumbnail preview. Documents show an icon with
 * filename and size. Pending attachments have a remove button.
 */
import { FileText, Image as ImageIcon, X } from 'lucide-react'
import type { AttachedFile } from './FileDropZone'

interface FileAttachmentCardProps {
  file: AttachedFile
  onRemove?: () => void
  compact?: boolean
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function FileAttachmentCard({ file, onRemove, compact }: FileAttachmentCardProps) {
  if (file.isImage && file.previewUrl) {
    return (
      <div className="group relative inline-block">
        <img
          src={file.previewUrl}
          alt={file.name}
          loading="lazy"
          decoding="async"
          className={`rounded-lg border border-border object-cover ${
            compact ? 'h-16 w-16' : 'max-h-48 max-w-xs'
          }`}
        />
        {onRemove && (
          <button
            onClick={onRemove}
            className="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-destructive-foreground shadow-sm opacity-0 transition-opacity group-hover:opacity-100"
          >
            <X className="h-3 w-3" />
          </button>
        )}
      </div>
    )
  }

  return (
    <div className="group relative inline-flex items-center gap-2 rounded-lg border border-border bg-secondary/50 px-3 py-2">
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded bg-primary/10">
        {file.isImage ? (
          <ImageIcon className="h-4 w-4 text-primary" />
        ) : (
          <FileText className="h-4 w-4 text-primary" />
        )}
      </div>
      <div className="min-w-0">
        <p className="truncate text-xs font-medium text-foreground max-w-[160px]">{file.name}</p>
        <p className="text-[10px] text-muted-foreground">{formatFileSize(file.size)}</p>
      </div>
      {onRemove && (
        <button
          onClick={onRemove}
          className="ml-1 flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
        >
          <X className="h-3 w-3" />
        </button>
      )}
    </div>
  )
}
