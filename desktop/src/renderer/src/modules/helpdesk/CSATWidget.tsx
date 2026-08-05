/**
 * Customer Satisfaction (CSAT) rating widget.
 *
 * Star rating (1-5) with optional comment for resolved/closed tickets.
 * Aggregate display for statistics tab.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Star, MessageSquare } from 'lucide-react'
import { toast } from 'sonner'
import { useTickets, useSaveCsat } from '@/api/hooks/useHelpdesk'
import { useHelpdeskStore } from '@/stores/helpdesk'

function StarRating({ value, onChange, size = 'md', readOnly }: {
  value: number; onChange?: (r: number) => void; size?: 'sm' | 'md'; readOnly?: boolean
}) {
  const [hover, setHover] = useState(0)
  const sz = size === 'sm' ? 'h-3.5 w-3.5' : 'h-5 w-5'

  return (
    <div className="flex items-center gap-0.5">
      {[1, 2, 3, 4, 5].map((s) => (
        <button
          key={s}
          onClick={() => !readOnly && onChange?.(s)}
          onMouseEnter={() => !readOnly && setHover(s)}
          onMouseLeave={() => !readOnly && setHover(0)}
          disabled={readOnly}
          className={readOnly ? 'cursor-default' : 'cursor-pointer'}
        >
          <Star className={`${sz} ${s <= (hover || value) ? 'fill-amber-400 text-amber-400' : 'fill-none text-gray-300 dark:text-gray-600'}`} />
        </button>
      ))}
    </div>
  )
}

interface CSATWidgetProps {
  ticketId: string
  ticketStatus: string
  /** Existing rating from the wire ticket (via the adapter). */
  csatRating?: number
  csatComment?: string
}

export function CSATWidget({ ticketId, ticketStatus, csatRating, csatComment }: CSATWidgetProps) {
  const { t } = useTranslation()
  const saveCsat = useSaveCsat()
  const csatQuestion = useHelpdeskStore((s) => s.csatQuestion)
  const hasRating = typeof csatRating === 'number'
  const [rating, setRating] = useState(csatRating ?? 0)
  const [comment, setComment] = useState(csatComment ?? '')
  const [submitted, setSubmitted] = useState(hasRating)

  if (ticketStatus !== 'resolved' && ticketStatus !== 'closed') return null

  const handleSubmit = () => {
    if (rating === 0) { toast.error(t('helpdesk.csat.selectRating')); return }
    saveCsat.mutate(
      { ticketId, rating, comment: comment.trim() || undefined },
      {
        onSuccess: () => {
          setSubmitted(true)
          toast.success(t('helpdesk.csat.saved'))
        },
        onError: () => toast.error(t('helpdesk.csat.saveError')),
      },
    )
  }

  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <h4 className="text-xs font-medium text-muted-foreground mb-3 flex items-center gap-1.5">
        <Star className="h-3.5 w-3.5" />
        {t('helpdesk.csat.customerFeedback')}
      </h4>
      {submitted ? (
        <div>
          <div className="flex items-center gap-2 mb-1">
            <StarRating value={rating} readOnly size="sm" />
            <span className="text-sm font-medium text-foreground">{rating}/5</span>
          </div>
          {comment && (
            <p className="text-xs text-muted-foreground mt-1.5 flex items-start gap-1">
              <MessageSquare className="h-3 w-3 mt-0.5 shrink-0" />
              {comment}
            </p>
          )}
        </div>
      ) : (
        <div className="space-y-3">
          {csatQuestion.trim() && <p className="text-sm text-foreground">{csatQuestion}</p>}
          <StarRating value={rating} onChange={setRating} />
          <textarea
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            placeholder={t('helpdesk.csat.commentPlaceholder')}
            rows={2}
            className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
          />
          <button onClick={handleSubmit} disabled={saveCsat.isPending} className="rounded-lg bg-primary px-3 py-1.5 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50">
            {t('helpdesk.csat.submit')}
          </button>
        </div>
      )}
    </div>
  )
}

export function CSATAggregate() {
  const { t } = useTranslation()
  const { data } = useTickets()
  const ratings = useMemo(
    () =>
      (data?.tickets ?? [])
        .filter((x) => typeof x.csat_rating === 'number')
        .map((x) => x.csat_rating as number),
    [data],
  )
  if (ratings.length === 0) return null
  const avg = ratings.reduce((s, r) => s + r, 0) / ratings.length
  const dist = [5, 4, 3, 2, 1].map((s) => ({ stars: s, count: ratings.filter((r) => r === s).length }))

  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <h3 className="text-sm font-medium text-foreground mb-3">{t('helpdesk.csat.satisfaction')}</h3>
      <div className="flex items-center gap-3 mb-4">
        <span className="text-3xl font-semibold text-foreground">{avg.toFixed(1)}</span>
        <div>
          <StarRating value={Math.round(avg)} readOnly size="sm" />
          <p className="text-xs text-muted-foreground mt-0.5">{t('helpdesk.csat.ratingsCount', { count: ratings.length })}</p>
        </div>
      </div>
      <div className="space-y-1.5">
        {dist.map((d) => (
          <div key={d.stars} className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground w-6 text-right">{d.stars}</span>
            <Star className="h-3 w-3 fill-amber-400 text-amber-400" />
            <div className="flex-1 h-2 rounded-full bg-secondary overflow-hidden">
              <div className="h-full rounded-full bg-amber-400 transition-all" style={{ width: `${(d.count / ratings.length) * 100}%` }} />
            </div>
            <span className="text-xs text-muted-foreground w-6">{d.count}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
