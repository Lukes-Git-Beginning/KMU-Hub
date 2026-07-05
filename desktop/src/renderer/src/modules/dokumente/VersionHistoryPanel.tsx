/**
 * Slide-over panel showing file version history with revert capability.
 *
 * Uses the shared DetailPanel component for consistent slide-over UX.
 * Supports manual named version creation and version revert with
 * confirmation dialog.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  History,
  Download,
  RotateCcw,
  Plus,
  Calendar,
  User,
  HardDrive,
} from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { DetailPanel, ConfirmDialog } from '@/components/shared'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  useFileVersions,
  useCreateVersion,
  useRevertVersion,
} from '@/api/hooks/useDocuments'
import { formatDateTime } from '@/lib/format'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// bytes may arrive as a string — protojson serializes int64 (file_size) as a JSON string.
function formatBytes(bytes: number | string): string {
  const n = Number(bytes)
  if (n >= 1073741824) return `${(n / 1073741824).toFixed(1)} GB`
  if (n >= 1048576) return `${(n / 1048576).toFixed(1)} MB`
  if (n >= 1024) return `${(n / 1024).toFixed(0)} KB`
  return `${n} B`
}

function formatDate(dateStr: string): string {
  return formatDateTime(dateStr, { day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface VersionHistoryPanelProps {
  fileId: string
  fileName: string
  open: boolean
  onClose: () => void
}

export function VersionHistoryPanel({
  fileId,
  fileName,
  open,
  onClose,
}: VersionHistoryPanelProps) {
  const { t } = useTranslation()
  const { data: versionsData, isLoading } = useFileVersions(fileId)
  const createVersion = useCreateVersion()
  const revertVersion = useRevertVersion()

  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [versionLabel, setVersionLabel] = useState('')
  const [revertTarget, setRevertTarget] = useState<number | null>(null)

  const versions = versionsData?.versions ?? []

  const handleCreateVersion = () => {
    if (!fileId) return
    createVersion.mutate(
      { fileId, versionLabel: versionLabel.trim() || undefined },
      {
        onSuccess: () => {
          toast.success(t('dokumente.version.created'))
          setShowCreateDialog(false)
          setVersionLabel('')
        },
        onError: (err) => {
          toast.error(`${t('common.error')}: ${err.message}`)
        },
      },
    )
  }

  const handleRevert = () => {
    if (revertTarget === null) return
    revertVersion.mutate(
      { fileId, versionNumber: revertTarget },
      {
        onSuccess: () => {
          toast.success(t('dokumente.version.restored', { version: revertTarget }))
          setRevertTarget(null)
        },
        onError: (err) => {
          toast.error(`${t('common.error')}: ${err.message}`)
        },
      },
    )
  }

  return (
    <>
      <DetailPanel
        open={open}
        onClose={onClose}
        title={t('dokumente.version.title')}
        subtitle={fileName}
        footer={
          <Button
            className="w-full"
            onClick={() => {
              setVersionLabel('')
              setShowCreateDialog(true)
            }}
          >
            <Plus className="mr-1.5 h-4 w-4" />
            {t('dokumente.version.createVersion')}
          </Button>
        }
      >
        {isLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div
                key={i}
                className="h-20 animate-pulse rounded-lg bg-secondary"
              />
            ))}
          </div>
        ) : versions.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <History className="h-10 w-10 text-muted-foreground mb-3 opacity-40" />
            <p className="text-sm text-muted-foreground">
              {t('dokumente.version.noVersions')}
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {versions.map((version, idx) => (
              <div
                key={version.id}
                className="rounded-lg border border-border p-3 space-y-2"
              >
                {/* Header */}
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div
                      className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-medium ${
                        idx === 0
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-secondary text-muted-foreground'
                      }`}
                    >
                      {version.version_number}
                    </div>
                    <div>
                      <p className="text-sm font-medium text-foreground">
                        {t('dokumente.version.versionNumber', { number: version.version_number })}
                        {idx === 0 && (
                          <span className="ml-1.5 text-xs text-primary">
                            ({t('dokumente.version.current')})
                          </span>
                        )}
                      </p>
                      {version.version_label && (
                        <p className="text-xs text-muted-foreground">
                          {version.version_label}
                        </p>
                      )}
                    </div>
                  </div>
                </div>

                {/* Meta */}
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <User className="h-3 w-3" />
                    {version.created_by_name}
                  </span>
                  <span className="flex items-center gap-1">
                    <Calendar className="h-3 w-3" />
                    {formatDate(version.created_at)}
                  </span>
                  <span className="flex items-center gap-1">
                    <HardDrive className="h-3 w-3" />
                    {formatBytes(version.file_size)}
                  </span>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-2 pt-1">
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-7 text-xs"
                    onClick={() =>
                      toast.success(t('dokumente.version.downloadStarting'))
                    }
                  >
                    <Download className="mr-1 h-3 w-3" />
                    {t('common.download')}
                  </Button>
                  {idx > 0 && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() =>
                        setRevertTarget(version.version_number)
                      }
                    >
                      <RotateCcw className="mr-1 h-3 w-3" />
                      {t('dokumente.version.restore')}
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </DetailPanel>

      {/* Create Version Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>{t('dokumente.version.createNewVersion')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-1.5 py-2">
            <Label>{t('dokumente.version.labelOptional')}</Label>
            <Input
              placeholder={t('dokumente.version.labelPlaceholder')}
              value={versionLabel}
              onChange={(e) => setVersionLabel(e.target.value)}
              autoFocus
              onKeyDown={(e) => e.key === 'Enter' && handleCreateVersion()}
            />
            <p className="text-xs text-muted-foreground">
              {t('dokumente.version.labelHint')}
            </p>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowCreateDialog(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              onClick={handleCreateVersion}
              disabled={createVersion.isPending}
            >
              {createVersion.isPending ? t('dokumente.version.creating') : t('common.create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Revert Confirmation */}
      <ConfirmDialog
        open={revertTarget !== null}
        onOpenChange={(open) => !open && setRevertTarget(null)}
        title={t('dokumente.version.restoreTitle')}
        description={t('dokumente.version.restoreDescription', { version: revertTarget })}
        confirmLabel={t('dokumente.version.restore')}
        variant="default"
        onConfirm={handleRevert}
      />
    </>
  )
}
