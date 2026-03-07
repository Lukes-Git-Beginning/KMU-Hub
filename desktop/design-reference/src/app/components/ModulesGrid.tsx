import { useState } from 'react';
import { FolderKanban, CheckSquare, FileText, MessageSquare, Users, Calculator, ArrowRight, ChevronDown, Minimize2, Maximize2 } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Badge } from './ui/badge';

const modules = [
  {
    id: 'projects',
    name: 'Projektverwaltung',
    description: 'Kanban-Boards, Zeiterfassung & Gantt-Charts',
    icon: FolderKanban,
    path: '/projekte',
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
    path: '/aufgaben',
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
    id: 'accounting',
    name: 'Buchhaltung',
    description: 'Bexio, Abacus & Run my Accounts Integration',
    icon: Calculator,
    path: '/buchhaltung',
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
    path: '/nachrichten',
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
    path: '/team',
    color: 'bg-cyan-500',
    bgColor: 'bg-cyan-500/10',
    iconColor: 'text-cyan-500',
    stats: { label: 'Team-Mitglieder', value: 12 },
    isActive: true,
  },
];

export function ModulesGrid() {
  const [viewState, setViewState] = useState<'minimized' | 'half' | 'full'>('full');

  // Determine which modules to show based on view state
  const displayModules = 
    viewState === 'minimized' 
      ? [] 
      : viewState === 'half' 
      ? modules.slice(0, 3)
      : modules;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-lg text-slate-900 dark:text-white">Ihre Module</h2>
        
        <div className="flex items-center gap-3">
          {/* View State Toggle Buttons */}
          <div className="flex items-center gap-1">
            <button
              onClick={() => setViewState('minimized')}
              className={`p-1.5 rounded transition-colors ${
                viewState === 'minimized'
                  ? 'bg-emerald-100 dark:bg-emerald-900 text-emerald-600 dark:text-emerald-400'
                  : 'text-slate-400 dark:text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-700'
              }`}
              title="Minimiert"
            >
              <Minimize2 className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={() => setViewState('half')}
              className={`p-1.5 rounded transition-colors ${
                viewState === 'half'
                  ? 'bg-emerald-100 dark:bg-emerald-900 text-emerald-600 dark:text-emerald-400'
                  : 'text-slate-400 dark:text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-700'
              }`}
              title="Halb"
            >
              <ChevronDown className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={() => setViewState('full')}
              className={`p-1.5 rounded transition-colors ${
                viewState === 'full'
                  ? 'bg-emerald-100 dark:bg-emerald-900 text-emerald-600 dark:text-emerald-400'
                  : 'text-slate-400 dark:text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-700'
              }`}
              title="Voll"
            >
              <Maximize2 className="w-3.5 h-3.5" />
            </button>
          </div>

          {viewState !== 'minimized' && (
            <Link 
              to="/module-verwalten" 
              className="text-sm text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300"
            >
              Module verwalten
            </Link>
          )}
        </div>
      </div>

      {viewState !== 'minimized' && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {displayModules.map((module) => {
            const Icon = module.icon;
            return (
              <Link
                key={module.id}
                to={module.path}
                className={`group relative bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6 hover:shadow-lg dark:hover:shadow-emerald-900/10 transition-all ${
                  !module.isActive ? 'opacity-75' : ''
                }`}
              >
                {/* Badge */}
                {module.badge && (
                  <Badge className="absolute top-4 right-4 bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300">
                    {module.badge}
                  </Badge>
                )}

                {/* Icon */}
                <div
                  className={`w-12 h-12 rounded-lg ${module.bgColor} dark:${module.bgColor} flex items-center justify-center mb-4`}
                >
                  <Icon className={`w-6 h-6 ${module.iconColor} dark:${module.iconColor}`} />
                </div>

                {/* Content */}
                <h3 className="text-slate-900 dark:text-white mb-2 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
                  {module.name}
                </h3>
                <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">{module.description}</p>

                {/* Stats */}
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-xs text-slate-500 dark:text-slate-400">{module.stats.label}</p>
                    <p className="text-2xl text-slate-900 dark:text-white">{module.stats.value}</p>
                  </div>
                  <ArrowRight className="w-5 h-5 text-slate-400 dark:text-slate-500 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 group-hover:translate-x-1 transition-all" />
                </div>

                {/* Inactive Overlay */}
                {!module.isActive && (
                  <div className="absolute inset-0 bg-white dark:bg-slate-800 bg-opacity-50 dark:bg-opacity-50 rounded-lg flex items-center justify-center">
                    <span className="px-3 py-1 bg-slate-900 dark:bg-slate-700 text-white text-xs rounded-full">
                      Nicht aktiviert
                    </span>
                  </div>
                )}
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}
