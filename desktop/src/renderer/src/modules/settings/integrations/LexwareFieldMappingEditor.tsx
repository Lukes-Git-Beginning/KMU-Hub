/**
 * Lexware field mapping editor component.
 *
 * Allows admins to configure field mappings between KMU Hub and Lexware
 * for a given entity type (contact, invoice, quote). Supports add,
 * remove, direction change, and required flag per mapping.
 * Handles nested Lexware field paths in dropdown (e.g. person.firstName).
 */
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Loader2, Plus, Trash2, RotateCcw } from 'lucide-react'
import {
  useLexwareFieldMappings,
  useLexwareUpdateFieldMappings,
} from '@/api/hooks/useLexware'
import type { LexwareFieldMappingEntry } from '@/api/lexware-types'
import {
  LEXWARE_CONTACT_FIELDS,
  KMUHUB_CONTACT_FIELDS,
  DEFAULT_CONTACT_MAPPINGS,
} from '@/api/lexware-types'

interface LexwareFieldMappingEditorProps {
  entityType: string
  onSave?: () => void
  compact?: boolean
}

const DIRECTION_OPTIONS = [
  { value: 'inbound', label: '\u2190 Eingehend' },
  { value: 'outbound', label: '\u2192 Ausgehend' },
  { value: 'both', label: '\u2194 Bidirektional' },
] as const

export function LexwareFieldMappingEditor({
  entityType,
  onSave,
  compact = false,
}: LexwareFieldMappingEditorProps) {
  const { data: serverMappings, isLoading } = useLexwareFieldMappings(entityType)
  const updateMappings = useLexwareUpdateFieldMappings(entityType)

  const [mappings, setMappings] = useState<LexwareFieldMappingEntry[]>([])
  const [isDirty, setIsDirty] = useState(false)

  // Initialize from server or defaults
  useEffect(() => {
    if (serverMappings && serverMappings.length > 0) {
      setMappings(serverMappings)
    } else if (!isLoading) {
      // Use defaults when no server mappings exist
      setMappings(entityType === 'contact' ? [...DEFAULT_CONTACT_MAPPINGS] : [])
    }
  }, [serverMappings, isLoading, entityType])

  const lexwareFields =
    entityType === 'contact' ? LEXWARE_CONTACT_FIELDS : LEXWARE_CONTACT_FIELDS
  const kmuhubFields =
    entityType === 'contact' ? KMUHUB_CONTACT_FIELDS : KMUHUB_CONTACT_FIELDS

  const handleUpdate = (
    index: number,
    field: keyof LexwareFieldMappingEntry,
    value: string | boolean,
  ) => {
    setMappings((prev) =>
      prev.map((m, i) => (i === index ? { ...m, [field]: value } : m)),
    )
    setIsDirty(true)
  }

  const handleAdd = () => {
    setMappings((prev) => [
      ...prev,
      {
        kmuhub_field: '',
        lexware_field: '',
        direction: 'both',
        required: false,
      },
    ])
    setIsDirty(true)
  }

  const handleRemove = (index: number) => {
    if (mappings[index].required) return
    setMappings((prev) => prev.filter((_, i) => i !== index))
    setIsDirty(true)
  }

  const handleResetDefaults = () => {
    if (!confirm('Alle Zuordnungen auf Standard zuruecksetzen?')) return
    setMappings(
      entityType === 'contact' ? [...DEFAULT_CONTACT_MAPPINGS] : [],
    )
    setIsDirty(true)
  }

  const handleSave = async () => {
    await updateMappings.mutateAsync(mappings)
    setIsDirty(false)
    onSave?.()
  }

  // Validation
  const duplicateLexwareFields = mappings
    .map((m) => m.lexware_field)
    .filter((f, i, arr) => f && arr.indexOf(f) !== i)

  const hasValidationErrors = duplicateLexwareFields.length > 0

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 py-4">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className="text-sm text-muted-foreground">
          Lade Feld-Zuordnungen...
        </span>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {!compact && (
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-medium">
            Feld-Zuordnung:{' '}
            {entityType === 'contact'
              ? 'Kontakte'
              : entityType === 'invoice'
                ? 'Rechnungen'
                : 'Angebote'}
          </h4>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleResetDefaults}
            className="text-xs"
          >
            <RotateCcw className="h-3 w-3 mr-1" />
            Standard
          </Button>
        </div>
      )}

      {/* Mapping table */}
      <div className="space-y-2">
        {/* Header */}
        <div className="grid grid-cols-[1fr,auto,1fr,auto,auto] gap-2 text-xs text-muted-foreground px-1">
          <span>KMU Hub</span>
          <span className="w-[120px]">Richtung</span>
          <span>Lexware</span>
          <span className="w-10 text-center">Pflicht</span>
          <span className="w-8" />
        </div>

        {mappings.map((mapping, index) => (
          <div
            key={index}
            className={`grid grid-cols-[1fr,auto,1fr,auto,auto] gap-2 items-center ${
              duplicateLexwareFields.includes(mapping.lexware_field) &&
              mapping.lexware_field
                ? 'rounded-md ring-1 ring-red-500/50 p-1'
                : ''
            }`}
          >
            <select
              className="text-xs rounded-md border border-border bg-background px-2 py-1.5"
              value={mapping.kmuhub_field}
              onChange={(e) =>
                handleUpdate(index, 'kmuhub_field', e.target.value)
              }
            >
              <option value="">-- Feld waehlen --</option>
              {kmuhubFields.map((f) => (
                <option key={f.value} value={f.value}>
                  {f.label}
                </option>
              ))}
            </select>

            <select
              className="text-xs rounded-md border border-border bg-background px-2 py-1.5 w-[120px]"
              value={mapping.direction}
              onChange={(e) =>
                handleUpdate(index, 'direction', e.target.value)
              }
            >
              {DIRECTION_OPTIONS.map((d) => (
                <option key={d.value} value={d.value}>
                  {d.label}
                </option>
              ))}
            </select>

            <select
              className="text-xs rounded-md border border-border bg-background px-2 py-1.5"
              value={mapping.lexware_field}
              onChange={(e) =>
                handleUpdate(index, 'lexware_field', e.target.value)
              }
            >
              <option value="">-- Feld waehlen --</option>
              {lexwareFields.map((f) => (
                <option key={f.value} value={f.value}>
                  {f.label}
                </option>
              ))}
            </select>

            <div className="w-10 flex justify-center">
              <input
                type="checkbox"
                checked={mapping.required}
                onChange={(e) =>
                  handleUpdate(index, 'required', e.target.checked)
                }
                className="rounded border-border"
              />
            </div>

            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-8 p-0"
              onClick={() => handleRemove(index)}
              disabled={mapping.required}
            >
              <Trash2 className="h-3 w-3 text-muted-foreground" />
            </Button>
          </div>
        ))}
      </div>

      {/* Add mapping button */}
      <Button
        variant="outline"
        size="sm"
        onClick={handleAdd}
        className="text-xs"
      >
        <Plus className="h-3 w-3 mr-1" />
        Zuordnung hinzufuegen
      </Button>

      {/* Validation errors */}
      {duplicateLexwareFields.length > 0 && (
        <p className="text-xs text-red-500">
          Doppelte Lexware-Felder:{' '}
          {[...new Set(duplicateLexwareFields)].join(', ')}
        </p>
      )}

      {/* Action buttons */}
      {!compact && (
        <div className="flex items-center gap-2 pt-2 border-t border-border">
          <Button
            size="sm"
            onClick={handleSave}
            disabled={
              !isDirty || hasValidationErrors || updateMappings.isPending
            }
          >
            {updateMappings.isPending && (
              <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
            )}
            Speichern
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleResetDefaults}
            className="text-xs"
          >
            <RotateCcw className="h-3 w-3 mr-1" />
            Standard wiederherstellen
          </Button>
        </div>
      )}

      {/* Compact mode: save + reset in a single row */}
      {compact && isDirty && (
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            onClick={handleSave}
            disabled={hasValidationErrors || updateMappings.isPending}
          >
            {updateMappings.isPending && (
              <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
            )}
            Speichern
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleResetDefaults}
            className="text-xs"
          >
            <RotateCcw className="h-3 w-3 mr-1" />
            Standard
          </Button>
        </div>
      )}
    </div>
  )
}
