/**
 * React Query hooks for the unified Custom Fields API (v1.1).
 *
 * All mutations invalidate the fields list cache so the UI stays in sync
 * without a full page reload.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { API_BASE_URL } from '@/lib/constants'
import type {
  CustomFieldDefinition,
  CustomFieldEntity,
  CreateCustomFieldInput,
  UpdateCustomFieldInput,
} from '@/mocks/data/custom-fields'

const API = API_BASE_URL

// ── Query keys ────────────────────────────────────────────────────────────────

const fieldsKey = (entity?: CustomFieldEntity) =>
  entity ? ['customization', 'fields', entity] : ['customization', 'fields']

// ── Fetch helpers ─────────────────────────────────────────────────────────────

async function fetchFields(entity?: CustomFieldEntity): Promise<CustomFieldDefinition[]> {
  const url = entity
    ? `${API}/api/v1/customization/fields?entity=${entity}`
    : `${API}/api/v1/customization/fields`
  const res = await fetch(url)
  if (!res.ok) throw new Error(`fetchFields failed: ${res.status}`)
  const data = await res.json() as { fields: CustomFieldDefinition[] }
  return data.fields
}

async function fetchField(id: string): Promise<CustomFieldDefinition> {
  const res = await fetch(`${API}/api/v1/customization/fields/${id}`)
  if (!res.ok) throw new Error(`fetchField failed: ${res.status}`)
  const data = await res.json() as { field: CustomFieldDefinition }
  return data.field
}

async function postField(input: CreateCustomFieldInput): Promise<CustomFieldDefinition> {
  const res = await fetch(`${API}/api/v1/customization/fields`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(`createField failed: ${res.status}`)
  const data = await res.json() as { field: CustomFieldDefinition }
  return data.field
}

async function putField(
  id: string,
  input: UpdateCustomFieldInput,
): Promise<CustomFieldDefinition> {
  const res = await fetch(`${API}/api/v1/customization/fields/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(`updateField failed: ${res.status}`)
  const data = await res.json() as { field: CustomFieldDefinition }
  return data.field
}

async function deleteField(id: string): Promise<void> {
  const res = await fetch(`${API}/api/v1/customization/fields/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteField failed: ${res.status}`)
}

// ── Hooks ─────────────────────────────────────────────────────────────────────

/** List all custom fields, optionally filtered by entity. */
export function useCustomFields(entity?: CustomFieldEntity) {
  return useQuery({
    queryKey: fieldsKey(entity),
    queryFn: () => fetchFields(entity),
  })
}

/** Single field by id. */
export function useCustomField(id: string) {
  return useQuery({
    queryKey: [...fieldsKey(), id],
    queryFn: () => fetchField(id),
    enabled: Boolean(id),
  })
}

/** Create a new custom field. Invalidates the list for the affected entity. */
export function useCreateCustomField() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: postField,
    onSuccess: (field) => {
      void qc.invalidateQueries({ queryKey: fieldsKey() })
      void qc.invalidateQueries({ queryKey: fieldsKey(field.entity) })
    },
  })
}

/** Update an existing field. */
export function useUpdateCustomField(entity?: CustomFieldEntity) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateCustomFieldInput }) =>
      putField(id, input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: fieldsKey() })
      if (entity) void qc.invalidateQueries({ queryKey: fieldsKey(entity) })
    },
  })
}

/** Delete a field. */
export function useDeleteCustomField(entity?: CustomFieldEntity) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: deleteField,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: fieldsKey() })
      if (entity) void qc.invalidateQueries({ queryKey: fieldsKey(entity) })
    },
  })
}
