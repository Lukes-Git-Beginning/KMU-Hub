/**
 * DeployDialog (Modul-Editor v1, E-5) — the "Übernehmen" flow: apply the draft
 * now, schedule a tenant-wide rollout for a future day, or keep it as a draft.
 *
 * Scheduled deployment is Cosmi's differentiator (no major competitor schedules
 * config rollouts natively — DRAFT-DEPLOY.md). The rollout job is mocked
 * (runDueScheduledDeploys); Luke's cron makes it real. "Jetzt" also broadcasts
 * the payload so the main window converges live (customization-sync).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Zap, CalendarClock, Users } from 'lucide-react'
import { DetailModal } from '@/components/shared/DetailModal'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib'
import { deployDraft, getDeploySnapshot } from '@/mocks/data/customization-drafts'
import { publishCustomizationDeploy, publishDraftMirror } from './customization-sync'
import type { CustomizationDraftPayload } from '@/api/customization-types'

/** Next morning at 06:00, formatted for <input type="datetime-local">. */
function defaultSchedule(): string {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  d.setHours(6, 0, 0, 0)
  const pad = (n: number): string => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

interface DeployDialogProps {
  open: boolean
  onClose: () => void
  /** Set when the editor continues a saved draft — deploy updates THAT record. */
  draftId?: string
  moduleKey: string
  draftName: string
  changeCount: number
  buildPayload: () => CustomizationDraftPayload
  /** Called after a successful deploy/schedule/save — closes the editor window. */
  onDeployed: () => void
}

export function DeployDialog({
  open,
  onClose,
  draftId,
  moduleKey,
  draftName,
  changeCount,
  buildPayload,
  onDeployed,
}: DeployDialogProps): React.ReactElement {
  const { t } = useTranslation()
  const [mode, setMode] = useState<'now' | 'scheduled'>('now')
  const [scheduledAt, setScheduledAt] = useState<string>(defaultSchedule)
  const [announcement, setAnnouncement] = useState('')

  const confirm = (): void => {
    const payload = buildPayload()
    if (mode === 'scheduled') {
      const iso = new Date(scheduledAt).toISOString()
      const d = deployDraft({ id: draftId, moduleKey, name: draftName, payload, mode: 'scheduled', scheduledAt: iso, announcement: announcement || undefined })
      publishDraftMirror(d)
      toast.success(t('customization.editor.deploy.toastScheduled'))
    } else {
      const d = deployDraft({ id: draftId, moduleKey, name: draftName, payload, mode: 'now', announcement: announcement || undefined })
      publishCustomizationDeploy(payload, d, getDeploySnapshot(d.id))
      toast.success(t('customization.editor.toast.applied'))
    }
    onDeployed()
  }

  const saveDraft = (): void => {
    const d = deployDraft({ id: draftId, moduleKey, name: draftName, payload: buildPayload(), mode: 'draft' })
    publishDraftMirror(d)
    toast.success(t('customization.editor.toast.draftSaved'))
    onDeployed()
  }

  const modeButton = (
    value: 'now' | 'scheduled',
    icon: React.ReactNode,
    label: string,
    hint: string,
  ): React.ReactElement => (
    <button
      type="button"
      onClick={() => setMode(value)}
      aria-pressed={mode === value}
      className={cn(
        'flex flex-1 flex-col items-start gap-1 rounded-lg border px-3 py-2.5 text-left transition-colors',
        mode === value
          ? 'border-[var(--accent-1)] bg-[var(--accent-1)]/5'
          : 'border-border hover:border-border/70 hover:bg-accent/30',
      )}
    >
      <span className="flex items-center gap-1.5 text-sm font-medium text-foreground">
        {icon}
        {label}
      </span>
      <span className="text-[11px] leading-tight text-muted-foreground">{hint}</span>
    </button>
  )

  const footer = (
    <div className="flex items-center justify-between gap-2">
      <Button variant="ghost" size="sm" onClick={saveDraft}>
        {t('customization.editor.footer.saveDraft')}
      </Button>
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={onClose}>
          {t('customization.editor.deploy.cancel')}
        </Button>
        <Button size="sm" onClick={confirm}>
          {mode === 'scheduled'
            ? t('customization.editor.deploy.confirmScheduled')
            : t('customization.editor.deploy.confirmNow')}
        </Button>
      </div>
    </div>
  )

  return (
    <DetailModal
      open={open}
      onClose={onClose}
      title={t('customization.editor.deploy.title')}
      maxWidth="max-w-md"
      footer={footer}
    >
      <div className="flex flex-col gap-4">
        {/* Summary */}
        <div className="rounded-lg border bg-muted/30 px-3 py-2.5 text-sm">
          <p className="font-medium text-foreground">
            {t('customization.editor.footer.changes', { count: changeCount })}
          </p>
          <p className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
            <Users className="h-3.5 w-3.5" aria-hidden="true" />
            {t('customization.editor.deploy.affects')}
          </p>
        </div>

        {/* Mode */}
        <div className="flex gap-2">
          {modeButton('now', <Zap className="h-3.5 w-3.5" aria-hidden="true" />, t('customization.editor.deploy.modeNow'), t('customization.editor.deploy.modeNowHint'))}
          {modeButton('scheduled', <CalendarClock className="h-3.5 w-3.5" aria-hidden="true" />, t('customization.editor.deploy.modeScheduled'), t('customization.editor.deploy.modeScheduledHint'))}
        </div>

        {mode === 'scheduled' && (
          <label className="flex flex-col gap-1.5 text-xs font-medium text-foreground">
            {t('customization.editor.deploy.scheduleLabel')}
            <input
              type="datetime-local"
              value={scheduledAt}
              onChange={(e) => setScheduledAt(e.target.value)}
              className="h-9 rounded-md border border-border bg-background px-2.5 text-sm outline-none focus:border-primary"
            />
          </label>
        )}

        {/* Optional announcement */}
        <label className="flex flex-col gap-1.5 text-xs font-medium text-foreground">
          {t('customization.editor.deploy.announceLabel')}
          <textarea
            value={announcement}
            onChange={(e) => setAnnouncement(e.target.value)}
            placeholder={t('customization.editor.deploy.announcePlaceholder')}
            rows={2}
            className="resize-none rounded-md border border-border bg-background px-2.5 py-1.5 text-sm outline-none focus:border-primary"
          />
        </label>
      </div>
    </DetailModal>
  )
}
