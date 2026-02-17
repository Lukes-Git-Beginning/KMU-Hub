/**
 * Dialog for renaming a file or folder.
 *
 * Connected to useUpdateFile / useUpdateFolder mutations.
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
import { toast } from 'sonner'
import { useUpdateFile, useUpdateFolder } from '@/api/hooks/useDocuments'

interface RenameDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  targetId: string
  currentName: string
  itemType: 'file' | 'folder'
}

export function RenameDialog({
  open,
  onOpenChange,
  targetId,
  currentName,
  itemType,
}: RenameDialogProps) {
  const [name, setName] = useState('')
  const updateFile = useUpdateFile()
  const updateFolder = useUpdateFolder()

  useEffect(() => {
    if (open) setName(currentName)
  }, [open, currentName])

  const isPending = updateFile.isPending || updateFolder.isPending

  const handleSubmit = () => {
    if (!name.trim() || name.trim() === currentName) return

    if (itemType === 'file') {
      updateFile.mutate(
        { id: targetId, filename: name.trim() },
        {
          onSuccess: () => {
            toast.success('Datei umbenannt')
            onOpenChange(false)
          },
          onError: (err) => {
            toast.error(`Fehler: ${err.message}`)
          },
        },
      )
    } else {
      updateFolder.mutate(
        { id: targetId, name: name.trim() },
        {
          onSuccess: () => {
            toast.success('Ordner umbenannt')
            onOpenChange(false)
          },
          onError: (err) => {
            toast.error(`Fehler: ${err.message}`)
          },
        },
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>
            {itemType === 'file' ? 'Datei' : 'Ordner'} umbenennen
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-1.5 py-2">
          <Label>Neuer Name</Label>
          <Input
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
            disabled={
              !name.trim() || name.trim() === currentName || isPending
            }
          >
            {isPending ? 'Speichere...' : 'Umbenennen'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
