/**
 * RolloutAnnouncement (Darien 2026-08-05) — where a rollout's message actually
 * lands.
 *
 * The commit dialog has always offered an "optional message", but it went
 * nowhere: it was not stored on the record and no user ever saw it ("man hat ja
 * gar keine Ahnung wie wo was diese Nachricht deployed wird"). This is the
 * receiving end — the message of the LIVE rollout for this module, shown once per
 * user at the top of that module, dismissible.
 *
 * Deliberately module-local rather than a global notification: the message
 * explains a change to THIS module, and it is most useful standing next to the
 * thing that changed. Dismissal is remembered per rollout id, so a later rollout
 * with a new message shows again.
 *
 * Silent (renders null) when there is no message, when it was dismissed, and
 * inside the editor sandbox — there the module is being configured, not used.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { Megaphone, X } from 'lucide-react'
import { listDrafts } from '@/mocks/data/customization-drafts'
import { useEditorSurface } from './EditorSurface'

const SEEN_KEY = 'cosmi:customization:seen-announcements'

function readSeen(): string[] {
  try {
    const raw = localStorage.getItem(SEEN_KEY)
    return raw ? (JSON.parse(raw) as string[]) : []
  } catch {
    return []
  }
}

function markSeen(id: string): void {
  try {
    const next = Array.from(new Set([...readSeen(), id])).slice(-50)
    localStorage.setItem(SEEN_KEY, JSON.stringify(next))
  } catch {
    // Nothing to do: without storage the note simply reappears next time.
  }
}

export function RolloutAnnouncement({ moduleId }: { moduleId: string }): React.ReactElement | null {
  const { t } = useTranslation()
  const { editing } = useEditorSurface()
  const [seen, setSeen] = useState<string[]>(readSeen)

  const { data: drafts = [] } = useQuery({
    queryKey: ['customization', 'drafts', 'announcement', moduleId],
    queryFn: () => listDrafts(moduleId),
    staleTime: 0,
  })

  const live = drafts.find((d) => d.status === 'live' && d.announcement)
  if (editing || !live || seen.includes(live.id)) return null

  const dismiss = (): void => {
    markSeen(live.id)
    setSeen(readSeen())
  }

  return (
    <div className="mb-4 flex items-start gap-2.5 rounded-lg border border-[var(--accent-1)]/30 bg-[var(--accent-1)]/5 px-3.5 py-2.5">
      <Megaphone className="mt-0.5 h-4 w-4 shrink-0 text-[var(--accent-1)]" aria-hidden="true" />
      <div className="min-w-0 flex-1">
        <p className="text-xs font-medium text-foreground">{t('customization.announcement.title')}</p>
        <p className="mt-0.5 text-sm leading-relaxed text-muted-foreground">{live.announcement}</p>
      </div>
      <button
        type="button"
        onClick={dismiss}
        aria-label={t('customization.announcement.dismiss')}
        title={t('customization.announcement.dismiss')}
        className="shrink-0 rounded-md p-1 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
      >
        <X className="h-3.5 w-3.5" aria-hidden="true" />
      </button>
    </div>
  )
}
