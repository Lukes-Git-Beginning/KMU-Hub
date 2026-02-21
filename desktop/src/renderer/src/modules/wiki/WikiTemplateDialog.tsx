import { useState } from 'react'
import { FileText, AlertTriangle, BookOpen } from 'lucide-react'
import { toast } from 'sonner'
import { useWikiStore } from '@/stores/wiki'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Icon map for templates
// ---------------------------------------------------------------------------

const templateIconMap: Record<string, typeof FileText> = {
  FileText,
  AlertTriangle,
  BookOpen,
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiTemplateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function WikiTemplateDialog({ open, onOpenChange }: WikiTemplateDialogProps) {
  const templates = useWikiStore((s) => s.templates)
  const addArticle = useWikiStore((s) => s.addArticle)
  const selectedCategoryId = useWikiStore((s) => s.selectedCategoryId)

  const [title, setTitle] = useState('')
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null)

  const handleCreate = () => {
    if (!title.trim()) return
    const template = selectedTemplate ? templates.find((t) => t.id === selectedTemplate) : null
    addArticle({
      title: title.trim(),
      content: template?.content ?? '<p>Neuer Artikel — hier Inhalt hinzufuegen.</p>',
      categoryId: selectedCategoryId ?? 'wc1',
      status: 'draft',
      authorId: 'c1',
      authorName: 'Anna Mueller',
      tags: [],
      isPinned: false,
      lastEditedBy: 'Anna Mueller',
      lastEditedAt: new Date().toISOString().split('T')[0],
    })
    toast.success('Artikel erstellt')
    onOpenChange(false)
    setTitle('')
    setSelectedTemplate(null)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Neuer Wiki-Artikel</DialogTitle>
          <DialogDescription>
            Erstelle einen neuen Artikel aus einer Vorlage oder von Grund auf.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Title */}
          <div>
            <label className="mb-1 block text-sm font-medium text-foreground">Titel</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="z.B. Einrichtung Arbeitsplatz"
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              autoFocus
              onKeyDown={(e) => { if (e.key === 'Enter') handleCreate() }}
            />
          </div>

          {/* Template selection */}
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">
              Vorlage <span className="text-muted-foreground font-normal">(optional)</span>
            </label>
            <div className="grid grid-cols-3 gap-2">
              {templates.map((t) => {
                const TIcon = templateIconMap[t.icon] ?? FileText
                const isActive = selectedTemplate === t.id
                return (
                  <button
                    key={t.id}
                    onClick={() => setSelectedTemplate(isActive ? null : t.id)}
                    className={`flex flex-col items-center gap-1.5 rounded-lg border px-3 py-3 text-center transition-colors ${
                      isActive
                        ? 'border-primary bg-primary/5 text-primary'
                        : 'border-border text-muted-foreground hover:bg-accent'
                    }`}
                  >
                    <TIcon className="h-5 w-5" />
                    <span className="text-xs font-medium">{t.name}</span>
                    <span className="text-[10px] leading-tight">{t.description}</span>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2 pt-1">
            <button
              onClick={() => { onOpenChange(false); setTitle(''); setSelectedTemplate(null) }}
              className="h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={handleCreate}
              disabled={!title.trim()}
              className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              Erstellen
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
