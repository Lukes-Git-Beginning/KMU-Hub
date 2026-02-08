/**
 * Displays and edits custom fields for a task.
 *
 * Fetches custom field definitions (entity_type='task') and current values,
 * renders input controls based on field_type, and persists changes via
 * debounced PUT calls. Reuses the CRM custom fields engine pattern.
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import {
  useTaskCustomFields,
  useSetTaskCustomFields,
} from '@/api/hooks/useTasks'
import { Input } from '@/components/ui/input'
import type { components } from '@/api/types'

type CustomFieldInfo = components['schemas']['CustomFieldInfo']

interface CustomFieldsSectionProps {
  taskId: string
  projectId: string
}

export default function CustomFieldsSection({
  taskId,
}: CustomFieldsSectionProps) {
  // Fetch custom field definitions for 'task' entity type
  // Note: The CRM custom field API currently only supports contact/company/deal/activity.
  // For now, we query with no entity_type filter and show all, matching the CRM pattern.
  // When task custom fields are added to the backend, the filter can be adjusted.
  const { data: defsData } = useQuery({
    queryKey: ['custom-field-definitions', 'task'],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/custom-fields', {
        params: { query: {} },
      })
      if (error) throw error
      return data
    },
  })

  const { data: valuesData } = useTaskCustomFields(taskId)
  const setCustomFields = useSetTaskCustomFields()

  const definitions: CustomFieldInfo[] = defsData?.customFields ?? []
  const currentValues: Record<string, string> = valuesData?.custom_fields ?? {}

  const [localValues, setLocalValues] = useState<Record<string, string>>({})
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout>>()

  // Sync local values from server
  useEffect(() => {
    setLocalValues(currentValues)
  }, [JSON.stringify(currentValues)])

  const persistValues = useCallback(
    (newValues: Record<string, string>) => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current)
      }
      debounceTimerRef.current = setTimeout(() => {
        const values = Object.entries(newValues)
          .filter(([, v]) => v !== '')
          .map(([fieldId, value]) => ({ field_id: fieldId, value }))
        if (values.length > 0) {
          setCustomFields.mutate({ taskId, values })
        }
      }, 500)
    },
    [taskId, setCustomFields]
  )

  function handleChange(fieldId: string, value: string) {
    const next = { ...localValues, [fieldId]: value }
    setLocalValues(next)
    persistValues(next)
  }

  if (definitions.length === 0) {
    return (
      <p className="text-xs text-muted-foreground italic py-1">
        Keine benutzerdefinierten Felder konfiguriert.
      </p>
    )
  }

  return (
    <div className="space-y-2">
      {definitions.map((field) => {
        const fieldId = field.id ?? ''
        const value = localValues[fieldId] ?? ''

        return (
          <div key={fieldId} className="space-y-0.5">
            <label className="text-xs text-muted-foreground">
              {field.name}
              {field.isRequired && (
                <span className="text-destructive ml-0.5">*</span>
              )}
            </label>

            {field.fieldType === 'text' && (
              <Input
                className="h-7 text-xs"
                value={value}
                onChange={(e) => handleChange(fieldId, e.target.value)}
                placeholder={field.name ?? ''}
              />
            )}

            {field.fieldType === 'number' && (
              <Input
                type="number"
                className="h-7 text-xs"
                value={value}
                onChange={(e) => handleChange(fieldId, e.target.value)}
                placeholder="0"
              />
            )}

            {field.fieldType === 'date' && (
              <Input
                type="date"
                className="h-7 text-xs"
                value={value}
                onChange={(e) => handleChange(fieldId, e.target.value)}
              />
            )}

            {field.fieldType === 'boolean' && (
              <div className="flex items-center gap-2 py-0.5">
                <input
                  type="checkbox"
                  className="h-3.5 w-3.5 rounded border-input accent-primary"
                  checked={value === 'true'}
                  onChange={(e) =>
                    handleChange(fieldId, String(e.target.checked))
                  }
                />
                <span className="text-xs">{value === 'true' ? 'Ja' : 'Nein'}</span>
              </div>
            )}

            {field.fieldType === 'select' && (
              <select
                className="flex h-7 w-full rounded-md border border-input bg-background px-2 text-xs ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                value={value}
                onChange={(e) => handleChange(fieldId, e.target.value)}
              >
                <option value="">-- Auswahl --</option>
                {(field.options ?? []).map((opt) => (
                  <option key={opt} value={opt}>
                    {opt}
                  </option>
                ))}
              </select>
            )}

            {field.fieldType === 'multiselect' && (
              <div className="flex flex-wrap gap-1 py-0.5">
                {(field.options ?? []).map((opt) => {
                  const selected = value.split(',').includes(opt)
                  return (
                    <button
                      key={opt}
                      type="button"
                      className={`rounded border px-1.5 py-0.5 text-xs transition-colors ${
                        selected
                          ? 'bg-primary text-primary-foreground border-primary'
                          : 'bg-background text-muted-foreground border-border hover:bg-accent'
                      }`}
                      onClick={() => {
                        const current = value
                          ? value.split(',').filter(Boolean)
                          : []
                        const next = selected
                          ? current.filter((v) => v !== opt)
                          : [...current, opt]
                        handleChange(fieldId, next.join(','))
                      }}
                    >
                      {opt}
                    </button>
                  )
                })}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
