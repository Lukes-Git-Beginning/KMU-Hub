/**
 * Bexio field mapping editor component.
 *
 * Allows admins to configure field mappings between KMU Hub and Bexio
 * for a given entity type (contact, invoice, quote). Supports add,
 * remove, direction change, and required flag per mapping.
 */
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Loader2, Plus, Trash2, RotateCcw } from 'lucide-react'
import {
  useBexioFieldMappings,
  useBexioUpdateFieldMappings,
} from '@/api/hooks/useBexio'
import type {
  BexioFieldMappingEntry,
  BexioEntityType,
} from '@/api/bexio-types'
import {
  BEXIO_CONTACT_FIELDS,
  KMUHUB_CONTACT_FIELDS,
  DEFAULT_CONTACT_MAPPINGS,
} from '@/api/bexio-types'

interface BexioFieldMappingEditorProps {
  entityType: BexioEntityType
  onSave?: () => void
  compact?: boolean
}

const DIRECTION_OPTIONS = [
  { value: 'inbound', label: '\u2190 Eingehend' },
  { value: 'outbound', label: '\u2192 Ausgehend' },
  { value: 'both', label: '\u2194 Bidirektional' },
] as const

export function BexioFieldMappingEditor({
  entityType,
  onSave,
  compact = false,
}: BexioFieldMappingEditorProps) {
  const { data: serverMappings, isLoading } = useBexioFieldMappings(entityType)
  const updateMappings = useBexioUpdateFieldMappings(entityType)

  const [mappings, setMappings] = useState<BexioFieldMappingEntry[]>([])
  const [isDirty, setIsDirty] = useState(false)

   
  useEffect(() => {
    if (serverMappings && serverMappings.length > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync local editable state from prop
      setMappings(serverMappings)
    } else if (!isLoading) {
      // Use defaults when no server mappings exist
      setMappings(entityType === 'contact' ? [...DEFAULT_CONTACT_MAPPINGS] : [])
    }
  }, [serverMappings, isLoading, entityType])

  const bexioFields =
    entityType === 'contact' ? BEXIO_CONTACT_FIELDS : BEXIO_CONTACT_FIELDS
  const kmuhubFields =
    entityType === 'contact' ? KMUHUB_CONTACT_FIELDS : KMUHUB_CONTACT_FIELDS

  const handleUpdate = (
    index: number,
    field: keyof BexioFieldMappingEntry,
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
        bexio_field: '',
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
  const duplicateBexioFields = mappings
    .map((m) => m.bexio_field)
    .filter((f, i, arr) => f && arr.indexOf(f) !== i)

  const hasValidationErrors = duplicateBexioFields.length > 0

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
            Feld-Zuordnung: {entityType === 'contact' ? 'Kontakte' : entityType === 'invoice' ? 'Rechnungen' : 'Offerten'}
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
          <span>Bexio</span>
          <span className="w-10 text-center">Pflicht</span>
          <span className="w-8" />
        </div>

        {mappings.map((mapping, index) => (
          <div
            key={index}
            className={`grid grid-cols-[1fr,auto,1fr,auto,auto] gap-2 items-center ${
              duplicateBexioFields.includes(mapping.bexio_field) && mapping.bexio_field
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
                <option key={f.id} value={f.id}>
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
              value={mapping.bexio_field}
              onChange={(e) =>
                handleUpdate(index, 'bexio_field', e.target.value)
              }
            >
              <option value="">-- Feld waehlen --</option>
              {bexioFields.map((f) => (
                <option key={f.id} value={f.id}>
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
      {duplicateBexioFields.length > 0 && (
        <p className="text-xs text-red-500">
          Doppelte Bexio-Felder: {[...new Set(duplicateBexioFields)].join(', ')}
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
          {compact && (
            <Button
              variant="ghost"
              size="sm"
              onClick={handleResetDefaults}
              className="text-xs"
            >
              <RotateCcw className="h-3 w-3 mr-1" />
              Standard wiederherstellen
            </Button>
          )}
        </div>
      )}
    </div>
  )
}
