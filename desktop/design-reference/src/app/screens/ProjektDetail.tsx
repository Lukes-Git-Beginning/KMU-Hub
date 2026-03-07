import { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ChevronLeft, Calendar, Users as UsersIcon, MoreVertical, Plus } from 'lucide-react';
import { DndProvider, useDrag, useDrop } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { Avatar, AvatarFallback } from '../components/ui/avatar';
import { Badge } from '../components/ui/badge';
import { Progress } from '../components/ui/progress';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { TimeTracker } from '../components/TimeTracker';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '../components/ui/breadcrumb';

const projectData = {
  id: 1,
  name: 'Website Relaunch',
  description: 'Komplette Neugestaltung der Unternehmenswebsite mit modernem Design und verbesserter User Experience. Ziel ist es, die Conversion Rate um 30% zu steigern.',
  progress: 75,
  status: 'active',
  priority: 'high',
  deadline: '2024-02-15',
  team: [
    { name: 'Anna Müller', avatar: 'AM', role: 'Project Lead' },
    { name: 'Michael Berg', avatar: 'MB', role: 'Developer' },
    { name: 'Sarah Klein', avatar: 'SK', role: 'Designer' },
  ],
  milestones: [
    { id: 1, title: 'Design Mockups', completed: true, date: '2024-01-10' },
    { id: 2, title: 'Frontend Development', completed: true, date: '2024-01-20' },
    { id: 3, title: 'Backend Integration', completed: false, date: '2024-02-05' },
    { id: 4, title: 'Testing & QA', completed: false, date: '2024-02-12' },
    { id: 5, title: 'Launch', completed: false, date: '2024-02-15' },
  ],
};

const initialTasks = {
  todo: [
    { id: '1', title: 'API Endpoints dokumentieren', assignee: 'MB', priority: 'medium' },
    { id: '2', title: 'E-Mail Templates erstellen', assignee: 'SK', priority: 'low' },
  ],
  inProgress: [
    { id: '3', title: 'Backend Integration testen', assignee: 'MB', priority: 'high' },
    { id: '4', title: 'Responsive Design anpassen', assignee: 'SK', priority: 'high' },
  ],
  review: [
    { id: '5', title: 'Performance Optimierung', assignee: 'MB', priority: 'medium' },
  ],
  done: [
    { id: '6', title: 'Design System aufsetzen', assignee: 'SK', priority: 'high' },
    { id: '7', title: 'Datenbank Schema erstellen', assignee: 'MB', priority: 'high' },
  ],
};

interface Task {
  id: string;
  title: string;
  assignee: string;
  priority: string;
}

interface DragItem {
  id: string;
  column: string;
}

function TaskCard({ task, column }: { task: Task; column: string }) {
  const [{ isDragging }, drag] = useDrag(() => ({
    type: 'TASK',
    item: { id: task.id, column },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  }));

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

  return (
    <div
      ref={drag}
      className={`bg-white border border-slate-200 rounded-lg p-3 cursor-move hover:shadow-md transition-shadow ${
        isDragging ? 'opacity-50' : ''
      }`}
    >
      <div className="flex items-start justify-between mb-2">
        <p className="text-sm text-slate-900 flex-1">{task.title}</p>
        <button className="p-1 hover:bg-slate-100 rounded">
          <MoreVertical className="w-4 h-4 text-slate-400" />
        </button>
      </div>
      <div className="flex items-center justify-between">
        <Avatar className="w-6 h-6">
          <AvatarFallback className="text-xs bg-emerald-100 text-emerald-700">
            {task.assignee}
          </AvatarFallback>
        </Avatar>
        <Badge className={`text-xs ${getPriorityColor(task.priority)}`}>
          {task.priority}
        </Badge>
      </div>
    </div>
  );
}

function KanbanColumn({
  title,
  tasks,
  columnId,
  onDrop,
}: {
  title: string;
  tasks: Task[];
  columnId: string;
  onDrop: (item: DragItem, targetColumn: string) => void;
}) {
  const [{ isOver }, drop] = useDrop(() => ({
    accept: 'TASK',
    drop: (item: DragItem) => onDrop(item, columnId),
    collect: (monitor) => ({
      isOver: monitor.isOver(),
    }),
  }));

  return (
    <div className="flex-1 min-w-[280px]">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm text-slate-700">{title}</h3>
        <span className="text-xs text-slate-500">{tasks.length}</span>
      </div>
      <div
        ref={drop}
        className={`space-y-3 min-h-[500px] p-3 rounded-lg border-2 border-dashed transition-colors ${
          isOver ? 'border-emerald-400 bg-emerald-50/50' : 'border-slate-200 bg-slate-50'
        }`}
      >
        {tasks.map((task) => (
          <TaskCard key={task.id} task={task} column={columnId} />
        ))}
        <button className="w-full p-3 border-2 border-dashed border-slate-200 rounded-lg text-sm text-slate-500 hover:border-emerald-400 hover:text-emerald-600 transition-colors">
          <Plus className="w-4 h-4 mx-auto" />
        </button>
      </div>
    </div>
  );
}

export function ProjektDetail() {
  const { id } = useParams();
  const [tasks, setTasks] = useState(initialTasks);

  const handleDrop = (item: DragItem, targetColumn: string) => {
    if (item.column === targetColumn) return;

    setTasks((prev) => {
      const sourceColumn = prev[item.column as keyof typeof prev];
      const targetCol = prev[targetColumn as keyof typeof prev];
      const taskToMove = sourceColumn.find((t) => t.id === item.id);

      if (!taskToMove) return prev;

      return {
        ...prev,
        [item.column]: sourceColumn.filter((t) => t.id !== item.id),
        [targetColumn]: [...targetCol, taskToMove],
      };
    });
  };

  return (
    <DndProvider backend={HTML5Backend}>
      <div className="flex-1 overflow-y-auto p-4 md:p-8">
        <Breadcrumb className="mb-6">
          <BreadcrumbList>
            <BreadcrumbItem>
              <BreadcrumbLink asChild>
                <Link to="/projekte">Projekte</Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbPage>{projectData.name}</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>

        <div className="bg-white border border-slate-200 rounded-lg p-6 mb-6">
          <div className="flex flex-col md:flex-row md:items-start justify-between mb-6 gap-4">
            <div className="flex-1">
              <h1 className="text-slate-900 mb-2">{projectData.name}</h1>
              <div className="flex flex-wrap items-center gap-2 mb-4">
                <Badge className="bg-blue-100 text-blue-700">In Arbeit</Badge>
                <Badge variant="outline">Hohe Priorität</Badge>
              </div>
            </div>
            <button className="px-4 py-2 border border-slate-200 text-slate-700 rounded-lg hover:bg-slate-50 transition-colors text-sm">
              Aktionen
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div>
              <p className="text-sm text-slate-500 mb-1">Team</p>
              <div className="flex -space-x-2">
                {projectData.team.map((member, idx) => (
                  <Avatar key={idx} className="w-8 h-8 border-2 border-white">
                    <AvatarFallback className="text-xs bg-emerald-100 text-emerald-700">
                      {member.avatar}
                    </AvatarFallback>
                  </Avatar>
                ))}
              </div>
            </div>
            <div>
              <p className="text-sm text-slate-500 mb-1">Deadline</p>
              <div className="flex items-center gap-2 text-slate-900">
                <Calendar className="w-4 h-4" />
                {new Date(projectData.deadline).toLocaleDateString('de-DE')}
              </div>
            </div>
            <div>
              <p className="text-sm text-slate-500 mb-1">Fortschritt</p>
              <div className="flex items-center gap-3">
                <Progress value={projectData.progress} className="flex-1 h-2" />
                <span className="text-sm text-slate-900">{projectData.progress}%</span>
              </div>
            </div>
          </div>
        </div>

        <Tabs defaultValue="overview" className="w-full">
          <TabsList className="mb-6">
            <TabsTrigger value="overview">Übersicht</TabsTrigger>
            <TabsTrigger value="tasks">Aufgaben</TabsTrigger>
            <TabsTrigger value="timeline">Timeline</TabsTrigger>
            <TabsTrigger value="files">Dokumente</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              <div className="lg:col-span-2 space-y-6">
                <div className="bg-white border border-slate-200 rounded-lg p-6">
                  <h3 className="text-slate-900 mb-3">Projektbeschreibung</h3>
                  <p className="text-slate-600">{projectData.description}</p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  <div className="bg-white border border-slate-200 rounded-lg p-6">
                    <p className="text-sm text-slate-500 mb-1">Fortschritt</p>
                    <p className="text-2xl text-slate-900">{projectData.progress}%</p>
                  </div>
                  <div className="bg-white border border-slate-200 rounded-lg p-6">
                    <p className="text-sm text-slate-500 mb-1">Offene Aufgaben</p>
                    <p className="text-2xl text-slate-900">{tasks.todo.length + tasks.inProgress.length}</p>
                  </div>
                  <div className="bg-white border border-slate-200 rounded-lg p-6">
                    <p className="text-sm text-slate-500 mb-1">Beteiligte</p>
                    <p className="text-2xl text-slate-900">{projectData.team.length}</p>
                  </div>
                </div>

                <div className="bg-white border border-slate-200 rounded-lg p-6">
                  <h3 className="text-slate-900 mb-4">Meilensteine</h3>
                  <div className="space-y-3">
                    {projectData.milestones.map((milestone) => (
                      <div key={milestone.id} className="flex items-center gap-3">
                        <input
                          type="checkbox"
                          checked={milestone.completed}
                          className="w-4 h-4 text-emerald-600 rounded"
                          readOnly
                        />
                        <span
                          className={`flex-1 text-sm ${
                            milestone.completed ? 'text-slate-400 line-through' : 'text-slate-900'
                          }`}
                        >
                          {milestone.title}
                        </span>
                        <span className="text-xs text-slate-500">
                          {new Date(milestone.date).toLocaleDateString('de-DE')}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              {/* Time Tracker Sidebar */}
              <div>
                <TimeTracker projectName={projectData.name} />
              </div>
            </div>
          </TabsContent>

          <TabsContent value="tasks">
            <div className="flex gap-4 overflow-x-auto pb-4">
              <KanbanColumn
                title="To Do"
                tasks={tasks.todo}
                columnId="todo"
                onDrop={handleDrop}
              />
              <KanbanColumn
                title="In Progress"
                tasks={tasks.inProgress}
                columnId="inProgress"
                onDrop={handleDrop}
              />
              <KanbanColumn
                title="Review"
                tasks={tasks.review}
                columnId="review"
                onDrop={handleDrop}
              />
              <KanbanColumn
                title="Done"
                tasks={tasks.done}
                columnId="done"
                onDrop={handleDrop}
              />
            </div>
          </TabsContent>

          <TabsContent value="timeline">
            <div className="bg-white border border-slate-200 rounded-lg p-6">
              <div className="flex items-center justify-between mb-6">
                <h3 className="text-slate-900">Projekt-Timeline</h3>
                <div className="flex items-center gap-2">
                  <button className="px-3 py-1.5 text-sm border border-slate-200 rounded-lg hover:bg-slate-50">
                    Heute
                  </button>
                  <button className="px-3 py-1.5 text-sm border border-slate-200 rounded-lg hover:bg-slate-50">
                    Monat
                  </button>
                  <button className="px-3 py-1.5 text-sm bg-emerald-600 text-white rounded-lg hover:bg-emerald-700">
                    Quartal
                  </button>
                </div>
              </div>

              {/* Gantt Chart */}
              <div className="overflow-x-auto">
                <div className="min-w-[800px]">
                  {/* Timeline Header */}
                  <div className="flex border-b border-slate-200 pb-2 mb-4">
                    <div className="w-48 flex-shrink-0">
                      <p className="text-xs font-medium text-slate-500 uppercase">Aufgabe</p>
                    </div>
                    <div className="flex-1 grid grid-cols-12 gap-2">
                      {['Jan', 'Feb', 'Mär', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dez'].map((month, idx) => (
                        <div key={idx} className="text-center">
                          <p className="text-xs text-slate-500">{month}</p>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Timeline Rows */}
                  <div className="space-y-3">
                    {projectData.milestones.map((milestone, idx) => {
                      // Calculate position and width (simplified)
                      const startMonth = idx * 2;
                      const duration = 2;
                      
                      return (
                        <div key={milestone.id} className="flex items-center">
                          <div className="w-48 flex-shrink-0 pr-4">
                            <p className="text-sm text-slate-900 truncate">{milestone.title}</p>
                            <p className="text-xs text-slate-500">
                              {new Date(milestone.date).toLocaleDateString('de-DE')}
                            </p>
                          </div>
                          <div className="flex-1 grid grid-cols-12 gap-2 relative">
                            <div
                              className={`col-start-${startMonth + 1} col-span-${duration} h-8 rounded flex items-center px-2 ${
                                milestone.completed
                                  ? 'bg-emerald-500 text-white'
                                  : 'bg-blue-500 text-white'
                              }`}
                              style={{
                                gridColumnStart: startMonth + 1,
                                gridColumnEnd: `span ${duration}`,
                              }}
                            >
                              <span className="text-xs truncate">{milestone.completed ? '✓' : ''}</span>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>

                  {/* Current Date Indicator */}
                  <div className="relative mt-4 pt-4 border-t border-slate-200">
                    <div className="flex">
                      <div className="w-48 flex-shrink-0"></div>
                      <div className="flex-1 grid grid-cols-12 gap-2">
                        <div className="col-start-1 col-span-1 flex justify-center">
                          <div className="w-0.5 h-full bg-red-500 relative">
                            <div className="absolute -top-2 left-1/2 -translate-x-1/2 px-2 py-0.5 bg-red-500 text-white text-xs rounded whitespace-nowrap">
                              Heute
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Legend */}
                  <div className="mt-6 flex flex-wrap items-center gap-4 text-sm">
                    <div className="flex items-center gap-2">
                      <div className="w-4 h-4 bg-emerald-500 rounded"></div>
                      <span className="text-slate-600">Abgeschlossen</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="w-4 h-4 bg-blue-500 rounded"></div>
                      <span className="text-slate-600">In Arbeit</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="w-0.5 h-4 bg-red-500"></div>
                      <span className="text-slate-600">Heutiges Datum</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="files">
            <div className="bg-white border border-slate-200 rounded-lg p-6">
              <p className="text-slate-500">Dokumente-Ansicht kommt bald...</p>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </DndProvider>
  );
}