/**
 * MeetingActionItems -- Action item list for during and after meetings.
 *
 * Features:
 * - List of action items with checkbox, description, assignee
 * - Inline add new item
 * - Edit/delete items
 * - Batch convert to tasks with project picker
 */
import { useState } from 'react'
import {
  Plus,
  X,
  Check,
  Trash2,
  FolderKanban,
  ExternalLink,
  Loader2,
} from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import {
  useActionItems,
  useCreateActionItem,
  useUpdateActionItem,
  useDeleteActionItem,
  useConvertActionItemsToTasks,
} from '@/api/hooks/useMeetings'
import { useProjects } from '@/api/hooks/useProjects'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MeetingActionItemsProps {
  meetingId: string
  isEditable: boolean
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function MeetingActionItems({ meetingId, isEditable }: MeetingActionItemsProps) {
  const { data: actionItems = [], isLoading } = useActionItems(meetingId)
  const { data: projectsData } = useProjects()
  const projects = projectsData?.projects ?? []
  const createMutation = useCreateActionItem()
  const updateMutation = useUpdateActionItem()
  const deleteMutation = useDeleteActionItem()
  const convertMutation = useConvertActionItemsToTasks()

  const [newDescription, setNewDescription] = useState('')
  const [newAssigneeId, setNewAssigneeId] = useState('')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editText, setEditText] = useState('')
  const [selectedProjectId, setSelectedProjectId] = useState<string>('')

  // Computed state
  const unconvertedItems = actionItems.filter((item) => !item.task_id)
  const allConverted = unconvertedItems.length === 0
  const canConvert = !allConverted && selectedProjectId && actionItems.length > 0

  const handleAdd = () => {
    const trimmed = newDescription.trim()
    if (!trimmed) return

    createMutation.mutate(
      {
        meetingId,
        data: {
          description: trimmed,
          assignee_id: newAssigneeId || undefined,
        },
      },
      {
        onSuccess: () => {
          setNewDescription('')
          setNewAssigneeId('')
        },
        onError: () => toast.error('Fehler beim Erstellen'),
      },
    )
  }

  const handleToggle = (itemId: string, isCompleted: boolean) => {
    updateMutation.mutate({
      itemId,
      data: { is_completed: !isCompleted },
    })
  }

  const handleEdit = (itemId: string) => {
    const trimmed = editText.trim()
    if (!trimmed) return

    updateMutation.mutate(
      { itemId, data: { description: trimmed } },
      {
        onSuccess: () => setEditingId(null),
        onError: () => toast.error('Fehler beim Aktualisieren'),
      },
    )
  }

  const handleDelete = (itemId: string) => {
    deleteMutation.mutate(itemId, {
      onError: () => toast.error('Fehler beim Löschen'),
    })
  }

  const handleConvert = () => {
    if (!selectedProjectId) {
      toast.error('Bitte wähle ein Projekt aus')
      return
    }

    convertMutation.mutate(
      { meetingId, projectId: selectedProjectId },
      {
        onSuccess: (data) => {
          toast.success(`${data.task_ids.length} Tasks erstellt`)
        },
        onError: () => toast.error('Fehler bei der Konvertierung'),
      },
    )
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin h-6 w-6 border-2 border-primary border-t-transparent rounded-full" />
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="px-4 py-3 border-b border-border">
        <h3 className="text-sm font-semibold text-foreground">
          Action Items ({actionItems.length})
        </h3>
      </div>

      {/* Item List */}
      <div className="flex-1 overflow-y-auto p-4 space-y-2">
        {actionItems.map((item) => (
          <div
            key={item.id}
            className="flex items-start gap-2 rounded-md border border-border p-2 group hover:bg-secondary/30 transition-colors"
          >
            {/* Checkbox */}
            <Checkbox
              checked={item.is_completed}
              onCheckedChange={() => handleToggle(item.id, item.is_completed)}
              disabled={!isEditable}
              className="mt-0.5"
            />

            {/* Content */}
            <div className="flex-1 min-w-0">
              {editingId === item.id ? (
                <div className="flex items-center gap-1">
                  <Input
                    value={editText}
                    onChange={(e) => setEditText(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleEdit(item.id)
                      if (e.key === 'Escape') setEditingId(null)
                    }}
                    className="text-sm h-7"
                    autoFocus
                  />
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={() => handleEdit(item.id)}
                  >
                    <Check className="h-3.5 w-3.5" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={() => setEditingId(null)}
                  >
                    <X className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <span
                    className={`text-sm flex-1 ${
                      item.is_completed
                        ? 'line-through text-muted-foreground'
                        : 'text-foreground'
                    } ${isEditable ? 'cursor-pointer' : ''}`}
                    onClick={() => {
                      if (isEditable) {
                        setEditingId(item.id)
                        setEditText(item.description)
                      }
                    }}
                  >
                    {item.description}
                  </span>

                  {/* Task link indicator */}
                  {item.task_id && (
                    <span className="flex items-center gap-0.5 text-[10px] text-primary">
                      <ExternalLink className="h-3 w-3" />
                      Task
                    </span>
                  )}

                  {/* Assignee avatar */}
                  {item.assignee_id && (
                    <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary/10 text-[8px] font-medium text-primary shrink-0">
                      {item.assignee_id.slice(0, 2).toUpperCase()}
                    </span>
                  )}
                </div>
              )}
            </div>

            {/* Delete button (visible on hover) */}
            {isEditable && editingId !== item.id && (
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity text-red-500 hover:text-red-600"
                onClick={() => handleDelete(item.id)}
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            )}
          </div>
        ))}

        {actionItems.length === 0 && (
          <p className="text-sm text-muted-foreground text-center py-4">
            Noch keine Action Items
          </p>
        )}
      </div>

      {/* Add New Item */}
      {isEditable && (
        <div className="px-4 py-3 border-t border-border">
          <div className="flex items-center gap-2">
            <Input
              placeholder="Neues Action Item..."
              value={newDescription}
              onChange={(e) => setNewDescription(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleAdd()
              }}
              className="text-sm"
            />
            <Button
              variant="outline"
              size="icon"
              onClick={handleAdd}
              disabled={!newDescription.trim() || createMutation.isPending}
            >
              <Plus className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {/* Batch Convert to Tasks */}
      <div className="px-4 py-3 border-t border-border bg-secondary/20">
        <div className="flex items-center gap-2 mb-2">
          <FolderKanban className="h-4 w-4 text-muted-foreground" />
          <span className="text-xs font-medium text-muted-foreground">
            Als Tasks erstellen
          </span>
        </div>
        <div className="flex items-center gap-2">
          <Select value={selectedProjectId} onValueChange={setSelectedProjectId}>
            <SelectTrigger className="text-sm flex-1">
              <SelectValue placeholder="Projekt wählen..." />
            </SelectTrigger>
            <SelectContent>
              {projects.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            onClick={handleConvert}
            disabled={!canConvert || convertMutation.isPending}
            size="sm"
            className="shrink-0"
          >
            {convertMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              'Alle Action Items als Tasks erstellen'
            )}
          </Button>
        </div>
        {allConverted && actionItems.length > 0 && (
          <p className="text-xs text-green-600 mt-1.5">
            Alle Action Items wurden bereits konvertiert.
          </p>
        )}
      </div>
    </div>
  )
}
