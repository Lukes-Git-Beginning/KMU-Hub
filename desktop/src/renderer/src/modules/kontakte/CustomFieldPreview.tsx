/**
 * CustomFieldPreview — Live preview of how custom fields render in forms.
 *
 * Shows a mock form with all defined custom fields, rendered by their type.
 */
import {
  Type,
  Hash,
  Calendar,
  ChevronDown,
  CheckSquare,
  Link2,
  Eye,
} from 'lucide-react'
import type { CustomFieldDefinition, CustomFieldType } from '@/stores/contacts'

// ---------------------------------------------------------------------------
// Type icons
// ---------------------------------------------------------------------------

const typeIcons: Record<CustomFieldType, typeof Type> = {
  text: Type,
  number: Hash,
  date: Calendar,
  dropdown: ChevronDown,
  checkbox: CheckSquare,
  url: Link2,
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface CustomFieldPreviewProps {
  fields: CustomFieldDefinition[]
}

export function CustomFieldPreview({ fields }: CustomFieldPreviewProps) {
  if (fields.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border p-6 text-center">
        <Eye className="mx-auto h-8 w-8 text-muted-foreground/30" />
        <p className="mt-2 text-sm text-muted-foreground">
          Noch keine Felder definiert.
        </p>
        <p className="text-xs text-muted-foreground/70">
          Erstelle ein benutzerdefiniertes Feld um die Vorschau zu sehen.
        </p>
      </div>
    )
  }

  return (
    <div className="rounded-lg border border-border p-4 space-y-3">
      <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground uppercase tracking-wide">
        <Eye className="h-3.5 w-3.5" />
        Formular-Vorschau
      </div>

      <div className="space-y-3">
        {fields.map((field) => {
          const Icon = typeIcons[field.type]
          return (
            <div key={field.id} className="space-y-1">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                {field.name}
                {field.required && <span className="text-error text-xs">*</span>}
              </label>
              {renderFieldPreview(field)}
            </div>
          )
        })}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Render helpers
// ---------------------------------------------------------------------------

function renderFieldPreview(field: CustomFieldDefinition) {
  const baseClasses =
    'h-9 w-full rounded-md border border-border bg-secondary/30 px-3 text-sm text-muted-foreground pointer-events-none'

  switch (field.type) {
    case 'text':
      return <div className={baseClasses + ' flex items-center'}>Beispieltext...</div>
    case 'number':
      return <div className={baseClasses + ' flex items-center'}>12345</div>
    case 'date':
      return <div className={baseClasses + ' flex items-center'}>21.02.2026</div>
    case 'url':
      return <div className={baseClasses + ' flex items-center text-primary/50'}>https://example.com</div>
    case 'checkbox':
      return (
        <div className="flex items-center gap-2">
          <div className="h-4 w-4 rounded border border-border bg-primary/20 flex items-center justify-center">
            <CheckSquare className="h-3 w-3 text-primary" />
          </div>
          <span className="text-sm text-muted-foreground">{field.name}</span>
        </div>
      )
    case 'dropdown':
      return (
        <div className={baseClasses + ' flex items-center justify-between'}>
          <span>{field.options?.[0] ?? 'Auswählen...'}</span>
          <ChevronDown className="h-3.5 w-3.5" />
        </div>
      )
    default:
      return <div className={baseClasses + ' flex items-center'}>—</div>
  }
}
