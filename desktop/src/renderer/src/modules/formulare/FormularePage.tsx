import { useState, useMemo, useCallback } from 'react'
import { useQueries } from '@tanstack/react-query'
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
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useFormulareStore,
  type DraftSchema,
  type FormField,
  type FormFieldType,
} from '@/stores/formulare'
import {
  useFormSchemas,
  useCreateFormSchema,
  useUpdateFormSchema,
  useDeleteFormSchema,
  useDuplicateFormSchema,
  useExportSubmissions,
  useUpdateSubmissionStatus,
  formulareKeys,
} from '@/api/hooks/useFormulare'
import { listSubmissions } from '@/api/formulare-client'
import type { FormSchema, FormSubmission, FormSubmissionStatus } from '@/api/formulare-types'
import { ItemActions, ConfirmDialog, EmptyState, DetailModal, PageHeader } from '@/components/shared'
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
  archived: 'formulare.status.archived',
}

const formStatusColors: Record<FormSchema['status'], string> = {
  active: 'bg-success-light text-success',
  draft: 'bg-secondary text-muted-foreground',
  archived: 'bg-warning-light text-warning',
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
]

// ---------------------------------------------------------------------------
// Helper: map FormSchema → DraftSchema for the editor
// ---------------------------------------------------------------------------

function schemaToDraft(schema: FormSchema): DraftSchema {
  return {
    id: schema.id,
    title: schema.title,
    description: schema.description,
    // Cast fields — backend fields are compatible at runtime
    fields: schema.fields as FormField[],
    isPublic: schema.isPublic,
    pageCount: schema.pageCount,
    actions: [],
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function FormularePage() {
  const { t } = useTranslation()

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

  // Mutations
  const createSchema = useCreateFormSchema()
  const updateSchema = useUpdateFormSchema()
  const deleteSchema = useDeleteFormSchema()
  const duplicateSchema = useDuplicateFormSchema()
  const _exportSubs = useExportSubmissions()
  const updateSubStatus = useUpdateSubmissionStatus()

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
    reorderFields: _reorderFields,
    updateField,
  } = useFormulareStore()

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

  // Tab & search
  const [tab, setTab] = useState<TabKey>('formulare')
  const [search, setSearch] = useState('')

  // Detail modal for a single submission
  const [selectedSubmission, setSelectedSubmission] =
    useState<FormSubmission | null>(null)

  // Detail modal for a single form (360° view + actions)
  const [selectedForm, setSelectedForm] = useState<FormSchema | null>(null)

  // Standalone public-facing form preview (decoupled from the editor draft)
  const [previewSchema, setPreviewSchema] = useState<FormSchema | null>(null)

  // Expanded submission groups
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set())

  // Dialogs
  const [confirmDelete, setConfirmDelete] = useState<FormSchema | null>(null)
  const [showNewFormDialog, setShowNewFormDialog] = useState(false)
  const [showShareDialog, setShowShareDialog] = useState<FormSchema | null>(null)
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

  // 10.5 — Export dropdown state
  const [showExportMenu, setShowExportMenu] = useState(false)

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

  // ---------------------------------------------------------------------------
  // Derived data
  // ---------------------------------------------------------------------------

  const filteredForms = useMemo(() => {
    if (!search) return activeForms
    const q = search.toLowerCase()
    return activeForms.filter(
      (f) =>
        f.title.toLowerCase().includes(q) ||
        f.description.toLowerCase().includes(q),
    )
  }, [activeForms, search])

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
      })
      toast.success(t('formulare.toast.gespeichert'))
      closeDraft()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('common.error'))
    }
  }

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------

  const formatDate = (d: string) =>
    libFormatDate(d.includes('T') ? d : d + 'T00:00:00')

  const formatDateTime = (d: string) =>
    libFormatDateTime(d, { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })

  const getFormActions = useCallback(
    (schema: FormSchema) => [
      {
        label: t('formulare.actions.bearbeiten'),
        icon: Edit,
        onClick: () => openEditor(schema),
      },
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
      {
        label: t('formulare.actions.teilen'),
        icon: Share2,
        onClick: () => setShowShareDialog(schema),
      },
      {
        label:
          schema.status === 'archived'
            ? t('formulare.actions.aktivieren')
            : t('formulare.actions.archivieren'),
        icon: Archive,
        onClick: () => {
          const newStatus =
            schema.status === 'archived' ? 'active' : 'archived'
          updateSchema.mutate(
            { id: schema.id, status: newStatus },
            {
              onSuccess: () =>
                toast.success(
                  schema.status === 'archived'
                    ? t('formulare.toast.aktiviert', { name: schema.title })
                    : t('formulare.toast.archiviert', { name: schema.title }),
                ),
              onError: (err) =>
                toast.error(
                  err instanceof Error ? err.message : t('common.error'),
                ),
            },
          )
        },
      },
      { separator: true as const, label: '', onClick: () => {} },
      {
        label: t('common.delete'),
        icon: Trash2,
        variant: 'destructive' as const,
        onClick: () => setConfirmDelete(schema),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, duplicateSchema, updateSchema],
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

  const handleAddField = (type: FormFieldType) => {
    addField({
      type,
      label: FIELD_TYPE_LABELS[type],
      required: false,
      placeholder: '',
      options:
        type === 'select' || type === 'radio'
          ? ['Option 1', 'Option 2']
          : undefined,
    })
    setShowAddFieldMenu(false)
  }

  const handleRemoveField = (fieldId: string) => {
    removeField(fieldId)
  }

  const openFieldConfig = (field: FormField) => {
    setConfigLabel(field.label)
    setConfigRequired(field.required)
    setConfigPlaceholder(field.placeholder ?? '')
    setConfigOptions(field.options?.join(', ') ?? '')
    setConfigConditionalEnabled(!!field.conditionalLogic)
    setConfigConditionalFieldId(field.conditionalLogic?.fieldId ?? '')
    setConfigConditionalOperator(field.conditionalLogic?.operator ?? 'equals')
    setConfigConditionalValue(field.conditionalLogic?.value ?? '')
    setShowFieldConfigDialog({ field })
  }

  const saveFieldConfig = () => {
    if (!showFieldConfigDialog) return
    const { field } = showFieldConfigDialog
    updateField(field.id, {
      label: configLabel,
      required: configRequired,
      placeholder: configPlaceholder || undefined,
      options:
        field.type === 'select' || field.type === 'radio'
          ? configOptions
              .split(',')
              .map((o) => o.trim())
              .filter(Boolean)
          : field.options,
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

  const evaluateCondition = (
    logic: FormField['conditionalLogic'],
    answers: Record<string, unknown>,
  ): boolean => {
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

  // ---------------------------------------------------------------------------
  // JSX — Editor View
  // ---------------------------------------------------------------------------

  if (draft) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        {/* Editor Header */}
        <div className="flex items-center gap-3 mb-6">
          <button
            onClick={closeEditor}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            <ArrowLeft className="h-4 w-4" />
            {t('formulare.editor.zurueck')}
          </button>
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
              <div className="space-y-2">
                {draft.fields.map((field) => {
                  // 10.2 — Page break divider
                  if (field.label === '__page_break__') {
                    const pageNum =
                      fieldPageMap.get(field.id) === -1
                        ? Array.from(fieldPageMap.entries())
                            .filter(([, v]) => v === -1)
                            .findIndex(([k]) => k === field.id) + 1
                        : 1
                    return (
                      <div
                        key={field.id}
                        className="flex items-center gap-3 py-2 group"
                      >
                        <div className="flex-1 border-t border-dashed border-primary/40" />
                        <span className="text-xs font-medium text-primary/70 whitespace-nowrap">
                          {t('formulare.editor.seitenumbruchLabel', {
                            page: pageNum + 1,
                          })}
                        </span>
                        <div className="flex-1 border-t border-dashed border-primary/40" />
                        <button
                          onClick={() => handleRemoveField(field.id)}
                          className="rounded-md p-1 text-muted-foreground hover:text-error hover:bg-error-light transition-colors opacity-0 group-hover:opacity-100"
                          title={t('formulare.editor.seitenumbruch')}
                        >
                          <Trash2 className="h-3 w-3" />
                        </button>
                      </div>
                    )
                  }

                  const Icon = FIELD_TYPE_ICONS[field.type]
                  return (
                    <div
                      key={field.id}
                      className="flex items-center gap-2 rounded-lg border border-border bg-card p-3 group hover:shadow-[var(--shadow-card-hover)] transition-shadow"
                    >
                      <GripVertical className="h-4 w-4 text-muted-foreground/40 shrink-0 cursor-grab" />
                      <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary-light shrink-0">
                        <Icon className="h-4 w-4 text-primary" />
                      </div>
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
                          {/* 10.1 — Conditional badge */}
                          {field.conditionalLogic && (
                            <span className="rounded bg-warning-light px-1.5 py-0.5 text-[9px] font-medium text-warning shrink-0">
                              {t('formulare.editor.bedingt')}
                            </span>
                          )}
                        </div>
                        <span className="text-[10px] text-muted-foreground">
                          {FIELD_TYPE_LABELS[field.type]}
                          {field.options
                            ? ` (${field.options.length} ${t('formulare.fieldConfig.optionen')})`
                            : ''}
                        </span>
                      </div>
                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button
                          onClick={() => openFieldConfig(field)}
                          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
                          title={t('formulare.actions.bearbeiten')}
                        >
                          <Edit className="h-3.5 w-3.5" />
                        </button>
                        <button
                          onClick={() => handleRemoveField(field.id)}
                          className="rounded-md p-1.5 text-muted-foreground hover:text-error hover:bg-error-light transition-colors"
                          title={t('common.delete')}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </div>
                  )
                })}
              </div>
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

            {/* Form content */}
            <div className="p-8">
              <div className="mb-6">
                <h1 className="text-xl font-semibold text-gray-900">
                  {draft.title || t('formulare.preview.formularname')}
                </h1>
                {draft.description && (
                  <p className="text-sm text-gray-500 mt-1">
                    {draft.description}
                  </p>
                )}
              </div>

              <div className="space-y-5">
                {draft.fields
                  .filter((f) => f.label !== '__page_break__')
                  .map((field) => (
                    <div key={field.id} className="space-y-1.5">
                      <label className="text-sm font-medium text-gray-700">
                        {field.label}
                        {field.required && (
                          <span className="text-destructive ml-0.5">*</span>
                        )}
                      </label>
                      {(field.type === 'text' ||
                        field.type === 'number' ||
                        field.type === 'email') && (
                        <input
                          type={field.type === 'email' ? 'email' : field.type}
                          placeholder={
                            field.placeholder ||
                            (field.type === 'email' ? 'name@beispiel.de' : field.label)
                          }
                          className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                      )}
                      {field.type === 'textarea' && (
                        <textarea
                          rows={3}
                          placeholder={field.placeholder || field.label}
                          className="w-full resize-none rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                      )}
                      {field.type === 'select' && (
                        <select className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500">
                          <option value="">
                            {t('formulare.editor.selectPlaceholder')}
                          </option>
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
                            className="h-4 w-4 rounded text-blue-600"
                          />
                          {field.label}
                        </label>
                      )}
                      {field.type === 'date' && (
                        <input
                          type="date"
                          className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                      )}
                      {field.type === 'file' && (
                        <div className="flex items-center gap-2 rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 px-4 py-6 text-sm text-gray-500">
                          <Paperclip className="h-4 w-4" />
                          {t('formulare.editor.dateiZiehen')}
                        </div>
                      )}
                    </div>
                  ))}
              </div>

              {draft.fields.filter((f) => f.label !== '__page_break__').length >
                0 && (
                <button className="mt-6 rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-blue-700 transition-colors">
                  {t('formulare.editor.absenden')}
                </button>
              )}
            </div>
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
              <p className="text-2xl font-semibold text-foreground">87%</p>
              <p className="text-xs text-muted-foreground">
                {t('formulare.stats.completionRate')}
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
        ).map((tabItem) => (
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

          {/* 10.5 — Export button (Eingänge tab only) */}
          {tab === 'eingänge' && allSubmissions.length > 0 && (
            <div className="relative">
              <button
                onClick={() => setShowExportMenu(!showExportMenu)}
                className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <Download className="h-4 w-4" />
                {t('formulare.export.exportieren')}
              </button>
              {showExportMenu && (
                <div className="absolute right-0 top-full mt-1 z-20 w-48 rounded-lg border border-border bg-card shadow-xl py-1">
                  <button
                    onClick={() => {
                      // Export requires a specific schema context; show hint
                      toast.success(
                        t('formulare.export.csvToast', {
                          count: allSubmissions.length,
                        }),
                      )
                      setShowExportMenu(false)
                    }}
                    className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
                  >
                    <Download className="h-4 w-4 text-muted-foreground" />
                    {t('formulare.export.csv')}
                  </button>
                  <button
                    onClick={() => {
                      toast.success(
                        t('formulare.export.excelToast', {
                          count: allSubmissions.length,
                        }),
                      )
                      setShowExportMenu(false)
                    }}
                    className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
                  >
                    <Download className="h-4 w-4 text-muted-foreground" />
                    {t('formulare.export.excel')}
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* ====================== MEINE FORMULARE TAB ====================== */}
      {tab === 'formulare' && (
        <>
          {filteredForms.length === 0 ? (
            <EmptyState
              icon={FileText}
              title={t('formulare.empty.noFormulare')}
              description={
                search
                  ? t('formulare.empty.suchHint')
                  : t('formulare.empty.createHint')
              }
              action={
                !search
                  ? {
                      label: t('formulare.empty.createLabel'),
                      onClick: () => {
                        resetNewFormDialog()
                        setShowNewFormDialog(true)
                      },
                    }
                  : undefined
              }
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {filteredForms.map((schema) => (
                <div
                  key={schema.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => setSelectedForm(schema)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault()
                      setSelectedForm(schema)
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
                    <button
                      onClick={() => toggleGroup(schema.id)}
                      className="flex w-full items-center justify-between px-4 py-3 hover:bg-secondary/50 transition-colors"
                    >
                      <div className="flex items-center gap-3">
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
                      </div>
                    </button>

                    {/* Submissions table — loaded lazily */}
                    {isExpanded && (
                      <SubmissionsPanel
                        schemaId={schema.id}
                        onSelectSubmission={setSelectedSubmission}
                        onUpdateStatus={(sub, status) => {
                          updateSubStatus.mutate(
                            {
                              id: sub.id,
                              status,
                              formSchemaId: schema.id,
                            },
                            {
                              onSuccess: () =>
                                toast.success(
                                  status === 'read'
                                    ? t('formulare.toast.alsGelesenMarkiert')
                                    : t('formulare.toast.eingangArchiviert'),
                                ),
                              onError: (err) =>
                                toast.error(
                                  err instanceof Error
                                    ? err.message
                                    : t('common.error'),
                                ),
                            },
                          )
                        }}
                        submissionStatusLabels={submissionStatusLabels}
                        submissionStatusColors={submissionStatusColors}
                        formatDateTime={formatDateTime}
                        t={t}
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
                  className="rounded-lg border-2 border-dashed border-border bg-card p-4 hover:border-primary/40 transition-colors"
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

                  <button
                    onClick={() => handleUseTemplate(tmpl)}
                    className="w-full rounded-lg border border-border px-3 py-2 text-sm font-medium text-foreground hover:bg-secondary transition-colors"
                  >
                    {t('formulare.vorlagen.verwenden')}
                  </button>
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
              {selectedSubmission.status === 'new' && (
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
              {selectedSubmission.status !== 'archived' && (
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
            {/* Submitter info */}
            <div className="rounded-lg border border-border bg-secondary/30 p-3">
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                {t('formulare.submission.absender')}
              </p>
              <p className="text-sm font-medium text-foreground">
                {selectedSubmission.submittedBy ?? '--'}
              </p>
            </div>

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
          </div>
        )}
      </DetailModal>

      {/* ====================== FORMULAR DETAIL MODAL ====================== */}
      <DetailModal
        open={!!selectedForm}
        onClose={() => setSelectedForm(null)}
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
                <button
                  onClick={() => {
                    const form = selectedForm
                    setSelectedForm(null)
                    setShowShareDialog(form)
                  }}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors"
                >
                  <Share2 className="h-3.5 w-3.5" />
                  {t('formulare.actions.teilen')}
                </button>
                <button
                  onClick={() => {
                    const newStatus =
                      selectedForm.status === 'archived' ? 'active' : 'archived'
                    updateSchema.mutate(
                      { id: selectedForm.id, status: newStatus },
                      {
                        onSuccess: () => {
                          toast.success(
                            selectedForm.status === 'archived'
                              ? t('formulare.toast.aktiviert', {
                                  name: selectedForm.title,
                                })
                              : t('formulare.toast.archiviert', {
                                  name: selectedForm.title,
                                }),
                          )
                          setSelectedForm({ ...selectedForm, status: newStatus })
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
                  <Archive className="h-3.5 w-3.5" />
                  {selectedForm.status === 'archived'
                    ? t('formulare.actions.aktivieren')
                    : t('formulare.actions.archivieren')}
                </button>
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
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setPreviewSchema(selectedForm)}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
                >
                  <Eye className="h-4 w-4" />
                  {t('formulare.editor.vorschau')}
                </button>
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
              </div>
            </div>
          ) : undefined
        }
      >
        {selectedForm && (
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
      </DetailModal>

      {/* ====================== STANDALONE FORM PREVIEW ====================== */}
      <DetailModal
        open={!!previewSchema}
        onClose={() => setPreviewSchema(null)}
        title={previewSchema?.title || t('formulare.preview.formularname')}
        subtitle={t('formulare.editor.vorschau')}
        onBack={selectedForm ? () => setPreviewSchema(null) : undefined}
      >
        {previewSchema && (
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
        )}
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

      {/* ====================== SHARE DIALOG ====================== */}
      <Dialog
        open={!!showShareDialog}
        onOpenChange={(o) => {
          if (!o) setShowShareDialog(null)
        }}
      >
        <DialogContent className="gap-0 p-0 max-w-sm">
          <DialogHeader className="border-b border-border px-5 py-4">
            <div className="flex items-center gap-2">
              <Share2 className="h-5 w-5 text-primary" />
              <DialogTitle className="text-base font-semibold text-foreground">
                {t('formulare.share.title')}
              </DialogTitle>
            </div>
            <DialogDescription className="sr-only">
              {t('formulare.share.title')}
            </DialogDescription>
          </DialogHeader>

          <div className="p-5 space-y-4">
            <div className="rounded-lg border border-border bg-secondary/30 p-3">
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                {t('formulare.share.formular')}
              </p>
              <p className="text-sm font-medium text-foreground">
                {showShareDialog?.title}
              </p>
            </div>

            <div className="space-y-2">
              <button
                onClick={() => {
                  toast.success(t('formulare.toast.linkKopiert'))
                  setShowShareDialog(null)
                }}
                className="flex w-full items-center gap-3 rounded-lg border border-border p-3 hover:bg-secondary/50 transition-colors"
              >
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
                  <Link2 className="h-4 w-4 text-primary" />
                </div>
                <div className="text-left">
                  <p className="text-sm font-medium text-foreground">
                    {t('formulare.share.linkKopieren')}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {t('formulare.share.linkHint')}
                  </p>
                </div>
              </button>

              <button
                onClick={() => {
                  toast.success(t('formulare.toast.emailGesendet'))
                  setShowShareDialog(null)
                }}
                className="flex w-full items-center gap-3 rounded-lg border border-border p-3 hover:bg-secondary/50 transition-colors"
              >
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-info-light">
                  <Mail className="h-4 w-4 text-info" />
                </div>
                <div className="text-left">
                  <p className="text-sm font-medium text-foreground">
                    {t('formulare.share.perEmail')}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {t('formulare.share.emailHint')}
                  </p>
                </div>
              </button>
            </div>
          </div>

          <DialogFooter className="border-t border-border px-5 py-4">
            <button
              onClick={() => setShowShareDialog(null)}
              className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              {t('formulare.share.schliessen')}
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
}

function SubmissionsPanel({
  schemaId,
  onSelectSubmission,
  onUpdateStatus,
  submissionStatusLabels,
  submissionStatusColors,
  formatDateTime,
  t,
}: SubmissionsPanelProps) {
  // Hits the warm cache populated by the page-level eager useQueries fetch.
  const { data, isLoading } = useSubmissions(schemaId)
  const items = data?.items ?? []

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

  return (
    <div className="border-t border-border">
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
          {items.map((sub) => (
            <tr
              key={sub.id}
              onClick={() => onSelectSubmission(sub)}
              className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer"
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
                  ]}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// Import useSubmissions here to keep the sub-component working
import { useSubmissions } from '@/api/hooks/useFormulare'
