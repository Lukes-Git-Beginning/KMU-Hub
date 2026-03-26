import { useState, useMemo, useCallback } from 'react'
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
  Star,
  Paperclip,
  Link2,
  Mail,
  StarOff,
  Download,
  Zap,
  ListTodo,
  UserPlus,
  Split,
  Info,
  Globe,
  FileInput,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useFormulareStore,
  type Form,
  type FormField,
  type FormFieldType,
  type FormSubmission,
} from '@/stores/formulare'
import { ItemActions, ConfirmDialog, EmptyState, DetailPanel, PageHeader } from '@/components/shared'
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

type TabKey = 'formulare' | 'eingaenge' | 'vorlagen'

const formStatusLabels: Record<Form['status'], string> = {
  active: 'Aktiv',
  draft: 'Entwurf',
  archived: 'Archiviert',
}

const formStatusColors: Record<Form['status'], string> = {
  active: 'bg-success-light text-success',
  draft: 'bg-secondary text-muted-foreground',
  archived: 'bg-warning-light text-warning',
}

const submissionStatusLabels: Record<FormSubmission['status'], string> = {
  new: 'Neu',
  read: 'Gelesen',
  archived: 'Archiviert',
}

const submissionStatusColors: Record<FormSubmission['status'], string> = {
  new: 'bg-info-light text-info',
  read: 'bg-secondary text-muted-foreground',
  archived: 'bg-warning-light text-warning',
}

const FIELD_TYPE_LABELS: Record<FormFieldType, string> = {
  text: 'Textfeld',
  textarea: 'Textbereich',
  select: 'Auswahl',
  checkbox: 'Checkbox',
  radio: 'Optionen',
  date: 'Datum',
  number: 'Zahl',
  rating: 'Bewertung',
  file: 'Datei',
}

const FIELD_TYPE_ICONS: Record<FormFieldType, typeof Type> = {
  text: Type,
  textarea: AlignLeft,
  select: ChevronDown,
  checkbox: CheckSquare,
  radio: Circle,
  date: Calendar,
  number: Hash,
  rating: Star,
  file: Paperclip,
}

const FIELD_TYPE_OPTIONS: { value: FormFieldType; label: string }[] = [
  { value: 'text', label: 'Textfeld' },
  { value: 'textarea', label: 'Textbereich' },
  { value: 'select', label: 'Auswahl (Dropdown)' },
  { value: 'radio', label: 'Optionen (Radio)' },
  { value: 'checkbox', label: 'Checkbox' },
  { value: 'date', label: 'Datum' },
  { value: 'number', label: 'Zahl' },
  { value: 'rating', label: 'Bewertung (Sterne)' },
  { value: 'file', label: 'Datei-Upload' },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function FormularePage() {
  const {
    forms,
    templates,
    submissions,
    addForm,
    updateForm,
    deleteForm,
    duplicateForm,
    updateSubmissionStatus,
    addField,
    removeField,
    reorderFields,
  } = useFormulareStore()

  // Tab & search
  const [tab, setTab] = useState<TabKey>('formulare')
  const [search, setSearch] = useState('')

  // Editor state
  const [editingFormId, setEditingFormId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  const [editDescription, setEditDescription] = useState('')

  // Detail panel for submissions
  const [selectedSubmission, setSelectedSubmission] = useState<FormSubmission | null>(null)

  // Expanded submission groups
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set())

  // Dialogs
  const [confirmDelete, setConfirmDelete] = useState<Form | null>(null)
  const [showNewFormDialog, setShowNewFormDialog] = useState(false)
  const [showShareDialog, setShowShareDialog] = useState<Form | null>(null)
  const [showFieldConfigDialog, setShowFieldConfigDialog] = useState<{
    formId: string
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
  const [configConditionalEnabled, setConfigConditionalEnabled] = useState(false)
  const [configConditionalFieldId, setConfigConditionalFieldId] = useState('')
  const [configConditionalOperator, setConfigConditionalOperator] = useState<'equals' | 'not_equals' | 'contains'>('equals')
  const [configConditionalValue, setConfigConditionalValue] = useState('')

  // 10.2 — Multi-page preview state
  const [previewPage, setPreviewPage] = useState(0)

  // 10.3 — Actions UI state
  const [showActionsSection, setShowActionsSection] = useState(false)
  const [showAddActionMenu, setShowAddActionMenu] = useState(false)
  const [editingActionIndex, setEditingActionIndex] = useState<number | null>(null)
  const [actionEmailTo, setActionEmailTo] = useState('')
  const [actionTaskTitle, setActionTaskTitle] = useState('')
  const [actionTaskAssignee, setActionTaskAssignee] = useState('')

  // 10.4 — Public preview state
  const [showPublicPreview, setShowPublicPreview] = useState(false)

  // 10.5 — Export dropdown state
  const [showExportMenu, setShowExportMenu] = useState(false)

  // ---------------------------------------------------------------------------
  // Derived data
  // ---------------------------------------------------------------------------

  const activeForms = useMemo(
    () => forms.filter((f) => !f.isTemplate),
    [forms]
  )

  const filteredForms = useMemo(() => {
    if (!search) return activeForms
    const q = search.toLowerCase()
    return activeForms.filter(
      (f) =>
        f.name.toLowerCase().includes(q) ||
        f.description.toLowerCase().includes(q) ||
        f.createdBy.toLowerCase().includes(q)
    )
  }, [activeForms, search])

  const filteredSubmissions = useMemo(() => {
    if (!search) return submissions
    const q = search.toLowerCase()
    return submissions.filter(
      (s) =>
        s.formName.toLowerCase().includes(q) ||
        s.submittedBy.toLowerCase().includes(q)
    )
  }, [submissions, search])

  const submissionsByForm = useMemo(() => {
    const grouped: Record<string, { formName: string; items: FormSubmission[] }> = {}
    for (const sub of filteredSubmissions) {
      if (!grouped[sub.formId]) {
        grouped[sub.formId] = { formName: sub.formName, items: [] }
      }
      grouped[sub.formId].items.push(sub)
    }
    return grouped
  }, [filteredSubmissions])

  const activeFormCount = activeForms.filter((f) => f.status === 'active').length

  const weekAgo = useMemo(() => {
    const d = new Date()
    d.setDate(d.getDate() - 7)
    return d.toISOString()
  }, [])

  const weeklySubmissionCount = submissions.filter(
    (s) => s.submittedAt >= weekAgo
  ).length

  const newSubmissionCount = submissions.filter((s) => s.status === 'new').length

  const editingForm = useMemo(
    () => forms.find((f) => f.id === editingFormId) ?? null,
    [forms, editingFormId]
  )

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------

  const formatDate = (d: string) =>
    new Date(d.includes('T') ? d : d + 'T00:00:00').toLocaleDateString('de-CH')

  const formatDateTime = (d: string) =>
    new Date(d).toLocaleDateString('de-CH', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })

  // Editor (declared before getFormActions to satisfy React compiler)
  const openEditor = (form: Form) => {
    setEditingFormId(form.id)
    setEditName(form.name)
    setEditDescription(form.description)
  }

  const getFormActions = useCallback(
    (form: Form) => [
      {
        label: 'Bearbeiten',
        icon: Edit,
        onClick: () => openEditor(form),
      },
      {
        label: 'Duplizieren',
        icon: Copy,
        onClick: () => {
          duplicateForm(form.id)
          toast.success(`"${form.name}" wurde dupliziert`)
        },
      },
      {
        label: 'Teilen',
        icon: Share2,
        onClick: () => setShowShareDialog(form),
      },
      {
        label:
          form.status === 'archived' ? 'Aktivieren' : 'Archivieren',
        icon: Archive,
        onClick: () => {
          const newStatus = form.status === 'archived' ? 'active' : 'archived'
          updateForm(form.id, { status: newStatus })
          toast.success(
            form.status === 'archived'
              ? `"${form.name}" wurde aktiviert`
              : `"${form.name}" wurde archiviert`
          )
        },
      },
      { separator: true as const, label: '', onClick: () => {} },
      {
        label: 'Loeschen',
        icon: Trash2,
        variant: 'destructive' as const,
        onClick: () => setConfirmDelete(form),
      },
    ],
    [duplicateForm, updateForm]
  )

  const handleDeleteForm = (form: Form) => {
    deleteForm(form.id)
    setConfirmDelete(null)
    toast.success(`"${form.name}" wurde geloescht`)
  }

  const closeEditor = () => {
    setEditingFormId(null)
    setShowAddFieldMenu(false)
  }

  const saveEditor = () => {
    if (!editingFormId) return
    updateForm(editingFormId, {
      name: editName,
      description: editDescription,
    })
    toast.success('Formular gespeichert')
    closeEditor()
  }

  const handleAddField = (type: FormFieldType) => {
    if (!editingFormId) return
    addField(editingFormId, {
      type,
      label: FIELD_TYPE_LABELS[type],
      required: false,
      placeholder: '',
      options: type === 'select' || type === 'radio' ? ['Option 1', 'Option 2'] : undefined,
    })
    setShowAddFieldMenu(false)
  }

  const handleRemoveField = (fieldId: string) => {
    if (!editingFormId) return
    removeField(editingFormId, fieldId)
  }

  const openFieldConfig = (field: FormField) => {
    if (!editingFormId) return
    setConfigLabel(field.label)
    setConfigRequired(field.required)
    setConfigPlaceholder(field.placeholder ?? '')
    setConfigOptions(field.options?.join(', ') ?? '')
    // 10.1 — populate conditional logic state
    setConfigConditionalEnabled(!!field.conditionalLogic)
    setConfigConditionalFieldId(field.conditionalLogic?.fieldId ?? '')
    setConfigConditionalOperator(field.conditionalLogic?.operator ?? 'equals')
    setConfigConditionalValue(field.conditionalLogic?.value ?? '')
    setShowFieldConfigDialog({ formId: editingFormId, field })
  }

  const saveFieldConfig = () => {
    if (!showFieldConfigDialog || !editingFormId) return
    const { field } = showFieldConfigDialog
    const form = forms.find((f) => f.id === editingFormId)
    if (!form) return

    const updatedFields = form.fields.map((f) =>
      f.id === field.id
        ? {
            ...f,
            label: configLabel,
            required: configRequired,
            placeholder: configPlaceholder || undefined,
            options:
              f.type === 'select' || f.type === 'radio'
                ? configOptions
                    .split(',')
                    .map((o) => o.trim())
                    .filter(Boolean)
                : f.options,
            // 10.1 — Save conditional logic
            conditionalLogic:
              configConditionalEnabled && configConditionalFieldId
                ? {
                    fieldId: configConditionalFieldId,
                    operator: configConditionalOperator,
                    value: configConditionalValue,
                  }
                : undefined,
          }
        : f
    )
    reorderFields(editingFormId, updatedFields)
    setShowFieldConfigDialog(null)
    toast.success('Feld aktualisiert')
  }

  // New form
  const resetNewFormDialog = () => {
    setNewFormName('')
    setNewFormDescription('')
    setNewFormTemplateId('')
  }

  const handleCreateForm = () => {
    if (!newFormName.trim()) {
      toast.error('Bitte einen Namen eingeben')
      return
    }
    if (newFormTemplateId) {
      duplicateForm(newFormTemplateId)
      // Update the name of the duplicated form
      const newest = useFormulareStore.getState().forms[0]
      if (newest) {
        updateForm(newest.id, {
          name: newFormName,
          description: newFormDescription,
        })
      }
    } else {
      addForm({
        name: newFormName,
        description: newFormDescription,
        status: 'draft',
        fields: [],
        createdBy: 'Aktueller Benutzer',
        isTemplate: false,
      })
    }
    toast.success(`Formular "${newFormName}" erstellt`)
    resetNewFormDialog()
    setShowNewFormDialog(false)
  }

  // Submissions
  const toggleGroup = (formId: string) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(formId)) next.delete(formId)
      else next.add(formId)
      return next
    })
  }

  const getFieldForSubmission = (formId: string, fieldId: string): FormField | undefined => {
    const form = [...forms, ...templates].find((f) => f.id === formId)
    return form?.fields.find((f) => f.id === fieldId)
  }

  // Template use
  const handleUseTemplate = (template: Form) => {
    duplicateForm(template.id)
    toast.success(`Neues Formular aus "${template.name}" erstellt`)
  }

  // ---------------------------------------------------------------------------
  // 10.2 — Page break helpers
  // ---------------------------------------------------------------------------

  /** Insert a page break after the last field. Uses a sentinel field type pattern. */
  const handleInsertPageBreak = () => {
    if (!editingFormId || !editingForm) return
    // We model page breaks as increments in field.page values.
    // Compute current max page
    const maxPage = editingForm.fields.reduce((m, f) => Math.max(m, f.page ?? 0), 0)
    // All subsequent fields added will be on the new page.
    // For now, just insert a marker: add a text field with a special label, then immediately
    // update all field pages. Actually, let's compute pages from divider positions.
    // Strategy: we track page breaks as field entries with type 'text' and label '__page_break__'
    // Better: just update the page numbers on existing fields.
    // We'll add a "page break" by bumping page numbers of all trailing fields.
    // Actually simplest: store page number on each field. When inserting a page break,
    // all fields after the break get page+1.
    const newPage = maxPage + 1
    // Update form's pageCount
    updateForm(editingFormId, { pageCount: newPage + 1 })
    // We need to mark where the break is. We'll set page = newPage on a new field.
    // Actually, for a page break *divider* the cleanest approach: we bump page on
    // all fields that don't yet have a page set, then add a placeholder.
    // Let's keep it simpler: add a special hidden text field as a page divider marker.
    addField(editingFormId, {
      type: 'text',
      label: '__page_break__',
      required: false,
      placeholder: '',
      page: newPage,
    })
    setShowAddFieldMenu(false)
  }

  /** Compute the page a field is on based on page-break markers */
  const computeFieldPages = (fields: FormField[]): Map<string, number> => {
    const map = new Map<string, number>()
    let currentPage = 0
    for (const f of fields) {
      if (f.label === '__page_break__') {
        currentPage++
        map.set(f.id, -1) // marker itself
      } else {
        map.set(f.id, currentPage)
      }
    }
    return map
  }

   
  const fieldPageMap = useMemo(
     
    () => (editingForm ? computeFieldPages(editingForm.fields) : new Map<string, number>()),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentionally depending on editingForm.fields only, not the whole object
    [editingForm?.fields]
  )

  const totalPages = useMemo(() => {
    if (!editingForm) return 1
    let pages = 0
    for (const f of editingForm.fields) {
      if (f.label === '__page_break__') pages++
    }
    return pages + 1
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentionally depending on editingForm.fields only, not the whole object
  }, [editingForm?.fields])

  // ---------------------------------------------------------------------------
  // 10.3 — Action handlers
  // ---------------------------------------------------------------------------

  const handleAddAction = (type: 'email' | 'task' | 'crm_contact') => {
    if (!editingFormId || !editingForm) return
    const existing = editingForm.actions ?? []
    const newAction = { type, config: {} as Record<string, string> }
    if (type === 'crm_contact') {
      // No config needed, just add
      updateForm(editingFormId, { actions: [...existing, newAction] })
      toast.success('CRM-Kontakt Aktion hinzugefuegt')
    } else {
      // Open inline editor
      updateForm(editingFormId, { actions: [...existing, newAction] })
      setEditingActionIndex(existing.length)
      if (type === 'email') {
        setActionEmailTo('')
      } else {
        setActionTaskTitle('')
        setActionTaskAssignee('')
      }
    }
    setShowAddActionMenu(false)
  }

  const handleSaveAction = (index: number) => {
    if (!editingFormId || !editingForm) return
    const actions = [...(editingForm.actions ?? [])]
    const action = actions[index]
    if (!action) return
    if (action.type === 'email') {
      actions[index] = { ...action, config: { to: actionEmailTo } }
    } else if (action.type === 'task') {
      actions[index] = { ...action, config: { title: actionTaskTitle, assignee: actionTaskAssignee } }
    }
    updateForm(editingFormId, { actions })
    setEditingActionIndex(null)
    toast.success('Aktion gespeichert')
  }

  const handleRemoveAction = (index: number) => {
    if (!editingFormId || !editingForm) return
    const actions = [...(editingForm.actions ?? [])]
    actions.splice(index, 1)
    updateForm(editingFormId, { actions })
    toast.success('Aktion entfernt')
  }

  const actionTypeLabels: Record<string, string> = {
    email: 'E-Mail senden',
    task: 'Task erstellen',
    crm_contact: 'CRM-Kontakt anlegen',
  }

  const actionTypeIcons: Record<string, typeof Mail> = {
    email: Mail,
    task: ListTodo,
    crm_contact: UserPlus,
  }

  // ---------------------------------------------------------------------------
  // 10.1 — Conditional logic evaluation helper
  // ---------------------------------------------------------------------------

  const evaluateCondition = (
    logic: FormField['conditionalLogic'],
    answers: Record<string, string | string[] | number | boolean>
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

  const _conditionOperatorLabels: Record<string, string> = {
    equals: 'ist gleich',
    not_equals: 'ist nicht gleich',
    contains: 'enthaelt',
  }

  // ---------------------------------------------------------------------------
  // Render helpers
  // ---------------------------------------------------------------------------

  const renderStars = (value: number, max = 5) => (
    <div className="flex items-center gap-0.5">
      {Array.from({ length: max }, (_, i) => (
        <Star
          key={i}
          className={`h-4 w-4 ${
            i < value ? 'fill-warning text-warning' : 'text-muted-foreground/30'
          }`}
        />
      ))}
    </div>
  )

  const renderFieldIcon = (type: FormFieldType, className = 'h-4 w-4') => {
    const Icon = FIELD_TYPE_ICONS[type]
    return <Icon className={className} />
  }

  /** Render a single answer value in the submission detail */
  const renderAnswer = (fieldId: string, value: string | string[] | number | boolean, formId: string) => {
    const field = getFieldForSubmission(formId, fieldId)
    if (!field) {
      return <span className="text-sm text-foreground">{String(value)}</span>
    }

    switch (field.type) {
      case 'rating':
        return renderStars(typeof value === 'number' ? value : 0)
      case 'checkbox':
        return (
          <span
            className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${
              value ? 'bg-success-light text-success' : 'bg-secondary text-muted-foreground'
            }`}
          >
            {value ? 'Ja' : 'Nein'}
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
            Datei hochgeladen
          </span>
        ) : (
          <span className="text-sm text-muted-foreground">Keine Datei</span>
        )
      default:
        return (
          <span className="text-sm text-foreground">
            {String(value) || <span className="text-muted-foreground">--</span>}
          </span>
        )
    }
  }

  /** Render a form field preview (right side of editor) */
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
            <option>Bitte waehlen...</option>
            {field.options?.map((opt) => (
              <option key={opt}>{opt}</option>
            ))}
          </select>
        )
      case 'radio':
        return (
          <div className="space-y-1.5">
            {field.options?.map((opt) => (
              <label key={opt} className="flex items-center gap-2 text-sm text-muted-foreground">
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
      case 'rating':
        return (
          <div className="flex items-center gap-1">
            {[1, 2, 3, 4, 5].map((n) => (
              <StarOff key={n} className="h-5 w-5 text-muted-foreground/30" />
            ))}
          </div>
        )
      case 'file':
        return (
          <div className="flex items-center gap-2 rounded-lg border-2 border-dashed border-border bg-secondary/30 px-4 py-6 text-sm text-muted-foreground">
            <Paperclip className="h-4 w-4" />
            Datei auswaehlen oder hierher ziehen
          </div>
        )
      default:
        return null
    }
  }

  // ---------------------------------------------------------------------------
  // JSX — Editor View
  // ---------------------------------------------------------------------------

  if (editingForm) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        {/* Editor Header */}
        <div className="flex items-center gap-3 mb-6">
          <button
            onClick={closeEditor}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            <ArrowLeft className="h-4 w-4" />
            Zurueck
          </button>
          <div className="flex-1" />

          {/* 10.4 — Oeffentlich toggle */}
          <div className="flex items-center gap-2 mr-2">
            <Globe className="h-4 w-4 text-muted-foreground" />
            <span className="text-xs text-muted-foreground">Oeffentlich</span>
            <button
              onClick={() => {
                if (!editingFormId) return
                updateForm(editingFormId, { isPublic: !editingForm.isPublic })
                toast.success(editingForm.isPublic ? 'Formular ist jetzt privat' : 'Formular ist jetzt oeffentlich')
              }}
              className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                editingForm.isPublic ? 'bg-primary' : 'bg-secondary'
              }`}
            >
              <span
                className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                  editingForm.isPublic ? 'translate-x-4.5' : 'translate-x-0.5'
                }`}
              />
            </button>
          </div>

          {/* 10.4 — Public preview button */}
          <button
            onClick={() => setShowPublicPreview(true)}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            title="Oeffentliche Vorschau"
          >
            <Eye className="h-4 w-4" />
            Vorschau
          </button>

          <button
            onClick={closeEditor}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            Abbrechen
          </button>
          <button
            onClick={saveEditor}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            Speichern
          </button>
        </div>

        {/* Name + Description */}
        <div className="mb-6 space-y-3">
          <input
            type="text"
            value={editName}
            onChange={(e) => setEditName(e.target.value)}
            placeholder="Formularname"
            className="w-full text-xl font-semibold text-foreground bg-transparent border-b border-border pb-2 focus:outline-none focus:border-primary transition-colors placeholder:text-input-placeholder"
          />
          <input
            type="text"
            value={editDescription}
            onChange={(e) => setEditDescription(e.target.value)}
            placeholder="Beschreibung hinzufuegen..."
            className="w-full text-sm text-muted-foreground bg-transparent border-b border-border-muted pb-2 focus:outline-none focus:border-primary transition-colors placeholder:text-input-placeholder"
          />
        </div>

        {/* Editor two-column layout */}
        <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
          {/* Left: Field list */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-medium text-foreground">
                Felder ({editingForm.fields.filter((f) => f.label !== '__page_break__').length})
              </h3>
              <div className="relative">
                <button
                  onClick={() => setShowAddFieldMenu(!showAddFieldMenu)}
                  className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
                >
                  <Plus className="h-3.5 w-3.5" />
                  Feld hinzufuegen
                </button>
                {showAddFieldMenu && (
                  <div className="absolute right-0 top-full mt-1 z-20 w-56 rounded-lg border border-border bg-card shadow-xl py-1">
                    {FIELD_TYPE_OPTIONS.map((opt) => {
                      const Icon = FIELD_TYPE_ICONS[opt.value]
                      return (
                        <button
                          key={opt.value}
                          onClick={() => handleAddField(opt.value)}
                          className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
                        >
                          <Icon className="h-4 w-4 text-muted-foreground" />
                          {opt.label}
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
                      Seitenumbruch einfuegen
                    </button>
                  </div>
                )}
              </div>
            </div>

            {editingForm.fields.length === 0 ? (
              <div className="rounded-lg border-2 border-dashed border-border p-8 text-center">
                <ClipboardList className="mx-auto h-8 w-8 text-muted-foreground/40" />
                <p className="mt-2 text-sm text-muted-foreground">
                  Noch keine Felder. Klicke auf "Feld hinzufuegen".
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                {editingForm.fields.map((field, _idx) => {
                  // 10.2 — Page break divider
                  if (field.label === '__page_break__') {
                    const pageNum = (fieldPageMap.get(field.id) === -1)
                      ? Array.from(fieldPageMap.entries())
                          .filter(([, v]) => v === -1)
                          .findIndex(([k]) => k === field.id) + 1
                      : 1
                    return (
                      <div key={field.id} className="flex items-center gap-3 py-2 group">
                        <div className="flex-1 border-t border-dashed border-primary/40" />
                        <span className="text-xs font-medium text-primary/70 whitespace-nowrap">
                          --- Seite {pageNum + 1} ---
                        </span>
                        <div className="flex-1 border-t border-dashed border-primary/40" />
                        <button
                          onClick={() => handleRemoveField(field.id)}
                          className="rounded-md p-1 text-muted-foreground hover:text-error hover:bg-error-light transition-colors opacity-0 group-hover:opacity-100"
                          title="Seitenumbruch entfernen"
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
                              Pflicht
                            </span>
                          )}
                          {/* 10.1 — Conditional badge */}
                          {field.conditionalLogic && (
                            <span className="rounded bg-warning-light px-1.5 py-0.5 text-[9px] font-medium text-warning shrink-0">
                              Bedingt
                            </span>
                          )}
                        </div>
                        <span className="text-[10px] text-muted-foreground">
                          {FIELD_TYPE_LABELS[field.type]}
                          {field.options ? ` (${field.options.length} Optionen)` : ''}
                        </span>
                      </div>
                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button
                          onClick={() => openFieldConfig(field)}
                          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
                          title="Bearbeiten"
                        >
                          <Edit className="h-3.5 w-3.5" />
                        </button>
                        <button
                          onClick={() => handleRemoveField(field.id)}
                          className="rounded-md p-1.5 text-muted-foreground hover:text-error hover:bg-error-light transition-colors"
                          title="Loeschen"
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
            <h3 className="text-sm font-medium text-foreground mb-3">Vorschau</h3>
            <div className="rounded-xl border border-border bg-card p-5 space-y-4">
              <div className="mb-4">
                <h2 className="text-lg font-semibold text-foreground">{editName || 'Formularname'}</h2>
                {editDescription && (
                  <p className="text-sm text-muted-foreground mt-1">{editDescription}</p>
                )}
              </div>

              {/* 10.2 — Page indicator */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between rounded-lg bg-secondary/50 px-3 py-2">
                  <button
                    onClick={() => setPreviewPage(Math.max(0, previewPage - 1))}
                    disabled={previewPage === 0}
                    className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground disabled:opacity-40 transition-colors"
                  >
                    <ChevronLeft className="h-3.5 w-3.5" />
                    Zurueck
                  </button>
                  <span className="text-xs font-medium text-foreground">
                    Seite {previewPage + 1} von {totalPages}
                  </span>
                  <button
                    onClick={() => setPreviewPage(Math.min(totalPages - 1, previewPage + 1))}
                    disabled={previewPage >= totalPages - 1}
                    className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground disabled:opacity-40 transition-colors"
                  >
                    Weiter
                    <ChevronRight className="h-3.5 w-3.5" />
                  </button>
                </div>
              )}

              {editingForm.fields.length === 0 ? (
                <p className="text-sm text-muted-foreground italic">
                  Fuege Felder hinzu, um eine Vorschau zu sehen.
                </p>
              ) : (
                editingForm.fields
                  .filter((field) => {
                    // Hide page break markers
                    if (field.label === '__page_break__') return false
                    // 10.2 — Filter by current preview page
                    if (totalPages > 1) {
                      const page = fieldPageMap.get(field.id) ?? 0
                      return page === previewPage
                    }
                    return true
                  })
                  .map((field) => (
                    <div key={field.id} className={`space-y-1.5 ${field.conditionalLogic ? 'opacity-60' : ''}`}>
                      <label className="text-sm font-medium text-foreground flex items-center gap-2">
                        {field.label}
                        {field.required && <span className="text-destructive ml-0.5">*</span>}
                        {/* 10.1 — Conditional badge in preview */}
                        {field.conditionalLogic && (
                          <span className="rounded bg-warning-light px-1.5 py-0.5 text-[9px] font-medium text-warning">
                            Bedingt
                          </span>
                        )}
                      </label>
                      {renderFieldPreview(field)}
                    </div>
                  ))
              )}
              {editingForm.fields.filter((f) => f.label !== '__page_break__').length > 0 && (
                <button
                  disabled
                  className="mt-2 rounded-lg bg-primary/50 px-4 py-2 text-sm text-primary-foreground cursor-not-allowed"
                >
                  {totalPages > 1 && previewPage < totalPages - 1 ? 'Weiter' : 'Absenden'}
                </button>
              )}
            </div>
          </div>
        </div>

        {/* ====================== 10.3 — AKTIONEN SECTION ====================== */}
        <div className="mt-6">
          <button
            onClick={() => setShowActionsSection(!showActionsSection)}
            className="flex items-center gap-2 text-sm font-medium text-foreground hover:text-primary transition-colors"
          >
            {showActionsSection ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            <Zap className="h-4 w-4 text-primary" />
            Automatische Aktionen
            {(editingForm.actions?.length ?? 0) > 0 && (
              <span className="rounded-full bg-primary-light px-2 py-0.5 text-[10px] font-medium text-primary">
                {editingForm.actions!.length}
              </span>
            )}
          </button>

          {showActionsSection && (
            <div className="mt-3 space-y-3">
              {/* Existing actions */}
              {(editingForm.actions ?? []).map((action, idx) => {
                const ActionIcon = actionTypeIcons[action.type] ?? Zap
                const isEditing = editingActionIndex === idx
                return (
                  <div
                    key={idx}
                    className="rounded-lg border border-border bg-card p-3"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary-light shrink-0">
                          <ActionIcon className="h-4 w-4 text-primary" />
                        </div>
                        <div>
                          <p className="text-sm font-medium text-foreground">
                            {actionTypeLabels[action.type]}
                          </p>
                          {action.type === 'email' && action.config.to && (
                            <p className="text-[10px] text-muted-foreground">an: {action.config.to}</p>
                          )}
                          {action.type === 'task' && action.config.title && (
                            <p className="text-[10px] text-muted-foreground">
                              {action.config.title}{action.config.assignee ? ` → ${action.config.assignee}` : ''}
                            </p>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-1">
                        {action.type !== 'crm_contact' && (
                          <button
                            onClick={() => {
                              setEditingActionIndex(idx)
                              if (action.type === 'email') setActionEmailTo(action.config.to ?? '')
                              if (action.type === 'task') {
                                setActionTaskTitle(action.config.title ?? '')
                                setActionTaskAssignee(action.config.assignee ?? '')
                              }
                            }}
                            className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
                          >
                            <Edit className="h-3.5 w-3.5" />
                          </button>
                        )}
                        <button
                          onClick={() => handleRemoveAction(idx)}
                          className="rounded-md p-1.5 text-muted-foreground hover:text-error hover:bg-error-light transition-colors"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </div>

                    {/* Inline edit for email */}
                    {isEditing && action.type === 'email' && (
                      <div className="mt-3 space-y-2 border-t border-border-muted pt-3">
                        <div className="space-y-1">
                          <label className="text-xs font-medium text-muted-foreground">Empfaenger</label>
                          <input
                            type="email"
                            value={actionEmailTo}
                            onChange={(e) => setActionEmailTo(e.target.value)}
                            placeholder="email@beispiel.de"
                            className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                          />
                        </div>
                        <div className="flex justify-end gap-2">
                          <button
                            onClick={() => setEditingActionIndex(null)}
                            className="rounded px-2 py-1 text-xs text-muted-foreground hover:bg-secondary"
                          >
                            Abbrechen
                          </button>
                          <button
                            onClick={() => handleSaveAction(idx)}
                            className="rounded bg-primary px-2 py-1 text-xs text-primary-foreground hover:bg-button-primary-hover"
                          >
                            Speichern
                          </button>
                        </div>
                      </div>
                    )}

                    {/* Inline edit for task */}
                    {isEditing && action.type === 'task' && (
                      <div className="mt-3 space-y-2 border-t border-border-muted pt-3">
                        <div className="space-y-1">
                          <label className="text-xs font-medium text-muted-foreground">Task-Titel</label>
                          <input
                            type="text"
                            value={actionTaskTitle}
                            onChange={(e) => setActionTaskTitle(e.target.value)}
                            placeholder="z.B. Feedback auswerten"
                            className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                          />
                        </div>
                        <div className="space-y-1">
                          <label className="text-xs font-medium text-muted-foreground">Zustaendig</label>
                          <input
                            type="text"
                            value={actionTaskAssignee}
                            onChange={(e) => setActionTaskAssignee(e.target.value)}
                            placeholder="z.B. Max Mustermann"
                            className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                          />
                        </div>
                        <div className="flex justify-end gap-2">
                          <button
                            onClick={() => setEditingActionIndex(null)}
                            className="rounded px-2 py-1 text-xs text-muted-foreground hover:bg-secondary"
                          >
                            Abbrechen
                          </button>
                          <button
                            onClick={() => handleSaveAction(idx)}
                            className="rounded bg-primary px-2 py-1 text-xs text-primary-foreground hover:bg-button-primary-hover"
                          >
                            Speichern
                          </button>
                        </div>
                      </div>
                    )}
                  </div>
                )
              })}

              {/* Add action button */}
              <div className="relative">
                <button
                  onClick={() => setShowAddActionMenu(!showAddActionMenu)}
                  className="flex items-center gap-1.5 rounded-lg border border-dashed border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:border-primary/40 transition-colors"
                >
                  <Plus className="h-3.5 w-3.5" />
                  Aktion hinzufuegen
                </button>
                {showAddActionMenu && (
                  <div className="absolute left-0 top-full mt-1 z-20 w-52 rounded-lg border border-border bg-card shadow-xl py-1">
                    <button
                      onClick={() => handleAddAction('email')}
                      className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
                    >
                      <Mail className="h-4 w-4 text-muted-foreground" />
                      E-Mail senden
                    </button>
                    <button
                      onClick={() => handleAddAction('task')}
                      className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
                    >
                      <ListTodo className="h-4 w-4 text-muted-foreground" />
                      Task erstellen
                    </button>
                    <button
                      onClick={() => handleAddAction('crm_contact')}
                      className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
                    >
                      <UserPlus className="h-4 w-4 text-muted-foreground" />
                      CRM-Kontakt anlegen
                    </button>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        {/* ====================== 10.4 — PUBLIC PREVIEW MODAL ====================== */}
        <Dialog open={showPublicPreview && !!editingForm} onOpenChange={(o) => { if (!o) setShowPublicPreview(false) }}>
          <DialogContent className="gap-0 p-0 max-w-2xl max-h-[90vh] overflow-y-auto bg-white">
              <DialogTitle className="sr-only">Formular-Vorschau</DialogTitle>
              <DialogDescription className="sr-only">Oeffentliche Vorschau des Formulars</DialogDescription>

              {/* Info banner */}
              <div className="flex items-center gap-2 bg-blue-50 border-b border-blue-100 px-6 py-3">
                <Info className="h-4 w-4 text-blue-500 shrink-0" />
                <p className="text-xs text-blue-700">
                  Diese Vorschau zeigt wie das Formular fuer externe Nutzer aussieht
                </p>
              </div>

              {/* Form content */}
              <div className="p-8">
                <div className="mb-6">
                  <h1 className="text-xl font-semibold text-gray-900">
                    {editName || 'Formularname'}
                  </h1>
                  {editDescription && (
                    <p className="text-sm text-gray-500 mt-1">{editDescription}</p>
                  )}
                </div>

                <div className="space-y-5">
                  {editingForm.fields
                    .filter((f) => f.label !== '__page_break__')
                    .map((field) => (
                      <div key={field.id} className="space-y-1.5">
                        <label className="text-sm font-medium text-gray-700">
                          {field.label}
                          {field.required && <span className="text-destructive ml-0.5">*</span>}
                        </label>
                        {/* Render fillable inputs */}
                        {(field.type === 'text' || field.type === 'number') && (
                          <input
                            type={field.type}
                            placeholder={field.placeholder || field.label}
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
                            <option value="">Bitte waehlen...</option>
                            {field.options?.map((opt) => (
                              <option key={opt}>{opt}</option>
                            ))}
                          </select>
                        )}
                        {field.type === 'radio' && (
                          <div className="space-y-1.5">
                            {field.options?.map((opt) => (
                              <label key={opt} className="flex items-center gap-2 text-sm text-gray-700">
                                <input type="radio" name={field.id} className="h-4 w-4 text-blue-600" />
                                {opt}
                              </label>
                            ))}
                          </div>
                        )}
                        {field.type === 'checkbox' && (
                          <label className="flex items-center gap-2 text-sm text-gray-700">
                            <input type="checkbox" className="h-4 w-4 rounded text-blue-600" />
                            {field.label}
                          </label>
                        )}
                        {field.type === 'date' && (
                          <input
                            type="date"
                            className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                        )}
                        {field.type === 'rating' && (
                          <div className="flex items-center gap-1">
                            {[1, 2, 3, 4, 5].map((n) => (
                              <Star key={n} className="h-6 w-6 text-gray-300 hover:text-warning cursor-pointer transition-colors" />
                            ))}
                          </div>
                        )}
                        {field.type === 'file' && (
                          <div className="flex items-center gap-2 rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 px-4 py-6 text-sm text-gray-500">
                            <Paperclip className="h-4 w-4" />
                            Datei auswaehlen oder hierher ziehen
                          </div>
                        )}
                      </div>
                    ))}
                </div>

                {editingForm.fields.filter((f) => f.label !== '__page_break__').length > 0 && (
                  <button className="mt-6 rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-blue-700 transition-colors">
                    Absenden
                  </button>
                )}
              </div>
          </DialogContent>
        </Dialog>

        {/* ====================== FIELD CONFIG DIALOG ====================== */}
        <Dialog open={!!showFieldConfigDialog} onOpenChange={(o) => { if (!o) setShowFieldConfigDialog(null) }}>
          <DialogContent className="gap-0 p-0 max-w-md">
              <DialogHeader className="border-b border-border px-5 py-4">
                <div className="flex items-center gap-2">
                  {showFieldConfigDialog && renderFieldIcon(showFieldConfigDialog.field.type, 'h-5 w-5 text-primary')}
                  <DialogTitle className="text-base font-semibold text-foreground">Feld konfigurieren</DialogTitle>
                </div>
                <DialogDescription className="sr-only">Feldeinstellungen bearbeiten</DialogDescription>
              </DialogHeader>

              <div className="p-5 space-y-4">
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">
                    Bezeichnung <span className="text-destructive">*</span>
                  </label>
                  <input
                    type="text"
                    value={configLabel}
                    onChange={(e) => setConfigLabel(e.target.value)}
                    className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>

                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium text-foreground">Pflichtfeld</label>
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

                {(showFieldConfigDialog.field.type === 'text' ||
                  showFieldConfigDialog.field.type === 'textarea' ||
                  showFieldConfigDialog.field.type === 'number') && (
                  <div className="space-y-1.5">
                    <label className="text-sm font-medium text-foreground">Platzhalter</label>
                    <input
                      type="text"
                      value={configPlaceholder}
                      onChange={(e) => setConfigPlaceholder(e.target.value)}
                      placeholder="Platzhaltertext..."
                      className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    />
                  </div>
                )}

                {(showFieldConfigDialog.field.type === 'select' ||
                  showFieldConfigDialog.field.type === 'radio') && (
                  <div className="space-y-1.5">
                    <label className="text-sm font-medium text-foreground">
                      Optionen <span className="text-xs text-muted-foreground">(kommagetrennt)</span>
                    </label>
                    <input
                      type="text"
                      value={configOptions}
                      onChange={(e) => setConfigOptions(e.target.value)}
                      placeholder="Option 1, Option 2, Option 3"
                      className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    />
                  </div>
                )}

                {/* 10.1 — Conditional Logic Section */}
                <div className="border-t border-border-muted pt-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Zap className="h-4 w-4 text-warning" />
                      <label className="text-sm font-medium text-foreground">Bedingte Anzeige</label>
                    </div>
                    <button
                      onClick={() => setConfigConditionalEnabled(!configConditionalEnabled)}
                      className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                        configConditionalEnabled ? 'bg-primary' : 'bg-secondary'
                      }`}
                    >
                      <span
                        className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                          configConditionalEnabled ? 'translate-x-4.5' : 'translate-x-0.5'
                        }`}
                      />
                    </button>
                  </div>

                  {configConditionalEnabled && (
                    <div className="rounded-lg border border-border bg-secondary/20 p-3 space-y-3">
                      <p className="text-xs text-muted-foreground">
                        Zeige dieses Feld nur wenn...
                      </p>

                      {/* Source field dropdown */}
                      <div className="space-y-1">
                        <label className="text-[10px] uppercase tracking-wider text-muted-foreground">Quellfeld</label>
                        <select
                          value={configConditionalFieldId}
                          onChange={(e) => setConfigConditionalFieldId(e.target.value)}
                          className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                        >
                          <option value="">Feld waehlen...</option>
                          {editingForm!.fields
                            .filter((f) => f.id !== showFieldConfigDialog.field.id && f.label !== '__page_break__')
                            .map((f) => (
                              <option key={f.id} value={f.id}>
                                {f.label}
                              </option>
                            ))}
                        </select>
                      </div>

                      {/* Operator dropdown */}
                      <div className="space-y-1">
                        <label className="text-[10px] uppercase tracking-wider text-muted-foreground">Bedingung</label>
                        <select
                          value={configConditionalOperator}
                          onChange={(e) => setConfigConditionalOperator(e.target.value as 'equals' | 'not_equals' | 'contains')}
                          className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                        >
                          <option value="equals">ist gleich</option>
                          <option value="not_equals">ist nicht gleich</option>
                          <option value="contains">enthaelt</option>
                        </select>
                      </div>

                      {/* Value input */}
                      <div className="space-y-1">
                        <label className="text-[10px] uppercase tracking-wider text-muted-foreground">Wert</label>
                        <input
                          type="text"
                          value={configConditionalValue}
                          onChange={(e) => setConfigConditionalValue(e.target.value)}
                          placeholder="Erwarteter Wert..."
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
                  Abbrechen
                </button>
                <button
                  onClick={saveFieldConfig}
                  className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
                >
                  Speichern
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
        title="Formulare"
        description={`${activeFormCount} aktive Formulare · ${newSubmissionCount} neue Eingaenge`}
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
            Neues Formular
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
              <p className="text-2xl font-semibold text-foreground">{activeFormCount}</p>
              <p className="text-xs text-muted-foreground">Aktive Formulare</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-border bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info-light">
              <Inbox className="h-5 w-5 text-info" />
            </div>
            <div>
              <p className="text-2xl font-semibold text-foreground">{weeklySubmissionCount}</p>
              <p className="text-xs text-muted-foreground">Eingaenge diese Woche</p>
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
              <p className="text-xs text-muted-foreground">Completion Rate</p>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'formulare' as const, label: `Meine Formulare (${activeForms.length})` },
          { key: 'eingaenge' as const, label: `Eingaenge (${submissions.length})` },
          { key: 'vorlagen' as const, label: `Vorlagen (${templates.length})` },
        ]).map((t) => (
          <button
            key={t.key}
            onClick={() => {
              setTab(t.key)
              setSearch('')
            }}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key
                ? 'border-primary text-primary font-medium tab-accent-active'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t.label}
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
                tab === 'formulare' ? 'Formular suchen...' : 'Eingang suchen...'
              }
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* 10.5 — Export button (Eingaenge tab only) */}
          {tab === 'eingaenge' && submissions.length > 0 && (
            <div className="relative">
              <button
                onClick={() => setShowExportMenu(!showExportMenu)}
                className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <Download className="h-4 w-4" />
                Exportieren
              </button>
              {showExportMenu && (
                <div className="absolute right-0 top-full mt-1 z-20 w-48 rounded-lg border border-border bg-card shadow-xl py-1">
                  <button
                    onClick={() => {
                      toast.success(`${submissions.length} Einreichungen als CSV exportiert`)
                      setShowExportMenu(false)
                    }}
                    className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
                  >
                    <Download className="h-4 w-4 text-muted-foreground" />
                    CSV exportieren
                  </button>
                  <button
                    onClick={() => {
                      toast.success(`${submissions.length} Einreichungen als Excel exportiert`)
                      setShowExportMenu(false)
                    }}
                    className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
                  >
                    <Download className="h-4 w-4 text-muted-foreground" />
                    Excel exportieren
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
              title="Keine Formulare gefunden"
              description={
                search
                  ? 'Passe deine Suche an'
                  : 'Erstelle dein erstes Formular'
              }
              action={
                !search
                  ? {
                      label: 'Neues Formular',
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
              {filteredForms.map((form) => (
                <div
                  key={form.id}
                  onClick={() => openEditor(form)}
                  className="rounded-xl border border-border bg-card p-4 transition-all duration-200 hover:shadow-md hover:-translate-y-0.5 cursor-pointer"
                >
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light shrink-0">
                        <FileText className="h-5 w-5 text-primary" />
                      </div>
                      <div className="min-w-0">
                        <h4 className="text-sm font-medium text-foreground truncate">
                          {form.name}
                        </h4>
                        <p className="text-xs text-muted-foreground truncate">
                          {form.createdBy}
                        </p>
                      </div>
                    </div>
                    <div onClick={(e) => e.stopPropagation()}>
                      <ItemActions items={getFormActions(form)} />
                    </div>
                  </div>

                  <p className="text-xs text-muted-foreground line-clamp-2 mb-3">
                    {form.description}
                  </p>

                  <div className="flex items-center gap-2 mb-3 flex-wrap">
                    <span
                      className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                        formStatusColors[form.status]
                      }`}
                    >
                      {formStatusLabels[form.status]}
                    </span>
                    {/* 10.3 — Actions badge */}
                    {(form.actions?.length ?? 0) > 0 && (
                      <span className="rounded-full bg-primary-light px-2 py-0.5 text-[10px] font-medium text-primary">
                        {form.actions!.length} {form.actions!.length === 1 ? 'Aktion' : 'Aktionen'}
                      </span>
                    )}
                    {/* 10.4 — Public badge */}
                    {form.isPublic && (
                      <span className="rounded-full bg-info-light px-2 py-0.5 text-[10px] font-medium text-info">
                        Oeffentlich
                      </span>
                    )}
                  </div>

                  <div className="flex items-center justify-between border-t border-border-muted pt-3 text-xs text-muted-foreground">
                    <span>{form.fields.filter((f) => f.label !== '__page_break__').length} Felder</span>
                    <span>{form.submissionCount} Eingaenge</span>
                    <span>{formatDate(form.createdAt)}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {/* ====================== EINGAENGE TAB ====================== */}
      {tab === 'eingaenge' && (
        <>
          {Object.keys(submissionsByForm).length === 0 ? (
            <EmptyState
              icon={Inbox}
              title="Keine Eingaenge gefunden"
              description={
                search
                  ? 'Passe deine Suche an'
                  : 'Sobald Formulare ausgefuellt werden, erscheinen die Eingaenge hier'
              }
            />
          ) : (
            <div className="space-y-4">
              {Object.entries(submissionsByForm).map(([formId, group]) => {
                const isExpanded = expandedGroups.has(formId)
                const newCount = group.items.filter((s) => s.status === 'new').length
                return (
                  <div key={formId} className="rounded-lg border border-border bg-card overflow-hidden">
                    {/* Group header */}
                    <button
                      onClick={() => toggleGroup(formId)}
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
                          {group.formName}
                        </span>
                        <span className="text-xs text-muted-foreground">
                          ({group.items.length} Eingaenge)
                        </span>
                        {newCount > 0 && (
                          <span className="rounded-full bg-info-light px-2 py-0.5 text-[10px] font-medium text-info">
                            {newCount} neu
                          </span>
                        )}
                      </div>
                    </button>

                    {/* Submissions table */}
                    {isExpanded && (
                      <div className="border-t border-border">
                        <table className="w-full text-sm">
                          <thead>
                            <tr className="border-b border-border-muted bg-secondary/30">
                              <th className="px-4 py-2 text-left font-medium text-muted-foreground">
                                Datum
                              </th>
                              <th className="px-4 py-2 text-left font-medium text-muted-foreground">
                                Absender
                              </th>
                              <th className="px-4 py-2 text-left font-medium text-muted-foreground">
                                Status
                              </th>
                              <th className="px-4 py-2 text-right font-medium text-muted-foreground" />
                            </tr>
                          </thead>
                          <tbody>
                            {group.items.map((sub) => (
                              <tr
                                key={sub.id}
                                onClick={() => setSelectedSubmission(sub)}
                                className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer"
                              >
                                <td className="px-4 py-2.5 text-foreground">
                                  {formatDateTime(sub.submittedAt)}
                                </td>
                                <td className="px-4 py-2.5 text-foreground">
                                  {sub.submittedBy}
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
                                        label: 'Details anzeigen',
                                        icon: Eye,
                                        onClick: () => setSelectedSubmission(sub),
                                      },
                                      {
                                        label: 'Als gelesen markieren',
                                        icon: Eye,
                                        onClick: () => {
                                          updateSubmissionStatus(sub.id, 'read')
                                          toast.success('Als gelesen markiert')
                                        },
                                        disabled: sub.status !== 'new',
                                      },
                                      {
                                        label: 'Archivieren',
                                        icon: Archive,
                                        onClick: () => {
                                          updateSubmissionStatus(sub.id, 'archived')
                                          toast.success('Eingang archiviert')
                                        },
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
              title="Keine Vorlagen verfuegbar"
              description="Vorlagen werden hier angezeigt, sobald welche erstellt werden."
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
                      <h4 className="text-sm font-medium text-foreground">{tmpl.name}</h4>
                      <p className="text-xs text-muted-foreground">
                        {tmpl.fields.length} Felder
                      </p>
                    </div>
                  </div>

                  <p className="text-xs text-muted-foreground line-clamp-2 mb-4">
                    {tmpl.description}
                  </p>

                  <div className="flex flex-wrap gap-1.5 mb-4">
                    {tmpl.fields.map((field) => {
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
                    Vorlage verwenden
                  </button>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {/* ====================== SUBMISSION DETAIL PANEL ====================== */}
      <DetailPanel
        open={!!selectedSubmission}
        onClose={() => setSelectedSubmission(null)}
        title={selectedSubmission?.formName ?? ''}
        subtitle={`Eingegangen am ${selectedSubmission ? formatDateTime(selectedSubmission.submittedAt) : ''}`}
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
        width="w-[440px]"
        footer={
          selectedSubmission ? (
            <div className="flex items-center gap-2">
              {selectedSubmission.status === 'new' && (
                <button
                  onClick={() => {
                    updateSubmissionStatus(selectedSubmission.id, 'read')
                    setSelectedSubmission({ ...selectedSubmission, status: 'read' })
                    toast.success('Als gelesen markiert')
                  }}
                  className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
                >
                  <Eye className="h-4 w-4" />
                  Als gelesen markieren
                </button>
              )}
              {selectedSubmission.status !== 'archived' && (
                <button
                  onClick={() => {
                    updateSubmissionStatus(selectedSubmission.id, 'archived')
                    setSelectedSubmission({ ...selectedSubmission, status: 'archived' })
                    toast.success('Eingang archiviert')
                  }}
                  className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
                >
                  Archivieren
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
                Absender
              </p>
              <p className="text-sm font-medium text-foreground">
                {selectedSubmission.submittedBy}
              </p>
            </div>

            {/* Answers */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">
                Antworten
              </h4>
              <div className="rounded-lg border border-border divide-y divide-border-muted">
                {Object.entries(selectedSubmission.answers).map(([fieldId, value]) => {
                  const field = getFieldForSubmission(
                    selectedSubmission.formId,
                    fieldId
                  )
                  // 10.1 — Skip fields hidden by conditional logic
                  if (field?.conditionalLogic) {
                    const visible = evaluateCondition(
                      field.conditionalLogic,
                      selectedSubmission.answers
                    )
                    if (!visible) return null
                  }
                  return (
                    <div key={fieldId} className="px-3 py-3">
                      <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5">
                        {field?.label ?? fieldId}
                      </p>
                      {renderAnswer(fieldId, value, selectedSubmission.formId)}
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        )}
      </DetailPanel>

      {/* ====================== NEUES FORMULAR DIALOG ====================== */}
      <Dialog open={showNewFormDialog} onOpenChange={(o) => { if (!o) setShowNewFormDialog(false) }}>
        <DialogContent className="gap-0 p-0 max-w-md">
            <DialogHeader className="border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <FileText className="h-5 w-5 text-primary" />
                <DialogTitle className="text-base font-semibold text-foreground">Neues Formular</DialogTitle>
              </div>
              <DialogDescription className="sr-only">Neues Formular erstellen</DialogDescription>
            </DialogHeader>

            <div className="p-5 space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  Name <span className="text-destructive">*</span>
                </label>
                <input
                  type="text"
                  value={newFormName}
                  onChange={(e) => setNewFormName(e.target.value)}
                  placeholder="z.B. Kundenfeedback"
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">Beschreibung</label>
                <textarea
                  value={newFormDescription}
                  onChange={(e) => setNewFormDescription(e.target.value)}
                  rows={3}
                  placeholder="Wofuer wird dieses Formular verwendet?"
                  className="w-full resize-none rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  Von Vorlage erstellen{' '}
                  <span className="text-xs text-muted-foreground">(optional)</span>
                </label>
                <select
                  value={newFormTemplateId}
                  onChange={(e) => setNewFormTemplateId(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  <option value="">Leer beginnen</option>
                  {templates.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.name} ({t.fields.length} Felder)
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
                Abbrechen
              </button>
              <button
                onClick={handleCreateForm}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                Formular erstellen
              </button>
            </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ====================== SHARE DIALOG ====================== */}
      <Dialog open={!!showShareDialog} onOpenChange={(o) => { if (!o) setShowShareDialog(null) }}>
        <DialogContent className="gap-0 p-0 max-w-sm">
            <DialogHeader className="border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <Share2 className="h-5 w-5 text-primary" />
                <DialogTitle className="text-base font-semibold text-foreground">Formular teilen</DialogTitle>
              </div>
              <DialogDescription className="sr-only">Formular per Link oder E-Mail teilen</DialogDescription>
            </DialogHeader>

            <div className="p-5 space-y-4">
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                  Formular
                </p>
                <p className="text-sm font-medium text-foreground">{showShareDialog.name}</p>
              </div>

              <div className="space-y-2">
                <button
                  onClick={() => {
                    toast.success('Link kopiert!')
                    setShowShareDialog(null)
                  }}
                  className="flex w-full items-center gap-3 rounded-lg border border-border p-3 hover:bg-secondary/50 transition-colors"
                >
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
                    <Link2 className="h-4 w-4 text-primary" />
                  </div>
                  <div className="text-left">
                    <p className="text-sm font-medium text-foreground">Link kopieren</p>
                    <p className="text-xs text-muted-foreground">Freigabe-Link in die Zwischenablage</p>
                  </div>
                </button>

                <button
                  onClick={() => {
                    toast.success('E-Mail-Einladung wird gesendet...')
                    setShowShareDialog(null)
                  }}
                  className="flex w-full items-center gap-3 rounded-lg border border-border p-3 hover:bg-secondary/50 transition-colors"
                >
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-info-light">
                    <Mail className="h-4 w-4 text-info" />
                  </div>
                  <div className="text-left">
                    <p className="text-sm font-medium text-foreground">Per E-Mail senden</p>
                    <p className="text-xs text-muted-foreground">Einladungslink per E-Mail verschicken</p>
                  </div>
                </button>
              </div>
            </div>

            <DialogFooter className="border-t border-border px-5 py-4">
              <button
                onClick={() => setShowShareDialog(null)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                Schliessen
              </button>
            </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ====================== CONFIRM DELETE ====================== */}
      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title="Formular loeschen?"
        description={`"${confirmDelete?.name}" und alle zugehoerigen Eingaenge werden unwiderruflich geloescht.`}
        confirmLabel="Loeschen"
        variant="destructive"
        onConfirm={() => confirmDelete && handleDeleteForm(confirmDelete)}
      />
    </div>
  )
}
