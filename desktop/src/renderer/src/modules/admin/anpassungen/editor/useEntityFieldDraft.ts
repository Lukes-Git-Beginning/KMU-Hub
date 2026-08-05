/**
 * useEntityFieldDraft — staging custom fields for one entity, shared by the Felder
 * panel and the Spalten panel (Darien 2026-08-05: columns must be creatable,
 * deletable and re-configurable too, not just switchable).
 *
 * Custom fields keep their own store, so — unlike labels/value-sets which live in
 * the tenant overlay — the editor stages the DESIRED FULL LIST per entity as a
 * snapshot in the draft. Deploy diffs that snapshot against the live store. Both
 * panels write through here so their edits land in the same snapshot instead of
 * fighting over it.
 */
import { useMemo } from 'react'
import { nanoid } from 'nanoid'
import { useCustomFields } from '@/api/hooks/useCustomFields'
import type {
  CustomFieldDefinition,
  CustomFieldEntity,
  CreateCustomFieldInput,
  UpdateCustomFieldInput,
} from '@/mocks/data/custom-fields'
import { useDraftConfig } from './DraftConfigProvider'

/** Signature of a field's meaningful props (order/inUse excluded — no reorder UI). */
export function fieldSig(f: CustomFieldDefinition): string {
  return JSON.stringify({
    id: f.id,
    label: f.label,
    type: f.type,
    required: f.required,
    visible: f.visible,
    options: f.options,
    valueSetId: f.valueSetId ?? null,
    validation: f.validation ?? null,
    defaultValue: f.defaultValue ?? '',
  })
}

/** Whole-list signature (order-independent) — drives dirty detection. */
function listSig(list: CustomFieldDefinition[]): string {
  return list.map(fieldSig).sort().join('|')
}

export interface EntityFieldDraft {
  /** Live baseline, sorted by display order. */
  baseline: CustomFieldDefinition[]
  /** Effective list = staged snapshot, or the baseline when nothing is staged. */
  effective: CustomFieldDefinition[]
  /** Baseline fields the snapshot dropped (staged deletions, restorable). */
  removed: CustomFieldDefinition[]
  isLoading: boolean
  isDraft: boolean
  /** Stage a full replacement list (drops the snapshot when it equals the baseline). */
  stage: (next: CustomFieldDefinition[]) => void
  create: (input: CreateCustomFieldInput) => CustomFieldDefinition
  update: (id: string, input: UpdateCustomFieldInput) => void
  remove: (id: string) => void
  reset: () => void
}

export function useEntityFieldDraft(entity: CustomFieldEntity): EntityFieldDraft {
  const { customFields, setDraftEntityFields, resetDraftEntityFields } = useDraftConfig()
  const { data = [], isLoading } = useCustomFields(entity)

  const baseline = useMemo(() => [...data].sort((a, b) => a.order - b.order), [data])
  const snapshot = customFields[entity]
  const effective = snapshot ?? baseline
  const effectiveIds = new Set(effective.map((f) => f.id))
  const removed = baseline.filter((f) => !effectiveIds.has(f.id))

  const stage = (next: CustomFieldDefinition[]): void => {
    if (listSig(next) === listSig(baseline)) resetDraftEntityFields(entity)
    else setDraftEntityFields(entity, next)
  }

  const create = (input: CreateCustomFieldInput): CustomFieldDefinition => {
    const field: CustomFieldDefinition = {
      id: `draft_${nanoid(8)}`,
      entity,
      key: `draft_${nanoid(4)}`,
      label: input.label,
      type: input.type,
      required: input.required ?? false,
      options: input.options ?? [],
      valueSetId: input.valueSetId,
      validation: input.validation,
      defaultValue: input.defaultValue,
      visible: input.visible ?? true,
      order: effective.length,
      inUse: false,
    }
    stage([...effective, field])
    return field
  }

  const update = (id: string, input: UpdateCustomFieldInput): void => {
    stage(
      effective.map((f) =>
        f.id === id
          ? {
              ...f,
              ...(input.label !== undefined && { label: input.label }),
              ...(input.type !== undefined && { type: input.type }),
              ...(input.required !== undefined && { required: input.required }),
              ...(input.options !== undefined && { options: input.options }),
              // Explicit, so unbinding a value list actually clears it.
              ...('valueSetId' in input && { valueSetId: input.valueSetId }),
              ...(input.validation !== undefined && { validation: input.validation }),
              ...(input.defaultValue !== undefined && { defaultValue: input.defaultValue }),
              ...(input.visible !== undefined && { visible: input.visible }),
            }
          : f,
      ),
    )
  }

  const remove = (id: string): void => stage(effective.filter((f) => f.id !== id))

  return {
    baseline,
    effective,
    removed,
    isLoading,
    isDraft: snapshot !== undefined,
    stage,
    create,
    update,
    remove,
    reset: () => resetDraftEntityFields(entity),
  }
}
