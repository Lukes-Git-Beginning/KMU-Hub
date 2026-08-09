/**
 * RolloutDetailModal (Darien 2026-08-05) — a rollout/draft opened as a real thing,
 * not as a row with three icons.
 *
 * Before this, the list let you click only the buttons on the right ("da kann man
 * ja gar nicht auf den Entwurf selbst klicken"), the scheduling lived in the
 * editor's commit dialog, and the announcement lived nowhere at all. This is the
 * one place where a rollout is read and managed: what it changes, when it goes
 * out, and what the affected users get told.
 *
 * Cosmi convention: detail on row click is a centered modal (shared/DetailModal),
 * the whole row is the trigger, inner buttons stop propagation.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { CalendarClock, Megaphone, Zap, RotateCcw, PencilLine, Trash2, CalendarX } from 'lucide-react'
import { DetailModal } from '@/components/shared/DetailModal'
import { Button } from '@/components/ui/button'
import { i18n } from '@/i18n/i18n'
import type { CustomizationDraft, CustomizationDraftPayload, DraftStatus } from '@/api/customization-types'
import { COLUMN_AREA_PREFIX } from '@/components/customization/EditorSurface'

const STATUS_STYLE: Record<DraftStatus, string> = {
  draft: 'bg-muted text-muted-foreground',
  scheduled: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
  live: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  superseded: 'bg-muted text-muted-foreground/70',
}

/** Next morning at 06:00, formatted for <input type="datetime-local">. */
function defaultSchedule(): string {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  d.setHours(6, 0, 0, 0)
  const pad = (n: number): string => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** ISO → the value shape <input type="datetime-local"> expects (local time). */
function toLocalInput(iso: string): string {
  const d = new Date(iso)
  const pad = (n: number): string => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/**
 * What this rollout actually changes, in the user's terms. The area overlay is
 * split by prefix, because "3 Bereiche" for what are really column settings would
 * be worse than saying nothing.
 */
function describeChanges(
  payload: CustomizationDraftPayload,
  t: ReturnType<typeof useTranslation>['t'],
): string[] {
  const out: string[] = []
  const labelCount = Object.values(payload.labels).reduce((acc, m) => acc + Object.keys(m).length, 0)
  if (labelCount > 0) out.push(t('customization.rollout.changeTerms', { count: labelCount }))
  const setCount = Object.keys(payload.valueSets).length
  if (setCount > 0) out.push(t('customization.rollout.changeValueSets', { count: setCount }))
  const fieldEntities = Object.keys(payload.customFields ?? {}).length
  if (fieldEntities > 0) out.push(t('customization.rollout.changeFields', { count: fieldEntities }))

  let areas = 0
  let columns = 0
  let stats = 0
  for (const map of Object.values(payload.moduleAreas ?? {})) {
    for (const key of Object.keys(map)) {
      if (key.startsWith(COLUMN_AREA_PREFIX)) columns += 1
      else if (key.startsWith('stat:')) stats += 1
      else areas += 1
    }
  }
  if (areas > 0) out.push(t('customization.rollout.changeAreas', { count: areas }))
  if (columns > 0) out.push(t('customization.rollout.changeColumns', { count: columns }))
  if (stats > 0) out.push(t('customization.rollout.changeStats', { count: stats }))
  return out
}

export interface RolloutDetailModalProps {
  draft: CustomizationDraft
  moduleName: string
  canRollback: boolean
  onClose: () => void
  onContinue: () => void
  onDeployNow: () => void
  onSchedule: (scheduledAt: string) => void
  onUnschedule: () => void
  onAnnouncementChange: (text: string) => void
  /** Rename the draft/rollout — same record, new label in the list. */
  onRename: (name: string) => void
  onRollback: () => void
  onDelete: () => void
}

export function RolloutDetailModal({
  draft,
  moduleName,
  canRollback,
  onClose,
  onContinue,
  onDeployNow,
  onSchedule,
  onUnschedule,
  onAnnouncementChange,
  onRename,
  onRollback,
  onDelete,
}: RolloutDetailModalProps): React.ReactElement {
  const { t } = useTranslation()
  const [announcement, setAnnouncement] = useState(draft.announcement ?? '')
  const [scheduledAt, setScheduledAt] = useState(
    draft.scheduledAt ? toLocalInput(draft.scheduledAt) : defaultSchedule(),
  )

  const pending = draft.status === 'draft' || draft.status === 'scheduled'
  const fmt = (iso?: string): string =>
    iso ? new Date(iso).toLocaleString(i18n.language, { dateStyle: 'medium', timeStyle: 'short' }) : '—'

  const changes = describeChanges(draft.payload, t)

  const saveAnnouncement = (): void => {
    if (announcement === (draft.announcement ?? '')) return
    onAnnouncementChange(announcement)
    toast.success(t('customization.rollout.announcementSaved'))
  }

  const footer = (
    <div className="flex flex-wrap items-center justify-between gap-2">
      <div className="flex items-center gap-2">
        {pending && (
          <Button variant="ghost" size="sm" onClick={onDelete}>
            <Trash2 className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
            {t('common.delete')}
          </Button>
        )}
        {draft.status === 'live' && canRollback && (
          <Button variant="ghost" size="sm" onClick={onRollback}>
            <RotateCcw className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
            {t('customization.editor.rollouts.rollback')}
          </Button>
        )}
      </div>
      <div className="flex items-center gap-2">
        {pending && (
          <Button variant="outline" size="sm" onClick={onContinue}>
            <PencilLine className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
            {t('customization.rollout.continue')}
          </Button>
        )}
        {pending && (
          <Button size="sm" onClick={onDeployNow}>
            <Zap className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
            {t('customization.rollout.deployNow')}
          </Button>
        )}
        {!pending && (
          <Button variant="outline" size="sm" onClick={onClose}>
            {t('common.close')}
          </Button>
        )}
      </div>
    </div>
  )

  return (
    <DetailModal open onClose={onClose} title={draft.name} maxWidth="max-w-lg" footer={footer}>
      <div className="flex flex-col gap-4">
        {/* Rename — a name given once at save time must stay changeable, otherwise
            a hasty "Helpdesk — 9. Aug." sticks around forever. */}
        <label className="flex flex-col gap-1.5">
          <span className="text-xs font-medium text-muted-foreground">
            {t('customization.editor.rollouts.rename')}
          </span>
          <input
            defaultValue={draft.name}
            onBlur={(e) => {
              const next = e.target.value.trim()
              if (next && next !== draft.name) onRename(next)
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
            }}
            aria-label={t('customization.editor.rollouts.rename')}
            className="h-9 w-full rounded-md border border-border bg-background px-2.5 text-sm outline-none focus:border-primary"
          />
        </label>

        {/* Facts */}
        <div className="flex flex-col gap-1.5 rounded-lg border bg-muted/30 px-3 py-2.5 text-sm">
          <Row label={t('customization.rollout.module')} value={moduleName} />
          <Row
            label={t('customization.rollout.status')}
            value={
              <span className={`rounded px-1.5 py-0.5 text-[11px] font-medium ${STATUS_STYLE[draft.status]}`}>
                {t(`customization.editor.rollouts.status.${draft.status}`)}
              </span>
            }
          />
          <Row label={t('customization.rollout.updated')} value={fmt(draft.updatedAt)} />
          {draft.scheduledAt && (
            <Row label={t('customization.rollout.scheduledAt')} value={fmt(draft.scheduledAt)} />
          )}
        </div>

        {/* What it changes — the reason a rollout is worth opening at all. */}
        <section className="flex flex-col gap-1.5">
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {t('customization.rollout.changesTitle')}
          </h4>
          {changes.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('customization.rollout.changesNone')}</p>
          ) : (
            <ul className="flex flex-wrap gap-1.5">
              {changes.map((c) => (
                <li key={c} className="rounded-md bg-secondary px-2 py-1 text-xs text-foreground">
                  {c}
                </li>
              ))}
            </ul>
          )}
        </section>

        {/* Schedule — only meaningful while the rollout has not gone out. */}
        {pending && (
          <section className="flex flex-col gap-1.5">
            <h4 className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              <CalendarClock className="h-3.5 w-3.5" aria-hidden="true" />
              {t('customization.rollout.scheduleTitle')}
            </h4>
            <div className="flex flex-wrap items-center gap-2">
              <input
                type="datetime-local"
                value={scheduledAt}
                onChange={(e) => setScheduledAt(e.target.value)}
                className="h-9 min-w-[13rem] flex-1 rounded-md border border-border bg-background px-2.5 text-sm outline-none focus:border-primary"
              />
              <Button
                variant="outline"
                size="sm"
                onClick={() => onSchedule(new Date(scheduledAt).toISOString())}
              >
                {draft.status === 'scheduled'
                  ? t('customization.rollout.rescheduleAction')
                  : t('customization.rollout.scheduleAction')}
              </Button>
              {draft.status === 'scheduled' && (
                <Button variant="ghost" size="sm" onClick={onUnschedule}>
                  <CalendarX className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                  {t('customization.rollout.unschedule')}
                </Button>
              )}
            </div>
            <p className="text-[11px] leading-relaxed text-muted-foreground">
              {t('customization.rollout.scheduleHint')}
            </p>
          </section>
        )}

        {/* Announcement — with the answer to "who sees this, and when?" */}
        <section className="flex flex-col gap-1.5">
          <h4 className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            <Megaphone className="h-3.5 w-3.5" aria-hidden="true" />
            {t('customization.rollout.announcementTitle')}
          </h4>
          <textarea
            value={announcement}
            onChange={(e) => setAnnouncement(e.target.value)}
            onBlur={saveAnnouncement}
            placeholder={t('customization.editor.deploy.announcePlaceholder')}
            rows={2}
            className="resize-none rounded-md border border-border bg-background px-2.5 py-1.5 text-sm outline-none focus:border-primary"
          />
          <p className="text-[11px] leading-relaxed text-muted-foreground">
            {t('customization.rollout.announcementHint', { module: moduleName })}
          </p>
        </section>
      </div>
    </DetailModal>
  )
}

function Row({ label, value }: { label: string; value: React.ReactNode }): React.ReactElement {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate text-right text-sm text-foreground">{value}</span>
    </div>
  )
}
