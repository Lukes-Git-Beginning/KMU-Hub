import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  ArrowRight,
  ChevronDown,
  Minimize2,
  Maximize2,
  Settings2,
  Lock,
  Info,
} from 'lucide-react'
import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/cn'
import { isModuleAllowedForProfile, getProfileById } from '@/config/business-profiles'
import { useProfileStore } from '@/stores/profile'
import { navItems, moduleHsl, moduleHslBg } from '@/components/layout/sidebar/nav-items'

type ViewState = 'minimized' | 'half' | 'full'

interface ModuleConfig {
  id: string
  nameKey: string
  descriptionKey: string
  path: string
  stats: { labelKey: string; value: number }
  isActive: boolean
  badgeKey?: string
}

/**
 * Module definitions — colors + icons come from nav-items.ts.
 * Only module-specific metadata (description, stats, badge) is here.
 * Labels use i18n keys resolved at render time.
 */
const ALL_MODULES: ModuleConfig[] = [
  { id: 'projects', nameKey: 'dashboard.modules.projects.name', descriptionKey: 'dashboard.modules.projects.description', path: '/work/projects', stats: { labelKey: 'dashboard.modules.projects.stat', value: 24 }, isActive: true },
  { id: 'tasks', nameKey: 'dashboard.modules.tasks.name', descriptionKey: 'dashboard.modules.tasks.description', path: '/work/my-tasks', stats: { labelKey: 'dashboard.modules.tasks.stat', value: 142 }, isActive: true },
  { id: 'documents', nameKey: 'dashboard.modules.documents.name', descriptionKey: 'dashboard.modules.documents.description', path: '/dokumente', stats: { labelKey: 'dashboard.modules.documents.stat', value: 89 }, isActive: true },
  { id: 'finance', nameKey: 'dashboard.modules.finance.name', descriptionKey: 'dashboard.modules.finance.description', path: '/finanzen', stats: { labelKey: 'dashboard.modules.finance.stat', value: 18 }, isActive: true, badgeKey: 'dashboard.modules.badgeNew' },
  { id: 'chat', nameKey: 'dashboard.modules.chat.name', descriptionKey: 'dashboard.modules.chat.description', path: '/chat', stats: { labelKey: 'dashboard.modules.chat.stat', value: 37 }, isActive: true },
  { id: 'crm', nameKey: 'dashboard.modules.crm.name', descriptionKey: 'dashboard.modules.crm.description', path: '/crm', stats: { labelKey: 'dashboard.modules.crm.stat', value: 12 }, isActive: true },
  { id: 'inventar', nameKey: 'dashboard.modules.inventar.name', descriptionKey: 'dashboard.modules.inventar.description', path: '/inventar', stats: { labelKey: 'dashboard.modules.inventar.stat', value: 245 }, isActive: true },
  { id: 'schichten', nameKey: 'dashboard.modules.schichten.name', descriptionKey: 'dashboard.modules.schichten.description', path: '/schichten', stats: { labelKey: 'dashboard.modules.schichten.stat', value: 32 }, isActive: true },
  { id: 'einkauf', nameKey: 'dashboard.modules.einkauf.name', descriptionKey: 'dashboard.modules.einkauf.description', path: '/einkauf', stats: { labelKey: 'dashboard.modules.einkauf.stat', value: 7 }, isActive: true },
  { id: 'helpdesk', nameKey: 'dashboard.modules.helpdesk.name', descriptionKey: 'dashboard.modules.helpdesk.description', path: '/helpdesk', stats: { labelKey: 'dashboard.modules.helpdesk.stat', value: 15 }, isActive: true },
  { id: 'fuhrpark', nameKey: 'dashboard.modules.fuhrpark.name', descriptionKey: 'dashboard.modules.fuhrpark.description', path: '/fuhrpark', stats: { labelKey: 'dashboard.modules.fuhrpark.stat', value: 6 }, isActive: true },
  { id: 'produktion', nameKey: 'dashboard.modules.produktion.name', descriptionKey: 'dashboard.modules.produktion.description', path: '/produktion', stats: { labelKey: 'dashboard.modules.produktion.stat', value: 8 }, isActive: true },
  { id: 'berichte', nameKey: 'dashboard.modules.berichte.name', descriptionKey: 'dashboard.modules.berichte.description', path: '/berichte', stats: { labelKey: 'dashboard.modules.berichte.stat', value: 3 }, isActive: true },
  { id: 'zeiterfassung', nameKey: 'dashboard.modules.zeiterfassung.name', descriptionKey: 'dashboard.modules.zeiterfassung.description', path: '/zeiterfassung', stats: { labelKey: 'dashboard.modules.zeiterfassung.stat', value: 6 }, isActive: true },
  { id: 'vertraege', nameKey: 'dashboard.modules.vertraege.name', descriptionKey: 'dashboard.modules.vertraege.description', path: '/vertraege', stats: { labelKey: 'dashboard.modules.vertraege.stat', value: 9 }, isActive: true },
  { id: 'formulare', nameKey: 'dashboard.modules.formulare.name', descriptionKey: 'dashboard.modules.formulare.description', path: '/formulare', stats: { labelKey: 'dashboard.modules.formulare.stat', value: 4 }, isActive: true },
  { id: 'vermietung', nameKey: 'dashboard.modules.vermietung.name', descriptionKey: 'dashboard.modules.vermietung.description', path: '/vermietung', stats: { labelKey: 'dashboard.modules.vermietung.stat', value: 5 }, isActive: true },
  { id: 'rapporte', nameKey: 'dashboard.modules.rapporte.name', descriptionKey: 'dashboard.modules.rapporte.description', path: '/rapporte', stats: { labelKey: 'dashboard.modules.rapporte.stat', value: 8 }, isActive: true },
]

/** Look up the icon from nav-items by module ID */
function getModuleIcon(id: string) {
  return navItems.find((i) => i.id === id)?.icon
}

export function ModulesGrid() {
  const { t } = useTranslation()
  const [viewState, setViewState] = useState<ViewState>('full')
  const [manageOpen, setManageOpen] = useState(false)
  const businessProfileId = useProfileStore((s) => s.businessProfileId)
  const devShowAll = useProfileStore((s) => s.devShowAllModules)
  const enabledOptionals = useProfileStore((s) => s.enabledOptionalModules)
  const enableModule = useProfileStore((s) => s.enableModule)
  const disableModule = useProfileStore((s) => s.disableModule)

  const filteredModules = ALL_MODULES.filter((mod) => {
    if (devShowAll) return true
    return isModuleAllowedForProfile(mod.id, businessProfileId, enabledOptionals)
  })

  const display =
    viewState === 'minimized'
      ? []
      : viewState === 'half'
        ? filteredModules.slice(0, 3)
        : filteredModules

  return (
    <div className="mb-8">
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">{t('dashboard.modules.title')}</h2>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1">
            {(
              [
                { value: 'minimized', icon: Minimize2, titleKey: 'dashboard.view.minimized' },
                { value: 'half', icon: ChevronDown, titleKey: 'dashboard.view.half' },
                { value: 'full', icon: Maximize2, titleKey: 'dashboard.view.full' },
              ] as const
            ).map((item) => {
              const Icon = item.icon
              return (
                <button
                  key={item.value}
                  onClick={() => setViewState(item.value)}
                  title={t(item.titleKey)}
                  aria-label={t(item.titleKey)}
                  className={cn(
                    'rounded p-1.5 transition-colors',
                    viewState === item.value
                      ? 'bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:bg-accent',
                  )}
                >
                  <Icon className="h-3.5 w-3.5" />
                </button>
              )
            })}
          </div>

          {viewState !== 'minimized' && (
            <button
              onClick={() => setManageOpen(true)}
              className="flex items-center gap-1.5 text-sm text-primary transition-colors hover:text-primary/80"
            >
              <Settings2 className="h-3.5 w-3.5" />
              {t('dashboard.modules.manage')}
            </button>
          )}
        </div>
      </div>

      {viewState !== 'minimized' && (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {display.map((mod, i) => {
            const Icon = getModuleIcon(mod.id)
            if (!Icon) return null
            const color = moduleHsl(mod.id)
            const bgColor = moduleHslBg(mod.id)

            return (
              <Link
                key={mod.id}
                to={mod.path}
                className={cn(
                  'group relative rounded-xl border border-border bg-card p-6 transition-all hover:shadow-lg hover:-translate-y-0.5',
                  !mod.isActive && 'opacity-75',
                  `animate-scale-in stagger-${Math.min(i + 1, 8)}`,
                )}
              >
                {mod.badgeKey && (
                  <Badge className="absolute right-4 top-4 badge-accent bg-primary/10 text-primary">
                    {t(mod.badgeKey)}
                  </Badge>
                )}

                <div
                  className="mb-4 flex h-12 w-12 items-center justify-center rounded-xl"
                  style={{ backgroundColor: bgColor }}
                >
                  <Icon className="h-6 w-6" style={{ color }} />
                </div>

                <h3 className="mb-2 font-medium text-foreground transition-colors group-hover:text-primary">
                  {t(mod.nameKey)}
                </h3>
                <p className="mb-4 text-sm text-muted-foreground">
                  {t(mod.descriptionKey)}
                </p>

                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-xs text-muted-foreground">
                      {t(mod.stats.labelKey)}
                    </p>
                    <p className="text-2xl text-foreground stat-accent">{mod.stats.value}</p>
                  </div>
                  <ArrowRight className="h-5 w-5 text-muted-foreground transition-all group-hover:translate-x-1 group-hover:text-primary" />
                </div>

                {!mod.isActive && (
                  <div className="absolute inset-0 flex items-center justify-center rounded-xl bg-card/50">
                    <span className="rounded-full bg-foreground px-3 py-1 text-xs text-background">
                      {t('dashboard.modules.notActivated')}
                    </span>
                  </div>
                )}
              </Link>
            )
          })}
        </div>
      )}
      <ManageModulesDialog
        open={manageOpen}
        onOpenChange={setManageOpen}
        businessProfileId={businessProfileId}
        enabledOptionals={enabledOptionals}
        onEnable={enableModule}
        onDisable={disableModule}
      />
    </div>
  )
}

function ManageModulesDialog({
  open,
  onOpenChange,
  businessProfileId,
  enabledOptionals,
  onEnable,
  onDisable,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  businessProfileId: string | null
  enabledOptionals: string[]
  onEnable: (id: string) => void
  onDisable: (id: string) => void
}) {
  const { t } = useTranslation()
  const profile = businessProfileId ? getProfileById(businessProfileId as Parameters<typeof getProfileById>[0]) : null

  const coreModules = profile
    ? ALL_MODULES.filter((m) => profile.defaultModules.includes(m.id))
    : ALL_MODULES
  const optionalModules = profile
    ? ALL_MODULES.filter((m) => profile.optionalModules.includes(m.id))
    : []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('dashboard.modules.manage')}</DialogTitle>
          <DialogDescription>{t('dashboard.modules.manage.description')}</DialogDescription>
        </DialogHeader>

        {!profile ? (
          <div className="flex items-start gap-3 rounded-lg border border-border bg-muted/50 p-4">
            <Info className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
            <div>
              <p className="text-sm font-medium text-foreground">{t('dashboard.modules.manage.allVisible')}</p>
              <p className="mt-1 text-xs text-muted-foreground">{t('dashboard.modules.manage.allVisibleHint')}</p>
            </div>
          </div>
        ) : (
          <div className="space-y-6">
            <div>
              <h4 className="mb-3 flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                <Lock className="h-3 w-3" />
                {t('dashboard.modules.manage.core')}
              </h4>
              <div className="space-y-1">
                {coreModules.map((mod) => {
                  const Icon = getModuleIcon(mod.id)
                  if (!Icon) return null
                  return (
                    <div key={mod.id} className="flex items-center justify-between rounded-lg px-3 py-2.5">
                      <div className="flex items-center gap-3">
                        <div
                          className="flex h-8 w-8 items-center justify-center rounded-lg"
                          style={{ backgroundColor: moduleHslBg(mod.id) }}
                        >
                          <Icon className="h-4 w-4" style={{ color: moduleHsl(mod.id) }} />
                        </div>
                        <span className="text-sm text-foreground">{t(mod.nameKey)}</span>
                      </div>
                      <Switch checked disabled />
                    </div>
                  )
                })}
              </div>
            </div>

            {optionalModules.length > 0 && (
              <div>
                <h4 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  {t('dashboard.modules.manage.optional')}
                </h4>
                <div className="space-y-1">
                  {optionalModules.map((mod) => {
                    const Icon = getModuleIcon(mod.id)
                    if (!Icon) return null
                    const isEnabled = enabledOptionals.includes(mod.id)
                    return (
                      <div key={mod.id} className="flex items-center justify-between rounded-lg px-3 py-2.5 transition-colors hover:bg-muted/50">
                        <div className="flex items-center gap-3">
                          <div
                            className="flex h-8 w-8 items-center justify-center rounded-lg"
                            style={{ backgroundColor: moduleHslBg(mod.id) }}
                          >
                            <Icon className="h-4 w-4" style={{ color: moduleHsl(mod.id) }} />
                          </div>
                          <span className="text-sm text-foreground">{t(mod.nameKey)}</span>
                        </div>
                        <Switch
                          checked={isEnabled}
                          onCheckedChange={(checked) =>
                            checked ? onEnable(mod.id) : onDisable(mod.id)
                          }
                        />
                      </div>
                    )
                  })}
                </div>
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
