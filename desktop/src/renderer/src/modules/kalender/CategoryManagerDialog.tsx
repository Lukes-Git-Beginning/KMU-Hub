import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Plus, Pencil, Trash2, Check, X } from 'lucide-react'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/shared'

interface EventCategory {
  id: string
  name: string
  color: string
}

const COLOR_PALETTE = [
  '#3d5c7d', '#4a7c6a', '#c4873a', '#7c5a8a', '#8a6b3d',
  '#1e7e74', '#a13f3f', '#5a6e8a', '#6b8a5a', '#8a5a6e',
  '#3a7ec4', '#c4563a', '#3ac4a8', '#c4c43a', '#8a3ac4',
]

interface CategoryManagerDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  categories: EventCategory[]
  onCategoriesChange: (categories: EventCategory[]) => void
}

export function CategoryManagerDialog({
  open,
  onOpenChange,
  categories,
  onCategoriesChange,
}: CategoryManagerDialogProps) {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  const [editColor, setEditColor] = useState('')
  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState(COLOR_PALETTE[0])
  const [showNewForm, setShowNewForm] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState<EventCategory | null>(null)

  const startEdit = (cat: EventCategory) => {
    setEditingId(cat.id)
    setEditName(cat.name)
    setEditColor(cat.color)
  }

  const saveEdit = () => {
    if (!editingId || !editName.trim()) return
    onCategoriesChange(
      categories.map((c) =>
        c.id === editingId ? { ...c, name: editName.trim(), color: editColor } : c,
      ),
    )
    setEditingId(null)
    toast.success('Kategorie aktualisiert')
  }

  const handleAdd = () => {
    if (!newName.trim()) return
    const id = `cat-${Date.now()}`
    onCategoriesChange([...categories, { id, name: newName.trim(), color: newColor }])
    setNewName('')
    setNewColor(COLOR_PALETTE[0])
    setShowNewForm(false)
    toast.success('Kategorie erstellt')
  }

  const handleDelete = (cat: EventCategory) => {
    onCategoriesChange(categories.filter((c) => c.id !== cat.id))
    setConfirmDelete(null)
    toast.success(`"${cat.name}" gelöscht`)
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Kategorien verwalten</DialogTitle>
          </DialogHeader>

          <div className="space-y-1 py-2">
            {categories.map((cat) => (
              <div key={cat.id}>
                {editingId === cat.id ? (
                  <div className="flex items-center gap-2 rounded-lg border border-primary/30 bg-primary/5 px-3 py-2">
                    {/* Color picker */}
                    <div className="relative group">
                      <button
                        className="h-6 w-6 rounded-full border-2 border-white shadow-sm shrink-0"
                        style={{ backgroundColor: editColor }}
                      />
                      <div className="absolute left-0 top-full mt-1 hidden group-hover:flex flex-wrap gap-1 p-2 rounded-lg border border-border bg-card shadow-[var(--shadow-medium)] z-10 w-40">
                        {COLOR_PALETTE.map((c) => (
                          <button
                            key={c}
                            onClick={() => setEditColor(c)}
                            className="h-5 w-5 rounded-full border-2 transition-transform hover:scale-110"
                            style={{
                              backgroundColor: c,
                              borderColor: c === editColor ? 'var(--foreground)' : 'transparent',
                            }}
                          />
                        ))}
                      </div>
                    </div>
                    <Input
                      autoFocus
                      value={editName}
                      onChange={(e) => setEditName(e.target.value)}
                      onKeyDown={(e) => { if (e.key === 'Enter') saveEdit() }}
                      className="h-7 text-xs flex-1"
                    />
                    <button onClick={saveEdit} className="rounded p-1 text-success hover:bg-success/10">
                      <Check className="h-3.5 w-3.5" />
                    </button>
                    <button onClick={() => setEditingId(null)} className="rounded p-1 text-muted-foreground hover:bg-secondary">
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2 rounded-lg px-3 py-2 hover:bg-secondary/50 transition-colors group">
                    <span
                      className="h-4 w-4 rounded-full shrink-0"
                      style={{ backgroundColor: cat.color }}
                    />
                    <span className="text-sm text-foreground flex-1">{cat.name}</span>
                    <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                      <button
                        onClick={() => startEdit(cat)}
                        className="rounded p-1 text-muted-foreground hover:bg-secondary hover:text-foreground"
                      >
                        <Pencil className="h-3 w-3" />
                      </button>
                      <button
                        onClick={() => setConfirmDelete(cat)}
                        className="rounded p-1 text-muted-foreground hover:bg-secondary hover:text-red-500"
                      >
                        <Trash2 className="h-3 w-3" />
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}

            {/* Add new category */}
            {showNewForm ? (
              <div className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 mt-2">
                <div className="relative group">
                  <button
                    className="h-6 w-6 rounded-full border-2 border-white shadow-sm shrink-0"
                    style={{ backgroundColor: newColor }}
                  />
                  <div className="absolute left-0 top-full mt-1 hidden group-hover:flex flex-wrap gap-1 p-2 rounded-lg border border-border bg-card shadow-[var(--shadow-medium)] z-10 w-40">
                    {COLOR_PALETTE.map((c) => (
                      <button
                        key={c}
                        onClick={() => setNewColor(c)}
                        className="h-5 w-5 rounded-full border-2 transition-transform hover:scale-110"
                        style={{
                          backgroundColor: c,
                          borderColor: c === newColor ? 'var(--foreground)' : 'transparent',
                        }}
                      />
                    ))}
                  </div>
                </div>
                <Input
                  autoFocus
                  placeholder="Kategorie-Name..."
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  onKeyDown={(e) => { if (e.key === 'Enter') handleAdd() }}
                  className="h-7 text-xs flex-1"
                />
                <button onClick={handleAdd} disabled={!newName.trim()} className="rounded p-1 text-success hover:bg-success/10 disabled:opacity-40">
                  <Check className="h-3.5 w-3.5" />
                </button>
                <button onClick={() => { setShowNewForm(false); setNewName('') }} className="rounded p-1 text-muted-foreground hover:bg-secondary">
                  <X className="h-3.5 w-3.5" />
                </button>
              </div>
            ) : (
              <button
                onClick={() => setShowNewForm(true)}
                className="flex items-center gap-2 w-full rounded-lg px-3 py-2 text-xs text-primary hover:bg-primary/5 transition-colors mt-1"
              >
                <Plus className="h-3.5 w-3.5" />
                Neue Kategorie
              </button>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Schliessen
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title="Kategorie löschen?"
        description={`"${confirmDelete?.name}" wird dauerhaft gelöscht. Bestehende Events behalten ihre Farbe.`}
        confirmLabel="Löschen"
        variant="destructive"
        onConfirm={() => confirmDelete && handleDelete(confirmDelete)}
      />
    </>
  )
}
