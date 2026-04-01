/* eslint-disable react-refresh/only-export-components */
import { useMemo } from 'react'
import { X } from 'lucide-react'
import { useContacts } from '@/api/hooks/useContacts'
import { backendContactToUI } from '@/modules/kontakte/adapters'
import { useSettingsStore } from '@/stores/settings'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type ContactSuggestion = { name: string; email: string }

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------

const FALLBACK_SIGNATURE = '<p>Mit freundlichen Grüßen<br/>Ihr Cosmi Team</p>'

/** Read HTML signature from settings store */
export function useEmailSignature(): string {
  const sig = useSettingsStore((s) => s.mail.signature)
  return sig || FALLBACK_SIGNATURE
}

/** Contact suggestions from CRM backend via React Query */
export function useContactSuggestions(): ContactSuggestion[] {
  const { data } = useContacts()
  const contacts = useMemo(() => (data?.contacts ?? []).map(backendContactToUI), [data])
  return useMemo(
    () =>
      contacts
        .filter((c) => c.email)
        .map((c) => ({
          name: `${c.firstName} ${c.lastName}`.trim(),
          email: c.email,
        })),
    [contacts],
  )
}

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

/** Build a full signature block for new emails */
export function buildSignatureHtml(signature: string): string {
  return `<p></p><p>--</p>${signature}`
}

/** Build reply body with signature + quoted text */
export function buildReplyHtml(
  signature: string,
  quotedHeader: string,
  quotedBody: string,
): string {
  return `<p></p><p>--</p>${signature}<hr><p><em>${quotedHeader}</em></p><blockquote>${quotedBody}</blockquote>`
}

/** Build forward body with signature + forwarded text */
export function buildForwardHtml(
  signature: string,
  forwardHeader: string,
  forwardBody: string,
): string {
  return `<p></p><p>--</p>${signature}<hr><p><em>${forwardHeader}</em></p>${forwardBody}`
}

/** Filter suggestions by input string */
export function filteredSuggestions(
  input: string,
  exclude: string[],
  source: ContactSuggestion[],
) {
  if (input.length === 0) return []
  const q = input.toLowerCase()
  return source.filter(
    (c) =>
      !exclude.includes(c.email) &&
      (c.name.toLowerCase().includes(q) || c.email.toLowerCase().includes(q)),
  )
}

/** Build email -> name lookup map */
export function buildContactMap(
  suggestions: ContactSuggestion[],
): Map<string, string> {
  const map = new Map<string, string>()
  suggestions.forEach((c) => map.set(c.email, c.name))
  return map
}

/** Strip HTML to plain text */
export function stripHtml(html: string): string {
  return html
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<\/p>/gi, '\n')
    .replace(/<\/blockquote>/gi, '\n')
    .replace(/<hr\s*\/?>/gi, '\n---\n')
    .replace(/<[^>]+>/g, '')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

// ---------------------------------------------------------------------------
// RecipientField Component
// ---------------------------------------------------------------------------

export function RecipientField({
  label,
  recipients,
  input,
  onInputChange,
  suggestions,
  onAdd,
  onRemove,
  onKeyDown,
  contactMap,
}: {
  label: string
  recipients: string[]
  input: string
  onInputChange: (v: string) => void
  suggestions: ContactSuggestion[]
  onAdd: (email: string) => void
  onRemove: (email: string) => void
  onKeyDown: (e: React.KeyboardEvent) => void
  contactMap?: Map<string, string>
}) {
  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <span className="text-xs text-muted-foreground w-6 shrink-0">
          {label}
        </span>
        <div className="flex flex-1 flex-wrap items-center gap-1 rounded-md border border-border px-2 py-1 min-h-[34px]">
          {recipients.map((email) => {
            const name = contactMap?.get(email)
            return (
              <span
                key={email}
                title={email}
                className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary"
              >
                {name || email}
                <button
                  onClick={() => onRemove(email)}
                  className="rounded-full hover:bg-primary/20 p-0.5"
                >
                  <X className="h-3 w-3" />
                </button>
              </span>
            )
          })}
          <input
            type="text"
            value={input}
            onChange={(e) => onInputChange(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder={recipients.length === 0 ? 'Empfaenger...' : ''}
            className="flex-1 min-w-[120px] bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
          />
        </div>
      </div>
      {suggestions.length > 0 && (
        <div className="ml-8 max-h-28 overflow-y-auto rounded-md border bg-card p-1 shadow-md z-50">
          {suggestions.slice(0, 5).map((c) => (
            <button
              key={c.email}
              onClick={() => onAdd(c.email)}
              className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-secondary"
            >
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                {c.name
                  .split(' ')
                  .map((n) => n[0])
                  .join('')}
              </span>
              <span className="text-foreground">{c.name}</span>
              <span className="text-xs text-muted-foreground ml-auto">
                {c.email}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
