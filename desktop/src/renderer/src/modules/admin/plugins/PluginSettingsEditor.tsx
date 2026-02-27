/**
 * Dynamic settings form rendered from a JSON Schema.
 *
 * Supports text, number, boolean (switch), and select fields based on
 * the schema properties. Used inside PluginDetailDialog for per-plugin
 * configuration.
 */
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Loader2, Save } from 'lucide-react'

interface PluginSettingsEditorProps {
  schema: Record<string, unknown>
  values: Record<string, unknown>
  onSave: (values: Record<string, unknown>) => void
  isSaving: boolean
}

interface SchemaProperty {
  type?: string
  title?: string
  description?: string
  default?: unknown
  enum?: unknown[]
}

export function PluginSettingsEditor({
  schema,
  values,
  onSave,
  isSaving,
}: PluginSettingsEditorProps) {
  const [formValues, setFormValues] = useState<Record<string, unknown>>(values)

  useEffect(() => {
    setFormValues(values)
  }, [values])

  const properties = (schema.properties ?? {}) as Record<string, SchemaProperty>
  const propertyKeys = Object.keys(properties)

  if (propertyKeys.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        Dieses Plugin hat keine konfigurierbaren Einstellungen.
      </p>
    )
  }

  const updateField = (key: string, value: unknown) => {
    setFormValues((prev) => ({ ...prev, [key]: value }))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSave(formValues)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {propertyKeys.map((key) => {
        const prop = properties[key]
        const currentValue = formValues[key] ?? prop.default ?? ''

        // Select field (enum)
        if (prop.enum && prop.enum.length > 0) {
          return (
            <div key={key} className="space-y-1.5">
              <Label className="text-sm font-medium">
                {prop.title ?? key}
              </Label>
              {prop.description && (
                <p className="text-xs text-muted-foreground">
                  {prop.description}
                </p>
              )}
              <select
                className="w-full text-sm rounded-md border border-border bg-background px-3 py-2"
                value={String(currentValue)}
                onChange={(e) => updateField(key, e.target.value)}
              >
                {prop.enum.map((val) => (
                  <option key={String(val)} value={String(val)}>
                    {String(val)}
                  </option>
                ))}
              </select>
            </div>
          )
        }

        // Boolean field (switch)
        if (prop.type === 'boolean') {
          return (
            <div
              key={key}
              className="flex items-center justify-between rounded-md border border-border p-3"
            >
              <div>
                <Label className="text-sm font-medium">
                  {prop.title ?? key}
                </Label>
                {prop.description && (
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {prop.description}
                  </p>
                )}
              </div>
              <Switch
                checked={Boolean(currentValue)}
                onCheckedChange={(v) => updateField(key, v)}
              />
            </div>
          )
        }

        // Number field
        if (prop.type === 'number' || prop.type === 'integer') {
          return (
            <div key={key} className="space-y-1.5">
              <Label className="text-sm font-medium">
                {prop.title ?? key}
              </Label>
              {prop.description && (
                <p className="text-xs text-muted-foreground">
                  {prop.description}
                </p>
              )}
              <Input
                type="number"
                value={String(currentValue)}
                onChange={(e) => updateField(key, Number(e.target.value))}
              />
            </div>
          )
        }

        // Default: text field
        return (
          <div key={key} className="space-y-1.5">
            <Label className="text-sm font-medium">
              {prop.title ?? key}
            </Label>
            {prop.description && (
              <p className="text-xs text-muted-foreground">
                {prop.description}
              </p>
            )}
            <Input
              type="text"
              value={String(currentValue)}
              onChange={(e) => updateField(key, e.target.value)}
            />
          </div>
        )
      })}

      <Button type="submit" size="sm" disabled={isSaving}>
        {isSaving ? (
          <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
        ) : (
          <Save className="h-3.5 w-3.5 mr-1.5" />
        )}
        Einstellungen speichern
      </Button>
    </form>
  )
}
