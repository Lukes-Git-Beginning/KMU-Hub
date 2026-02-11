import {
  FileText,
  FileSpreadsheet,
  Image,
  Film,
  Archive,
  File,
  Calendar,
  User,
  Tag,
  History,
  Users,
  Download,
  Share2,
  Lock,
  Star,
  Pencil,
  Trash2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { DetailPanel } from '@/components/shared'
import type { DocFile } from '@/stores/documents'

interface FileDetailPanelProps {
  file: DocFile | null
  open: boolean
  onClose: () => void
  onPreview: (file: DocFile) => void
  onRename: (file: DocFile) => void
  onShare: (file: DocFile) => void
  onDelete: (fileId: string) => void
  onToggleFavorite: (fileId: string) => void
}

const typeIcons: Record<string, typeof FileText> = {
  pdf: FileText,
  word: FileText,
  excel: FileSpreadsheet,
  image: Image,
  video: Film,
  archive: Archive,
  other: File,
}

const typeLabels: Record<string, string> = {
  pdf: 'PDF-Dokument',
  word: 'Word-Dokument',
  excel: 'Excel-Tabelle',
  image: 'Bilddatei',
  video: 'Videodatei',
  archive: 'Archiv',
  other: 'Datei',
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
}: FileDetailPanelProps) {
  if (!file) return null

  const Icon = typeIcons[file.type] || File

  return (
    <DetailPanel
      open={open}
      onClose={onClose}
      title={file.name}
      subtitle={typeLabels[file.type]}
      badge={
        <div className="flex items-center gap-1.5 ml-2">
          {file.isVault && (
            <Badge variant="outline" className="text-xs">
              <Lock className="mr-1 h-3 w-3 text-green-500" />
              Tresor
            </Badge>
          )}
          {file.isShared && (
            <Badge variant="outline" className="text-xs">
              <Share2 className="mr-1 h-3 w-3 text-blue-500" />
              Geteilt
            </Badge>
          )}
        </div>
      }
      footer={
        <div className="flex gap-2">
          <Button variant="outline" className="flex-1" onClick={() => onPreview(file)}>
            <FileText className="mr-1.5 h-4 w-4" />
            Vorschau
          </Button>
          <Button variant="outline" size="icon" onClick={() => onToggleFavorite(file.id)}>
            <Star className={`h-4 w-4 ${file.isFavorite ? 'fill-yellow-400 text-yellow-400' : ''}`} />
          </Button>
          <Button
            variant="outline"
            size="icon"
            className="text-red-500 hover:text-red-600 hover:bg-red-50"
            onClick={() => onDelete(file.id)}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      }
    >
      {/* Metadata */}
      <div className="space-y-3">
        <div className="flex items-center gap-2 text-sm">
          <Icon className="h-4 w-4 text-muted-foreground" />
          <span className="text-foreground">{typeLabels[file.type]} · {file.size}</span>
        </div>
        <div className="flex items-center gap-2 text-sm">
          <Calendar className="h-4 w-4 text-muted-foreground" />
          <span className="text-foreground">
            {new Date(file.date).toLocaleDateString('de-CH', {
              weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
            })}
          </span>
        </div>
        <div className="flex items-center gap-2 text-sm">
          <User className="h-4 w-4 text-muted-foreground" />
          <span className="text-foreground">Erstellt von {file.createdBy}</span>
        </div>
      </div>

      {/* Quick actions */}
      <div className="flex flex-wrap gap-2 mt-4">
        <Button variant="outline" size="sm" onClick={() => onRename(file)}>
          <Pencil className="mr-1.5 h-3.5 w-3.5" />
          Umbenennen
        </Button>
        <Button variant="outline" size="sm" onClick={() => onShare(file)}>
          <Share2 className="mr-1.5 h-3.5 w-3.5" />
          Teilen
        </Button>
        <Button variant="outline" size="sm">
          <Download className="mr-1.5 h-3.5 w-3.5" />
          Herunterladen
        </Button>
      </div>

      {/* Tags */}
      {file.tags.length > 0 && (
        <>
          <Separator className="my-4" />
          <div>
            <h4 className="mb-2 text-xs font-medium uppercase text-muted-foreground flex items-center gap-1">
              <Tag className="h-3.5 w-3.5" />
              Tags
            </h4>
            <div className="flex flex-wrap gap-1.5">
              {file.tags.map((tag) => (
                <Badge key={tag} variant="secondary" className="text-xs">{tag}</Badge>
              ))}
            </div>
          </div>
        </>
      )}

      {/* Shared with */}
      {file.sharedWith.length > 0 && (
        <>
          <Separator className="my-4" />
          <div>
            <h4 className="mb-2 text-xs font-medium uppercase text-muted-foreground flex items-center gap-1">
              <Users className="h-3.5 w-3.5" />
              Geteilt mit ({file.sharedWith.length})
            </h4>
            <div className="space-y-1.5">
              {file.sharedWith.map((s) => (
                <div key={s.name} className="flex items-center justify-between rounded-md border border-border px-3 py-2">
                  <span className="text-sm text-foreground">{s.name}</span>
                  <Badge variant="outline" className="text-xs">
                    {s.permission === 'edit' ? 'Bearbeiten' : 'Ansehen'}
                  </Badge>
                </div>
              ))}
            </div>
          </div>
        </>
      )}

      {/* Version history */}
      {file.versions.length > 0 && (
        <>
          <Separator className="my-4" />
          <div>
            <h4 className="mb-2 text-xs font-medium uppercase text-muted-foreground flex items-center gap-1">
              <History className="h-3.5 w-3.5" />
              Versionen ({file.versions.length})
            </h4>
            <div className="space-y-2">
              {file.versions.map((v, i) => (
                <div key={v.version} className="flex items-center gap-3">
                  <div className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-medium ${
                    i === 0 ? 'bg-primary text-primary-foreground' : 'bg-secondary text-muted-foreground'
                  }`}>
                    {v.version}
                  </div>
                  <div>
                    <p className="text-sm text-foreground">{v.author}</p>
                    <p className="text-xs text-muted-foreground">
                      {new Date(v.date).toLocaleDateString('de-CH')}
                      {i === 0 && ' · Aktuell'}
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
