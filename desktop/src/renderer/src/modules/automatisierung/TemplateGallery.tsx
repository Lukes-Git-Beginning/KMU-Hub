/**
 * Template gallery for automation module.
 *
 * Displays pre-built automation templates with module-based or complexity-based
 * grouping toggle. Template cards show category, name, description, and complexity.
 * "Vorlage verwenden" creates an automation from template and opens wizard.
 */
import { useState, useMemo } from 'react'
import {
  Search,
  Users,
  Receipt,
  UserCheck,
  MessageSquare,
  Eye,
} from 'lucide-react'
import * as Switch from '@radix-ui/react-switch'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { useAutomationTemplates, useCreateFromTemplate } from '@/api/hooks/useAutomation'
import { useAutomatisierungStore } from '@/stores/automatisierung'
import type {
  AutomationTemplate,
  TemplateCategory,
  TemplateComplexity,
} from '@/api/automation-types'

// ---------------------------------------------------------------------------
// Category config
// ---------------------------------------------------------------------------

const CATEGORY_CONFIG: Record<
  TemplateCategory,
  { icon: React.ElementType; label: string; color: string; band: string }
> = {
  vertrieb: {
    icon: Users,
    label: 'Vertrieb',
    color: 'text-blue-500',
    band: 'bg-blue-500',
  },
  finanzen: {
    icon: Receipt,
    label: 'Finanzen',
    color: 'text-amber-500',
    band: 'bg-amber-500',
  },
  personal: {
    icon: UserCheck,
    label: 'Personal',
    color: 'text-rose-500',
    band: 'bg-rose-500',
  },
  kommunikation: {
    icon: MessageSquare,
    label: 'Kommunikation',
    color: 'text-green-500',
    band: 'bg-green-500',
  },
}

const COMPLEXITY_CONFIG: Record<
  TemplateComplexity,
  { label: string; className: string }
> = {
  einfach: {
    label: 'Einfach',
    className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  },
  mittel: {
    label: 'Mittel',
    className:
      'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
  },
  fortgeschritten: {
    label: 'Fortgeschritten',
    className:
      'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
  },
}

// ---------------------------------------------------------------------------
// Template card
// ---------------------------------------------------------------------------

function TemplateCard({
  template,
  onPreview,
}: {
  template: AutomationTemplate
  onPreview: (t: AutomationTemplate) => void
}) {
  const catConfig =
    CATEGORY_CONFIG[template.category] ?? CATEGORY_CONFIG.vertrieb
  const cxConfig =
    COMPLEXITY_CONFIG[template.complexity] ?? COMPLEXITY_CONFIG.einfach

  return (
    <div className="group relative rounded-lg border border-border overflow-hidden hover:border-primary/50 transition-colors">
      {/* Category color band */}
      <div className={`h-1 ${catConfig.band}`} />

      <div className="p-4 space-y-2">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            <catConfig.icon className={`h-4 w-4 ${catConfig.color}`} />
            <h3 className="text-sm font-medium text-foreground">{template.name}</h3>
          </div>
          <span
            className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${cxConfig.className}`}
          >
            {cxConfig.label}
          </span>
        </div>

        <p className="text-xs text-muted-foreground line-clamp-2">
          {template.description}
        </p>

        <button
          onClick={() => onPreview(template)}
          className="flex items-center gap-1 rounded-md border border-border px-2.5 py-1 text-[11px] text-foreground hover:bg-secondary transition-colors"
        >
          <Eye className="h-3 w-3" />
          Vorschau
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Template preview dialog
// ---------------------------------------------------------------------------

function TemplatePreviewDialog({
  template,
  open,
  onOpenChange,
  onUse,
}: {
  template: AutomationTemplate | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onUse: (t: AutomationTemplate) => void
}) {
  if (!template) return null

  const catConfig =
    CATEGORY_CONFIG[template.category] ?? CATEGORY_CONFIG.vertrieb
  const cxConfig =
    COMPLEXITY_CONFIG[template.complexity] ?? COMPLEXITY_CONFIG.einfach

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <catConfig.icon className={`h-5 w-5 ${catConfig.color}`} />
            {template.name}
          </DialogTitle>
          <DialogDescription>{template.description}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Meta */}
          <div className="flex items-center gap-3">
            <span
              className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${cxConfig.className}`}
            >
              {cxConfig.label}
            </span>
            <span className="text-xs text-muted-foreground">
              {catConfig.label}
            </span>
          </div>

          {/* Trigger */}
          <div className="rounded-lg border border-border p-3">
            <h4 className="text-xs font-medium text-muted-foreground mb-1">
              Trigger
            </h4>
            <p className="text-sm text-foreground">{template.trigger_type}</p>
          </div>

          {/* Conditions */}
          <div className="rounded-lg border border-border p-3">
            <h4 className="text-xs font-medium text-muted-foreground mb-1">
              Bedingung
            </h4>
            <p className="text-sm text-foreground">
              {template.conditions?.mode === 'expression'
                ? template.conditions.expression || 'Keine Bedingung'
                : template.conditions?.simple
                  ? 'Einfache Bedingung konfiguriert'
                  : 'Keine Bedingung'}
            </p>
          </div>

          {/* Actions */}
          <div className="rounded-lg border border-border p-3">
            <h4 className="text-xs font-medium text-muted-foreground mb-1">
              Aktionen ({template.actions?.length ?? 0})
            </h4>
            {(template.actions ?? []).length === 0 ? (
              <p className="text-sm text-muted-foreground">Keine Aktionen</p>
            ) : (
              <ul className="space-y-1">
                {(template.actions ?? []).map((action, idx) => (
                  <li
                    key={idx}
                    className="text-sm text-foreground flex items-center gap-2"
                  >
                    <span className="flex items-center justify-center h-5 w-5 rounded-full bg-secondary text-[10px] font-bold">
                      {idx + 1}
                    </span>
                    {action.type}
                  </li>
                ))}
              </ul>
            )}
          </div>

          {/* Action button */}
          <div className="flex justify-end">
            <button
              onClick={() => onUse(template)}
              className="rounded-md bg-primary px-4 py-2 text-xs font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              Vorlage verwenden
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function TemplateGallery() {
  const { data, isLoading } = useAutomationTemplates()
  const { templateGrouping, setTemplateGrouping } = useAutomatisierungStore()
  const createFromTemplate = useCreateFromTemplate()
  const [search, setSearch] = useState('')
  const [previewTemplate, setPreviewTemplate] = useState<AutomationTemplate | null>(null)

  const templates = useMemo(() => data?.templates ?? [], [data?.templates])

  // Filter by search
  const filtered = useMemo(() => {
    if (!search) return templates
    const q = search.toLowerCase()
    return templates.filter(
      (t) =>
        t.name.toLowerCase().includes(q) ||
        t.description.toLowerCase().includes(q),
    )
  }, [templates, search])

  // Group by selected mode
  const grouped = useMemo(() => {
    const groups: Record<string, AutomationTemplate[]> = {}

    for (const t of filtered) {
      const key =
        templateGrouping === 'module' ? t.category : t.complexity
      if (!groups[key]) groups[key] = []
      groups[key].push(t)
    }

    return groups
  }, [filtered, templateGrouping])

  const handleUseTemplate = (template: AutomationTemplate) => {
    setPreviewTemplate(null)
    createFromTemplate.mutate({
      templateId: template.id,
      name: `${template.name} (Kopie)`,
    })
  }

  const groupLabel = (key: string): string => {
    if (templateGrouping === 'module') {
      return CATEGORY_CONFIG[key as TemplateCategory]?.label ?? key
    }
    return COMPLEXITY_CONFIG[key as TemplateComplexity]?.label ?? key
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Controls */}
      <div className="flex items-center justify-between gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Vorlagen suchen..."
            className="w-full rounded-md border border-border bg-background pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
        </div>

        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>Nach Modul</span>
          <Switch.Root
            checked={templateGrouping === 'complexity'}
            onCheckedChange={(checked) =>
              setTemplateGrouping(checked ? 'complexity' : 'module')
            }
            className="relative inline-flex h-5 w-9 items-center rounded-full transition-colors data-[state=checked]:bg-primary data-[state=unchecked]:bg-secondary"
          >
            <Switch.Thumb className="block h-4 w-4 rounded-full bg-white transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0.5 shadow-sm" />
          </Switch.Root>
          <span>Nach Komplexitaet</span>
        </div>
      </div>

      {/* Grouped templates */}
      {Object.entries(grouped).map(([key, groupTemplates]) => (
        <div key={key}>
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">
            {groupLabel(key)}
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {groupTemplates.map((t) => (
              <TemplateCard
                key={t.id}
                template={t}
                onPreview={setPreviewTemplate}
              />
            ))}
          </div>
        </div>
      ))}

      {Object.keys(grouped).length === 0 && (
        <div className="py-8 text-center text-sm text-muted-foreground">
          Keine Vorlagen gefunden
        </div>
      )}

      {/* Preview dialog */}
      <TemplatePreviewDialog
        template={previewTemplate}
        open={!!previewTemplate}
        onOpenChange={(open) => !open && setPreviewTemplate(null)}
        onUse={handleUseTemplate}
      />
    </div>
  )
}
