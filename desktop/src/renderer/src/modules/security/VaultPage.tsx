/**
 * Vault secrets management page (admin only).
 *
 * Displays a list of encrypted vault secrets with metadata.
 * Supports reveal (temporary 30s display), add, edit, and delete operations.
 * Secret values are never cached in TanStack Query (mutation-based retrieval).
 */
import { useState, useCallback, useEffect, useRef } from 'react'
import { FormattedMessage, FormattedDate, useIntl } from 'react-intl'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  useVaultSecrets,
  useGetVaultSecret,
  useSetVaultSecret,
  useDeleteVaultSecret,
} from '@/api/hooks/useSecurity'

/** Duration (ms) to display a revealed secret value before auto-hiding. */
const REVEAL_DURATION_MS = 30_000

interface RevealedSecret {
  keyName: string
  value: string
  expiresAt: number
}

export default function VaultPage() {
  const intl = useIntl()
  const { data: secrets, isLoading } = useVaultSecrets()
  const revealMutation = useGetVaultSecret()
  const setMutation = useSetVaultSecret()
  const deleteMutation = useDeleteVaultSecret()

  // Revealed secrets (auto-expire after 30s)
  const [revealed, setRevealed] = useState<Map<string, RevealedSecret>>(new Map())
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Add/Edit dialog state
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editKeyName, setEditKeyName] = useState('')
  const [formKeyName, setFormKeyName] = useState('')
  const [formValue, setFormValue] = useState('')
  const [formDescription, setFormDescription] = useState('')
  const isEditing = editKeyName !== ''

  // Auto-hide expired revealed secrets
  useEffect(() => {
    if (revealed.size === 0) {
      if (timerRef.current) {
        clearInterval(timerRef.current)
        timerRef.current = null
      }
      return
    }

    timerRef.current = setInterval(() => {
      const now = Date.now()
      setRevealed((prev) => {
        const next = new Map(prev)
        let changed = false
        for (const [key, entry] of next) {
          if (entry.expiresAt <= now) {
            next.delete(key)
            changed = true
          }
        }
        return changed ? next : prev
      })
    }, 1000)

    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current)
        timerRef.current = null
      }
    }
  }, [revealed.size])

  const handleReveal = useCallback(
    (keyName: string) => {
      // If already revealed, hide it
      if (revealed.has(keyName)) {
        setRevealed((prev) => {
          const next = new Map(prev)
          next.delete(keyName)
          return next
        })
        return
      }

      revealMutation.mutate(keyName, {
        onSuccess: (result) => {
          setRevealed((prev) => {
            const next = new Map(prev)
            next.set(keyName, {
              keyName,
              value: result.value,
              expiresAt: Date.now() + REVEAL_DURATION_MS,
            })
            return next
          })
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : intl.formatMessage({ id: 'common.error' }))
        },
      })
    },
    [revealed, revealMutation, intl],
  )

  const handleOpenAdd = useCallback(() => {
    setEditKeyName('')
    setFormKeyName('')
    setFormValue('')
    setFormDescription('')
    setDialogOpen(true)
  }, [])

  const handleOpenEdit = useCallback(
    (keyName: string, description: string) => {
      setEditKeyName(keyName)
      setFormKeyName(keyName)
      setFormValue('')
      setFormDescription(description)
      setDialogOpen(true)
    },
    [],
  )

  const handleSave = useCallback(() => {
    const keyName = isEditing ? editKeyName : formKeyName.trim()
    if (!keyName || !formValue) return

    setMutation.mutate(
      { keyName, value: formValue, description: formDescription || undefined },
      {
        onSuccess: () => {
          toast.success(intl.formatMessage({ id: 'common.success' }))
          setDialogOpen(false)
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : intl.formatMessage({ id: 'common.error' }))
        },
      },
    )
  }, [isEditing, editKeyName, formKeyName, formValue, formDescription, setMutation, intl])

  const handleDelete = useCallback(
    (keyName: string) => {
      deleteMutation.mutate(keyName, {
        onSuccess: () => {
          toast.success(intl.formatMessage({ id: 'common.success' }))
          // Clear revealed if deleted
          setRevealed((prev) => {
            const next = new Map(prev)
            next.delete(keyName)
            return next
          })
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : intl.formatMessage({ id: 'common.error' }))
        },
      })
    },
    [deleteMutation, intl],
  )

  const secretsList = secrets ?? []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">
            <FormattedMessage id="vault.title" />
          </h2>
          <p className="text-sm text-muted-foreground">
            <FormattedMessage id="vault.description" />
          </p>
        </div>
        <Button size="sm" onClick={handleOpenAdd}>
          <FormattedMessage id="vault.addSecret" />
        </Button>
      </div>

      {/* Secrets table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead><FormattedMessage id="vault.keyName" /></TableHead>
              <TableHead><FormattedMessage id="vault.description_field" /></TableHead>
              <TableHead><FormattedMessage id="vault.value" /></TableHead>
              <TableHead>Version</TableHead>
              <TableHead><FormattedMessage id="audit.timestamp" /></TableHead>
              <TableHead><FormattedMessage id="common.actions" /></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading && (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                  <FormattedMessage id="common.loading" />
                </TableCell>
              </TableRow>
            )}
            {!isLoading && secretsList.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                  <FormattedMessage id="vault.empty" />
                </TableCell>
              </TableRow>
            )}
            {secretsList.map((secret) => {
              const revealedEntry = revealed.get(secret.key_name)
              const remainingSec = revealedEntry
                ? Math.max(0, Math.ceil((revealedEntry.expiresAt - Date.now()) / 1000))
                : 0

              return (
                <TableRow key={secret.id}>
                  <TableCell className="font-mono text-sm font-medium">
                    {secret.key_name}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {secret.description || '-'}
                  </TableCell>
                  <TableCell className="font-mono text-sm">
                    {revealedEntry ? (
                      <span className="flex items-center gap-2">
                        <code className="rounded bg-muted px-1.5 py-0.5 text-xs break-all">
                          {revealedEntry.value}
                        </code>
                        <span className="text-xs text-muted-foreground whitespace-nowrap">
                          ({remainingSec}s)
                        </span>
                      </span>
                    ) : (
                      <span className="text-muted-foreground">
                        <FormattedMessage id="vault.valueHidden" />
                      </span>
                    )}
                  </TableCell>
                  <TableCell className="text-sm text-center">
                    v{secret.key_version}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                    <FormattedDate
                      value={secret.updated_at}
                      year="numeric"
                      month="2-digit"
                      day="2-digit"
                    />
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleReveal(secret.key_name)}
                        disabled={revealMutation.isPending}
                      >
                        <FormattedMessage
                          id={revealedEntry ? 'vault.hideValue' : 'vault.showValue'}
                        />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleOpenEdit(secret.key_name, secret.description)}
                      >
                        <FormattedMessage id="common.edit" />
                      </Button>
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <Button variant="ghost" size="sm" className="text-destructive">
                            <FormattedMessage id="common.delete" />
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>
                              <FormattedMessage id="vault.deleteSecret" />
                            </AlertDialogTitle>
                            <AlertDialogDescription>
                              <FormattedMessage
                                id="vault.deleteConfirm"
                                values={{ key: secret.key_name }}
                              />
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>
                              <FormattedMessage id="common.cancel" />
                            </AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => handleDelete(secret.key_name)}
                              disabled={deleteMutation.isPending}
                            >
                              <FormattedMessage id="common.confirm" />
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>

      {/* Add/Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>
              <FormattedMessage id={isEditing ? 'vault.editSecret' : 'vault.addSecret'} />
            </DialogTitle>
            <DialogDescription>
              <FormattedMessage id="vault.description" />
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="vault-key">
                <FormattedMessage id="vault.keyName" />
              </Label>
              <Input
                id="vault-key"
                value={formKeyName}
                onChange={(e) => setFormKeyName(e.target.value)}
                disabled={isEditing}
                placeholder="API_KEY"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="vault-value">
                <FormattedMessage id="vault.value" />
              </Label>
              <Textarea
                id="vault-value"
                value={formValue}
                onChange={(e) => setFormValue(e.target.value)}
                placeholder={intl.formatMessage({ id: 'vault.value' })}
                rows={3}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="vault-desc">
                <FormattedMessage id="vault.description_field" />
              </Label>
              <Input
                id="vault-desc"
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                placeholder={intl.formatMessage({ id: 'common.optional' })}
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              onClick={handleSave}
              disabled={
                setMutation.isPending ||
                (!isEditing && !formKeyName.trim()) ||
                !formValue
              }
            >
              {setMutation.isPending ? (
                <span className="flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
                  <FormattedMessage id="common.loading" />
                </span>
              ) : (
                <FormattedMessage id="common.save" />
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
