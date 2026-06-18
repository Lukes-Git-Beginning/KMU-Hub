import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Pencil, Trash2, Check, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/cn'
import {
  useTimeCategories,
  useCreateTimeCategory,
  useUpdateTimeCategory,
  useDeleteTimeCategory,
  useTimeTemplates,
  useCreateTimeTemplate,
  useDeleteTimeTemplate,
  useHRSettings,
} from '@/api/hooks/hr-hooks'
import type { TimeCategory } from '@/api/hr-types'

const COLOR_PRESETS = [
  '#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b',
  '#10b981', '#ef4444', '#6b7280', '#94a3b8',
  '#14b8a6', '#f97316', '#06b6d4', '#84cc16',
]

export default function CategoriesView() {
  const { t } = useTranslation()

  // Categories
  const { data: categories = [] } = useTimeCategories()
  const createCategory = useCreateTimeCategory()
  const updateCategory = useUpdateTimeCategory()
  const deleteCategory = useDeleteTimeCategory()

  // Templates
  const { data: templates = [] } = useTimeTemplates()
  const createTemplate = useCreateTimeTemplate()
  const deleteTemplate = useDeleteTimeTemplate()

  // HR Settings for work targets (read-only display; editing via HR-Settings module)
  const { data: hrSettings } = useHRSettings()

  // Category form state
  const [showAddCat, setShowAddCat] = useState(false)
  const [newCatName, setNewCatName] = useState('')
  const [newCatColor, setNewCatColor] = useState(COLOR_PRESETS[0])
  const [editingCat, setEditingCat] = useState<string | null>(null)
  const [editName, setEditName] = useState('')

  // Template form state
  const [showAddTpl, setShowAddTpl] = useState(false)
  const [newTplName, setNewTplName] = useState('')
  const [newTplCatId, setNewTplCatId] = useState('')
  const [newTplDesc, setNewTplDesc] = useState('')
  const [newTplMins, setNewTplMins] = useState('30')

  const handleAddCategory = () => {
    if (!newCatName.trim()) return
    createCategory.mutate(
      { name: newCatName.trim(), color: newCatColor, icon: 'Tag', is_default: false },
      {
        onSuccess: () => {
          setNewCatName('')
          setShowAddCat(false)
        },
      },
    )
  }

  const handleSaveEdit = (id: string) => {
    if (!editName.trim()) return
    updateCategory.mutate(
      { id, data: { name: editName.trim() } },
      { onSuccess: () => setEditingCat(null) },
    )
  }

  const handleAddTemplate = () => {
    if (!newTplName.trim() || !newTplCatId) return
    createTemplate.mutate(
      {
        name: newTplName.trim(),
        category_id: newTplCatId,
        description: newTplDesc,
        estimated_minutes: parseInt(newTplMins) || 30,
      },
      {
        onSuccess: () => {
          setNewTplName('')
          setNewTplDesc('')
          setShowAddTpl(false)
        },
      },
    )
  }

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-8">
      {/* Categories */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-foreground">
            {t('profil.zeiterfassung.categories.title')} ({categories.length})
          </h3>
          <Button size="sm" variant="outline" onClick={() => setShowAddCat(true)} className="gap-2">
            <Plus className="h-3.5 w-3.5" />
            {t('profil.zeiterfassung.categories.newCategory')}
          </Button>
        </div>

        <div className="space-y-2">
          {(categories as TimeCategory[]).map((cat) => (
            <div
              key={cat.id}
              className="flex items-center gap-3 p-3 rounded-lg border border-border bg-card group"
            >
              <span className="h-4 w-4 rounded-full shrink-0" style={{ backgroundColor: cat.color }} />

              {editingCat === cat.id ? (
                <div className="flex items-center gap-2 flex-1">
                  <Input
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                    className="h-8 text-sm"
                    autoFocus
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleSaveEdit(cat.id)
                      if (e.key === 'Escape') setEditingCat(null)
                    }}
                  />
                  <button
                    onClick={() => handleSaveEdit(cat.id)}
                    className="p-1 text-success hover:bg-success/10 rounded"
                  >
                    <Check className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => setEditingCat(null)}
                    className="p-1 text-muted-foreground hover:bg-accent rounded"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ) : (
                <>
                  <span className="text-sm font-medium text-foreground flex-1">{cat.name}</span>
                  {cat.is_default && (
                    <span className="text-[10px] text-muted-foreground bg-secondary px-1.5 py-0.5 rounded">
                      {t('profil.zeiterfassung.categories.default')}
                    </span>
                  )}
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={() => { setEditingCat(cat.id); setEditName(cat.name) }}
                      className="p-1 rounded hover:bg-accent text-muted-foreground"
                    >
                      <Pencil className="h-3.5 w-3.5" />
                    </button>
                    {!cat.is_default && (
                      <button
                        onClick={() => deleteCategory.mutate(cat.id)}
                        className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </div>
                </>
              )}
            </div>
          ))}

          {/* Add Category Form */}
          {showAddCat && (
            <div className="p-4 rounded-lg border-2 border-dashed border-primary/30 bg-primary/5 space-y-3">
              <Input
                placeholder={t('profil.zeiterfassung.categories.categoryNamePlaceholder')}
                value={newCatName}
                onChange={(e) => setNewCatName(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleAddCategory()}
                autoFocus
              />
              <div className="flex flex-wrap gap-2">
                {COLOR_PRESETS.map((color) => (
                  <button
                    key={color}
                    onClick={() => setNewCatColor(color)}
                    className={cn(
                      'h-6 w-6 rounded-full border-2 transition-all',
                      newCatColor === color ? 'border-foreground scale-110' : 'border-transparent',
                    )}
                    style={{ backgroundColor: color }}
                  />
                ))}
              </div>
              <div className="flex justify-end gap-2">
                <Button size="sm" variant="outline" onClick={() => setShowAddCat(false)}>
                  {t('common.cancel')}
                </Button>
                <Button
                  size="sm"
                  onClick={handleAddCategory}
                  disabled={createCategory.isPending}
                >
                  {t('profil.zeiterfassung.categories.add')}
                </Button>
              </div>
            </div>
          )}
        </div>
      </section>

      {/* Templates */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-foreground">
            {t('profil.zeiterfassung.categories.templates')} ({templates.length})
          </h3>
          <Button
            size="sm"
            variant="outline"
            onClick={() => {
              setShowAddTpl(true)
              setNewTplCatId(categories[0]?.id || '')
            }}
            className="gap-2"
          >
            <Plus className="h-3.5 w-3.5" />
            {t('profil.zeiterfassung.categories.newTemplate')}
          </Button>
        </div>

        <div className="space-y-2">
          {templates.map((tpl) => {
            const cat = (categories as TimeCategory[]).find((c) => c.id === tpl.category_id)
            return (
              <div
                key={tpl.id}
                className="flex items-center gap-3 p-3 rounded-lg border border-border bg-card group"
              >
                <span
                  className="h-3 w-3 rounded-full shrink-0"
                  style={{ backgroundColor: cat?.color || '#6b7280' }}
                />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-foreground">{tpl.name}</p>
                  <p className="text-xs text-muted-foreground truncate">
                    {cat?.name} — {tpl.description} ({tpl.estimated_minutes}min)
                  </p>
                </div>
                <button
                  onClick={() => deleteTemplate.mutate(tpl.id)}
                  className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            )
          })}

          {/* Add Template Form */}
          {showAddTpl && (
            <div className="p-4 rounded-lg border-2 border-dashed border-primary/30 bg-primary/5 space-y-3">
              <Input
                placeholder={t('profil.zeiterfassung.categories.templateNamePlaceholder')}
                value={newTplName}
                onChange={(e) => setNewTplName(e.target.value)}
                autoFocus
              />
              <div className="grid grid-cols-2 gap-3">
                <select
                  value={newTplCatId}
                  onChange={(e) => setNewTplCatId(e.target.value)}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                >
                  {(categories as TimeCategory[]).map((c) => (
                    <option key={c.id} value={c.id}>{c.name}</option>
                  ))}
                </select>
                <Input
                  type="number"
                  placeholder={t('profil.zeiterfassung.categories.minutes')}
                  value={newTplMins}
                  onChange={(e) => setNewTplMins(e.target.value)}
                />
              </div>
              <Input
                placeholder={t('profil.zeiterfassung.categories.descriptionPlaceholder')}
                value={newTplDesc}
                onChange={(e) => setNewTplDesc(e.target.value)}
              />
              <div className="flex justify-end gap-2">
                <Button size="sm" variant="outline" onClick={() => setShowAddTpl(false)}>
                  {t('common.cancel')}
                </Button>
                <Button
                  size="sm"
                  onClick={handleAddTemplate}
                  disabled={createTemplate.isPending}
                >
                  {t('profil.zeiterfassung.categories.add')}
                </Button>
              </div>
            </div>
          )}
        </div>
      </section>

      {/* Work Targets (read-only from HR settings — editing via Einstellungen module) */}
      <section>
        <h3 className="text-sm font-semibold text-foreground mb-4">
          {t('profil.zeiterfassung.categories.workTargets')}
        </h3>
        <div className="rounded-xl border border-border bg-card p-5 space-y-3">
          {hrSettings ? (
            <>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">{t('profil.zeiterfassung.categories.hoursPerDay')}</span>
                <span className="font-medium text-foreground">{hrSettings.workHoursPerDay}h</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">{t('profil.zeiterfassung.categories.maxDailyHours')}</span>
                <span className="font-medium text-foreground">{hrSettings.maxDailyHours}h</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">{t('profil.zeiterfassung.categories.breakAfterHours')}</span>
                <span className="font-medium text-foreground">{hrSettings.breakAfterHours}h</span>
              </div>
            </>
          ) : (
            <p className="text-sm text-muted-foreground">
              {t('profil.zeiterfassung.categories.workTargetsHint')}
            </p>
          )}
        </div>
      </section>
    </div>
  )
}
