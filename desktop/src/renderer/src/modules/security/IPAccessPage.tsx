/**
 * Admin page for IP access control (allowlist/blocklist).
 *
 * Displays existing IP rules in a table with CIDR, type, description, and
 * creation date. Provides a dialog to add new rules and delete existing ones.
 * Wired to useIPRules, useCreateIPRule, useDeleteIPRule hooks.
 */
import { useState, useCallback } from 'react'
import { Navigate } from 'react-router-dom'
import { Plus, Trash2, Shield, Info } from 'lucide-react'
import { FormattedMessage, FormattedDate } from 'react-intl'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
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
          toast.success(<FormattedMessage id="common.success" />)
          setShowAdd(false)
          setNewCidr('')
          setNewType('allow')
          setNewDescription('')
        },
        onError: () => toast.error(<FormattedMessage id="common.error" />),
      }
    )
  }, [newCidr, newType, newDescription, createRule])

  const handleDelete = useCallback(() => {
    if (!deleteId) return
    deleteRule.mutate(deleteId, {
      onSuccess: () => {
        toast.success(<FormattedMessage id="common.success" />)
        setDeleteId(null)
      },
      onError: () => toast.error(<FormattedMessage id="common.error" />),
    })
  }, [deleteId, deleteRule])

  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-4xl px-6 py-6 space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">
              <FormattedMessage id="ipAccess.title" />
            </h1>
            <p className="mt-1 text-sm text-muted-foreground">
              <FormattedMessage id="ipAccess.description" />
            </p>
          </div>
          <Button onClick={() => setShowAdd(true)}>
            <Plus className="mr-2 h-4 w-4" />
            <FormattedMessage id="ipAccess.addRule" />
          </Button>
        </div>

        {/* Info Box */}
        <div className="flex gap-3 rounded-lg bg-blue-50 p-4 dark:bg-blue-950/30">
          <Info className="h-5 w-5 shrink-0 text-blue-600 dark:text-blue-400 mt-0.5" />
          <p className="text-sm text-blue-800 dark:text-blue-300">
            <FormattedMessage id="ipAccess.allowlistInfo" />
          </p>
        </div>

        {/* Rules Table */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <Shield className="h-5 w-5" />
              <FormattedMessage id="ipAccess.title" />
            </CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <p className="text-sm text-muted-foreground py-8 text-center">
                <FormattedMessage id="common.loading" />
              </p>
            ) : !rules || rules.length === 0 ? (
              <p className="text-sm text-muted-foreground py-8 text-center">
                <FormattedMessage id="ipAccess.noRules" />
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-border text-left">
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="ipAccess.cidr" />
                      </th>
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="ipAccess.type" />
                      </th>
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="ipAccess.description_field" />
                      </th>
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="ipAccess.createdAt" />
                      </th>
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="common.actions" />
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {rules.map((rule) => (
                      <tr key={rule.id} className="group">
                        <td className="py-3 text-sm font-mono text-foreground">
                          {rule.ip_cidr}
                        </td>
                        <td className="py-3">
                          <Badge
                            variant="secondary"
                            className={
                              rule.rule_type === 'allow'
                                ? 'bg-green-100 text-green-800'
                                : 'bg-red-100 text-red-800'
                            }
                          >
                            <FormattedMessage
                              id={
                                rule.rule_type === 'allow'
                                  ? 'ipAccess.type.allow'
                                  : 'ipAccess.type.block'
                              }
                            />
                          </Badge>
                        </td>
                        <td className="py-3 text-sm text-muted-foreground">
                          {rule.description ?? '-'}
                        </td>
                        <td className="py-3 text-sm text-muted-foreground">
                          {rule.created_at ? (
                            <FormattedDate
                              value={rule.created_at}
                              year="numeric"
                              month="short"
                              day="numeric"
                            />
                          ) : (
                            '-'
                          )}
                        </td>
                        <td className="py-3">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
                            onClick={() => setDeleteId(rule.id)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Add Rule Dialog */}
        <Dialog open={showAdd} onOpenChange={setShowAdd}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                <FormattedMessage id="ipAccess.addRule" />
              </DialogTitle>
              <DialogDescription>
                <FormattedMessage id="ipAccess.description" />
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-1.5">
                <Label>
                  <FormattedMessage id="ipAccess.cidr" />
                </Label>
                <Input
                  value={newCidr}
                  onChange={(e) => setNewCidr(e.target.value)}
                  placeholder="192.168.1.0/24"
                />
              </div>
              <div className="space-y-1.5">
                <Label>
                  <FormattedMessage id="ipAccess.type" />
                </Label>
                <Select value={newType} onValueChange={(v) => setNewType(v as 'allow' | 'block')}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="allow">
                      <FormattedMessage id="ipAccess.type.allow" />
                    </SelectItem>
                    <SelectItem value="block">
                      <FormattedMessage id="ipAccess.type.block" />
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label>
                  <FormattedMessage id="ipAccess.description_field" /> (
                  <FormattedMessage id="common.optional" />)
                </Label>
                <Input
                  value={newDescription}
                  onChange={(e) => setNewDescription(e.target.value)}
                  placeholder="Office network"
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setShowAdd(false)}>
                <FormattedMessage id="common.cancel" />
              </Button>
              <Button onClick={handleAdd} disabled={!newCidr.trim() || createRule.isPending}>
                <FormattedMessage id="common.create" />
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Delete Confirmation Dialog */}
        <Dialog open={deleteId !== null} onOpenChange={(open) => { if (!open) setDeleteId(null) }}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                <FormattedMessage id="ipAccess.deleteConfirm" />
              </DialogTitle>
            </DialogHeader>
            <DialogFooter>
              <Button variant="outline" onClick={() => setDeleteId(null)}>
                <FormattedMessage id="common.cancel" />
              </Button>
              <Button
                variant="destructive"
                onClick={handleDelete}
                disabled={deleteRule.isPending}
              >
                <FormattedMessage id="common.delete" />
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  )
}
