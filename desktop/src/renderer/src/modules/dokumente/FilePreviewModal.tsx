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
} from 'lucide-react'
import type { DocFile } from '@/stores/documents'

interface FilePreviewModalProps {
  file: DocFile | null
  open: boolean
  onOpenChange: (open: boolean) => void
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

const typeColors: Record<string, string> = {
  pdf: 'bg-red-50 text-red-600 dark:bg-red-950 dark:text-red-400',
  word: 'bg-blue-50 text-blue-600 dark:bg-blue-950 dark:text-blue-400',
  excel: 'bg-green-50 text-green-600 dark:bg-green-950 dark:text-green-400',
  image: 'bg-purple-50 text-purple-600 dark:bg-purple-950 dark:text-purple-400',
  video: 'bg-pink-50 text-pink-600 dark:bg-pink-950 dark:text-pink-400',
  archive: 'bg-amber-50 text-amber-600 dark:bg-amber-950 dark:text-amber-400',
  other: 'bg-gray-50 text-gray-600 dark:bg-gray-950 dark:text-gray-400',
}

export function FilePreviewModal({ file, open, onOpenChange }: FilePreviewModalProps) {
  if (!file) return null

  const Icon = typeIcons[file.type] || File
  const colorClass = typeColors[file.type] || typeColors.other

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 pr-8">
            <Icon className="h-5 w-5 shrink-0" />
            <span className="truncate">{file.name}</span>
          </DialogTitle>
        </DialogHeader>

        {/* Preview area */}
        <div className="flex-1 overflow-y-auto rounded-lg border border-border bg-secondary/30 min-h-[300px]">
          {file.type === 'image' ? (
            <div className="flex items-center justify-center p-8">
              <div className={`flex h-64 w-full items-center justify-center rounded-lg ${colorClass}`}>
                <div className="text-center">
                  <Image className="h-16 w-16 mx-auto mb-3 opacity-60" />
                  <p className="text-sm font-medium">Bildvorschau</p>
                  <p className="text-xs opacity-60 mt-1">{file.name}</p>
                </div>
              </div>
            </div>
          ) : file.type === 'pdf' ? (
            <div className="flex items-center justify-center p-8">
              <div className="w-full max-w-lg rounded-lg bg-white dark:bg-gray-800 shadow-sm p-8 text-center">
                <FileText className="h-16 w-16 mx-auto mb-4 text-red-500 opacity-60" />
                <p className="text-sm font-medium text-foreground mb-1">PDF-Dokument</p>
                <p className="text-xs text-muted-foreground mb-4">{file.name} · {file.size}</p>
                <p className="text-xs text-muted-foreground italic">
                  PDF-Vorschau wird in der finalen Version unterstuetzt
                </p>
              </div>
            </div>
          ) : file.type === 'video' ? (
            <div className="flex items-center justify-center p-8">
              <div className="w-full max-w-lg rounded-lg bg-gray-900 p-8 text-center">
                <Film className="h-16 w-16 mx-auto mb-4 text-pink-400 opacity-60" />
                <p className="text-sm font-medium text-white mb-1">Video</p>
                <p className="text-xs text-gray-400 mb-4">{file.name} · {file.size}</p>
                <div className="h-2 rounded-full bg-gray-700 mx-auto max-w-xs">
                  <div className="h-full w-1/3 rounded-full bg-primary" />
                </div>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center p-8">
              <div className={`flex h-48 w-full max-w-lg items-center justify-center rounded-lg ${colorClass}`}>
                <div className="text-center">
                  <Icon className="h-16 w-16 mx-auto mb-3 opacity-60" />
                  <p className="text-sm font-medium">{file.name}</p>
                  <p className="text-xs opacity-60 mt-1">{file.size}</p>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer info */}
        <div className="flex items-center justify-between pt-2">
          <div className="text-xs text-muted-foreground">
            {file.size} · Erstellt von {file.createdBy} · {new Date(file.date).toLocaleDateString('de-CH')}
          </div>
          <Button variant="outline" size="sm">
            <Download className="mr-1.5 h-3.5 w-3.5" />
            Herunterladen
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
