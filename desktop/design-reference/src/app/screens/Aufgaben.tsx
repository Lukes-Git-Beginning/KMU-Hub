import { useState } from 'react';
import { Plus, Calendar, User } from 'lucide-react';
import { Checkbox } from '../components/ui/checkbox';
import { Badge } from '../components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';

const mockTasks = [
  {
    id: 1,
    title: 'Homepage Design finalisieren',
    project: 'Website Relaunch',
    assignee: 'AM',
    priority: 'high',
    deadline: '2024-01-25',
    completed: false,
    category: 'today',
  },
  {
    id: 2,
    title: 'API Dokumentation schreiben',
    project: 'Mobile App',
    assignee: 'JD',
    priority: 'medium',
    deadline: '2024-01-26',
    completed: false,
    category: 'today',
  },
  {
    id: 3,
    title: 'Datenbank Migration testen',
    project: 'CRM Integration',
    assignee: 'MB',
    priority: 'high',
    deadline: '2024-01-27',
    completed: false,
    category: 'week',
  },
  {
    id: 4,
    title: 'Social Media Posts erstellen',
    project: 'Marketing Kampagne',
    assignee: 'SK',
    priority: 'low',
    deadline: '2024-01-28',
    completed: false,
    category: 'week',
  },
  {
    id: 5,
    title: 'Code Review durchführen',
    project: 'Mobile App',
    assignee: 'LS',
    priority: 'medium',
    deadline: '2024-02-02',
    completed: false,
    category: 'later',
  },
  {
    id: 6,
    title: 'Server Konfiguration aktualisieren',
    project: 'Security Audit',
    assignee: 'PK',
    priority: 'high',
    deadline: '2024-01-24',
    completed: true,
    category: 'completed',
  },
];

export function Aufgaben() {
  const [view, setView] = useState('list');
  const [tasks, setTasks] = useState(mockTasks);

  const toggleTask = (id: number) => {
    setTasks(tasks.map(task => 
      task.id === id ? { ...task, completed: !task.completed } : task
    ));
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'high':
        return 'bg-red-100 text-red-700';
      case 'medium':
        return 'bg-yellow-100 text-yellow-700';
      case 'low':
        return 'bg-green-100 text-green-700';
      default:
        return 'bg-slate-100 text-slate-700';
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

  const renderTaskList = (category: string) => {
    const filteredTasks = tasks.filter(task => task.category === category);

    return (
      <div className="space-y-2">
        {filteredTasks.map((task) => (
          <div
            key={task.id}
            className="flex flex-col sm:flex-row items-start sm:items-center gap-3 sm:gap-4 p-4 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg hover:shadow-sm dark:hover:shadow-emerald-900/10 transition-shadow"
          >
            <div className="flex items-center gap-3 sm:gap-4 flex-1 min-w-0">
              <Checkbox
                checked={task.completed}
                onCheckedChange={() => toggleTask(task.id)}
              />
              <div
                className={`w-2 h-2 rounded-full flex-shrink-0 ${
                  task.priority === 'high'
                    ? 'bg-red-500'
                    : task.priority === 'medium'
                    ? 'bg-yellow-500'
                    : 'bg-green-500'
                }`}
              />
              <div className="flex-1 min-w-0">
                <p className={`text-sm sm:text-base text-slate-900 dark:text-white truncate ${task.completed ? 'line-through text-slate-400 dark:text-slate-500' : ''}`}>
                  {task.title}
                </p>
                <p className="text-xs sm:text-sm text-slate-500 dark:text-slate-400 truncate">{task.project}</p>
              </div>
            </div>
            <div className="flex items-center gap-2 ml-9 sm:ml-0">
              <Badge variant="outline" className="text-xs">
                {task.assignee}
              </Badge>
              <div className="flex items-center gap-1 text-xs sm:text-sm text-slate-500 dark:text-slate-400 whitespace-nowrap">
                <Calendar className="w-3 h-3 sm:w-4 sm:h-4" />
                {new Date(task.deadline).toLocaleDateString('de-DE')}
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  };

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-8 bg-slate-50 dark:bg-slate-900">
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between mb-6 md:mb-8 gap-4">
        <div>
          <h1 className="text-slate-900 dark:text-white mb-1">Aufgaben</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400">Behalte den Überblick über alle Aufgaben</p>
        </div>
        <button className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors">
          <Plus className="w-4 h-4" />
          Neue Aufgabe
        </button>
      </div>

      <Tabs defaultValue="all" className="w-full">
        <TabsList className="mb-6">
          <TabsTrigger value="all">Alle</TabsTrigger>
          <TabsTrigger value="assigned">Mir zugewiesen</TabsTrigger>
          <TabsTrigger value="created">Erstellt von mir</TabsTrigger>
          <TabsTrigger value="overdue">Überfällig</TabsTrigger>
        </TabsList>

        <TabsContent value="all" className="space-y-6">
          <div>
            <h3 className="text-sm text-slate-500 dark:text-slate-400 mb-3">Heute</h3>
            {renderTaskList('today')}
          </div>

          <div>
            <h3 className="text-sm text-slate-500 dark:text-slate-400 mb-3">Diese Woche</h3>
            {renderTaskList('week')}
          </div>

          <div>
            <h3 className="text-sm text-slate-500 dark:text-slate-400 mb-3">Später</h3>
            {renderTaskList('later')}
          </div>

          <div>
            <h3 className="text-sm text-slate-500 dark:text-slate-400 mb-3">Abgeschlossen</h3>
            {renderTaskList('completed')}
          </div>
        </TabsContent>

        <TabsContent value="assigned">
          <p className="text-slate-500 dark:text-slate-400">Aufgaben die dir zugewiesen sind...</p>
        </TabsContent>

        <TabsContent value="created">
          <p className="text-slate-500 dark:text-slate-400">Aufgaben die du erstellt hast...</p>
        </TabsContent>

        <TabsContent value="overdue">
          <p className="text-slate-500 dark:text-slate-400">Überfällige Aufgaben...</p>
        </TabsContent>
      </Tabs>
    </div>
  );
}