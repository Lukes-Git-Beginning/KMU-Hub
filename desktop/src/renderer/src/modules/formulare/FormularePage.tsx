import { useState, useMemo, useCallback, useEffect, type SyntheticEvent } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQueries } from '@tanstack/react-query'
import {
  DndContext,
  closestCenter,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  arrayMove,
  useSortable,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useTranslation } from 'react-i18next'
import { formatDate as libFormatDate, formatDateTime as libFormatDateTime } from '@/lib/format'
import {
  Search,
  Plus,
  FileText,
  ClipboardList,
  Inbox,
  LayoutTemplate,
  Eye,
  Edit,
  Trash2,
  Copy,
  Share2,
  Archive,
  ArrowLeft,
  GripVertical,
  ChevronDown,
  ChevronRight,
  ChevronLeft,
  Type,
  AlignLeft,
  CheckSquare,
  Circle,
  Calendar,
  Hash,
  Paperclip,
  Link2,
  Mail,
  Download,
  Zap,
  Split,
  Info,
  Globe,
  FileInput,
  ShieldCheck,
  ExternalLink,
  Send,
  Lock,
  Unlock,
  RotateCcw,
  QrCode,
  Code2,
  Check,
  LayoutGrid,
  LayoutList,
  Ban,
  Ruler,
  Star,
  BarChart3,
  X,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useFormulareStore,
  type DraftSchema,
  type FormField,
  type FormFieldType,
  type FieldPatternType,
} from '@/stores/formulare'
import { useFormularePrefsStore } from '@/stores/formularePrefs'
import {
  useFormulareTenantStore,
  DEFAULT_CONSENT_TEXT,
  DEFAULT_PRIVACY_URL,
} from '@/stores/formulareTenant'
import {
  useFormSchemas,
  useCreateFormSchema,
  useUpdateFormSchema,
  useDeleteFormSchema,
  useDuplicateFormSchema,
  useExportSubmissions,
  useUpdateSubmissionStatus,
  useCreateShareLink,
  useShareLinks,
  useUpdateShareLink,
  useDeleteShareLink,
  useFormStats,
  formulareKeys,
} from '@/api/hooks/useFormulare'
import { listSubmissions } from '@/api/formulare-client'
import type {
  FormSchema,
  FormShareLink,
  FormSubmission,
  FormSubmissionStatus,
  ShareChannel,
  ShareLinkStatus,
} from '@/api/formulare-types'
import {
  ItemActions,
  ConfirmDialog,
  EmptyState,
  DetailModal,
  PageHeader,
  SortMenu,
  type ActionItem,
  type SortDirection,
} from '@/components/shared'
import {
  listIntakeTargets,
  getIntakeTarget,
  getIntakeRoles,
  useIntakeSubmit,
} from '@/components/shared/intake'
import { useCapabilitySet, useHasCapability } from '@/hooks/useCapability'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

type TabKey = 'formulare' | 'eingänge' | 'vorlagen'

const formStatusLabelKeys: Record<FormSchema['status'], string> = {
  active: 'formulare.status.active',
  draft: 'formulare.status.draft',
  closed: 'formulare.status.closed',
  archived: 'formulare.status.archived',
}

const formStatusColors: Record<FormSchema['status'], string> = {
  active: 'bg-success-light text-success',
  draft: 'bg-secondary text-muted-foreground',
  closed: 'bg-warning-light text-warning',
  archived: 'bg-secondary text-muted-foreground opacity-80',
}

const submissionStatusLabelKeys: Record<FormSubmissionStatus, string> = {
  new: 'formulare.submission.status.new',
  read: 'formulare.submission.status.read',
  archived: 'formulare.submission.status.archived',
}

const submissionStatusColors: Record<FormSubmissionStatus, string> = {
  new: 'bg-info-light text-info',
  read: 'bg-secondary text-muted-foreground',
  archived: 'bg-warning-light text-warning',
}

const FIELD_TYPE_LABEL_KEYS: Record<FormFieldType, string> = {
  text: 'formulare.fieldType.text',
  textarea: 'formulare.fieldType.textarea',
  email: 'formulare.fieldType.email',
  select: 'formulare.fieldType.select',
  checkbox: 'formulare.fieldType.checkbox',
  radio: 'formulare.fieldType.radio',
  date: 'formulare.fieldType.date',
  number: 'formulare.fieldType.number',
  file: 'formulare.fieldType.file',
  consent: 'formulare.fieldType.consent',
  rating: 'formulare.fieldType.rating',
}

const FIELD_TYPE_ICONS: Record<FormFieldType, typeof Type> = {
  text: Type,
  textarea: AlignLeft,
  email: Mail,
  select: ChevronDown,
  checkbox: CheckSquare,
  radio: Circle,
  date: Calendar,
  number: Hash,
  file: Paperclip,
  consent: ShieldCheck,
  rating: Star,
}

const FIELD_TYPE_OPTION_KEYS: { value: FormFieldType; labelKey: string }[] = [
  { value: 'text', labelKey: 'formulare.fieldType.text' },
  { value: 'textarea', labelKey: 'formulare.fieldType.textarea' },
  { value: 'email', labelKey: 'formulare.fieldType.email' },
  { value: 'select', labelKey: 'formulare.fieldType.selectDropdown' },
  { value: 'radio', labelKey: 'formulare.fieldType.radioOption' },
  { value: 'checkbox', labelKey: 'formulare.fieldType.checkbox' },
  { value: 'date', labelKey: 'formulare.fieldType.date' },
  { value: 'number', labelKey: 'formulare.fieldType.number' },
  { value: 'file', labelKey: 'formulare.fieldType.fileUpload' },
  { value: 'rating', labelKey: 'formulare.fieldType.rating' },
  { value: 'consent', labelKey: 'formulare.fieldType.consent' },
]

// ---------------------------------------------------------------------------
// FT-2a — field validation: pattern presets + value checker
// ---------------------------------------------------------------------------

/**
 * Text-field pattern presets. The field stores only the preset key
 * (FieldPatternType); the regex lives here so we never persist a free-form
 * regex — too error-prone for KMU users. DACH-tuned defaults.
 */
const FIELD_PATTERN_PRESETS: Record<
  Exclude<FieldPatternType, 'free'>,
  { regex: RegExp; errorKey: string }
> = {
  plz: { regex: /^\d{5}$/, errorKey: 'formulare.validation.error.plz' },
  phone: { regex: /^[+0][\d\s/()-]{5,19}$/, errorKey: 'formulare.validation.error.phone' },
  iban: { regex: /^[A-Z]{2}\d{2}[A-Z0-9]{11,30}$/, errorKey: 'formulare.validation.error.iban' },
}

const PATTERN_TYPE_OPTION_KEYS: { value: FieldPatternType; labelKey: string }[] = [
  { value: 'free', labelKey: 'formulare.validation.pattern.free' },
  { value: 'plz', labelKey: 'formulare.validation.pattern.plz' },
  { value: 'phone', labelKey: 'formulare.validation.pattern.phone' },
  { value: 'iban', labelKey: 'formulare.validation.pattern.iban' },
]

/** A localized validation failure: an i18n key plus interpolation params. */
interface ValidationError {
  key: string
  params?: Record<string, unknown>
}

/**
 * Validate one field value against its rules (FT-2a). Returns the first
 * violation as an i18n key (+params) or null when valid. Empty values pass —
 * the `required` check is the caller's responsibility.
 */
function validateFieldValue(field: FormField, rawValue: unknown): ValidationError | null {
  const v = field.validation
  const value =
    typeof rawValue === 'string' ? rawValue : rawValue == null ? '' : String(rawValue)
  if (value.trim() === '') return null

  if (field.type === 'text' || field.type === 'textarea') {
    if (v?.minLength != null && value.length < v.minLength) {
      return { key: 'formulare.validation.error.minLength', params: { min: v.minLength } }
    }
    if (v?.maxLength != null && value.length > v.maxLength) {
      return { key: 'formulare.validation.error.maxLength', params: { max: v.maxLength } }
    }
  }
  if (field.type === 'number') {
    const num = Number(value)
    if (!Number.isNaN(num)) {
      if (v?.min != null && num < v.min) {
        return { key: 'formulare.validation.error.min', params: { min: v.min } }
      }
      if (v?.max != null && num > v.max) {
        return { key: 'formulare.validation.error.max', params: { max: v.max } }
      }
    }
  }
  if (field.type === 'text' && v?.patternType && v.patternType !== 'free') {
    const preset = FIELD_PATTERN_PRESETS[v.patternType]
    const normalized =
      v.patternType === 'iban' ? value.replace(/\s+/g, '').toUpperCase() : value.trim()
    if (!preset.regex.test(normalized)) return { key: preset.errorKey }
  }
  return null
}

/**
 * FO-6 — evaluate a field's conditional-logic rule against a set of answers.
 * Returns true when the field should be visible (no rule = always visible).
 * Module-scoped so both the page component and the interactive fill preview
 * (`PreviewFillForm`) apply the exact same visibility logic.
 */
function evaluateCondition(
  logic: FormField['conditionalLogic'],
  answers: Record<string, unknown>,
): boolean {
  if (!logic) return true
  const sourceVal = String(answers[logic.fieldId] ?? '')
  switch (logic.operator) {
    case 'equals':
      return sourceVal === logic.value
    case 'not_equals':
      return sourceVal !== logic.value
    case 'contains':
      return sourceVal.toLowerCase().includes(logic.value.toLowerCase())
    default:
      return true
  }
}

// ---------------------------------------------------------------------------
// FT-2b — rating field renderer (1–5 stars or 1–10 NPS scale). Reused by the
// builder preview (disabled), the submission detail (read-only value) and the
// interactive fill preview (`light` = public-page styling, not theme tokens).
// ---------------------------------------------------------------------------

function RatingInput({
  scale,
  value,
  onChange,
  disabled,
  light,
}: {
  scale: number
  value: number
  onChange?: (v: number) => void
  disabled?: boolean
  light?: boolean
}) {
  const steps = Array.from({ length: Math.max(1, scale) }, (_, i) => i + 1)

  if (scale >= 10) {
    return (
      <div className="flex flex-wrap gap-1.5">
        {steps.map((n) => {
          const selected = value === n
          return (
            <button
              key={n}
              type="button"
              disabled={disabled}
              onClick={() => onChange?.(n)}
              className={`h-8 w-8 rounded-md border text-xs font-medium transition-colors ${
                selected
                  ? 'border-primary bg-primary text-primary-foreground'
                  : light
                    ? 'border-gray-300 text-gray-600 hover:bg-gray-50'
                    : 'border-border text-muted-foreground hover:bg-secondary'
              } ${disabled ? 'cursor-default' : ''}`}
            >
              {n}
            </button>
          )
        })}
      </div>
    )
  }

  return (
    <div className="flex items-center gap-1">
      {steps.map((n) => {
        const filled = value >= n
        const emptyCls = light ? 'text-gray-300' : 'text-muted-foreground/40'
        return (
          <button
            key={n}
            type="button"
            disabled={disabled}
            onClick={() => onChange?.(n)}
            className={`transition-transform ${disabled ? 'cursor-default' : 'hover:scale-110'}`}
            aria-label={String(n)}
          >
            <Star
              className={`h-6 w-6 ${
                filled
                  ? light
                    ? 'fill-amber-400 text-amber-400'
                    : 'fill-warning text-warning'
                  : emptyCls
              }`}
            />
          </button>
        )
      })}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Helper: map FormSchema → DraftSchema for the editor
// ---------------------------------------------------------------------------

export function schemaToDraft(schema: FormSchema): DraftSchema {
  return {
    id: schema.id,
    title: schema.title,
    description: schema.description,
    // Cast fields — backend fields are compatible at runtime
    fields: schema.fields as FormField[],
    isPublic: schema.isPublic,
    pageCount: schema.pageCount,
    thankYouMessage: schema.thankYouMessage,
    redirectUrl: schema.redirectUrl,
    notifications: schema.notifications
      ? {
          recipients: [...schema.notifications.recipients],
          confirmToSubmitter: schema.notifications.confirmToSubmitter,
        }
      : { recipients: [], confirmToSubmitter: false },
    actions: [],
    intakeTargetId: schema.intakeTargetId,
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function FormularePage({ embedded = false }: { embedded?: boolean } = {}) {
  const { t } = useTranslation()

  // -------------------------------------------------------------------------
  // RBAC capability checks (R-3) — ALL hooks before any early return
  // -------------------------------------------------------------------------
  const { has: capHas, ready: capReady } = useCapabilitySet()
  const canSchemasRead    = useHasCapability('formulare:schemas:read')
  const canSchemasCreate  = useHasCapability('formulare:schemas:create')
  const canSchemasEdit    = useHasCapability('formulare:schemas:edit')
  const canSchemasDelete  = useHasCapability('formulare:schemas:delete')
  const canSchemasPublish = useHasCapability('formulare:schemas:publish')
  const canSubsRead       = useHasCapability('formulare:submissions:read')
  const canSubsWrite      = useHasCapability('formulare:submissions:write')
  const canShareManage    = useHasCapability('formulare:share:manage')
  const canExportRun      = useHasCapability('formulare:export:run')

  // Tab gating: formulare+vorlagen need schemas:read, eingänge needs submissions:read
  const TAB_CAPABILITY: Record<TabKey, string> = {
    'formulare': 'formulare:schemas:read',
    'eingänge': 'formulare:submissions:read',
    'vorlagen': 'formulare:schemas:read',
  }
  const visibleTabs = (Object.keys(TAB_CAPABILITY) as TabKey[]).filter(
    (key) => !capReady || capHas(TAB_CAPABILITY[key]),
  )

  // -------------------------------------------------------------------------
  // Server state (React Query)
  // -------------------------------------------------------------------------

  const {
    data: schemasData,
    isLoading: schemasLoading,
    isError: schemasError,
  } = useFormSchemas()

  const schemasItems = useMemo(() => schemasData?.items ?? [], [schemasData])
  const activeForms = useMemo(
    () => schemasItems.filter((s) => !s.isTemplate),
    [schemasItems],
  )
  const templates = useMemo(
    () => schemasItems.filter((s) => s.isTemplate),
    [schemasItems],
  )
  // Stable reference for rendering
  const schemas = schemasItems

  // P2 — dispatch a form submission to its bound intake target (e.g. create a
  // Helpdesk ticket). Used by the interactive preview to prove the vertical slice.
  const dispatchIntake = useIntakeSubmit()

  // Mutations
  const createSchema = useCreateFormSchema()
  const updateSchema = useUpdateFormSchema()
  const deleteSchema = useDeleteFormSchema()
  const duplicateSchema = useDuplicateFormSchema()
  const exportSubs = useExportSubmissions()
  const updateSubStatus = useUpdateSubmissionStatus()
  const createShareLink = useCreateShareLink()
  const updateShareLinkMut = useUpdateShareLink()
  const deleteShareLinkMut = useDeleteShareLink()

  // -------------------------------------------------------------------------
  // Client state (Zustand — editor draft)
  // -------------------------------------------------------------------------

  const {
    draft,
    openDraft,
    closeDraft,
    updateDraftMeta,
    addField,
    removeField,
    duplicateField,
    reorderFields,
    updateField,
  } = useFormulareStore()

  // DnD sensors for the field-builder list (pointer drag + keyboard a11y).
  const fieldSensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const handleFieldDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event
      if (!over || active.id === over.id || !draft) return
      const oldIndex = draft.fields.findIndex((f) => f.id === active.id)
      const newIndex = draft.fields.findIndex((f) => f.id === over.id)
      if (oldIndex === -1 || newIndex === -1) return
      reorderFields(arrayMove(draft.fields, oldIndex, newIndex))
    },
    [draft, reorderFields],
  )

  // -------------------------------------------------------------------------
  // Local UI state
  // -------------------------------------------------------------------------

  const formStatusLabels = useMemo(
    () =>
      Object.fromEntries(
        Object.entries(formStatusLabelKeys).map(([k, v]) => [k, t(v)]),
      ) as Record<FormSchema['status'], string>,
    [t],
  )

  const submissionStatusLabels = useMemo(
    () =>
      Object.fromEntries(
        Object.entries(submissionStatusLabelKeys).map(([k, v]) => [k, t(v)]),
      ) as Record<FormSubmissionStatus, string>,
    [t],
  )

  const FIELD_TYPE_LABELS = useMemo(
    () =>
      Object.fromEntries(
        Object.entries(FIELD_TYPE_LABEL_KEYS).map(([k, v]) => [k, t(v)]),
      ) as Record<FormFieldType, string>,
    [t],
  )

  // Module preferences (personal + tenant) — see formularePrefs store.
  const prefsDefaultTab = useFormularePrefsStore((s) => s.defaultTab)
  const prefsExportFormat = useFormularePrefsStore((s) => s.defaultExportFormat)
  const prefsConsentText = useFormulareTenantStore((s) => s.defaultConsentText)
  const prefsPrivacyUrl = useFormulareTenantStore((s) => s.defaultPrivacyUrl)
  const prefsThankYouMessage = useFormulareTenantStore((s) => s.defaultThankYouMessage)
  // FO-4 — tenant notification default, used to seed new forms' recipient list.
  const prefsNotifyOnSubmission = useFormulareTenantStore((s) => s.notifyOnSubmission)
  const prefsNotifyEmail = useFormulareTenantStore((s) => s.notifyEmail)
  // FT-5 — the grid/list toggle reads + writes the persisted personal default.
  const formView = useFormularePrefsStore((s) => s.defaultFormView)
  const setFormView = useFormularePrefsStore((s) => s.setDefaultFormView)

  // Tab & search — initial tab honours the personal default.
  const [tab, setTab] = useState<TabKey>(prefsDefaultTab)

  // Fallback-Effect: if the currently active tab loses visibility (RBAC),
  // switch to the first still-visible tab. Guard on `capReady` to avoid
  // a premature redirect while permissions are still loading.
  useEffect(() => {
    if (!capReady) return
    if (visibleTabs.length > 0 && !visibleTabs.includes(tab)) setTab(visibleTabs[0])
  }, [capReady, visibleTabs, tab])

  const [search, setSearch] = useState('')
  // FT-4 — Formulare tab: status quick-filter (grid/list view via prefs above)
  const [formStatusFilter, setFormStatusFilter] = useState<'all' | FormSchema['status']>('all')

  // Detail modal for a single submission
  const [selectedSubmission, setSelectedSubmission] =
    useState<FormSubmission | null>(null)

  // Detail modal for a single form (360° view + actions)
  const [selectedForm, setSelectedForm] = useState<FormSchema | null>(null)
  // FT-1 — tab inside the form detail modal (Details / Eingänge; FD-2 adds Verteilung)
  const [formDetailTab, setFormDetailTab] = useState<
    'details' | 'eingaenge' | 'auswertung' | 'verteilung'
  >('details')
  // FT-1 — template detail modal (clickable template cards)
  const [selectedTemplate, setSelectedTemplate] = useState<FormSchema | null>(null)
  // FT-5 — template delete confirmation
  const [templateToDelete, setTemplateToDelete] = useState<FormSchema | null>(null)
  // FT-1 — confirm archiving a form that still has unread submissions
  const [archiveConfirm, setArchiveConfirm] = useState<{
    schema: FormSchema
    newCount: number
  } | null>(null)
  // FD-2 — confirm deleting a single share link from the distribution overview
  const [shareLinkToDelete, setShareLinkToDelete] = useState<FormShareLink | null>(null)

  // Standalone public-facing form preview (decoupled from the editor draft)
  const [previewSchema, setPreviewSchema] = useState<FormSchema | null>(null)

  // Expanded submission groups
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set())

  // Dialogs
  const [confirmDelete, setConfirmDelete] = useState<FormSchema | null>(null)
  const [showNewFormDialog, setShowNewFormDialog] = useState(false)
  const [showShareDialog, setShowShareDialog] = useState<FormSchema | null>(null)
  // FD-0 — lifecycle: close dialog (captures the optional closed message)
  const [closeDialogForm, setCloseDialogForm] = useState<FormSchema | null>(null)
  const [closeMessageInput, setCloseMessageInput] = useState('')
  // FD-1 — share dialog: channel + settings, then the generated token link
  const [shareChannel, setShareChannel] = useState<ShareChannel>('link')
  const [shareExpiry, setShareExpiry] = useState('')
  const [shareMaxSubs, setShareMaxSubs] = useState('')
  const [shareCreatedLink, setShareCreatedLink] = useState<FormShareLink | null>(null)
  const [shareCopied, setShareCopied] = useState(false)
  const [showFieldConfigDialog, setShowFieldConfigDialog] = useState<{
    field: FormField
  } | null>(null)
  const [showAddFieldMenu, setShowAddFieldMenu] = useState(false)

  // New form dialog state
  const [newFormName, setNewFormName] = useState('')
  const [newFormDescription, setNewFormDescription] = useState('')
  const [newFormTemplateId, setNewFormTemplateId] = useState('')

  // Field config dialog state
  const [configLabel, setConfigLabel] = useState('')
  const [configRequired, setConfigRequired] = useState(false)
  const [configPlaceholder, setConfigPlaceholder] = useState('')
  const [configOptions, setConfigOptions] = useState('')
  const [configConsentText, setConfigConsentText] = useState('')
  const [configPrivacyUrl, setConfigPrivacyUrl] = useState('')

  // FT-2a — validation rule state (stored as strings, parsed on save)
  const [configMinLength, setConfigMinLength] = useState('')
  const [configMaxLength, setConfigMaxLength] = useState('')
  const [configMin, setConfigMin] = useState('')
  const [configMax, setConfigMax] = useState('')
  const [configPatternType, setConfigPatternType] = useState<FieldPatternType>('free')
  // FT-2b — rating scale (5 = stars, 10 = NPS)
  const [configRatingScale, setConfigRatingScale] = useState(5)
  // P2 — intake field role (subject/priority/… or '' = extra/custom field)
  const [configRole, setConfigRole] = useState('')

  // 10.1 — Conditional logic state
  const [configConditionalEnabled, setConfigConditionalEnabled] =
    useState(false)
  const [configConditionalFieldId, setConfigConditionalFieldId] = useState('')
  const [configConditionalOperator, setConfigConditionalOperator] = useState<
    'equals' | 'not_equals' | 'contains'
  >('equals')
  const [configConditionalValue, setConfigConditionalValue] = useState('')

  // 10.2 — Multi-page preview state
  const [previewPage, setPreviewPage] = useState(0)

  // 10.4 — Public preview state
  const [showPublicPreview, setShowPublicPreview] = useState(false)

  // FO-4 — staged email being typed into the notification recipient editor.
  const [newRecipient, setNewRecipient] = useState('')


  // Submissions are fetched eagerly for every active form (one query per form,
  // deduped with the per-group SubmissionsPanel via a shared query key). This
  // keeps the header + stat cards accurate on first load instead of only after
  // a group is expanded.
  const submissionQueries = useQueries({
    queries: activeForms.map((s) => ({
      queryKey: formulareKeys.submissions.listBySchema(s.id),
      queryFn: () => listSubmissions(s.id),
    })),
  })

  const submissionsBySchemaId = useMemo(() => {
    const map: Record<string, FormSubmission[]> = {}
    activeForms.forEach((s, i) => {
      const items = submissionQueries[i]?.data?.items
      if (items) map[s.id] = items
    })
    return map
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeForms, submissionQueries.map((q) => q.dataUpdatedAt).join(',')])

  // FD-2 — share links for the open form (powers the Verteilung tab + count).
  const formShareLinks = useShareLinks(selectedForm?.id ?? '')
  const shareLinkCount = formShareLinks.data?.length ?? 0

  // ---------------------------------------------------------------------------
  // Derived data
  // ---------------------------------------------------------------------------

  const filteredForms = useMemo(() => {
    let list = activeForms
    if (formStatusFilter !== 'all') {
      list = list.filter((f) => f.status === formStatusFilter)
    }
    if (search) {
      const q = search.toLowerCase()
      list = list.filter(
        (f) =>
          f.title.toLowerCase().includes(q) ||
          f.description.toLowerCase().includes(q),
      )
    }
    return list
  }, [activeForms, search, formStatusFilter])

  // FT-4 — counts per status for the quick-filter chips.
  const formStatusCounts = useMemo(
    () => ({
      all: activeForms.length,
      draft: activeForms.filter((f) => f.status === 'draft').length,
      active: activeForms.filter((f) => f.status === 'active').length,
      closed: activeForms.filter((f) => f.status === 'closed').length,
      archived: activeForms.filter((f) => f.status === 'archived').length,
    }),
    [activeForms],
  )

  // All known submissions flattened
  const allSubmissions = useMemo(
    () => Object.values(submissionsBySchemaId).flat(),
    [submissionsBySchemaId],
  )

  const filteredSubmissions = useMemo(() => {
    if (!search) return allSubmissions
    const q = search.toLowerCase()
    return allSubmissions.filter((s) =>
      (s.submittedBy ?? '').toLowerCase().includes(q),
    )
  }, [allSubmissions, search])

  const submissionsByForm = useMemo(() => {
    const grouped: Record<
      string,
      { schemaTitle: string; items: FormSubmission[] }
    > = {}
    for (const sub of filteredSubmissions) {
      const schemaId = sub.formSchemaId ?? 'unknown'
      if (!grouped[schemaId]) {
        const schema = schemas.find((s) => s.id === schemaId)
        grouped[schemaId] = {
          schemaTitle: schema?.title ?? schemaId,
          items: [],
        }
      }
      grouped[schemaId].items.push(sub)
    }
    return grouped
  }, [filteredSubmissions, schemas])

  const activeFormCount = activeForms.filter((f) => f.status === 'active').length

  const weekAgo = useMemo(() => {
    const d = new Date()
    d.setDate(d.getDate() - 7)
    return d.toISOString()
  }, [])

  const weeklySubmissionCount = allSubmissions.filter(
    (s) => s.submittedAt >= weekAgo,
  ).length

  const newSubmissionCount = allSubmissions.filter(
    (s) => s.status === 'new',
  ).length

  // -------------------------------------------------------------------------
  // Editor helpers (operate on store draft)
  // -------------------------------------------------------------------------

  const openEditor = (schema: FormSchema) => {
    openDraft(schemaToDraft(schema))
    setPreviewPage(0)
    setShowAddFieldMenu(false)
  }

  // Ticket-Intake A4 — deep-link target: the editor's Kanäle panel navigates to
  // /formulare?edit=<id> so the concrete bound form opens directly (both windows
  // share the app's hash router). Clear the param once handled so a refresh/back
  // doesn't reopen it.
  const [editParams, setEditParams] = useSearchParams()
  useEffect(() => {
    const editId = editParams.get('edit')
    if (!editId || schemasLoading) return
    const schema = schemasItems.find((s) => s.id === editId)
    if (!schema) return
    openEditor(schema)
    const next = new URLSearchParams(editParams)
    next.delete('edit')
    setEditParams(next, { replace: true })
    // openEditor is stable in effect (guarded by the param delete above)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editParams, schemasLoading, schemasItems, setEditParams])

  // FT-1 — open the form detail modal on its Details tab.
  const openFormDetail = (schema: FormSchema) => {
    setFormDetailTab('details')
    setSelectedForm(schema)
  }

  const closeEditor = () => {
    closeDraft()
    setShowAddFieldMenu(false)
  }

  const saveEditor = async () => {
    if (!draft?.id) return
    try {
      await updateSchema.mutateAsync({
        id: draft.id,
        title: draft.title,
        description: draft.description,
        fields: draft.fields as Parameters<typeof updateSchema.mutateAsync>[0]['fields'],
        isPublic: draft.isPublic,
        pageCount: draft.pageCount,
        thankYouMessage: draft.thankYouMessage?.trim() || undefined,
        redirectUrl: draft.redirectUrl?.trim() || undefined,
        notifications: draft.notifications,
        intakeTargetId: draft.intakeTargetId,
      })
      toast.success(t('formulare.toast.gespeichert'))
      closeDraft()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('common.error'))
    }
  }

  // FO-4 — notification recipient management (operates on the draft).
  const addRecipient = () => {
    if (!draft) return
    const email = newRecipient.trim()
    if (!email) return
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      toast.error(t('formulare.notify.invalidEmail'))
      return
    }
    if (draft.notifications.recipients.includes(email)) {
      setNewRecipient('')
      return
    }
    updateDraftMeta({
      notifications: {
        ...draft.notifications,
        recipients: [...draft.notifications.recipients, email],
      },
    })
    setNewRecipient('')
  }

  const removeRecipient = (email: string) => {
    if (!draft) return
    updateDraftMeta({
      notifications: {
        ...draft.notifications,
        recipients: draft.notifications.recipients.filter((r) => r !== email),
      },
    })
  }

  const toggleConfirmToSubmitter = () => {
    if (!draft) return
    updateDraftMeta({
      notifications: {
        ...draft.notifications,
        confirmToSubmitter: !draft.notifications.confirmToSubmitter,
      },
    })
  }

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------

  const formatDate = (d: string) =>
    libFormatDate(d.includes('T') ? d : d + 'T00:00:00')

  const formatDateTime = (d: string) =>
    libFormatDateTime(d, { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })

  // -------------------------------------------------------------------------
  // FD-0 — lifecycle transitions (draft → active → closed → archived)
  // -------------------------------------------------------------------------

  /** Real fields (page breaks are pseudo-fields and don't count). */
  const realFieldCount = (schema: FormSchema) =>
    schema.fields.filter((f) => (f as FormField).label !== '__page_break__')
      .length

  /** Only live forms can be shared — draft must be published, closed/archived can't. */
  const isShareable = (schema: FormSchema) => schema.status === 'active'

  const applyStatus = (
    schema: FormSchema,
    status: FormSchema['status'],
    toastKey: string,
    extra?: { closedMessage?: string },
  ) => {
    updateSchema.mutate(
      { id: schema.id, status, ...(extra ?? {}) },
      {
        onSuccess: () => {
          toast.success(t(toastKey, { name: schema.title }))
          // Keep an open detail modal in sync with the new status.
          setSelectedForm((cur) =>
            cur && cur.id === schema.id ? { ...cur, status, ...(extra ?? {}) } : cur,
          )
        },
        onError: (err) =>
          toast.error(err instanceof Error ? err.message : t('common.error')),
      },
    )
  }

  const publishForm = (schema: FormSchema) => {
    if (realFieldCount(schema) === 0) {
      toast.error(t('formulare.lifecycle.publishNoFields'))
      return
    }
    applyStatus(schema, 'active', 'formulare.lifecycle.published')
  }

  const reopenForm = (schema: FormSchema) =>
    applyStatus(schema, 'active', 'formulare.lifecycle.reopened')

  const restoreForm = (schema: FormSchema) =>
    applyStatus(schema, 'draft', 'formulare.lifecycle.restored')

  const doArchive = (schema: FormSchema) =>
    applyStatus(schema, 'archived', 'formulare.toast.archiviert')

  /** FT-1 — guard: confirm before archiving a form with unread submissions. */
  const archiveForm = (schema: FormSchema) => {
    const newCount = (submissionsBySchemaId[schema.id] ?? []).filter(
      (s) => s.status === 'new',
    ).length
    if (newCount > 0) {
      setArchiveConfirm({ schema, newCount })
      return
    }
    doArchive(schema)
  }

  const openCloseDialog = (schema: FormSchema) => {
    setCloseMessageInput(schema.closedMessage ?? '')
    setCloseDialogForm(schema)
  }

  const confirmCloseForm = () => {
    if (!closeDialogForm) return
    applyStatus(closeDialogForm, 'closed', 'formulare.lifecycle.closed', {
      closedMessage: closeMessageInput.trim() || undefined,
    })
    setCloseDialogForm(null)
  }

  // -------------------------------------------------------------------------
  // FD-1 — share dialog (real token link, clipboard, channels, settings)
  // -------------------------------------------------------------------------

  const resetShareDialog = () => {
    setShareChannel('link')
    setShareExpiry('')
    setShareMaxSubs('')
    setShareCreatedLink(null)
    setShareCopied(false)
  }

  /** Guarded share entry point — blocks non-live forms with an explanation. */
  const requestShare = (schema: FormSchema) => {
    if (!isShareable(schema)) {
      toast.error(t('formulare.lifecycle.shareNotActive'))
      return
    }
    resetShareDialog()
    setShowShareDialog(schema)
  }

  const closeShareDialog = () => {
    setShowShareDialog(null)
    resetShareDialog()
  }

  /** Embed channel copies an iframe snippet; everything else copies the URL. */
  const shareValueOf = (link: FormShareLink, title = ''): string =>
    link.channel === 'embed'
      ? `<iframe src="${link.url}" width="100%" height="640" style="border:0" title="${title}"></iframe>`
      : link.url

  // FD-2 — distribution overview actions (toggle / delete a share link).
  const toggleShareLink = (link: FormShareLink) => {
    const next: ShareLinkStatus = link.status === 'disabled' ? 'active' : 'disabled'
    updateShareLinkMut.mutate(
      { id: link.id, formSchemaId: link.formSchemaId, status: next },
      {
        onSuccess: () =>
          toast.success(
            next === 'disabled'
              ? t('formulare.share.toast.deactivated')
              : t('formulare.share.toast.reactivated'),
          ),
        onError: (err) =>
          toast.error(err instanceof Error ? err.message : t('common.error')),
      },
    )
  }

  const confirmDeleteShareLink = () => {
    if (!shareLinkToDelete) return
    const link = shareLinkToDelete
    deleteShareLinkMut.mutate(
      { id: link.id, formSchemaId: link.formSchemaId },
      {
        onSuccess: () => toast.success(t('formulare.share.toast.deleted')),
        onError: (err) =>
          toast.error(err instanceof Error ? err.message : t('common.error')),
      },
    )
    setShareLinkToDelete(null)
  }

  const copyToClipboard = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value)
      setShareCopied(true)
      toast.success(t('formulare.toast.linkKopiert'))
      setTimeout(() => setShareCopied(false), 2000)
    } catch {
      toast.error(t('common.error'))
    }
  }

  const handleCreateShareLink = () => {
    if (!showShareDialog) return
    createShareLink.mutate(
      {
        formSchemaId: showShareDialog.id,
        channel: shareChannel,
        expiresAt: shareExpiry
          ? new Date(`${shareExpiry}T23:59:59`).toISOString()
          : null,
        maxSubmissions: shareMaxSubs ? Number(shareMaxSubs) : null,
      },
      {
        onSuccess: (link) => {
          setShareCreatedLink(link)
          void copyToClipboard(shareValueOf(link, showShareDialog?.title))
        },
        onError: (err) =>
          toast.error(err instanceof Error ? err.message : t('common.error')),
      },
    )
  }

  /** Status-dependent lifecycle entries — only when schemas:publish is granted. */
  const lifecycleItems = (schema: FormSchema): ActionItem[] => {
    if (!canSchemasPublish) return []
    switch (schema.status) {
      case 'draft':
        return [
          {
            label: t('formulare.lifecycle.publish'),
            icon: Send,
            onClick: () => publishForm(schema),
          },
        ]
      case 'active':
        return [
          {
            label: t('formulare.lifecycle.close'),
            icon: Lock,
            onClick: () => openCloseDialog(schema),
          },
          {
            label: t('formulare.actions.archivieren'),
            icon: Archive,
            onClick: () => archiveForm(schema),
          },
        ]
      case 'closed':
        return [
          {
            label: t('formulare.lifecycle.reopen'),
            icon: Unlock,
            onClick: () => reopenForm(schema),
          },
          {
            label: t('formulare.actions.archivieren'),
            icon: Archive,
            onClick: () => archiveForm(schema),
          },
        ]
      case 'archived':
        return [
          {
            label: t('formulare.lifecycle.restore'),
            icon: RotateCcw,
            onClick: () => restoreForm(schema),
          },
        ]
      default:
        return []
    }
  }

  const getFormActions = useCallback(
    (schema: FormSchema): ActionItem[] => [
      ...(canSchemasEdit
        ? [
            {
              label: t('formulare.actions.bearbeiten'),
              icon: Edit,
              onClick: () => openEditor(schema),
            },
          ]
        : []),
      ...(canSchemasCreate
        ? [
            {
              label: t('formulare.actions.duplizieren'),
              icon: Copy,
              onClick: () => {
                duplicateSchema.mutate(
                  { id: schema.id },
                  {
                    onSuccess: () =>
                      toast.success(
                        t('formulare.toast.dupliziert', { name: schema.title }),
                      ),
                    onError: (err) =>
                      toast.error(
                        err instanceof Error ? err.message : t('common.error'),
                      ),
                  },
                )
              },
            },
          ]
        : []),
      // Share opens the link-create dialog — that IS share management.
      ...(canShareManage
        ? [
            {
              label: t('formulare.actions.teilen'),
              icon: Share2,
              onClick: () => requestShare(schema),
              disabled: !isShareable(schema),
            },
          ]
        : []),
      ...(canSchemasEdit
        ? [
            {
              label: t('formulare.vorlagen.saveAsTemplate'),
              icon: LayoutTemplate,
              onClick: () => saveAsTemplate(schema),
            },
          ]
        : []),
      ...lifecycleItems(schema),
      ...(canSchemasDelete
        ? [
            {
              separator: true as const,
              label: t('common.delete'),
              icon: Trash2,
              variant: 'destructive' as const,
              onClick: () => setConfirmDelete(schema),
            },
          ]
        : []),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, duplicateSchema, updateSchema, closeMessageInput, canSchemasEdit, canSchemasCreate, canSchemasDelete, canSchemasPublish],
  )

  const handleDeleteForm = (schema: FormSchema) => {
    deleteSchema.mutate(schema.id, {
      onSuccess: () => {
        setConfirmDelete(null)
        toast.success(t('formulare.toast.geloescht', { name: schema.title }))
      },
      onError: (err) =>
        toast.error(err instanceof Error ? err.message : t('common.error')),
    })
  }

  // Real CSV/XLSX download via the export endpoint (blob handled in the hook).
  const handleExportSubmissions = (
    schema: FormSchema,
    format: 'csv' | 'xlsx',
  ) => {
    exportSubs.mutate(
      { formSchemaId: schema.id, format },
      {
        onSuccess: () =>
          toast.success(
            t('formulare.export.downloadStarted', {
              name: schema.title,
              format: format.toUpperCase(),
            }),
          ),
        onError: (err) =>
          toast.error(err instanceof Error ? err.message : t('common.error')),
      },
    )
  }

  // Shared submission status mutation (Eingänge tab + form-detail Eingänge tab).
  const handleSubmissionStatus = (
    sub: FormSubmission,
    status: FormSubmissionStatus,
    schemaId: string,
  ) => {
    updateSubStatus.mutate(
      { id: sub.id, status, formSchemaId: schemaId },
      {
        onSuccess: () =>
          toast.success(
            status === 'read'
              ? t('formulare.toast.alsGelesenMarkiert')
              : t('formulare.toast.eingangArchiviert'),
          ),
        onError: (err) =>
          toast.error(err instanceof Error ? err.message : t('common.error')),
      },
    )
  }

  const handleAddField = (type: FormFieldType) => {
    if (type === 'consent') {
      // DSGVO consent is a required, purpose-bound checkbox by definition.
      // Defaults come from the tenant settings (overridable per field).
      addField({
        type,
        label: t('formulare.consent.defaultLabel'),
        required: true,
        consentText: prefsConsentText || DEFAULT_CONSENT_TEXT,
        privacyUrl: prefsPrivacyUrl || DEFAULT_PRIVACY_URL,
      })
      setShowAddFieldMenu(false)
      return
    }
    addField({
      type,
      label: FIELD_TYPE_LABELS[type],
      required: false,
      placeholder: '',
      options:
        type === 'select' || type === 'radio'
          ? ['Option 1', 'Option 2']
          : undefined,
      ratingScale: type === 'rating' ? 5 : undefined,
    })
    setShowAddFieldMenu(false)
  }

  const handleRemoveField = (fieldId: string) => {
    removeField(fieldId)
  }

  const handleDuplicateField = (fieldId: string) => {
    duplicateField(fieldId)
    toast.success(t('formulare.toast.feldDupliziert'))
  }

  const openFieldConfig = (field: FormField) => {
    setConfigLabel(field.label)
    setConfigRequired(field.required)
    setConfigPlaceholder(field.placeholder ?? '')
    setConfigOptions(field.options?.join(', ') ?? '')
    setConfigConsentText(field.consentText ?? '')
    setConfigPrivacyUrl(field.privacyUrl ?? '')
    setConfigMinLength(field.validation?.minLength?.toString() ?? '')
    setConfigMaxLength(field.validation?.maxLength?.toString() ?? '')
    setConfigMin(field.validation?.min?.toString() ?? '')
    setConfigMax(field.validation?.max?.toString() ?? '')
    setConfigPatternType(field.validation?.patternType ?? 'free')
    setConfigRatingScale(field.ratingScale ?? 5)
    setConfigRole(field.role ?? '')
    setConfigConditionalEnabled(!!field.conditionalLogic)
    setConfigConditionalFieldId(field.conditionalLogic?.fieldId ?? '')
    setConfigConditionalOperator(field.conditionalLogic?.operator ?? 'equals')
    setConfigConditionalValue(field.conditionalLogic?.value ?? '')
    setShowFieldConfigDialog({ field })
  }

  // FT-2a — assemble the optional validation object from the dialog inputs.
  const buildValidation = (type: FormFieldType): FormField['validation'] => {
    const num = (s: string): number | undefined => {
      const n = Number(s)
      return s.trim() !== '' && Number.isFinite(n) ? n : undefined
    }
    const rules: NonNullable<FormField['validation']> = {}
    if (type === 'text' || type === 'textarea') {
      const minL = num(configMinLength)
      const maxL = num(configMaxLength)
      if (minL != null) rules.minLength = minL
      if (maxL != null) rules.maxLength = maxL
    }
    if (type === 'number') {
      const mn = num(configMin)
      const mx = num(configMax)
      if (mn != null) rules.min = mn
      if (mx != null) rules.max = mx
    }
    if (type === 'text' && configPatternType !== 'free') {
      rules.patternType = configPatternType
    }
    return Object.keys(rules).length > 0 ? rules : undefined
  }

  const saveFieldConfig = () => {
    if (!showFieldConfigDialog) return
    const { field } = showFieldConfigDialog
    updateField(field.id, {
      label: configLabel,
      required: field.type === 'consent' ? true : configRequired,
      placeholder: configPlaceholder || undefined,
      validation: buildValidation(field.type),
      consentText:
        field.type === 'consent' ? configConsentText || undefined : field.consentText,
      privacyUrl:
        field.type === 'consent' ? configPrivacyUrl || undefined : field.privacyUrl,
      options:
        field.type === 'select' || field.type === 'radio'
          ? configOptions
              .split(',')
              .map((o) => o.trim())
              .filter(Boolean)
          : field.options,
      ratingScale: field.type === 'rating' ? configRatingScale : field.ratingScale,
      role: configRole || undefined,
      conditionalLogic:
        configConditionalEnabled && configConditionalFieldId
          ? {
              fieldId: configConditionalFieldId,
              operator: configConditionalOperator,
              value: configConditionalValue,
            }
          : undefined,
    })
    setShowFieldConfigDialog(null)
    toast.success(t('formulare.toast.feldAktualisiert'))
  }

  // New form
  const resetNewFormDialog = () => {
    setNewFormName('')
    setNewFormDescription('')
    setNewFormTemplateId('')
  }

  const handleCreateForm = () => {
    if (!newFormName.trim()) {
      toast.error(t('formulare.toast.nameRequired'))
      return
    }
    if (newFormTemplateId) {
      duplicateSchema.mutate(
        { id: newFormTemplateId, title: newFormName },
        {
          onSuccess: () => {
            toast.success(
              t('formulare.toast.formularErstellt', { name: newFormName }),
            )
            resetNewFormDialog()
            setShowNewFormDialog(false)
          },
          onError: (err) =>
            toast.error(
              err instanceof Error ? err.message : t('common.error'),
            ),
        },
      )
    } else {
      createSchema.mutate(
        {
          title: newFormName,
          description: newFormDescription,
          status: 'draft',
          fields: [],
          isTemplate: false,
          // FT-5 — seed the configurable tenant default thank-you message.
          thankYouMessage: prefsThankYouMessage.trim() || undefined,
          // FO-4 — seed the recipient list from the tenant notification default.
          notifications: {
            recipients:
              prefsNotifyOnSubmission && prefsNotifyEmail.trim()
                ? [prefsNotifyEmail.trim()]
                : [],
            confirmToSubmitter: false,
          },
        },
        {
          onSuccess: () => {
            toast.success(
              t('formulare.toast.formularErstellt', { name: newFormName }),
            )
            resetNewFormDialog()
            setShowNewFormDialog(false)
          },
          onError: (err) =>
            toast.error(
              err instanceof Error ? err.message : t('common.error'),
            ),
        },
      )
    }
  }

  // Submissions
  const toggleGroup = (schemaId: string) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(schemaId)) next.delete(schemaId)
      else next.add(schemaId)
      return next
    })
  }

  const getFieldForSubmission = (
    schemaId: string,
    fieldId: string,
  ): FormField | undefined => {
    const schema = schemas.find((s) => s.id === schemaId)
    return (schema?.fields as FormField[] | undefined)?.find(
      (f) => f.id === fieldId,
    )
  }

  // Template use
  const handleUseTemplate = (template: FormSchema) => {
    duplicateSchema.mutate(
      { id: template.id },
      {
        onSuccess: () =>
          toast.success(
            t('formulare.toast.vorlagenFormularErstellt', {
              name: template.title,
            }),
          ),
        onError: (err) =>
          toast.error(err instanceof Error ? err.message : t('common.error')),
      },
    )
  }

  // FT-5 — turn an existing form into a reusable template (moves it to the
  // Vorlagen tab) and the reverse delete flow for templates.
  const saveAsTemplate = (schema: FormSchema) => {
    updateSchema.mutate(
      { id: schema.id, isTemplate: true },
      {
        onSuccess: () => {
          toast.success(t('formulare.vorlagen.savedAsTemplate', { name: schema.title }))
          setSelectedForm((cur) => (cur && cur.id === schema.id ? null : cur))
        },
        onError: (err) =>
          toast.error(err instanceof Error ? err.message : t('common.error')),
      },
    )
  }

  const handleDeleteTemplate = (template: FormSchema) => {
    deleteSchema.mutate(template.id, {
      onSuccess: () => {
        setTemplateToDelete(null)
        setSelectedTemplate(null)
        toast.success(t('formulare.vorlagen.deleted', { name: template.title }))
      },
      onError: (err) =>
        toast.error(err instanceof Error ? err.message : t('common.error')),
    })
  }

  // ---------------------------------------------------------------------------
  // 10.2 — Page break helpers (operate on draft.fields)
  // ---------------------------------------------------------------------------

  const handleInsertPageBreak = () => {
    if (!draft) return
    const maxPage = draft.fields.reduce(
      (m, f) => Math.max(m, f.page ?? 0),
      0,
    )
    const newPage = maxPage + 1
    updateDraftMeta({ pageCount: newPage + 1 })
    addField({
      type: 'text',
      label: '__page_break__',
      required: false,
      placeholder: '',
      page: newPage,
    })
    setShowAddFieldMenu(false)
  }

  const computeFieldPages = (fields: FormField[]): Map<string, number> => {
    const map = new Map<string, number>()
    let currentPage = 0
    for (const f of fields) {
      if (f.label === '__page_break__') {
        currentPage++
        map.set(f.id, -1)
      } else {
        map.set(f.id, currentPage)
      }
    }
    return map
  }

  const fieldPageMap = useMemo(
    () => (draft ? computeFieldPages(draft.fields) : new Map<string, number>()),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [draft?.fields],
  )

  const totalPages = useMemo(() => {
    if (!draft) return 1
    let pages = 0
    for (const f of draft.fields) {
      if (f.label === '__page_break__') pages++
    }
    return pages + 1
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft?.fields])

  // ---------------------------------------------------------------------------
  // 10.1 — Conditional logic evaluation helper
  // ---------------------------------------------------------------------------

  // ---------------------------------------------------------------------------
  // Render helpers
  // ---------------------------------------------------------------------------

  const renderFieldIcon = (type: FormFieldType, className = 'h-4 w-4') => {
    const Icon = FIELD_TYPE_ICONS[type]
    return <Icon className={className} />
  }

  const renderAnswer = (
    fieldId: string,
    value: unknown,
    schemaId: string,
  ) => {
    const field = getFieldForSubmission(schemaId, fieldId)
    if (!field) {
      return <span className="text-sm text-foreground">{String(value)}</span>
    }

    switch (field.type) {
      case 'checkbox':
        return (
          <span
            className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${
              value
                ? 'bg-success-light text-success'
                : 'bg-secondary text-muted-foreground'
            }`}
          >
            {value
              ? t('formulare.submission.ja')
              : t('formulare.submission.nein')}
          </span>
        )
      case 'select':
      case 'radio':
        return (
          <span className="inline-block rounded-full bg-primary-light px-2 py-0.5 text-xs font-medium text-primary">
            {String(value)}
          </span>
        )
      case 'date':
        return (
          <span className="text-sm text-foreground">
            {typeof value === 'string' && value ? formatDate(value) : '--'}
          </span>
        )
      case 'file':
        return value ? (
          <span className="inline-flex items-center gap-1.5 text-sm text-primary">
            <Paperclip className="h-3.5 w-3.5" />
            {t('formulare.submission.dateiHochgeladen')}
          </span>
        ) : (
          <span className="text-sm text-muted-foreground">
            {t('formulare.submission.keineDatei')}
          </span>
        )
      case 'rating': {
        const n = Number(value) || 0
        const scale = field.ratingScale ?? 5
        return (
          <span className="inline-flex items-center gap-2">
            <RatingInput scale={scale} value={n} disabled />
            <span className="text-xs text-muted-foreground">
              {n}/{scale}
            </span>
          </span>
        )
      }
      default:
        return (
          <span className="text-sm text-foreground">
            {String(value) || (
              <span className="text-muted-foreground">--</span>
            )}
          </span>
        )
    }
  }

  const renderFieldPreview = (field: FormField) => {
    switch (field.type) {
      case 'text':
        return (
          <input
            type="text"
            disabled
            placeholder={field.placeholder || field.label}
            className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm text-muted-foreground placeholder:text-input-placeholder"
          />
        )
      case 'email':
        return (
          <input
            type="email"
            disabled
            placeholder={field.placeholder || 'name@beispiel.de'}
            className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm text-muted-foreground placeholder:text-input-placeholder"
          />
        )
      case 'textarea':
        return (
          <textarea
            disabled
            rows={3}
            placeholder={field.placeholder || field.label}
            className="w-full resize-none rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm text-muted-foreground placeholder:text-input-placeholder"
          />
        )
      case 'select':
        return (
          <select
            disabled
            className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm text-muted-foreground"
          >
            <option>{t('formulare.editor.selectPlaceholder')}</option>
            {field.options?.map((opt) => (
              <option key={opt}>{opt}</option>
            ))}
          </select>
        )
      case 'radio':
        return (
          <div className="space-y-1.5">
            {field.options?.map((opt) => (
              <label
                key={opt}
                className="flex items-center gap-2 text-sm text-muted-foreground"
              >
                <div className="h-4 w-4 rounded-full border-2 border-border" />
                {opt}
              </label>
            ))}
          </div>
        )
      case 'checkbox':
        return (
          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            <div className="h-4 w-4 rounded border-2 border-border" />
            {field.label}
          </label>
        )
      case 'date':
        return (
          <input
            type="date"
            disabled
            className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm text-muted-foreground"
          />
        )
      case 'number':
        return (
          <input
            type="number"
            disabled
            placeholder="0"
            className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm text-muted-foreground placeholder:text-input-placeholder"
          />
        )
      case 'file':
        return (
          <div className="flex items-center gap-2 rounded-lg border-2 border-dashed border-border bg-secondary/30 px-4 py-6 text-sm text-muted-foreground">
            <Paperclip className="h-4 w-4" />
            {t('formulare.editor.dateiZiehen')}
          </div>
        )
      case 'rating':
        return <RatingInput scale={field.ratingScale ?? 5} value={0} disabled />
      case 'consent':
        return (
          <label className="flex items-start gap-2.5 rounded-lg border border-border bg-secondary/30 p-3">
            <div className="mt-0.5 h-4 w-4 shrink-0 rounded border-2 border-border" />
            <span className="text-xs leading-relaxed text-muted-foreground">
              {field.consentText || DEFAULT_CONSENT_TEXT}
              {field.privacyUrl && (
                <a
                  href={field.privacyUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  onClick={(e) => e.preventDefault()}
                  className="ml-1 inline-flex items-center gap-0.5 text-primary underline"
                >
                  {t('formulare.consent.privacyLink')}
                  <ExternalLink className="h-3 w-3" />
                </a>
              )}
            </span>
          </label>
        )
      default:
        return null
    }
  }

  // ---------------------------------------------------------------------------
  // JSX — Loading / Error states
  // ---------------------------------------------------------------------------

  if (schemasLoading) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <div className="animate-pulse space-y-4">
          <div className="h-12 rounded-lg bg-secondary" />
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <div key={i} className="h-36 rounded-xl bg-secondary" />
            ))}
          </div>
        </div>
      </div>
    )
  }

  if (schemasError) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <EmptyState
          icon={FileText}
          title={t('common.errorTitle')}
          description={t('common.errorDescription')}
        />
      </div>
    )
  }

  // moduleEmpty: no read right on any tab → honest empty state
  if (capReady && visibleTabs.length === 0) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <PageHeader
          title={t('formulare.page.title')}
          description={t('rbac.gate.moduleEmpty')}
          icon={FileInput}
          moduleId="formulare"
          className="mb-6"
          actions={<RestrictedModeBadge module="formulare" />}
        />
        <EmptyState
          icon={FileText}
          title={t('rbac.gate.moduleEmpty')}
          description={t('rbac.gate.noPermission')}
        />
      </div>
    )
  }

  // ---------------------------------------------------------------------------
  // JSX — Editor View
  // ---------------------------------------------------------------------------

  if (draft) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        {/* Editor Header. Embedded in the module editor the surrounding shell
            already carries a sticky "Zurück zum Modul" — a second back button
            right beneath it would just be noise. */}
        <div className="flex items-center gap-3 mb-6">
          {!embedded && (
            <button
              onClick={closeEditor}
              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              <ArrowLeft className="h-4 w-4" />
              {t('formulare.editor.zurueck')}
            </button>
          )}
          <div className="flex-1" />

          {/* 10.4 — Öffentlich toggle */}
          <div className="flex items-center gap-2 mr-2">
            <Globe className="h-4 w-4 text-muted-foreground" />
            <span className="text-xs text-muted-foreground">
              {t('formulare.editor.oeffentlich')}
            </span>
            <button
              onClick={() =>
                updateDraftMeta({ isPublic: !draft.isPublic })
              }
              className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                draft.isPublic ? 'bg-primary' : 'bg-secondary'
              }`}
            >
              <span
                className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                  draft.isPublic ? 'translate-x-4.5' : 'translate-x-0.5'
                }`}
              />
            </button>
          </div>

          {/* 10.4 — Public preview button */}
          <button
            onClick={() => setShowPublicPreview(true)}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            title={t('formulare.editor.vorschau')}
          >
            <Eye className="h-4 w-4" />
            {t('formulare.editor.vorschau')}
          </button>

          <button
            onClick={closeEditor}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={saveEditor}
            disabled={updateSchema.isPending}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-60"
          >
            {updateSchema.isPending ? t('common.saving') : t('common.save')}
          </button>
        </div>

        {/* Name + Description */}
        <div className="mb-6 space-y-3">
          <input
            type="text"
            value={draft.title}
            onChange={(e) => updateDraftMeta({ title: e.target.value })}
            placeholder={t('formulare.preview.formularname')}
            className="w-full text-xl font-semibold text-foreground bg-transparent border-b border-border pb-2 focus:outline-none focus:border-primary transition-colors placeholder:text-input-placeholder"
          />
          <input
            type="text"
            value={draft.description}
            onChange={(e) => updateDraftMeta({ description: e.target.value })}
            placeholder={t('formulare.newForm.beschreibungPlaceholder')}
            className="w-full text-sm text-muted-foreground bg-transparent border-b border-border-muted pb-2 focus:outline-none focus:border-primary transition-colors placeholder:text-input-placeholder"
          />
        </div>

        {/* Editor two-column layout */}
        <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
          {/* Left: Field list */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-medium text-foreground">
                {t('formulare.editor.felder', {
                  count: draft.fields.filter(
                    (f) => f.label !== '__page_break__',
                  ).length,
                })}
              </h3>
              <div className="relative">
                <button
                  onClick={() => setShowAddFieldMenu(!showAddFieldMenu)}
                  className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
                >
                  <Plus className="h-3.5 w-3.5" />
                  {t('formulare.editor.feldHinzufuegen')}
                </button>
                {showAddFieldMenu && (
                  <div className="absolute right-0 top-full mt-1 z-20 w-56 rounded-lg border border-border bg-card shadow-xl py-1">
                    {FIELD_TYPE_OPTION_KEYS.map((opt) => {
                      const Icon = FIELD_TYPE_ICONS[opt.value]
                      return (
                        <button
                          key={opt.value}
                          onClick={() => handleAddField(opt.value)}
                          className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
                        >
                          <Icon className="h-4 w-4 text-muted-foreground" />
                          {t(opt.labelKey)}
                        </button>
                      )
                    })}
                    {/* 10.2 — Page break option */}
                    <div className="border-t border-border my-1" />
                    <button
                      onClick={handleInsertPageBreak}
                      className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
                    >
                      <Split className="h-4 w-4 text-muted-foreground" />
                      {t('formulare.editor.seitenumbruch')}
                    </button>
                  </div>
                )}
              </div>
            </div>

            {draft.fields.length === 0 ? (
              <div className="rounded-lg border-2 border-dashed border-border p-8 text-center">
                <ClipboardList className="mx-auto h-8 w-8 text-muted-foreground/40" />
                <p className="mt-2 text-sm text-muted-foreground">
                  {t('formulare.editor.noFelder')}
                </p>
              </div>
            ) : (
              <DndContext
                sensors={fieldSensors}
                collisionDetection={closestCenter}
                onDragEnd={handleFieldDragEnd}
              >
                <SortableContext
                  items={draft.fields.map((f) => f.id)}
                  strategy={verticalListSortingStrategy}
                >
                  <div className="space-y-2">
                    {draft.fields.map((field) => {
                      if (field.label === '__page_break__') {
                        const pageNum =
                          fieldPageMap.get(field.id) === -1
                            ? Array.from(fieldPageMap.entries())
                                .filter(([, v]) => v === -1)
                                .findIndex(([k]) => k === field.id) + 1
                            : 1
                        return (
                          <SortableFieldItem
                            key={field.id}
                            field={field}
                            pageBreakLabel={t('formulare.editor.seitenumbruchLabel', {
                              page: pageNum + 1,
                            })}
                            onPageTitleChange={(value) =>
                              updateField(field.id, { pageTitle: value })
                            }
                            t={t}
                            onRemove={() => handleRemoveField(field.id)}
                          />
                        )
                      }

                      return (
                        <SortableFieldItem
                          key={field.id}
                          field={field}
                          icon={FIELD_TYPE_ICONS[field.type]}
                          typeLabel={FIELD_TYPE_LABELS[field.type]}
                          t={t}
                          onEdit={() => openFieldConfig(field)}
                          onDuplicate={() => handleDuplicateField(field.id)}
                          onRemove={() => handleRemoveField(field.id)}
                        />
                      )
                    })}
                  </div>
                </SortableContext>
              </DndContext>
            )}
          </div>

          {/* Right: Live preview */}
          <div>
            <h3 className="text-sm font-medium text-foreground mb-3">
              {t('formulare.editor.vorschauTitle')}
            </h3>
            <div className="rounded-xl border border-border bg-card p-5 space-y-4">
              <div className="mb-4">
                <h2 className="text-lg font-semibold text-foreground">
                  {draft.title || t('formulare.preview.formularname')}
                </h2>
                {draft.description && (
                  <p className="text-sm text-muted-foreground mt-1">
                    {draft.description}
                  </p>
                )}
              </div>

              {/* 10.2 — Page indicator */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between rounded-lg bg-secondary/50 px-3 py-2">
                  <button
                    onClick={() =>
                      setPreviewPage(Math.max(0, previewPage - 1))
                    }
                    disabled={previewPage === 0}
                    className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground disabled:opacity-40 transition-colors"
                  >
                    <ChevronLeft className="h-3.5 w-3.5" />
                    {t('formulare.editor.zurueck')}
                  </button>
                  <span className="text-xs font-medium text-foreground">
                    {t('formulare.editor.seite', {
                      current: previewPage + 1,
                      total: totalPages,
                    })}
                  </span>
                  <button
                    onClick={() =>
                      setPreviewPage(Math.min(totalPages - 1, previewPage + 1))
                    }
                    disabled={previewPage >= totalPages - 1}
                    className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground disabled:opacity-40 transition-colors"
                  >
                    {t('formulare.editor.weiter')}
                    <ChevronRight className="h-3.5 w-3.5" />
                  </button>
                </div>
              )}

              {/* FT-2b — title of the current page (from the page-break entry) */}
              {totalPages > 1 &&
                previewPage > 0 &&
                (() => {
                  const breaks = draft.fields.filter(
                    (f) => f.label === '__page_break__',
                  )
                  const title = breaks[previewPage - 1]?.pageTitle?.trim()
                  return title ? (
                    <h3 className="border-b border-border pb-1.5 text-base font-semibold text-foreground">
                      {title}
                    </h3>
                  ) : null
                })()}

              {draft.fields.length === 0 ? (
                <p className="text-sm text-muted-foreground italic">
                  {t('formulare.editor.vorschauHinweis')}
                </p>
              ) : (
                draft.fields
                  .filter((field) => {
                    if (field.label === '__page_break__') return false
                    if (totalPages > 1) {
                      const page = fieldPageMap.get(field.id) ?? 0
                      return page === previewPage
                    }
                    return true
                  })
                  .map((field) => (
                    <div
                      key={field.id}
                      className={`space-y-1.5 ${field.conditionalLogic ? 'opacity-60' : ''}`}
                    >
                      <label className="text-sm font-medium text-foreground flex items-center gap-2">
                        {field.label}
                        {field.required && (
                          <span className="text-destructive ml-0.5">*</span>
                        )}
                        {/* 10.1 — Conditional badge in preview */}
                        {field.conditionalLogic && (
                          <span className="rounded bg-warning-light px-1.5 py-0.5 text-[9px] font-medium text-warning">
                            {t('formulare.editor.bedingt')}
                          </span>
                        )}
                      </label>
                      {renderFieldPreview(field)}
                    </div>
                  ))
              )}
              {draft.fields.filter((f) => f.label !== '__page_break__').length >
                0 && (
                <button
                  disabled
                  className="mt-2 rounded-lg bg-primary/50 px-4 py-2 text-sm text-primary-foreground cursor-not-allowed"
                >
                  {totalPages > 1 && previewPage < totalPages - 1
                    ? t('formulare.editor.weiter')
                    : t('formulare.editor.absenden')}
                </button>
              )}
            </div>
          </div>
        </div>

        {/* P2 — intake binding: which module record a submission of this form
            creates. Field roles map answers onto that record's properties. */}
        <div className="mt-6 rounded-xl border border-border bg-card p-5">
          <div className="mb-1 flex items-center gap-2">
            <Zap className="h-4 w-4 text-muted-foreground" />
            <h3 className="text-sm font-medium text-foreground">
              {t('formulare.intake.title')}
            </h3>
          </div>
          <p className="mb-4 text-xs text-muted-foreground">
            {t('formulare.intake.subtitle')}
          </p>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('formulare.intake.targetLabel')}
            </label>
            <select
              value={draft.intakeTargetId ?? ''}
              onChange={(e) =>
                updateDraftMeta({ intakeTargetId: e.target.value || undefined })
              }
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            >
              <option value="">{t('formulare.intake.targetNone')}</option>
              {listIntakeTargets().map((target) => (
                <option key={target.id} value={target.id}>
                  {t(target.labelKey)}
                </option>
              ))}
            </select>
          </div>
          {draft.intakeTargetId && (
            <div className="mt-4 rounded-lg bg-info-light px-3 py-3">
              <div className="flex items-center gap-1.5 text-xs font-medium text-info">
                <Info className="h-3.5 w-3.5" />
                {t('formulare.intake.coverageTitle')}
              </div>
              <ul className="mt-2 space-y-1">
                {getIntakeRoles(draft.intakeTargetId).map((role) => {
                  const field = draft.fields.find((f) => f.role === role.role)
                  return (
                    <li key={role.role} className="flex items-center gap-1.5 text-xs">
                      {field ? (
                        <Check className="h-3.5 w-3.5 shrink-0 text-success" />
                      ) : (
                        <Circle className="h-3 w-3 shrink-0 text-muted-foreground" />
                      )}
                      <span
                        className={field ? 'text-foreground' : 'text-muted-foreground'}
                      >
                        {t(role.labelKey)}
                        {field ? ` — ${field.label}` : ''}
                      </span>
                    </li>
                  )
                })}
              </ul>
              <p className="mt-2 text-[10px] text-muted-foreground">
                {t('formulare.intake.coverageHint')}
              </p>
            </div>
          )}
        </div>

        {/* FT-2b — "After submit": thank-you message + optional redirect */}
        <div className="mt-6 rounded-xl border border-border bg-card p-5">
          <div className="mb-1 flex items-center gap-2">
            <Check className="h-4 w-4 text-muted-foreground" />
            <h3 className="text-sm font-medium text-foreground">
              {t('formulare.thankYou.title')}
            </h3>
          </div>
          <p className="mb-4 text-xs text-muted-foreground">
            {t('formulare.thankYou.subtitle')}
          </p>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('formulare.thankYou.messageLabel')}
              </label>
              <textarea
                value={draft.thankYouMessage ?? ''}
                onChange={(e) => updateDraftMeta({ thankYouMessage: e.target.value })}
                rows={3}
                placeholder={t('formulare.thankYou.messagePlaceholder')}
                className="w-full resize-none rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('formulare.thankYou.redirectLabel')}
              </label>
              <input
                type="url"
                value={draft.redirectUrl ?? ''}
                onChange={(e) => updateDraftMeta({ redirectUrl: e.target.value })}
                placeholder="https://…"
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
              <p className="text-[10px] text-muted-foreground">
                {t('formulare.thankYou.redirectHint')}
              </p>
            </div>
          </div>
        </div>

        {/* FO-4 — submission notifications: recipients + submitter confirmation */}
        <div className="mt-6 rounded-xl border border-border bg-card p-5">
          <div className="mb-1 flex items-center gap-2">
            <Mail className="h-4 w-4 text-muted-foreground" />
            <h3 className="text-sm font-medium text-foreground">
              {t('formulare.notify.title')}
            </h3>
          </div>
          <p className="mb-4 text-xs text-muted-foreground">
            {t('formulare.notify.subtitle')}
          </p>

          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('formulare.notify.recipientsLabel')}
            </label>
            <div className="flex gap-2">
              <input
                type="email"
                value={newRecipient}
                onChange={(e) => setNewRecipient(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    addRecipient()
                  }
                }}
                placeholder={t('formulare.notify.recipientsPlaceholder')}
                className="flex-1 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
              <button
                type="button"
                onClick={addRecipient}
                className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-foreground transition-colors hover:bg-secondary"
              >
                <Plus className="h-4 w-4" />
                {t('formulare.notify.add')}
              </button>
            </div>
            {draft.notifications.recipients.length > 0 ? (
              <div className="flex flex-wrap gap-1.5 pt-1">
                {draft.notifications.recipients.map((email) => (
                  <span
                    key={email}
                    className="inline-flex items-center gap-1.5 rounded-full bg-primary-light px-2.5 py-1 text-xs text-primary"
                  >
                    {email}
                    <button
                      type="button"
                      onClick={() => removeRecipient(email)}
                      className="rounded-full p-0.5 transition-colors hover:bg-primary/15"
                      aria-label={t('formulare.notify.remove', { email })}
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </span>
                ))}
              </div>
            ) : (
              <p className="pt-1 text-xs text-muted-foreground">
                {t('formulare.notify.noRecipients')}
              </p>
            )}
          </div>

          <div className="mt-4 flex items-center justify-between gap-4 border-t border-border-muted pt-4">
            <div className="min-w-0">
              <label className="text-sm font-medium text-foreground">
                {t('formulare.notify.confirmLabel')}
              </label>
              <p className="text-xs text-muted-foreground">
                {t('formulare.notify.confirmHint')}
              </p>
            </div>
            <button
              type="button"
              onClick={toggleConfirmToSubmitter}
              className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors ${
                draft.notifications.confirmToSubmitter ? 'bg-primary' : 'bg-secondary'
              }`}
              aria-pressed={draft.notifications.confirmToSubmitter}
            >
              <span
                className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                  draft.notifications.confirmToSubmitter
                    ? 'translate-x-4.5'
                    : 'translate-x-0.5'
                }`}
              />
            </button>
          </div>
        </div>

        {/* ====================== 10.4 — PUBLIC PREVIEW MODAL ====================== */}
        <Dialog
          open={showPublicPreview && !!draft}
          onOpenChange={(o) => {
            if (!o) setShowPublicPreview(false)
          }}
        >
          <DialogContent className="gap-0 p-0 max-w-2xl max-h-[90vh] overflow-y-auto bg-white">
            <DialogTitle className="sr-only">
              {t('formulare.editor.vorschau')}
            </DialogTitle>
            <DialogDescription className="sr-only">
              {t('formulare.preview.banner')}
            </DialogDescription>

            {/* Info banner */}
            <div className="flex items-center gap-2 bg-blue-50 border-b border-blue-100 px-6 py-3">
              <Info className="h-4 w-4 text-blue-500 shrink-0" />
              <p className="text-xs text-blue-700">
                {t('formulare.preview.banner')}
              </p>
            </div>

            {/* Form content — interactive fill preview with FT-2a validation */}
            <PreviewFillForm
              fields={draft.fields as FormField[]}
              title={draft.title || t('formulare.preview.formularname')}
              description={draft.description}
              thankYouMessage={draft.thankYouMessage}
              redirectUrl={draft.redirectUrl}
              notifications={draft.notifications}
              intakeTargetId={draft.intakeTargetId}
              intakeTargetLabel={
                draft.intakeTargetId
                  ? t(getIntakeTarget(draft.intakeTargetId)?.labelKey ?? '')
                  : undefined
              }
              onIntakeSubmit={
                draft.intakeTargetId
                  ? async (answers) => {
                      const targetId = draft.intakeTargetId as string
                      const res = await dispatchIntake({
                        targetId,
                        fields: draft.fields,
                        answers,
                        context: { channel: 'external' },
                      })
                      toast.success(
                        t('formulare.preview.intakeToast', {
                          target: t(getIntakeTarget(targetId)?.labelKey ?? ''),
                        }),
                      )
                      return { id: res.id }
                    }
                  : undefined
              }
              t={t}
            />
          </DialogContent>
        </Dialog>

        {/* ====================== FIELD CONFIG DIALOG ====================== */}
        <Dialog
          open={!!showFieldConfigDialog}
          onOpenChange={(o) => {
            if (!o) setShowFieldConfigDialog(null)
          }}
        >
          <DialogContent className="gap-0 p-0 max-w-md">
            <DialogHeader className="border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                {showFieldConfigDialog &&
                  renderFieldIcon(
                    showFieldConfigDialog.field.type,
                    'h-5 w-5 text-primary',
                  )}
                <DialogTitle className="text-base font-semibold text-foreground">
                  {t('formulare.fieldConfig.title')}
                </DialogTitle>
              </div>
              <DialogDescription className="sr-only">
                {t('formulare.fieldConfig.title')}
              </DialogDescription>
            </DialogHeader>

            <div className="p-5 space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  {t('formulare.fieldConfig.bezeichnung')}{' '}
                  <span className="text-destructive">*</span>
                </label>
                <input
                  type="text"
                  value={configLabel}
                  onChange={(e) => setConfigLabel(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>

              {/* P2 — intake field role: only when the form feeds a target. Marks
                  this field as a first-class record property; unmarked = extra. */}
              {draft?.intakeTargetId && showFieldConfigDialog && (
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">
                    {t('formulare.fieldConfig.roleLabel')}
                  </label>
                  <select
                    value={configRole}
                    onChange={(e) => setConfigRole(e.target.value)}
                    className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  >
                    <option value="">{t('formulare.fieldConfig.roleExtra')}</option>
                    {getIntakeRoles(draft.intakeTargetId)
                      .filter(
                        (r) =>
                          !r.fieldTypes ||
                          r.fieldTypes.includes(showFieldConfigDialog.field.type),
                      )
                      .map((r) => (
                        <option key={r.role} value={r.role}>
                          {t(r.labelKey)}
                        </option>
                      ))}
                  </select>
                  <p className="text-xs text-muted-foreground">
                    {t('formulare.fieldConfig.roleHint')}
                  </p>
                </div>
              )}

              {showFieldConfigDialog?.field.type !== 'consent' && (
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium text-foreground">
                    {t('formulare.fieldConfig.pflichtfeld')}
                  </label>
                  <button
                    onClick={() => setConfigRequired(!configRequired)}
                    className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                      configRequired ? 'bg-primary' : 'bg-secondary'
                    }`}
                  >
                    <span
                      className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                        configRequired ? 'translate-x-4.5' : 'translate-x-0.5'
                      }`}
                    />
                  </button>
                </div>
              )}

              {/* DSGVO consent: purpose-binding text + privacy policy link */}
              {showFieldConfigDialog?.field.type === 'consent' && (
                <>
                  <div className="flex items-start gap-2 rounded-lg bg-info-light px-3 py-2">
                    <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-info" />
                    <p className="text-xs text-info">
                      {t('formulare.consent.configHint')}
                    </p>
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-sm font-medium text-foreground">
                      {t('formulare.consent.text')}
                    </label>
                    <textarea
                      value={configConsentText}
                      onChange={(e) => setConfigConsentText(e.target.value)}
                      rows={4}
                      placeholder={t('formulare.consent.textPlaceholder')}
                      className="w-full resize-none rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-sm font-medium text-foreground">
                      {t('formulare.consent.privacyUrl')}
                    </label>
                    <input
                      type="url"
                      value={configPrivacyUrl}
                      onChange={(e) => setConfigPrivacyUrl(e.target.value)}
                      placeholder="https://…/datenschutz"
                      className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    />
                  </div>
                </>
              )}

              {(showFieldConfigDialog?.field.type === 'text' ||
                showFieldConfigDialog?.field.type === 'textarea' ||
                showFieldConfigDialog?.field.type === 'email' ||
                showFieldConfigDialog?.field.type === 'number') && (
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">
                    {t('formulare.fieldConfig.platzhalter')}
                  </label>
                  <input
                    type="text"
                    value={configPlaceholder}
                    onChange={(e) => setConfigPlaceholder(e.target.value)}
                    placeholder={t(
                      'formulare.fieldConfig.platzhalterPlaceholder',
                    )}
                    className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>
              )}

              {(showFieldConfigDialog?.field.type === 'select' ||
                showFieldConfigDialog?.field.type === 'radio') && (
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">
                    {t('formulare.fieldConfig.optionen')}{' '}
                    <span className="text-xs text-muted-foreground">
                      {t('formulare.fieldConfig.optionenHint')}
                    </span>
                  </label>
                  <input
                    type="text"
                    value={configOptions}
                    onChange={(e) => setConfigOptions(e.target.value)}
                    placeholder={t('formulare.fieldConfig.optionenPlaceholder')}
                    className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>
              )}

              {/* FT-2b — Rating scale (stars vs NPS) */}
              {showFieldConfigDialog?.field.type === 'rating' && (
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">
                    {t('formulare.rating.scaleLabel')}
                  </label>
                  <div className="flex gap-2">
                    {([5, 10] as const).map((s) => {
                      const on = configRatingScale === s
                      return (
                        <button
                          key={s}
                          type="button"
                          onClick={() => setConfigRatingScale(s)}
                          className={`flex-1 rounded-lg border px-3 py-2 text-sm transition-colors ${
                            on
                              ? 'border-primary bg-primary-light text-primary'
                              : 'border-border text-muted-foreground hover:bg-secondary'
                          }`}
                          aria-pressed={on}
                        >
                          {s === 5
                            ? t('formulare.rating.scaleStars')
                            : t('formulare.rating.scaleNps')}
                        </button>
                      )
                    })}
                  </div>
                  <p className="text-[10px] text-muted-foreground">
                    {t('formulare.rating.scaleHint')}
                  </p>
                </div>
              )}

              {/* FT-2a — Validation rules (length / value / pattern) */}
              {(showFieldConfigDialog?.field.type === 'text' ||
                showFieldConfigDialog?.field.type === 'textarea' ||
                showFieldConfigDialog?.field.type === 'number') && (
                <div className="border-t border-border-muted pt-4 space-y-3">
                  <div className="flex items-center gap-2">
                    <Ruler className="h-4 w-4 text-muted-foreground" />
                    <label className="text-sm font-medium text-foreground">
                      {t('formulare.validation.title')}
                    </label>
                  </div>

                  {(showFieldConfigDialog?.field.type === 'text' ||
                    showFieldConfigDialog?.field.type === 'textarea') && (
                    <div className="grid grid-cols-2 gap-3">
                      <div className="space-y-1">
                        <label className="text-[10px] uppercase tracking-wider text-muted-foreground">
                          {t('formulare.validation.minLength')}
                        </label>
                        <input
                          type="number"
                          min={0}
                          value={configMinLength}
                          onChange={(e) => setConfigMinLength(e.target.value)}
                          placeholder={t('formulare.validation.optional')}
                          className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                        />
                      </div>
                      <div className="space-y-1">
                        <label className="text-[10px] uppercase tracking-wider text-muted-foreground">
                          {t('formulare.validation.maxLength')}
                        </label>
                        <input
                          type="number"
                          min={0}
                          value={configMaxLength}
                          onChange={(e) => setConfigMaxLength(e.target.value)}
                          placeholder={t('formulare.validation.optional')}
                          className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                        />
                      </div>
                    </div>
                  )}

                  {showFieldConfigDialog?.field.type === 'number' && (
                    <div className="grid grid-cols-2 gap-3">
                      <div className="space-y-1">
                        <label className="text-[10px] uppercase tracking-wider text-muted-foreground">
                          {t('formulare.validation.minValue')}
                        </label>
                        <input
                          type="number"
                          value={configMin}
                          onChange={(e) => setConfigMin(e.target.value)}
                          placeholder={t('formulare.validation.optional')}
                          className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                        />
                      </div>
                      <div className="space-y-1">
                        <label className="text-[10px] uppercase tracking-wider text-muted-foreground">
                          {t('formulare.validation.maxValue')}
                        </label>
                        <input
                          type="number"
                          value={configMax}
                          onChange={(e) => setConfigMax(e.target.value)}
                          placeholder={t('formulare.validation.optional')}
                          className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                        />
                      </div>
                    </div>
                  )}

                  {showFieldConfigDialog?.field.type === 'text' && (
                    <div className="space-y-1">
                      <label className="text-[10px] uppercase tracking-wider text-muted-foreground">
                        {t('formulare.validation.patternLabel')}
                      </label>
                      <select
                        value={configPatternType}
                        onChange={(e) =>
                          setConfigPatternType(e.target.value as FieldPatternType)
                        }
                        className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                      >
                        {PATTERN_TYPE_OPTION_KEYS.map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {t(opt.labelKey)}
                          </option>
                        ))}
                      </select>
                      {configPatternType !== 'free' && (
                        <p className="text-[10px] text-muted-foreground">
                          {t('formulare.validation.patternHint')}
                        </p>
                      )}
                    </div>
                  )}
                </div>
              )}

              {/* 10.1 — Conditional Logic Section */}
              <div className="border-t border-border-muted pt-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Zap className="h-4 w-4 text-warning" />
                    <label className="text-sm font-medium text-foreground">
                      {t('formulare.fieldConfig.bedingteAnzeige')}
                    </label>
                  </div>
                  <button
                    onClick={() =>
                      setConfigConditionalEnabled(!configConditionalEnabled)
                    }
                    className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                      configConditionalEnabled ? 'bg-primary' : 'bg-secondary'
                    }`}
                  >
                    <span
                      className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                        configConditionalEnabled
                          ? 'translate-x-4.5'
                          : 'translate-x-0.5'
                      }`}
                    />
                  </button>
                </div>

                {configConditionalEnabled && (
                  <div className="rounded-lg border border-border bg-secondary/20 p-3 space-y-3">
                    <p className="text-xs text-muted-foreground">
                      {t('formulare.fieldConfig.zeigeWenn')}
                    </p>

                    {/* Source field dropdown */}
                    <div className="space-y-1">
                      <label className="text-[10px] uppercase tracking-wider text-muted-foreground">
                        {t('formulare.fieldConfig.quellfeld')}
                      </label>
                      <select
                        value={configConditionalFieldId}
                        onChange={(e) =>
                          setConfigConditionalFieldId(e.target.value)
                        }
                        className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                      >
                        <option value="">
                          {t('formulare.fieldConfig.quellFeldSelect')}
                        </option>
                        {draft.fields
                          .filter(
                            (f) =>
                              f.id !== showFieldConfigDialog?.field.id &&
                              f.label !== '__page_break__',
                          )
                          .map((f) => (
                            <option key={f.id} value={f.id}>
                              {f.label}
                            </option>
                          ))}
                      </select>
                    </div>

                    {/* Operator dropdown */}
                    <div className="space-y-1">
                      <label className="text-[10px] uppercase tracking-wider text-muted-foreground">
                        {t('formulare.fieldConfig.bedingung')}
                      </label>
                      <select
                        value={configConditionalOperator}
                        onChange={(e) =>
                          setConfigConditionalOperator(
                            e.target.value as
                              | 'equals'
                              | 'not_equals'
                              | 'contains',
                          )
                        }
                        className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                      >
                        <option value="equals">
                          {t('formulare.fieldConfig.istGleich')}
                        </option>
                        <option value="not_equals">
                          {t('formulare.fieldConfig.istNichtGleich')}
                        </option>
                        <option value="contains">
                          {t('formulare.fieldConfig.enthaelt')}
                        </option>
                      </select>
                    </div>

                    {/* Value input */}
                    <div className="space-y-1">
                      <label className="text-[10px] uppercase tracking-wider text-muted-foreground">
                        {t('formulare.fieldConfig.wert')}
                      </label>
                      <input
                        type="text"
                        value={configConditionalValue}
                        onChange={(e) =>
                          setConfigConditionalValue(e.target.value)
                        }
                        placeholder={t('formulare.fieldConfig.wertPlaceholder')}
                        className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                      />
                    </div>
                  </div>
                )}
              </div>
            </div>

            <DialogFooter className="border-t border-border px-5 py-4">
              <button
                onClick={() => setShowFieldConfigDialog(null)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('common.cancel')}
              </button>
              <button
                onClick={saveFieldConfig}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                {t('common.save')}
              </button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    )
  }

  // ---------------------------------------------------------------------------
  // JSX — Main View
  // ---------------------------------------------------------------------------

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <PageHeader
        title={t('formulare.page.title')}
        description={t('formulare.page.description', {
          count: activeFormCount,
          new: newSubmissionCount,
        })}
        icon={FileInput}
        moduleId="formulare"
        className="mb-6"
        actions={
          <div className="flex items-center gap-2">
            <RestrictedModeBadge module="formulare" />
            {canSchemasCreate && (
              <button
                onClick={() => {
                  resetNewFormDialog()
                  setShowNewFormDialog(true)
                }}
                className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Plus className="h-4 w-4" />
                {t('formulare.actions.neuesFormular')}
              </button>
            )}
          </div>
        }
      />

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
        <div className="rounded-xl border border-border bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
              <FileText className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-2xl font-semibold text-foreground">
                {activeFormCount}
              </p>
              <p className="text-xs text-muted-foreground">
                {t('formulare.stats.aktiveFormulare')}
              </p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-border bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info-light">
              <Inbox className="h-5 w-5 text-info" />
            </div>
            <div>
              <p className="text-2xl font-semibold text-foreground">
                {weeklySubmissionCount}
              </p>
              <p className="text-xs text-muted-foreground">
                {t('formulare.stats.eingaengeWoche')}
              </p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-border bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-success-light">
              <ClipboardList className="h-5 w-5 text-success" />
            </div>
            <div>
              <p className="text-2xl font-semibold text-foreground">
                {newSubmissionCount}
              </p>
              <p className="text-xs text-muted-foreground">
                {t('formulare.stats.neueEingaenge')}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {(
          [
            {
              key: 'formulare' as const,
              label: t('formulare.tabs.meineFormulare', {
                count: activeForms.length,
              }),
            },
            {
              key: 'eingänge' as const,
              label: t('formulare.tabs.eingaenge', {
                count: allSubmissions.length,
              }),
            },
            {
              key: 'vorlagen' as const,
              label: t('formulare.tabs.vorlagen', {
                count: templates.length,
              }),
            },
          ] as const
        ).filter((tabItem) => visibleTabs.includes(tabItem.key)).map((tabItem) => (
          <button
            key={tabItem.key}
            onClick={() => {
              setTab(tabItem.key)
              setSearch('')
            }}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === tabItem.key
                ? 'border-primary text-primary font-medium tab-accent-active'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {tabItem.label}
          </button>
        ))}
      </div>

      {/* Search */}
      {tab !== 'vorlagen' && (
        <div className="flex flex-wrap items-center gap-3 mb-4">
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder={
                tab === 'formulare'
                  ? t('formulare.search.formulare')
                  : t('formulare.search.eingaenge')
              }
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {tab === 'eingänge' && allSubmissions.length > 0 && (
            <p className="text-xs text-muted-foreground">
              {t('formulare.export.perFormHint')}
            </p>
          )}
        </div>
      )}

      {/* ====================== MEINE FORMULARE TAB ====================== */}
      {tab === 'formulare' && (
        <>
          {activeForms.length === 0 ? (
            <EmptyState
              icon={FileText}
              title={t('formulare.empty.noFormulare')}
              description={t('formulare.empty.createHint')}
              action={canSchemasCreate ? {
                label: t('formulare.empty.createLabel'),
                onClick: () => {
                  resetNewFormDialog()
                  setShowNewFormDialog(true)
                },
              } : undefined}
            />
          ) : (
            <>
              {/* FT-4 — status quick-filter chips + grid/list view toggle */}
              <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
                <div className="flex flex-wrap items-center gap-1.5">
                  {(
                    [
                      { value: 'all', label: t('formulare.submission.filter.all'), count: formStatusCounts.all },
                      { value: 'draft', label: formStatusLabels.draft, count: formStatusCounts.draft },
                      { value: 'active', label: formStatusLabels.active, count: formStatusCounts.active },
                      { value: 'closed', label: formStatusLabels.closed, count: formStatusCounts.closed },
                      { value: 'archived', label: formStatusLabels.archived, count: formStatusCounts.archived },
                    ] as const
                  ).map((chip) => {
                    const active = formStatusFilter === chip.value
                    return (
                      <button
                        key={chip.value}
                        onClick={() => setFormStatusFilter(chip.value)}
                        className={`flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs transition-colors ${
                          active
                            ? 'bg-primary-light text-primary font-medium'
                            : 'text-muted-foreground hover:bg-secondary'
                        }`}
                      >
                        {chip.label}
                        <span
                          className={`rounded-full px-1.5 text-[10px] ${
                            active ? 'bg-primary/15' : 'bg-secondary'
                          }`}
                        >
                          {chip.count}
                        </span>
                      </button>
                    )
                  })}
                </div>
                <div className="flex items-center rounded-md border border-border">
                  <button
                    onClick={() => setFormView('grid')}
                    title={t('formulare.view.grid')}
                    aria-label={t('formulare.view.grid')}
                    className={`rounded-l-md p-1.5 transition-colors ${
                      formView === 'grid'
                        ? 'bg-secondary text-foreground'
                        : 'text-muted-foreground hover:bg-secondary'
                    }`}
                  >
                    <LayoutGrid className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => setFormView('list')}
                    title={t('formulare.view.list')}
                    aria-label={t('formulare.view.list')}
                    className={`rounded-r-md border-l border-border p-1.5 transition-colors ${
                      formView === 'list'
                        ? 'bg-secondary text-foreground'
                        : 'text-muted-foreground hover:bg-secondary'
                    }`}
                  >
                    <LayoutList className="h-4 w-4" />
                  </button>
                </div>
              </div>

              {filteredForms.length === 0 ? (
                <EmptyState
                  icon={FileText}
                  title={t('formulare.empty.noFormulare')}
                  description={t('formulare.empty.suchHint')}
                  action={{
                    label: t('formulare.empty.resetFilter'),
                    onClick: () => {
                      setSearch('')
                      setFormStatusFilter('all')
                    },
                  }}
                />
              ) : formView === 'grid' ? (
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                  {filteredForms.map((schema) => (
                    <div
                      key={schema.id}
                      role="button"
                      tabIndex={0}
                      onClick={() => openFormDetail(schema)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          openFormDetail(schema)
                        }
                      }}
                      className="rounded-xl border border-border bg-card p-4 transition-all duration-200 hover:shadow-md hover:-translate-y-0.5 cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                    >
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex items-center gap-3 min-w-0">
                          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light shrink-0">
                            <FileText className="h-5 w-5 text-primary" />
                          </div>
                          <div className="min-w-0">
                            <h4 className="text-sm font-medium text-foreground truncate">
                              {schema.title}
                            </h4>
                            <p className="text-xs text-muted-foreground truncate">
                              {schema.createdBy ?? ''}
                            </p>
                          </div>
                        </div>
                        <div onClick={(e) => e.stopPropagation()}>
                          <ItemActions items={getFormActions(schema)} />
                        </div>
                      </div>

                      <p className="text-xs text-muted-foreground line-clamp-2 mb-3">
                        {schema.description}
                      </p>

                      <div className="flex items-center gap-2 mb-3 flex-wrap">
                        <span
                          className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                            formStatusColors[schema.status]
                          }`}
                        >
                          {formStatusLabels[schema.status]}
                        </span>
                        {/* 10.4 — Public badge */}
                        {schema.isPublic && (
                          <span className="rounded-full bg-info-light px-2 py-0.5 text-[10px] font-medium text-info">
                            {t('formulare.card.oeffentlich')}
                          </span>
                        )}
                      </div>

                      <div className="flex items-center justify-between border-t border-border-muted pt-3 text-xs text-muted-foreground">
                        <span>
                          {t('formulare.card.felder', {
                            count: schema.fields.filter(
                              (f) => (f as FormField).label !== '__page_break__',
                            ).length,
                          })}
                        </span>
                        <span>
                          {t('formulare.card.eingaenge', {
                            count: schema.submissionCount,
                          })}
                        </span>
                        <span>{formatDate(schema.createdAt)}</span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                /* FT-4 — compact list view */
                <div className="overflow-hidden rounded-xl border border-border">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border-muted bg-secondary/30 text-xs text-muted-foreground">
                        <th className="px-4 py-2.5 text-left font-medium">
                          {t('formulare.list.title')}
                        </th>
                        <th className="px-4 py-2.5 text-left font-medium">
                          {t('formulare.submission.table.status')}
                        </th>
                        <th className="px-4 py-2.5 text-right font-medium">
                          {t('formulare.list.fields')}
                        </th>
                        <th className="px-4 py-2.5 text-right font-medium">
                          {t('formulare.detail.submissions')}
                        </th>
                        <th className="hidden px-4 py-2.5 text-left font-medium sm:table-cell">
                          {t('formulare.detail.createdAt')}
                        </th>
                        <th className="px-4 py-2.5 text-right font-medium" />
                      </tr>
                    </thead>
                    <tbody>
                      {filteredForms.map((schema) => (
                        <tr
                          key={schema.id}
                          role="button"
                          tabIndex={0}
                          onClick={() => openFormDetail(schema)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' || e.key === ' ') {
                              e.preventDefault()
                              openFormDetail(schema)
                            }
                          }}
                          className="cursor-pointer border-b border-border-muted last:border-0 transition-colors hover:bg-secondary/50 focus:outline-none focus-visible:bg-secondary/50"
                        >
                          <td className="px-4 py-2.5">
                            <div className="flex items-center gap-2.5 min-w-0">
                              <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary-light">
                                <FileText className="h-3.5 w-3.5 text-primary" />
                              </div>
                              <div className="min-w-0">
                                <p className="truncate font-medium text-foreground">
                                  {schema.title}
                                </p>
                                <p className="truncate text-xs text-muted-foreground">
                                  {schema.createdBy ?? ''}
                                </p>
                              </div>
                            </div>
                          </td>
                          <td className="px-4 py-2.5">
                            <div className="flex flex-wrap items-center gap-1.5">
                              <span
                                className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                                  formStatusColors[schema.status]
                                }`}
                              >
                                {formStatusLabels[schema.status]}
                              </span>
                              {schema.isPublic && (
                                <span className="rounded-full bg-info-light px-2 py-0.5 text-[10px] font-medium text-info">
                                  {t('formulare.card.oeffentlich')}
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="px-4 py-2.5 text-right text-muted-foreground">
                            {schema.fields.filter(
                              (f) => (f as FormField).label !== '__page_break__',
                            ).length}
                          </td>
                          <td className="px-4 py-2.5 text-right text-muted-foreground">
                            {schema.submissionCount}
                          </td>
                          <td className="hidden px-4 py-2.5 text-muted-foreground sm:table-cell">
                            {formatDate(schema.createdAt)}
                          </td>
                          <td
                            className="px-4 py-2.5 text-right"
                            onClick={(e) => e.stopPropagation()}
                          >
                            <ItemActions items={getFormActions(schema)} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </>
      )}

      {/* ====================== EINGAENGE TAB ====================== */}
      {tab === 'eingänge' && (
        <>
          {Object.keys(submissionsByForm).length === 0 &&
          activeForms.length === 0 ? (
            <EmptyState
              icon={Inbox}
              title={t('formulare.empty.noEingaenge')}
              description={
                search
                  ? t('formulare.empty.suchHint')
                  : t('formulare.empty.eingaengeHint')
              }
            />
          ) : (
            <div className="space-y-4">
              {/* Show schemas with known submissions first, then others */}
              {activeForms.map((schema) => {
                const isExpanded = expandedGroups.has(schema.id)
                const schemaSubmissions =
                  submissionsByForm[schema.id]?.items ?? []
                const newCount = schemaSubmissions.filter(
                  (s) => s.status === 'new',
                ).length

                return (
                  <div
                    key={schema.id}
                    className="rounded-lg border border-border bg-card overflow-hidden"
                  >
                    {/* Group header */}
                    <div className="flex items-center gap-2 pr-3 hover:bg-secondary/50 transition-colors">
                      <button
                        onClick={() => toggleGroup(schema.id)}
                        className="flex flex-1 items-center gap-3 px-4 py-3 text-left"
                      >
                        {isExpanded ? (
                          <ChevronDown className="h-4 w-4 text-muted-foreground" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-muted-foreground" />
                        )}
                        <FileText className="h-4 w-4 text-primary" />
                        <span className="text-sm font-medium text-foreground">
                          {schema.title}
                        </span>
                        <span className="text-xs text-muted-foreground">
                          {t('formulare.submission.eingaenge', {
                            count: schema.submissionCount,
                          })}
                        </span>
                        {newCount > 0 && (
                          <span className="rounded-full bg-info-light px-2 py-0.5 text-[10px] font-medium text-info">
                            {t('formulare.submission.neu', { count: newCount })}
                          </span>
                        )}
                      </button>
                      {schema.submissionCount > 0 && canExportRun && (
                        <ExportMenu
                          schema={schema}
                          onExport={handleExportSubmissions}
                          t={t}
                          compact
                          defaultFormat={prefsExportFormat}
                        />
                      )}
                    </div>

                    {/* Submissions table — loaded lazily */}
                    {isExpanded && (
                      <SubmissionsPanel
                        schemaId={schema.id}
                        onSelectSubmission={setSelectedSubmission}
                        onUpdateStatus={(sub, status) =>
                          handleSubmissionStatus(sub, status, schema.id)
                        }
                        submissionStatusLabels={submissionStatusLabels}
                        submissionStatusColors={submissionStatusColors}
                        formatDateTime={formatDateTime}
                        t={t}
                        canWrite={canSubsWrite}
                      />
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </>
      )}

      {/* ====================== VORLAGEN TAB ====================== */}
      {tab === 'vorlagen' && (
        <>
          {templates.length === 0 ? (
            <EmptyState
              icon={LayoutTemplate}
              title={t('formulare.empty.noVorlagen')}
              description={t('formulare.empty.vorlagenHint')}
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {templates.map((tmpl) => (
                <div
                  key={tmpl.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => setSelectedTemplate(tmpl)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault()
                      setSelectedTemplate(tmpl)
                    }
                  }}
                  className="cursor-pointer rounded-lg border-2 border-dashed border-border bg-card p-4 transition-colors hover:border-primary/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                >
                  <div className="flex items-center gap-3 mb-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-secondary">
                      <LayoutTemplate className="h-5 w-5 text-muted-foreground" />
                    </div>
                    <div>
                      <h4 className="text-sm font-medium text-foreground">
                        {tmpl.title}
                      </h4>
                      <p className="text-xs text-muted-foreground">
                        {t('formulare.card.felder', {
                          count: tmpl.fields.length,
                        })}
                      </p>
                    </div>
                  </div>

                  <p className="text-xs text-muted-foreground line-clamp-2 mb-4">
                    {tmpl.description}
                  </p>

                  <div className="flex flex-wrap gap-1.5 mb-4">
                    {(tmpl.fields as FormField[]).map((field) => {
                      const Icon = FIELD_TYPE_ICONS[field.type]
                      return (
                        <span
                          key={field.id}
                          className="inline-flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground"
                        >
                          <Icon className="h-3 w-3" />
                          {field.label}
                        </span>
                      )
                    })}
                  </div>

                  {canSchemasCreate && (
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        handleUseTemplate(tmpl)
                      }}
                      className="w-full rounded-lg border border-border px-3 py-2 text-sm font-medium text-foreground hover:bg-secondary transition-colors"
                    >
                      {t('formulare.vorlagen.verwenden')}
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {/* ====================== SUBMISSION DETAIL MODAL ====================== */}
      <DetailModal
        open={!!selectedSubmission}
        onClose={() => setSelectedSubmission(null)}
        title={
          selectedSubmission
            ? (schemas.find((s) => s.id === selectedSubmission.formSchemaId)
                ?.title ?? '')
            : ''
        }
        subtitle={
          selectedSubmission
            ? t('formulare.submission.eingegangen', {
                date: formatDateTime(selectedSubmission.submittedAt),
              })
            : ''
        }
        badge={
          selectedSubmission ? (
            <span
              className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                submissionStatusColors[selectedSubmission.status]
              }`}
            >
              {submissionStatusLabels[selectedSubmission.status]}
            </span>
          ) : undefined
        }
        maxWidth="max-w-lg"
        footer={
          selectedSubmission ? (
            <div className="flex items-center gap-2">
              {canSubsWrite && selectedSubmission.status === 'new' && (
                <button
                  onClick={() => {
                    updateSubStatus.mutate(
                      {
                        id: selectedSubmission.id,
                        status: 'read',
                        formSchemaId: selectedSubmission.formSchemaId ?? '',
                      },
                      {
                        onSuccess: () => {
                          setSelectedSubmission({
                            ...selectedSubmission,
                            status: 'read',
                          })
                          toast.success(t('formulare.toast.alsGelesenMarkiert'))
                        },
                      },
                    )
                  }}
                  className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
                >
                  <Eye className="h-4 w-4" />
                  {t('formulare.submission.alsGelesenButton')}
                </button>
              )}
              {canSubsWrite && selectedSubmission.status !== 'archived' && (
                <button
                  onClick={() => {
                    updateSubStatus.mutate(
                      {
                        id: selectedSubmission.id,
                        status: 'archived',
                        formSchemaId: selectedSubmission.formSchemaId ?? '',
                      },
                      {
                        onSuccess: () => {
                          setSelectedSubmission({
                            ...selectedSubmission,
                            status: 'archived',
                          })
                          toast.success(t('formulare.toast.eingangArchiviert'))
                        },
                      },
                    )
                  }}
                  className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
                >
                  {t('formulare.submission.archivieren')}
                </button>
              )}
            </div>
          ) : undefined
        }
      >
        {selectedSubmission && (
          <div className="space-y-5">
            {/* Submitter meta */}
            <div className="grid grid-cols-2 gap-3">
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                  {t('formulare.submission.absender')}
                </p>
                <p className="text-sm font-medium text-foreground truncate">
                  {selectedSubmission.submittedBy ?? '--'}
                </p>
              </div>
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                  {t('formulare.submission.ipAdresse')}
                </p>
                <p className="text-sm font-medium text-foreground truncate">
                  {selectedSubmission.ipAddress ?? '--'}
                </p>
              </div>
            </div>

            {/* DSGVO consent confirmation (if the form captured one) */}
            {(() => {
              const schemaFields =
                (schemas.find((s) => s.id === selectedSubmission.formSchemaId)
                  ?.fields as FormField[] | undefined) ?? []
              const consentField = schemaFields.find((f) => f.type === 'consent')
              if (!consentField) return null
              const accepted = !!selectedSubmission.answers[consentField.id]
              return (
                <div
                  className={`rounded-lg border p-3 ${
                    accepted
                      ? 'border-success/30 bg-success-light'
                      : 'border-warning/30 bg-warning-light'
                  }`}
                >
                  <div className="flex items-start gap-2.5">
                    <ShieldCheck
                      className={`mt-0.5 h-4 w-4 shrink-0 ${
                        accepted ? 'text-success' : 'text-warning'
                      }`}
                    />
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-foreground">
                        {accepted
                          ? t('formulare.consent.accepted', {
                              date: formatDateTime(selectedSubmission.submittedAt),
                            })
                          : t('formulare.consent.notAccepted')}
                      </p>
                      {consentField.consentText && (
                        <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                          {consentField.consentText}
                        </p>
                      )}
                    </div>
                  </div>
                </div>
              )
            })()}

            {/* Answers */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">
                {t('formulare.submission.antworten')}
              </h4>
              <div className="rounded-lg border border-border divide-y divide-border-muted">
                {Object.entries(selectedSubmission.answers).map(
                  ([fieldId, value]) => {
                    const field = getFieldForSubmission(
                      selectedSubmission.formSchemaId ?? '',
                      fieldId,
                    )
                    // Consent is surfaced in its own block above — skip here.
                    if (field?.type === 'consent') return null
                    // 10.1 — Skip fields hidden by conditional logic
                    if (field?.conditionalLogic) {
                      const visible = evaluateCondition(
                        field.conditionalLogic,
                        selectedSubmission.answers,
                      )
                      if (!visible) return null
                    }
                    return (
                      <div key={fieldId} className="px-3 py-3">
                        <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5">
                          {field?.label ?? fieldId}
                        </p>
                        {renderAnswer(
                          fieldId,
                          value,
                          selectedSubmission.formSchemaId ?? '',
                        )}
                      </div>
                    )
                  },
                )}
              </div>
            </div>

            {/* FO-4 — notifications dispatched when this submission arrived (mocked) */}
            {selectedSubmission.notifications &&
              (selectedSubmission.notifications.recipients.length > 0 ||
                selectedSubmission.notifications.confirmationTo) && (
                <div>
                  <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">
                    {t('formulare.submission.notifyTitle')}
                  </h4>
                  <div className="space-y-2 rounded-lg border border-border bg-secondary/30 p-3">
                    {selectedSubmission.notifications.recipients.length > 0 && (
                      <div className="flex items-start gap-2 text-sm text-foreground">
                        <Send className="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                        <span>
                          {t('formulare.submission.notifySent', {
                            list: selectedSubmission.notifications.recipients.join(', '),
                          })}
                        </span>
                      </div>
                    )}
                    {selectedSubmission.notifications.confirmationTo && (
                      <div className="flex items-start gap-2 text-sm text-foreground">
                        <Check className="mt-0.5 h-3.5 w-3.5 shrink-0 text-success" />
                        <span>
                          {t('formulare.submission.notifyConfirmation', {
                            email: selectedSubmission.notifications.confirmationTo,
                          })}
                        </span>
                      </div>
                    )}
                    <p className="text-[10px] text-muted-foreground">
                      {t('formulare.submission.notifySentAt', {
                        date: formatDateTime(selectedSubmission.notifications.sentAt),
                      })}
                    </p>
                  </div>
                </div>
              )}
          </div>
        )}
      </DetailModal>

      {/* ====================== FORMULAR DETAIL MODAL ====================== */}
      <DetailModal
        open={!!selectedForm}
        onClose={() => setSelectedForm(null)}
        maxWidth="max-w-3xl"
        title={selectedForm?.title ?? ''}
        subtitle={
          selectedForm
            ? t('formulare.detail.subtitle', {
                fields: selectedForm.fields.filter(
                  (f) => (f as FormField).label !== '__page_break__',
                ).length,
                submissions: selectedForm.submissionCount,
              })
            : ''
        }
        badge={
          selectedForm ? (
            <span
              className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                formStatusColors[selectedForm.status]
              }`}
            >
              {formStatusLabels[selectedForm.status]}
            </span>
          ) : undefined
        }
        footer={
          selectedForm ? (
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex flex-wrap items-center gap-1.5">
                {canSchemasCreate && (
                  <button
                    onClick={() => {
                      duplicateSchema.mutate(
                        { id: selectedForm.id },
                        {
                          onSuccess: () => {
                            toast.success(
                              t('formulare.toast.dupliziert', {
                                name: selectedForm.title,
                              }),
                            )
                            setSelectedForm(null)
                          },
                          onError: (err) =>
                            toast.error(
                              err instanceof Error ? err.message : t('common.error'),
                            ),
                        },
                      )
                    }}
                    className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors"
                  >
                    <Copy className="h-3.5 w-3.5" />
                    {t('formulare.actions.duplizieren')}
                  </button>
                )}
                {canShareManage && (
                  <button
                    onClick={() => requestShare(selectedForm)}
                    disabled={!isShareable(selectedForm)}
                    title={
                      !isShareable(selectedForm)
                        ? t('formulare.lifecycle.shareOnlyActive')
                        : undefined
                    }
                    className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-transparent"
                  >
                    <Share2 className="h-3.5 w-3.5" />
                    {t('formulare.actions.teilen')}
                  </button>
                )}
                {/* FD-0 — status-dependent lifecycle action (schemas:publish required) */}
                {canSchemasPublish && selectedForm.status === 'draft' && (
                  <button
                    onClick={() => publishForm(selectedForm)}
                    className="flex items-center gap-1.5 rounded-lg border border-success/30 bg-success-light px-2.5 py-1.5 text-xs font-medium text-success transition-colors hover:opacity-80"
                  >
                    <Send className="h-3.5 w-3.5" />
                    {t('formulare.lifecycle.publish')}
                  </button>
                )}
                {canSchemasPublish && selectedForm.status === 'active' && (
                  <button
                    onClick={() => openCloseDialog(selectedForm)}
                    className="flex items-center gap-1.5 rounded-lg border border-warning/30 bg-warning-light px-2.5 py-1.5 text-xs font-medium text-warning transition-colors hover:opacity-80"
                  >
                    <Lock className="h-3.5 w-3.5" />
                    {t('formulare.lifecycle.close')}
                  </button>
                )}
                {canSchemasPublish && selectedForm.status === 'closed' && (
                  <button
                    onClick={() => reopenForm(selectedForm)}
                    className="flex items-center gap-1.5 rounded-lg border border-success/30 bg-success-light px-2.5 py-1.5 text-xs font-medium text-success transition-colors hover:opacity-80"
                  >
                    <Unlock className="h-3.5 w-3.5" />
                    {t('formulare.lifecycle.reopen')}
                  </button>
                )}
                {canSchemasPublish && selectedForm.status === 'archived' && (
                  <button
                    onClick={() => restoreForm(selectedForm)}
                    className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors"
                  >
                    <RotateCcw className="h-3.5 w-3.5" />
                    {t('formulare.lifecycle.restore')}
                  </button>
                )}
                {canSchemasPublish &&
                  (selectedForm.status === 'active' ||
                    selectedForm.status === 'closed') && (
                  <button
                    onClick={() => archiveForm(selectedForm)}
                    className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors"
                  >
                    <Archive className="h-3.5 w-3.5" />
                    {t('formulare.actions.archivieren')}
                  </button>
                )}
                {selectedForm.submissionCount > 0 && canExportRun && (
                  <ExportMenu
                    schema={selectedForm}
                    onExport={handleExportSubmissions}
                    t={t}
                    compact
                    defaultFormat={prefsExportFormat}
                  />
                )}
                {canSchemasDelete && (
                  <button
                    onClick={() => {
                      const form = selectedForm
                      setSelectedForm(null)
                      setConfirmDelete(form)
                    }}
                    className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-error hover:bg-error-light transition-colors"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                    {t('common.delete')}
                  </button>
                )}
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setPreviewSchema(selectedForm)}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
                >
                  <Eye className="h-4 w-4" />
                  {t('formulare.editor.vorschau')}
                </button>
                {canSchemasEdit && (
                  <button
                    onClick={() => {
                      const form = selectedForm
                      setSelectedForm(null)
                      openEditor(form)
                    }}
                    className="flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
                  >
                    <Edit className="h-4 w-4" />
                    {t('formulare.actions.bearbeiten')}
                  </button>
                )}
              </div>
            </div>
          ) : undefined
        }
      >
        {selectedForm && (
          <div className="space-y-5">
            {/* FT-1 — tabs: Details / Eingänge / Auswertung / Verteilung.
                RBAC: eingaenge+auswertung require submissions:read,
                verteilung requires share:manage, details is always free. */}
            {(() => {
              const allDetailTabs = [
                { key: 'details' as const, label: t('formulare.detail.tabDetails'), visible: true },
                {
                  key: 'eingaenge' as const,
                  label: t('formulare.detail.tabEingaenge', { count: selectedForm.submissionCount }),
                  visible: canSubsRead,
                },
                {
                  key: 'auswertung' as const,
                  label: t('formulare.detail.tabAuswertung'),
                  visible: canSubsRead,
                },
                {
                  key: 'verteilung' as const,
                  label: t('formulare.detail.tabVerteilung', { count: shareLinkCount }),
                  visible: canShareManage,
                },
              ].filter((dt) => dt.visible)
              // If the active detail-tab is no longer visible, redirect to first visible.
              const activeDetailTab = allDetailTabs.some((dt) => dt.key === formDetailTab)
                ? formDetailTab
                : (allDetailTabs[0]?.key ?? 'details')
              // Sync state if needed (run outside render via a stable comparison).
              if (activeDetailTab !== formDetailTab) {
                // Use a microtask to avoid setState-in-render.
                Promise.resolve().then(() => setFormDetailTab(activeDetailTab))
              }
              return (
                <div className="flex items-center gap-4 border-b border-border">
                  {allDetailTabs.map((dtTab) => (
                    <button
                      key={dtTab.key}
                      onClick={() => setFormDetailTab(dtTab.key)}
                      className={`-mb-px border-b-2 px-1 pb-2 text-sm transition-colors ${
                        activeDetailTab === dtTab.key
                          ? 'border-primary text-primary font-medium'
                          : 'border-transparent text-muted-foreground hover:text-foreground'
                      }`}
                    >
                      {dtTab.label}
                    </button>
                  ))}
                </div>
              )
            })()}

            {formDetailTab === 'details' && (
              <div className="space-y-5">
            {selectedForm.description && (
              <p className="text-sm text-muted-foreground">
                {selectedForm.description}
              </p>
            )}

            {/* Meta grid */}
            <div className="grid grid-cols-2 gap-3">
              {[
                {
                  label: t('formulare.detail.visibility'),
                  value: selectedForm.isPublic
                    ? t('formulare.card.oeffentlich')
                    : t('formulare.detail.private'),
                },
                {
                  label: t('formulare.detail.submissions'),
                  value: String(selectedForm.submissionCount),
                },
                {
                  label: t('formulare.detail.createdBy'),
                  value: selectedForm.createdBy ?? '--',
                },
                {
                  label: t('formulare.detail.createdAt'),
                  value: formatDate(selectedForm.createdAt),
                },
                {
                  label: t('formulare.detail.updatedAt'),
                  value: formatDate(selectedForm.updatedAt),
                },
                {
                  label: t('formulare.detail.pages'),
                  value: String(
                    Math.max(1, selectedForm.pageCount),
                  ),
                },
              ].map((item) => (
                <div
                  key={item.label}
                  className="rounded-lg border border-border bg-secondary/30 p-3"
                >
                  <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                    {item.label}
                  </p>
                  <p className="text-sm font-medium text-foreground truncate">
                    {item.value}
                  </p>
                </div>
              ))}
            </div>

            {/* Field list */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">
                {t('formulare.editor.felder', {
                  count: selectedForm.fields.filter(
                    (f) => (f as FormField).label !== '__page_break__',
                  ).length,
                })}
              </h4>
              {selectedForm.fields.filter(
                (f) => (f as FormField).label !== '__page_break__',
              ).length === 0 ? (
                <p className="text-sm text-muted-foreground italic">
                  {t('formulare.detail.noFields')}
                </p>
              ) : (
                <div className="rounded-lg border border-border divide-y divide-border-muted">
                  {(selectedForm.fields as FormField[])
                    .filter((f) => f.label !== '__page_break__')
                    .map((field) => {
                      const Icon = FIELD_TYPE_ICONS[field.type]
                      return (
                        <div
                          key={field.id}
                          className="flex items-center gap-3 px-3 py-2.5"
                        >
                          <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary-light shrink-0">
                            <Icon className="h-3.5 w-3.5 text-primary" />
                          </div>
                          <span className="flex-1 text-sm text-foreground truncate">
                            {field.label}
                          </span>
                          {field.required && (
                            <span className="rounded bg-error-light px-1.5 py-0.5 text-[9px] font-medium text-error shrink-0">
                              {t('formulare.editor.pflicht')}
                            </span>
                          )}
                          <span className="text-[10px] text-muted-foreground shrink-0">
                            {FIELD_TYPE_LABELS[field.type]}
                          </span>
                        </div>
                      )
                    })}
                </div>
              )}
            </div>
              </div>
            )}

            {/* FT-1 — submissions for this form, without leaving the modal */}
            {formDetailTab === 'eingaenge' && canSubsRead && (
              <div className="overflow-hidden rounded-lg border border-t-0 border-border">
                <SubmissionsPanel
                  schemaId={selectedForm.id}
                  onSelectSubmission={setSelectedSubmission}
                  onUpdateStatus={(sub, status) =>
                    handleSubmissionStatus(sub, status, selectedForm.id)
                  }
                  submissionStatusLabels={submissionStatusLabels}
                  submissionStatusColors={submissionStatusColors}
                  formatDateTime={formatDateTime}
                  t={t}
                  canWrite={canSubsWrite}
                />
              </div>
            )}

            {/* FT-3a — per-field evaluation dashboard for this form */}
            {formDetailTab === 'auswertung' && canSubsRead && (
              <EvaluationPanel
                schema={selectedForm}
                formatDate={formatDate}
                t={t}
              />
            )}

            {/* FD-2 — distribution overview: shared links per form */}
            {formDetailTab === 'verteilung' && canShareManage && (
              <ShareLinksPanel
                schemaId={selectedForm.id}
                canShare={isShareable(selectedForm)}
                onShare={() => requestShare(selectedForm)}
                onCopy={(link) =>
                  copyToClipboard(shareValueOf(link, selectedForm.title))
                }
                onToggle={toggleShareLink}
                onDelete={(link) => setShareLinkToDelete(link)}
                formatDate={formatDate}
                t={t}
              />
            )}
          </div>
        )}
      </DetailModal>

      {/* ====================== STANDALONE FORM PREVIEW ====================== */}
      <DetailModal
        open={!!previewSchema}
        onClose={() => setPreviewSchema(null)}
        title={previewSchema?.title || t('formulare.preview.formularname')}
        subtitle={t('formulare.editor.vorschau')}
        onBack={selectedForm ? () => setPreviewSchema(null) : undefined}
      >
        {previewSchema && previewSchema.status === 'closed' ? (
          // FD-0 — closed forms show the close notice externals would see.
          <div className="space-y-5">
            <div className="flex items-center gap-2 rounded-lg bg-info-light px-3 py-2">
              <Info className="h-4 w-4 text-info shrink-0" />
              <p className="text-xs text-info">{t('formulare.preview.banner')}</p>
            </div>
            <div className="flex flex-col items-center gap-3 rounded-xl border border-warning/30 bg-warning-light px-6 py-10 text-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-warning/15">
                <Lock className="h-6 w-6 text-warning" />
              </div>
              <p className="text-base font-semibold text-foreground">
                {t('formulare.preview.closedTitle')}
              </p>
              <p className="max-w-sm text-sm leading-relaxed text-muted-foreground">
                {previewSchema.closedMessage?.trim() ||
                  t('formulare.preview.closedDefault')}
              </p>
            </div>
          </div>
        ) : previewSchema ? (
          <div className="space-y-5">
            <div className="flex items-center gap-2 rounded-lg bg-info-light px-3 py-2">
              <Info className="h-4 w-4 text-info shrink-0" />
              <p className="text-xs text-info">{t('formulare.preview.banner')}</p>
            </div>
            {previewSchema.description && (
              <p className="text-sm text-muted-foreground">
                {previewSchema.description}
              </p>
            )}
            {(previewSchema.fields as FormField[]).filter(
              (f) => f.label !== '__page_break__',
            ).length === 0 ? (
              <p className="text-sm text-muted-foreground italic">
                {t('formulare.detail.noFields')}
              </p>
            ) : (
              <div className="space-y-4">
                {(previewSchema.fields as FormField[])
                  .filter((f) => f.label !== '__page_break__')
                  .map((field) => (
                    <div key={field.id} className="space-y-1.5">
                      <label className="text-sm font-medium text-foreground flex items-center gap-1">
                        {field.label}
                        {field.required && (
                          <span className="text-destructive">*</span>
                        )}
                      </label>
                      {renderFieldPreview(field)}
                    </div>
                  ))}
                <button
                  disabled
                  className="mt-2 rounded-lg bg-primary/50 px-4 py-2 text-sm text-primary-foreground cursor-not-allowed"
                >
                  {t('formulare.editor.absenden')}
                </button>
              </div>
            )}
          </div>
        ) : null}
      </DetailModal>

      {/* ====================== NEUES FORMULAR DIALOG ====================== */}
      <Dialog
        open={showNewFormDialog}
        onOpenChange={(o) => {
          if (!o) setShowNewFormDialog(false)
        }}
      >
        <DialogContent className="gap-0 p-0 max-w-md">
          <DialogHeader className="border-b border-border px-5 py-4">
            <div className="flex items-center gap-2">
              <FileText className="h-5 w-5 text-primary" />
              <DialogTitle className="text-base font-semibold text-foreground">
                {t('formulare.newForm.title')}
              </DialogTitle>
            </div>
            <DialogDescription className="sr-only">
              {t('formulare.newForm.title')}
            </DialogDescription>
          </DialogHeader>

          <div className="p-5 space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('formulare.share.formular')}{' '}
                <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                value={newFormName}
                onChange={(e) => setNewFormName(e.target.value)}
                placeholder={t('formulare.newForm.namePlaceholder')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('formulare.newForm.beschreibung')}
              </label>
              <textarea
                value={newFormDescription}
                onChange={(e) => setNewFormDescription(e.target.value)}
                rows={3}
                placeholder={t('formulare.newForm.beschreibungPlaceholder')}
                className="w-full resize-none rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('formulare.newForm.vonVorlage')}{' '}
                <span className="text-xs text-muted-foreground">
                  {t('formulare.newForm.vonVorlageHint')}
                </span>
              </label>
              <select
                value={newFormTemplateId}
                onChange={(e) => setNewFormTemplateId(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                <option value="">{t('formulare.newForm.leerBeginnen')}</option>
                {templates.map((tmplItem) => (
                  <option key={tmplItem.id} value={tmplItem.id}>
                    {tmplItem.title} (
                    {t('formulare.card.felder', {
                      count: tmplItem.fields.length,
                    })}
                    )
                  </option>
                ))}
              </select>
            </div>
          </div>

          <DialogFooter className="border-t border-border px-5 py-4">
            <button
              onClick={() => setShowNewFormDialog(false)}
              className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleCreateForm}
              disabled={createSchema.isPending || duplicateSchema.isPending}
              className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-60"
            >
              {t('formulare.newForm.erstellen')}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ====================== SHARE DIALOG (FD-1) ====================== */}
      <Dialog
        open={!!showShareDialog}
        onOpenChange={(o) => {
          if (!o) closeShareDialog()
        }}
      >
        <DialogContent className="gap-0 p-0 max-w-md">
          <DialogHeader className="border-b border-border px-5 py-4">
            <div className="flex items-center gap-2">
              <Share2 className="h-5 w-5 text-primary" />
              <DialogTitle className="text-base font-semibold text-foreground">
                {t('formulare.share.title')}
              </DialogTitle>
            </div>
            <DialogDescription className="pt-1 text-sm text-muted-foreground">
              {t('formulare.share.subtitle')}
            </DialogDescription>
          </DialogHeader>

          <div className="p-5 space-y-4">
            <div className="rounded-lg border border-border bg-secondary/30 p-3">
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                {t('formulare.share.formular')}
              </p>
              <p className="text-sm font-medium text-foreground truncate">
                {showShareDialog?.title}
              </p>
            </div>

            {!shareCreatedLink ? (
              <>
                {/* Stage 1 — channel + settings */}
                <div className="space-y-2">
                  <p className="text-sm font-medium text-foreground">
                    {t('formulare.share.channelLabel')}
                  </p>
                  <div className="grid grid-cols-4 gap-2">
                    {(
                      [
                        { value: 'link', labelKey: 'formulare.share.channel.link', Icon: Link2 },
                        { value: 'email', labelKey: 'formulare.share.channel.email', Icon: Mail },
                        { value: 'embed', labelKey: 'formulare.share.channel.embed', Icon: Code2 },
                        { value: 'qr', labelKey: 'formulare.share.channel.qr', Icon: QrCode },
                      ] as const
                    ).map((ch) => {
                      const active = shareChannel === ch.value
                      return (
                        <button
                          key={ch.value}
                          onClick={() => setShareChannel(ch.value)}
                          className={`flex flex-col items-center gap-1.5 rounded-lg border px-2 py-3 text-xs transition-colors ${
                            active
                              ? 'border-primary bg-primary-light text-primary font-medium'
                              : 'border-border text-muted-foreground hover:bg-secondary'
                          }`}
                        >
                          <ch.Icon className="h-4 w-4" />
                          {t(ch.labelKey)}
                        </button>
                      )
                    })}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1.5">
                    <label className="text-sm font-medium text-foreground">
                      {t('formulare.share.expiry')}
                    </label>
                    <input
                      type="date"
                      value={shareExpiry}
                      min={new Date().toISOString().slice(0, 10)}
                      onChange={(e) => setShareExpiry(e.target.value)}
                      className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-sm font-medium text-foreground">
                      {t('formulare.share.maxSubmissions')}
                    </label>
                    <input
                      type="number"
                      min={1}
                      value={shareMaxSubs}
                      onChange={(e) => setShareMaxSubs(e.target.value)}
                      placeholder={t('formulare.share.maxSubmissionsPlaceholder')}
                      className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    />
                  </div>
                </div>
              </>
            ) : (
              <>
                {/* Stage 2 — generated link */}
                <div className="flex items-center gap-2 rounded-lg bg-success-light px-3 py-2">
                  <Check className="h-4 w-4 text-success shrink-0" />
                  <p className="text-xs text-success">
                    {t('formulare.share.created')}
                  </p>
                </div>

                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">
                    {shareCreatedLink.channel === 'embed'
                      ? t('formulare.share.embedLabel')
                      : t('formulare.share.linkLabel')}
                  </label>
                  <div className="flex items-start gap-2">
                    {shareCreatedLink.channel === 'embed' ? (
                      <textarea
                        readOnly
                        rows={3}
                        value={shareValueOf(shareCreatedLink, showShareDialog?.title)}
                        onFocus={(e) => e.currentTarget.select()}
                        className="w-full resize-none rounded-lg border border-border bg-secondary/30 px-3 py-2 font-mono text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                      />
                    ) : (
                      <input
                        readOnly
                        value={shareValueOf(shareCreatedLink, showShareDialog?.title)}
                        onFocus={(e) => e.currentTarget.select()}
                        className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                      />
                    )}
                    <button
                      onClick={() => copyToClipboard(shareValueOf(shareCreatedLink, showShareDialog?.title))}
                      className="flex shrink-0 items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
                    >
                      {shareCopied ? (
                        <Check className="h-4 w-4" />
                      ) : (
                        <Copy className="h-4 w-4" />
                      )}
                      {shareCopied
                        ? t('formulare.share.copied')
                        : t('formulare.share.copy')}
                    </button>
                  </div>
                </div>

                <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                  <span className="inline-flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5">
                    <Calendar className="h-3 w-3" />
                    {shareCreatedLink.expiresAt
                      ? t('formulare.share.expiresOn', {
                          date: formatDate(shareCreatedLink.expiresAt),
                        })
                      : t('formulare.share.noExpiry')}
                  </span>
                  <span className="inline-flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5">
                    <Hash className="h-3 w-3" />
                    {shareCreatedLink.maxSubmissions != null
                      ? t('formulare.share.maxLabel', {
                          count: shareCreatedLink.maxSubmissions,
                        })
                      : t('formulare.share.unlimited')}
                  </span>
                </div>

                <p className="text-xs text-muted-foreground">
                  {t('formulare.share.overviewHint')}
                </p>
              </>
            )}
          </div>

          <DialogFooter className="border-t border-border px-5 py-4">
            {!shareCreatedLink ? (
              <>
                <button
                  onClick={closeShareDialog}
                  className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
                >
                  {t('common.cancel')}
                </button>
                <button
                  onClick={handleCreateShareLink}
                  disabled={createShareLink.isPending}
                  className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:opacity-60"
                >
                  <Share2 className="h-4 w-4" />
                  {t('formulare.share.create')}
                </button>
              </>
            ) : (
              <>
                <button
                  onClick={resetShareDialog}
                  className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
                >
                  {t('formulare.share.another')}
                </button>
                <button
                  onClick={closeShareDialog}
                  className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
                >
                  {t('formulare.share.done')}
                </button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ====================== VORLAGEN DETAIL MODAL (FT-1) ====================== */}
      <DetailModal
        open={!!selectedTemplate}
        onClose={() => setSelectedTemplate(null)}
        title={selectedTemplate?.title ?? ''}
        subtitle={
          selectedTemplate
            ? t('formulare.card.felder', {
                count: (selectedTemplate.fields as FormField[]).filter(
                  (f) => f.label !== '__page_break__',
                ).length,
              })
            : ''
        }
        badge={
          <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
            {t('formulare.vorlagen.badge')}
          </span>
        }
        footer={
          selectedTemplate ? (
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                {canSchemasEdit && (
                  <button
                    onClick={() => {
                      const tmpl = selectedTemplate
                      setSelectedTemplate(null)
                      openEditor(tmpl)
                    }}
                    className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary"
                  >
                    <Edit className="h-4 w-4" />
                    {t('formulare.actions.bearbeiten')}
                  </button>
                )}
                {canSchemasDelete && (
                  <button
                    onClick={() => setTemplateToDelete(selectedTemplate)}
                    className="flex items-center gap-1.5 rounded-lg border border-destructive/30 px-3 py-2 text-sm text-destructive transition-colors hover:bg-error-light"
                  >
                    <Trash2 className="h-4 w-4" />
                    {t('common.delete')}
                  </button>
                )}
              </div>
              {canSchemasCreate && (
                <button
                  onClick={() => {
                    const tmpl = selectedTemplate
                    setSelectedTemplate(null)
                    handleUseTemplate(tmpl)
                  }}
                  className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
                >
                  <Copy className="h-4 w-4" />
                  {t('formulare.vorlagen.verwenden')}
                </button>
              )}
            </div>
          ) : undefined
        }
      >
        {selectedTemplate && (
          <div className="space-y-5">
            {selectedTemplate.description && (
              <p className="text-sm text-muted-foreground">
                {selectedTemplate.description}
              </p>
            )}
            <div>
              <h4 className="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {t('formulare.editor.felder', {
                  count: (selectedTemplate.fields as FormField[]).filter(
                    (f) => f.label !== '__page_break__',
                  ).length,
                })}
              </h4>
              <div className="divide-y divide-border-muted rounded-lg border border-border">
                {(selectedTemplate.fields as FormField[])
                  .filter((f) => f.label !== '__page_break__')
                  .map((field) => {
                    const Icon = FIELD_TYPE_ICONS[field.type]
                    return (
                      <div key={field.id} className="flex items-center gap-3 px-3 py-2.5">
                        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary-light">
                          <Icon className="h-3.5 w-3.5 text-primary" />
                        </div>
                        <span className="flex-1 truncate text-sm text-foreground">
                          {field.label}
                        </span>
                        {field.required && (
                          <span className="shrink-0 rounded bg-error-light px-1.5 py-0.5 text-[9px] font-medium text-error">
                            {t('formulare.editor.pflicht')}
                          </span>
                        )}
                        <span className="shrink-0 text-[10px] text-muted-foreground">
                          {FIELD_TYPE_LABELS[field.type]}
                        </span>
                      </div>
                    )
                  })}
              </div>
            </div>
          </div>
        )}
      </DetailModal>

      {/* ====================== ARCHIVE CONFIRM (FT-1) ====================== */}
      <ConfirmDialog
        open={!!archiveConfirm}
        onOpenChange={() => setArchiveConfirm(null)}
        title={t('formulare.archiveConfirm.title')}
        description={t('formulare.archiveConfirm.description', {
          count: archiveConfirm?.newCount ?? 0,
        })}
        confirmLabel={t('formulare.actions.archivieren')}
        variant="warning"
        onConfirm={() => {
          if (archiveConfirm) doArchive(archiveConfirm.schema)
          setArchiveConfirm(null)
        }}
      />

      {/* ====================== SHARE LINK DELETE (FD-2) ====================== */}
      <ConfirmDialog
        open={!!shareLinkToDelete}
        onOpenChange={() => setShareLinkToDelete(null)}
        title={t('formulare.share.deleteTitle')}
        description={t('formulare.share.deleteDescription')}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={confirmDeleteShareLink}
      />

      {/* ====================== CLOSE FORM DIALOG (FD-0) ====================== */}
      <Dialog
        open={!!closeDialogForm}
        onOpenChange={(o) => {
          if (!o) setCloseDialogForm(null)
        }}
      >
        <DialogContent className="gap-0 p-0 max-w-md">
          <DialogHeader className="border-b border-border px-5 py-4">
            <div className="flex items-center gap-2">
              <Lock className="h-5 w-5 text-warning" />
              <DialogTitle className="text-base font-semibold text-foreground">
                {t('formulare.closeDialog.title')}
              </DialogTitle>
            </div>
            <DialogDescription className="pt-1 text-sm text-muted-foreground">
              {t('formulare.closeDialog.description', {
                name: closeDialogForm?.title ?? '',
              })}
            </DialogDescription>
          </DialogHeader>

          <div className="p-5 space-y-2">
            <label className="text-sm font-medium text-foreground">
              {t('formulare.closeDialog.messageLabel')}
            </label>
            <textarea
              value={closeMessageInput}
              onChange={(e) => setCloseMessageInput(e.target.value)}
              rows={3}
              placeholder={t('formulare.closeDialog.messagePlaceholder')}
              className="w-full resize-none rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
            <p className="text-xs text-muted-foreground">
              {t('formulare.closeDialog.messageHint')}
            </p>
          </div>

          <DialogFooter className="border-t border-border px-5 py-4">
            <button
              onClick={() => setCloseDialogForm(null)}
              className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={confirmCloseForm}
              className="flex items-center gap-2 rounded-lg bg-warning px-4 py-2 text-sm font-medium text-warning-foreground transition-colors hover:opacity-90"
            >
              <Lock className="h-4 w-4" />
              {t('formulare.closeDialog.confirm')}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ====================== CONFIRM DELETE ====================== */}
      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title={t('formulare.delete.title')}
        description={t('formulare.delete.description', {
          name: confirmDelete?.title ?? '',
        })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={() => confirmDelete && handleDeleteForm(confirmDelete)}
      />

      {/* ====================== FT-5 — CONFIRM DELETE TEMPLATE ====================== */}
      <ConfirmDialog
        open={!!templateToDelete}
        onOpenChange={() => setTemplateToDelete(null)}
        title={t('formulare.vorlagen.deleteTitle')}
        description={t('formulare.vorlagen.deleteDescription', {
          name: templateToDelete?.title ?? '',
        })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={() => templateToDelete && handleDeleteTemplate(templateToDelete)}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// PreviewFillForm — interactive fill-out preview (FT-2a). Holds its own answer
// + error state, applies the per-field validation rules on submit and shows a
// thank-you state. Mirrors the future public page (white surface, blue accents).
// Extended in FT-2b with rating + a configurable thank-you message.
// ---------------------------------------------------------------------------

interface PreviewFillFormProps {
  fields: FormField[]
  title: string
  description?: string
  /** FT-2b — custom thank-you message + optional redirect shown after submit. */
  thankYouMessage?: string
  redirectUrl?: string
  /** FO-4 — config used to show the (mocked) dispatch summary after submit. */
  notifications?: { recipients: string[]; confirmToSubmitter: boolean }
  /** P2 — intake target id this form feeds (undefined = plain form). */
  intakeTargetId?: string
  /** P2 — already-translated target name for the success message. */
  intakeTargetLabel?: string
  /**
   * P2 — dispatch the answers to the bound intake target (create the record).
   * Called on a successful submit when `intakeTargetId` is set; resolves with the
   * created record's id.
   */
  onIntakeSubmit?: (answers: Record<string, unknown>) => Promise<{ id: string }>
  t: (key: string, opts?: Record<string, unknown>) => string
}

function PreviewFillForm({
  fields,
  title,
  description,
  thankYouMessage,
  redirectUrl,
  notifications,
  intakeTargetId,
  intakeTargetLabel,
  onIntakeSubmit,
  t,
}: PreviewFillFormProps) {
  const dataFields = useMemo(
    () => fields.filter((f) => f.label !== '__page_break__'),
    [fields],
  )
  // FT-2b — group fields into pages so each page break's title renders as <h3>.
  const pages = useMemo(() => {
    const out: { title?: string; fields: FormField[] }[] = [{ fields: [] }]
    for (const f of fields) {
      if (f.label === '__page_break__') {
        out.push({ title: f.pageTitle?.trim() || undefined, fields: [] })
      } else {
        out[out.length - 1].fields.push(f)
      }
    }
    return out.filter((p) => p.fields.length > 0 || p.title)
  }, [fields])

  const [values, setValues] = useState<Record<string, unknown>>({})
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [submitted, setSubmitted] = useState(false)
  // P2 — intake dispatch state (create the bound record on submit).
  const [creating, setCreating] = useState(false)
  const [createdId, setCreatedId] = useState<string | null>(null)
  const [submitError, setSubmitError] = useState<string | null>(null)

  const setVal = (id: string, v: unknown) => {
    setValues((prev) => ({ ...prev, [id]: v }))
    setErrors((prev) => {
      if (!prev[id]) return prev
      const next = { ...prev }
      delete next[id]
      return next
    })
  }

  const handleSubmit = async () => {
    const next: Record<string, string> = {}
    for (const field of dataFields) {
      // FO-6 — skip validation for fields hidden by their conditional rule.
      if (!evaluateCondition(field.conditionalLogic, values)) continue
      const raw = values[field.id]
      const str = typeof raw === 'string' ? raw : raw == null ? '' : String(raw)
      const missing =
        field.type === 'consent' || field.type === 'checkbox'
          ? raw !== true
          : field.type === 'rating'
            ? !(Number(raw) > 0)
            : str.trim() === ''
      if (field.required && missing) {
        next[field.id] = t('formulare.validation.error.required')
        continue
      }
      const err = validateFieldValue(field, raw)
      if (err) next[field.id] = t(err.key, err.params)
    }
    setErrors(next)
    if (Object.keys(next).length > 0) return

    // P2 — when the form feeds an intake target, create the record now.
    if (intakeTargetId && onIntakeSubmit) {
      setSubmitError(null)
      setCreating(true)
      try {
        const created = await onIntakeSubmit(values)
        setCreatedId(created.id)
        setSubmitted(true)
      } catch (err) {
        setSubmitError(err instanceof Error ? err.message : String(err))
      } finally {
        setCreating(false)
      }
      return
    }
    setSubmitted(true)
  }

  const reset = () => {
    setValues({})
    setErrors({})
    setSubmitted(false)
    setCreatedId(null)
    setSubmitError(null)
  }

  if (submitted) {
    // FO-4 — mocked dispatch summary derived from the form's notification config
    // and the email answer just entered (no real send in the demo preview).
    const emailField = dataFields.find((f) => f.type === 'email')
    const submitterEmail = emailField
      ? String(values[emailField.id] ?? '').trim()
      : ''
    const notifyRecipients = notifications?.recipients ?? []
    const confirmationTo =
      notifications?.confirmToSubmitter && submitterEmail ? submitterEmail : null
    const hasDispatch = notifyRecipients.length > 0 || !!confirmationTo
    return (
      <div className="p-8">
        <div className="flex flex-col items-center gap-3 rounded-xl border border-green-200 bg-green-50 px-6 py-10 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-green-100">
            <Check className="h-6 w-6 text-green-600" />
          </div>
          <p className="text-base font-semibold text-gray-900">
            {t('formulare.preview.thankYouTitle')}
          </p>
          <p className="max-w-sm whitespace-pre-line text-sm leading-relaxed text-gray-600">
            {thankYouMessage?.trim() || t('formulare.preview.thankYouDefault')}
          </p>
          {redirectUrl?.trim() && (
            <p className="inline-flex items-center gap-1.5 text-xs text-gray-500">
              <ExternalLink className="h-3.5 w-3.5" />
              {t('formulare.preview.redirectHint', { url: redirectUrl.trim() })}
            </p>
          )}
          {createdId && intakeTargetLabel && (
            <div className="mt-2 w-full max-w-sm rounded-lg border border-green-200 bg-white px-4 py-3 text-left">
              <div className="flex items-center gap-1.5 text-xs font-medium text-gray-700">
                <Zap className="h-3.5 w-3.5 text-green-600" />
                {t('formulare.preview.intakeCreated', { target: intakeTargetLabel })}
              </div>
              <p className="mt-1 text-xs text-gray-600">
                {t('formulare.preview.intakeCreatedRef', { id: createdId })}
              </p>
            </div>
          )}
          {hasDispatch && (
            <div className="mt-2 w-full max-w-sm rounded-lg border border-green-200 bg-white px-4 py-3 text-left">
              <div className="flex items-center gap-1.5 text-xs font-medium text-gray-700">
                <Mail className="h-3.5 w-3.5 text-green-600" />
                {t('formulare.preview.notifyTitle')}
              </div>
              <ul className="mt-1.5 space-y-1 text-xs text-gray-600">
                {notifyRecipients.length > 0 && (
                  <li>
                    {t('formulare.preview.notifyRecipients', {
                      list: notifyRecipients.join(', '),
                    })}
                  </li>
                )}
                {confirmationTo && (
                  <li>
                    {t('formulare.preview.notifyConfirmation', { email: confirmationTo })}
                  </li>
                )}
              </ul>
            </div>
          )}
          <button
            onClick={reset}
            className="mt-1 text-xs font-medium text-blue-600 hover:underline"
          >
            {t('formulare.preview.fillAgain')}
          </button>
        </div>
      </div>
    )
  }

  const errorCls = (id: string) =>
    errors[id]
      ? 'border-red-400 focus:ring-red-400'
      : 'border-gray-300 focus:ring-blue-500'

  const renderField = (field: FormField) => {
    // FO-6 — hide fields whose conditional rule isn't met by the current answers.
    if (!evaluateCondition(field.conditionalLogic, values)) return null
    const val = values[field.id]
    return (
      <div key={field.id} className="space-y-1.5">
        <label className="text-sm font-medium text-gray-700">
          {field.label}
          {field.required && <span className="ml-0.5 text-destructive">*</span>}
        </label>
        {(field.type === 'text' ||
          field.type === 'number' ||
          field.type === 'email') && (
          <input
            type={field.type === 'email' ? 'email' : field.type}
            value={typeof val === 'string' ? val : ''}
            onChange={(e) => setVal(field.id, e.target.value)}
            placeholder={
              field.placeholder ||
              (field.type === 'email' ? 'name@beispiel.de' : field.label)
            }
            className={`w-full rounded-lg border bg-white px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 ${errorCls(field.id)}`}
          />
        )}
        {field.type === 'textarea' && (
          <textarea
            rows={3}
            value={typeof val === 'string' ? val : ''}
            onChange={(e) => setVal(field.id, e.target.value)}
            placeholder={field.placeholder || field.label}
            className={`w-full resize-none rounded-lg border bg-white px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 ${errorCls(field.id)}`}
          />
        )}
        {field.type === 'select' && (
          <select
            value={typeof val === 'string' ? val : ''}
            onChange={(e) => setVal(field.id, e.target.value)}
            className={`w-full rounded-lg border bg-white px-3 py-2 text-sm text-gray-900 focus:outline-none focus:ring-2 ${errorCls(field.id)}`}
          >
            <option value="">{t('formulare.editor.selectPlaceholder')}</option>
            {field.options?.map((opt) => (
              <option key={opt}>{opt}</option>
            ))}
          </select>
        )}
        {field.type === 'radio' && (
          <div className="space-y-1.5">
            {field.options?.map((opt) => (
              <label
                key={opt}
                className="flex items-center gap-2 text-sm text-gray-700"
              >
                <input
                  type="radio"
                  name={field.id}
                  checked={val === opt}
                  onChange={() => setVal(field.id, opt)}
                  className="h-4 w-4 text-blue-600"
                />
                {opt}
              </label>
            ))}
          </div>
        )}
        {field.type === 'checkbox' && (
          <label className="flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={val === true}
              onChange={(e) => setVal(field.id, e.target.checked)}
              className="h-4 w-4 rounded text-blue-600"
            />
            {field.label}
          </label>
        )}
        {field.type === 'rating' && (
          <RatingInput
            scale={field.ratingScale ?? 5}
            value={Number(val) || 0}
            onChange={(v) => setVal(field.id, v)}
            light
          />
        )}
        {field.type === 'date' && (
          <input
            type="date"
            value={typeof val === 'string' ? val : ''}
            onChange={(e) => setVal(field.id, e.target.value)}
            className={`w-full rounded-lg border bg-white px-3 py-2 text-sm text-gray-900 focus:outline-none focus:ring-2 ${errorCls(field.id)}`}
          />
        )}
        {field.type === 'file' && (
          <div className="flex items-center gap-2 rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 px-4 py-6 text-sm text-gray-500">
            <Paperclip className="h-4 w-4" />
            {t('formulare.editor.dateiZiehen')}
          </div>
        )}
        {field.type === 'consent' && (
          <label className="flex items-start gap-2.5 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={val === true}
              onChange={(e) => setVal(field.id, e.target.checked)}
              className="mt-0.5 h-4 w-4 rounded text-blue-600"
            />
            <span className="text-xs leading-relaxed text-gray-600">
              {field.consentText || DEFAULT_CONSENT_TEXT}
              {field.privacyUrl && (
                <a
                  href={field.privacyUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="ml-1 text-blue-600 underline"
                >
                  {t('formulare.consent.privacyLink')}
                </a>
              )}
            </span>
          </label>
        )}
        {errors[field.id] && (
          <p className="text-xs text-red-600">{errors[field.id]}</p>
        )}
      </div>
    )
  }

  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-gray-900">{title}</h1>
        {description && <p className="mt-1 text-sm text-gray-500">{description}</p>}
      </div>

      <div className="space-y-6">
        {pages.map((page, i) => (
          <div key={i} className="space-y-5">
            {page.title && (
              <h3 className="border-b border-gray-100 pb-1.5 text-base font-semibold text-gray-900">
                {page.title}
              </h3>
            )}
            {page.fields.map(renderField)}
          </div>
        ))}
      </div>

      {submitError && (
        <div className="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {t('formulare.preview.intakeError', { message: submitError })}
        </div>
      )}
      {dataFields.length > 0 && (
        <button
          onClick={handleSubmit}
          disabled={creating}
          className="mt-6 rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-blue-700 transition-colors disabled:cursor-not-allowed disabled:opacity-60"
        >
          {creating ? t('formulare.preview.intakeCreating') : t('formulare.editor.absenden')}
        </button>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// ExportMenu — a small CSV/XLSX export dropdown bound to one schema. Reused by
// the Eingänge group headers and the form detail modal. Triggers a real blob
// download via the parent's onExport handler.
// ---------------------------------------------------------------------------

interface ExportMenuProps {
  schema: FormSchema
  onExport: (schema: FormSchema, format: 'csv' | 'xlsx') => void
  t: (key: string, opts?: Record<string, unknown>) => string
  compact?: boolean
  /** Personal default — listed first and marked as the default option. */
  defaultFormat?: 'csv' | 'xlsx'
}

function ExportMenu({ schema, onExport, t, compact, defaultFormat = 'csv' }: ExportMenuProps) {
  const [open, setOpen] = useState(false)
  const formats: { value: 'csv' | 'xlsx'; labelKey: string }[] = [
    { value: 'csv', labelKey: 'formulare.export.csv' },
    { value: 'xlsx', labelKey: 'formulare.export.excel' },
  ]
  formats.sort((a, b) =>
    a.value === defaultFormat ? -1 : b.value === defaultFormat ? 1 : 0,
  )
  return (
    <div className="relative" onClick={(e) => e.stopPropagation()}>
      <button
        onClick={() => setOpen((o) => !o)}
        className={
          compact
            ? 'flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-secondary'
            : 'flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary'
        }
      >
        <Download className={compact ? 'h-3.5 w-3.5' : 'h-4 w-4'} />
        {t('formulare.export.exportieren')}
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-full z-20 mt-1 w-48 rounded-lg border border-border bg-card py-1 shadow-xl">
            {formats.map((f) => (
              <button
                key={f.value}
                onClick={() => {
                  onExport(schema, f.value)
                  setOpen(false)
                }}
                className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-foreground transition-colors hover:bg-secondary"
              >
                <Download className="h-4 w-4 text-muted-foreground" />
                <span className="flex-1 text-left">{t(f.labelKey)}</span>
                {f.value === defaultFormat && (
                  <span className="text-[10px] text-muted-foreground">
                    {t('formulare.export.standard')}
                  </span>
                )}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// SortableFieldItem — one draggable row of the field builder (dnd-kit). The
// drag listeners live on a dedicated grip handle; inline controls carry a
// data-field-control guard + onPointerDown stop so editing never starts a drag.
// ---------------------------------------------------------------------------

interface SortableFieldItemProps {
  field: FormField
  icon?: typeof Type
  typeLabel?: string
  /** When set, renders the page-break divider variant instead of a field row. */
  pageBreakLabel?: string
  /** FT-2b — page-break only: edit the title of the page that follows. */
  onPageTitleChange?: (value: string) => void
  t: (key: string, opts?: Record<string, unknown>) => string
  onEdit?: () => void
  onDuplicate?: () => void
  onRemove: () => void
}

function SortableFieldItem({
  field,
  icon: Icon,
  typeLabel,
  pageBreakLabel,
  onPageTitleChange,
  t,
  onEdit,
  onDuplicate,
  onRemove,
}: SortableFieldItemProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: field.id })
  const style = { transform: CSS.Transform.toString(transform), transition }
  const stop = (e: SyntheticEvent) => e.stopPropagation()

  if (pageBreakLabel) {
    return (
      <div
        ref={setNodeRef}
        style={style}
        className={`space-y-1.5 py-2 group ${isDragging ? 'opacity-50' : ''}`}
      >
        <div className="flex items-center gap-2">
          <button
            type="button"
            {...attributes}
            {...listeners}
            className="shrink-0 cursor-grab text-primary/40 transition-colors hover:text-primary/70 active:cursor-grabbing"
            aria-label={t('formulare.editor.feldVerschieben')}
          >
            <GripVertical className="h-4 w-4" />
          </button>
          <div className="flex-1 border-t border-dashed border-primary/40" />
          <span className="text-xs font-medium text-primary/70 whitespace-nowrap">
            {pageBreakLabel}
          </span>
          <div className="flex-1 border-t border-dashed border-primary/40" />
          <button
            data-field-control
            onPointerDown={stop}
            onClick={onRemove}
            className="rounded-md p-1 text-muted-foreground transition-colors hover:bg-error-light hover:text-error opacity-0 group-hover:opacity-100"
            title={t('formulare.editor.seitenumbruch')}
          >
            <Trash2 className="h-3 w-3" />
          </button>
        </div>
        {/* FT-2b — optional title for the page that starts after this break */}
        {onPageTitleChange && (
          <input
            data-field-control
            onPointerDown={stop}
            value={field.pageTitle ?? ''}
            onChange={(e) => onPageTitleChange(e.target.value)}
            placeholder={t('formulare.editor.seitentitelPlaceholder')}
            className="ml-6 w-[calc(100%-1.5rem)] rounded-md border border-border bg-card px-2.5 py-1 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
        )}
      </div>
    )
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`flex items-center gap-2 rounded-lg border border-border bg-card p-3 group transition-shadow hover:shadow-[var(--shadow-card-hover)] ${
        isDragging ? 'opacity-50 shadow-lg' : ''
      }`}
    >
      <button
        type="button"
        {...attributes}
        {...listeners}
        className="shrink-0 cursor-grab text-muted-foreground/40 transition-colors hover:text-muted-foreground active:cursor-grabbing"
        aria-label={t('formulare.editor.feldVerschieben')}
      >
        <GripVertical className="h-4 w-4" />
      </button>
      {Icon && (
        <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary-light shrink-0">
          <Icon className="h-4 w-4 text-primary" />
        </div>
      )}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-foreground truncate">
            {field.label}
          </span>
          {field.required && (
            <span className="rounded bg-error-light px-1.5 py-0.5 text-[9px] font-medium text-error shrink-0">
              {t('formulare.editor.pflicht')}
            </span>
          )}
          {field.conditionalLogic && (
            <span className="rounded bg-warning-light px-1.5 py-0.5 text-[9px] font-medium text-warning shrink-0">
              {t('formulare.editor.bedingt')}
            </span>
          )}
        </div>
        <span className="text-[10px] text-muted-foreground">
          {typeLabel}
          {field.options
            ? ` (${field.options.length} ${t('formulare.fieldConfig.optionen')})`
            : ''}
        </span>
      </div>
      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
        {onEdit && (
          <button
            data-field-control
            onPointerDown={stop}
            onClick={onEdit}
            className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-secondary"
            title={t('formulare.actions.bearbeiten')}
          >
            <Edit className="h-3.5 w-3.5" />
          </button>
        )}
        {onDuplicate && (
          <button
            data-field-control
            onPointerDown={stop}
            onClick={onDuplicate}
            className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-secondary"
            title={t('formulare.editor.feldDuplizieren')}
          >
            <Copy className="h-3.5 w-3.5" />
          </button>
        )}
        <button
          data-field-control
          onPointerDown={stop}
          onClick={onRemove}
          className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-error-light hover:text-error"
          title={t('common.delete')}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// SubmissionsPanel — lazy-loads submissions for a single schema
// ---------------------------------------------------------------------------

interface SubmissionsPanelProps {
  schemaId: string
  onSelectSubmission: (sub: FormSubmission) => void
  onUpdateStatus: (sub: FormSubmission, status: FormSubmissionStatus) => void
  submissionStatusLabels: Record<FormSubmissionStatus, string>
  submissionStatusColors: Record<FormSubmissionStatus, string>
  formatDateTime: (d: string) => string
  t: (key: string, opts?: Record<string, unknown>) => string
  /** Gate: show write actions (mark read / archive) only when true. */
  canWrite?: boolean
}

function SubmissionsPanel({
  schemaId,
  onSelectSubmission,
  onUpdateStatus,
  submissionStatusLabels,
  submissionStatusColors,
  formatDateTime,
  t,
  canWrite = false,
}: SubmissionsPanelProps) {
  // Hits the warm cache populated by the page-level eager useQueries fetch.
  const { data, isLoading } = useSubmissions(schemaId)
  const items = useMemo(() => data?.items ?? [], [data])

  // FT-1 — per-group status quick-filter + sort (field + direction).
  const [statusFilter, setStatusFilter] = useState<'all' | FormSubmissionStatus>('all')
  const [sortField, setSortField] = useState<'submittedAt' | 'submittedBy'>('submittedAt')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')

  const counts = useMemo(
    () => ({
      all: items.length,
      new: items.filter((s) => s.status === 'new').length,
      read: items.filter((s) => s.status === 'read').length,
      archived: items.filter((s) => s.status === 'archived').length,
    }),
    [items],
  )

  const visible = useMemo(() => {
    const filtered =
      statusFilter === 'all'
        ? items
        : items.filter((s) => s.status === statusFilter)
    const dir = sortDir === 'asc' ? 1 : -1
    return [...filtered].sort((a, b) => {
      if (sortField === 'submittedBy') {
        return (
          dir * (a.submittedBy ?? '').localeCompare(b.submittedBy ?? '', 'de')
        )
      }
      return dir * a.submittedAt.localeCompare(b.submittedAt)
    })
  }, [items, statusFilter, sortField, sortDir])

  if (isLoading) {
    return (
      <div className="border-t border-border p-4">
        <div className="animate-pulse space-y-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-8 rounded bg-secondary" />
          ))}
        </div>
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className="border-t border-border px-4 py-6 text-center text-sm text-muted-foreground">
        {t('formulare.empty.noEingaenge')}
      </div>
    )
  }

  const filterChips: { value: 'all' | FormSubmissionStatus; label: string; count: number }[] = [
    { value: 'all', label: t('formulare.submission.filter.all'), count: counts.all },
    { value: 'new', label: submissionStatusLabels.new, count: counts.new },
    { value: 'read', label: submissionStatusLabels.read, count: counts.read },
    { value: 'archived', label: submissionStatusLabels.archived, count: counts.archived },
  ]

  return (
    <div className="border-t border-border">
      {/* Toolbar: status quick-filter + sort */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border-muted bg-secondary/20 px-4 py-2.5">
        <div className="flex flex-wrap items-center gap-1.5">
          {filterChips.map((chip) => {
            const active = statusFilter === chip.value
            return (
              <button
                key={chip.value}
                onClick={() => setStatusFilter(chip.value)}
                className={`flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs transition-colors ${
                  active
                    ? 'bg-primary-light text-primary font-medium'
                    : 'text-muted-foreground hover:bg-secondary'
                }`}
              >
                {chip.label}
                <span
                  className={`rounded-full px-1.5 text-[10px] ${
                    active ? 'bg-primary/15' : 'bg-secondary'
                  }`}
                >
                  {chip.count}
                </span>
              </button>
            )
          })}
        </div>
        <SortMenu
          options={[
            { value: 'submittedAt', label: t('formulare.submission.table.datum') },
            { value: 'submittedBy', label: t('formulare.submission.table.absender') },
          ]}
          field={sortField}
          direction={sortDir}
          onChange={(f, d) => {
            setSortField(f as 'submittedAt' | 'submittedBy')
            setSortDir(d)
          }}
        />
      </div>

      {visible.length === 0 ? (
        <div className="px-4 py-6 text-center text-sm text-muted-foreground">
          {t('formulare.submission.noFilterMatch')}
        </div>
      ) : (
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border-muted bg-secondary/30">
            <th className="px-4 py-2 text-left font-medium text-muted-foreground">
              {t('formulare.submission.table.datum')}
            </th>
            <th className="px-4 py-2 text-left font-medium text-muted-foreground">
              {t('formulare.submission.table.absender')}
            </th>
            <th className="px-4 py-2 text-left font-medium text-muted-foreground">
              {t('formulare.submission.table.status')}
            </th>
            <th className="px-4 py-2 text-right font-medium text-muted-foreground" />
          </tr>
        </thead>
        <tbody>
          {visible.map((sub) => (
            <tr
              key={sub.id}
              role="button"
              tabIndex={0}
              onClick={() => onSelectSubmission(sub)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  onSelectSubmission(sub)
                }
              }}
              className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer focus:outline-none focus-visible:bg-secondary/50"
            >
              <td className="px-4 py-2.5 text-foreground">
                {formatDateTime(sub.submittedAt)}
              </td>
              <td className="px-4 py-2.5 text-foreground">
                {sub.submittedBy ?? '--'}
              </td>
              <td className="px-4 py-2.5">
                <span
                  className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                    submissionStatusColors[sub.status]
                  }`}
                >
                  {submissionStatusLabels[sub.status]}
                </span>
              </td>
              <td
                className="px-4 py-2.5 text-right"
                onClick={(e) => e.stopPropagation()}
              >
                <ItemActions
                  items={[
                    {
                      label: t('formulare.submission.detailsAnzeigen'),
                      icon: Eye,
                      onClick: () => onSelectSubmission(sub),
                    },
                    ...(canWrite
                      ? [
                          {
                            label: t('formulare.submission.alsGelesenMarkieren'),
                            icon: Eye,
                            onClick: () => onUpdateStatus(sub, 'read'),
                            disabled: sub.status !== 'new',
                          },
                          {
                            label: t('formulare.submission.archivieren'),
                            icon: Archive,
                            onClick: () => onUpdateStatus(sub, 'archived'),
                            disabled: sub.status === 'archived',
                          },
                        ]
                      : []),
                  ]}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      )}
    </div>
  )
}

// Import useSubmissions here to keep the sub-component working
import { useSubmissions } from '@/api/hooks/useFormulare'

// ---------------------------------------------------------------------------
// ShareLinksPanel — FD-2 distribution overview: every share link of one form
// with channel, validity, views, responses (+conversion) and per-link actions.
// ---------------------------------------------------------------------------

const SHARE_CHANNEL_META: Record<ShareChannel, { icon: typeof Type; labelKey: string }> = {
  link: { icon: Link2, labelKey: 'formulare.share.channel.link' },
  email: { icon: Mail, labelKey: 'formulare.share.channel.email' },
  embed: { icon: Code2, labelKey: 'formulare.share.channel.embed' },
  qr: { icon: QrCode, labelKey: 'formulare.share.channel.qr' },
}

const SHARE_STATUS_META: Record<ShareLinkStatus, { cls: string; labelKey: string }> = {
  active: { cls: 'bg-success-light text-success', labelKey: 'formulare.share.status.active' },
  expired: { cls: 'bg-secondary text-muted-foreground', labelKey: 'formulare.share.status.expired' },
  disabled: { cls: 'bg-warning-light text-warning', labelKey: 'formulare.share.status.disabled' },
}

interface ShareLinksPanelProps {
  schemaId: string
  canShare: boolean
  onShare: () => void
  onCopy: (link: FormShareLink) => void
  onToggle: (link: FormShareLink) => void
  onDelete: (link: FormShareLink) => void
  formatDate: (d: string) => string
  t: (key: string, opts?: Record<string, unknown>) => string
}

function ShareLinksPanel({
  schemaId,
  canShare,
  onShare,
  onCopy,
  onToggle,
  onDelete,
  formatDate,
  t,
}: ShareLinksPanelProps) {
  const { data, isLoading } = useShareLinks(schemaId)
  const links = data ?? []

  if (isLoading) {
    return (
      <div className="space-y-2">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-12 animate-pulse rounded-lg bg-secondary" />
        ))}
      </div>
    )
  }

  if (links.length === 0) {
    return (
      <EmptyState
        icon={Share2}
        title={t('formulare.share.empty.title')}
        description={
          canShare
            ? t('formulare.share.empty.hint')
            : t('formulare.share.empty.hintInactive')
        }
        action={
          canShare
            ? { label: t('formulare.actions.teilen'), onClick: onShare }
            : undefined
        }
      />
    )
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border-muted bg-secondary/30 text-xs text-muted-foreground">
            <th className="px-4 py-2.5 text-left font-medium">
              {t('formulare.share.channelLabel')}
            </th>
            <th className="hidden px-4 py-2.5 text-left font-medium sm:table-cell">
              {t('formulare.detail.createdAt')}
            </th>
            <th className="px-4 py-2.5 text-left font-medium">
              {t('formulare.share.col.expires')}
            </th>
            <th className="px-4 py-2.5 text-right font-medium">
              {t('formulare.share.col.views')}
            </th>
            <th className="px-4 py-2.5 text-right font-medium">
              {t('formulare.submission.antworten')}
            </th>
            <th className="px-4 py-2.5 text-left font-medium">
              {t('formulare.submission.table.status')}
            </th>
            <th className="px-4 py-2.5 text-right font-medium" />
          </tr>
        </thead>
        <tbody>
          {links.map((link) => {
            const ch = SHARE_CHANNEL_META[link.channel]
            const st = SHARE_STATUS_META[link.status]
            const ChIcon = ch.icon
            const conv =
              link.views > 0 ? Math.round((link.submissions / link.views) * 100) : 0
            return (
              <tr
                key={link.id}
                className="border-b border-border-muted last:border-0"
              >
                <td className="px-4 py-2.5">
                  <span className="inline-flex items-center gap-1.5 text-foreground">
                    <ChIcon className="h-3.5 w-3.5 text-muted-foreground" />
                    {t(ch.labelKey)}
                  </span>
                </td>
                <td className="hidden px-4 py-2.5 text-muted-foreground sm:table-cell">
                  {formatDate(link.createdAt)}
                </td>
                <td className="px-4 py-2.5 text-muted-foreground">
                  {link.expiresAt
                    ? formatDate(link.expiresAt)
                    : t('formulare.share.noExpiry')}
                </td>
                <td className="px-4 py-2.5 text-right text-muted-foreground">
                  {link.views}
                </td>
                <td className="px-4 py-2.5 text-right text-muted-foreground">
                  {link.submissions}
                  <span className="ml-1 text-[10px] text-muted-foreground/70">
                    ({conv}%)
                  </span>
                </td>
                <td className="px-4 py-2.5">
                  <span
                    className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${st.cls}`}
                  >
                    {t(st.labelKey)}
                  </span>
                </td>
                <td className="px-4 py-2.5 text-right">
                  <ItemActions
                    items={[
                      {
                        label: t('formulare.share.linkKopieren'),
                        icon: Copy,
                        onClick: () => onCopy(link),
                      },
                      {
                        label:
                          link.status === 'disabled'
                            ? t('formulare.share.reactivate')
                            : t('formulare.share.deactivate'),
                        icon: link.status === 'disabled' ? Check : Ban,
                        onClick: () => onToggle(link),
                        disabled: link.status === 'expired',
                      },
                      {
                        separator: true,
                        label: t('common.delete'),
                        icon: Trash2,
                        variant: 'destructive',
                        onClick: () => onDelete(link),
                      },
                    ]}
                  />
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

// ---------------------------------------------------------------------------
// EvaluationPanel — KPI row from the stats endpoint. Reduced to the 4 counters
// the backend actually tracks (total / new / last 7 days / last 30 days); the
// richer per-field/conversion/drop-off/time-series analytics previously shown
// here had no backing data (real view-tracking doesn't exist yet) and were
// pure frontend fiction — tracked as a follow-up product ticket.
// ---------------------------------------------------------------------------

interface EvaluationPanelProps {
  schema: FormSchema
  formatDate: (d: string) => string
  t: (key: string, opts?: Record<string, unknown>) => string
}

function EvaluationPanel({ schema, t }: EvaluationPanelProps) {
  const { data: stats, isLoading } = useFormStats(schema.id)

  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-16 animate-pulse rounded-lg bg-secondary" />
        ))}
      </div>
    )
  }

  if (!stats) return null

  if (stats.totalCount === 0) {
    return (
      <EmptyState
        icon={BarChart3}
        title={t('formulare.eval.emptyTitle')}
        description={t('formulare.eval.emptyHint')}
      />
    )
  }

  const kpis = [
    { label: t('formulare.eval.kpiTotal'), value: String(stats.totalCount) },
    { label: t('formulare.eval.kpiNew'), value: String(stats.newCount) },
    { label: t('formulare.eval.kpiWeek'), value: String(stats.last7dCount) },
    { label: t('formulare.eval.kpiMonth'), value: String(stats.last30dCount) },
  ]

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
      {kpis.map((kpi) => (
        <div key={kpi.label} className="rounded-lg border border-border bg-secondary/30 p-3">
          <p className="text-xl font-semibold text-foreground">{kpi.value}</p>
          <p className="text-[10px] uppercase tracking-wider text-muted-foreground">
            {kpi.label}
          </p>
        </div>
      ))}
    </div>
  )
}
