import { useState } from 'react'
import {
  FolderKanban,
  CheckSquare,
  FileText,
  MessageSquare,
  Users,
  Calculator,
  ArrowRight,
  ChevronDown,
  Minimize2,
  Maximize2,
  Package,
  CalendarClock,
  ShoppingCart,
  Headphones,
  Truck,
  Factory,
  BarChart3,
  FileSignature,
  HardHat,
  Timer,
  ClipboardList,
  KeyRound,
} from 'lucide-react'
import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/cn'
import { isModuleAllowedForProfile } from '@/config/business-profiles'
import { useProfileStore } from '@/stores/profile'

type ViewState = 'minimized' | 'half' | 'full'

interface ModuleConfig {
  id: string
  name: string
  description: string
  icon: typeof FolderKanban
  path: string
  color: string
  bgColor: string
  iconColor: string
  stats: { label: string; value: number }
  isActive: boolean
  badge?: string
}

const ALL_MODULES: ModuleConfig[] = [
  {
    id: 'projects',
    name: 'Projektverwaltung',
    description: 'Kanban-Boards, Zeiterfassung & Gantt-Charts',
    icon: FolderKanban,
    path: '/work/projects',
    color: 'bg-blue-500',
    bgColor: 'bg-blue-500/10',
    iconColor: 'text-blue-500',
    stats: { label: 'Aktive Projekte', value: 24 },
    isActive: true,
  },
  {
    id: 'tasks',
    name: 'Aufgabenverwaltung',
    description: 'To-Dos, Deadlines & Team-Zuweisungen',
    icon: CheckSquare,
    path: '/work/my-tasks',
    color: 'bg-emerald-500',
    bgColor: 'bg-emerald-500/10',
    iconColor: 'text-emerald-500',
    stats: { label: 'Offene Aufgaben', value: 142 },
    isActive: true,
  },
  {
    id: 'documents',
    name: 'Dokumentenmanagement',
    description: 'Zentrale Ablage mit Versionierung',
    icon: FileText,
    path: '/dokumente',
    color: 'bg-purple-500',
    bgColor: 'bg-purple-500/10',
    iconColor: 'text-purple-500',
    stats: { label: 'Dokumente', value: 89 },
    isActive: true,
  },
  {
    id: 'finance',
    name: 'Buchhaltung',
    description: 'Bexio, Abacus & Run my Accounts Integration',
    icon: Calculator,
    path: '/buchhaltung',
    color: 'bg-orange-500',
    bgColor: 'bg-orange-500/10',
    iconColor: 'text-orange-500',
    stats: { label: 'Rechnungen', value: 18 },
    isActive: true,
    badge: 'Neu',
  },
  {
    id: 'chat',
    name: 'Kommunikation',
    description: 'Team-Chat & Channels',
    icon: MessageSquare,
    path: '/chat',
    color: 'bg-pink-500',
    bgColor: 'bg-pink-500/10',
    iconColor: 'text-pink-500',
    stats: { label: 'Neue Nachrichten', value: 37 },
    isActive: true,
  },
  {
    id: 'crm',
    name: 'Team & CRM',
    description: 'Mitarbeiterverwaltung & Kontakte',
    icon: Users,
    path: '/crm',
    color: 'bg-cyan-500',
    bgColor: 'bg-cyan-500/10',
    iconColor: 'text-cyan-500',
    stats: { label: 'Team-Mitglieder', value: 12 },
    isActive: true,
  },
  // -- New industry modules --
  {
    id: 'inventar',
    name: 'Inventar',
    description: 'Lagerverwaltung mit Barcode-Scanning',
    icon: Package,
    path: '/inventar',
    color: 'bg-amber-500',
    bgColor: 'bg-amber-500/10',
    iconColor: 'text-amber-500',
    stats: { label: 'Artikel', value: 245 },
    isActive: true,
  },
  {
    id: 'schichten',
    name: 'Schichtplanung',
    description: 'Wochenpläne, Vorlagen & Tausch-Anfragen',
    icon: CalendarClock,
    path: '/schichten',
    color: 'bg-indigo-500',
    bgColor: 'bg-indigo-500/10',
    iconColor: 'text-indigo-500',
    stats: { label: 'Diese Woche', value: 32 },
    isActive: true,
  },
  {
    id: 'einkauf',
    name: 'Einkauf',
    description: 'Lieferanten, Bestellungen & Liefertracking',
    icon: ShoppingCart,
    path: '/einkauf',
    color: 'bg-lime-600',
    bgColor: 'bg-lime-600/10',
    iconColor: 'text-lime-600',
    stats: { label: 'Offene Bestellungen', value: 7 },
    isActive: true,
  },
  {
    id: 'helpdesk',
    name: 'Helpdesk',
    description: 'Tickets, SLA-Tracking & Wissensdatenbank',
    icon: Headphones,
    path: '/helpdesk',
    color: 'bg-violet-500',
    bgColor: 'bg-violet-500/10',
    iconColor: 'text-violet-500',
    stats: { label: 'Offene Tickets', value: 15 },
    isActive: true,
  },
  {
    id: 'fuhrpark',
    name: 'Fuhrpark',
    description: 'Fahrzeuge, Wartung & Tankprotokoll',
    icon: Truck,
    path: '/fuhrpark',
    color: 'bg-sky-600',
    bgColor: 'bg-sky-600/10',
    iconColor: 'text-sky-600',
    stats: { label: 'Fahrzeuge', value: 6 },
    isActive: true,
  },
  {
    id: 'produktion',
    name: 'Produktion',
    description: 'Stücklisten, Aufträge & Qualitätskontrolle',
    icon: Factory,
    path: '/produktion',
    color: 'bg-yellow-600',
    bgColor: 'bg-yellow-600/10',
    iconColor: 'text-yellow-600',
    stats: { label: 'Aufträge', value: 8 },
    isActive: true,
  },
  {
    id: 'berichte',
    name: 'Berichte',
    description: 'KPI-Dashboard, Diagramme & Exporte',
    icon: BarChart3,
    path: '/berichte',
    color: 'bg-fuchsia-500',
    bgColor: 'bg-fuchsia-500/10',
    iconColor: 'text-fuchsia-500',
    stats: { label: 'Berichte', value: 3 },
    isActive: true,
  },
  {
    id: 'zeiterfassung',
    name: 'Zeiterfassung',
    description: 'Stunden erfassen, Projekte zuordnen & Auswertungen',
    icon: Timer,
    path: '/zeiterfassung',
    color: 'bg-emerald-600',
    bgColor: 'bg-emerald-600/10',
    iconColor: 'text-emerald-600',
    stats: { label: 'Heute erfasst', value: 6 },
    isActive: true,
  },
  {
    id: 'vertraege',
    name: 'Vertraege',
    description: 'Vertragsverwaltung & Fristen-Tracking',
    icon: FileSignature,
    path: '/vertraege',
    color: 'bg-teal-600',
    bgColor: 'bg-teal-600/10',
    iconColor: 'text-teal-600',
    stats: { label: 'Aktive Vertraege', value: 9 },
    isActive: true,
  },
  {
    id: 'formulare',
    name: 'Formulare',
    description: 'Eigene Formulare erstellen & Eingaenge verwalten',
    icon: ClipboardList,
    path: '/formulare',
    color: 'bg-rose-500',
    bgColor: 'bg-rose-500/10',
    iconColor: 'text-rose-500',
    stats: { label: 'Aktive Formulare', value: 4 },
    isActive: true,
  },
  {
    id: 'vermietung',
    name: 'Vermietung',
    description: 'Objekte, Reservierungen & Verfuegbarkeit',
    icon: KeyRound,
    path: '/vermietung',
    color: 'bg-cyan-600',
    bgColor: 'bg-cyan-600/10',
    iconColor: 'text-cyan-600',
    stats: { label: 'Verfuegbare Objekte', value: 5 },
    isActive: true,
  },
  {
    id: 'rapporte',
    name: 'Rapporte',
    description: 'Tagesberichte, Aufmass & Feldberichte',
    icon: HardHat,
    path: '/rapporte',
    color: 'bg-orange-600',
    bgColor: 'bg-orange-600/10',
    iconColor: 'text-orange-600',
    stats: { label: 'Berichte diese Woche', value: 8 },
    isActive: true,
  },
]

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
          {display.map((mod) => {
            const Icon = mod.icon
            return (
              <Link
                key={mod.id}
                to={mod.path}
                className={cn(
                  'group relative rounded-lg border border-border bg-card p-6 transition-all hover:shadow-lg',
                  !mod.isActive && 'opacity-75',
                )}
              >
                {mod.badge && (
                  <Badge className="absolute right-4 top-4 bg-primary/10 text-primary">
                    {mod.badge}
                  </Badge>
                )}

                <div
                  className={`mb-4 flex h-12 w-12 items-center justify-center rounded-lg ${mod.bgColor}`}
                >
                  <Icon className={`h-6 w-6 ${mod.iconColor}`} />
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
                    <p className="text-2xl text-foreground">{mod.stats.value}</p>
                  </div>
                  <ArrowRight className="h-5 w-5 text-muted-foreground transition-all group-hover:translate-x-1 group-hover:text-primary" />
                </div>

                {!mod.isActive && (
                  <div className="absolute inset-0 flex items-center justify-center rounded-lg bg-card/50">
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
