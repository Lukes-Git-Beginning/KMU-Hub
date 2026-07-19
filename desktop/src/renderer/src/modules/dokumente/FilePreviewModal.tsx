/**
 * Document viewer — the "open the document like a document" view.
 *
 * Near-fullscreen dialog with real content rendering:
 * - Images (image/*): <img> with presigned URL
 * - PDF (application/pdf): embedded <iframe> (Electron supports natively)
 * - Text/Markdown (text/*, application/json): fetched and rendered in <pre>
 * - Unsupported: file info with download hint
 *
 * Toolbar carries the common actions (download, rename, share, versions);
 * an info sidebar (collapsed by default) shows details, tags and the
 * activity trail (who uploaded/renamed/moved/downloaded — mock-first).
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
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
  Info,
  Pencil,
  Share2,
  History,
  Plus,
  X,
  Tag,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  useFileDownloadURL,
  useFileActivity,
  useDocumentTags,
  useTagFile,
  useUntagFile,
} from '@/api/hooks/useDocuments'
import type { DocumentFile, DocumentActivityAction } from '@/api/types/document-types'
import { formatDate } from '@/lib/format'
import { downloadDocumentFile } from './download'
import { useHasCapability, useScopedCapability } from '@/hooks/useCapability'

interface FilePreviewModalProps {
  file: DocumentFile | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onRename?: (file: DocumentFile) => void
  onShare?: (file: DocumentFile) => void
  onVersionHistory?: (file: DocumentFile) => void
}

function formatBytes(bytes: number): string {
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(1)} GB`
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(1)} MB`
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${bytes} B`
}

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
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

const ACTIVITY_LABEL_KEYS = {
  uploaded: 'dokumente.activity.uploaded',
  renamed: 'dokumente.activity.renamed',
  moved: 'dokumente.activity.moved',
  copied: 'dokumente.activity.copied',
  downloaded: 'dokumente.activity.downloaded',
  shared: 'dokumente.activity.shared',
  version_created: 'dokumente.activity.versionCreated',
  reverted: 'dokumente.activity.reverted',
} as const satisfies Record<DocumentActivityAction, string>

function TextPreview({ url }: { url: string }) {
  const { t } = useTranslation()
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
          {t('dokumente.preview.textLoadError')}
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
    <pre className="overflow-auto p-6 text-sm text-foreground font-mono whitespace-pre-wrap break-words">
      {content}
    </pre>
  )
}

/** Info sidebar: details, tags (add/remove) and the activity trail. */
function ViewerInfoPanel({ file }: { file: DocumentFile }) {
  const { t } = useTranslation()
  const [tagPickerOpen, setTagPickerOpen] = useState(false)
  const { data: activityData } = useFileActivity(file.id)
  const { data: allTagsData } = useDocumentTags()
  const tagFile = useTagFile()
  const untagFile = useUntagFile()

  // Tag-Aktionen brauchen edit-Recht
  const canEditTags = useScopedCapability('documents:file:edit', file.owner_id)

  const activities = activityData?.activities ?? []
  const fileTagIds = new Set((file.tags ?? []).map((tg) => tg.id))
  const availableTags = (allTagsData?.tags ?? []).filter((tg) => !fileTagIds.has(tg.id))

  const detailRows: { labelKey: string; value: string }[] = [
    { labelKey: 'dokumente.viewer.type', value: file.mime_type },
    { labelKey: 'dokumente.viewer.size', value: formatBytes(file.file_size) },
    { labelKey: 'dokumente.viewer.created', value: formatDate(file.created_at) },
    { labelKey: 'dokumente.viewer.modified', value: formatDate(file.updated_at) },
    { labelKey: 'dokumente.viewer.version', value: `v${file.current_version ?? 1}` },
  ]

  return (
    <aside className="flex w-80 shrink-0 flex-col gap-5 overflow-y-auto border-l border-border bg-card/50 p-4">
      {/* Details */}
      <section>
        <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          {t('dokumente.viewer.details')}
        </h3>
        <dl className="space-y-1.5">
          {detailRows.map((row) => (
            <div key={row.labelKey} className="flex items-baseline justify-between gap-3 text-sm">
              <dt className="shrink-0 text-muted-foreground">{t(row.labelKey as 'dokumente.viewer.type')}</dt>
              <dd className="truncate text-right text-foreground" title={row.value}>
                {row.value}
              </dd>
            </div>
          ))}
        </dl>
      </section>

      {/* Tags */}
      <section>
        <h3 className="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          <Tag className="h-3 w-3" />
          {t('dokumente.viewer.tags')}
        </h3>
        <div className="flex flex-wrap items-center gap-1.5">
          {(file.tags ?? []).map((tag) => (
            <span
              key={tag.id}
              className="flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-xs text-foreground"
            >
              {tag.name}
              {canEditTags && (
                <button
                  onClick={() =>
                    untagFile.mutate(
                      { fileId: file.id, tagId: tag.id },
                      { onError: (err) => toast.error(`${t('common.error')}: ${err.message}`) },
                    )
                  }
                  className="text-muted-foreground hover:text-foreground"
                  aria-label={t('common.delete')}
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </span>
          ))}
          {canEditTags && (
            <button
              onClick={() => setTagPickerOpen((o) => !o)}
              className="flex items-center gap-1 rounded-full border border-dashed border-border px-2 py-0.5 text-xs text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
            >
              <Plus className="h-3 w-3" />
              {t('dokumente.viewer.addTag')}
            </button>
          )}
        </div>
        {canEditTags && tagPickerOpen && (
          <div className="mt-2 flex flex-wrap gap-1.5">
            {availableTags.length === 0 ? (
              <p className="text-xs text-muted-foreground">{t('dokumente.viewer.noMoreTags')}</p>
            ) : (
              availableTags.map((tag) => (
                <button
                  key={tag.id}
                  onClick={() => {
                    tagFile.mutate(
                      { fileId: file.id, tagId: tag.id },
                      { onError: (err) => toast.error(`${t('common.error')}: ${err.message}`) },
                    )
                    setTagPickerOpen(false)
                  }}
                  className="rounded-full bg-secondary/60 px-2 py-0.5 text-xs text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
                >
                  {tag.name}
                </button>
              ))
            )}
          </div>
        )}
      </section>

      {/* Activity trail */}
      <section className="min-h-0">
        <h3 className="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          <History className="h-3 w-3" />
          {t('dokumente.viewer.activity')}
        </h3>
        {activities.length === 0 ? (
          <p className="text-xs text-muted-foreground">{t('dokumente.viewer.noActivity')}</p>
        ) : (
          <ol className="space-y-2.5">
            {activities.map((act) => {
              const labelKey =
                ACTIVITY_LABEL_KEYS[act.action as DocumentActivityAction] ??
                ACTIVITY_LABEL_KEYS.uploaded
              return (
                <li key={act.id} className="text-xs leading-relaxed">
                  <span className="font-medium text-foreground">{act.actor_name}</span>{' '}
                  <span className="text-muted-foreground">{t(labelKey)}</span>
                  {act.detail && (
                    <span className="text-muted-foreground"> · {act.detail}</span>
                  )}
                  <span className="block text-[11px] text-muted-foreground/70">
                    {formatDateTime(act.created_at)}
                  </span>
                </li>
              )
            })}
          </ol>
        )}
      </section>
    </aside>
  )
}

export function FilePreviewModal({
  file,
  open,
  onOpenChange,
  onRename,
  onShare,
  onVersionHistory,
}: FilePreviewModalProps) {
  const { t } = useTranslation()
  const [infoOpen, setInfoOpen] = useState(false)
  const { data: downloadData, isLoading: urlLoading } = useFileDownloadURL(
    file?.id ?? '',
  )
  const presignedURL = downloadData?.url

  // RBAC checks (called unconditionally — hooks must not be conditional)
  const canDownload = useHasCapability('documents:file:download')
  const canEdit = useScopedCapability('documents:file:edit', file?.owner_id)
  const canShare = useHasCapability('documents:share:manage')
  const canVersionRestore = useHasCapability('documents:version:restore')

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- info panel starts collapsed per document
    if (open) setInfoOpen(false)
  }, [open, file?.id])

  if (!file) return null

  const Icon = getFileIcon(file.mime_type)
  const category = getMimeCategory(file.mime_type)

  const handleDownload = () => {
    downloadDocumentFile(file.id, file.filename)
      .then(() => toast.success(t('dokumente.downloading', { name: file.filename })))
      .catch((err: Error) => toast.error(`${t('common.error')}: ${err.message}`))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[92vh] w-[92vw] max-w-[92vw] flex-col overflow-hidden p-0 gap-0">
        {/* Header: title + toolbar */}
        <DialogHeader className="flex-row items-center gap-3 space-y-0 border-b border-border px-5 py-3 pr-12">
          <DialogTitle className="flex min-w-0 flex-1 items-center gap-2 text-sm">
            {/* eslint-disable-next-line react-hooks/static-components -- Icon is a dynamic component variable */}
            <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
            <span className="truncate">{file.filename}</span>
            <span className="shrink-0 text-xs font-normal text-muted-foreground">
              {formatBytes(file.file_size)}
            </span>
          </DialogTitle>
          <div className="flex shrink-0 items-center gap-1">
            {/* Download: Ausnahme-Muster — immer sichtbar, ohne Recht deaktiviert */}
            {canDownload ? (
              <Button variant="ghost" size="sm" onClick={handleDownload}>
                <Download className="mr-1.5 h-3.5 w-3.5" />
                {t('common.download')}
              </Button>
            ) : (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="inline-flex" tabIndex={0}>
                    <Button variant="ghost" size="sm" disabled className="pointer-events-none">
                      <Download className="mr-1.5 h-3.5 w-3.5" />
                      {t('common.download')}
                    </Button>
                  </span>
                </TooltipTrigger>
                <TooltipContent>{t('rbac.gate.downloadDisabled')}</TooltipContent>
              </Tooltip>
            )}
            {onRename && canEdit && (
              <Button variant="ghost" size="sm" onClick={() => onRename(file)}>
                <Pencil className="mr-1.5 h-3.5 w-3.5" />
                {t('dokumente.context.rename')}
              </Button>
            )}
            {onShare && canShare && (
              <Button variant="ghost" size="sm" onClick={() => onShare(file)}>
                <Share2 className="mr-1.5 h-3.5 w-3.5" />
                {t('dokumente.context.share')}
              </Button>
            )}
            {onVersionHistory && canVersionRestore && (
              <Button variant="ghost" size="sm" onClick={() => onVersionHistory(file)}>
                <History className="mr-1.5 h-3.5 w-3.5" />
                {t('dokumente.context.versionHistory')}
              </Button>
            )}
            <Button
              variant={infoOpen ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => setInfoOpen((o) => !o)}
              aria-pressed={infoOpen}
            >
              <Info className="mr-1.5 h-3.5 w-3.5" />
              {t('dokumente.viewer.info')}
            </Button>
          </div>
        </DialogHeader>

        {/* Body: preview + collapsible info sidebar */}
        <div className="flex min-h-0 flex-1">
          <div className="min-w-0 flex-1 overflow-y-auto bg-secondary/30">
            {urlLoading ? (
              <div className="flex h-full items-center justify-center p-8">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            ) : category === 'image' && presignedURL ? (
              <div className="flex h-full items-center justify-center p-6">
                <img
                  src={presignedURL}
                  alt={file.filename}
                  className="max-h-full max-w-full rounded object-contain"
                />
              </div>
            ) : category === 'pdf' && presignedURL ? (
              <iframe
                src={presignedURL}
                title={file.filename}
                className="h-full w-full border-0"
              />
            ) : category === 'text' && presignedURL ? (
              <TextPreview url={presignedURL} />
            ) : category === 'video' && presignedURL ? (
              <div className="flex h-full items-center justify-center p-6">
                <video src={presignedURL} controls className="max-h-full max-w-full rounded">
                  {t('dokumente.preview.videoNotSupported')}
                </video>
              </div>
            ) : (
              <div className="flex h-full items-center justify-center p-8">
                <div className="w-full max-w-lg text-center">
                  {/* eslint-disable-next-line react-hooks/static-components -- Icon is a dynamic component variable */}
                  <Icon className="mx-auto mb-4 h-16 w-16 text-muted-foreground opacity-40" />
                  <p className="mb-1 text-sm font-medium text-foreground">{file.filename}</p>
                  <p className="mb-4 text-xs text-muted-foreground">
                    {file.mime_type} &middot; {formatBytes(file.file_size)}
                  </p>
                  <p className="text-xs italic text-muted-foreground">
                    {t('dokumente.preview.notAvailable')}
                  </p>
                </div>
              </div>
            )}
          </div>
          {infoOpen && <ViewerInfoPanel file={file} />}
        </div>
      </DialogContent>
    </Dialog>
  )
}
