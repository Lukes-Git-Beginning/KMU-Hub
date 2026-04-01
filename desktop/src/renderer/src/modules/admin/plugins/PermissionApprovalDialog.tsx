/**
 * Dialog for approving plugin permissions.
 *
 * Shows the list of required permissions and allows the admin to
 * approve them before a plugin can become active.
 */
import { useState } from 'react'
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
function permissionLabel(permission: string): string {
  const labels: Record<string, string> = {
    'read:contacts': 'Kontakte lesen',
    'write:contacts': 'Kontakte schreiben',
    'read:deals': 'Deals lesen',
    'write:deals': 'Deals schreiben',
    'read:invoices': 'Rechnungen lesen',
    'write:invoices': 'Rechnungen schreiben',
    'read:settings': 'Einstellungen lesen',
    'write:settings': 'Einstellungen schreiben',
    'execute:hooks': 'Hooks ausführen',
    'read:custom_fields': 'Benutzerdefinierte Felder lesen',
    'write:custom_fields': 'Benutzerdefinierte Felder schreiben',
  }
  return labels[permission] ?? permission
}

export function PermissionApprovalDialog({
  isOpen,
  onClose,
  installation,
}: PermissionApprovalDialogProps) {
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
            Berechtigungen genehmigen
          </DialogTitle>
          <DialogDescription>
            {installation.manifest_name} benötigt folgende Berechtigungen:
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          {/* Already granted */}
          {grantedPermissions.size > 0 && (
            <div className="space-y-1.5">
              <p className="text-xs font-medium text-muted-foreground">
                Bereits genehmigt:
              </p>
              {Array.from(grantedPermissions).map((p) => (
                <div
                  key={p}
                  className="flex items-center gap-2 rounded-md border border-green-500/30 bg-green-50/10 px-3 py-1.5"
                >
                  <Checkbox checked disabled />
                  <Label className="text-sm text-green-700 dark:text-green-400">
                    {permissionLabel(p)}
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
                  Ausstehende Genehmigung:
                </p>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 text-xs"
                  onClick={selectAll}
                >
                  Alle auswählen
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
                    {permissionLabel(p)}
                  </Label>
                </div>
              ))}
            </div>
          )}

          {pendingPermissions.length === 0 && (
            <div className="rounded-md border border-green-500/30 bg-green-50/10 p-3">
              <p className="text-sm text-green-700 dark:text-green-400">
                Alle Berechtigungen wurden bereits genehmigt.
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="ghost" size="sm" onClick={onClose}>
            Abbrechen
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
            {selectedPermissions.size} Berechtigung
            {selectedPermissions.size !== 1 ? 'en' : ''} genehmigen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
