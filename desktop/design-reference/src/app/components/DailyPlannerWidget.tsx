import { useState, useRef, useEffect } from 'react';
import { Plus, X, ArrowRight, Clock, Check, Calendar } from 'lucide-react';
import { Input } from './ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';

interface Task {
  id: string;
  text: string;
  completed: boolean;
  priority?: 'high' | 'medium' | 'low' | 'none';
  reminder?: {
    day: string;
    time: string;
  };
  completedAt?: string;
  date?: string; // For later tasks
  movedFrom?: string; // Track if moved from today
}

type TabType = 'today' | 'tomorrow' | 'later';

export function DailyPlannerWidget() {
  const [isOpen, setIsOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<TabType>('today');
  const [tasks, setTasks] = useState<Task[]>([
    { id: '1', text: 'Kundenpräsentation vorbereiten', completed: false, priority: 'high', reminder: { day: 'today', time: '10:00' } },
    { id: '2', text: 'E-Mail an Lieferant schreiben', completed: false, priority: 'medium' },
    { id: '3', text: 'Code Review für PR #234', completed: false, priority: 'low', reminder: { day: 'today', time: '14:30' } },
    { id: '4', text: 'Sprint Planning Meeting', completed: true, completedAt: 'Heute, 11:15' },
    { id: '5', text: 'Design Mockups reviewen', completed: true, completedAt: 'Heute, 09:42' },
    { id: '6', text: 'Projekt-Deadline abschließen', completed: false, priority: 'high', reminder: { day: 'tomorrow', time: '09:00' } },
    { id: '7', text: 'Team-Standup vorbereiten', completed: false, priority: 'low', reminder: { day: 'tomorrow', time: '' } },
    { id: '8', text: 'Quartalsbericht vorbereiten', completed: false, priority: 'low', date: '23.12.2024' },
    { id: '9', text: 'Budget-Meeting', completed: false, priority: 'high', date: '24.12.2024' },
    { id: '10', text: 'Dokumentation aktualisieren', completed: false, priority: 'low' },
  ]);
  const [showAddForm, setShowAddForm] = useState(false);
  const [newTaskText, setNewTaskText] = useState('');
  const [newTaskPriority, setNewTaskPriority] = useState<'high' | 'medium' | 'low' | 'none'>('none');
  const [newTaskDay, setNewTaskDay] = useState('');
  const [newTaskTime, setNewTaskTime] = useState('');
  const [showCompleted, setShowCompleted] = useState(true);
  const [showPriorityMenu, setShowPriorityMenu] = useState<string | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const getPriorityColor = (priority?: string) => {
    switch (priority) {
      case 'high': return '#EF4444';
      case 'medium': return '#F97316';
      case 'low': return '#F59E0B';
      default: return 'transparent';
    }
  };

  const getPriorityEmoji = (priority?: string) => {
    switch (priority) {
      case 'high': return '🔴';
      case 'medium': return '🟠';
      case 'low': return '🟡';
      default: return '⚪';
    }
  };

  const sortByPriority = (tasksList: Task[]) => {
    const priorityOrder = { high: 0, medium: 1, low: 2, none: 3 };
    return [...tasksList].sort((a, b) => {
      const aPriority = priorityOrder[a.priority || 'none'];
      const bPriority = priorityOrder[b.priority || 'none'];
      return aPriority - bPriority;
    });
  };

  const todayTasks = sortByPriority(tasks.filter(t => !t.completed && !t.reminder?.day && !t.date));
  const tomorrowTasks = sortByPriority(tasks.filter(t => !t.completed && t.reminder?.day === 'tomorrow'));
  const movedToTomorrow = sortByPriority(tasks.filter(t => !t.completed && t.movedFrom === 'today'));
  const laterTasksWithDate = sortByPriority(tasks.filter(t => !t.completed && t.date));
  const laterTasksNoDate = sortByPriority(tasks.filter(t => !t.completed && !t.date && !t.reminder?.day && t.id !== '1' && t.id !== '2' && t.id !== '3'));
  const completedTasks = tasks.filter(t => t.completed);

  const highPriorityCount = tasks.filter(t => !t.completed && t.priority === 'high').length;

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setShowAddForm(false);
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  const handleToggleTask = (id: string) => {
    setTasks(tasks.map(task => {
      if (task.id === id) {
        const now = new Date();
        return {
          ...task,
          completed: !task.completed,
          completedAt: !task.completed ? `Heute, ${now.getHours()}:${String(now.getMinutes()).padStart(2, '0')}` : undefined,
        };
      }
      return task;
    }));
  };

  const handleDeleteTask = (id: string) => {
    setTasks(tasks.filter(task => task.id !== id));
  };

  const handleMoveToTomorrow = (id: string) => {
    // In real app, this would update the task's date
    handleDeleteTask(id);
    // Show toast: "Auf morgen verschoben"
  };

  const handleAddTask = () => {
    if (!newTaskText.trim()) return;

    const newTask: Task = {
      id: Date.now().toString(),
      text: newTaskText,
      completed: false,
      priority: newTaskPriority,
      reminder: newTaskDay && newTaskTime ? { day: newTaskDay, time: newTaskTime } : undefined,
    };

    setTasks([newTask, ...tasks]);
    setNewTaskText('');
    setNewTaskPriority('none');
    setNewTaskDay('');
    setNewTaskTime('');
    setShowAddForm(false);
  };

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Icon Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="relative p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
      >
        <div className="w-5 h-5 flex items-center justify-center">
          <Check className="w-5 h-5 text-slate-600 dark:text-slate-300" />
        </div>
        {highPriorityCount > 0 && (
          <div className="absolute -top-1 -right-1 bg-red-500 text-black text-xs font-bold rounded-full w-5 h-5 flex items-center justify-center">
            {highPriorityCount}
          </div>
        )}
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-96 bg-[#1e293b] border border-slate-700 rounded-xl shadow-2xl z-50 animate-in slide-in-from-top-2 duration-200">
          {/* Header */}
          <div className="flex items-center justify-between p-4 border-b border-slate-700">
            <div className="flex items-center gap-2">
              <Check className="w-5 h-5 text-emerald-400" />
              <h3 className="font-semibold text-white">Meine Tagesplanung</h3>
            </div>
            <button
              onClick={() => setIsOpen(false)}
              className="p-1 hover:bg-slate-700 rounded transition-colors"
            >
              <X className="w-4 h-4 text-slate-400" />
            </button>
          </div>

          {/* Content */}
          <div className="max-h-[600px] overflow-y-auto">
            {/* Add New Task Button */}
            {!showAddForm && (
              <button
                onClick={() => setShowAddForm(true)}
                className="w-full mx-4 my-3 p-3 border border-dashed border-emerald-500 rounded-lg text-emerald-400 hover:bg-emerald-500/10 transition-colors text-sm font-medium flex items-center justify-center gap-2"
                style={{ width: 'calc(100% - 2rem)' }}
              >
                <Plus className="w-4 h-4" />
                Neue Aufgabe hinzufügen
              </button>
            )}

            {/* Add Task Form */}
            {showAddForm && (
              <div className="m-4 p-4 bg-[#0f172a] border border-emerald-500 rounded-lg space-y-3">
                <div>
                  <label className="text-xs text-slate-400 mb-1 block">Aufgabe hinzufügen:</label>
                  <Input
                    value={newTaskText}
                    onChange={(e) => setNewTaskText(e.target.value)}
                    placeholder="z.B. Meeting-Notizen schreiben..."
                    className="bg-[#1e293b] border-slate-700 text-white"
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleAddTask();
                      if (e.key === 'Escape') setShowAddForm(false);
                    }}
                    autoFocus
                  />
                </div>
                <div>
                  <label className="text-xs text-slate-400 mb-1 block">Priorität (optional):</label>
                  <div className="relative">
                    <Select value={newTaskPriority} onValueChange={setNewTaskPriority}>
                      <SelectTrigger className="bg-[#1e293b] border-slate-700 text-white">
                        <SelectValue placeholder="Priorität..." />
                      </SelectTrigger>
                      <SelectContent className="bg-[#1e293b] border-slate-700">
                        <SelectItem value="high" className="text-white">Hoch</SelectItem>
                        <SelectItem value="medium" className="text-white">Mittel</SelectItem>
                        <SelectItem value="low" className="text-white">Niedrig</SelectItem>
                      </SelectContent>
                    </Select>
                    <div
                      className="absolute top-0 right-0 h-full w-6 flex items-center justify-center pointer-events-none"
                      style={{ backgroundColor: getPriorityColor(newTaskPriority) }}
                    >
                      {getPriorityEmoji(newTaskPriority)}
                    </div>
                  </div>
                </div>
                <div>
                  <label className="text-xs text-slate-400 mb-1 block">Erinnerung (optional):</label>
                  <div className="flex gap-2">
                    <Select value={newTaskDay} onValueChange={setNewTaskDay}>
                      <SelectTrigger className="flex-1 bg-[#1e293b] border-slate-700 text-white">
                        <SelectValue placeholder="Tag..." />
                      </SelectTrigger>
                      <SelectContent className="bg-[#1e293b] border-slate-700">
                        <SelectItem value="today" className="text-white">Heute</SelectItem>
                        <SelectItem value="tomorrow" className="text-white">Morgen</SelectItem>
                        <SelectItem value="dayafter" className="text-white">Übermorgen</SelectItem>
                      </SelectContent>
                    </Select>
                    <Select value={newTaskTime} onValueChange={setNewTaskTime}>
                      <SelectTrigger className="flex-1 bg-[#1e293b] border-slate-700 text-white">
                        <SelectValue placeholder="Uhrzeit..." />
                      </SelectTrigger>
                      <SelectContent className="bg-[#1e293b] border-slate-700">
                        <SelectItem value="09:00" className="text-white">09:00</SelectItem>
                        <SelectItem value="10:00" className="text-white">10:00</SelectItem>
                        <SelectItem value="11:00" className="text-white">11:00</SelectItem>
                        <SelectItem value="14:00" className="text-white">14:00</SelectItem>
                        <SelectItem value="15:00" className="text-white">15:00</SelectItem>
                        <SelectItem value="16:00" className="text-white">16:00</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <div className="flex gap-2 justify-end">
                  <button
                    onClick={() => setShowAddForm(false)}
                    className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-300 rounded-lg text-sm transition-colors"
                  >
                    Abbrechen
                  </button>
                  <button
                    onClick={handleAddTask}
                    className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-sm transition-colors"
                  >
                    Hinzufügen
                  </button>
                </div>
              </div>
            )}

            {/* Tabs */}
            <div className="px-4 pb-4">
              <div className="flex items-center justify-between mb-2">
                <h4 className="text-xs font-bold text-slate-500 uppercase tracking-wide">
                  HEUTE ({todayTasks.length + movedToTomorrow.length})
                </h4>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setActiveTab('today')}
                    className={`px-2 py-1 text-xs border border-slate-600 hover:border-emerald-500 text-slate-400 hover:text-emerald-400 rounded transition-colors flex items-center gap-1 ${activeTab === 'today' ? 'bg-emerald-500/20' : ''}`}
                  >
                    Heute
                  </button>
                  <button
                    onClick={() => setActiveTab('tomorrow')}
                    className={`px-2 py-1 text-xs border border-slate-600 hover:border-emerald-500 text-slate-400 hover:text-emerald-400 rounded transition-colors flex items-center gap-1 ${activeTab === 'tomorrow' ? 'bg-emerald-500/20' : ''}`}
                  >
                    Morgen
                  </button>
                  <button
                    onClick={() => setActiveTab('later')}
                    className={`px-2 py-1 text-xs border border-slate-600 hover:border-emerald-500 text-slate-400 hover:text-emerald-400 rounded transition-colors flex items-center gap-1 ${activeTab === 'later' ? 'bg-emerald-500/20' : ''}`}
                  >
                    Später
                  </button>
                </div>
              </div>
              <div className="space-y-2">
                {activeTab === 'today' && (
                  <>
                    {todayTasks.map((task) => (
                      <div
                        key={task.id}
                        className="bg-[#0f172a] border border-slate-700 rounded-lg p-3 hover:border-emerald-500/50 transition-colors"
                      >
                        <div className="flex items-start gap-3">
                          <button
                            onClick={() => handleToggleTask(task.id)}
                            className="mt-0.5 w-5 h-5 border-2 border-emerald-500 rounded flex items-center justify-center hover:bg-emerald-500/20 transition-colors flex-shrink-0"
                          >
                            {task.completed && <Check className="w-3 h-3 text-emerald-500" />}
                          </button>
                          <div className="flex-1 min-w-0">
                            <p className="text-sm text-slate-200">{task.text}</p>
                            {task.reminder && (
                              <div className="flex items-center gap-1 mt-1 text-xs text-yellow-500">
                                <Clock className="w-3 h-3" />
                                <span>{task.reminder.time}</span>
                              </div>
                            )}
                          </div>
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleMoveToTomorrow(task.id)}
                              className="px-2 py-1 text-xs border border-slate-600 hover:border-emerald-500 text-slate-400 hover:text-emerald-400 rounded transition-colors flex items-center gap-1"
                            >
                              <ArrowRight className="w-3 h-3" />
                              Morgen
                            </button>
                            <button
                              onClick={() => handleDeleteTask(task.id)}
                              className="p-1 hover:bg-red-500/20 rounded transition-colors"
                            >
                              <X className="w-4 h-4 text-slate-500 hover:text-red-400" />
                            </button>
                          </div>
                        </div>
                      </div>
                    ))}
                    {movedToTomorrow.map((task) => (
                      <div
                        key={task.id}
                        className="bg-[#0f172a] border border-slate-700 rounded-lg p-3 hover:border-emerald-500/50 transition-colors"
                      >
                        <div className="flex items-start gap-3">
                          <button
                            onClick={() => handleToggleTask(task.id)}
                            className="mt-0.5 w-5 h-5 border-2 border-emerald-500 rounded flex items-center justify-center hover:bg-emerald-500/20 transition-colors flex-shrink-0"
                          >
                            {task.completed && <Check className="w-3 h-3 text-emerald-500" />}
                          </button>
                          <div className="flex-1 min-w-0">
                            <p className="text-sm text-slate-200">{task.text}</p>
                            {task.reminder && (
                              <div className="flex items-center gap-1 mt-1 text-xs text-yellow-500">
                                <Clock className="w-3 h-3" />
                                <span>{task.reminder.time}</span>
                              </div>
                            )}
                          </div>
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleMoveToTomorrow(task.id)}
                              className="px-2 py-1 text-xs border border-slate-600 hover:border-emerald-500 text-slate-400 hover:text-emerald-400 rounded transition-colors flex items-center gap-1"
                            >
                              <ArrowRight className="w-3 h-3" />
                              Morgen
                            </button>
                            <button
                              onClick={() => handleDeleteTask(task.id)}
                              className="p-1 hover:bg-red-500/20 rounded transition-colors"
                            >
                              <X className="w-4 h-4 text-slate-500 hover:text-red-400" />
                            </button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </>
                )}
                {activeTab === 'tomorrow' && (
                  <>
                    {tomorrowTasks.map((task) => (
                      <div
                        key={task.id}
                        className="bg-[#0f172a] border border-slate-700 rounded-lg p-3 hover:border-emerald-500/50 transition-colors"
                      >
                        <div className="flex items-start gap-3">
                          <button
                            onClick={() => handleToggleTask(task.id)}
                            className="mt-0.5 w-5 h-5 border-2 border-emerald-500 rounded flex items-center justify-center hover:bg-emerald-500/20 transition-colors flex-shrink-0"
                          >
                            {task.completed && <Check className="w-3 h-3 text-emerald-500" />}
                          </button>
                          <div className="flex-1 min-w-0">
                            <p className="text-sm text-slate-200">{task.text}</p>
                            {task.reminder && (
                              <div className="flex items-center gap-1 mt-1 text-xs text-yellow-500">
                                <Clock className="w-3 h-3" />
                                <span>{task.reminder.time}</span>
                              </div>
                            )}
                          </div>
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleMoveToTomorrow(task.id)}
                              className="px-2 py-1 text-xs border border-slate-600 hover:border-emerald-500 text-slate-400 hover:text-emerald-400 rounded transition-colors flex items-center gap-1"
                            >
                              <ArrowRight className="w-3 h-3" />
                              Morgen
                            </button>
                            <button
                              onClick={() => handleDeleteTask(task.id)}
                              className="p-1 hover:bg-red-500/20 rounded transition-colors"
                            >
                              <X className="w-4 h-4 text-slate-500 hover:text-red-400" />
                            </button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </>
                )}
                {activeTab === 'later' && (
                  <>
                    {laterTasksWithDate.map((task) => (
                      <div
                        key={task.id}
                        className="bg-[#0f172a] border border-slate-700 rounded-lg p-3 hover:border-emerald-500/50 transition-colors"
                      >
                        <div className="flex items-start gap-3">
                          <button
                            onClick={() => handleToggleTask(task.id)}
                            className="mt-0.5 w-5 h-5 border-2 border-emerald-500 rounded flex items-center justify-center hover:bg-emerald-500/20 transition-colors flex-shrink-0"
                          >
                            {task.completed && <Check className="w-3 h-3 text-emerald-500" />}
                          </button>
                          <div className="flex-1 min-w-0">
                            <p className="text-sm text-slate-200">{task.text}</p>
                            {task.date && (
                              <div className="flex items-center gap-1 mt-1 text-xs text-yellow-500">
                                <Calendar className="w-3 h-3" />
                                <span>{task.date}</span>
                              </div>
                            )}
                          </div>
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleMoveToTomorrow(task.id)}
                              className="px-2 py-1 text-xs border border-slate-600 hover:border-emerald-500 text-slate-400 hover:text-emerald-400 rounded transition-colors flex items-center gap-1"
                            >
                              <ArrowRight className="w-3 h-3" />
                              Morgen
                            </button>
                            <button
                              onClick={() => handleDeleteTask(task.id)}
                              className="p-1 hover:bg-red-500/20 rounded transition-colors"
                            >
                              <X className="w-4 h-4 text-slate-500 hover:text-red-400" />
                            </button>
                          </div>
                        </div>
                      </div>
                    ))}
                    {laterTasksNoDate.map((task) => (
                      <div
                        key={task.id}
                        className="bg-[#0f172a] border border-slate-700 rounded-lg p-3 hover:border-emerald-500/50 transition-colors"
                      >
                        <div className="flex items-start gap-3">
                          <button
                            onClick={() => handleToggleTask(task.id)}
                            className="mt-0.5 w-5 h-5 border-2 border-emerald-500 rounded flex items-center justify-center hover:bg-emerald-500/20 transition-colors flex-shrink-0"
                          >
                            {task.completed && <Check className="w-3 h-3 text-emerald-500" />}
                          </button>
                          <div className="flex-1 min-w-0">
                            <p className="text-sm text-slate-200">{task.text}</p>
                          </div>
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleMoveToTomorrow(task.id)}
                              className="px-2 py-1 text-xs border border-slate-600 hover:border-emerald-500 text-slate-400 hover:text-emerald-400 rounded transition-colors flex items-center gap-1"
                            >
                              <ArrowRight className="w-3 h-3" />
                              Morgen
                            </button>
                            <button
                              onClick={() => handleDeleteTask(task.id)}
                              className="p-1 hover:bg-red-500/20 rounded transition-colors"
                            >
                              <X className="w-4 h-4 text-slate-500 hover:text-red-400" />
                            </button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </>
                )}
              </div>
            </div>

            {/* Completed Tasks */}
            {completedTasks.length > 0 && (
              <div className="px-4 pb-4 border-t border-slate-700 pt-4">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="text-xs font-bold text-slate-500 uppercase tracking-wide">
                    ERLEDIGT ({completedTasks.length})
                  </h4>
                  <button
                    onClick={() => setShowCompleted(!showCompleted)}
                    className="text-xs text-emerald-400 hover:text-emerald-300"
                  >
                    {showCompleted ? 'Ausblenden' : 'Anzeigen'}
                  </button>
                </div>
                {showCompleted && (
                  <div className="space-y-2 opacity-60">
                    {completedTasks.slice(0, 2).map((task) => (
                      <div
                        key={task.id}
                        className="bg-[#0f172a] border border-slate-700 rounded-lg p-3"
                      >
                        <div className="flex items-start gap-3">
                          <div className="mt-0.5 w-5 h-5 bg-emerald-500 rounded flex items-center justify-center flex-shrink-0">
                            <Check className="w-3 h-3 text-white" />
                          </div>
                          <div className="flex-1 min-w-0">
                            <p className="text-sm text-slate-500 line-through">{task.text}</p>
                            {task.completedAt && (
                              <div className="flex items-center gap-1 mt-1 text-xs text-slate-600">
                                <Check className="w-3 h-3" />
                                <span>{task.completedAt}</span>
                              </div>
                            )}
                          </div>
                          <button
                            onClick={() => handleDeleteTask(task.id)}
                            className="p-1 hover:bg-red-500/20 rounded transition-colors"
                          >
                            <X className="w-4 h-4 text-slate-600 hover:text-red-400" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Empty State */}
            {todayTasks.length === 0 && tomorrowTasks.length === 0 && movedToTomorrow.length === 0 && laterTasksWithDate.length === 0 && laterTasksNoDate.length === 0 && !showAddForm && (
              <div className="py-12 text-center px-4">
                <div className="text-5xl mb-3">✓</div>
                <h4 className="text-sm font-medium text-white mb-1">Keine Aufgaben für heute!</h4>
                <p className="text-xs text-slate-400">
                  Füge Tasks hinzu, die du heute erledigen möchtest.
                </p>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}