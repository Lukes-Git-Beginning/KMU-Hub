/**
 * Outgoing webhook configuration (tenant scope, mock-shell, Phase 5).
 *
 * No webhook backend for the inbox exists yet (see backend-gaps.md). This is a
 * verifiable-looking shell: add/remove webhook endpoints (URL + event) held in
 * local state. Wiring later = swap local state for a webhook CRUD hook.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, Webhook } from 'lucide-react'

interface WebhookEntry {
  id: string
  url: string
  event: string
}

const EVENTS = ['message.received', 'message.assigned', 'conversation.resolved']

export function WebhookConfig() {
  const { t } = useTranslation()
  const [hooks, setHooks] = useState<WebhookEntry[]>([])
  const [url, setUrl] = useState('')
  const [event, setEvent] = useState(EVENTS[0])
  let nextId = hooks.length

  const add = () => {
    if (!url.trim()) return
    setHooks((prev) => [...prev, { id: `wh-${nextId++}-${prev.length}`, url: url.trim(), event }])
    setUrl('')
  }

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium text-foreground">{t('kommunikation.webhook.title')}</h3>

      <div className="space-y-1">
        {hooks.map((h) => (
          <div key={h.id} className="flex items-center gap-3 rounded-md border border-border px-3 py-2">
            <Webhook className="h-4 w-4 shrink-0 text-muted-foreground" />
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm text-foreground">{h.url}</p>
              <p className="text-[10px] text-muted-foreground">{h.event}</p>
            </div>
            <button
              onClick={() => setHooks((prev) => prev.filter((x) => x.id !== h.id))}
              className="rounded p-1 text-muted-foreground hover:text-error hover:bg-secondary transition-colors"
              title={t('common.delete')}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        ))}
        {hooks.length === 0 && (
          <p className="py-2 text-center text-xs text-muted-foreground">{t('kommunikation.webhook.empty')}</p>
        )}
      </div>

      <div className="flex gap-2">
        <input
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder={t('kommunikation.webhook.urlPlaceholder')}
          className="flex-1 rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
        />
        <select
          value={event}
          onChange={(e) => setEvent(e.target.value)}
          className="rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
        >
          {EVENTS.map((ev) => (
            <option key={ev} value={ev}>{ev}</option>
          ))}
        </select>
        <button
          onClick={add}
          disabled={!url.trim()}
          className="flex items-center gap-1 rounded-md bg-primary px-2.5 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
        >
          <Plus className="h-3.5 w-3.5" />
        </button>
      </div>
      <p className="text-[10px] text-muted-foreground">{t('kommunikation.webhook.hint')}</p>
    </div>
  )
}
