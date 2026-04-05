import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Plus, Pencil, Trash2, Check, X, Users } from 'lucide-react'
import { useContactsStore, type ContactGroup } from '@/stores/contacts'

interface GroupManagerDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const GROUP_COLORS = [
  '#3B82F6', '#8B5CF6', '#10B981', '#F59E0B',
  '#EF4444', '#EC4899', '#6B7280', '#06B6D4',
]

export function GroupManagerDialog({ open, onOpenChange }: GroupManagerDialogProps) {
  const { t } = useTranslation()
  const { groups, addGroup, updateGroup, deleteGroup } = useContactsStore()

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  const [editColor, setEditColor] = useState(GROUP_COLORS[0])
  const [isCreating, setIsCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState(GROUP_COLORS[0])
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)

  const startEdit = (group: ContactGroup) => {
    setEditingId(group.id)
    setEditName(group.name)
    setEditColor(group.color)
    setIsCreating(false)
  }

  const saveEdit = () => {
    if (editingId && editName.trim()) {
      updateGroup(editingId, { name: editName.trim(), color: editColor })
      setEditingId(null)
    }
  }

  const handleCreate = () => {
    if (newName.trim()) {
      addGroup(newName.trim(), newColor)
      setNewName('')
      setNewColor(GROUP_COLORS[0])
      setIsCreating(false)
    }
  }

  const handleDelete = (id: string) => {
    deleteGroup(id)
    setDeleteConfirmId(null)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('kontakte.groups.manage')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-3 py-2">
          {/* Existing groups */}
          {groups.map((group) => {
            const memberCount = group.contactIds.length

            if (editingId === group.id) {
              return (
                <div key={group.id} className="rounded-md border border-primary/30 bg-primary/5 p-3 space-y-2">
                  <Input
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && saveEdit()}
                    placeholder={t('kontakte.groups.groupName')}
                    autoFocus
                  />
                  <div className="flex items-center gap-1.5">
                    {GROUP_COLORS.map((c) => (
                      <button
                        key={c}
                        onClick={() => setEditColor(c)}
                        className={`h-5 w-5 rounded-full transition-all ${
                          editColor === c ? 'ring-2 ring-offset-1 ring-primary scale-110' : ''
                        }`}
                        style={{ backgroundColor: c }}
                      />
                    ))}
                  </div>
                  <div className="flex justify-end gap-1.5">
                    <Button variant="ghost" size="sm" onClick={() => setEditingId(null)}>
                      <X className="h-3.5 w-3.5 mr-1" />
                      {t('common.cancel')}
                    </Button>
                    <Button size="sm" onClick={saveEdit} disabled={!editName.trim()}>
                      <Check className="h-3.5 w-3.5 mr-1" />
                      {t('common.save')}
                    </Button>
                  </div>
                </div>
              )
            }

            if (deleteConfirmId === group.id) {
              return (
                <div key={group.id} className="rounded-md border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-3">
                  <p className="text-sm text-red-700 dark:text-red-400 mb-2">
                    {t('kontakte.groups.deleteConfirm', { name: group.name })}
                  </p>
                  <div className="flex justify-end gap-1.5">
                    <Button variant="ghost" size="sm" onClick={() => setDeleteConfirmId(null)}>
                      {t('common.cancel')}
                    </Button>
                    <Button variant="destructive" size="sm" onClick={() => handleDelete(group.id)}>
                      {t('common.delete')}
                    </Button>
                  </div>
                </div>
              )
            }

            return (
              <div
                key={group.id}
                className="group flex items-center gap-3 rounded-md border border-border px-3 py-2.5 hover:bg-secondary/50 transition-colors"
              >
                <span
                  className="h-3 w-3 rounded-full shrink-0"
                  style={{ backgroundColor: group.color }}
                />
                <div className="flex-1 min-w-0">
                  <span className="text-sm font-medium text-[var(--body)]">{group.name}</span>
                  <span className="ml-2 text-xs text-[var(--muted)]">
                    {t('kontakte.groups.memberCount', { count: memberCount })}
                  </span>
                </div>
                <div className="hidden group-hover:flex items-center gap-0.5">
                  <button
                    onClick={() => startEdit(group)}
                    className="rounded p-1 text-[var(--muted)] hover:text-[var(--body)] transition-colors"
                  >
                    <Pencil className="h-3.5 w-3.5" />
                  </button>
                  <button
                    onClick={() => setDeleteConfirmId(group.id)}
                    className="rounded p-1 text-[var(--muted)] hover:text-red-500 transition-colors"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            )
          })}

          {groups.length === 0 && !isCreating && (
            <div className="py-6 text-center">
              <Users className="mx-auto h-8 w-8 text-[var(--muted)] mb-2" />
              <p className="text-sm text-[var(--muted)]">{t('kontakte.groups.noGroups')}</p>
              <p className="text-xs text-[var(--muted)] mt-1">
                {t('kontakte.groups.noGroupsHint')}
              </p>
            </div>
          )}

          {/* Create new */}
          {isCreating ? (
            <div className="rounded-md border border-primary/30 bg-primary/5 p-3 space-y-2">
              <Input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
                placeholder={t('kontakte.groups.newGroupName')}
                autoFocus
              />
              <div className="flex items-center gap-1.5">
                {GROUP_COLORS.map((c) => (
                  <button
                    key={c}
                    onClick={() => setNewColor(c)}
                    className={`h-5 w-5 rounded-full transition-all ${
                      newColor === c ? 'ring-2 ring-offset-1 ring-primary scale-110' : ''
                    }`}
                    style={{ backgroundColor: c }}
                  />
                ))}
              </div>
              <div className="flex justify-end gap-1.5">
                <Button variant="ghost" size="sm" onClick={() => setIsCreating(false)}>
                  {t('common.cancel')}
                </Button>
                <Button size="sm" onClick={handleCreate} disabled={!newName.trim()}>
                  <Plus className="h-3.5 w-3.5 mr-1" />
                  {t('common.create')}
                </Button>
              </div>
            </div>
          ) : (
            <Button
              variant="outline"
              size="sm"
              className="w-full"
              onClick={() => {
                setIsCreating(true)
                setEditingId(null)
              }}
            >
              <Plus className="h-4 w-4 mr-1.5" />
              {t('kontakte.groups.newGroup')}
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
