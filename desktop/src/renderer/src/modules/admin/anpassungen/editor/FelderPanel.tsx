/**
 * FelderPanel (Modul-Editor v1, E-3c) — the module-scoped custom-field editor
 * inside the properties panel. Custom fields keep their own store (own BE
 * persistence, mocks/data/custom-fields), so — unlike Begriffe/Wertelisten which
 * live in the tenant overlay — the editor stages field changes as a per-entity
 * SNAPSHOT in the draft (the desired full field list). Nothing goes live until
 * "Übernehmen": deploy diffs the snapshot against the live store (create/update/
 * delete). The panel therefore carries its own preview (the staged list), the
 * same way WertelistenPanel does.
 *
 * Draft-consistent (Darien 2026-07-22): a field change rides the same deploy/
 * schedule/rollback rails as labels and value-sets.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { nanoid } from 'nanoid'
import {
  Plus,
  RotateCcw,
  EyeOff,
  Type,
  Hash,
  Calendar,
  ToggleLeft,
  ChevronDown,
  List,
  Link2,
  Mail,
  Phone,
} from 'lucide-react'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { resolveValueSet } from '@/mocks/data/customization'
import { useCustomFields } from '@/api/hooks/useCustomFields'
import type {
  CustomFieldDefinition,
  CustomFieldEntity,
  CustomFieldType,
  CreateCustomFieldInput,
  UpdateCustomFieldInput,
} from '@/mocks/data/custom-fields'
import { FieldEditorModal, type ValueSetChoice } from '../FieldEditorModal'
import { getEditorModule } from './editorModules'
import { useDraftConfig } from './DraftConfigProvider'

// ── Type icon ─────────────────────────────────────────────────────────────────

function typeIcon(type: CustomFieldType): typeof Type {
  const map: Record<CustomFieldType, typeof Type> = {
    text: Type,
    number: Hash,
    date: Calendar,
    boolean: ToggleLeft,
    select: ChevronDown,
    multi_select: List,
    url: Link2,
    email: Mail,
    phone: Phone,
  }
  return map[type] ?? Type
}

// ── Draft-vs-baseline diffing ──────────────────────────────────────────────────

/** Signature of a field's meaningful props (order/inUse excluded — no reorder UI). */
function fieldSig(f: CustomFieldDefinition): string {
  return JSON.stringify({
    id: f.id,
    label: f.label,
    type: f.type,
    required: f.required,
    visible: f.visible,
    options: f.options,
    validation: f.validation ?? null,
    defaultValue: f.defaultValue ?? '',
  })
}

/** Whole-list signature (order-independent) — drives dirty detection. */
function listSig(list: CustomFieldDefinition[]): string {
  return list
    .map(fieldSig)
    .sort()
    .join('|')
}

type FieldBadge = 'added' | 'modified' | null

// ── Per-entity editor ───────────────────────────────────────────────────────────

function EntityFieldsEditor({
  entity,
  valueSetChoices,
}: {
  entity: CustomFieldEntity
  valueSetChoices: ValueSetChoice[]
}): React.ReactElement {
  const { t } = useTranslation()
  const { customFields, setDraftEntityFields, resetDraftEntityFields } = useDraftConfig()
  const { data: baseline = [], isLoading } = useCustomFields(entity)

  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<CustomFieldDefinition | null>(null)
  const [deleteCandidate, setDeleteCandidate] = useState<CustomFieldDefinition | null>(null)

  const baseSorted = useMemo(
    () => [...baseline].sort((a, b) => a.order - b.order),
    [baseline],
  )
  const draftSnapshot = customFields[entity]
  const effective = draftSnapshot ?? baseSorted
  const isDraft = draftSnapshot !== undefined

  const baseById = useMemo(() => new Map(baseSorted.map((f) => [f.id, f])), [baseSorted])
  const effectiveIds = new Set(effective.map((f) => f.id))
  const removed = baseSorted.filter((f) => !effectiveIds.has(f.id))

  /** Stage a new full list — or drop the snapshot if it equals the live baseline. */
  const stage = (next: CustomFieldDefinition[]): void => {
    if (listSig(next) === listSig(baseSorted)) resetDraftEntityFields(entity)
    else setDraftEntityFields(entity, next)
  }

  const badgeFor = (f: CustomFieldDefinition): FieldBadge => {
    const base = baseById.get(f.id)
    if (!base) return 'added'
    if (fieldSig(f) !== fieldSig(base)) return 'modified'
    return null
  }

  const handleCreate = (input: CreateCustomFieldInput): void => {
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
    setCreateOpen(false)
  }

  const handleUpdate = (id: string, input: UpdateCustomFieldInput): void => {
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
    setEditTarget(null)
  }

  const toggleVisible = (id: string): void => {
    stage(effective.map((f) => (f.id === id ? { ...f, visible: !f.visible } : f)))
  }

  const removeField = (field: CustomFieldDefinition): void => {
    stage(effective.filter((f) => f.id !== field.id))
    setDeleteCandidate(null)
    setEditTarget(null)
  }

  const requestRemove = (field: CustomFieldDefinition): void => {
    // Draft-created field (never persisted) → drop straight away, no consequence.
    if (field.id.startsWith('draft_') || !field.inUse) removeField(field)
    else setDeleteCandidate(field)
  }

  const restore = (field: CustomFieldDefinition): void => {
    stage([...effective, field].sort((a, b) => a.order - b.order))
  }

  return (
    <div className="flex flex-1 flex-col gap-3 overflow-y-auto px-4 py-3">
      {/* Intro: explain what custom fields are and where they surface (G2). */}
      <p className="rounded-lg bg-secondary/40 px-3 py-2 text-[11px] leading-relaxed text-muted-foreground">
        {t('customization.editor.felder.intro')}
      </p>

      {/* Header: count + create + reset */}
      <div className="flex items-center justify-between gap-2">
        <span className="text-[11px] text-muted-foreground">
          {isLoading ? t('common.loading') : t('customization.fields.fieldCount', { count: effective.length })}
        </span>
        <div className="flex items-center gap-1">
          {isDraft && (
            <button
              type="button"
              onClick={() => resetDraftEntityFields(entity)}
              aria-label={t('customization.editor.begriffe.reset')}
              title={t('customization.editor.begriffe.reset')}
              className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
            >
              <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
            </button>
          )}
          <button
            type="button"
            onClick={() => setCreateOpen(true)}
            className="flex items-center gap-1.5 rounded-md bg-primary px-2.5 py-1.5 text-xs font-medium text-primary-foreground transition-opacity hover:opacity-90"
          >
            <Plus className="h-3.5 w-3.5" aria-hidden="true" />
            {t('customization.fields.newField')}
          </button>
        </div>
      </div>

      {/* Active fields */}
      {effective.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border px-4 py-8 text-center">
          <Type className="mx-auto h-6 w-6 text-muted-foreground/25" aria-hidden="true" />
          <p className="mt-2 text-xs font-medium text-foreground">{t('customization.fields.emptyTitle')}</p>
          <p className="mt-0.5 text-[11px] text-muted-foreground">{t('customization.fields.emptyHint')}</p>
        </div>
      ) : (
        <div className="flex flex-col gap-1.5">
          {effective.map((field) => {
            const Icon = typeIcon(field.type)
            const badge = badgeFor(field)
            return (
              <div
                key={field.id}
                role="button"
                tabIndex={0}
                onClick={() => setEditTarget(field)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setEditTarget(field) }
                }}
                className="group flex cursor-pointer items-center gap-2 rounded-lg border border-border bg-card px-2.5 py-2 transition-colors hover:border-primary/40 hover:bg-accent/40"
              >
                <Icon className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
                <div className="min-w-0 flex-1">
                  <p className={`truncate text-sm ${field.visible ? 'text-foreground' : 'text-muted-foreground line-through'}`}>
                    {field.label}
                  </p>
                  {/* Type + options → makes clear WHAT the field is (e.g. a
                      dropdown with these options), not just its name. */}
                  {/* Typ + woher die Auswahl kommt: eigene Optionen im Klartext, oder
                      der Name der gebundenen Werteliste — sonst sähe ein gebundenes
                      Feld aus wie eines ganz ohne Optionen. */}
                  <p className="truncate text-[11px] text-muted-foreground">
                    {t(`customization.fields.type.${field.type}`)}
                    {field.valueSetId ? (
                      <span className="text-muted-foreground/70">
                        {' · '}
                        {t('customization.fields.boundToSet', {
                          set: valueSetChoices.find((s) => s.id === field.valueSetId)?.name ?? field.valueSetId,
                        })}
                      </span>
                    ) : (
                      (field.type === 'select' || field.type === 'multi_select') && field.options.length > 0 && (
                        <span className="text-muted-foreground/70"> · {field.options.join(', ')}</span>
                      )
                    )}
                  </p>
                </div>

                {field.required && (
                  <span className="shrink-0 rounded bg-primary/10 px-1 py-0.5 text-[10px] font-medium text-primary">
                    {t('customization.fields.requiredBadge')}
                  </span>
                )}
                {badge && (
                  <span className="shrink-0 rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] font-medium text-amber-600 dark:text-amber-400">
                    {t(badge === 'added' ? 'customization.editor.felder.badgeNew' : 'customization.editor.felder.badgeChanged')}
                  </span>
                )}
                <button
                  type="button"
                  onClick={(e) => { e.stopPropagation(); toggleVisible(field.id) }}
                  aria-pressed={!field.visible}
                  aria-label={t('customization.editor.wertelisten.toggleHidden')}
                  title={t(field.visible ? 'customization.editor.wertelisten.optionActive' : 'customization.editor.wertelisten.optionHidden')}
                  className={`flex h-6 w-6 shrink-0 items-center justify-center rounded transition-colors hover:bg-secondary ${field.visible ? 'text-muted-foreground/50' : 'text-amber-600 dark:text-amber-400'}`}
                >
                  <EyeOff className="h-3.5 w-3.5" aria-hidden="true" />
                </button>
              </div>
            )
          })}
        </div>
      )}

      {/* Removed (staged deletions) — restorable, mirrors soft-delete UX */}
      {removed.length > 0 && (
        <div className="rounded-lg border border-dashed border-amber-500/30 bg-amber-500/5 px-3 py-2.5">
          <p className="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-amber-600 dark:text-amber-400">
            {t('customization.editor.felder.removedTitle')}
          </p>
          <div className="flex flex-col gap-1">
            {removed.map((field) => (
              <div key={field.id} className="flex items-center gap-2">
                <span className="min-w-0 flex-1 truncate text-xs text-muted-foreground line-through">{field.label}</span>
                <button
                  type="button"
                  onClick={() => restore(field)}
                  className="shrink-0 rounded px-1.5 py-0.5 text-[11px] font-medium text-primary transition-colors hover:bg-primary/10"
                >
                  {t('customization.editor.felder.restore')}
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      <p className="text-[11px] leading-relaxed text-muted-foreground">
        {t('customization.editor.felder.stagedHint')}
      </p>

      {/* Create modal */}
      {createOpen && (
        <FieldEditorModal
          open={createOpen}
          mode="create"
          entity={entity}
          valueSetChoices={valueSetChoices}
          onClose={() => setCreateOpen(false)}
          onCreate={handleCreate}
        />
      )}

      {/* Edit modal */}
      {editTarget && (
        <FieldEditorModal
          open={Boolean(editTarget)}
          mode="edit"
          entity={entity}
          field={editTarget}
          valueSetChoices={valueSetChoices}
          onClose={() => { setEditTarget(null); setDeleteCandidate(null) }}
          onUpdate={(input) => handleUpdate(editTarget.id, input)}
          onDeleteRequest={(field) => requestRemove(field)}
        />
      )}

      {/* Soft-delete consequence (only for persisted, in-use fields) */}
      {deleteCandidate && (
        <AlertDialog open onOpenChange={(v) => { if (!v) setDeleteCandidate(null) }}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{t('customization.fields.deleteInUseTitle')}</AlertDialogTitle>
              <AlertDialogDescription>
                {t('customization.fields.deleteInUseBody', { label: deleteCandidate.label })}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel onClick={() => setDeleteCandidate(null)}>{t('common.cancel')}</AlertDialogCancel>
              <AlertDialogAction
                onClick={() => removeField(deleteCandidate)}
                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                {t('common.delete')}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}
    </div>
  )
}

// ── Panel (entity switcher for multi-entity modules) ────────────────────────────

export function FelderPanel({ moduleKey }: { moduleKey: string }): React.ReactElement {
  const { t } = useTranslation()
  const { customFields, valueSets: draftSets } = useDraftConfig()
  const moduleDef = getEditorModule(moduleKey)
  const entities = moduleDef?.fieldEntities ?? []
  const [activeEntity, setActiveEntity] = useState<CustomFieldEntity>(entities[0] ?? 'crm_contact')

  // Value lists an Auswahl-Feld can bind to: the module's predefined ones plus any
  // created in the Wertelisten panel this session. Binding is what gives a NEW list
  // a place in the module — otherwise it has no column and stays invisible
  // (Darien 2026-08-04). Resolved here so the dialog can preview the options.
  const valueSetChoices = useMemo<ValueSetChoice[]>(() => {
    const ids = [
      ...(moduleDef?.valueSetIds ?? []),
      ...Object.keys(draftSets).filter((id) => !(moduleDef?.valueSetIds ?? []).includes(id)),
    ]
    return ids.flatMap((id) => {
      const resolved = resolveValueSet(id, false, draftSets)
      if (!resolved) return []
      return [{
        id,
        name: resolved.name,
        options: resolved.options
          .filter((o) => o.active)
          .map((o) => ({ id: o.id, label: o.label, color: o.color })),
      }]
    })
  }, [moduleDef, draftSets])

  if (entities.length === 0) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center px-6 text-center text-sm text-muted-foreground">
        {t('customization.fields.emptyTitle')}
      </div>
    )
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {moduleDef?.intake && (
        <p className="border-b bg-muted/30 px-4 py-2 text-[11px] leading-relaxed text-muted-foreground">
          {t('customization.fields.internalHint')}
        </p>
      )}
      {entities.length > 1 && (
        <div className="border-b px-4 py-2.5">
          <div role="radiogroup" aria-label={t('customization.fields.entityLabel')} className="flex flex-wrap gap-1">
            {entities.map((e) => {
              const active = activeEntity === e
              const dirty = customFields[e] !== undefined
              return (
                <button
                  key={e}
                  role="radio"
                  aria-checked={active}
                  onClick={() => setActiveEntity(e)}
                  className={`flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors ${active ? 'bg-primary text-primary-foreground' : 'bg-secondary text-muted-foreground hover:text-foreground'}`}
                >
                  {t(`customization.fields.entity.${e}`)}
                  {dirty && (
                    <span className={`h-1.5 w-1.5 rounded-full ${active ? 'bg-primary-foreground' : 'bg-amber-500'}`} aria-hidden="true" />
                  )}
                </button>
              )
            })}
          </div>
        </div>
      )}
      <EntityFieldsEditor key={activeEntity} entity={activeEntity} valueSetChoices={valueSetChoices} />
    </div>
  )
}
