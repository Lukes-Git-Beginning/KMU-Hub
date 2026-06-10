/**
 * TemplateGalleryDialog — browse and create documents from templates.
 *
 * 5 categories with ~12 mock templates. Search within templates.
 * "Erstellen" uploads a generated placeholder file into the current folder
 * (real template copies need backend template storage — see backend-gaps).
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  FileText,
  Presentation,
  Search,
  Plus,
  FileCheck,
  Receipt,
  BarChart3,
  Mail,
  ScrollText,
  ClipboardList,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { useDocumentUpload } from '@/api/hooks/useDocumentUpload'

// `as const` keeps the keys as literal types for the typed t().
const CATEGORY_KEYS = {
  all: 'dokumente.template.categoryAll',
  'verträge': 'dokumente.template.categoryContracts',
  briefe: 'dokumente.template.categoryLetters',
  formulare: 'dokumente.template.categoryForms',
  rechnungen: 'dokumente.template.categoryInvoices',
  berichte: 'dokumente.template.categoryReports',
} as const

const CATEGORIES = [
  { id: 'all' },
  { id: 'verträge' },
  { id: 'briefe' },
  { id: 'formulare' },
  { id: 'rechnungen' },
  { id: 'berichte' },
] as const satisfies readonly { id: keyof typeof CATEGORY_KEYS }[]

const FORMAT_MIME: Record<string, string> = {
  '.docx': 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  '.xlsx': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  '.pptx': 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
}

const TEMPLATES = [
  // Verträge
  { id: 't1', nameKey: 'dokumente.template.t1Name', descKey: 'dokumente.template.t1Desc', category: 'verträge', icon: FileCheck, format: '.docx' },
  { id: 't2', nameKey: 'dokumente.template.t2Name', descKey: 'dokumente.template.t2Desc', category: 'verträge', icon: FileCheck, format: '.docx' },
  { id: 't3', nameKey: 'dokumente.template.t3Name', descKey: 'dokumente.template.t3Desc', category: 'verträge', icon: ScrollText, format: '.docx' },
  // Briefe
  { id: 't4', nameKey: 'dokumente.template.t4Name', descKey: 'dokumente.template.t4Desc', category: 'briefe', icon: Mail, format: '.docx' },
  { id: 't5', nameKey: 'dokumente.template.t5Name', descKey: 'dokumente.template.t5Desc', category: 'briefe', icon: FileText, format: '.docx' },
  // Formulare
  { id: 't6', nameKey: 'dokumente.template.t6Name', descKey: 'dokumente.template.t6Desc', category: 'formulare', icon: ClipboardList, format: '.docx' },
  { id: 't7', nameKey: 'dokumente.template.t7Name', descKey: 'dokumente.template.t7Desc', category: 'formulare', icon: ClipboardList, format: '.xlsx' },
  { id: 't8', nameKey: 'dokumente.template.t8Name', descKey: 'dokumente.template.t8Desc', category: 'formulare', icon: ClipboardList, format: '.xlsx' },
  // Rechnungen
  { id: 't9', nameKey: 'dokumente.template.t9Name', descKey: 'dokumente.template.t9Desc', category: 'rechnungen', icon: Receipt, format: '.xlsx' },
  { id: 't10', nameKey: 'dokumente.template.t10Name', descKey: 'dokumente.template.t10Desc', category: 'rechnungen', icon: Receipt, format: '.xlsx' },
  // Berichte
  { id: 't11', nameKey: 'dokumente.template.t11Name', descKey: 'dokumente.template.t11Desc', category: 'berichte', icon: BarChart3, format: '.docx' },
  { id: 't12', nameKey: 'dokumente.template.t12Name', descKey: 'dokumente.template.t12Desc', category: 'berichte', icon: Presentation, format: '.pptx' },
] as const

type Template = (typeof TEMPLATES)[number]

interface TemplateGalleryDialogProps {
  open: boolean
  onClose: () => void
  /** Folder the created document lands in ('' = root, same as page uploads). */
  targetFolderId?: string | null
}

export function TemplateGalleryDialog({ open, onClose, targetFolderId }: TemplateGalleryDialogProps) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [activeCategory, setActiveCategory] = useState<(typeof CATEGORIES)[number]['id']>('all')
  const [creatingId, setCreatingId] = useState<string | null>(null)
  const upload = useDocumentUpload()

  const filtered = useMemo(() => {
    let result: readonly Template[] = TEMPLATES
    if (activeCategory !== 'all') {
      result = result.filter((tmpl) => tmpl.category === activeCategory)
    }
    if (search) {
      const q = search.toLowerCase()
      result = result.filter(
        (tmpl) => t(tmpl.nameKey).toLowerCase().includes(q) || t(tmpl.descKey).toLowerCase().includes(q),
      )
    }
    return result
  }, [activeCategory, search, t])

  // Creates a real document: generated placeholder content uploaded into the
  // current folder (works in demo and against the real upload endpoint).
  const handleCreate = (template: Template) => {
    if (creatingId) return
    const filename = `${t(template.nameKey)}${template.format}`
    const content = `${t(template.nameKey)}\n\n${t(template.descKey)}\n\n${t('dokumente.template.placeholderBody')}\n`
    const file = new File([content], filename, {
      type: FORMAT_MIME[template.format] ?? 'application/octet-stream',
    })
    setCreatingId(template.id)
    upload
      .mutateAsync({ folderId: targetFolderId ?? '', file })
      .then(() => {
        toast.success(t('dokumente.template.created', { name: filename }))
        onClose()
      })
      .catch((err: Error) => {
        toast.error(`${t('common.error')}: ${err.message}`)
      })
      .finally(() => setCreatingId(null))
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-3xl max-h-[85vh] flex flex-col">
        {/* Header */}
        <DialogHeader className="border-b border-border px-6 py-4">
          <DialogTitle className="text-sm font-semibold text-foreground">{t('dokumente.template.title')}</DialogTitle>
          <DialogDescription className="text-xs text-muted-foreground">{t('dokumente.template.available', { count: TEMPLATES.length })}</DialogDescription>
        </DialogHeader>

        {/* Search + Category filter */}
        <div className="border-b border-border px-6 py-3 space-y-3">
          <div className="relative max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder={t('dokumente.template.searchPlaceholder')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
            />
          </div>
          <div className="flex gap-1 flex-wrap">
            {CATEGORIES.map((cat) => (
              <button
                key={cat.id}
                onClick={() => setActiveCategory(cat.id)}
                className={`rounded-lg px-3 py-1 text-xs font-medium transition-colors ${
                  activeCategory === cat.id
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-muted'
                }`}
              >
                {t(CATEGORY_KEYS[cat.id])}
              </button>
            ))}
          </div>
        </div>

        {/* Template grid */}
        <div className="flex-1 overflow-y-auto p-6">
          {filtered.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <FileText className="h-10 w-10 text-muted-foreground/40 mb-3" />
              <p className="text-sm text-muted-foreground">{t('dokumente.template.noResults')}</p>
            </div>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {filtered.map((template) => (
                <div
                  key={template.id}
                  className={`group rounded-lg border border-border bg-card p-4 hover:border-primary/30 hover:shadow-md transition-all cursor-pointer ${
                    creatingId ? 'pointer-events-none opacity-60' : ''
                  }`}
                  onClick={() => handleCreate(template)}
                >
                  <div className="flex items-start gap-3 mb-2">
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10">
                      <template.icon className="h-4.5 w-4.5 text-primary" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <h3 className="text-sm font-medium text-foreground truncate">{t(template.nameKey)}</h3>
                      <span className="text-[10px] text-muted-foreground">{template.format}</span>
                    </div>
                  </div>
                  <p className="text-xs text-muted-foreground line-clamp-2">{t(template.descKey)}</p>
                  <div className="mt-3 flex justify-end opacity-0 group-hover:opacity-100 transition-opacity">
                    <span className="flex items-center gap-1 text-xs text-primary font-medium">
                      <Plus className="h-3.5 w-3.5" />
                      {t('common.create')}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
