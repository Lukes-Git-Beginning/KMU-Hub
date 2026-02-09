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

interface FolderCreateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  parentName?: string
  onSubmit: (name: string) => void
}

export function FolderCreateDialog({ open, onOpenChange, parentName, onSubmit }: FolderCreateDialogProps) {
  const [name, setName] = useState('')

  useEffect(() => {
    if (open) setName('')
  }, [open])

  const handleSubmit = () => {
    if (!name.trim()) return
    onSubmit(name.trim())
    onOpenChange(false)
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
          <Button variant="outline" onClick={() => onOpenChange(false)}>Abbrechen</Button>
          <Button onClick={handleSubmit} disabled={!name.trim()}>
            <FolderPlus className="mr-1.5 h-4 w-4" />
            Erstellen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
