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
import { Lock, Plus, Eye, EyeOff, Edit, Trash2, X, Clock } from 'lucide-react'
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
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div className="flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-warning-light">
            <Lock className="h-6 w-6 text-warning" />
          </div>
          <div>
            <h1 className="text-foreground">
              <FormattedMessage id="vault.title" />
            </h1>
            <div className="flex items-center gap-2 mt-1">
              <span className="rounded-full bg-secondary px-2.5 py-0.5 text-xs font-medium text-muted-foreground">
                {secretsList.length} Secrets
              </span>
              {revealed.size > 0 && (
                <span className="rounded-full bg-warning-light px-2.5 py-0.5 text-xs font-medium text-warning">
                  {revealed.size} sichtbar
                </span>
              )}
            </div>
          </div>
        </div>
        <button
          onClick={handleOpenAdd}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Plus className="h-4 w-4" />
          <FormattedMessage id="vault.addSecret" />
        </button>
      </div>

      {/* Secrets table */}
      <div className="rounded-lg border border-border bg-card overflow-hidden glass-surface">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="vault.keyName" />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="vault.description_field" />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="vault.value" />
                </th>
                <th className="px-4 py-3 text-center text-xs font-medium text-muted-foreground">
                  Version
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="audit.timestamp" />
                </th>
                <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">
                  <FormattedMessage id="common.actions" />
                </th>
              </tr>
            </thead>
            <tbody>
              {isLoading && (
                <tr>
                  <td colSpan={6} className="text-center py-12 text-muted-foreground">
                    <FormattedMessage id="common.loading" />
                  </td>
                </tr>
              )}
              {!isLoading && secretsList.length === 0 && (
                <tr>
                  <td colSpan={6} className="text-center py-12">
                    <Lock className="mx-auto h-10 w-10 text-muted-foreground/40 mb-2" />
                    <p className="text-sm text-muted-foreground">
                      <FormattedMessage id="vault.empty" />
                    </p>
                  </td>
                </tr>
              )}
              {secretsList.map((secret) => {
                const revealedEntry = revealed.get(secret.key_name)
                const remainingSec = revealedEntry
                  // eslint-disable-next-line react-hooks/purity -- Date.now() needed for countdown display
                  ? Math.max(0, Math.ceil((revealedEntry.expiresAt - Date.now()) / 1000))
                  : 0

                return (
                  <tr
                    key={secret.id}
                    className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors"
                  >
                    <td className="px-4 py-3 font-mono text-sm font-medium text-foreground">
                      {secret.key_name}
                    </td>
                    <td className="px-4 py-3 text-sm text-muted-foreground">
                      {secret.description || '-'}
                    </td>
                    <td className="px-4 py-3 font-mono text-sm">
                      {revealedEntry ? (
                        <span className="flex items-center gap-2">
                          <code className="rounded-lg bg-secondary/50 px-2 py-1 text-xs text-foreground break-all">
                            {revealedEntry.value}
                          </code>
                          <span className="flex items-center gap-1 rounded-full bg-warning-light px-2 py-0.5 text-[10px] font-medium text-warning whitespace-nowrap">
                            <Clock className="h-3 w-3" />
                            {remainingSec}s
                          </span>
                        </span>
                      ) : (
                        <span className="text-muted-foreground/60">
                          <FormattedMessage id="vault.valueHidden" />
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-center">
                      <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
                        v{secret.key_version}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                      <FormattedDate
                        value={secret.updated_at}
                        year="numeric"
                        month="2-digit"
                        day="2-digit"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-1">
                        <button
                          onClick={() => handleReveal(secret.key_name)}
                          disabled={revealMutation.isPending}
                          className="rounded-lg p-1.5 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors disabled:opacity-40"
                          title={intl.formatMessage({ id: revealedEntry ? 'vault.hideValue' : 'vault.showValue' })}
                        >
                          {revealedEntry ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                        </button>
                        <button
                          onClick={() => handleOpenEdit(secret.key_name, secret.description)}
                          className="rounded-lg p-1.5 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
                          title={intl.formatMessage({ id: 'common.edit' })}
                        >
                          <Edit className="h-4 w-4" />
                        </button>
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <button
                              className="rounded-lg p-1.5 text-muted-foreground hover:text-error hover:bg-error-light transition-colors"
                              title={intl.formatMessage({ id: 'common.delete' })}
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </AlertDialogTrigger>
                          <AlertDialogContent className="glass-elevated">
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
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Add/Edit Dialog */}
      {dialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setDialogOpen(false)} />
          <div className="relative z-10 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated">
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-lg font-semibold text-foreground">
                <FormattedMessage id={isEditing ? 'vault.editSecret' : 'vault.addSecret'} />
              </h2>
              <button
                onClick={() => setDialogOpen(false)}
                className="rounded-lg p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  <FormattedMessage id="vault.keyName" />
                </label>
                <input
                  value={formKeyName}
                  onChange={(e) => setFormKeyName(e.target.value)}
                  disabled={isEditing}
                  placeholder="API_KEY"
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground font-mono placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring disabled:opacity-50"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  <FormattedMessage id="vault.value" />
                </label>
                <textarea
                  value={formValue}
                  onChange={(e) => setFormValue(e.target.value)}
                  placeholder={intl.formatMessage({ id: 'vault.value' })}
                  rows={3}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground font-mono placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  <FormattedMessage id="vault.description_field" />{' '}
                  <span className="text-xs text-muted-foreground font-normal">
                    (<FormattedMessage id="common.optional" />)
                  </span>
                </label>
                <input
                  value={formDescription}
                  onChange={(e) => setFormDescription(e.target.value)}
                  placeholder={intl.formatMessage({ id: 'common.optional' })}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
            </div>

            <div className="flex justify-end gap-2 mt-6">
              <button
                onClick={() => setDialogOpen(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <FormattedMessage id="common.cancel" />
              </button>
              <button
                onClick={handleSave}
                disabled={setMutation.isPending || (!isEditing && !formKeyName.trim()) || !formValue}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
              >
                {setMutation.isPending ? (
                  <span className="flex items-center gap-2">
                    <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
                    <FormattedMessage id="common.loading" />
                  </span>
                ) : (
                  <FormattedMessage id="common.save" />
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
