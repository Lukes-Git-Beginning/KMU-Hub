/**
 * File preview modal with real content rendering.
 *
 * Supports:
 * - Images (image/*): Rendered via <img> with presigned URL
 * - PDF (application/pdf): Embedded <iframe> (Electron supports natively)
 * - Text/Markdown (text/*, application/json): Content fetched and rendered in <pre>
 * - Unsupported: File info with download button
 */
import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import {
  FileText,
  FileSpreadsheet,
  Image,
  Film,
  Archive,
  File,
  Download,
  Loader2,
} from 'lucide-react'
import { useFileDownloadURL } from '@/api/hooks/useDocuments'
import type { DocumentFile } from '@/api/types/document-types'

interface FilePreviewModalProps {
  file: DocumentFile | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

function formatBytes(bytes: number): string {
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(1)} GB`
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(1)} MB`
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${bytes} B`
}

function getMimeCategory(mimeType: string): string {
  if (mimeType.startsWith('image/')) return 'image'
  if (mimeType === 'application/pdf') return 'pdf'
  if (mimeType.startsWith('text/') || mimeType === 'application/json') return 'text'
  if (mimeType.startsWith('video/')) return 'video'
  return 'other'
}

function getFileIcon(mimeType: string) {
  const cat = getMimeCategory(mimeType)
  switch (cat) {
    case 'image':
      return Image
    case 'pdf':
      return FileText
    case 'video':
      return Film
    default:
      if (mimeType.includes('spreadsheet') || mimeType.includes('excel'))
        return FileSpreadsheet
      if (mimeType.includes('zip') || mimeType.includes('archive'))
        return Archive
      return File
  }
}

function TextPreview({ url }: { url: string }) {
  const [content, setContent] = useState<string | null>(null)
  const [error, setError] = useState(false)

  useEffect(() => {
    let cancelled = false
    fetch(url)
      .then((res) => res.text())
      .then((text) => {
        if (!cancelled) setContent(text)
      })
      .catch(() => {
        if (!cancelled) setError(true)
      })
    return () => {
      cancelled = true
    }
  }, [url])

  if (error) {
    return (
      <div className="flex items-center justify-center p-8">
        <p className="text-sm text-muted-foreground">
          Textvorschau konnte nicht geladen werden.
        </p>
      </div>
    )
  }

  if (content === null) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <pre className="overflow-auto p-4 text-sm text-foreground font-mono whitespace-pre-wrap break-words max-h-[60vh]">
      {content}
    </pre>
  )
}

export function FilePreviewModal({
  file,
  open,
  onOpenChange,
}: FilePreviewModalProps) {
  const { data: downloadData, isLoading: urlLoading } = useFileDownloadURL(
    file?.id ?? '',
  )
  const presignedURL = downloadData?.url

  if (!file) return null

  const Icon = getFileIcon(file.mime_type)
  const category = getMimeCategory(file.mime_type)

  const handleDownload = () => {
    if (presignedURL) {
      window.open(presignedURL, '_blank')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 pr-8">
            {/* eslint-disable-next-line react-hooks/static-components -- Icon is a dynamic component variable */}
            <Icon className="h-5 w-5 shrink-0" />
            <span className="truncate">{file.filename}</span>
          </DialogTitle>
        </DialogHeader>

        {/* Preview area */}
        <div className="flex-1 overflow-y-auto rounded-lg border border-border bg-secondary/30 min-h-[300px]">
          {urlLoading ? (
            <div className="flex items-center justify-center p-8 h-full">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : category === 'image' && presignedURL ? (
            <div className="flex items-center justify-center p-4">
              <img
                src={presignedURL}
                alt={file.filename}
                className="max-w-full max-h-[60vh] object-contain rounded"
              />
            </div>
          ) : category === 'pdf' && presignedURL ? (
            <iframe
              src={presignedURL}
              title={file.filename}
              className="w-full h-[60vh] border-0"
            />
          ) : category === 'text' && presignedURL ? (
            <TextPreview url={presignedURL} />
          ) : category === 'video' && presignedURL ? (
            <div className="flex items-center justify-center p-8">
              <video
                src={presignedURL}
                controls
                className="max-w-full max-h-[60vh] rounded"
              >
                Video wird nicht unterstuetzt
              </video>
            </div>
          ) : (
            <div className="flex items-center justify-center p-8">
              <div className="w-full max-w-lg text-center">
                {/* eslint-disable-next-line react-hooks/static-components -- Icon is a dynamic component variable */}
                <Icon className="h-16 w-16 mx-auto mb-4 text-muted-foreground opacity-40" />
                <p className="text-sm font-medium text-foreground mb-1">
                  {file.filename}
                </p>
                <p className="text-xs text-muted-foreground mb-4">
                  {file.mime_type} &middot; {formatBytes(file.file_size)}
                </p>
                <p className="text-xs text-muted-foreground italic">
                  Vorschau fuer diesen Dateityp nicht verfuegbar
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between pt-2">
          <div className="text-xs text-muted-foreground">
            {formatBytes(file.file_size)} &middot; Version{' '}
            {file.current_version} &middot;{' '}
            {new Date(file.created_at).toLocaleDateString('de-CH')}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={handleDownload}
            disabled={!presignedURL}
          >
            <Download className="mr-1.5 h-3.5 w-3.5" />
            Herunterladen
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
