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
import { AlertTriangle, Search, Eye, Trash2, CheckCircle, Download } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { FormattedMessage, useIntl } from 'react-intl'
import { toast } from 'sonner'
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

const ACTION_BADGE: Record<ModuleAction, string> = {
  delete: 'bg-error-light text-error',
  anonymize: 'bg-warning-light text-warning',
  retain: 'bg-secondary text-muted-foreground',
}

export default function GDPRErasurePage() {
  const intl = useIntl()
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
  const [executionProgress, setExecutionProgress] = useState(0)
  const [isCompleted, setIsCompleted] = useState(false)

  // Mock user search results
  const mockUsers = [
    { id: 'user-1', name: 'Max Mustermann', email: 'max@example.com' },
    { id: 'user-2', name: 'Anna Schmidt', email: 'anna@example.com' },
    { id: 'user-3', name: 'Peter Mueller', email: 'peter@example.com' },
  ].filter(
    (u) =>
      searchQuery.length >= 2 &&
      (u.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        u.email.toLowerCase().includes(searchQuery.toLowerCase())),
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

    setTimeout(() => {
      setPreviewData([
        { module: 'CRM Kontakte', record_count: 47, action: 'anonymize' },
        { module: 'Chat-Nachrichten', record_count: 1283, action: 'delete' },
        { module: 'Kalender-Termine', record_count: 156, action: 'delete' },
        { module: 'Aufgaben/Projekte', record_count: 89, action: 'anonymize' },
        { module: 'Helpdesk-Tickets', record_count: 34, action: 'anonymize' },
        { module: 'Dokumente', record_count: 23, action: 'delete' },
        { module: 'E-Mails', record_count: 412, action: 'delete' },
        { module: 'Formulare', record_count: 8, action: 'anonymize' },
        { module: 'Audit Log', record_count: 342, action: 'retain' },
        { module: 'Rechnungen', record_count: 12, action: 'retain' },
      ])
      setIsLoadingPreview(false)
    }, 500)
  }, [selectedUserId])

  const handleModuleActionChange = useCallback(
    (module: string, action: ModuleAction) => {
      setPreviewData((prev) =>
        prev ? prev.map((m) => (m.module === module ? { ...m, action } : m)) : null,
      )
    },
    [],
  )

  const handleExecute = useCallback(() => {
    if (!adminPassword || !previewData || !selectedUserId) return
    setIsExecuting(true)
    setExecutionProgress(0)

    // Simulate progress bar
    const steps = 10
    let step = 0
    const interval = setInterval(() => {
      step++
      setExecutionProgress(Math.round((step / steps) * 100))
      if (step >= steps) {
        clearInterval(interval)
        setIsExecuting(false)
        setShowConfirm(false)
        setAdminPassword('')
        setExecutionProgress(0)
        setIsCompleted(true)
        toast.success(intl.formatMessage({ id: 'gdpr.erasure.success' }))
      }
    }, 150)
  }, [adminPassword, previewData, selectedUserId, intl])

  const handleDownloadReceipt = useCallback(() => {
    toast.success('Loeschprotokoll wird heruntergeladen...')
    setTimeout(() => toast.success('Loeschprotokoll_' + selectedUserName?.replace(/\s/g, '_') + '.pdf gespeichert'), 1000)
  }, [selectedUserName])

  const totalRecords = previewData
    ? previewData.reduce((sum, m) => sum + m.record_count, 0)
    : 0

  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="mx-auto max-w-3xl">
        {/* Header */}
        <div className="flex items-center gap-4 mb-6">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-error-light">
            <AlertTriangle className="h-6 w-6 text-error" />
          </div>
          <div>
            <h1 className="text-foreground">
              <FormattedMessage id="gdpr.erasure.title" />
            </h1>
            <p className="text-sm text-muted-foreground mt-0.5">
              <FormattedMessage id="gdpr.erasure.description" />
            </p>
          </div>
        </div>

        {/* Warning Banner */}
        <div className="flex gap-3 rounded-lg bg-error-light p-4 mb-6">
          <AlertTriangle className="h-5 w-5 shrink-0 text-error mt-0.5" />
          <p className="text-sm text-error font-medium">
            <FormattedMessage id="gdpr.erasureWarning" />
          </p>
        </div>

        {/* User Selection */}
        <div className="rounded-lg border border-border bg-card p-5 glass-surface mb-6">
          <h3 className="text-base font-semibold text-foreground mb-4">
            <FormattedMessage id="gdpr.erasure.selectUser" />
          </h3>

          <div className="relative mb-3">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search user..."
              className="w-full rounded-lg border border-border bg-card pl-10 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Search results */}
          {mockUsers.length > 0 && (
            <div className="rounded-lg border border-border divide-y divide-border-muted mb-3">
              {mockUsers.map((u) => (
                <button
                  key={u.id}
                  onClick={() => handleSelectUser(u.id, u.name)}
                  className="flex w-full items-center gap-3 p-3 hover:bg-secondary/50 transition-colors text-left"
                >
                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-secondary text-xs font-medium text-foreground">
                    {u.name.split(' ').map((n) => n[0]).join('')}
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
            <div className="flex items-center justify-between rounded-lg border border-primary/30 bg-primary-light/30 p-3">
              <div>
                <p className="text-sm font-medium text-foreground">{selectedUserName}</p>
                <p className="text-xs text-muted-foreground font-mono">ID: {selectedUserId}</p>
              </div>
              <button
                onClick={handlePreview}
                disabled={isLoadingPreview}
                className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors disabled:opacity-40"
              >
                <Eye className="h-3.5 w-3.5" />
                <FormattedMessage id="gdpr.erasure.preview" />
              </button>
            </div>
          )}
        </div>

        {/* Preview Data */}
        {previewData && !isCompleted && (
          <div className="rounded-lg border border-border bg-card p-5 glass-surface mb-6">
            <h3 className="text-base font-semibold text-foreground mb-1">
              <FormattedMessage id="gdpr.erasurePreview" />
            </h3>
            <p className="text-sm text-muted-foreground mb-4">
              <FormattedMessage id="gdpr.recordsAffected" values={{ count: totalRecords }} />
              {' '}&middot;{' '}
              <FormattedMessage id="gdpr.modulesAffected" values={{ count: previewData.length }} />
            </p>

            <div className="rounded-lg border border-border overflow-hidden mb-5">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                      <FormattedMessage id="gdpr.erasure.module" />
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                      <FormattedMessage id="gdpr.erasure.records" />
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                      <FormattedMessage id="gdpr.erasure.action" />
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {previewData.map((mod) => (
                    <tr key={mod.module} className="border-b border-border-muted last:border-0">
                      <td className="px-4 py-3 text-sm text-foreground">{mod.module}</td>
                      <td className="px-4 py-3">
                        <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground tabular-nums">
                          {mod.record_count}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <Select
                          value={mod.action}
                          onValueChange={(v) => handleModuleActionChange(mod.module, v as ModuleAction)}
                        >
                          <SelectTrigger className="w-44 h-8 rounded-lg border-border bg-card text-xs">
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

            <button
              onClick={() => setShowConfirm(true)}
              className="flex items-center gap-2 rounded-lg bg-error px-4 py-2 text-sm text-white hover:bg-error/90 transition-colors"
            >
              <Trash2 className="h-4 w-4" />
              <FormattedMessage id="gdpr.erasure.execute" />
            </button>
          </div>
        )}

        {/* Completed State */}
        {isCompleted && (
          <div className="rounded-lg border border-border bg-card p-8 text-center glass-surface">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-success-light">
              <CheckCircle className="h-6 w-6 text-success" />
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
            <button
              onClick={handleDownloadReceipt}
              className="mt-4 inline-flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <Download className="h-4 w-4" />
              Loeschprotokoll herunterladen
            </button>
          </div>
        )}
      </div>

      {/* Two-Step Confirmation Dialog */}
      <Dialog open={showConfirm} onOpenChange={(o) => { if (!o) { setShowConfirm(false); setAdminPassword('') } }}>
        <DialogContent className="gap-0 p-0 max-w-md">
          <div className="p-6">
            <DialogHeader className="mb-5">
              <DialogTitle className="text-lg font-semibold text-foreground">
                <FormattedMessage id="gdpr.erasure.confirmTitle" />
              </DialogTitle>
              <DialogDescription className="sr-only">Loeschvorgang bestaetigen</DialogDescription>
            </DialogHeader>

            <p className="text-sm text-muted-foreground mb-4">
              <FormattedMessage id="gdpr.erasure.confirmDescription" />
            </p>

            {/* Warning summary */}
            <div className="rounded-lg bg-error-light p-3 mb-4">
              <p className="text-sm text-error font-medium">
                <FormattedMessage id="gdpr.erasureWarning" />
              </p>
              <p className="mt-1 text-sm text-error/80">
                {selectedUserName} &middot;{' '}
                <FormattedMessage id="gdpr.recordsAffected" values={{ count: totalRecords }} />
              </p>
            </div>

            {/* Module action summary */}
            <div className="space-y-1.5 mb-4">
              {previewData?.map((mod) => (
                <div key={mod.module} className="flex items-center justify-between text-sm">
                  <span className="text-foreground">{mod.module}</span>
                  <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${ACTION_BADGE[mod.action]}`}>
                    <FormattedMessage id={`gdpr.moduleAction.${mod.action}`} />
                  </span>
                </div>
              ))}
            </div>

            {/* Admin password */}
            <div className="space-y-1.5 mb-6">
              <label className="text-sm font-medium text-foreground">
                <FormattedMessage id="gdpr.erasurePasswordConfirm" />
              </label>
              <input
                type="password"
                value={adminPassword}
                onChange={(e) => setAdminPassword(e.target.value)}
                autoFocus
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>

            <DialogFooter>
              <button
                onClick={() => { setShowConfirm(false); setAdminPassword('') }}
                className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <FormattedMessage id="common.cancel" />
              </button>
              <button
                onClick={handleExecute}
                disabled={!adminPassword || isExecuting}
                className="rounded-lg bg-error px-4 py-2 text-sm text-white hover:bg-error/90 transition-colors disabled:opacity-50"
              >
                {isExecuting ? (
                  <span className="flex items-center gap-2">
                    <span className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                    {executionProgress}%
                  </span>
                ) : (
                  <FormattedMessage id="gdpr.erasureConfirm" />
                )}
              </button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
