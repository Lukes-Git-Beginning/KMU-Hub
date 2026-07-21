/**
 * CloneRoleDialog (R-2 + R-5) — the only way to create a custom role.
 *
 * Two modes accessible via a tab/segment control:
 *   1. "Klonen" (default, unchanged): clone a preset/custom base, name it,
 *      pick an accent. Grants start as a copy of the base; the editor then
 *      captures the deviations.
 *   2. "Aus Vorlage" (R-5 new): pick from one of 3 industry sets × 4 roles.
 *      Selecting a template pre-fills the form on step 2 (name/description/
 *      basedOn/color). Saving creates a normal custom role — ROLE_DEFS stay
 *      UNANGETASTET.
 *
 * Similar-name warning guards against role sprawl (KONZEPT §3 Fallen).
 */
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Copy, LayoutTemplate, ArrowLeft, Check } from 'lucide-react'
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
import { useCreateRole, useSetRoleGrants } from '@/api/hooks/useRbacRoles'
import { rbacErrorMessage, roleDisplayName } from '@/lib/rbac-format'
import {
  orderedSetsForProfile,
  type IndustryRoleSet,
  type IndustryRoleTemplate,
} from '@/mocks/data/industry-role-templates'
import { useProfileStore } from '@/stores/profile'

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

type DialogMode = 'clone' | 'template'
type TemplateStep = 'set' | 'role' | 'form'

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
  const setGrantsMutation = useSetRoleGrants()

  // Clone-mode state
  const [mode, setMode] = useState<DialogMode>('clone')
  const [basedOn, setBasedOn] = useState<string>('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [color, setColor] = useState(ROLE_COLORS[0])

  // Template-mode state
  const [templateStep, setTemplateStep] = useState<TemplateStep>('set')
  const [selectedSet, setSelectedSet] = useState<IndustryRoleSet | null>(null)
  const [selectedTemplate, setSelectedTemplate] = useState<IndustryRoleTemplate | null>(null)

  const businessProfileId = useProfileStore((s) => s.businessProfileId)
  const orderedSets = useMemo(
    () => orderedSetsForProfile(businessProfileId),
    [businessProfileId],
  )

  // Seed form state per open (remount-light: reset when dialog opens).
  useEffect(() => {
    if (open) {
      setMode(initialBase ? 'clone' : 'clone') // default to clone; user can switch
      setBasedOn(initialBase?.id ?? roles.find((r) => r.isSystem)?.id ?? '')
      setName('')
      setDescription('')
      setColor(ROLE_COLORS[0])
      setTemplateStep('set')
      setSelectedSet(null)
      setSelectedTemplate(null)
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

  // ── Template flow helpers ────────────────────────────────────────────────

  const handleSelectSet = (set: IndustryRoleSet) => {
    setSelectedSet(set)
    setTemplateStep('role')
  }

  const handleSelectTemplate = (template: IndustryRoleTemplate) => {
    setSelectedTemplate(template)
    // Pre-fill form fields
    setName(t(template.labelKey))
    setDescription(t(template.descriptionKey))
    setColor(template.color)
    setBasedOn(template.basedOn)
    setTemplateStep('form')
  }

  const handleTemplateBack = () => {
    if (templateStep === 'form') {
      setTemplateStep('role')
      setSelectedTemplate(null)
    } else if (templateStep === 'role') {
      setTemplateStep('set')
      setSelectedSet(null)
    }
  }

  // ── Submit (shared between both modes) ──────────────────────────────────

  const submit = () => {
    if (!name.trim() || !basedOn) return
    const templateGrants = mode === 'template' && selectedTemplate
      ? selectedTemplate.grants
      : undefined

    createRole.mutate(
      { name: name.trim(), description: description.trim(), color, basedOn },
      {
        onSuccess: (role) => {
          if (templateGrants) {
            // Convert GrantSpec (scope values) → RoleGrants ({ scope } objects)
            const roleGrants = Object.fromEntries(
              Object.entries(templateGrants).map(([key, scope]) => [key, { scope }]),
            )
            // Apply the template's curated grants on top of the just-created role
            setGrantsMutation.mutate(
              { roleId: role.id, input: { grants: roleGrants } },
              {
                onSuccess: () => {
                  toast.success(t('rbac.builder.cloneDone', { name: role.name }))
                  onCreated(role)
                },
                onError: (err) => {
                  // Role was created but grant-set failed — still surface the role
                  toast.error(rbacErrorMessage(t, err))
                  onCreated(role)
                },
              },
            )
          } else {
            toast.success(t('rbac.builder.cloneDone', { name: role.name }))
            onCreated(role)
          }
        },
        onError: (err) => toast.error(rbacErrorMessage(t, err)),
      },
    )
  }

  // Form step is shared between clone-mode and template-mode (step 3)
  const showForm = mode === 'clone' || (mode === 'template' && templateStep === 'form')

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {mode === 'clone' ? (
              <Copy className="h-4 w-4 text-primary" aria-hidden="true" />
            ) : (
              <LayoutTemplate className="h-4 w-4 text-primary" aria-hidden="true" />
            )}
            {mode === 'clone'
              ? t('rbac.builder.cloneTitle')
              : t('rbac.template.dialogTitle')}
          </DialogTitle>
          <DialogDescription>
            {mode === 'clone'
              ? t('rbac.builder.cloneSubtitle')
              : t('rbac.template.dialogSubtitle')}
          </DialogDescription>
        </DialogHeader>

        {/* Mode switcher tabs */}
        <div className="flex gap-1 rounded-lg border border-border bg-secondary/30 p-1" role="tablist">
          <button
            role="tab"
            aria-selected={mode === 'clone'}
            onClick={() => { setMode('clone'); setTemplateStep('set') }}
            className={`flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
              mode === 'clone'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <Copy className="h-3.5 w-3.5" aria-hidden="true" />
            {t('rbac.builder.clone')}
          </button>
          <button
            role="tab"
            aria-selected={mode === 'template'}
            onClick={() => { setMode('template'); setTemplateStep('set') }}
            className={`flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
              mode === 'template'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <LayoutTemplate className="h-3.5 w-3.5" aria-hidden="true" />
            {t('rbac.template.tabLabel')}
          </button>
        </div>

        {/* ── Template mode: Set selection ─────────────────────────────── */}
        {mode === 'template' && templateStep === 'set' && (
          <div className="space-y-2">
            <p className="text-xs text-muted-foreground">
              {t('rbac.template.setPrompt')}
            </p>
            <div className="space-y-2">
              {orderedSets.map((set) => (
                <button
                  key={set.id}
                  type="button"
                  onClick={() => handleSelectSet(set)}
                  className="flex w-full items-center justify-between rounded-lg border border-border bg-card px-4 py-3 text-left transition-all hover:border-primary/50 hover:bg-secondary/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                >
                  <div>
                    <p className="text-sm font-medium text-foreground">{t(set.labelKey)}</p>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {set.roles.length} {t('rbac.template.rolesCount')}
                    </p>
                  </div>
                  <div className="flex items-center gap-1.5 ml-3 shrink-0">
                    {set.roles.slice(0, 4).map((role) => (
                      <span
                        key={role.id}
                        className="h-2.5 w-2.5 rounded-full"
                        style={{ background: role.color }}
                        aria-hidden="true"
                      />
                    ))}
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* ── Template mode: Role selection ─────────────────────────────── */}
        {mode === 'template' && templateStep === 'role' && selectedSet && (
          <div className="space-y-2">
            <button
              type="button"
              onClick={handleTemplateBack}
              className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
              {t('rbac.template.backToSets')}
            </button>
            <p className="text-xs font-medium text-foreground">{t(selectedSet.labelKey)}</p>
            <div className="space-y-1.5">
              {selectedSet.roles.map((role) => (
                <button
                  key={role.id}
                  type="button"
                  onClick={() => handleSelectTemplate(role)}
                  className="flex w-full flex-col gap-1.5 rounded-lg border border-border bg-card px-4 py-3 text-left transition-all hover:border-primary/50 hover:bg-secondary/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                >
                  <div className="flex items-center gap-2">
                    <span
                      className="h-3 w-3 shrink-0 rounded-full"
                      style={{ background: role.color }}
                      aria-hidden="true"
                    />
                    <p className="text-sm font-medium text-foreground">{t(role.labelKey)}</p>
                  </div>
                  <ul className="ml-5 space-y-0.5">
                    {role.highlightKeys.map((hk) => (
                      <li key={hk} className="text-xs text-muted-foreground">
                        {t(hk)}
                      </li>
                    ))}
                  </ul>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* ── Shared form (clone-mode always, template-mode on step 3) ──── */}
        {showForm && (
          <div className="space-y-4">
            {mode === 'template' && templateStep === 'form' && (
              <div className="flex items-center justify-between">
                <button
                  type="button"
                  onClick={handleTemplateBack}
                  className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
                >
                  <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
                  {t('rbac.template.backToRoles')}
                </button>
                {selectedTemplate && (
                  <div className="flex items-center gap-1.5">
                    <Check className="h-3.5 w-3.5 text-success" aria-hidden="true" />
                    <span className="text-xs text-muted-foreground">
                      {t(selectedTemplate.labelKey)}
                    </span>
                  </div>
                )}
              </div>
            )}

            {/* basedOn select (clone-mode only — template-mode has it set implicitly) */}
            {mode === 'clone' && (
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
            )}

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
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          {showForm && (
            <Button
              onClick={submit}
              disabled={!name.trim() || !basedOn || createRole.isPending || setGrantsMutation.isPending}
            >
              {t('rbac.builder.cloneAction')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
