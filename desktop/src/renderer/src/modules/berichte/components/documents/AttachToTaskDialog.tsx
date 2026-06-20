import { useTranslation } from 'react-i18next'
import { CheckSquare } from 'lucide-react'
import { toast } from 'sonner'
import type { ReportDocument } from '@/api/berichte-types'
import { useAttachReportToTask } from '@/api/hooks/useBerichte'
import { useTasks } from '@/api/hooks/useTasks'
import { AttachReportDialog, type AttachItem } from './AttachReportDialog'

interface AttachToTaskDialogProps {
  doc: ReportDocument
  open: boolean
  onClose: () => void
}

/** R-5a — attach the report as a reference to a work task (file-list entry). */
export function AttachToTaskDialog({ doc, open, onClose }: AttachToTaskDialogProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useTasks({ page_size: 200 })
  const attach = useAttachReportToTask()

  const items: AttachItem[] = (data?.tasks ?? []).map((tk) => ({
    id: tk.id as string,
    label: tk.title ?? '',
  }))

  return (
    <AttachReportDialog
      open={open}
      onClose={onClose}
      subtitle={doc.title}
      icon={CheckSquare}
      title={t('berichte.docs.attach.taskTitle')}
      searchPlaceholder={t('berichte.docs.attach.searchTask')}
      emptyLabel={t('berichte.docs.attach.noTasks')}
      items={items}
      isLoading={isLoading}
      pending={attach.isPending}
      onPick={(item) =>
        attach.mutate(
          { taskId: item.id, doc: { id: doc.id, title: doc.title } },
          {
            onSuccess: () => {
              toast.success(t('berichte.docs.attach.taskDone', { task: item.label }))
              onClose()
            },
            onError: (err) => toast.error((err as Error).message),
          },
        )
      }
    />
  )
}
