/**
 * Project creation dialog with name, key, description, and initial statuses.
 *
 * Auto-generates a project key from the name (first 3-4 uppercase letters).
 * Allows configuring initial status columns with color pickers before creation.
 */
import { useState, useEffect } from 'react'
import { Plus, X } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

interface StatusDraft {
  name: string
  color: string
  is_default: boolean
  is_closed: boolean
}

const DEFAULT_STATUSES: StatusDraft[] = [
  { name: 'Offen', color: '#6b7280', is_default: true, is_closed: false },
  { name: 'In Arbeit', color: '#3b82f6', is_default: false, is_closed: false },
  { name: 'Review', color: '#f59e0b', is_default: false, is_closed: false },
  { name: 'Erledigt', color: '#22c55e', is_default: false, is_closed: true },
]

const STATUS_COLORS = [
  '#6b7280', '#ef4444', '#f59e0b', '#22c55e',
  '#3b82f6', '#8b5cf6', '#ec4899', '#14b8a6',
]

interface ProjectCreateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (values: {
    name: string
    project_key: string
    description?: string
    statuses?: StatusDraft[]
  }) => Promise<void>
  isSubmitting: boolean
}

function generateKey(name: string): string {
  return name
    .replace(/[^a-zA-Z0-9]/g, '')
    .toUpperCase()
    .slice(0, 4)
}

export default function ProjectCreateDialog({
  open,
  onOpenChange,
  onSubmit,
  isSubmitting,
}: ProjectCreateDialogProps) {
  const [name, setName] = useState('')
  const [key, setKey] = useState('')
  const [keyManual, setKeyManual] = useState(false)
  const [description, setDescription] = useState('')
  const [statuses, setStatuses] = useState<StatusDraft[]>(DEFAULT_STATUSES)
  const [error, setError] = useState<string | null>(null)

  // Auto-generate key from name unless manually edited
  useEffect(() => {
    if (!keyManual) {
      setKey(generateKey(name))
    }
  }, [name, keyManual])

  function resetForm() {
    setName('')
    setKey('')
    setKeyManual(false)
    setDescription('')
    setStatuses(DEFAULT_STATUSES)
    setError(null)
  }

  function handleOpenChange(open: boolean) {
    if (!open) resetForm()
    onOpenChange(open)
  }

  function addStatus() {
    setStatuses((prev) => [
      ...prev,
      { name: '', color: '#6b7280', is_default: false, is_closed: false },
    ])
  }

  function removeStatus(index: number) {
    setStatuses((prev) => prev.filter((_, i) => i !== index))
  }

  function updateStatus(index: number, updates: Partial<StatusDraft>) {
    setStatuses((prev) =>
      prev.map((s, i) => {
        if (i !== index) {
          // If setting a new default, clear default on others
          if (updates.is_default) return { ...s, is_default: false }
          return s
        }
        return { ...s, ...updates }
      })
    )
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)

    if (!name.trim()) {
      setError('Projektname ist erforderlich.')
      return
    }
    if (!key.trim() || key.length < 2) {
      setError('Projektkuerzel muss mindestens 2 Zeichen lang sein.')
      return
    }
    if (!/^[A-Z0-9]+$/.test(key)) {
      setError('Projektkuerzel darf nur Grossbuchstaben und Zahlen enthalten.')
      return
    }
    if (statuses.some((s) => !s.name.trim())) {
      setError('Alle Status muessen einen Namen haben.')
      return
    }

    try {
      await onSubmit({
        name: name.trim(),
        project_key: key,
        description: description.trim() || undefined,
        statuses,
      })
      resetForm()
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Projekt konnte nicht erstellt werden.'
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Neues Projekt</DialogTitle>
          <DialogDescription>
            Erstelle ein neues Projekt mit Statusen und Mitgliedern.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Name */}
          <div className="space-y-2">
            <Label htmlFor="project-name">Projektname *</Label>
            <Input
              id="project-name"
              placeholder="z.B. Website Relaunch"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
          </div>

          {/* Key */}
          <div className="space-y-2">
            <Label htmlFor="project-key">Projektkuerzel *</Label>
            <Input
              id="project-key"
              placeholder="z.B. WEB"
              value={key}
              onChange={(e) => {
                setKey(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 10))
                setKeyManual(true)
              }}
              maxLength={10}
              className="font-mono uppercase"
            />
            <p className="text-xs text-muted-foreground">
              Wird als Prefix fuer Aufgabennummern verwendet (z.B. {key || 'KEY'}-1)
            </p>
          </div>

          {/* Description */}
          <div className="space-y-2">
            <Label htmlFor="project-description">Beschreibung</Label>
            <Textarea
              id="project-description"
              placeholder="Optionale Projektbeschreibung..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
            />
          </div>

          {/* Status configuration */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Status</Label>
              <Button type="button" variant="ghost" size="sm" onClick={addStatus}>
                <Plus className="h-3 w-3 mr-1" />
                Status hinzufuegen
              </Button>
            </div>
            <div className="space-y-2">
              {statuses.map((status, index) => (
                <div key={index} className="flex items-center gap-2">
                  {/* Color picker */}
                  <div className="relative">
                    <input
                      type="color"
                      value={status.color}
                      onChange={(e) => updateStatus(index, { color: e.target.value })}
                      className="h-8 w-8 cursor-pointer rounded border border-border"
                      title="Farbe waehlen"
                    />
                  </div>
                  {/* Name */}
                  <Input
                    placeholder="Status Name"
                    value={status.name}
                    onChange={(e) => updateStatus(index, { name: e.target.value })}
                    className="flex-1"
                  />
                  {/* Badges for default/closed */}
                  <button
                    type="button"
                    className={`text-xs px-2 py-1 rounded border ${
                      status.is_default
                        ? 'bg-primary text-primary-foreground border-primary'
                        : 'bg-background text-muted-foreground border-border hover:border-primary'
                    }`}
                    onClick={() => updateStatus(index, { is_default: true })}
                    title="Als Standard setzen"
                  >
                    Standard
                  </button>
                  <button
                    type="button"
                    className={`text-xs px-2 py-1 rounded border ${
                      status.is_closed
                        ? 'bg-green-100 text-green-700 border-green-300'
                        : 'bg-background text-muted-foreground border-border hover:border-green-300'
                    }`}
                    onClick={() => updateStatus(index, { is_closed: !status.is_closed })}
                    title="Abgeschlossen-Status"
                  >
                    Fertig
                  </button>
                  {/* Remove */}
                  {statuses.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 shrink-0"
                      onClick={() => removeStatus(index)}
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  )}
                </div>
              ))}
            </div>
            <div className="flex flex-wrap gap-1 mt-1">
              {STATUS_COLORS.map((color) => (
                <button
                  key={color}
                  type="button"
                  className="h-5 w-5 rounded-full border border-border"
                  style={{ backgroundColor: color }}
                  title={color}
                  onClick={() => {
                    // Apply to the last status that has default color
                    const lastDefault = statuses.findLastIndex((s) => s.color === '#6b7280')
                    if (lastDefault >= 0) {
                      updateStatus(lastDefault, { color })
                    }
                  }}
                />
              ))}
            </div>
          </div>

          {/* Error */}
          {error && (
            <p className="text-sm text-destructive">{error}</p>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
            >
              Abbrechen
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? 'Erstelle...' : 'Projekt erstellen'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
