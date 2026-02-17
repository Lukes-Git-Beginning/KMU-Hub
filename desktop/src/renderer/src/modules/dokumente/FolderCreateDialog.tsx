/**
 * Dialog for creating a new folder.
 *
 * Connected to useCreateFolder mutation. Keeps existing UI but
 * uses API mutation instead of Zustand store action.
 */
import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { FolderPlus } from 'lucide-react'
import { toast } from 'sonner'
import { useCreateFolder } from '@/api/hooks/useDocuments'
import type { FolderSpaceType } from '@/api/types/document-types'

interface FolderCreateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  parentId?: string | null
  parentName?: string
  spaceType: FolderSpaceType
  spaceId: string
}

export function FolderCreateDialog({
  open,
  onOpenChange,
  parentId,
  parentName,
  spaceType,
  spaceId,
}: FolderCreateDialogProps) {
  const [name, setName] = useState('')
  const createFolder = useCreateFolder()

  useEffect(() => {
    if (open) setName('')
  }, [open])

  const handleSubmit = () => {
    if (!name.trim()) return
    createFolder.mutate(
      {
        name: name.trim(),
        parent_id: parentId ?? undefined,
        space_type: spaceType,
        space_id: spaceId,
      },
      {
        onSuccess: () => {
          toast.success(`Ordner "${name.trim()}" erstellt`)
          onOpenChange(false)
        },
        onError: (err) => {
          toast.error(`Fehler: ${err.message}`)
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Neuer Ordner</DialogTitle>
        </DialogHeader>
        {parentName && (
          <p className="text-xs text-muted-foreground">In: {parentName}</p>
        )}
        <div className="space-y-1.5 py-2">
          <Label>Ordnername</Label>
          <Input
            placeholder="z.B. Kundenprojekte"
            value={name}
            onChange={(e) => setName(e.target.value)}
            autoFocus
            onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!name.trim() || createFolder.isPending}
          >
            <FolderPlus className="mr-1.5 h-4 w-4" />
            {createFolder.isPending ? 'Erstelle...' : 'Erstellen'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
