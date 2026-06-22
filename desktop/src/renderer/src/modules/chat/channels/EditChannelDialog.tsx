/**
 * Dialog for editing an existing channel (KO-9).
 *
 * Owners/admins can rename, change the description and toggle privacy. Changes
 * persist in the chat-store via useUpdateChannel and reflect across the list.
 */
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useUpdateChannel, type ChannelInfo } from '@/api/hooks/useChannels'
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

interface EditChannelDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  channel: ChannelInfo | undefined
}

export function EditChannelDialog({ open, onOpenChange, channel }: EditChannelDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [isPrivate, setIsPrivate] = useState(false)
  const updateChannel = useUpdateChannel()

  // Sync form with the channel whenever the dialog opens for a (different) channel.
  useEffect(() => {
    if (open && channel) {
      setName(channel.name ?? '')
      setDescription(channel.description ?? '')
      setIsPrivate(Boolean(channel.is_private))
    }
  }, [open, channel])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!channel?.id || !name.trim()) return
    try {
      await updateChannel.mutateAsync({
        id: channel.id,
        name: name.trim(),
        description: description.trim(),
        is_private: isPrivate,
      })
      onOpenChange(false)
    } catch {
      // surfaced via mutation state
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{t('chat.editChannel.title')}</DialogTitle>
            <DialogDescription>{t('chat.editChannel.description')}</DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="edit-channel-name">{t('chat.createChannel.nameLabel')}</Label>
              <Input
                id="edit-channel-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoFocus
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-channel-description">{t('chat.createChannel.descriptionLabel')}</Label>
              <Textarea
                id="edit-channel-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
              />
            </div>

            <div className="flex items-center gap-3">
              <input
                type="checkbox"
                id="edit-channel-private"
                checked={isPrivate}
                onChange={(e) => setIsPrivate(e.target.checked)}
                className="h-4 w-4 rounded border-border"
              />
              <Label htmlFor="edit-channel-private" className="text-sm font-normal">
                {t('chat.createChannel.privateLabel')}
              </Label>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={!name.trim() || updateChannel.isPending}>
              {updateChannel.isPending ? t('chat.editChannel.saving') : t('common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
