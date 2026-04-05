/**
 * Channel-to-module mapping editor for integration setup wizards.
 *
 * Displays existing mappings with colored module chips and provides
 * inline forms for creating and editing mappings. Used by both
 * TeamsSetupWizard and SlackSetupWizard.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Plus, Pencil, Trash2, Save, X } from 'lucide-react'
import { toast } from 'sonner'
import {
  useChannelMappings,
  useCreateChannelMapping,
  useUpdateChannelMapping,
  useDeleteChannelMapping,
} from '@/api/hooks/useIntegration'
import { INTEGRATION_MODULES } from '@/api/integration-types'
import type { Platform, ChannelMapping } from '@/api/integration-types'

interface ChannelMappingEditorProps {
  platform: Platform
  /** Called when mappings change (for wizard step validation) */
  onMappingsChange?: (mappings: ChannelMapping[]) => void
}

interface MappingFormState {
  channelId: string
  channelName: string
  modules: string[]
}

const EMPTY_FORM: MappingFormState = {
  channelId: '',
  channelName: '',
  modules: [],
}

export function ChannelMappingEditor({
  platform,
  onMappingsChange,
}: ChannelMappingEditorProps) {
  const { t } = useTranslation()
  const { data: mappings, isLoading } = useChannelMappings(platform)
  const createMapping = useCreateChannelMapping(platform)
  const updateMapping = useUpdateChannelMapping(platform)
  const deleteMapping = useDeleteChannelMapping(platform)

  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState<MappingFormState>(EMPTY_FORM)
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)

  const toggleModule = (moduleId: string) => {
    setForm((prev) => ({
      ...prev,
      modules: prev.modules.includes(moduleId)
        ? prev.modules.filter((m) => m !== moduleId)
        : [...prev.modules, moduleId],
    }))
  }

  const handleCreate = async () => {
    if (!form.channelId.trim() || !form.channelName.trim()) {
      toast.error('Bitte Kanal-ID und Name eingeben')
      return
    }
    if (form.modules.length === 0) {
      toast.error('Bitte mindestens ein Modul auswählen')
      return
    }
    await createMapping.mutateAsync({
      channel_id: form.channelId.trim(),
      channel_name: form.channelName.trim(),
      modules: form.modules,
    })
    setForm(EMPTY_FORM)
    setShowForm(false)
    onMappingsChange?.(mappings ?? [])
  }

  const handleUpdate = async () => {
    if (!editingId) return
    if (!form.channelName.trim()) {
      toast.error('Bitte Kanalnamen eingeben')
      return
    }
    await updateMapping.mutateAsync({
      id: editingId,
      channel_id: form.channelId.trim(),
      channel_name: form.channelName.trim(),
      modules: form.modules,
    })
    setEditingId(null)
    setForm(EMPTY_FORM)
    onMappingsChange?.(mappings ?? [])
  }

  const handleDelete = async (id: string) => {
    await deleteMapping.mutateAsync(id)
    setDeleteConfirmId(null)
    onMappingsChange?.(mappings ?? [])
  }

  const startEdit = (mapping: ChannelMapping) => {
    setEditingId(mapping.id)
    setShowForm(false)
    setForm({
      channelId: mapping.channel_id,
      channelName: mapping.channel_name,
      modules: [...mapping.modules],
    })
  }

  const cancelEdit = () => {
    setEditingId(null)
    setForm(EMPTY_FORM)
  }

  if (isLoading) {
    return (
      <div className="animate-pulse space-y-3">
        <div className="h-16 bg-muted rounded" />
        <div className="h-16 bg-muted rounded" />
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {/* Existing mappings */}
      {mappings && mappings.length > 0 ? (
        <div className="space-y-2">
          {mappings.map((mapping) => (
            <div key={mapping.id}>
              {editingId === mapping.id ? (
                <MappingForm
                  form={form}
                  setForm={setForm}
                  toggleModule={toggleModule}
                  onSave={handleUpdate}
                  onCancel={cancelEdit}
                  isSaving={updateMapping.isPending}
                  isEdit
                />
              ) : (
                <div className="rounded-md border border-border p-3">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">
                        {mapping.channel_name}
                      </span>
                      <span className="text-xs text-muted-foreground font-mono">
                        {mapping.channel_id}
                      </span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <Switch
                        checked={mapping.is_active}
                        onCheckedChange={(checked) =>
                          updateMapping.mutate({
                            id: mapping.id,
                            is_active: checked,
                          })
                        }
                      />
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => startEdit(mapping)}
                        className="h-7 w-7 p-0"
                      >
                        <Pencil className="h-3.5 w-3.5" />
                      </Button>
                      {deleteConfirmId === mapping.id ? (
                        <div className="flex items-center gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(mapping.id)}
                            className="h-7 px-2 text-destructive hover:text-destructive"
                          >
                            Löschen
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setDeleteConfirmId(null)}
                            className="h-7 w-7 p-0"
                          >
                            <X className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      ) : (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setDeleteConfirmId(mapping.id)}
                          className="h-7 w-7 p-0 text-destructive hover:text-destructive"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </div>
                  </div>
                  {/* Module chips */}
                  <div className="flex flex-wrap gap-1.5">
                    {mapping.modules.map((modId) => {
                      const mod = INTEGRATION_MODULES.find(
                        (m) => m.id === modId,
                      )
                      if (!mod) return null
                      return (
                        <span
                          key={modId}
                          className="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium text-white"
                          style={{ backgroundColor: mod.color }}
                        >
                          {mod.label}
                        </span>
                      )
                    })}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">
          Noch keine Kanalzuordnungen erstellt.
        </p>
      )}

      {/* Add new mapping form */}
      {showForm ? (
        <MappingForm
          form={form}
          setForm={setForm}
          toggleModule={toggleModule}
          onSave={handleCreate}
          onCancel={() => {
            setShowForm(false)
            setForm(EMPTY_FORM)
          }}
          isSaving={createMapping.isPending}
        />
      ) : (
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setShowForm(true)
            setEditingId(null)
            setForm(EMPTY_FORM)
          }}
        >
          <Plus className="h-3.5 w-3.5 mr-1.5" />
          Neue Zuordnung
        </Button>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Inline mapping form sub-component
// ---------------------------------------------------------------------------

function MappingForm({
  form,
  setForm,
  toggleModule,
  onSave,
  onCancel,
  isSaving,
  isEdit = false,
}: {
  form: MappingFormState
  setForm: React.Dispatch<React.SetStateAction<MappingFormState>>
  toggleModule: (id: string) => void
  onSave: () => void
  onCancel: () => void
  isSaving: boolean
  isEdit?: boolean
}) {
  return (
    <div className="rounded-md border border-primary/30 bg-primary/5 p-3 space-y-3">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <Label htmlFor="channel-id" className="text-xs">
            Kanal-ID
          </Label>
          <Input
            id="channel-id"
            placeholder="z.B. C0123456789"
            value={form.channelId}
            onChange={(e) =>
              setForm((prev) => ({ ...prev, channelId: e.target.value }))
            }
            disabled={isEdit}
          />
        </div>
        <div>
          <Label htmlFor="channel-name" className="text-xs">
            Kanalname
          </Label>
          <Input
            id="channel-name"
            placeholder="z.B. #benachrichtigungen"
            value={form.channelName}
            onChange={(e) =>
              setForm((prev) => ({ ...prev, channelName: e.target.value }))
            }
          />
        </div>
      </div>

      {/* Module toggles as colored chips */}
      <div>
        <Label className="text-xs mb-1.5 block">Module</Label>
        <div className="flex flex-wrap gap-1.5">
          {INTEGRATION_MODULES.map((mod) => {
            const isSelected = form.modules.includes(mod.id)
            return (
              <button
                key={mod.id}
                type="button"
                onClick={() => toggleModule(mod.id)}
                className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium transition-colors ${
                  isSelected
                    ? 'text-white'
                    : 'bg-muted text-muted-foreground hover:bg-muted/80'
                }`}
                style={isSelected ? { backgroundColor: mod.color } : undefined}
              >
                {mod.label}
              </button>
            )
          })}
        </div>
      </div>

      {/* Form actions */}
      <div className="flex items-center gap-2">
        <Button size="sm" onClick={onSave} disabled={isSaving}>
          <Save className="h-3.5 w-3.5 mr-1" />
          {isEdit ? 'Aktualisieren' : 'Speichern'}
        </Button>
        <Button variant="ghost" size="sm" onClick={onCancel}>
          Abbrechen
        </Button>
      </div>
    </div>
  )
}
