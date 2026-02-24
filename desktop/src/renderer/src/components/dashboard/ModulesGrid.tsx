import { useState } from 'react'
import {
  ArrowRight,
  ChevronDown,
  Minimize2,
  Maximize2,
} from 'lucide-react'
import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/cn'
import { isModuleAllowedForProfile } from '@/config/business-profiles'
import { useProfileStore } from '@/stores/profile'
import { navItems, moduleHsl, moduleHslBg } from '@/components/layout/sidebar/nav-items'

type ViewState = 'minimized' | 'half' | 'full'

interface ModuleConfig {
  id: string
  name: string
  description: string
  path: string
  stats: { label: string; value: number }
  isActive: boolean
  badge?: string
}

/**
 * Module definitions — colors + icons come from nav-items.ts.
 * Only module-specific metadata (description, stats, badge) is here.
 */
const ALL_MODULES: ModuleConfig[] = [
  { id: 'projects', name: 'Projektverwaltung', description: 'Kanban-Boards, Zeiterfassung & Gantt-Charts', path: '/work/projects', stats: { label: 'Aktive Projekte', value: 24 }, isActive: true },
  { id: 'tasks', name: 'Aufgabenverwaltung', description: 'To-Dos, Deadlines & Team-Zuweisungen', path: '/work/my-tasks', stats: { label: 'Offene Aufgaben', value: 142 }, isActive: true },
  { id: 'documents', name: 'Dokumentenmanagement', description: 'Zentrale Ablage mit Versionierung', path: '/dokumente', stats: { label: 'Dokumente', value: 89 }, isActive: true },
  { id: 'finance', name: 'Finanzen', description: 'Rechnungen, Angebote & DATEV-Export', path: '/finanzen', stats: { label: 'Rechnungen', value: 18 }, isActive: true, badge: 'Neu' },
  { id: 'chat', name: 'Kommunikation', description: 'Team-Chat & Channels', path: '/chat', stats: { label: 'Neue Nachrichten', value: 37 }, isActive: true },
  { id: 'crm', name: 'Team & CRM', description: 'Mitarbeiterverwaltung & Kontakte', path: '/crm', stats: { label: 'Team-Mitglieder', value: 12 }, isActive: true },
  { id: 'inventar', name: 'Inventar', description: 'Lagerverwaltung mit Barcode-Scanning', path: '/inventar', stats: { label: 'Artikel', value: 245 }, isActive: true },
  { id: 'schichten', name: 'Schichtplanung', description: 'Wochenpläne, Vorlagen & Tausch-Anfragen', path: '/schichten', stats: { label: 'Diese Woche', value: 32 }, isActive: true },
  { id: 'einkauf', name: 'Einkauf', description: 'Lieferanten, Bestellungen & Liefertracking', path: '/einkauf', stats: { label: 'Offene Bestellungen', value: 7 }, isActive: true },
  { id: 'helpdesk', name: 'Helpdesk', description: 'Tickets, SLA-Tracking & Wissensdatenbank', path: '/helpdesk', stats: { label: 'Offene Tickets', value: 15 }, isActive: true },
  { id: 'fuhrpark', name: 'Fuhrpark', description: 'Fahrzeuge, Wartung & Tankprotokoll', path: '/fuhrpark', stats: { label: 'Fahrzeuge', value: 6 }, isActive: true },
  { id: 'produktion', name: 'Produktion', description: 'Stücklisten, Aufträge & Qualitätskontrolle', path: '/produktion', stats: { label: 'Aufträge', value: 8 }, isActive: true },
  { id: 'berichte', name: 'Berichte', description: 'KPI-Dashboard, Diagramme & Exporte', path: '/berichte', stats: { label: 'Berichte', value: 3 }, isActive: true },
  { id: 'zeiterfassung', name: 'Zeiterfassung', description: 'Stunden erfassen, Projekte zuordnen & Auswertungen', path: '/zeiterfassung', stats: { label: 'Heute erfasst', value: 6 }, isActive: true },
  { id: 'vertraege', name: 'Vertraege', description: 'Vertragsverwaltung & Fristen-Tracking', path: '/vertraege', stats: { label: 'Aktive Vertraege', value: 9 }, isActive: true },
  { id: 'formulare', name: 'Formulare', description: 'Eigene Formulare erstellen & Eingaenge verwalten', path: '/formulare', stats: { label: 'Aktive Formulare', value: 4 }, isActive: true },
  { id: 'vermietung', name: 'Vermietung', description: 'Objekte, Reservierungen & Verfuegbarkeit', path: '/vermietung', stats: { label: 'Verfuegbare Objekte', value: 5 }, isActive: true },
  { id: 'rapporte', name: 'Rapporte', description: 'Tagesberichte, Aufmass & Feldberichte', path: '/rapporte', stats: { label: 'Berichte diese Woche', value: 8 }, isActive: true },
]

/** Look up the icon from nav-items by module ID */
function getModuleIcon(id: string) {
  return navItems.find((i) => i.id === id)?.icon
}

export function ModulesGrid() {
  const [viewState, setViewState] = useState<ViewState>('full')
  const businessProfileId = useProfileStore((s) => s.businessProfileId)
  const devShowAll = useProfileStore((s) => s.devShowAllModules)
  const enabledOptionals = useProfileStore((s) => s.enabledOptionalModules)

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
        <h2 className="text-lg font-semibold text-foreground">Ihre Module</h2>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1">
            {(
              [
                { value: 'minimized', icon: Minimize2, title: 'Minimiert' },
                { value: 'half', icon: ChevronDown, title: 'Halb' },
                { value: 'full', icon: Maximize2, title: 'Voll' },
              ] as const
            ).map((item) => {
              const Icon = item.icon
              return (
                <button
                  key={item.value}
                  onClick={() => setViewState(item.value)}
                  title={item.title}
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
            <span className="text-sm text-primary">Module verwalten</span>
          )}
        </div>
      </div>

      {viewState !== 'minimized' && (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
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
                {mod.badge && (
                  <Badge className="absolute right-4 top-4 badge-accent bg-primary/10 text-primary">
                    {mod.badge}
                  </Badge>
                )}

                <div
                  className="mb-4 flex h-12 w-12 items-center justify-center rounded-xl"
                  style={{ backgroundColor: bgColor }}
                >
                  <Icon className="h-6 w-6" style={{ color }} />
                </div>

                <h3 className="mb-2 font-medium text-foreground transition-colors group-hover:text-primary">
                  {mod.name}
                </h3>
                <p className="mb-4 text-sm text-muted-foreground">
                  {mod.description}
                </p>

                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-xs text-muted-foreground">
                      {mod.stats.label}
                    </p>
                    <p className="text-2xl text-foreground stat-accent">{mod.stats.value}</p>
                  </div>
                  <ArrowRight className="h-5 w-5 text-muted-foreground transition-all group-hover:translate-x-1 group-hover:text-primary" />
                </div>

                {!mod.isActive && (
                  <div className="absolute inset-0 flex items-center justify-center rounded-xl bg-card/50">
                    <span className="rounded-full bg-foreground px-3 py-1 text-xs text-background">
                      Nicht aktiviert
                    </span>
                  </div>
                )}
              </Link>
            )
          })}
        </div>
      )}
    </div>
  )
}
