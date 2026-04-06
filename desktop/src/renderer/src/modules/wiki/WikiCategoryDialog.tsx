import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Building2, GitBranch, Monitor, Users, BookOpen } from 'lucide-react'
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
// Icon options
// ---------------------------------------------------------------------------

const iconOptionKeys = [
  { id: 'Building2', icon: Building2, labelKey: 'wiki.category.iconCompany' },
  { id: 'GitBranch', icon: GitBranch, labelKey: 'wiki.category.iconProcesses' },
  { id: 'Monitor', icon: Monitor, labelKey: 'wiki.category.iconIT' },
  { id: 'Users', icon: Users, labelKey: 'wiki.category.iconHR' },
  { id: 'BookOpen', icon: BookOpen, labelKey: 'wiki.category.iconGeneral' },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiCategoryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function WikiCategoryDialog({ open, onOpenChange }: WikiCategoryDialogProps) {
  const { t } = useTranslation()
  const addCategory = useWikiStore((s) => s.addCategory)
  const categories = useWikiStore((s) => s.categories)

  const [name, setName] = useState('')
  const [selectedIcon, setSelectedIcon] = useState('BookOpen')

  const handleCreate = () => {
    if (!name.trim()) return
    addCategory({
      name: name.trim(),
      icon: selectedIcon,
      sortOrder: categories.length + 1,
    })
    toast.success(t('wiki.category.created'))
    onOpenChange(false)
    setName('')
    setSelectedIcon('BookOpen')
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('wiki.category.newTitle')}</DialogTitle>
          <DialogDescription>
            {t('wiki.category.newDescription')}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Name */}
          <div>
            <label className="mb-1 block text-sm font-medium text-foreground">{t('wiki.category.nameLabel')}</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('wiki.category.namePlaceholder')}
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              autoFocus
              onKeyDown={(e) => { if (e.key === 'Enter') handleCreate() }}
            />
          </div>

          {/* Icon */}
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">{t('wiki.category.iconLabel')}</label>
            <div className="flex gap-2">
              {iconOptionKeys.map((opt) => {
                const Icon = opt.icon
                const isActive = selectedIcon === opt.id
                return (
                  <button
                    key={opt.id}
                    onClick={() => setSelectedIcon(opt.id)}
                    className={`flex flex-col items-center gap-1 rounded-md border px-3 py-2 transition-colors ${
                      isActive
                        ? 'border-primary bg-primary/5 text-primary'
                        : 'border-border text-muted-foreground hover:bg-accent'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    <span className="text-[10px]">{t(opt.labelKey)}</span>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2 pt-1">
            <button
              onClick={() => { onOpenChange(false); setName('') }}
              className="h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleCreate}
              disabled={!name.trim()}
              className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {t('common.create')}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
