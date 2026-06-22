import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * User-defined email templates (demo-stateful, persisted locally — no backend).
 * Built-in templates live in EmailTemplateDialog and are read-only; these are
 * the user's own, fully CRUD-able. Placeholders use the {{name}} convention and
 * are filled in when the template is inserted into a compose window.
 */
export interface UserTemplate {
  id: string
  name: string
  category: 'vertrieb' | 'kommunikation' | 'finanzen'
  subject: string
  /** HTML body with {{placeholder}} variables. */
  body: string
}

interface MailTemplatesState {
  userTemplates: UserTemplate[]
  addTemplate: (tpl: Omit<UserTemplate, 'id'>) => UserTemplate
  updateTemplate: (id: string, patch: Partial<Omit<UserTemplate, 'id'>>) => void
  removeTemplate: (id: string) => void
}

let counter = 0
function newId(): string {
  counter += 1
  return `utpl-${Date.now().toString(36)}-${counter}`
}

export const useMailTemplatesStore = create<MailTemplatesState>()(
  persist(
    (set) => ({
      userTemplates: [
        {
          id: 'utpl-demo-1',
          name: 'Kurze Rückmeldung',
          category: 'kommunikation',
          subject: 'Ihre Anfrage — {{firma}}',
          body: '<p>{{anrede}} {{name}},</p><p>vielen Dank für Ihre Nachricht. Ich melde mich bis spätestens {{datum}} mit allen Details zurück.</p><p>Beste Grüße</p>',
        },
      ],
      addTemplate: (tpl) => {
        const created: UserTemplate = { id: newId(), ...tpl }
        set((s) => ({ userTemplates: [...s.userTemplates, created] }))
        return created
      },
      updateTemplate: (id, patch) =>
        set((s) => ({
          userTemplates: s.userTemplates.map((t) => (t.id === id ? { ...t, ...patch } : t)),
        })),
      removeTemplate: (id) =>
        set((s) => ({ userTemplates: s.userTemplates.filter((t) => t.id !== id) })),
    }),
    { name: 'cosmi-mail-templates' },
  ),
)

/** Extract {{placeholder}} variable names from a template's subject + body. */
export function extractPlaceholders(...parts: string[]): string[] {
  const found = new Set<string>()
  const re = /\{\{\s*(\w+)\s*\}\}/g
  for (const part of parts) {
    let m: RegExpExecArray | null
    while ((m = re.exec(part)) !== null) found.add(m[1])
  }
  return [...found]
}

/** Replace {{placeholder}} occurrences using the given values (blanks kept as-is). */
export function fillPlaceholders(text: string, values: Record<string, string>): string {
  return text.replace(/\{\{\s*(\w+)\s*\}\}/g, (whole, key: string) => {
    const v = values[key]
    return v && v.trim() ? v : whole
  })
}
