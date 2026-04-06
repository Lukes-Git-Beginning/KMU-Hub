/**
 * Dialog for approving plugin permissions.
 *
 * Shows the list of required permissions and allows the admin to
 * approve them before a plugin can become active.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Shield, Loader2 } from 'lucide-react'
import { useApprovePermissions } from '@/api/hooks/usePlugins'
import type { PluginInstallation } from '@/api/plugin-types'

interface PermissionApprovalDialogProps {
  isOpen: boolean
  onClose: () => void
  installation: PluginInstallation | null
}

/** Human-readable label for known permission strings */
function permissionLabel(permission: string, t: (key: string) => string): string {
  const labels: Record<string, string> = {
    'read:contacts': t('admin.plugins.permissions.readContacts'),
    'write:contacts': t('admin.plugins.permissions.writeContacts'),
    'read:deals': t('admin.plugins.permissions.readDeals'),
    'write:deals': t('admin.plugins.permissions.writeDeals'),
    'read:invoices': t('admin.plugins.permissions.readInvoices'),
    'write:invoices': t('admin.plugins.permissions.writeInvoices'),
    'read:settings': t('admin.plugins.permissions.readSettings'),
    'write:settings': t('admin.plugins.permissions.writeSettings'),
    'execute:hooks': t('admin.plugins.permissions.executeHooks'),
    'read:custom_fields': t('admin.plugins.permissions.readCustomFields'),
    'write:custom_fields': t('admin.plugins.permissions.writeCustomFields'),
  }
  return labels[permission] ?? permission
}

export function PermissionApprovalDialog({
  isOpen,
  onClose,
  installation,
}: PermissionApprovalDialogProps) {
  const { t } = useTranslation()
  const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(
    new Set(),
  )
  const approvePermissions = useApprovePermissions()

  if (!installation) return null

  const requiredPermissions = installation.required_permissions ?? []
  const grantedPermissions = new Set(installation.granted_permissions ?? [])
  const pendingPermissions = requiredPermissions.filter(
    (p) => !grantedPermissions.has(p),
  )

  const togglePermission = (permission: string) => {
    setSelectedPermissions((prev) => {
      const next = new Set(prev)
      if (next.has(permission)) {
        next.delete(permission)
      } else {
        next.add(permission)
      }
      return next
    })
  }

  const selectAll = () => {
    setSelectedPermissions(new Set(pendingPermissions))
  }

  const handleApprove = async () => {
    if (selectedPermissions.size === 0) return
    await approvePermissions.mutateAsync({
      installationId: installation.id,
      data: { permissions: Array.from(selectedPermissions) },
    })
    setSelectedPermissions(new Set())
    onClose()
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Shield className="h-4 w-4 text-muted-foreground" />
            {t('admin.plugins.permissions.approveTitle')}
          </DialogTitle>
          <DialogDescription>
            {t('admin.plugins.permissions.approveDescription', { name: installation.manifest_name })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          {/* Already granted */}
          {grantedPermissions.size > 0 && (
            <div className="space-y-1.5">
              <p className="text-xs font-medium text-muted-foreground">
                {t('admin.plugins.permissions.alreadyApproved')}
              </p>
              {Array.from(grantedPermissions).map((p) => (
                <div
                  key={p}
                  className="flex items-center gap-2 rounded-md border border-green-500/30 bg-green-50/10 px-3 py-1.5"
                >
                  <Checkbox checked disabled />
                  <Label className="text-sm text-green-700 dark:text-green-400">
                    {permissionLabel(p, t)}
                  </Label>
                </div>
              ))}
            </div>
          )}

          {/* Pending approval */}
          {pendingPermissions.length > 0 && (
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium text-muted-foreground">
                  {t('admin.plugins.permissions.pendingApproval')}
                </p>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 text-xs"
                  onClick={selectAll}
                >
                  {t('admin.plugins.permissions.selectAll')}
                </Button>
              </div>
              {pendingPermissions.map((p) => (
                <div
                  key={p}
                  className="flex items-center gap-2 rounded-md border border-border px-3 py-1.5 cursor-pointer hover:bg-accent/30"
                  onClick={() => togglePermission(p)}
                >
                  <Checkbox
                    checked={selectedPermissions.has(p)}
                    onCheckedChange={() => togglePermission(p)}
                  />
                  <Label className="text-sm cursor-pointer">
                    {permissionLabel(p, t)}
                  </Label>
                </div>
              ))}
            </div>
          )}

          {pendingPermissions.length === 0 && (
            <div className="rounded-md border border-green-500/30 bg-green-50/10 p-3">
              <p className="text-sm text-green-700 dark:text-green-400">
                {t('admin.plugins.permissions.allApproved')}
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="ghost" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            size="sm"
            onClick={handleApprove}
            disabled={
              selectedPermissions.size === 0 || approvePermissions.isPending
            }
          >
            {approvePermissions.isPending ? (
              <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
            ) : (
              <Shield className="h-3.5 w-3.5 mr-1.5" />
            )}
            {t('admin.plugins.permissions.approveCount', { count: selectedPermissions.size })}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
