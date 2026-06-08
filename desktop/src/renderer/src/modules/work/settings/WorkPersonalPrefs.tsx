/**
 * WorkPersonalPrefs — personal-scope settings for the work module (everyone can
 * adapt these to their own workflow). Embedded as the "Persönlich" section of
 * the work settings panel. Backed by stores/workPrefs.
 */
import { useTranslation } from 'react-i18next'
import { Columns3, List, GanttChartSquare, Rows3, Rows4 } from 'lucide-react'
import {
  useWorkPrefsStore,
  type WorkView,
  type MyTasksGroupBy,
  type WorkDensity,
} from '@/stores/workPrefs'
import { useProjects } from '@/api/hooks/useProjects'

export function WorkPersonalPrefs() {
  const { t } = useTranslation()
  const defaultView = useWorkPrefsStore((s) => s.defaultView)
  const myTasksGroupBy = useWorkPrefsStore((s) => s.myTasksGroupBy)
  const density = useWorkPrefsStore((s) => s.density)
  const defaultProjectId = useWorkPrefsStore((s) => s.defaultProjectId)
  const setDefaultView = useWorkPrefsStore((s) => s.setDefaultView)
  const setMyTasksGroupBy = useWorkPrefsStore((s) => s.setMyTasksGroupBy)
  const setDensity = useWorkPrefsStore((s) => s.setDensity)
  const setDefaultProjectId = useWorkPrefsStore((s) => s.setDefaultProjectId)

  const { data } = useProjects()
  const projects = data?.projects ?? []

  const viewOptions: { id: WorkView; labelKey: string; icon: typeof List }[] = [
    { id: 'kanban', labelKey: 'work.settings.personal.viewKanban', icon: Columns3 },
    { id: 'list', labelKey: 'work.settings.personal.viewList', icon: List },
    { id: 'gantt', labelKey: 'work.settings.personal.viewGantt', icon: GanttChartSquare },
  ]
  const groupOptions: { id: MyTasksGroupBy; labelKey: string }[] = [
    { id: 'project', labelKey: 'work.settings.personal.groupProject' },
    { id: 'priority', labelKey: 'work.settings.personal.groupPriority' },
    { id: 'dueDate', labelKey: 'work.settings.personal.groupDueDate' },
    { id: 'status', labelKey: 'work.settings.personal.groupStatus' },
  ]
  const densityOptions: { id: WorkDensity; labelKey: string; icon: typeof Rows3 }[] = [
    { id: 'comfortable', labelKey: 'work.settings.personal.densityComfortable', icon: Rows3 },
    { id: 'compact', labelKey: 'work.settings.personal.densityCompact', icon: Rows4 },
  ]

  const segBtn = (active: boolean) =>
    `flex flex-1 items-center justify-center gap-2 rounded-lg border py-2 text-sm transition-colors ${
      active
        ? 'border-primary bg-primary/5 font-medium text-primary'
        : 'border-border text-foreground hover:bg-secondary'
    }`

  return (
    <div className="space-y-5">
      {/* Default board view */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('work.settings.personal.defaultView')}</label>
        <div className="flex gap-2">
          {viewOptions.map((opt) => {
            const Icon = opt.icon
            return (
              <button key={opt.id} onClick={() => setDefaultView(opt.id)} className={segBtn(defaultView === opt.id)}>
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
      </div>

      {/* My-tasks grouping */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('work.settings.personal.myTasksGroupBy')}</label>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          {groupOptions.map((opt) => (
            <button key={opt.id} onClick={() => setMyTasksGroupBy(opt.id)} className={segBtn(myTasksGroupBy === opt.id)}>
              {t(opt.labelKey)}
            </button>
          ))}
        </div>
      </div>

      {/* Density */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('work.settings.personal.density')}</label>
        <div className="flex gap-2">
          {densityOptions.map((opt) => {
            const Icon = opt.icon
            return (
              <button key={opt.id} onClick={() => setDensity(opt.id)} className={segBtn(density === opt.id)}>
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
      </div>

      {/* Default project */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('work.settings.personal.defaultProject')}</label>
        <select
          value={defaultProjectId ?? ''}
          onChange={(e) => setDefaultProjectId(e.target.value || null)}
          className="h-9 w-full rounded-lg border border-border bg-transparent px-2 text-sm text-foreground outline-none focus:border-primary"
        >
          <option value="">{t('work.settings.personal.defaultProjectNone')}</option>
          {projects.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
        <p className="text-xs text-muted-foreground">{t('work.settings.personal.defaultProjectHint')}</p>
      </div>
    </div>
  )
}
