import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FolderDown } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { ReportDocument } from '@/api/berichte-types'
import { useSaveReportToDocuments } from '@/api/hooks/useBerichte'
import { useDocumentFolders } from '@/api/hooks/useDocuments'

interface SaveToDocumentsDialogProps {
  doc: ReportDocument
  open: boolean
  onClose: () => void
}

const DEFAULT_FOLDER = 'fld-berichte'

/** R-5b — file the report as a PDF into a documents folder (default "Berichte"). */
export function SaveToDocumentsDialog({ doc, open, onClose }: SaveToDocumentsDialogProps) {
  const { t } = useTranslation()
  const { data } = useDocumentFolders({ space_type: 'personal' })
  const save = useSaveReportToDocuments()
  const [folderId, setFolderId] = useState(DEFAULT_FOLDER)

  const folders = (data?.folders ?? []) as { id: string; name: string }[]

  // Default to the "Berichte" folder when the dialog opens.
  useEffect(() => {
    if (open) setFolderId(folders.some((f) => f.id === DEFAULT_FOLDER) ? DEFAULT_FOLDER : (folders[0]?.id ?? DEFAULT_FOLDER))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  const handleSave = () => {
    save.mutate(
      { folderId, doc: { id: doc.id, title: doc.title } },
      {
        onSuccess: () => {
          const folderName = folders.find((f) => f.id === folderId)?.name ?? ''
          toast.success(t('berichte.docs.saveDoc.done', { folder: folderName }))
          onClose()
        },
        onError: (err) => toast.error((err as Error).message),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-md gap-0 p-0">
        <DialogHeader className="border-b border-border px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
              <FolderDown className="h-4 w-4 text-primary" />
            </div>
            <div className="min-w-0">
              <DialogTitle className="text-sm">{t('berichte.docs.saveDoc.title')}</DialogTitle>
              <DialogDescription className="truncate">{doc.title}</DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="px-6 py-5">
          <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
            {t('berichte.docs.saveDoc.folder')}
          </label>
          <select
            value={folderId}
            onChange={(e) => setFolderId(e.target.value)}
            className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
          >
            {folders.map((f) => (
              <option key={f.id} value={f.id}>
                {f.name}
              </option>
            ))}
          </select>
          <p className="mt-2 text-xs text-muted-foreground">{t('berichte.docs.saveDoc.hint')}</p>
        </div>

        <DialogFooter className="border-t border-border px-6 py-4">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary"
          >
            {t('berichte.docs.saveDoc.cancel')}
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={save.isPending}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:opacity-60"
          >
            <FolderDown className="h-4 w-4" />
            {t('berichte.docs.saveDoc.save')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
