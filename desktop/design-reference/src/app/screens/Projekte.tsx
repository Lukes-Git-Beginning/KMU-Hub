import { useState } from 'react';
import { Plus, Calendar, Users as UsersIcon, MoreVertical, Clock } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Avatar, AvatarFallback } from '../components/ui/avatar';
import { Badge } from '../components/ui/badge';
import { Progress } from '../components/ui/progress';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../components/ui/dropdown-menu';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../components/ui/select';

const mockProjects = [
  {
    id: 1,
    name: 'Website Relaunch',
    description: 'Komplette Neugestaltung der Unternehmenswebsite mit modernem Design',
    progress: 75,
    status: 'active',
    priority: 'high',
    deadline: '2024-02-15',
    team: ['AM', 'MB', 'SK'],
    color: '#059669',
  },
  {
    id: 2,
    name: 'Mobile App',
    description: 'Entwicklung der iOS und Android App für Kunden',
    progress: 45,
    status: 'active',
    priority: 'high',
    deadline: '2024-03-20',
    team: ['JD', 'LS', 'AM'],
    color: '#3b82f6',
  },
  {
    id: 3,
    name: 'CRM Integration',
    description: 'Integration des neuen CRM Systems in bestehende Infrastruktur',
    progress: 92,
    status: 'active',
    priority: 'medium',
    deadline: '2024-01-30',
    team: ['MB', 'PK'],
    color: '#8b5cf6',
  },
  {
    id: 4,
    name: 'Marketing Kampagne',
    description: 'Q1 Marketing Kampagne für neue Produktlinie',
    progress: 30,
    status: 'pending',
    priority: 'medium',
    deadline: '2024-04-10',
    team: ['SK', 'LS', 'JD', 'AM'],
    color: '#f59e0b',
  },
  {
    id: 5,
    name: 'Datenmigration',
    description: 'Migration aller Kundendaten in neues System',
    progress: 100,
    status: 'completed',
    priority: 'high',
    deadline: '2024-01-15',
    team: ['PK', 'MB'],
    color: '#10b981',
  },
  {
    id: 6,
    name: 'Security Audit',
    description: 'Umfassende Sicherheitsüberprüfung aller Systeme',
    progress: 15,
    status: 'active',
    priority: 'high',
    deadline: '2024-05-01',
    team: ['JD', 'PK'],
    color: '#ef4444',
  },
];

export function Projekte() {
  const [statusFilter, setStatusFilter] = useState('all');
  const [sortBy, setSortBy] = useState('name');

  const getStatusBadgeVariant = (status: string) => {
    switch (status) {
      case 'completed':
        return 'default';
      case 'active':
        return 'default';
      case 'pending':
        return 'secondary';
      default:
        return 'secondary';
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'completed':
        return 'Abgeschlossen';
      case 'active':
        return 'In Arbeit';
      case 'pending':
        return 'Ausstehend';
      default:
        return status;
    }
  };

  const getPriorityLabel = (priority: string) => {
    switch (priority) {
      case 'high':
        return 'Hoch';
      case 'medium':
        return 'Mittel';
      case 'low':
        return 'Niedrig';
      default:
        return priority;
    }
  };

  const filteredProjects = mockProjects.filter(project => {
    if (statusFilter === 'all') return true;
    return project.status === statusFilter;
  });

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-8 bg-slate-50 dark:bg-slate-900">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 md:mb-8 gap-4">
        <div>
          <h1 className="text-slate-900 dark:text-white mb-1">Projekte</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400">Verwalte alle deine Projekte an einem Ort</p>
        </div>
        <button className="flex items-center justify-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors">
          <Plus className="w-4 h-4" />
          <span className="hidden sm:inline">Neues Projekt</span>
          <span className="sm:hidden">Neu</span>
        </button>
      </div>

      <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 md:gap-4 mb-6">
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-full sm:w-48">
            <SelectValue placeholder="Status filtern" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Alle</SelectItem>
            <SelectItem value="active">Aktiv</SelectItem>
            <SelectItem value="pending">Ausstehend</SelectItem>
            <SelectItem value="completed">Abgeschlossen</SelectItem>
          </SelectContent>
        </Select>

        <Select value={sortBy} onValueChange={setSortBy}>
          <SelectTrigger className="w-full sm:w-48">
            <SelectValue placeholder="Sortieren nach" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="name">Name</SelectItem>
            <SelectItem value="deadline">Deadline</SelectItem>
            <SelectItem value="progress">Fortschritt</SelectItem>
            <SelectItem value="priority">Priorität</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 md:gap-6">
        {filteredProjects.map((project) => (
          <Link
            key={project.id}
            to={`/projekte/${project.id}`}
            className="block bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-4 md:p-6 hover:shadow-lg dark:hover:shadow-emerald-900/10 transition-shadow"
          >
            <div
              className="w-12 h-12 rounded-lg mb-4 flex items-center justify-center"
              style={{ backgroundColor: `${project.color}20` }}
            >
              <div
                className="w-6 h-6 rounded"
                style={{ backgroundColor: project.color }}
              />
            </div>

            <h3 className="text-slate-900 dark:text-white mb-2">{project.name}</h3>
            <p className="text-sm text-slate-500 dark:text-slate-400 mb-4 line-clamp-2">
              {project.description}
            </p>

            <div className="mb-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-slate-600 dark:text-slate-400">Fortschritt</span>
                <span className="text-sm text-slate-900 dark:text-white">{project.progress}%</span>
              </div>
              <Progress value={project.progress} className="h-2" />
            </div>

            <div className="flex items-center gap-2 mb-4">
              <div className="flex -space-x-2">
                {project.team.map((member, idx) => (
                  <Avatar key={idx} className="w-8 h-8 border-2 border-white dark:border-slate-800">
                    <AvatarFallback className="text-xs bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300">
                      {member}
                    </AvatarFallback>
                  </Avatar>
                ))}
              </div>
            </div>

            <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2">
              <div className="flex items-center gap-2 flex-wrap">
                <Badge
                  variant={getStatusBadgeVariant(project.status)}
                  className={
                    project.status === 'completed'
                      ? 'bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300'
                      : project.status === 'active'
                      ? 'bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300'
                      : 'bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-300'
                  }
                >
                  {getStatusLabel(project.status)}
                </Badge>
                <Badge variant="outline" className="text-xs">
                  {getPriorityLabel(project.priority)}
                </Badge>
              </div>
              <div className="flex items-center gap-1 text-sm text-slate-500 dark:text-slate-400">
                <Calendar className="w-4 h-4" />
                {new Date(project.deadline).toLocaleDateString('de-DE')}
              </div>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}