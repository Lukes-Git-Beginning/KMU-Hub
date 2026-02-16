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
  X,
  Copy,
  Share2,
  Archive,
  ArrowLeft,
  GripVertical,
  ChevronDown,
  ChevronRight,
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
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useFormulareStore,
  type Form,
  type FormField,
  type FormFieldType,
  type FormSubmission,
} from '@/stores/formulare'
import { ItemActions, ConfirmDialog, EmptyState, DetailPanel } from '@/components/shared'

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

  // Editor
  const openEditor = (form: Form) => {
    setEditingFormId(form.id)
    setEditName(form.name)
    setEditDescription(form.description)
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
  // Render helpers
  // ---------------------------------------------------------------------------

  const renderStars = (value: number, max = 5) => (
    <div className="flex items-center gap-0.5">
      {Array.from({ length: max }, (_, i) => (
        <Star
          key={i}
          className={`h-4 w-4 ${
            i < value ? 'fill-amber-400 text-amber-400' : 'text-muted-foreground/30'
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
                Felder ({editingForm.fields.length})
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
                {editingForm.fields.map((field, idx) => {
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
              {editingForm.fields.length === 0 ? (
                <p className="text-sm text-muted-foreground italic">
                  Fuege Felder hinzu, um eine Vorschau zu sehen.
                </p>
              ) : (
                editingForm.fields.map((field) => (
                  <div key={field.id} className="space-y-1.5">
                    <label className="text-sm font-medium text-foreground">
                      {field.label}
                      {field.required && <span className="text-red-500 ml-0.5">*</span>}
                    </label>
                    {renderFieldPreview(field)}
                  </div>
                ))
              )}
              {editingForm.fields.length > 0 && (
                <button
                  disabled
                  className="mt-2 rounded-lg bg-primary/50 px-4 py-2 text-sm text-primary-foreground cursor-not-allowed"
                >
                  Absenden
                </button>
              )}
            </div>
          </div>
        </div>

        {/* ====================== FIELD CONFIG DIALOG ====================== */}
        {showFieldConfigDialog && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
            <div className="w-full max-w-md rounded-xl bg-card border border-border shadow-xl mx-4">
              <div className="flex items-center justify-between border-b border-border px-5 py-4">
                <div className="flex items-center gap-2">
                  {renderFieldIcon(showFieldConfigDialog.field.type, 'h-5 w-5 text-primary')}
                  <h2 className="text-base font-semibold text-foreground">Feld konfigurieren</h2>
                </div>
                <button
                  onClick={() => setShowFieldConfigDialog(null)}
                  className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>

              <div className="p-5 space-y-4">
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">
                    Bezeichnung <span className="text-red-500">*</span>
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
              </div>

              <div className="flex items-center justify-end gap-2 border-t border-border px-5 py-4">
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
              </div>
            </div>
          </div>
        )}
      </div>
    )
  }

  // ---------------------------------------------------------------------------
  // JSX — Main View
  // ---------------------------------------------------------------------------

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div>
          <h1 className="text-foreground">Formulare</h1>
          <p className="text-sm text-muted-foreground">
            {activeFormCount} aktive Formulare · {newSubmissionCount} neue Eingaenge
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              resetNewFormDialog()
              setShowNewFormDialog(true)
            }}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            Neues Formular
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
        <div className="rounded-lg border border-border bg-card p-4">
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
        <div className="rounded-lg border border-border bg-card p-4">
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
        <div className="rounded-lg border border-border bg-card p-4">
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
                ? 'border-primary text-primary font-medium'
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
                  className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer"
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

                  <div className="flex items-center gap-2 mb-3">
                    <span
                      className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                        formStatusColors[form.status]
                      }`}
                    >
                      {formStatusLabels[form.status]}
                    </span>
                  </div>

                  <div className="flex items-center justify-between border-t border-border-muted pt-3 text-xs text-muted-foreground">
                    <span>{form.fields.length} Felder</span>
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
      {showNewFormDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-xl bg-card border border-border shadow-xl mx-4">
            <div className="flex items-center justify-between border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <FileText className="h-5 w-5 text-primary" />
                <h2 className="text-base font-semibold text-foreground">Neues Formular</h2>
              </div>
              <button
                onClick={() => setShowNewFormDialog(false)}
                className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="p-5 space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  Name <span className="text-red-500">*</span>
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

            <div className="flex items-center justify-end gap-2 border-t border-border px-5 py-4">
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
            </div>
          </div>
        </div>
      )}

      {/* ====================== SHARE DIALOG ====================== */}
      {showShareDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-sm rounded-xl bg-card border border-border shadow-xl mx-4">
            <div className="flex items-center justify-between border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <Share2 className="h-5 w-5 text-primary" />
                <h2 className="text-base font-semibold text-foreground">Formular teilen</h2>
              </div>
              <button
                onClick={() => setShowShareDialog(null)}
                className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

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

            <div className="flex items-center justify-end border-t border-border px-5 py-4">
              <button
                onClick={() => setShowShareDialog(null)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                Schliessen
              </button>
            </div>
          </div>
        </div>
      )}

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
