/**
 * Task dependency list with add/remove capability.
 *
 * Groups dependencies by type: blocks, blocked_by, relates_to, duplicates.
 * Shows each linked task with key + title (clickable for navigation).
 * Add dependency via popover with type selector and task search.
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Lock,
  ArrowRight,
  Link2,
  Copy,
  Plus,
  X,
  Search,
  Loader2,
} from 'lucide-react'
import { cn } from '@/lib'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  useTaskDependencies,
  useCreateDependency,
  useDeleteDependency,
  useTasks,
  type TaskDependency,
} from '@/api/hooks/useTasks'

interface DependencyListProps {
  taskId: string
  projectId: string
  /** RBAC (R-3): hides add/remove controls; linked tasks stay readable. */
  canEdit?: boolean
}

type DependencyType = 'blocks' | 'blocked_by' | 'relates_to' | 'duplicates'

const depTypeConfig: Record<
  DependencyType,
  {
    labelKey: string
    icon: React.ComponentType<{ className?: string }>
    color: string
  }
> = {
  blocks: {
    labelKey: 'work.dependencies.blocks',
    icon: ArrowRight,
    color: 'text-destructive',
  },
  blocked_by: {
    labelKey: 'work.dependencies.blockedBy',
    icon: Lock,
    color: 'text-warning-foreground',
  },
  relates_to: {
    labelKey: 'work.dependencies.relatesTo',
    icon: Link2,
    color: 'text-blue-500',
  },
  duplicates: {
    labelKey: 'work.dependencies.duplicateOf',
    icon: Copy,
    color: 'text-gray-500',
  },
}

export default function DependencyList({
  taskId,
  projectId,
  canEdit = true,
}: DependencyListProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data, isLoading } = useTaskDependencies(taskId)
  const createDep = useCreateDependency()
  const deleteDep = useDeleteDependency()

  const [addOpen, setAddOpen] = useState(false)
  const [addType, setAddType] = useState<DependencyType>('blocks')
  const [searchQuery, setSearchQuery] = useState('')

  // Fetch project tasks for search
  const { data: tasksData } = useTasks({
    project_id: projectId,
    search: searchQuery || undefined,
    page_size: 20,
  })

  const dependencies = useMemo(() => data?.dependencies ?? [], [data?.dependencies])
  const searchResults = useMemo(() => tasksData?.tasks ?? [], [tasksData?.tasks])

  // Filter out current task and already-linked tasks from search results
  const existingTargetIds = useMemo(() => {
    const ids = new Set<string>()
    for (const dep of dependencies) {
      if (dep.target_task_id) ids.add(dep.target_task_id)
      if (dep.source_task_id) ids.add(dep.source_task_id)
    }
    return ids
  }, [dependencies])

  const filteredSearchResults = useMemo(
    () =>
      searchResults.filter(
        (t) => t.id !== taskId && !existingTargetIds.has(t.id ?? '')
      ),
    [searchResults, taskId, existingTargetIds]
  )

  // Group dependencies by type
  const grouped = useMemo(() => {
    const groups: Record<DependencyType, TaskDependency[]> = {
      blocks: [],
      blocked_by: [],
      relates_to: [],
      duplicates: [],
    }
    for (const dep of dependencies) {
      const type = dep.dependency_type as DependencyType
      if (groups[type]) {
        groups[type].push(dep)
      }
    }
    return groups
  }, [dependencies])

  async function handleAddDependency(targetTaskId: string) {
    try {
      await createDep.mutateAsync({
        taskId,
        target_task_id: targetTaskId,
        dependency_type: addType,
      })
      setAddOpen(false)
      setSearchQuery('')
    } catch {
      // Error handled by mutation state
    }
  }

  async function handleRemoveDependency(depId: string) {
    try {
      await deleteDep.mutateAsync({ dependencyId: depId, taskId })
    } catch {
      // Error handled by mutation state
    }
  }

  function navigateToTask(depTaskId: string) {
    navigate(`/work/projects/${projectId}/tasks/${depTaskId}`)
  }

  if (isLoading) {
    return (
      <div className="py-2 text-center text-xs text-muted-foreground">
        {t('work.dependencies.loading')}
      </div>
    )
  }

  const hasAny = dependencies.length > 0

  return (
    <div>
      {/* Dependency groups */}
      {hasAny ? (
        <div className="space-y-3">
          {(Object.keys(depTypeConfig) as DependencyType[]).map((type) => {
            const deps = grouped[type]
            if (deps.length === 0) return null
            const config = depTypeConfig[type]
            const Icon = config.icon

            return (
              <div key={type}>
                <div className="flex items-center gap-1.5 mb-1">
                  <Icon className={cn('h-3.5 w-3.5', config.color)} />
                  <span className="text-xs font-medium text-muted-foreground">
                    {t(config.labelKey)} ({deps.length})
                  </span>
                </div>
                <div className="space-y-0.5 pl-5">
                  {deps.map((dep) => {
                    const linkedTaskId =
                      dep.target_task_id === taskId
                        ? dep.source_task_id
                        : dep.target_task_id

                    return (
                      <div
                        key={dep.id}
                        className="group flex items-center gap-2 rounded px-1 py-0.5 hover:bg-accent/50 transition-colors"
                      >
                        <button
                          type="button"
                          className="flex-1 text-left text-xs text-foreground hover:underline truncate"
                          onClick={() =>
                            linkedTaskId && navigateToTask(linkedTaskId)
                          }
                        >
                          {linkedTaskId?.slice(0, 8)}...
                        </button>
                        {canEdit && (
                          <button
                            type="button"
                            className="rounded p-0.5 text-muted-foreground opacity-0 group-hover:opacity-100 hover:text-destructive transition-all"
                            onClick={() => dep.id && handleRemoveDependency(dep.id)}
                            title={t('work.dependencies.remove')}
                          >
                            <X className="h-3 w-3" />
                          </button>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            )
          })}
        </div>
      ) : (
        <p className="text-xs text-muted-foreground text-center py-2">
          {t('work.dependencies.empty')}
        </p>
      )}

      {/* Add dependency */}
      {canEdit && (
      <Popover open={addOpen} onOpenChange={setAddOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            className="mt-2 w-full gap-1 text-xs"
          >
            <Plus className="h-3.5 w-3.5" />
            {t('work.dependencies.add')}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-72 p-3" align="start">
          <div className="space-y-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">
                {t('work.dependencies.type')}
              </label>
              <Select
                value={addType}
                onValueChange={(v) => setAddType(v as DependencyType)}
              >
                <SelectTrigger className="h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="blocks">{t('work.dependencies.blocks')}</SelectItem>
                  <SelectItem value="blocked_by">{t('work.dependencies.blockedBy')}</SelectItem>
                  <SelectItem value="relates_to">{t('work.dependencies.relatesTo')}</SelectItem>
                  <SelectItem value="duplicates">{t('work.dependencies.duplicateOf')}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">
                {t('work.dependencies.searchTask')}
              </label>
              <div className="relative">
                <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  className="h-8 pl-7 text-xs"
                  placeholder={t('work.dependencies.searchPlaceholder')}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  autoFocus
                />
              </div>
            </div>

            {/* Search results */}
            <div className="max-h-40 overflow-y-auto space-y-0.5">
              {filteredSearchResults.length === 0 ? (
                <p className="text-xs text-muted-foreground text-center py-2">
                  {searchQuery
                    ? t('work.tasks.noTasksFound')
                    : t('work.search.enterQuery')}
                </p>
              ) : (
                filteredSearchResults.map((task) => (
                  <button
                    key={task.id}
                    type="button"
                    className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent transition-colors text-left"
                    onClick={() => task.id && handleAddDependency(task.id)}
                    disabled={createDep.isPending}
                  >
                    {createDep.isPending ? (
                      <Loader2 className="h-3 w-3 shrink-0 animate-spin" />
                    ) : null}
                    <span className="font-mono text-muted-foreground shrink-0">
                      {task.project_key && task.task_number
                        ? `${task.project_key}-${task.task_number}`
                        : task.id?.slice(0, 8)}
                    </span>
                    <span className="truncate">{task.title}</span>
                  </button>
                ))
              )}
            </div>
          </div>
        </PopoverContent>
      </Popover>
      )}
    </div>
  )
}
