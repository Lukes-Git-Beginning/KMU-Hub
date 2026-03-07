import { FolderKanban, CheckSquare, FileText, MessageSquare, Users, Calculator, ArrowRight, Clock, TrendingUp, AlertCircle } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Badge } from '../components/ui/badge';
import { Avatar, AvatarFallback } from '../components/ui/avatar';
import { Progress } from '../components/ui/progress';
import { NotificationsFeed } from '../components/NotificationsFeed';
import { ModulesGrid } from '../components/ModulesGrid';

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

const recentActivity = [
  {
    id: 1,
    user: 'Michael Berg',
    avatar: 'MB',
    action: 'hat Projekt "Website Relaunch" aktualisiert',
    time: 'vor 5 Minuten',
    type: 'project',
  },
  {
    id: 2,
    user: 'Sarah Klein',
    avatar: 'SK',
    action: 'hat 3 neue Aufgaben erstellt',
    time: 'vor 1 Stunde',
    type: 'task',
  },
  {
    id: 3,
    user: 'Anna Mller',
    avatar: 'AM',
    action: 'hat Dokument "Q1 Budget.xlsx" hochgeladen',
    time: 'vor 2 Stunden',
    type: 'document',
  },
];

const alerts = [
  {
    id: 1,
    title: '3 Projekte mit Deadline diese Woche',
    type: 'warning',
    action: 'Projekte anzeigen',
    path: '/projekte',
  },
  {
    id: 2,
    title: 'Buchhaltungs-Integration verfügbar',
    type: 'info',
    action: 'Jetzt verbinden',
    path: '/buchhaltung',
  },
];

export function Dashboard() {
  const currentHour = new Date().getHours();
  const greeting = currentHour < 12 ? 'Guten Morgen' : currentHour < 18 ? 'Guten Tag' : 'Guten Abend';

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-8 bg-slate-50 dark:bg-slate-900">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-slate-900 dark:text-white mb-1">Dashboard</h1>
        <p className="text-slate-500 dark:text-slate-400">Willkommen im KMU Digital Hub – Ihre All-in-One Plattform</p>
      </div>

      {/* Alerts */}
      {alerts.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
          {alerts.map((alert) => (
            <div
              key={alert.id}
              className={`flex items-center justify-between p-4 rounded-lg border ${
                alert.type === 'warning'
                  ? 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800'
                  : 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800'
              }`}
            >
              <div className="flex items-center gap-3">
                <AlertCircle
                  className={`w-5 h-5 ${
                    alert.type === 'warning' ? 'text-yellow-600 dark:text-yellow-400' : 'text-blue-600 dark:text-blue-400'
                  }`}
                />
                <span className="text-sm text-slate-900 dark:text-slate-100">{alert.title}</span>
              </div>
              <Link
                to={alert.path}
                className={`text-sm font-medium hover:underline ${
                  alert.type === 'warning' ? 'text-yellow-700 dark:text-yellow-400' : 'text-blue-700 dark:text-blue-400'
                }`}
              >
                {alert.action}
              </Link>
            </div>
          ))}
        </div>
      )}

      {/* Notifications Feed - Full Width */}
      <div className="mb-8">
        <NotificationsFeed />
      </div>

      {/* Module Grid */}
      <div className="mb-8">
        <ModulesGrid />
      </div>

      {/* Two Column Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Recent Activity */}
        <div className="lg:col-span-2">
          <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-lg text-slate-900 dark:text-white">Letzte Aktivitäten</h2>
              <Clock className="w-5 h-5 text-slate-400 dark:text-slate-500" />
            </div>

            <div className="space-y-4">
              {recentActivity.map((activity) => (
                <div key={activity.id} className="flex items-start gap-3 pb-4 border-b border-slate-100 dark:border-slate-700 last:border-0">
                  <Avatar className="w-10 h-10 flex-shrink-0">
                    <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300 text-sm">
                      {activity.avatar}
                    </AvatarFallback>
                  </Avatar>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-slate-900 dark:text-slate-100">
                      <span className="font-medium">{activity.user}</span>{' '}
                      {activity.action}
                    </p>
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">{activity.time}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Quick Stats */}
        <div className="space-y-6">
          <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
            <div className="flex items-center gap-3 mb-4">
              <TrendingUp className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
              <h3 className="text-sm text-slate-900 dark:text-white">Fortschritt heute</h3>
            </div>
            <div className="space-y-4">
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-slate-600 dark:text-slate-400">Aufgaben erledigt</span>
                  <span className="text-xs text-slate-900 dark:text-white font-medium">12/28</span>
                </div>
                <Progress value={43} className="h-2" />
              </div>
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-slate-600 dark:text-slate-400">Projekt-Fortschritt</span>
                  <span className="text-xs text-slate-900 dark:text-white font-medium">75%</span>
                </div>
                <Progress value={75} className="h-2" />
              </div>
            </div>
          </div>

          <div className="bg-gradient-to-br from-emerald-500 to-emerald-600 rounded-lg p-6 text-white">
            <h3 className="text-lg mb-2">Benötigen Sie Hilfe?</h3>
            <p className="text-sm text-emerald-100 mb-4">
              Unser Support-Team steht Ihnen zur Verfügung
            </p>
            <button className="w-full px-4 py-2 bg-white text-emerald-600 rounded-lg hover:bg-emerald-50 transition-colors text-sm font-medium">
              Support kontaktieren
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}