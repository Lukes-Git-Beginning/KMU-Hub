/**
 * Admin page for IP access control (allowlist/blocklist).
 *
 * Displays existing IP rules in a table with CIDR, type, description, and
 * creation date. Provides a dialog to add new rules and delete existing ones.
 * Wired to useIPRules, useCreateIPRule, useDeleteIPRule hooks.
 */
import { useState, useCallback } from 'react'
import { Navigate } from 'react-router-dom'
import { Globe, Plus, Trash2, Info } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { useTranslation } from 'react-i18next'
import { useFormatDate } from '@/hooks/useFormatters'
import { toast } from 'sonner'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useAuthStore } from '@/stores/auth'
import { useIPRules, useCreateIPRule, useDeleteIPRule } from '@/api/hooks/useSecurity'

export default function IPAccessPage() {
  const { t } = useTranslation()
  const formatDate = useFormatDate()
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin')

  const { data: rules, isLoading } = useIPRules()
  const createRule = useCreateIPRule()
  const deleteRule = useDeleteIPRule()

  // Add rule dialog state
  const [showAdd, setShowAdd] = useState(false)
  const [newCidr, setNewCidr] = useState('')
  const [newType, setNewType] = useState<'allow' | 'block'>('allow')
  const [newDescription, setNewDescription] = useState('')

  // Delete confirmation
  const [deleteId, setDeleteId] = useState<string | null>(null)

  const allowCount = rules?.filter((r) => r.rule_type === 'allow').length ?? 0
  const blockCount = rules?.filter((r) => r.rule_type === 'block').length ?? 0

  const handleAdd = useCallback(() => {
    if (!newCidr.trim()) return
    createRule.mutate(
      {
        ipCidr: newCidr.trim(),
        ruleType: newType,
        description: newDescription.trim(),
      },
      {
        onSuccess: () => {
          toast.success(t('common.success'))
          setShowAdd(false)
          setNewCidr('')
          setNewType('allow')
          setNewDescription('')
        },
        onError: () => toast.error(t('common.error')),
      },
    )
  }, [newCidr, newType, newDescription, createRule, t])

  const handleDelete = useCallback(() => {
    if (!deleteId) return
    deleteRule.mutate(deleteId, {
      onSuccess: () => {
        toast.success(t('common.success'))
        setDeleteId(null)
      },
      onError: () => toast.error(t('common.error')),
    })
  }, [deleteId, deleteRule, t])

  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div className="flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-info-light">
            <Globe className="h-6 w-6 text-info" />
          </div>
          <div>
            <h1 className="text-foreground">
              {t('ipAccess.title')}
            </h1>
            <div className="flex items-center gap-2 mt-1">
              <span className="rounded-full bg-success-light px-2.5 py-0.5 text-xs font-medium text-success">
                {allowCount} Allow
              </span>
              <span className="rounded-full bg-error-light px-2.5 py-0.5 text-xs font-medium text-error">
                {blockCount} Block
              </span>
            </div>
          </div>
        </div>
        <button
          onClick={() => setShowAdd(true)}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Plus className="h-4 w-4" />
          {t('ipAccess.addRule')}
        </button>
      </div>

      {/* Info Box */}
      <div className="flex gap-3 rounded-lg bg-info-light p-4 mb-6">
        <Info className="h-5 w-5 shrink-0 text-info mt-0.5" />
        <p className="text-sm text-info">
          {t('ipAccess.allowlistInfo')}
        </p>
      </div>

      {/* Rules Table */}
      <div className="rounded-lg border border-border bg-card overflow-hidden glass-surface">
        <div className="overflow-x-auto">
          {isLoading ? (
            <p className="text-sm text-muted-foreground py-12 text-center">
              {t('common.loading')}
            </p>
          ) : !rules || rules.length === 0 ? (
            <div className="py-12 text-center">
              <Globe className="mx-auto h-10 w-10 text-muted-foreground/40 mb-2" />
              <p className="text-sm text-muted-foreground">
                {t('ipAccess.noRules')}
              </p>
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border">
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                    {t('ipAccess.cidr')}
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                    {t('ipAccess.type')}
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                    {t('ipAccess.description_field')}
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                    {t('ipAccess.createdAt')}
                  </th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">
                    {t('common.actions')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {rules.map((rule) => (
                  <tr
                    key={rule.id}
                    className="group border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors"
                  >
                    <td className="px-4 py-3 font-mono text-sm font-medium text-foreground">
                      {rule.ip_cidr}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                          rule.rule_type === 'allow'
                            ? 'bg-success-light text-success'
                            : 'bg-error-light text-error'
                        }`}
                      >
                        {t(rule.rule_type === 'allow' ? 'ipAccess.type.allow' : 'ipAccess.type.block')}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-muted-foreground">
                      {rule.description ?? '-'}
                    </td>
                    <td className="px-4 py-3 text-sm text-muted-foreground">
                      {rule.created_at ? (
                        formatDate(rule.created_at, {
                          year: 'numeric',
                          month: 'short',
                          day: 'numeric',
                        })
                      ) : (
                        '-'
                      )}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        onClick={() => setDeleteId(rule.id)}
                        className="rounded-lg p-1.5 text-muted-foreground opacity-0 group-hover:opacity-100 hover:text-error hover:bg-error-light transition-all"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Add Rule Dialog */}
      <Dialog open={showAdd} onOpenChange={(o) => { if (!o) setShowAdd(false) }}>
        <DialogContent className="gap-0 p-0 max-w-md">
          <div className="p-6">
            <DialogHeader className="mb-5">
              <DialogTitle className="text-lg font-semibold text-foreground">
                {t('ipAccess.addRule')}
              </DialogTitle>
              <DialogDescription className="sr-only">Neue IP-Regel hinzufügen</DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  {t('ipAccess.cidr')}
                </label>
                <input
                  value={newCidr}
                  onChange={(e) => setNewCidr(e.target.value)}
                  placeholder="192.168.1.0/24"
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground font-mono placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  {t('ipAccess.type')}
                </label>
                <Select value={newType} onValueChange={(v) => setNewType(v as 'allow' | 'block')}>
                  <SelectTrigger className="rounded-lg border-border bg-card text-sm">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="allow">
                      {t('ipAccess.type.allow')}
                    </SelectItem>
                    <SelectItem value="block">
                      {t('ipAccess.type.block')}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  {t('ipAccess.description_field')}{' '}
                  <span className="text-xs text-muted-foreground font-normal">
                    ({t('common.optional')})
                  </span>
                </label>
                <input
                  value={newDescription}
                  onChange={(e) => setNewDescription(e.target.value)}
                  placeholder="Office network"
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
            </div>

            <DialogFooter className="mt-6">
              <button
                onClick={() => setShowAdd(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('common.cancel')}
              </button>
              <button
                onClick={handleAdd}
                disabled={!newCidr.trim() || createRule.isPending}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
              >
                {t('common.create')}
              </button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteId !== null} onOpenChange={(o) => { if (!o) setDeleteId(null) }}>
        <DialogContent className="gap-0 p-0 max-w-sm">
          <div className="p-6">
            <DialogHeader className="mb-2">
              <DialogTitle className="text-lg font-semibold text-foreground">
                {t('ipAccess.deleteConfirm')}
              </DialogTitle>
              <DialogDescription className="sr-only">IP-Regel löschen bestätigen</DialogDescription>
            </DialogHeader>
            <DialogFooter className="mt-6">
              <button
                onClick={() => setDeleteId(null)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('common.cancel')}
              </button>
              <button
                onClick={handleDelete}
                disabled={deleteRule.isPending}
                className="rounded-lg bg-error px-4 py-2 text-sm text-white hover:bg-error/90 transition-colors disabled:opacity-50"
              >
                {t('common.delete')}
              </button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
