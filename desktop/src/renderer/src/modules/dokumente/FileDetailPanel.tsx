/**
 * File detail slide-over panel connected to real API.
 *
 * Shows: file metadata, tags (with add/remove), entity links,
 * sharing status, version count. Uses hooks from useDocuments.ts.
 */
import { useTranslation } from 'react-i18next'
import {
  FileText,
  FileSpreadsheet,
  Image,
  Film,
  Archive,
  File,
  Calendar,
  Tag,
  History,
  Users,
  Download,
  Share2,
  Star,
  Pencil,
  Trash2,
  Link2,
  HardDrive,
  Plus,
  X,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { toast } from 'sonner'
import { DetailPanel } from '@/components/shared'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import {
  useShares,
  useFileVersions,
  useFileLinks,
  useDocumentTags,
  useTagFile,
  useUntagFile,
} from '@/api/hooks/useDocuments'
import type { DocumentFile } from '@/api/types/document-types'
import { formatDate } from '@/lib/format'
import { downloadDocumentFile } from './download'
import { useHasCapability, useScopedCapability } from '@/hooks/useCapability'

interface FileDetailPanelProps {
  file: DocumentFile | null
  open: boolean
  onClose: () => void
  onPreview: (file: DocumentFile) => void
  onRename: (file: DocumentFile) => void
  onShare: (file: DocumentFile) => void
  onDelete: (fileId: string) => void
  onToggleFavorite: (fileId: string) => void
  onVersionHistory: (fileId: string) => void
}

function formatBytes(bytes: number): string {
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(1)} GB`
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(1)} MB`
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${bytes} B`
}

function getMimeIcon(mimeType: string) {
  if (mimeType.startsWith('image/')) return Image
  if (mimeType === 'application/pdf') return FileText
  if (mimeType.startsWith('video/')) return Film
  if (mimeType.includes('spreadsheet') || mimeType.includes('excel'))
    return FileSpreadsheet
  if (mimeType.includes('zip') || mimeType.includes('archive')) return Archive
  return File
}

// Return type stays the literal-key union so the typed t() accepts it.
function getMimeLabelKey(mimeType: string) {
  if (mimeType.startsWith('image/')) return 'dokumente.mimeLabel.image'
  if (mimeType === 'application/pdf') return 'dokumente.mimeLabel.pdf'
  if (mimeType.startsWith('video/')) return 'dokumente.mimeLabel.video'
  if (mimeType.includes('word') || mimeType.includes('document'))
    return 'dokumente.mimeLabel.word'
  if (mimeType.includes('spreadsheet') || mimeType.includes('excel'))
    return 'dokumente.mimeLabel.excel'
  if (mimeType.includes('zip') || mimeType.includes('archive')) return 'dokumente.mimeLabel.archive'
  return 'dokumente.mimeLabel.file'
}

export function FileDetailPanel({
  file,
  open,
  onClose,
  onPreview,
  onRename,
  onShare,
  onDelete,
  onToggleFavorite,
  onVersionHistory,
}: FileDetailPanelProps) {
  const { t } = useTranslation()

  // RBAC checks (unconditional — hooks always called)
  const canDownload = useHasCapability('documents:file:download')
  const canEdit = useScopedCapability('documents:file:edit', file?.owner_id)
  const canDelete = useScopedCapability('documents:file:delete', file?.owner_id)
  const canShare = useHasCapability('documents:share:manage')
  const canVersionRestore = useHasCapability('documents:version:restore')

  // API data
  const { data: sharesData } = useShares('file', file?.id ?? '')
  const { data: versionsData } = useFileVersions(file?.id ?? '')
  const { data: linksData } = useFileLinks(file?.id ?? '')
  const { data: allTagsData } = useDocumentTags()
  const tagFile = useTagFile()
  const untagFile = useUntagFile()

  if (!file) return null

  const Icon = getMimeIcon(file.mime_type)
  const shares = sharesData?.shares ?? []
  const versions = versionsData?.versions ?? []
  const links = linksData?.links ?? []
  const allTags = allTagsData?.tags ?? []
  const fileTags = file.tags ?? []
  const fileTagIds = new Set(fileTags.map((t) => t.id))
  const availableTags = allTags.filter((t) => !fileTagIds.has(t.id))

  return (
    <DetailPanel
      open={open}
      onClose={onClose}
      title={file.filename}
      subtitle={t(getMimeLabelKey(file.mime_type))}
      badge={
        <div className="flex items-center gap-1.5 ml-2">
          {file.is_favorite && (
            <Star className="h-3.5 w-3.5 fill-warning text-warning" />
          )}
        </div>
      }
      footer={
        <div className="flex gap-2">
          <Button
            variant="outline"
            className="flex-1"
            onClick={() => onPreview(file)}
          >
            <FileText className="mr-1.5 h-4 w-4" />
            {t('dokumente.detail.preview')}
          </Button>
          {canEdit && (
            <Button
              variant="outline"
              size="icon"
              onClick={() => onToggleFavorite(file.id)}
            >
              <Star
                className={`h-4 w-4 ${
                  file.is_favorite
                    ? 'fill-warning text-warning'
                    : ''
                }`}
              />
            </Button>
          )}
          {canDelete && (
            <Button
              variant="outline"
              size="icon"
              className="text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={() => onDelete(file.id)}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        </div>
      }
    >
      {/* Metadata */}
      <div className="space-y-3">
        <div className="flex items-center gap-2 text-sm">
          {/* eslint-disable-next-line react-hooks/static-components -- Icon is a dynamic component variable */}
          <Icon className="h-4 w-4 text-muted-foreground" />
          <span className="text-foreground">
            {t(getMimeLabelKey(file.mime_type))} &middot;{' '}
            {formatBytes(file.file_size)}
          </span>
        </div>
        <div className="flex items-center gap-2 text-sm">
          <Calendar className="h-4 w-4 text-muted-foreground" />
          <span className="text-foreground">
            {formatDate(file.created_at, { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })}
          </span>
        </div>
        <div className="flex items-center gap-2 text-sm">
          <HardDrive className="h-4 w-4 text-muted-foreground" />
          <span className="text-foreground">
            Version {file.current_version}
          </span>
        </div>
      </div>

      {/* Quick actions */}
      <div className="flex flex-wrap gap-2 mt-4">
        {canEdit && (
          <Button variant="outline" size="sm" onClick={() => onRename(file)}>
            <Pencil className="mr-1.5 h-3.5 w-3.5" />
            {t('dokumente.context.rename')}
          </Button>
        )}
        {canShare && (
          <Button variant="outline" size="sm" onClick={() => onShare(file)}>
            <Share2 className="mr-1.5 h-3.5 w-3.5" />
            {t('dokumente.context.share')}
          </Button>
        )}
        {canVersionRestore && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => onVersionHistory(file.id)}
          >
            <History className="mr-1.5 h-3.5 w-3.5" />
            {t('dokumente.detail.versions', { count: versions.length })}
          </Button>
        )}
        {/* Download: Ausnahme-Muster — immer sichtbar, ohne Recht deaktiviert */}
        {canDownload ? (
          <Button
            variant="outline"
            size="sm"
            onClick={() =>
              downloadDocumentFile(file.id, file.filename)
                .then(() => toast.success(t('dokumente.downloading', { name: file.filename })))
                .catch((err: Error) => toast.error(`${t('common.error')}: ${err.message}`))
            }
          >
            <Download className="mr-1.5 h-3.5 w-3.5" />
            {t('common.download')}
          </Button>
        ) : (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="inline-flex" tabIndex={0}>
                <Button variant="outline" size="sm" disabled className="pointer-events-none">
                  <Download className="mr-1.5 h-3.5 w-3.5" />
                  {t('common.download')}
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>{t('rbac.gate.downloadDisabled')}</TooltipContent>
          </Tooltip>
        )}
      </div>

      {/* Tags */}
      <Separator className="my-4" />
      <div>
        <h4 className="mb-2 text-xs font-medium uppercase text-muted-foreground flex items-center gap-1">
          <Tag className="h-3.5 w-3.5" />
          Tags
        </h4>
        <div className="flex flex-wrap gap-1.5">
          {fileTags.map((tag) => (
            canEdit ? (
              <Badge
                key={tag.id}
                variant="secondary"
                className="text-xs group cursor-pointer"
                onClick={() =>
                  untagFile.mutate({ fileId: file.id, tagId: tag.id })
                }
              >
                {tag.name}
                <X className="ml-1 h-3 w-3 opacity-0 group-hover:opacity-100 transition-opacity" />
              </Badge>
            ) : (
              <Badge key={tag.id} variant="secondary" className="text-xs">
                {tag.name}
              </Badge>
            )
          ))}
          {canEdit && availableTags.length > 0 && (
            <Badge
              variant="outline"
              className="text-xs cursor-pointer hover:bg-secondary"
              onClick={() => {
                // Add first available tag as quick action
                const first = availableTags[0]
                if (first) tagFile.mutate({ fileId: file.id, tagId: first.id })
              }}
            >
              <Plus className="mr-0.5 h-3 w-3" />
              Tag
            </Badge>
          )}
        </div>
      </div>

      {/* Shared with */}
      {shares.length > 0 && (
        <>
          <Separator className="my-4" />
          <div>
            <h4 className="mb-2 text-xs font-medium uppercase text-muted-foreground flex items-center gap-1">
              <Users className="h-3.5 w-3.5" />
              {t('dokumente.detail.sharedWith', { count: shares.length })}
            </h4>
            <div className="space-y-1.5">
              {shares.map((s) => (
                <div
                  key={s.id}
                  className="flex items-center justify-between rounded-md border border-border px-3 py-2"
                >
                  <span className="text-sm text-foreground">
                    {s.shared_with_user_name}
                  </span>
                  <Badge variant="outline" className="text-xs">
                    {s.permission === 'write' ? t('dokumente.detail.permissionWrite') : t('dokumente.detail.permissionRead')}
                  </Badge>
                </div>
              ))}
            </div>
          </div>
        </>
      )}

      {/* Entity links */}
      {links.length > 0 && (
        <>
          <Separator className="my-4" />
          <div>
            <h4 className="mb-2 text-xs font-medium uppercase text-muted-foreground flex items-center gap-1">
              <Link2 className="h-3.5 w-3.5" />
              {t('dokumente.detail.links', { count: links.length })}
            </h4>
            <div className="space-y-1.5">
              {links.map((link) => (
                <div
                  key={link.id}
                  className="flex items-center justify-between rounded-md border border-border px-3 py-2"
                >
                  <div>
                    <p className="text-sm text-foreground">
                      {link.entity_name}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {link.entity_type}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </DetailPanel>
  )
}
