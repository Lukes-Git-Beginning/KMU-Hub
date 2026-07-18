/**
 * CloneRoleDialog (R-2) — the only way to create a custom role: clone a
 * preset/custom base, name it, pick an accent. Grants start as a copy of the
 * base; the editor then captures the deviations. Similar-name warning guards
 * against role sprawl (KONZEPT §3 Fallen).
 */
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Copy } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { Role } from '@/api/rbac-types'
import { useCreateRole } from '@/api/hooks/useRbacRoles'
import { rbacErrorMessage, roleDisplayName } from '@/lib/rbac-format'

/** Curated accents for custom roles (presets own the strong semantic hues). */
const ROLE_COLORS = [
  'hsl(38 92% 50%)',
  'hsl(160 84% 39%)',
  'hsl(200 98% 39%)',
  'hsl(245 58% 61%)',
  'hsl(330 81% 60%)',
  'hsl(15 86% 50%)',
  'hsl(84 59% 40%)',
  'hsl(190 77% 40%)',
]

export default function CloneRoleDialog({
  open,
  onOpenChange,
  roles,
  initialBase,
  onCreated,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  roles: Role[]
  /** Preselected base role (card "Klonen" action) — null for the toolbar button. */
  initialBase: Role | null
  onCreated: (role: Role) => void
}) {
  const { t } = useTranslation()
  const createRole = useCreateRole()
  const [basedOn, setBasedOn] = useState<string>('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [color, setColor] = useState(ROLE_COLORS[0])

  // Seed form state per open (remount-light: reset when dialog opens).
  useEffect(() => {
    if (open) {
      setBasedOn(initialBase?.id ?? roles.find((r) => r.isSystem)?.id ?? '')
      setName('')
      setDescription('')
      setColor(ROLE_COLORS[0])
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, initialBase])

  const similarName = useMemo(() => {
    const needle = name.trim().toLowerCase()
    if (needle.length < 3) return null
    return (
      roles.find((r) => {
        const existing = roleDisplayName(t, r).toLowerCase()
        return existing.includes(needle) || needle.includes(existing)
      }) ?? null
    )
  }, [name, roles, t])

  const submit = () => {
    if (!name.trim() || !basedOn) return
    createRole.mutate(
      { name: name.trim(), description: description.trim(), color, basedOn },
      {
        onSuccess: (role) => {
          toast.success(t('rbac.builder.cloneDone', { name: role.name }))
          onCreated(role)
        },
        onError: (err) => toast.error(rbacErrorMessage(t, err)),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Copy className="h-4 w-4 text-primary" aria-hidden="true" />
            {t('rbac.builder.cloneTitle')}
          </DialogTitle>
          <DialogDescription>{t('rbac.builder.cloneSubtitle')}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="clone-base">{t('rbac.builder.cloneBase')}</Label>
            <Select value={basedOn} onValueChange={setBasedOn}>
              <SelectTrigger id="clone-base">
                <SelectValue placeholder={t('rbac.builder.cloneBasePlaceholder')} />
              </SelectTrigger>
              <SelectContent>
                {roles.map((role) => (
                  <SelectItem key={role.id} value={role.id}>
                    <span className="flex items-center gap-2">
                      <span className="h-2 w-2 rounded-full" style={{ background: role.color }} aria-hidden="true" />
                      {roleDisplayName(t, role)}
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">{t('rbac.builder.cloneBaseHint')}</p>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="clone-name">{t('rbac.builder.cloneName')}</Label>
            <Input
              id="clone-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('rbac.builder.cloneNamePlaceholder')}
              maxLength={60}
            />
            {similarName && (
              <p className="text-xs text-warning">
                {t('rbac.builder.similarNameWarning', { name: roleDisplayName(t, similarName) })}
              </p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="clone-description">{t('rbac.builder.cloneDescription')}</Label>
            <Textarea
              id="clone-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t('rbac.builder.cloneDescriptionPlaceholder')}
              rows={2}
              maxLength={160}
            />
          </div>

          <div className="space-y-1.5">
            <Label>{t('rbac.builder.cloneColor')}</Label>
            <div className="flex flex-wrap gap-2" role="radiogroup" aria-label={t('rbac.builder.cloneColor')}>
              {ROLE_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  role="radio"
                  aria-checked={color === c}
                  aria-label={c}
                  onClick={() => setColor(c)}
                  className={`h-7 w-7 rounded-full border-2 transition-transform focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring ${
                    color === c ? 'scale-110 border-foreground' : 'border-transparent hover:scale-105'
                  }`}
                  style={{ background: c }}
                />
              ))}
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button onClick={submit} disabled={!name.trim() || !basedOn || createRole.isPending}>
            {t('rbac.builder.cloneAction')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
