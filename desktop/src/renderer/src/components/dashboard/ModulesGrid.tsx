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
} from 'lucide-react'
import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/cn'

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

const modules: ModuleConfig[] = [
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
    path: '/documents',
    color: 'bg-purple-500',
    bgColor: 'bg-purple-500/10',
    iconColor: 'text-purple-500',
    stats: { label: 'Dokumente', value: 89 },
    isActive: true,
  },
  {
    id: 'accounting',
    name: 'Buchhaltung',
    description: 'Bexio, Abacus & Run my Accounts Integration',
    icon: Calculator,
    path: '/finance',
    color: 'bg-orange-500',
    bgColor: 'bg-orange-500/10',
    iconColor: 'text-orange-500',
    stats: { label: 'Integrationen', value: 0 },
    isActive: false,
    badge: 'Neu',
  },
  {
    id: 'communication',
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
    id: 'team',
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
]

export function ModulesGrid() {
  const [viewState, setViewState] = useState<ViewState>('full')

  const display =
    viewState === 'minimized'
      ? []
      : viewState === 'half'
        ? modules.slice(0, 3)
        : modules

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
