/**
 * Dialog for creating a new chat channel.
 *
 * Fields: channel name (required), description (optional), private toggle.
 * On success, closes dialog and selects the new channel.
 */
import { useState } from 'react'
import { useCreateChannel } from '@/api/hooks/useChannels'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface CreateChannelDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (channelId: string) => void
}

export function CreateChannelDialog({ open, onOpenChange, onCreated }: CreateChannelDialogProps) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [isPrivate, setIsPrivate] = useState(false)

  const createChannel = useCreateChannel()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return

    try {
      const result = await createChannel.mutateAsync({
        name: name.trim(),
        description: description.trim() || undefined,
        is_private: isPrivate,
      })
      const channelId = result?.channel?.id
      if (channelId) {
        onCreated(channelId)
      }
      resetForm()
    } catch {
      // Error is handled by useMutation error state
    }
  }

  const resetForm = () => {
    setName('')
    setDescription('')
    setIsPrivate(false)
  }

  return (
    <Dialog open={open} onOpenChange={(val) => {
      if (!val) resetForm()
      onOpenChange(val)
    }}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Neuen Channel erstellen</DialogTitle>
            <DialogDescription>
              Erstelle einen Channel, um mit deinem Team zu kommunizieren.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="channel-name">Name</Label>
              <Input
                id="channel-name"
                placeholder="z.B. allgemein, marketing, development"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoFocus
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="channel-description">Beschreibung (optional)</Label>
              <Textarea
                id="channel-description"
                placeholder="Worum geht es in diesem Channel?"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
              />
            </div>

            <div className="flex items-center gap-3">
              <input
                type="checkbox"
                id="channel-private"
                checked={isPrivate}
                onChange={(e) => setIsPrivate(e.target.checked)}
                className="h-4 w-4 rounded border-border"
              />
              <Label htmlFor="channel-private" className="text-sm font-normal">
                Privater Channel (nur auf Einladung)
              </Label>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Abbrechen
            </Button>
            <Button
              type="submit"
              disabled={!name.trim() || createChannel.isPending}
            >
              {createChannel.isPending ? 'Erstelle...' : 'Erstellen'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
