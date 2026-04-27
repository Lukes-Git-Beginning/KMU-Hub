/**
 * Share dialog connected to real API.
 *
 * Uses useShares and useShareEntity/useUnshareEntity mutations.
 * Shows current shares with permission levels and allows
 * adding/removing share recipients.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Link, X, Users, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  useShares,
  useShareEntity,
  useUnshareEntity,
} from '@/api/hooks/useDocuments'
import type { SharePermission } from '@/api/types/document-types'
import { formatDate } from '@/lib/format'

interface ShareDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  entityType: 'file' | 'folder'
  entityId: string
  entityName: string
}

export function ShareDialog({
  open,
  onOpenChange,
  entityType,
  entityId,
  entityName,
}: ShareDialogProps) {
  const { t } = useTranslation()
  const { data: sharesData, isLoading } = useShares(entityType, entityId)
  const shareEntity = useShareEntity()
  const unshareEntity = useUnshareEntity()

  const [searchUserId, setSearchUserId] = useState('')
  const [permission, setPermission] = useState<SharePermission>('read')

  const shares = sharesData?.shares ?? []

  const handleAddShare = () => {
    if (!searchUserId.trim()) return

    shareEntity.mutate(
      {
        entity_type: entityType,
        entity_id: entityId,
        shared_with_user_id: searchUserId.trim(),
        permission,
      },
      {
        onSuccess: () => {
          toast.success(t('dokumente.share.added'))
          setSearchUserId('')
        },
        onError: (err) => {
          toast.error(`${t('common.error')}: ${err.message}`)
        },
      },
    )
  }

  const handleRemoveShare = (shareId: string) => {
    unshareEntity.mutate(shareId, {
      onSuccess: () => {
        toast.success(t('dokumente.share.removed'))
      },
      onError: (err) => {
        toast.error(`${t('common.error')}: ${err.message}`)
      },
    })
  }

  const copyLink = () => {
    toast.success(t('dokumente.share.linkCopied'))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            {entityType === 'file' ? t('dokumente.share.fileTitle') : t('dokumente.share.folderTitle')}
          </DialogTitle>
        </DialogHeader>

        <p className="text-sm text-muted-foreground truncate">{entityName}</p>

        <div className="space-y-4 py-2">
          {/* Add person */}
          <div className="space-y-1.5">
            <Label>{t('dokumente.share.addPerson')}</Label>
            <div className="flex gap-2">
              <Input
                placeholder={t('dokumente.share.userIdPlaceholder')}
                value={searchUserId}
                onChange={(e) => setSearchUserId(e.target.value)}
                className="flex-1"
                onKeyDown={(e) => e.key === 'Enter' && handleAddShare()}
              />
              <Select
                value={permission}
                onValueChange={(v) =>
                  setPermission(v as SharePermission)
                }
              >
                <SelectTrigger className="w-32">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="read">{t('dokumente.share.permissionRead')}</SelectItem>
                  <SelectItem value="write">{t('dokumente.share.permissionWrite')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {searchUserId.trim() && (
              <Button
                size="sm"
                className="w-full"
                onClick={handleAddShare}
                disabled={shareEntity.isPending}
              >
                {shareEntity.isPending ? (
                  <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
                ) : null}
                {t('dokumente.share.share')}
              </Button>
            )}
          </div>

          {/* Current shares */}
          {isLoading ? (
            <div className="flex justify-center py-4">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : shares.length > 0 ? (
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">
                {t('dokumente.share.sharedWith')}
              </Label>
              {shares.map((s) => (
                <div
                  key={s.id}
                  className="flex items-center justify-between rounded-md border border-border px-3 py-2"
                >
                  <div className="flex items-center gap-2">
                    <span className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                      {s.shared_with_user_name
                        .split(' ')
                        .map((n) => n[0])
                        .join('')
                        .slice(0, 2)}
                    </span>
                    <div>
                      <span className="text-sm text-foreground">
                        {s.shared_with_user_name}
                      </span>
                      <p className="text-[10px] text-muted-foreground">
                        {t('dokumente.share.by')} {s.shared_by_name} &middot;{' '}
                        {formatDate(s.created_at)}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <span className="text-xs text-muted-foreground">
                      {s.permission === 'write'
                        ? t('dokumente.share.permissionWrite')
                        : t('dokumente.share.permissionRead')}
                    </span>
                    <button
                      onClick={() => handleRemoveShare(s.id)}
                      className="rounded p-0.5 text-muted-foreground hover:text-destructive"
                      disabled={unshareEntity.isPending}
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          ) : null}

          {/* Copy link */}
          <Button variant="outline" className="w-full" onClick={copyLink}>
            <Link className="mr-1.5 h-4 w-4" />
            {t('dokumente.share.copyLink')}
          </Button>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
