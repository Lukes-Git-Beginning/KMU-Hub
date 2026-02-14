import { X } from 'lucide-react'

export const defaultSignature = '\n\n--\nMit freundlichen Gruessen\nDarien\nKMU Hub AG'

export const contactSuggestions = [
  { name: 'Anna Mueller', email: 'anna.mueller@kmuhub.ch' },
  { name: 'Michael Berg', email: 'michael.berg@kmuhub.ch' },
  { name: 'Sarah Klein', email: 'sarah@designstudio.ch' },
  { name: 'Thomas Weber', email: 'thomas.weber@abc-gmbh.ch' },
  { name: 'Lisa Schmidt', email: 'lisa.schmidt@kmuhub.ch' },
  { name: 'Peter Koch', email: 'peter.koch@kmuhub.ch' },
  { name: 'Jonas Diaz', email: 'jonas.diaz@kmuhub.ch' },
  { name: 'Eva Brunner', email: 'eva@brunner-partner.ch' },
  { name: 'Markus Steiner', email: 'markus@steiner-bau.ch' },
  { name: 'Claudia Frei', email: 'claudia.frei@techventures.at' },
]

export function filteredSuggestions(input: string, exclude: string[]) {
  return input.length > 0
    ? contactSuggestions.filter(
        (c) =>
          !exclude.includes(c.email) &&
          (c.name.toLowerCase().includes(input.toLowerCase()) ||
            c.email.toLowerCase().includes(input.toLowerCase()))
      )
    : []
}

export function RecipientField({
  label,
  recipients,
  input,
  onInputChange,
  suggestions,
  onAdd,
  onRemove,
  onKeyDown,
}: {
  label: string
  recipients: string[]
  input: string
  onInputChange: (v: string) => void
  suggestions: typeof contactSuggestions
  onAdd: (email: string) => void
  onRemove: (email: string) => void
  onKeyDown: (e: React.KeyboardEvent) => void
}) {
  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <span className="text-xs text-muted-foreground w-6 shrink-0">{label}</span>
        <div className="flex flex-1 flex-wrap items-center gap-1 rounded-md border border-border px-2 py-1 min-h-[34px]">
          {recipients.map((email) => (
            <span
              key={email}
              className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary"
            >
              {email}
              <button onClick={() => onRemove(email)} className="rounded-full hover:bg-primary/20 p-0.5">
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
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
        <div className="ml-8 max-h-28 overflow-y-auto rounded-md border bg-card p-1">
          {suggestions.slice(0, 5).map((c) => (
            <button
              key={c.email}
              onClick={() => onAdd(c.email)}
              className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-secondary"
            >
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                {c.name.split(' ').map((n) => n[0]).join('')}
              </span>
              <span className="text-foreground">{c.name}</span>
              <span className="text-xs text-muted-foreground ml-auto">{c.email}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
