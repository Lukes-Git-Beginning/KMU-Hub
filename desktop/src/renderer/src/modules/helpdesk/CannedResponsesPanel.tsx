/**
 * 5.6 — Canned Responses Library
 *
 * Slide-over panel for managing reusable text responses.
 * CRUD operations, TipTap editor for content, search & category filtering.
 */
import { useState, useMemo } from 'react'
import {
  X,
  Search,
  Plus,
  Pencil,
  Trash2,
  Copy,
  ChevronDown,
  Zap,
  ArrowLeft,
} from 'lucide-react'
import { toast } from 'sonner'
import { LazyRichTextEditor as RichTextEditor } from '@/components/shared/RichTextEditor'
import { useHelpdeskStore, type CannedResponse, MOCK_CATEGORIES } from '@/stores/helpdesk'

// ---------------------------------------------------------------------------
// Category list for canned responses (includes "Allgemein" beyond ticket cats)
// ---------------------------------------------------------------------------

const CANNED_CATEGORIES = ['Allgemein', ...MOCK_CATEGORIES] as const

interface CannedResponsesPanelProps {
  open: boolean
  onClose: () => void
  onInsert?: (content: string) => void
}

export function CannedResponsesPanel({ open, onClose, onInsert }: CannedResponsesPanelProps) {
  const { cannedResponses } = useHelpdeskStore()
  const [search, setSearch] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<string>('all')

  // Edit/Create mode
  const [editing, setEditing] = useState<CannedResponse | null>(null)
  const [creating, setCreating] = useState(false)
  const [formTitle, setFormTitle] = useState('')
  const [formContent, setFormContent] = useState('')
  const [formCategory, setFormCategory] = useState<string>('Allgemein')
  const [formShortcut, setFormShortcut] = useState('')

  const filtered = useMemo(() => {
    return cannedResponses.filter((r) => {
      if (categoryFilter !== 'all' && r.category !== categoryFilter) return false
      if (search) {
        const q = search.toLowerCase()
        return (
          r.title.toLowerCase().includes(q) ||
          r.shortcut.toLowerCase().includes(q) ||
          r.category.toLowerCase().includes(q)
        )
      }
      return true
    })
  }, [cannedResponses, search, categoryFilter])

  const handleStartCreate = () => {
    setFormTitle('')
    setFormContent('')
    setFormCategory('Allgemein')
    setFormShortcut('/')
    setCreating(true)
    setEditing(null)
  }

  const handleStartEdit = (r: CannedResponse) => {
    setFormTitle(r.title)
    setFormContent(r.content)
    setFormCategory(r.category)
    setFormShortcut(r.shortcut)
    setEditing(r)
    setCreating(false)
  }

  const handleSave = () => {
    if (!formTitle.trim()) {
      toast.error('Bitte Titel eingeben')
      return
    }
    if (editing) {
      toast.success(`Textbaustein "${formTitle}" aktualisiert`)
    } else {
      toast.success(`Textbaustein "${formTitle}" erstellt`)
    }
    setEditing(null)
    setCreating(false)
  }

  const handleDelete = (r: CannedResponse) => {
    toast.success(`Textbaustein "${r.title}" gelöscht`)
  }

  const handleInsert = (r: CannedResponse) => {
    onInsert?.(r.content)
    toast.success(`"${r.title}" eingefuegt`)
    onClose()
  }

  const handleCancelForm = () => {
    setEditing(null)
    setCreating(false)
  }

  if (!open) return null

  const isFormMode = editing !== null || creating

  return (
    <div className="fixed inset-y-0 right-0 z-50 w-[480px] max-w-full border-l border-border bg-card shadow-xl flex flex-col overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-5 py-4 shrink-0">
        <div className="flex items-center gap-2">
          {isFormMode && (
            <button onClick={handleCancelForm} className="rounded-lg p-1 text-muted-foreground hover:bg-secondary transition-colors">
              <ArrowLeft className="h-4 w-4" />
            </button>
          )}
          <div>
            <h3 className="text-sm font-semibold text-foreground">
              {creating ? 'Neuer Textbaustein' : editing ? 'Textbaustein bearbeiten' : 'Textbausteine'}
            </h3>
            {!isFormMode && (
              <p className="text-xs text-muted-foreground">{cannedResponses.length} Vorlagen verfügbar</p>
            )}
          </div>
        </div>
        <button onClick={onClose} className="rounded-lg p-1.5 text-muted-foreground hover:bg-secondary transition-colors">
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* ---------- Form Mode ---------- */}
      {isFormMode && (
        <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
          {/* Title */}
          <div>
            <label className="mb-1.5 block text-xs font-medium text-foreground">Titel</label>
            <input
              type="text"
              value={formTitle}
              onChange={(e) => setFormTitle(e.target.value)}
              placeholder="z.B. Begrüßung"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Category */}
          <div>
            <label className="mb-1.5 block text-xs font-medium text-foreground">Kategorie</label>
            <div className="relative">
              <select
                value={formCategory}
                onChange={(e) => setFormCategory(e.target.value)}
                className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
              >
                {CANNED_CATEGORIES.map((c) => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </select>
              <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            </div>
          </div>

          {/* Shortcut */}
          <div>
            <label className="mb-1.5 block text-xs font-medium text-foreground">Kuerzel</label>
            <input
              type="text"
              value={formShortcut}
              onChange={(e) => setFormShortcut(e.target.value)}
              placeholder="/gruss"
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground font-mono placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
            <p className="text-[10px] text-muted-foreground mt-1">Tipp: Mit "/" beginnen, z.B. /gruss, /vpn</p>
          </div>

          {/* Content (TipTap) */}
          <div>
            <label className="mb-1.5 block text-xs font-medium text-foreground">Inhalt</label>
            <RichTextEditor
              content={formContent}
              onChange={(html) => setFormContent(html)}
              placeholder="Vorlagentext schreiben..."
              showFooter={false}
              compact
              minHeight="120px"
              maxHeight="300px"
            />
          </div>

          {/* Save / Cancel */}
          <div className="flex items-center justify-end gap-3 pt-2">
            <button
              onClick={handleCancelForm}
              className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={handleSave}
              className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              {editing ? 'Aktualisieren' : 'Erstellen'}
            </button>
          </div>
        </div>
      )}

      {/* ---------- List Mode ---------- */}
      {!isFormMode && (
        <>
          {/* Search + filter + add */}
          <div className="px-5 pt-4 pb-2 space-y-3 shrink-0">
            <div className="flex gap-2">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <input
                  type="text"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Suchen oder Kuerzel eingeben..."
                  className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
              <button
                onClick={handleStartCreate}
                className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors shrink-0"
              >
                <Plus className="h-4 w-4" />
                Neu
              </button>
            </div>
            <div className="relative">
              <select
                value={categoryFilter}
                onChange={(e) => setCategoryFilter(e.target.value)}
                className="w-full appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
              >
                <option value="all">Alle Kategorien</option>
                {CANNED_CATEGORIES.map((c) => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </select>
              <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            </div>
          </div>

          {/* Response list */}
          <div className="flex-1 overflow-y-auto px-5 pb-4 space-y-2">
            {filtered.length === 0 && (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <Zap className="h-8 w-8 mb-2 opacity-40" />
                <p className="text-sm font-medium">Keine Textbausteine gefunden</p>
              </div>
            )}
            {filtered.map((r) => (
              <div
                key={r.id}
                className="rounded-lg border border-border bg-card p-3 hover:shadow-[var(--shadow-card-hover)] transition-shadow group"
              >
                <div className="flex items-start justify-between mb-1.5">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <h4 className="text-sm font-medium text-foreground truncate">{r.title}</h4>
                      <span className="rounded bg-secondary px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground shrink-0">
                        {r.shortcut}
                      </span>
                    </div>
                    <span className="text-[10px] text-muted-foreground">{r.category}</span>
                  </div>
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity shrink-0 ml-2">
                    <button
                      onClick={() => handleStartEdit(r)}
                      className="rounded p-1 text-muted-foreground hover:bg-secondary transition-colors"
                      title="Bearbeiten"
                    >
                      <Pencil className="h-3.5 w-3.5" />
                    </button>
                    <button
                      onClick={() => handleDelete(r)}
                      className="rounded p-1 text-muted-foreground hover:bg-error-light hover:text-error transition-colors"
                      title="Löschen"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
                {/* Preview content (stripped HTML) */}
                <p className="text-xs text-muted-foreground line-clamp-2 mb-2">
                  {r.content.replace(/<[^>]*>/g, '').slice(0, 120)}
                </p>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleInsert(r)}
                    className="flex items-center gap-1 rounded-lg bg-primary/10 px-2.5 py-1 text-xs text-primary font-medium hover:bg-primary/20 transition-colors"
                  >
                    <Copy className="h-3 w-3" />
                    Einfügen
                  </button>
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}
