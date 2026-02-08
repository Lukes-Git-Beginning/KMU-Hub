import { useState, useRef, useEffect } from 'react'
import { Plus, X, ArrowRight, Clock, Check, Calendar } from 'lucide-react'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface PlannerTask {
  id: string
  text: string
  completed: boolean
  priority: 'high' | 'medium' | 'low' | 'none'
  reminder?: { day: string; time: string }
  completedAt?: string
  date?: string
}

type TabType = 'today' | 'tomorrow' | 'later'

const INITIAL_TASKS: PlannerTask[] = [
  { id: '1', text: 'Kundenpraesentation vorbereiten', completed: false, priority: 'high', reminder: { day: 'today', time: '10:00' } },
  { id: '2', text: 'E-Mail an Lieferant schreiben', completed: false, priority: 'medium' },
  { id: '3', text: 'Code Review fuer PR #234', completed: false, priority: 'low', reminder: { day: 'today', time: '14:30' } },
  { id: '4', text: 'Sprint Planning Meeting', completed: true, priority: 'none', completedAt: 'Heute, 11:15' },
  { id: '5', text: 'Projekt-Deadline abschliessen', completed: false, priority: 'high', reminder: { day: 'tomorrow', time: '09:00' } },
  { id: '6', text: 'Team-Standup vorbereiten', completed: false, priority: 'low', reminder: { day: 'tomorrow', time: '' } },
  { id: '7', text: 'Quartalsbericht vorbereiten', completed: false, priority: 'low', date: '23.12.2024' },
  { id: '8', text: 'Budget-Meeting', completed: false, priority: 'high', date: '24.12.2024' },
]

function sortByPriority(tasksList: PlannerTask[]) {
  const order = { high: 0, medium: 1, low: 2, none: 3 }
  return [...tasksList].sort(
    (a, b) => order[a.priority] - order[b.priority],
  )
}

export function DailyPlannerWidget() {
  const [isOpen, setIsOpen] = useState(false)
  const [activeTab, setActiveTab] = useState<TabType>('today')
  const [tasks, setTasks] = useState<PlannerTask[]>(INITIAL_TASKS)
  const [showAddForm, setShowAddForm] = useState(false)
  const [newText, setNewText] = useState('')
  const [newPriority, setNewPriority] = useState<PlannerTask['priority']>('none')
  const [showCompleted, setShowCompleted] = useState(true)
  const dropdownRef = useRef<HTMLDivElement>(null)

  const todayTasks = sortByPriority(
    tasks.filter((t) => !t.completed && !t.reminder?.day && !t.date),
  )
  const tomorrowTasks = sortByPriority(
    tasks.filter((t) => !t.completed && t.reminder?.day === 'tomorrow'),
  )
  const laterTasks = sortByPriority(
    tasks.filter((t) => !t.completed && t.date),
  )
  const completedTasks = tasks.filter((t) => t.completed)
  const highPriorityCount = tasks.filter(
    (t) => !t.completed && t.priority === 'high',
  ).length

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false)
        setShowAddForm(false)
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () =>
        document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  const toggleTask = (id: string) => {
    setTasks(
      tasks.map((t) => {
        if (t.id !== id) return t
        const now = new Date()
        return {
          ...t,
          completed: !t.completed,
          completedAt: !t.completed
            ? `Heute, ${now.getHours()}:${String(now.getMinutes()).padStart(2, '0')}`
            : undefined,
        }
      }),
    )
  }

  const deleteTask = (id: string) => {
    setTasks(tasks.filter((t) => t.id !== id))
  }

  const moveToTomorrow = (id: string) => {
    setTasks(
      tasks.map((t) =>
        t.id === id
          ? { ...t, reminder: { day: 'tomorrow', time: '' }, date: undefined }
          : t,
      ),
    )
  }

  const addTask = () => {
    if (!newText.trim()) return
    const task: PlannerTask = {
      id: Date.now().toString(),
      text: newText,
      completed: false,
      priority: newPriority,
    }
    setTasks([task, ...tasks])
    setNewText('')
    setNewPriority('none')
    setShowAddForm(false)
  }

  const currentTasks =
    activeTab === 'today'
      ? todayTasks
      : activeTab === 'tomorrow'
        ? tomorrowTasks
        : laterTasks

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Trigger */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="relative p-2 hover:bg-accent rounded-lg transition-colors"
      >
        <Check className="h-5 w-5 text-muted-foreground" />
        {highPriorityCount > 0 && (
          <div className="absolute -top-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-[10px] font-bold text-white">
            {highPriorityCount}
          </div>
        )}
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-96 rounded-xl border border-slate-700 bg-[#1e293b] shadow-2xl z-50">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-slate-700 p-4">
            <div className="flex items-center gap-2">
              <Check className="h-5 w-5 text-emerald-400" />
              <h3 className="font-semibold text-white">Meine Tagesplanung</h3>
            </div>
            <button
              onClick={() => setIsOpen(false)}
              className="rounded p-1 transition-colors hover:bg-slate-700"
            >
              <X className="h-4 w-4 text-slate-400" />
            </button>
          </div>

          <div className="max-h-[600px] overflow-y-auto">
            {/* Add Task Button */}
            {!showAddForm && (
              <button
                onClick={() => setShowAddForm(true)}
                className="mx-4 my-3 flex w-[calc(100%-2rem)] items-center justify-center gap-2 rounded-lg border border-dashed border-emerald-500 p-3 text-sm font-medium text-emerald-400 transition-colors hover:bg-emerald-500/10"
              >
                <Plus className="h-4 w-4" />
                Neue Aufgabe hinzufuegen
              </button>
            )}

            {/* Add Form */}
            {showAddForm && (
              <div className="m-4 space-y-3 rounded-lg border border-emerald-500 bg-[#0f172a] p-4">
                <div>
                  <label className="mb-1 block text-xs text-slate-400">
                    Aufgabe hinzufuegen:
                  </label>
                  <Input
                    value={newText}
                    onChange={(e) => setNewText(e.target.value)}
                    placeholder="z.B. Meeting-Notizen schreiben..."
                    className="border-slate-700 bg-[#1e293b] text-white"
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') addTask()
                      if (e.key === 'Escape') setShowAddForm(false)
                    }}
                    autoFocus
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs text-slate-400">
                    Prioritaet (optional):
                  </label>
                  <Select value={newPriority} onValueChange={(v) => setNewPriority(v as PlannerTask['priority'])}>
                    <SelectTrigger className="border-slate-700 bg-[#1e293b] text-white">
                      <SelectValue placeholder="Prioritaet..." />
                    </SelectTrigger>
                    <SelectContent className="border-slate-700 bg-[#1e293b]">
                      <SelectItem value="high" className="text-white">Hoch</SelectItem>
                      <SelectItem value="medium" className="text-white">Mittel</SelectItem>
                      <SelectItem value="low" className="text-white">Niedrig</SelectItem>
                      <SelectItem value="none" className="text-white">Keine</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex justify-end gap-2">
                  <button
                    onClick={() => setShowAddForm(false)}
                    className="rounded-lg bg-slate-700 px-4 py-2 text-sm text-slate-300 transition-colors hover:bg-slate-600"
                  >
                    Abbrechen
                  </button>
                  <button
                    onClick={addTask}
                    className="rounded-lg bg-emerald-600 px-4 py-2 text-sm text-white transition-colors hover:bg-emerald-700"
                  >
                    Hinzufuegen
                  </button>
                </div>
              </div>
            )}

            {/* Tabs + Tasks */}
            <div className="px-4 pb-4">
              <div className="mb-2 flex items-center justify-between">
                <h4 className="text-xs font-bold uppercase tracking-wide text-slate-500">
                  {activeTab === 'today'
                    ? `HEUTE (${todayTasks.length})`
                    : activeTab === 'tomorrow'
                      ? `MORGEN (${tomorrowTasks.length})`
                      : `SPAETER (${laterTasks.length})`}
                </h4>
                <div className="flex items-center gap-2">
                  {(['today', 'tomorrow', 'later'] as const).map((tab) => (
                    <button
                      key={tab}
                      onClick={() => setActiveTab(tab)}
                      className={`rounded border border-slate-600 px-2 py-1 text-xs text-slate-400 transition-colors hover:border-emerald-500 hover:text-emerald-400 ${
                        activeTab === tab ? 'bg-emerald-500/20' : ''
                      }`}
                    >
                      {tab === 'today'
                        ? 'Heute'
                        : tab === 'tomorrow'
                          ? 'Morgen'
                          : 'Spaeter'}
                    </button>
                  ))}
                </div>
              </div>

              <div className="space-y-2">
                {currentTasks.length === 0 && !showAddForm && (
                  <div className="py-8 text-center">
                    <div className="mb-2 text-4xl">&#10003;</div>
                    <p className="text-sm font-medium text-white">
                      Keine Aufgaben!
                    </p>
                    <p className="text-xs text-slate-400">
                      Fuege Tasks hinzu, die du erledigen moechtest.
                    </p>
                  </div>
                )}

                {currentTasks.map((task) => (
                  <TaskItem
                    key={task.id}
                    task={task}
                    onToggle={toggleTask}
                    onDelete={deleteTask}
                    onMoveToTomorrow={moveToTomorrow}
                  />
                ))}
              </div>
            </div>

            {/* Completed Section */}
            {completedTasks.length > 0 && (
              <div className="border-t border-slate-700 px-4 pb-4 pt-4">
                <div className="mb-2 flex items-center justify-between">
                  <h4 className="text-xs font-bold uppercase tracking-wide text-slate-500">
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
                    {completedTasks.slice(0, 3).map((task) => (
                      <div
                        key={task.id}
                        className="rounded-lg border border-slate-700 bg-[#0f172a] p-3"
                      >
                        <div className="flex items-start gap-3">
                          <div className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded bg-emerald-500">
                            <Check className="h-3 w-3 text-white" />
                          </div>
                          <div className="min-w-0 flex-1">
                            <p className="text-sm text-slate-500 line-through">
                              {task.text}
                            </p>
                            {task.completedAt && (
                              <div className="mt-1 flex items-center gap-1 text-xs text-slate-600">
                                <Check className="h-3 w-3" />
                                <span>{task.completedAt}</span>
                              </div>
                            )}
                          </div>
                          <button
                            onClick={() => deleteTask(task.id)}
                            className="rounded p-1 transition-colors hover:bg-red-500/20"
                          >
                            <X className="h-4 w-4 text-slate-600 hover:text-red-400" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function TaskItem({
  task,
  onToggle,
  onDelete,
  onMoveToTomorrow,
}: {
  task: PlannerTask
  onToggle: (id: string) => void
  onDelete: (id: string) => void
  onMoveToTomorrow: (id: string) => void
}) {
  return (
    <div className="rounded-lg border border-slate-700 bg-[#0f172a] p-3 transition-colors hover:border-emerald-500/50">
      <div className="flex items-start gap-3">
        <button
          onClick={() => onToggle(task.id)}
          className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded border-2 border-emerald-500 transition-colors hover:bg-emerald-500/20"
        >
          {task.completed && <Check className="h-3 w-3 text-emerald-500" />}
        </button>
        <div className="min-w-0 flex-1">
          <p className="text-sm text-slate-200">{task.text}</p>
          {task.reminder?.time && (
            <div className="mt-1 flex items-center gap-1 text-xs text-yellow-500">
              <Clock className="h-3 w-3" />
              <span>{task.reminder.time}</span>
            </div>
          )}
          {task.date && (
            <div className="mt-1 flex items-center gap-1 text-xs text-yellow-500">
              <Calendar className="h-3 w-3" />
              <span>{task.date}</span>
            </div>
          )}
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => onMoveToTomorrow(task.id)}
            className="flex items-center gap-1 rounded border border-slate-600 px-2 py-1 text-xs text-slate-400 transition-colors hover:border-emerald-500 hover:text-emerald-400"
          >
            <ArrowRight className="h-3 w-3" />
            Morgen
          </button>
          <button
            onClick={() => onDelete(task.id)}
            className="rounded p-1 transition-colors hover:bg-red-500/20"
          >
            <X className="h-4 w-4 text-slate-500 hover:text-red-400" />
          </button>
        </div>
      </div>
    </div>
  )
}
