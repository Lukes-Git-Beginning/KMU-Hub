/**
 * CustomFieldsTab — Tab „Felder" im Anpassungen-Hub.
 *
 * Zeigt alle Custom Field Definitions für die ausgewählte Entität.
 * Entity-Auswahl via Segmented-Control oben. Feldliste mit Typ-Badge,
 * Pflicht-Marker, inUse-Indikator.
 *
 * Klick auf Zeile öffnet FieldEditorModal (Detail-Modal-Muster).
 * „+ Feld anlegen" öffnet dasselbe Modal im Create-Modus.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Plus,
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
import { toast } from 'sonner'
import type { CustomFieldDefinition, CustomFieldEntity, CustomFieldType } from '@/mocks/data/custom-fields'
import {
  useCustomFields,
  useCreateCustomField,
  useUpdateCustomField,
  useDeleteCustomField,
} from '@/api/hooks/useCustomFields'
import { FieldEditorModal } from './FieldEditorModal'

// ── Entity-Konfiguration ──────────────────────────────────────────────────────

const ENTITIES: { key: CustomFieldEntity; labelKey: string }[] = [
  { key: 'work_task', labelKey: 'customization.fields.entity.work_task' },
  { key: 'crm_contact', labelKey: 'customization.fields.entity.crm_contact' },
  { key: 'crm_company', labelKey: 'customization.fields.entity.crm_company' },
  { key: 'crm_deal', labelKey: 'customization.fields.entity.crm_deal' },
  { key: 'crm_activity', labelKey: 'customization.fields.entity.crm_activity' },
]

// ── Typ-Icon-Mapping ──────────────────────────────────────────────────────────

function typeIcon(type: CustomFieldType) {
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

// ── Typ-Label-Key ─────────────────────────────────────────────────────────────

function typeLabelKey(type: CustomFieldType): string {
  return `customization.fields.type.${type}`
}

// ── Komponente ────────────────────────────────────────────────────────────────

export default function CustomFieldsTab() {
  const { t } = useTranslation()
  const [activeEntity, setActiveEntity] = useState<CustomFieldEntity>('work_task')
  const [editTarget, setEditTarget] = useState<CustomFieldDefinition | null>(null)
  const [createMode, setCreateMode] = useState(false)
  const [deleteCandidate, setDeleteCandidate] = useState<CustomFieldDefinition | null>(null)

  const { data: fields = [], isLoading } = useCustomFields(activeEntity)
  const createMutation = useCreateCustomField()
  const updateMutation = useUpdateCustomField(activeEntity)
  const deleteMutation = useDeleteCustomField(activeEntity)

  const handleCreate = async (values: Parameters<typeof createMutation.mutateAsync>[0]) => {
    await createMutation.mutateAsync(values)
    toast.success(t('customization.fields.created', { label: values.label }))
    setCreateMode(false)
  }

  const handleUpdate = async (
    id: string,
    values: Parameters<typeof updateMutation.mutateAsync>[0]['input'],
  ) => {
    await updateMutation.mutateAsync({ id, input: values })
    toast.success(t('customization.fields.updated'))
    setEditTarget(null)
  }

  const handleDelete = async (field: CustomFieldDefinition) => {
    await deleteMutation.mutateAsync(field.id)
    toast.success(t('customization.fields.deleted', { label: field.label }))
    setDeleteCandidate(null)
    setEditTarget(null)
  }

  return (
    <div className="px-6 py-6 space-y-6">
      {/* Entity-Auswahl */}
      <div className="space-y-1.5">
        <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {t('customization.fields.entityLabel')}
        </p>
        <div
          role="radiogroup"
          aria-label={t('customization.fields.entityLabel')}
          className="flex flex-wrap gap-1.5"
        >
          {ENTITIES.map((e) => (
            <button
              key={e.key}
              role="radio"
              aria-checked={activeEntity === e.key}
              onClick={() => setActiveEntity(e.key)}
              className={[
                'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                activeEntity === e.key
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-secondary text-muted-foreground hover:text-foreground',
              ].join(' ')}
            >
              {t(e.labelKey)}
            </button>
          ))}
        </div>
      </div>

      {/* Header mit Zähler + Anlegen-Button */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">
          {isLoading
            ? t('common.loading')
            : t('customization.fields.fieldCount', { count: fields.length })}
        </span>
        <button
          onClick={() => setCreateMode(true)}
          className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-opacity hover:opacity-90"
        >
          <Plus className="h-3.5 w-3.5" />
          {t('customization.fields.newField')}
        </button>
      </div>

      {/* Feld-Liste */}
      {isLoading ? (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-12 animate-pulse rounded-lg bg-secondary" />
          ))}
        </div>
      ) : fields.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-10 text-center">
          <Type className="mx-auto h-8 w-8 text-muted-foreground/25" />
          <p className="mt-3 text-sm font-medium text-foreground">
            {t('customization.fields.emptyTitle')}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {t('customization.fields.emptyHint')}
          </p>
        </div>
      ) : (
        <div className="space-y-1.5">
          {fields.map((field) => {
            const Icon = typeIcon(field.type)
            return (
              <div
                key={field.id}
                role="button"
                tabIndex={0}
                onClick={() => setEditTarget(field)}
                onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') setEditTarget(field) }}
                className="group flex cursor-pointer items-center gap-3 rounded-lg border border-border bg-card px-4 py-3 transition-colors hover:border-primary/40 hover:bg-accent/40"
              >
                <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />

                <span className="flex-1 text-sm font-medium text-foreground">
                  {field.label}
                </span>

                {/* Typ-Badge */}
                <span className="rounded bg-secondary px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground">
                  {t(typeLabelKey(field.type))}
                </span>

                {/* Pflicht-Marker */}
                {field.required && (
                  <span className="rounded bg-primary/10 px-1.5 py-0.5 text-[11px] font-medium text-primary">
                    {t('customization.fields.requiredBadge')}
                  </span>
                )}

                {/* inUse-Indikator */}
                {field.inUse && (
                  <span
                    className="h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-500"
                    title={t('customization.fields.inUseHint')}
                  />
                )}

                {/* Nicht-sichtbar-Indikator */}
                {!field.visible && (
                  <span className="text-[11px] text-muted-foreground/50">
                    {t('customization.fields.hidden')}
                  </span>
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* Create-Modal */}
      {createMode && (
        <FieldEditorModal
          open={createMode}
          mode="create"
          entity={activeEntity}
          onClose={() => setCreateMode(false)}
          onCreate={handleCreate}
        />
      )}

      {/* Edit-Modal */}
      {editTarget && (
        <FieldEditorModal
          open={Boolean(editTarget)}
          mode="edit"
          field={editTarget}
          entity={activeEntity}
          onClose={() => { setEditTarget(null); setDeleteCandidate(null) }}
          onUpdate={(input) => handleUpdate(editTarget.id, input)}
          onDeleteRequest={(field) => setDeleteCandidate(field)}
        />
      )}

      {/* Soft-Delete-Konsequenz-Dialog */}
      {deleteCandidate && (
        <DeleteConsequenceDialog
          field={deleteCandidate}
          onConfirm={() => handleDelete(deleteCandidate)}
          onCancel={() => setDeleteCandidate(null)}
        />
      )}
    </div>
  )
}

// ── Soft-Delete-Konsequenz-Dialog ─────────────────────────────────────────────

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

function DeleteConsequenceDialog({
  field,
  onConfirm,
  onCancel,
}: {
  field: CustomFieldDefinition
  onConfirm: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation()

  return (
    <AlertDialog open onOpenChange={(v) => { if (!v) onCancel() }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {field.inUse
              ? t('customization.fields.deleteInUseTitle')
              : t('customization.fields.deleteTitle')}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {field.inUse
              ? t('customization.fields.deleteInUseBody', { label: field.label })
              : t('customization.fields.deleteBody', { label: field.label })}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onCancel}>
            {t('common.cancel')}
          </AlertDialogCancel>
          <AlertDialogAction
            onClick={onConfirm}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {t('common.delete')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
