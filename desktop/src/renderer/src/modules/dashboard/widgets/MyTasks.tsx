/**
 * My Tasks widget — personal open tasks with deadlines.
 */
import { memo } from 'react'
import { CheckCircle2, Circle, AlertTriangle, Clock } from 'lucide-react'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

interface MockTask {
  id: string
  title: string
  project: string
  priority: 'high' | 'medium' | 'low'
  due: string
  done: boolean
}

const MOCK_TASKS: MockTask[] = [
  { id: '1', title: 'Angebot fuer Meier GmbH erstellen', project: 'CRM', priority: 'high', due: 'Heute', done: false },
  { id: '2', title: 'Protokoll Sprint Review', project: 'Intern', priority: 'medium', due: 'Heute', done: false },
  { id: '3', title: 'Kundenfeedback auswerten', project: 'Produkt', priority: 'medium', due: 'Morgen', done: false },
  { id: '4', title: 'Rechnung #2024-087 pruefen', project: 'Finanzen', priority: 'low', due: 'Mi, 26.02.', done: false },
  { id: '5', title: 'Onboarding-Unterlagen vorbereiten', project: 'HR', priority: 'high', due: 'Do, 27.02.', done: false },
  { id: '6', title: 'Wochenbericht abschliessen', project: 'Intern', priority: 'low', due: 'Fr, 28.02.', done: true },
]

const PRIORITY_STYLE = {
  high: 'text-red-500',
  medium: 'text-amber-500',
  low: 'text-muted-foreground',
}

function MyTasks(_props: WidgetProps) {
  const open = MOCK_TASKS.filter((t) => !t.done)
  const done = MOCK_TASKS.filter((t) => t.done)

  return (
    <div className="flex h-full flex-col">
      {/* Summary */}
      <div className="flex items-center justify-between px-4 pt-4 pb-2">
        <span className="text-xs text-muted-foreground">{open.length} offen · {done.length} erledigt</span>
        <span className="flex items-center gap-1 text-xs text-red-500">
          <AlertTriangle className="h-3 w-3" />
          {open.filter((t) => t.due === 'Heute').length} heute faellig
        </span>
      </div>

      {/* Task list */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {MOCK_TASKS.map((task) => (
          <div
            key={task.id}
            className="flex items-start gap-3 px-4 py-2.5 hover:bg-accent/50 cursor-pointer transition-colors"
          >
            {task.done ? (
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-emerald-500" />
            ) : (
              <Circle className={`mt-0.5 h-4 w-4 shrink-0 ${PRIORITY_STYLE[task.priority]}`} />
            )}
            <div className="min-w-0 flex-1">
              <p className={`text-sm ${task.done ? 'line-through text-muted-foreground' : 'font-medium text-foreground'}`}>
                {task.title}
              </p>
              <div className="flex items-center gap-2 mt-0.5">
                <span className="text-[10px] rounded-full bg-secondary px-1.5 py-0.5 text-muted-foreground">
                  {task.project}
                </span>
                <span className="flex items-center gap-0.5 text-[10px] text-muted-foreground">
                  <Clock className="h-2.5 w-2.5" />
                  {task.due}
                </span>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default memo(MyTasks)
