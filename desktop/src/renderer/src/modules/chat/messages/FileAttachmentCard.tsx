/**
 * Displays an attached file in the message area.
 *
 * Images show a thumbnail preview (a generated placeholder for server files
 * that have no local blob). Documents show an icon with filename and size.
 * Pending attachments have a remove button; sent attachments have a download
 * button that produces a real file (demo blobs, since there are no real bytes).
 */
import { useTranslation } from 'react-i18next'
import { FileText, Image as ImageIcon, X, Download } from 'lucide-react'
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

/** A deterministic SVG placeholder preview for server images without a local blob. */
function placeholderPreview(name: string): string {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="200"><rect width="100%" height="100%" fill="#e2e8f0"/><text x="50%" y="50%" font-family="sans-serif" font-size="16" fill="#64748b" text-anchor="middle" dominant-baseline="middle">${name}</text></svg>`
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`
}

/** Produce a real downloadable file. Demo mode has no real bytes, so generate a placeholder blob. */
function downloadDemoFile(file: AttachedFile): void {
  let blob: Blob
  if (file.isImage) {
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="640" height="480"><rect width="100%" height="100%" fill="#e2e8f0"/><text x="50%" y="50%" font-family="sans-serif" font-size="24" fill="#475569" text-anchor="middle" dominant-baseline="middle">${file.name}</text></svg>`
    blob = new Blob([svg], { type: 'image/svg+xml' })
  } else {
    blob = new Blob(
      [
        `Cosmi Demo-Datei\n\nDateiname: ${file.name}\nTyp: ${file.type || 'unbekannt'}\nGröße: ${file.size} Bytes\n\nIn der Demo enthalten Anhänge Platzhalter-Inhalte.`,
      ],
      { type: 'text/plain' },
    )
  }
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = file.name || 'datei'
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

export function FileAttachmentCard({ file, onRemove, compact }: FileAttachmentCardProps) {
  const { t } = useTranslation()
  // A sent attachment (no remove handler) can be downloaded.
  const canDownload = !onRemove

  if (file.isImage) {
    const src = file.previewUrl ?? placeholderPreview(file.name)
    return (
      <div className="group relative inline-block">
        <img
          src={src}
          alt={file.name}
          loading="lazy"
          decoding="async"
          className={`rounded-lg border border-border object-cover ${
            compact ? 'h-16 w-16' : 'max-h-48 max-w-xs'
          } ${canDownload ? 'cursor-pointer' : ''}`}
          onClick={canDownload ? () => downloadDemoFile(file) : undefined}
        />
        {canDownload && (
          <button
            onClick={() => downloadDemoFile(file)}
            className="absolute bottom-1 right-1 flex h-6 w-6 items-center justify-center rounded-md bg-card/90 text-foreground opacity-0 shadow-sm transition-opacity group-hover:opacity-100"
            aria-label={t('chat.files.download')}
          >
            <Download className="h-3.5 w-3.5" />
          </button>
        )}
        {onRemove && (
          <button
            onClick={onRemove}
            className="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-destructive-foreground shadow-sm opacity-0 transition-opacity group-hover:opacity-100"
            aria-label={t('chat.files.remove')}
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
      {canDownload && (
        <button
          onClick={() => downloadDemoFile(file)}
          className="ml-1 flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-primary/10 hover:text-primary"
          aria-label={t('chat.files.download')}
        >
          <Download className="h-3.5 w-3.5" />
        </button>
      )}
      {onRemove && (
        <button
          onClick={onRemove}
          className="ml-1 flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
          aria-label={t('common.remove')}
        >
          <X className="h-3 w-3" />
        </button>
      )}
    </div>
  )
}
