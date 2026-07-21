/**
 * UserOverrideEditorPage (R-6) — per-user permission overrides at
 * /admin/users/:userId/overrides. Same two-pane shape as the role editor, but
 * every row is TRI-STATE: inherited (grayed, follows the roles live), allow
 * (explicitly granted on top), deny (explicitly revoked). While no override is
 * set the whole tree is read-only and shows the role-union baseline — exactly
 * the preset-role optics Darien asked for. "Benutzerdefiniert" unlocks editing.
 *
 * Guardrails at commit (mirrors the role editor): escalation block (grant a key
 * the editing admin doesn't hold), self-lockout (own account), last-admin (deny
 * admin:* on the last full-access holder) — all enforced before PUT.
 */
import { useEffect, useMemo, useState } from 'react'
import { Navigate, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  ArrowLeft,
  Check,
  Eye,
  Minus,
  RotateCcw,
  Search,
  ShieldAlert,
  SlidersHorizontal,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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
import type {
  CapabilityGrant,
  CapabilityOverride,
  CapabilityScope,
  UserOverrides,
} from '@/api/rbac-types'
import {
  useUserEffectivePermissions,
  useUserOverrides,
  useUserRoleBaseline,
  useSetUserOverrides,
} from '@/api/hooks/useRbacRoles'
import { useAdminUsers } from '@/api/hooks/useAdminUsers'
import { useCapabilitySet, useHasCapability } from '@/hooks/useCapability'
import { useAuthStore } from '@/stores/auth'
import { usePermissionsStore } from '@/stores/permissions'
import { MODULE_KEYS, moduleViewKey, type ModuleKey } from '@/config/capabilities'
import {
  CAPABILITY_CATALOG,
  MODULE_CATEGORY,
  type CapabilityDef,
  type ModuleCategory,
} from '@/config/capability-catalog'
import { capabilityLabel, moduleLabel } from '@/lib/rbac-format'

const CATEGORY_ORDER: ModuleCategory[] = ['standard', 'industry', 'verwaltung']
const SCOPES: CapabilityScope[] = ['own', 'team', 'all']

/** Effective row state for one capability key given baseline + override. */
type RowState =
  | { kind: 'inherited-on'; scope: CapabilityScope }
  | { kind: 'inherited-off' }
  | { kind: 'allow'; scope: CapabilityScope }
  | { kind: 'deny' }

function rowState(
  key: string,
  baseline: Record<string, CapabilityGrant>,
  overrides: UserOverrides,
): RowState {
  const ov = overrides[key]
  if (ov) {
    return ov.mode === 'deny' ? { kind: 'deny' } : { kind: 'allow', scope: ov.scope }
  }
  const base = baseline[key]
  return base ? { kind: 'inherited-on', scope: base.scope } : { kind: 'inherited-off' }
}

/** Whether the key is effectively granted given baseline + override. */
function isEffective(state: RowState): boolean {
  return state.kind === 'inherited-on' || state.kind === 'allow'
}

export default function UserOverrideEditorPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { userId = '' } = useParams()

  const canManage = useHasCapability('admin:user_override:manage')
  const { has: selfHas, ready: selfReady } = useCapabilitySet()
  const authUserId = useAuthStore((s) => s.user?.id)

  const { data: users = [] } = useAdminUsers()
  const user = users.find((u) => u.id === userId)
  const { data: baselinePerm } = useUserRoleBaseline(userId || null)
  const { data: serverOverrides } = useUserOverrides(userId || null)
  const { data: effectivePerm } = useUserEffectivePermissions(userId || null)
  const setOverrides = useSetUserOverrides()

  const baseline = baselinePerm?.capabilities ?? {}
  const roleNames = (effectivePerm?.roles ?? baselinePerm?.roles ?? []).map((r) => r.name)

  const [draft, setDraft] = useState<UserOverrides | null>(null)
  const [customMode, setCustomMode] = useState(false)
  const [selectedModule, setSelectedModule] = useState<ModuleKey>('work')
  const [search, setSearch] = useState('')
  const [confirmOpen, setConfirmOpen] = useState(false)

  // Draft (re)seeds from the server overrides per user.
  useEffect(() => {
    setDraft(null)
    setCustomMode(false)
  }, [userId])
  useEffect(() => {
    if (serverOverrides && draft === null) setDraft(serverOverrides)
  }, [serverOverrides, draft])

  const overrides: UserOverrides = draft ?? serverOverrides ?? {}
  const editable = canManage && customMode
  const isSelf = userId === authUserId

  const overrideCount = Object.keys(overrides).length
  const serverCount = Object.keys(serverOverrides ?? {}).length
  const dirty = useMemo(
    () => JSON.stringify(overrides) !== JSON.stringify(serverOverrides ?? {}),
    [overrides, serverOverrides],
  )

  // Escalation guard: allow-overrides on keys the editing admin doesn't hold.
  const escalationKeys = useMemo(() => {
    if (!selfReady) return []
    return Object.entries(overrides)
      .filter(([key, ov]) => ov.mode === 'allow' && !selfHas(key))
      .map(([key]) => key)
  }, [overrides, selfHas, selfReady])

  // Last-admin guard: a deny that would strip the module/role admin keys off an
  // account whose effective set otherwise has them (self or last full-access).
  const adminLockKeys = useMemo(() => {
    const critical = ['admin:module:view', 'admin:role:edit']
    return Object.entries(overrides)
      .filter(([key, ov]) => ov.mode === 'deny' && critical.includes(key) && baseline[key])
      .map(([key]) => key)
  }, [overrides, baseline])

  const visibleModules = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return [...MODULE_KEYS]
    return MODULE_KEYS.filter(
      (module) =>
        moduleLabel(t, module).toLowerCase().includes(q) ||
        CAPABILITY_CATALOG[module].some((c) => capabilityLabel(t, c.key).toLowerCase().includes(q)),
    )
  }, [search, t])

  if (!canManage) return <Navigate to="/" replace />

  // ── Draft ops ─────────────────────────────────────────────────────────────
  const setOverride = (key: string, next: CapabilityOverride | null) => {
    if (!editable) return
    setDraft((d) => {
      const map = { ...(d ?? serverOverrides ?? {}) }
      if (next === null) delete map[key]
      else map[key] = next
      return map
    })
  }

  /** Cycle a row: inherited → (opposite of inherited) → the other override → inherited. */
  const cycleRow = (key: string) => {
    if (!editable) return
    const state = rowState(key, baseline, overrides)
    const baseOn = Boolean(baseline[key])
    switch (state.kind) {
      case 'inherited-on':
        // roles grant it → first tap denies it
        setOverride(key, { mode: 'deny', scope: baseline[key]?.scope ?? 'all' })
        break
      case 'inherited-off':
        // roles withhold it → first tap allows it
        setOverride(key, { mode: 'allow', scope: 'all' })
        break
      case 'allow':
        setOverride(key, baseOn ? { mode: 'deny', scope: baseline[key]?.scope ?? 'all' } : null)
        break
      case 'deny':
        setOverride(key, baseOn ? null : { mode: 'allow', scope: 'all' })
        break
    }
  }

  const setAllowScope = (key: string, scope: CapabilityScope) => {
    if (!editable) return
    setOverride(key, { mode: 'allow', scope })
  }

  const resetKey = (key: string) => setOverride(key, null)

  const resetAll = () => {
    if (!editable) return
    setDraft({})
  }

  const commit = () => {
    setOverrides.mutate(
      { userId, overrides },
      {
        onSuccess: () => {
          setConfirmOpen(false)
          setDraft(null)
          setCustomMode(false)
          toast.success(t('rbac.override.saveDone'))
          if (isSelf) void usePermissionsStore.getState().refresh()
        },
        onError: () => toast.error(t('rbac.override.saveError')),
      },
    )
  }

  const moduleOverrideCount = (module: ModuleKey): number => {
    const prefix = `${module}:`
    let n = 0
    for (const key of Object.keys(overrides)) if (key.startsWith(prefix)) n += 1
    return n
  }

  if (!user && users.length > 0) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
        {t('rbac.override.userNotFound')}
      </div>
    )
  }

  const fullName = user ? `${user.firstName} ${user.lastName}` : userId
  const targetIsAdmin = (effectivePerm?.roles ?? []).some((r) => r.id === 'admin')

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Sticky header — back always visible */}
      <div className="shrink-0 border-b border-border bg-background px-6 py-4">
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => navigate('/admin/users')}
            aria-label={t('common.back')}
            className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          </button>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="truncate text-base font-semibold text-foreground">
                {t('rbac.override.title', { name: fullName })}
              </h1>
              {overrideCount > 0 && (
                <span className="inline-flex items-center gap-1 rounded-full bg-info-light px-2 py-0.5 text-[10px] font-medium text-info">
                  <SlidersHorizontal className="h-2.5 w-2.5" aria-hidden="true" />
                  {t('rbac.override.count', { count: overrideCount })}
                </span>
              )}
            </div>
            <p className="mt-0.5 truncate text-xs text-muted-foreground">
              {t('rbac.override.rolesLine', { roles: roleNames.join(', ') || '—' })}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                usePermissionsStore.getState().startPreview({
                  label: t('rbac.override.previewLabel', { name: fullName }),
                  capabilities: effectivePerm?.capabilities ?? {},
                })
              }}
              title={t('rbac.override.previewHint')}
            >
              <Eye className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
              {t('rbac.override.preview')}
            </Button>
            {!customMode ? (
              <Button size="sm" onClick={() => setCustomMode(true)} disabled={isSelf}>
                <SlidersHorizontal className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                {t('rbac.override.customize')}
              </Button>
            ) : (
              overrideCount > 0 && (
                <Button variant="outline" size="sm" onClick={resetAll}>
                  <RotateCcw className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                  {t('rbac.override.resetAll')}
                </Button>
              )
            )}
          </div>
        </div>
        {!customMode ? (
          <p className="mt-3 flex items-center gap-2 rounded-lg bg-secondary/60 px-3 py-2 text-xs text-muted-foreground">
            <SlidersHorizontal className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            {isSelf ? t('rbac.override.selfLocked') : t('rbac.override.readonlyHint')}
          </p>
        ) : (
          <p className="mt-3 flex items-center gap-2 rounded-lg border border-info/40 bg-info-light px-3 py-2 text-xs text-info">
            <SlidersHorizontal className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            {t('rbac.override.customHint')}
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
                  <div className="px-2 py-1">
                    <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                      {t(`rbac.editor.category.${category}`)}
                    </span>
                  </div>
                  <ul>
                    {modules.map((module) => {
                      const state = rowState(moduleViewKey(module), baseline, overrides)
                      const visible = isEffective(state)
                      const ovs = moduleOverrideCount(module)
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
                            {ovs > 0 && (
                              <span className="shrink-0 rounded-full bg-info-light px-1.5 text-[10px] font-medium tabular-nums text-info">
                                {ovs}
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
        <ModuleOverridePane
          module={selectedModule}
          baseline={baseline}
          overrides={overrides}
          editable={editable}
          onCycle={cycleRow}
          onScope={setAllowScope}
          onReset={resetKey}
        />
      </div>

      {/* Staged footer */}
      {customMode && dirty && (
        <div className="shrink-0 border-t border-border bg-card px-6 py-3">
          <div className="flex items-center justify-between gap-3">
            <Popover>
              <PopoverTrigger asChild>
                <button
                  type="button"
                  className="text-sm font-medium text-foreground underline-offset-4 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                >
                  {t('rbac.override.stagedCount', { count: overrideCount })}
                </button>
              </PopoverTrigger>
              <PopoverContent align="start" className="max-h-72 w-96 overflow-y-auto">
                <OverrideList overrides={overrides} />
              </PopoverContent>
            </Popover>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => setDraft(serverOverrides ?? {})}>
                {t('rbac.editor.discard')}
              </Button>
              <Button size="sm" onClick={() => setConfirmOpen(true)} disabled={setOverrides.isPending}>
                <Check className="mr-1.5 h-4 w-4" aria-hidden="true" />
                {t('rbac.override.apply')}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Apply confirm + guardrails */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('rbac.override.applyTitle', { name: fullName })}</AlertDialogTitle>
            <AlertDialogDescription>
              {overrideCount === 0 && serverCount > 0
                ? t('rbac.override.applyClearBody')
                : t('rbac.override.applyBody', { count: overrideCount })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {escalationKeys.length > 0 && (
            <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive">
              <p className="flex items-start gap-2 font-medium">
                <ShieldAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                {t('rbac.override.escalationBlocked')}
              </p>
              <ul className="mt-1.5 list-inside list-disc space-y-0.5">
                {escalationKeys.slice(0, 4).map((key) => (
                  <li key={key}>{capabilityLabel(t, key)}</li>
                ))}
              </ul>
            </div>
          )}
          {adminLockKeys.length > 0 && targetIsAdmin && (
            <p className="flex items-start gap-2 rounded-lg border border-warning/40 bg-warning-light px-3 py-2 text-xs text-warning">
              <ShieldAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              {t('rbac.override.adminLockWarning')}
            </p>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={commit}
              disabled={escalationKeys.length > 0 || setOverrides.isPending}
            >
              {t('rbac.override.applyAction')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

// ── Right pane ───────────────────────────────────────────────────────────────

function ModuleOverridePane({
  module,
  baseline,
  overrides,
  editable,
  onCycle,
  onScope,
  onReset,
}: {
  module: ModuleKey
  baseline: Record<string, CapabilityGrant>
  overrides: UserOverrides
  editable: boolean
  onCycle: (key: string) => void
  onScope: (key: string, scope: CapabilityScope) => void
  onReset: (key: string) => void
}) {
  const { t } = useTranslation()
  const defs = CAPABILITY_CATALOG[module]
  const visibleKey = moduleViewKey(module)

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

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto max-w-2xl space-y-4 p-6">
        <div>
          <h2 className="text-base font-semibold text-foreground">{moduleLabel(t, module)}</h2>
          <p className="mt-0.5 text-xs text-muted-foreground">{t('rbac.override.paneHint')}</p>
        </div>

        {/* Level 1 — visibility */}
        <OverrideRow
          keyName={visibleKey}
          label={t('rbac.effective.moduleVisible')}
          scopeable={false}
          baseline={baseline}
          overrides={overrides}
          editable={editable}
          onCycle={onCycle}
          onScope={onScope}
          onReset={onReset}
        />

        {defs.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border px-4 py-6 text-center">
            <p className="text-sm text-muted-foreground">{t('rbac.editor.noFineCapabilities')}</p>
          </div>
        ) : (
          <div className="space-y-4">
            {subjects.map(([subject, subjectDefs]) => (
              <section key={subject} className="rounded-xl border border-border bg-card">
                <header className="border-b border-border px-4 py-2.5">
                  <h3 className="text-sm font-medium text-foreground">
                    {t(`rbac.subject.${module}_${subject}`, {
                      defaultValue: t(`rbac.subject.${subject}`, { defaultValue: subject }),
                    })}
                  </h3>
                </header>
                <div className="divide-y divide-border/60">
                  {subjectDefs.map((def) => (
                    <OverrideRow
                      key={def.key}
                      keyName={def.key}
                      label={t(`rbac.action.${def.key.split(':')[2]}`, {
                        defaultValue: def.key.split(':')[2],
                      })}
                      scopeable={def.scopeable}
                      baseline={baseline}
                      overrides={overrides}
                      editable={editable}
                      onCycle={onCycle}
                      onScope={onScope}
                      onReset={onReset}
                      compact
                    />
                  ))}
                </div>
              </section>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

// ── One tri-state row ────────────────────────────────────────────────────────

const STATE_PILL: Record<RowState['kind'], { labelKey: string; cls: string }> = {
  'inherited-on': { labelKey: 'rbac.override.state.inheritedOn', cls: 'bg-secondary text-muted-foreground' },
  'inherited-off': { labelKey: 'rbac.override.state.inheritedOff', cls: 'bg-secondary text-muted-foreground' },
  allow: { labelKey: 'rbac.override.state.allow', cls: 'bg-success-light text-success' },
  deny: { labelKey: 'rbac.override.state.deny', cls: 'bg-error-light text-error' },
}

function OverrideRow({
  keyName,
  label,
  scopeable,
  baseline,
  overrides,
  editable,
  onCycle,
  onScope,
  onReset,
  compact = false,
}: {
  keyName: string
  label: string
  scopeable: boolean
  baseline: Record<string, CapabilityGrant>
  overrides: UserOverrides
  editable: boolean
  onCycle: (key: string) => void
  onScope: (key: string, scope: CapabilityScope) => void
  onReset: (key: string) => void
  compact?: boolean
}) {
  const { t } = useTranslation()
  const state = rowState(keyName, baseline, overrides)
  const isOverridden = state.kind === 'allow' || state.kind === 'deny'
  const pill = STATE_PILL[state.kind]
  const effective = isEffective(state)

  return (
    <div className={`flex items-center gap-3 ${compact ? 'px-4 py-2' : 'rounded-xl border border-border bg-card px-4 py-3'}`}>
      <div className="min-w-0 flex-1">
        <span className={`text-sm ${state.kind === 'deny' ? 'text-muted-foreground line-through' : 'text-foreground'}`}>
          {label}
        </span>
      </div>
      <span className={`inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${pill.cls}`}>
        {t(pill.labelKey)}
      </span>
      {/* Scope select only for an allow-override on a scopeable key */}
      {editable && scopeable && state.kind === 'allow' && (
        <Select value={state.scope} onValueChange={(v) => onScope(keyName, v as CapabilityScope)}>
          <SelectTrigger className="h-7 w-24 text-xs" aria-label={t('rbac.editor.scopeLabel')}>
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
      {editable && isOverridden && (
        <button
          type="button"
          onClick={() => onReset(keyName)}
          aria-label={t('rbac.override.resetKey')}
          title={t('rbac.override.resetKey')}
          className="shrink-0 rounded p-1 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
        >
          <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      )}
      <button
        type="button"
        onClick={() => onCycle(keyName)}
        disabled={!editable}
        aria-label={t('rbac.override.cycleLabel', { label })}
        className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-full border transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring disabled:opacity-40 ${
          effective ? 'border-success bg-success text-success-foreground' : 'border-border bg-secondary text-muted-foreground'
        }`}
      >
        {effective ? <Check className="h-3.5 w-3.5" aria-hidden="true" /> : <Minus className="h-3.5 w-3.5" aria-hidden="true" />}
      </button>
    </div>
  )
}

// ── Staged override list (footer popover) ────────────────────────────────────

function OverrideList({ overrides }: { overrides: UserOverrides }) {
  const { t } = useTranslation()
  const entries = Object.entries(overrides)
  if (entries.length === 0) {
    return <p className="px-1 py-2 text-xs text-muted-foreground">{t('rbac.override.noneStaged')}</p>
  }
  return (
    <ul className="space-y-1 text-xs">
      {entries.map(([key, ov]) => (
        <li
          key={key}
          className={`flex items-center gap-2 ${ov.mode === 'deny' ? 'text-error' : 'text-success'}`}
        >
          {ov.mode === 'deny' ? (
            <Minus className="h-3 w-3 shrink-0" aria-hidden="true" />
          ) : (
            <Check className="h-3 w-3 shrink-0" aria-hidden="true" />
          )}
          <span className="min-w-0 flex-1 truncate">
            {moduleLabel(t, key.split(':')[0])} · {capabilityLabel(t, key)}
            {ov.mode === 'allow' && (
              <span className="text-muted-foreground"> ({t(`rbac.scope.${ov.scope}`)})</span>
            )}
          </span>
        </li>
      ))}
    </ul>
  )
}
