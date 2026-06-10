/**
 * Dialog for moving or copying a file into another folder.
 *
 * Renders the folder tree (all spaces) as an indented picker; confirm calls
 * the existing useMoveFile/useCopyFile mutations (backend endpoints
 * POST /files/{id}/move|copy exist since the documents backend phase).
 */
import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Folder, FolderInput, Copy } from 'lucide-react'
import { toast } from 'sonner'
import { useDocumentFolders, useMoveFile, useCopyFile } from '@/api/hooks/useDocuments'
import type { DocumentFile, DocumentFolder } from '@/api/types/document-types'

export type MoveCopyMode = 'move' | 'copy'

interface MoveCopyDialogProps {
  mode: MoveCopyMode
  file: DocumentFile
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface FolderNode {
  folder: DocumentFolder
  children: FolderNode[]
}

function buildTree(folders: DocumentFolder[]): FolderNode[] {
  const byParent = new Map<string | null, DocumentFolder[]>()
  const ids = new Set(folders.map((f) => f.id))
  for (const f of folders) {
    // Treat folders whose parent is not in the result set as roots
    const key = f.parent_id && ids.has(f.parent_id) ? f.parent_id : null
    const list = byParent.get(key) ?? []
    list.push(f)
    byParent.set(key, list)
  }
  const toNode = (folder: DocumentFolder): FolderNode => ({
    folder,
    children: (byParent.get(folder.id) ?? []).map(toNode),
  })
  return (byParent.get(null) ?? []).map(toNode)
}

export function MoveCopyDialog({ mode, file, open, onOpenChange }: MoveCopyDialogProps) {
  const { t } = useTranslation()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const { data: personal } = useDocumentFolders({ space_type: 'personal' })
  const { data: team } = useDocumentFolders({ space_type: 'team' })
  const { data: project } = useDocumentFolders({ space_type: 'project' })
  const moveFile = useMoveFile()
  const copyFile = useCopyFile()

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- reset selection when reopened
    if (open) setSelectedId(null)
  }, [open])

  const tree = useMemo(() => {
    const all = [
      ...(personal?.folders ?? []),
      ...(team?.folders ?? []),
      ...(project?.folders ?? []),
    ]
    // De-dupe (spaces can overlap in demo data)
    const seen = new Set<string>()
    const unique = all.filter((f) => (seen.has(f.id) ? false : (seen.add(f.id), true)))
    return buildTree(unique)
  }, [personal?.folders, team?.folders, project?.folders])

  const isPending = moveFile.isPending || copyFile.isPending

  const handleConfirm = () => {
    if (!selectedId) return
    const mutation = mode === 'move' ? moveFile : copyFile
    mutation.mutate(
      { id: file.id, targetFolderId: selectedId },
      {
        onSuccess: () => {
          toast.success(
            t(mode === 'move' ? 'dokumente.moveCopy.movedToast' : 'dokumente.moveCopy.copiedToast', {
              name: file.filename,
            }),
          )
          onOpenChange(false)
        },
        onError: (err) => {
          toast.error(`${t('common.error')}: ${err.message}`)
        },
      },
    )
  }

  const renderNode = (node: FolderNode, depth: number) => {
    const isCurrent = node.folder.id === file.folder_id
    const disabled = mode === 'move' && isCurrent
    const isSelected = selectedId === node.folder.id
    return (
      <div key={node.folder.id}>
        <button
          disabled={disabled}
          onClick={() => setSelectedId(node.folder.id)}
          className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
            isSelected
              ? 'bg-primary-light text-primary font-medium'
              : disabled
                ? 'text-muted-foreground/50 cursor-not-allowed'
                : 'text-foreground hover:bg-secondary'
          }`}
          style={{ paddingLeft: `${8 + depth * 16}px` }}
        >
          <Folder className="h-4 w-4 shrink-0" />
          <span className="truncate">{node.folder.name}</span>
          {isCurrent && (
            <span className="ml-auto shrink-0 text-[10px] text-muted-foreground">
              {t('dokumente.moveCopy.currentFolder')}
            </span>
          )}
        </button>
        {node.children.map((child) => renderNode(child, depth + 1))}
      </div>
    )
  }

  const ActionIcon = mode === 'move' ? FolderInput : Copy

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {t(mode === 'move' ? 'dokumente.moveCopy.titleMove' : 'dokumente.moveCopy.titleCopy')}
          </DialogTitle>
        </DialogHeader>
        <p className="text-xs text-muted-foreground truncate" title={file.filename}>
          {t('dokumente.moveCopy.subtitle', { name: file.filename })}
        </p>
        <div className="max-h-72 overflow-y-auto rounded-lg border border-border p-1.5">
          {tree.length === 0 ? (
            <p className="px-2 py-4 text-center text-sm text-muted-foreground">
              {t('dokumente.moveCopy.noFolders')}
            </p>
          ) : (
            tree.map((node) => renderNode(node, 0))
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleConfirm} disabled={!selectedId || isPending}>
            <ActionIcon className="mr-1.5 h-4 w-4" />
            {t(mode === 'move' ? 'dokumente.moveCopy.actionMove' : 'dokumente.moveCopy.actionCopy')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
