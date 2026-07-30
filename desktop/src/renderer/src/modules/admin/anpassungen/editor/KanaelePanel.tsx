/**
 * KanaelePanel (Modul-Editor, Ticket-Intake P6+) — the intake-channel editor in
 * the properties panel. Toggles the three ways a ticket can be created (agent /
 * self-service / external) and binds ONE ticket form per channel — so agents can
 * get a different template than self-service submitters (forms may also be
 * shared). Each channel gets its own form picker + "edit form →" deep-link; a
 * "new template" button duplicates the seed intake form and opens it.
 *
 * Unlike the trio dimensions (labels/value-sets/fields/areas), channels are a
 * FUNCTIONAL tenant toggle, not a content overlay — so this writes directly to
 * the helpdesk store (like csatEnabled), applied immediately, no draft/deploy.
 * The sandbox preview (same window, shared store) reflects the change live.
 *
 * Editing a form opens the real form editor (Formulare module) at
 * `/formulare?edit=<id>` — the editor window shares the app's hash router, so the
 * concrete form opens directly instead of just landing on the overview.
 */
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Inbox, User, Building2, Globe, PencilLine, Plus } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { toast } from 'sonner'
import { useHelpdeskStore } from '@/stores/helpdesk'
import { useFormSchemas, useDuplicateFormSchema } from '@/api/hooks/useFormulare'
import { getEditorModule } from './editorModules'

type ChannelKey = 'agent' | 'selfservice' | 'external'

const CHANNELS: { key: ChannelKey; icon: LucideIcon; labelKey: string; descKey: string }[] = [
  { key: 'agent', icon: User, labelKey: 'customization.editor.channels.agent', descKey: 'customization.editor.channels.agentDesc' },
  { key: 'selfservice', icon: Building2, labelKey: 'customization.editor.channels.selfservice', descKey: 'customization.editor.channels.selfserviceDesc' },
  { key: 'external', icon: Globe, labelKey: 'customization.editor.channels.external', descKey: 'customization.editor.channels.externalDesc' },
]

const SEED_TICKET_FORM = 'tmpl-ticket-intake'

export function KanaelePanel({ moduleKey }: { moduleKey: string }): React.ReactElement {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const channels = useHelpdeskStore((s) => s.intakeChannels)
  const setChannel = useHelpdeskStore((s) => s.setIntakeChannel)
  const intakeForms = useHelpdeskStore((s) => s.intakeForms)
  const setIntakeForm = useHelpdeskStore((s) => s.setIntakeForm)
  const { data } = useFormSchemas()
  const duplicate = useDuplicateFormSchema()

  const module = getEditorModule(moduleKey)
  if (!module?.intake) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-muted text-muted-foreground">
          <Inbox className="h-5 w-5" aria-hidden="true" />
        </div>
        <p className="text-sm text-muted-foreground">{t('customization.editor.channels.empty')}</p>
      </div>
    )
  }

  // Forms that feed a Helpdesk ticket (bound via intakeTargetId) — the choices
  // each channel can render.
  const ticketForms = (data?.items ?? []).filter((s) => s.intakeTargetId === 'helpdesk_ticket')

  // Open the concrete bound form in the real form editor (deep-link, shared router).
  const editForm = (id: string): void => {
    if (!id) return
    navigate(`/formulare?edit=${encodeURIComponent(id)}`)
  }

  // Create a new ticket form by duplicating the seed intake template, then open it.
  const createForm = async (): Promise<void> => {
    try {
      const created = await duplicate.mutateAsync({
        id: SEED_TICKET_FORM,
        title: t('customization.editor.channels.newFormTitle'),
      })
      if (created?.id) navigate(`/formulare?edit=${encodeURIComponent(created.id)}`)
    } catch {
      toast.error(t('customization.editor.channels.newFormError'))
    }
  }

  return (
    <div className="flex flex-1 flex-col gap-3 overflow-y-auto px-4 py-3">
      <p className="px-0.5 text-xs leading-relaxed text-muted-foreground">
        {t('customization.editor.channels.hint')}
      </p>

      {CHANNELS.map(({ key, icon: Icon, labelKey, descKey }) => {
        const enabled = channels[key]
        const boundId = intakeForms[key]
        const bound = ticketForms.find((s) => s.id === boundId)
        return (
          <div
            key={key}
            className={`rounded-lg border px-3 py-2.5 ${enabled ? 'bg-card' : 'bg-muted/40'}`}
          >
            <div className="flex items-center gap-2.5">
              <Icon className={`h-4 w-4 shrink-0 ${enabled ? 'text-foreground' : 'text-muted-foreground'}`} aria-hidden="true" />
              <span className={`min-w-0 flex-1 truncate text-sm ${enabled ? 'text-foreground' : 'text-muted-foreground'}`}>
                {t(labelKey)}
              </span>
              <button
                type="button"
                role="switch"
                aria-checked={enabled}
                onClick={() => setChannel(key, !enabled)}
                aria-label={t(labelKey)}
                className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors ${enabled ? 'bg-emerald-500' : 'bg-secondary'}`}
              >
                <span className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${enabled ? 'translate-x-4.5' : 'translate-x-0.5'}`} />
              </button>
            </div>
            <p className="mt-1 pl-6 text-xs leading-relaxed text-muted-foreground">{t(descKey)}</p>
            {key === 'external' && enabled && (
              <p className="mt-1 pl-6 text-[11px] leading-relaxed text-amber-600 dark:text-amber-400">
                {t('customization.editor.channels.externalPending')}
              </p>
            )}

            {/* Per-channel form binding — one ticket form per channel. */}
            {enabled && (
              <div className="mt-2.5 pl-6">
                <label className="text-[11px] font-medium text-muted-foreground">
                  {t('customization.editor.channels.formLabel')}
                </label>
                <div className="mt-1 flex items-center gap-1.5">
                  <select
                    value={boundId}
                    onChange={(e) => setIntakeForm(key, e.target.value)}
                    className="min-w-0 flex-1 rounded-lg border border-border bg-background px-2.5 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  >
                    {!bound && <option value={boundId}>{t('customization.editor.channels.formNone')}</option>}
                    {ticketForms.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.title}
                      </option>
                    ))}
                  </select>
                  <button
                    type="button"
                    onClick={() => editForm(boundId)}
                    disabled={!bound}
                    title={t('customization.editor.channels.editForm')}
                    aria-label={t('customization.editor.channels.editForm')}
                    className="inline-flex shrink-0 items-center gap-1 rounded-lg border border-border px-2 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-secondary disabled:opacity-40"
                  >
                    <PencilLine className="h-3.5 w-3.5" aria-hidden="true" />
                  </button>
                </div>
              </div>
            )}
          </div>
        )
      })}

      <button
        type="button"
        onClick={createForm}
        disabled={duplicate.isPending}
        className="mt-1 inline-flex items-center justify-center gap-1.5 rounded-lg border border-dashed border-border px-2.5 py-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground disabled:opacity-50"
      >
        <Plus className="h-3.5 w-3.5" aria-hidden="true" />
        {t('customization.editor.channels.newForm')}
      </button>
    </div>
  )
}
