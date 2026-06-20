import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CheckSquare, Search } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/shared'
import type { ReportDocument } from '@/api/berichte-types'
import { useAttachReportToTask } from '@/api/hooks/useBerichte'
import { useTasks } from '@/api/hooks/useTasks'

interface AttachToTaskDialogProps {
  doc: ReportDocument
  open: boolean
  onClose: () => void
}

/**
 * R-5a — attach the report as a reference to a work task. The report is linked
 * (not copied): it shows up in the task's file list as a "Bericht"-entry.
 */
export function AttachToTaskDialog({ doc, open, onClose }: AttachToTaskDialogProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useTasks({ page_size: 200 })
  const attachMutation = useAttachReportToTask()
  const [query, setQuery] = useState('')

  const tasks = data?.tasks ?? []
  const matches = useMemo(() => {
    const q = query.trim().toLowerCase()
    const list = q ? tasks.filter((tk) => (tk.title ?? '').toLowerCase().includes(q)) : tasks
    return list.slice(0, 8)
  }, [tasks, query])

  const handleAttach = (taskId: string, taskTitle: string) => {
    attachMutation.mutate(
      { taskId, doc: { id: doc.id, title: doc.title } },
      {
        onSuccess: () => {
          toast.success(t('berichte.docs.attach.taskDone', { task: taskTitle }))
          onClose()
        },
        onError: (err) => toast.error((err as Error).message),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-md gap-0 p-0">
        <DialogHeader className="border-b border-border px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
              <CheckSquare className="h-4 w-4 text-primary" />
            </div>
            <div className="min-w-0">
              <DialogTitle className="text-sm">{t('berichte.docs.attach.taskTitle')}</DialogTitle>
              <DialogDescription className="truncate">{doc.title}</DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="px-6 py-5">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={t('berichte.docs.attach.searchTask')}
              className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          <div className="mt-3 max-h-72 space-y-1 overflow-y-auto">
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full rounded-lg" />
              ))
            ) : matches.length === 0 ? (
              <p className="px-1 py-6 text-center text-sm text-muted-foreground">
                {t('berichte.docs.attach.noTasks')}
              </p>
            ) : (
              matches.map((tk) => (
                <button
                  key={tk.id}
                  type="button"
                  disabled={attachMutation.isPending}
                  onClick={() => handleAttach(tk.id as string, tk.title ?? '')}
                  className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left transition-colors hover:bg-secondary disabled:opacity-50"
                >
                  <CheckSquare className="h-4 w-4 shrink-0 text-muted-foreground" />
                  <span className="min-w-0 flex-1 truncate text-sm text-foreground">
                    {tk.title}
                  </span>
                </button>
              ))
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
