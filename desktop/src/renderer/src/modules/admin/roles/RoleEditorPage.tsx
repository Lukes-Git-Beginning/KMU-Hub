/**
 * RoleEditorPage (R-2) — the two-pane role editor at /admin/roles/:roleId.
 *
 * Left: module tree (category groups, search, deviation dots, visibility
 * state). Right: the focused module — level-1 visibility, level-2 base
 * actions with per-row scope, level-3 fine switches, plain-language summary.
 * Draft-first (weclapp lesson): changes stage locally, the footer bar shows
 * the diff and "Übernehmen" commits via PUT — guardrail checks run there
 * (self-lockout warning, escalation block on newly added grants).
 * Presets render read-only with a clone CTA.
 */
import { useEffect, useMemo, useState } from 'react'
import { Navigate, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  ArrowLeft,
  Check,
  ChevronDown,
  Copy,
  Eye,
  Lock,
  Minus,
  Pencil,
  Plus,
  Search,
  ShieldAlert,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
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
} from '@/components/ui/alert-dialog'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import type { CapabilityScope, Role, RoleGrants } from '@/api/rbac-types'
import { useRoleGrants, useRoles, useSetRoleGrants, useUpdateRole } from '@/api/hooks/useRbacRoles'
import { useCapabilitySet, useHasCapability } from '@/hooks/useCapability'
import { usePermissionsStore } from '@/stores/permissions'
import { MODULE_KEYS, moduleViewKey, type ModuleKey } from '@/config/capabilities'
import {
  CAPABILITY_CATALOG,
  MODULE_CATEGORY,
  type CapabilityDef,
  type ModuleCategory,
} from '@/config/capability-catalog'
import { capabilityLabel, moduleLabel, rbacErrorMessage, roleDisplayName } from '@/lib/rbac-format'
import { diffCount, diffGrants, diffKeySet, type GrantsDiff } from '@/lib/rbac-diff'
import { roleDescriptionKey } from '@/config/roles'
import CloneRoleDialog from './CloneRoleDialog'

const CATEGORY_ORDER: ModuleCategory[] = ['standard', 'industry', 'verwaltung']
const SCOPES: CapabilityScope[] = ['own', 'team', 'all']

export default function RoleEditorPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { roleId = '' } = useParams()

  const canAccess = useHasCapability('admin:role:read')
  const canEdit = useHasCapability('admin:role:edit')
  const canCreate = useHasCapability('admin:role:create')
  const { has: selfHas, ready: selfReady } = useCapabilitySet()
  const ownRoles = usePermissionsStore((s) => s.roles)

  const { data: roles = [] } = useRoles()
  const role = roles.find((r) => r.id === roleId)
  const { data: serverGrants } = useRoleGrants(roleId || null)
  const { data: baseGrants } = useRoleGrants(role?.basedOn ?? null)
  const setRoleGrants = useSetRoleGrants()
  const updateRole = useUpdateRole()

  const readonly = (role?.isSystem ?? true) || !canEdit

  const [draft, setDraft] = useState<RoleGrants | null>(null)
  const [selectedModule, setSelectedModule] = useState<ModuleKey>('work')
  const [search, setSearch] = useState('')
  const [cloneOpen, setCloneOpen] = useState(false)
  const [metaOpen, setMetaOpen] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)

  // Draft (re)seeds from the server state per role — deliberately NOT on every
  // refetch so staged edits survive background revalidation.
  useEffect(() => {
    setDraft(null)
  }, [roleId])
  useEffect(() => {
    if (serverGrants && draft === null) setDraft(serverGrants)
  }, [serverGrants, draft])

  const grants: RoleGrants = draft ?? serverGrants ?? {}

  // Diffs: staged changes (vs server) + deviations (vs based_on preset).
  const stagedDiff: GrantsDiff = useMemo(
    () => diffGrants(grants, serverGrants ?? {}),
    [grants, serverGrants],
  )
  const stagedCount = diffCount(stagedDiff)
  const deviationDiff: GrantsDiff | null = useMemo(
    () => (role?.basedOn && baseGrants ? diffGrants(grants, baseGrants) : null),
    [grants, baseGrants, role?.basedOn],
  )
  const deviationKeys = useMemo(
    () => (deviationDiff ? diffKeySet(deviationDiff) : new Set<string>()),
    [deviationDiff],
  )

  // Escalation guard: newly ADDED keys (vs server) the editing account itself
  // doesn't hold. Existing grants never block (removal must stay possible).
  const escalationKeys = useMemo(() => {
    if (!selfReady) return []
    return stagedDiff.added.filter((key) => !selfHas(key))
  }, [stagedDiff.added, selfHas, selfReady])

  // Self-lockout warning: the role is one of the account's own roles and the
  // draft drops builder access it previously granted.
  const selfLockoutRisk = useMemo(() => {
    if (!ownRoles.some((r) => r.id === roleId)) return false
    const critical = ['admin:module:view', 'admin:role:edit']
    return critical.some((key) => (serverGrants?.[key] && !grants[key]) ?? false)
  }, [ownRoles, roleId, serverGrants, grants])

  // Module tree: search matches module names AND capability labels.
  const visibleModules = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return [...MODULE_KEYS]
    return MODULE_KEYS.filter(
      (module) =>
        moduleLabel(t, module).toLowerCase().includes(q) ||
        CAPABILITY_CATALOG[module].some((c) => capabilityLabel(t, c.key).toLowerCase().includes(q)),
    )
  }, [search, t])

  if (!canAccess) {
    return <Navigate to="/" replace />
  }

  // ── Draft ops ─────────────────────────────────────────────────────────────
  const toggleKey = (key: string) => {
    if (readonly) return
    setDraft((d) => {
      const next = { ...(d ?? serverGrants ?? {}) }
      if (next[key]) delete next[key]
      else next[key] = { scope: 'all' }
      return next
    })
  }

  const setScope = (key: string, scope: CapabilityScope) => {
    if (readonly) return
    setDraft((d) => {
      const next = { ...(d ?? serverGrants ?? {}) }
      if (next[key]) next[key] = { scope }
      return next
    })
  }

  const setCategoryVisibility = (category: ModuleCategory, visible: boolean) => {
    if (readonly) return
    setDraft((d) => {
      const next = { ...(d ?? serverGrants ?? {}) }
      for (const module of MODULE_KEYS) {
        if (MODULE_CATEGORY[module] !== category) continue
        const key = moduleViewKey(module)
        if (visible) next[key] = next[key] ?? { scope: 'all' }
        else delete next[key]
      }
      return next
    })
  }

  const resetModuleToBase = (module: ModuleKey) => {
    if (readonly || !baseGrants) return
    setDraft((d) => {
      const next = { ...(d ?? serverGrants ?? {}) }
      const prefix = `${module}:`
      for (const key of Object.keys(next)) {
        if (key.startsWith(prefix)) delete next[key]
      }
      for (const [key, grant] of Object.entries(baseGrants)) {
        if (key.startsWith(prefix)) next[key] = grant
      }
      return next
    })
  }

  const commit = () => {
    setRoleGrants.mutate(
      { roleId, input: { grants } },
      {
        onSuccess: () => {
          setConfirmOpen(false)
          setDraft(null)
          toast.success(t('rbac.editor.saveDone'))
          // Live effect without re-login when the edited role is one of ours.
          if (ownRoles.some((r) => r.id === roleId)) {
            void usePermissionsStore.getState().refresh()
          }
        },
        onError: (err) => toast.error(rbacErrorMessage(t, err)),
      },
    )
  }

  const moduleDeviationCount = (module: ModuleKey): number => {
    if (!deviationDiff) return 0
    const prefix = `${module}:`
    let n = 0
    for (const key of deviationKeys) if (key.startsWith(prefix)) n += 1
    return n
  }

  if (!role) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
        {t('rbac.editor.notFound')}
      </div>
    )
  }

  const deviationTotal = deviationDiff ? diffCount(deviationDiff) : 0
  const basedOnRole = role.basedOn ? roles.find((r) => r.id === role.basedOn) : undefined

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Sticky header — back always visible */}
      <div className="shrink-0 border-b border-border bg-background px-6 py-4">
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => navigate('/admin/roles')}
            aria-label={t('common.back')}
            className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          </button>
          <span className="h-3 w-3 shrink-0 rounded-full" style={{ background: role.color }} aria-hidden="true" />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="truncate text-base font-semibold text-foreground">{roleDisplayName(t, role)}</h1>
              {role.isSystem ? (
                <span className="inline-flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
                  <Lock className="h-2.5 w-2.5" aria-hidden="true" />
                  {t('rbac.builder.systemBadge')}
                </span>
              ) : (
                basedOnRole && (
                  <span className="inline-flex items-center rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
                    {t('rbac.builder.basedOnBadge', { name: roleDisplayName(t, basedOnRole) })}
                  </span>
                )
              )}
              {deviationTotal > 0 && (
                <span className="inline-flex items-center rounded-full bg-info-light px-2 py-0.5 text-[10px] font-medium text-info">
                  {t('rbac.builder.deviationCount', { count: deviationTotal })}
                </span>
              )}
            </div>
            <p className="mt-0.5 truncate text-xs text-muted-foreground">
              {role.isSystem ? t(roleDescriptionKey(role.id)) : role.description}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                usePermissionsStore.getState().startPreview({
                  label: roleDisplayName(t, role),
                  capabilities: Object.fromEntries(
                    Object.entries(grants).map(([key, g]) => [key, { scope: g.scope, sources: [roleId] }]),
                  ),
                })
              }}
              title={t('rbac.preview.startHint')}
            >
              <Eye className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
              {t('rbac.preview.start')}
            </Button>
            {!role.isSystem && canEdit && (
              <Button variant="outline" size="sm" onClick={() => setMetaOpen(true)}>
                <Pencil className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                {t('rbac.editor.editMeta')}
              </Button>
            )}
            {canCreate && (
              <Button variant="outline" size="sm" onClick={() => setCloneOpen(true)}>
                <Copy className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                {t('rbac.builder.clone')}
              </Button>
            )}
          </div>
        </div>
        {role.isSystem && (
          <p className="mt-3 flex items-center gap-2 rounded-lg bg-secondary/60 px-3 py-2 text-xs text-muted-foreground">
            <Lock className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            {t('rbac.editor.presetReadonly')}
          </p>
        )}
      </div>

      {/* Two panes */}
      <div className="flex min-h-0 flex-1">
        {/* Left: module tree */}
        <div className="flex w-64 shrink-0 flex-col border-r border-border">
          <div className="shrink-0 p-3">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t('rbac.editor.searchPlaceholder')}
                className="h-8 pl-8 text-sm"
              />
            </div>
          </div>
          <nav className="min-h-0 flex-1 overflow-y-auto px-2 pb-3" aria-label={t('rbac.editor.moduleTree')}>
            {CATEGORY_ORDER.map((category) => {
              const modules = visibleModules.filter((m) => MODULE_CATEGORY[m] === category)
              if (modules.length === 0) return null
              return (
                <div key={category} className="mb-3">
                  <div className="flex items-center justify-between px-2 py-1">
                    <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                      {t(`rbac.editor.category.${category}`)}
                    </span>
                    {!readonly && (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <button
                            type="button"
                            aria-label={t('rbac.editor.bulkMenu', { category: t(`rbac.editor.category.${category}`) })}
                            className="rounded p-0.5 text-muted-foreground hover:bg-secondary hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                          >
                            <ChevronDown className="h-3.5 w-3.5" aria-hidden="true" />
                          </button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => setCategoryVisibility(category, true)}>
                            <Eye className="mr-2 h-3.5 w-3.5" aria-hidden="true" />
                            {t('rbac.editor.bulkShowAll')}
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => setCategoryVisibility(category, false)}>
                            <Minus className="mr-2 h-3.5 w-3.5" aria-hidden="true" />
                            {t('rbac.editor.bulkHideAll')}
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    )}
                  </div>
                  <ul>
                    {modules.map((module) => {
                      const visible = Boolean(grants[moduleViewKey(module)])
                      const devs = moduleDeviationCount(module)
                      const active = selectedModule === module
                      return (
                        <li key={module}>
                          <button
                            type="button"
                            onClick={() => setSelectedModule(module)}
                            aria-current={active ? 'true' : undefined}
                            className={`flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring ${
                              active
                                ? 'bg-primary-light font-medium text-primary'
                                : 'text-foreground hover:bg-secondary/60'
                            }`}
                          >
                            <span
                              className={`h-1.5 w-1.5 shrink-0 rounded-full ${visible ? 'bg-success' : 'bg-border'}`}
                              aria-hidden="true"
                            />
                            <span className="min-w-0 flex-1 truncate">{moduleLabel(t, module)}</span>
                            {devs > 0 && (
                              <span className="shrink-0 rounded-full bg-info-light px-1.5 text-[10px] font-medium tabular-nums text-info">
                                {devs}
                              </span>
                            )}
                          </button>
                        </li>
                      )
                    })}
                  </ul>
                </div>
              )
            })}
          </nav>
        </div>

        {/* Right: focused module */}
        <ModulePane
          module={selectedModule}
          grants={grants}
          readonly={readonly}
          deviationKeys={deviationKeys}
          hasBase={Boolean(role.basedOn && baseGrants)}
          onToggle={toggleKey}
          onScope={setScope}
          onResetModule={() => resetModuleToBase(selectedModule)}
        />
      </div>

      {/* Staged-changes footer */}
      {!readonly && stagedCount > 0 && (
        <div className="shrink-0 border-t border-border bg-card px-6 py-3">
          <div className="flex items-center justify-between gap-3">
            <Popover>
              <PopoverTrigger asChild>
                <button
                  type="button"
                  className="text-sm font-medium text-foreground underline-offset-4 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                >
                  {t('rbac.editor.stagedCount', { count: stagedCount })}{' '}
                  <span className="text-xs text-muted-foreground">
                    (+{stagedDiff.added.length + stagedDiff.scopeChanged.length} −{stagedDiff.removed.length})
                  </span>
                </button>
              </PopoverTrigger>
              <PopoverContent align="start" className="max-h-72 w-96 overflow-y-auto">
                <StagedChangesList diff={stagedDiff} />
              </PopoverContent>
            </Popover>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => setDraft(serverGrants ?? {})}>
                {t('rbac.editor.discard')}
              </Button>
              <Button size="sm" onClick={() => setConfirmOpen(true)} disabled={setRoleGrants.isPending}>
                <Check className="mr-1.5 h-4 w-4" aria-hidden="true" />
                {t('rbac.editor.apply')}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Apply confirm + guardrails */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('rbac.editor.applyTitle', { name: roleDisplayName(t, role) })}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('rbac.editor.applyBody', { count: stagedCount, members: role.memberCount })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {selfLockoutRisk && (
            <p className="flex items-start gap-2 rounded-lg border border-warning/40 bg-warning-light px-3 py-2 text-xs text-warning">
              <ShieldAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              {t('rbac.editor.selfLockoutWarning')}
            </p>
          )}
          {escalationKeys.length > 0 && (
            <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive">
              <p className="flex items-start gap-2 font-medium">
                <ShieldAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                {t('rbac.editor.escalationBlocked')}
              </p>
              <ul className="mt-1.5 list-inside list-disc space-y-0.5">
                {escalationKeys.slice(0, 4).map((key) => (
                  <li key={key}>{capabilityLabel(t, key)}</li>
                ))}
              </ul>
              {escalationKeys.length > 4 && (
                <p className="mt-1">{t('rbac.editor.escalationMore', { count: escalationKeys.length - 4 })}</p>
              )}
            </div>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction onClick={commit} disabled={escalationKeys.length > 0 || setRoleGrants.isPending}>
              {t('rbac.editor.applyAction')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Clone from here */}
      <CloneRoleDialog
        open={cloneOpen}
        onOpenChange={setCloneOpen}
        roles={roles}
        initialBase={role}
        onCreated={(created) => {
          setCloneOpen(false)
          navigate(`/admin/roles/${created.id}`)
        }}
      />

      {/* Meta edit (name/description/color) */}
      <RoleMetaDialog
        open={metaOpen}
        onOpenChange={setMetaOpen}
        role={role}
        onSave={(input) =>
          updateRole.mutate(
            { roleId, input },
            {
              onSuccess: () => {
                setMetaOpen(false)
                toast.success(t('rbac.editor.metaSaved'))
              },
              onError: (err) => toast.error(rbacErrorMessage(t, err)),
            },
          )
        }
        saving={updateRole.isPending}
      />
    </div>
  )
}

// ── Right pane ───────────────────────────────────────────────────────────────

function ModulePane({
  module,
  grants,
  readonly,
  deviationKeys,
  hasBase,
  onToggle,
  onScope,
  onResetModule,
}: {
  module: ModuleKey
  grants: RoleGrants
  readonly: boolean
  deviationKeys: Set<string>
  hasBase: boolean
  onToggle: (key: string) => void
  onScope: (key: string, scope: CapabilityScope) => void
  onResetModule: () => void
}) {
  const { t } = useTranslation()
  const defs = CAPABILITY_CATALOG[module]
  const visibleKey = moduleViewKey(module)
  const visible = Boolean(grants[visibleKey])

  // Group defs by subject (middle key segment), keep catalogue order.
  const subjects = useMemo(() => {
    const map = new Map<string, CapabilityDef[]>()
    for (const def of defs) {
      const subject = def.key.split(':')[1]
      const list = map.get(subject) ?? []
      list.push(def)
      map.set(subject, list)
    }
    return [...map.entries()]
  }, [defs])

  const moduleDeviations = useMemo(() => {
    const prefix = `${module}:`
    return [...deviationKeys].some((k) => k.startsWith(prefix))
  }, [deviationKeys, module])

  const activeLabels = defs.filter((d) => grants[d.key]).map((d) => capabilityLabel(t, d.key))
  const inactiveLabels = defs.filter((d) => !grants[d.key]).map((d) => capabilityLabel(t, d.key))

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto max-w-2xl space-y-4 p-6">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold text-foreground">{moduleLabel(t, module)}</h2>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {t('rbac.editor.moduleActive', { active: activeLabels.length + (visible ? 1 : 0), total: defs.length + 1 })}
            </p>
          </div>
          {!readonly && hasBase && moduleDeviations && (
            <Button variant="outline" size="sm" onClick={onResetModule}>
              {t('rbac.editor.resetModule')}
            </Button>
          )}
        </div>

        {/* Level 1 — visibility */}
        <CapabilityRow
          label={t('rbac.effective.moduleVisible')}
          description={t('rbac.editor.visibilityHint')}
          checked={visible}
          deviating={deviationKeys.has(visibleKey)}
          readonly={readonly}
          onToggle={() => onToggle(visibleKey)}
        />

        {!visible && defs.length > 0 && (
          <p className="rounded-lg bg-secondary/60 px-3 py-2 text-xs text-muted-foreground">
            {t('rbac.editor.hiddenModuleHint')}
          </p>
        )}

        {/* Level 2/3 per subject */}
        {defs.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border px-4 py-6 text-center">
            <p className="text-sm text-muted-foreground">{t('rbac.editor.noFineCapabilities')}</p>
          </div>
        ) : (
          <div className={visible ? '' : 'pointer-events-none opacity-50'} aria-disabled={!visible}>
            <div className="space-y-4">
              {subjects.map(([subject, subjectDefs]) => {
                const baseDefs = subjectDefs.filter((d) => !d.fine)
                const fineDefs = subjectDefs.filter((d) => d.fine)
                return (
                  <section key={subject} className="rounded-xl border border-border bg-card">
                    <header className="border-b border-border px-4 py-2.5">
                      <h3 className="text-sm font-medium text-foreground">
                        {t(`rbac.subject.${subject}`, { defaultValue: subject })}
                      </h3>
                    </header>
                    <div className="divide-y divide-border/60">
                      {baseDefs.map((def) => (
                        <CapabilityRow
                          key={def.key}
                          label={t(`rbac.action.${def.key.split(':')[2]}`, { defaultValue: def.key.split(':')[2] })}
                          checked={Boolean(grants[def.key])}
                          scope={def.scopeable ? grants[def.key]?.scope : undefined}
                          deviating={deviationKeys.has(def.key)}
                          readonly={readonly}
                          onToggle={() => onToggle(def.key)}
                          onScope={def.scopeable ? (s) => onScope(def.key, s) : undefined}
                          compact
                        />
                      ))}
                      {fineDefs.length > 0 && (
                        <div className="bg-secondary/30 px-4 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                          {t('rbac.editor.fineSwitches')}
                        </div>
                      )}
                      {fineDefs.map((def) => (
                        <CapabilityRow
                          key={def.key}
                          label={t(`rbac.action.${def.key.split(':')[2]}`, { defaultValue: def.key.split(':')[2] })}
                          checked={Boolean(grants[def.key])}
                          scope={def.scopeable ? grants[def.key]?.scope : undefined}
                          deviating={deviationKeys.has(def.key)}
                          readonly={readonly}
                          onToggle={() => onToggle(def.key)}
                          onScope={def.scopeable ? (s) => onScope(def.key, s) : undefined}
                          compact
                        />
                      ))}
                    </div>
                  </section>
                )
              })}
            </div>

            {/* Plain-language summary */}
            <div className="mt-4 rounded-xl border border-border bg-secondary/30 px-4 py-3 text-xs">
              <p className="text-foreground">
                <span className="font-medium">{t('rbac.editor.summaryCan')}</span>{' '}
                {activeLabels.length > 0
                  ? summarize(activeLabels, t('rbac.editor.summaryMore', { count: Math.max(0, activeLabels.length - 6) }))
                  : t('rbac.editor.summaryNothing')}
              </p>
              {inactiveLabels.length > 0 && (
                <p className="mt-1 text-muted-foreground">
                  <span className="font-medium">{t('rbac.editor.summaryCannot')}</span>{' '}
                  {summarize(inactiveLabels, t('rbac.editor.summaryMore', { count: Math.max(0, inactiveLabels.length - 6) }))}
                </p>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function summarize(labels: string[], moreLabel: string): string {
  if (labels.length <= 6) return labels.join(' · ')
  return `${labels.slice(0, 6).join(' · ')} ${moreLabel}`
}

function CapabilityRow({
  label,
  description,
  checked,
  scope,
  deviating,
  readonly,
  onToggle,
  onScope,
  compact = false,
}: {
  label: string
  description?: string
  checked: boolean
  scope?: CapabilityScope
  deviating: boolean
  readonly: boolean
  onToggle: () => void
  onScope?: (scope: CapabilityScope) => void
  compact?: boolean
}) {
  const { t } = useTranslation()
  return (
    <div className={`flex items-center gap-3 ${compact ? 'px-4 py-2' : 'rounded-xl border border-border bg-card px-4 py-3'}`}>
      <div className="min-w-0 flex-1">
        <span className="flex items-center gap-1.5 text-sm text-foreground">
          {label}
          {deviating && (
            <span
              className="inline-flex h-1.5 w-1.5 shrink-0 rounded-full bg-info"
              title={t('rbac.editor.deviatingHint')}
              aria-label={t('rbac.editor.deviatingHint')}
            />
          )}
        </span>
        {description && <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>}
      </div>
      {checked && onScope && scope && (
        <Select value={scope} onValueChange={(v) => onScope(v as CapabilityScope)} disabled={readonly}>
          <SelectTrigger className="h-7 w-28 text-xs" aria-label={t('rbac.editor.scopeLabel')}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {SCOPES.map((s) => (
              <SelectItem key={s} value={s} className="text-xs">
                {t(`rbac.scope.${s}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
      <Switch checked={checked} onCheckedChange={onToggle} disabled={readonly} aria-label={label} />
    </div>
  )
}

// ── Staged changes list (footer popover) ─────────────────────────────────────

function StagedChangesList({ diff }: { diff: GrantsDiff }) {
  const { t } = useTranslation()
  return (
    <ul className="space-y-1 text-xs">
      {diff.added.map((key) => (
        <li key={`a-${key}`} className="flex items-center gap-2 text-success">
          <Plus className="h-3 w-3 shrink-0" aria-hidden="true" />
          <span className="min-w-0 flex-1 truncate">
            {moduleLabelForKey(t, key)} · {capabilityLabel(t, key)}
          </span>
        </li>
      ))}
      {diff.removed.map((key) => (
        <li key={`r-${key}`} className="flex items-center gap-2 text-destructive">
          <Minus className="h-3 w-3 shrink-0" aria-hidden="true" />
          <span className="min-w-0 flex-1 truncate">
            {moduleLabelForKey(t, key)} · {capabilityLabel(t, key)}
          </span>
        </li>
      ))}
      {diff.scopeChanged.map(({ key, from, to }) => (
        <li key={`s-${key}`} className="flex items-center gap-2 text-info">
          <Pencil className="h-3 w-3 shrink-0" aria-hidden="true" />
          <span className="min-w-0 flex-1 truncate">
            {moduleLabelForKey(t, key)} · {capabilityLabel(t, key)}:{' '}
            {t(`rbac.scope.${from}`)} → {t(`rbac.scope.${to}`)}
          </span>
        </li>
      ))}
    </ul>
  )
}

function moduleLabelForKey(t: ReturnType<typeof useTranslation>['t'], key: string): string {
  return moduleLabel(t, key.split(':')[0])
}

// ── Meta dialog ──────────────────────────────────────────────────────────────

const META_COLORS = [
  'hsl(38 92% 50%)',
  'hsl(160 84% 39%)',
  'hsl(200 98% 39%)',
  'hsl(245 58% 61%)',
  'hsl(330 81% 60%)',
  'hsl(15 86% 50%)',
  'hsl(84 59% 40%)',
  'hsl(190 77% 40%)',
]

function RoleMetaDialog({
  open,
  onOpenChange,
  role,
  onSave,
  saving,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  role: Role
  onSave: (input: { name: string; description: string; color: string }) => void
  saving: boolean
}) {
  const { t } = useTranslation()
  const [name, setName] = useState(role.name)
  const [description, setDescription] = useState(role.description)
  const [color, setColor] = useState(role.color)

  useEffect(() => {
    if (open) {
      setName(role.name)
      setDescription(role.description)
      setColor(role.color)
    }
  }, [open, role])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('rbac.editor.metaTitle')}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="meta-name">{t('rbac.builder.cloneName')}</Label>
            <Input id="meta-name" value={name} onChange={(e) => setName(e.target.value)} maxLength={60} />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="meta-description">{t('rbac.builder.cloneDescription')}</Label>
            <Textarea
              id="meta-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
              maxLength={160}
            />
          </div>
          <div className="space-y-1.5">
            <Label>{t('rbac.builder.cloneColor')}</Label>
            <div className="flex flex-wrap gap-2" role="radiogroup" aria-label={t('rbac.builder.cloneColor')}>
              {META_COLORS.map((c) => (
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
          <Button onClick={() => onSave({ name: name.trim(), description: description.trim(), color })} disabled={!name.trim() || saving}>
            {t('common.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
