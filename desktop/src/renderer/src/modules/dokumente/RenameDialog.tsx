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

interface RenameDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentName: string
  itemType: 'file' | 'folder'
  onSubmit: (name: string) => void
}

export function RenameDialog({ open, onOpenChange, currentName, itemType, onSubmit }: RenameDialogProps) {
  const [name, setName] = useState('')

  useEffect(() => {
    if (open) setName(currentName)
  }, [open, currentName])

  const handleSubmit = () => {
    if (!name.trim() || name.trim() === currentName) return
    onSubmit(name.trim())
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{itemType === 'file' ? 'Datei' : 'Ordner'} umbenennen</DialogTitle>
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
          <Button variant="outline" onClick={() => onOpenChange(false)}>Abbrechen</Button>
          <Button onClick={handleSubmit} disabled={!name.trim() || name.trim() === currentName}>
            Umbenennen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
