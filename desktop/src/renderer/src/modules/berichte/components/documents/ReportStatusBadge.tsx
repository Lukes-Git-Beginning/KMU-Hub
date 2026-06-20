import { useTranslation } from 'react-i18next'
import type { ReportDocStatus } from '@/api/berichte-types'

const STATUS_STYLE: Record<ReportDocStatus, string> = {
  draft: 'bg-secondary text-muted-foreground',
  final: 'bg-info-light text-info',
  released: 'bg-success-light text-success',
  archived: 'bg-secondary/60 text-muted-foreground/70',
}

const STATUS_KEY: Record<ReportDocStatus, string> = {
  draft: 'berichte.docs.status.draft',
  final: 'berichte.docs.status.final',
  released: 'berichte.docs.status.released',
  archived: 'berichte.docs.status.archived',
}

export function ReportStatusBadge({
  status,
  className = '',
}: {
  status: ReportDocStatus
  className?: string
}) {
  const { t } = useTranslation()
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${STATUS_STYLE[status]} ${className}`}
    >
      {t(STATUS_KEY[status])}
    </span>
  )
}
