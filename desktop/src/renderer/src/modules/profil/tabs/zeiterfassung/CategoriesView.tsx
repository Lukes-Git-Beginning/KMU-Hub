import { useState } from 'react'
import { Plus, Pencil, Trash2, Check, X, Clock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore, type TimeCategory } from '@/stores/timetracking'
import { toast } from 'sonner'

const COLOR_PRESETS = [
  '#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b',
  '#10b981', '#ef4444', '#6b7280', '#94a3b8',
  '#14b8a6', '#f97316', '#06b6d4', '#84cc16',
]

export default function CategoriesView() {
  const categories = useTimeTrackingStore((s) => s.categories)
  const templates = useTimeTrackingStore((s) => s.templates)
  const targets = useTimeTrackingStore((s) => s.targets)
  const addCategory = useTimeTrackingStore((s) => s.addCategory)
  const updateCategory = useTimeTrackingStore((s) => s.updateCategory)
  const deleteCategory = useTimeTrackingStore((s) => s.deleteCategory)
  const addTemplate = useTimeTrackingStore((s) => s.addTemplate)
  const deleteTemplate = useTimeTrackingStore((s) => s.deleteTemplate)
  const updateTargets = useTimeTrackingStore((s) => s.updateTargets)

  const [showAddCat, setShowAddCat] = useState(false)
  const [newCatName, setNewCatName] = useState('')
  const [newCatColor, setNewCatColor] = useState(COLOR_PRESETS[0])
  const [editingCat, setEditingCat] = useState<string | null>(null)
  const [editName, setEditName] = useState('')

  const [showAddTpl, setShowAddTpl] = useState(false)
  const [newTplName, setNewTplName] = useState('')
  const [newTplCatId, setNewTplCatId] = useState('')
  const [newTplDesc, setNewTplDesc] = useState('')
  const [newTplMins, setNewTplMins] = useState('30')

  const [dailyHours, setDailyHours] = useState(String(targets.dailyHours))
  const [weeklyHours, setWeeklyHours] = useState(String(targets.weeklyHours))

  const handleAddCategory = () => {
    if (!newCatName.trim()) return
    addCategory({ name: newCatName, color: newCatColor, icon: 'Tag', isDefault: false })
    setNewCatName('')
    setShowAddCat(false)
    toast.success('Kategorie hinzugefuegt')
  }

  const handleSaveEdit = (id: string) => {
    if (!editName.trim()) return
    updateCategory(id, { name: editName })
    setEditingCat(null)
    toast.success('Kategorie aktualisiert')
  }

  const handleAddTemplate = () => {
    if (!newTplName.trim() || !newTplCatId) return
    addTemplate({
      name: newTplName,
      categoryId: newTplCatId,
      description: newTplDesc,
      estimatedMinutes: parseInt(newTplMins) || 30,
    })
    setNewTplName('')
    setNewTplDesc('')
    setShowAddTpl(false)
    toast.success('Vorlage hinzugefuegt')
  }

  const handleSaveTargets = () => {
    const dh = parseFloat(dailyHours)
    const wh = parseFloat(weeklyHours)
    if (isNaN(dh) || isNaN(wh) || dh <= 0 || wh <= 0) {
      toast.error('Bitte gueltige Zahlen eingeben')
      return
    }
    updateTargets({ dailyHours: dh, weeklyHours: wh, monthlyHours: Math.round(wh * 4.33) })
    toast.success('Arbeitsziele aktualisiert')
  }

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-8">
      {/* Categories */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-foreground">Kategorien ({categories.length})</h3>
          <Button size="sm" variant="outline" onClick={() => setShowAddCat(true)} className="gap-2">
            <Plus className="h-3.5 w-3.5" />
            Neue Kategorie
          </Button>
        </div>

        <div className="space-y-2">
          {categories.map((cat) => (
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
                  <button onClick={() => handleSaveEdit(cat.id)} className="p-1 text-emerald-600 hover:bg-emerald-500/10 rounded">
                    <Check className="h-4 w-4" />
                  </button>
                  <button onClick={() => setEditingCat(null)} className="p-1 text-muted-foreground hover:bg-accent rounded">
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ) : (
                <>
                  <span className="text-sm font-medium text-foreground flex-1">{cat.name}</span>
                  {cat.isDefault && (
                    <span className="text-[10px] text-muted-foreground bg-secondary px-1.5 py-0.5 rounded">Standard</span>
                  )}
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={() => { setEditingCat(cat.id); setEditName(cat.name) }}
                      className="p-1 rounded hover:bg-accent text-muted-foreground"
                    >
                      <Pencil className="h-3.5 w-3.5" />
                    </button>
                    {!cat.isDefault && (
                      <button
                        onClick={() => { deleteCategory(cat.id); toast.success('Kategorie geloescht') }}
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
                placeholder="Kategorie-Name..."
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
                <Button size="sm" variant="outline" onClick={() => setShowAddCat(false)}>Abbrechen</Button>
                <Button size="sm" onClick={handleAddCategory}>Hinzufuegen</Button>
              </div>
            </div>
          )}
        </div>
      </section>

      {/* Templates */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-foreground">Vorlagen ({templates.length})</h3>
          <Button size="sm" variant="outline" onClick={() => { setShowAddTpl(true); setNewTplCatId(categories[0]?.id || '') }} className="gap-2">
            <Plus className="h-3.5 w-3.5" />
            Neue Vorlage
          </Button>
        </div>

        <div className="space-y-2">
          {templates.map((tpl) => {
            const cat = categories.find((c) => c.id === tpl.categoryId)
            return (
              <div
                key={tpl.id}
                className="flex items-center gap-3 p-3 rounded-lg border border-border bg-card group"
              >
                <span className="h-3 w-3 rounded-full shrink-0" style={{ backgroundColor: cat?.color || '#6b7280' }} />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-foreground">{tpl.name}</p>
                  <p className="text-xs text-muted-foreground truncate">
                    {cat?.name} — {tpl.description} ({tpl.estimatedMinutes}min)
                  </p>
                </div>
                <button
                  onClick={() => { deleteTemplate(tpl.id); toast.success('Vorlage geloescht') }}
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
                placeholder="Vorlagen-Name..."
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
                  {categories.map((c) => (
                    <option key={c.id} value={c.id}>{c.name}</option>
                  ))}
                </select>
                <Input
                  type="number"
                  placeholder="Minuten"
                  value={newTplMins}
                  onChange={(e) => setNewTplMins(e.target.value)}
                />
              </div>
              <Input
                placeholder="Beschreibung..."
                value={newTplDesc}
                onChange={(e) => setNewTplDesc(e.target.value)}
              />
              <div className="flex justify-end gap-2">
                <Button size="sm" variant="outline" onClick={() => setShowAddTpl(false)}>Abbrechen</Button>
                <Button size="sm" onClick={handleAddTemplate}>Hinzufuegen</Button>
              </div>
            </div>
          )}
        </div>
      </section>

      {/* Work Targets */}
      <section>
        <h3 className="text-sm font-semibold text-foreground mb-4">Arbeitsziele</h3>
        <div className="rounded-xl border border-border bg-card p-5 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Stunden / Tag</label>
              <Input
                type="number"
                step="0.1"
                value={dailyHours}
                onChange={(e) => setDailyHours(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Stunden / Woche</label>
              <Input
                type="number"
                step="0.5"
                value={weeklyHours}
                onChange={(e) => setWeeklyHours(e.target.value)}
              />
            </div>
          </div>
          <div className="flex justify-end">
            <Button size="sm" onClick={handleSaveTargets}>Speichern</Button>
          </div>
        </div>
      </section>
    </div>
  )
}
