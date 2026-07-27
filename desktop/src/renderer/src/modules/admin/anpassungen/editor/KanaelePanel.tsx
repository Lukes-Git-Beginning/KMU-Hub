/**
 * KanaelePanel (Modul-Editor, Ticket-Intake P6) — the intake-channel editor in
 * the properties panel. Toggles the three ways a ticket can be created (agent /
 * self-service / external) and binds the form that the self-service + external
 * channels render.
 *
 * Unlike the trio dimensions (labels/value-sets/fields/areas), channels are a
 * FUNCTIONAL tenant toggle, not a content overlay — so this writes directly to
 * the helpdesk store (like csatEnabled), applied immediately, no draft/deploy.
 * The sandbox preview (same window, shared store) reflects the change live.
 *
 * The full form-builder embed (§7) is a later refinement — for now field/role
 * editing happens in the bound intake form (Formulare module), reachable via
 * "Formular bearbeiten →".
 */
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Inbox, User, Building2, Globe, ExternalLink, ArrowRight } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { useHelpdeskStore } from '@/stores/helpdesk'
import { useFormSchemas } from '@/api/hooks/useFormulare'
import { getEditorModule } from './editorModules'

type ChannelKey = 'agent' | 'selfservice' | 'external'

const CHANNELS: { key: ChannelKey; icon: LucideIcon; labelKey: string; descKey: string }[] = [
  { key: 'agent', icon: User, labelKey: 'customization.editor.channels.agent', descKey: 'customization.editor.channels.agentDesc' },
  { key: 'selfservice', icon: Building2, labelKey: 'customization.editor.channels.selfservice', descKey: 'customization.editor.channels.selfserviceDesc' },
  { key: 'external', icon: Globe, labelKey: 'customization.editor.channels.external', descKey: 'customization.editor.channels.externalDesc' },
]

export function KanaelePanel({ moduleKey }: { moduleKey: string }): React.ReactElement {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const channels = useHelpdeskStore((s) => s.intakeChannels)
  const setChannel = useHelpdeskStore((s) => s.setIntakeChannel)
  const intakeFormId = useHelpdeskStore((s) => s.intakeFormId)
  const setIntakeFormId = useHelpdeskStore((s) => s.setIntakeFormId)
  const { data } = useFormSchemas()

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
  // the self-service + external channels can render.
  const ticketForms = (data?.items ?? []).filter((s) => s.intakeTargetId === 'helpdesk_ticket')

  return (
    <div className="flex flex-1 flex-col gap-3 overflow-y-auto px-4 py-3">
      <p className="px-0.5 text-xs leading-relaxed text-muted-foreground">
        {t('customization.editor.channels.hint')}
      </p>

      {/* Channel on/off toggles */}
      <div className="flex flex-col gap-2">
        {CHANNELS.map(({ key, icon: Icon, labelKey, descKey }) => {
          const enabled = channels[key]
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
            </div>
          )
        })}
      </div>

      {/* Bound intake form */}
      <div className="mt-1 rounded-lg border bg-card px-3 py-3">
        <label className="text-xs font-medium text-foreground">
          {t('customization.editor.channels.formLabel')}
        </label>
        <select
          value={intakeFormId}
          onChange={(e) => setIntakeFormId(e.target.value)}
          className="mt-1.5 w-full rounded-lg border border-border bg-background px-2.5 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
        >
          {ticketForms.length === 0 && (
            <option value={intakeFormId}>{t('customization.editor.channels.formNone')}</option>
          )}
          {ticketForms.map((s) => (
            <option key={s.id} value={s.id}>
              {s.title}
            </option>
          ))}
        </select>
        <p className="mt-1.5 text-[11px] leading-relaxed text-muted-foreground">
          {t('customization.editor.channels.formHint')}
        </p>
        <button
          type="button"
          onClick={() => navigate('/formulare')}
          className="mt-2.5 inline-flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-secondary"
        >
          <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
          {t('customization.editor.channels.editForm')}
          <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      </div>
    </div>
  )
}
