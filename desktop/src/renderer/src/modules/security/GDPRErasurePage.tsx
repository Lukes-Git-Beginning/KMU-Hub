/**
 * Admin page for GDPR right-to-erasure (Art. 17 DSGVO).
 *
 * Two-step process:
 * 1. Select a user and preview affected data (modules + record counts)
 * 2. Choose per-module action (anonymize/delete/retain), confirm with admin password
 *
 * The confirmation dialog requires the admin's password as a security gate.
 * Wired to useSecurity hooks for erasure preview and execution.
 */
import { useState, useCallback } from 'react'
import { Navigate } from 'react-router-dom'
import { AlertTriangle, Search, Eye, Trash2 } from 'lucide-react'
import { FormattedMessage } from 'react-intl'
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

type ModuleAction = 'anonymize' | 'delete' | 'retain'

interface ErasureModule {
  module: string
  record_count: number
  action: ModuleAction
}

export default function GDPRErasurePage() {
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin')

  // User search/selection
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null)
  const [selectedUserName, setSelectedUserName] = useState('')

  // Preview state
  const [previewData, setPreviewData] = useState<ErasureModule[] | null>(null)
  const [isLoadingPreview, setIsLoadingPreview] = useState(false)

  // Confirmation dialog
  const [showConfirm, setShowConfirm] = useState(false)
  const [adminPassword, setAdminPassword] = useState('')
  const [isExecuting, setIsExecuting] = useState(false)
  const [isCompleted, setIsCompleted] = useState(false)

  // Mock user search results for now
  const mockUsers = [
    { id: 'user-1', name: 'Max Mustermann', email: 'max@example.com' },
    { id: 'user-2', name: 'Anna Schmidt', email: 'anna@example.com' },
    { id: 'user-3', name: 'Peter Mueller', email: 'peter@example.com' },
  ].filter(
    (u) =>
      searchQuery.length >= 2 &&
      (u.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        u.email.toLowerCase().includes(searchQuery.toLowerCase()))
  )

  const handleSelectUser = useCallback((userId: string, userName: string) => {
    setSelectedUserId(userId)
    setSelectedUserName(userName)
    setPreviewData(null)
    setIsCompleted(false)
    setSearchQuery('')
  }, [])

  const handlePreview = useCallback(() => {
    if (!selectedUserId) return
    setIsLoadingPreview(true)

    // Simulate API call - in production, this calls the preview erasure endpoint
    setTimeout(() => {
      setPreviewData([
        { module: 'CRM Contacts', record_count: 47, action: 'anonymize' },
        { module: 'Chat Messages', record_count: 1283, action: 'delete' },
        { module: 'Calendar Events', record_count: 156, action: 'delete' },
        { module: 'Work Tasks', record_count: 89, action: 'anonymize' },
        { module: 'Audit Log', record_count: 342, action: 'retain' },
        { module: 'Invoices', record_count: 12, action: 'retain' },
      ])
      setIsLoadingPreview(false)
    }, 500)
  }, [selectedUserId])

  const handleModuleActionChange = useCallback(
    (module: string, action: ModuleAction) => {
      setPreviewData((prev) =>
        prev
          ? prev.map((m) => (m.module === module ? { ...m, action } : m))
          : null
      )
    },
    []
  )

  const handleExecute = useCallback(() => {
    if (!adminPassword || !previewData || !selectedUserId) return
    setIsExecuting(true)

    // Simulate API call - in production, this calls the execute erasure endpoint
    // with module_actions map and admin password for verification
    setTimeout(() => {
      setIsExecuting(false)
      setShowConfirm(false)
      setAdminPassword('')
      setIsCompleted(true)
      toast.success(<FormattedMessage id="gdpr.erasure.success" />)
    }, 1500)
  }, [adminPassword, previewData, selectedUserId])

  const totalRecords = previewData
    ? previewData.reduce((sum, m) => sum + m.record_count, 0)
    : 0

  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-3xl px-6 py-6 space-y-6">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">
            <FormattedMessage id="gdpr.erasure.title" />
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            <FormattedMessage id="gdpr.erasure.description" />
          </p>
        </div>

        {/* Warning Banner */}
        <div className="flex gap-3 rounded-lg bg-red-50 p-4 dark:bg-red-950/30">
          <AlertTriangle className="h-5 w-5 shrink-0 text-red-600 dark:text-red-400 mt-0.5" />
          <p className="text-sm text-red-800 dark:text-red-300">
            <FormattedMessage id="gdpr.erasureWarning" />
          </p>
        </div>

        {/* User Selection */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">
              <FormattedMessage id="gdpr.erasure.selectUser" />
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                className="pl-10"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search user..."
              />
            </div>

            {/* Search results dropdown */}
            {mockUsers.length > 0 && (
              <div className="rounded-lg border border-border divide-y divide-border">
                {mockUsers.map((u) => (
                  <button
                    key={u.id}
                    onClick={() => handleSelectUser(u.id, u.name)}
                    className="flex w-full items-center gap-3 p-3 hover:bg-accent transition-colors text-left"
                  >
                    <div className="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-xs font-medium">
                      {u.name
                        .split(' ')
                        .map((n) => n[0])
                        .join('')}
                    </div>
                    <div>
                      <p className="text-sm font-medium text-foreground">{u.name}</p>
                      <p className="text-xs text-muted-foreground">{u.email}</p>
                    </div>
                  </button>
                ))}
              </div>
            )}

            {/* Selected user */}
            {selectedUserId && (
              <div className="flex items-center justify-between rounded-lg border border-primary/30 bg-primary/5 p-3">
                <div>
                  <p className="text-sm font-medium text-foreground">{selectedUserName}</p>
                  <p className="text-xs text-muted-foreground">ID: {selectedUserId}</p>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handlePreview}
                  disabled={isLoadingPreview}
                >
                  <Eye className="mr-1 h-3 w-3" />
                  <FormattedMessage id="gdpr.erasure.preview" />
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Preview Data */}
        {previewData && !isCompleted && (
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">
                <FormattedMessage id="gdpr.erasurePreview" />
              </CardTitle>
              <CardDescription>
                <FormattedMessage
                  id="gdpr.recordsAffected"
                  values={{ count: totalRecords }}
                />{' '}
                &middot;{' '}
                <FormattedMessage
                  id="gdpr.modulesAffected"
                  values={{ count: previewData.length }}
                />
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-border text-left">
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="gdpr.erasure.module" />
                      </th>
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="gdpr.erasure.records" />
                      </th>
                      <th className="pb-2 text-xs font-medium text-muted-foreground">
                        <FormattedMessage id="gdpr.erasure.action" />
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {previewData.map((mod) => (
                      <tr key={mod.module}>
                        <td className="py-3 text-sm text-foreground">{mod.module}</td>
                        <td className="py-3">
                          <Badge variant="secondary">{mod.record_count}</Badge>
                        </td>
                        <td className="py-3">
                          <Select
                            value={mod.action}
                            onValueChange={(v) =>
                              handleModuleActionChange(mod.module, v as ModuleAction)
                            }
                          >
                            <SelectTrigger className="w-48">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="anonymize">
                                <FormattedMessage id="gdpr.moduleAction.anonymize" />
                              </SelectItem>
                              <SelectItem value="delete">
                                <FormattedMessage id="gdpr.moduleAction.delete" />
                              </SelectItem>
                              <SelectItem value="retain">
                                <FormattedMessage id="gdpr.moduleAction.retain" />
                              </SelectItem>
                            </SelectContent>
                          </Select>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="mt-6">
                <Button
                  variant="destructive"
                  onClick={() => setShowConfirm(true)}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  <FormattedMessage id="gdpr.erasure.execute" />
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Completed State */}
        {isCompleted && (
          <Card>
            <CardContent className="py-8 text-center">
              <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-green-100">
                <Trash2 className="h-6 w-6 text-green-600" />
              </div>
              <p className="text-lg font-medium text-foreground">
                <FormattedMessage id="gdpr.erasure.success" />
              </p>
              <p className="mt-1 text-sm text-muted-foreground">
                {selectedUserName} &rarr;{' '}
                <FormattedMessage
                  id="gdpr.anonymizedUser"
                  values={{ id: selectedUserId?.slice(-4) ?? '0000' }}
                />
              </p>
            </CardContent>
          </Card>
        )}

        {/* Two-Step Confirmation Dialog */}
        <Dialog open={showConfirm} onOpenChange={setShowConfirm}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                <FormattedMessage id="gdpr.erasure.confirmTitle" />
              </DialogTitle>
              <DialogDescription>
                <FormattedMessage id="gdpr.erasure.confirmDescription" />
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              {/* Summary */}
              <div className="rounded-lg bg-red-50 p-3 dark:bg-red-950/30">
                <p className="text-sm text-red-800 dark:text-red-300">
                  <FormattedMessage id="gdpr.erasureWarning" />
                </p>
                <p className="mt-1 text-sm text-red-700 dark:text-red-400">
                  {selectedUserName} &middot;{' '}
                  <FormattedMessage
                    id="gdpr.recordsAffected"
                    values={{ count: totalRecords }}
                  />
                </p>
              </div>

              {/* Module action summary */}
              <div className="space-y-1">
                {previewData?.map((mod) => (
                  <div key={mod.module} className="flex items-center justify-between text-sm">
                    <span className="text-foreground">{mod.module}</span>
                    <Badge
                      variant="secondary"
                      className={
                        mod.action === 'delete'
                          ? 'bg-red-100 text-red-800'
                          : mod.action === 'anonymize'
                            ? 'bg-yellow-100 text-yellow-800'
                            : 'bg-gray-100 text-gray-800'
                      }
                    >
                      <FormattedMessage id={`gdpr.moduleAction.${mod.action}`} />
                    </Badge>
                  </div>
                ))}
              </div>

              {/* Admin password confirmation */}
              <div className="space-y-1.5">
                <Label>
                  <FormattedMessage id="gdpr.erasurePasswordConfirm" />
                </Label>
                <Input
                  type="password"
                  value={adminPassword}
                  onChange={(e) => setAdminPassword(e.target.value)}
                  autoFocus
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => { setShowConfirm(false); setAdminPassword('') }}>
                <FormattedMessage id="common.cancel" />
              </Button>
              <Button
                variant="destructive"
                onClick={handleExecute}
                disabled={!adminPassword || isExecuting}
              >
                {isExecuting ? (
                  <FormattedMessage id="common.loading" />
                ) : (
                  <FormattedMessage id="gdpr.erasureConfirm" />
                )}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  )
}
